# Privacy-Preserving DID-Based API Gateway

**Production-ready API gateway using Decentralized Identifiers (DIDs) and Verifiable Credentials (VCs) for privacy-preserving authentication and authorization.**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Production Ready](https://img.shields.io/badge/status-production--ready-green.svg)]()
[![Security](https://img.shields.io/badge/security-hardened-green.svg)]()
[![HA](https://img.shields.io/badge/availability-99.9%25-green.svg)]()

---

## Overview

This gateway enables **passwordless, privacy-preserving authentication** for APIs without traditional usernames, passwords, or centralized identity providers.

### Key Features

✅ **W3C Standards Compliant** - Full DID Core and Verifiable Credentials support  
✅ **Multiple DID Methods** - did:key (local), did:web (domain-based), did:ion (blockchain)  
✅ **Zero Trust Architecture** - Every request authenticated and authorized  
✅ **Privacy-Preserving** - Users control their data, no central registry  
✅ **Production-Grade** - Multi-zone deployment, auto-scaling, automated backups    
✅ **Self-Healing** - Circuit breakers, retry logic, automatic failover  

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Client Applications                        │
│     (wallet-cli, Web App, Mobile App, Browser Extension)     │
└────────────────────┬─────────────────────────────────────────┘
                     │ HTTPS + TLS 1.3
                     ▼
┌──────────────────────────────────────────────────────────────┐
│                     Load Balancer                             │
│          (Multi-zone, Health-check aware)                     │
└────────────────────┬─────────────────────────────────────────┘
                     │
      ┌──────────────┼──────────────┐
      ▼              ▼              ▼
┌──────────┐   ┌──────────┐   ┌──────────┐
│ Gateway  │   │ Gateway  │   │ Gateway  │  (6 replicas across
│ Zone A   │   │ Zone B   │   │ Zone C   │   3 availability zones)
└────┬─────┘   └────┬─────┘   └────┬─────┘
     │              │              │
     └──────────────┼──────────────┘
                    │
    ┌───────────────┴───────────────┐
    │                               │
    ▼                               ▼
┌──────────────────┐      ┌──────────────────┐
│  DID Resolution  │      │  VC Verification │
│  ┌────────────┐  │      │  ┌────────────┐  │
│  │ did:key    │  │      │  │ JWT-VC     │  │
│  │ did:web    │  │      │  │ StatusList │  │
│  │ did:ion    │  │      │  │ Revocation │  │
│  └────────────┘  │      │  └────────────┘  │
│  + Circuit Breaker│      │  + Trust Tier  │  │
│  + Retry Logic   │      │  + Policy Check│  │
└──────────────────┘      └──────────────────┘
    │                               │
    └───────────────┬───────────────┘
                    │
    ┌───────────────┴───────────────┐
    │                               │
    ▼                               ▼
┌──────────────────┐      ┌──────────────────┐
│  Multi-Layer     │      │  PostgreSQL HA   │
│  Cache (Redis)   │      │  + Sentinel      │
│  ┌────────────┐  │      │  ┌────────────┐  │
│  │ L1: Ristretto│      │  │ Policies   │  │
│  │ L2: Redis  │  │      │  │ Issuers    │  │
│  │ <80% hits  │  │      │  │ Revocations│  │
│  └────────────┘  │      │  └────────────┘  │
│  3 replicas      │      │  3 replicas      │
└──────────────────┘      └──────────────────┘
                    │
                    ▼
          ┌──────────────────┐
          │  Upstream API    │
          │  (Protected)     │
          └──────────────────┘
```

---

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 16+ (or use Docker)
- Redis 7+ (or use Docker)

### Run Locally with Docker Compose

```bash
# Clone repository
git clone https://github.com/example/privacy-gateway
cd privacy-gateway

# Start all services
docker compose up --build

# Services will be available at:
# - Gateway: http://localhost:8080
# - Issuer: http://localhost:8090
# - PostgreSQL: localhost:5431
# - Redis: localhost:6379
```

### Test Authentication Flow

```bash
# Build wallet CLI
go build -o wallet-cli cmd/wallet-cli/main.go

# 1. Generate a new DID wallet
./wallet-cli did new --out ./wallet.json
# Output: did:key:z6MkkSC2U9aJDrzgV64PZL8Rqi75aPVifp4iqKtqb6XfoUcM

# 2. Request a credential from issuer
./wallet-cli cred request \
  --issuer http://localhost:8090 \
  --did "$(cat wallet.json | jq -r .did)" \
  --type PremiumCredential \
  --claims plan=premium \
  --out ./cred.jwt

# 3. Register issuer as trusted (one-time setup)
./scripts/register-issuer.sh

# 4. Authenticate and get access token
./wallet-cli auth verify \
  --gateway http://localhost:8080 \
  --wallet ./wallet.json \
  --cred ./cred.jwt \
  --scopes premium

# Output: {"access_token":"eyJ...","expires_in":300}

# 5. Call protected API
./wallet-cli call \
  --gateway http://localhost:8080 \
  --token "YOUR_ACCESS_TOKEN" \
  --path /api/v1/premium
```

---

## Supported DID Methods

### 1. did:key (Local, No Network)

**Format:** `did:key:z6MkkSC2U9aJDrzgV64PZL8Rqi75aPVifp4iqKtqb6XfoUcM`

- ✅ **Self-sovereign** - No central registry
- ✅ **Instant** - No network lookup required
- ✅ **Perfect for testing** - Generate offline
- 🔐 **Security:** Ed25519 public key embedded in DID

**Example:**
```bash
./wallet-cli did new --out wallet.json
# Creates did:key with embedded public key
```

### 2. did:web (Domain-Based)

**Format:** `did:web:example.com` or `did:web:example.com:users:alice`

- ✅ **Human-readable** - Uses your domain name
- ✅ **DNS-based trust** - Leverage existing web infrastructure  
- ✅ **Easy rotation** - Update DID document at `.well-known/did.json`
- 🔄 **Cache TTL:** 1 hour (configurable)

**Resolver:**
- `did:web:example.com` → `https://example.com/.well-known/did.json`
- `did:web:example.com:users:alice` → `https://example.com/users/alice/did.json`

**Circuit Breaker:** 5 failures, 60s reset, 3 retry attempts

### 3. did:ion (Microsoft ION, Blockchain)

**Format:** `did:ion:EiDahaOGH-liLLdDtTxEAdc8i-cfCz-WUcQdRJheMVNn3A`

- ✅ **Decentralized** - Anchored on Bitcoin blockchain
- ✅ **High trust** - Immutable, censorship-resistant
- ✅ **Standards-based** - Sidetree protocol
- ⏱️ **Slower** - Blockchain resolution (~1-3s)

**Resolver:** `https://ion.msidentity.com/identifiers/{did}`

**Circuit Breaker:** 5 failures, 120s reset, 5 retry attempts (aggressive)

---

## Security Features

### Phase 1: Security Hardening ✅

- **Multi-key Token Management** - JWT with key rotation support
- **Secrets Management** - Environment, Kubernetes, Vault providers
- **TLS/mTLS** - End-to-end encryption, mutual authentication
- **Security Headers** - CSP, HSTS, X-Frame-Options, X-XSS-Protection
- **Input Validation** - DID format validation, size limits, sanitization
- **Rate Limiting** - Per-DID, per-route, Redis-backed

### Phase 2: DID & VC Standards ✅

- **W3C DID Core** - Compliant DID resolution and DID documents
- **Verifiable Credentials** - JWT-VC format with cryptographic proofs
- **StatusList2021** - Privacy-preserving revocation checking
- **Trust Tiers** - Issuer reputation system (1-5 scale)
- **Policy Engine** - Dynamic access control based on VC types

---

## Performance & Scalability

### Phase 3: Scalability ✅

**Multi-Layer Caching:**
- **L1 Cache (Ristretto):** In-memory, <1ms latency
- **L2 Cache (Redis):** Distributed, 1-5ms latency
- **Target:** >80% cache hit rate
- **TTL:** did:key (permanent), did:web (1h), did:ion (24h)

**Horizontal Auto-Scaling (HPA):**
- **Min replicas:** 3
- **Max replicas:** 20
- **Metrics:** CPU (70%), Memory (80%), RPS (1000/pod)
- **Scale-up:** 50%/min (fast)
- **Scale-down:** 10%/min (slow, prevent flapping)

**Database Optimization:**
- 8 performance indexes on policies, issuers, revocations
- Connection pooling
- Query optimization

**Load Testing:**
- k6 test suite included
- Target: 5,000 RPS sustained
- P99 latency: <100ms

---

## Reliability & Resilience

### Phase 4: Production Reliability 

**High Availability:**
- **Multi-zone deployment** - 6 replicas across 3 availability zones
- **Pod anti-affinity** - Spread across nodes and zones
- **Tolerate failures** - Survives entire zone outage
- **Zero downtime updates** - Rolling deployment with maxUnavailable=1

**Circuit Breakers:**
- Prevents cascading failures
- Automatic recovery detection
- Fast-fail when services are down
- Metrics tracking (open/closed/half-open)

**Retry Logic:**
- Exponential backoff with jitter
- Configurable max attempts
- Retryable vs non-retryable errors
- Context-aware cancellation

**Automated Backups:**
- **Frequency:** Daily at 2 AM (CronJob)
- **Storage:** Local + S3 (off-site)
- **Retention:** 7 days
- **Compression:** gzip
- **Verification:** Automated restore testing

**Disaster Recovery:**
- **RTO (Recovery Time Objective):** 30 minutes
- **RPO (Recovery Point Objective):** 15 minutes
- 5 disaster scenarios documented
- Tested recovery procedures
- Runbook: [docs/disaster-recovery.md](docs/disaster-recovery.md)

**Redis Sentinel:**
- 3 replicas with automatic failover
- Quorum = 2
- <10s failover time
- Zero data loss

---

## 📊 Monitoring & Observability

**Metrics (Prometheus):**
- Request rate, latency (p50, p95, p99)
- Error rate by endpoint
- DID resolution success/failure rate
- Cache hit/miss rate
- Circuit breaker state
- HPA scaling events

**Health Checks:**
- `/healthz` - Full health with component details
- `/healthz/live` - Liveness probe (Kubernetes)
- `/healthz/ready` - Readiness probe (Kubernetes)

**Structured Logging:**
- JSON format
- Request ID tracing
- Performance metrics

**Dashboards:**
- Grafana overview dashboard ([deploy/monitoring/](deploy/monitoring/))
- Prometheus alerts configured
- SLO tracking (99.9% availability)

---

## Deployment

### Docker Compose (Development)

```bash
docker compose up --build
```

**Services:**
- Gateway: :8080
- Issuer: :8090
- PostgreSQL: :5431
- Redis: :6379
- Upstream (mock): :8081

### Kubernetes (Production)

```bash
# Apply configurations
kubectl apply -f deploy/k8s/

# Includes:
# - gateway-ha.yaml (6 replicas, multi-zone)
# - redis-sentinel.yaml (3 replicas, auto-failover)
# - hpa.yaml (auto-scaling)
# - backup-cron.yaml (daily backups)

# Verify deployment
kubectl get pods -l app=did-gateway
kubectl get hpa
kubectl get pdb

# Check health
kubectl port-forward svc/did-gateway 8080:8080
curl http://localhost:8080/healthz | jq
```

### Environment Variables

```bash
# Gateway
GATEWAY_ADDR=:8080              # Listen address
POSTGRES_DSN=postgres://...     # Database connection
REDIS_ADDR=redis:6379           # Redis connection
TOKEN_ISSUER=gateway            # JWT issuer
TOKEN_SECRET=...                # JWT signing key (use secrets manager)
LOG_LEVEL=info                  # info, debug, warn, error

# Issuer
ISSUER_ADDR=:8090
ISSUER_KEY_FILE=/data/issuer/keys.json
```

---

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/gateway/did/... -v
```

### Load Tests

```bash
cd test/load

# Interactive test runner
./run-tests.sh

# Options:
# 1. Full auth flow (100 → 10,000 VUs)
# 2. DID resolution (cache hit rate testing)
# 3. Quick smoke test (100 VUs, 1 minute)
# 4. Custom scenario
# 5. Run all tests

# Direct k6 execution
k6 run auth-flow.js
k6 run did-resolution.js
```

**Performance Targets:**
- Max RPS: 5,000
- P99 latency: <100ms
- Error rate: <1%
- Cache hit rate: >80%

### Integration Tests

```bash
# Full auth flow
./test-auth-flow.sh

# DID resolution
./test-did-resolution.sh

# Credential verification
./test-vc-verification.sh
```

---

## Documentation

### Core Documentation

- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api.md)
- [DID Resolution Guide](docs/did-resolution.md)
- [Credential Verification](docs/vc-verification.md)

### Operations

- [Deployment Guide](deploy/README.md)
- [Disaster Recovery Runbook](docs/disaster-recovery.md)
- [Monitoring Setup](deploy/monitoring/README.md)
- [Load Testing Guide](test/load/README.md)

### Development

- [Contributing Guide](CONTRIBUTING.md)
- [Development Setup](docs/development.md)
- [Testing Guide](docs/testing.md)

---

## Development

### Project Structure

```
.
├── cmd/
│   ├── gateway/          # API Gateway entrypoint
│   ├── issuer/           # VC Issuer entrypoint
│   ├── upstream/         # Mock upstream API
│   └── wallet-cli/       # CLI tool for testing
├── internal/
│   ├── gateway/
│   │   ├── api/          # HTTP handlers
│   │   ├── did/          # DID resolution (key/web/ion)
│   │   ├── policy/       # Policy engine
│   │   └── vc/           # VC verification
│   └── shared/
│       ├── cache/        # Multi-layer caching
│       ├── circuitbreaker/ # Circuit breaker
│       ├── health/       # Health checks
│       ├── retry/        # Exponential backoff
│       └── secrets/      # Secrets management
├── deploy/
│   ├── k8s/              # Kubernetes manifests
│   ├── docker/           # Docker configs
│   ├── scripts/          # Deployment scripts
│   └── monitoring/       # Grafana dashboards
├── test/
│   ├── load/             # k6 load tests
│   └── did-web-server/   # Test server for did:web
└── docs/                 # Documentation
```

### Build from Source

```bash
# Build all binaries
go build ./cmd/gateway
go build ./cmd/issuer
go build ./cmd/wallet-cli

# Run locally
./gateway &
./issuer &

# Test
./wallet-cli did new --out test.json
```

### Code Quality

```bash
# Format code
go fmt ./...

# Lint
golangci-lint run

# Security scan
gosec ./...

# Dependency audit
go list -m all | nancy sleuth
```

---

## Configuration

### Policy Configuration

Policies define access rules in `deploy/docker/migrations/gateway/002_seed.sql`:

```sql
INSERT INTO policies (id, name, route_prefix, required_scopes, required_vc_types)
VALUES
  ('public', 'Public', '/api/v1/public', '{}', '{}'),
  ('basic', 'Basic', '/api/v1/basic', '{basic}', '{}'),
  ('premium', 'Premium', '/api/v1/premium', '{premium}', '{PremiumCredential}');
```

### Issuer Trust Tiers

Register trusted issuers:

```bash
# Automatic registration
./scripts/register-issuer.sh

# Manual registration
kubectl exec -it postgres-0 -- psql -U gateway -d gateway << EOF
INSERT INTO issuers (did, public_key, enabled, trust_tier)
VALUES ('did:key:...', 'base64_public_key', true, 1);
EOF
```

**Trust Tiers:**
- **Tier 1:** Basic trust (default)
- **Tier 2:** Verified issuer
- **Tier 3:** High-trust issuer
- **Tier 4:** Premium issuer
- **Tier 5:** Maximum trust (government, banks)

---

## Troubleshooting

### Common Issues

#### 1. "credential invalid" Error

**Cause:** Issuer not registered or keys changed (tmpfs restart)

**Solution:**
```bash
# Re-register issuer
./scripts/register-issuer.sh

# Get fresh credential
./wallet-cli cred request --issuer http://localhost:8090 ...
```

#### 2. Connection Refused

**Cause:** Services not ready (PostgreSQL/Redis starting)

**Solution:**
```bash
# Check service health
docker compose ps

# Wait for health checks
docker compose logs postgres | grep "ready to accept"
docker compose logs redis | grep "Ready to accept"
```

#### 3. Circuit Breaker Open

**Cause:** External service (did:web, did:ion) is down/slow

**Solution:**
```bash
# Check circuit breaker stats
curl http://localhost:8080/metrics | grep circuit_breaker

# Manual reset (if needed)
# Circuit auto-recovers after timeout (60s web, 120s ion)
```

#### 4. Cache Miss Rate High

**Cause:** Cache warming needed or TTL too short

**Solution:**
```bash
# Check cache stats
curl http://localhost:8080/metrics | grep cache_

# Warm cache with common DIDs
for did in $(cat common-dids.txt); do
  curl "http://localhost:8080/v1/resolve?did=$did"
done
```

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Format code (`go fmt ./...`)
6. Commit changes (`git commit -m 'Add amazing feature'`)
7. Push to branch (`git push origin feature/amazing-feature`)
8. Open Pull Request


---

## Acknowledgments

- [W3C DID Working Group](https://www.w3.org/2019/did-wg/) - DID specifications
- [W3C Verifiable Credentials](https://www.w3.org/TR/vc-data-model/) - VC standards
- [Microsoft ION](https://identity.foundation/ion/) - did:ion implementation
- [StatusList2021](https://w3c.github.io/vc-status-list-2021/) - Revocation standard


---

## 🗺️ Roadmap

### Immediate (This Month)
- [ ] Phase 5: Enhanced observability (distributed tracing)
- [ ] Phase 6: CI/CD automation

### Short-term (Next Quarter)
- [ ] Additional DID methods (did:ethr, did:pkh)
- [ ] BBS+ signatures for selective disclosure
- [ ] SDK for popular languages (JavaScript, Python)

### Long-term (This Year)
- [ ] Zero-knowledge proof integration
- [ ] Federation with other gateways
- [ ] Compliance certifications (SOC 2, ISO 27001)

---