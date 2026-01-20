# Authentication Reference

## Quick Reference

| Method | Protocol | Port | Use Case |
|--------|----------|------|----------|
| None | - | 9092 | Local dev |
| SASL/PLAIN | SASL_SSL | 9093 | Confluent Cloud, simple |
| SCRAM-256 | SASL_SSL | 9093 | Secure password auth |
| SCRAM-512 | SASL_SSL | 9096 | AWS MSK, highest security |
| SSL/TLS | SSL | 9093 | Server auth only |
| mTLS | SSL | 9093 | Mutual certificate auth |

## Examples

### Confluent Cloud
```bash
kmap -brokers pkc-xxxxx.us-east-1.aws.confluent.cloud:9092 \
  -security-protocol SASL_SSL \
  -sasl-username <API-KEY> \
  -sasl-password <API-SECRET>
```

### AWS MSK with SCRAM
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
  -sasl-password 'Endpoint=sb://<namespace>.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=<KEY>'
```

### Self-hosted with SCRAM-SHA-256
```bash
kmap -brokers broker:9093 \
  -security-protocol SASL_SSL \
  -sasl-mechanism SCRAM-SHA-256 \
  -sasl-username admin \
  -sasl-password secret
```

### mTLS (Mutual TLS)
```bash
kmap -brokers broker:9093 \
  -security-protocol SSL \
  -tls-ca-cert /path/to/ca.pem \
  -tls-client-cert /path/to/client.pem \
  -tls-client-key /path/to/client.key
```

### SSL with Custom CA
```bash
kmap -brokers broker:9093 \
  -security-protocol SSL \
  -tls-ca-cert /path/to/ca.pem
```

### SASL_PLAINTEXT (No TLS)
```bash
kmap -brokers broker:9092 \
  -security-protocol SASL_PLAINTEXT \
  -sasl-mechanism PLAIN \
  -sasl-username developer \
  -sasl-password devpass
```

## Environment Variables

```bash
export KAFKA_BROKERS="broker:9093"
export KAFKA_USERNAME="admin"
export KAFKA_PASSWORD="secret"

kmap -brokers "$KAFKA_BROKERS" \
  -security-protocol SASL_SSL \
  -sasl-username "$KAFKA_USERNAME" \
  -sasl-password "$KAFKA_PASSWORD"
```

## Certificate Formats

Certificates must be PEM format:
```
-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKL...
-----END CERTIFICATE-----
```

### Convert from Other Formats

**DER to PEM:**
```bash
openssl x509 -inform der -in cert.cer -out cert.pem
```

**PKCS12 to PEM:**
```bash
openssl pkcs12 -in keystore.p12 -nokeys -out cert.pem
openssl pkcs12 -in keystore.p12 -nocerts -nodes -out key.pem
```

**JKS to PEM:**
```bash
keytool -importkeystore -srckeystore keystore.jks -destkeystore keystore.p12 -deststoretype PKCS12
openssl pkcs12 -in keystore.p12 -nokeys -out cert.pem
```

## Troubleshooting

**"SASL username and password are required"**  
Add `-sasl-username` and `-sasl-password`

**"Connection timeout"**  
Check port: 9092 (plain), 9093 (SSL), 9096 (AWS MSK)

**"Authentication failed"**  
Verify credentials and mechanism match cluster

**"Certificate verify failed"**  
Provide CA cert with `-tls-ca-cert`

## Security Best Practices

✅ Use SASL_SSL in production  
✅ Store credentials in environment variables  
✅ Prefer SCRAM-SHA-256/512 over PLAIN  
✅ Verify TLS certificates (default behavior)  
✅ Use mTLS for highest security  

❌ Don't use PLAINTEXT in production  
❌ Don't hard-code credentials  
❌ Don't use `-tls-skip-verify` in production
