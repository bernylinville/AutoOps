---
name: write-tech-spec
description: Write a TECH.md spec for a significant AutoOps feature after researching the current codebase and implementation constraints. Use when the user asks for a technical spec, implementation plan, or architecture doc tied to a product spec.
---

# write-tech-spec

Write a `TECH.md` spec for a significant feature in AutoOps.

## Overview

The tech spec is the implementation plan, grounded in the current codebase. It translates the product spec's behavioral invariants into concrete changes to Go backend code, Vue frontend code, database schemas, and infrastructure configuration.

Tech specs should be written by agents, informed by current codebase patterns. Reference specific files and line numbers. Use `grep` and file reads to ground the spec in real code, not assumptions.

## Prerequisites

- A completed `PRODUCT.md` in `docs/specs/<id>/PRODUCT.md` (use `write-product-spec` first)
- Understanding of the AutoOps architecture (read `docs/architecture.md` if needed)

## Structure

Write to `docs/specs/<id>/TECH.md`.

### Required sections

1. **Context** — Current state of the relevant subsystems with file references.
2. **Proposed Changes** — Backend, Frontend, and Infra subsections.
3. **Testing and Validation** — Invariant → verification mapping table.
4. **Risks and Mitigations** — Concrete risks with mitigations.

### Context

Cite specific files and their current behavior:

```markdown
- `api/api/cmdb/model/ciType.go` — CI type model, reference for new model conventions
- `api/api/cmdb/dao/ciType.go` — DAO pattern to follow
- `api/pkg/db/migrate.go` — GORM model registration (must be updated if adding models)
- `api/router/cmdb/cmdb.go` — Route registration pattern for RBAC
```

### Proposed Changes

#### Backend / API
- Models: new GORM structs, TableName(), registration in migrate.go
- DAOs: new data access patterns
- Services: business logic changes
- Controllers: new endpoints
- Routes: path, method, RBAC permission code
- Error codes: next available segment in `api/common/constant/constant.go`

#### Frontend
- New Vue components / pages
- API calls (do NOT hard-code `/api/v1` prefix; request.js interceptor adds it)
- UI states: loading, empty, error, success
- Vuex store changes (if any)

#### Deploy / K8s / Infra
- Docker config changes
- Kubernetes manifests
- CI/CD workflow changes
- Rollback considerations

### Testing and Validation

Map each product spec invariant to a verification method:

```markdown
| Invariant | Verification | File / Command |
|-----------|-------------|----------------|
| BI-1: When user creates X, AutoOps returns Y | Unit test | `api/api/module/model/x_test.go` |
| BI-2: RBAC deny returns 403 | Integration test | `api/api/module/controller/x_test.go` |
```

Required baseline commands:
```bash
cd api && go test ./... -v -count=1
cd web && npm run lint && npm run build
./scripts/check-migrations  # if GORM models changed
./scripts/presubmit          # complete pre-PR check
```

### Risks and Mitigations

```markdown
| Risk | Mitigation |
|------|-----------|
| DB migration failure | Test migration rollback path; run check-migrations |
| RBAC misconfiguration | Verify with RbacMiddleware test; audit log review |
| Frontend API prefix duplication | Remove hard-coded /api/v1; let interceptor handle it |
| K8s deploy regression | Smoke test after deploy; keep rollback manifest ready |
```

## AutoOps-Specific Constraints

When writing tech specs, account for these project constraints:
- **GORM migration**: New models must be registered in `api/pkg/db/migrate.go`
- **RBAC**: Permission codes follow `module:sub:action` format, applied via `middleware.RbacMiddleware("code")`
- **Error codes**: Next available segment is 470+ (defined in `api/common/constant/constant.go`)
- **Frontend API**: Never hard-code `/api/v1` — request.js interceptor adds it
- **JWT**: Production requires `JWT_SECRET` env var
- **Config**: Uses `gopkg.in/yaml.v2`, loaded with `-c config.yaml`

## Related Skills

- `write-product-spec` — prerequisite PRODUCT.md
- `spec-driven-implementation` — full spec→code workflow
