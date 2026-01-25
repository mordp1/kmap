#!/bin/bash
# Example: Calculate topic sizes for Kafka cluster

# Set your Kafka broker and configuration
BROKERS="localhost:9092"
# For AWS MSK with authentication:
# BROKERS="b-1.msk-cluster.kafka.us-east-1.amazonaws.com:9096"
# SECURITY="-security-protocol SASL_SSL -sasl-mechanism SCRAM-SHA-512"
# AUTH="-sasl-username myuser -sasl-password mypassword"

# Example 1: Check all topics
echo "=== Checking all topics ==="
./kmap -brokers $BROKERS -topic-sizes

# Example 2: Check specific topics
echo ""
echo "=== Checking specific topics ==="
./kmap -brokers $BROKERS -topic-sizes -topic-list "topic1,topic2,topic3"

# Example 3: Save to JSON file
echo ""
echo "=== Saving report to JSON ==="
./kmap -brokers $BROKERS -topic-sizes -topic-sizes-output topic-sizes-report.json

# Example 4: With authentication (AWS MSK)
# echo ""
# echo "=== With authentication ==="
# ./kmap -brokers $BROKERS $SECURITY $AUTH -topic-sizes

echo ""
echo "Done! Check topic-sizes-report.json for the detailed report."
