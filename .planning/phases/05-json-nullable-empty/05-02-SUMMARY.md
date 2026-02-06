---
phase: 05-json-nullable-empty
plan: 02
subsystem: api
tags: [nullable, json, protobuf, codegen, openapi, typescript]

# Dependency graph
requires:
  - phase: 05-01
    provides: "nullable and empty_behavior proto annotations and annotation functions"
  - phase: 02-shared-annotations
    provides: "shared annotations package pattern"
  - phase: 04-json-primitive-encoding
    provides: "int64/enum encoding pattern (MarshalJSON/UnmarshalJSON generation)"
provides:
  - "Nullable MarshalJSON/UnmarshalJSON generation in go-http and go-client"
  - "TypeScript T | null type generation for nullable fields"
  - "OpenAPI 3.1 type array syntax for nullable fields"
  - "Golden file tests for nullable across all 4 generators"
affects: [05-03, 05-04, 05-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Nullable encoding: protojson base + map modification for null emission"
    - "OpenAPI 3.1 nullable: type array [T, null] syntax"
    - "TypeScript nullable: T | null (not optional ?) for always-present-but-nullable fields"

key-files:
  created:
    - "internal/httpgen/nullable.go"
    - "internal/clientgen/nullable.go"
    - "internal/httpgen/testdata/proto/nullable.proto"
    - "internal/httpgen/testdata/golden/nullable_nullable.pb.go"
    - "internal/clientgen/testdata/golden/nullable_nullable.pb.go"
    - "internal/tsclientgen/testdata/golden/nullable_client.ts"
    - "internal/openapiv3/testdata/golden/yaml/NullableService.openapi.yaml"
    - "internal/openapiv3/testdata/golden/json/NullableService.openapi.json"
  modified:
    - "internal/httpgen/generator.go"
    - "internal/clientgen/generator.go"
    - "internal/tsclientgen/types.go"
    - "internal/openapiv3/types.go"
    - "internal/httpgen/golden_test.go"
    - "internal/clientgen/golden_test.go"
    - "internal/tsclientgen/golden_test.go"
    - "internal/openapiv3/exhaustive_golden_test.go"

key-decisions:
  - "D-05-02-01: Identical nullable.go in httpgen and clientgen for server/client JSON consistency"
  - "D-05-02-02: Nullable fields in TypeScript use T | null (not optional ?) - always present with value or null"
  - "D-05-02-03: OpenAPI 3.1 type array syntax [T, null] instead of deprecated nullable: true"
  - "D-05-02-04: Nullable encoding placed before service check in clientgen (like httpgen) for message-only files"

patterns-established:
  - "Nullable generation: validate annotations -> collect context -> generate MarshalJSON/UnmarshalJSON"
  - "Three-state fields in TypeScript: absent (not in interface), optional (?), nullable (T | null)"

# Metrics
duration: 7min
completed: 2026-02-06
---

# Phase 05 Plan 02: Nullable Encoding Summary

**Nullable primitive support across all 4 generators: Go MarshalJSON/UnmarshalJSON with null emission, TypeScript T | null types, OpenAPI 3.1 type array syntax**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-06T09:47:22Z
- **Completed:** 2026-02-06T09:54:26Z
- **Tasks:** 3
- **Files modified:** 19

## Accomplishments
- Go generators (httpgen + clientgen) generate MarshalJSON that emits `"field": null` for unset nullable fields, and UnmarshalJSON that accepts null
- TypeScript generator produces `fieldName: T | null` for nullable fields (not optional `?`)
- OpenAPI generator produces `type: ["T", "null"]` per OpenAPI 3.1 spec
- Golden file tests added for all 4 generators with zero regression on existing tests
- Validation errors for invalid nullable annotations (non-optional fields, message fields) propagated correctly

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement nullable in Go generators** - `74116e0` (feat)
2. **Task 2: Implement nullable in TypeScript and OpenAPI generators** - `b98bb86` (feat)
3. **Task 3: Add nullable test proto and golden file tests** - `21c719c` (test)

## Files Created/Modified
- `internal/httpgen/nullable.go` - MarshalJSON/UnmarshalJSON generation for nullable fields (go-http)
- `internal/clientgen/nullable.go` - Identical implementation for go-client consistency
- `internal/httpgen/generator.go` - Calls generateNullableEncodingFile
- `internal/clientgen/generator.go` - Calls generateNullableEncodingFile before service check
- `internal/tsclientgen/types.go` - Nullable field detection with `T | null` type generation
- `internal/openapiv3/types.go` - makeNullableSchema helper for type array syntax
- `internal/httpgen/testdata/proto/nullable.proto` - Test proto with nullable string, int32, bool fields
- Golden files for httpgen, clientgen, tsclientgen, openapiv3 (7 golden files total)
- Test cases added to all 4 golden test files

## Decisions Made
- D-05-02-01: Identical nullable.go in httpgen and clientgen ensures server/client JSON consistency (same pattern as int64 encoding)
- D-05-02-02: Nullable TypeScript fields use `T | null` (not optional `?`) because nullable fields are always present - they have either a value or null, never absent
- D-05-02-03: OpenAPI 3.1 type array syntax `["T", "null"]` used instead of the deprecated `nullable: true` from OpenAPI 3.0
- D-05-02-04: Nullable encoding generation placed before the service existence check in clientgen, matching httpgen pattern, so message-only proto files can still get nullable encoding

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Pre-existing `empty_behavior.go` file from a previous interrupted session causes lint warnings (11 unused + 1 exhaustive). This is untracked code not part of the nullable plan and was not committed. It will be addressed in a future plan (05-03 or later).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Nullable encoding complete across all generators with full test coverage
- Ready for 05-03 (empty_behavior implementation) or 05-04 (cross-generator consistency tests)
- The pre-existing empty_behavior.go skeleton in httpgen needs to be either completed or removed in its own plan

---
*Phase: 05-json-nullable-empty*
*Completed: 2026-02-06*
