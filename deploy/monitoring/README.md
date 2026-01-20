# Monitoring Setup for DID Gateway

This directory contains monitoring configuration for the DID Gateway.

## Components

### Grafana Dashboards
- **`grafana-dashboard-overview.json`** - Main overview dashboard with:
  - Request rate (RPS)
  - Error rate gauge
  - Latency percentiles (p50, p95, p99)
  - Auth verification metrics
  - DID cache hit rate
  - VC verification metrics

### Prometheus Alerts
- **`prometheus-alerts.yaml`** - Alert rules for:
  - **Critical**: Gateway down, high error rate, high latency, database failures
  - **Warning**: Rate limits, DID resolution failures, low cache hit rate, resource usage
  - **Info**: Pod restarts
  - **SLOs**: 99.9% availability, p99 latency < 200ms

## Quick Start

### 1. Deploy Prometheus & Grafana

```bash
# Add Prometheus & Grafana Helm repos
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install Prometheus (with AlertManager)
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus -n monitoring --timeout=300s
```

### 2. Apply Alert Rules

```bash
# Create ConfigMap with alert rules
kubectl create configmap prometheus-did-gateway-rules \
  --from-file=prometheus-alerts.yaml \
  --namespace monitoring

# Or apply as PrometheusRule custom resource (if using Prometheus Operator)
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: did-gateway-alerts
  namespace: monitoring
spec:
  $(cat prometheus-alerts.yaml)
EOF
```

### 3. Import Grafana Dashboard

```bash
# Get Grafana admin password
kubectl get secret prometheus-grafana -n monitoring -o jsonpath="{.data.admin-password}" | base64 --decode

# Port-forward to Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 -n monitoring

# Open browser: http://localhost:3000
# Login: admin / <password from above>
# Go to: Dashboards → Import → Upload JSON file
# Select: grafana-dashboard-overview.json
```

### 4. Configure ServiceMonitor

```bash
# Apply ServiceMonitor for automatic service discovery
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: did-gateway
  namespace: default
  labels:
    app: did-gateway
spec:
  selector:
    matchLabels:
      app: did-gateway
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
EOF
```

## Metrics Exposed

The DID Gateway exposes the following custom metrics:

### HTTP Metrics
- `http_requests_total` - Total HTTP requests (labels: method, path, code)
- `http_request_duration_seconds` - Request duration histogram

### Authentication Metrics
- `auth_challenge_total` - Challenge requests
- `auth_verify_total` - Verification attempts (labels: status, did_method, vc_type)

### DID Resolution Metrics
- `did_resolve_total` - Total DID resolutions (labels: method)
- `did_resolve_cache_hits_total` - Cache hits
- `did_resolve_errors_total` - Resolution errors (labels: method, error_type)
- `did_resolve_duration_seconds` - Resolution duration histogram

### VC Verification Metrics
- `vc_verify_total` - VC verification attempts (labels: status)
- `vc_verify_duration_seconds` - Verification duration

### Rate Limiting Metrics
- `rate_limit_exceeded_total` - Rate limit violations (labels: did)

### Policy Metrics
- `policy_denials_total` - Policy denials (labels: policy_name, reason)

## Accessing Dashboards

### Port Forward (Development)
```bash
# Prometheus
kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 -n monitoring

# Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 -n monitoring

# AlertManager
kubectl port-forward svc/prometheus-kube-prometheus-alertmanager 9093:9093 -n monitoring
```

### Production URLs
- **Grafana**: https://grafana.yourdomain.com
- **Prometheus**: https://prometheus.yourdomain.com (internal only)
- **AlertManager**: https://alerts.yourdomain.com (internal only)

## Alert Routing

Configure AlertManager to route alerts:

```yaml
# alertmanager-config.yaml
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  
  routes:
    # Critical alerts → PagerDuty
    - match:
        severity: critical
      receiver: pagerduty
      
    # Warnings → Slack
    - match:
        severity: warning
      receiver: slack
      
    # Info → Slack (low priority)
    - match:
        severity: info
      receiver: slack-low-priority

receivers:
  - name: 'default'
    # Default receiver
    
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: '<PAGERDUTY_SERVICE_KEY>'
        
  - name: 'slack'
    slack_configs:
      - api_url: '<SLACK_WEBHOOK_URL>'
        channel: '#did-gateway-alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

## SLO Dashboard

Key SLO metrics to track:

- **Availability SLO**: 99.9% (43 minutes downtime/month)
  - Query: `1 - (sum(rate(http_requests_total{code=~"5.."}[30d])) / sum(rate(http_requests_total[30d])))`

- **Latency SLO**: p99 < 200ms
  - Query: `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`

- **Error Budget**: 0.1% (=99.9% availability)
  - Remaining: `ERROR_BUDGET - ERRORS_ACTUAL`

## Troubleshooting

### No metrics showing up
1. Check ServiceMonitor is created: `kubectl get servicemonitor -n default`
2. Check Prometheus targets: http://localhost:9090/targets
3. Verify app exposes `/metrics`: `curl http://did-gateway:8080/metrics`

### Alerts not firing
1. Check Prometheus rules: http://localhost:9090/rules
2. Verify AlertManager config: http://localhost:9093
3. Check alert state: http://localhost:9090/alerts

### Dashboard not loading
1. Check Grafana datasource configuration
2. Verify Prometheus URL in datasource
3. Test connectivity: `curl http://prometheus-kube-prometheus-prometheus:9090/api/v1/query?query=up`

## Next Steps

1. **Add custom panels** to dashboard for your specific metrics
2. **Tune alert thresholds** based on your traffic patterns
3. **Set up log aggregation** (ELK/Loki) for detailed troubleshooting
4. **Configure dashboards for each component** (Issuer, Upstream)
5. **Create runbooks** for each alert type
