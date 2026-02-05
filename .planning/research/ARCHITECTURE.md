# Architecture Research: JSON Mapping Annotations and Multi-Language Generator Structure

## 1. Current Architecture Analysis

### 1.1 Plugin Structure (As-Is)

Each of the four generators lives in its own `internal/` package with a separate `cmd/` entry point:

```
cmd/protoc-gen-go-http/main.go      --> internal/httpgen/
cmd/protoc-gen-go-client/main.go    --> internal/clientgen/
cmd/protoc-gen-ts-client/main.go    --> internal/tsclientgen/
cmd/protoc-gen-openapiv3/main.go    --> internal/openapiv3/
```

Each generator package has its own `annotations.go` file that independently extracts protobuf extension options from `proto/sebuf/http/*.proto`. The generated Go types for these extensions live in the `http/` runtime package (e.g., `http.E_Config`, `http.E_ServiceHeaders`).

### 1.2 Measured Duplication

Annotation parsing is duplicated across all four generators -- 1,289 total lines across four `annotations.go` files:

| Generator | File | Lines | Duplicated Functions |
|-----------|------|-------|---------------------|
| httpgen | `internal/httpgen/annotations.go` | 392 | getMethodHTTPConfig, httpMethodToString, extractPathParams, getServiceHTTPConfig, getServiceHeaders, getMethodHeaders, getQueryParams, hasUnwrapAnnotation, getUnwrapField, getFieldExamples |
| clientgen | `internal/clientgen/annotations.go` | 241 | getMethodHTTPConfig, httpMethodToString, extractPathParams, getServiceHTTPConfig, getServiceHeaders, getMethodHeaders, getQueryParams |
| tsclientgen | `internal/tsclientgen/annotations.go` | 250 | getMethodHTTPConfig, httpMethodToString, extractPathParams, getServiceHTTPConfig, getServiceHeaders, getMethodHeaders, getQueryParams, hasUnwrapAnnotation |
| openapiv3 | `internal/openapiv3/http_annotations.go` | 406 | getMethodHTTPConfig, httpMethodToString, extractPathParams, getServiceHTTPConfig, getServiceHeaders, getMethodHeaders, getQueryParams, combineHeaders, convertHeadersToParameters, hasUnwrapAnnotation |

The functions `getMethodHTTPConfig`, `httpMethodToString`, `extractPathParams`, `getServiceHTTPConfig`, `getServiceHeaders`, `getMethodHeaders`, and `getQueryParams` are copy-pasted across all four packages with near-identical implementations. The only differences are:

1. **Return types for QueryParam**: Each generator uses a slightly different `QueryParam` struct (httpgen uses `FieldGoName`, clientgen adds `FieldKind`, tsclientgen uses `FieldJSONName` + `FieldKind`, openapiv3 carries `*protogen.Field` directly).
2. **HTTP method casing**: httpgen/clientgen/tsclientgen use uppercase ("GET"), openapiv3 uses lowercase ("get").
3. **Unwrap handling**: httpgen has full unwrap validation (`getUnwrapField` with error return), tsclientgen and openapiv3 have simpler `hasUnwrapAnnotation` checks, clientgen has none.

### 1.3 Data Flow (As-Is)

```
Proto Definitions (proto/sebuf/http/*.proto)
         |
         v
Generated Go Types (http/*.pb.go)    <-- proto extension descriptors
         |
         v
Each Generator's annotations.go      <-- duplicated extraction logic
         |
         v
Generator-specific types (HTTPConfig, QueryParam variants)
         |
         v
Code generation output (Go handlers, Go client, TS client, OpenAPI spec)
```

### 1.4 Additional Duplication

Beyond annotations, there are also duplicated helper functions:

- `lowerFirst` is defined in httpgen, clientgen, and tsclientgen
- `getMapValueField` is defined in both httpgen/unwrap.go and openapiv3/types.go
- `hasUnwrapAnnotation` is defined in httpgen, tsclientgen, and openapiv3
- Path building logic exists in clientgen, tsclientgen, and openapiv3

---

## 2. Proposed Architecture for JSON Mapping Layer

### 2.1 New Shared Package: `internal/annotations`

