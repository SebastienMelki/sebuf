---
phase: 13-gateway-features
plan: 02
subsystem: gateway
tags: [krakend, jwt, auth-validator, claim-propagation, input-headers]

# Dependency graph
requires:
  - phase: 13-gateway-features/01
    provides: ExtraConfig struct fields, buildEndpointExtraConfig extension point, NamespaceAuthValidator constant, JWTConfig proto message
provides:
  - JWT auth/validator generation from service-level proto annotations
  - Claim-to-header propagation with array-of-arrays KrakenD format
  - Automatic input_headers augmentation for propagated claim headers
  - buildAuthValidatorConfig and buildPropagateClaims builder functions
affects: [13-03, gateway-features]

# Tech tracking
tech-stack:
  added: []
  patterns: [service-level-only JWT config, claim propagation auto-augments input_headers]

key-files:
  created:
    - internal/krakendgen/testdata/proto/jwt_auth_service.proto
    - internal/krakendgen/testdata/golden/JWTAuthService.krakend.json
  modified:
    - internal/krakendgen/generator.go
    - internal/krakendgen/golden_test.go

key-decisions:
  - "JWT is service-level only -- same auth config on every endpoint, no method-level override"
  - "Propagated claim headers auto-added to input_headers with dedup and sort for KrakenD zero-trust model"
  - "propagate_claims serialized as array-of-arrays per KrakenD spec, not array of objects"
  - "cache field only included when true (false is default, omitted)"

patterns-established:
  - "JWT auth generation follows same resolve/build pattern but is service-only (no epConfig check)"
  - "input_headers augmentation: derive from annotations, then append propagated JWT claim headers"

requirements-completed: [AUTH-01, AUTH-02, AUTH-03]

# Metrics
duration: 3min
completed: 2026-02-25
---

# Phase 13 Plan 02: JWT Authentication Summary

**JWT auth/validator generation with claim-to-header propagation and automatic input_headers augmentation from service-level proto annotations**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-25T14:22:08Z
- **Completed:** 2026-02-25T14:25:35Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- JWT auth/validator config generated for every endpoint from service-level gateway_config annotation
- Claim propagation serializes as KrakenD array-of-arrays format: `[["sub", "X-User"], ...]`
- Propagated claim header names auto-added to input_headers (dedup + sort), satisfying KrakenD zero-trust model
- Golden test locks down full JWT output including merged headers from service, method, and JWT claims

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement JWT auth/validator generation with claim propagation** - `c81b9e5` (feat)
2. **Task 2: Create JWT auth test proto and golden file** - `839a600` (test)

## Files Created/Modified
- `internal/krakendgen/generator.go` - Added buildAuthValidatorConfig, buildPropagateClaims, getJWTPropagatedHeaderNames, containsString; updated buildEndpointExtraConfig and input_headers derivation
- `internal/krakendgen/testdata/proto/jwt_auth_service.proto` - Test proto with JWT config, claim propagation, service headers, and method headers
- `internal/krakendgen/testdata/golden/JWTAuthService.krakend.json` - Golden file verifying auth/validator config on all endpoints
- `internal/krakendgen/golden_test.go` - Added jwt_auth_service test case (11 total: 8 golden + 3 validation)

## Decisions Made
- JWT is service-level only -- same auth config applied to every endpoint, no method-level override needed
- Propagated claim headers auto-added to input_headers with dedup and sort, ensuring KrakenD zero-trust compliance
- propagate_claims uses array-of-arrays format per KrakenD specification (not array of objects)
- cache field only included when true (false is default, omitted via conditional)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `make lint-fix` fails due to pre-existing golangci-lint Go version incompatibility (go1.26 vs go1.24). Not caused by our changes. `go vet ./...` passes cleanly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- JWT auth generation complete, ready for circuit breaker and cache in Plan 03
- buildEndpointExtraConfig now handles both rate limiting and JWT namespaces
- Same pattern (resolve/build + namespace constant) applies to circuit breaker and cache additions

## Self-Check: PASSED

All 4 key files verified present. Both task commits (c81b9e5, 839a600) verified in git log.

---
*Phase: 13-gateway-features*
*Completed: 2026-02-25*
