# Phase 3: Existing Client Review - Research

**Researched:** 2026-02-05
**Domain:** Cross-generator consistency audit for Go HTTP server, Go HTTP client, TypeScript HTTP client, and OpenAPI v3 generators -- all measured against the proto3 JSON (protojson) specification
**Confidence:** HIGH

## Summary

This research investigates the current state of all four sebuf generators to identify the patterns, tools, and pitfalls relevant to a comprehensive cross-generator consistency audit. The phase is an audit-and-fix phase, not a feature phase -- the research focuses on understanding what the generators currently do, where inconsistencies are likely to exist, and how to structure the audit work.

The codebase already has solid infrastructure: four generators share the `internal/annotations` package for configuration extraction, all use golden file testing, and the test proto files cover HTTP verbs, query parameters, headers, unwrap variants, and backward compatibility. However, the generators were built at different times and have never been cross-audited for semantic JSON consistency. The test protos are shared via symlinks for some generators but not all, and there is no dedicated cross-generator comparison test.

The protojson specification (google.golang.org/protobuf/encoding/protojson) is the source of truth for JSON serialization. Key spec behaviors that must be verified across all generators: field names map to lowerCamelCase, default/zero values are omitted from output, 64-bit integers serialize as strings, enums serialize as string names, bytes as base64, and optional fields follow presence semantics.

**Primary recommendation:** Structure the audit around a shared exhaustive test proto that exercises every annotation combination, then systematically compare all four generator outputs for each RPC, verifying semantic JSON identity, query/path parameter consistency, error response consistency, and OpenAPI schema accuracy.

## Standard Stack

The established libraries/tools already in use -- no new dependencies needed for this phase:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/protobuf | v1.36.11 | Proto runtime + protojson | Official Google protobuf library, source of truth for JSON mapping |
| google.golang.org/protobuf/encoding/protojson | (same module) | JSON serialization/deserialization | Canonical proto3 JSON implementation |
| google.golang.org/protobuf/compiler/protogen | (same module) | Plugin framework for all 4 generators | Official protoc plugin API |
| github.com/pb33f/libopenapi | v0.33.0 | OpenAPI v3.1 document model | Used by openapiv3 generator for schema construction |
| buf.build/go/protovalidate | (buf.validate) | Request validation | Used by server for automatic validation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| go.yaml.in/yaml/v4 | v4.0.0-rc.4 | YAML serialization for OpenAPI output | OpenAPI golden file generation |
| sigs.k8s.io/yaml | v1.6.0 | YAML-to-JSON conversion | OpenAPI JSON format output |

### Tools
| Tool | Purpose | When to Use |
|------|---------|-------------|
| protoc | Proto compiler | Golden file test execution |
| make build | Build all 4 plugin binaries | Required before golden file tests |
| UPDATE_GOLDEN=1 | Update golden files after intentional changes | After each fix that changes generated output |

**No new dependencies required.** This phase uses existing tooling exclusively.

## Architecture Patterns

### Current Generator Architecture

```
internal/
  annotations/       # Shared annotation parsing (Phase 2 output)
    doc.go           # Package documentation
    http_config.go   # GetMethodHTTPConfig, GetServiceBasePath
    headers.go       # GetServiceHeaders, GetMethodHeaders, CombineHeaders
    query.go         # GetQueryParams (returns unified QueryParam struct)
    unwrap.go        # HasUnwrapAnnotation, FindUnwrapField, IsRootUnwrap
    field_examples.go# GetFieldExamples
    path.go          # ExtractPathParams, BuildHTTPPath
    method.go        # HTTPMethodToString, HTTPMethodToLower
    helpers.go       # LowerFirst
  httpgen/           # Go HTTP server generator
    generator.go     # Main generation (1554 lines)
    unwrap.go        # Unwrap JSON method generation
    validation.go    # Validation code generation
  clientgen/         # Go HTTP client generator
    generator.go     # Full client generation (714 lines)
  tsclientgen/       # TypeScript HTTP client generator
    generator.go     # Client class generation (437 lines)
    types.go         # Type mapping, interface generation (294 lines)
    helpers.go       # String helpers (34 lines)
  openapiv3/         # OpenAPI v3.1 specification generator
    generator.go     # Schema/path generation (625 lines)
    types.go         # Type conversion, header mapping (322 lines)
    validation.go    # buf.validate constraint extraction (412 lines)
```

