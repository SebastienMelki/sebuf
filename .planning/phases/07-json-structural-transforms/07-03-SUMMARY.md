---
phase: 07-json-structural-transforms
plan: 03
subsystem: api
tags: [flatten, protobuf, json, codegen, marshal, openapi, typescript]

# Dependency graph
requires:
  - phase: 07-01
    provides: "Flatten annotation parsing (IsFlattenField, GetFlattenPrefix, ValidateFlattenField, ValidateFlattenCollisions, HasFlattenFields)"
provides:
  - "Flatten MarshalJSON/UnmarshalJSON generation in go-http and go-client"
  - "Flattened TypeScript interface generation with prefix support"
  - "OpenAPI allOf schema representation for flattened structures"
  - "Golden test coverage for flatten across all 4 generators"
affects: [07-04, phase-08, phase-09, phase-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Flatten MarshalJSON: protojson base -> delete nested key -> json.Marshal child (composable) -> merge with prefix"
    - "Flatten UnmarshalJSON: extract prefixed fields -> json.Unmarshal into child -> protojson for remainder"
    - "OpenAPI allOf for flattened structures: base properties + per-flatten-field property groups"
    - "TypeScript field inlining: skip flatten fields, emit child fields with prefix at parent level"

key-files:
  created:
    - internal/httpgen/flatten.go
    - internal/clientgen/flatten.go
    - internal/httpgen/testdata/proto/flatten.proto
    - internal/httpgen/testdata/golden/flatten_flatten.pb.go
    - internal/httpgen/testdata/golden/flatten_http.pb.go
    - internal/httpgen/testdata/golden/flatten_http_binding.pb.go
    - internal/httpgen/testdata/golden/flatten_http_config.pb.go
    - internal/clientgen/testdata/golden/flatten_client.pb.go
    - internal/clientgen/testdata/golden/flatten_flatten.pb.go
    - internal/tsclientgen/testdata/golden/flatten_client.ts
    - internal/openapiv3/testdata/golden/yaml/FlattenService.openapi.yaml
    - internal/openapiv3/testdata/golden/json/FlattenService.openapi.json
  modified:
    - internal/httpgen/generator.go
    - internal/clientgen/generator.go
    - internal/tsclientgen/types.go
    - internal/openapiv3/generator.go
    - internal/httpgen/golden_test.go
    - internal/clientgen/golden_test.go
    - internal/tsclientgen/golden_test.go
    - internal/openapiv3/exhaustive_golden_test.go

key-decisions:
  - "D-07-03-01: Use json.Marshal for child messages in flatten MarshalJSON to enable annotation composability (child's own MarshalJSON invoked)"
  - "D-07-03-02: OpenAPI allOf pattern chosen over flat properties for semantic clarity (base + per-flatten-field groups)"
  - "D-07-03-03: MarshalJSON conflict detection rejects messages with both flatten AND another encoding feature"
  - "D-07-03-04: nolint:dupl added for intentional similarity between flatten and oneof_discriminator MarshalJSON patterns"

patterns-established:
  - "Flatten MarshalJSON/UnmarshalJSON: protojson-then-manipulate-raw-JSON with json.Marshal for child composability"
  - "OpenAPI allOf for structural transforms: each structural feature uses allOf to compose schema components"
  - "TypeScript flatten inlining: check IsFlattenField -> iterate child fields with prefix -> emit at parent level"

# Metrics
duration: ~18min
completed: 2026-02-06
---

# Phase 7 Plan 3: Nested Message Flattening Summary

**MarshalJSON/UnmarshalJSON flatten generation across go-http/go-client, TypeScript interface inlining, and OpenAPI allOf schema representation with full golden test coverage**

## Performance

- **Duration:** ~18 min
- **Started:** 2026-02-06T17:30:00Z
- **Completed:** 2026-02-06T17:47:52Z
- **Tasks:** 2
- **Files modified:** 21

## Accomplishments
- Go generators (httpgen + clientgen) produce custom MarshalJSON/UnmarshalJSON for messages with flatten-annotated fields, promoting child fields to parent level with optional prefix
- TypeScript generator inlines flattened child fields at parent level in interfaces (billing_street, billing_city, etc.)
- OpenAPI generator uses allOf pattern to represent flattened structures with descriptive per-section schema
- Golden tests created and verified for all 4 generators covering SimpleFlatten, DualFlatten (dual prefix), MixedFlatten (mixed flatten/non-flatten), and PlainNested (backward compat)
- Generation-time validation for field collisions, invalid field types, and MarshalJSON conflicts with other encoding features
- Annotation composability: json.Marshal for child messages ensures child's own MarshalJSON (int64, timestamps, etc.) is invoked

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement flatten in Go generators (go-http and go-client)** - `7c997df` (feat)
2. **Task 2: Implement flatten in ts-client and openapiv3 with golden tests** - `361fa81` (feat)

## Files Created/Modified
- `internal/httpgen/flatten.go` - FlattenContext, validation, MarshalJSON/UnmarshalJSON generation for go-http
- `internal/clientgen/flatten.go` - Identical flatten encoding for go-client (server/client JSON consistency)
- `internal/httpgen/generator.go` - Added generateFlattenFile call in file generation pipeline
- `internal/clientgen/generator.go` - Added generateFlattenFile call in file generation pipeline
- `internal/tsclientgen/types.go` - generateFlattenedFields helper inlines child fields with prefix in TS interfaces
- `internal/openapiv3/generator.go` - buildFlattenedObjectSchema using allOf pattern for flatten representation
- `internal/httpgen/testdata/proto/flatten.proto` - Test proto with SimpleFlatten, DualFlatten, MixedFlatten, PlainNested
- `internal/*/testdata/proto/flatten.proto` - Symlinks from clientgen/tsclientgen/openapiv3 to httpgen source
- `internal/httpgen/testdata/golden/flatten_*.pb.go` - Golden files for go-http flatten
- `internal/clientgen/testdata/golden/flatten_*.pb.go` - Golden files for go-client flatten
- `internal/tsclientgen/testdata/golden/flatten_client.ts` - Golden file for TypeScript flatten
- `internal/openapiv3/testdata/golden/*/FlattenService.openapi.*` - Golden files for OpenAPI flatten (YAML + JSON)
- `internal/*/golden_test.go` - Added flatten test cases to all 4 golden test runners

## Decisions Made
- D-07-03-01: Use json.Marshal (not protojson.Marshal) for child messages in flatten MarshalJSON -- enables annotation composability where child's own encoding annotations are respected
- D-07-03-02: OpenAPI allOf pattern chosen over flat properties -- provides semantic grouping (base properties + per-flatten-field groups with descriptions)
- D-07-03-03: MarshalJSON conflict detection rejects messages with both flatten AND another encoding feature (int64, nullable, empty_behavior, timestamps, bytes) on the same message
- D-07-03-04: Added nolint:dupl for intentional similarity between flatten and oneof_discriminator MarshalJSON patterns (both use protojson-then-manipulate-raw-JSON approach)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added nolint:gocognit to generateFile function**
- **Found during:** Task 1
- **Issue:** Adding generateFlattenFile call (plus parallel agent's generateOneofDiscriminatorFile call) pushed cognitive complexity of generateFile to 21, exceeding the 20 threshold
- **Fix:** Added `//nolint:gocognit // Sequential encoding file generation adds unavoidable branching` directive
- **Files modified:** internal/httpgen/generator.go
- **Verification:** make lint-fix passes for gocognit on this function
- **Committed in:** 7c997df (Task 1 commit)

**2. [Rule 3 - Blocking] Added nolint:dupl for expected cross-feature similarity**
- **Found during:** Task 2
- **Issue:** generateFlattenMarshalJSON flagged as duplicate of oneof_discriminator's generateOneofDiscriminatorMarshalJSON (both use protojson-marshal-then-manipulate pattern)
- **Fix:** Added `//nolint:dupl // Intentionally similar to oneof_discriminator MarshalJSON` to flatten.go in both httpgen and clientgen
- **Files modified:** internal/httpgen/flatten.go, internal/clientgen/flatten.go
- **Verification:** make lint-fix no longer reports dupl for flatten files
- **Committed in:** 361fa81 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking -- lint compliance)
**Impact on plan:** Both auto-fixes necessary for lint compliance with parallel agent's oneof code. No scope creep.

## Issues Encountered
- Parallel agent (07-02: oneof discriminator) was working on the same branch simultaneously, modifying shared files (generator.go, types.go, golden tests). Handled by staging only flatten-related files individually and avoiding the other agent's uncommitted changes. Some shared files (openapiv3/generator.go, tsclientgen/types.go, exhaustive_golden_test.go) inevitably contain both agents' changes since they modify the same files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Flatten implementation complete across all 4 generators
- Golden tests verify correct output for all flatten scenarios
- Plan 07-04 (cross-generator consistency tests) can validate flatten consistency
- Ready for Phase 8-10 language client work after Phase 7 completes

---
*Phase: 07-json-structural-transforms*
*Completed: 2026-02-06*
