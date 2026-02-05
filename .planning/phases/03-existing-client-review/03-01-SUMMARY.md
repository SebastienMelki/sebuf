---
phase: 03-existing-client-review
plan: 01
subsystem: testing
tags: [protobuf, golden-files, symlinks, test-infrastructure, proto3-optional]

# Dependency graph
requires:
  - phase: 02-shared-annotations
    provides: shared annotations package used by all 4 generators
provides:
  - canonical exhaustive test proto covering all annotation combinations
  - shared test proto infrastructure via symlinks across all 4 generators
  - proto3 optional field support in go-http and go-client plugins
  - OpenAPI golden test coverage for shared protos
affects: [03-existing-client-review, 04-json-mapping]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Canonical test proto in httpgen, symlinked to other generators"
    - "UnwrapService added to unwrap.proto for cross-generator root-level unwrap testing"

key-files:
  created:
    - internal/openapiv3/testdata/proto/http_verbs_comprehensive.proto (symlink)
    - internal/openapiv3/testdata/proto/query_params.proto (symlink)
    - internal/openapiv3/testdata/proto/backward_compat.proto (symlink)
    - internal/openapiv3/testdata/golden/yaml/RESTfulAPIService.openapi.yaml
    - internal/openapiv3/testdata/golden/yaml/BackwardCompatService.openapi.yaml
    - internal/openapiv3/testdata/golden/yaml/QueryParamService.openapi.yaml
    - internal/openapiv3/testdata/golden/yaml/NoAnnotationsService.openapi.yaml
    - internal/openapiv3/testdata/golden/yaml/BasePathOnlyService.openapi.yaml
    - internal/openapiv3/testdata/golden/yaml/OptionDataService.openapi.yaml
  modified:
    - internal/httpgen/testdata/proto/http_verbs_comprehensive.proto
    - internal/httpgen/testdata/proto/unwrap.proto
    - internal/openapiv3/exhaustive_golden_test.go
    - internal/openapiv3/testdata/proto/unwrap.proto (replaced with symlink)
    - cmd/protoc-gen-go-http/main.go
    - cmd/protoc-gen-go-client/main.go

key-decisions:
  - "D-03-01-01: Added UnwrapService to httpgen unwrap.proto (alongside existing OptionDataService) to enable OpenAPI root-level unwrap testing via shared symlink"
  - "D-03-01-02: Root-level unwrap RPCs use POST method (not GET) to satisfy httpgen GET-with-body validation"
  - "D-03-01-03: Proto3 optional support added to go-http and go-client plugins via SupportedFeatures declaration"

patterns-established:
  - "Shared test protos: httpgen is canonical source, all other generators symlink via ../../../httpgen/testdata/proto/"
  - "OpenAPI golden test: one test case per service per format (yaml+json)"

# Metrics
duration: 10min
completed: 2026-02-05
---

# Phase 3 Plan 1: Shared Test Proto Infrastructure Summary

**Canonical exhaustive test proto with all annotation types, shared across all 4 generators via symlinks, with proto3 optional support fix**

## Performance

- **Duration:** 10 min
- **Started:** 2026-02-05T18:22:55Z
- **Completed:** 2026-02-05T18:33:25Z
- **Tasks:** 2
- **Files modified:** 21

## Accomplishments
- Expanded exhaustive test proto with int64/uint64/float/double query params, enums, optional fields, nested messages, and search RPC
- Created symlinks from OpenAPI test directory to httpgen canonical test protos (http_verbs_comprehensive, query_params, backward_compat, unwrap)
- Fixed proto3 optional field support in protoc-gen-go-http and protoc-gen-go-client plugins
- Added 12 new OpenAPI golden test cases for shared proto services
- All 4 generators pass with the expanded proto and shared infrastructure

## Task Commits

Each task was committed atomically:

1. **Task 1: Expand exhaustive test proto and fix proto3 optional support** - `1cd141a` (feat/fix)
2. **Task 2: Symlink shared test protos into OpenAPI and update golden infrastructure** - `a0e69d5` (feat)

## Files Created/Modified

