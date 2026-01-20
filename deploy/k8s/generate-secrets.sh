#!/bin/bash
# Generate secrets for local development

set -e

echo "Generating secrets for Phase 1 security hardening..."

# Generate PASETO key (32 bytes, base64url encoded)
echo "Generating PASETO key..."
PASETO_KEY=$(openssl rand -base64 32 | tr -d '=')
echo "PASETO_KEY=$PASETO_KEY"

# Generate admin token
echo "Generating admin token..."
ADMIN_TOKEN=$(openssl rand -hex 32)
echo "ADMIN_TOKEN=$ADMIN_TOKEN"

# Generate PostgreSQL password
echo "Generating PostgreSQL password..."
POSTGRES_PASSWORD=$(openssl rand -hex 16)
echo "POSTGRES_PASSWORD=$POSTGRES_PASSWORD"

# Generate Redis password
echo "Generating Redis password..."
REDIS_PASSWORD=$(openssl rand -hex 16)
echo "REDIS_PASSWORD=$REDIS_PASSWORD"

# Create Kubernetes secrets
echo ""
echo "Creating Kubernetes secrets..."

kubectl create secret generic gateway-secrets \
  --from-literal=paseto-key="$PASETO_KEY" \
  --from-literal=admin-token="$ADMIN_TOKEN" \
  --from-literal=postgres-password="$POSTGRES_PASSWORD" \
  --from-literal=redis-password="$REDIS_PASSWORD" \
  --dry-run=client -o yaml > deploy/k8s/generated-secrets.yaml

echo "Secrets written to deploy/k8s/generated-secrets.yaml"
echo ""
echo "⚠️  IMPORTANT: Store these secrets securely!"
echo "⚠️  In production, use a secrets management system (Vault, AWS Secrets Manager, etc.)"
echo ""

# Generate self-signed TLS certificates for local development
echo "Generating self-signed TLS certificates for local development..."

if ! command -v openssl &> /dev/null; then
    echo "Error: openssl not found. Please install openssl."
    exit 1
fi

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Generate private key
openssl genrsa -out "$TMP_DIR/tls.key" 2048

# Generate certificate
openssl req -new -x509 -key "$TMP_DIR/tls.key" -out "$TMP_DIR/tls.crt" -days 365 \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,DNS:gateway,DNS:issuer,IP:127.0.0.1"

# Create Kubernetes TLS secret
kubectl create secret tls tls-certs \
  --cert="$TMP_DIR/tls.crt" \
  --key="$TMP_DIR/tls.key" \
  --dry-run=client -o yaml >> deploy/k8s/generated-secrets.yaml

echo "TLS certificates written to deploy/k8s/generated-secrets.yaml"
echo ""
echo "✅ Secrets generation complete!"
echo ""
echo "To apply secrets to Kubernetes:"
echo "  kubectl apply -f deploy/k8s/generated-secrets.yaml"
echo ""
echo "For production, use cert-manager for TLS certificates:"
echo "  kubectl apply -f deploy/k8s/certificates.yaml"
