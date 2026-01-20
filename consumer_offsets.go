package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/IBM/sarama"
)

// fetchConsumerOffsets retrieves current offset positions for all consumer groups
func fetchConsumerOffsets(admin sarama.ClusterAdmin, groups []ConsumerGroupInfo, cluster string) (*ConsumerOffsetsBackup, error) {
	backup := &ConsumerOffsetsBackup{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Cluster:        cluster,
		ConsumerGroups: make([]ConsumerGroupOffsets, 0),
	}

	for _, group := range groups {
		if len(group.Topics) == 0 {
			continue
		}

		groupOffsets := ConsumerGroupOffsets{
			Group:     group.Name,
			Topics:    make(map[string][]PartitionOffset),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		// Fetch offsets for each topic
		for _, topic := range group.Topics {
			offsets, err := admin.ListConsumerGroupOffsets(group.Name, map[string][]int32{topic: nil})
			if err != nil {
				log.Printf("Warning: Could not fetch offsets for group %s, topic %s: %v", group.Name, topic, err)
				continue
			}

			if block, ok := offsets.Blocks[topic]; ok {
				partitionOffsets := make([]PartitionOffset, 0)
				for partition, offsetInfo := range block {
					partitionOffsets = append(partitionOffsets, PartitionOffset{
						Partition: int(partition),
						Offset:    offsetInfo.Offset,
					})
				}

				// Sort by partition number
				sort.Slice(partitionOffsets, func(i, j int) bool {
					return partitionOffsets[i].Partition < partitionOffsets[j].Partition
				})

				if len(partitionOffsets) > 0 {
					groupOffsets.Topics[topic] = partitionOffsets
				}
			}
		}

		if len(groupOffsets.Topics) > 0 {
			backup.ConsumerGroups = append(backup.ConsumerGroups, groupOffsets)
		}
	}

	return backup, nil
}

// saveConsumerOffsetsToFile saves consumer group offsets to a JSON file
func saveConsumerOffsetsToFile(backup *ConsumerOffsetsBackup, filename string) error {
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal offsets: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// generateRestoreOffsetsScript generates a shell script to restore consumer group offsets
func generateRestoreOffsetsScript(backup *ConsumerOffsetsBackup, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	// Write script header
	fmt.Fprintf(w, `#!/bin/bash
# Consumer Group Offsets Restore Script
# Generated: %s
# Source Cluster: %s
# Total Consumer Groups: %d
#
# Usage:
#   1. Edit BOOTSTRAP_SERVERS to point to your target cluster
#   2. Add authentication flags if needed (--command-config, etc.)
#   3. Run: chmod +x %s && ./%s
#

set -e  # Exit on error

# Target cluster configuration
BOOTSTRAP_SERVERS="localhost:9092"  # CHANGE THIS
# Uncomment and configure if authentication is needed:
# COMMAND_CONFIG="--command-config client.properties"
COMMAND_CONFIG=""

# Kafka consumer groups command (adjust path if needed)
KAFKA_CONSUMER_GROUPS="kafka-consumer-groups.sh"

echo "========================================"
echo "Restoring offsets for %d consumer groups"
echo "Target: $BOOTSTRAP_SERVERS"
echo "========================================"
echo ""

RESTORED=0
FAILED=0

`, backup.Timestamp, backup.Cluster, len(backup.ConsumerGroups), filename, filename, len(backup.ConsumerGroups))

	// Write offset restoration commands for each group
	for groupIdx, group := range backup.ConsumerGroups {
		fmt.Fprintf(w, "# Consumer Group %d/%d: %s\n", groupIdx+1, len(backup.ConsumerGroups), group.Group)
		fmt.Fprintf(w, "echo \"[%d/%d] Restoring offsets for consumer group: %s\"\n", groupIdx+1, len(backup.ConsumerGroups), group.Group)

		for topic, partitions := range group.Topics {
			fmt.Fprintf(w, "echo \"  Topic: %s (%d partitions)\"\n", topic, len(partitions))

			// Create JSON for offset reset
			fmt.Fprintf(w, "cat > /tmp/offsets_%s_%s.json << 'OFFSET_EOF'\n", group.Group, topic)
			fmt.Fprintf(w, "{\n")
			fmt.Fprintf(w, "  \"partitions\": [\n")

			for i, partition := range partitions {
				comma := ","
				if i == len(partitions)-1 {
					comma = ""
				}
				fmt.Fprintf(w, "    {\"topic\": \"%s\", \"partition\": %d, \"offset\": %d}%s\n",
					topic, partition.Partition, partition.Offset, comma)
			}

			fmt.Fprintf(w, "  ]\n")
			fmt.Fprintf(w, "}\n")
			fmt.Fprintf(w, "OFFSET_EOF\n\n")

			// Execute offset reset
			fmt.Fprintf(w, "if $KAFKA_CONSUMER_GROUPS --bootstrap-server \"$BOOTSTRAP_SERVERS\" $COMMAND_CONFIG \\\n")
			fmt.Fprintf(w, "  --group \"%s\" \\\n", group.Group)
			fmt.Fprintf(w, "  --topic \"%s\" \\\n", topic)
			fmt.Fprintf(w, "  --reset-offsets \\\n")
			fmt.Fprintf(w, "  --from-json-file /tmp/offsets_%s_%s.json \\\n", group.Group, topic)
			fmt.Fprintf(w, "  --execute; then\n")
			fmt.Fprintf(w, "  echo \"    ✓ Restored %d partitions\"\n", len(partitions))
			fmt.Fprintf(w, "  ((RESTORED++))\n")
			fmt.Fprintf(w, "else\n")
			fmt.Fprintf(w, "  echo \"    ✗ Failed to restore offsets\"\n")
			fmt.Fprintf(w, "  ((FAILED++))\n")
			fmt.Fprintf(w, "fi\n")
			fmt.Fprintf(w, "rm -f /tmp/offsets_%s_%s.json\n", group.Group, topic)
			fmt.Fprintf(w, "\n")
		}

		fmt.Fprintf(w, "\n")
	}

	// Write summary
	fmt.Fprintf(w, `echo "========================================"
echo "Offset Restore Summary:"
echo "  Successfully restored: $RESTORED topics"
echo "  Failed: $FAILED topics"
echo "========================================"
echo ""
echo "Note: Verify offsets were restored correctly:"
echo "  $KAFKA_CONSUMER_GROUPS --bootstrap-server \"$BOOTSTRAP_SERVERS\" $COMMAND_CONFIG --group <group-name> --describe"
`)

	if err := os.Chmod(filename, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	return nil
}
