package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// KafkaLogDirsResponse represents the JSON output from kafka-log-dirs.sh
type KafkaLogDirsResponse struct {
	Version int                  `json:"version"`
	Brokers []KafkaBrokerLogDirs `json:"brokers"`
}

type KafkaBrokerLogDirs struct {
	Broker  int             `json:"broker"`
	LogDirs []KafkaLogDir   `json:"logDirs"`
}

type KafkaLogDir struct {
	LogDir     string               `json:"logDir"`
	Error      *string              `json:"error"`
	Partitions []KafkaPartitionInfo `json:"partitions"`
}

type KafkaPartitionInfo struct {
	Partition string `json:"partition"` // Format: "topic-name-partition-number"
	Size      int64  `json:"size"`
	OffsetLag int64  `json:"offsetLag"`
	IsFuture  bool   `json:"isFuture"`
}

// getTopicSizesFromKafkaCLI uses kafka-log-dirs.sh to get topic sizes (KRaft-compatible)
func getTopicSizesFromKafkaCLI(config *sarama.Config, brokers []string, topicList []string) (*TopicSizesReport, error) {
	// Find kafka-log-dirs.sh in PATH or common locations
	kafkaLogDirsPath, err := findKafkaLogDirs()
	if err != nil {
		return nil, fmt.Errorf("kafka-log-dirs.sh not found: %v\nPlease ensure Kafka bin directory is in your PATH", err)
	}

	log.Printf("Using kafka-log-dirs.sh: %s", kafkaLogDirsPath)

	// Build command arguments
	args := []string{
		"--bootstrap-server", strings.Join(brokers, ","),
		"--describe",
	}

	// Add topic list if specified
	if len(topicList) > 0 {
		args = append(args, "--topic-list", strings.Join(topicList, ","))
	}

	// Create temporary config file if authentication is needed
	configFile, cleanup, err := createKafkaConfigFile(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create config file: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}
	if configFile != "" {
		args = append(args, "--command-config", configFile)
	}

	// Execute kafka-log-dirs.sh
	log.Printf("Executing: %s %s", kafkaLogDirsPath, strings.Join(args, " "))
	cmd := exec.Command(kafkaLogDirsPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("kafka-log-dirs.sh failed: %v\nStderr: %s", err, stderr.String())
	}

	// Parse JSON output (skip first 2 lines which are status messages)
	output := stdout.String()
	lines := strings.Split(output, "\n")
	var jsonLine string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			jsonLine = line
			break
		}
	}
	
	if jsonLine == "" {
		return nil, fmt.Errorf("no JSON found in kafka-log-dirs.sh output: %s", output)
	}

	// Parse JSON output
	var kafkaResponse KafkaLogDirsResponse
	if err := json.Unmarshal([]byte(jsonLine), &kafkaResponse); err != nil {
		return nil, fmt.Errorf("failed to parse kafka-log-dirs.sh output: %v\nJSON: %s", err, jsonLine[:min(len(jsonLine), 500)])
	}

	// Convert to our TopicSizesReport format
	return convertKafkaLogDirsToReport(kafkaResponse, brokers), nil
}

// findKafkaLogDirs searches for kafka-log-dirs.sh in common locations
func findKafkaLogDirs() (string, error) {
	// Common names
	names := []string{"kafka-log-dirs.sh", "kafka-log-dirs"}

	// Check PATH first
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Check common Kafka installation directories
	commonPaths := []string{
		"/usr/local/kafka/bin",
		"/opt/kafka/bin",
		"/usr/local/bin",
		"/opt/homebrew/bin",
		filepath.Join(os.Getenv("HOME"), "kafka", "bin"),
		filepath.Join(os.Getenv("HOME"), "kafka_2.13-3.6.0", "bin"),
		filepath.Join(os.Getenv("HOME"), "kafka_2.13-3.7.0", "bin"),
	}

	// Add KAFKA_HOME/bin if set
	if kafkaHome := os.Getenv("KAFKA_HOME"); kafkaHome != "" {
		commonPaths = append([]string{filepath.Join(kafkaHome, "bin")}, commonPaths...)
	}

	for _, dir := range commonPaths {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("kafka-log-dirs.sh not found in PATH or common locations")
}

// createKafkaConfigFile creates a temporary properties file for authentication
func createKafkaConfigFile(config *sarama.Config) (string, func(), error) {
	if config.Net.SASL.Enable == false && config.Net.TLS.Enable == false {
		// No auth needed
		return "", nil, nil
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "kafka-config-*.properties")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	var lines []string

	// Add SASL configuration
	if config.Net.SASL.Enable {
		protocol := "SASL_PLAINTEXT"
		if config.Net.TLS.Enable {
			protocol = "SASL_SSL"
		}
		lines = append(lines, fmt.Sprintf("security.protocol=%s", protocol))
		lines = append(lines, fmt.Sprintf("sasl.mechanism=%s", config.Net.SASL.Mechanism))

		switch config.Net.SASL.Mechanism {
		case "PLAIN":
			lines = append(lines, fmt.Sprintf("sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username=\"%s\" password=\"%s\";",
				config.Net.SASL.User, config.Net.SASL.Password))
		case "SCRAM-SHA-256", "SCRAM-SHA-512":
			lines = append(lines, fmt.Sprintf("sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username=\"%s\" password=\"%s\";",
				config.Net.SASL.User, config.Net.SASL.Password))
		}
	} else if config.Net.TLS.Enable {
		lines = append(lines, "security.protocol=SSL")
	}

	// Add TLS configuration if needed
	if config.Net.TLS.Enable && config.Net.TLS.Config != nil {
		// Note: For full TLS support, we'd need to handle certificates
		// This basic implementation handles TLS without client certs
		lines = append(lines, "ssl.endpoint.identification.algorithm=")
	}

	// Write configuration
	if _, err := tmpFile.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		cleanup()
		return "", nil, err
	}

	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpFile.Name(), cleanup, nil
}

