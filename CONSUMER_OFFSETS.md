# Consumer Offsets Backup and Restore

This guide explains how to use kmap's consumer offset backup and restore functionality for migrating consumer group positions between Kafka clusters.

## Overview

The consumer offset backup feature captures the current committed offsets for all consumer groups in your Kafka cluster and generates a restore script that can recreate those positions on another cluster.

**Use Cases:**
- Disaster Recovery - Quickly restore consumer positions after cluster failure
- Cluster Migration - Preserve consumer progress when moving to a new cluster
- Environment Sync - Copy production consumer positions to test/staging
- Rollback Scenarios - Restore consumer groups to a previous state
- Reprocessing - Set consumer groups back to specific offsets for data reprocessing

## Quick Start

### 1. Backup Consumer Offsets

```bash
./kmap -brokers localhost:9092 \
  -save-offsets consumer-offsets.json \
  -restore-offsets-script restore-offsets.sh
```

This creates two files:
- `consumer-offsets.json` - JSON backup of all consumer group offsets
- `restore-offsets.sh` - Executable bash script to restore the offsets

### 2. Review the Backup

```bash
# Check what was captured
cat consumer-offsets.json | jq '.consumer_groups[] | {group: .group, topics: .topics | keys}'

# See backup metadata
cat consumer-offsets.json | jq '{timestamp, cluster, total_groups: (.consumer_groups | length)}'
```

### 3. Restore on Target Cluster

```bash
# Edit the script to point to your target cluster
vim restore-offsets.sh  # Change BOOTSTRAP_SERVERS

# Make executable and run
chmod +x restore-offsets.sh
./restore-offsets.sh
```

## Backup File Format

The JSON backup contains:
- Timestamp of when backup was created
- Source cluster address
- All consumer groups with committed offsets
- For each group: topics, partitions, and current offset values

Example `consumer-offsets.json`:
```json
{
  "timestamp": "2026-01-20T15:30:04Z",
  "cluster": "localhost:9092",
  "consumer_groups": [
    {
      "group": "my-consumer-group",
      "topics": {
        "orders": [
          {"partition": 0, "offset": 12345},
          {"partition": 1, "offset": 67890}
        ],
        "payments": [
          {"partition": 0, "offset": 98765}
        ]
      },
      "timestamp": "2026-01-20T15:30:04Z"
    }
  ]
}
```

## Restore Script

The generated `restore-offsets.sh` script:
- Sets consumer group offsets using `kafka-consumer-groups.sh --reset-offsets`
- Processes each consumer group and topic combination
- Tracks successful and failed restorations
- Supports authentication via `--command-config`
- Provides progress indicators and final summary

## Authentication

### Source Cluster (Backup)

Use kmap's standard authentication flags:

**SASL/SCRAM:**
```bash
./kmap -brokers kafka:9092 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-256 \
  -sasl-username admin \
  -sasl-password secret \
  -save-offsets offsets.json \
  -restore-offsets-script restore.sh
```

**mTLS:**
```bash
./kmap -brokers kafka:9092 \
  -security-protocol SSL \
  -tls-cert /path/to/client.crt \
  -tls-key /path/to/client.key \
  -tls-ca /path/to/ca.crt \
  -save-offsets offsets.json \
  -restore-offsets-script restore.sh
```

### Target Cluster (Restore)

Edit the generated script to add authentication:

**Using command-config file:**
```bash
# In restore-offsets.sh, uncomment and set:
COMMAND_CONFIG="--command-config /path/to/client.properties"
```

**client.properties example (SASL/SCRAM):**
```properties
security.protocol=SASL_SSL
sasl.mechanism=SCRAM-SHA-256
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required \
  username="admin" \
  password="secret";
```

**client.properties example (mTLS):**
```properties
security.protocol=SSL
ssl.keystore.location=/path/to/keystore.jks
ssl.keystore.password=keystore-password
ssl.key.password=key-password
ssl.truststore.location=/path/to/truststore.jks
ssl.truststore.password=truststore-password
```

## Advanced Usage

### Backup Only Specific Groups

The tool backs up all groups automatically, but you can filter the JSON:

```bash
# Save only specific group
cat consumer-offsets.json | jq \
  '.consumer_groups |= map(select(.group == "my-group"))' \
  > filtered-offsets.json
```

### Combine with Topic Recreation

For complete cluster migration, use both features:

```bash
# Step 1: Backup source cluster structure
./kmap -brokers source:9092 [auth-flags] \
  -output json \
  -recreate-script recreate-topics.sh \
  -save-offsets consumer-offsets.json \
  -restore-offsets-script restore-offsets.sh

# Step 2: Recreate topics on target
./recreate-topics.sh  # Edit bootstrap servers first

# Step 3: Restore consumer positions
./restore-offsets.sh  # Edit bootstrap servers first
```

### Dry Run Testing

Test the restore script without making changes:

```bash
# Edit restore-offsets.sh and add --dry-run to the kafka-consumer-groups command
# Change line:
#   --reset-offsets --from-file $TEMP_JSON --execute
# To:
#   --reset-offsets --from-file $TEMP_JSON --dry-run
```

### Cross-Platform Restore

The restore script requires `kafka-consumer-groups.sh` from Apache Kafka. If it's not in PATH:

