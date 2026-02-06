# Phase 5: JSON - Nullable & Empty - Research

**Researched:** 2026-02-06
**Domain:** JSON nullable primitive semantics and empty message handling across all generators
**Confidence:** HIGH

## Summary

This phase implements two complementary JSON serialization controls: (1) `nullable` annotation for primitive fields on proto3 `optional` fields, enabling explicit `null` representation distinct from absent, and (2) `empty_behavior` annotation for message fields, controlling whether empty messages serialize as `{}`, `null`, or are omitted.

The key implementation insight is that sebuf already uses protojson for serialization, which:
- Omits fields with default values by default
- Emits `{}` for empty messages when set
- Already has presence tracking via `HasOptionalKeyword()` for proto3 optional fields

The standard approach involves:
1. Adding new proto extensions (`nullable`, `empty_behavior`) in annotations.proto following Phase 4 patterns
2. Creating shared annotation parsing functions in `internal/annotations/`
3. Generating custom marshal/unmarshal code when annotations modify default protojson behavior
4. Updating OpenAPI schema generation with `type: ["string", "null"]` array syntax (OpenAPI 3.1)
5. Updating TypeScript type generation with `T | null` union types
6. Adding compile-time validation errors for invalid annotation usage

**Primary recommendation:** Follow the Phase 4 annotation pattern exactly. Use `proto.Size(msg) == 0` for empty message detection. Generate MarshalJSON/UnmarshalJSON that intercepts protojson output to apply nullable/empty_behavior transformations.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| protojson | v1.x | JSON marshaling baseline | Official proto3 JSON spec implementation |
| proto | v1.x | Empty message detection via `proto.Size()` | Official protobuf runtime |
| encoding/json | stdlib | Custom marshal/unmarshal | Integrates with Go's json.Marshaler interface |
| protogen | v1.x | Code generation framework | Official protoc plugin framework |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| reflect | stdlib | Type inspection for optional fields | Detecting pointer types in generated Go code |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| proto.Size() == 0 | proto.Equal(msg, &T{}) | Size() is simpler and sufficient for "all defaults" definition |
| Custom marshal | protojson.MarshalOptions.EmitUnpopulated | Only affects global behavior, not per-field control |
| Per-message empty check | Recursive empty check | User decision: "not recursive" - simpler, predictable |

**Installation:**
No new dependencies required. Uses existing protobuf libraries.

## Architecture Patterns

### Recommended Proto Extension Structure

```protobuf
// In proto/sebuf/http/annotations.proto (extend existing)

// EmptyBehavior controls how empty message fields serialize to JSON
enum EmptyBehavior {
  // Follow default: serialize as {} (equivalent to PRESERVE)
  EMPTY_BEHAVIOR_UNSPECIFIED = 0;
  // Serialize empty messages as {} (explicit same as default)
  EMPTY_BEHAVIOR_PRESERVE = 1;
  // Serialize empty messages as null
  EMPTY_BEHAVIOR_NULL = 2;
  // Omit the field entirely when message is empty
  EMPTY_BEHAVIOR_OMIT = 3;
}

extend google.protobuf.FieldOptions {
  // ... existing extensions (unwrap, int64_encoding, enum_encoding) ...

  // Mark a primitive field as nullable (explicit null vs absent).
  // Only valid on proto3 optional fields (HasOptionalKeyword=true).
  // When true: unset field serializes as null, set field serializes normally.
  // When false (default): unset field is omitted from JSON.
  optional bool nullable = 50013;

  // Controls how empty message fields serialize to JSON.
  // Only valid on singular message fields (not repeated, not map).
  // "Empty" = all fields at proto default (proto.Size() == 0).
  optional EmptyBehavior empty_behavior = 50014;
}
```

### Recommended Project Structure (additions)

```
internal/annotations/
    nullable.go           # GetNullable(field), IsNullableField(field), ValidateNullableAnnotation(field)
    empty_behavior.go     # GetEmptyBehavior(field), ValidateEmptyBehaviorAnnotation(field)
```

### Pattern 1: Nullable Primitives with MarshalJSON

**What:** Generate `MarshalJSON`/`UnmarshalJSON` methods on messages containing nullable primitive fields.

**When to use:** Any message with at least one field having `nullable = true`.

