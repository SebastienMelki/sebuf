---
phase: 05-json-nullable-empty
plan: 03
subsystem: api
tags: [empty-behavior, json, protobuf, marshaljson, openapi, oneOf]

# Dependency graph
requires:
  - phase: 05-01
    provides: "empty_behavior annotation definitions and annotation functions"
provides:
  - "Empty behavior MarshalJSON/UnmarshalJSON generation in go-http and go-client"
  - "OpenAPI oneOf schema for empty_behavior=NULL message fields"
  - "Golden file test coverage for all 3 generators"
affects: ["05-04", "05-05"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "proto.Size() == 0 for empty message detection"
    - "oneOf with null type for nullable message fields in OpenAPI 3.1"
    - "Identical empty_behavior.go in httpgen and clientgen for server/client consistency"

key-files:
  created:
    - "internal/httpgen/empty_behavior.go"
    - "internal/clientgen/empty_behavior.go"
    - "internal/httpgen/testdata/proto/empty_behavior.proto"
    - "internal/httpgen/testdata/golden/empty_behavior_empty_behavior.pb.go"
    - "internal/openapiv3/testdata/golden/yaml/EmptyBehaviorService.openapi.yaml"
    - "internal/openapiv3/testdata/golden/json/EmptyBehaviorService.openapi.json"
    - "internal/tsclientgen/testdata/golden/empty_behavior_client.ts"
  modified:
    - "internal/httpgen/generator.go"
    - "internal/clientgen/generator.go"
    - "internal/openapiv3/types.go"
    - "internal/httpgen/golden_test.go"
    - "internal/tsclientgen/golden_test.go"
    - "internal/openapiv3/exhaustive_golden_test.go"

key-decisions:
  - "D-05-03-01: Identical empty_behavior.go in httpgen and clientgen for server/client JSON consistency"
  - "D-05-03-02: OpenAPI oneOf schema for NULL fields ({$ref} | {type: null}) instead of nullable:true (deprecated in 3.1)"
  - "D-05-03-03: OMIT fields use standard $ref in OpenAPI (serialization-only behavior, schema unchanged)"
  - "D-05-03-04: Exhaustive switch for EmptyBehavior enum to satisfy linter"

patterns-established:
  - "Empty message detection via proto.Size() == 0"
  - "oneOf with null type for message fields that can be null in OpenAPI 3.1"

# Metrics
duration: 7min
completed: 2026-02-06
---

# Phase 5 Plan 3: Empty Behavior Implementation Summary

**Empty behavior encoding across go-http, go-client, and openapiv3 generators with PRESERVE/{}/NULL/null/OMIT/delete semantics and proto.Size() == 0 empty detection**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-06T09:58:43Z
- **Completed:** 2026-02-06T10:06:22Z
- **Tasks:** 3/3
- **Files modified:** 13

## Accomplishments
- Implemented empty_behavior MarshalJSON/UnmarshalJSON in both go-http and go-client generators
- Added oneOf schema for empty_behavior=NULL message fields in OpenAPI generator
- Created comprehensive test proto with all 3 behaviors and golden file tests across all generators

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement empty_behavior in Go generators** - `c77bdc3` (feat)
2. **Task 2: Implement empty_behavior in OpenAPI generator** - `1157bab` (feat)
3. **Task 3: Add empty_behavior test proto and golden file tests** - `f45b750` (test)

## Files Created/Modified

- `internal/httpgen/empty_behavior.go` - Generates MarshalJSON/UnmarshalJSON for empty_behavior fields (NULL, OMIT, PRESERVE)
- `internal/httpgen/generator.go` - Added generateEmptyBehaviorEncodingFile call
- `internal/clientgen/empty_behavior.go` - Identical to httpgen for server/client consistency
- `internal/clientgen/generator.go` - Added generateEmptyBehaviorEncodingFile call
- `internal/openapiv3/types.go` - Added makeNullableOneOfSchema for empty_behavior=NULL fields
- `internal/httpgen/testdata/proto/empty_behavior.proto` - Test proto with PRESERVE, NULL, OMIT modes
- `internal/httpgen/testdata/golden/empty_behavior_empty_behavior.pb.go` - Golden file showing generated encoding
- `internal/openapiv3/testdata/golden/yaml/EmptyBehaviorService.openapi.yaml` - OpenAPI golden with oneOf
- `internal/openapiv3/testdata/golden/json/EmptyBehaviorService.openapi.json` - JSON variant
- `internal/tsclientgen/testdata/golden/empty_behavior_client.ts` - TS client golden

## Decisions Made

- **D-05-03-01:** Identical empty_behavior.go in httpgen and clientgen - guarantees server/client JSON match
- **D-05-03-02:** OpenAPI 3.1 oneOf schema for NULL fields - proper semantic representation vs deprecated nullable:true
- **D-05-03-03:** OMIT fields use standard $ref in OpenAPI - OMIT only affects serialization, not type definition
- **D-05-03-04:** Added explicit EMPTY_BEHAVIOR_UNSPECIFIED case in switch to satisfy exhaustive linter

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed exhaustive switch lint for EmptyBehavior enum**
- **Found during:** Task 1 (lint check)
- **Issue:** Switch on EmptyBehavior missing EMPTY_BEHAVIOR_UNSPECIFIED case
- **Fix:** Added explicit case for EMPTY_BEHAVIOR_UNSPECIFIED in both httpgen and clientgen
- **Files modified:** internal/httpgen/empty_behavior.go, internal/clientgen/empty_behavior.go
- **Verification:** make lint-fix shows 0 issues
- **Committed in:** c77bdc3 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Lint compliance fix, no scope creep.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Empty behavior encoding complete across all generators
- Ready for plan 05-04 (cross-generator consistency tests) and 05-05 (integration tests)
- All existing tests continue to pass (zero regression)

---
*Phase: 05-json-nullable-empty*
*Completed: 2026-02-06*
