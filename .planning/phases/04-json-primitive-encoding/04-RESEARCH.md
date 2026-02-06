# Phase 4: JSON - Primitive Encoding - Research

**Researched:** 2026-02-05
**Domain:** JSON encoding for int64/uint64 and enum fields across all generators
**Confidence:** HIGH

## Summary

This phase implements field-level annotations to control how int64/uint64 and enum fields serialize to JSON. The key challenge is that sebuf currently uses protojson for serialization, which follows the proto3 JSON spec where int64/uint64 serialize as strings by default. To support NUMBER encoding, we must generate custom MarshalJSON/UnmarshalJSON methods that override protojson's default behavior.

The standard approach involves:
1. Adding new proto extensions (`int64_encoding`, `enum_encoding`, `enum_value`) in annotations.proto
2. Creating shared annotation parsing functions in `internal/annotations/`
3. Generating custom marshal/unmarshal code in each generator when annotations are present
4. Updating OpenAPI schema generation to reflect the configured encoding

**Primary recommendation:** Generate custom MarshalJSON/UnmarshalJSON methods per-message when any field has a non-default encoding annotation, using strconv for int64 conversion and lookup maps for enum conversion.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| protojson | v1.x | JSON marshaling baseline | Official proto3 JSON spec implementation |
| strconv | stdlib | int64 string conversion | Standard library, no external dependencies |
| encoding/json | stdlib | Custom marshal/unmarshal | Integrates with Go's json.Marshaler interface |
| protogen | v1.x | Code generation framework | Official protoc plugin framework |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| strings | stdlib | String manipulation | Enum value transformations |
| fmt | stdlib | Formatting | Error messages, sprintf patterns |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom marshal | protojson.MarshalOptions with UseEnumNumbers | Only covers enum-as-number globally, not per-field; no int64-as-number option exists |
| Generated code | Runtime reflection | Performance penalty, less type safety |

**Installation:**
No new dependencies required. Uses existing protobuf libraries.

## Architecture Patterns

### Recommended Proto Extension Structure

```protobuf
// In proto/sebuf/http/annotations.proto (extend existing)

// Int64Encoding controls how int64/uint64 fields serialize to JSON
enum Int64Encoding {
  INT64_ENCODING_UNSPECIFIED = 0;  // Follow protojson default (STRING)
  INT64_ENCODING_STRING = 1;       // Explicit string: "12345"
  INT64_ENCODING_NUMBER = 2;       // Numeric: 12345 (precision warning)
}

// EnumEncoding controls how enum fields serialize to JSON
enum EnumEncoding {
  ENUM_ENCODING_UNSPECIFIED = 0;  // Follow protojson default (STRING name)
  ENUM_ENCODING_STRING = 1;       // Explicit string: "STATUS_ACTIVE"
  ENUM_ENCODING_NUMBER = 2;       // Numeric: 1
}

extend google.protobuf.FieldOptions {
  // ... existing extensions ...

  // Controls int64/uint64 JSON encoding for this field
  optional Int64Encoding int64_encoding = 50010;

  // Controls enum JSON encoding for this field (only valid on enum fields)
  optional EnumEncoding enum_encoding = 50011;
}

extend google.protobuf.EnumValueOptions {
  // Custom JSON string for this enum value
  optional string enum_value = 50012;
}
```

### Recommended Project Structure (additions)

```
internal/annotations/
    int64_encoding.go     # GetInt64Encoding(field) -> Int64Encoding
    enum_encoding.go      # GetEnumEncoding(field), GetEnumValueMapping(enum)
```

### Pattern 1: Custom MarshalJSON for Messages with Annotated Fields

**What:** Generate `MarshalJSON`/`UnmarshalJSON` methods on messages containing fields with non-default encoding annotations.

**When to use:** Any message with at least one field having `int64_encoding = NUMBER` or `enum_encoding` with custom values.

