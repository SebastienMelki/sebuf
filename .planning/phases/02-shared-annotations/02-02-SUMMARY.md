---
phase: 02-shared-annotations
plan: 02
subsystem: api
tags: [protogen, annotations, httpgen, migration, code-deduplication]

# Dependency graph
requires:
  - phase: 02-shared-annotations plan 01
    provides: internal/annotations package with shared types and functions
provides:
  - httpgen fully migrated to shared annotations package
  - 580 lines of duplicated annotation code removed from httpgen
  - Validation that shared annotations API covers the most complex generator
affects: [02-shared-annotations plans 03-04 (clientgen, tsclientgen, openapiv3 migration)]

# Tech tracking
tech-stack:
  added: []
  patterns: [shared annotation imports across generator boundaries]

key-files:
  created: []
  modified:
    - internal/httpgen/generator.go
    - internal/httpgen/unwrap.go
    - internal/httpgen/validation.go
    - internal/httpgen/mock_generator.go
    - internal/httpgen/generator_test.go
    - internal/httpgen/golden_test.go
    - internal/httpgen/unwrap_test.go
  deleted:
    - internal/httpgen/annotations.go (392 lines)
    - internal/httpgen/annotations_test.go (188 lines)

key-decisions:
  - "annotations.GetServiceBasePath returns string directly -- simplified getServiceBasePath to single-line delegation"
  - "Removed parseExistingAnnotation dead code (always returned empty string) during migration"
  - "TestLowerFirst and BenchmarkLowerFirst removed from httpgen -- covered by shared package tests"
  - "Test files (golden_test.go, unwrap_test.go) updated to reference annotations.HTTPConfig, annotations.QueryParam, annotations.UnwrapValidationError"

# Metrics
duration: 6min
completed: 2026-02-05
---

# Phase 2 Plan 2: Migrate httpgen to Shared Annotations Summary

**httpgen fully migrated to internal/annotations with 580 lines of duplicated annotation code deleted, all golden file tests passing unchanged confirming zero behavior change**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-05T16:54:53Z
- **Completed:** 2026-02-05T17:00:49Z
- **Tasks:** 2/2
- **Lines deleted:** 580 (392 from annotations.go + 188 from annotations_test.go)

## Accomplishments
- Migrated all 4 httpgen source files (generator.go, unwrap.go, validation.go, mock_generator.go) to use `internal/annotations` package
- Deleted `internal/httpgen/annotations.go` (392 lines) containing 9 duplicated annotation parsing functions
- Deleted `internal/httpgen/annotations_test.go` (188 lines) with tests already covered by shared package
- Validated shared package API covers httpgen's full annotation surface area (the most complex generator)
- All 4 golden file tests pass unchanged -- zero behavior change confirmed
- Full test suite passes (6/6 packages)
- Lint clean (0 issues)

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace httpgen annotation calls with shared package** - `b9e6e0d` (refactor)
   - Updated imports in generator.go, unwrap.go, validation.go, mock_generator.go
   - Replaced all local annotation calls with `annotations.GetMethodHTTPConfig`, `annotations.GetServiceBasePath`, etc.
   - Removed `lowerFirst` function definition, replaced with `annotations.LowerFirst`
   - Removed `parseExistingAnnotation` dead code
   - Removed `TestLowerFirst`/`BenchmarkLowerFirst` from generator_test.go

2. **Task 2: Delete annotations.go and annotations_test.go** - `355632c` (feat)
   - Deleted `internal/httpgen/annotations.go` (392 lines)
   - Deleted `internal/httpgen/annotations_test.go` (188 lines)
   - Updated `golden_test.go` and `unwrap_test.go` to reference shared types
   - Full test suite verification: all tests pass, golden files unchanged

## Files Created/Modified
- **Deleted:** `internal/httpgen/annotations.go` (392 lines of duplicated annotation parsing)
- **Deleted:** `internal/httpgen/annotations_test.go` (188 lines of tests now in shared package)
- **Modified:** `internal/httpgen/generator.go` -- imports annotations package, uses shared functions
- **Modified:** `internal/httpgen/unwrap.go` -- uses `annotations.UnwrapFieldInfo`, `annotations.GetUnwrapField`
- **Modified:** `internal/httpgen/validation.go` -- uses `annotations.GetMethodHTTPConfig`, `annotations.QueryParam`
- **Modified:** `internal/httpgen/mock_generator.go` -- uses `annotations.GetFieldExamples`
- **Modified:** `internal/httpgen/generator_test.go` -- removed lowerFirst tests (covered in shared package)
- **Modified:** `internal/httpgen/golden_test.go` -- type references updated to annotations package
- **Modified:** `internal/httpgen/unwrap_test.go` -- type references updated to annotations package

## Decisions Made
- **GetServiceBasePath simplification:** httpgen's `getServiceBasePath` method simplified to single-line delegation to `annotations.GetServiceBasePath(service)` since the shared function already returns empty string for missing config.
- **Dead code removal:** `parseExistingAnnotation` was dead code (always returned `""`) -- removed during migration instead of porting to shared package.
- **Test deduplication:** `TestLowerFirst`, `BenchmarkLowerFirst`, `TestHttpMethodToString`, `TestExtractPathParams`, `TestHTTPConfig_Struct`, `TestQueryParam_Struct`, `TestServiceConfigImpl_Struct`, and all related benchmarks removed from httpgen since they test functions/types now in the shared package (and already covered by `internal/annotations/annotations_test.go`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed test files referencing deleted types**
- **Found during:** Task 2 (after deleting annotations.go)
- **Issue:** `golden_test.go` referenced `HTTPConfig` and `QueryParam` types; `unwrap_test.go` referenced `UnwrapValidationError` -- all now in shared package
- **Fix:** Added `annotations` import to both test files, updated type references to `annotations.HTTPConfig`, `annotations.QueryParam`, `annotations.UnwrapValidationError`
- **Files modified:** `internal/httpgen/golden_test.go`, `internal/httpgen/unwrap_test.go`
- **Commit:** `355632c`

**Total deviations:** 1 auto-fixed (blocking issue in test files)
**Impact on plan:** Minimal -- test files were not listed in plan's `files_modified` but required updates for compilation.

## Issues Encountered
None -- migration executed cleanly with all tests passing.

## User Setup Required
None.

## Next Phase Readiness
- httpgen fully migrated, validating the shared package API is complete for the most complex generator
- Ready for plan 02-03: clientgen migration (simpler annotation surface area)
- Ready for plan 02-04: tsclientgen + openapiv3 migration
- All generators still function correctly (full test suite passes)

---
*Phase: 02-shared-annotations*
*Completed: 2026-02-05*
