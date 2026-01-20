# kmap - Kafka Cluster Inspector

Analyze Kafka clusters and export to JSON/HTML/DOT formats. Single binary, all authentication methods supported.

## Features

- �️ **Broker metrics** - Version, partition/leader distribution, under-replicated partitions (URPs)
- 📊 **Topic discovery** - Partitions, replication, configs
- 👥 **Consumer groups** - Members, subscriptions, state  
- 📁 **JSON export** - Recreate topics elsewhere
- 📈 **HTML reports** - Clean tables with broker and topic details
- 🗺️ **Graphviz DOT** - High-quality visualizations for large clusters
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

### Local/No Auth
```bash
kmap -brokers localhost:9092
```

### Confluent Cloud
```bash
kmap -brokers xxxxx.region.aws.confluent.cloud:9092 \
  -security-protocol SASL_SSL \
  -sasl-username <API-KEY> \
  -sasl-password <API-SECRET>
```

### SCRAM
```bash
kmap -brokers broker:9093 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-256 \
  -sasl-username admin \
  -sasl-password secret
```

### mTLS
```bash
kmap -brokers broker:9093 \
  -security-protocol SSL \
  -tls-ca-cert ca.pem \
  -tls-client-cert client.pem \
  -tls-client-key client.key
```

### AWS MSK (SCRAM)
```bash
kmap -brokers b-1.cluster.kafka.us-east-1.amazonaws.com:9096 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username <USER> \
  -sasl-password <PASS>
```

### Azure Event Hubs
```bash
kmap -brokers <namespace>.servicebus.windows.net:9093 \
  -security-protocol SASL_SSL \
  -sasl-username '$ConnectionString' \
  -sasl-password '<connection-string>'
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

# Install Graphviz
brew install graphviz              # macOS
sudo apt install graphviz          # Linux

# Convert
dot -Tpng -Gdpi=300 topology.dot -o diagram.png
dot -Tsvg topology.dot -o diagram.svg
dot -Tpdf topology.dot -o diagram.pdf

# Or use helper
./dot-to-image.sh topology.dot
```

**View DOT online (no install):**
- https://dreampuf.github.io/GraphvizOnline/
- https://edotor.net/

## Deployment

### Server
```bash
scp bin/kmap-linux-amd64 server:/usr/local/bin/kmap
ssh server "kmap -brokers kafka:9092"
```

### Kubernetes Pod
```bash
kubectl cp bin/kmap-linux-amd64 pod:/tmp/kmap
kubectl exec pod -- /tmp/kmap -brokers kafka:9092
kubectl cp pod:/kafka-cluster-info.json ./
```

### Docker
```bash
docker run --rm --network kafka-net -v $(pwd):/output alpine sh -c "
  wget -O /tmp/kmap https://releases.company.com/kmap-linux-amd64
  chmod +x /tmp/kmap
  /tmp/kmap -brokers kafka:9092 -output /output/cluster.json
"
```

### K8s Job
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: kmap
spec:
  template:
    spec:
      containers:
      - name: kmap
        image: alpine
        command: ["/bin/sh", "-c"]
        args:
          - |
            wget -O /tmp/kmap https://releases.company.com/kmap-linux-amd64
            chmod +x /tmp/kmap
            /tmp/kmap -brokers kafka:9092
      restartPolicy: Never
```

## Building

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

## Security Best Practices

✅ Use `SASL_SSL` in production  
✅ Store creds in env vars  
✅ Prefer SCRAM over PLAIN  
✅ Verify TLS certificates  
✅ Use mTLS for highest security  

❌ Don't use PLAINTEXT in prod  
❌ Don't hard-code credentials  
❌ Don't use `-tls-skip-verify` in prod  

## Use Cases

- **Cluster Health** - Check broker versions, detect under-replicated partitions (URPs)
- **Load Balancing** - View partition/leader distribution across brokers
- **Cluster Discovery** - What topics/consumers exist?
- **Migration** - Export/recreate topics in new cluster
- **Documentation** - Generate topology diagrams
- **Monitoring** - Track cluster changes over time
- **Audit** - Who consumes what topics?
- **Capacity Planning** - Identify heavily-used topics

## Examples

### Daily Backup
```bash
#!/bin/bash
DATE=$(date +%Y-%m-%d)
kmap -brokers kafka:9092 -output backup-$DATE.json
```

### Weekly Report
```bash
#!/bin/bash
kmap -brokers kafka:9092 -html report.html -dot topology.dot
dot -Tpng topology.dot -o topology.png
# Email report.html and topology.png
```

### CI/CD Health Check
```bash
# GitHub Actions
- run: |
    ./kmap -brokers $KAFKA_BROKERS -output cluster.json
    test $(jq '.total_topics' cluster.json) -gt 0
```

## Contributing

```bash
go mod download
go run main.go -brokers localhost:9092
go test ./...
go fmt ./...
```

## License

MIT

---

**Project:** https://github.com/yourorg/kmap  
**Binaries:** `bin/kmap-*`  
**Docs:** This README  
**Support:** Open an issue
