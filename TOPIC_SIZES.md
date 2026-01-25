# Topic Size Calculation

Calculate actual disk usage for Kafka topics across all brokers and partitions.

## Overview

The topic size feature queries all Kafka brokers using the `DescribeLogDirs` API to calculate the total disk space used by each topic. This includes all replicas across all brokers and partitions.

## Usage

### Basic Usage

Check all topics in the cluster:
```bash
kmap -brokers localhost:9092 -topic-sizes
```

### Check Specific Topics

Filter to specific topics only:
```bash
kmap -brokers localhost:9092 -topic-sizes -topic-list "orders,users,events"
```

### Save to JSON File

Export the report to a JSON file for further processing:
```bash
kmap -brokers localhost:9092 -topic-sizes -topic-sizes-output topic-sizes.json
```

### With Authentication

#### AWS MSK with SCRAM-SHA-512
```bash
kmap -brokers b-1.cluster.kafka.us-east-1.amazonaws.com:9096 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username myuser \
  -sasl-password mypassword \
  -topic-sizes
```

#### Confluent Cloud
```bash
kmap -brokers pkc-xxxxx.us-east-1.aws.confluent.cloud:9092 \
  -security-protocol SASL_SSL \
  -sasl-mechanism PLAIN \
  -sasl-username <API-KEY> \
  -sasl-password <API-SECRET> \
  -topic-sizes
```

#### With mTLS
```bash
kmap -brokers broker:9093 \
  -security-protocol SSL \
  -tls-client-cert client.crt \
  -tls-client-key client.key \
  -tls-ca-cert ca.crt \
  -topic-sizes
```

## Output Format

### Console Output

The tool displays a formatted table with:
- **Topic**: Topic name
- **Partitions**: Number of partitions
- **Total Size**: Human-readable size (GiB, TiB, etc.)
- **Size (Bytes)**: Exact byte count

Example:
```
================================================================================
Kafka Topic Sizes Report
Generated: 2026-01-25T10:30:00Z
Cluster: broker1:9092,broker2:9092,broker3:9092
================================================================================

TOPIC                                   PARTITIONS   TOTAL SIZE    SIZE (BYTES)
----------------------------------------  ----------   ------------  ---------------
aws.traffic.cdc.shipping-v1             20           2.14 TiB      2,353,578,067,422
orders.events                           12           456.78 GiB    490,463,289,344
user.activity                           8            123.45 GiB    132,547,821,568
payment.transactions                    6            89.12 GiB     95,698,234,112

================================================================================
Summary:
  Total Topics: 4
  Total Partitions: 46
  Total Size: 2.84 TiB (3,072,287,392,446 bytes)
================================================================================
```

### JSON Output

When using `-topic-sizes-output`, the data is saved as:

```json
{
  "timestamp": "2026-01-25T10:30:00Z",
  "cluster": "broker1:9092,broker2:9092,broker3:9092",
  "topics": [
    {
      "topic": "aws.traffic.cdc.shipping-v1",
      "partitions": 20,
      "total_size_bytes": 2353578067422,
      "total_size_human": "2.14 TiB"
    },
    {
      "topic": "orders.events",
      "partitions": 12,
      "total_size_bytes": 490463289344,
      "total_size_human": "456.78 GiB"
    }
  ],
  "total_size_bytes": 3072287392446,
  "total_size_human": "2.84 TiB",
  "total_topics": 4,
  "total_partitions": 46
}
```

## Important Notes

### Replication Factor

The sizes shown **include replication**. For example:
- Topic with 10 GiB logical data and RF=3 will show as 30 GiB
- This represents the actual disk space used across all brokers
- To get logical data size, divide by the replication factor

### Calculation Method

The tool:
1. Connects to all brokers in the cluster
2. Queries each broker's log directories via `DescribeLogDirs` API
3. Sums the partition sizes across all brokers
4. Accounts for all replicas (leader + followers)

### Performance

- Query time depends on number of brokers and topics
- Typically completes in seconds for most clusters
- For very large clusters (1000+ topics), may take 10-30 seconds
- No impact on cluster performance (read-only metadata query)

## Use Cases

### 1. Storage Capacity Planning
Identify which topics consume the most storage:
```bash
kmap -brokers kafka:9092 -topic-sizes | head -20
```

### 2. Cost Optimization
Find candidates for retention policy adjustments:
```bash
kmap -brokers kafka:9092 -topic-sizes -topic-sizes-output sizes.json
# Process JSON to find topics > 1 TiB
jq '.topics[] | select(.total_size_bytes > 1099511627776)' sizes.json
```

### 3. Migration Planning
Estimate target cluster storage requirements:
```bash
# Check source cluster
kmap -brokers source:9092 -topic-sizes -topic-sizes-output source-sizes.json

# Review total size needed for target
jq '.total_size_human, .total_size_bytes' source-sizes.json
```

