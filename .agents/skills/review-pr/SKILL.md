---
name: review-pr
description: Review a pull request for correctness, security, and quality in the AutoOps codebase. Use when reviewing a PR, checking code before merge, or producing structured review feedback.
---

# review-pr

Review a pull request in the AutoOps codebase and produce structured feedback.

## Review Scope

Prioritize correctness, security, error handling, and meaningful performance issues. Include style or nit comments only when you can provide a concrete suggestion.

Review dimensions (ordered by importance):

1. **Correctness** — Does the code do what it claims? Are edge cases handled?
2. **Security** — Any credential leaks, injection risks, auth bypasses, or audit log gaps? Check against `docs/security-audit.md` if the change touches auth, data, secrets, or logging.
3. **AutoOps conventions** — GORM models registered in migrate.go? RBAC permission codes follow `module:sub:action`? Frontend API calls don't hard-code `/api/v1`? Error codes in the right segment?
4. **Error handling** — Are errors surfaced to the user appropriately? Are they logged with enough context for debugging?
5. **Performance** — Any obvious N+1 queries, missing indexes, or unnecessary allocations?
6. **Style** — Only when a concrete suggestion block is provided.

## Review Output Format

Use severity labels:

- **CRITICAL** — Bugs, security issues, data loss, crashes. Must fix before merge.
- **IMPORTANT** — Logic problems, missing edge cases, missing error handling. Should fix before merge.
- **SUGGESTION** — Worthwhile improvements or better patterns. Optional.
- **NIT** — Cleanup only when a concrete suggestion block is included.

Output format:

```markdown
## Review Summary

**Verdict:** Approve / Approve with nits / Request changes

Found: X critical, Y important, Z suggestions

### Concerns
- <high-level concerns that span multiple files>

## Inline Comments

### `path/to/file.go`

**Line 42** — IMPORTANT: <explanation>

` ` `suggestion
replacement code
` ` `

### `web/src/path/to/Component.vue`

**Line 15** — SUGGESTION: <explanation>
```

## Pre-Review Checks

Before diving into the diff:
1. Run `./scripts/presubmit` from the PR branch to verify fmt, vet, test, lint, and build all pass.
2. If presubmit fails, include the failure output in the review and set verdict to Request changes immediately.
3. Check that the PR template's Review Checklist items are addressed.

## AutoOps-Specific Checks

- **GORM migration**: Does every new `TableName()` model appear in `api/pkg/db/migrate.go` or have an allowlist rationale?
- **RBAC**: Do new API routes have permission codes registered? Are codes in `module:sub:action` format?
- **Frontend API**: No hard-coded `/api/v1` prefixes? (request.js interceptor adds it)
- **Error codes**: New error codes in the 470+ segment? Defined in `api/common/constant/constant.go`?
- **JWT**: If auth changes touch JWT, is `JWT_SECRET` handled correctly (panic in production if unset)?