### Test Infrastructure Architecture

```
internal/
  httpgen/testdata/
    proto/              # Canonical test protos (source of truth)
      backward_compat.proto
      http_verbs_comprehensive.proto
      query_params.proto
      unwrap.proto
      same_pkg_service.proto
      same_pkg_wrapper.proto
    golden/             # 13 golden files (http, binding, config, unwrap)
  clientgen/testdata/
    proto/              # Symlinks to httpgen/testdata/proto/
      backward_compat.proto -> ../../../httpgen/testdata/proto/backward_compat.proto
      http_verbs_comprehensive.proto -> (same pattern)
      query_params.proto -> (same pattern)
    golden/             # 3 golden files (client.pb.go per proto)
  tsclientgen/testdata/
    proto/              # Mix of symlinks + unique proto
      backward_compat.proto -> (symlink)
      http_verbs_comprehensive.proto -> (symlink)
      query_params.proto -> (symlink)
      complex_features.proto  # TS-specific: enums, unwrap, maps, optional
    golden/             # 4 golden files (client.ts per proto)
  openapiv3/testdata/
    proto/              # Independent proto files (NOT symlinked)
      simple_service.proto
      complex_types.proto
      headers.proto
      http_annotations.proto
      http_verbs.proto
      multiple_services.proto
      nested_messages.proto
      no_services.proto
      validation_constraints.proto
      unwrap.proto
    golden/
      yaml/             # 14 YAML golden files (per-service)
      json/             # 14 JSON golden files (per-service)
```

### Pattern 1: Golden File Testing
**What:** Each generator runs protoc with its plugin, captures output, compares byte-for-byte against golden files.
**When to use:** Every test run; UPDATE_GOLDEN=1 after intentional changes.
**Key detail:** All 4 generators use this pattern identically. The test infrastructure is consistent across generators.

### Pattern 2: Shared Proto Symlinks
**What:** clientgen and tsclientgen symlink to httpgen's test protos for shared test cases.
**When to use:** When the same proto definitions should produce consistent output across generators.
**Gap found:** openapiv3 does NOT share test protos -- it has independent, different proto files. This is a major gap for cross-generator consistency checking.

### Pattern 3: Annotation-Driven Code Generation
**What:** All 4 generators read the same annotations via `internal/annotations` package, then each generates its own output format.
**When to use:** This is the fundamental architecture -- every RPC method config, header config, query param config flows through shared annotation extraction.
**Audit implication:** If annotations are parsed consistently (Phase 2 guarantee), inconsistencies must be in how each generator applies those annotations to its output format.

### Anti-Patterns to Avoid
- **Fixing one generator without checking others:** Every fix must be verified across all 4 generators.
- **Updating golden files without reviewing the diff:** Golden file updates must be intentional and reviewed for correctness.
- **Testing JSON serialization without also testing protobuf binary:** The decision requires both content type paths to be audited.
- **Adding test cases to only one generator:** New test cases should cover all generators that handle the same functionality.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON serialization for Go types | Custom JSON encoding | `protojson.Marshal` / `protojson.Unmarshal` | protojson handles all proto3 JSON spec edge cases (64-bit as string, enum names, default omission) |
| Proto3 field name mapping | Manual camelCase conversion | `field.Desc.JSONName()` | protoreflect already knows the correct JSON name per the spec |
| Query param zero-value checks | Hardcoded zero values | Derive from proto field kind | Both Go and TS generators already do this, but must verify they agree |
| Error type deserialization | Custom error parsing | `sebufhttp.ValidationError` / `sebufhttp.Error` proto messages | Both clients already use these; verify they parse identically |
| Path parameter extraction | Regex parsing | `annotations.ExtractPathParams` / `annotations.GetMethodHTTPConfig` | Already shared via annotations package |

**Key insight:** The primary risk in this phase is not missing libraries but inconsistencies in how the four generators apply the same shared annotations to their respective output formats. The audit is about verifying behavioral consistency, not adding new capabilities.