Create a single shared package that owns all annotation extraction logic. All generators import from this package instead of maintaining their own copies.

```
internal/annotations/
    annotations.go     -- HTTPConfig, ServiceConfig extraction
    headers.go         -- ServiceHeaders, MethodHeaders extraction
    query.go           -- QueryParam extraction (unified struct)
    unwrap.go          -- Unwrap annotation extraction and validation
    jsonmapping.go     -- New JSON mapping annotations (nullable, timestamps, etc.)
    helpers.go         -- Shared utilities (pathParamRegex, lowerFirst, etc.)
    annotations_test.go
```

#### 2.1.1 Unified Data Types

The shared package defines canonical data types that generators consume:

```go
package annotations

// HTTPConfig is the parsed HTTP configuration for an RPC method.
type HTTPConfig struct {
    Path       string
    Method     string   // Always uppercase: "GET", "POST", etc.
    PathParams []string
}

// ServiceConfig is the parsed HTTP configuration for a service.
type ServiceConfig struct {
    BasePath string
}

// QueryParam is the parsed query parameter configuration.
// Contains all fields any generator might need.
type QueryParam struct {
    ProtoFieldName string              // "user_id"
    GoFieldName    string              // "UserId"
    JSONFieldName  string              // "userId"
    ParamName      string              // URL query param name
    Required       bool
    FieldKind      string              // "string", "int32", etc.
    Field          *protogen.Field     // Full field reference for generators that need it
}

// UnwrapInfo is the parsed unwrap annotation for a field.
type UnwrapInfo struct {
    Field        *protogen.Field
    ElementType  *protogen.Message
    IsRootUnwrap bool
    IsMapField   bool
}
```

Each generator picks the subset of fields it needs. The openapiv3 generator uses `Field` for schema generation; tsclientgen uses `JSONFieldName` and `FieldKind`; clientgen uses `GoFieldName` and `FieldKind`.

#### 2.1.2 JSON Mapping Annotation Types

New annotations for v1.0 JSON mapping features will be defined in `proto/sebuf/http/annotations.proto` (extending FieldOptions, MessageOptions, and FileOptions) and parsed in `internal/annotations/jsonmapping.go`:

```go
// NullableConfig controls null vs absent vs default semantics.
type NullableConfig struct {
    Behavior NullBehavior // OMIT_DEFAULT, NULL_DEFAULT, ALWAYS_PRESENT
}

// TimestampConfig controls timestamp serialization format.
type TimestampConfig struct {
    Format TimestampFormat // RFC3339, UNIX_SECONDS, UNIX_MILLIS, DATE_ONLY
}

// FieldCasingConfig controls JSON field name casing.
type FieldCasingConfig struct {
    Style CasingStyle // CAMEL_CASE, SNAKE_CASE, PASCAL_CASE, KEBAB_CASE
    Name  string      // Explicit override
}

// BytesEncodingConfig controls bytes field encoding.
type BytesEncodingConfig struct {
    Encoding BytesEncoding // BASE64, BASE64URL, HEX
}

// OneofConfig controls oneof serialization strategy.
type OneofConfig struct {
    Style    OneofStyle // FLATTENED (discriminated union) vs WRAPPED
    TypeField string    // Discriminator field name
}

// EnumEncodingConfig controls enum string encoding.
type EnumEncodingConfig struct {
    Style EnumStyle // NAME (default), NUMBER, CUSTOM
}
```

### 2.2 Component Boundaries

```
                   Proto Definitions
                   (proto/sebuf/http/*.proto)
                          |
                          v
                   Generated Go Types
                   (http/*.pb.go)
                          |
                          v
               +---------------------+
               | internal/annotations|   <-- SINGLE source of truth
               | (shared parsing)    |       for annotation extraction
               +---------------------+
                    |    |    |    |
         +------+  |    |    |  +-------+
         |      |  |    |    |  |       |
         v      v  v    v    v  v       v
    httpgen  clientgen  tsclientgen  openapiv3   [future: pyclientgen, ...]
    (server)  (Go client) (TS client) (OpenAPI)
```

**Boundary rules:**

