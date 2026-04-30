---
name: incident-recovery
description: Recover from common AutoOps production failures. Use when services are down, deployments fail, database issues occur, or critical alerts fire.
---

# incident-recovery

Recover from common AutoOps production failures.

## When to Use

- AutoOps web/API is unreachable
- Deployment failed and service is down
- Database connection errors
- Kubernetes pod crash loops
- Critical alerts firing (N9E/FlashDuty)
- Certificate expiry
- Disk/storage full

## Prerequisites

1. SSH access to production server or kubectl access to K8s cluster
2. Docker Compose or kubectl available
3. Access to logs (docker logs or kubectl logs)

## Common Scenarios

### Scenario 1: API Service Down

```bash
# 1. Check service status
docker compose ps
# OR
kubectl get pods -n autoops

# 2. Check logs
docker compose logs -f devops-api --tail 100
# OR
kubectl logs -n autoops deployment/devops-api --tail 100

# 3. Common causes & fixes:

# Cause A: PostgreSQL not ready
# Fix: Restart postgres and wait for health check
docker compose restart postgres

# Cause B: Migration failed on startup
# Fix: Check migration error, fix model, restart
# Look for "AutoMigrate failed" in logs

# Cause C: Port conflict
# Fix: Check port usage
ss -tlnp | grep 8000

# Cause D: Out of memory
# Fix: Restart with memory limit check
docker stats devops-api
```

### Scenario 2: Deployment Failed (Jenkins → K8s)

```bash
# 1. Check Jenkins build status
# AutoOps UI → Project → Deploy → Build History

# 2. If build succeeded but K8s deployment failed:
kubectl get events -n <namespace> --sort-by='.lastTimestamp'

# 3. Check pod status
kubectl describe pod <pod-name> -n <namespace>

# 4. Common K8s issues:
# ImagePullBackOff → Wrong image tag or Harbor credential
# CrashLoopBackOff → Application panic on startup
# Pending → Resource quota exceeded or node affinity issue

# 5. Quick rollback
kubectl rollout undo deployment/<app-name> -n <namespace>
```

### Scenario 3: Database Issues

```bash
# 1. Check PostgreSQL status
docker compose exec postgres pg_isready

# 2. Check connections
psql -h localhost -U devops -d autoops -c "SELECT count(*) FROM pg_stat_activity;"

# 3. If connection pool exhausted:
# Restart API to clear connections
docker compose restart devops-api

# 4. If disk full:
df -h
docker system prune -f  # Clean unused images

# 5. If slow queries:
psql -h localhost -U devops -d autoops -c "
SELECT query, mean_exec_time 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;
"
```

### Scenario 4: Certificate Expiry

```bash
# Check certificate expiry
echo | openssl s_client -servername autoops.com.cn -connect autoops.com.cn:443 2>/dev/null | openssl x509 -noout -dates

# If expired:
# 1. Renew certificate (Let's Encrypt or vendor)
# 2. Update secret in K8s or mount path in Docker
# 3. Restart web/API pods
```

### Scenario 5: Valkey/Cache Issues

```bash
# Check Valkey status
docker compose exec valkey valkey-cli ping

# If memory issues:
valkey-cli info memory
valkey-cli --eval "return redis.call('keys', '*')" 0 | wc -l  # Count keys

# Clear specific cache if corrupted
valkey-cli del "autoops:session:*"
```

## Recovery Decision Tree

```
Service Down?
├── API unreachable → Check docker/k8s status → Check logs → Restart
├── Deployment failed → Check Jenkins → Check K8s events → Rollback if needed
├── Database error → Check PG status → Check connections → Restart/fix
├── Alert firing → Check N9E/FlashDuty → Identify metric → Scale/fix
└── Certificate expiry → Check dates → Renew → Update secret → Restart
```

## Post-Recovery Actions

1. **Document incident** in FlashDuty or incident log
2. **Add monitoring** if new failure mode discovered
3. **Update runbook** if recovery steps changed
4. **Schedule root cause analysis** if severity ≥ P1

## Verification Checklist

- [ ] Service health check passes (`curl /healthz` returns 200)
- [ ] Key user flows work (login, dashboard, CMDB list)
- [ ] No critical alerts firing
- [ ] Logs show normal operation (no ERROR/FATAL)
- [ ] If rollback performed, previous version is stable
- [ ] Incident documented with timeline and root cause

## Related

- `deploy-flow` — If recovery involves re-deployment
- `db-migration` — If database schema fix needed
