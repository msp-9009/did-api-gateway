# DID:Web Test Server

A simple HTTP server for testing `did:web` DID resolution.

## Quick Start

```bash
cd test/did-web-server
chmod +x start.sh
./start.sh
```

The server will start on port 8888 and serve:
- **DID Document**: http://localhost:8888/.well-known/did.json
- **Web UI**: http://localhost:8888

## Testing with Gateway

Once the server is running:

```bash
# Test DID resolution
curl 'http://localhost:8080/v1/auth/challenge?did=did:web:localhost:8888'
```

## Custom Configuration

### Use Your Own Public Key

```bash
# Generate Ed25519 key pair (using wallet-cli or openssl)
# Then start server with your public key:
./did-web-test-server -pubkey YOUR_BASE64URL_ENCODED_PUBKEY -domain localhost:8888
```

### Change Port

```bash
./did-web-test-server -port 9999 -domain localhost:9999
```

### Production Domain

For production testing with a real domain:

```bash
# Deploy on your server
./did-web-test-server -port 80 -domain example.com

# DID will be: did:web:example.com
# Serves at: https://example.com/.well-known/did.json
```

## Features

- ✅ Serves W3C compliant DID documents
- ✅ Ed25519VerificationKey2020 format
- ✅ CORS enabled for cross-origin requests
- ✅ Health check endpoint at `/health`
- ✅ Interactive web UI with instructions
- ✅ Configurable domain and public key

## DID Document Format

The server generates a DID document like:

```json
{
  "@context": [
    "https://www.w3.org/ns/did/v1",
    "https://w3id.org/security/suites/ed25519-2020/v1"
  ],
  "id": "did:web:localhost:8888",
  "verificationMethod": [{
    "id": "did:web:localhost:8888#key-1",
    "type": "Ed25519VerificationKey2020",
    "controller": "did:web:localhost:8888",
    "publicKeyJwk": {
      "kty": "OKP",
      "crv": "Ed25519",
      "x": "BASE64URL_ENCODED_PUBLIC_KEY"
    }
  }],
  "authentication": ["did:web:localhost:8888#key-1"]
}
```

## Endpoints

- `GET /.well-known/did.json` - DID Document
- `GET /health` - Health check (returns "OK")
- `GET /` - Web UI with instructions

## Building

```bash
go build -o did-web-test-server main.go
```

## Docker

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY main.go .
RUN go build -o did-web-test-server main.go
EXPOSE 8888
CMD ["./did-web-test-server", "-port", "8888", "-domain", "localhost:8888"]
```
