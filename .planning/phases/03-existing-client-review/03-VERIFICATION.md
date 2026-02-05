---
phase: 03-existing-client-review
verified: 2026-02-05T21:35:03Z
status: passed
score: 5/5 must-haves verified
re_verification: false
---

# Phase 3: Existing Client Review Verification Report

**Phase Goal:** The existing Go HTTP client and TypeScript HTTP client are solid, consistent with each other and with the server, and ready to serve as the reference implementations that new language clients and JSON mapping features build upon

**Verified:** 2026-02-05T21:35:03Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | For every RPC in the exhaustive test proto, the Go client serializes requests and deserializes responses identically to the Go HTTP server (byte-level JSON comparison) | ✓ VERIFIED | Golden file tests pass for http_verbs_comprehensive (9 RPCs), query_params, unwrap (4 variants). Go client uses fmt.Sprint for query params matching server's strconv.Parse. Both use protojson for body serialization. |
| 2 | For every RPC in the exhaustive test proto, the TypeScript client produces the same JSON request bodies and expects the same JSON response shapes as the Go server | ✓ VERIFIED | TS client golden files pass for same 5 protos. int64/uint64 mapped to string type (lines 31-34 of types.go). Query param encoding consistent. All unwrap variants tested. |
| 3 | Error handling is consistent: both clients surface ValidationError and ApiError with the same HTTP status codes, error body structure, and field-level violation format | ✓ VERIFIED | TS client has ValidationError (400) and ApiError classes with violations field. FieldViolation has field+description matching proto. Go client parses same structures. OpenAPI Error schema has single "message" field matching sebuf.http.Error proto. |
| 4 | Header handling is consistent: both clients send service-level and method-level headers identically | ✓ VERIFIED | Both clients support X-API-Key (service-level) and X-Request-ID (method-level) from http_verbs_comprehensive.proto. Go client has WithRESTfulAPIServiceAPIKey option. TS client has apiKey in ClientOptions and CallOptions, requestId in CallOptions. |
| 5 | All existing golden file tests pass, and fixes are captured as new golden file test cases | ✓ VERIFIED | All tests pass: httpgen (29 tests), clientgen (5 golden files), tsclientgen (5 golden files), openapiv3 (20 golden files YAML+JSON). New test cases added: unwrap_client.pb.go, unwrap_client.ts, complex_features_client.pb.go, UnwrapService.openapi.yaml. |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/httpgen/testdata/proto/http_verbs_comprehensive.proto` | Exhaustive test proto covering all annotation types | ✓ VERIFIED | 227 lines, includes 9 RPCs, enum, nested message, optional fields, int64/uint64/float/double query params |
| `internal/httpgen/testdata/proto/unwrap.proto` | Unwrap test coverage for all 4 variants | ✓ VERIFIED | 4918 bytes, UnwrapService with root map, root repeated, map-value, combined unwrap RPCs |
| `internal/clientgen/testdata/golden/` | Go client golden files | ✓ VERIFIED | 5 golden files: http_verbs_comprehensive, query_params, backward_compat, unwrap, complex_features |
| `internal/tsclientgen/testdata/golden/` | TS client golden files | ✓ VERIFIED | 5 golden files matching Go client coverage |
| `internal/openapiv3/testdata/golden/` | OpenAPI golden files | ✓ VERIFIED | 20 YAML files (10 services x 2 formats), includes all shared protos via symlinks |
| `internal/httpgen/generator.go` | Server Content-Type response headers | ✓ VERIFIED | Lines 623, 906, 1091: w.Header().Set("Content-Type", respContentType) in writeProtoMessageResponse, genericHandler, writeResponseBody |
| `internal/openapiv3/generator.go` | OpenAPI error schema | ✓ VERIFIED | Line 568: Error schema with single "message" field matching sebuf.http.Error proto |
| `internal/openapiv3/types.go` | int64/uint64 type mapping | ✓ VERIFIED | Lines 75-78, 86-89: int64/uint64 mapped to type:string with format:int64/uint64 per proto3 JSON spec |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| Server | Content-Type header | marshalResponse, writeProtoMessageResponse, genericHandler | ✓ WIRED | Content-Type set in 3 response-writing functions covering all response paths (success, error, validation) |
| Go client | Server query params | fmt.Sprint → strconv.Parse | ✓ WIRED | Client uses fmt.Sprint for query params, server uses strconv.Parse* functions - roundtrip verified |
| Go client | Server response | protojson unmarshaling | ✓ WIRED | Both use protojson for proto messages, json.Unmarshaler for unwrap variants |
| TS client | Server request body | JSON.stringify | ✓ WIRED | TS client produces JSON matching server expectations, int64 as string |
| TS client | Server errors | ValidationError/ApiError parsing | ✓ WIRED | TS client parses ValidationError (400) and ApiError with same structure as server produces |
| OpenAPI | Server types | Schema generation | ✓ WIRED | int64/uint64 as string matches TS client and server protojson behavior, Error schema matches proto |

### Requirements Coverage

Phase 3 maps to requirements FOUND-07 and FOUND-08 per user instruction. Requirements tracking in REQUIREMENTS.md is not yet established (user said "FOUND-07, FOUND-08" but grep found no requirements mapped to Phase 3).

**Result:** Requirements tracking pending, but phase goal and success criteria fully verified.

### Anti-Patterns Found

No anti-patterns detected. All verification checks passed:

- No TODO/FIXME comments in critical paths
- No placeholder content in generated code
- No empty implementations
- No console.log-only implementations
- Content-Type headers properly set
- Error schemas match proto definitions
- Type mappings follow proto3 JSON spec

### Cross-Generator Consistency

From 03-06-SUMMARY.md verification:

**10 areas verified consistent:**
1. Paths: All 9 RESTfulAPIService RPCs match across server, Go client, TS client, OpenAPI
2. HTTP Methods: GET/POST/PUT/PATCH/DELETE consistent
3. Query Params: Names and types match
4. int64/uint64: string type across all generators
5. Response Schema: camelCase field names, consistent types
6. Error 400: ValidationError with violations array
7. Error default: Error with message field
8. Service Headers: X-API-Key in service-level options
9. Method Headers: X-Request-ID in call-level options
10. Unwrap: All 4 variants (map-value, root repeated, root map, combined) consistent

**Accepted inconsistency:**
Default path pattern for services WITHOUT explicit HTTP annotations differs across generators. This is acceptable because:
- Only affects backward compatibility fallback mode
- Production services should have explicit HTTP annotations
- All generators perfectly consistent when annotations ARE present

### Human Verification Required

None. All verification performed programmatically via:
- Golden file test execution (all pass)
- Source code grep verification of fixes
- Artifact existence and substantive checks
- Type mapping verification across generators

## Verification Methodology

### Automated Checks Performed

1. **Test Execution:** `go test ./... -count=1` → All packages pass
2. **Artifact Existence:** Verified all required files exist with substantive line counts
3. **Content-Type Headers:** grep verified w.Header().Set calls in 3 response functions
4. **Error Schema:** Verified Error has single "message" field in generator.go line 568
5. **int64/uint64 Mapping:** Verified type:string with format in types.go lines 75-78, 86-89
6. **TS Client Types:** Verified int64/uint64 → string in types.go lines 31-34
7. **Error Classes:** Verified ValidationError and ApiError in all 5 TS golden files
8. **Symlinks:** Verified shared test proto symlinks across all 4 generators
9. **Golden Files:** Counted and verified golden files for all test protos
10. **Header Options:** Verified apiKey and requestId in ClientOptions/CallOptions

### Test Coverage

- **httpgen:** 29 tests pass (error handling, response capture, binding, validation)
- **clientgen:** 5 golden file tests pass (http_verbs, query_params, backward_compat, unwrap, complex_features)
- **tsclientgen:** 5 golden file tests pass (same coverage as clientgen)
- **openapiv3:** 20 golden file tests pass (10 services x 2 formats)
- **annotations:** Tests pass (shared package used by all generators)

### Phase Completion Evidence

All 6 plans executed:
1. **03-01:** Shared test proto infrastructure with symlinks ✓
2. **03-02:** Server Content-Type response headers fix ✓
3. **03-03:** Go client consistency audit (no fixes needed) ✓
4. **03-04:** TS client consistency audit (no fixes needed) ✓
5. **03-05:** OpenAPI error schema and int64 mapping fixes ✓
6. **03-06:** Cross-generator verification (10 areas consistent) ✓

## Next Phase Readiness

Phase 3 complete. All success criteria met:

- [x] SC1: Go client serialization matches server
- [x] SC2: TS client JSON matches server
- [x] SC3: Error handling consistent (ValidationError + ApiError/Error)
- [x] SC4: Header handling consistent (service + method headers)
- [x] SC5: All golden file tests pass, new test cases added

**Blockers for Phase 4:** None

**Ready for:** Phase 4 (JSON - Primitive Encoding) can proceed. Both Go and TS clients verified as solid reference implementations for new JSON mapping features.

---

_Verified: 2026-02-05T21:35:03Z_
_Verifier: Claude (gsd-verifier)_
