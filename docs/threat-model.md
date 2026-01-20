# Threat Model (MVP)

## Assets

- Access tokens minted by the gateway
- Verifiable credentials issued by the issuer
- DID private keys held by wallet/issuer
- Policy definitions and issuer registry

## Threats and mitigations

- **Replay of auth challenge**: Nonce stored in Redis with TTL; nonce is single-use.
- **Signature forgery**: DID signature verification over canonical challenge string.
- **Credential replay**: Short-lived access tokens; revocation list checked by jti.
- **Issuer impersonation**: Issuer registry enforces allowed DIDs and public keys.
- **Token abuse**: Rate limiting per DID and policy.
- **Privilege escalation**: Policy enforcement checks required scopes, VC types, issuer allowlist, and trust tier.
- **Data exfiltration**: Minimal PII in audit logs.

## Out of scope (MVP)

- Hardware-backed key storage
- Secure enclaves or mTLS between services
- Advanced VC status mechanisms (StatusList2021, accumulators)
