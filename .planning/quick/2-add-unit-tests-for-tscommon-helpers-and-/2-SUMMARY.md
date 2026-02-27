---
phase: quick
plan: 2
subsystem: testing
tags: [tscommon, unit-tests, enum_value, golden-files, table-driven-tests]

# Dependency graph
requires:
  - phase: quick-1
    provides: "Fixed TS generator bugs for enum/repeated query params"
provides:
  - "Unit tests for tscommon pure functions (TSScalarType, TSZeroCheck, SnakeToLowerCamel, HeaderNameToPropertyName)"
  - "Golden-output validation for TSEnumUnspecifiedValue"
  - "Custom enum_value coverage on Region enum in query param context"
affects: [ts-client, ts-server, openapiv3]

# Tech tracking
tech-stack:
  added: []
  patterns: [golden-output-based validation for functions requiring protoc extensions]

key-files:
  created:
    - internal/tscommon/types_test.go
    - internal/tscommon/helpers_test.go
  modified:
    - internal/httpgen/testdata/proto/query_params.proto
    - internal/tsclientgen/testdata/golden/query_params_client.ts
    - internal/tsservergen/testdata/golden/query_params_server.ts
    - internal/openapiv3/testdata/golden/json/QueryParamService.openapi.json
    - internal/openapiv3/testdata/golden/yaml/QueryParamService.openapi.yaml

key-decisions:
  - "Golden-output-based validation for TSEnumUnspecifiedValue since protogen.Field with extensions cannot be easily mocked"
  - "readGoldenFile helper to avoid variable shadowing lint errors in subtests"

patterns-established:
  - "readGoldenFile test helper: avoids variable shadowing when reading golden files in subtests"

requirements-completed: []

# Metrics
duration: 5min
completed: 2026-02-27
---

# Quick Task 2: Add Unit Tests for tscommon Helpers and Enum Value Coverage

**Table-driven unit tests for tscommon pure functions with custom enum_value coverage on Region enum in query param context**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-27T14:52:34Z
- **Completed:** 2026-02-27T14:57:21Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Created comprehensive table-driven tests for all tscommon pure functions (TSScalarType, TSZeroCheck, SnakeToLowerCamel, SnakeToUpperCamel, HeaderNameToPropertyName)
- Validated TSEnumUnspecifiedValue behavior through golden file output for custom, default, and query param enum contexts
- Added custom enum_value annotations to Region enum in query_params.proto, exercising GetEnumValueMapping code path in query parameter context
- Updated golden files across tsclientgen, tsservergen, and openapiv3

## Task Commits

Each task was committed atomically:

1. **Task 1: Unit tests for tscommon pure functions** - `2189de5` (test)
2. **Task 2: Add custom enum_value to Region enum** - `20957fd` (feat)

## Files Created/Modified
- `internal/tscommon/helpers_test.go` - Table-driven tests for SnakeToLowerCamel, SnakeToUpperCamel, HeaderNameToPropertyName (14 subtests)
- `internal/tscommon/types_test.go` - Table-driven tests for TSScalarType (18 subtests), TSZeroCheck (16 subtests), and golden-based TSEnumUnspecifiedValue validation (3 subtests)
- `internal/httpgen/testdata/proto/query_params.proto` - Region enum now uses custom enum_value annotations
- `internal/tsclientgen/testdata/golden/query_params_client.ts` - Updated Region type and zero check to use custom values
- `internal/tsservergen/testdata/golden/query_params_server.ts` - Updated Region type and default value to use custom values
- `internal/openapiv3/testdata/golden/json/QueryParamService.openapi.json` - Updated enum values in schema
- `internal/openapiv3/testdata/golden/yaml/QueryParamService.openapi.yaml` - Updated enum values in schema

## Decisions Made
- Used golden-output-based validation for TSEnumUnspecifiedValue instead of direct function calls, since protogen.Field with populated extension options cannot be easily mocked without running protoc
- Added readGoldenFile helper function to avoid variable shadowing lint errors (govet shadow) when reading files inside subtests

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed lint issues in types_test.go**
- **Found during:** Task 2 (after adding Region golden validation)
- **Issue:** govet shadow warnings for `err` variable in subtests, golines formatting
- **Fix:** Extracted readGoldenFile helper to avoid shadowing; reformatted long lines
- **Files modified:** internal/tscommon/types_test.go
- **Verification:** golangci-lint run ./internal/tscommon/... reports 0 issues
- **Committed in:** 20957fd (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Lint fix necessary for CI compliance. No scope creep.

## Issues Encountered
- `make lint-fix` and `make build` fail due to Xcode license agreement issue on this machine. Used direct `go build` and `golangci-lint` commands as workaround.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- tscommon package now has test coverage for all pure functions
- GetEnumValueMapping code path exercised in both enum_encoding and query_params contexts
- Ready for Phase 8+ language client work

## Self-Check: PASSED

- [x] internal/tscommon/types_test.go exists (152 lines >= 80 min)
- [x] internal/tscommon/helpers_test.go exists (68 lines >= 40 min)
- [x] Commit 2189de5 exists (Task 1)
- [x] Commit 20957fd exists (Task 2)
- [x] All 8 test packages pass
- [x] 0 lint issues on tscommon

---
*Quick Task: 2*
*Completed: 2026-02-27*
