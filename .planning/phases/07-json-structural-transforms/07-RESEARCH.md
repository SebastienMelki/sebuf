# Phase 7: JSON - Structural Transforms - Research

**Researched:** 2026-02-06
**Domain:** Protobuf oneof discriminated unions + nested message flattening in JSON across 4 generators
**Confidence:** HIGH (codebase-first research, verified patterns from existing phases)

## Summary

Phase 7 adds two structural JSON transforms to sebuf: (1) oneof fields as discriminated unions with an explicit type discriminator, and (2) nested message flattening that promotes child fields to parent level. These are fundamentally different from Phase 4-6 features (which transform individual field values) because they change the **shape** of JSON objects.

The codebase already has a well-established pattern for custom JSON transforms via `MarshalJSON`/`UnmarshalJSON` generation (see `unwrap.go`, `encoding.go`, `nullable.go`). The new features follow the same pattern: detect annotations, collect context, generate custom marshal/unmarshal methods. The key new challenge is that oneof discriminated unions require reading a JSON discriminator field before knowing which type to deserialize into -- a pattern not yet present in the codebase.

**Primary recommendation:** Implement oneof discriminator first (higher complexity, more critical feature), then flatten. Both features use the same annotation extension mechanism (oneof-level via `google.protobuf.OneofOptions`, field-level via existing `google.protobuf.FieldOptions`). Couple `oneof_discriminator` and `oneof_flatten` into a single annotation message for simplicity. When no variant is set, emit the JSON object with only the discriminator field present (e.g., `{"type": ""}` or omit entirely).

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/protobuf | v1.36.11 | protogen plugin framework, protojson | Already in use; provides `protogen.Oneof`, `protogen.Field.Oneof` for detecting oneof membership |
| pb33f/libopenapi | v0.33.0 | OpenAPI generation with `Discriminator` struct support | Already in use; `base.Schema.Discriminator` and `base.Schema.OneOf` directly support discriminated unions |
| internal/annotations | N/A | Shared annotation parsing across all 4 generators | Established Phase 2 pattern; new files `oneof_discriminator.go` and `flatten.go` follow convention |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json | stdlib | Raw JSON map manipulation in generated MarshalJSON/UnmarshalJSON | Same pattern as unwrap.go, nullable.go -- parse to `map[string]json.RawMessage`, modify, re-marshal |
| google.golang.org/protobuf/encoding/protojson | v1.36.11 | Base serialization for non-transformed fields | Same as existing patterns |
| google.golang.org/protobuf/types/descriptorpb | v1.36.11 | Access `OneofOptions` for oneof-level annotations | New: previous phases only used `FieldOptions`, `ServiceOptions`, `MethodOptions`, `EnumValueOptions` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Oneof-level annotations | Field-level annotations on each variant | Field-level is noisier; oneof-level is cleaner since discriminator applies to the whole oneof, not individual fields. DECISION: Use oneof-level. |
| Coupled oneof_discriminator+flatten | Independent annotations | Independent allows discriminator-without-flatten (just adds a "type" field alongside the standard proto oneof encoding). But the user decision says coupling is at Claude's discretion. RECOMMENDATION: Coupled into a single `OneofConfig` message for fewer combinations. |
| Single-level flatten | Recursive flatten | Recursive adds complexity (cycle detection, deep collision checking). RECOMMENDATION: Single-level only. |

## Architecture Patterns

### Recommended File Structure for New Code

```
proto/sebuf/http/annotations.proto     # Add OneofConfig, extend OneofOptions + flatten FieldOptions
internal/annotations/oneof.go          # GetOneofDiscriminator(), GetOneofFlatten(), GetOneofVariantValue()
internal/annotations/flatten.go        # IsFlattenField(), GetFlattenPrefix()
internal/httpgen/oneof.go              # Oneof discriminated union MarshalJSON/UnmarshalJSON generation
internal/httpgen/flatten.go            # Flatten MarshalJSON/UnmarshalJSON generation
internal/clientgen/oneof.go            # Same oneof code (mirrors httpgen)
internal/clientgen/flatten.go          # Same flatten code (mirrors httpgen)
internal/tsclientgen/types.go          # Update generateInterface() for oneof + flatten
internal/openapiv3/types.go            # Update buildObjectSchema() for discriminator + allOf
internal/httpgen/testdata/proto/oneof_discriminator.proto  # Test proto
internal/httpgen/testdata/proto/flatten.proto               # Test proto
```

