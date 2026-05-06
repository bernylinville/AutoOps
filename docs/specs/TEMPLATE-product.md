# PRODUCT: <feature or issue title>

**Issue:** <GitHub issue / internal ticket / none>
**Figma:** <link or `none provided` for UI work>

## Summary

1-3 sentences describing the desired outcome.

## Problem

Include only when the motivation is not obvious.

## User Scenarios

- **US-1:** As a <role>, I want <capability>, so that <outcome>.
- **US-2:** As a <role>, I want <capability>, so that <outcome>.

## Goals / Non-goals

**Goals:**
- <goal>

**Non-goals:**
- <explicitly excluded scope>

## Behavior Invariants

Write numbered, testable invariants. These are the source of truth for implementation and review.

- **BI-1:** When <condition/action>, AutoOps must <observable behavior>.
- **BI-2:** If <edge case>, AutoOps must <safe behavior> and must not <forbidden behavior>.
- **BI-3:** For permission/RBAC failure, AutoOps must <expected response/log/audit behavior>.

Cover relevant states: default, loading/pending, empty, error, cancellation, retry, permission denied, stale data, concurrent requests, and adjacent feature interactions.

## Acceptance Criteria

- [ ] BI-1 is verified by <unit/integration/manual check>.
- [ ] BI-2 is verified by <unit/integration/manual check>.
- [ ] UI changes include screenshot/video evidence, or a reason visual evidence is unnecessary.

## Open Questions

- <question, owner, decision deadline>