**Example:**
```go
// Source: Generated code pattern for go-http handler generator
func (m *User) MarshalJSON() ([]byte, error) {
    if m == nil {
        return []byte("null"), nil
    }

    // Use protojson for base serialization
    data, err := protojson.Marshal(m)
    if err != nil {
        return nil, err
    }

    // Parse into map to handle nullable fields
    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, err
    }

    // Handle nullable field: middleName
    // proto3 optional + nullable=true: emit null when not set
    if m.MiddleName == nil {
        raw["middleName"] = []byte("null")
    }

    return json.Marshal(raw)
}
```

### Pattern 2: Empty Behavior for Message Fields

**What:** Check if nested message is "empty" (all defaults) and apply configured behavior.

**When to use:** Any message field with `empty_behavior` annotation.

**Example:**
```go
// Source: Generated code pattern for empty_behavior handling
func (m *Response) MarshalJSON() ([]byte, error) {
    if m == nil {
        return []byte("null"), nil
    }

    // Use protojson for base serialization
    data, err := protojson.Marshal(m)
    if err != nil {
        return nil, err
    }

    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, err
    }

    // Handle empty_behavior for metadata field
    if m.Metadata != nil && proto.Size(m.Metadata) == 0 {
        switch EmptyBehavior_EMPTY_BEHAVIOR_NULL {  // from annotation
        case EMPTY_BEHAVIOR_NULL:
            raw["metadata"] = []byte("null")
        case EMPTY_BEHAVIOR_OMIT:
            delete(raw, "metadata")
        // PRESERVE: keep as {} (default protojson behavior)
        }
    }

    return json.Marshal(raw)
}
```

### Pattern 3: TypeScript Type Mapping for Nullable

**What:** Adjust TypeScript types based on nullable annotation.

**When to use:** All nullable primitive field type generation in ts-client.

**Example:**
```typescript
// Source: Generated TypeScript pattern
export interface User {
  firstName: string;           // regular required field
  middleName: string | null;   // nullable = true on optional field
  lastName: string;            // regular required field
}
```

### Pattern 4: OpenAPI 3.1 Nullable Schema

**What:** Use JSON Schema type array syntax for nullable types (not deprecated `nullable: true`).

**When to use:** All nullable fields in OpenAPI schema generation.

**Example:**
```yaml
# Source: OpenAPI 3.1 spec pattern (JSON Schema Draft 2020-12)
# nullable = true on optional string field
middleName:
  type: ["string", "null"]

# nullable = true on optional integer field
optionalAge:
  type: ["integer", "null"]
  format: int32

# empty_behavior = NULL on message field
metadata:
  oneOf:
    - $ref: '#/components/schemas/Metadata'
    - type: "null"
```

### Anti-Patterns to Avoid

- **Nullable on non-optional fields:** Decision specifies compile-time error. Non-optional proto3 fields cannot distinguish "not set" from "default" - nullable makes no semantic sense.
- **Empty_behavior on primitives:** Decision specifies compile-time error. Only message fields can be "empty".
- **Empty_behavior on repeated/map:** Decision specifies only singular message fields. Collections already have `[]`/`{}` empty representation.
- **Recursive empty check:** Decision specifies "all fields at proto default" - simple, not recursive into nested messages.
- **Using deprecated nullable: true in OpenAPI:** OpenAPI 3.1 aligned with JSON Schema. Use `type: ["T", "null"]` array syntax.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Empty message detection | Custom field iteration | `proto.Size(msg) == 0` | Official, handles all field types correctly |
| Nullable JSON emission | String concatenation | `json.RawMessage("null")` | Proper JSON encoding |
| Optional field detection | Manual proto syntax parsing | `field.Desc.HasOptionalKeyword()` | protogen API provides this |
| OpenAPI nullable syntax | `nullable: true` (3.0 style) | `type: ["T", "null"]` | OpenAPI 3.1 / JSON Schema standard |

**Key insight:** Use `proto.Size()` for empty message detection. It returns 0 when all fields are at their default values, which matches the user's decision for "empty" definition.

## Common Pitfalls

### Pitfall 1: Confusing Absent vs Null vs Default

**What goes wrong:** Developer expects `nullable=true` to emit `null` for zero values, but it only applies when the optional field is explicitly unset.

**Why it happens:** Three distinct states exist for proto3 optional primitives:
1. Absent (not set, no presence) - omit from JSON
2. Null (nullable=true AND not set) - emit `null`
3. Set to value (including zero) - emit the value

**How to avoid:** Document clearly: `nullable=true` on optional field means "emit null when HasField returns false". A field explicitly set to "" or 0 is NOT null.

