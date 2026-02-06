---
phase: 07-json-structural-transforms
plan: 01
subsystem: api
tags: [protobuf, annotations, oneof, discriminator, flatten, validation]

# Dependency graph
requires:
  - phase: 06-json-data-encoding
    provides: Extension number sequence up to 50016, annotation parsing patterns in internal/annotations
provides:
  - OneofConfig proto message with discriminator and flatten fields
  - Proto extensions oneof_config (50017), oneof_value (50018), flatten (50019), flatten_prefix (50020)
  - Shared annotation parsing functions (GetOneofConfig, GetOneofVariantValue, GetOneofDiscriminatorInfo)
  - Shared flatten functions (IsFlattenField, GetFlattenPrefix, HasFlattenFields)
  - Validation functions (ValidateOneofDiscriminator, ValidateFlattenField, ValidateFlattenCollisions)
affects: [07-02 (go-http oneof/flatten), 07-03 (cross-generator), 07-04 (consistency tests)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "OneofOptions extensions for oneof-level annotations (distinct from FieldOptions)"
    - "Validation split into helper functions to satisfy cognitive complexity limits"

key-files:
  created:
    - internal/annotations/oneof_discriminator.go
    - internal/annotations/flatten.go
  modified:
    - proto/sebuf/http/annotations.proto
    - http/annotations.pb.go
    - internal/annotations/annotations_test.go

key-decisions:
  - "D-07-01-01: Extension numbers 50017-50020 continue sequence from 50016 (bytes_encoding)"
  - "D-07-01-02: OneofConfig uses OneofOptions (not FieldOptions) -- first use of this extension type in project"
  - "D-07-01-03: ValidateOneofDiscriminator split into 3 helper functions to stay under cognitive complexity limit"

patterns-established:
  - "OneofOptions pattern: cast to *descriptorpb.OneofOptions + proto.HasExtension/GetExtension"
  - "Validation decomposition: top-level function delegates to helpers for complex validation"

# Metrics
duration: 7min
completed: 2026-02-06
---

# Phase 7 Plan 1: Annotations Definition Summary

**Proto annotations for oneof discriminated unions (oneof_config, oneof_value) and nested message flattening (flatten, flatten_prefix) with shared validation in internal/annotations**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-06T17:19:08Z
- **Completed:** 2026-02-06T17:26:43Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Defined 4 new proto extensions (50017-50020) for oneof and flatten annotations
- Created shared annotation parsing functions usable by all 4 generators
- Created comprehensive validation functions detecting collisions at generation time
- All existing tests continue to pass (zero regression)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add OneofConfig, oneof_value, flatten, and flatten_prefix to annotations.proto** - `12a83b5` (feat)
2. **Task 2: Create oneof_discriminator.go and flatten.go with tests** - `1e36f94` (feat)

**Plan metadata:** (pending)

## Files Created/Modified
- `proto/sebuf/http/annotations.proto` - Added OneofConfig message, oneof_config/oneof_value/flatten/flatten_prefix extensions
- `http/annotations.pb.go` - Regenerated Go code with new extension descriptors
- `internal/annotations/oneof_discriminator.go` - Oneof config parsing, variant resolution, discriminator validation
- `internal/annotations/flatten.go` - Flatten detection, prefix parsing, collision validation
- `internal/annotations/annotations_test.go` - Tests for extension descriptors, struct types, number sequence

## Decisions Made
- D-07-01-01: Extension numbers 50017-50020 continue the existing sequence from 50016 (bytes_encoding)
- D-07-01-02: OneofConfig uses google.protobuf.OneofOptions (first use of this extension target type in the project, distinct from the FieldOptions used by all prior annotations)
- D-07-01-03: ValidateOneofDiscriminator was split into 3 helper functions (validateDiscriminatorNameCollision, validateOneofFlatten, buildReservedNames) to stay under the gocognit complexity limit of 20

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Lint reported cognitive complexity 28 on ValidateOneofDiscriminator -- resolved by extracting 3 helper functions (deviation Rule 1 not applicable since this was a lint issue, not a bug; resolved within normal task flow)
- Lint flagged unnecessary string() conversions on JSONName() which already returns string -- removed the conversions
- Lint auto-fixed test function signatures (removed explicit protogen type annotations on blank identifiers)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All annotation infrastructure ready for generator implementation (Plan 07-02)
- E_OneofConfig, E_OneofValue, E_Flatten, E_FlattenPrefix all accessible from any generator
- Validation functions ready to call during code generation
- No blockers

---
*Phase: 07-json-structural-transforms*
*Completed: 2026-02-06*
