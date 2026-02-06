---
phase: 05-json-nullable-empty
plan: 01
subsystem: api
tags: [protobuf, annotations, nullable, empty-behavior, json-serialization]

# Dependency graph
requires:
  - phase: 04-json-primitive-encoding
    provides: "Int64Encoding, EnumEncoding annotations and shared annotation patterns"
provides:
  - EmptyBehavior enum (UNSPECIFIED, PRESERVE, NULL, OMIT)
  - nullable bool field extension for optional primitives
  - empty_behavior field extension for message fields
  - IsNullableField, ValidateNullableAnnotation functions
  - GetEmptyBehavior, HasEmptyBehaviorAnnotation, ValidateEmptyBehaviorAnnotation functions
affects: [05-02, 05-03, 05-04, 05-05]

# Tech tracking
tech-stack:
  added: []
  patterns: [nullable annotation pattern, empty_behavior enum pattern]

key-files:
  created:
    - proto/sebuf/http/annotations.proto (extended)
    - internal/annotations/nullable.go
    - internal/annotations/empty_behavior.go
  modified:
    - http/annotations.pb.go
    - internal/annotations/annotations_test.go

key-decisions:
  - "D-05-01-01: Extension numbers 50013 (nullable) and 50014 (empty_behavior) continue sequence from 50012"
  - "D-05-01-02: UNSPECIFIED (0) means default behavior (same as PRESERVE for empty_behavior)"

patterns-established:
  - "NullableValidationError/EmptyBehaviorValidationError error types follow existing pattern"
  - "Validation functions check annotation validity at generation time"

# Metrics
duration: 3min
completed: 2026-02-06
---

# Phase 5 Plan 1: Nullable and Empty Behavior Annotations Summary

**Proto nullable/empty_behavior annotations with shared IsNullableField and GetEmptyBehavior parsing functions for all 4 generators**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-05T23:36:26Z
- **Completed:** 2026-02-05T23:39:32Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added EmptyBehavior enum with UNSPECIFIED, PRESERVE, NULL, OMIT values to annotations.proto
- Added nullable (bool) and empty_behavior field extensions with numbers 50013 and 50014
- Created nullable.go with IsNullableField and ValidateNullableAnnotation functions
- Created empty_behavior.go with GetEmptyBehavior, HasEmptyBehaviorAnnotation, ValidateEmptyBehaviorAnnotation functions
- Added comprehensive unit tests for extension descriptors, enum values, and error types

## Task Commits

Each task was committed atomically:

1. **Task 1: Add nullable and empty_behavior annotations to proto** - `e0681c7` (feat)
2. **Task 2: Create nullable.go and empty_behavior.go** - `30c0efb` (feat)

## Files Created/Modified
- `proto/sebuf/http/annotations.proto` - Added EmptyBehavior enum and nullable/empty_behavior field extensions
- `http/annotations.pb.go` - Regenerated with E_Nullable and E_EmptyBehavior extension descriptors
- `internal/annotations/nullable.go` - IsNullableField, ValidateNullableAnnotation, NullableValidationError
- `internal/annotations/empty_behavior.go` - GetEmptyBehavior, HasEmptyBehaviorAnnotation, ValidateEmptyBehaviorAnnotation, EmptyBehaviorValidationError
- `internal/annotations/annotations_test.go` - Unit tests for new extensions and error types

## Decisions Made
- D-05-01-01: Extension numbers 50013 (nullable) and 50014 (empty_behavior) continue sequence from existing 50012 (enum_value)
- D-05-01-02: UNSPECIFIED (0) always means "use default behavior" - for empty_behavior this means PRESERVE (serialize as {})

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Pre-existing conflict between generated errors_error_impl.pb.go and manual errors_impl.go - resolved by removing regenerated file (not part of this plan's scope, existing issue)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Proto annotations ready for use in all 4 generators
- Shared parsing functions available in internal/annotations package
- Plan 05-02 (go-http nullable implementation) can proceed

---
*Phase: 05-json-nullable-empty*
*Completed: 2026-02-06*
