---
phase: 07-json-structural-transforms
plan: 04
subsystem: testing
tags: [cross-generator, consistency, oneof, discriminator, flatten, golden-files]

# Dependency graph
requires:
  - phase: 07-02
    provides: oneof_discriminator golden files for all 4 generators
  - phase: 07-03
    provides: flatten golden files for all 4 generators
provides:
  - Cross-generator consistency tests for oneof discriminator annotation
  - Cross-generator consistency tests for flatten annotation
  - Final Phase 7 validation confirming all 4 generators agree
affects: [08-go-client-language, 09-ts-client-language, 10-openapi-language]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Oneof discriminator consistency test pattern with helper functions"
    - "Flatten consistency test pattern with containsInInterface helper"
    - "verifyOpenAPIFlattenedProperties helper for allOf schema validation"

key-files:
  created:
    - internal/httpgen/oneof_discriminator_consistency_test.go
    - internal/httpgen/flatten_consistency_test.go
  modified: []

key-decisions:
  - "D-07-04-01: Helper functions verifyOneofDiscriminatorPresent/Absent to stay under nestif complexity limit"
  - "D-07-04-02: containsInInterface helper for precise TypeScript interface field verification"
  - "D-07-04-03: 800-char window for NestedEvent OpenAPI lookup (600 was insufficient for deeply nested YAML)"

patterns-established:
  - "Structural transform consistency tests follow same 4-test split pattern as encoding tests"
  - "Cross-generator agreement uses table-driven tests with helper dispatch for complex verification"

# Metrics
duration: 7min
completed: 2026-02-06
---

# Phase 7, Plan 4: Cross-Generator Consistency Tests Summary

**Oneof discriminator and flatten cross-generator consistency verified across go-http, go-client, ts-client, and OpenAPI with 8 targeted test functions**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-02-06T19:27:32Z
- **Completed:** 2026-02-06T21:24:16Z
- **Tasks:** 3/3
- **Files created:** 2

## Accomplishments
- Byte-level equality verified between go-http and go-client for both oneof_discriminator and flatten
- TypeScript discriminated union types (FlattenedEvent, NestedEvent) match Go serialization including custom oneof_values
- OpenAPI schemas correctly use oneOf + discriminator keyword with propertyName and mapping for all variant combinations
- OpenAPI flatten schemas correctly use allOf composition with flattened properties and prefixes
- All Phase 4, 5, 6 consistency tests still pass (zero regression)
- Phase 7 success criterion 6 met: cross-generator consistency test confirms agreement for all structural transforms

## Task Commits

Each task was committed atomically:

1. **Task 1: Create oneof_discriminator cross-generator consistency tests** - `0503226` (test)
2. **Task 2: Create flatten cross-generator consistency tests** - `6c522f5` (test)
3. **Task 3: Run full test suite and verify all golden files** - (verification only, no commit)

**Plan metadata:** (pending)

## Files Created/Modified
- `internal/httpgen/oneof_discriminator_consistency_test.go` - 4 test functions verifying oneof discriminator across all generators
- `internal/httpgen/flatten_consistency_test.go` - 4 test functions verifying flatten across all generators

## Decisions Made
- D-07-04-01: Extracted verifyOneofDiscriminatorPresent/Absent helper functions to stay under golangci-lint nestif complexity limit (14 > threshold)
- D-07-04-02: Created containsInInterface helper to precisely verify TypeScript fields exist within specific interface bodies rather than anywhere in the file
- D-07-04-03: Increased OpenAPI YAML window size to 800 chars for NestedEvent schema (original 600 chars truncated before reaching discriminator mapping section)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed OpenAPI window size for NestedEvent discriminator verification**
- **Found during:** Task 1 (oneof_discriminator consistency tests)
- **Issue:** 600-char window from `NestedEvent:` position in YAML cut off at `mappi` before reaching the `vid:` discriminator mapping entry
- **Fix:** Increased window to 800 characters to capture the full NestedEvent schema including all discriminator mappings
- **Files modified:** internal/httpgen/oneof_discriminator_consistency_test.go
- **Verification:** TestOneofDiscriminatorOpenAPISchemas passes
- **Committed in:** 0503226 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed nestif complexity lint error**
- **Found during:** Task 1 (oneof_discriminator consistency tests)
- **Issue:** TestOneofDiscriminatorCrossGeneratorAgreement had nested complexity of 14, exceeding golangci-lint threshold
- **Fix:** Extracted discriminator present/absent verification into verifyOneofDiscriminatorPresent and verifyOneofDiscriminatorAbsent helper functions
- **Files modified:** internal/httpgen/oneof_discriminator_consistency_test.go
- **Verification:** `make lint-fix` reports 0 issues
- **Committed in:** 0503226 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both auto-fixes necessary for test correctness and lint compliance. No scope creep.

## Issues Encountered
None -- all golden files existed and matched expectations.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 7 (JSON Structural Transforms) is now fully complete
- All 4 plans executed: annotations (07-01), generator implementation (07-02, 07-03), cross-generator consistency (07-04)
- All 4 generators produce consistent output for oneof discriminator and flatten
- Ready to proceed to Phase 8 (Go Client Language) or Phase 9 (TS Client Language)

---
*Phase: 07-json-structural-transforms*
*Completed: 2026-02-06*
