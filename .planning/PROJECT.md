# sebuf

## What This Is

A specialized Go protobuf toolkit for building HTTP APIs without gRPC dependencies. Four protoc plugins generate HTTP handlers (Go), type-safe HTTP clients (Go, TypeScript), and OpenAPI v3.1 specs — all from protobuf service definitions with custom HTTP annotations. Targets web and mobile API development with built-in request/header validation, structured error handling, and flexible JSON serialization.

## Core Value

Proto definitions are the single source of truth for HTTP APIs — every generator (server, client, docs) must produce consistent, correct output that interoperates seamlessly.

## Requirements

### Validated

- ✓ HTTP handler generation with automatic request/response binding — existing
- ✓ Service and method-level header validation middleware — existing
- ✓ Request body validation via buf.validate/protovalidate — existing
- ✓ HTTP verb support (GET, POST, PUT, DELETE, PATCH) with path/query parameters — existing
- ✓ Go HTTP client generation with functional options pattern — existing
- ✓ TypeScript HTTP client generation with typed interfaces and error handling — existing
- ✓ OpenAPI v3.1 specification generation (one file per service) — existing
- ✓ Unwrap annotation for map-value and root-level JSON serialization — existing
- ✓ Structured error responses (ValidationError, ApiError, custom proto errors) — existing
- ✓ Custom error handler support (WithErrorHandler ServerOption) — existing
- ✓ Proto3 optional field support across generators — existing
- ✓ Mock server generation for testing — existing
- ✓ Multi-platform binary distribution (Homebrew, Docker, deb/rpm/apk) — existing

### Active

**v1.0 — Polish & JSON Mapping:**
- [ ] Fix #105: Conditional net/url import in go-client (only when query params used)
- [ ] Land PR #98: Cross-file unwrap resolution (same Go package)
- [ ] #87: Nullable primitives (null vs absent vs default) — all 4 generators
- [ ] #88: int64/uint64 as string encoding — all 4 generators
- [ ] #89: Enum string encoding with custom values — all 4 generators
- [ ] #90: Oneof as discriminated union (flattened with type field) — all 4 generators
- [ ] #91: Root-level arrays (verify coverage via existing unwrap, close if done)
- [ ] #92: Multiple timestamp formats (RFC3339, Unix seconds/millis, date-only) — all 4 generators
- [ ] #93: Empty object handling (preserve vs omit vs null) — all 4 generators
- [ ] #94: Field name casing options (file-level default, field override) — all 4 generators
- [ ] #95: Bytes encoding options (base64, base64url, hex) — all 4 generators
- [ ] #96: Nested message flattening with prefix — all 4 generators
- [ ] Review and improve test coverage across all generators
- [ ] Review and improve documentation quality (README, examples, inline docs)
- [ ] Ensure proto/OpenAPI/JSON consistency across all generators
- [ ] Improve examples (including #50: multi-auth patterns)

**v2.0 — Multi-Language Clients:**
- [ ] Python HTTP client generator (protoc-gen-py-client)
- [ ] Rust HTTP client generator (protoc-gen-rs-client)
- [ ] Swift HTTP client generator (protoc-gen-swift-client)
- [ ] Kotlin HTTP client generator (protoc-gen-kt-client)
- [ ] Java HTTP client generator (protoc-gen-java-client)
- [ ] C# HTTP client generator (protoc-gen-csharp-client)
- [ ] Ruby HTTP client generator (protoc-gen-rb-client)
- [ ] Dart HTTP client generator (protoc-gen-dart-client)

**v3.0 — Multi-Language HTTP Servers:**
- [ ] HTTP server generation for additional languages

### Out of Scope

- gRPC support — sebuf targets HTTP APIs specifically, not gRPC
- GraphQL generation — different API paradigm, out of scope
- Database/ORM integration — sebuf generates HTTP layer only
- Runtime framework (router, middleware library) — generates code for standard library
- #101 deprecated option support — deferred to v1.1
- #102 binary response types — deferred to v1.1
- Streaming/WebSocket support — deferred to future version

## Context

**Current state:** Four working protoc plugins with comprehensive golden file tests. Codebase is stable with 85% coverage threshold enforced. Community contributors have started work (Swift service generation draft PR #72). Ten JSON mapping issues filed as 1.0 blockers — these represent the gap between protobuf's type system and real-world REST API JSON patterns.

**JSON mapping gap:** The core problem is that protobuf's wire format doesn't map 1:1 to how REST APIs serialize JSON. Custom annotations (sebuf.http.*) bridge this gap. Each annotation must work consistently across go-http (server marshal/unmarshal), go-client (same marshaling), ts-client (TypeScript equivalent), and openapiv3 (schema documentation). This cross-generator consistency is the hardest part.

**Multi-language expansion:** Go and TypeScript clients provide reference implementations. Each new language client follows the same conceptual pattern (service class, method calls, option pattern, error types) but uses idiomatic language constructs. The protoc plugin architecture makes this extensible — each language is a new cmd/ entry point and internal/ generator.

**Open PRs:** PR #98 (cross-file unwrap) needs review and merge. PR #72 (Swift, community) is a draft — may inform the v2.0 Swift client approach.

## Constraints

- **Tech stack**: Go for all plugin implementations (protogen framework requirement)
- **Compatibility**: All JSON mapping features must be backward-compatible (opt-in via annotations)
- **Cross-generator consistency**: Every annotation must work identically across all 4 generators
- **Proto conventions**: Custom annotations live in `proto/sebuf/http/` namespace
- **Testing**: Golden file tests required for all generated output changes, 85% coverage minimum
- **Distribution**: Must maintain existing multi-platform binary distribution (Homebrew, Docker, packages)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| All 10 JSON mapping issues are 1.0 blockers | Users need complete JSON control for real-world API compatibility | — Pending |
| Defer #101 (deprecated) and #102 (binary) to v1.1 | Keep v1.0 focused on JSON mapping consistency | — Pending |
| 8 languages for v2.0 clients: Python, Rust, Swift, Kotlin, Java, C#, Ruby, Dart | Top mainstream languages covering web, mobile, systems, and scripting | — Pending |
| HTTP server generation deferred to v3.0 | Clients are higher value — servers only needed in Go currently | — Pending |
| Use existing unwrap annotation pattern for root-level arrays | PR #103 already merged this approach, consistent with existing API | — Pending |

---
*Last updated: 2026-02-05 after initialization*
