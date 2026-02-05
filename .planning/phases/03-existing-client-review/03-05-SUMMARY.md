---
phase: 03-existing-client-review
plan: 05
subsystem: api
tags: [openapi, proto3, json-spec, int64, error-schema]

# Dependency graph
requires:
  - phase: 03-01
    provides: shared test proto infrastructure
  - phase: 03-02
    provides: server Content-Type response headers
provides:
  - OpenAPI generator with correct Error schema matching sebuf.http.Error proto
  - int64/uint64 fields typed as string per proto3 JSON spec
  - Consistent type mapping between OpenAPI and protojson behavior
affects: [phase-07-json-mapping, phase-11-test-harness]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Proto3 JSON spec compliance for 64-bit integers (type: string, format: int64/uint64)
    - Component schema references for standard error types (Error, ValidationError)

key-files:
  created: []
  modified:
    - internal/openapiv3/generator.go
    - internal/openapiv3/types.go
    - internal/openapiv3/testdata/golden/yaml/*.openapi.yaml
    - internal/openapiv3/testdata/golden/json/*.openapi.json

key-decisions:
  - "D-03-05-01: Error schema uses single 'message' field matching sebuf.http.Error proto (not error+code)"
  - "D-03-05-02: int64/uint64 mapped to type:string per proto3 JSON spec for JavaScript precision safety"
  - "D-03-05-03: Added headerTypeUint64 constant and removed minimum constraint since uint64 is now string type"

patterns-established:
  - "Proto3 JSON type mapping: 64-bit integers serialize as strings, 32-bit as integers"
  - "Error component schemas referenced by $ref instead of inline definitions"

# Metrics
duration: 7min
completed: 2026-02-05
---

# Phase 3 Plan 5: OpenAPI Protojson Consistency Summary

**OpenAPI generator corrected to match proto3 JSON spec: Error schema with single message field, int64/uint64 as string type with format**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-05T21:16:12Z
- **Completed:** 2026-02-05T21:23:XX Z
- **Tasks:** 2
- **Files modified:** 46 (generator, types, all golden files)

## Accomplishments

- Fixed Error response schema from incorrect `{error: string, code: integer}` to correct `{message: string}` matching sebuf.http.Error proto
- Changed int64/sint64/sfixed64 from `type: integer` to `type: string` with `format: int64` per proto3 JSON spec
- Changed uint64/fixed64 from `type: integer` to `type: string` with `format: uint64` per proto3 JSON spec
- Added Error component schema alongside existing ValidationError and FieldViolation schemas
- All OpenAPI golden files updated to reflect correct type mappings

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix OpenAPI error response schema** - `80d1833` (fix)
   - Corrected Error schema to match sebuf.http.Error proto definition
   - Changed from inline error+code schema to component reference
   - Added Error to addBuiltinErrorSchemas function

2. **Task 2: Audit and fix type mapping for protojson consistency** - `51f0e3a` (fix)
   - Changed int64/uint64 to string type per proto3 JSON spec
   - Updated both convertScalarField (body schemas) and createFieldSchema (parameters)
   - Added headerTypeUint64 constant for lint compliance

## Files Created/Modified

- `internal/openapiv3/generator.go` - Fixed buildResponses to use Error component ref, renamed addValidationErrorSchemas to addBuiltinErrorSchemas, added Error schema definition, fixed int64/uint64 in createFieldSchema
- `internal/openapiv3/types.go` - Fixed int64/uint64 type mapping in convertScalarField, added headerTypeUint64 constant
- `internal/openapiv3/testdata/golden/yaml/*.yaml` - All 20 YAML golden files updated with correct schemas
- `internal/openapiv3/testdata/golden/json/*.json` - All 20 JSON golden files updated with correct schemas

## Decisions Made

1. **D-03-05-01:** Error schema uses single 'message' field matching sebuf.http.Error proto definition exactly, not the incorrect error+code schema that was previously generated
2. **D-03-05-02:** int64/uint64 mapped to `type: string` per proto3 JSON specification to avoid JavaScript precision loss with 53-bit integers
3. **D-03-05-03:** Removed `minimum: 0` constraint from uint64 since the type is now string (constraints don't apply to strings the same way)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - straightforward fixes with clear proto specification guidance.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- OpenAPI generator now produces schemas consistent with:
  - The actual sebuf.http.Error proto definition
  - The proto3 JSON specification for 64-bit integers
  - What the Go server actually returns and what clients expect
- Ready for Phase 7 JSON mapping work which will build on consistent type foundations
- Cross-generator consistency improved: OpenAPI int64/uint64 now matches TS client string types

---
*Phase: 03-existing-client-review*
*Completed: 2026-02-05*