## Common Pitfalls

### Pitfall 1: Server Uses Request Content-Type for Response Format
**What goes wrong:** The Go HTTP server reads `Content-Type` from the request header and uses it to determine both request parsing AND response serialization format. This means if a client sends `application/json`, the server responds in JSON, and if `application/x-protobuf`, the server responds in protobuf.
**Why it happens:** The `marshalResponse` function at line 630-653 in httpgen/generator.go reads `r.Header.Get("Content-Type")` for the response format.
**How to avoid:** Document this behavior clearly. Both clients must be aware that request Content-Type controls response format. The Go client correctly passes `contentType` to both marshal and unmarshal. The TS client only supports JSON (hardcoded `"Content-Type": "application/json"`).
**Warning signs:** A mismatch where the client sends one Content-Type but expects another response format.

### Pitfall 2: Query Parameter Default Value Semantics Differ Between Go and TS
**What goes wrong:** The Go client uses Go zero-value comparisons (`req.Field != 0`, `req.Field != ""`) to decide whether to include a query parameter. The TS client uses JavaScript truthy checks and explicit zero comparisons. These may not agree on edge cases.
**Why it happens:** Go's `fmt.Sprint()` on an int64 produces a numeric string; the TS client uses `String()` which also produces a numeric string. But the Go client checks `req.Offset != 0` (int64 zero) while the TS client checks `req.offset !== "0"` (string zero for int64 which is represented as a string in TS).
**How to avoid:** Verify that for every query param type, both clients produce the same encoding and the same omission behavior for zero values. Pay special attention to int64/uint64 which are strings in TS but numeric in Go.
**Warning signs:** Query strings that differ between Go and TS for the same proto input.

### Pitfall 3: Go Client Query Params Always Use `fmt.Sprint()` Regardless of Type
**What goes wrong:** The Go client generator at line 602 uses `fmt.Sprint(req.Field)` for ALL query parameter types. For most types this works, but for boolean it produces "true"/"false" which is correct, and for int64 it produces numeric strings. However, fmt.Sprint on a float32 may produce different precision than expected.
**Why it happens:** The generator uses a single `fmt.Sprint()` call for all field types rather than type-specific formatting.
**How to avoid:** Verify that `fmt.Sprint()` output matches what the server expects for each scalar type in `convertStringToFieldValue()`.
**Warning signs:** Parsing errors when the server receives query params from the Go client.

### Pitfall 4: TS Client Hardcoded to JSON Only
**What goes wrong:** The TypeScript client always sends `"Content-Type": "application/json"` and always calls `JSON.stringify(req)` for request bodies and `resp.json()` for responses. It does not support protobuf binary format.
**Why it happens:** TypeScript in browser/Node.js environments typically use JSON. Protobuf binary support would require a runtime library.
**How to avoid:** This is an intentional design choice but must be explicitly documented. The audit should verify that the TS client's JSON-only behavior is fully consistent with the server's JSON path. The Go client supports both JSON and protobuf; the TS client supports only JSON.
**Warning signs:** Any TS client behavior that assumes protobuf binary format.

### Pitfall 5: OpenAPI Error Response Schema Inconsistent with Actual Error Types
**What goes wrong:** The OpenAPI generator builds error responses with a generic inline schema (`error` string + `code` integer) for the "default" error response (lines 434-437 in openapiv3/generator.go). But the actual server returns `sebufhttp.Error` which has only a `message` field. Also, the 400 response correctly references `#/components/schemas/ValidationError`, but the inline "default" schema doesn't match the actual `sebufhttp.Error` proto definition.
**Why it happens:** The OpenAPI error schemas were likely defined before the proto error types were finalized and never updated to match.
**How to avoid:** Verify that all OpenAPI error response schemas exactly match the proto definitions of `sebufhttp.Error` and `sebufhttp.ValidationError`. The inline default error schema should reference the `Error` component instead.
**Warning signs:** API consumers generate client code from OpenAPI that doesn't match actual server responses.

