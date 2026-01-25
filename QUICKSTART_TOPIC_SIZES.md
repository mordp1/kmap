# Quick Start: Topic Sizes

Get the disk space used by your Kafka topics in seconds.

## Simplest Usage

```bash
# All topics on local Kafka
./kmap -brokers localhost:9092 -topic-sizes
```

## Common Scenarios

### AWS MSK (with SCRAM authentication)
```bash
./kmap -brokers b-1.mycluster.kafka.us-east-1.amazonaws.com:9096 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username myuser \
  -sasl-password mypassword \
  -topic-sizes
```

### Confluent Cloud
```bash
./kmap -brokers pkc-xxxxx.us-east-1.aws.confluent.cloud:9092 \
  -security-protocol SASL_SSL \
  -sasl-mechanism PLAIN \
  -sasl-username YOUR-API-KEY \
  -sasl-password YOUR-API-SECRET \
  -topic-sizes
```

### Check Specific Topics
```bash
./kmap -brokers localhost:9092 \
  -topic-sizes \
  -topic-list "orders,users,events"
```

### Save to File
```bash
./kmap -brokers localhost:9092 \
  -topic-sizes \
  -topic-sizes-output sizes.json
```

## Understanding the Output

```
TOPIC                    PARTITIONS   TOTAL SIZE    SIZE (BYTES)
my-topic                 12           456.78 GiB    490,463,289,344
```

- **TOPIC**: Topic name
- **PARTITIONS**: Number of partitions
- **TOTAL SIZE**: Human-readable (includes replication!)
- **SIZE (BYTES)**: Exact bytes

⚠️ **Important**: Size includes replication factor!
- If RF=3, shown size = 3× actual data size
- This is the real disk space used

## Quick Recipes

### Find biggest topic
```bash
./kmap -brokers localhost:9092 -topic-sizes | head -20
```

### Get total cluster storage
```bash
./kmap -brokers localhost:9092 -topic-sizes | grep "Total Size"
```

### Track growth over time
```bash
# Today
./kmap -brokers localhost:9092 -topic-sizes -topic-sizes-output today.json

# Tomorrow
./kmap -brokers localhost:9092 -topic-sizes -topic-sizes-output tomorrow.json

# Compare
./compare-topic-sizes.sh today.json tomorrow.json
```

### Find topics over 1TB
```bash
./kmap -brokers localhost:9092 -topic-sizes -topic-sizes-output sizes.json
jq '.topics[] | select(.total_size_bytes > 1099511627776)' sizes.json
```

## Using Environment Variables

```bash
# Set once
export KAFKA_BROKERS="broker:9092"
export KAFKA_USER="myuser"
export KAFKA_PASS="mypass"

# Use many times
./kmap -brokers $KAFKA_BROKERS \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username $KAFKA_USER \
  -sasl-password $KAFKA_PASS \
  -topic-sizes
```

## Equivalent kafka-log-dirs.sh Command

Instead of this complex pipeline:
```bash
kafka-log-dirs.sh --bootstrap-server $BROKER \
  --command-config $CONFIG \
  --topic-list $TOPIC \
  --describe | \
  grep -oP '(?<=size":)\d+' | \
  awk '{ sum += $1 } END { print sum }' | \
  numfmt --to=iec-i --suffix=B
```

Just use:
```bash
./kmap -brokers $BROKER -topic-sizes -topic-list $TOPIC
```

## Need More Help?

- Full documentation: [TOPIC_SIZES.md](TOPIC_SIZES.md)
- Main README: [README.md](README.md)
- Examples: [examples/topic-sizes-example.sh](examples/topic-sizes-example.sh)
