---
phase: 03-existing-client-review
plan: 04
subsystem: api
tags: [ts-client, typescript, http, serialization, consistency, golden-files, unwrap]

# Dependency graph
requires:
  - phase: 03-01
    provides: shared test proto infrastructure with symlinks
  - phase: 03-03
    provides: verified Go client as reference implementation
provides:
  - Verified TS client generator consistency with Go server and Go client
  - Golden file coverage for all unwrap variants in tsclientgen
  - Confirmed int64/uint64 correctly mapped to string type
affects: [05-openapi-review, 06-openapi-json-consistency, 08-go-lang-client, json-mapping]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - int64/uint64 mapped to TypeScript string (proto3 JSON spec compliance)
    - Query parameter encoding matches server strconv.Parse expectations
    - Error handling with ValidationError (400) and ApiError (other statuses)

key-files:
  created:
    - internal/tsclientgen/testdata/golden/unwrap_client.ts
  modified:
    - internal/tsclientgen/golden_test.go
    - internal/tsclientgen/testdata/proto/unwrap.proto (symlink)

key-decisions:
  - "D-03-04-01: TS client already consistent with Go server - no fixes needed"
  - "D-03-04-02: No JSDoc generation by design - minimalist generated code"

patterns-established:
  - "int64/uint64 type mapping: proto int64/uint64 -> TS string for JSON precision"
  - "Query zero-value omission: int64 fields check !== '0' (string comparison for TS string type)"
  - "Unwrap return types: root unwrap returns bare type, method returns Promise<Type[]> or Promise<Record<string, Type>>"

# Metrics
duration: 7min
completed: 2026-02-05
---

# Phase 3 Plan 4: TS Client Consistency Audit Summary

**TS client verified consistent with Go server and Go client across type mapping, query encoding, error handling, headers, and all 4 unwrap variants - added dedicated unwrap golden file coverage**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-05T21:15:30Z
- **Completed:** 2026-02-05T21:22:00Z
- **Tasks:** 2
- **Files modified:** 3 (1 symlink, 1 golden file, 1 test file)

## Accomplishments

- Added unwrap.proto symlink and unwrap_client.ts golden file for all 4 unwrap variants
- Verified int64/uint64 correctly mapped to TypeScript string type (proto3 JSON spec)
- Confirmed query parameter encoding matches Go server's strconv.Parse expectations
- Verified FieldViolation fields match proto exactly (field, description)
- Confirmed header handling identical to Go client (names, application order)
- All 5 tsclientgen golden file tests pass

## Task Commits

Task 1 changes were included in a parallel execution commit:

1. **Task 1: Audit and fix TS client type mapping, int64 handling, query parameter encoding, and unwrap handling** - `80d1833` (fix)
   - Added unwrap.proto symlink to tsclientgen testdata
   - Generated unwrap_client.ts golden file
   - Added test case to golden_test.go
   - Audit confirmed int64/uint64 already mapped to string

2. **Task 2: Audit and fix TS client error handling, header handling, and generated JSDoc** - No commit (audit verified consistency, no changes needed)

**Plan metadata:** Pending

## Files Created/Modified

- `internal/tsclientgen/testdata/proto/unwrap.proto` - Symlink to shared unwrap test proto
- `internal/tsclientgen/testdata/golden/unwrap_client.ts` - Golden file verifying all unwrap variants
- `internal/tsclientgen/golden_test.go` - Added test case for unwrap variants

## Decisions Made

**D-03-04-01: TS client already consistent with Go server - no fixes needed**

Systematic audit of 6 areas found complete consistency:
1. **int64/uint64 type mapping**: Already correctly mapped to `string` in TS interfaces (types.go lines 31-34)
2. **Query param encoding**: Zero-value checks use `!== "0"` for int64 (string comparison) which is correct since the TS type is string
3. **Error handling**: FieldViolation has `field` and `description` matching proto exactly
4. **Header handling**: Service-level headers in both ClientOptions and CallOptions, method-level in CallOptions only
5. **Path param encoding**: `encodeURIComponent()` produces URL-safe output compatible with server
6. **Unwrap handling**: All 4 variants (map-value, root repeated, root map, combined) produce correct TS types

**D-03-04-02: No JSDoc generation by design - minimalist generated code**

The TS client generator does not produce JSDoc comments. This is intentional design for minimalist generated code - documentation lives in proto comments and separate API docs.

## Deviations from Plan

None - plan executed exactly as written. The audit tasks were designed to find and fix inconsistencies, but the TS client was already correctly implemented.

## Issues Encountered

Task 1 commit was included in a parallel execution (`80d1833`). The work was verified to be complete and tests pass.

## Next Phase Readiness

- TypeScript client fully verified as second reference implementation
- Both Go and TS clients consistent with HTTP server
- OpenAPI review (plans 03-05/06) can proceed with confidence in cross-generator consistency
- All unwrap variants verified across all 3 generators (server, Go client, TS client)

---
*Phase: 03-existing-client-review*
*Completed: 2026-02-05*
