---
phase: quick-1
plan: 01
subsystem: codegen
tags: [typescript, protoc-plugin, query-params, enum, repeated-fields]

# Dependency graph
requires:
  - phase: 02-shared-annotations
    provides: shared annotations package for query params
provides:
  - "Fixed enum query param zero-check in TS client (compare against UNSPECIFIED variant)"
  - "Fixed enum query param default/cast in TS server (use UNSPECIFIED + enum type cast)"
  - "Fixed repeated string query param in TS server (getAll instead of get)"
  - "Fixed repeated string query param in TS client (forEach+append instead of set)"
  - "Fixed duplicate const url in TS server for mixed path+query routes"
  - "Fixed unused req parameter in TS client for empty request messages"
  - "TSEnumUnspecifiedValue helper in tscommon for enum zero-value detection"
affects: [ts-client, ts-server, tscommon]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TSEnumUnspecifiedValue for enum zero-check in query params"
    - "IsList() early-return pattern for repeated field handling"

key-files:
  created: []
  modified:
    - "internal/tscommon/types.go"
    - "internal/tsclientgen/generator.go"
    - "internal/tsservergen/generator.go"
    - "internal/httpgen/testdata/proto/query_params.proto"

key-decisions:
  - "Enum zero-check uses first enum value name (UNSPECIFIED variant) from proto definition"
  - "Repeated fields in client use forEach+append for multi-value URL params"
  - "Repeated fields in server use getAll() for proper array extraction"
  - "Server reuses existing url variable when path params already declared it"

patterns-established:
  - "TSEnumUnspecifiedValue: extract UNSPECIFIED enum value with custom enum_value annotation support"
  - "Field.Desc.IsList() early-return before type switch for repeated field handling"

requirements-completed: [BUG-1, BUG-2, BUG-3, BUG-4, BUG-5, BUG-6]

# Metrics
duration: 6min
completed: 2026-02-27
---

# Quick Task 1: Fix 6 TS Generator Bugs Summary

**Fixed enum/repeated query param generation, duplicate URL const, and unused req parameter across protoc-gen-ts-client and protoc-gen-ts-server**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-27T14:37:41Z
- **Completed:** 2026-02-27T14:43:41Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Fixed all 6 TypeScript generation bugs affecting enum query params, repeated fields, duplicate URL declarations, and unused parameters
- Added TSEnumUnspecifiedValue helper and updated TSZeroCheckForField for enum/repeated field awareness
- Added test proto cases (Region enum, SearchAdvancedRequest, EmptyRequest) exercising all bug scenarios
- Updated golden files across all 5 generators (httpgen, clientgen, tsclientgen, tsservergen, openapiv3)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add enum/repeated query param test cases and helper functions** - `cadce9b` (feat)
2. **Task 2: Fix all 6 bugs in ts-client and ts-server generators** - `a7627b3` (fix)

**Plan metadata:** (pending final commit)

## Files Created/Modified
- `internal/tscommon/types.go` - Added TSEnumUnspecifiedValue, updated TSZeroCheckForField for enum+repeated, added "enum" case to TSZeroCheck
- `internal/tsclientgen/generator.go` - Bug #4 (repeated forEach+append), Bug #6 (unused _req prefix)
- `internal/tsservergen/generator.go` - Bug #2 (enum cast+default), Bug #3 (repeated getAll), Bug #5 (no duplicate const url)
- `internal/httpgen/testdata/proto/query_params.proto` - Added Region enum, SearchAdvancedRequest, EmptyRequest, two new RPCs
- `internal/tsclientgen/testdata/golden/query_params_client.ts` - New methods for SearchAdvanced + GetDefaults
- `internal/tsservergen/testdata/golden/query_params_server.ts` - New handlers for SearchAdvanced + GetDefaults
- `internal/tsclientgen/testdata/golden/http_verbs_comprehensive_client.ts` - Existing enum field now uses UNSPECIFIED check
- `internal/tsservergen/testdata/golden/http_verbs_comprehensive_server.ts` - Existing enum field now uses UNSPECIFIED cast
- `internal/clientgen/testdata/golden/query_params_client.pb.go` - Updated for new RPCs
- `internal/httpgen/testdata/golden/query_params_http.pb.go` - Updated for new RPCs
- `internal/openapiv3/testdata/golden/yaml/QueryParamService.openapi.yaml` - Updated for new RPCs
- `internal/openapiv3/testdata/golden/json/QueryParamService.openapi.json` - Updated for new RPCs

## Decisions Made
- Enum zero-check uses the first enum value name (the UNSPECIFIED variant) from the proto definition, with custom enum_value annotation support
- Repeated fields in client use `forEach` + `params.append` for multi-value URL params (not `params.set` with join)
- Repeated fields in server use `params.getAll()` which returns `string[]` matching the TS interface type
- Server reuses existing `url` variable when path params already declared it (checks `len(cfg.pathParams) > 0`)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated golden files for all 5 generators**
- **Found during:** Task 2 (golden file update)
- **Issue:** Adding new RPCs to query_params.proto affected all generators that symlink to it (httpgen, clientgen, openapiv3), not just tsclientgen and tsservergen
- **Fix:** Ran UPDATE_GOLDEN=1 for all affected test packages
- **Files modified:** clientgen, httpgen, openapiv3 golden files
- **Verification:** `go test ./...` all pass
- **Committed in:** a7627b3 (Task 2 commit)

**2. [Rule 3 - Blocking] Applied lint auto-fix for fmt.Fprintf**
- **Found during:** Task 2 (lint run)
- **Issue:** golangci-lint auto-fixed `sb.WriteString(fmt.Sprintf(...))` to `fmt.Fprintf(&sb, ...)` in tscommon/types.go
- **Fix:** Accepted lint auto-fix (correct and more efficient)
- **Files modified:** internal/tscommon/types.go
- **Verification:** Tests pass, lint clean (only pre-existing gosec issues in httpgen remain)
- **Committed in:** a7627b3 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both auto-fixes necessary for test correctness. No scope creep.

## Issues Encountered
- Xcode license not agreed to on this machine, preventing `make build` and `make lint-fix` from working. Worked around by calling `go build` and `golangci-lint` directly.
- Pre-existing gosec warnings in httpgen/generator.go (integer overflow conversion) unrelated to this task.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All TS generators now produce correct TypeScript for enum query params, repeated fields, mixed path+query routes, and empty requests
- Ready for Phase 8+ language work

---
*Quick Task: 1-fix-6-ts-generator-bugs-enum-query-param*
*Completed: 2026-02-27*