**Example:**
```go
// Source: Generated code pattern for go-http handler generator
func (m *Tweet) MarshalJSON() ([]byte, error) {
    type alias Tweet // Prevent infinite recursion

    // Build intermediate struct with string fields for int64
    raw := struct {
        alias
        Id       int64  `json:"id"`        // Override: encode as number
        AuthorId string `json:"authorId"`  // Default: encode as string
    }{
        alias:    alias(*m),
        Id:       m.Id,                    // Direct number
        AuthorId: strconv.FormatInt(m.AuthorId, 10),
    }

    return json.Marshal(raw)
}
```

### Pattern 2: Enum Value Lookup Maps

**What:** Generate bidirectional lookup maps for custom enum JSON values.

**When to use:** Any enum with `enum_value` annotations on its values.

**Example:**
```go
// Source: Generated code pattern for enum with custom values
var statusToJSON = map[Status]string{
    Status_STATUS_UNSPECIFIED: "unknown",
    Status_STATUS_ACTIVE:      "active",
    Status_STATUS_INACTIVE:    "inactive",
}

var statusFromJSON = map[string]Status{
    "unknown":  Status_STATUS_UNSPECIFIED,
    "active":   Status_STATUS_ACTIVE,
    "inactive": Status_STATUS_INACTIVE,
}

func (s Status) MarshalJSON() ([]byte, error) {
    if v, ok := statusToJSON[s]; ok {
        return json.Marshal(v)
    }
    // Fallback to proto name if no custom value
    return json.Marshal(s.String())
}

func (s *Status) UnmarshalJSON(data []byte) error {
    var str string
    if err := json.Unmarshal(data, &str); err != nil {
        // Try as number
        var num int32
        if numErr := json.Unmarshal(data, &num); numErr != nil {
            return err
        }
        *s = Status(num)
        return nil
    }
    if v, ok := statusFromJSON[str]; ok {
        *s = v
        return nil
    }
    // Try as proto name
    if v, ok := Status_value[str]; ok {
        *s = Status(v)
        return nil
    }
    return fmt.Errorf("unknown status: %s", str)
}
```

### Pattern 3: TypeScript Type Mapping

**What:** Adjust TypeScript types based on encoding annotation.

**When to use:** All int64/enum field type generation in ts-client.

**Example:**
```typescript
// Source: Generated TypeScript pattern
export interface Tweet {
  id: number;         // int64_encoding = NUMBER -> number
  authorId: string;   // int64_encoding = STRING (default) -> string
  status: "active" | "inactive" | "unknown";  // enum with custom values
}
```

### Anti-Patterns to Avoid

- **Global protojson.MarshalOptions modification:** Doesn't support per-field control
- **Runtime reflection for encoding decisions:** Performance overhead on every marshal
- **Mixing encoding/json and protojson in same message:** Inconsistent behavior
- **File-level defaults:** Decision explicitly rejected in CONTEXT.md for simplicity

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| int64 <-> string conversion | Manual parsing | `strconv.FormatInt`/`strconv.ParseInt` | Handles edge cases (negative, overflow) |
| JSON number precision check | Custom validation | Document warning + format: int64 | Standard spec acknowledgment |
| Enum name lookup | String comparison | `Enum_value` map from protobuf | Protobuf provides canonical name mapping |
| JSON struct tags | Manual string building | struct literals with tags | Go's json package handles escaping |

**Key insight:** The standard library's `strconv` package handles all int64 edge cases including negative numbers, parsing errors, and overflow detection. Don't reimplement this.

## Common Pitfalls

### Pitfall 1: Infinite Recursion in MarshalJSON

**What goes wrong:** Calling `json.Marshal(m)` inside `MarshalJSON()` causes infinite recursion.

**Why it happens:** Go's json package calls MarshalJSON if defined on the type.

**How to avoid:** Use type alias pattern: `type alias Tweet` without methods, then marshal the alias.

**Warning signs:** Stack overflow, program hangs during serialization.

### Pitfall 2: Inconsistent Handling Across Generators

**What goes wrong:** Go server encodes int64 as number but TypeScript client expects string.

**Why it happens:** Each generator implements encoding independently.

**How to avoid:** Mandate cross-generator golden file tests for every encoding combination.

