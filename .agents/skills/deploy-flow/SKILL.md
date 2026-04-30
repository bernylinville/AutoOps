---
name: deploy-flow
description: Deploy an application through the AutoOps pipeline. Use when releasing a new version, hotfixing production, or onboarding a new project to the deploy system.
---

# deploy-flow

Deploy an application through the AutoOps pipeline (Jenkins → Harbor → Kubernetes).

## When to Use

- Releasing a new version of an application
- Hotfixing a production issue
- Onboarding a new project to AutoOps deploy system
- Debugging a failed deployment

## Prerequisites

Before starting:

1. Confirm the project has a `Jenkinsfile` in its repository
2. Confirm the project is registered in AutoOps (Project → Deploy Profile)
3. Confirm GitLab credentials are configured in Jenkins
4. Confirm Harbor image repository exists and robot account has push permission

## Workflow

### 1. Prepare the Release

```bash
# 1. Ensure you're on the correct branch
git checkout main && git pull origin main

# 2. Update version (follow semver: patch for bugfix, minor for feature, major for breaking)
# Edit version in relevant files (e.g., pom.xml for Java, package.json for Node)

# 3. Commit version bump
git add .
git commit -m "chore: bump version to x.y.z"
git push origin main
```

### 2. Trigger Build via AutoOps

Option A: Web UI
1. Log in to AutoOps → Project → Deploy
2. Select project and environment (dev/staging/prod)
3. Click "Build" to trigger Jenkins pipeline
4. Wait for build number assignment

Option B: API / Bot
```bash
# Trigger deploy via API (if bot integration is configured)
curl -X POST "https://autoops.com.cn/api/v1/deploy/build" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"project_id": "xxx", "environment": "prod", "branch": "main"}'
```

### 3. Monitor Build Progress

```bash
# Check Jenkins build status
# Build stages: checkout → build → test → Jib push → update deploy record

# If build fails, check:
# 1. Jenkins console output for compilation errors
# 2. Harbor push logs (network / credential issues)
# 3. Maven/Gradle dependency resolution
```

### 4. Verify Deployment

```bash
# Check Kubernetes deployment status
kubectl get deployment $APP_NAME -n $NAMESPACE
kubectl get pods -n $NAMESPACE -l app=$APP_NAME

# Verify image tag matches expected version
kubectl get deployment $APP_NAME -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}'

# Check application health
curl https://autoops.com.cn/api/v1/healthz
```

### 5. Rollback (if needed)

```bash
# In AutoOps UI: Project → Deploy → History → select previous version → Rollback
# Or via Kubernetes directly:
kubectl rollout undo deployment/$APP_NAME -n $NAMESPACE
```

## Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| Build stuck at "building" | Jenkins agent not started | Check Jenkins agent pod status |
| Harbor push failed | Network or credential | Verify Jenkins credential ID matches Harbor robot account |
| ImagePullBackOff | Wrong image tag or missing image | Verify image exists in Harbor with correct tag |
| DB migration failed | Migration not registered | Check `api/pkg/db/migrate.go` for model registration |

## Verification Checklist

- [ ] Version bump committed and pushed
- [ ] Jenkins build completed successfully
- [ ] Harbor image pushed with correct tag
- [ ] Kubernetes pods running with new image
- [ ] Application health check passes
- [ ] (If hotfix) Rollback plan confirmed

## Related

- `db-migration` — If deployment includes database changes
- `incident-recovery` — If deployment causes production issues
