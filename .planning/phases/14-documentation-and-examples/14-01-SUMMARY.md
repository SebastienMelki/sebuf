---
phase: 14-documentation-and-examples
plan: 01
subsystem: gateway
tags: [krakend, protobuf, enums, validation, schema]

# Dependency graph
requires:
  - phase: 13-gateway-features
    provides: KrakenD generator with rate limiting, JWT, circuit breaker, caching
provides:
  - RateLimitStrategy and JWTAlgorithm proto enums for type-safe gateway config
  - Cache validation rejecting shared+max_items/max_size (KrakenD oneOf)
  - krakend check -lc schema validation test for all golden files
affects: [14-documentation-and-examples]

# Tech tracking
tech-stack:
  added: []
  patterns: [enum-to-string mapping for KrakenD JSON output, krakend CLI schema validation in tests]

key-files:
  created: []
  modified:
    - proto/sebuf/krakend/krakend.proto
    - krakend/krakend.pb.go
    - internal/krakendgen/generator.go
    - internal/krakendgen/golden_test.go
    - internal/krakendgen/testdata/proto/rate_limit_service.proto
    - internal/krakendgen/testdata/proto/jwt_auth_service.proto
    - internal/krakendgen/testdata/proto/full_gateway_service.proto
    - internal/krakendgen/testdata/proto/cache_concurrent_service.proto
    - internal/krakendgen/testdata/golden/CacheConcurrentService.krakend.json

key-decisions:
  - "Enum-to-string mapping via explicit switch statements for clarity and compile-time safety"
  - "Cache shared+max_items/max_size validation added before existing max_items/max_size pairing check"
  - "krakend check test skips gracefully when CLI not installed for CI compatibility"

patterns-established:
  - "Proto enum to KrakenD string: dedicated xxxToString functions with explicit switch cases"
  - "Schema validation test: iterate golden files, run krakend check -lc, skip if CLI absent"

requirements-completed: [DOCS-01]

# Metrics
duration: 5min
completed: 2026-02-25
---

# Phase 14 Plan 01: Proto Enums and Schema Validation Summary

**Proto enums for rate limit strategy and JWT algorithm, cache oneOf fix, and krakend check -lc validation test**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-25T15:04:01Z
- **Completed:** 2026-02-25T15:09:38Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Replaced raw string fields with RateLimitStrategy and JWTAlgorithm proto enums for compile-time type safety
- Fixed cache oneOf constraint violation: shared=true can no longer be combined with max_items/max_size
- Added TestKrakenDSchemaValidation running krakend check -lc on all 12 golden files
- All 33 test cases pass (11 golden, 5 validation errors, 5 namespaces, 12 schema validation)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add proto enums, fix cache oneOf, update generator and test protos** - `60c014e` (feat)
2. **Task 2: Add krakend check -lc schema validation test** - `7840fde` (test)

## Files Created/Modified
- `proto/sebuf/krakend/krakend.proto` - Added RateLimitStrategy and JWTAlgorithm enums, changed field types
- `krakend/krakend.pb.go` - Regenerated Go code with enum types
- `internal/krakendgen/generator.go` - Added enum-to-string mapping functions, cache shared+max validation
- `internal/krakendgen/golden_test.go` - Added TestKrakenDSchemaValidation test function
- `internal/krakendgen/testdata/proto/rate_limit_service.proto` - Changed strategy strings to enum values
- `internal/krakendgen/testdata/proto/jwt_auth_service.proto` - Changed alg string to enum value
- `internal/krakendgen/testdata/proto/full_gateway_service.proto` - Changed strategy and alg to enum values
- `internal/krakendgen/testdata/proto/cache_concurrent_service.proto` - Removed shared+max_items conflict
- `internal/krakendgen/testdata/golden/CacheConcurrentService.krakend.json` - Updated: GetProduct no longer has shared+max_items

## Decisions Made
- Enum-to-string mapping uses explicit switch statements rather than string trimming or maps for clarity and compile-time safety
- Cache shared+max_items/max_size validation added before existing max_items/max_size pairing check for fail-fast behavior
- Schema validation test skips gracefully when krakend CLI is not installed, maintaining CI compatibility

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Proto enums provide IDE autocomplete and compile-time validation for KrakenD config
- All golden files validated against KrakenD's official JSON schema
- Ready for remaining phase 14 plans (documentation and examples)

## Self-Check: PASSED

All 10 files verified present. Both task commits (60c014e, 7840fde) verified in git log.

---
*Phase: 14-documentation-and-examples*
*Completed: 2026-02-25*
