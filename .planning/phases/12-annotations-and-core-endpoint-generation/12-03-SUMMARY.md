---
phase: 12-annotations-and-core-endpoint-generation
plan: 03
subsystem: api
tags: [protobuf, krakend, protoc-plugin, code-generation, gateway, header-forwarding, query-forwarding]

# Dependency graph
requires:
  - phase: 12-02
    provides: "GenerateService function, Endpoint/Backend structs with InputHeaders/InputQueryStrings fields, annotation reading patterns"
provides:
  - "Auto-derived InputHeaders from sebuf.http.service_headers and method_headers annotations with merge semantics"
  - "Auto-derived InputQueryStrings from sebuf.http.query annotations on request message fields"
  - "Deterministic sorted output for golden file stability"
affects: [12-04]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Reuse existing annotation extractors (GetServiceHeaders, GetMethodHeaders, CombineHeaders, GetQueryParams) instead of re-implementing", "Nil-not-empty-slice convention for omitempty JSON omission"]

key-files:
  created: []
  modified:
    - "internal/krakendgen/generator.go"

key-decisions:
  - "Reuse annotations.CombineHeaders for header merge semantics rather than re-implementing -- method overrides service for same-name headers"
  - "Return nil (not empty slice) when no annotations exist, so omitempty JSON tag omits the field entirely"
  - "Sort all output lists for deterministic golden file comparison"

patterns-established:
  - "deriveInputHeaders/deriveInputQueryStrings pattern: small unexported helpers that extract annotation data and return nil-or-sorted-list"

requirements-completed: [FWD-01, FWD-02, FWD-03]

# Metrics
duration: 3min
completed: 2026-02-25
---

# Phase 12 Plan 03: Header and Query String Forwarding Summary

**Auto-derived KrakenD input_headers from service/method header annotations and input_query_strings from query annotations with merge semantics and deterministic sort**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-25T13:24:31Z
- **Completed:** 2026-02-25T13:27:13Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Added deriveInputHeaders helper that reads service-level and method-level headers via CombineHeaders, extracts names, and populates InputHeaders on each endpoint
- Added deriveInputQueryStrings helper that reads query annotations from request message fields via GetQueryParams and populates InputQueryStrings on each endpoint
- Both helpers return nil when no annotations exist, leveraging omitempty to omit fields from JSON entirely
- Verified end-to-end: service headers, method header merge, query params, and empty omission all work correctly in generated JSON

## Task Commits

Each task was committed atomically:

1. **Task 1: Add header forwarding from service and method header annotations** - `48c9ab3` (feat)
2. **Task 2: Add query string forwarding from query annotations** - `cfb80e7` (feat)

## Files Created/Modified
- `internal/krakendgen/generator.go` - Added deriveInputHeaders and deriveInputQueryStrings helpers, integrated both into GenerateService loop

## Decisions Made
- Reuse annotations.CombineHeaders for merge semantics rather than re-implementing -- method-level headers override service-level headers with the same name, which is the existing behavior from the annotations package
- Return nil (not empty slice) when no annotations exist -- this ensures omitempty JSON tags omit the field entirely (FWD-03 requirement)
- Sort all output lists alphabetically for deterministic golden file comparison in Plan 04

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- golangci-lint panics on generated .pb.go files (pre-existing Go 1.24 vs 1.26 mismatch) -- not caused by plan changes. go vet passes on all changed packages. Same issue noted in 12-02 SUMMARY.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Generator now produces complete KrakenD endpoint JSON with routing, host/timeout, header forwarding, and query string forwarding
- Plan 04 will add golden file tests for regression detection across all features
- All forwarding features verified end-to-end with manual protoc execution

## Self-Check: PASSED