### Pattern 1: Oneof-Level Annotation via OneofOptions

**What:** Define discriminator configuration at the oneof level using `google.protobuf.OneofOptions` extension.
**When to use:** Always for oneof discriminator/flatten annotations.

Proto annotation design:
```protobuf
// OneofConfig controls oneof serialization as discriminated union.
message OneofConfig {
  // The JSON field name for the discriminator (e.g., "type", "kind").
  string discriminator = 1;
  // Whether to flatten variant message fields to the same level as discriminator.
  bool flatten = 2;
}

extend google.protobuf.OneofOptions {
  optional OneofConfig oneof_config = 50017;
}

// Per-variant custom discriminator value (on the oneof field itself).
extend google.protobuf.FieldOptions {
  optional string oneof_value = 50018;
}
```

Usage:
```protobuf
message Event {
  string id = 1;
  oneof content {
    option (sebuf.http.oneof_config) = {
      discriminator: "type"
      flatten: true
    };
    TextContent text = 2;
    ImageContent image = 3 [(sebuf.http.oneof_value) = "img"];
  }
}
```

This produces JSON: `{"id": "123", "type": "text", "body": "hello"}` (flattened) or `{"id": "123", "type": "text", "text": {"body": "hello"}}` (non-flattened).

### Pattern 2: Protogen Oneof API Usage

**What:** Use `protogen.Message.Oneofs` and `protogen.Field.Oneof` to detect and process oneof fields.
**When to use:** In annotation extraction and all generators.

```go
// In internal/annotations/oneof.go
func GetOneofConfig(oneof *protogen.Oneof) *http.OneofConfig {
    options := oneof.Desc.Options()
    if options == nil {
        return nil
    }
    oneofOptions, ok := options.(*descriptorpb.OneofOptions)
    if !ok {
        return nil
    }
    ext := proto.GetExtension(oneofOptions, http.E_OneofConfig)
    if ext == nil {
        return nil
    }
    config, ok := ext.(*http.OneofConfig)
    if !ok {
        return nil
    }
    return config
}
```

Key protogen types:
- `message.Oneofs` -- slice of `*protogen.Oneof` in a message
- `oneof.Fields` -- slice of `*protogen.Field` that are variants of this oneof
- `field.Oneof` -- pointer to containing `*protogen.Oneof` (nil if not in a oneof)
- `oneof.Desc` -- `protoreflect.OneofDescriptor` for accessing options
- `oneof.GoName` -- Go identifier for the oneof (e.g., "Content")

### Pattern 3: Generated MarshalJSON for Discriminated Union (Flattened)

**What:** Generate custom MarshalJSON that reads the oneof interface, adds discriminator, and flattens variant fields.
**When to use:** When `oneof_config.discriminator` is set and `flatten = true`.

Generated code pattern (conceptual):
```go
func (x *Event) MarshalJSON() ([]byte, error) {
    // 1. Use protojson for base serialization of non-oneof fields
    data, err := protojson.Marshal(x)
    if err != nil { return nil, err }

    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil { return nil, err }

    // 2. Remove the standard oneof field(s) added by protojson
    delete(raw, "text")
    delete(raw, "image")

    // 3. Add discriminator and flatten variant fields
    switch v := x.Content.(type) {
    case *Event_Text:
        raw["type"] = []byte(`"text"`)
        if v.Text != nil {
            variantData, err := protojson.Marshal(v.Text)
            // ... merge variant fields into raw
        }
    case *Event_Image:
        raw["type"] = []byte(`"img"`) // custom oneof_value
        if v.Image != nil {
            variantData, err := protojson.Marshal(v.Image)
            // ... merge variant fields into raw
        }
    case nil:
        // Unset oneof: omit discriminator entirely
    }

    return json.Marshal(raw)
}
```

### Pattern 4: Generated UnmarshalJSON for Discriminated Union (Flattened)

**What:** Generate custom UnmarshalJSON that reads discriminator first, then routes to correct variant.
**When to use:** When `oneof_config.discriminator` is set.

