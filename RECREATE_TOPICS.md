# Topic Recreation Script

The `-recreate-script` feature generates an executable bash script that recreates all topics with their exact configurations on a target cluster.

## Example Script

```bash
#!/bin/bash
# Kafka Topic Recreation Script
# Generated: 2026-01-20T12:55:56Z
# Source Cluster: pkc-z9doz.eu-west-1.aws.confluent.cloud:9092
# Total Topics: 326
#
# Usage:
#   1. Edit BOOTSTRAP_SERVERS to point to your target cluster
#   2. Add authentication flags if needed (--command-config, etc.)
#   3. Run: chmod +x recreate-topics.sh && ./recreate-topics.sh
#

set -e  # Exit on error

# Target cluster configuration
BOOTSTRAP_SERVERS="localhost:9092"  # CHANGE THIS
# Uncomment and configure if authentication is needed:
# COMMAND_CONFIG="--command-config client.properties"
COMMAND_CONFIG=""

# Kafka topics command (adjust path if needed)
KAFKA_TOPICS="kafka-topics.sh"

echo "========================================"
echo "Recreating 326 topics from source cluster"
echo "Target: $BOOTSTRAP_SERVERS"
echo "========================================"
echo ""

CREATED=0
FAILED=0

# Topic 1/326: acknowledgement
echo "[1/326] Creating topic: acknowledgement"
if $KAFKA_TOPICS --bootstrap-server "$BOOTSTRAP_SERVERS" $COMMAND_CONFIG \
  --create \
  --topic "acknowledgement" \
  --partitions 10 \
  --replication-factor 3 \
  --config "cleanup.policy=delete" \
  --config "max.message.bytes=2097164" \
  --config "message.timestamp.after.max.ms=9223372036854775807" \
  --config "message.timestamp.before.max.ms=9223372036854775807" \
  --config "min.insync.replicas=2" \
  --config "retention.ms=2592000000" \
  --config "segment.bytes=104857600"; then
  echo "  ✓ Created successfully"
  ((CREATED++))
else
  echo "  ✗ Failed to create (may already exist)"
  ((FAILED++))
fi
echo ""

# Topic 2/326: orders
echo "[2/326] Creating topic: orders"
if $KAFKA_TOPICS --bootstrap-server "$BOOTSTRAP_SERVERS" $COMMAND_CONFIG \
  --create \
  --topic "orders" \
  --partitions 12 \
  --replication-factor 3 \
  --config "compression.type=snappy" \
  --config "retention.ms=604800000"; then
  echo "  ✓ Created successfully"
  ((CREATED++))
else
  echo "  ✗ Failed to create (may already exist)"
  ((FAILED++))
fi
echo ""

# ... (324 more topics)

echo "========================================"
echo "Topic Recreation Summary:"
echo "  Successfully created: $CREATED"
echo "  Failed/Skipped: $FAILED"
echo "========================================"

# Note: To verify topics were created correctly:
# $KAFKA_TOPICS --bootstrap-server "$BOOTSTRAP_SERVERS" $COMMAND_CONFIG --list
# $KAFKA_TOPICS --bootstrap-server "$BOOTSTRAP_SERVERS" $COMMAND_CONFIG --describe --topic <topic-name>
```

## Usage

### 1. Generate Script from Source Cluster

```bash
kmap -brokers source-kafka:9092 \
  -security-protocol SASL_SSL \
  -sasl-username admin \
  -sasl-password secret \
  -recreate-script recreate-topics.sh
```

### 2. Configure Target Cluster

Edit the script to point to your target cluster:

```bash
nano recreate-topics.sh
```

Change these variables:
```bash
BOOTSTRAP_SERVERS="target-kafka:9092"  # Your target cluster
```

### 3. Add Authentication (if needed)

Create a `client.properties` file:

```properties
# client.properties
security.protocol=SASL_SSL
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required \
  username="admin" \
  password="secret";
```

Then uncomment in script:
```bash
COMMAND_CONFIG="--command-config client.properties"
```

### 4. Run the Script

```bash
chmod +x recreate-topics.sh
./recreate-topics.sh
```

## Output Example