**Warning signs:** Client parsing errors, type mismatches.

### Pitfall 3: Precision Loss Without Warning

**What goes wrong:** JavaScript client receives `9223372036854775807` and silently rounds to `9223372036854776000`.

**Why it happens:** JavaScript Number uses IEEE 754 double precision (max safe: 2^53).

**How to avoid:** Per CONTEXT.md decision, generate warning during code generation AND add inline comment in generated code.

**Warning signs:** Incorrect ID values in frontend, data corruption.

### Pitfall 4: Enum Value Conflicts

**What goes wrong:** User annotates `enum_encoding = NUMBER` AND `enum_value = "active"` on same field.

**Why it happens:** Conflicting instructions (number vs custom string).

**How to avoid:** Per CONTEXT.md decision, fail generation with clear error message.

**Warning signs:** Ambiguous serialization behavior.

### Pitfall 5: Forgetting UnmarshalJSON

**What goes wrong:** Server can serialize but cannot deserialize the same JSON.

**Why it happens:** Only implementing MarshalJSON, forgetting the inverse.

**How to avoid:** Always generate both marshal and unmarshal together.

**Warning signs:** 400 errors on request body parsing.

### Pitfall 6: Default Value Handling

**What goes wrong:** int64 field with value 0 serializes as `"0"` (string) but user expects omission.

**Why it happens:** Encoding annotation doesn't change omit-empty behavior.

**How to avoid:** Document that `int64_encoding` only affects non-zero values unless `EmitUnpopulated` is used.

**Warning signs:** Unexpected `"0"` in JSON output.

## Code Examples

Verified patterns from official sources and codebase analysis:

### Annotation Parsing (following existing pattern)

```go
// Source: Pattern from internal/annotations/field_examples.go
package annotations

import (
    "google.golang.org/protobuf/compiler/protogen"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/descriptorpb"

    "github.com/SebastienMelki/sebuf/http"
)

// GetInt64Encoding returns the int64 encoding for a field.
// Returns INT64_ENCODING_UNSPECIFIED if not set (use protojson default: STRING).
func GetInt64Encoding(field *protogen.Field) http.Int64Encoding {
    options := field.Desc.Options()
    if options == nil {
        return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
    }

    fieldOptions, ok := options.(*descriptorpb.FieldOptions)
    if !ok {
        return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
    }

    ext := proto.GetExtension(fieldOptions, http.E_Int64Encoding)
    if ext == nil {
        return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
    }

    encoding, ok := ext.(http.Int64Encoding)
    if !ok {
        return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
    }

    return encoding
}
```

### protojson Default Behavior Reference

```go
// Source: https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson
// Current sebuf behavior uses protojson for serialization

// Default: int64/uint64 -> JSON string
msg := &pb.Tweet{Id: 12345}
bytes, _ := protojson.Marshal(msg)
// Result: {"id":"12345"}

// With UseEnumNumbers (global only, not per-field):
opts := protojson.MarshalOptions{UseEnumNumbers: true}
bytes, _ = opts.Marshal(msg)
// Result: {"status":1} instead of {"status":"STATUS_ACTIVE"}
```

### OpenAPI Schema for int64 Encoding

```yaml
# Source: OpenAPI 3.1 spec pattern
# int64_encoding = STRING (default)
id:
  type: string
  format: int64

# int64_encoding = NUMBER (with precision warning)
id:
  type: integer
  format: int64
  description: "Warning: Values > 2^53 may lose precision in JavaScript"
```

### TypeScript Type Generation

