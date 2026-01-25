# kmap v1.3.0 Release Notes

**Release Date**: January 25, 2026

## 🎉 New Feature: Topic Size Calculation

Calculate actual disk space used by Kafka topics across all brokers and partitions!

### What's New

**Topic Size Analysis** - Query real disk usage for topics
- Calculate total size per topic including all replicas
- Support for filtering specific topics
- Human-readable output (KiB, MiB, GiB, TiB)
- JSON export for automation
- Fast queries using Kafka's DescribeLogDirs API

### New Command-Line Flags

- `-topic-sizes` - Calculate and display topic sizes
- `-topic-sizes-output <file>` - Save topic sizes report to JSON file
- `-topic-list <topics>` - Comma-separated list of topics to check

### Usage Examples

```bash
# Check all topics
kmap -brokers localhost:9092 -topic-sizes

# Check specific topics
kmap -brokers localhost:9092 -topic-sizes -topic-list "orders,users,events"

# Save to JSON
kmap -brokers localhost:9092 -topic-sizes -topic-sizes-output sizes.json

# With AWS MSK authentication
kmap -brokers broker:9096 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username user \
  -sasl-password pass \
  -topic-sizes
```

### Sample Output

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

================================================================================
Summary:
  Total Topics: 3
  Total Partitions: 40
  Total Size: 2.71 TiB (2,976,589,178,334 bytes)
================================================================================
```

### Helper Scripts

- **calculate-topic-sizes.sh** - User-friendly wrapper with config file support
- **compare-topic-sizes.sh** - Compare two size reports to track growth over time
- **examples/topic-sizes-example.sh** - Complete usage examples

### Documentation

- **TOPIC_SIZES.md** - Comprehensive guide with use cases
- **QUICKSTART_TOPIC_SIZES.md** - Quick reference guide
- **TOPIC_SIZE_IMPLEMENTATION.md** - Technical implementation details

### Use Cases

✅ **Storage Capacity Planning** - Identify which topics consume the most storage  
✅ **Cost Optimization** - Find candidates for retention policy adjustments  
✅ **Migration Planning** - Estimate target cluster storage requirements  
✅ **Monitoring & Alerts** - Track storage growth over time  
✅ **Cleanup Decisions** - Identify topics to delete or compact  

### Comparison with kafka-log-dirs.sh

Instead of complex shell pipelines:
```bash
kafka-log-dirs.sh --bootstrap-server $BROKER --command-config $CONFIG \
  --topic-list $TOPIC --describe | grep -oP '(?<=size":)\d+' | \
  awk '{ sum += $1 } END { print sum }' | numfmt --to=iec-i --suffix=B
```

Simply use:
```bash
kmap -brokers $BROKER -topic-sizes -topic-list $TOPIC
```

### Technical Details

- **API Used**: Kafka DescribeLogDirs API via Sarama library
- **Performance**: Typically completes in seconds, even for large clusters
- **Accuracy**: Direct from broker log directories
- **Replication**: Sizes include all replicas (RF=3 means 3x data size)
- **Impact**: Read-only metadata query, no performance impact on cluster

## 📦 New Files

- `topic_sizes.go` - Core implementation
- `TOPIC_SIZES.md` - Comprehensive documentation
- `QUICKSTART_TOPIC_SIZES.md` - Quick start guide
- `TOPIC_SIZE_IMPLEMENTATION.md` - Technical details
- `calculate-topic-sizes.sh` - Helper wrapper script
- `compare-topic-sizes.sh` - Size comparison tool
- `examples/topic-sizes-example.sh` - Usage examples

## 🔧 Updated Files

- `main.go` - Integrated topic sizes functionality
- `README.md` - Added topic sizes feature documentation

## 🐛 Bug Fixes

None in this release.

## ⚡ Performance Improvements

- Efficient parallel querying of all brokers
- Optimized data aggregation across partitions

## 📚 Documentation

- Comprehensive documentation for topic size feature
- Updated main README with new feature
- Added multiple example scripts
- Quick start guide for common scenarios

## 🔄 Breaking Changes

None. This release is fully backward compatible.

## 📝 Upgrade Instructions

Simply replace the binary:

```bash
# Download new version
wget https://github.com/yourorg/kmap/releases/download/v1.3.0/kmap-<platform>

# Replace old binary
mv kmap-<platform> kmap
chmod +x kmap

# Verify
./kmap -version
```

Or rebuild from source:

```bash
git pull
make build
```

## 🙏 Acknowledgments

This release adds a highly requested feature that simplifies topic size analysis for Kafka cluster management and capacity planning.

## 📋 Full Changelog

- ✨ NEW: Topic size calculation feature (`-topic-sizes`)
- ✨ NEW: Topic filtering support (`-topic-list`)
- ✨ NEW: JSON export for topic sizes (`-topic-sizes-output`)
- ✨ NEW: Helper scripts for common workflows
- 📖 NEW: Comprehensive documentation and guides
- 🔧 UPDATED: Main README with new feature
- 🎯 IMPROVED: Overall cluster analysis capabilities

## 🔗 Links

- [GitHub Repository](https://github.com/yourorg/kmap)
- [Full Documentation](README.md)
- [Topic Sizes Guide](TOPIC_SIZES.md)
- [Quick Start](QUICKSTART_TOPIC_SIZES.md)

---

**Previous Release**: [v1.2.2](release-notes-v1.2.2.md)