### Pitfall 6: Go Client Error Response Parsing Uses Response Content-Type, Not Request Content-Type
**What goes wrong:** The Go client's `handleErrorResponse` at line 631 passes `contentType` (from the request's content type setting, not the response's Content-Type header) to `unmarshalResponse`. Since the server uses request Content-Type for response format, this actually works correctly -- but only by coincidence of the server's design.
**Why it happens:** The client assumes the server will respond in the same format the client requested.
**How to avoid:** This is actually correct behavior given the server's design, but it's fragile. Document the contract explicitly: "server responds in the same format as the request Content-Type."
**Warning signs:** If the server ever starts returning a different Content-Type than what was requested.

### Pitfall 7: Missing Response Content-Type Header Setting on Server
**What goes wrong:** The Go HTTP server generates response bytes but never sets the `Content-Type` header on the response. The `writeProtoMessageResponse` function writes bytes to `w` with `w.WriteHeader(statusCode)` but does not call `w.Header().Set("Content-Type", ...)`. This means responses have no explicit Content-Type header, relying on Go's `http.DetectContentType` default behavior.
**Why it happens:** The generator focuses on request parsing and response serialization but skips the response header.
**How to avoid:** The server should set `Content-Type` on all responses to match the format used for serialization.
**Warning signs:** Clients that rely on the response Content-Type header to determine how to parse the response will fail.

### Pitfall 8: Shared Test Proto Coverage Gaps
**What goes wrong:** The test protos across generators don't cover the same features. The httpgen has unwrap.proto but clientgen doesn't. The tsclientgen has complex_features.proto (enums, optional, nested, unwrap variants) but clientgen doesn't. OpenAPI has 10+ different proto files that are completely independent from the other generators.
**Why it happens:** Each generator's tests were developed independently.
**How to avoid:** Create a unified exhaustive test proto that all 4 generators share. This ensures every annotation combination is tested consistently.
**Warning signs:** A fix in one generator that passes its tests but would fail the same scenario in another generator.

### Pitfall 9: Enum Serialization Consistency
**What goes wrong:** The proto3 JSON spec says enums serialize as their string names. The Go server uses `protojson.Marshal` which does this automatically. But the TS client uses `JSON.stringify(req)` on a plain JavaScript object where enum values are just strings -- the client must send the same enum string names that the server expects.
**Why it happens:** The Go client uses `protojson.Marshal` which handles enums correctly, but the TS client uses standard `JSON.stringify` which just sends whatever string value was provided.
**How to avoid:** Verify that TS interface types for enums match the proto enum value names (not numbers). The current `generateEnumType` in tsclientgen/types.go already generates string union types with enum names, which is correct.
**Warning signs:** The TS client sending a numeric enum value that the server rejects.

### Pitfall 10: Int64 Query Parameter Handling Differs Between Go Client and Server
**What goes wrong:** The Go client uses `fmt.Sprint(req.Offset)` for int64 query params, which produces a decimal number string (e.g., "12345678901234"). The server's `bindQueryParams` uses `convertStringToFieldValue` with `strconv.ParseInt` which accepts this format. But protojson represents int64 as a JSON string. The query parameter path correctly uses numeric strings (not JSON-quoted strings), which is consistent since query params are not JSON.
**Why it happens:** Query parameters are URL-encoded strings, not JSON values, so the int64-as-string JSON convention does not apply to them.
**How to avoid:** Clearly distinguish between JSON serialization (where int64 is a quoted string like `"12345"`) and query parameter serialization (where int64 is just `12345`). Both clients must use the numeric string format for query params, not the JSON-quoted format.
**Warning signs:** TS client sending `"12345"` (with quotes) as a query param value.

## Code Examples

### Go Client: Query Parameter Encoding (Current)
```go
// Generated by protoc-gen-go-client (from generator.go:596-604)
// Same pattern for all scalar types:
if req.Offset != 0 {
    queryParams.Set("offset", fmt.Sprint(req.Offset))
}
```

### TS Client: Query Parameter Encoding (Current)
```typescript
// Generated by protoc-gen-ts-client (from generator.go:340-358)
// For int64 (represented as string in TS):
if (req.offset != null && req.offset !== "0") params.set("offset", String(req.offset));
// For int32:
if (req.limit != null && req.limit !== 0) params.set("limit", String(req.limit));
// For bool:
if (req.active) params.set("active", String(req.active));
```

