# Load Testing Guide

k6-based load testing suite for the DID Gateway Phase 3 scalability testing.

## Quick Start

```bash
cd test/load
chmod +x run-tests.sh
./run-tests.sh
```

## Prerequisites

The script will auto-install k6 on Linux/macOS. For manual installation:

```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

## Test Scenarios

### 1. Auth Flow Load Test (`auth-flow.js`)

Tests the complete authentication flow including DID resolution, challenge/response, and VC verification.

**Load Profile:**
- Warm up: 100 VUs for 2 min
- Baseline: 1000 VUs for 5 min
- Peak: 5000 VUs for 5 min sustained
- Spike: 10000 VUs for 1 min
- Cool down: Scale to 0

**Thresholds:**
- P99 latency < 100ms
- Error rate < 1%
- Challenge latency < 50ms (P95)
- Verify latency < 150ms (P95)

**Run:**
```bash
GATEWAY_HOST=localhost:8080 k6 run auth-flow.js
```

### 2. DID Resolution Test (`did-resolution.js`)

Tests DID resolution performance and cache hit rates.

**Load Profile:**
- 50 → 500 → 1000 VUs
- Mixed traffic: 80% did:key, 20% did:web
- 10 minute duration

**Thresholds:**
- P99 latency < 100ms
- Cache hit rate > 80%
- Error rate < 1%

**Run:**
```bash
GATEWAY_HOST=localhost:8080 k6 run did-resolution.js
```

### 3. Quick Smoke Test

```bash
k6 run --vus 100 --duration 1m auth-flow.js
```

## Configuration

Set environment variables to customize:

```bash
# Target gateway
export GATEWAY_HOST=gateway.example.com:443

# Run specific test
./run-tests.sh
```

## Interpreting Results

### Key Metrics

**http_req_duration:**
- P50: Median latency
- P95: 95th percentile
- P99: 99th percentile (target <100ms)

**http_req_failed:**
- Error rate (target <1%)

**Custom Metrics:**
- `challenge_latency`: Time to get challenge
- `verify_latency`: Time to verify and get token
- `resolution_latency`: DID resolution time
- `cache_hits`: Cache hit rate

### Example Output

```
✓ challenge status 200
✓ challenge latency < 50ms
✓ verify latency < 150ms

checks.........................: 99.5% ✓ 29850  ✗ 150
data_received..................: 15 MB  1.5 MB/s
data_sent......................: 12 MB  1.2 MB/s
http_req_duration..............: avg=45ms min=5ms  med=40ms max=250ms p(90)=80  p(95)=95  p(99)=98
http_req_failed................: 0.5%  ✓ 150    ✗ 29850
iterations.....................: 30000  3000/s
vus............................: 5000   max 5000
```

## Performance Targets

| Metric | Target | Acceptable |
|--------|--------|------------|
| Peak RPS | 5,000 | 3,000 |
| P99 Latency (auth) | < 100ms | < 150ms |
| P99 Latency (proxy) | < 50ms | < 75ms |
| Error Rate | < 0.1% | < 1% |
| Cache Hit Rate | > 80% | > 60% |

## Troubleshooting

### High Latency

1. Check database query performance
2. Verify cache hit rates
3. Check HPA scaling status
4. Review Grafana dashboards

### High Error Rates

1. Check gateway logs: `kubectl logs -f deployment/did-gateway`
2. Verify database/Redis connectivity
3. Check resource limits (CPU/memory)
4. Review Prometheus alerts

### Low Cache Hit Rate

1. Verify cache configuration
2. Check cache size limits
3. Review TTL settings
4. Ensure cache warming on startup

## Integration with CI/CD

```yaml
# .github/workflows/load-test.yml
name: Load Test
on:
  schedule:
    - cron: '0 2 * * *'  # Nightly
jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install k6
        run: |
          curl https://github.com/grafana/k6/releases/download/v0.48.0/k6-v0.48.0-linux-amd64.tar.gz -L | tar xvz
          sudo mv k6-v0.48.0-linux-amd64/k6 /usr/local/bin/
      - name: Run load test
        env:
          GATEWAY_HOST: staging.example.com
        run: |
          cd test/load
          k6 run auth-flow.js
```

## Next Steps

After running load tests:

1. **Analyze Results**: Review p99 latency, error rates, and bottlenecks
2. **Tune HPA**: Adjust min/max replicas and thresholds
3. **Optimize Cache**: Tune L1/L2 cache sizes based on hit rates
4. **Database**: Add indexes for slow queries
5. **Production**: Gradually roll out with canary deployment
