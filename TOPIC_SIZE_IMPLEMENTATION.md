# Topic Size Feature - Implementation Summary

## What Was Implemented

A comprehensive topic size calculation feature for kmap that queries Kafka brokers to determine the actual disk space used by topics.

## New Features

### 1. Core Functionality (`topic_sizes.go`)
- **getTopicSizes()**: Queries all brokers using DescribeLogDirs API
- **printTopicSizes()**: Formatted console output with tables
- **saveTopicSizesJSON()**: Export to JSON format
- **formatBytes()**: Human-readable size formatting (KiB, MiB, GiB, TiB)

### 2. Command-Line Flags
- `-topic-sizes`: Enable topic size calculation mode
- `-topic-sizes-output <file>`: Save report to JSON file
- `-topic-list <topics>`: Filter to specific topics (comma-separated)

### 3. Helper Scripts
- **calculate-topic-sizes.sh**: Wrapper script with friendly interface
- **compare-topic-sizes.sh**: Compare two size reports (track growth)
- **examples/topic-sizes-example.sh**: Usage examples

### 4. Documentation
- **TOPIC_SIZES.md**: Comprehensive guide with examples
- **README.md**: Updated with feature description
- Inline code documentation

## Usage Examples

### Basic Usage
```bash
# All topics
kmap -brokers localhost:9092 -topic-sizes

# Specific topics
kmap -brokers localhost:9092 -topic-sizes -topic-list "orders,users"

# Save to file
kmap -brokers localhost:9092 -topic-sizes -topic-sizes-output report.json
```

### With Authentication (AWS MSK)
```bash
kmap -brokers broker:9096 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username user \
  -sasl-password pass \
  -topic-sizes
```

### Using Helper Scripts
```bash
# Simple wrapper
./calculate-topic-sizes.sh -b localhost:9092 -t "topic1,topic2"

# Compare two reports
./compare-topic-sizes.sh before.json after.json
```

## Output Format

### Console
```
================================================================================
Kafka Topic Sizes Report
Generated: 2026-01-25T10:30:00Z
Cluster: broker1:9092,broker2:9092
================================================================================

TOPIC                    PARTITIONS   TOTAL SIZE    SIZE (BYTES)
----------------------   ----------   ------------  ---------------
large-topic              20           2.14 TiB      2,353,578,067,422
medium-topic             12           456.78 GiB    490,463,289,344

================================================================================
Summary:
  Total Topics: 2
  Total Partitions: 32
  Total Size: 2.57 TiB (2,844,041,356,766 bytes)
================================================================================
```

### JSON
```json
{
  "timestamp": "2026-01-25T10:30:00Z",
  "cluster": "broker1:9092,broker2:9092",
  "topics": [
    {
      "topic": "large-topic",
      "partitions": 20,
      "total_size_bytes": 2353578067422,
      "total_size_human": "2.14 TiB"
    }
  ],
  "total_size_bytes": 2844041356766,
  "total_size_human": "2.57 TiB",
  "total_topics": 2,
  "total_partitions": 32
}
```

## Technical Details

### How It Works
1. Connects to Kafka cluster
2. Queries all brokers via `DescribeLogDirs` API
3. Iterates through all log directories on each broker
4. Sums partition sizes across all replicas
5. Aggregates data by topic
6. Formats and displays results

### API Used
- **Sarama**: `broker.DescribeLogDirs()`
- Reads actual log segment sizes from disk
- Includes all replicas (leader + followers)

### Key Considerations
- **Replication**: Sizes include all replicas (RF=3 means 3x data)
- **Read-only**: No impact on cluster performance
- **Fast**: Typically completes in seconds
- **Accurate**: Direct from broker log directories

## Comparison with kafka-log-dirs.sh

### Traditional Approach
```bash
kafka-log-dirs.sh --bootstrap-server $BROKER \
  --command-config $CONFIG \
  --topic-list $TOPIC \
  --describe | \
  grep -oP '(?<=size":)\d+' | \
  awk '{ sum += $1 } END { print sum }' | \
  numfmt --to=iec-i --suffix=B
```

### kmap Approach
```bash
kmap -brokers $BROKER -topic-sizes -topic-list $TOPIC
```

### Advantages
- ✅ No shell pipelines required
- ✅ Works for all topics at once
- ✅ Formatted table output
- ✅ JSON export included
- ✅ Sorted by size
- ✅ Summary statistics
- ✅ Single command

## Use Cases

1. **Capacity Planning**: Identify storage requirements
2. **Cost Optimization**: Find largest topics for retention tuning
3. **Migration Planning**: Estimate target cluster storage needs
4. **Monitoring**: Track growth over time
5. **Cleanup**: Identify topics to delete or compact
6. **Troubleshooting**: Find unexpected storage usage

## Files Modified/Created

### New Files
- `kmap/topic_sizes.go` - Core implementation
- `kmap/TOPIC_SIZES.md` - Detailed documentation
- `kmap/calculate-topic-sizes.sh` - Helper script
- `kmap/compare-topic-sizes.sh` - Comparison tool
- `kmap/examples/topic-sizes-example.sh` - Examples

### Modified Files
- `kmap/main.go` - Added flags and integration
- `kmap/README.md` - Updated with feature description

## Testing

Build and test:
```bash
cd kmap
go build -o kmap
./kmap -h | grep topic-sizes  # Verify flags present
```

## Future Enhancements (Optional)

Potential improvements:
- Add size per partition breakdown
- Include offset lag in size calculations
- Historical trending (store multiple snapshots)
- Alert thresholds
- Integration with monitoring systems
- CSV export format
- Filtering by size (e.g., topics > 1TB)

## References

- [TOPIC_SIZES.md](TOPIC_SIZES.md) - Full documentation
- [README.md](README.md) - Main project docs
- Kafka DescribeLogDirs API documentation
- Sarama library documentation
