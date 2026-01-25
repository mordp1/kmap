package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/IBM/sarama"
)

// TopicSize represents the size of a topic across all brokers
type TopicSize struct {
	Topic        string `json:"topic"`
	TotalSize    int64  `json:"total_size_bytes"`
	TotalSizeStr string `json:"total_size_human"`
	Partitions   int    `json:"partitions"`
}

// TopicSizesReport represents the complete size report
type TopicSizesReport struct {
	Timestamp       string      `json:"timestamp"`
	Cluster         string      `json:"cluster"`
	Topics          []TopicSize `json:"topics"`
	TotalSize       int64       `json:"total_size_bytes"`
	TotalSizeStr    string      `json:"total_size_human"`
	TotalTopics     int         `json:"total_topics"`
	TotalPartitions int         `json:"total_partitions"`
}

// DescribeLogDirsResponse represents the response from DescribeLogDirs API
type DescribeLogDirsResponse struct {
	Brokers []BrokerLogDirs `json:"brokers"`
	Version int             `json:"version"`
}

type BrokerLogDirs struct {
	Broker  int32    `json:"broker"`
	LogDirs []LogDir `json:"logDirs"`
}

type LogDir struct {
	Error      *string           `json:"error"`
	LogDir     string            `json:"logDir"`
	Partitions []PartitionLogDir `json:"partitions"`
}

type PartitionLogDir struct {
	Partition string `json:"partition"`
	Size      int64  `json:"size"`
	OffsetLag int64  `json:"offsetLag"`
	IsFuture  bool   `json:"isFuture"`
}

// getTopicSizes queries all brokers for topic sizes
func getTopicSizes(brokers []string, config *sarama.Config, topicFilter []string) (*TopicSizesReport, error) {
	log.Println("Querying brokers for log directory information...")

	// Create client
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}
	defer client.Close()

	// Get all brokers
	brokerIDs := client.Brokers()
	log.Printf("Found %d brokers", len(brokerIDs))

	// Refresh metadata to ensure brokers are up to date
	if err := client.RefreshMetadata(); err != nil {
		log.Printf("Warning: Could not refresh metadata: %v", err)
	}

	// Map to store topic sizes (topic -> total size)
	topicSizes := make(map[string]int64)
	topicPartitions := make(map[string]map[int32]bool) // Track unique partitions per topic

	// Query each broker
	for _, broker := range brokerIDs {
		log.Printf("Querying broker %d at %s...", broker.ID(), broker.Addr())
		
		// Ensure broker is connected
		if ok, _ := broker.Connected(); !ok {
			if err := broker.Open(config); err != nil {
				log.Printf("Warning: Could not connect to broker %d at %s: %v", broker.ID(), broker.Addr(), err)
				continue
			}
		}

		// Create DescribeLogDirs request - Version 0, empty DescribeTopics means all topics
		request := &sarama.DescribeLogDirsRequest{
			Version:        0,
			DescribeTopics: []sarama.DescribeLogDirsRequestTopic{},
		}

		log.Printf("Sending DescribeLogDirs request to broker %d...", broker.ID())
		response, err := broker.DescribeLogDirs(request)
		
		if err != nil {
			log.Printf("Warning: Error querying log dirs from broker %d at %s: %v", broker.ID(), broker.Addr(), err)
			continue
		}

		// Process response
		if len(response.LogDirs) == 0 {
			log.Printf("Warning: No log directories returned from broker %d", broker.ID())
			continue
		}

		for _, logDirInfo := range response.LogDirs {
			if logDirInfo.ErrorCode != sarama.ErrNoError {
				log.Printf("Warning: Log directory error on broker %d: %v", broker.ID(), logDirInfo.ErrorCode)
				continue
			}

			// Topics is a slice of DescribeLogDirsResponseTopic
			for _, topicInfo := range logDirInfo.Topics {
				topicName := topicInfo.Topic

				// Apply topic filter if specified
				if len(topicFilter) > 0 && !contains(topicFilter, topicName) {
					continue
				}

				// Initialize maps if needed
				if _, exists := topicPartitions[topicName]; !exists {
					topicPartitions[topicName] = make(map[int32]bool)
				}

				// Partitions is a slice of DescribeLogDirsResponsePartition
				for _, partitionInfo := range topicInfo.Partitions {
					topicSizes[topicName] += partitionInfo.Size
					topicPartitions[topicName][partitionInfo.PartitionID] = true
				}
			}
		}
	}

	if len(topicSizes) == 0 {
		return nil, fmt.Errorf("no topic size data retrieved")
	}

	// Build report
	report := &TopicSizesReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Cluster:   strings.Join(brokers, ","),
	}

	// Convert map to slice and sort
	topics := make([]TopicSize, 0, len(topicSizes))
	for topic, size := range topicSizes {
		topics = append(topics, TopicSize{
			Topic:        topic,
			TotalSize:    size,
			TotalSizeStr: formatBytes(size),
			Partitions:   len(topicPartitions[topic]),
		})
		report.TotalSize += size
		report.TotalPartitions += len(topicPartitions[topic])
	}

	// Sort by size (descending)
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].TotalSize > topics[j].TotalSize
	})

	report.Topics = topics
	report.TotalTopics = len(topics)
	report.TotalSizeStr = formatBytes(report.TotalSize)

	log.Printf("Successfully retrieved size information for %d topics", len(topics))

	return report, nil
}

// printTopicSizes prints the topic sizes report in table format
func printTopicSizes(report *TopicSizesReport) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("Kafka Topic Sizes Report\n")
	fmt.Printf("Generated: %s\n", report.Timestamp)
	fmt.Printf("Cluster: %s\n", report.Cluster)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TOPIC\tPARTITIONS\tTOTAL SIZE\tSIZE (BYTES)")
	fmt.Fprintln(w, strings.Repeat("-", 40)+"\t"+strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 15))

	for _, topic := range report.Topics {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			topic.Topic,
			topic.Partitions,
			topic.TotalSizeStr,
			formatNumber(topic.TotalSize),
		)
	}

	w.Flush()

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total Topics: %d\n", report.TotalTopics)
	fmt.Printf("  Total Partitions: %d\n", report.TotalPartitions)
	fmt.Printf("  Total Size: %s (%s bytes)\n", report.TotalSizeStr, formatNumber(report.TotalSize))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}

// saveTopicSizesJSON saves the report to a JSON file
func saveTopicSizesJSON(report *TopicSizesReport, filename string) error {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	log.Printf("Saved topic sizes report to %s", filename)
	return nil
}

// formatBytes formats bytes to human-readable format (IEC units)
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
