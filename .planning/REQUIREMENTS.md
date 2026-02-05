# Requirements: sebuf v1.0

**Defined:** 2026-02-05
**Core Value:** Proto definitions are the single source of truth — every generator must produce consistent, correct output that interoperates seamlessly.

## v1 Requirements

Requirements for v1.0 release. Each must work across all 4 generators (go-http, go-client, ts-client, openapiv3) unless noted.

### Foundation

- [ ] **FOUND-01**: Extract shared annotation parsing into `internal/annotations/` package (eliminate 1,289 lines of duplication across 4 generators)
- [ ] **FOUND-02**: Fix #105 — conditional net/url import in go-client (only when query params used)
- [ ] **FOUND-03**: Land PR #98 — cross-file unwrap resolution in same Go package
- [ ] **FOUND-04**: Audit serialization path — ensure protojson vs encoding/json consistency in HTTP handler generation
- [ ] **FOUND-05**: Verify #91 (root-level arrays) is fully covered by existing unwrap annotation, close GitHub issue
- [ ] **FOUND-06**: Close #94 (field name casing) on GitHub — document proto3 `json_name` as the existing solution

### JSON Mapping

- [ ] **JSON-01**: #87 Nullable primitives — per-field `nullable` annotation; generates pointer types in Go, `| null` union in TS, `nullable: true` in OpenAPI
- [ ] **JSON-02**: #88 int64/uint64 as string encoding — per-field `int64_encoding` annotation with NUMBER/STRING options
- [ ] **JSON-03**: #89 Enum string encoding with custom values — per-enum `enum_encoding` and per-value `enum_value` annotations
- [ ] **JSON-04**: #90 Oneof as discriminated union — per-oneof `oneof_discriminator` and `oneof_flatten` annotations with field collision detection at generation time
- [ ] **JSON-05**: #92 Multiple timestamp formats — per-field `timestamp_format` annotation (RFC3339, UNIX_SECONDS, UNIX_MILLIS, DATE)
- [ ] **JSON-06**: #93 Empty object handling — per-field `omit_empty` and `empty_behavior` annotations (PRESERVE, NULL, OMIT)
- [ ] **JSON-07**: #95 Bytes encoding options — per-field `bytes_encoding` annotation (BASE64, BASE64_RAW, BASE64URL, BASE64URL_RAW, HEX)
- [ ] **JSON-08**: #96 Nested message flattening — per-field `flatten` and `flatten_prefix` annotations with collision detection at generation time

### Polish

- [ ] **POL-01**: Comprehensive README review and improvement
- [ ] **POL-02**: Add examples for all JSON mapping features (proto definitions + expected JSON output)
- [ ] **POL-03**: Add multi-auth patterns example (#50)
- [ ] **POL-04**: Expand test coverage — golden file tests for every annotation across all generators
- [ ] **POL-05**: Review and improve inline documentation across all generators
- [ ] **POL-06**: End-to-end consistency validation — verify proto definitions produce matching output across all 4 generators

## v2 Requirements

Deferred to v2.0 milestone. Tracked but not in current roadmap.

### Multi-Language Clients

- **LANG-01**: Python HTTP client generator (protoc-gen-py-client)
- **LANG-02**: Rust HTTP client generator (protoc-gen-rs-client)
- **LANG-03**: Swift HTTP client generator (protoc-gen-swift-client)
- **LANG-04**: Kotlin HTTP client generator (protoc-gen-kt-client)
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
| FOUND-01 | — | Pending |
| FOUND-02 | — | Pending |
| FOUND-03 | — | Pending |
| FOUND-04 | — | Pending |
| FOUND-05 | — | Pending |
| FOUND-06 | — | Pending |
| JSON-01 | — | Pending |
| JSON-02 | — | Pending |
| JSON-03 | — | Pending |
| JSON-04 | — | Pending |
| JSON-05 | — | Pending |
| JSON-06 | — | Pending |
| JSON-07 | — | Pending |
| JSON-08 | — | Pending |
| POL-01 | — | Pending |
| POL-02 | — | Pending |
| POL-03 | — | Pending |
| POL-04 | — | Pending |
| POL-05 | — | Pending |
| POL-06 | — | Pending |

**Coverage:**
- v1 requirements: 20 total
- Mapped to phases: 0
- Unmapped: 20 ⚠️

---
*Requirements defined: 2026-02-05*
*Last updated: 2026-02-05 after initial definition*
