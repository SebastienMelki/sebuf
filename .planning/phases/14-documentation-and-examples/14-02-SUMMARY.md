---
phase: 14-documentation-and-examples
plan: 02
subsystem: gateway
tags: [krakend, flexible-config, example, protobuf, gateway]

# Dependency graph
requires:
  - phase: 14-documentation-and-examples
    provides: Proto enums (RateLimitStrategy, JWTAlgorithm) and schema validation
provides:
  - Multi-service KrakenD gateway example with 2 services and 8 endpoints
  - Flexible Config integration guide showing generate -> partials -> compose workflow
  - Feature distribution across UserService (JWT, rate limit, headers) and ProductService (CB, cache, concurrent)
affects: [14-documentation-and-examples]

# Tech tracking
tech-stack:
  added: []
  patterns: [protoc-based example generation (not buf), jq partial extraction for FC, FC_ENABLE compose workflow]

key-files:
  created:
    - examples/krakend-gateway/proto/services/user_service.proto
    - examples/krakend-gateway/proto/services/product_service.proto
    - examples/krakend-gateway/proto/models/common.proto
    - examples/krakend-gateway/gateway/krakend.tmpl
    - examples/krakend-gateway/gateway/settings/service_files.json
    - examples/krakend-gateway/Makefile
    - examples/krakend-gateway/README.md
    - examples/krakend-gateway/.gitignore
    - examples/krakend-gateway/buf.yaml
    - examples/krakend-gateway/buf.gen.yaml
  modified: []

key-decisions:
  - "Use protoc directly instead of buf generate because krakend proto not yet on BSR"
  - "Partials extracted via jq + sed to strip outer array brackets for FC include"
  - "krakend check -d output is human-readable debug, not JSON dump -- compose validates only"

patterns-established:
  - "Example generation: protoc with --proto_path for local proto resolution"
  - "FC workflow: generate per-service -> jq extract partials -> FC_ENABLE validate template"

requirements-completed: [DOCS-01, DOCS-02]

# Metrics
duration: 8min
completed: 2026-02-25
---

# Phase 14 Plan 02: KrakenD Gateway Example with Flexible Config Integration

**Multi-service KrakenD example (8 endpoints across 2 services) with Flexible Config composition and comprehensive annotation reference**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-25T15:15:22Z
- **Completed:** 2026-02-25T15:23:30Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- UserService proto demonstrates JWT (RS256), IP rate limiting, backend rate limiting, service/method headers, and query parameter forwarding
- ProductService proto demonstrates circuit breaker, shared/sized caching, concurrent calls, and header-based rate limiting
- Method-level overrides shown: UpdateUser (header strategy), GetProduct (sized cache + concurrent calls), CreateProduct (aggressive CB)
- Flexible Config template composes per-service endpoint fragments; validated with krakend check
- README serves as both example documentation and Flexible Config integration guide with step-by-step workflow

## Task Commits

Each task was committed atomically:

1. **Task 1: Create example proto files, buf config, and Makefile** - `3af2112` (feat)
2. **Task 2: Create Flexible Config template and example README** - `57a36dd` (feat)

## Files Created/Modified
- `examples/krakend-gateway/proto/services/user_service.proto` - JWT, rate limiting, headers, query params
- `examples/krakend-gateway/proto/services/product_service.proto` - Circuit breaker, caching, concurrent calls
- `examples/krakend-gateway/proto/models/common.proto` - Shared Pagination message
- `examples/krakend-gateway/gateway/krakend.tmpl` - Flexible Config template composing service partials
- `examples/krakend-gateway/gateway/settings/service_files.json` - Service registry for template
- `examples/krakend-gateway/Makefile` - Workflow: generate, partials, validate, compose, clean
- `examples/krakend-gateway/README.md` - Annotation reference + Flexible Config integration guide
- `examples/krakend-gateway/.gitignore` - Excludes generated/ and gateway/partials/
- `examples/krakend-gateway/buf.yaml` - Buf config (linting only)
- `examples/krakend-gateway/buf.gen.yaml` - Reference config (generation uses protoc)

## Decisions Made
- Used protoc directly instead of buf generate because the krakend proto (sebuf/krakend/krakend.proto) is not yet published to the BSR. Proto paths resolve locally via --proto_path.
- Partials are extracted using jq to get the endpoints array, then sed strips the outer [ and ] brackets. This produces bare comma-separated endpoint objects suitable for FC {{ include }}.
- krakend check -d produces human-readable debug output, not JSON. The compose target validates only (krakend check -l -c) rather than dumping a combined JSON file.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Switched from buf generate to protoc for code generation**
- **Found during:** Task 1 (Generate step)
- **Issue:** buf dep update failed because krakend proto not on BSR: `import "sebuf/krakend/krakend.proto": file does not exist`
- **Fix:** Changed Makefile to use protoc directly with --proto_path pointing to local proto/ and ../../proto/, matching the pattern used by golden tests
- **Files modified:** examples/krakend-gateway/Makefile, buf.yaml, buf.gen.yaml
- **Verification:** make generate produces valid per-service configs
- **Committed in:** 3af2112 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed krakend CLI flag format**
- **Found during:** Task 2 (Compose step)
- **Issue:** `-dpc` is not valid; krakend CLI does not support combined shorthand flags
- **Fix:** Changed to separate flags: `-d -c` for debug, `-l -c` for lint+config
- **Files modified:** examples/krakend-gateway/Makefile
- **Verification:** make compose validates successfully (Syntax OK!)
- **Committed in:** 57a36dd (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both fixes were necessary for the example to work. No scope creep.

## Issues Encountered
None beyond the deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Complete KrakenD example ready for users to copy and adapt
- README doubles as the Flexible Config integration guide (DOCS-02 deliverable)
- Ready for phase 14 plan 03 (remaining documentation tasks)

## Self-Check: PASSED

All 11 files verified present. Both task commits (3af2112, 57a36dd) verified in git log.

---
*Phase: 14-documentation-and-examples*
*Completed: 2026-02-25*