### Server: Request Body Parsing (Current)
```go
// Generated by protoc-gen-go-http (from generator.go:366-393)
// JSON path:
func bindDataFromJSONRequest[Req any](r *http.Request, toBind *Req) error {
    // Check for custom JSON unmarshaler (unwrap support)
    if unmarshaler, ok := any(toBind).(json.Unmarshaler); ok {
        return unmarshaler.UnmarshalJSON(bodyBytes)
    }
    return protojson.Unmarshal(bodyBytes, protoRequest)
}
```

### Server: Response Serialization (Current)
```go
// Generated by protoc-gen-go-http (from generator.go:629-653)
func marshalResponse(r *http.Request, response any) ([]byte, error) {
    contentType := r.Header.Get("Content-Type")  // Uses REQUEST Content-Type
    switch filterFlags(contentType) {
    case JSONContentType:
        if marshaler, ok := response.(json.Marshaler); ok {
            return marshaler.MarshalJSON()  // Unwrap path
        }
        return protojson.Marshal(msg)  // Standard protojson path
    case BinaryContentType, ProtoContentType:
        return proto.Marshal(msg)
    default:
        return nil, fmt.Errorf("unsupported content type: %s", contentType)
    }
}
```

### OpenAPI: Error Response Schema (Current -- Incorrect)
```yaml
# The "default" error response uses an inline schema that does NOT match
# the actual sebufhttp.Error proto message which has only "message" field:
responses:
  default:
    description: Error response
    content:
      application/json:
        schema:
          type: object
          properties:
            error:    # WRONG: actual field is "message"
              type: string
            code:     # WRONG: does not exist in sebufhttp.Error
              type: integer
```

### Cross-Generator Comparison Pattern
```
For each RPC in the exhaustive test proto:
1. Go server: What does bindDataFromJSONRequest expect? What does marshalResponse produce?
2. Go client: What does marshalRequest send? What does unmarshalResponse expect?
3. TS client: What does JSON.stringify(req) send? What does resp.json() expect?
4. OpenAPI: Do the request/response schemas match all of the above?
```

## Identified Inconsistencies (Pre-Audit Findings)

These are inconsistencies identified during research. The audit will likely find more.

### Finding 1: OpenAPI Default Error Schema Wrong
**Severity:** HIGH
**What:** The OpenAPI default error response schema has `error` and `code` fields, but `sebufhttp.Error` has only a `message` field.
**Location:** openapiv3/generator.go lines 434-437
**Fix:** Reference `#/components/schemas/Error` or build schema from the actual `sebufhttp.Error` proto definition.

### Finding 2: Server Does Not Set Response Content-Type Header
**Severity:** MEDIUM
**What:** The server's `writeProtoMessageResponse` and `marshalResponse` functions write response bytes but never set `Content-Type` on the response.
**Location:** httpgen/generator.go lines 860-893
**Fix:** Add `w.Header().Set("Content-Type", contentType)` before writing response bytes.

### Finding 3: Server Rejects Unknown Content-Types on Response
**Severity:** MEDIUM
**What:** The `marshalResponse` function returns an error for unrecognized Content-Types (`unsupported content type: %s`) instead of defaulting to JSON. But `bindDataBasedOnContentType` defaults to binary for unknown types. Inconsistent default behavior.
**Location:** httpgen/generator.go line 651 vs line 361
**Fix:** Both should default to JSON (matching protojson spec as source of truth) or both should default to binary. JSON default is recommended since protojson is the spec.

### Finding 4: OpenAPI Test Protos Not Shared
**Severity:** HIGH (for audit coverage)
**What:** OpenAPI uses completely independent test proto files that do not overlap with the httpgen/clientgen/tsclientgen test protos. This means there is no test that verifies all 4 generators produce consistent output for the same proto.
**Location:** internal/openapiv3/testdata/proto/ vs internal/httpgen/testdata/proto/
**Fix:** Create a unified exhaustive test proto or add the shared protos (via symlinks) to the OpenAPI test suite.

