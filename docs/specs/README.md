# Spec-Driven Development

This directory contains product and technical specs for significant AutoOps changes.
Specs are an input-quality tool for humans and agents, not ceremony. Keep them short,
behavior-oriented, and synchronized with the implementation that ships.

## When to Write Specs

**Required for:**
- Cross-module changes (for example deploy + CMDB + auth interaction)
- Database schema / GORM model / migration changes
- Security-sensitive work (auth, RBAC, secrets, audit logs)
- Deploy/K8s workflow changes where rollback or approval behavior matters
- Features likely to be handed off to another human or AI agent
- >200 LOC or behaviorally ambiguous changes

**Skip for:**
- Narrow bug fixes with clear reproduction and regression test
- <200 LOC straightforward changes
- Mechanical refactors with no behavior change
- Single-file UI copy/style tweaks

## Directory Structure

```
docs/specs/
├── TEMPLATE-product.md      # Product behavior template
├── TEMPLATE-tech.md         # Technical implementation and validation template
└── <issue-or-feature>/
    ├── PRODUCT.md
    └── TECH.md
```

Use an issue id (`gh-123`), internal ticket id, or short kebab-case feature name for `<issue-or-feature>`.

## Workflow

1. **Create Product Spec first** (`PRODUCT.md`)
   - Define user-facing or API-facing behavior.
   - Use numbered, testable behavior invariants (`BI-1`, `BI-2`, ...).
   - Capture edge cases a reasonable implementer might miss.

2. **Create Tech Spec when warranted** (`TECH.md`)
   - Ground the plan in current AutoOps files and module boundaries.
   - Call out DB/RBAC/API/K8s/security implications.
   - Map each important product invariant to a validation step.

3. **Implement in the same PR when practical**
   - Keep PRODUCT/TECH updated as decisions change.
   - Specs should describe what actually ships, not stale intent.

4. **Verify against specs**
   - Behavior must match PRODUCT.md invariants.
   - Architecture and validation must match TECH.md.

## Spec Format

### PRODUCT.md
- **Summary**: 1-3 sentences.
- **Problem**: Clear pain point if not obvious.
- **User Scenarios**: Who does what, and why.
- **Goals / Non-goals**: Scope boundaries.
- **Behavior Invariants**: Numbered, testable behavior statements (the core of the spec).
- **Acceptance Criteria**: Verifiable outcomes that reference invariants.
- **Open Questions**: Only if unresolved decisions remain.

### TECH.md
- **Context**: Current system and relevant files.
- **Proposed Changes**: Modules, types, APIs, DB/RBAC/K8s changes, data flow.
- **Testing and Validation**: How to verify each invariant.
- **Risks and Mitigations**: Failure modes, rollback, migration, permission, or observability risks.

## Validation Commands

Use the smallest command set that proves the shipped behavior:

- Backend: `cd api && go test ./... -v -count=1`
- Frontend: `cd web && npm run lint && npm run build`
- Migration checker, if model/schema changed: `./scripts/check-migrations`

Do not use `npm run test:unit` unless the frontend test framework and script exist.

## Quality Bar

- Every important behavior has an invariant.
- Every important invariant has a validation method.
- GORM model changes mention `api/pkg/db/migrate.go`.
- Frontend API examples do not include a hard-coded `/api/v1` prefix.
- Security-sensitive changes mention logs/secrets/RBAC/audit implications.
