# Sample Output

## JSON Structure

```json
{
  "timestamp": "2026-01-20T12:38:19Z",
  "broker_addresses": [
    "broker1:9092",
    "broker2:9092",
    "broker3:9092"
  ],
  "brokers": [
    {
      "id": 1,
      "address": "broker1:9092",
      "version": "Kafka detected",
      "partitions": 45,
      "leaders": 15,
      "under_replicated_partitions": 0
    },
    {
      "id": 2,
      "address": "broker2:9092",
      "version": "Kafka detected",
      "partitions": 43,
      "leaders": 16,
      "under_replicated_partitions": 2
    },
    {
      "id": 3,
      "address": "broker3:9092",
      "version": "Kafka detected",
      "partitions": 44,
      "leaders": 14,
      "under_replicated_partitions": 0
    }
  ],
  "topics": [
    {
      "name": "orders",
      "partitions": 12,
      "replication_factor": 3,
      "configs": {
        "retention.ms": "604800000",
        "compression.type": "snappy"
      }
    },
    {
      "name": "payments",
      "partitions": 6,
      "replication_factor": 3,
      "configs": {}
    }
  ],
  "consumer_groups": [
    {
      "name": "order-processor",
      "topics": ["orders"],
      "members": 4,
      "state": "Stable"
    },
    {
      "name": "payment-service",
      "topics": ["payments", "orders"],
      "members": 2,
      "state": "Stable"
    }
  ],
  "total_topics": 2,
  "total_consumer_groups": 2,
  "total_partitions": 18,
  "total_under_replicated_partitions": 2
}
```

## Broker Metrics Explained

### `partitions`
Total number of partition replicas assigned to this broker. This includes all replicas (leader + followers) that this broker is responsible for storing.

### `leaders`
Number of partitions where this broker is the leader. The leader handles all read/write requests for its partitions.

### `under_replicated_partitions`
Number of partitions on this broker that don't have all their replicas in sync. This indicates:
- Broker lag or performance issues
- Network problems
- Disk I/O bottlenecks
- Recent broker restart

**Action:** Investigate brokers with URPs > 0.

## HTML Report Features

### Summary Cards
- **Brokers** - Total active brokers
- **Topics** - Total topic count
- **Total Partitions** - Sum of all partitions across all topics
- **Consumer Groups** - Total active consumer groups
- **⚠️ Under-Replicated** - URP warning (appears only if URPs > 0)

### Broker Table
| Field | Description |
|-------|-------------|
| Broker ID | Unique broker identifier |
| Address | Hostname:port |
| Version | Kafka version (if detectable) |
| Partitions | Total partition replicas on broker |
| Leaders | Number of leader partitions |
| Under-Replicated | URPs on this broker (red badge if > 0) |

### Topics Table
- Name, partition count, replication factor
- Custom configurations (non-default values only)

### Consumer Groups Table
- Group name, state, member count
- Subscribed topics list
- State badges (green=Stable, yellow=other states)

## Console Output

```
2026/01/20 12:38:19 Connecting to Kafka brokers: [broker1:9092 broker2:9092]
2026/01/20 12:38:19 Using SASL authentication (mechanism: PLAIN, username: admin)
2026/01/20 12:38:20 Fetching broker information...
2026/01/20 12:38:20 Fetching topics...
2026/01/20 12:38:21 Calculating broker metrics...
2026/01/20 12:38:22 Fetching consumer groups...
2026/01/20 12:38:23 Writing JSON to kafka-cluster-info.json...
2026/01/20 12:38:23 Generating HTML report to kafka-cluster-report.html...
2026/01/20 12:38:23 Done!
2026/01/20 12:38:23 Summary:
2026/01/20 12:38:23   Total Brokers: 3
2026/01/20 12:38:23   Total Topics: 326
2026/01/20 12:38:23   Total Partitions: 1248
2026/01/20 12:38:23   Total Consumer Groups: 169
2026/01/20 12:38:23   ⚠️  Under-Replicated Partitions: 2
```

## Interpreting URP Warnings

### Healthy Cluster (URPs = 0)
```
✓ All partitions are fully replicated
✓ No action needed
```

### Cluster with URPs (URPs > 0)
```
⚠️ Under-Replicated Partitions: 5

Investigation steps:
1. Check broker logs for errors
2. Check disk space: df -h
3. Check broker CPU/memory
4. Check network connectivity
5. Look for replication throttling
6. Consider increasing broker resources
```

### Common URP Causes

1. **Broker Down**
   - One or more brokers offline
   - URPs across many partitions
   - Check broker availability

2. **Disk Full**
   - Broker can't write new data
   - URPs on specific broker
   - Clear disk space immediately

3. **High Load**
   - Broker falling behind
   - Temporary URPs during high traffic
   - Monitor and scale if persistent

4. **Network Issues**
   - Replication lag
   - URPs fluctuate
   - Check network latency/drops

5. **Recent Broker Restart**
   - Normal after restart
   - Should resolve automatically
   - Wait 5-10 minutes
