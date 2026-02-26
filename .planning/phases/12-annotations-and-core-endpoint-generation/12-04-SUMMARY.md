---
phase: 12-annotations-and-core-endpoint-generation
plan: 04
subsystem: api
tags: [protobuf, krakend, protoc-plugin, code-generation, gateway, validation, golden-tests]

# Dependency graph
requires:
  - phase: 12-03
    provides: "GenerateService with header/query forwarding, complete endpoint JSON output"
provides:
  - "Route conflict validation (duplicate endpoints, static vs parameterized) at generation time"
  - "Golden file test suite covering all KrakenD generation scenarios (7 services, 6 proto files)"
  - "Validation error test suite covering 3 error scenarios (duplicate routes, route conflicts, missing host)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: ["Path segment trie for route conflict detection", "Golden file test pattern adapted from openapiv3 exhaustive tests", "Validation error test pattern from tsservergen"]

key-files:
  created:
    - "internal/krakendgen/validation.go"
    - "internal/krakendgen/golden_test.go"
    - "internal/krakendgen/testdata/proto/simple_service.proto"
    - "internal/krakendgen/testdata/proto/timeout_config.proto"
    - "internal/krakendgen/testdata/proto/host_config.proto"
    - "internal/krakendgen/testdata/proto/headers_forwarding.proto"
    - "internal/krakendgen/testdata/proto/query_forwarding.proto"
    - "internal/krakendgen/testdata/proto/combined_forwarding.proto"
    - "internal/krakendgen/testdata/proto/invalid_duplicate_routes.proto"
    - "internal/krakendgen/testdata/proto/invalid_route_conflict.proto"
    - "internal/krakendgen/testdata/proto/invalid_no_host.proto"
    - "internal/krakendgen/testdata/golden/UserService.krakend.json"
    - "internal/krakendgen/testdata/golden/TimeoutService.krakend.json"
    - "internal/krakendgen/testdata/golden/HostService.krakend.json"
    - "internal/krakendgen/testdata/golden/HeaderForwardingService.krakend.json"
    - "internal/krakendgen/testdata/golden/NoHeaderService.krakend.json"
    - "internal/krakendgen/testdata/golden/QueryForwardingService.krakend.json"
    - "internal/krakendgen/testdata/golden/CombinedForwardingService.krakend.json"
  modified:
    - "internal/krakendgen/generator.go"

key-decisions:
  - "Path segment trie for conflict detection -- simple recursive structure, efficient for typical API route counts"
  - "Error messages reference endpoint indices (not RPC names) since Endpoint struct does not carry RPC name metadata"

patterns-established:
  - "ValidateRoutes called at end of GenerateService before returning -- fail-fast on conflicts"
  - "Golden test pattern: UPDATE_GOLDEN=1 creates files, normal run compares byte-for-byte"

requirements-completed: [VALD-01, VALD-02, TEST-01, TEST-03, TEST-04]

# Metrics
duration: 5min
completed: 2026-02-25
---

# Phase 12 Plan 04: Validation and Golden Tests Summary

**Route conflict validation (duplicate endpoints, static/param conflicts) with comprehensive golden file test suite covering routing, timeouts, hosts, headers, queries, and error scenarios**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-25T13:29:37Z
- **Completed:** 2026-02-25T13:35:34Z
- **Tasks:** 2
- **Files created:** 18 (1 validation, 1 test, 9 protos, 7 golden files)

## Accomplishments
- ValidateRoutes detects duplicate (path, method) tuples and static-vs-parameterized segment conflicts using a path segment trie, with actionable error messages
- Golden file tests lock down output for 7 services across 6 proto files: simple CRUD routing, timeout inheritance/override, multi-host with override, header forwarding (service + method + combined), query forwarding, combined headers + queries, and no-header baseline
- Validation error tests confirm 3 failure modes: duplicate routes, static/param route conflicts, and missing gateway_config annotation
- Full project test suite passes (10/10 packages, 0 regressions)

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement route validation and integrate into generator** - `eec25f7` (feat)
2. **Task 2: Create test protos and golden file test suite** - `65056a2` (test)

## Files Created/Modified
- `internal/krakendgen/validation.go` - ValidateRoutes with duplicate detection and path trie conflict detection
- `internal/krakendgen/generator.go` - Integrated ValidateRoutes call before returning from GenerateService
- `internal/krakendgen/golden_test.go` - TestKrakenDGoldenFiles (6 test cases, 7 services) and TestKrakenDValidationErrors (3 error cases)
- `internal/krakendgen/testdata/proto/*.proto` - 9 test proto files covering all generation and error scenarios
- `internal/krakendgen/testdata/golden/*.krakend.json` - 7 golden files for byte-for-byte regression detection

## Decisions Made
- Used path segment trie for conflict detection -- simple recursive structure that scales linearly with route count
- Error messages reference endpoint indices rather than RPC names because the Endpoint struct does not carry RPC name metadata; the index is sufficient for debugging since endpoints appear in proto declaration order
- Fixed golden test to use `continue` instead of `return` when creating golden files for multi-service protos, ensuring all services in a single proto get their golden files created in one pass

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed early return in multi-service golden test**
- **Found during:** Task 2 (golden file creation)
- **Issue:** `tryCreateGoldenFile` returned true causing the test to `return` (exit), which skipped golden file creation for the second service (NoHeaderService) in headers_forwarding.proto
- **Fix:** Changed `return` to `continue` in the service loop so remaining services still get their golden files created
- **Files modified:** internal/krakendgen/golden_test.go
- **Verification:** Re-ran UPDATE_GOLDEN=1 for headers_forwarding, both HeaderForwardingService and NoHeaderService golden files created
- **Committed in:** 65056a2 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor test infrastructure fix, no scope creep.

## Issues Encountered
- golangci-lint panics on generated .pb.go files (pre-existing Go 1.24 vs 1.26 mismatch) -- not caused by plan changes. go vet passes clean. Same issue noted in previous plans.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 12 is complete: all 4 plans executed, all requirements covered
- KrakenD generator produces per-service endpoint JSON fragments with routing, timeouts, host config, header forwarding, query forwarding, and route conflict validation
- Golden test suite provides regression safety for future changes

## Self-Check: PASSED
