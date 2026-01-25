#!/bin/bash
# Quick reference for calculating Kafka topic sizes
# This mimics the kafka-log-dirs.sh command but with kmap

set -e

# Display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Calculate Kafka topic sizes using kmap

OPTIONS:
    -b, --brokers BROKERS       Kafka broker addresses (required)
    -t, --topics TOPICS         Comma-separated list of topics (optional, default: all)
    -o, --output FILE           Save JSON report to file (optional)
    -c, --config FILE           Authentication config file (optional)
    
    Authentication options (if not using config file):
    --security-protocol PROTO   SASL_SSL, SASL_PLAINTEXT, SSL, or empty
    --sasl-mechanism MECH       PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
    --sasl-username USER        SASL username
    --sasl-password PASS        SASL password

EXAMPLES:
    # Local cluster, all topics
    $0 -b localhost:9092

    # Specific topics
    $0 -b localhost:9092 -t "orders,users,events"

    # AWS MSK with SCRAM
    $0 -b broker.kafka.us-east-1.amazonaws.com:9096 \\
       --security-protocol SASL_SSL \\
       --sasl-mechanism SCRAM-SHA-512 \\
       --sasl-username myuser \\
       --sasl-password mypass

    # Save to file
    $0 -b localhost:9092 -o topic-sizes.json

COMPARISON WITH kafka-log-dirs.sh:
    
    # Old way (kafka-log-dirs.sh):
    kafka-log-dirs.sh --bootstrap-server \$BROKER \\
      --command-config \$CONFIG --topic-list \$TOPIC --describe | \\
      grep -oP '(?<=size":)\\d+' | awk '{ sum += \$1 } END { print sum }' | \\
      numfmt --to=iec-i --suffix=B
    
    # New way (kmap):
    $0 -b \$BROKER -c \$CONFIG -t \$TOPIC

EOF
    exit 1
}

# Check if kmap is available
if [ ! -f "./kmap" ] && ! command -v kmap &> /dev/null; then
    echo "Error: kmap binary not found"
    echo "Please build it first: make build"
    exit 1
fi

# Use local kmap if available, otherwise use system kmap
KMAP="./kmap"
if [ ! -f "./kmap" ]; then
    KMAP="kmap"
fi

# Initialize variables
BROKERS=""
TOPICS=""
OUTPUT=""
CONFIG_FILE=""
SECURITY_PROTOCOL=""
SASL_MECHANISM=""
SASL_USERNAME=""
SASL_PASSWORD=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -b|--brokers)
            BROKERS="$2"
            shift 2
            ;;
        -t|--topics)
            TOPICS="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --security-protocol)
            SECURITY_PROTOCOL="$2"
            shift 2
            ;;
        --sasl-mechanism)
            SASL_MECHANISM="$2"
            shift 2
            ;;
        --sasl-username)
            SASL_USERNAME="$2"
            shift 2
            ;;
        --sasl-password)
            SASL_PASSWORD="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Validate required parameters
if [ -z "$BROKERS" ]; then
    echo "Error: --brokers is required"
    usage
fi

# Build kmap command
CMD="$KMAP -brokers $BROKERS -topic-sizes"

# Add topic filter if specified
if [ -n "$TOPICS" ]; then
    CMD="$CMD -topic-list $TOPICS"
fi

# Add output file if specified
if [ -n "$OUTPUT" ]; then
    CMD="$CMD -topic-sizes-output $OUTPUT"
fi

# Add authentication from config file
if [ -n "$CONFIG_FILE" ]; then
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "Error: Config file not found: $CONFIG_FILE"
        exit 1
    fi
    
    # Parse config file for authentication parameters
    # This is a simple parser for common config formats
    while IFS='=' read -r key value; do
        # Remove whitespace and comments
        key=$(echo "$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | cut -d'#' -f1)
        value=$(echo "$value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | cut -d'#' -f1)
        
        case "$key" in
            security.protocol)
                SECURITY_PROTOCOL="$value"
                ;;
            sasl.mechanism)
                SASL_MECHANISM="$value"
                ;;
            sasl.jaas.config)
                # Extract username and password from JAAS config
                SASL_USERNAME=$(echo "$value" | sed -n 's/.*username="\([^"]*\)".*/\1/p')
                SASL_PASSWORD=$(echo "$value" | sed -n 's/.*password="\([^"]*\)".*/\1/p')
                ;;
        esac
    done < "$CONFIG_FILE"
fi

# Add authentication parameters
if [ -n "$SECURITY_PROTOCOL" ]; then
    CMD="$CMD -security-protocol $SECURITY_PROTOCOL"
fi

if [ -n "$SASL_MECHANISM" ]; then
    CMD="$CMD -sasl-mechanism $SASL_MECHANISM"
fi

if [ -n "$SASL_USERNAME" ]; then
    CMD="$CMD -sasl-username $SASL_USERNAME"
fi

if [ -n "$SASL_PASSWORD" ]; then
    CMD="$CMD -sasl-password $SASL_PASSWORD"
fi

# Execute command
echo "Executing: ${CMD//$SASL_PASSWORD/***}"  # Hide password in output
eval "$CMD"
