#!/bin/bash
# Register Issuer as Trusted
# This script fetches the issuer's DID and registers it in the gateway database

set -e

ISSUER_URL="${ISSUER_URL:-http://localhost:8090}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-did-code-postgres-1}"

echo "🔍 Fetching issuer DID from $ISSUER_URL..."
ISSUER_INFO=$(curl -s "$ISSUER_URL/v1/issuer/did")
ISSUER_DID=$(echo "$ISSUER_INFO" | jq -r '.did')
ISSUER_PUBLIC_KEY=$(echo "$ISSUER_INFO" | jq -r '.public_key')

if [ -z "$ISSUER_DID" ] || [ "$ISSUER_DID" = "null" ]; then
    echo "❌ Failed to fetch issuer DID from $ISSUER_URL"
    echo "Response: $ISSUER_INFO"
    exit 1
fi

echo "✅ Issuer DID: $ISSUER_DID"
echo "📝 Public Key: $ISSUER_PUBLIC_KEY"
echo ""

echo "💾 Registering issuer in gateway database..."
docker compose exec -T postgres psql -U gateway -d gateway << EOF
INSERT INTO issuers (did, public_key, enabled, trust_tier, created_at, updated_at) 
VALUES 
    ('$ISSUER_DID', '$ISSUER_PUBLIC_KEY', true, 1, NOW(), NOW())
ON CONFLICT (did) 
DO UPDATE SET 
    public_key = EXCLUDED.public_key,
    enabled = true,
    trust_tier = 1,
    updated_at = NOW();

SELECT did, trust_tier, enabled FROM issuers WHERE did = '$ISSUER_DID';
EOF

echo ""
echo "✅ Issuer registered successfully!"
echo ""
echo "You can now authenticate with credentials:"
echo "  ./wallet-cli auth verify \\"
echo "    --gateway http://localhost:8080 \\"
echo "    --wallet ./wallet.json \\"
echo "    --cred ./cred.jwt \\"
echo "    --scopes premium"
