---
phase: 04-json-primitive-encoding
plan: 04
subsystem: api
tags: [enum, encoding, json, enum_value, enum_encoding, custom-values, number-encoding]

# Dependency graph
requires:
  - phase: 04-01
    provides: enum_encoding and enum_value annotations in shared annotations package
provides:
  - Custom enum value JSON marshaling in Go generators (MarshalJSON/UnmarshalJSON)
  - Enum lookup maps for custom value mapping (toJSON/fromJSON)
  - NUMBER encoding for enums producing integer JSON values
  - TypeScript union types with custom enum values
  - OpenAPI schemas with custom enum values and integer types
affects: [phase-07-json-mapping-advanced, language-clients]

# Tech tracking
tech-stack:
  added: []
  patterns: [enum-lookup-maps, bidirectional-enum-mapping]

key-files:
  created:
    - internal/httpgen/enum_encoding.go
    - internal/clientgen/enum_encoding.go
    - internal/httpgen/testdata/proto/enum_encoding.proto
    - internal/httpgen/testdata/golden/enum_encoding_enum_encoding.pb.go
    - internal/tsclientgen/testdata/golden/enum_encoding_client.ts
    - internal/openapiv3/testdata/golden/yaml/EnumEncodingService.openapi.yaml
  modified:
    - internal/httpgen/generator.go
    - internal/clientgen/generator.go
    - internal/tsclientgen/types.go
    - internal/openapiv3/types.go

key-decisions:
  - "D-04-04-01: Separate enum_encoding.go files in httpgen/clientgen to avoid import conflicts with int64 encoding.go"
  - "D-04-04-02: Both proto name and custom value accepted in UnmarshalJSON for backward compatibility"
  - "D-04-04-03: NUMBER encoding returns 'number' type in TypeScript, 'integer' type in OpenAPI"

patterns-established:
  - "Enum lookup maps: toJSON map[EnumType]string and fromJSON map[string]EnumType for bidirectional mapping"
  - "Custom MarshalJSON/UnmarshalJSON generation for enums with custom enum_value annotations"

# Metrics
duration: ~25min
completed: 2026-02-06
---

# Phase 04 Plan 04: Enum Encoding Summary

**Custom enum value JSON mapping with enum_value annotation and NUMBER encoding support across all 4 generators**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-02-06
- **Completed:** 2026-02-06
- **Tasks:** 3
- **Files modified:** 21

## Accomplishments

- Implemented enum encoding in Go generators (httpgen, clientgen) with custom MarshalJSON/UnmarshalJSON
- Added enum lookup maps (toJSON/fromJSON) for bidirectional custom value mapping
- Updated ts-client to generate custom union types and number type for NUMBER encoding
- Updated OpenAPI to generate custom enum values and integer type for NUMBER encoding
- Added comprehensive golden file tests for enum encoding across all 4 generators

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement enum encoding in Go generators** - `9874a8e` (feat)
2. **Task 2: Implement enum encoding in ts-client and OpenAPI** - `ed5bb67` (feat)
3. **Task 3: Add golden file tests for enum encoding** - `e470e25` (test)

## Files Created/Modified

**Created:**
- `internal/httpgen/enum_encoding.go` - Enum MarshalJSON/UnmarshalJSON generation for httpgen
- `internal/clientgen/enum_encoding.go` - Identical enum encoding for clientgen
- `internal/httpgen/testdata/proto/enum_encoding.proto` - Test proto with Status (custom values) and Priority enums
- `internal/httpgen/testdata/golden/enum_encoding_enum_encoding.pb.go` - Generated enum encoding code
- `internal/tsclientgen/testdata/golden/enum_encoding_client.ts` - TypeScript with custom union types
- `internal/openapiv3/testdata/golden/yaml/EnumEncodingService.openapi.yaml` - OpenAPI with custom enum schemas

**Modified:**
- `internal/httpgen/generator.go` - Calls validateEnumAnnotationsInFile and generateEnumEncodingFile
- `internal/clientgen/generator.go` - Same changes for client generator
- `internal/tsclientgen/types.go` - Updated generateEnumType, tsFieldType, tsElementType for custom values and NUMBER encoding
- `internal/openapiv3/types.go` - Updated convertEnumField for custom values and integer enum schema

## Decisions Made

- **D-04-04-01:** Created separate enum_encoding.go files to avoid import conflicts with existing encoding.go (int64)
- **D-04-04-02:** UnmarshalJSON accepts both proto name AND custom value for backward compatibility
- **D-04-04-03:** NUMBER encoding produces `number` in TypeScript and `integer` in OpenAPI

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Enum encoding complete across all 4 generators
- Ready for Plan 05 (cross-generator consistency tests)
- All existing tests pass

---
*Phase: 04-json-primitive-encoding*
*Completed: 2026-02-06*