### Finding 5: Go Client Missing Unwrap Test Coverage
**Severity:** MEDIUM
**What:** The Go client generator (clientgen) does NOT have test protos for unwrap scenarios. The TS client has `complex_features.proto` with all unwrap variants, but clientgen has no equivalent.
**Location:** internal/clientgen/testdata/proto/ (only 3 symlinked protos, no unwrap)
**Fix:** Add unwrap test proto (via symlink or new file) to clientgen test suite.

### Finding 6: TS Client Uses JSON.stringify Instead of Protojson-Equivalent
**Severity:** LOW (likely correct but needs verification)
**What:** The TS client serializes request bodies with `JSON.stringify(req)` where `req` is a plain JavaScript object. This relies on the caller to pass correctly-structured data matching protojson conventions (camelCase keys, string enum names, string int64 values). Unlike the Go client which uses `protojson.Marshal`, there is no enforcement.
**Location:** tsclientgen/generator.go line 396
**Impact:** If a caller passes `{status: 1}` instead of `{status: "STATUS_PENDING"}`, the server will receive an integer enum value, which protojson accepts (per spec: "Parsers accept both enum names and integer values") but it's not the canonical form.

### Finding 7: Go Client Bool Query Param Zero Value Check
**Severity:** LOW
**What:** The Go client checks `if req.Active != false` for bool query params, which is equivalent to `if req.Active`. This means `false` is never sent as a query parameter. The TS client has the same behavior: `if (req.active)` only sends when true. The server's `convertStringToFieldValue` parses `strconv.ParseBool(value)`. This is consistent -- both clients omit `false` booleans.
**Impact:** Consistent behavior but worth documenting: there is no way to explicitly send `?active=false` via either client when the field is not set. This matches protojson's "omit default values" behavior.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Duplicated annotation parsing in each generator | Shared `internal/annotations` package | Phase 2 (2026-02-05) | All 4 generators now read annotations identically |
| Independent test protos per generator | Symlinked shared protos (httpgen is canonical) | Recent | clientgen and tsclientgen share httpgen protos; openapiv3 still independent |
| No cross-generator consistency validation | Not yet implemented | Phase 3 (this phase) | Will be established by this phase |

**Current protojson version:** v1.36.11 (google.golang.org/protobuf)
- Uses proto3 JSON canonical mapping
- Default values omitted from output (unless EmitDefaultValues option is set)
- 64-bit integers encoded as strings
- Enum values encoded as string names
- Field names use lowerCamelCase (or json_name override)

## Open Questions

Things that couldn't be fully resolved during research:

1. **Empty Response (204 No Content) handling**
   - What we know: No generator currently handles 204 empty responses. The server always writes response bytes. The Go client's `unmarshalResponse` returns nil for empty body, which is correct. The TS client calls `resp.json()` which would throw on empty body.
   - What's unclear: Should the server return 204 for RPCs that return empty messages? Currently it returns 200 with `{}`.
   - Recommendation: Audit both empty-message and truly-empty-body scenarios. The TS client's `resp.json()` on empty body is a potential bug.

2. **Repeated field query parameters**
   - What we know: The server's `bindQueryParams` handles repeated fields by iterating `query[param.QueryName]` (multiple values for the same key). But neither Go client nor TS client generates code for repeated query parameters.
   - What's unclear: Should clients support repeated query params? The proto definitions currently don't annotate repeated fields as query params.
   - Recommendation: If any test proto has repeated query params, verify all generators handle them. If not, consider adding test coverage.

3. **Optional field query parameter behavior**
   - What we know: The Go client checks zero values to decide whether to include query params. Proto3 optional fields have presence semantics (can distinguish "not set" from "set to zero"). The current generator uses `getZeroValue()` which doesn't distinguish optional from required.
   - What's unclear: Should an optional field set to zero value be included in query params?
   - Recommendation: This is an edge case worth auditing but likely deferred to a future enhancement.

4. **Unicode and special characters in path/query/header values**
   - What we know: Go client uses `url.PathEscape(fmt.Sprint(req.Field))` for path params. TS client uses `encodeURIComponent(String(req.field))`. Both should produce RFC 3986 percent-encoding.
   - What's unclear: Whether both produce identical encoding for all Unicode characters and URL-unsafe characters.
   - Recommendation: Add test cases with unicode and special characters to verify encoding consistency.

