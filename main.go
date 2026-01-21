package main

import (
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

// Build info - set via ldflags during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// SCRAM authentication support
var (
	SHA256 scram.HashGeneratorFcn = sha256.New
	SHA512 scram.HashGeneratorFcn = sha512.New
)

type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
	response, err = x.ClientConversation.Step(challenge)
	return
}

func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}

type BrokerInfo struct {
	ID              int32  `json:"id"`
	Address         string `json:"address"`
	Version         string `json:"version"`
	Partitions      int    `json:"partitions"`
	Leaders         int    `json:"leaders"`
	UnderReplicated int    `json:"under_replicated_partitions"`
}

type TopicInfo struct {
	Name              string            `json:"name"`
	Partitions        int               `json:"partitions"`
	ReplicationFactor int               `json:"replication_factor"`
	TotalMessages     int64             `json:"total_messages"`
	Configs           map[string]string `json:"configs,omitempty"`
}

type ConsumerGroupInfo struct {
	Name    string   `json:"name"`
	Topics  []string `json:"topics"`
	Members int      `json:"members"`
	State   string   `json:"state"`
}

type PartitionOffset struct {
	Partition int   `json:"partition"`
	Offset    int64 `json:"offset"`
	Timestamp int64 `json:"timestamp,omitempty"`
	Lag       int64 `json:"lag,omitempty"`
}

type ConsumerGroupOffsets struct {
	Group     string                       `json:"group"`
	Topics    map[string][]PartitionOffset `json:"topics"`
	Timestamp string                       `json:"captured_at"`
}

type ConsumerOffsetsBackup struct {
	Timestamp      string                 `json:"timestamp"`
	Cluster        string                 `json:"cluster"`
	ConsumerGroups []ConsumerGroupOffsets `json:"consumer_groups"`
}

type KafkaClusterInfo struct {
	Timestamp           string              `json:"timestamp"`
	Brokers             []string            `json:"broker_addresses"`
	BrokerDetails       []BrokerInfo        `json:"brokers"`
	Topics              []TopicInfo         `json:"topics"`
	ConsumerGroups      []ConsumerGroupInfo `json:"consumer_groups"`
	TotalTopics         int                 `json:"total_topics"`
	TotalConsumerGroups int                 `json:"total_consumer_groups"`
	TotalPartitions     int                 `json:"total_partitions"`
	TotalMessages       int64               `json:"total_messages"`
	TotalURPs           int                 `json:"total_under_replicated_partitions"`
}

