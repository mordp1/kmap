#!/bin/bash
#
# AWS MSK 2.6 Cluster Analyzer Script
# Configure your MSK cluster details below and run this script
#

set -e

# MSK Cluster Configuration
MSK_BOOTSTRAP_SERVERS="b-1.your-cluster.kafka.region.amazonaws.com:9096,b-2.your-cluster.kafka.region.amazonaws.com:9096"
SASL_USERNAME="your-username"
SASL_PASSWORD="your-password"

# Output files
OUTPUT_JSON="msk-cluster-info.json"
OUTPUT_HTML="msk-cluster-report.html"
OUTPUT_RECREATE="recreate-msk-topics.sh"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}AWS MSK 2.6 Cluster Analyzer${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# Check if kmap binary exists
if [ ! -f "./kmap" ]; then
    echo -e "${YELLOW}Error: kmap binary not found in current directory${NC}"
    echo "Please ensure you're running this script from the msk-deploy folder"
    exit 1
fi

# Make sure kmap is executable
chmod +x ./kmap

echo -e "${GREEN}Analyzing MSK cluster...${NC}"
echo "Bootstrap servers: $MSK_BOOTSTRAP_SERVERS"
echo ""

# Run kmap with MSK SCRAM-SHA-512 configuration
./kmap \
    -brokers "$MSK_BOOTSTRAP_SERVERS" \
    -security-protocol SASL_SSL \
    -sasl-mechanism SCRAM-SHA-512 \
    -sasl-username "$SASL_USERNAME" \
    -sasl-password "$SASL_PASSWORD" \
    -json "$OUTPUT_JSON" \
    -html "$OUTPUT_HTML" \
    -recreate-script "$OUTPUT_RECREATE"

echo ""
echo -e "${GREEN}✓ Analysis complete!${NC}"
echo ""
echo "Generated files:"
echo "  - $OUTPUT_JSON (cluster metadata)"
echo "  - $OUTPUT_HTML (HTML report)"
echo "  - $OUTPUT_RECREATE (topic recreation script)"
echo ""
echo -e "${BLUE}Broker metrics, topics, and consumer groups have been analyzed.${NC}"
