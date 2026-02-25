# Roadmap: sebuf

## Milestones

- [x] **v1.0 JSON Mapping & Polish** - Phases 1-11 (in progress)
- [ ] **v1.1 KrakenD Config Generator** - Phases 12-14 (planned)

## Phases

<details>
<summary>v1.0 JSON Mapping & Polish (Phases 1-11)</summary>

### Phase 1: Foundation - Quick Wins
**Goal**: Existing bugs are fixed, pending work is landed, and resolved issues are closed so the codebase is clean before structural changes
**Depends on**: Nothing (first phase)
**Requirements**: FOUND-02, FOUND-03, FOUND-05, FOUND-06
**Success Criteria** (what must be TRUE):
  1. Running `go build ./cmd/protoc-gen-go-client/` produces a binary that does not emit unused `net/url` imports when generating clients without query parameters
  2. Protobuf messages referencing unwrap-annotated types from other `.proto` files in the same Go package resolve correctly at generation time
  3. GitHub issue #91 (root-level arrays) is closed with a comment documenting how existing unwrap annotation covers the use case
  4. GitHub issue #94 (field name casing) is closed with a comment documenting proto3 `json_name` as the existing solution
**Plans**: 2 plans

Plans:
- [x] 01-01-PLAN.md -- Fix conditional net/url import (#105) and land cross-file unwrap PR (#98)
- [x] 01-02-PLAN.md -- Verify and close resolved GitHub issues #91 and #94

### Phase 2: Foundation - Shared Annotations
**Goal**: All generators consume annotation metadata through a single shared package, eliminating duplication and ensuring consistency for the 8 new annotations coming next
**Depends on**: Phase 1
**Requirements**: FOUND-01, FOUND-04
**Success Criteria** (what must be TRUE):
  1. A new `internal/annotations` package exists that all 4 generators import for annotation parsing (HTTPConfig, QueryParam, UnwrapInfo, HeaderConfig types)
  2. All existing golden file tests pass without changes to expected output (zero behavior change)
  3. The duplicated annotation parsing code (~1,289 lines across httpgen, clientgen, tsclientgen, openapiv3) is removed and replaced with shared package calls
  4. The HTTP handler generator uses consistent protojson-based serialization (no accidental encoding/json usage for proto messages)
  5. Cross-file annotation resolution never silently suppresses errors
**Plans**: 4 plans

Plans:
- [x] 02-01-PLAN.md -- Create internal/annotations package with canonical types, functions, and tests
- [x] 02-02-PLAN.md -- Migrate httpgen to shared annotations, delete old annotation code
- [x] 02-03-PLAN.md -- Migrate clientgen and tsclientgen to shared annotations
- [x] 02-04-PLAN.md -- Migrate openapiv3, fix error suppression, final verification

### Phase 3: Existing Client Review
**Goal**: The existing Go HTTP client and TypeScript HTTP client are solid, consistent with each other and with the server, and ready to serve as the reference implementations that new language clients and JSON mapping features build upon
**Depends on**: Phase 2 (shared annotations extracted, duplication eliminated)
**Requirements**: FOUND-07, FOUND-08
**Success Criteria** (what must be TRUE):
  1. For every RPC in the exhaustive test proto, the Go client serializes requests and deserializes responses identically to the Go HTTP server (byte-level JSON comparison of the same proto message through both paths)
  2. For every RPC in the exhaustive test proto, the TypeScript client produces the same JSON request bodies and expects the same JSON response shapes as the Go server
  3. Error handling is consistent: both clients surface ValidationError and ApiError with the same HTTP status codes, the same error body structure, and the same field-level violation format
  4. Header handling is consistent: both clients send service-level and method-level headers identically, including the same default values, same required/optional semantics, and same header name casing
  5. All existing golden file tests pass, and any fixes made during the review are captured as new golden file test cases to prevent regression
**Plans**: 6 plans

Plans:
- [x] 03-01-PLAN.md -- Expand exhaustive test proto and align OpenAPI test infrastructure with shared symlinks
- [x] 03-02-PLAN.md -- Fix server Content-Type response headers and marshalResponse default behavior
- [x] 03-03-PLAN.md -- Audit and fix Go client consistency with server (unwrap coverage, query params, errors, headers)
- [x] 03-04-PLAN.md -- Audit and fix TypeScript client consistency (int64 as string, query params, errors, headers)
- [x] 03-05-PLAN.md -- Fix OpenAPI error schemas and type mapping for protojson consistency
- [x] 03-06-PLAN.md -- Cross-generator golden file verification and final semantic comparison

### Phase 4: JSON - Primitive Encoding
**Goal**: Developers can control how int64/uint64 fields and enum fields are encoded in JSON across all generators
**Depends on**: Phase 3 (existing clients verified solid)
**Requirements**: JSON-02, JSON-03
**Success Criteria** (what must be TRUE):
  1. A proto field annotated with `int64_encoding = STRING` serializes int64/uint64 values as JSON strings in go-http, go-client, ts-client, and documents as `type: string` in OpenAPI
  2. A proto field annotated with `int64_encoding = NUMBER` serializes int64/uint64 values as JSON numbers in all generators (with a generation-time warning about JavaScript precision loss for values exceeding 2^53)
  3. A proto enum annotated with `enum_encoding = STRING` serializes enum values as their proto names in JSON across all generators
  4. Per-value `enum_value` annotations map proto enum names to custom JSON strings (e.g., `STATUS_ACTIVE` serializes as `"active"`) across all generators
  5. OpenAPI schemas for int64/enum fields accurately reflect the configured encoding
  6. A cross-generator consistency test confirms that go-http, go-client, ts-client, and openapiv3 produce semantically identical JSON for every int64_encoding and enum_encoding combination
**Plans**: 5 plans

Plans:
- [x] 04-01-PLAN.md -- Define int64_encoding and enum_encoding annotations in proto and shared annotations package
- [x] 04-02-PLAN.md -- Implement int64 encoding in Go generators (go-http and go-client)
- [x] 04-03-PLAN.md -- Implement int64 encoding in ts-client and openapiv3 generators
- [x] 04-04-PLAN.md -- Implement enum encoding across all 4 generators
- [x] 04-05-PLAN.md -- Cross-generator consistency validation for primitive encoding

### Phase 5: JSON - Nullable & Empty
**Goal**: Developers can express null vs absent vs default semantics for primitive fields and control empty object serialization behavior
**Depends on**: Phase 4 (annotations infrastructure proven), Phase 2 (shared package)
**Requirements**: JSON-01, JSON-06
**Success Criteria** (what must be TRUE):
  1. A proto field annotated with `nullable = true` generates pointer types in Go (`*string`, `*int32`), union types in TypeScript (`string | null`), and `nullable: true` in OpenAPI schemas
  2. Three distinct states are representable per nullable field: absent (key omitted from JSON), null (key present with `null` value), and default value (key present with value)
  3. A proto message field annotated with `empty_behavior = PRESERVE` serializes empty messages as `{}`, `empty_behavior = NULL` as `null`, and `empty_behavior = OMIT` omits the key entirely
  4. All nullable and empty-behavior semantics are consistent across go-http, go-client, ts-client, and OpenAPI generators
  5. A cross-generator consistency test confirms that the same nullable/empty proto definitions produce semantically identical JSON across all generators (server serializes what clients expect, OpenAPI documents what both produce)
**Plans**: 4 plans

Plans:
- [x] 05-01-PLAN.md -- Define nullable and empty_behavior annotations in proto and shared annotations package
- [x] 05-02-PLAN.md -- Implement nullable primitives across all 4 generators
- [x] 05-03-PLAN.md -- Implement empty object handling across all 4 generators
- [x] 05-04-PLAN.md -- Cross-generator consistency validation for nullable and empty semantics

### Phase 6: JSON - Data Encoding
**Goal**: Developers can choose timestamp formats and bytes encoding options for their API's JSON representation
**Depends on**: Phase 2 (shared annotations package)
**Requirements**: JSON-05, JSON-07
**Success Criteria** (what must be TRUE):
  1. A `google.protobuf.Timestamp` field annotated with `timestamp_format = UNIX_SECONDS` serializes as a numeric Unix timestamp (not RFC 3339 string) across all generators
  2. All four timestamp formats work correctly: RFC3339 (default), UNIX_SECONDS, UNIX_MILLIS, DATE (date-only string)
  3. A `bytes` field annotated with `bytes_encoding = HEX` serializes as a hex string instead of base64 across all generators
  4. All five bytes encoding options work correctly: BASE64 (default), BASE64_RAW, BASE64URL, BASE64URL_RAW, HEX
  5. OpenAPI schemas document the actual encoding format used (e.g., `format: unix-timestamp` or `format: hex`)
  6. A cross-generator consistency test confirms that go-http, go-client, ts-client, and openapiv3 agree on serialization format for every timestamp_format and bytes_encoding combination
**Plans**: 4 plans

Plans:
- [x] 06-01-PLAN.md -- Define timestamp_format and bytes_encoding annotations in proto and shared annotations package
- [x] 06-02-PLAN.md -- Implement timestamp format options across all 4 generators
- [x] 06-03-PLAN.md -- Implement bytes encoding options across all 4 generators
- [x] 06-04-PLAN.md -- Cross-generator consistency validation for data encoding

### Phase 7: JSON - Structural Transforms
**Goal**: Developers can represent oneof fields as discriminated unions and flatten nested messages in their API's JSON output
**Depends on**: Phase 2 (shared annotations package)
**Requirements**: JSON-04, JSON-08
**Success Criteria** (what must be TRUE):
  1. A proto oneof annotated with `oneof_discriminator = "type"` and `oneof_flatten = true` serializes as a flat JSON object with a discriminator field (e.g., `{"type": "text", "body": "hello"}`) across all generators
  2. Field name collisions between oneof variants and the discriminator field are detected and reported as generation-time errors (not silent runtime failures)
  3. A nested message field annotated with `flatten = true` promotes its child fields to the parent level in JSON (e.g., `address.street` becomes `street` in the parent JSON object)
  4. `flatten_prefix` annotation prepends a prefix to flattened field names to avoid collisions (e.g., `flatten_prefix = "billing_"` produces `billing_street`)
  5. OpenAPI schemas accurately represent discriminated unions using the `discriminator` keyword and flattened structures using `allOf`
  6. A cross-generator consistency test confirms that go-http, go-client, ts-client, and openapiv3 produce semantically identical JSON structure for every oneof and flatten combination
**Plans**: 4 plans

Plans:
- [x] 07-01-PLAN.md -- Define oneof_config, oneof_value, flatten, and flatten_prefix annotations in proto and shared annotations package
- [x] 07-02-PLAN.md -- Implement oneof discriminated union across all 4 generators
- [x] 07-03-PLAN.md -- Implement nested message flattening across all 4 generators
- [x] 07-04-PLAN.md -- Cross-generator consistency validation for structural transforms

### Phase 8: Language - Swift Client
**Goal**: Swift developers can generate a type-safe HTTP client from proto definitions that supports all sebuf annotations including JSON mapping features
**Depends on**: Phase 7 (all JSON mapping features complete)
**Requirements**: LANG-01
**Plans**: TBD

Plans:
- [ ] 08-01: Scaffold protoc-gen-swift-client plugin structure and type mapping
- [ ] 08-02: Implement Swift service client generation with HTTP methods and headers
- [ ] 08-03: Implement Swift JSON mapping annotation support and golden file tests
- [ ] 08-04: Cross-generator consistency validation against Go client baseline

### Phase 9: Language - Kotlin Client
**Goal**: Kotlin developers can generate a type-safe HTTP client from proto definitions that supports all sebuf annotations including JSON mapping features
**Depends on**: Phase 7 (all JSON mapping features complete)
**Requirements**: LANG-02
**Plans**: TBD

Plans:
- [ ] 09-01: Scaffold protoc-gen-kt-client plugin structure and type mapping
- [ ] 09-02: Implement Kotlin service client generation with HTTP methods and headers
- [ ] 09-03: Implement Kotlin JSON mapping annotation support and golden file tests
- [ ] 09-04: Cross-generator consistency validation against Go client baseline

### Phase 10: Language - Python Client
**Goal**: Python developers can generate a type-safe HTTP client from proto definitions that supports all sebuf annotations including JSON mapping features
**Depends on**: Phase 7 (all JSON mapping features complete)
**Requirements**: LANG-03
**Plans**: TBD

Plans:
- [ ] 10-01: Scaffold protoc-gen-py-client plugin structure and type mapping
- [ ] 10-02: Implement Python service client generation with HTTP methods and headers
- [ ] 10-03: Implement Python JSON mapping annotation support and golden file tests
- [ ] 10-04: Cross-generator consistency validation against Go client baseline

### Phase 11: Polish & Release
**Goal**: sebuf v1.0 is documented, tested, and passes a formal consistency audit confirming zero inconsistencies between all 7 generators, with zero backward compatibility breaks
**Depends on**: Phase 10 (all features and language clients complete)
**Requirements**: POL-01, POL-02, POL-03, POL-04, POL-05, POL-06
**Plans**: TBD

Plans:
- [ ] 11-01: README overhaul and inline documentation review
- [ ] 11-02: JSON mapping examples and multi-auth patterns example
- [ ] 11-03: Expand golden file test coverage across all generators
- [ ] 11-04: Formal cross-generator consistency audit (all 7 generators)
- [ ] 11-05: Backward compatibility verification suite

</details>

## v1.1 KrakenD Config Generator

**Milestone Goal:** Add a protoc-gen-krakend plugin that generates KrakenD API gateway configuration from proto service definitions, eliminating manual config drift between services and the gateway.

**Phase Numbering:**
- Integer phases (12, 13, 14): Planned milestone work
- Decimal phases (12.1, 12.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 12: Annotations and Core Endpoint Generation** - Proto annotation package, plugin scaffold, endpoint/backend generation, auto-derived forwarding, and route validation (completed 2026-02-25)
- [ ] **Phase 13: Gateway Features** - Rate limiting, JWT authentication, circuit breaker, caching, concurrent calls, and namespace validation
- [ ] **Phase 14: Documentation and Examples** - Example proto with all annotations and Flexible Config integration guide

## Phase Details

### Phase 12: Annotations and Core Endpoint Generation
**Goal**: Users can run protoc-gen-krakend and get correct, minimal KrakenD endpoint fragments with auto-derived header and query string forwarding from their existing proto service definitions
**Depends on**: Nothing (first phase of v1.1 milestone; reuses existing sebuf.http annotations from v1.0)
**Requirements**: ANNO-01, ANNO-02, ANNO-03, ANNO-04, CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, FWD-01, FWD-02, FWD-03, VALD-01, VALD-02, TEST-01, TEST-03, TEST-04
**Success Criteria** (what must be TRUE):
  1. Running `protoc --krakend_out=. service.proto` on a service annotated with `gateway_config` produces a `{ServiceName}.krakend.json` file containing a valid JSON array of KrakenD endpoint objects with correct HTTP methods and paths derived from `sebuf.http.config` annotations, and backend host from the `gateway_config` annotation
  2. KrakenD endpoint objects include `input_headers` auto-populated from `sebuf.http.service_headers` and `sebuf.http.method_headers`, and `input_query_strings` auto-populated from `sebuf.http.query` annotations -- never empty arrays, never wildcards
  3. Service-level `gateway_config` annotation sets defaults (host, timeout) for all endpoints in a service, and method-level `endpoint_config` annotation overrides those defaults for individual RPCs
  4. Generation fails with a clear error message when two RPCs produce identical (path, method) tuples, or when static and parameterized routes conflict at the same path level (e.g., `/users/search` vs `/users/{id}`)
  5. Golden file tests cover endpoint routing, backend mapping, timeouts, auto-derived forwarding, and all validation error scenarios
**Plans**: 4 plans

Plans:
- [ ] 12-01-PLAN.md -- Proto annotation package (sebuf.krakend), generated Go code, plugin scaffold, KrakenD JSON types
- [ ] 12-02-PLAN.md -- Core endpoint/backend generation with host and timeout config, annotation override semantics
- [ ] 12-03-PLAN.md -- Auto-derived header and query string forwarding from sebuf.http annotations
- [ ] 12-04-PLAN.md -- Route validation (duplicate endpoints, static vs param conflicts) and golden file test suite

### Phase 13: Gateway Features
**Goal**: Users can annotate their proto services with rate limiting, JWT authentication, circuit breaker, caching, and concurrency settings that generate correct KrakenD extra_config entries
**Depends on**: Phase 12 (endpoint skeleton must exist for extra_config to attach to)
**Requirements**: RLIM-01, RLIM-02, RLIM-03, AUTH-01, AUTH-02, AUTH-03, RESL-01, RESL-02, RESL-03, RESL-04, VALD-03, TEST-02
**Success Criteria** (what must be TRUE):
  1. A service or method annotated with rate limit settings produces endpoint-level `extra_config` with `"qos/ratelimit/router"` containing max_rate, capacity, and strategy, and/or backend-level `extra_config` with `"qos/ratelimit/proxy"` -- with method-level always overriding service-level
  2. A service annotated with JWT validation settings produces endpoint-level `extra_config` with `"auth/validator"` containing JWK URL, algorithm, issuer, audience, and optional claim-to-header propagation rules
  3. A service or method annotated with circuit breaker settings produces backend-level `extra_config` with `"qos/circuit-breaker"` containing interval, timeout, and max_errors -- with method-level always overriding service-level
  4. Backend caching (`"qos/http-cache"`) and concurrent calls are configurable per endpoint, with service-level defaults and method-level overrides
  5. All extra_config namespace strings (e.g., `"qos/ratelimit/router"`, `"auth/validator"`) are Go constants validated against a known allowlist, and golden file tests cover every gateway feature combination
**Plans**: TBD

Plans:
- [ ] 13-01: Rate limiting (endpoint and backend) with service/method override semantics
- [ ] 13-02: JWT authentication and claim propagation
- [ ] 13-03: Circuit breaker, caching, concurrent calls, and namespace validation

### Phase 14: Documentation and Examples
**Goal**: Users have a working example and a clear guide showing how to use protoc-gen-krakend annotations and compose per-service fragments into a complete KrakenD configuration
**Depends on**: Phase 13 (all features must exist before documenting them)
**Requirements**: DOCS-01, DOCS-02
**Success Criteria** (what must be TRUE):
  1. An example proto file in `examples/` demonstrates every KrakenD annotation (gateway_config, endpoint_config, rate limiting, JWT, circuit breaker, caching) with inline comments explaining each setting
  2. A Flexible Config integration guide shows how to use KrakenD's `FC_ENABLE=1` with `{{ include }}` directives to compose per-service `.krakend.json` fragments into a complete `krakend.json`, including the comma-handling pattern for multi-service includes
**Plans**: TBD

Plans:
- [ ] 14-01: Example proto and Flexible Config integration guide

## Progress

**Execution Order:**
Phases execute in numeric order: 12 -> 13 -> 14

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 12. Annotations and Core Endpoint Generation | 4/4 | Complete   | 2026-02-25 |
| 13. Gateway Features | 0/3 | Not started | - |
| 14. Documentation and Examples | 0/1 | Not started | - |
