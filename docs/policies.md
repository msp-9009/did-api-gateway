# Policies

Policies are stored in Postgres and matched by **longest route prefix**.

Fields:

- `id`, `name`
- `route_prefix`: path prefix for matching
- `required_scopes`: required access token scopes
- `required_vc_types`: required VC types (optional)
- `allowed_issuers`: allowlist of issuer DIDs (optional)
- `min_trust_tier`: minimum issuer trust tier (optional)
- `rate_limit`: per DID window and max requests
- `token_ttl_seconds`: access token TTL for tokens minted with matching scopes

## Default policies

- `public`: `/api/v1/public`, no auth
- `basic`: `/api/v1/basic`, scope `basic`
- `premium`: `/api/v1/premium`, scope `premium` + VC type `PremiumCredential`

## Scope minting

During `/v1/auth/verify`, requested scopes are validated against the allowed scopes derived from VC types:

- Always allow `basic`
- Add `premium` if VC types include `PremiumCredential`
