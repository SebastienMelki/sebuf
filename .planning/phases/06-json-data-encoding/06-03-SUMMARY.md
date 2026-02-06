---
phase: 06-json-data-encoding
plan: 03
subsystem: api
tags: [protobuf, bytes, base64, hex, encoding, json, marshal, openapi]

# Dependency graph
requires:
  - phase: 06-01
    provides: "bytes_encoding proto annotation and shared parsing functions"
  - phase: 04-02
    provides: "Protojson-base-then-modify-map pattern for MarshalJSON/UnmarshalJSON"
  - phase: 05-02
    provides: "Nullable encoding pattern for generated code (same file generation approach)"
provides:
  - "Custom MarshalJSON/UnmarshalJSON for messages with non-default bytes encoding (BASE64_RAW, BASE64URL, BASE64URL_RAW, HEX)"
  - "OpenAPI schema generation with encoding-aware format and pattern for bytes fields"
  - "Golden test coverage for bytes encoding across all 4 generators"
affects: [06-04, 07-json-complex-types]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Bytes encoding MarshalJSON: protojson base serialize, modify map for non-default encodings"
    - "Bytes encoding UnmarshalJSON: decode from custom encoding, re-encode as standard base64, pass to protojson"
    - "Dynamic imports: only import encoding/hex or encoding/base64 based on which encodings are used"

key-files:
  created:
    - internal/httpgen/bytes_encoding.go
    - internal/clientgen/bytes_encoding.go
    - internal/httpgen/testdata/proto/bytes_encoding.proto
    - internal/httpgen/testdata/golden/bytes_encoding_bytes_encoding.pb.go
    - internal/clientgen/testdata/golden/bytes_encoding_bytes_encoding.pb.go
    - internal/tsclientgen/testdata/golden/bytes_encoding_client.ts
    - internal/openapiv3/testdata/golden/yaml/BytesEncodingService.openapi.yaml
    - internal/openapiv3/testdata/golden/json/BytesEncodingService.openapi.json
  modified:
    - internal/httpgen/generator.go
    - internal/clientgen/generator.go
    - internal/openapiv3/types.go
    - internal/tsclientgen/types.go
    - internal/httpgen/golden_test.go
    - internal/clientgen/golden_test.go
    - internal/tsclientgen/golden_test.go
    - internal/openapiv3/exhaustive_golden_test.go

key-decisions:
  - "D-06-03-01: HEX UnmarshalJSON needs both encoding/hex AND encoding/base64 imports (re-encodes decoded hex bytes as standard base64 for protojson)"
  - "D-06-03-02: nolint:dupl on MarshalJSON/UnmarshalJSON across empty_behavior, timestamp_format, bytes_encoding (three similar encoding files trigger dupl threshold)"
  - "D-06-03-03: OpenAPI HEX uses format:hex with regex pattern ^[0-9a-fA-F]*$ for validation"
  - "D-06-03-04: OpenAPI BASE64URL uses format:base64url (not base64 with modifier) for clarity"

patterns-established:
  - "Dynamic import generation: writeBytesEncodingImports checks which encodings are used before adding imports"
  - "Three encoding files pattern: empty_behavior + timestamp_format + bytes_encoding with nolint:dupl on boilerplate functions"

# Metrics
duration: 8min
completed: 2026-02-06
---

# Phase 6 Plan 3: Bytes Encoding Summary

**Custom bytes encoding (HEX, BASE64_RAW, BASE64URL, BASE64URL_RAW) across go-http, go-client, ts-client, and openapiv3 generators with golden test coverage**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-02-06T16:20:00Z
- **Completed:** 2026-02-06T16:28:35Z
- **Tasks:** 2
- **Files modified:** 20+

## Accomplishments
- Go generators (httpgen/clientgen) produce identical MarshalJSON/UnmarshalJSON for 4 non-default bytes encoding variants
- OpenAPI generator produces encoding-aware schemas (format:hex with pattern, format:base64url, format:byte with descriptions)
- TypeScript client correctly maps all bytes variants to string type (no change needed)
- Golden test protos and generated files created for all 4 generators, all tests pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement bytes encoding in Go generators** - `6cf132f` (feat)
2. **Task 2: Implement bytes encoding in ts-client and openapiv3 with golden tests** - `11718a8` (feat, committed by parallel 06-02 executor which included bytes_encoding files)

## Files Created/Modified

