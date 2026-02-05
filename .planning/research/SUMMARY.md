# Project Research Summary

**Project:** sebuf v1.0 - JSON Mapping Features & Multi-Language Client Generators
**Domain:** Protobuf Code Generation Toolkit
**Researched:** 2026-02-05
**Confidence:** HIGH

## Executive Summary

sebuf is a mature Go protobuf toolkit with 4 generators (go-http, go-client, ts-client, openapiv3) targeting HTTP API development. The v1.0 milestone targets 10 JSON mapping annotations to give developers fine-grained control over JSON serialization beyond proto3's canonical mapping. The research reveals this is a strong differentiator: competing tools (grpc-gateway, connect-go, twirp) use global runtime options, while sebuf's annotation-driven approach provides per-field granularity baked into the schema. The codebase is architecturally sound but has critical technical debt: 1,289 lines of duplicated annotation parsing code across 4 generators that must be consolidated before adding 10 new annotations.

The recommended approach is annotation-driven code generation using custom protobuf field options. Each generator reads the same annotations and emits language-specific code at generation time, not runtime configuration. This maximizes cross-generator consistency and zero-runtime overhead. The core technologies are already in place: protogen for code generation, protojson as the baseline serializer (with custom code layered on top for features it doesn't support), and libopenapi for schema generation.

The key risk is cross-generator annotation inconsistency. With 4 existing generators and 8 planned for v2.0, any annotation implementation must work identically across all of them. The mitigation is a shared annotation parsing package (`internal/annotations`) and cross-generator conformance tests before implementing any new features. Secondary risks include int64 JSON encoding pitfalls (JavaScript precision loss), proto3 field presence semantics (null vs absent vs default), and golden file test explosion.

## Key Findings

### Recommended Stack

sebuf should continue using its existing Go-based protoc plugin architecture. All new work builds on `google.golang.org/protobuf/compiler/protogen` for code generation. JSON mapping features layer custom behavior on top of `protojson` rather than replacing it, because protojson correctly handles Well-Known Types (Timestamp, Duration, Struct, etc.) that are error-prone to reimplement.

**Core technologies:**
- **protogen v1.36.11** (already in use): Foundation for all protoc plugins — annotation-driven code generation pattern
- **protojson v1.36.11** (already in use): Baseline Go JSON serializer — layer custom code on top for features it doesn't support (nullable primitives, custom timestamp formats, etc.)
- **libopenapi v0.33.0** (already in use): OpenAPI 3.1 document construction — no version change needed, natively supports all planned features
- **Custom annotations in proto/sebuf/http/** (extend existing): Define 10 new field/message options for JSON mapping — each generator reads these and emits appropriate code

For the 8 new language client generators planned for v2.0, each uses the target language's standard HTTP library (no external dependencies where possible): httpx (Python), reqwest (Rust), URLSession (Swift), OkHttp (Kotlin), java.net.http (Java), System.Net.Http (C#), Faraday (Ruby), package:http (Dart). The generators produce standalone, dependency-free client code with native language types (dataclasses, Codable structs, records) rather than requiring protobuf runtime libraries.

### Expected Features

**Must have (table stakes — competitors provide these, sebuf must too):**
- **#87 Nullable primitives**: JSON null vs absent vs default distinction — REST APIs require this for PATCH semantics, proto3 optional keyword alone insufficient
- **#88 int64 as string/number**: Proto3 spec mandates string encoding, but many REST APIs use numbers — per-field control essential for interop
- **#89 Enum string encoding**: Switch between enum names as strings vs numbers, plus custom JSON values (STATUS_ACTIVE -> "active") — REST conventions demand lowercase strings

**Should have (competitive differentiators — sebuf's unique value):**
- **#90 Oneof discriminated union**: Flatten oneofs with explicit type discriminator ({"type": "text", "body": "..."}) — REST API standard pattern, no protobuf tool does this
- **#92 Timestamp formats**: RFC 3339 / Unix seconds / Unix millis / date-only — interop with existing APIs that don't use proto's RFC 3339 default
- **#93 Empty object handling**: Preserve {} vs emit null vs omit for empty messages — PATCH semantics require this control
- **#95 Bytes encoding**: base64 / base64url / hex encoding options — crypto/blockchain APIs universally use hex, JWT contexts use base64url

**Defer (v2+ — either low value or high complexity):**
- **#91 Root-level arrays**: ALREADY SOLVED by existing unwrap annotation — close issue, document existing functionality
- **#94 Field name casing**: ANTI-FEATURE, overlaps with proto3's built-in json_name option — document existing mechanism instead of adding duplicate
- **#96 Nested message flattening**: High complexity, niche use case (legacy API interop) — defer until user demand proven

### Architecture Approach

The codebase follows a clean protoc plugin architecture with separated concerns. Each of the 4 generators is a standalone protoc plugin with its own internal package. However, annotation parsing is currently duplicated across all 4 packages (1,289 lines total), creating maintenance burden and consistency risk. The v1.0 work requires extracting this into a shared `internal/annotations` package before adding new features.

**Major components:**
1. **internal/annotations (NEW)** — Single source of truth for annotation extraction from protobuf descriptors, consumed by all generators
2. **internal/httpgen** — Go HTTP handler generation including automatic validation, header middleware, and custom JSON marshaling
3. **internal/clientgen** — Go HTTP client generation with functional options pattern
4. **internal/tsclientgen** — TypeScript HTTP client generation with typed interfaces and error handling
5. **internal/openapiv3** — OpenAPI 3.1 specification generation, one file per service

The build order is critical: (1) extract shared annotation package from existing duplicated code, (2) add JSON mapping features using the shared package, (3) build new language generators that import the shared package from day one. Each phase validates the work of the previous phase through golden file tests.

### Critical Pitfalls

1. **Cross-generator annotation inconsistency** — Already happening: unwrap annotation parsed 4 different ways across 4 generators. Fix: extract shared annotation parsing package (`internal/annotations`) before adding 10 new annotations. This is the single most impactful structural change.

2. **int64/uint64 string-vs-number JSON encoding** — Proto3 spec mandates string encoding (JavaScript precision loss beyond 2^53), but feature #88 allows opt-in number encoding. Every generator must agree on the encoding per field. Default to string (proto3 canonical), make number opt-in with validation warnings, test boundary values (MAX_SAFE_INTEGER, int64 max).

3. **Proto3 field presence: null vs absent vs default** — Proto3 optional keyword gives presence tracking but protojson doesn't emit JSON null for unset fields. Feature #87 requires explicit three-state semantics: Go uses *string (nil=absent), TypeScript uses string | null | undefined, JSON omitted key vs null vs empty. Critical for PATCH endpoints.

4. **Cross-file type resolution** — When a message in file A references a message in file B with annotations, the generator processing file A must see file B's annotations. Currently has silent error suppression (CONCERNS.md documents this). Fix before v1.0: never suppress annotation resolution errors, add multi-file test fixtures, build annotation collection as preprocessing pass.

5. **Golden file test explosion** — Current: ~48 golden files (4 generators x ~4 fixtures x ~3 output files). After v1.0: ~168 files. After v2.0 with 8 new languages: ~504 files. Mitigation: reorganize by feature not generator, add diff summary tooling, separate structural from content tests, use parameterized tests.

## Implications for Roadmap

Based on research, v1.0 should follow a two-phase structure: Foundation (extract shared code) then Features (implement annotations). The architecture research shows attempting to add 10 annotations on top of duplicated parsing code will create 40+ inconsistencies (10 annotations x 4 generators). The pitfalls research confirms cross-generator inconsistency is already happening and will only worsen.

### Phase 1: Foundation Refactoring
**Rationale:** Extract shared annotation parsing package before adding new annotations — prevents 10x duplication (10 new annotations times 4 generators). The dependency analysis shows all JSON features are independent except #87 (nullable) blocks #93 (empty objects), so there's no inherent ordering constraint beyond "foundation first."

**Delivers:**
- New `internal/annotations` package with canonical types (HTTPConfig, QueryParam, UnwrapInfo)
- All 4 existing generators refactored to import shared code
- Zero behavior change (verified by golden files)
- 1,289 lines of duplicated code reduced to ~300 shared lines

**Addresses:**
- Pitfall #1 (cross-generator inconsistency)
- Pitfall #4 (cross-file resolution) — fix silent error suppression
- Pitfall #11 (recursive messages) — add cycle detection to all walkers

**Avoids:**
- Implementing 10 annotations with 4x duplication each
- Inconsistent annotation implementations across generators
- PR #98 (cross-file unwrap) merge conflicts

### Phase 2: Table Stakes Features (v1.0 MVP)
**Rationale:** Implement the 3 must-have features first. These are what competing tools provide, so sebuf needs them for credibility. All are independent (no dependency chain) and low-to-medium complexity.

**Delivers:**
- #88 int64/uint64 as string/number (LOW complexity, table stakes)
- #89 Enum string encoding with custom values (LOW-MEDIUM complexity, table stakes)
- #87 Nullable primitives (MEDIUM complexity, table stakes, blocks #93)

**Uses:**
- Shared annotation package from Phase 1
- Custom protobuf field options in proto/sebuf/http/annotations.proto
- Custom MarshalJSON/UnmarshalJSON generation (similar to existing unwrap pattern)

**Implements:**
- Per-field annotation semantics across all 4 generators
- Cross-generator conformance test suite (one fixture, 4 outputs)
- OpenAPI schema mapping for each feature

### Phase 3: Differentiator Features (v1.0 Complete)
**Rationale:** Implement the features that distinguish sebuf from competitors. Oneof discriminated union is high complexity, so defer it if needed. Bytes encoding and timestamp formats are simpler.

**Delivers:**
- #95 Bytes encoding (LOW complexity, strong differentiator)
- #92 Timestamp formats (MEDIUM complexity, moderate differentiator)
- #93 Empty object handling (MEDIUM complexity, moderate differentiator, depends on #87)
- #90 Oneof discriminated union (HIGH complexity, phased: discriminator first, flatten later)

**Addresses:**
- Features FEATURES.md categorized as "should have"
- Pitfall #7 (Well-Known Types) — timestamp format annotation only applies to google.protobuf.Timestamp
- Pitfall #5 (oneof flattening) — validate field name conflicts at generation time

**Avoids:**
- #96 Nested flattening (defer to post-1.0, niche use case)
- #94 Field name casing (skip, use proto3 json_name instead)

### Phase 4: Multi-Language Generators (v2.0)
**Rationale:** After v1.0 JSON mapping features are complete, add 8 new language generators. Each must implement all JSON mapping features from day one. Start with Python (largest ecosystem) and Rust (strictest type system) to validate the client contract, then add mobile (Swift, Kotlin), enterprise (Java, C#), and web (Ruby, Dart).

**Delivers:**
- Python client generator (httpx-based, pure dataclasses)
- Rust client generator (reqwest + serde, standalone structs)
- Swift client generator (URLSession + Codable)
- Kotlin client generator (OkHttp + kotlinx.serialization, JVM-only initially)
- Java client generator (java.net.http + inline JSON or Gson)
- C# client generator (System.Net.Http + System.Text.Json)
- Ruby client generator (Faraday-based)
- Dart client generator (package:http-based)

**Addresses:**
- Pitfall #9 (multi-language idiom mismatch) — define language-agnostic client contract first
- Pitfall #13 (generated code dependencies) — minimize runtime deps, use stdlib where possible
- Pitfall #14 (community contributor quality) — conformance checklist and canonical test proto

**Implements:**
- Conformance test suite for all generators (same proto, N outputs)
- Language-specific golden file tests
- Runtime library versioning strategy

### Phase Ordering Rationale

- **Foundation before features**: Extracting shared annotations prevents 40+ duplication points. Architecture research measured 1,289 lines of duplication across 4 files — adding 10 features without fixing this creates unmaintainable code.

- **Table stakes before differentiators**: Features #87, #88, #89 are what users expect from any HTTP API toolkit. Without them, sebuf lacks credibility. They're also simpler than #90 (oneof) and can be delivered faster.

- **Simple before complex**: Within each phase, order by complexity. #88 (int64) is a simple type swap. #90 (oneof discriminated union) requires structural JSON changes and custom unmarshal logic. Deliver quick wins first.

- **Multi-language after JSON mapping**: New language generators must implement all JSON features. Attempting to add 8 generators while JSON mapping is still in flux creates 8x rework. Lock down the annotation design in v1.0, then expand to new languages.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (#90 oneof discriminated):** Complex structural transformation with field collision detection — needs protojson interaction research and discriminated union pattern research
- **Phase 4 (Java generator):** Java ecosystem fragmented (Jackson vs Gson, Records vs classes) — needs Java HTTP client best practices research
- **Phase 4 (Kotlin generator):** Kotlin Multiplatform (Ktor) vs JVM-only (OkHttp) decision — needs KMP adoption research

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** Extract shared code is pure refactoring — well-understood Go patterns, golden files validate correctness
- **Phase 2 (#88, #89):** int64 and enum encoding are well-documented protobuf issues with established solutions
- **Phase 3 (#95 bytes encoding):** Simple encoding function swap — standard library support in all languages

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Existing protogen/protojson foundation proven, no new major dependencies |
| Features | HIGH | 3 table stakes + 4 differentiators clearly defined, 2 correctly identified as skip/defer |
| Architecture | HIGH | Duplication measured (1,289 lines), shared package approach standard Go pattern |
| Pitfalls | HIGH | 15 pitfalls documented with real examples from grpc-gateway, connect-rpc, protobuf issues |

**Overall confidence:** HIGH

The research is comprehensive (779 lines STACK.md, 429 lines FEATURES.md, 409 lines ARCHITECTURE.md, 702 lines PITFALLS.md). Source quality is HIGH: official protobuf docs, protojson godoc, competing tool documentation (grpc-gateway, connect-rpc, twirp), and GitHub issues from protobuf/golang/protobuf demonstrating real-world problems.

### Gaps to Address

- **Annotation design details**: The research identifies WHAT annotations are needed (#87-#96) but not the exact proto syntax. During Phase 2 planning, design the actual field options (enum vs message types, field names, defaults). Pitfall #12 recommends message types for extensibility.

- **TypeScript client dependency strategy**: Current ts-client has zero dependencies (pure fetch). Some JSON mapping features may benefit from helper libraries. During Phase 2-3 planning, decide on dependency policy (inline all code vs selective imports).

- **Kotlin Multiplatform timing**: STACK.md recommends starting with JVM-only OkHttp for simplicity, defer Ktor/KMP. During Phase 4 planning, reassess based on KMP adoption trends and community demand.

- **Java JSON library choice**: STACK.md notes Jackson vs Gson fragmentation. Recommendation is inline JSON code to avoid ANY dependency. During Phase 4 planning, validate this is practical with protobuf type complexity.

- **Golden file organization**: Pitfall #6 identifies coming explosion (504 files after v2.0). Before Phase 1 completes, decide on new directory structure (by feature not generator) and implement diff summary tooling.

## Sources

### Primary (HIGH confidence)
- [ProtoJSON Format Specification](https://protobuf.dev/programming-guides/json/) — Canonical JSON mapping rules
- [protojson Go Package](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) — MarshalOptions API and limitations
- [Protobuf Field Presence](https://protobuf.dev/programming-guides/field_presence/) — Optional field semantics
- [gRPC-Gateway Customization Docs](https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_your_gateway/) — Runtime JSON options approach
- [Connect Serialization Docs](https://connectrpc.com/docs/go/serialization-and-compression/) — Codec-based approach
- [Twirp Serialization Docs](https://twitchtv.github.io/twirp/docs/proto_and_json.html/) — Proto name preservation rationale
- [ts-proto GitHub](https://github.com/stephenh/ts-proto) — TypeScript code generation patterns
- [Protobuf-ES npm](https://www.npmjs.com/package/@bufbuild/protobuf) — Reference TypeScript implementation
- [protoc-gen-go-json by mfridman](https://github.com/mfridman/protoc-gen-go-json) — Custom JSON marshaling example

### Secondary (MEDIUM confidence)
- [golang/protobuf issue #1414](https://github.com/golang/protobuf/issues/1414) — int64 as number request (not implemented)
- [golang/protobuf issue #1030](https://github.com/golang/protobuf/issues/1030) — Hex bytes encoding request (not implemented)
- [grpc-gateway issue #438](https://github.com/grpc-ecosystem/grpc-gateway/issues/438) — int64 string encoding complaints
- [grpc-gateway issue #82](https://github.com/grpc-ecosystem/grpc-gateway/issues/82) — Oneof query param support request
- [grpc-gateway issue #585](https://github.com/grpc-ecosystem/grpc-gateway/issues/585) — Discriminator support request
- [protobuf issue #2679](https://github.com/protocolbuffers/protobuf/issues/2679) — Why int64 as string (canonical answer)
- [protobuf issue #6355](https://github.com/protocolbuffers/protobuf/issues/6355) — Unknown enum value handling
- [protobuf issue #4549](https://github.com/protocolbuffers/protobuf/issues/4549) — Ruby well-known types incompatibility
- [OpenAPI discriminator guide](https://redocly.com/learn/openapi/discriminator) — Discriminated union pattern
- [Connect-Swift announcement](https://buf.build/blog/announcing-connect-swift) — Multi-language idiom patterns
- [GoGo Protobuf lessons](https://jbrandhorst.com/post/gogoproto/) — Avoid optimization-driven incompatibility

### Tertiary (LOW confidence)
- [betterproto2 on PyPI](https://pypi.org/project/betterproto2/) — Python client generation reference (project in transition)
- [httpx vs requests comparison](https://www.morethanmonkeys.co.uk/article/comparing-requests-and-httpx-in-python-which-http-client-should-you-use-in-2025/) — Python HTTP client recommendation

---
*Research completed: 2026-02-05*
*Ready for roadmap: yes*
