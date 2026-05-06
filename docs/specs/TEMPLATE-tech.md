# TECH: <feature or issue title>

**Product spec:** `docs/specs/<id>/PRODUCT.md`
**Issue:** <GitHub issue / internal ticket / none>

## Context

Describe the current AutoOps implementation and cite relevant files.

- `api/...` — <current behavior>
- `web/...` — <current behavior>
- `api/pkg/db/migrate.go` — mention when GORM models change

## Proposed Changes

### Backend / API
- Files: `api/...`
- API/RBAC changes: <permission codes, routes, middleware>
- DB changes: <models, migration registration, PostgreSQL considerations>

### Frontend
- Files: `web/...`
- API calls: do not hard-code `/api/v1`; request interceptor adds it.
- UI states: loading, empty, error, success.

### Deploy / K8s / Infra
- Files: `docker/**`, `deploy/**`, `.github/workflows/**` if applicable.
- Rollback or operational considerations.

## Testing and Validation

Map each important PRODUCT invariant to verification.

| Invariant | Verification | File / Command |
|---|---|---|
| BI-1 | <unit/integration/manual> | `<path>` or command |
| BI-2 | <unit/integration/manual> | `<path>` or command |

Required baseline commands when relevant:

```bash
cd api && go test ./... -v -count=1
cd web && npm run lint && npm run build
./scripts/check-migrations  # for GORM model/schema changes
```

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| <DB migration risk> | <rollback/baseline/check> |
| <RBAC/security risk> | <permission/audit/log check> |
| <K8s/deploy risk> | <smoke/rollback check> |

## Follow-ups

- <deferred cleanup or observability work>