### Created
- `internal/openapiv3/testdata/proto/http_verbs_comprehensive.proto` - Symlink to httpgen canonical
- `internal/openapiv3/testdata/proto/query_params.proto` - Symlink to httpgen canonical
- `internal/openapiv3/testdata/proto/backward_compat.proto` - Symlink to httpgen canonical
- `internal/openapiv3/testdata/golden/yaml/RESTfulAPIService.openapi.yaml` - Golden file for shared proto
- `internal/openapiv3/testdata/golden/yaml/BackwardCompatService.openapi.yaml` - Golden file for shared proto
- `internal/openapiv3/testdata/golden/yaml/QueryParamService.openapi.yaml` - Golden file for shared proto
- `internal/openapiv3/testdata/golden/yaml/NoAnnotationsService.openapi.yaml` - Golden file for shared proto
- `internal/openapiv3/testdata/golden/yaml/BasePathOnlyService.openapi.yaml` - Golden file for shared proto
- `internal/openapiv3/testdata/golden/yaml/OptionDataService.openapi.yaml` - Golden file for shared proto
- Plus corresponding JSON golden files for each

### Modified
- `internal/httpgen/testdata/proto/http_verbs_comprehensive.proto` - Added enum, nested message, optional fields, int64/uint64/float/double query params, SearchResources RPC
- `internal/httpgen/testdata/proto/unwrap.proto` - Added UnwrapService with root-level unwrap RPCs
- `internal/openapiv3/testdata/proto/unwrap.proto` - Replaced with symlink to httpgen
- `internal/openapiv3/exhaustive_golden_test.go` - Added 12 test cases for shared protos + OptionDataService
- `cmd/protoc-gen-go-http/main.go` - Added proto3 optional SupportedFeatures
- `cmd/protoc-gen-go-client/main.go` - Added proto3 optional SupportedFeatures
- All 4 generators' golden files updated for expanded proto

## Decisions Made

- **D-03-01-01:** Added UnwrapService to httpgen unwrap.proto as a second service (alongside OptionDataService) rather than renaming. This preserves backward compatibility for existing httpgen golden files while enabling OpenAPI root-level unwrap testing.
- **D-03-01-02:** Root-level unwrap RPCs use POST method instead of GET. The httpgen validator correctly enforces that GET requests cannot have body fields. The original OpenAPI unwrap.proto used GET, but that was semantically incorrect.
- **D-03-01-03:** Proto3 optional fields require plugins to declare `FEATURE_PROTO3_OPTIONAL` in SupportedFeatures. Only tsclientgen and openapiv3 had this; go-http and go-client were missing it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Proto3 optional support missing in go-http and go-client plugins**
- **Found during:** Task 1 (expanded proto with `optional string tag`)
- **Issue:** protoc-gen-go-http and protoc-gen-go-client did not declare FEATURE_PROTO3_OPTIONAL, causing protoc to reject proto files with optional fields
- **Fix:** Added `plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)` to both plugins
- **Files modified:** cmd/protoc-gen-go-http/main.go, cmd/protoc-gen-go-client/main.go
- **Verification:** All 4 generators now handle optional fields in proto3
- **Committed in:** 1cd141a (Task 1 commit)

**2. [Rule 2 - Missing Critical] UnwrapService missing from httpgen unwrap.proto**
- **Found during:** Task 2 (symlinking unwrap.proto to OpenAPI)
- **Issue:** OpenAPI unwrap.proto had UnwrapService with root-level unwrap RPCs; httpgen version only had OptionDataService with one RPC. Symlinking would lose root-level unwrap test coverage.
- **Fix:** Added UnwrapService to httpgen unwrap.proto with all 4 root-level unwrap RPCs, using POST method to satisfy httpgen validation
- **Files modified:** internal/httpgen/testdata/proto/unwrap.proto
- **Verification:** All 4 generators handle the expanded unwrap.proto without errors
- **Committed in:** a0e69d5 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both auto-fixes necessary for correctness. No scope creep.

## Issues Encountered

- Task 1 commit `1cd141a` was already present in the repository from a prior session that included the proto expansion and proto3 optional fix. The changes were re-verified to confirm all done criteria were met.
- OpenAPI unwrap.proto had different package name and service structure from httpgen version, requiring careful merging rather than simple replacement.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 4 generators now share canonical test protos via symlinks
- Expanded exhaustive test proto provides comprehensive coverage for cross-generator consistency auditing in Plan 03-03+
- OpenAPI golden test infrastructure expanded to 38 test cases (19 services x 2 formats)
- No blockers for subsequent plans

---
*Phase: 03-existing-client-review*
*Completed: 2026-02-05*