5. **Server default path format inconsistency**
   - What we know: When no HTTP config annotation exists, the server generates path `/{package}/{camel_to_snake(method)}` while the Go client generates `/{lowerFirst(method)}` and uses `BuildHTTPPath`. The OpenAPI generates `/{ServiceName}/{MethodName}`.
   - What's unclear: Whether these defaults align when no annotations are present.
   - Recommendation: Verify backward compatibility default paths are identical across all generators.

## Audit Approach Recommendation (Claude's Discretion)

Based on the research, here is the recommended approach:

### Audit Order
1. **Start with the Go server** -- it's the source of truth for "what actually happens at runtime"
2. **Audit Go client against server** -- same language, uses protojson, easiest to verify
3. **Audit TS client against server** -- cross-language, JSON-only, most likely to have discrepancies
4. **Audit OpenAPI against all three** -- documentation must match actual behavior

### Cross-Generator Comparison Structure
**Use a unified exhaustive test proto** rather than separate protos per generator. This proto should be the single source of truth for what all 4 generators must handle consistently.

The proto should exercise:
- All HTTP methods (GET, POST, PUT, PATCH, DELETE)
- Path parameters (single, multiple, nested)
- Query parameters (all scalar types: string, int32, int64, uint64, bool, float, double)
- Request/response bodies with all field types
- Service headers + method headers
- Enums (in messages and as standalone types)
- Optional fields
- Map fields (string keys, scalar values, message values)
- Repeated fields (scalar elements, message elements)
- All unwrap variants (map-value, root repeated, root map, combined)
- Nested messages
- Default/no annotations (backward compat)

### Consistency Test Infrastructure
**Recommendation: Build a lightweight cross-generator comparison test** in addition to golden files. This test would:
1. Run all 4 generators against the same proto
2. For each RPC, verify:
   - Go client and server agree on URL path construction
   - Go client and TS client produce the same query string for the same input
   - OpenAPI path matches server path
   - OpenAPI request schema matches what server accepts
   - OpenAPI response schema matches what server returns
3. This would be a Go test that parses generated Go code, TS code, and OpenAPI YAML/JSON

However, building automated comparison tooling may be overkill for a one-time audit. An alternative is to rely on golden files across all 4 generators for the same unified test proto, and manually verify semantic consistency during the initial audit. The golden files then prevent regression going forward.

**Recommendation: Start with golden files + manual audit, add automated comparison tests only if the audit reveals many issues that golden files alone can't catch.**

### Grouping Fixes
Organize fixes by concern area rather than by generator:
1. **Plan 1:** Unified test proto + golden file infrastructure alignment
2. **Plan 2:** Server correctness fixes (Content-Type header, default handling)
3. **Plan 3:** Go client consistency with server (query params, error handling, edge cases)
4. **Plan 4:** TS client consistency with server (query params, error handling, int64, enums)
5. **Plan 5:** OpenAPI schema accuracy (error schemas, type mapping, parameter docs)
6. **Plan 6:** Cross-generator golden file verification + final validation

## Sources

### Primary (HIGH confidence)
- Codebase analysis: All 4 generator source files read in full
- `internal/annotations/` package: All shared annotation parsing verified
- `go.mod`: Library versions confirmed
- Golden test infrastructure: All test files and protos examined
- protojson spec: https://protobuf.dev/programming-guides/json/ -- canonical JSON mapping rules verified

### Secondary (MEDIUM confidence)
- Proto3 JSON field mapping rules from official documentation
- Server response Content-Type behavior inferred from generated code reading

### Tertiary (LOW confidence)
- The exact behavior of `JSON.stringify` in TypeScript for all proto types needs runtime verification
- The exact equivalence of `url.PathEscape` vs `encodeURIComponent` for all Unicode ranges needs verification

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Directly read from go.mod and existing codebase
- Architecture: HIGH - Full codebase reading of all 4 generators
- Pitfalls: HIGH - Identified from actual code analysis, not hypothetical scenarios
- Pre-audit findings: HIGH - Specific line numbers and code paths identified
- Cross-generator recommendations: MEDIUM - Based on code analysis but untested

**Research date:** 2026-02-05
**Valid until:** N/A -- this research is specific to the current codebase state
