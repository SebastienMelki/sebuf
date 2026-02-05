# Phase 2: Foundation - Shared Annotations - Research

**Researched:** 2026-02-05
**Domain:** Go code refactoring -- extracting duplicated annotation parsing across 4 protoc plugin generators
**Confidence:** HIGH

## Summary

This research examines the exact duplication across the four generators (httpgen, clientgen, tsclientgen, openapiv3) to determine what belongs in a shared `internal/annotations` package and how to design it for zero behavior change.

The duplication is confirmed at exactly 1,289 lines across the 4 annotation files, with additional annotation-related code in `openapiv3/types.go` (~80 lines of `hasUnwrapAnnotation`, `getUnwrapField`, `getFieldExamples`). All four generators follow the same pattern: extract protobuf extension options using `proto.GetExtension`, cast to the appropriate type, and return a domain struct. The key challenge is that each generator uses slightly different domain structs (`QueryParam` has different fields per generator, `ServiceConfigImpl` vs `ServiceHTTPConfig` naming, HTTP methods are uppercase in 3 generators but lowercase in openapiv3).

The serialization audit is narrowly scoped: the `encoding/json` import in httpgen's generated binding code is legitimate (used for `json.Marshaler`/`json.Unmarshaler` interface checks on unwrap types), while `protojson` is correctly used for actual proto message serialization. The unwrap code in `unwrap.go` correctly uses both `encoding/json` for structural marshaling and `protojson` for proto message marshaling. No incorrect `encoding/json` usage for proto message serialization was found.

**Primary recommendation:** Create `internal/annotations/` with one file per annotation type following `GetXxx(*protogen.Type) *ReturnType` convention. Use transparent structs. Accept `*protogen.Method`, `*protogen.Service`, `*protogen.Message`, `*protogen.Field` (protogen types, not protoreflect). Return the proto types directly for headers (`[]*http.Header`) since all 4 generators already use them identically.

## Standard Stack

No new libraries needed. This is a pure refactoring within the existing codebase.

### Core Dependencies (Already in go.mod)
| Library | Version | Purpose | Role in Shared Package |
|---------|---------|---------|------------------------|
| `google.golang.org/protobuf/compiler/protogen` | v1.36.11 | protoc plugin framework | Input types for all Get functions |
| `google.golang.org/protobuf/proto` | v1.36.11 | proto extension access | `proto.GetExtension` calls |
| `google.golang.org/protobuf/types/descriptorpb` | v1.36.11 | descriptor option types | Type assertions for options |
| `github.com/SebastienMelki/sebuf/http` | (local) | Annotation proto types | Extension variables and message types |

### Not Needed in Shared Package
| Library | Why Not |
|---------|---------|
| `github.com/pb33f/libopenapi` | OpenAPI-specific, stays in openapiv3 |
| `buf.build/.../protovalidate` | Validation-specific, stays in openapiv3 and httpgen |

## Architecture Patterns

### Recommended Package Structure

```
internal/annotations/
  http_config.go       # GetMethodHTTPConfig, GetServiceBasePath, HTTPConfig, ServiceConfig structs
  headers.go           # GetServiceHeaders, GetMethodHeaders, CombineHeaders
  query.go             # GetQueryParams, QueryParamBase struct
  unwrap.go            # HasUnwrapAnnotation, GetUnwrapField, UnwrapFieldInfo, UnwrapValidationError
  field_examples.go    # GetFieldExamples
  path.go              # ExtractPathParams, BuildHTTPPath, pathParamRegex
  method.go            # HTTPMethodToString (uppercase), HTTPMethodToLower (for OpenAPI)
  doc.go               # Package documentation
```

### Pattern: Convention-Based Extensibility

Each file follows the same convention:
1. A `Get`-prefixed exported function that accepts a protogen type
2. Returns a struct or proto type
3. Handles nil/missing options gracefully (returns nil, not error)
4. One file = one annotation concept