```
========================================
Recreating 326 topics from source cluster
Target: new-kafka.company.com:9092
========================================

[1/326] Creating topic: acknowledgement
  ✓ Created successfully

[2/326] Creating topic: orders
  ✓ Created successfully

[3/326] Creating topic: payments
  ✗ Failed to create (may already exist)

...

========================================
Topic Recreation Summary:
  Successfully created: 324
  Failed/Skipped: 2
========================================
```

## Features

### All Configurations Preserved
- Partition count
- Replication factor
- Custom configs: `retention.ms`, `compression.type`, `min.insync.replicas`, etc.
- All non-default settings included

### Error Handling
- Continues on error (doesn't stop if topic exists)
- Tracks success/failure count
- Exit on critical errors with `set -e`

### Progress Tracking
- Shows current progress: `[1/326]`
- Real-time status: ✓ success, ✗ failure
- Summary at the end

### Flexible Authentication
- Easy to configure for SASL, SSL, mTLS
- Works with `--command-config` for complex auth
- Supports Confluent Cloud, AWS MSK, Azure Event Hubs

## Use Cases

### 1. Cluster Migration
```bash
# Source cluster
kmap -brokers old-kafka:9092 -recreate-script migrate.sh

# Edit target
sed -i 's/localhost:9092/new-kafka:9092/' migrate.sh

# Migrate
./migrate.sh
```

### 2. DR Environment Setup
```bash
# Production
kmap -brokers prod-kafka:9092 -recreate-script prod-topics.sh

# DR
sed 's/localhost:9092/dr-kafka:9092/' prod-topics.sh > dr-setup.sh
./dr-setup.sh
```

### 3. Dev/Test Environment
```bash
# From production
kmap -brokers prod:9092 -recreate-script prod.sh

# To dev (adjust replication for smaller cluster)
sed 's/replication-factor 3/replication-factor 1/' prod.sh > dev.sh
./dev.sh
```

### 4. Backup & Restore
```bash
# Daily backup
DATE=$(date +%Y-%m-%d)
kmap -brokers kafka:9092 -recreate-script backup-$DATE.sh

# Restore if needed
chmod +x backup-2026-01-15.sh
./backup-2026-01-15.sh
```

## Advanced Tips

### Filter Specific Topics
```bash
# Only recreate topics matching pattern
grep -A 15 "orders\|payments" recreate-topics.sh > selected.sh
chmod +x selected.sh
./selected.sh
```

### Modify Before Creation
```bash
# Reduce retention for dev
sed -i 's/retention.ms=[0-9]*/retention.ms=86400000/' recreate-topics.sh

# Reduce partitions
sed -i 's/--partitions [0-9]*/--partitions 1/' recreate-topics.sh
```

### Dry Run
```bash
# See what would be created without executing
bash -n recreate-topics.sh  # Syntax check
cat recreate-topics.sh | grep "Creating topic"  # List topics
```

### Parallel Execution
```bash
# Split into chunks for faster creation
split -l 2000 recreate-topics.sh chunk_
for chunk in chunk_*; do
  bash $chunk &
done
wait
```

## Verification

After running the script, verify topics were created correctly:

```bash
# List all topics
kafka-topics.sh --bootstrap-server target:9092 --list

# Compare counts
SOURCE_COUNT=$(jq '.total_topics' kafka-cluster-info.json)
TARGET_COUNT=$(kafka-topics.sh --bootstrap-server target:9092 --list | wc -l)
echo "Source: $SOURCE_COUNT, Target: $TARGET_COUNT"

# Describe specific topic
kafka-topics.sh --bootstrap-server target:9092 --describe --topic orders
```

## Troubleshooting

### "Topic already exists"
This is normal if re-running the script. Failed topics are tracked but don't stop execution.

### Authentication errors
Ensure `client.properties` is configured correctly and `COMMAND_CONFIG` is uncommented.

### "kafka-topics.sh: command not found"
Update `KAFKA_TOPICS` variable to full path:
```bash
KAFKA_TOPICS="/opt/kafka/bin/kafka-topics.sh"
```

### Different Kafka versions
The script uses standard `kafka-topics.sh` command compatible with Kafka 2.0+. For older versions, you may need to adjust flags.
