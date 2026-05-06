---
name: go-modern-practices
description: Enforce modern Go 1.25+ idioms, standard-library patterns, and AutoOps-specific conventions when writing or reviewing Go code. Use when writing new Go code, refactoring, or reviewing PRs.
---

# go-modern-practices

Apply modern Go idioms and AutoOps conventions to all Go code in this project. This skill is the authoritative reference — when it conflicts with existing code in the repo, the skill wins.

## Standard Library Upgrades (Go 1.18–1.25)

### IPv6-safe addressing

```go
// BEFORE: breaks on IPv6
addr := fmt.Sprintf("%s:%d", host, port)

// AFTER: IPv4/IPv6 safe
addr := net.JoinHostPort(host, fmt.Sprint(port))
```

### Replace `interface{}` with `any`

```go
// BEFORE
func get(key string) interface{} { ... }

// AFTER
func get(key string) any { ... }
```

### Structured logging with `log/slog`

```go
// BEFORE
log.Println("deploy failed:", err)

// AFTER
slog.Error("deploy failed", "error", err, "app_id", appID)
```

Prefer `slog` over `log` and `fmt.Println` for all new code. Use `slog.Info` for operational events, `slog.Warn` for recoverable issues, `slog.Error` for failures.

### Error wrapping with `%w` and `errors.Join`

```go
// BEFORE
return fmt.Errorf("failed: %v", err)

// AFTER
return fmt.Errorf("build image %s: %w", imageName, err)

// Multiple errors
return errors.Join(err1, err2)
```

### `cmp.Or` for defaults (Go 1.22+)

```go
// BEFORE
timeout := userTimeout
if timeout == 0 {
    timeout = defaultTimeout
}

// AFTER
timeout := cmp.Or(userTimeout, defaultTimeout)
```

### `maps` and `slices` packages (Go 1.21+)

Use `maps.Clone`, `maps.Copy`, `slices.Contains`, `slices.SortFunc` instead of hand-written loops.

### Context propagation

```go
// ALWAYS pass context to DB queries and external calls
func (d *Dao) GetByID(ctx context.Context, id uint) (*Model, error) {
    return d.db.WithContext(ctx).First(&Model{}, id).Error
}
```

## Package Migration Targets

| Current | Target | Priority |
|---------|--------|----------|
| `github.com/dgrijalva/jwt-go` v3 | `github.com/golang-jwt/jwt/v5` | P0 — archived, unmaintained |
| `gopkg.in/yaml.v2` | `gopkg.in/yaml.v3` | P2 — v2 still works, migrate opportunistically |

### JWT v5 migration notes

```go
// BEFORE (dgrijalva/jwt-go)
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
token.SignedString(Secret)

// AFTER (golang-jwt/jwt/v5)
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
token.SignedString(Secret) // API is compatible, but import changes
```

The API is largely compatible. Main change: import path. Validate all token parsing call sites after migration.

## AutoOps-Specific Conventions

### GORM models

- Every `TableName()` model MUST be registered in `api/pkg/db/migrate.go`
- Unregistered models require an allowlist entry in `scripts/check-migrations` with rationale
- Use `gorm:"..."` tags for column definitions, `json:"..."` for serialization

### RBAC

- Permission codes: `module:sub:action` (e.g., `cmdb:sql:select`)
- Applied per-route via `middleware.RbacMiddleware("code")`
- New routes must declare a permission code; do not create routes without RBAC

### Error codes

- Defined in `api/common/constant/constant.go`
- Next available segment: 470+
- Allocated: 400-434 (general), 440-456 (CI), 460-465 (project)

### Frontend API calls

- Never hard-code `/api/v1` prefix in frontend — `request.js` interceptor adds it
- Use relative paths in Vue API calls

### Config

- Uses `gopkg.in/yaml.v2` with `-c config.yaml` flag
- Not Viper. Follow the existing pattern in `common/db.go` and `common/redis.go`

## Pre-Commit Checklist

Before committing Go changes:

```bash
cd api
go fmt ./...          # formatting
go vet ./...          # static analysis
golangci-lint run     # lint (enforced in CI)
go test ./... -count=1  # tests
```

Full presubmit:
```bash
./scripts/presubmit
```

## When to Use This Skill

Reference this skill when:
- Writing new Go code in any AutoOps module
- Reviewing a PR that touches `api/`
- Refactoring existing code to modern idioms
- Adding a new dependency — check against migration targets first
- Diagnosing CI failures related to `go vet` or `golangci-lint`