Generated code pattern (conceptual):
```go
func (x *Event) UnmarshalJSON(data []byte) error {
    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil { return err }

    // 1. Read discriminator
    var discriminatorValue string
    if discRaw, ok := raw["type"]; ok {
        json.Unmarshal(discRaw, &discriminatorValue)
    }

    // 2. Route to correct variant based on discriminator
    switch discriminatorValue {
    case "text":
        variant := &TextContent{}
        // Re-marshal the remaining fields for protojson deserialization
        variantData, _ := json.Marshal(raw) // raw still has all fields
        if err := protojson.Unmarshal(variantData, variant); err != nil {
            return err
        }
        x.Content = &Event_Text{Text: variant}
    case "img":
        variant := &ImageContent{}
        variantData, _ := json.Marshal(raw)
        if err := protojson.Unmarshal(variantData, variant); err != nil {
            return err
        }
        x.Content = &Event_Image{Image: variant}
    }

    // 3. Remove oneof fields from raw, then unmarshal remaining fields via protojson
    delete(raw, "type")
    // Remove variant-specific fields too
    // ... re-marshal remaining and protojson.Unmarshal into x for non-oneof fields

    return nil
}
```

### Pattern 5: Flatten Annotation (Field-Level)

**What:** `flatten = true` on a message field promotes child fields to parent level.
**When to use:** For nested message flattening.

```protobuf
extend google.protobuf.FieldOptions {
  optional bool flatten = 50019;
  optional string flatten_prefix = 50020;
}
```

Usage:
```protobuf
message Order {
  string id = 1;
  Address billing = 2 [(sebuf.http.flatten) = true, (sebuf.http.flatten_prefix) = "billing_"];
  Address shipping = 3 [(sebuf.http.flatten) = true, (sebuf.http.flatten_prefix) = "shipping_"];
}
```

JSON: `{"id": "123", "billing_street": "...", "billing_city": "...", "shipping_street": "...", "shipping_city": "..."}`

### Pattern 6: OpenAPI Discriminated Union

**What:** Use libopenapi's `Discriminator` struct with `OneOf` for OpenAPI representation.
**When to use:** When a message has a oneof with discriminator annotation.

```go
// In openapiv3 generator, when building schema for a message with discriminated oneof:
discriminator := &base.Discriminator{
    PropertyName: "type",
    Mapping: orderedmap.New[string, string](),
}
discriminator.Mapping.Set("text", "#/components/schemas/TextContent")
discriminator.Mapping.Set("img", "#/components/schemas/ImageContent")

schema.OneOf = []*base.SchemaProxy{
    base.CreateSchemaProxyRef("#/components/schemas/TextContent"),
    base.CreateSchemaProxyRef("#/components/schemas/ImageContent"),
}
schema.Discriminator = discriminator
```

For flattened variants, use `allOf` to combine variant properties with the discriminator property:
```yaml
Event:
  oneOf:
    - $ref: '#/components/schemas/Event_text'
    - $ref: '#/components/schemas/Event_image'
  discriminator:
    propertyName: type
    mapping:
      text: '#/components/schemas/Event_text'
      img: '#/components/schemas/Event_image'

Event_text:
  allOf:
    - type: object
      properties:
        type:
          type: string
          enum: [text]
    - $ref: '#/components/schemas/TextContent'
    - type: object
      properties:
        id:
          type: string
```

### Anti-Patterns to Avoid

- **Do NOT use reflection at runtime for oneof type-switching**: All type-switch code must be generated at compile time with concrete types. The generated code uses Go type switches on the oneof interface.
- **Do NOT try to merge multiple MarshalJSON methods**: If a message has both oneof discriminator AND int64 encoding, one `MarshalJSON` must handle all transforms. This means the generated file must combine all encoding concerns.
- **Do NOT allow flatten on repeated or map fields**: Flatten only makes sense for singular message fields. Validate at generation time.
- **Do NOT allow flatten on oneof variant fields**: The oneof_flatten annotation handles promotion of variant fields. Mixing per-field flatten with oneof flatten creates ambiguity.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON field collision detection | Custom string comparison | `map[string]struct{}` tracking seen JSON names | Map lookup is O(1) and handles case sensitivity correctly |
| Oneof type info extraction | Manual descriptor walking | `protogen.Message.Oneofs` + `protogen.Oneof.Fields` | protogen already exposes the full oneof structure |
| OpenAPI discriminator schema | Manual YAML construction | `base.Discriminator{PropertyName, Mapping}` from libopenapi | Library handles proper YAML serialization with ordered maps |
| Oneof variant value resolution | Custom proto option parsing | `proto.GetExtension(fieldOptions, http.E_OneofValue)` | Same pattern as every other annotation in the codebase |

