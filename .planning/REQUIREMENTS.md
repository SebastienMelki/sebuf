# Requirements: sebuf v1.0

**Defined:** 2026-02-05
**Core Value:** Proto definitions are the single source of truth -- every generator must produce consistent, correct output that interoperates seamlessly.

## v1 Requirements

Requirements for v1.0 release. Each must work across all generators (go-http, go-client, ts-client, swift-client, kt-client, py-client, openapiv3) unless noted.

### Foundation

- [x] **FOUND-01**: Extract shared annotation parsing into `internal/annotations/` package (eliminate 1,289 lines of duplication across 4 generators)
- [x] **FOUND-02**: Fix #105 -- conditional net/url import in go-client (only when query params used)
- [x] **FOUND-03**: Land PR #98 -- cross-file unwrap resolution in same Go package
- [x] **FOUND-04**: Audit serialization path -- ensure protojson vs encoding/json consistency in HTTP handler generation
- [x] **FOUND-05**: Verify #91 (root-level arrays) is fully covered by existing unwrap annotation, close GitHub issue
- [x] **FOUND-06**: Close #94 (field name casing) on GitHub -- document proto3 `json_name` as the existing solution
- [x] **FOUND-07**: Review and polish existing Go HTTP client (protoc-gen-go-client) -- audit serialization consistency with server, error handling, header handling, and edge cases; fix any inconsistencies found
- [x] **FOUND-08**: Review and polish existing TypeScript HTTP client (protoc-gen-ts-client) -- audit cross-language consistency with Go client and server, error handling, header handling, and edge cases; fix any inconsistencies found

### JSON Mapping

- [ ] **JSON-01**: #87 Nullable primitives -- per-field `nullable` annotation; generates pointer types in Go, `| null` union in TS, `nullable: true` in OpenAPI
- [x] **JSON-02**: #88 int64/uint64 as string encoding -- per-field `int64_encoding` annotation with NUMBER/STRING options
- [x] **JSON-03**: #89 Enum string encoding with custom values -- per-enum `enum_encoding` and per-value `enum_value` annotations
- [ ] **JSON-04**: #90 Oneof as discriminated union -- per-oneof `oneof_discriminator` and `oneof_flatten` annotations with field collision detection at generation time
- [ ] **JSON-05**: #92 Multiple timestamp formats -- per-field `timestamp_format` annotation (RFC3339, UNIX_SECONDS, UNIX_MILLIS, DATE)
- [ ] **JSON-06**: #93 Empty object handling -- per-field `omit_empty` and `empty_behavior` annotations (PRESERVE, NULL, OMIT)
- [ ] **JSON-07**: #95 Bytes encoding options -- per-field `bytes_encoding` annotation (BASE64, BASE64_RAW, BASE64URL, BASE64URL_RAW, HEX)
- [ ] **JSON-08**: #96 Nested message flattening -- per-field `flatten` and `flatten_prefix` annotations with collision detection at generation time

### Language Clients

- [ ] **LANG-01**: Swift HTTP client generator (protoc-gen-swift-client) -- idiomatic Swift using URLSession, Codable structs
- [ ] **LANG-02**: Kotlin HTTP client generator (protoc-gen-kt-client) -- idiomatic Kotlin using OkHttp/Ktor, data classes
- [ ] **LANG-03**: Python HTTP client generator (protoc-gen-py-client) -- idiomatic Python using httpx, dataclasses/Pydantic

### Polish

- [ ] **POL-01**: Comprehensive README review and improvement
- [ ] **POL-02**: Add examples for all JSON mapping features (proto definitions + expected JSON output)
- [ ] **POL-03**: Add multi-auth patterns example (#50)
- [ ] **POL-04**: Expand test coverage -- golden file tests for every annotation across all generators
- [ ] **POL-05**: Review and improve inline documentation across all generators
- [ ] **POL-06**: End-to-end consistency validation -- verify proto definitions produce matching output across all generators (go-http, go-client, ts-client, swift-client, kt-client, py-client, openapiv3)

## v2 Requirements

Deferred to v2.0 milestone. Tracked but not in current roadmap.

### Multi-Language Clients

- **LANG-04**: Rust HTTP client generator (protoc-gen-rs-client)
- **LANG-05**: Java HTTP client generator (protoc-gen-java-client)
- **LANG-06**: C# HTTP client generator (protoc-gen-csharp-client)
- **LANG-07**: Ruby HTTP client generator (protoc-gen-rb-client)
- **LANG-08**: Dart HTTP client generator (protoc-gen-dart-client)

### Additional Features

- **FEAT-01**: #101 Support deprecated option in protoc-gen-go-client
- **FEAT-02**: #102 Support binary response types in protoc-gen-go-http

## Out of Scope

| Feature | Reason |
|---------|--------|
| gRPC support | sebuf targets HTTP APIs specifically, not gRPC |
| GraphQL generation | Different API paradigm, out of scope |
| Database/ORM integration | sebuf generates HTTP layer only |
| Runtime framework (router, middleware) | Generates code for standard library |
| Streaming/WebSocket support | Different transport paradigm, defer to future |
| #94 Field name casing annotations | Proto3's built-in `json_name` already handles per-field override; adding sebuf-specific option creates confusion |
| Multi-language HTTP servers (v3.0) | Clients are higher value; servers only needed in Go currently |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FOUND-01 | Phase 2 | Complete |
| FOUND-02 | Phase 1 | Complete |
| FOUND-03 | Phase 1 | Complete |
| FOUND-04 | Phase 2 | Complete |
| FOUND-05 | Phase 1 | Complete |
| FOUND-06 | Phase 1 | Complete |
| FOUND-07 | Phase 3 | Complete |
| FOUND-08 | Phase 3 | Complete |
| JSON-01 | Phase 5 | Pending |
| JSON-02 | Phase 4 | Complete |
| JSON-03 | Phase 4 | Complete |
| JSON-04 | Phase 7 | Pending |
| JSON-05 | Phase 6 | Pending |
| JSON-06 | Phase 5 | Pending |
| JSON-07 | Phase 6 | Pending |
| JSON-08 | Phase 7 | Pending |
| LANG-01 | Phase 8 | Pending |
| LANG-02 | Phase 9 | Pending |
| LANG-03 | Phase 10 | Pending |
| POL-01 | Phase 11 | Pending |
| POL-02 | Phase 11 | Pending |
| POL-03 | Phase 11 | Pending |
| POL-04 | Phase 11 | Pending |
| POL-05 | Phase 11 | Pending |
| POL-06 | Phase 11 | Pending |

**Coverage:**
- v1 requirements: 25 total
- Mapped to phases: 25
- Unmapped: 0

---
*Requirements defined: 2026-02-05*
*Last updated: 2026-02-06 after Phase 4 completion (JSON-02, JSON-03 marked Complete)*
