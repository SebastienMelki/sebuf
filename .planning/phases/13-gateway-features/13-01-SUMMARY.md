---
phase: 13-gateway-features
plan: 01
subsystem: gateway
tags: [krakend, rate-limiting, protobuf, extra-config, namespaces]

# Dependency graph
requires:
  - phase: 12-annotations-and-core-endpoint-generation
    provides: KrakenD plugin with endpoint generation, golden test infrastructure
provides:
  - Standalone KrakenD config output format ($schema, version 3, endpoints array)
  - Proto annotation messages for rate limiting, JWT, circuit breaker, cache
  - Namespace constants for all KrakenD extra_config keys
  - ExtraConfig fields on Endpoint and Backend structs
  - Rate limiting generation (endpoint-level router + backend-level proxy)
  - buildEndpointExtraConfig/buildBackendExtraConfig extension points
affects: [13-02, 13-03, gateway-features]

# Tech tracking
tech-stack:
  added: []
  patterns: [resolve-then-build extra_config pipeline, namespace constants for KrakenD keys]

key-files:
  created:
    - internal/krakendgen/namespaces.go
    - internal/krakendgen/namespaces_test.go
    - internal/krakendgen/testdata/proto/rate_limit_service.proto
    - internal/krakendgen/testdata/golden/RateLimitService.krakend.json
  modified:
    - cmd/protoc-gen-krakend/main.go
    - proto/sebuf/krakend/krakend.proto
    - krakend/krakend.pb.go
    - internal/krakendgen/types.go
    - internal/krakendgen/generator.go
    - internal/krakendgen/golden_test.go
    - internal/krakendgen/testdata/golden/UserService.krakend.json
    - internal/krakendgen/testdata/golden/TimeoutService.krakend.json
    - internal/krakendgen/testdata/golden/HostService.krakend.json
    - internal/krakendgen/testdata/golden/HeaderForwardingService.krakend.json
    - internal/krakendgen/testdata/golden/NoHeaderService.krakend.json
    - internal/krakendgen/testdata/golden/QueryForwardingService.krakend.json
    - internal/krakendgen/testdata/golden/CombinedForwardingService.krakend.json

key-decisions:
  - "KrakenDConfig wrapper struct lives in types.go alongside Endpoint/Backend -- keeps all KrakenD output types in one place"
  - "ExtraConfig is map[string]any with omitempty -- nil maps omitted from JSON so existing golden files unaffected"
  - "resolve/build pattern for extra_config: resolve picks service or method level, build creates the config map"
  - "Rate limit int32 fields stored as int32 in map (not float64) -- Go json.Marshal handles correctly"

patterns-established:
  - "resolve/build pipeline: resolveX picks service vs method annotation, buildXConfig creates map for namespace"
  - "Namespace constants: all KrakenD extra_config keys are Go constants in namespaces.go, never inline strings"
  - "buildEndpointExtraConfig/buildBackendExtraConfig: central extension points for adding new extra_config namespaces"

requirements-completed: [RLIM-01, RLIM-02, RLIM-03, VALD-03]

# Metrics
duration: 5min
completed: 2026-02-25
---

# Phase 13 Plan 01: Foundation and Rate Limiting Summary

**Standalone KrakenD config output with rate limiting at endpoint (qos/ratelimit/router) and backend (qos/ratelimit/proxy) levels, plus proto messages for all Phase 13 features**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-25T14:12:43Z
- **Completed:** 2026-02-25T14:17:44Z
- **Tasks:** 2
- **Files modified:** 17

## Accomplishments
- Changed output format from bare JSON array to full standalone KrakenD config object with $schema and version 3
- Added proto annotation messages for rate limiting, JWT validation, circuit breaker, and cache
- Created namespace constants with unit tests for all 5 KrakenD extra_config keys
- Extended Endpoint and Backend structs with ExtraConfig and ConcurrentCalls fields
- Implemented rate limiting generation with service-level defaults and method-level overrides
- All 10 test cases pass (8 golden + 3 namespace + 3 validation error tests)

## Task Commits

Each task was committed atomically:

1. **Task 1: Wrap output in standalone KrakenD config, add all proto messages, create namespace constants, extend structs** - `95f1c4d` (feat)
2. **Task 2: Implement rate limiting generation with golden test coverage** - `2d626fe` (feat)

## Files Created/Modified
- `internal/krakendgen/namespaces.go` - Go constants for all KrakenD extra_config namespace strings
- `internal/krakendgen/namespaces_test.go` - Unit tests for namespace constants (value, completeness, no duplicates)
- `internal/krakendgen/testdata/proto/rate_limit_service.proto` - Test proto with service and method-level rate limits
- `internal/krakendgen/testdata/golden/RateLimitService.krakend.json` - Golden file for rate limiting output
- `cmd/protoc-gen-krakend/main.go` - Uses KrakenDConfig wrapper for JSON output
- `proto/sebuf/krakend/krakend.proto` - New messages: RateLimitConfig, BackendRateLimitConfig, JWTConfig, CircuitBreakerConfig, CacheConfig, ClaimToHeader
- `krakend/krakend.pb.go` - Regenerated Go code with all new proto types
- `internal/krakendgen/types.go` - KrakenDConfig wrapper struct, ExtraConfig/ConcurrentCalls fields on Endpoint and Backend
- `internal/krakendgen/generator.go` - Rate limiting resolve/build functions and extra_config integration
- `internal/krakendgen/golden_test.go` - Added rate_limit_service test case
- 7 existing golden files updated to wrapped format

## Decisions Made
- KrakenDConfig wrapper struct lives in types.go alongside Endpoint/Backend, keeping all output types co-located
- ExtraConfig is map[string]any with omitempty so nil maps are omitted from JSON (existing golden files unaffected)
- resolve/build pattern for extra_config: resolveX picks service or method level, buildXConfig creates the config map
- Rate limit int32 fields stored as int32 in map (not float64) for correct JSON serialization

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `make lint-fix` fails due to pre-existing golangci-lint Go version incompatibility (go1.26 required, go1.24 installed). This is an environment issue unrelated to our changes. `go vet ./...` passes cleanly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Standalone config format and ExtraConfig struct fields ready for JWT, circuit breaker, and cache in Plan 02
- buildEndpointExtraConfig/buildBackendExtraConfig are the extension points for adding new namespaces
- Namespace constants pre-defined for all planned features
- Proto messages pre-defined for all Phase 13 features

## Self-Check: PASSED

All 10 key files verified present. Both task commits (95f1c4d, 2d626fe) verified in git log.

---
*Phase: 13-gateway-features*
*Completed: 2026-02-25*
