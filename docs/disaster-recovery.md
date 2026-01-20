# Disaster Recovery Runbook

**Last Updated:** January 20, 2026  
**Version:** 1.0

## Overview

This runbook provides step-by-step procedures for recovering from various disaster scenarios in the DID Gateway infrastructure.

**Recovery Time Objective (RTO):** 30 minutes  
**Recovery Point Objective (RPO):** 15 minutes (last backup)

---

## Scenario 1: Complete Database Loss

### Symptoms
- Gateway logs show database connection errors
- PostgreSQL pod not starting
- Data corruption detected

### Recovery Steps

#### 1. Assess the Situation
```bash
# Check PostgreSQL pod status
kubectl get pods -l app=postgres

# Check logs
kubectl logs -l app=postgres --tail=100

# Verify database is truly unrecoverable
kubectl exec -it postgres-0 -- psql -U gateway -d gateway -c "SELECT 1;"
```

#### 2. Identify Latest Valid Backup
```bash
# List available backups
aws s3 ls s3://my-did-gateway-backups/postgres-backups/ --recursive | sort

# Or check local backups
ls -lh /backups/postgres/
```

#### 3. Restore from Backup
```bash
# Download latest backup (if from S3)
aws s3 cp s3://my-did-gateway-backups/postgres-backups/latest.sql.gz /tmp/

# Run restore script
./deploy/scripts/restore-postgres.sh /tmp/latest.sql.gz

# Or restore to new database instance
POSTGRES_HOST=new-db-host ./deploy/scripts/restore-postgres.sh /tmp/latest.sql.gz
```

#### 4. Update Application Configuration
```bash
# Update connection string if using new database
kubectl set env deployment/did-gateway \
  POSTGRES_DSN="postgres://gateway:password@new-db:5432/gateway"

# Or create new secret
kubectl create secret generic gateway-secrets \
  --from-literal=postgres-dsn="postgres://gateway:password@new-db:5432/gateway" \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart gateway pods to pick up changes
kubectl rollout restart deployment/did-gateway
```

#### 5. Verify Restoration
```bash
# Check data integrity
kubectl exec -it postgres-0 -- psql -U gateway -d gateway << 'EOF'
SELECT count(*) FROM policies;
SELECT count(*) FROM issuers;
SELECT count(*) FROM revocation_lists;
EOF

# Test authentication flow
./wallet-cli auth challenge --gateway http://gateway:8080 --did "did:key:z6Mk..."
```

#### 6. Monitor
```bash
# Watch gateway logs
kubectl logs -f deployment/did-gateway

# Check metrics
curl http://gateway:9090/metrics | grep postgres
```

**Estimated Downtime:** 15-30 minutes

---

## Scenario 2: Redis Sentinel Failover

### Symptoms
- Redis master unavailable
- Sentinel logs show failover activity
- Caching not working

### Recovery Steps

#### 1. Identify Current Master
```bash
# Query Sentinel for master info
kubectl exec -it redis-0 -- redis-cli -p 26379 SENTINEL get-master-addr-by-name mymaster

# Check Sentinel status
kubectl exec -it redis-0 -- redis-cli -p 26379 SENTINEL masters
```

#### 2. Verify Failover
```bash
# Sentinel should auto-failover, wait 30 seconds
sleep 30

# Check new master
kubectl exec -it redis-0 -- redis-cli -p 26379 SENTINEL get-master-addr-by-name mymaster
```

#### 3. Verify Application Connectivity
```bash
# Gateway should reconnect automatically
# Check logs for reconnection
kubectl logs -f deployment/did-gateway | grep redis

# Test caching
curl http://gateway:8080/v1/auth/challenge?did=did:key:z6Mk...
```

#### 4. Manual Failover (if needed)
```bash
# Force failover to specific replica
kubectl exec -it redis-0 -- redis-cli -p 26379 \
  SENTINEL failover mymaster
```

**Estimated Downtime:** <10 seconds (automatic)

---

## Scenario 3: Zone Failure

### Symptoms
- Pods in one zone are down
- HPA scaling up in remaining zones
- Increased latency

### Recovery Steps

#### 1. Verify Zone Status
```bash
# Check node status
kubectl get nodes -L topology.kubernetes.io/zone

# Check pod distribution
kubectl get pods -o wide -l app=did-gateway
```

#### 2. Monitor Auto-Recovery
```bash
# HPA should auto-scale in healthy zones
kubectl get hpa did-gateway-hpa -w

# Watch pod creation
kubectl get pods -l app=did-gateway -w
```

#### 3. Verify Traffic Distribution
```bash
# Check service endpoints
kubectl get endpoints did-gateway

# Test from multiple clients
for i in {1..10}; do 
  curl -s http://gateway:8080/healthz | jq .
  sleep 1
done
```

#### 4. No Action Usually Required
- Kubernetes and HPA handle automatically
- Monitor for 10-15 minutes
- Ensure healthy pods serve traffic