**Key insight:** The codebase already has 6+ encoding features that follow the exact same pattern (detect annotation -> collect context -> generate MarshalJSON/UnmarshalJSON). Reuse this pattern wholesale.

## Common Pitfalls

### Pitfall 1: MarshalJSON Method Conflict When Multiple Features Apply

**What goes wrong:** A message might have both a discriminated oneof AND nullable fields AND int64 NUMBER encoding. Each feature currently generates its own independent MarshalJSON. Having two MarshalJSON on the same type is a compile error.

**Why it happens:** Each encoding feature (unwrap.go, encoding.go, nullable.go, etc.) generates MarshalJSON independently. When multiple apply to the same message, they conflict.

**How to avoid:** The generator must detect ALL applicable transforms for a message and generate a single combined MarshalJSON that handles all of them. Currently this is mitigated because different feature combinations rarely co-occur on the same message. But oneof discriminator makes it more likely since the discriminator applies to the whole message while other annotations apply to individual fields.

**Warning signs:** Compilation errors on generated code with "method redeclared" for MarshalJSON.

**Recommended approach for Phase 7:** For now, generate oneof_discriminator and flatten in their own files (like unwrap.go, nullable.go). If a message has BOTH oneof discriminator AND another encoding feature, the generation should fail with a clear error: "message X has both oneof_discriminator and [other feature] -- this combination is not yet supported." This defers the MarshalJSON merging problem to a future phase while being explicit. Alternatively, check the codebase to see if any existing test messages trigger multiple MarshalJSON -- if not, the risk is low.

### Pitfall 2: Field Name Collisions in Flattened Oneofs

**What goes wrong:** Two oneof variants might have fields with the same JSON name, or a variant field might collide with the discriminator name, or with parent message fields.

**Why it happens:** Flattening merges fields from different namespaces into one flat object.

**How to avoid:** Validate ALL of these at generation time:
1. Discriminator name vs. parent message field JSON names
2. Discriminator name vs. each variant's field JSON names (when flatten=true)
3. Variant field JSON names vs. parent message field JSON names (when flatten=true)
4. Variant field JSON names vs. other variant field JSON names (when flatten=true) -- this is actually OK since only one variant is set at a time, but worth noting

**Warning signs:** `return fmt.Errorf(...)` in the validation functions.

### Pitfall 3: Unset Oneof Serialization Inconsistency

**What goes wrong:** Different generators produce different JSON when no oneof variant is set.

**Why it happens:** Proto3's default is to omit unset fields. With a discriminator, do you omit the discriminator too? Or emit `{"type": ""}` or `{"type": null}`?

**How to avoid:** Decision: When no variant is set, omit the discriminator field entirely. This matches proto3 conventions (unset = absent). Document this clearly. All 4 generators must implement identical behavior.

### Pitfall 4: Flatten Collision Between Sibling Flattened Fields

**What goes wrong:** Two flattened fields at the same level have child fields with the same JSON name (e.g., both have a `name` field).

**Why it happens:** Without `flatten_prefix`, promoting fields from two different message types creates naming conflicts.

**How to avoid:** Validate at generation time. If two flatten fields produce the same JSON key, emit a generation error suggesting `flatten_prefix`.

### Pitfall 5: Proto Oneof with Only Scalar Fields (No Message Variants)

**What goes wrong:** A oneof might contain only scalar fields like `string text = 1; int32 count = 2;`. Flattening makes no sense here because scalars have no sub-fields to promote.

**Why it happens:** The discriminator annotation is valid on any oneof, but flatten only makes sense when variants are message types.

**How to avoid:** When `flatten = true`, validate that ALL oneof variant fields are message types. If any variant is a scalar, emit a generation error: "oneof_flatten requires all variants to be message types."

### Pitfall 6: Interaction Between Flattened Fields and Existing Annotations

**What goes wrong:** A field inside a flattened message has `int64_encoding=NUMBER`. When promoted to the parent, does the encoding still apply?

**Why it happens:** Per CONTEXT.md decision: "Existing per-field annotations travel with their fields when promoted/flattened."

