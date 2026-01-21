# kmap - Kafka Cluster Inspector

Analyze Kafka clusters and export to JSON/HTML/DOT formats. Single binary, all authentication methods supported.

## Features

- �️ **Broker metrics** - Version, partition/leader distribution, under-replicated partitions (URPs)
- 📊 **Topic discovery** - Partitions, replication, configs
- 👥 **Consumer groups** - Members, subscriptions, state  
- 📁 **JSON export** - Recreate topics elsewhere
- 📈 **HTML reports** - Clean tables with broker and topic details
- 🗺️ **Graphviz DOT** - High-quality visualizations for large clusters
- 🔄 **Topic recreation script** - Generate executable scripts to recreate all topics on another cluster
- 💾 **Consumer offset backup** - Save and restore consumer group positions for migration/DR
- 📊 **Message counting** - Total messages per topic and cluster-wide for migration validation
- 🔍 **Cluster comparison** - Compare source/target clusters to validate migrations
- 🔐 **All auth methods** - SASL/PLAIN/SCRAM, TLS, mTLS
- 🚀 **Single binary** - No dependencies

## Quick Start

```bash
# Build
make build

# Run
kmap -brokers localhost:9092

# With auth
kmap -brokers broker:9093 \
  -security-protocol SASL_SSL \
  -sasl-username <API-KEY> \
  -sasl-password <SECRET>

# Export DOT for large clusters
kmap -brokers kafka:9092 -dot topology.dot
```

## Command-line Options

```
-brokers string          Kafka brokers (default "localhost:9092")
-output string           JSON file (default "kafka-cluster-info.json")
-html string             HTML report (default "kafka-cluster-report.html")
-dot string              Graphviz DOT file (optional)
-recreate-script string  Shell script to recreate topics (optional)
-save-offsets string     Save consumer group offsets to JSON file
-restore-offsets-script string  Generate script to restore consumer offsets
-version                 Show version

Authentication:
-security-protocol       SASL_SSL, SASL_PLAINTEXT, SSL, or empty
-sasl-mechanism          PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
-sasl-username          
-sasl-password

TLS:
-tls-ca-cert            CA certificate
-tls-client-cert        Client cert (mTLS)
-tls-client-key         Client key (mTLS)
-tls-skip-verify        Skip verification (dev only)
```

## Authentication

kmap supports all major authentication methods. See [AUTH.md](AUTH.md) for complete examples.

**Quick examples:**
```bash
# No auth (local)
kmap -brokers localhost:9092

# Confluent Cloud
kmap -brokers xxx.confluent.cloud:9092 -security-protocol SASL_SSL \
  -sasl-username <API-KEY> -sasl-password <API-SECRET>

# AWS MSK with SCRAM
kmap -brokers b-1.cluster.kafka.aws.com:9096 -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 -sasl-username <USER> -sasl-password <PASS>
```

## Output Formats

### JSON
Structured data for automation/backup. Includes:
- **Broker details** - ID, address, version, partitions, leaders, URPs
- **Topics** - Name, partitions, replication, configs
- **Consumer groups** - Name, state, members, subscriptions
- **Cluster summary** - Total counts, URP warnings

### Recreation Script
Generate executable bash script to recreate all topics with exact configurations:

```bash
# Generate script
kmap -brokers kafka:9092 -recreate-script recreate-topics.sh

# Edit target cluster
nano recreate-topics.sh  # Change BOOTSTRAP_SERVERS

# Run on target cluster
chmod +x recreate-topics.sh
./recreate-topics.sh
```

The script includes:
- All topic names, partitions, replication factors
- All custom configurations (retention, compression, etc.)
- Error handling and progress tracking
- Summary of created/failed topics

**Perfect for:**
- Cluster migration
- DR environment setup
- Dev/Test environment creation
- Cluster replication

**See [RECREATE_TOPICS.md](RECREATE_TOPICS.md) for detailed documentation.**

### Consumer Offset Backup & Restore
Save consumer group positions and generate restore script:

```bash
# Backup offsets
kmap -brokers kafka:9092 \
  -save-offsets consumer-offsets.json \
  -restore-offsets-script restore-offsets.sh

# Edit target cluster in script
nano restore-offsets.sh  # Change BOOTSTRAP_SERVERS

# Restore on target cluster
chmod +x restore-offsets.sh
./restore-offsets.sh
```