When Phases 4-7 add new annotations, the pattern is: create a new `.go` file, define a `GetXxx` function following the same shape. No interfaces to implement, no registration -- just files with functions.

### Pattern: Struct Design for QueryParam

The `QueryParam` struct varies across generators:

| Field | httpgen | clientgen | tsclientgen | openapiv3 |
|-------|---------|-----------|-------------|-----------|
| FieldName | yes | yes | yes | yes |
| FieldGoName | yes | yes | no | no |
| FieldJSONName | no | no | yes | no |
| ParamName | yes | yes | yes | yes |
| Required | yes | yes | yes | yes |
| FieldKind | no | yes | yes | no |
| Field (*protogen.Field) | no | no | no | yes |

**Recommendation:** Use a base struct with all fields that any generator needs. Each generator already selects what it uses. The shared struct should include ALL fields -- generators ignore what they don't need. Include the raw `*protogen.Field` reference so any generator can derive whatever it needs.

```go
// QueryParam represents a query parameter extracted from a proto field annotation.
type QueryParam struct {
    FieldName     string          // Proto field name (e.g., "page_number")
    FieldGoName   string          // Go field name (e.g., "PageNumber")
    FieldJSONName string          // JSON field name (e.g., "pageNumber")
    ParamName     string          // Query parameter name from annotation
    Required      bool            // Whether this parameter is required
    FieldKind     string          // Proto field kind (e.g., "string", "int32")
    Field         *protogen.Field // Raw field reference for generator-specific needs
}
```

### Pattern: HTTP Method Case Handling

Three generators use UPPERCASE ("GET", "POST"), openapiv3 uses lowercase ("get", "post"). The shared package should:
- Store as uppercase (standard HTTP convention, matches `net/http` constants)
- Provide `HTTPMethodToString(m http.HttpMethod) string` returning uppercase
- Openapiv3 calls `strings.ToLower()` on the result (1 line in caller, not worth a second function)

Actually, on reflection, providing both avoids the caller needing to know this. A simple helper:

```go
func HTTPMethodToString(m http.HttpMethod) string { ... }     // "GET", "POST", etc.
func HTTPMethodToLower(m http.HttpMethod) string { ... }       // "get", "post", etc. (for OpenAPI)
```

### Pattern: BuildHTTPPath Consolidation

Three slightly different path-building implementations exist:
1. **httpgen**: Custom logic in `generator.go` lines 770-786 (uses `camelToSnake` for default paths, not relevant to shared)
2. **clientgen**: Inline in `buildRPCMethodConfig` (lines 425-433) -- `TrimSuffix` + `HasPrefix` + concatenation
3. **tsclientgen**: Identical to clientgen (lines 281-288)
4. **openapiv3**: `buildHTTPPath` function (lines 182-200) with `ensureLeadingSlash` helper

The clientgen and tsclientgen path-building logic is identical. The openapiv3 version is slightly more defensive. **Recommendation:** Extract `BuildHTTPPath(basePath, methodPath string) string` using the openapiv3 implementation (most robust), but keep generator-specific default-path logic (like `camelToSnake` fallbacks) in each generator.

### Anti-Patterns to Avoid

