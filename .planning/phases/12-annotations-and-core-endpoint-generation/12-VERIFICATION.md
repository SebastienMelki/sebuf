---
phase: 12-annotations-and-core-endpoint-generation
verified: 2026-02-25T14:10:00Z
status: passed
score: 5/5 must-haves verified
re_verification: false
---

# Phase 12: Annotations and Core Endpoint Generation Verification Report

**Phase Goal:** Users can run protoc-gen-krakend and get correct, minimal KrakenD endpoint fragments with auto-derived header and query string forwarding from their existing proto service definitions
**Verified:** 2026-02-25T14:10:00Z
**Status:** PASSED
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | Running `protoc --krakend_out=. service.proto` produces `{ServiceName}.krakend.json` with correct endpoint objects | VERIFIED | `TestKrakenDGoldenFiles` passes for all 6 test cases (7 services); `UserService.krakend.json` confirms correct paths, methods, backend host from annotation |
| 2 | `input_headers` auto-populated from service/method headers, `input_query_strings` from query annotations; never empty arrays | VERIFIED | `HeaderForwardingService.krakend.json` shows sorted header arrays; `NoHeaderService.krakend.json` omits `input_headers` entirely; `QueryForwardingService.krakend.json` shows sorted query strings; `NoQueryParams` RPC omits `input_query_strings` |
| 3 | Service-level `gateway_config` sets defaults; method-level `endpoint_config` overrides for individual RPCs | VERIFIED | `TimeoutService.krakend.json` shows service timeout `"3s"` inherited and `"500ms"` override; `HostService.krakend.json` shows multi-host service and single-host override |
| 4 | Generation fails with clear error for duplicate (path, method) tuples or static/param route conflicts | VERIFIED | `TestKrakenDValidationErrors` passes for all 3 cases: `"duplicate route: GET /api/v1/users"`, `"route conflict"`, `"has no (sebuf.krakend.gateway_config) annotation"` |
| 5 | Golden file tests cover routing, timeouts, forwarding, and all validation error scenarios | VERIFIED | 6 golden test cases (7 services) + 3 validation error cases; all pass byte-for-byte without `UPDATE_GOLDEN` |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `proto/sebuf/krakend/krakend.proto` | Gateway annotation definitions with GatewayConfig/EndpointConfig | VERIFIED | Exists; contains `gateway_config = 51001`, `endpoint_config = 51002`, both messages defined correctly |
| `krakend/krakend.pb.go` | Generated Go code with E_GatewayConfig and E_EndpointConfig | VERIFIED | 240 lines; `E_GatewayConfig` at line 163, `E_EndpointConfig` at line 169; `go vet` passes |
| `cmd/protoc-gen-krakend/main.go` | Plugin entry point with readRequest/createPlugin/generateFiles/writeResponse | VERIFIED | All 4 functions present; calls `krakendgen.GenerateService`, marshals JSON, writes per-service files, propagates errors via `plugin.Error` |
| `internal/krakendgen/types.go` | Endpoint and Backend structs with correct JSON tags | VERIFIED | Both structs present; `timeout`, `input_headers`, `input_query_strings` use `omitempty`; `backend` is always-present slice |
| `internal/krakendgen/generator.go` | Core generation logic with GenerateService function | VERIFIED | 220 lines; `GenerateService` reads HTTP config, gateway_config, endpoint_config, derives headers and query strings, validates routes |
| `internal/krakendgen/validation.go` | Route conflict detection with ValidateRoutes | VERIFIED | 154 lines; `ValidateRoutes` delegates to `checkDuplicateRoutes` (map-based) and `checkSegmentConflicts` (trie-based); actionable error messages |
| `internal/krakendgen/golden_test.go` | TestKrakenDGoldenFiles and TestKrakenDValidationErrors | VERIFIED | Both test functions present; 6 success cases, 3 validation error cases; UPDATE_GOLDEN support; byte-for-byte comparison |
| `internal/krakendgen/testdata/proto/` | 9 test proto files | VERIFIED | All 9 files present: simple_service, timeout_config, host_config, headers_forwarding, query_forwarding, combined_forwarding, invalid_duplicate_routes, invalid_route_conflict, invalid_no_host |
| `internal/krakendgen/testdata/golden/` | 7 golden JSON files | VERIFIED | All 7 files present: UserService, TimeoutService, HostService, HeaderForwardingService, NoHeaderService, QueryForwardingService, CombinedForwardingService |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/protoc-gen-krakend/main.go` | `internal/krakendgen/` | `import "github.com/SebastienMelki/sebuf/internal/krakendgen"` | WIRED | Calls `krakendgen.GenerateService` in `writeServiceFile` |
| `cmd/protoc-gen-krakend/main.go` | `krakend/krakend.pb.go` | Imported transitively via krakendgen | WIRED | krakendgen imports krakend package directly |
| `internal/krakendgen/generator.go` | `internal/annotations/` | `import "github.com/SebastienMelki/sebuf/internal/annotations"` | WIRED | Calls `annotations.GetMethodHTTPConfig`, `GetServiceBasePath`, `BuildHTTPPath`, `GetServiceHeaders`, `GetMethodHeaders`, `CombineHeaders`, `GetQueryParams` |
| `internal/krakendgen/generator.go` | `krakend/krakend.pb.go` | `import "github.com/SebastienMelki/sebuf/krakend"` | WIRED | Uses `krakend.E_GatewayConfig`, `krakend.E_EndpointConfig`, `krakend.GatewayConfig`, `krakend.EndpointConfig` |
| `internal/krakendgen/generator.go` | `internal/krakendgen/validation.go` | Direct function call | WIRED | `ValidateRoutes(endpoints, serviceName)` called at end of `GenerateService` before returning |
| `internal/krakendgen/golden_test.go` | `cmd/protoc-gen-krakend/main.go` | Builds and invokes binary via `make build` + `protoc --plugin` | WIRED | Test builds `bin/protoc-gen-krakend` and runs protoc with `--plugin=protoc-gen-krakend=pluginPath` |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| ANNO-01 | 12-01 | New proto package `proto/sebuf/krakend/` with gateway-specific annotations (extension numbers 51000+) | SATISFIED | `proto/sebuf/krakend/krakend.proto` exists with ext 51001, 51002 |
| ANNO-02 | 12-01 | Service-level `gateway_config` annotation for service-wide defaults | SATISFIED | `message GatewayConfig` with host/timeout; ext 51001 on ServiceOptions |
| ANNO-03 | 12-01 | Method-level `endpoint_config` annotation for per-RPC overrides | SATISFIED | `message EndpointConfig` with host/timeout; ext 51002 on MethodOptions |
| ANNO-04 | 12-02 | Method-level config always overrides service-level config for the same setting | SATISFIED | `resolveHost` and `resolveTimeout` functions in generator.go implement override semantics; verified by `HostService.krakend.json` and `TimeoutService.krakend.json` golden files |
| CORE-01 | 12-02 | Plugin reads `sebuf.http.config` annotations to extract HTTP path and method | SATISFIED | `annotations.GetMethodHTTPConfig(method)` called for each RPC; methods without config skipped silently |
| CORE-02 | 12-02 | Plugin generates one JSON file per proto service (`{ServiceName}.krakend.json`) | SATISFIED | `filename := fmt.Sprintf("%s.krakend.json", service.Desc.Name())` in `writeServiceFile`; 7 golden files confirm |
| CORE-03 | 12-02 | Backend host required via `gateway_config` annotation | SATISFIED | `getGatewayConfig` returns error if annotation missing; `TestKrakenDValidationErrors/missing_gateway_config` passes |
| CORE-04 | 12-02 | Backend host configurable via service-level annotation | SATISFIED | `resolveHost` reads from `gwConfig.GetHost()` as service default; `HostService.krakend.json` confirms multi-host |
| CORE-05 | 12-02 | Per-endpoint timeout via service-level default and method-level override | SATISFIED | `resolveTimeout` implements inheritance; `TimeoutService.krakend.json` shows both cases |
| CORE-06 | 12-02 | Output encoding defaults to JSON for all endpoints | SATISFIED | `OutputEncoding: "json"` hardcoded in generator.go; `Encoding: "json"` on Backend; every golden file confirms |
| FWD-01 | 12-03 | `input_headers` auto-populated from service_headers and method_headers | SATISFIED | `deriveInputHeaders` calls `CombineHeaders`; `HeaderForwardingService.krakend.json` shows sorted merged headers |
| FWD-02 | 12-03 | `input_query_strings` auto-populated from `sebuf.http.query` annotations | SATISFIED | `deriveInputQueryStrings` calls `annotations.GetQueryParams`; `QueryForwardingService.krakend.json` confirms |
| FWD-03 | 12-03 | Auto-derived headers and query strings are never empty arrays | SATISFIED | Both helpers return `nil` (not `[]string{}`) when no annotations exist; `omitempty` JSON tag omits field; `NoHeaderService.krakend.json` confirms absent `input_headers` |
| VALD-01 | 12-04 | Generation fails with clear error for identical (path, method) tuples | SATISFIED | `checkDuplicateRoutes` in validation.go; `TestKrakenDValidationErrors/duplicate_routes` checks error substring |
| VALD-02 | 12-04 | Generation fails with clear error for static/parameterized route conflicts | SATISFIED | `checkSegmentConflicts` trie in validation.go; `TestKrakenDValidationErrors/static_vs_param_conflict` passes |
| TEST-01 | 12-04 | Golden tests cover core generation scenarios | SATISFIED | simple_service (4 RPCs, CRUD), timeout_config, host_config golden tests pass byte-for-byte |
| TEST-03 | 12-04 | Golden tests cover auto-derived header and query string forwarding | SATISFIED | headers_forwarding (2 services), query_forwarding, combined_forwarding golden tests pass |
| TEST-04 | 12-04 | Golden tests cover generation-time validation errors | SATISFIED | 3 error cases: duplicate routes, route conflict, missing host -- all pass |

**Requirements not in Phase 12 scope** (deferred to Phase 13):
- VALD-03: Constants for extra_config namespaces (Phase 13, Pending)
- TEST-02: Gateway feature golden tests -- rate limiting, JWT, circuit breaker (Phase 13, Pending)

These are correctly marked as Phase 13 in REQUIREMENTS.md and are NOT orphaned requirements for Phase 12.

### Anti-Patterns Found

None. Scanned `internal/krakendgen/*.go`, `cmd/protoc-gen-krakend/main.go`, and `krakend/krakend.pb.go` for TODOs, FIXMEs, placeholders, empty returns, and stub implementations. Zero issues found.

### Human Verification Required

None. All success criteria are verifiable through code inspection and automated tests.

The following items could optionally be tested manually but are not blocking:
- Running `protoc --krakend_out=.` on an actual user proto file (the golden tests cover this path via the plugin binary and real protoc execution)
- Importing the generated `{ServiceName}.krakend.json` into an actual KrakenD gateway instance

### Gaps Summary

No gaps. All phase 12 requirements are satisfied and all 5 success criteria from the ROADMAP are verified against the actual codebase.

Key verification facts:
- `go build ./...` compiles clean with no errors
- `go vet ./internal/krakendgen/... ./cmd/protoc-gen-krakend/... ./krakend/...` reports no issues
- `go test ./...` passes all 8 test packages with no failures
- `TestKrakenDGoldenFiles` passes 6 test cases (7 services) with byte-for-byte golden file matches
- `TestKrakenDValidationErrors` passes all 3 error detection cases
- Binary `bin/protoc-gen-krakend` exists and is used by the test suite

---

_Verified: 2026-02-25T14:10:00Z_
_Verifier: Claude (gsd-verifier)_