**How to avoid:** This should work naturally because the generated MarshalJSON for the parent message will delegate to the child message's serialization. The child message's own MarshalJSON (if it has one for int64 encoding) will handle the field correctly. However, if the parent message generates its own MarshalJSON for flattening AND the child has its own MarshalJSON, we need to ensure the child's custom serialization is used. This works because we'll marshal the child with `protojson.Marshal` or via `json.Marshaler` interface check, then merge its fields into the parent's map.

## Code Examples

### Example 1: Annotation Extraction for Oneof (internal/annotations/oneof.go)

```go
package annotations

import (
    "google.golang.org/protobuf/compiler/protogen"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/descriptorpb"

    "github.com/SebastienMelki/sebuf/http"
)

// OneofDiscriminatorInfo holds parsed oneof discriminator configuration.
type OneofDiscriminatorInfo struct {
    Oneof         *protogen.Oneof
    Discriminator string         // JSON field name for discriminator (e.g., "type")
    Flatten       bool           // Whether to flatten variant fields
    Variants      []OneofVariant // Resolved variant info
}

// OneofVariant holds information about a single oneof variant.
type OneofVariant struct {
    Field            *protogen.Field
    DiscriminatorVal string          // Value for this variant (field name or custom)
    IsMessage        bool            // Whether variant is a message type
}

// GetOneofConfig returns the oneof config for a oneof, or nil if not annotated.
func GetOneofConfig(oneof *protogen.Oneof) *http.OneofConfig {
    options := oneof.Desc.Options()
    if options == nil {
        return nil
    }
    oneofOptions, ok := options.(*descriptorpb.OneofOptions)
    if !ok {
        return nil
    }
    ext := proto.GetExtension(oneofOptions, http.E_OneofConfig)
    if ext == nil {
        return nil
    }
    config, ok := ext.(*http.OneofConfig)
    if !ok || config == nil {
        return nil
    }
    if config.GetDiscriminator() == "" {
        return nil // No discriminator = no config
    }
    return config
}

// GetOneofVariantValue returns the custom discriminator value for a oneof field,
// or empty string if not set (caller should use the proto field name as default).
func GetOneofVariantValue(field *protogen.Field) string {
    options := field.Desc.Options()
    if options == nil {
        return ""
    }
    fieldOptions, ok := options.(*descriptorpb.FieldOptions)
    if !ok {
        return ""
    }
    ext := proto.GetExtension(fieldOptions, http.E_OneofValue)
    if ext == nil {
        return ""
    }
    value, ok := ext.(string)
    if !ok {
        return ""
    }
    return value
}
```

### Example 2: Collision Detection (internal/annotations/oneof.go)

```go
// ValidateOneofDiscriminator validates a oneof with discriminator annotation.
// Returns error if field name collisions are detected.
func ValidateOneofDiscriminator(message *protogen.Message, oneof *protogen.Oneof, config *http.OneofConfig) error {
    discriminator := config.GetDiscriminator()

    // Check discriminator vs parent message fields
    for _, field := range message.Fields {
        if field.Oneof == oneof {
            continue // Skip oneof's own fields
        }
        if field.Desc.JSONName() == discriminator {
            return fmt.Errorf(
                "oneof %s.%s: discriminator name %q collides with field %q",
                message.Desc.Name(), oneof.Desc.Name(), discriminator, field.Desc.Name(),
            )
        }
    }

    if config.GetFlatten() {
        // Check variant fields vs discriminator and parent fields
        parentFields := collectParentJSONNames(message, oneof)
        parentFields[discriminator] = true // discriminator is also reserved

        for _, variantField := range oneof.Fields {
            if variantField.Message == nil {
                return fmt.Errorf(
                    "oneof %s.%s with flatten=true: variant %q must be a message type",
                    message.Desc.Name(), oneof.Desc.Name(), variantField.Desc.Name(),
                )
            }
            for _, childField := range variantField.Message.Fields {
                childJSON := childField.Desc.JSONName()
                if parentFields[childJSON] {
                    return fmt.Errorf(
                        "oneof %s.%s with flatten=true: variant %q field %q (JSON: %q) collides with parent field or discriminator",
                        message.Desc.Name(), oneof.Desc.Name(), variantField.Desc.Name(),
                        childField.Desc.Name(), childJSON,
                    )
                }
            }
        }
    }

    return nil
}
```