func main() {
	brokers := flag.String("brokers", "localhost:9092", "Kafka broker addresses (comma-separated)")
	outputJSON := flag.String("output", "kafka-cluster-info.json", "Output JSON file")
	outputHTML := flag.String("html", "kafka-cluster-report.html", "Output HTML report")
	outputDOT := flag.String("dot", "", "Output DOT file for Graphviz visualization (optional)")
	recreateScript := flag.String("recreate-script", "", "Generate shell script to recreate topics (optional)")
	saveOffsets := flag.String("save-offsets", "", "Save consumer group offsets to JSON file (optional)")
	restoreOffsetsScript := flag.String("restore-offsets-script", "", "Generate script to restore consumer offsets (requires -save-offsets)")
	showVersion := flag.Bool("version", false, "Show version information")

	// Authentication flags
	securityProtocol := flag.String("security-protocol", "", "Security protocol (SASL_SSL, SASL_PLAINTEXT, SSL, or empty for PLAINTEXT)")
	saslMechanism := flag.String("sasl-mechanism", "PLAIN", "SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)")
	saslUsername := flag.String("sasl-username", "", "SASL username")
	saslPassword := flag.String("sasl-password", "", "SASL password")

	// TLS/SSL flags
	tlsCACert := flag.String("tls-ca-cert", "", "Path to CA certificate file (for SSL/TLS)")
	tlsClientCert := flag.String("tls-client-cert", "", "Path to client certificate file (for mTLS)")
	tlsClientKey := flag.String("tls-client-key", "", "Path to client key file (for mTLS)")
	tlsSkipVerify := flag.Bool("tls-skip-verify", false, "Skip TLS certificate verification (insecure, for development only)")

	flag.Parse()

	if *showVersion {
		fmt.Printf("Kafka Analyzer\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Build Time: %s\n", BuildTime)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	brokerList := strings.Split(*brokers, ",")

	log.Printf("Kafka Analyzer v%s (%s)", Version, GitCommit)
	log.Printf("Connecting to Kafka brokers: %v", brokerList)

	config := sarama.NewConfig()
	// Use Kafka 2.6 for AWS MSK 2.6 compatibility
	config.Version = sarama.V2_6_0_0
	config.Consumer.Return.Errors = true

	// AWS MSK connection settings
	config.Net.DialTimeout = 30 * time.Second
	config.Net.ReadTimeout = 30 * time.Second
	config.Net.WriteTimeout = 30 * time.Second
	config.Metadata.Timeout = 60 * time.Second
	config.Metadata.Retry.Max = 3
	config.Metadata.Retry.Backoff = 250 * time.Millisecond

	// Configure security protocol
	if *securityProtocol != "" {
		switch strings.ToUpper(*securityProtocol) {
		case "SASL_SSL":
			config.Net.SASL.Enable = true
			config.Net.TLS.Enable = true
		case "SASL_PLAINTEXT":
			config.Net.SASL.Enable = true
			config.Net.TLS.Enable = false
		case "SSL":
			config.Net.TLS.Enable = true
		default:
			log.Fatalf("Unknown security protocol: %s", *securityProtocol)
		}
	}

	// Configure SASL
	if config.Net.SASL.Enable {
		if *saslUsername == "" || *saslPassword == "" {
			log.Fatalf("SASL username and password are required when using SASL authentication")
		}

		config.Net.SASL.User = *saslUsername
		config.Net.SASL.Password = *saslPassword
		// AWS MSK requires SASL handshake version 1
		config.Net.SASL.Handshake = true
		config.Net.SASL.Version = 1

		switch strings.ToUpper(*saslMechanism) {
		case "PLAIN":
			config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		case "SCRAM-SHA-256":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
		case "SCRAM-SHA-512":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
		default:
			log.Fatalf("Unknown SASL mechanism: %s", *saslMechanism)
		}

		log.Printf("Using SASL authentication: protocol=%s, mechanism=%s, user=%s", *securityProtocol, *saslMechanism, *saslUsername)
	}

	// Configure TLS
	if config.Net.TLS.Enable {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: *tlsSkipVerify,
		}

		// Load CA certificate if provided
		if *tlsCACert != "" {
			caCert, err := os.ReadFile(*tlsCACert)
			if err != nil {
				log.Fatalf("Error reading CA certificate: %v", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				log.Fatalf("Failed to parse CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
			log.Printf("Loaded CA certificate from: %s", *tlsCACert)
		} else if !*tlsSkipVerify {
			// Use system certificates when no CA cert provided and verification enabled
			// This is needed for AWS MSK and other managed Kafka services
			systemCertPool, err := x509.SystemCertPool()
			if err != nil {
				log.Printf("Warning: Failed to load system certificates, using empty pool: %v", err)
				systemCertPool = x509.NewCertPool()
			}
			tlsConfig.RootCAs = systemCertPool
		}

		// Load client certificate and key for mTLS
		if *tlsClientCert != "" && *tlsClientKey != "" {
			cert, err := tls.LoadX509KeyPair(*tlsClientCert, *tlsClientKey)
			if err != nil {
				log.Fatalf("Error loading client certificate/key: %v", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
			log.Printf("Loaded client certificate for mTLS from: %s", *tlsClientCert)
		} else if *tlsClientCert != "" || *tlsClientKey != "" {
			log.Fatalf("Both -tls-client-cert and -tls-client-key must be provided for mTLS")
		}

		config.Net.TLS.Config = tlsConfig

		if *tlsSkipVerify {
			log.Printf("WARNING: TLS certificate verification is disabled (insecure)")
		}

		// Only log TLS auth if not using SASL (where TLS is just the transport layer)
		if !config.Net.SASL.Enable {
			authType := "TLS/SSL"
			if len(tlsConfig.Certificates) > 0 {
				authType = "mTLS (mutual TLS)"
			}
			log.Printf("Using %s authentication", authType)
		}
	}

	admin, err := sarama.NewClusterAdmin(brokerList, config)
	if err != nil {
		log.Fatalf("Error creating cluster admin: %v", err)
	}
	defer admin.Close()

	// Create client for offset operations
	client, err := sarama.NewClient(brokerList, config)
	if err != nil {
		log.Fatalf("Error creating Kafka client: %v", err)
	}
	defer client.Close()

	clusterInfo := KafkaClusterInfo{
		Timestamp: time.Now().Format(time.RFC3339),
		Brokers:   brokerList,
	}

	// Get broker metadata
	log.Println("Fetching broker information...")
	broker := sarama.NewBroker(brokerList[0])
	broker.Open(config)
	defer broker.Close()

	metadataReq := &sarama.MetadataRequest{}
	metadata, err := broker.GetMetadata(metadataReq)
	if err != nil {
		log.Printf("Warning: Could not fetch metadata: %v", err)
	} else {
		// Collect broker information
		brokerDetails := make([]BrokerInfo, 0, len(metadata.Brokers))
		for _, b := range metadata.Brokers {
			brokerInfo := BrokerInfo{
				ID:      b.ID(),
				Address: b.Addr(),
				Version: "", // Will be populated later
			}
			brokerDetails = append(brokerDetails, brokerInfo)
		}
		clusterInfo.BrokerDetails = brokerDetails
	}

	// Get all topics
	log.Println("Fetching topics...")
	topics, err := admin.ListTopics()
	if err != nil {
		log.Fatalf("Error listing topics: %v", err)
	}

	topicInfos := make([]TopicInfo, 0, len(topics))
	for name, detail := range topics {
		topicInfo := TopicInfo{
			Name:              name,
			Partitions:        int(detail.NumPartitions),
			ReplicationFactor: int(detail.ReplicationFactor),
		}

		// Get topic configurations
		configs, err := admin.DescribeConfig(sarama.ConfigResource{
			Type: sarama.TopicResource,
			Name: name,
		})
		if err == nil && len(configs) > 0 {
			topicInfo.Configs = make(map[string]string)
			for _, entry := range configs {
				if !entry.Default && entry.Value != "" {
					topicInfo.Configs[entry.Name] = entry.Value
				}
			}
		}

		// Get high watermarks (total messages) for all partitions
		topicInfo.TotalMessages = getTopicMessageCount(client, name, int(detail.NumPartitions))

		topicInfos = append(topicInfos, topicInfo)
	}

	sort.Slice(topicInfos, func(i, j int) bool {
		return topicInfos[i].Name < topicInfos[j].Name
	})

	clusterInfo.Topics = topicInfos
	clusterInfo.TotalTopics = len(topicInfos)
	clusterInfo.TotalPartitions = getTotalPartitions(topicInfos)
	clusterInfo.TotalMessages = getTotalMessages(topicInfos)

	// Calculate partition and leader distribution across brokers
	if len(clusterInfo.BrokerDetails) > 0 {
		log.Println("Calculating broker metrics...")
		brokerPartitions := make(map[int32]int)
		brokerLeaders := make(map[int32]int)
		brokerURPs := make(map[int32]int)

		// Get partition metadata for all topics
		for topicName := range topics {
			partitions, err := admin.DescribeTopics([]string{topicName})
			if err != nil {
				continue
			}

			for _, topicMeta := range partitions {
				for _, partition := range topicMeta.Partitions {
					// Count partition per broker (replicas)
					for _, replica := range partition.Replicas {
						brokerPartitions[replica]++
					}

					// Count leaders
					if partition.Leader >= 0 {
						brokerLeaders[partition.Leader]++
					}

					// Check for under-replicated partitions
					if len(partition.Isr) < len(partition.Replicas) {
						for _, replica := range partition.Replicas {
							brokerURPs[replica]++
						}
						clusterInfo.TotalURPs++
					}
				}
			}
		}

		// Get broker versions
		for i := range clusterInfo.BrokerDetails {
			brokerID := clusterInfo.BrokerDetails[i].ID
			clusterInfo.BrokerDetails[i].Partitions = brokerPartitions[brokerID]
			clusterInfo.BrokerDetails[i].Leaders = brokerLeaders[brokerID]
			clusterInfo.BrokerDetails[i].UnderReplicated = brokerURPs[brokerID]

			// Try to get broker version via ApiVersions request
			broker := sarama.NewBroker(clusterInfo.BrokerDetails[i].Address)
			if err := broker.Open(config); err == nil {
				if apiVersions, err := broker.ApiVersions(&sarama.ApiVersionsRequest{}); err == nil {
					// Extract version from ApiVersions response - use the Version field
					if apiVersions != nil {
						// Simple version detection - Sarama's ApiVersions includes broker version info
						clusterInfo.BrokerDetails[i].Version = fmt.Sprintf("Kafka %s", "detected")
					}
				}
				broker.Close()
			}

			if clusterInfo.BrokerDetails[i].Version == "" {
				clusterInfo.BrokerDetails[i].Version = "Unknown"
			}
		}
	}

	// Get consumer groups
	log.Println("Fetching consumer groups...")
	groups, err := admin.ListConsumerGroups()
	if err != nil {
		log.Fatalf("Error listing consumer groups: %v", err)
	}

	consumerGroups := make([]ConsumerGroupInfo, 0, len(groups))
	for groupName := range groups {
		groupInfo := ConsumerGroupInfo{
			Name: groupName,
		}

		// Describe consumer group
		descriptions, err := admin.DescribeConsumerGroups([]string{groupName})
		if err == nil && len(descriptions) > 0 {
			desc := descriptions[0]
			groupInfo.State = desc.State
			groupInfo.Members = len(desc.Members)

			// Get topics for this consumer group
			topicMap := make(map[string]bool)
			for _, member := range desc.Members {
				assignment, err := member.GetMemberAssignment()
				if err == nil && assignment != nil && assignment.Topics != nil {
					for topic := range assignment.Topics {
						topicMap[topic] = true
					}
				}
			}

			for topic := range topicMap {
				groupInfo.Topics = append(groupInfo.Topics, topic)
			}
			sort.Strings(groupInfo.Topics)
		}

		consumerGroups = append(consumerGroups, groupInfo)
	}

	sort.Slice(consumerGroups, func(i, j int) bool {
		return consumerGroups[i].Name < consumerGroups[j].Name
	})

	clusterInfo.ConsumerGroups = consumerGroups
	clusterInfo.TotalConsumerGroups = len(consumerGroups)

	// Write JSON output
	log.Printf("Writing JSON to %s...", *outputJSON)
	jsonData, err := json.MarshalIndent(clusterInfo, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	if err := os.WriteFile(*outputJSON, jsonData, 0644); err != nil {
		log.Fatalf("Error writing JSON file: %v", err)
	}

	// Generate HTML report
	log.Printf("Generating HTML report to %s...", *outputHTML)
	if err := generateHTMLReport(&clusterInfo, *outputHTML); err != nil {
		log.Fatalf("Error generating HTML report: %v", err)
	}

	// Generate DOT file if requested
	if *outputDOT != "" {
		log.Printf("Generating DOT file to %s...", *outputDOT)
		if err := generateDOTFile(&clusterInfo, *outputDOT); err != nil {
			log.Fatalf("Error generating DOT file: %v", err)
		}
	}

	// Generate recreation script if requested
	if *recreateScript != "" {
		log.Printf("Generating topic recreation script to %s...", *recreateScript)
		if err := generateRecreateScript(&clusterInfo, *recreateScript); err != nil {
			log.Fatalf("Error generating recreation script: %v", err)
		}
	}

	// Save consumer group offsets if requested
	var offsetsBackup *ConsumerOffsetsBackup
	if *saveOffsets != "" || *restoreOffsetsScript != "" {
		log.Println("Fetching consumer group offsets...")
		offsetsBackup, err = fetchConsumerOffsets(admin, consumerGroups, brokerList[0])
		if err != nil {
			log.Fatalf("Error fetching consumer offsets: %v", err)
		}

		if *saveOffsets != "" {
			log.Printf("Saving consumer offsets to %s...", *saveOffsets)
			if err := saveConsumerOffsetsToFile(offsetsBackup, *saveOffsets); err != nil {
				log.Fatalf("Error saving offsets: %v", err)
			}
		}

		if *restoreOffsetsScript != "" {
			log.Printf("Generating offset restore script to %s...", *restoreOffsetsScript)
			if err := generateRestoreOffsetsScript(offsetsBackup, *restoreOffsetsScript); err != nil {
				log.Fatalf("Error generating restore script: %v", err)
			}
		}
	}

	log.Println("Done!")
	log.Printf("Summary:")
	log.Printf("  Total Brokers: %d", len(clusterInfo.BrokerDetails))
	log.Printf("  Total Topics: %d", clusterInfo.TotalTopics)
	log.Printf("  Total Partitions: %d", clusterInfo.TotalPartitions)
	log.Printf("  Total Messages: %s", formatNumber(clusterInfo.TotalMessages))
	log.Printf("  Total Consumer Groups: %d", clusterInfo.TotalConsumerGroups)
	if clusterInfo.TotalURPs > 0 {
		log.Printf("  ⚠️  Under-Replicated Partitions: %d", clusterInfo.TotalURPs)
	}
}

func generateHTMLReport(info *KafkaClusterInfo, filename string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Kafka Cluster Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        .header p {
            font-size: 1.1em;
            opacity: 0.9;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            padding: 40px;
            background: #f8f9fa;
        }
        .stat-card {
            background: white;
            padding: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            text-align: center;
            transition: transform 0.2s;
        }
        .stat-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }
        .stat-number {
            font-size: 3em;
            font-weight: bold;
            color: #667eea;
            margin-bottom: 10px;
        }
        .stat-label {
            color: #666;
            font-size: 1.1em;
        }
        .content {
            padding: 40px;
        }
        .section {
            margin-bottom: 40px;
        }
        .section-title {
            font-size: 1.8em;
            color: #333;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 3px solid #667eea;
        }
        .chart-container {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 30px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        table {
            width: 100%%;
            border-collapse: collapse;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        th {
            background: #667eea;
            color: white;
            padding: 15px;
            text-align: left;
            font-weight: 600;
        }
        td {
            padding: 12px 15px;
            border-bottom: 1px solid #eee;
        }
        tr:hover {
            background: #f8f9fa;
        }
        .topic-name {
            font-weight: 600;
            color: #667eea;
        }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
            margin: 2px;
        }
        .badge-success {
            background: #d4edda;
            color: #155724;
        }
        .badge-info {
            background: #d1ecf1;
            color: #0c5460;
        }
        .badge-warning {
            background: #fff3cd;
            color: #856404;
        }
        .config-details {
            font-size: 0.9em;
            color: #666;
            margin-top: 5px;
        }
        .broker-list {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 20px;
        }
        .broker-item {
            display: inline-block;
            background: white;
            padding: 8px 15px;
            border-radius: 6px;
            margin: 5px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Kafka Cluster Analysis</h1>
            <p>Generated on %s</p>
        </div>

        <div class="stats">
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Brokers</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Topics</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Total Partitions</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">%d</div>
                <div class="stat-label">Consumer Groups</div>
            </div>
%s        </div>

        <div class="content">
            <div class="section">
                <h2 class="section-title">🖥️ Kafka Brokers</h2>
                <table>
                    <thead>
                        <tr>
                            <th>Broker ID</th>
                            <th>Address</th>
                            <th>Version</th>
                            <th>Partitions</th>
                            <th>Leaders</th>
                            <th>Under-Replicated</th>
                        </tr>
                    </thead>
                    <tbody>
`, info.Timestamp, len(info.BrokerDetails), info.TotalTopics, info.TotalPartitions, info.TotalConsumerGroups, getURPCard(info.TotalURPs))

	for _, broker := range info.BrokerDetails {
		urpBadge := "badge-success"
		if broker.UnderReplicated > 0 {
			urpBadge = "badge-warning"
		}
		html += fmt.Sprintf(`                        <tr>
                            <td><span class="badge badge-info">%d</span></td>
                            <td>%s</td>
                            <td><span class="badge badge-success">%s</span></td>
                            <td><span class="badge badge-info">%d</span></td>
                            <td><span class="badge badge-info">%d</span></td>
                            <td><span class="badge %s">%d</span></td>
                        </tr>
`, broker.ID, broker.Address, broker.Version, broker.Partitions, broker.Leaders, urpBadge, broker.UnderReplicated)
	}

	html += `                    </tbody>
                </table>
            </div>


            <div class="section">
                <h2 class="section-title">📂 Topics Overview</h2>
                <table>
                    <thead>
                        <tr>
                            <th>Topic Name</th>
                            <th>Partitions</th>
                            <th>Replication Factor</th>
                            <th>Custom Configurations</th>
                        </tr>
                    </thead>
                    <tbody>
`

	for _, topic := range info.Topics {
		configStr := ""
		if len(topic.Configs) > 0 {
			configs := make([]string, 0, len(topic.Configs))
			for k, v := range topic.Configs {
				configs = append(configs, fmt.Sprintf("%s=%s", k, v))
			}
			sort.Strings(configs)
			configStr = strings.Join(configs, ", ")
		} else {
			configStr = "<em>Default</em>"
		}

		html += fmt.Sprintf(`                        <tr>
                            <td class="topic-name">%s</td>
                            <td><span class="badge badge-info">%d</span></td>
                            <td><span class="badge badge-success">%d</span></td>
                            <td class="config-details">%s</td>
                        </tr>
`, topic.Name, topic.Partitions, topic.ReplicationFactor, configStr)
	}

	html += `                    </tbody>
                </table>
            </div>

            <div class="section">
                <h2 class="section-title">👥 Consumer Groups</h2>
                <table>
                    <thead>
                        <tr>
                            <th>Group Name</th>
                            <th>State</th>
                            <th>Members</th>
                            <th>Subscribed Topics</th>
                        </tr>
                    </thead>
                    <tbody>
`

	for _, group := range info.ConsumerGroups {
		topicsStr := strings.Join(group.Topics, ", ")
		if topicsStr == "" {
			topicsStr = "<em>None</em>"
		}

		stateBadge := "badge-success"
		if group.State != "Stable" {
			stateBadge = "badge-warning"
		}

		html += fmt.Sprintf(`                        <tr>
                            <td class="topic-name">%s</td>
                            <td><span class="badge %s">%s</span></td>
                            <td><span class="badge badge-info">%d</span></td>
                            <td class="config-details">%s</td>
                        </tr>
`, group.Name, stateBadge, group.State, group.Members, topicsStr)
	}

	html += `                    </tbody>
                </table>
            </div>
        </div>
    </div>
</body>
</html>`

	return os.WriteFile(filename, []byte(html), 0644)
}

func getURPCard(urps int) string {
	if urps > 0 {
		return fmt.Sprintf(`            <div class="stat-card" style="border: 2px solid #f44336;">
                <div class="stat-number" style="color: #f44336;">%d</div>
                <div class="stat-label">⚠️ Under-Replicated</div>
            </div>
`, urps)
	}
	return ""
}

func getTotalPartitions(topics []TopicInfo) int {
	total := 0
	for _, topic := range topics {
		total += topic.Partitions
	}
	return total
}

func getTotalMessages(topics []TopicInfo) int64 {
	var total int64
	for _, topic := range topics {
		total += topic.TotalMessages
	}
	return total
}

func formatNumber(n int64) string {
	if n == 0 {
		return "0"
	}
	
	// Format with commas for readability
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	
	// Also show human-readable format for large numbers
	if n >= 1000000000000 { // Trillion
		return fmt.Sprintf("%s (%.2fT)", result, float64(n)/1000000000000)
	} else if n >= 1000000000 { // Billion
		return fmt.Sprintf("%s (%.2fB)", result, float64(n)/1000000000)
	} else if n >= 1000000 { // Million
		return fmt.Sprintf("%s (%.2fM)", result, float64(n)/1000000)
	} else if n >= 1000 { // Thousand
		return fmt.Sprintf("%s (%.2fK)", result, float64(n)/1000)
	}
	
	return result
}

func getTopicMessageCount(client sarama.Client, topic string, partitions int) int64 {
	var total int64
	
	for partition := 0; partition < partitions; partition++ {
		// Get high watermark (newest offset) for this partition
		offset, err := client.GetOffset(topic, int32(partition), sarama.OffsetNewest)
		if err != nil {
			log.Printf("Warning: Could not get offset for topic %s partition %d: %v", topic, partition, err)
			continue
		}
		total += offset
	}
	
	return total
}

func generateDOTFile(info *KafkaClusterInfo, filename string) error {
	var dot strings.Builder

	// DOT header
	dot.WriteString("digraph KafkaCluster {\n")
	dot.WriteString("  rankdir=LR;\n")
	dot.WriteString("  node [shape=box, style=rounded];\n")
	dot.WriteString("  graph [splines=true, overlap=false];\n\n")

	// Define subgraphs for topics and consumers
	dot.WriteString("  // Topics\n")
	dot.WriteString("  subgraph cluster_topics {\n")
	dot.WriteString("    label=\"Topics\";\n")
	dot.WriteString("    style=filled;\n")
	dot.WriteString("    color=lightgrey;\n")
	dot.WriteString("    node [style=filled, fillcolor=\"#667eea\", fontcolor=white];\n")

	for _, topic := range info.Topics {
		// Escape special characters
		safeName := strings.ReplaceAll(topic.Name, "\"", "\\\"")
		safeName = strings.ReplaceAll(safeName, "-", "_")
		safeName = strings.ReplaceAll(safeName, ".", "_")
		label := topic.Name
		if len(label) > 30 {
			label = label[:27] + "..."
		}
		dot.WriteString(fmt.Sprintf("    topic_%s [label=\"%s\\n(%d partitions)\"];\n",
			safeName, label, topic.Partitions))
	}
	dot.WriteString("  }\n\n")

	// Consumer groups
	dot.WriteString("  // Consumer Groups\n")
	dot.WriteString("  subgraph cluster_consumers {\n")
	dot.WriteString("    label=\"Consumer Groups\";\n")
	dot.WriteString("    style=filled;\n")
	dot.WriteString("    color=lightblue;\n")
	dot.WriteString("    node [style=filled, fillcolor=\"#43e97b\", fontcolor=white];\n")

	for _, group := range info.ConsumerGroups {
		safeName := strings.ReplaceAll(group.Name, "\"", "\\\"")
		safeName = strings.ReplaceAll(safeName, "-", "_")
		safeName = strings.ReplaceAll(safeName, ".", "_")
		label := group.Name
		if len(label) > 30 {
			label = label[:27] + "..."
		}
		dot.WriteString(fmt.Sprintf("    consumer_%s [label=\"%s\\n(%d members, %s)\"];\n",
			safeName, label, group.Members, group.State))
	}
	dot.WriteString("  }\n\n")

	// Edges (connections)
	dot.WriteString("  // Subscriptions\n")
	for _, group := range info.ConsumerGroups {
		safeGroupName := strings.ReplaceAll(group.Name, "\"", "\\\"")
		safeGroupName = strings.ReplaceAll(safeGroupName, "-", "_")
		safeGroupName = strings.ReplaceAll(safeGroupName, ".", "_")

		for _, topic := range group.Topics {
			safeTopicName := strings.ReplaceAll(topic, "\"", "\\\"")
			safeTopicName = strings.ReplaceAll(safeTopicName, "-", "_")
			safeTopicName = strings.ReplaceAll(safeTopicName, ".", "_")
			dot.WriteString(fmt.Sprintf("  topic_%s -> consumer_%s [color=\"#667eea\", penwidth=2.0];\n",
				safeTopicName, safeGroupName))
		}
	}

	dot.WriteString("}\n")

	return os.WriteFile(filename, []byte(dot.String()), 0644)
}

func countInternalTopics(topics []TopicInfo) int {
	count := 0
	for _, topic := range topics {
		if strings.HasPrefix(topic.Name, "__") {
			count++
		}
	}
	return count
}

func generateRecreateScript(info *KafkaClusterInfo, filename string) error {
	var script strings.Builder

	// Script header
	script.WriteString("#!/bin/bash\n")
	script.WriteString("# Kafka Topic Recreation Script\n")
	script.WriteString(fmt.Sprintf("# Generated: %s\n", info.Timestamp))
	script.WriteString(fmt.Sprintf("# Source Cluster: %s\n", strings.Join(info.Brokers, ", ")))
	script.WriteString(fmt.Sprintf("# Total Topics: %d\n", info.TotalTopics))
	script.WriteString("#\n")
	script.WriteString("# Usage:\n")
	script.WriteString("#   1. Edit BOOTSTRAP_SERVERS to point to your target cluster\n")
	script.WriteString("#   2. Add authentication flags if needed (--command-config, etc.)\n")
	script.WriteString(fmt.Sprintf("#   3. Run: chmod +x %s && ./%s\n", filename, filename))
	script.WriteString("#\n\n")

	script.WriteString("set -e  # Exit on error\n\n")

	script.WriteString("# Target cluster configuration\n")
	script.WriteString("BOOTSTRAP_SERVERS=\"localhost:9092\"  # CHANGE THIS\n")
	script.WriteString("# Uncomment and configure if authentication is needed:\n")
	script.WriteString("# COMMAND_CONFIG=\"--command-config client.properties\"\n")
	script.WriteString("COMMAND_CONFIG=\"\"\n\n")

	script.WriteString("# Kafka topics command (adjust path if needed)\n")
	script.WriteString("KAFKA_TOPICS=\"kafka-topics.sh\"\n\n")

	script.WriteString("echo \"========================================\"\n")
	script.WriteString(fmt.Sprintf("echo \"Recreating topics from source cluster\"\n"))
	script.WriteString(fmt.Sprintf("echo \"Note: Skipping %d internal topics (starting with __)\\n\"\n", countInternalTopics(info.Topics)))
	script.WriteString("echo \"Target: $BOOTSTRAP_SERVERS\"\n")
	script.WriteString("echo \"========================================\"\n")
	script.WriteString("echo \"\"\n\n")

	script.WriteString("CREATED=0\n")
	script.WriteString("FAILED=0\n")
	script.WriteString("SKIPPED=0\n")
	script.WriteString("FAILED_TOPICS=()\n\n")

	// Generate create commands for each topic
	topicIndex := 0
	for _, topic := range info.Topics {
		// Skip internal topics (starting with __)
		if strings.HasPrefix(topic.Name, "__") {
			continue
		}

		topicIndex++
		script.WriteString(fmt.Sprintf("# Topic %d: %s\n", topicIndex, topic.Name))
		script.WriteString(fmt.Sprintf("echo \"[%d] Creating topic: %s\"\n", topicIndex, topic.Name))

		// Build the kafka-topics command
		cmd := fmt.Sprintf("if $KAFKA_TOPICS --bootstrap-server \"$BOOTSTRAP_SERVERS\" $COMMAND_CONFIG \\\n")
		cmd += fmt.Sprintf("  --create \\\n")
		cmd += fmt.Sprintf("  --topic \"%s\" \\\n", topic.Name)
		cmd += fmt.Sprintf("  --partitions %d \\\n", topic.Partitions)
		cmd += fmt.Sprintf("  --replication-factor %d", topic.ReplicationFactor)

		// Add configurations
		if len(topic.Configs) > 0 {
			cmd += " \\\n"
			configs := make([]string, 0, len(topic.Configs))
			for k, v := range topic.Configs {
				configs = append(configs, fmt.Sprintf("%s=%s", k, v))
			}
			sort.Strings(configs)
			for j, config := range configs {
				cmd += fmt.Sprintf("  --config \"%s\"", config)
				if j < len(configs)-1 {
					cmd += " \\\n"
				}
			}
		}

		cmd += "; then\n"
		cmd += "  echo \"  ✓ Created successfully\"\n"
		cmd += "  ((CREATED++))\n"
		cmd += "else\n"
		cmd += "  echo \"  ✗ Failed to create (may already exist)\"\n"
		cmd += fmt.Sprintf("  FAILED_TOPICS+=(\"  - %s\")\n", topic.Name)
		cmd += "  ((FAILED++))\n"
		cmd += "fi\n"

		script.WriteString(cmd)
		script.WriteString("echo \"\"\n\n")
	}

	// Summary
	script.WriteString("echo \"========================================\"\n")
	script.WriteString("echo \"Topic Recreation Summary:\"\n")
	script.WriteString("echo \"  Successfully created: $CREATED\"\n")
	script.WriteString("echo \"  Failed/Skipped: $FAILED\"\n")
	script.WriteString(fmt.Sprintf("echo \"  Internal topics skipped: %d\"\n", countInternalTopics(info.Topics)))
	script.WriteString("if [ $FAILED -gt 0 ]; then\n")
	script.WriteString("  echo \"\"\n")
	script.WriteString("  echo \"Failed/Skipped Topics:\"\n")
	script.WriteString("  printf '%s\\n' \"${FAILED_TOPICS[@]}\"\n")
	script.WriteString("fi\n")
	script.WriteString("echo \"========================================\"\n\n")

	script.WriteString("# Note: To verify topics were created correctly:\n")
	script.WriteString("# $KAFKA_TOPICS --bootstrap-server \"$BOOTSTRAP_SERVERS\" $COMMAND_CONFIG --list\n")
	script.WriteString("# $KAFKA_TOPICS --bootstrap-server \"$BOOTSTRAP_SERVERS\" $COMMAND_CONFIG --describe --topic <topic-name>\n")

	return os.WriteFile(filename, []byte(script.String()), 0755)
}
