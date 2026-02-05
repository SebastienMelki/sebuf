---
phase: 04-json-primitive-encoding
plan: 01
subsystem: api
tags: [protobuf, annotations, int64, enum, json-encoding]

# Dependency graph
requires:
  - phase: 02-shared-annotations
    provides: Shared annotation parsing patterns and package structure
provides:
  - Int64Encoding proto annotation for controlling int64/uint64 JSON serialization
  - EnumEncoding proto annotation for controlling enum JSON serialization
  - EnumValue proto annotation for custom enum value JSON names
  - GetInt64Encoding, IsInt64NumberEncoding functions in internal/annotations
  - GetEnumEncoding, GetEnumValueMapping, HasAnyEnumValueMapping, HasConflictingEnumAnnotations functions in internal/annotations
affects: [04-02, 04-03, 04-04, 04-05, httpgen, clientgen, tsclientgen, openapiv3]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Enum-based encoding control (UNSPECIFIED/STRING/NUMBER pattern)
    - EnumValueOptions extension for per-value customization

key-files:
  created:
    - internal/annotations/int64_encoding.go
    - internal/annotations/enum_encoding.go
  modified:
    - proto/sebuf/http/annotations.proto
    - http/annotations.pb.go
    - internal/annotations/annotations_test.go

key-decisions:
  - "Extension numbers 50010-50012 for new annotations (continuing from 50009)"
  - "UNSPECIFIED value means follow protojson defaults"
  - "STRING is explicit (same as protojson default but documented)"
  - "NUMBER encoding available with precision warning for int64 > 2^53"

patterns-established:
  - "Encoding control via optional enum field extensions"
  - "GetXxxEncoding pattern returns enum type (not bool) for flexibility"
  - "HasConflictingAnnotations validation pattern for incompatible annotation combinations"

# Metrics
duration: 4min
completed: 2026-02-06
---

# Phase 4 Plan 1: Annotation Infrastructure Summary

**Proto annotations and shared parsing functions for int64/enum JSON encoding control across all 4 generators**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-05T22:20:49Z
- **Completed:** 2026-02-05T22:24:23Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added Int64Encoding and EnumEncoding proto enums with UNSPECIFIED/STRING/NUMBER values
- Added int64_encoding (50010), enum_encoding (50011), and enum_value (50012) extensions
- Created GetInt64Encoding and GetEnumEncoding annotation parsing functions following established patterns
- Created GetEnumValueMapping for custom enum value JSON names
- Added HasConflictingEnumAnnotations validation for incompatible annotation combinations
- All existing tests pass (zero regression)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add encoding annotations to proto** - `cb8f275` (feat)
2. **Task 2: Create annotation parsing functions** - `f0dae37` (feat)

## Files Created/Modified
- `proto/sebuf/http/annotations.proto` - Added Int64Encoding, EnumEncoding enums and field extensions
- `http/annotations.pb.go` - Regenerated with new extension descriptors
- `internal/annotations/int64_encoding.go` - GetInt64Encoding, IsInt64NumberEncoding functions
- `internal/annotations/enum_encoding.go` - GetEnumEncoding, GetEnumValueMapping, HasAnyEnumValueMapping, HasConflictingEnumAnnotations functions
- `internal/annotations/annotations_test.go` - Unit tests for encoding types and extension descriptors

## Decisions Made
- D-04-01-01: Extension numbers 50010-50012 continue sequence from existing 50009 (unwrap)
- D-04-01-02: UNSPECIFIED (0) always means "use protojson default" - explicit STRING value available for documentation
- D-04-01-03: GetEnumValueMapping returns empty string (not nil) for consistency with Go string semantics

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

**buf generate conflict with errors_impl.go:**
- The project's buf.gen.yaml runs protoc-gen-go-http on proto files, which generates errors_error_impl.pb.go
- This conflicts with hand-written errors_impl.go (duplicate method declarations)
- Resolution: Removed generated conflict file - this is a pre-existing project configuration issue
- Impact: None on this plan's deliverables

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Annotation infrastructure complete, ready for generator integration
- Plan 04-02 can integrate int64_encoding into go-http handler generator
- Plans 04-03 through 04-05 can follow similar integration patterns
- All generators can use shared annotation functions from internal/annotations

---
*Phase: 04-json-primitive-encoding*
*Completed: 2026-02-06*