### Example 3: TypeScript Discriminated Union Type Generation

```typescript
// For flattened oneof with discriminator "type":
export type EventContent =
  | { type: "text"; body: string; }
  | { type: "img"; url: string; width: number; height: number; };

export interface Event {
  id: string;
  content: EventContent;  // or flattened directly into Event
}

// If flatten=true, Event itself becomes:
export type Event =
  | { id: string; type: "text"; body: string; }
  | { id: string; type: "img"; url: string; width: number; height: number; };
```

### Example 4: OpenAPI Discriminated Union Schema

```yaml
# Flattened discriminated union
Event:
  oneOf:
    - $ref: '#/components/schemas/Event_text'
    - $ref: '#/components/schemas/Event_img'
  discriminator:
    propertyName: type
    mapping:
      text: '#/components/schemas/Event_text'
      img: '#/components/schemas/Event_img'

Event_text:
  type: object
  required: [type]
  properties:
    id:
      type: string
    type:
      type: string
      enum: [text]
    body:
      type: string

Event_img:
  type: object
  required: [type]
  properties:
    id:
      type: string
    type:
      type: string
      enum: [img]
    url:
      type: string
    width:
      type: integer
    height:
      type: integer
```

### Example 5: Flatten Annotation Extraction (internal/annotations/flatten.go)

