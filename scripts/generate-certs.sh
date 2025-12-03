#!/bin/bash

# Script to generate TLS certificates for the webhook
# Usage: ./generate-certs.sh [namespace] [service-name]

set -e

NAMESPACE=${1:-harvester-nested-virt}
SERVICE_NAME=${2:-nested-virt-webhook}
SECRET_NAME=nested-virt-webhook-certs

echo "Generating TLS certificates for webhook..."
echo "Namespace: $NAMESPACE"
echo "Service: $SERVICE_NAME"

# Create a temporary directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Generate CA key and certificate
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1825 -out ca.crt \
  -subj "/CN=webhook-ca"

# Generate server key
openssl genrsa -out tls.key 2048

# Create CSR config
cat > csr.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${SERVICE_NAME}
DNS.2 = ${SERVICE_NAME}.${NAMESPACE}
DNS.3 = ${SERVICE_NAME}.${NAMESPACE}.svc
DNS.4 = ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local
EOF

# Generate CSR
openssl req -new -key tls.key -out tls.csr \
  -subj "/CN=${SERVICE_NAME}.${NAMESPACE}.svc" \
  -config csr.conf

# Sign the certificate
openssl x509 -req -in tls.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out tls.crt -days 1825 -sha256 \
  -extensions v3_req -extfile csr.conf

echo ""
echo "Certificates generated successfully!"
echo ""

# Create namespace if it doesn't exist
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Create or update the secret
kubectl create secret tls "$SECRET_NAME" \
  --cert=tls.crt \
  --key=tls.key \
  --namespace="$NAMESPACE" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Secret $SECRET_NAME created in namespace $NAMESPACE"

# Get the CA bundle for the webhook configuration
CA_BUNDLE=$(base64 < ca.crt | tr -d '\n')

echo ""
echo "CA Bundle (base64 encoded):"
echo "$CA_BUNDLE"
echo ""
echo "Update the caBundle field in deploy/webhook.yaml with the above value"

# Copy CA bundle to current directory for reference
cp ca.crt /tmp/webhook-ca.crt

echo ""
echo "CA certificate saved to: /tmp/webhook-ca.crt"

# Cleanup
cd - > /dev/null
rm -rf "$TMP_DIR"

echo ""
echo "Done!"