**Estimated Downtime:** 0 seconds (unless database/Redis also affected)

---

## Scenario 4: Complete Cluster Failure

### Symptoms
- Entire Kubernetes cluster unreachable
- All services down

### Recovery Steps

#### 1. Deploy New Cluster
```bash
# Set up new Kubernetes cluster (cloud provider specific)
# AWS EKS example:
eksctl create cluster --name did-gateway-dr --region us-west-2 --nodes 6

# Configure kubectl
aws eks update-kubeconfig --name did-gateway-dr --region us-west-2
```

#### 2. Restore Database
```bash
# Create new PostgreSQL instance or deploy StatefulSet
kubectl apply -f deploy/k8s/postgres-ha.yaml

# Restore latest backup
aws s3 cp s3://my-did-gateway-backups/postgres-backups/latest.sql.gz /tmp/
kubectl cp /tmp/latest.sql.gz postgres-0:/tmp/
kubectl exec -it postgres-0 -- bash
# Inside pod:
gunzip < /tmp/latest.sql.gz | pg_restore -U gateway -d gateway
```

#### 3. Deploy Redis Sentinel
```bash
kubectl apply -f deploy/k8s/redis-sentinel.yaml

# Wait for all 3 replicas to be ready
kubectl wait --for=condition=ready pod -l app=redis-sentinel --timeout=300s
```

#### 4. Deploy Gateway
```bash
# Create secrets
kubectl create secret generic gateway-secrets \
  --from-literal=postgres-dsn="..." \
  --from-literal=jwt-secret="..."

# Deploy gateway
kubectl apply -f deploy/k8s/gateway-ha.yaml
kubectl apply -f deploy/k8s/hpa.yaml

# Wait for rollout
kubectl rollout status deployment/did-gateway
```

#### 5. Update DNS
```bash
# Get new load balancer IP
kubectl get svc did-gateway-lb

# Update DNS records to point to new IP
# (Provider specific - Route53, CloudFlare, etc.)
```

#### 6. Verify Full Functionality
```bash
# Run smoke tests
./test/load/run-tests.sh # Option 3: Quick smoke test

# Test authentication
./wallet-cli auth verify --gateway https://new-gateway.example.com --wallet test.json
```

**Estimated Downtime:** 1-2 hours

---

## Scenario 5: Data Corruption

### Symptoms
- Inconsistent query results
- Application errors about invalid data
- Constraint violations in logs

### Recovery Steps

#### 1. Identify Corruption Scope
```bash
# Check for constraint violations
kubectl exec -it postgres-0 -- psql -U gateway -d gateway << 'EOF'
SELECT 
  schemaname, 
  tablename, 
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
EOF
```

#### 2. Restore Specific Tables
```bash
# Extract specific tables from backup
gunzip < backup.sql.gz | \
  pg_restore --table=policies --clean -U gateway -d gateway

# Or restore to temp database and copy
pg_restore -U gateway -d gateway_temp < backup.sql.gz
psql -U gateway << 'EOF'
BEGIN;
TRUNCATE policies CASCADE;
INSERT INTO policies SELECT * FROM gateway_temp.policies;
COMMIT;
EOF
```

#### 3. Verify Data Integrity
```bash
# Run integrity checks
kubectl exec -it postgres-0 -- psql -U gateway -d gateway << 'EOF'
-- Check for orphaned records
SELECT COUNT(*) FROM issuers WHERE did NOT LIKE 'did:%';

-- Verify foreign key constraints
SELECT conname, conrelid::regclass, confrelid::regclass
FROM pg_constraint
WHERE contype = 'f';
EOF
```

**Estimated Downtime:** 5-15 minutes (table-level)

---

## Recovery Contacts

| Role | Name | Contact |
|------|------|---------|
| On-Call Engineer | Rotate | +1-XXX-XXX-XXXX |
| Database Admin | TBD | email@example.com |
| DevOps Lead | TBD | email@example.com |
| Product Owner | TBD | email@example.com |

---

## Post-Recovery Checklist

After any disaster recovery:

- [ ] Document what happened and root cause
- [ ] Update this runbook with lessons learned
- [ ] Verify all backups are working
- [ ] Test the recovery procedure in staging
- [ ] Review and update monitoring alerts
- [ ] Conduct post-mortem meeting
- [ ] Update disaster recovery plan

---

## Backup Verification Schedule

| Frequency | Activity | Owner |
|-----------|----------|-------|
| Daily | Automated backups run | CronJob |
| Weekly | Verify backup exists in S3 | On-call |
| Monthly | Test restore to staging | DevOps |
| Quarterly | Full DR drill | Team |

---

## Additional Resources

- [Production Readiness Plan](./production-readiness-plan.md)
- [Monitoring Setup](../deploy/monitoring/README.md)
- [Backup Scripts](../deploy/scripts/)
- [Kubernetes Manifests](../deploy/k8s/)

---

*This is a living document. Update after each incident or DR test.*