```go
// IsFlattenField returns true if the field has flatten=true annotation.
func IsFlattenField(field *protogen.Field) bool {
    // Same pattern as IsNullableField, HasUnwrapAnnotation, etc.
    options := field.Desc.Options()
    if options == nil { return false }
    fieldOptions, ok := options.(*descriptorpb.FieldOptions)
    if !ok { return false }
    ext := proto.GetExtension(fieldOptions, http.E_Flatten)
    if ext == nil { return false }
    flatten, ok := ext.(bool)
    return ok && flatten
}

// GetFlattenPrefix returns the flatten prefix for a field, or empty string.
func GetFlattenPrefix(field *protogen.Field) string {
    options := field.Desc.Options()
    if options == nil { return "" }
    fieldOptions, ok := options.(*descriptorpb.FieldOptions)
    if !ok { return "" }
    ext := proto.GetExtension(fieldOptions, http.E_FlattenPrefix)
    if ext == nil { return "" }
    prefix, ok := ext.(string)
    if !ok { return "" }
    return prefix
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| proto3 canonical oneof JSON | Discriminated union with type field | REST API standard (Stripe, GitHub, AWS) | sebuf can generate the standard REST pattern |
| Nested message wrappers | Flat JSON with prefixed fields | Common REST API pattern | Reduces JSON nesting depth |
| ts-proto `$case` (type-only, not wire) | Wire-level discriminator | sebuf Phase 7 | Actual JSON wire format support, not just TS types |

**Key differentiator:** No existing protobuf tool generates wire-level discriminated unions from oneof fields. ts-proto's `$case` is TypeScript-only and doesn't change the JSON format.

## Open Questions

1. **MarshalJSON merge strategy for messages with multiple encoding features**
   - What we know: Currently each feature generates its own MarshalJSON. No existing test message triggers multiple MarshalJSON.
   - What's unclear: When a message has both oneof discriminator AND (e.g.) nullable fields, how to generate a single combined MarshalJSON.
   - Recommendation: For Phase 7, emit a generation-time error if a message needs both oneof/flatten MarshalJSON AND another feature's MarshalJSON. Defer MarshalJSON merging to a future phase. This is pragmatic since the combination is unlikely in practice.

2. **Unset oneof discriminator: omit or emit empty?**
   - What we know: Proto3 convention is to omit unset fields. protojson omits unset oneofs entirely.
   - Recommendation: Omit the discriminator field entirely when no variant is set. This is consistent with proto3 semantics and avoids ambiguity. All 4 generators must agree on this.

3. **Non-flattened discriminated union JSON shape**
   - What we know: User decided non-flattened keeps variant nested under its field name with discriminator identifying which key is populated.
   - Recommendation: Non-flattened with discriminator produces `{"type": "text", "text": {"body": "hello"}}`. The discriminator is added alongside the standard proto oneof encoding. This is backward-compatible with existing proto3 JSON consumers that ignore unknown fields.

4. **Flatten recursion depth**
   - What we know: User marked this as Claude's discretion. Research says single-level is sufficient.
   - Recommendation: Single-level only. If someone needs deeper flattening, they can annotate at each level. This matches the common REST API pattern and avoids complexity.

## Discretion Recommendations

Based on the CONTEXT.md "Claude's Discretion" items:

### 1. Couple oneof_discriminator and oneof_flatten: YES, couple them

Use a single `OneofConfig` message with both `discriminator` and `flatten` fields. Reasons:
- Fewer annotation types to maintain
- Flatten without discriminator makes no sense (you need to know which variant's fields are present)
- Discriminator without flatten is still useful (adds type field to standard proto oneof JSON)
- Follows the pattern of `OneofConfig` message rather than two separate annotations

### 2. Unset oneof: omit discriminator entirely

When no variant is set:
- Go: omit discriminator from JSON map (don't add "type" key at all)
- TypeScript: field is absent from response
- OpenAPI: discriminator property not marked as required (it's conditional)
- Rationale: Matches proto3 "omit default/unset" convention. Consistent across generators.

### 3. Flatten recursion depth: single-level only

- `flatten = true` on a message field promotes direct child fields only
- If child also has nested messages, those remain nested
- To flatten deeper, user explicitly annotates deeper fields
- Rationale: Simpler implementation, fewer edge cases, matches REST API conventions

### 4. OpenAPI representation

- Discriminated union: `oneOf` with `discriminator` keyword (direct libopenapi support)
- Flattened structures: Each variant becomes an inline schema with `allOf` combining parent common fields + discriminator const + variant fields
- Non-flattened: Standard `oneOf` with discriminator mapping

### 5. Go marshaling implementation approach

Follow the established pattern:
1. Use `protojson.Marshal(x)` for base serialization
2. Parse into `map[string]json.RawMessage`
3. Delete standard oneof fields
4. Add discriminator and (if flatten) merge variant fields
5. `json.Marshal(raw)` for final output

For unmarshal:
1. Parse into `map[string]json.RawMessage`
2. Read discriminator value
3. Switch on discriminator to determine variant type
4. Extract variant fields, construct variant proto message
5. Remove oneof-related fields, re-marshal remainder for protojson

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/httpgen/unwrap.go`, `internal/httpgen/encoding.go`, `internal/httpgen/nullable.go`, `internal/httpgen/enum_encoding.go` -- established patterns for MarshalJSON/UnmarshalJSON generation
- Codebase analysis: `internal/annotations/*.go` -- shared annotation extraction pattern
- Codebase analysis: `internal/openapiv3/types.go`, `internal/openapiv3/generator.go` -- OpenAPI schema generation patterns
- [protogen package documentation](https://pkg.go.dev/google.golang.org/protobuf/compiler/protogen) -- Message.Oneofs, Oneof.Fields, Field.Oneof API
- [libopenapi base.Schema](https://pkg.go.dev/github.com/pb33f/libopenapi/datamodel/high/base#Schema) -- Discriminator field, OneOf field, AllOf field
- [Go Generated Code Guide - oneof](https://protobuf.dev/reference/go/go-generated/) -- interface pattern, wrapper types, type switch

### Secondary (MEDIUM confidence)
- [OpenAPI discriminator usage](https://redocly.com/learn/openapi/discriminator) -- propertyName, mapping structure
- [protobuf OneofOptions extension](https://protobuf.dev/programming-guides/proto3/) -- custom option extension pattern for oneofs
- Codebase prior research: `.planning/research/PITFALLS.md` (Section 5), `.planning/research/FEATURES.md` (Section #90) -- complexity analysis and edge cases

### Tertiary (LOW confidence)
- [ts-proto oneof unions](https://github.com/stephenh/ts-proto/issues/314) -- alternative approach comparison (TypeScript-only $case, not wire format)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all tools already in use, no new dependencies needed
- Architecture: HIGH -- follows established patterns from 6+ existing encoding features in the codebase
- Pitfalls: HIGH -- well-documented in prior research and verified through codebase analysis
- OpenAPI discriminator: HIGH -- libopenapi has native `Discriminator` struct support verified in API docs
- Oneof protogen API: HIGH -- `Message.Oneofs`, `Oneof.Fields`, `Field.Oneof` verified in protogen docs

**Research date:** 2026-02-06
**Valid until:** 2026-03-08 (stable domain, no fast-moving dependencies)