```bash
# Edit restore-offsets.sh and set:
KAFKA_CONSUMER_GROUPS="/opt/kafka/bin/kafka-consumer-groups.sh"
```

## Important Notes

1. **Consumer Groups Must Be Inactive**
   - Stop all consumers before running the restore script
   - Active consumers will continue from their current position
   - The tool will warn if groups are active

2. **Topics Must Exist**
   - Target cluster must have matching topics
   - Use topic recreation script first if needed
   - Partition counts must match or be higher

3. **Offset Validity**
   - Tool captures current committed offsets
   - Offsets may be stale if backup is old
   - Verify offset ranges match target cluster data

4. **Empty Groups Excluded**
   - Groups with no committed offsets are not backed up
   - This is normal for newly created or inactive groups
   - Only groups with actual offset commits are included

5. **Error Handling**
   - Script continues on individual topic failures
   - Check summary for failed restorations
   - Review logs for specific error messages

## Verification

After restoration, verify offsets were set correctly:

```bash
# List all consumer groups
kafka-consumer-groups.sh --bootstrap-server target:9092 --list

# Check specific group
kafka-consumer-groups.sh --bootstrap-server target:9092 \
  --group my-consumer-group --describe

# Verify offset positions match backup
cat consumer-offsets.json | jq -r \
  '.consumer_groups[] | select(.group == "my-consumer-group") | 
   .topics | to_entries[] | 
   "\(.key):\t\(.value | map("\(.partition)=\(.offset)") | join(" "))"'
```

## Troubleshooting

### Warning: expected int32 array to be non null

This means the consumer group has no committed offsets for that topic. Common causes:
- Consumer group is newly created
- Group hasn't consumed from that topic yet
- Offsets have expired (based on `offsets.retention.minutes`)

**Solution:** This is normal. Only groups with committed offsets are backed up.

### Error: Group is not empty

The consumer group has active members. Solutions:
- Stop all consumers in the group
- Wait for session timeout (usually 10-30 seconds)
- Use `--force` flag in restore script (may cause issues)

### Error: Topic does not exist

The topic doesn't exist on target cluster:
- Create the topic first using topic recreation script
- Or manually create with matching configuration

### Error: Offset out of range

The offset from backup is invalid for target cluster:
- Source and target data may differ
- Use earliest/latest instead: `--reset-offsets --to-earliest`
- Or manually adjust offsets in JSON file

### Permission Denied

Authentication or authorization issues:
- Verify credentials in `client.properties`
- Ensure user has `ALTER` permission on consumer groups
- Check `--command-config` path is correct

## Examples

### AWS MSK Migration

```bash
# Backup from source MSK
./kmap -brokers source-msk.kafka.amazonaws.com:9092 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username admin \
  -sasl-password $SOURCE_PASSWORD \
  -save-offsets msk-offsets.json \
  -restore-offsets-script restore-msk.sh

# Edit restore-msk.sh:
# - Set BOOTSTRAP_SERVERS to target MSK endpoint
# - Configure authentication

# Run restore on target
./restore-msk.sh
```

### Confluent Cloud to Self-Hosted

```bash
# Backup from Confluent Cloud
./kmap -brokers pkc-xxxxx.aws.confluent.cloud:9092 \
  -security-protocol SASL_SSL \
  -sasl-mechanism PLAIN \
  -sasl-username $API_KEY \
  -sasl-password $API_SECRET \
  -save-offsets confluent-offsets.json \
  -restore-offsets-script restore-self-hosted.sh

# Edit restore script for self-hosted cluster
vim restore-self-hosted.sh

# Run restore
./restore-self-hosted.sh
```

### Scheduled Backups

Create a cron job for regular backups:

```bash
#!/bin/bash
# backup-kafka-offsets.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/kafka"
mkdir -p $BACKUP_DIR

./kmap -brokers kafka:9092 \
  -sasl-username admin \
  -sasl-password $KAFKA_PASSWORD \
  -save-offsets $BACKUP_DIR/offsets_$DATE.json \
  -restore-offsets-script $BACKUP_DIR/restore_$DATE.sh

# Keep only last 30 days
find $BACKUP_DIR -name "offsets_*.json" -mtime +30 -delete
find $BACKUP_DIR -name "restore_*.sh" -mtime +30 -delete
```

Add to crontab:
```cron
# Daily backup at 2 AM
0 2 * * * /usr/local/bin/backup-kafka-offsets.sh >> /var/log/kafka-backup.log 2>&1
```

## Related Features

- See [RECREATE_TOPICS.md](RECREATE_TOPICS.md) for topic structure backup
- See [AUTH.md](AUTH.md) for authentication configuration
- See [README.md](README.md) for general usage

## Limitations

1. No timestamp-based restore (yet) - only offset-based
2. Consumer group metadata (members, state) is not preserved
3. Requires `kafka-consumer-groups.sh` on target system
4. Manual script editing required for different target clusters
5. No automatic validation of offset ranges on target

## Future Enhancements

Planned features:
- Timestamp-based offset reset (`--to-datetime`)
- Lag calculation during backup
- Automatic validation of offset ranges
- Direct restore without external scripts
- Consumer group filtering by pattern
- Differential backups (only changed offsets)