**Warning signs:** Tests expecting `{"value": null}` when value was set to default.

### Pitfall 2: Empty Message != Nil Message

**What goes wrong:** Code treats nil pointer and empty struct as equivalent for empty_behavior.

**Why it happens:** In Go, `(*Msg)(nil)` and `&Msg{}` are different. protojson handles them differently.

**How to avoid:** Check `m.Field != nil && proto.Size(m.Field) == 0` for the "allocated but empty" case. A nil field is already omitted by default protojson behavior.

**Warning signs:** Nil message fields getting `null` when they should be omitted.

### Pitfall 3: Inconsistent Handling Across Generators

**What goes wrong:** Go server emits `null` but TypeScript client doesn't handle it, or OpenAPI schema doesn't reflect nullable.

**Why it happens:** Each generator implements nullable independently.

**How to avoid:** Mandate cross-generator golden file tests for every nullable/empty_behavior combination. TypeScript must use `| null`, OpenAPI must use type array.

**Warning signs:** Client parsing errors, TypeScript type errors.

### Pitfall 4: Forgetting UnmarshalJSON

**What goes wrong:** Server can serialize `null` but cannot deserialize it back.

**Why it happens:** Only implementing MarshalJSON, forgetting the inverse.

**How to avoid:** Always generate both marshal and unmarshal together. For nullable: accept both `null` and absent as "unset".

**Warning signs:** 400 errors on request body parsing with `null` values.

### Pitfall 5: Nullable on Non-Optional Field

**What goes wrong:** User annotates `nullable=true` on a regular (non-optional) proto3 field. Generation continues but behavior is undefined.

**Why it happens:** No validation during code generation.

**How to avoid:** Per CONTEXT.md decision: fail generation with clear error message. Check `!field.Desc.HasOptionalKeyword()` and error.

**Warning signs:** Should never happen if validation is implemented correctly.

### Pitfall 6: Empty Behavior on Primitive/Repeated Field

**What goes wrong:** User annotates `empty_behavior` on a string field or repeated message.

**Why it happens:** No validation during code generation.

**How to avoid:** Per CONTEXT.md decision: fail generation with clear error message. Validate field is singular message type.

**Warning signs:** Should never happen if validation is implemented correctly.

## Code Examples

Verified patterns from official sources and codebase analysis:

### Annotation Parsing (following existing Phase 4 pattern)

```go
// Source: Pattern from internal/annotations/int64_encoding.go
package annotations

import (
    "google.golang.org/protobuf/compiler/protogen"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/reflect/protoreflect"
    "google.golang.org/protobuf/types/descriptorpb"

    "github.com/SebastienMelki/sebuf/http"
)

// NullableValidationError represents an error in nullable annotation validation.
type NullableValidationError struct {
    MessageName string
    FieldName   string
    Reason      string
}

func (e *NullableValidationError) Error() string {
    return "invalid nullable annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// IsNullableField returns true if the field has nullable=true annotation.
func IsNullableField(field *protogen.Field) bool {
    options := field.Desc.Options()
    if options == nil {
        return false
    }

    fieldOptions, ok := options.(*descriptorpb.FieldOptions)
    if !ok {
        return false
    }

    ext := proto.GetExtension(fieldOptions, http.E_Nullable)
    if ext == nil {
        return false
    }

    nullable, ok := ext.(bool)
    return ok && nullable
}

// ValidateNullableAnnotation checks if nullable annotation is valid for a field.
// Returns error if nullable=true on a non-optional field.
func ValidateNullableAnnotation(field *protogen.Field, messageName string) error {
    if !IsNullableField(field) {
        return nil // No annotation, nothing to validate
    }

    // Nullable only valid on proto3 optional fields
    if !field.Desc.HasOptionalKeyword() {
        return &NullableValidationError{
            MessageName: messageName,
            FieldName:   string(field.Desc.Name()),
            Reason:      "nullable annotation is only valid on proto3 optional fields",
        }
    }

    // Nullable only valid on primitive types (not messages)
    if field.Desc.Kind() == protoreflect.MessageKind {
        return &NullableValidationError{
            MessageName: messageName,
            FieldName:   string(field.Desc.Name()),
            Reason:      "nullable annotation is only valid on primitive fields, not message fields",
        }
    }

    return nil
}
```

### Empty Behavior Annotation Parsing