- **Generator-specific logic in shared package:** The shared package must ONLY contain annotation extraction. Generator-specific transformations (like openapiv3's `convertHeadersToParameters` or `mapHeaderTypeToOpenAPI`) stay in their generators.
- **Breaking the function signature convention:** All exported functions should accept protogen types and return simple types. No `interface{}` or complex option patterns.
- **Abstracting too much:** Do NOT create an "annotation registry" or "annotation interface." Keep it as simple functions.

## Don't Hand-Roll

Problems with existing solutions that should be used:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Proto extension extraction | Custom reflection | `proto.GetExtension` | Already used, battle-tested |
| Path param regex | Per-generator regex | Single compiled `pathParamRegex` | Identical regex in all 4 generators |
| Header merging | N/A | `CombineHeaders` in shared | Currently only in openapiv3, but all generators will need it for consistency |

**Key insight:** The annotation extraction pattern is identical across all 4 generators -- the only differences are in the return types. The shared package eliminates copy-paste drift.

## Common Pitfalls

### Pitfall 1: Breaking the function call graph

**What goes wrong:** Renaming a function or changing its signature breaks callers silently at first (since each generator had its own copy).
**Why it happens:** The migration changes from package-local calls to cross-package calls.
**How to avoid:** After each generator migration, run `make lint-fix && go build ./...` before running tests. The compiler catches all call-site mismatches.
**Warning signs:** Build failures mentioning undefined functions.

### Pitfall 2: QueryParam struct field mismatch

**What goes wrong:** A generator accesses a field that existed in its local struct but wasn't populated by the shared function.
**Why it happens:** The four `QueryParam` structs have different fields. If the shared function doesn't populate all fields, generators break.
**How to avoid:** The shared `GetQueryParams` must populate ALL fields in the unified struct: `FieldName`, `FieldGoName`, `FieldJSONName`, `ParamName`, `Required`, `FieldKind`, and `Field`. Each generator uses a subset.
**Warning signs:** Empty strings or zero values where generators expect data.

### Pitfall 3: OpenAPI lowercase method mismatch

**What goes wrong:** OpenAPI generator starts producing uppercase HTTP methods ("GET" instead of "get") in YAML output.
**Why it happens:** The shared `HTTPMethodToString` returns uppercase, and the openapiv3 generator forgets to lowercase.
**How to avoid:** Either provide `HTTPMethodToLower` in the shared package, or ensure the openapiv3 migration explicitly lowercases. Golden file tests will catch this.
**Warning signs:** Golden file diff showing case changes in OpenAPI output.

### Pitfall 4: Exporting previously-unexported types

**What goes wrong:** The shared package must export types that were package-private in each generator.
**Why it happens:** Functions like `getMethodHTTPConfig` (lowercase) become `GetMethodHTTPConfig` (uppercase) in the shared package.
**How to avoid:** This is expected and correct. Update all call sites from `getMethodHTTPConfig(...)` to `annotations.GetMethodHTTPConfig(...)`.
**Warning signs:** None -- this is the normal migration pattern.

### Pitfall 5: Import cycle risk

**What goes wrong:** If the shared package accidentally imports a generator package, Go refuses to compile.
**Why it happens:** Shared package depends only on protogen and sebuf/http. This direction is safe. The risk is accidentally making a generator depend on another generator.
**How to avoid:** The shared package imports ONLY: `google.golang.org/protobuf/*`, `github.com/SebastienMelki/sebuf/http`, and stdlib. Nothing from `internal/httpgen`, `internal/clientgen`, etc.
**Warning signs:** Import cycle compiler errors.

### Pitfall 6: Test files referencing private annotation functions

**What goes wrong:** Test files in generators (e.g., `annotations_test.go`) call local package functions. After migration, those functions no longer exist locally.
**Why it happens:** Tests for `httpMethodToString` and `extractPathParams` are in `httpgen/annotations_test.go`.
**How to avoid:** Move annotation-specific tests to `internal/annotations/` test files. Generator test files that tested annotation parsing need to either: (a) import from `annotations` package, or (b) test the generator's usage through golden file tests (which already exist).
**Warning signs:** Test compilation failures after deleting old annotation code.

### Pitfall 7: openapiv3/types.go contains annotation code

**What goes wrong:** Migrating only `http_annotations.go` misses the `hasUnwrapAnnotation`, `getUnwrapField`, and `getFieldExamples` duplicated in `openapiv3/types.go`.
**Why it happens:** These annotation functions were placed in the types file instead of the annotations file.
**How to avoid:** Audit `types.go` (lines 242-298) and migrate those functions too. The non-annotation functions in `types.go` (like `convertField`, `convertScalarField`, `convertMapField`) stay in openapiv3.
**Warning signs:** Duplicate function definitions if both shared package and types.go define `hasUnwrapAnnotation`.

## Code Examples

### Shared HTTPConfig extraction (canonical pattern)

```go
// internal/annotations/http_config.go
package annotations

import (
    "google.golang.org/protobuf/compiler/protogen"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/descriptorpb"

    "github.com/SebastienMelki/sebuf/http"
)

// HTTPConfig represents the HTTP configuration for a method.
type HTTPConfig struct {
    Path       string
    Method     string   // "GET", "POST", "PUT", "DELETE", "PATCH" (uppercase)
    PathParams []string // Path variable names extracted from path
}

// ServiceConfig represents the HTTP configuration for a service.
type ServiceConfig struct {
    BasePath string
}

// GetMethodHTTPConfig extracts HTTP configuration from method options.
func GetMethodHTTPConfig(method *protogen.Method) *HTTPConfig {
    options := method.Desc.Options()
    if options == nil {
        return nil
    }

    methodOptions, ok := options.(*descriptorpb.MethodOptions)
    if !ok {
        return nil
    }

    ext := proto.GetExtension(methodOptions, http.E_Config)
    if ext == nil {
        return nil
    }

    httpConfig, ok := ext.(*http.HttpConfig)
    if !ok || httpConfig == nil {
        return nil
    }

    path := httpConfig.GetPath()

    return &HTTPConfig{
        Path:       path,
        Method:     HTTPMethodToString(httpConfig.GetMethod()),
        PathParams: ExtractPathParams(path),
    }
}

// GetServiceConfig extracts HTTP configuration from service options.
func GetServiceConfig(service *protogen.Service) *ServiceConfig {
    options := service.Desc.Options()
    if options == nil {
        return nil
    }

    serviceOptions, ok := options.(*descriptorpb.ServiceOptions)
    if !ok {
        return nil
    }

    ext := proto.GetExtension(serviceOptions, http.E_ServiceConfig)
    if ext == nil {
        return nil
    }

    serviceConfig, ok := ext.(*http.ServiceConfig)
    if !ok || serviceConfig == nil {
        return nil
    }

    return &ServiceConfig{
        BasePath: serviceConfig.GetBasePath(),
    }
}
```

### Shared QueryParam extraction (unified struct)

```go
// internal/annotations/query.go
package annotations

import (
    "google.golang.org/protobuf/compiler/protogen"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/descriptorpb"

    "github.com/SebastienMelki/sebuf/http"
)

// QueryParam represents a query parameter configuration extracted from a field.
type QueryParam struct {
    FieldName     string          // Proto field name (e.g., "page_number")
    FieldGoName   string          // Go field name (e.g., "PageNumber")
    FieldJSONName string          // JSON field name (e.g., "pageNumber")
    ParamName     string          // Query parameter name from annotation
    Required      bool            // Whether this parameter is required
    FieldKind     string          // Proto field kind (e.g., "string", "int32")
    Field         *protogen.Field // Raw field reference
}

// GetQueryParams extracts query parameter configurations from message fields.
func GetQueryParams(message *protogen.Message) []QueryParam {
    var params []QueryParam

    for _, field := range message.Fields {
        options := field.Desc.Options()
        if options == nil {
            continue
        }

        fieldOptions, ok := options.(*descriptorpb.FieldOptions)
        if !ok {
            continue
        }

        ext := proto.GetExtension(fieldOptions, http.E_Query)
        if ext == nil {
            continue
        }

        queryConfig, ok := ext.(*http.QueryConfig)
        if !ok || queryConfig == nil {
            continue
        }

        paramName := queryConfig.GetName()
        if paramName == "" {
            paramName = string(field.Desc.Name())
        }

        params = append(params, QueryParam{
            FieldName:     string(field.Desc.Name()),
            FieldGoName:   field.GoName,
            FieldJSONName: field.Desc.JSONName(),
            ParamName:     paramName,
            Required:      queryConfig.GetRequired(),
            FieldKind:     field.Desc.Kind().String(),
            Field:         field,
        })
    }

    return params
}
```

### Migration pattern for a generator (httpgen example)

```go
// Before (internal/httpgen/generator.go):
config := getMethodHTTPConfig(method)

// After (internal/httpgen/generator.go):
import "github.com/SebastienMelki/sebuf/internal/annotations"

config := annotations.GetMethodHTTPConfig(method)
// config.Path, config.Method, config.PathParams -- same fields, same types
```

### Unwrap shared types with validation errors

```go
// internal/annotations/unwrap.go
package annotations

// UnwrapFieldInfo contains information about an unwrap field in a message.
type UnwrapFieldInfo struct {
    Field        *protogen.Field   // The field with unwrap=true
    ElementType  *protogen.Message // The element type of the repeated field (if message type)
    IsRootUnwrap bool              // True if this is a root-level unwrap (single field in message)
    IsMapField   bool              // True if the unwrap field is a map (only for root unwrap)
}

// UnwrapValidationError represents an error in unwrap annotation validation.
type UnwrapValidationError struct {
    MessageName string
    FieldName   string
    Reason      string
}

func (e *UnwrapValidationError) Error() string {
    return "invalid unwrap annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// HasUnwrapAnnotation checks if a field has the unwrap=true annotation.
func HasUnwrapAnnotation(field *protogen.Field) bool { ... }

// GetUnwrapField returns the unwrap field info for a message, or nil if none exists.
// Returns an error if the annotation is invalid.
func GetUnwrapField(message *protogen.Message) (*UnwrapFieldInfo, error) { ... }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Copy annotation code per generator | Shared annotations package | Phase 2 (this phase) | Eliminates 1,289+ lines of duplication |
| Per-file unwrap resolution | Two-pass global unwrap (Phase 1) | 2026-02-05 | Cross-file unwrap already works, shared package preserves this |

**Not deprecated/outdated:** The `proto.GetExtension` API, `protogen` types, and `descriptorpb` option types are all stable in protobuf Go v1.36.11. No upcoming changes affect this work.

## Detailed Duplication Map

### Exact duplicates (identical logic, possibly different naming)

| Function | httpgen | clientgen | tsclientgen | openapiv3 | Notes |
|----------|---------|-----------|-------------|-----------|-------|
| `getMethodHTTPConfig` | annotations.go:46 | annotations.go:47 | annotations.go:47 | http_annotations.go:41 | Identical extraction, same return struct shape |
| `httpMethodToString` | annotations.go:79 | annotations.go:80 | annotations.go:78 | http_annotations.go:74 | openapiv3 returns lowercase; others uppercase |
| `extractPathParams` | annotations.go:101 | annotations.go:102 | annotations.go:97 | http_annotations.go:94 | Identical logic |
| `pathParamRegex` | annotations.go:23 | annotations.go:23 | annotations.go:23 | http_annotations.go:18 | Same compiled regex |
| `getServiceHTTPConfig` | annotations.go:117 | annotations.go:118 | annotations.go:113 | http_annotations.go:154 | openapiv3 uses `ServiceHTTPConfig` name |
| `getServiceHeaders` | annotations.go:146 | annotations.go:147 | annotations.go:140 | http_annotations.go:214 | Identical |
| `getMethodHeaders` | annotations.go:173 | annotations.go:174 | annotations.go:165 | http_annotations.go:241 | Identical |
| `getQueryParams` | annotations.go:237 | annotations.go:201 | annotations.go:190 | http_annotations.go:112 | Struct fields vary (see QueryParam table) |
| `hasUnwrapAnnotation` | annotations.go:287 | N/A | annotations.go:232 | types.go:253 | 3 copies |
| `getFieldExamples` | annotations.go:210 | N/A | N/A | types.go:274 | 2 copies |

### Partially duplicated (same concept, different implementations)

| Function | Where | Notes |
|----------|-------|-------|
| `getUnwrapField` | httpgen/annotations.go:316 (full validation) | Most complete -- returns `UnwrapFieldInfo` with error |
| `getUnwrapField` | openapiv3/types.go:243 (simple) | Simpler version -- returns `*protogen.Field` only |
| `findUnwrapField` | tsclientgen/types.go:212 | Another simple version -- returns `*protogen.Field` |
| `buildHTTPPath` | openapiv3/http_annotations.go:182 | Full function with `ensureLeadingSlash` |
| (inline path building) | clientgen/generator.go:425-433 | Inline equivalent |
| (inline path building) | tsclientgen/generator.go:281-288 | Identical to clientgen inline |
| `combineHeaders` | openapiv3/http_annotations.go:267 | Only in openapiv3 currently |
| `lowerFirst` | httpgen, clientgen, tsclientgen | 3 copies, identical |

### Not duplicated (generator-specific, stays in generator)

| Function | Package | Why Not Shared |
|----------|---------|----------------|
| `convertHeadersToParameters` | openapiv3 | OpenAPI-specific conversion |
| `mapHeaderTypeToOpenAPI` | openapiv3 | OpenAPI-specific mapping |
| `extractValidationConstraints` | openapiv3 | buf.validate-specific, uses libopenapi types |
| `ValidateMethodConfig` | httpgen | HTTP handler-specific validation logic |
| `collectUnwrapContext` | httpgen | Generator-specific unwrap orchestration |
| `generateUnwrapFile` | httpgen | Code generation logic |
| `camelToSnake` | httpgen | Generator-specific path formatting |
| `snakeToLowerCamel` | tsclientgen | TS-specific naming |
| `headerNameToPropertyName` | tsclientgen | TS-specific naming |
| `parseExistingAnnotation` | httpgen | Legacy/temporary, always returns "" |

## Serialization Audit Findings

**FOUND-04 scope:** Audit `encoding/json` vs `protojson` consistency in HTTP handler generation.

### Current usage in httpgen generated code:

1. **Binding file (`_http_binding.pb.go`):** Uses `encoding/json` for:
   - `json.Unmarshaler` interface check (line 369): Correct -- checks if message implements custom unmarshaler (unwrap types do)
   - `json.Marshaler` interface check (line 634): Correct -- checks if message implements custom marshaler (unwrap types do)
   - `protojson.Unmarshal`/`protojson.Marshal`: Used for all standard proto message (de)serialization

2. **Unwrap file (`_unwrap.pb.go`):** Uses both:
   - `encoding/json`: For structural marshaling (building `map[string]json.RawMessage`, marshaling arrays of `json.RawMessage`)
   - `protojson`: For individual proto message marshaling within the unwrap structure

**Assessment:** The current usage is CORRECT. The `encoding/json` import is required because:
- Generated unwrap types implement `json.Marshaler`/`json.Unmarshaler` interfaces
- The binding middleware correctly checks for these interfaces before falling back to `protojson`
- Structural JSON (maps, arrays of raw messages) uses `encoding/json`
- Proto message serialization uses `protojson`

**No changes needed for FOUND-04.** The usage is consistent and intentional. The only action item is to document this pattern clearly so future phases don't accidentally introduce `encoding/json` for proto message serialization where `protojson` should be used.

## Migration Order Detailed Analysis

### 1. httpgen first (widest API surface)

httpgen uses ALL annotation types: HTTPConfig, ServiceConfig, QueryParam, UnwrapFieldInfo (full version with validation), FieldExamples, headers. It defines the most complete versions of shared types, especially `UnwrapFieldInfo` and `UnwrapValidationError`. Migrating this first establishes the complete shared API.

**Files affected:**
- `annotations.go` (392 lines) -- delete entirely after migration
- `unwrap.go` -- keep unwrap LOGIC (generation, context collection), but `UnwrapFieldInfo`, `UnwrapValidationError`, `hasUnwrapAnnotation`, `getUnwrapField` move to shared
- `validation.go` -- stays, but changes calls to `annotations.GetMethodHTTPConfig(...)`, `annotations.GetQueryParams(...)`
- `generator.go` -- changes calls from `getMethodHTTPConfig(...)` to `annotations.GetMethodHTTPConfig(...)`
- `mock_generator.go` -- changes `getFieldExamples(...)` call
- `annotations_test.go` -- tests for `httpMethodToString`, `extractPathParams` etc. move to shared package tests

### 2. clientgen second

**Files affected:**
- `annotations.go` (241 lines) -- delete entirely
- `generator.go` -- change all annotation function calls to use shared package

### 3. tsclientgen third

**Files affected:**
- `annotations.go` (250 lines) -- delete entirely
- `types.go` -- `findUnwrapField` and `isRootUnwrap` call `annotations.HasUnwrapAnnotation`
- `generator.go` -- change annotation function calls

### 4. openapiv3 last

**Files affected:**
- `http_annotations.go` (406 lines) -- delete most of it; keep `convertHeadersToParameters` and `mapHeaderTypeToOpenAPI` (OpenAPI-specific, ~90 lines), or move those to `generator.go`
- `types.go` -- remove `hasUnwrapAnnotation`, `getUnwrapField`, `getFieldExamples` (~80 lines); keep `convertField`, `convertScalarField`, etc.
- `generator.go` -- change annotation function calls

## Error Handling Recommendation

**Use standard Go errors.** The protogen error reporting (`plugin.Error()`) is designed for user-facing errors that stop generation. The annotation parsing functions currently return `nil` for missing annotations (which is correct -- missing is not an error) and return `error` for invalid annotations (e.g., unwrap on non-repeated field).

Pattern:
- Missing annotation = return nil (caller checks nil)
- Invalid annotation = return error with proto file path + message/field name
- The `UnwrapValidationError` type already follows this pattern -- keep it

Do NOT use `protogen.Plugin.Error()` inside the shared package because the shared package should not have a dependency on the plugin instance. Return errors; let callers decide how to report them.

## Open Questions

1. **Should `CombineHeaders` move to shared?**
   - Currently only in openapiv3, but the concept (method headers override service headers) is universal
   - Recommendation: YES -- extract it, other generators may need it
   - Confidence: HIGH -- the logic is annotation-level, not generator-level

2. **Should `lowerFirst` move to shared?**
   - Duplicated in 3 generators, pure string utility
   - Recommendation: YES -- put it in a `helpers.go` in the annotations package or a separate `internal/stringutil` package
   - Confidence: MEDIUM -- it's not annotation-related, but extracting it alongside annotations makes practical sense. Could go in annotations or a tiny shared helpers package.

3. **Should `BuildHTTPPath` move to shared?**
   - openapiv3 has it as a function; clientgen and tsclientgen have identical inline logic
   - Recommendation: YES -- extract to shared, eliminate the inline duplication
   - Confidence: HIGH

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of all 4 annotation files (1,289 lines total)
- Direct codebase analysis of openapiv3/types.go (additional annotation code)
- Proto definitions at `proto/sebuf/http/annotations.proto` and `headers.proto`
- `go.mod` for exact dependency versions

### Secondary (MEDIUM confidence)
- Go protobuf API stability: `proto.GetExtension` and `descriptorpb` are stable public APIs in google.golang.org/protobuf v1.36.x

### Tertiary (LOW confidence)
- None -- all findings are from direct codebase analysis

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, pure refactoring
- Architecture: HIGH -- all patterns derived from existing codebase analysis
- Pitfalls: HIGH -- identified from structural analysis of the actual code
- Serialization audit: HIGH -- direct code inspection confirms correct usage

**Research date:** 2026-02-05
**Valid until:** Indefinite (internal refactoring, no external dependencies changing)
