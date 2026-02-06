---
phase: 05-json-nullable-empty
plan: 04
subsystem: testing
tags: [consistency, cross-generator, nullable, empty-behavior, golden-files]

# Dependency graph
requires:
  - phase: 05-02
    provides: "Nullable encoding across all 4 generators with golden files"
  - phase: 05-03
    provides: "Empty behavior encoding across go-http, go-client, openapiv3 with golden files"
  - phase: 04-05
    provides: "Phase 4 cross-generator consistency test pattern (encoding_consistency_test.go)"
provides:
  - "Cross-generator consistency tests for nullable annotation"
  - "Cross-generator consistency tests for empty_behavior annotation"
  - "Empty behavior test proto and golden files for clientgen"
affects: ["05-05"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cross-generator consistency: normalize generator comment, then byte-compare golden files"
    - "TypeScript nullable verification: T | null syntax vs optional ? syntax"
    - "OpenAPI nullable verification: type array [T, null] vs simple type"
    - "OpenAPI empty_behavior verification: oneOf for NULL, $ref for PRESERVE/OMIT"

key-files:
  created:
    - "internal/httpgen/nullable_consistency_test.go"
    - "internal/httpgen/empty_behavior_consistency_test.go"
    - "internal/clientgen/testdata/proto/empty_behavior.proto"
    - "internal/clientgen/testdata/golden/empty_behavior_client.pb.go"
    - "internal/clientgen/testdata/golden/empty_behavior_empty_behavior.pb.go"
  modified:
    - "internal/clientgen/golden_test.go"

key-decisions:
  - "D-05-04-01: Added empty_behavior test proto to clientgen (Rule 3 deviation) to enable byte-level golden file comparison"

patterns-established:
  - "Nullable consistency: go-http vs go-client byte comparison, TS T|null, OpenAPI type array"
  - "Empty behavior consistency: go-http vs go-client byte comparison, OpenAPI oneOf vs $ref"

# Metrics
duration: 4min
completed: 2026-02-06
---

# Phase 5 Plan 4: Cross-Generator Consistency Tests Summary

**Nullable and empty_behavior cross-generator consistency validated: byte-identical Go JSON, TypeScript T | null types, OpenAPI 3.1 type arrays and oneOf schemas**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-06T10:16:48Z
- **Completed:** 2026-02-06T10:21:00Z
- **Tasks:** 3/3
- **Files modified:** 6

## Accomplishments
- Validated go-http and go-client produce byte-identical nullable JSON encoding (after generator name normalization)
- Validated go-http and go-client produce byte-identical empty_behavior JSON encoding
- Verified TypeScript correctly uses T | null for nullable fields (not optional ?)
- Verified OpenAPI correctly uses type array [T, "null"] for nullable and oneOf for empty_behavior=NULL
- Added empty_behavior test proto and golden files to clientgen (was missing from 05-03)
- Zero regressions across all 4 generators

## Task Commits

Each task was committed atomically:

1. **Task 1: Nullable cross-generator consistency tests** - `47a9867` (test)
2. **Task 2: Empty behavior cross-generator consistency tests** - `ff65053` (test)
3. **Task 3: Full test suite verification** - (verification only, no commit)

## Files Created/Modified
- `internal/httpgen/nullable_consistency_test.go` - 4 test functions: GoHTTPvsGoClient, TypeScript, OpenAPI, BackwardCompat
- `internal/httpgen/empty_behavior_consistency_test.go` - 3 test functions: GoHTTPvsGoClient, OpenAPI, BackwardCompat
- `internal/clientgen/testdata/proto/empty_behavior.proto` - Test proto for clientgen empty_behavior golden generation
- `internal/clientgen/testdata/golden/empty_behavior_client.pb.go` - Clientgen empty_behavior client golden file
- `internal/clientgen/testdata/golden/empty_behavior_empty_behavior.pb.go` - Clientgen empty_behavior encoding golden file
- `internal/clientgen/golden_test.go` - Added empty_behavior test case

## Decisions Made
- **D-05-04-01:** Added empty_behavior.proto and golden files to clientgen testdata as a Rule 3 deviation. The 05-03 plan created the empty_behavior.go generator code in clientgen but not the test proto/golden files, which blocked byte-level golden file comparison. Identical proto file used (same pattern as nullable).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added empty_behavior test proto and golden files to clientgen**
- **Found during:** Task 1 (preparing for empty_behavior consistency test)
- **Issue:** clientgen had empty_behavior.go (generator code) but no test proto or golden files, making byte-level output comparison impossible
- **Fix:** Copied empty_behavior.proto from httpgen testdata, added test case to clientgen golden_test.go, generated golden files with UPDATE_GOLDEN=1
- **Files modified:** internal/clientgen/testdata/proto/empty_behavior.proto, internal/clientgen/golden_test.go, internal/clientgen/testdata/golden/empty_behavior_client.pb.go, internal/clientgen/testdata/golden/empty_behavior_empty_behavior.pb.go
- **Verification:** All clientgen golden tests pass including new empty_behavior case
- **Committed in:** 47a9867 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Essential for complete cross-generator consistency testing. No scope creep.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Cross-generator consistency fully validated for nullable and empty_behavior
- Ready for plan 05-05 (integration tests or phase completion)
- All existing tests continue to pass (zero regression)

---
*Phase: 05-json-nullable-empty*
*Completed: 2026-02-06*
