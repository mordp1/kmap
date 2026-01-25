# AWS MSK 2.6 Cluster Analyzer

This folder contains a Linux x86_64 binary specifically for analyzing AWS MSK 2.6 clusters with SASL_SSL SCRAM-SHA-512 authentication.

## Quick Start

### 1. Upload to Your EC2/Container

```bash
# Copy the entire msk-deploy folder to your EC2 instance or container
scp -r msk-deploy/ ec2-user@your-instance:/home/ec2-user/

# Or use AWS Systems Manager Session Manager
aws s3 cp msk-deploy/ s3://your-bucket/msk-deploy/ --recursive
```

### 2. Configure MSK Connection

Edit `msk-cluster.sh` and update these variables:

```bash
MSK_BOOTSTRAP_SERVERS="b-1.your-cluster.kafka.region.amazonaws.com:9096,b-2.your-cluster.kafka.region.amazonaws.com:9096"
SASL_USERNAME="your-username"
SASL_PASSWORD="your-password"
```

### 3. Run the Analysis

```bash
cd msk-deploy
chmod +x msk-cluster.sh kmap
./msk-cluster.sh
```

## Output Files

The script generates three files:

- **msk-cluster-info.json** - Complete cluster metadata in JSON format
- **msk-cluster-report.html** - HTML report with broker metrics, topics, and consumer groups
- **recreate-msk-topics.sh** - Executable script to recreate all topics with exact configurations

## Manual Usage

You can also run kmap directly with custom parameters:

```bash
./kmap \
    -brokers b-1.cluster.kafka.us-east-1.amazonaws.com:9096 \
    -security-protocol SASL_SSL \
    -sasl-mechanism SCRAM-SHA-512 \
    -sasl-username your-username \
    -sasl-password your-password
```

## Common MSK Configurations

### MSK with SCRAM-SHA-512 (this setup)
```bash
./kmap -brokers $MSK_BOOTSTRAP \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-512 \
  -sasl-username $USER \
  -sasl-password $PASS
```

### MSK with IAM Authentication
```bash
# Note: IAM authentication requires AWS credentials configured on the instance
# and is not yet supported in this version
```

### MSK with mTLS
```bash
./kmap -brokers $MSK_BOOTSTRAP \
  -security-protocol SSL \
  -tls-ca-cert /path/to/ca.pem \
  -tls-client-cert /path/to/client-cert.pem \
  -tls-client-key /path/to/client-key.pem
```

## Features

✓ Broker metrics (version, partitions, leaders, URPs)  
✓ Topic discovery with all configurations  
✓ Consumer group analysis with lag  
✓ Topic recreation script generation  
✓ JSON/HTML output formats  
✓ Compatible with MSK 2.6.x  

## Troubleshooting

**Connection issues:**
- Verify security group allows inbound on port 9096
- Ensure your EC2 instance can reach MSK cluster
- Check MSK cluster security settings allow SASL/SCRAM

**Authentication errors:**
- Verify SCRAM credentials in AWS Secrets Manager
- Ensure the secret has been associated with the MSK cluster
- Check username and password are correct

**Binary not executing:**
```bash
chmod +x kmap
file kmap  # Should show: ELF 64-bit LSB executable, x86-64
```

## Binary Information

- **Architecture**: Linux x86_64 (amd64)
- **Size**: ~7.8 MB
- **Version**: 1.0.0
- **Kafka Compatibility**: 2.6.x and newer

## Need Help?

See the main project documentation: https://github.com/mordp1/kmap
