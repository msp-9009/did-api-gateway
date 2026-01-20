# API

## Gateway

### GET /healthz

### GET /readyz

### GET /v1/auth/challenge?did={did}

Response:

```json
{
  "challenge": "did=...\nnonce=...\naud=did-gateway\ndomain=localhost\nexp=1700000000\n",
  "nonce": "...",
  "expiresAt": 1700000000,
  "audience": "did-gateway",
  "domain": "localhost"
}
```

Challenge canonical format:

```
did=<did>
nonce=<nonce>
aud=<audience>
domain=<domain>
exp=<unix>
```

### POST /v1/auth/verify

Request:

```json
{
  "did": "did:key:z...",
  "challenge": "...",
  "signature": "<base64url ed25519 signature>",
  "scopes": ["basic", "premium"],
  "credential": "<jwt-vc>"
}
```

If `scopes` is omitted, the gateway defaults to `basic` and adds `premium` when a `PremiumCredential` is presented.

Response:

```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 300
}
```

### Admin endpoints

- GET `/v1/policies`
- PUT `/v1/policies/{id}`
- GET `/v1/issuers`
- PUT `/v1/issuers/{did}`
- PUT `/v1/revocations/{listId}`

Admin requests must include `X-Admin-Token`.

Revocation list payload:

```json
{
  "listId": "default",
  "revoked": ["jti1", "jti2"],
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

### Proxy

`/api/*` is forwarded to the upstream after authz/ratelimit.

## Issuer

- POST `/v1/issue`
- GET `/v1/issuer/did`
- GET `/healthz`

## Upstream

- GET `/v1/public`
- GET `/v1/basic`
- GET `/v1/premium`