// convertKafkaLogDirsToReport converts kafka-log-dirs.sh output to our report format
func convertKafkaLogDirsToReport(kafkaResponse KafkaLogDirsResponse, brokers []string) *TopicSizesReport {
	topicSizes := make(map[string]*TopicSize)
	partitionCounts := make(map[string]map[int]bool) // topic -> set of partition numbers

	// Regex to extract topic name and partition from format "topic-name-partition-number"
	// The last dash separates the partition number from the topic name
	partitionRegex := regexp.MustCompile(`^(.+)-(\d+)$`)

	// Aggregate sizes by topic
	for _, broker := range kafkaResponse.Brokers {
		for _, logDir := range broker.LogDirs {
			if logDir.Error != nil {
				log.Printf("Warning: Error in log dir %s on broker %d: %s", logDir.LogDir, broker.Broker, *logDir.Error)
				continue
			}

			for _, partition := range logDir.Partitions {
				// Extract topic name from partition string (format: "topic-name-partition-number")
				matches := partitionRegex.FindStringSubmatch(partition.Partition)
				if len(matches) != 3 {
					log.Printf("Warning: Cannot parse partition format: %s", partition.Partition)
					continue
				}

				topicName := matches[1]
				partitionNum, err := strconv.Atoi(matches[2])
				if err != nil {
					log.Printf("Warning: Invalid partition number in: %s", partition.Partition)
					continue
				}

				// Initialize topic if not seen before
				if _, exists := topicSizes[topicName]; !exists {
					topicSizes[topicName] = &TopicSize{
						Topic:      topicName,
						TotalSize:  0,
						Partitions: 0,
					}
					partitionCounts[topicName] = make(map[int]bool)
				}

				// Add size (each replica counts)
				topicSizes[topicName].TotalSize += partition.Size

				// Track unique partitions
				partitionCounts[topicName][partitionNum] = true
			}
		}
	}

	// Update partition counts
	for topicName, partitions := range partitionCounts {
		topicSizes[topicName].Partitions = len(partitions)
	}

	// Build report
	report := &TopicSizesReport{
		Timestamp:   time.Now().Format(time.RFC3339),
		Cluster:     strings.Join(brokers, ","),
		Topics:      []TopicSize{},
		TotalSize:   0,
		TotalTopics: len(topicSizes),
	}

	// Convert map to slice and calculate totals
	for _, ts := range topicSizes {
		ts.TotalSizeStr = formatBytes(ts.TotalSize)
		report.Topics = append(report.Topics, *ts)
		report.TotalSize += ts.TotalSize
		report.TotalPartitions += ts.Partitions
	}

	// Sort by size (largest first)
	sort.Slice(report.Topics, func(i, j int) bool {
		return report.Topics[i].TotalSize > report.Topics[j].TotalSize
	})

	report.TotalSizeStr = formatBytes(report.TotalSize)
	report.TotalTopics = len(report.Topics)

	return report
}

// getTopicSizesViaCLI is the main entry point using kafka-log-dirs.sh
func getTopicSizesViaCLI(config *sarama.Config, brokers []string, topicList []string) (*TopicSizesReport, error) {
	log.Println("Using kafka-log-dirs.sh for KRaft compatibility...")
	return getTopicSizesFromKafkaCLI(config, brokers, topicList)
}
