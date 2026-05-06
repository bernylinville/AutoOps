---
name: write-product-spec
description: Write a PRODUCT.md spec for a significant feature in AutoOps, focused on detailed behavior invariants and validation. Use when the user asks for a product spec, desired behavior doc, or PRD, wants to define feature behavior before implementation, or when the feature is substantial or behaviorally ambiguous.
---

# write-product-spec

Write a `PRODUCT.md` spec for a significant feature in AutoOps.

## Overview

The product spec should make the desired behavior unambiguous enough that an agent can implement it correctly and avoid regressions. Describe the feature purely from the user's perspective — what the user sees, does, and experiences, and the invariants that must hold for them. Do not include implementation details (internal types, state layout, module boundaries, data flow, algorithms).

"User" is not limited to the end user of the AutoOps web UI:
- For UI/UX features: the human using AutoOps.
- For an API endpoint: the caller of that API — frontend code, CLI tools, bots, or external services.
- For a data model: the code that reads and writes that model.
- For an infrastructure change: the operator or deploy pipeline consuming it.

### When specs are needed

Specs are required for changes that are:
- Cross-module (touching multiple api/api/{module}/ directories)
- Behavioral or ambiguous (no clear "right answer" from code alone)
- Risk-bearing (DB schema, RBAC, deploy pipeline, security, K8s)
- >200 LOC expected

Specs are often unnecessary for:
- Small, local bug fixes
- Straightforward refactors (no behavior change)
- Narrow UI tweaks with little ambiguity
- Single-file changes with no cross-cutting impact

## Structure

Write to `docs/specs/<id>/PRODUCT.md` where `<id>` is a GitHub issue number or short kebab-case feature name. Do not use engineer names as directory names.

### Required sections

1. **Summary** — 1-3 sentences describing the feature and desired outcome.
2. **Behavior** — The core of the spec. Numbered, testable invariants written in English. See "The Behavior section" below.

### Optional sections

Include only when they add signal. Omit the heading entirely if empty.

- **Problem** — Only when the motivation is not obvious from Summary.
- **Goals / Non-goals** — When scope is ambiguous or contested.
- **Figma** — Link or `none provided` for UI work.
- **Open questions** — When there are unresolved design decisions.

Do not include Validation, Success criteria, or Testing sections. Those live in the companion TECH.md (produced by `write-tech-spec`).

## The Behavior section

Behavior is the spec. Everything else is framing.

Describe, at minimum:
- Default behavior and the happy-path user flow.
- Every user-visible state and the transitions between them.
- All inputs and how the feature responds.
- Empty states, error states, loading/pending states, and cancellation.
- Edge cases: permission denied (RBAC), stale data, concurrent requests, missing data, timeout, retry, offline.
- Invariants that must hold at all times and behaviors that must not regress.

Write Behavior as numbered invariants:

```markdown
## Behavior

1. When <condition/action>, AutoOps must <observable behavior>.
2. When <edge case>, AutoOps must <safe behavior> and must not <forbidden behavior>.
3. For RBAC denial, AutoOps must return HTTP 403 with audit log containing <fields>.
```

Cover the relevant AutoOps-specific states: loading/pending, empty table/list, error (API error, network error, timeout), permission denied (RBAC), stale cache, concurrent modification, deployment in progress.

## Keep the spec current

Approved specs ship in the same PR as the implementation. When implementation diverges from the spec, update the spec — do not leave it stale.

## Related Skills

- `write-tech-spec` — companion TECH.md for implementation planning
- `spec-driven-implementation` — full spec→code workflow
