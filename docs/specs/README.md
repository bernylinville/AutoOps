# Spec-Driven Development

This directory contains product and technical specs for significant features.

## When to Write Specs

**Required for:**
- Cross-module changes (deploy + CMDB + auth interaction)
- Database schema changes
- Security-sensitive work
- Features likely to be handed off
- >200 LOC or behaviorally ambiguous changes

**Skip for:**
- Bug fixes
- <200 LOC changes
- Straightforward refactors
- Single-file UI tweaks

## Directory Structure

```
docs/specs/
├── TEMPLATE-product.md      # Product spec template
├── TEMPLATE-tech.md         # Technical spec template
└── examples/                # Example specs for reference
    └── knowledge-base/
        ├── PRODUCT.md
        └── TECH.md
```

## Workflow

1. **Create Product Spec first** (`PRODUCT.md`)
   - Define user-facing behavior
   - Numbered, testable invariants
   - Edge cases and success criteria

2. **Create Tech Spec when warranted** (`TECH.md`)
   - Implementation plan grounded in codebase
   - Architecture, tradeoffs, testing strategy
   - Reference PRODUCT.md for behavior

3. **Implement in same PR**
   - Keep specs updated as implementation evolves
   - Specs describe what actually ships

4. **Verify against specs**
   - Ensure behavior matches PRODUCT.md invariants
   - Ensure architecture matches TECH.md plan

## Spec Format

### PRODUCT.md
- **Summary**: 1-3 sentences
- **Problem**: Clear pain point (if not obvious)
- **Goals / Non-goals**: Scope boundaries
- **Behavior**: Numbered invariants (the core of the spec)
- **Edge Cases**: What a reasonable implementer might miss
- **Success Criteria**: Verifiable outcomes

### TECH.md
- **Context**: Current system and relevant files
- **Proposed Changes**: Modules, types, APIs, data flow
- **Testing and Validation**: How to verify each invariant
- **Risks**: Failure modes and mitigations

## Example

See `examples/knowledge-base/` for a complete product + tech spec pair.