```go
// Source: Pattern from internal/annotations/enum_encoding.go
package annotations

// EmptyBehaviorValidationError represents an error in empty_behavior annotation validation.
type EmptyBehaviorValidationError struct {
    MessageName string
    FieldName   string
    Reason      string
}

func (e *EmptyBehaviorValidationError) Error() string {
    return "invalid empty_behavior annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// GetEmptyBehavior returns the empty behavior for a field.
// Returns EMPTY_BEHAVIOR_UNSPECIFIED if not set (use default: PRESERVE).
func GetEmptyBehavior(field *protogen.Field) http.EmptyBehavior {
    options := field.Desc.Options()
    if options == nil {
        return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
    }

    fieldOptions, ok := options.(*descriptorpb.FieldOptions)
    if !ok {
        return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
    }

    ext := proto.GetExtension(fieldOptions, http.E_EmptyBehavior)
    if ext == nil {
        return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
    }

    behavior, ok := ext.(http.EmptyBehavior)
    if !ok {
        return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
    }

    return behavior
}

// ValidateEmptyBehaviorAnnotation checks if empty_behavior annotation is valid for a field.
func ValidateEmptyBehaviorAnnotation(field *protogen.Field, messageName string) error {
    behavior := GetEmptyBehavior(field)
    if behavior == http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED {
        return nil // No annotation, nothing to validate
    }

    // Empty behavior only valid on message fields
    if field.Desc.Kind() != protoreflect.MessageKind {
        return &EmptyBehaviorValidationError{
            MessageName: messageName,
            FieldName:   string(field.Desc.Name()),
            Reason:      "empty_behavior annotation is only valid on message fields",
        }
    }

    // Not valid on repeated fields
    if field.Desc.IsList() {
        return &EmptyBehaviorValidationError{
            MessageName: messageName,
            FieldName:   string(field.Desc.Name()),
            Reason:      "empty_behavior annotation is not valid on repeated fields",
        }
    }

    // Not valid on map fields
    if field.Desc.IsMap() {
        return &EmptyBehaviorValidationError{
            MessageName: messageName,
            FieldName:   string(field.Desc.Name()),
            Reason:      "empty_behavior annotation is not valid on map fields",
        }
    }

    return nil
}
```

### Empty Message Detection

```go
// Source: https://pkg.go.dev/google.golang.org/protobuf/proto
// Using proto.Size() for empty detection as recommended by official docs

import "google.golang.org/protobuf/proto"

// IsEmptyMessage returns true if the message has all fields at default values.
// This matches the user's decision: "Empty" = all fields at proto default.
func IsEmptyMessage(msg proto.Message) bool {
    return proto.Size(msg) == 0
}
```

### TypeScript Nullable Type Generation

```typescript
// Source: Pattern from internal/tsclientgen/types.go analysis

// Current behavior for optional fields (lines 343-352):
// - proto3 optional -> fieldName?: T
// - message fields -> fieldName?: T

// With nullable=true:
// - optional string with nullable=true -> fieldName: string | null
// - optional int32 with nullable=true -> fieldName: number | null

// Note: nullable removes the "?" (optional) modifier because
// the field is now always present in JSON (either value or null)
export interface User {
  firstName: string;           // required
  middleName: string | null;   // optional + nullable=true
  lastName?: string;           // optional without nullable
}
```

### OpenAPI 3.1 Nullable Schema Generation

```yaml
# Source: OpenAPI 3.1 spec pattern (JSON Schema Draft 2020-12)
# Reference: https://github.com/OAI/OpenAPI-Specification/issues/3148

# nullable = true on optional string field
middleName:
  type: ["string", "null"]

# nullable = true on optional integer field
optionalAge:
  type: ["integer", "null"]
  format: int32

# nullable = true on optional boolean field
isActive:
  type: ["boolean", "null"]

# empty_behavior = NULL on message field - use oneOf for message refs
metadata:
  oneOf:
    - $ref: '#/components/schemas/Metadata'
    - type: "null"

# empty_behavior = OMIT - field not marked required, just optional
# (no schema change needed, just omit from required array)
```

## Implementation Strategy by Generator

### go-http (HTTP Server)

**Touch points:**
- `internal/httpgen/generator.go`: Add validation for nullable/empty_behavior annotations
- New file `internal/httpgen/nullable.go`: Nullable marshal/unmarshal generation
- New file `internal/httpgen/empty_behavior.go`: Empty behavior marshal/unmarshal generation
- `proto/sebuf/http/annotations.proto`: Add nullable and empty_behavior extensions

