---
phase: 13-gateway-features
plan: 03
subsystem: gateway
tags: [krakend, circuit-breaker, caching, concurrent-calls, resilience, golden-tests]

# Dependency graph
requires:
  - phase: 13-gateway-features/02
    provides: JWT auth and rate limiting generation infrastructure in krakendgen
provides:
  - Circuit breaker generation in backend extra_config (qos/circuit-breaker)
  - Backend HTTP cache generation in backend extra_config (qos/http-cache)
  - Concurrent calls as top-level endpoint field
  - Generation-time validation for circuit breaker required fields and cache max_items/max_size pairing
  - Full-features-combined golden test proving all gateway features compose correctly
affects: [phase-14]

# Tech tracking
tech-stack:
  added: []
  patterns: [resolve/validate/build pattern extended to circuit breaker and cache]

key-files:
  created:
    - internal/krakendgen/testdata/proto/circuit_breaker_service.proto
    - internal/krakendgen/testdata/proto/cache_concurrent_service.proto
    - internal/krakendgen/testdata/proto/full_gateway_service.proto
    - internal/krakendgen/testdata/proto/invalid_circuit_breaker.proto
    - internal/krakendgen/testdata/proto/invalid_cache.proto
    - internal/krakendgen/testdata/golden/CircuitBreakerService.krakend.json
    - internal/krakendgen/testdata/golden/CacheConcurrentService.krakend.json
    - internal/krakendgen/testdata/golden/FullGatewayService.krakend.json
  modified:
    - internal/krakendgen/generator.go
    - internal/krakendgen/golden_test.go

key-decisions:
  - "Circuit breaker and cache configs validated before endpoint loop for fail-fast on invalid service-level config"
  - "Method-level circuit breaker and cache validated per-endpoint inside the loop"
  - "Circuit breaker int32 fields stored as int32 in map -- Go json.Marshal handles correctly (consistent with rate limit pattern)"

patterns-established:
  - "All backend-scoped features (rate limit proxy, circuit breaker, cache) in buildBackendExtraConfig"
  - "All endpoint-scoped features (rate limit router, JWT) in buildEndpointExtraConfig"
  - "concurrent_calls is always top-level endpoint field, never in extra_config"

requirements-completed: [RESL-01, RESL-02, RESL-03, RESL-04, TEST-02]

# Metrics
duration: 4min
completed: 2026-02-25
---

# Phase 13 Plan 03: Circuit Breaker, Caching, and Concurrent Calls Summary

**Circuit breaker, HTTP cache, and concurrent calls generation with validation and full-features-combined golden test proving all KrakenD gateway features compose correctly**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-25T14:30:12Z
- **Completed:** 2026-02-25T14:34:09Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Circuit breaker generation with method-level override of service-level config, including validation requiring all three fields (interval, timeout, max_errors) be positive
- Backend HTTP cache generation with shared flag, max_items/max_size pairing validation, and method-level override
- Concurrent calls as a top-level endpoint field (NOT in extra_config), with method-level override
- Full-features-combined golden test (FullGatewayService) proving all five namespace entries compose correctly: qos/ratelimit/router, auth/validator, qos/ratelimit/proxy, qos/circuit-breaker, qos/http-cache

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement circuit breaker, caching, and concurrent calls generation with validation** - `58bdccd` (feat)
2. **Task 2: Create test protos, golden files, and validation error tests** - `52f34c5` (test)

## Files Created/Modified
- `internal/krakendgen/generator.go` - Added resolveCircuitBreaker, validateCircuitBreaker, buildCircuitBreakerConfig, resolveCache, validateCache, buildHTTPCacheConfig, resolveConcurrentCalls; updated buildBackendExtraConfig and GenerateService
- `internal/krakendgen/golden_test.go` - Added 3 golden test cases and 2 validation error test cases
- `internal/krakendgen/testdata/proto/circuit_breaker_service.proto` - Service/method-level circuit breaker test
- `internal/krakendgen/testdata/proto/cache_concurrent_service.proto` - Cache and concurrent calls test
- `internal/krakendgen/testdata/proto/full_gateway_service.proto` - All gateway features combined test
- `internal/krakendgen/testdata/proto/invalid_circuit_breaker.proto` - Validation error test (missing required fields)
- `internal/krakendgen/testdata/proto/invalid_cache.proto` - Validation error test (mismatched max_items/max_size)
- `internal/krakendgen/testdata/golden/CircuitBreakerService.krakend.json` - Circuit breaker golden file
- `internal/krakendgen/testdata/golden/CacheConcurrentService.krakend.json` - Cache + concurrent calls golden file
- `internal/krakendgen/testdata/golden/FullGatewayService.krakend.json` - Full features golden file (2929 bytes)

## Decisions Made
- Circuit breaker and cache configs validated before endpoint loop for fail-fast on invalid service-level config, with additional per-endpoint validation for method-level overrides
- Circuit breaker int32 fields stored as int32 in config map, consistent with rate limit pattern
- Cache validation enforces max_items and max_size as a pair -- both set or both unset

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `make lint-fix` panics due to golangci-lint v2.8 built with Go 1.24 encountering a go1.26 dependency -- pre-existing environment issue, not caused by our changes. `go vet ./...` passes cleanly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 13 (Gateway Features) is now complete with all resilience features implemented
- All 11 golden tests pass, 5 validation error tests pass, full project test suite green
- Ready for Phase 14 or any follow-up milestone work

## Self-Check: PASSED

All 9 created files verified. All 2 task commits verified.

---
*Phase: 13-gateway-features*
*Completed: 2026-02-25*
