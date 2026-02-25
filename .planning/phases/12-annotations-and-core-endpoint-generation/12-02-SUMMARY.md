---
phase: 12-annotations-and-core-endpoint-generation
plan: 02
subsystem: api
tags: [protobuf, krakend, protoc-plugin, code-generation, gateway, json]

# Dependency graph
requires:
  - phase: 12-01
    provides: "Proto annotation package (sebuf.krakend), generated Go code (E_GatewayConfig, E_EndpointConfig), KrakenD Endpoint/Backend struct types"
provides:
  - "GenerateService function that reads sebuf.http and sebuf.krakend annotations to produce KrakenD endpoint structs"
  - "Plugin entry point producing per-service pretty-printed JSON endpoint arrays"
  - "Method-level endpoint_config overrides for host and timeout"
affects: [12-03, 12-04]

# Tech tracking
tech-stack:
  added: []
  patterns: ["annotation reading via proto.GetExtension with descriptorpb type assertion", "two-pass RPC scan (collect HTTP methods first, then require gateway_config)", "nil-to-empty-slice normalization for JSON array output"]

key-files:
  created:
    - "internal/krakendgen/generator.go"
  modified:
    - "cmd/protoc-gen-krakend/main.go"

key-decisions:
  - "Only require gateway_config when service has at least one HTTP-annotated RPC (bare services produce empty array)"
  - "Timeout field omitted from JSON via omitempty when not annotated at any level"
  - "Nil endpoint slice normalized to empty slice in entry point to produce [] not null"

patterns-established:
  - "Two-pass generation: collect HTTP methods first, then validate gateway requirements"
  - "Override semantics: method-level endpoint_config fields override service-level gateway_config when non-empty"

requirements-completed: [ANNO-04, CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06]

# Metrics
duration: 6min
completed: 2026-02-25
---

# Phase 12 Plan 02: Annotations and Core Endpoint Generation Summary

**Core KrakenD endpoint generator reading sebuf.http routing and sebuf.krakend host/timeout annotations with method-level override semantics and per-service JSON output**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-25T13:14:36Z
- **Completed:** 2026-02-25T13:20:38Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Implemented GenerateService function that reads both sebuf.http (routing) and sebuf.krakend (host/timeout) annotations to produce KrakenD endpoint structs
- Wired generator into protoc plugin entry point producing per-service pretty-printed JSON with trailing newline
- Method-level endpoint_config correctly overrides service-level gateway_config for host and timeout
- Services without gateway_config fail with clear error; services without HTTP RPCs produce empty array

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement core generator with annotation reading and endpoint assembly** - `39414ff` (feat)
2. **Task 2: Wire generator into plugin entry point with JSON output** - `67a91d3` (feat)

## Files Created/Modified
- `internal/krakendgen/generator.go` - Core generation logic: GenerateService reads annotations, builds endpoint/backend structs, resolves host/timeout overrides
- `cmd/protoc-gen-krakend/main.go` - Updated entry point: calls generator, marshals to JSON, writes per-service files, propagates errors via plugin.Error

## Decisions Made
- Only require gateway_config when the service has at least one HTTP-annotated RPC -- bare services (no HTTP RPCs) produce an empty array without error, avoiding failures on unrelated services in the same proto file
- Timeout field uses omitempty so it is omitted from JSON when not annotated at any level, letting KrakenD use its built-in default
- Nil endpoint slices are normalized to empty slices in the entry point so JSON output is `[]` not `null`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed gateway_config check order for bare services**
- **Found during:** Task 2 (manual verification)
- **Issue:** Original implementation checked gateway_config before iterating RPCs, causing bare services (no HTTP RPCs) to fail with "backend host is required" even though they have nothing to generate
- **Fix:** Restructured to two-pass approach: first collect HTTP-annotated RPCs, then only require gateway_config if at least one exists
- **Files modified:** internal/krakendgen/generator.go
- **Verification:** Bare service proto produces `[]` with exit code 0
- **Committed in:** 67a91d3 (Task 2 commit)

**2. [Rule 1 - Bug] Fixed nil slice JSON marshaling producing null instead of []**
- **Found during:** Task 2 (manual verification)
- **Issue:** When GenerateService returns nil (no HTTP RPCs), json.MarshalIndent produces `null` instead of `[]`
- **Fix:** Added nil-to-empty-slice normalization in entry point before marshaling
- **Files modified:** cmd/protoc-gen-krakend/main.go
- **Verification:** Bare service proto produces `[]` not `null`
- **Committed in:** 67a91d3 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for correctness. No scope creep.

## Issues Encountered
- golangci-lint panics on generated .pb.go files (pre-existing Go 1.24 vs 1.26 mismatch) -- not caused by plan changes. go vet passes on all changed packages.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Generator produces correct KrakenD endpoint JSON from annotated proto services
- Plan 03 will populate InputHeaders and InputQueryStrings fields (currently nil/omitted)
- Plan 04 will add golden tests for regression detection
- All override semantics are in place and manually verified

## Self-Check: PASSED

All 2 files verified on disk. Both task commits (39414ff, 67a91d3) verified in git log. Binary bin/protoc-gen-krakend verified.
