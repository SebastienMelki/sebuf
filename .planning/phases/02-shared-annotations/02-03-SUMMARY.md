---
phase: 02-shared-annotations
plan: 03
subsystem: api
tags: [protogen, annotations, clientgen, tsclientgen, migration, code-deduplication]

# Dependency graph
requires:
  - phase: 02-shared-annotations plan 02
    provides: httpgen migrated to shared annotations, validating the shared package API
provides:
  - clientgen fully migrated to shared annotations package
  - tsclientgen fully migrated to shared annotations package
  - 491 lines of duplicated annotation code removed (241 clientgen + 250 tsclientgen)
  - 3 of 4 generators now use shared annotations
affects: [02-shared-annotations plan 04 (openapiv3 migration)]

# Tech tracking
tech-stack:
  added: []
  patterns: [shared annotation imports across all client generators]

key-files:
  created: []
  modified:
    - internal/clientgen/generator.go
    - internal/tsclientgen/generator.go
    - internal/tsclientgen/types.go
    - internal/tsclientgen/helpers.go
    - internal/tsclientgen/helpers_test.go
  deleted:
    - internal/clientgen/annotations.go (241 lines)
    - internal/tsclientgen/annotations.go (251 lines)

key-decisions:
  - "BuildHTTPPath safe for both generators: httpPath always initialized to '/' + lowerFirst(methodName) before path building, so empty-path divergence is unreachable"
  - "TestLowerFirst removed from tsclientgen/helpers_test.go -- covered by shared package tests"
  - "snakeToUpperCamel kept in clientgen (used for URL path param building) -- generator-specific naming logic"
  - "snakeToLowerCamel and headerNameToPropertyName kept in tsclientgen -- generator-specific naming helpers"

# Metrics
duration: 5min
completed: 2026-02-05
---

# Phase 2 Plan 3: Migrate clientgen and tsclientgen to Shared Annotations Summary

**clientgen and tsclientgen fully migrated to internal/annotations with 491 lines of duplicated annotation code deleted, all golden file tests passing unchanged confirming zero behavior change across both generators**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-05T17:03:56Z
- **Completed:** 2026-02-05T17:09:16Z
- **Tasks:** 2/2
- **Lines deleted:** 491 (241 from clientgen/annotations.go + 250 from tsclientgen/annotations.go)

## Accomplishments
- Migrated clientgen/generator.go to use shared `internal/annotations` package for all annotation parsing
- Migrated tsclientgen/generator.go, types.go, and helpers.go to use shared `internal/annotations` package
- Replaced inline path building in both generators with `annotations.BuildHTTPPath`
- Deleted both `annotations.go` files (491 total lines of duplicated code)
- Removed `findUnwrapField` and `isRootUnwrap` definitions from types.go (now in shared package)
- Removed `lowerFirst` definition from helpers.go (now in shared package)
- All 7 golden file tests (3 clientgen + 4 tsclientgen) pass unchanged -- zero behavior change confirmed
- Full test suite passes (6/6 packages)
- Lint clean (0 issues)

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrate clientgen to shared annotations and delete annotations.go** - `105195c` (refactor)
   - Updated imports in generator.go, added `internal/annotations` import
   - Replaced all local annotation calls with shared package equivalents
   - Replaced inline path building with `annotations.BuildHTTPPath`
   - Replaced `getServiceHTTPConfig` pattern with `annotations.GetServiceBasePath`
   - Removed `lowerFirst` function definition
   - Updated `QueryParam` type references to `annotations.QueryParam`
   - Deleted `internal/clientgen/annotations.go` (241 lines)

2. **Task 2: Migrate tsclientgen to shared annotations and delete annotations.go** - `f708593` (refactor)
   - Updated imports in generator.go, types.go, added `internal/annotations` import
   - Replaced all local annotation calls with shared package equivalents
   - Replaced inline path building with `annotations.BuildHTTPPath`
   - Replaced `findUnwrapField` calls with `annotations.FindUnwrapField` in types.go
   - Removed `findUnwrapField` and `isRootUnwrap` definitions from types.go
   - Removed `lowerFirst` definition from helpers.go
   - Removed `TestLowerFirst` from helpers_test.go (covered by shared package)
   - Deleted `internal/tsclientgen/annotations.go` (251 lines)

## Files Created/Modified
- **Deleted:** `internal/clientgen/annotations.go` (241 lines of duplicated annotation parsing)
- **Deleted:** `internal/tsclientgen/annotations.go` (251 lines of duplicated annotation parsing)
- **Modified:** `internal/clientgen/generator.go` -- imports annotations package, uses shared functions
- **Modified:** `internal/tsclientgen/generator.go` -- imports annotations package, uses shared functions
- **Modified:** `internal/tsclientgen/types.go` -- uses `annotations.FindUnwrapField`, removed local definitions
- **Modified:** `internal/tsclientgen/helpers.go` -- removed lowerFirst (now in shared package)
- **Modified:** `internal/tsclientgen/helpers_test.go` -- removed TestLowerFirst (covered by shared package)

## Decisions Made
- **BuildHTTPPath safety:** Both generators always initialize httpPath to "/" + lowerFirst(methodName) before the path building logic, making the empty-path divergence between BuildHTTPPath and inline code unreachable. Safe to replace.
- **Generator-specific helpers kept:** `snakeToUpperCamel` (clientgen), `snakeToLowerCamel`, `headerNameToPropertyName`, `headerNameToFuncName` (tsclientgen) are generator-specific naming logic, not annotation-related. Kept in respective packages.
- **Test deduplication:** `TestLowerFirst` removed from tsclientgen since lowerFirst is now in shared package (tested in `internal/annotations/annotations_test.go`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed TestLowerFirst from helpers_test.go**
- **Found during:** Task 2 (tsclientgen migration)
- **Issue:** `TestLowerFirst` calls `lowerFirst` which was removed from helpers.go -- test would fail to compile
- **Fix:** Removed `TestLowerFirst` test function (lowerFirst already tested in shared package)
- **Files modified:** `internal/tsclientgen/helpers_test.go`
- **Commit:** `f708593`

**Total deviations:** 1 auto-fixed (blocking issue in test file)
**Impact on plan:** Minimal -- test file update was necessary for compilation after removing lowerFirst.

## Issues Encountered
None -- migration executed cleanly with all tests passing.

## User Setup Required
None.

## Next Phase Readiness
- 3 of 4 generators now use shared annotations (httpgen, clientgen, tsclientgen)
- Ready for plan 02-04: openapiv3 migration (the final generator)
- All generators function correctly (full test suite passes, all golden files unchanged)

---
*Phase: 02-shared-annotations*
*Completed: 2026-02-05*
