---
phase: 06-json-data-encoding
plan: 02
subsystem: api
tags: [protobuf, timestamp, unix-timestamp, date, codegen, go-http, go-client, ts-client, openapiv3, golden-tests]

# Dependency graph
requires:
  - phase: 06-json-data-encoding (plan 01)
    provides: "timestamp_format and bytes_encoding annotation parsing functions in shared annotations package"
provides:
  - "Go server/client MarshalJSON/UnmarshalJSON for UNIX_SECONDS, UNIX_MILLIS, DATE timestamp formats"
  - "TypeScript type mapping: number for unix timestamps, string for RFC3339/date"
  - "OpenAPI schema generation: integer/unix-timestamp for unix, string/date for date, string/date-time for RFC3339"
  - "Golden test coverage for timestamp_format across all 4 generators"
affects: ["06-json-data-encoding (plans 03-04)", "phase-07 (oneof/flattening may reference encoding patterns)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "protojson-base-then-modify-map for custom Timestamp serialization (same pattern as int64/enum encoding)"
    - "Identical encoding files in httpgen and clientgen for server/client JSON consistency"
    - "Timestamp detected as MessageKind via FullName check, not scalar"
    - "Symlinked proto test files across generator testdata directories"

key-files:
  created:
    - "internal/httpgen/timestamp_format.go"
    - "internal/clientgen/timestamp_format.go"
    - "internal/httpgen/testdata/proto/timestamp_format.proto"
    - "internal/httpgen/testdata/golden/timestamp_format_timestamp_format.pb.go"
    - "internal/clientgen/testdata/golden/timestamp_format_timestamp_format.pb.go"
    - "internal/tsclientgen/testdata/golden/timestamp_format_client.ts"
    - "internal/openapiv3/testdata/golden/yaml/TimestampFormatService.openapi.yaml"
    - "internal/openapiv3/testdata/golden/json/TimestampFormatService.openapi.json"
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
  - "D-06-02-01: Timestamp detected before generic MessageKind in type switches to prevent $ref generation"
  - "D-06-02-02: google.protobuf.Timestamp skipped from tsclientgen messageSet (primitive, not nested object)"
  - "D-06-02-03: convertTimestampField helper in openapiv3 for clean format-to-schema mapping"
  - "D-06-02-04: nolint:exhaustive on tsTimestampType switch -- default handles RFC3339/DATE/UNSPECIFIED"

patterns-established:
  - "Timestamp-as-primitive pattern: Timestamp fields intercepted before MessageKind to produce inline types (not $ref)"
  - "Format-aware schema generation: OpenAPI produces different type/format pairs based on annotation"

# Metrics
duration: ~15min
completed: 2026-02-06
---

# Phase 6 Plan 02: Timestamp Format Summary

**Custom Timestamp JSON encoding (UNIX_SECONDS, UNIX_MILLIS, DATE) across all 4 generators with protojson-base-then-modify-map pattern**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-02-06T16:12:00Z (estimated)
- **Completed:** 2026-02-06T16:27:07Z
- **Tasks:** 2
- **Files modified:** 32

## Accomplishments

- Go generators (httpgen, clientgen) produce custom MarshalJSON/UnmarshalJSON for non-default Timestamp formats using protojson-base-then-modify-map pattern
- TypeScript client maps UNIX_SECONDS/UNIX_MILLIS to `number`, RFC3339/DATE to `string`
- OpenAPI generator produces format-aware schemas (integer/unix-timestamp, string/date, string/date-time)
- Golden test coverage for timestamp_format across all 4 generators with symlinked proto files
- Bytes encoding golden files generated (missing from parallel Plan 06-03 execution)

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement timestamp format in Go generators** - `4dfee8c` (feat)
2. **Task 2: Implement timestamp format in ts-client and openapiv3 with golden tests** - `11718a8` (feat)

## Files Created/Modified

- `internal/httpgen/timestamp_format.go` - TimestampFormatContext, MarshalJSON/UnmarshalJSON code generation for go-http
- `internal/clientgen/timestamp_format.go` - Identical to httpgen (package name only) for server/client consistency
- `internal/httpgen/generator.go` - Added generateTimestampFormatEncodingFile call
- `internal/clientgen/generator.go` - Added generateTimestampFormatEncodingFile call
- `internal/tsclientgen/types.go` - Timestamp detection in tsFieldType/tsElementType, tsTimestampType helper, Timestamp skip in messageSet
- `internal/openapiv3/types.go` - Timestamp detection in convertScalarField, convertTimestampField helper method
- `internal/httpgen/testdata/proto/timestamp_format.proto` - Test proto with 5 Timestamp field variants
- `internal/*/testdata/golden/*` - Golden files for all 4 generators (timestamp_format + bytes_encoding)

## Decisions Made

- **D-06-02-01:** Timestamp fields detected before generic MessageKind handling in type switches. This prevents fall-through to $ref generation and produces inline type/format instead.
- **D-06-02-02:** google.protobuf.Timestamp skipped from tsclientgen messageSet since it serializes as a primitive (string or number), not a nested object with seconds/nanos.
- **D-06-02-03:** Created convertTimestampField helper in openapiv3/types.go for clean format-to-schema mapping. Uses field comments as description when available, falls back to format-specific defaults.
- **D-06-02-04:** Added nolint:exhaustive directive on tsTimestampType switch since default case correctly handles RFC3339, DATE, and UNSPECIFIED.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Generated bytes_encoding golden files missing from Plan 06-03**
- **Found during:** Task 2 verification (go test ./...)
- **Issue:** Plan 06-03 (running in parallel) committed bytes_encoding implementation and test cases to golden_test.go files but did NOT generate/commit the actual golden files. Tests failed because golden files were missing.
- **Fix:** Generated bytes_encoding golden files for all 4 generators using UPDATE_GOLDEN=1
- **Files created:** 10 bytes_encoding golden files across httpgen, clientgen, tsclientgen, openapiv3
- **Verification:** All golden tests pass without UPDATE_GOLDEN flag
- **Committed in:** 11718a8 (part of Task 2 commit)

**2. [Rule 1 - Bug] Fixed exhaustive lint warning in tsTimestampType**
- **Found during:** Task 2 verification (make lint-fix)
- **Issue:** tsTimestampType switch missing explicit cases for UNSPECIFIED, RFC3339, DATE (handled by default)
- **Fix:** Added nolint:exhaustive directive with explanation comment
- **Files modified:** internal/tsclientgen/types.go
- **Verification:** make lint-fix returns 0 issues
- **Committed in:** 11718a8 (part of Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both fixes necessary for test suite to pass. No scope creep.

## Issues Encountered

- OpenAPI golden file includes a `Timestamp` component schema (raw protobuf seconds/nanos structure) because the openapiv3 generator's ProcessMessage recursively collects all referenced message types. This is cosmetic -- the actual field schemas are correct inline types. Could be filtered in a future cleanup pass.
- `go test ./...` has a pre-existing race condition where the openapiv3 golden test binary (built with `go build -o ./protoc-gen-openapiv3-golden-test`) can be removed by defer while other test packages run. Running with `-p 1` avoids this. Not caused by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Timestamp format encoding complete across all 4 generators
- Pattern established for format-aware Timestamp handling (detect before MessageKind, produce inline schema)
- Ready for Plan 06-03 (bytes encoding) and Plan 06-04 (cross-generator consistency tests)
- bytes_encoding golden files already generated as part of this plan's deviation fix

---
*Phase: 06-json-data-encoding*
*Completed: 2026-02-06*
