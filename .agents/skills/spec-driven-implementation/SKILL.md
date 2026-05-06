---
name: spec-driven-implementation
description: Drive a spec-first workflow for substantial features in AutoOps by writing PRODUCT.md before implementation, writing TECH.md when warranted, and keeping both specs updated as implementation evolves. Use when starting a significant feature, planning agent-driven implementation, or when product and tech specs should be checked into source control.
---

# spec-driven-implementation

Drive a spec-first workflow for substantial features in AutoOps.

## Overview

Use this skill for significant features where a written spec will improve implementation quality, reduce ambiguity, or make review easier. Be pragmatic: not every change needs specs.

Specs live in `docs/specs/<id>/`:
- `docs/specs/<id>/PRODUCT.md` — user-facing behavior
- `docs/specs/<id>/TECH.md` — implementation plan

Use a GitHub issue number or kebab-case feature name as `<id>`. Do not use engineer names.

## When specs are required

Strongly prefer specs when:
- Product or architectural ambiguity exists
- Expected implementation size around 200+ LOC
- Deep or cross-cutting stack changes (touches multiple api/api/{module}/ directories)
- Risky behavior changes where regressions would be expensive (DB schema, RBAC, deploy pipeline, K8s)
- Work where agent quality will improve materially from clearer inputs

Specs are often unnecessary for:
- Small, local bug fixes
- Straightforward refactors
- Narrow UI tweaks with little ambiguity
- Single-file changes with no cross-cutting impact

## Workflow

### 1. Decide whether the feature needs specs

Evaluate the size, ambiguity, and risk of the feature. If specs will not meaningfully improve execution or review, skip them and focus on verification instead.

### 2. Write the product spec first

Use `write-product-spec` to create `PRODUCT.md`. The product spec defines:
- What problem is being solved
- The desired user experience
- Behavior invariants and edge cases
- RBAC, error states, and loading states where relevant

### 3. Write the tech spec when warranted

Use `write-tech-spec` for substantial or ambiguous implementation work. Prefer a tech spec when:
- The implementation spans multiple subsystems
- Architecture or extensibility matters
- There are meaningful tradeoffs to document (e.g., DB schema design, RBAC model, API shape)
- Reviewers will benefit more from reviewing the plan than the raw code

It is acceptable to write the tech spec after a prototype if that leads to a more accurate plan.

### 4. Implement approved specs

After specs are approved, implement from `PRODUCT.md` and `TECH.md`. Implementation can ship in the same PR as the specs. Keep `PRODUCT.md`, `TECH.md`, code changes, and tests in that same PR so review reflects the feature that will actually ship.

### 5. Keep specs current during implementation

If implementation diverges from the spec, update the spec rather than leaving it stale.

Update `PRODUCT.md` when user-facing behavior, success criteria, or edge cases change.

Update `TECH.md` when the implementation approach, architectural boundaries, risks, or testing plan change.

### 6. Verify behavior against the spec

Before considering work complete:
- Run `cd api && go test ./... -v -count=1` for backend changes
- Run `cd web && npm run lint && npm run build` for frontend changes
- Run `./scripts/check-migrations` for GORM model/schema changes
- Run `./scripts/presubmit` for complete pre-PR verification

## Best Practices

- Be pragmatic above all else.
- Write specs to improve input quality for agents, not as ceremony.
- Keep product specs behavior-oriented and implementation-light.
- Keep tech specs implementation-oriented and grounded in current codebase patterns (reference specific files with line numbers in `api/` and `web/`).
- Use review time to validate specs and behavior, not to over-index on code style.

## Related Skills

- `write-product-spec`
- `write-tech-spec`