### 4. Retention Policy Analysis
Compare topic sizes with retention settings:
```bash
# Get sizes
kmap -brokers kafka:9092 -topic-sizes -topic-sizes-output sizes.json

# Get topic configs (from main kmap output)
kmap -brokers kafka:9092 -output cluster-info.json

# Compare retention.bytes vs actual size
```

### 5. Monitoring & Alerts
Track storage growth over time:
```bash
#!/bin/bash
# Save daily snapshots
DATE=$(date +%Y%m%d)
kmap -brokers kafka:9092 -topic-sizes -topic-sizes-output "sizes-$DATE.json"

# Alert if any topic > 1 TiB
BIG_TOPICS=$(jq -r '.topics[] | select(.total_size_bytes > 1099511627776) | .topic' "sizes-$DATE.json")
if [ -n "$BIG_TOPICS" ]; then
  echo "Alert: Large topics found:"
  echo "$BIG_TOPICS"
fi
```

### 6. Cleanup Decisions
Identify topics to delete or compact:
```bash
# Find smallest topics (candidates for deletion if unused)
kmap -brokers kafka:9092 -topic-sizes -topic-sizes-output sizes.json
jq '.topics | sort_by(.total_size_bytes) | .[:10]' sizes.json
```

## Comparison with kafka-log-dirs.sh

The kmap topic sizes feature is equivalent to:
```bash
kafka-log-dirs.sh --bootstrap-server broker:9092 \
  --command-config client.properties \
  --topic-list topic-name \
  --describe | \
  grep -oP '(?<=size":)\d+' | \
  awk '{ sum += $1 } END { print sum }' | \
  numfmt --to=iec-i --suffix=B
```

**Advantages of kmap:**
- ✅ No need for shell pipelines and text processing
- ✅ Handles multiple topics automatically
- ✅ Shows all topics by default
- ✅ Formatted table output
- ✅ JSON export for automation
- ✅ Sorted by size (largest first)
- ✅ Summary statistics included
- ✅ Single binary, no dependencies

## Troubleshooting

### Error: "Could not connect to broker"
Check network connectivity and authentication:
```bash
# Test basic connectivity
telnet broker-hostname 9092

# Verify credentials
kmap -brokers broker:9092 -version  # Should connect successfully first
```

### Error: "Error querying log dirs"
Some Kafka versions or configurations may restrict DescribeLogDirs API:
- Ensure you have appropriate ACLs/permissions
- Check Kafka version supports the API (Kafka 1.0+)
- Verify broker configuration allows log directory queries

### No topics returned
If using `-topic-list`, ensure topic names are exact (case-sensitive):
```bash
# List all topics first
kafka-topics.sh --bootstrap-server broker:9092 --list

# Then query specific topics
kmap -brokers broker:9092 -topic-sizes -topic-list "exact-topic-name"
```

### Size seems too large
Remember: sizes include replication factor.
```
Displayed Size = Logical Data Size × Replication Factor
```

For a topic with:
- 100 GiB logical data
- Replication factor = 3
- Displayed size = 300 GiB

## Integration Examples

### CI/CD Pipeline
```yaml
# GitHub Actions example
- name: Check Kafka Storage
  run: |
    ./kmap -brokers $KAFKA_BROKERS \
      -security-protocol SASL_SSL \
      -sasl-username ${{ secrets.KAFKA_USER }} \
      -sasl-password ${{ secrets.KAFKA_PASS }} \
      -topic-sizes -topic-sizes-output sizes.json
    
    # Fail if total size exceeds threshold
    TOTAL_GB=$(jq -r '.total_size_bytes / 1073741824 | floor' sizes.json)
    if [ $TOTAL_GB -gt 5000 ]; then
      echo "Storage limit exceeded: ${TOTAL_GB} GB"
      exit 1
    fi
```

### Prometheus Monitoring
```bash
#!/bin/bash
# Export metrics for Prometheus
kmap -brokers kafka:9092 -topic-sizes -topic-sizes-output /tmp/sizes.json

# Convert to Prometheus format
jq -r '.topics[] | "kafka_topic_size_bytes{topic=\"\(.topic)\"} \(.total_size_bytes)"' \
  /tmp/sizes.json > /var/lib/node_exporter/kafka_sizes.prom
```

### Slack Notifications
```bash
#!/bin/bash
# Daily storage report
REPORT=$(kmap -brokers kafka:9092 -topic-sizes -topic-sizes-output sizes.json)
SUMMARY=$(jq -r '"Topics: \(.total_topics), Total: \(.total_size_human)"' sizes.json)

curl -X POST $SLACK_WEBHOOK \
  -H 'Content-Type: application/json' \
  -d "{\"text\": \"Kafka Storage Report: $SUMMARY\"}"
```

## See Also

- [README.md](README.md) - Main documentation
- [AUTH.md](AUTH.md) - Authentication examples
- [examples/topic-sizes-example.sh](examples/topic-sizes-example.sh) - Usage examples
