# Quickstart

## Start services

```bash
docker-compose up --build
```

The compose file sets `GATEWAY_PASETO_KEY` to a sample 32-byte base64url secret.

## Create a wallet DID

Build the CLI if needed:

```bash
go build -o wallet-cli ./cmd/wallet-cli
```

```bash
./wallet-cli did new --out ./wallet.json
```

## Register issuer with the gateway

Fetch issuer DID/public key and register it:

```bash
curl -s http://localhost:8090/v1/issuer/did
```

```bash
curl -s -X PUT http://localhost:8080/v1/issuers/<issuer_did> \
  -H 'X-Admin-Token: admin-token' \
  -H 'Content-Type: application/json' \
  -d '{"public_key":"<issuer_public_key>","enabled":true,"trust_tier":1}'
```

## Request a premium credential

```bash
DID=$(cat ./wallet.json | jq -r .did) # or copy the DID field manually
./wallet-cli cred request --issuer http://localhost:8090 --did "$DID" --type PremiumCredential --claims plan=premium --out ./cred.jwt
```

The CLI prints the credential `jti` after issuing.

## Verify auth and mint token

```bash
./wallet-cli auth verify --gateway http://localhost:8080 --wallet ./wallet.json --cred ./cred.jwt --scopes premium
```

Copy the `access_token` from the response.

## Call premium endpoint

```bash
./wallet-cli call --gateway http://localhost:8080 --token <access_token> --path /api/v1/premium
```

## Revoke credential

Find the `jti` from the issuer response when issuing, then update the revocation list:

```bash
curl -s -X PUT http://localhost:8080/v1/revocations/default \
  -H 'X-Admin-Token: admin-token' \
  -H 'Content-Type: application/json' \
  -d '{"listId":"default","revoked":["<jti>"],"updatedAt":"2024-01-01T00:00:00Z"}'
```

Then repeat `/v1/auth/verify` with the revoked credential; it will fail.