The backup includes:
- All consumer groups with committed offsets
- Per-topic, per-partition offset positions
- Timestamp of backup
- Automatic filtering of empty groups

The restore script:
- Uses `kafka-consumer-groups.sh --reset-offsets`
- Supports authentication via `--command-config`
- Tracks success/failure for each group
- Provides detailed progress and summary

**Perfect for:**
- Cluster migration (preserve consumer progress)
- DR environment sync
- Rolling back consumer positions
- Testing with consistent offsets

### Migration Validation
Compare message counts between clusters to validate migrations:

```bash
# Capture source cluster
./kmap -brokers source:9092 -output source-cluster.json

# Capture target cluster
./kmap -brokers target:9092 -output target-cluster.json

# Compare
./compare-clusters.sh source-cluster.json target-cluster.json
```

The comparison shows:
- **Metrics**: Topics, partitions, messages, brokers comparison
- **Replication %**: Message count match percentage
- **Missing/Extra Topics**: Topics in one cluster but not the other
- **Top Topics**: Largest topics by message count comparison
- **Status**: ✅ Success / ⚠️ Warning / ❌ Issues

**Perfect for:**
- Validating cluster migrations
- Monitoring replication progress
- Detecting data loss
- Capacity planning

**Note**: Message counting adds ~1-3 minutes for 300+ topics
- Disaster recovery
- Environment synchronization (dev/test/prod)
- Reprocessing scenarios
- Consumer group rollback

**See [CONSUMER_OFFSETS.md](CONSUMER_OFFSETS.md) for detailed documentation.**

### Manual topic recreation
Extract topics from JSON:

```bash
jq -r '.topics[] | "\(.name) \(.partitions) \(.replication_factor)"' kafka-cluster-info.json | \
while read name parts repl; do
  kafka-topics.sh --create --bootstrap-server new-kafka:9092 \
    --topic "$name" --partitions "$parts" --replication-factor "$repl"
done
```

### HTML
Self-contained report with:
- **Summary dashboard** - Brokers, topics, partitions, consumer groups
- **⚠️ URP alerts** - Highlighted under-replicated partition warnings
- **Broker table** - ID, address, version, partition count, leader count, URPs
- **Topic details** - Full configuration listing
- **Consumer groups** - State, members, subscriptions

Clean, professional tables optimized for viewing and printing.

### DOT (Graphviz)
For large clusters (100+ topics) or external tooling:

```bash
# Generate
kmap -brokers kafka:9092 -dot topology.dot
```

See [GRAPHVIZ_GUIDE.md](GRAPHVIZ_GUIDE.md) for rendering options and visualization techniques.

## Deployment

```bash
make build              # All platforms
make linux-amd64        # Specific
go build -o kmap        # Local
make clean
```

Requires: Go 1.21+

Binaries: `kmap-linux-{amd64,arm64}`, `kmap-darwin-{amd64,arm64}`, `kmap-windows-amd64.exe`

## Troubleshooting

**Connection timeout**
- Check port: 9092 (plain), 9093 (SSL)
- Test: `telnet broker 9092`
- Check firewall

**Auth failed**
- Verify credentials
- Check mechanism matches cluster
- Use port 9093 for SASL_SSL

**Permission denied**
- Check ACLs: `kafka-acls.sh --list --bootstrap-server broker:9092`

**Large cluster visualization**
Use DOT instead of HTML:
```bash
kmap -brokers kafka:9092 -dot topology.dot
dot -Tsvg topology.dot -o topology.svg
```

## Building

```bash
make build              # All platforms
make linux-amd64        # Specific platform
go build -o kmap        # Local development
```

## Security Best Practices

- ✅ Use `SASL_SSL` in production
- ✅ Store credentials in environment variables or secure vaults
- ✅ Prefer SCRAM over PLAIN authentication
- ✅ Verify TLS certificates (avoid `-tls-skip-verify` in production)
- ✅ Use mTLS for highest security requirements

## License

MIT

---

**GitHub:** https://github.com/mordp1/kmap  
**Documentation:** [README.md](README.md) • [AUTH.md](AUTH.md) • [RECREATE_TOPICS.md](RECREATE_TOPICS.md) • [CONSUMER_OFFSETS.md](CONSUMER_OFFSETS.md) • [GRAPHVIZ_GUIDE.md](GRAPHVIZ_GUIDE.md)