```typescript
// Source: Pattern from internal/tsclientgen/types.go analysis

// Current behavior: int64 -> string (lines 31-34)
// int64_encoding = NUMBER should generate:
export interface Tweet {
  id: number;  // int64 with NUMBER encoding
}

// int64_encoding = STRING (default, current behavior):
export interface Tweet {
  id: string;  // int64 with STRING encoding
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| protojson global UseEnumNumbers | Per-field enum_encoding | This phase | Enables mixed enum encoding within same message |
| No int64-as-number option | Per-field int64_encoding | This phase | Enables number encoding where precision loss is acceptable |
| Enum values = proto names only | Custom enum_value per value | This phase | Enables REST-friendly lowercase strings |

**Deprecated/outdated:**
- File-level default annotations: Rejected during discuss phase for simplicity
- Message-level inheritance: Rejected for field-level only granularity

## Implementation Strategy by Generator

### go-http (HTTP Server)

**Touch points:**
- `internal/httpgen/generator.go`: Check for annotated fields, generate marshal/unmarshal
- New file `internal/httpgen/encoding.go`: Marshal/unmarshal generation logic
- `proto/sebuf/http/annotations.proto`: Add new extensions

**Key consideration:** Generated code must integrate with existing unwrap marshal/unmarshal if both annotations present on same message.

### go-client (HTTP Client)

**Touch points:**
- `internal/clientgen/generator.go`: Same marshal/unmarshal as server
- Should share generation logic with go-http via internal package

**Key consideration:** Client and server must produce identical JSON for interoperability.

### ts-client (TypeScript Client)

**Touch points:**
- `internal/tsclientgen/types.go`: Modify `tsScalarType()` for int64
- `internal/tsclientgen/generator.go`: Enum type generation with custom values

**Key consideration:** TypeScript `number` type has same precision limits as JavaScript.

### openapiv3 (OpenAPI Spec)

**Touch points:**
- `internal/openapiv3/types.go`: Modify `convertScalarField()` and `convertEnumField()`
- Add precision warning description for NUMBER encoding

**Key consideration:** OpenAPI schema must accurately document the actual JSON format.

## Open Questions

Things that couldn't be fully resolved:

1. **Collection field support for int64_encoding**
   - What we know: CONTEXT.md marks this as Claude's discretion
   - What's unclear: Should `repeated int64` with `int64_encoding = NUMBER` produce `[1, 2, 3]` vs `["1", "2", "3"]`?
   - Recommendation: Support it - the annotation applies to the element encoding, not the collection structure. Technical feasibility confirmed.

2. **Interaction with existing unwrap annotation**
   - What we know: Messages can have both unwrap and encoding annotations
   - What's unclear: Order of operations in generated marshal code
   - Recommendation: Encoding applies to individual fields within the unwrapped structure. Generate unified MarshalJSON that handles both.

3. **Enum value annotation on field vs enum definition**
   - What we know: `enum_value` goes on EnumValueOptions per CONTEXT.md
   - What's unclear: GitHub issue #89 shows annotation on enum definition, but CONTEXT.md specifies per-value
   - Recommendation: Follow CONTEXT.md - per-value annotation is more flexible (partial override)

## Sources

### Primary (HIGH confidence)

- [Protocol Buffers JSON Mapping](https://protobuf.dev/programming-guides/json/) - Official proto3 JSON spec
- [protojson package docs](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) - MarshalOptions reference
- Codebase: `internal/annotations/*.go` - Existing annotation parsing patterns
- Codebase: `internal/openapiv3/types.go:75-89` - Current int64/uint64 handling
- Codebase: `internal/tsclientgen/types.go:31-34` - Current int64 type mapping

### Secondary (MEDIUM confidence)

- [GitHub Issue #88](https://github.com/SebastienMelki/sebuf/issues/88) - int64 encoding requirement
- [GitHub Issue #89](https://github.com/SebastienMelki/sebuf/issues/89) - enum encoding requirement
- [protobuf/issues/2679](https://github.com/protocolbuffers/protobuf/issues/2679) - Why int64 serializes as string
- [protobuf/issues/8331](https://github.com/protocolbuffers/protobuf/issues/8331) - int64 as JSON number option request

### Tertiary (LOW confidence)

- WebSearch results for custom marshal patterns - verified against stdlib documentation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - uses existing protobuf/stdlib libraries
- Architecture: HIGH - follows established codebase patterns
- Pitfalls: HIGH - documented in official protobuf issues and spec

**Research date:** 2026-02-05
**Valid until:** 90 days (stable domain, proto3 JSON spec is mature)
