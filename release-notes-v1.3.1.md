# Release Notes - v1.3.1

**Release Date:** January 25, 2026

## 🎯 KRaft Mode Support

This release adds **full compatibility with Kafka KRaft mode** (ZooKeeper-free Kafka) for the topic-sizes feature.

## ✨ New Features

### KRaft Mode Compatibility
- **Hybrid implementation** for topic size calculation
  - Primary: Uses `kafka-log-dirs.sh` CLI tool (KRaft-compatible ✅)
  - Fallback: Uses Sarama API (ZooKeeper-compatible)
  - Automatic detection and intelligent fallback
  
- **kafka-log-dirs.sh integration**
  - Automatic discovery in common paths (`$KAFKA_HOME/bin`, `/usr/local/kafka/bin`, etc.)
  - JSON output parsing with partition-to-topic mapping
  - Full authentication support (SASL, SCRAM, TLS, mTLS)
  - Temporary config file generation for secure connections

### Files Added
- `topic_sizes_kafka_cli.go` - CLI-based topic size calculation
- `KRAFT_COMPATIBILITY.md` - Comprehensive KRaft mode documentation

## 🔧 Improvements

- **No more EOF errors** on KRaft-mode clusters
- **Zero configuration** - Works out of the box
- **Same commands** - No workflow changes needed
- **Better error messages** - Clear indication of which method was used

## 🐛 Bug Fixes

- Fixed Sarama DescribeLogDirs compatibility issues with KRaft mode
- Improved broker connection handling and error logging
- Better JSON parsing for kafka-log-dirs.sh output

## 📊 Testing

Verified on:
- ✅ **KRaft mode**: localhost:29092 (3 brokers)
- ✅ **All topic filtering**: Specific topics, all topics
- ✅ **JSON export**: Correct output format
- ✅ **Authentication**: SASL/SCRAM support validated

## 📖 Documentation

- New comprehensive KRaft compatibility guide
- Updated README.md with KRaft support highlight
- Enhanced troubleshooting documentation

## 🚀 Usage

Same commands work for both KRaft and ZooKeeper modes:

```bash
# All topics
kmap -brokers localhost:9092 -topic-sizes

# Specific topics
kmap -brokers localhost:9092 -topic-sizes -topic-list "orders,users"

# With authentication
kmap -brokers broker:9092 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username user \
  -sasl-password pass \
  -topic-sizes
```

## 📦 Binaries

Available for:
- Linux (amd64, arm64)
- macOS (Intel, Apple Silicon)
- Windows (amd64)

## 🔗 Links

- [KRAFT_COMPATIBILITY.md](KRAFT_COMPATIBILITY.md) - KRaft mode guide
- [TOPIC_SIZES.md](TOPIC_SIZES.md) - Topic sizes documentation
- [QUICKSTART_TOPIC_SIZES.md](QUICKSTART_TOPIC_SIZES.md) - Quick start guide

## ⬆️ Upgrade Notes

This is a drop-in replacement for v1.3.0. No configuration changes needed.

If you're using KRaft mode:
- Ensure kafka-log-dirs.sh is in your PATH for best results
- Or set `KAFKA_HOME` environment variable

If kafka-log-dirs.sh is not available, kmap will automatically use the Sarama API (works with ZooKeeper-based Kafka).

## 🙏 Credits

Thanks to the Apache Kafka team for the excellent kafka-log-dirs.sh tool that makes KRaft compatibility possible.

---

**Full Changelog:** v1.3.0...v1.3.1
