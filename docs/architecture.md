# Architecture

## Components

- **Gateway**: authenticates DIDs via challenge-response, verifies JWT-VCs, enforces policies, rate limits per DID and policy, and proxies to upstream. Access tokens are JWT HS256 for MVP.
- **Issuer**: issues JWT-VCs signed with an ed25519 key and exposes its DID and public key.
- **Upstream**: a sample service with public/basic/premium endpoints.
- **Wallet CLI**: holder tool to generate a DID, fetch challenge, sign it, request credentials, and call the gateway.

## Request flow

1. Wallet requests a challenge from the gateway `/v1/auth/challenge`.
2. Wallet signs the challenge string with its DID key and calls `/v1/auth/verify` (optionally with a JWT-VC).
3. Gateway verifies the DID signature, validates the JWT-VC (issuer allowlist + revocation), and mints a short-lived access token.
4. Client calls `/api/*` with the token; gateway enforces policy + rate limit and proxies to upstream.

## Data stores

- Postgres: policies, issuer registry, revocation lists.
- Redis: nonces, rate limiting counters, revocation cache.

## Observability

- JSON structured logs.
- Prometheus metrics at `/metrics`.
- OpenTelemetry tracing (`OTEL_EXPORTER_OTLP_ENDPOINT` optional).
