# STACK.md -- Technology Stack for JSON Mapping & Multi-Language Client Generation

*Research date: 2026-02-05*
*Scope: JSON mapping layer implementation + 8 new language client generators*
*Does NOT cover: existing Go/protogen stack, existing plugin architecture*

---

## Table of Contents

1. [JSON Mapping Layer: The Core Problem](#1-json-mapping-layer-the-core-problem)
2. [How Existing Tools Handle JSON Mapping](#2-how-existing-tools-handle-json-mapping)
3. [Recommended JSON Mapping Implementation for sebuf](#3-recommended-json-mapping-implementation-for-sebuf)
4. [Multi-Language Client Generation Stack](#4-multi-language-client-generation-stack)
5. [What NOT to Use and Why](#5-what-not-to-use-and-why)
6. [Cross-Generator Consistency Strategy](#6-cross-generator-consistency-strategy)
7. [Version Reference Table](#7-version-reference-table)

---

## 1. JSON Mapping Layer: The Core Problem

The ProtoJSON specification (protobuf.dev/programming-guides/json/) defines canonical mappings that do not match many real-world REST API conventions. The 10 JSON mapping features sebuf needs fall into three categories:

### Category A: Type Representation Overrides
These change HOW a protobuf type serializes to JSON:

| Feature | ProtoJSON Default | What APIs Actually Need |
|---------|------------------|------------------------|
| Nullable primitives (#87) | No null for scalars; `optional` gives presence tracking but emits default, not null | Explicit `null` in JSON for unset optional fields |
| int64-as-string (#88) | int64/uint64 always serialize as JSON strings | Option to emit as JSON numbers (safe for values < 2^53) |
| Enum encoding (#89) | Enum name string (e.g., `"STATUS_ACTIVE"`) | Custom JSON values, numeric encoding option |
| Bytes encoding (#95) | Standard base64 (RFC 4648 section 4) | base64url (section 5), hex encoding |
| Timestamp formats (#92) | RFC 3339 string (`"2025-01-01T00:00:00Z"`) | Unix seconds (number), Unix millis, date-only |

### Category B: Structure Transformations
These change the JSON SHAPE:

| Feature | ProtoJSON Default | What APIs Actually Need |
|---------|------------------|------------------------|
| Oneof discriminated union (#90) | Flat: only the set field appears | Discriminated: `{"type": "email", "email": {...}}` |
| Root-level arrays (#91) | Always wrapped in object | Direct `[...]` at response root |
| Nested flattening (#96) | Full nesting: `{"address": {"city": "NYC"}}` | Flattened with prefix: `{"address_city": "NYC"}` |
| Empty object handling (#93) | Omit empty messages/fields | Preserve `{}`, emit `null`, or omit |

### Category C: Naming Conventions

| Feature | ProtoJSON Default | What APIs Actually Need |
|---------|------------------|------------------------|
| Field name casing (#94) | lowerCamelCase (or json_name override) | snake_case, SCREAMING_SNAKE, kebab-case, PascalCase |

---

## 2. How Existing Tools Handle JSON Mapping

### 2.1 grpc-gateway (v2)

**Approach:** Runtime configuration via `protojson.MarshalOptions` on the `runtime.ServeMux`.

**What it exposes:**
- `UseProtoNames: true` -- use proto field names instead of camelCase
- `EmitUnpopulated: true` -- emit zero-value fields
- `EmitDefaultValues: true` -- emit fields with default values (replaces deprecated EmitUnpopulated)
- `UseEnumNumbers: true` -- emit enum values as numbers instead of strings
- `Write64KindsAsInteger: true` -- write int64/uint64 as JSON numbers instead of strings
- `DiscardUnknown: true` -- ignore unknown fields during unmarshal

**What it does NOT handle:**
- No nullable primitive support (null vs absent vs default)
- No custom timestamp formats (locked to RFC 3339)
- No bytes encoding options (locked to base64)
- No oneof discriminated union format
- No field name casing options beyond proto name vs camelCase
- No nested flattening

**Key insight:** grpc-gateway delegates entirely to `protojson` for serialization. Its JSON customization is limited to what `protojson.MarshalOptions` provides. This is insufficient for sebuf's needs.

**Confidence: HIGH** -- based on official docs and source code review.

### 2.2 Connect (connectrpc.com)

**Approach:** Codec-based. Default JSON codec uses standard protobuf JSON mapping. Custom codecs can be registered under the `"json"` name to override behavior.

**What it exposes:**
- `WithProtoJSON()` -- use JSON instead of binary protobuf for clients
- `WithCodec()` -- register a fully custom codec for serialization
- Binary and JSON supported by default on handlers

**What it does NOT handle:**
- No field-level JSON customization annotations
- No per-field type overrides (int64 encoding, timestamp format, etc.)
- Custom behavior requires replacing the entire codec

**Key insight:** Connect explicitly chose NOT to provide field-level JSON customization. Their philosophy is "use the protobuf JSON mapping or write your own codec." This is intentional -- they view JSON as a compatibility layer, not a first-class serialization target.

**Confidence: HIGH** -- based on official docs and Connect protocol spec.

### 2.3 Twirp (twitchtv/twirp)

**Approach:** Server-level options for JSON serialization behavior.

**What it exposes:**
- `WithServerJSONCamelCaseNames(true)` -- use camelCase (default is proto field names)
- `WithServerJSONSkipDefaults(true)` -- skip zero-value fields (was default before v7)

**What it does NOT handle:**
- No field-level annotations
- No int64/enum/bytes encoding options
- No timestamp format options
- No structural transformations

**Key insight:** Twirp intentionally uses proto field names by default (not camelCase) because "JSON encoding is often used for manual debugging of the API." They prioritize predictability over convention.

**Confidence: HIGH** -- based on official docs.

### 2.4 ts-proto (stephenh/ts-proto)

**Approach:** Code generation flags that control TypeScript output. Most relevant to sebuf's TypeScript client generator.

**What it exposes (via protoc options):**
- `--ts_proto_opt=snakeToCamel=false` -- keep field names as snake_case
- `--ts_proto_opt=snakeToCamel=keys,json` -- granular control over key vs JSON name casing
- `--ts_proto_opt=oneof=unions` -- generate ADTs for oneof fields (discriminated unions)
- `--ts_proto_opt=useDate=false` -- don't map Timestamp to Date
- `--ts_proto_opt=protoJsonFormat=true` -- follow ProtoJSON format in toJSON/fromJSON

**Key insight:** ts-proto provides the closest model to what sebuf needs for TypeScript. Its option-based approach at code generation time (not runtime) is the right pattern. However, ts-proto uses global flags, not per-field annotations.

**Latest version:** 2.7.7 (December 2025).
**Confidence: HIGH** -- based on GitHub README and npm registry.

### 2.5 Protobuf-ES (@bufbuild/protobuf)

**Approach:** Full ProtoJSON conformance test compliance. Reflection-based serialization in v2.

**What it exposes:**
- Full ProtoJSON spec compliance (the ONLY JavaScript library that passes conformance tests)
- Reflection API for field-level introspection
- Custom options support via Protobuf custom options

**Key insight:** Protobuf-ES 2.0 is the reference implementation for JavaScript/TypeScript protobuf. Its reflection API could be used by sebuf's generated TypeScript clients to implement custom JSON mapping at runtime, but sebuf's approach of generating the mapping code directly is more appropriate for a protoc plugin.

**Latest version:** 2.10.2 (January 2026).
**Confidence: HIGH** -- based on npm registry and official blog posts.

### 2.6 betterproto (python-betterproto2)

**Approach:** Code generation that produces idiomatic Python dataclasses with built-in JSON serialization.

**What it exposes:**
- Snake_case field names by default (Pythonic)
- Dataclass-based messages
- Built-in `to_dict()` and `from_dict()` methods
- DateTime conversion for Timestamp fields

**Key insight:** betterproto2 demonstrates the "idiomatic language output" approach that sebuf should follow for Python. However, it is a gRPC-focused tool and its JSON mapping is not customizable per-field.

**Latest version:** 0.9.1 (October 2025).
**Confidence: MEDIUM** -- project is in active redesign (betterproto -> betterproto2 transition).

---

## 3. Recommended JSON Mapping Implementation for sebuf

### 3.1 Architecture Decision: Annotation-Driven Code Generation

**Decision:** Implement JSON mapping via custom protobuf annotations in `proto/sebuf/http/` that control code generation output. Do NOT use runtime configuration like grpc-gateway.

**Rationale:**
1. sebuf is a code generator, not a runtime library -- annotations are the natural control mechanism
2. Per-field granularity (e.g., "this int64 is a string, that one is a number") requires field-level annotations, not global flags
3. Generated code is inspectable, testable, and has zero runtime overhead
4. Cross-generator consistency is enforced at generation time, not runtime

**What this means concretely:**
- Each of the 10 JSON features gets one or more annotation definitions in `proto/sebuf/http/`
- Each generator (go-http, go-client, ts-client, openapiv3) reads these annotations and generates appropriate serialization/deserialization code
- The OpenAPI generator maps these annotations to equivalent JSON Schema / OpenAPI constructs

### 3.2 Go JSON Serialization Library

**Primary: `google.golang.org/protobuf/encoding/protojson` v1.36.11**

Use `protojson` as the baseline for Go serialization, then layer custom behavior on top via generated code.

**How to layer custom behavior:**

For features that `protojson.MarshalOptions` supports directly:
- int64-as-number: Use `Write64KindsAsInteger: true` (available in v1.36+)
- Enum-as-number: Use `UseEnumNumbers: true`
- Emit defaults: Use `EmitDefaultValues: true`
- Proto field names: Use `UseProtoNames: true`

For features that `protojson` does NOT support (nullable, timestamps, bytes, flattening, oneof discriminated, custom casing):
- Generate custom marshal/unmarshal functions that wrap protojson
- Use `protojson.Marshal()` for the base serialization, then post-process the JSON
- OR generate `encoding/json` marshal/unmarshal methods directly on wrapper types

**Recommended pattern for custom features:**
```
// Generated code pattern for custom JSON features:
// 1. Use protojson for standard fields
// 2. Generate custom MarshalJSON/UnmarshalJSON for messages with custom annotations
// 3. The custom marshaler handles annotated fields, delegates rest to protojson
```

**Do NOT use `encoding/json` directly for the whole message** -- protojson handles Well-Known Types (Timestamp, Duration, Struct, Any, wrappers) correctly, and reimplementing that is error-prone.

**Confidence: HIGH** -- protojson is the canonical Go library, maintained by the protobuf team.

### 3.3 TypeScript JSON Serialization

**Approach:** Generate custom `toJSON()` and static `fromJSON()` methods on TypeScript interfaces/classes.

The existing ts-client generator already produces JSON serialization code. Extend it to handle annotated fields differently:

- Nullable primitives: emit `null` instead of omitting the field when annotation is present
- int64: generate `number` type when annotation says numeric, `string` when it says string
- Enum: generate string literal union types for the custom enum values
- Timestamps: generate appropriate serialization (number for Unix, string for RFC 3339, etc.)
- Bytes: generate appropriate encoding function calls (btoa for base64, custom for base64url/hex)

**No additional npm dependencies needed** -- all serialization is generated inline.

**Confidence: HIGH** -- this follows the existing sebuf pattern.

### 3.4 OpenAPI Schema Mapping

Each JSON mapping annotation must produce the correct OpenAPI 3.1 / JSON Schema representation:

| Feature | OpenAPI Schema Effect |
|---------|----------------------|
| Nullable primitives | `"nullable": true` or `{"type": ["string", "null"]}` (OpenAPI 3.1 uses JSON Schema 2020-12) |
| int64-as-string | `{"type": "string", "format": "int64"}` |
| int64-as-number | `{"type": "integer", "format": "int64"}` |
| Enum custom values | `{"enum": ["active", "inactive"]}` with custom strings |
| Timestamp as Unix | `{"type": "number"}` or `{"type": "integer"}` |
| Timestamp as RFC 3339 | `{"type": "string", "format": "date-time"}` |
| Bytes as base64url | `{"type": "string", "format": "byte"}` with description |
| Oneof discriminated | `{"oneOf": [...], "discriminator": {"propertyName": "type"}}` |
| Nested flattening | Flatten properties into parent schema with prefixed names |
| Field name casing | Use the cased name as the JSON property name in schema |

**Library:** Continue using `libopenapi v0.33.0` for OpenAPI document construction. No version change needed.

**Confidence: HIGH** -- OpenAPI 3.1 natively supports all these constructs.

### 3.5 Annotation Design Pattern

**Recommended annotation structure (for `proto/sebuf/http/annotations.proto`):**

Field-level annotations (extending `google.protobuf.FieldOptions`):
- `sebuf.http.json_type` -- override JSON type encoding (e.g., int64 as number)
- `sebuf.http.json_name_style` -- override field name casing
- `sebuf.http.nullable` -- emit null for unset optional fields
- `sebuf.http.timestamp_format` -- RFC3339, UNIX_SECONDS, UNIX_MILLIS, DATE_ONLY
- `sebuf.http.bytes_encoding` -- BASE64, BASE64URL, HEX
- `sebuf.http.flatten` -- flatten nested message with optional prefix
- `sebuf.http.oneof_style` -- FLAT (default) or DISCRIMINATED with discriminator field name

File-level or message-level annotations:
- `sebuf.http.default_name_style` -- default casing for all fields in file/message
- `sebuf.http.empty_handling` -- OMIT, PRESERVE, NULL for empty messages

**Confidence: HIGH** -- this follows the existing `sebuf.http.unwrap` pattern and is backward-compatible.

---

## 4. Multi-Language Client Generation Stack

All 8 client generators are protoc plugins written in Go using `google.golang.org/protobuf/compiler/protogen`. They read the same protobuf descriptors and annotations, then emit language-specific HTTP client code.

### 4.1 Code Generation Architecture

**Pattern: Direct string emission via `protogen.GeneratedFile`**

This is the same pattern used by the existing go-http, go-client, and ts-client generators. Each generator:
1. Iterates over services and methods from the protobuf descriptor
2. Reads HTTP annotations (method, path, headers) and JSON mapping annotations
3. Emits language-specific source code as strings

**Do NOT use template engines** (text/template, Mustache, etc.) for code generation. The existing codebase uses programmatic string building via `g.P()` (print line) calls, which is:
- Easier to debug (step through in a debugger)
- Easier to test (golden file comparison)
- More flexible for conditional logic
- Consistent with how protoc-gen-go itself works

**Confidence: HIGH** -- this is the proven pattern in the existing codebase.

### 4.2 Python Client Generator (`protoc-gen-py-client`)

**Target output:** A single `.py` file per service with typed dataclasses and HTTP client class.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client (generated code uses) | `httpx` | 0.28.x | Sync+async in one library, HTTP/2 support, modern Python standard |
| Type hints | Built-in `typing` | Python 3.10+ | Native dataclasses + type hints, no runtime dependency |
| JSON serialization | Built-in `json` | stdlib | No dependency needed; generated code handles proto-to-dict conversion |
| Protobuf runtime | None | - | Generated code is pure Python, no protobuf dependency |

**Generated code pattern:**
```python
@dataclass
class CreateUserRequest:
    name: str
    email: str
    age: int | None = None  # nullable primitive

class UserServiceClient:
    def __init__(self, base_url: str, *, api_key: str | None = None):
        self._client = httpx.Client(base_url=base_url)

    def create_user(self, request: CreateUserRequest, **options) -> User:
        ...
```

**Why httpx over requests:** httpx supports both sync and async APIs, HTTP/2, and is the modern Python HTTP standard. requests is sync-only and aging. Generated code should use httpx for forward-compatibility.

**Why no protobuf runtime dependency:** The generated client should be standalone -- users should not need `google-protobuf` or `betterproto` installed. The generator produces pure Python dataclasses that serialize to JSON directly. This maximizes adoption.

**Confidence: HIGH** for httpx, **MEDIUM** for pure-Python approach (some users may want proto-native types).

### 4.3 Rust Client Generator (`protoc-gen-rs-client`)

**Target output:** A Rust module (`.rs` file) per service with typed structs and client implementation.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client | `reqwest` | 0.13.x | De facto Rust HTTP client, async-first, well-maintained |
| Serialization | `serde` + `serde_json` | 1.x / 1.x | Rust's universal serialization framework |
| Protobuf types | `prost` | 0.14.x | De facto Rust protobuf library, tokio-maintained |
| Serde for proto | `pbjson` / `pbjson-build` | 0.8.x | Generates serde impls that follow ProtoJSON conventions |
| Async runtime | `tokio` | 1.x | Required by reqwest, de facto async runtime |

**Generated code pattern:**
```rust
#[derive(Clone, Debug, serde::Serialize, serde::Deserialize)]
pub struct CreateUserRequest {
    pub name: String,
    pub email: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub age: Option<i32>,
}

pub struct UserServiceClient {
    base_url: String,
    client: reqwest::Client,
}

impl UserServiceClient {
    pub async fn create_user(&self, req: &CreateUserRequest) -> Result<User, ApiError> {
        ...
    }
}
```

**Two possible approaches:**
1. **prost + pbjson-build:** Generate prost types and serde impls. Users add prost, serde, reqwest to their Cargo.toml. More "Rust-native" but requires protobuf build tooling.
2. **Pure serde structs:** Generate standalone serde structs without prost dependency. Simpler for users but loses proto binary serialization.

**Recommendation:** Start with approach 2 (pure serde structs) for v2.0. This matches the "no protobuf runtime dependency" philosophy and is simpler for users who just want an HTTP client. Add prost-compatible output as an opt-in flag later.

**Confidence: HIGH** for reqwest + serde, **MEDIUM** for pure-serde vs prost decision.

### 4.4 Swift Client Generator (`protoc-gen-swift-client`)

**Target output:** A Swift file per service with Codable structs and async client.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client | `URLSession` | Foundation (stdlib) | Built into iOS/macOS, no dependency needed |
| JSON serialization | `Codable` + `JSONEncoder/Decoder` | Foundation (stdlib) | Swift's native serialization, no dependency |
| Protobuf runtime | None (or optional `swift-protobuf` 1.33.x) | - | Generated Codable structs are standalone |
| Concurrency | `async/await` | Swift 5.5+ | Native Swift concurrency |

**Generated code pattern:**
```swift
struct CreateUserRequest: Codable, Sendable {
    let name: String
    let email: String
    let age: Int?  // nullable primitive
}

actor UserServiceClient {
    private let baseURL: URL
    private let session: URLSession

    func createUser(_ request: CreateUserRequest) async throws -> User {
        ...
    }
}
```

**Why URLSession over Alamofire:** URLSession is built-in, supports async/await natively since iOS 15, and keeps the generated code dependency-free. Alamofire adds complexity for minimal benefit in generated code.

**Why no swift-protobuf dependency:** Same philosophy as Python -- generated code should be standalone Codable structs. Users who need proto binary can use swift-protobuf separately.

**Note:** Connect-Swift exists (by Buf) and uses URLSession. It is a good reference implementation but targets gRPC/Connect protocol, not generic HTTP APIs.

**Confidence: HIGH** -- URLSession + Codable is the undisputed Swift standard.

### 4.5 Kotlin Client Generator (`protoc-gen-kt-client`)

**Target output:** Kotlin source files with data classes and coroutine-based client.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client (JVM) | `OkHttp` | 5.2.x | De facto JVM HTTP client, Kotlin-native |
| HTTP client (KMP) | `Ktor` | 3.4.x | Kotlin Multiplatform support (JVM + iOS + JS) |
| JSON serialization | `kotlinx.serialization` | 1.10.x | Kotlin-native, compiler plugin, multiplatform |
| Coroutines | `kotlinx.coroutines` | 1.10.x | Standard Kotlin async |

**Two target profiles:**
1. **JVM-only (simpler):** OkHttp + kotlinx.serialization.json. Best for server-side Kotlin and Android.
2. **Kotlin Multiplatform:** Ktor client + kotlinx.serialization.json. Targets JVM, iOS, JS, Native.

**Recommendation:** Start with JVM-only (OkHttp) for v2.0. Ktor/KMP adds significant complexity. Offer KMP as a future option.

**Generated code pattern:**
```kotlin
@Serializable
data class CreateUserRequest(
    val name: String,
    val email: String,
    val age: Int? = null
)

class UserServiceClient(
    private val baseUrl: String,
    private val client: OkHttpClient = OkHttpClient()
) {
    suspend fun createUser(request: CreateUserRequest): User {
        ...
    }
}
```

**Confidence: HIGH** for OkHttp + kotlinx.serialization, **MEDIUM** for KMP/Ktor timing.

### 4.6 Java Client Generator (`protoc-gen-java-client`)

**Target output:** Java source files with record types (Java 17+) and HttpClient-based client.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client | `java.net.http.HttpClient` | Java 11+ stdlib | No dependency needed, modern, async-capable |
| JSON serialization | `com.google.code.gson:gson` | 2.x | Simpler than Jackson, lightweight, widely used |
| Alternative JSON | `com.fasterxml.jackson:jackson-databind` | 2.x | More feature-rich, better for complex mappings |
| Protobuf runtime | None (or optional `protobuf-java` 4.33.x) | - | Generated records are standalone |

**Generated code pattern:**
```java
public record CreateUserRequest(
    String name,
    String email,
    @Nullable Integer age
) {}

public class UserServiceClient {
    private final HttpClient httpClient;
    private final String baseUrl;

    public CompletableFuture<User> createUser(CreateUserRequest request) {
        ...
    }
}
```

**Decision point: Jackson vs Gson:**
- Jackson is more powerful but heavier (pulls in many transitive dependencies)
- Gson is lighter and simpler, sufficient for generated serialization code
- **Recommendation:** Generate serialization code inline (like toJson/fromJson methods) to avoid ANY JSON library dependency. Use `java.net.http` + manual JSON building for maximum portability. Fall back to Gson if manual JSON is too complex.

**Confidence: MEDIUM** -- Java ecosystem is fragmented; inline JSON may be impractical for complex nested types.

### 4.7 C# Client Generator (`protoc-gen-csharp-client`)

**Target output:** C# source files with record types and HttpClient-based client.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client | `System.Net.Http.HttpClient` | .NET 6+ stdlib | Built-in, no dependency |
| JSON serialization | `System.Text.Json` | .NET 6+ stdlib | Built-in, performant, source-gen capable |
| Protobuf runtime | None (or optional `Google.Protobuf` 3.33.x) | - | Generated records are standalone |

**Generated code pattern:**
```csharp
public record CreateUserRequest(
    string Name,
    string Email,
    int? Age = null
);

public class UserServiceClient : IDisposable {
    private readonly HttpClient _httpClient;

    public async Task<User> CreateUserAsync(
        CreateUserRequest request,
        CancellationToken ct = default) {
        ...
    }
}
```

**Why System.Text.Json over Newtonsoft.Json:** System.Text.Json is built into .NET 6+, has better performance, supports source generation for AOT scenarios, and is the Microsoft-recommended path forward. Newtonsoft.Json is legacy at this point.

**Confidence: HIGH** -- HttpClient + System.Text.Json is the clear modern .NET stack.

### 4.8 Ruby Client Generator (`protoc-gen-rb-client`)

**Target output:** Ruby files with plain classes and Faraday-based HTTP client.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client | `faraday` | 2.14.x | De facto Ruby HTTP client, middleware architecture |
| JSON serialization | Built-in `json` | stdlib | Ruby's standard JSON library |
| Protobuf runtime | None (or optional `google-protobuf` gem) | - | Generated classes are standalone |

**Generated code pattern:**
```ruby
class CreateUserRequest
  attr_accessor :name, :email, :age

  def initialize(name:, email:, age: nil)
    @name = name
    @email = email
    @age = age
  end

  def to_json(*args)
    { name: @name, email: @email, age: @age }.compact.to_json(*args)
  end
end

class UserServiceClient
  def initialize(base_url, api_key: nil)
    @conn = Faraday.new(url: base_url)
  end

  def create_user(request, **options)
    ...
  end
end
```

**Why Faraday over net/http:** While Ruby's `net/http` is stdlib, Faraday provides a clean middleware architecture that matches sebuf's header/auth patterns. It also abstracts HTTP adapters, letting users swap backends.

**Confidence: HIGH** for Faraday, **MEDIUM** for Ruby ecosystem relevance (smaller community than other targets).

### 4.9 Dart Client Generator (`protoc-gen-dart-client`)

**Target output:** Dart files with classes and http package-based client.

| Component | Library | Version | Rationale |
|-----------|---------|---------|-----------|
| HTTP client | `package:http` | 1.3.x | Official Dart HTTP client, works in Flutter |
| JSON serialization | Built-in `dart:convert` | stdlib | Dart's standard JSON codec |
| Protobuf runtime | None (or optional `package:protobuf` 5.2.x) | - | Generated classes are standalone |

**Generated code pattern:**
```dart
class CreateUserRequest {
  final String name;
  final String email;
  final int? age;

  CreateUserRequest({required this.name, required this.email, this.age});

  Map<String, dynamic> toJson() => {
    'name': name,
    'email': email,
    if (age != null) 'age': age,
  };

  factory CreateUserRequest.fromJson(Map<String, dynamic> json) => ...;
}

class UserServiceClient {
  final String baseUrl;
  final http.Client _client;

  Future<User> createUser(CreateUserRequest request) async {
    ...
  }
}
```

**Why package:http over dio:** `package:http` is maintained by the Dart team, is simpler, and has no transitive dependencies. Dio is popular but heavier and adds unnecessary abstraction for generated code.

**Confidence: HIGH** -- package:http + dart:convert is the standard Dart/Flutter pattern.

---

## 5. What NOT to Use and Why

### 5.1 Do NOT use `encoding/json` as primary Go serializer

**Why not:** `encoding/json` does not understand protobuf Well-Known Types (Timestamp, Duration, Struct, Any, FieldMask, wrapper types). Using it directly would require reimplementing the entire WKT mapping layer. `protojson` handles all of this correctly.

**When it IS appropriate:** For the custom JSON mapping layer that sits on top of protojson (e.g., nullable primitives post-processing, custom timestamp formats).

### 5.2 Do NOT use `protoc-gen-star` (Lyft)

**Why not:** protoc-gen-star is an older abstraction layer over protoc plugins. It adds complexity without benefit when using `protogen` directly. The sebuf codebase already uses protogen effectively, and protoc-gen-star has not been actively maintained since 2023.

### 5.3 Do NOT use Go `text/template` for code generation

**Why not:** Template-based generation is harder to debug, test, and maintain than programmatic string building. The existing codebase uses `g.P()` calls consistently. Templates would introduce a second paradigm and make golden file tests harder to reason about.

### 5.4 Do NOT use `gogo/protobuf`

**Why not:** The gogo/protobuf project is deprecated and unmaintained. All Go protobuf work should use `google.golang.org/protobuf`.

### 5.5 Do NOT require language-specific protobuf runtimes

**Why not:** Generated HTTP clients should be standalone. Requiring users to install protobuf runtimes (google-protobuf for Python, swift-protobuf for Swift, etc.) creates friction. Generate plain language-native types (dataclasses, Codable structs, records) that serialize to JSON without protobuf dependencies.

**Exception:** Offer opt-in flags for users who want proto-native types (e.g., `--with-prost` for Rust, `--with-protobuf-java` for Java).

### 5.6 Do NOT use Alamofire (Swift), Retrofit (Kotlin), or RestSharp (C#)

**Why not:** These are higher-level HTTP abstraction libraries designed for hand-written API clients. Generated code should use the platform's standard HTTP library directly -- it's simpler, has fewer dependencies, and gives the generator full control over request/response handling.

### 5.7 Do NOT use Newtonsoft.Json for C#

**Why not:** `System.Text.Json` is the modern .NET standard (built-in since .NET 6), has better performance, and supports source generation. Newtonsoft.Json is legacy, though still widely used. For new generated code, System.Text.Json is the correct choice.

---

## 6. Cross-Generator Consistency Strategy

### 6.1 The Consistency Problem

Every JSON mapping annotation must produce semantically equivalent behavior across all generators:
- `go-http` (server): marshals response, unmarshals request
- `go-client`: marshals request, unmarshals response
- `ts-client`: same as go-client but in TypeScript
- `openapiv3`: documents the JSON schema
- `py-client`, `rs-client`, etc.: same as go-client in target language

If `sebuf.http.timestamp_format = UNIX_SECONDS` produces `1706140800` from go-http but `"1706140800"` (string) from ts-client, the API is broken.

### 6.2 Consistency Enforcement

**Golden file tests per annotation:**
For each JSON mapping annotation, create a test proto file and golden files for EVERY generator. The golden files for a single annotation test must be cross-validated:

```
testdata/json_mapping/
  nullable_primitives.proto
  nullable_primitives.go          # go-http golden
  nullable_primitives_client.go   # go-client golden
  nullable_primitives_client.ts   # ts-client golden
  nullable_primitives.openapi.yaml # openapiv3 golden
  nullable_primitives_client.py   # py-client golden
  ...
```

**Semantic equivalence tests:**
Write integration tests that:
1. Generate a Go server with go-http
2. Generate a client in each language
3. Send requests from each client to the Go server
4. Verify the JSON on the wire is identical regardless of which client sent it

### 6.3 Annotation Specification Document

Each JSON mapping annotation needs a specification that defines:
1. The proto annotation syntax
2. The JSON wire format (with examples)
3. The Go marshaling behavior
4. The Go unmarshaling behavior
5. The TypeScript equivalent
6. The OpenAPI schema equivalent
7. Edge cases and error handling

This specification is the authoritative reference for implementing the annotation in each generator.

---

## 7. Version Reference Table

All versions verified via web search on 2026-02-05.

### Core Dependencies (Already in Use)

| Library | Current in sebuf | Latest Available | Action |
|---------|-----------------|------------------|--------|
| Go | 1.24.7 | 1.24.7 | No change needed |
| google.golang.org/protobuf | 1.36.11 | 1.36.11 | Already latest |
| libopenapi | 0.33.0 | 0.33.0 | Already latest |
| buf.build/protovalidate | (current) | (current) | No change needed |

### New Dependencies for Generated Client Code

These are NOT dependencies of sebuf itself. They are libraries that GENERATED client code will use/import. sebuf's go.mod is unchanged.

| Language | Library | Recommended Version | Confidence |
|----------|---------|-------------------|------------|
| **Python** | httpx | >= 0.28.0 | HIGH |
| **Rust** | reqwest | >= 0.13.0 | HIGH |
| **Rust** | serde + serde_json | >= 1.0 | HIGH |
| **Rust** | tokio | >= 1.0 | HIGH |
| **Swift** | URLSession (Foundation) | iOS 15+ / macOS 12+ | HIGH |
| **Kotlin** | OkHttp | >= 5.0 | HIGH |
| **Kotlin** | kotlinx.serialization | >= 1.10.0 | HIGH |
| **Java** | java.net.http (stdlib) | Java 11+ | HIGH |
| **C#** | System.Net.Http + System.Text.Json | .NET 6+ | HIGH |
| **Ruby** | faraday | >= 2.0 | HIGH |
| **Dart** | package:http | >= 1.0 | HIGH |

### Protobuf Runtimes (Optional, for Users Who Want Proto-Native Types)

| Language | Library | Latest Version | Notes |
|----------|---------|---------------|-------|
| Python | google-protobuf | 5.x | Not needed for basic HTTP client |
| Python | betterproto2 | 0.9.1 | Alternative; in beta |
| Rust | prost | 0.14.2 | Only if --with-prost flag |
| Rust | pbjson-build | 0.8.0 | Serde impls for prost types |
| Swift | swift-protobuf | 1.33.3 | Only if proto binary needed |
| Kotlin | protobuf-kotlin | (matches protobuf-java) | JVM only |
| Java | protobuf-java | 4.33.3 | Only if proto binary needed |
| C# | Google.Protobuf | 3.33.5 | Only if proto binary needed |
| Ruby | google-protobuf | 4.33.x | Only if proto binary needed |
| Dart | package:protobuf | 5.2.0 | Only if proto binary needed |

---

## Summary of Key Decisions

| # | Decision | Confidence |
|---|----------|------------|
| 1 | Use annotation-driven code generation, not runtime configuration, for JSON mapping | HIGH |
| 2 | Layer custom JSON features on top of protojson, do not replace it | HIGH |
| 3 | Generate standalone language-native types (no protobuf runtime dependency by default) | HIGH |
| 4 | Use programmatic string emission (g.P()), not templates, for all generators | HIGH |
| 5 | Use platform-standard HTTP libraries (URLSession, HttpClient, httpx, reqwest, etc.) | HIGH |
| 6 | Start Kotlin with OkHttp/JVM-only, defer KMP/Ktor to later | MEDIUM |
| 7 | Start Rust with pure serde structs, defer prost integration to later | MEDIUM |
| 8 | Java: use java.net.http stdlib, minimal JSON dependency | MEDIUM |
| 9 | Enforce cross-generator consistency via golden files + wire-format integration tests | HIGH |
| 10 | Each JSON mapping annotation needs a formal specification document | HIGH |

---

## Sources

- [gRPC-Gateway: Customizing Your Gateway](https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_your_gateway/)
- [Connect: Serialization & Compression](https://connectrpc.com/docs/go/serialization-and-compression/)
- [Twirp: Serialization Schemes](https://twitchtv.github.io/twirp/docs/proto_and_json.html)
- [ts-proto GitHub](https://github.com/stephenh/ts-proto)
- [ProtoJSON Format Specification](https://protobuf.dev/programming-guides/json/)
- [protojson Go Package](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson)
- [Protobuf Field Presence](https://protobuf.dev/programming-guides/field_presence/)
- [Protobuf Well-Known Types](https://protobuf.dev/reference/protobuf/google.protobuf/)
- [protobuf-go Releases](https://github.com/protocolbuffers/protobuf-go/releases)
- [Connect-ES 2.0 Announcement](https://buf.build/blog/connect-es-v2)
- [Protobuf-ES v2.0 Announcement](https://buf.build/blog/protobuf-es-v2)
- [@bufbuild/protobuf on npm](https://www.npmjs.com/package/@bufbuild/protobuf)
- [prost on crates.io](https://crates.io/crates/prost)
- [reqwest on crates.io](https://crates.io/crates/reqwest)
- [pbjson on GitHub](https://github.com/influxdata/pbjson)
- [swift-protobuf on GitHub](https://github.com/apple/swift-protobuf)
- [Connect-Swift Announcement](https://buf.build/blog/announcing-connect-swift)
- [Ktor 3.4.0 Release](https://blog.jetbrains.com/kotlin/2026/01/ktor-3-4-0-is-now-available/)
- [OkHttp on GitHub](https://github.com/square/okhttp)
- [kotlinx.serialization Releases](https://github.com/Kotlin/kotlinx.serialization/releases)
- [Google.Protobuf on NuGet](https://www.nuget.org/packages/google.protobuf/)
- [protobuf-java on Maven Central](https://mvnrepository.com/artifact/com.google.protobuf/protobuf-java)
- [Faraday on RubyGems](https://rubygems.org/gems/faraday)
- [httpx on PyPI](https://pypi.org/project/httpx/)
- [betterproto2 on PyPI](https://pypi.org/project/betterproto2/)
- [Dart http Package on pub.dev](https://pub.dev/packages/http)
- [Dart protoc_plugin on pub.dev](https://pub.dev/packages/protoc_plugin)
- [Buf Code Generation Docs](https://buf.build/docs/generate/)
- [protoc-gen-prost-serde on crates.io](https://crates.io/crates/protoc-gen-prost-serde)
- [HTTPX vs Requests Comparison](https://www.morethanmonkeys.co.uk/article/comparing-requests-and-httpx-in-python-which-http-client-should-you-use-in-2025/)
- [Protobuf Editions Overview](https://protobuf.dev/editions/overview/)
