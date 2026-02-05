---
phase: 03-existing-client-review
plan: 02
subsystem: httpgen-server
tags: [content-type, http-headers, response-serialization, server-correctness]
dependency-graph:
  requires: []
  provides:
    - "Server sets Content-Type on all HTTP responses"
    - "Server defaults to JSON for unknown content types (request and response)"
    - "Consistent content-type handling across request parsing and response serialization"
  affects:
    - "03-03 (Go client review) -- client can now rely on response Content-Type"
    - "03-04 (TS client review) -- client can now rely on response Content-Type"
tech-stack:
  added: []
  patterns:
    - "Response Content-Type mirroring: response serialization format matches request Content-Type"
    - "JSON-by-default: unknown/missing content types default to JSON in all paths"
key-files:
  created: []
  modified:
    - internal/httpgen/generator.go
    - internal/httpgen/testdata/golden/backward_compat_http_binding.pb.go
    - internal/httpgen/testdata/golden/http_verbs_comprehensive_http_binding.pb.go
    - internal/httpgen/testdata/golden/query_params_http_binding.pb.go
    - internal/httpgen/testdata/golden/unwrap_http_binding.pb.go
    - internal/httpgen/testdata/golden/http_verbs_comprehensive_http.pb.go
    - internal/httpgen/testdata/proto/http_verbs_comprehensive.proto
decisions:
  - id: D-03-02-01
    description: "JSON default for unknown content types everywhere -- bindDataBasedOnContentType, marshalResponse, writeProtoMessageResponse, writeResponseBody all default to JSON"
  - id: D-03-02-02
    description: "Content-Type set in three response-writing functions: writeProtoMessageResponse, genericHandler success path, writeResponseBody. All other paths delegate to these."
metrics:
  duration: 4m
  completed: 2026-02-05
---

# Phase 03 Plan 02: Server Content-Type Response Headers Summary

Server correctness fix: Content-Type headers on all HTTP responses and consistent JSON default for unknown content types.

## What Was Done

### Task 1: Fix Content-Type response header and marshalResponse default behavior

Fixed three categories of issues in the generated HTTP server code:

**Content-Type Response Headers (3 functions):**

1. **`writeProtoMessageResponse`** -- Added `respContentType` variable set per-case in the content-type switch, then `w.Header().Set("Content-Type", respContentType)` before `w.WriteHeader()`. This function is the main response writer used by all error response paths.

2. **`genericHandler` success path** -- Added Content-Type determination based on request Content-Type (JSON for default/JSON, protobuf for binary/proto), set before `w.Write(responseBytes)`.

3. **`writeResponseBody`** -- Same pattern as `writeProtoMessageResponse`: `respContentType` variable per-case, then `w.Header().Set` before `w.Write`.

**marshalResponse default to JSON:**

Changed the `default` case from returning `fmt.Errorf("unsupported content type: %s", contentType)` to defaulting to JSON serialization (checking for `json.Marshaler` first for unwrap support, then `protojson.Marshal`).

**bindDataBasedOnContentType default to JSON:**

Changed the `default` case from `bindDataFromBinaryRequest` to `bindDataFromJSONRequest` for consistency. Request parsing and response serialization now both default to JSON for unknown/missing content types.

### Task 2: Verify error response functions also set Content-Type headers

Audited all response-writing paths in the generated server code:

| Function | Content-Type Mechanism | Coverage |
|----------|----------------------|----------|
| `writeProtoMessageResponse` | Sets directly (Task 1) | All error responses |
| `writeValidationErrorResponse` | Delegates to writeProtoMessageResponse | Validation errors |
| `writeValidationError` | Delegates via writeValidationErrorResponse | Protovalidate errors |
| `writeErrorResponse` | Delegates to writeProtoMessageResponse | Handler errors |
| `writeErrorWithHandler` | Delegates to writeProtoMessageResponse or writeResponseBody | Custom error handlers |
| `writeResponseBody` | Sets directly (Task 1) | Custom handler with pre-set status |
| `genericHandler` success | Sets directly (Task 1) | Successful responses |
| `http.Error()` fallback | Go stdlib sets text/plain | Marshal failure fallback |

**Result:** No additional changes needed. All paths covered by Task 1's three-function fix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] bindDataBasedOnContentType default changed from binary to JSON**

- **Found during:** Task 1
- **Issue:** Request parsing defaulted to binary for unknown content types while response serialization was being changed to default to JSON. This created an asymmetry.
- **Fix:** Changed `bindDataBasedOnContentType` default case from `bindDataFromBinaryRequest` to `bindDataFromJSONRequest`
- **Files modified:** `internal/httpgen/generator.go`
- **Commit:** 1cd141a

**2. [Rule 3 - Blocking] Pre-existing proto test file changes included**

- **Found during:** Task 1 golden file update
- **Issue:** The `http_verbs_comprehensive.proto` test file had been enhanced with additional test coverage (SearchResources method, enum query params) in a prior session. Running the golden file update regenerated ALL golden files from this modified proto, requiring inclusion of both the proto change and the `_http.pb.go` golden file.
- **Fix:** Included the proto and non-binding golden file in the commit to maintain test consistency
- **Files modified:** `internal/httpgen/testdata/proto/http_verbs_comprehensive.proto`, `internal/httpgen/testdata/golden/http_verbs_comprehensive_http.pb.go`
- **Commit:** 1cd141a

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| D-03-02-01 | JSON default everywhere | Consistency: all 4 content-type switch statements (bindData, marshalResponse, writeProtoMessage, writeResponseBody) use JSON as default. Clients can reliably expect JSON when Content-Type is unknown/missing. |
| D-03-02-02 | Three-function Content-Type coverage | writeProtoMessageResponse covers all error paths (validation, handler, header validation). genericHandler covers success path. writeResponseBody covers custom-handler-with-status path. No redundant settings. |

## Verification Results

- `go test ./internal/httpgen/ -count=1` -- PASS (29/29 tests)
- `make build` -- PASS
- `make lint-fix` -- 0 issues
- Content-Type `w.Header().Set` appears in all 4 binding golden files, 3 times each
- `"unsupported content type"` error removed from all golden files
- `"Default to JSON for unrecognized content types"` comment present in all default cases

## Next Phase Readiness

No blockers for subsequent plans. The server now establishes a correct baseline:
- Clients can detect response format from Content-Type header
- Unknown content types default to JSON consistently
- Plan 03 (Go client) and Plan 04 (TS client) can rely on Content-Type being set