### Created
- `internal/httpgen/bytes_encoding.go` - BytesEncodingContext, detection, validation, MarshalJSON/UnmarshalJSON code generation
- `internal/clientgen/bytes_encoding.go` - Identical to httpgen (except package name and writeEncodingHeader)
- `internal/httpgen/testdata/proto/bytes_encoding.proto` - Test proto with all 6 encoding variants (default, BASE64, BASE64_RAW, BASE64URL, BASE64URL_RAW, HEX)
- `internal/clientgen/testdata/proto/bytes_encoding.proto` - Symlink to httpgen proto
- `internal/tsclientgen/testdata/proto/bytes_encoding.proto` - Symlink to httpgen proto
- `internal/openapiv3/testdata/proto/bytes_encoding.proto` - Symlink to httpgen proto
- `internal/httpgen/testdata/golden/bytes_encoding_*.pb.go` - 4 golden files (http, binding, config, bytes_encoding)
- `internal/clientgen/testdata/golden/bytes_encoding_*.pb.go` - 2 golden files (client, bytes_encoding)
- `internal/tsclientgen/testdata/golden/bytes_encoding_client.ts` - TypeScript client golden file
- `internal/openapiv3/testdata/golden/yaml/BytesEncodingService.openapi.yaml` - OpenAPI YAML golden file
- `internal/openapiv3/testdata/golden/json/BytesEncodingService.openapi.json` - OpenAPI JSON golden file

### Modified
- `internal/httpgen/generator.go` - Added generateBytesEncodingFile call
- `internal/clientgen/generator.go` - Added generateBytesEncodingFile call
- `internal/openapiv3/types.go` - BytesKind switch with encoding-aware format/pattern
- `internal/tsclientgen/types.go` - Updated comment for bytes encoding variants
- `internal/httpgen/empty_behavior.go` - Added nolint:dupl
- `internal/clientgen/empty_behavior.go` - Added nolint:dupl
- `internal/httpgen/timestamp_format.go` - Added nolint:dupl
- `internal/clientgen/timestamp_format.go` - Added nolint:dupl

## Decisions Made

1. **D-06-03-01: HEX needs base64 import in UnmarshalJSON** - HEX decoding produces raw bytes that must be re-encoded as standard base64 for protojson.Unmarshal, requiring both encoding/hex and encoding/base64 imports.

2. **D-06-03-02: nolint:dupl across 3 encoding files** - With bytes_encoding as the third encoding file (after empty_behavior and timestamp_format), the duplicate code detection threshold was exceeded. Added nolint:dupl to MarshalJSON/UnmarshalJSON boilerplate in all three files across both httpgen and clientgen.

3. **D-06-03-03: OpenAPI HEX format** - HEX encoding uses format:hex with pattern `^[0-9a-fA-F]*$` to give API consumers clear validation guidance.

4. **D-06-03-04: OpenAPI BASE64URL format** - URL-safe base64 variants use format:base64url (distinct from format:byte) for API clarity.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] HEX encoding missing base64 import for UnmarshalJSON**
- **Found during:** Task 1 (Go generators implementation)
- **Issue:** HEX UnmarshalJSON decodes hex bytes then re-encodes as standard base64 for protojson, but the dynamic import system only added encoding/hex without encoding/base64
- **Fix:** Added `needsBase64 = true` in the HEX case of writeBytesEncodingImports
- **Files modified:** internal/httpgen/bytes_encoding.go, internal/clientgen/bytes_encoding.go
- **Verification:** Generated golden files include both imports, tests pass
- **Committed in:** 6cf132f (Task 1 commit)

**2. [Rule 3 - Blocking] exhaustive lint warnings on BytesEncoding switch**
- **Found during:** Task 1 (Go generators implementation)
- **Issue:** Switch statements on BytesEncoding enum missing UNSPECIFIED/BASE64 cases triggered exhaustive linter
- **Fix:** Added `//exhaustive:ignore` comments explaining these values are filtered before reaching the switch
- **Files modified:** internal/httpgen/bytes_encoding.go, internal/clientgen/bytes_encoding.go, internal/openapiv3/types.go
- **Verification:** make lint-fix reports 0 issues
- **Committed in:** 6cf132f (Task 1), 11718a8 (Task 2)

**3. [Rule 3 - Blocking] dupl lint warnings across 3 encoding files**
- **Found during:** Task 1 (Go generators implementation)
- **Issue:** Adding third encoding file (bytes_encoding) pushed duplicate detection threshold for MarshalJSON/UnmarshalJSON boilerplate across empty_behavior, timestamp_format, and bytes_encoding
- **Fix:** Added nolint:dupl to all affected functions in both httpgen and clientgen
- **Files modified:** internal/httpgen/empty_behavior.go, internal/httpgen/timestamp_format.go, internal/clientgen/empty_behavior.go, internal/clientgen/timestamp_format.go
- **Verification:** make lint-fix reports 0 issues
- **Committed in:** 6cf132f (Task 1 commit)

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All auto-fixes necessary for correct compilation and lint compliance. No scope creep.

## Issues Encountered

- **Parallel execution with Plan 06-02:** Both plans (06-02 timestamp_format, 06-03 bytes_encoding) ran simultaneously in wave 2. The 06-02 executor committed golden test files and types.go changes that included bytes_encoding files created by this plan. This was detected and confirmed harmless -- all changes are present in the commit history. Task 2 commit was attributed to 11718a8 (06-02's commit).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 3 encoding annotation types (int64, timestamp_format, bytes_encoding) now implemented across all 4 generators
- Plan 06-04 (cross-generator consistency tests) can verify encoding consistency
- Phase 7 (complex types) can build on established encoding patterns

---
*Phase: 06-json-data-encoding*
*Completed: 2026-02-06*
