---
phase: 01-foundation-quick-wins
plan: 02
subsystem: project-management
tags: [github-issues, documentation, issue-cleanup, unwrap, json-name]

dependency-graph:
  requires: []
  provides:
    - "GitHub issues #91 and #94 closed with documentation"
    - "Clean issue tracker for Phase 1 completion"
  affects: []

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

decisions:
  - id: "01-02-D1"
    decision: "Issue #91 resolved by existing (sebuf.http.unwrap) = true annotation -- no new feature needed"
    rationale: "Root-level array serialization already works via unwrap annotation on repeated fields with full test coverage"
  - id: "01-02-D2"
    decision: "Issue #94 resolved by proto3 built-in json_name field option -- no sebuf-specific annotation needed"
    rationale: "Proto3 already provides per-field JSON name override; adding a sebuf annotation would create confusion"

metrics:
  duration: "<1 minute"
  completed: "2026-02-05"
---

# Phase 01 Plan 02: Issue Cleanup Summary

**Closed GitHub issues #91 (root-level arrays) and #94 (field name casing) with documentation comments explaining existing solutions**

## Performance

| Metric | Value |
|--------|-------|
| Duration | <1 minute |
| Tasks | 2/2 |
| Code commits | 0 (no code changes) |
| Issues closed | 2 |

## Accomplishments

### Task 1: Close issue #91 (root-level arrays) with documentation

Closed GitHub issue #91 with a detailed comment documenting that the existing `(sebuf.http.unwrap) = true` annotation on repeated fields already provides root-level array serialization. The comment includes:
- Proto usage example
- Explanation of generated MarshalJSON/UnmarshalJSON methods
- Links to implementation files (unwrap.go, unwrap.proto, unwrap_test.go)
- Cross-generator coverage (go-http, go-client, openapiv3)

### Task 2: Verify issue #94 (field name casing) is properly closed

Confirmed issue #94 was already closed by @SebastienMelki with an adequate comment explaining:
- Proto3's built-in `json_name` field option for per-field override
- `protojson.MarshalOptions{UseProtoNames: true}` for global snake_case
- Rationale that adding a sebuf-specific annotation would create confusion

No additional action was required.

## Task Commits

No code commits -- this plan was purely GitHub issue management.

## Files Created/Modified

None. This plan involved no code changes.

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 01-02-D1 | Issue #91 resolved by existing unwrap annotation | Root-level array serialization already works via `(sebuf.http.unwrap) = true` with full test coverage |
| 01-02-D2 | Issue #94 resolved by proto3 `json_name` | Proto3 already provides per-field JSON name override; sebuf-specific annotation would create confusion |

## Deviations from Plan

None -- plan executed exactly as written.

## Issues

None.

## Next Phase Readiness

- Issue tracker is clean for Phase 1 completion
- Both issues confirmed as non-blockers (existing features cover the use cases)
- No outstanding issues blocking Phase 2 work