**Key considerations:**
- Integrate with existing int64 encoding and unwrap MarshalJSON generation
- Messages may need combined MarshalJSON handling multiple features
- Validation errors should fail generation early (same pattern as enum annotation conflicts)

### go-client (HTTP Client)

**Touch points:**
- `internal/clientgen/generator.go`: Add validation (mirror go-http)
- `internal/clientgen/nullable.go`: Mirror go-http implementation
- `internal/clientgen/empty_behavior.go`: Mirror go-http implementation

**Key consideration:** Client and server must produce identical JSON for interoperability. Share or duplicate logic carefully.

### ts-client (TypeScript Client)

**Touch points:**
- `internal/tsclientgen/types.go`: Modify `isOptionalField()` and `tsFieldType()` for nullable
- `internal/tsclientgen/generator.go`: Handle nullable in interface generation

**Key considerations:**
- `nullable=true` changes type from `T?` to `T | null`
- empty_behavior affects only serialization, not TypeScript types (message fields remain optional)

### openapiv3 (OpenAPI Spec)

**Touch points:**
- `internal/openapiv3/types.go`: Add nullable type array generation
- `internal/openapiv3/generator.go`: Handle empty_behavior for required array

**Key considerations:**
- Use OpenAPI 3.1 `type: ["T", "null"]` syntax, not deprecated `nullable: true`
- For message fields with empty_behavior=NULL, use `oneOf` with null type

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| protojson default omit | Per-field nullable annotation | This phase | Enables explicit null vs absent |
| Empty messages = {} | Per-field empty_behavior | This phase | Control over empty message serialization |
| OpenAPI 3.0 nullable: true | OpenAPI 3.1 type array | OpenAPI 3.1 (2021) | Aligns with JSON Schema Draft 2020-12 |

**Deprecated/outdated:**
- OpenAPI `nullable: true` keyword: Replaced by type array in 3.1
- protojson EmitUnpopulated as solution: Global, not per-field control

## Open Questions

Things that couldn't be fully resolved:

1. **Absent with explicit default value presence**
   - What we know: CONTEXT.md says "Absent = key omitted from JSON" and marks supporting explicit default as Claude's discretion
   - What's unclear: Should we add a third mode where optional+set-to-default emits the default value?
   - Recommendation: Defer this to a future phase. Focus on null vs absent for now. The two-state model (nullable=true means null when unset, value when set) is simpler and covers the primary use case.

2. **Combined MarshalJSON with multiple features**
   - What we know: Messages may have unwrap, int64_encoding, nullable, and empty_behavior
   - What's unclear: Best way to combine multiple MarshalJSON transformations
   - Recommendation: Generate unified MarshalJSON that handles all features. Check existing pattern in encoding.go where protojson output is modified in-place via map[string]json.RawMessage.

3. **Error message wording**
   - What we know: CONTEXT.md marks exact wording as Claude's discretion
   - Recommendation: Follow existing pattern from UnwrapValidationError. Format: "invalid [annotation] annotation on [Message].[field]: [reason]"

## Sources

### Primary (HIGH confidence)

- [protojson package docs](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) - EmitUnpopulated, EmitDefaultValues options
- [proto package docs](https://pkg.go.dev/google.golang.org/protobuf/proto) - proto.Size() for empty detection
- [ProtoJSON Format Guide](https://protobuf.dev/programming-guides/json/) - null handling, presence semantics
- [OpenAPI 3.1 nullable syntax](https://github.com/OAI/OpenAPI-Specification/issues/3148) - type array instead of nullable keyword
- Codebase: `internal/annotations/*.go` - Existing annotation parsing patterns
- Codebase: `internal/openapiv3/types.go` - Current optional field handling (line 54)
- Codebase: `internal/tsclientgen/types.go` - Current optional field handling (lines 343-352)
- Codebase: `internal/httpgen/encoding.go` - Phase 4 MarshalJSON pattern

### Secondary (MEDIUM confidence)

- Phase 4 RESEARCH.md - Annotation infrastructure patterns
- Phase 4 04-01-PLAN.md - Annotation definition patterns

### Tertiary (LOW confidence)

- WebSearch results for proto3 empty message patterns - verified against official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - uses existing protobuf/stdlib libraries, follows Phase 4 pattern
- Architecture: HIGH - follows established codebase patterns from Phase 4
- Pitfalls: HIGH - documented in official protojson spec and verified behavior

**Research date:** 2026-02-06
**Valid until:** 90 days (stable domain, proto3 JSON spec is mature)