1. `internal/annotations` knows about `protogen` types and `http` extension types. It does NOT know about any generator's output format.
2. Each generator imports `internal/annotations` for parsed data. Generators never import `http.E_Config` directly -- they go through the shared package.
3. Each generator owns its own output format logic (Go code generation, TypeScript code generation, YAML/JSON schemas).
4. The `proto/sebuf/http/` directory owns all annotation definitions. The `http/` runtime package owns the generated Go types for these protos.

### 2.3 OpenAPI-specific Annotation Consumers

The openapiv3 generator has additional annotation consumers (e.g., `combineHeaders`, `convertHeadersToParameters`, validation constraint extraction). These should remain in `internal/openapiv3/` because they produce OpenAPI-specific output types (libopenapi's `v3.Parameter`, `base.Schema`). The shared `internal/annotations` only handles extraction from protobuf descriptors -- not conversion to output formats.

Similarly, `internal/httpgen/unwrap.go` contains Go code generation logic for MarshalJSON/UnmarshalJSON methods. The unwrap detection logic (`collectUnwrapContext`, `collectAllUnwrapFields`) should move to `internal/annotations`, but the code generation methods stay in httpgen.

---

## 3. Multi-Language Generator Structure

### 3.1 Pattern for New Language Generators

Each new language generator follows the established pattern:

```
cmd/protoc-gen-{lang}-client/main.go     -- Entry point (thin)
internal/{lang}clientgen/
    generator.go         -- Main Generator struct, file iteration
    types.go             -- Language-specific type mapping (proto -> lang types)
    helpers.go           -- Naming conventions (snake_case, camelCase, PascalCase)
    golden_test.go       -- Golden file tests
    testdata/
        proto/           -- Test proto files
        golden/          -- Expected output files
```

All generators import `internal/annotations` for annotation parsing. They never duplicate annotation extraction logic.

### 3.2 What Is Shared vs. Per-Language

**Shared (in `internal/annotations`):**
- All annotation extraction (HTTP config, headers, query params, unwrap, JSON mapping)
- Validation of annotation correctness (e.g., unwrap on non-repeated fields)
- Canonical data types for parsed annotations

**Per-language (in `internal/{lang}clientgen`):**
- Type mapping: `protoreflect.Kind` -> language-specific type strings
- Naming conventions: proto field names -> language-idiomatic names
- Code generation: producing the actual source file content
- Error type generation: language-specific error classes/structs
- Client class/struct generation: constructor, method calls, options pattern
- Test infrastructure: golden files specific to that language's output

### 3.3 No Shared Code Generation Template

Each language generator writes its output differently. Go generators use `protogen.GeneratedFile.P()` (which handles imports automatically). TypeScript uses a `printer` function. OpenAPI uses libopenapi structs rendered to YAML/JSON. New languages will use string building appropriate to their output format.

Attempting to share code generation templates across languages would create coupling without benefit, since each language has fundamentally different syntax, conventions, and packaging.

---

## 4. Build Order

### 4.1 Foundation Phase (Do First)

These changes establish the shared infrastructure. They should be done before any new JSON mapping features.

| Step | Task | Rationale |
|------|------|-----------|
| 1 | Create `internal/annotations/` package with canonical types | Foundation for all subsequent work |
| 2 | Move `getMethodHTTPConfig`, `getServiceHTTPConfig`, `extractPathParams`, `httpMethodToString` to shared package | These are 100% identical across all 4 generators |
| 3 | Move `getServiceHeaders`, `getMethodHeaders` to shared package | Also 100% identical |
| 4 | Unify `getQueryParams` with a single `QueryParam` struct that has all needed fields | Currently 4 different struct definitions |
| 5 | Move `hasUnwrapAnnotation` and core unwrap detection to shared package | Currently in httpgen, tsclientgen, openapiv3 |
| 6 | Move shared helpers (`lowerFirst`, `getMapValueField`, path building) to shared package | Eliminate remaining utility duplication |
| 7 | Update all 4 generators to import from `internal/annotations` | Remove all `annotations.go` files from individual generators |
| 8 | Update all golden files (`UPDATE_GOLDEN=1`) | Generated output should not change -- this is a pure refactor |

**Expected outcome**: 1,289 lines of duplicated annotation code across 4 files reduced to ~300 lines in one shared package, with 4 thin import wrappers (or no wrappers at all).

**Risk mitigation**: Run golden file tests after each step to verify no output changes. This is a pure refactoring -- no behavior change.

### 4.2 JSON Mapping Features Phase (v1.0 Features)

After the shared annotation package is in place, add JSON mapping annotations one at a time. Each feature follows the same pattern:

1. Define the annotation in `proto/sebuf/http/annotations.proto`
2. Generate Go types (`make proto`)
3. Add extraction logic in `internal/annotations/jsonmapping.go`
4. Implement in all 4 generators simultaneously
5. Add golden file test cases for each generator

**Recommended feature order** (based on dependencies and complexity):

| Priority | Feature | Issue | Why This Order |
|----------|---------|-------|----------------|
| 1 | Fix conditional imports (#105) | #105 | Quick fix, unblocks PR merges |
| 2 | Land cross-file unwrap (#98) | PR #98 | Already in PR, validates shared package approach |
| 3 | Nullable primitives | #87 | Fundamental -- affects how all other features handle zero/null |
| 4 | int64/uint64 as string | #88 | Simple, well-defined -- changes one type mapping per generator |
| 5 | Enum string encoding | #89 | Builds on type mapping, well-defined scope |
| 6 | Field name casing options | #94 | Affects JSON field names throughout, should come before complex features |
| 7 | Empty object handling | #93 | Related to nullable, defines baseline behavior |
| 8 | Bytes encoding options | #95 | Simple, isolated to bytes fields |
| 9 | Timestamp formats | #92 | Well-defined formats, touches type system |
| 10 | Oneof as discriminated union | #90 | Most complex -- requires structural changes to generated output |
| 11 | Nested message flattening | #96 | Most invasive -- changes message structure in output |
| 12 | Root-level arrays (#91) | #91 | Verify coverage via existing unwrap, may be already done |

### 4.3 Multi-Language Generators Phase (v2.0)

New language generators are independent of each other and can be developed in parallel. However, the JSON mapping foundation (Phase 4.2) must be complete first, because each new generator must implement all JSON mapping features from day one.

**Recommended starter order** (based on ecosystem value and complexity):

| Priority | Language | Rationale |
|----------|----------|-----------|
| 1 | Python | Largest API consumer base, simple type system |
| 2 | Kotlin | Android market, coroutine-based async aligns with RPC pattern |
| 3 | Swift | iOS market, complements Kotlin for full mobile coverage |
| 4 | Rust | Growing systems ecosystem, strict type system validates design |
| 5 | Java | Enterprise market, shares patterns with Kotlin |
| 6 | C# | .NET ecosystem, async/await pattern |
| 7 | Dart | Flutter market, similar to TypeScript in pattern |
| 8 | Ruby | Web ecosystem, dynamic typing |

Each generator should be developed as:
1. Scaffold: cmd entry point + empty generator struct
2. Type mapping: proto types -> language types
3. Basic client: constructor, single method, no options
4. Full client: all annotations, headers, query params, error handling
5. JSON mapping: all v1.0 JSON mapping features
6. Golden file tests: comprehensive test coverage

---

## 5. Impact on Existing Tests

### 5.1 Golden File Tests Are the Safety Net

The project's golden file tests (`UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles`) capture the exact output of each generator. During the refactoring phase (4.1), golden files must NOT change -- any change means the refactoring introduced a bug.

During the feature phase (4.2), golden files will be updated to include new annotation behaviors. Each new test proto file should exercise the specific JSON mapping feature being added.

### 5.2 Test Strategy for Shared Package

`internal/annotations` should have its own unit tests that verify annotation extraction against mock protogen structures. These tests complement (not replace) the golden file tests in each generator.

---

## 6. File-Level Impact Map

### Files Created

| New File | Purpose |
|----------|---------|
| `internal/annotations/annotations.go` | HTTPConfig, ServiceConfig extraction |
| `internal/annotations/headers.go` | ServiceHeaders, MethodHeaders extraction |
| `internal/annotations/query.go` | Unified QueryParam extraction |
| `internal/annotations/unwrap.go` | Unwrap detection and validation |
| `internal/annotations/jsonmapping.go` | New JSON mapping annotation parsing |
| `internal/annotations/helpers.go` | Shared utilities |
| `internal/annotations/annotations_test.go` | Unit tests |
| `proto/sebuf/http/annotations.proto` | Extended with JSON mapping options |

### Files Removed

| Removed File | Replaced By |
|-------------|-------------|
| `internal/httpgen/annotations.go` | `internal/annotations/` (extraction only; httpgen-specific types like UnwrapFieldInfo may stay or move) |
| `internal/clientgen/annotations.go` | `internal/annotations/` |
| `internal/tsclientgen/annotations.go` | `internal/annotations/` |

### Files Modified

| Modified File | Change |
|--------------|--------|
| `internal/openapiv3/http_annotations.go` | Remove duplicated extraction functions, keep OpenAPI-specific converters (`combineHeaders`, `convertHeadersToParameters`, `mapHeaderTypeToOpenAPI`) |
| `internal/httpgen/generator.go` | Import from `internal/annotations` |
| `internal/httpgen/unwrap.go` | Import shared unwrap detection, keep Go code generation |
| `internal/httpgen/validation.go` | Import shared helpers |
| `internal/clientgen/generator.go` | Import from `internal/annotations` |
| `internal/tsclientgen/generator.go` | Import from `internal/annotations` |
| `internal/tsclientgen/types.go` | Import shared `hasUnwrapAnnotation` |
| `internal/openapiv3/generator.go` | Import from `internal/annotations` |
| `internal/openapiv3/types.go` | Import shared `hasUnwrapAnnotation`, `getMapValueField` |

---

## 7. Key Architectural Decisions

### 7.1 Shared Package vs. Interface-Based Abstraction

**Decision**: Use a shared package with concrete types, not an interface/plugin architecture.

**Rationale**: All generators run as separate protoc plugins (separate OS processes). They cannot share code at runtime via interfaces. The shared package is a compile-time dependency only. An interface-based abstraction would add complexity without enabling runtime composition.

### 7.2 Unified QueryParam vs. Generator-Specific Wrappers

**Decision**: Single `QueryParam` struct with all fields, consumed selectively by each generator.

**Rationale**: The alternative (shared extraction returning a minimal type, with each generator wrapping it) adds an unnecessary translation layer. A single struct with fields like `GoFieldName`, `JSONFieldName`, `FieldKind`, and `Field` is simpler. Generators ignore fields they don't need.

### 7.3 HTTP Method Case Convention

**Decision**: Store as uppercase in shared package ("GET", "POST"). OpenAPI generator lowercases on output.

**Rationale**: Uppercase matches Go's `net/http` constants and is the convention in httpgen, clientgen, and tsclientgen. Only OpenAPI needs lowercase, and that's an output format concern.

### 7.4 JSON Mapping Annotations as Proto Options

**Decision**: All new JSON mapping annotations are proto field/message/file options in `proto/sebuf/http/annotations.proto`, not generator parameters.

**Rationale**: Proto-level annotations are portable across all generators, visible in the proto source, and version-controlled with the API definition. Generator parameters (like `format=json` for OpenAPI) are for output format concerns, not API semantics.

### 7.5 Annotation Proto Location

**Decision**: Keep all annotations in `proto/sebuf/http/annotations.proto` (one file), rather than splitting across multiple proto files per feature.

**Rationale**: The annotations.proto file is manageable at its current size (74 lines) and will grow to perhaps 200 lines with all JSON mapping options. Users import a single proto path. Splitting would complicate imports for marginal organizational benefit.

---

## 8. Summary

The core architectural insight is that annotation parsing is a cross-cutting concern that belongs in a shared package, while code generation output is generator-specific. The `internal/annotations` package forms the stable foundation that all current and future generators build on.

The build order is: (1) extract shared annotation package from existing duplicated code, (2) add JSON mapping features using the shared package, (3) build new language generators that import the shared package from day one. Each phase validates the work of the previous phase -- the shared package is proven correct by golden file tests before new features build on it.
