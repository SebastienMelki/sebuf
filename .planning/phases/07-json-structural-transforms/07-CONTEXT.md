# Phase 7: JSON - Structural Transforms - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Oneof discriminated unions and nested message flattening in JSON output across all 4 generators (go-http, go-client, ts-client, openapiv3). These are structural transforms that change the shape of JSON — not new field types or new transport features.

</domain>

<decisions>
## Implementation Decisions

### Oneof discriminator values
- Default discriminator value = the proto field name (e.g., `text`, `image`) — matches what developers write in .proto files
- Optional per-variant `oneof_value` annotation for custom discriminator strings (e.g., `"txt"` instead of `"text"`)
- This mirrors the Phase 4 pattern: `enum_value` for custom enum strings, field name as sensible default

### Oneof flatten behavior
- `oneof_flatten = true` promotes variant message fields to the same level as the discriminator (flat discriminated union pattern)
- Without flatten, variant stays nested under its field name with discriminator identifying which key is populated
- Whether discriminator and flatten are coupled or independent is Claude's discretion — optimize for simplicity and fewer test combinations

### Unset oneof handling
- Claude's discretion on what JSON looks like when no oneof variant is set
- Should follow protojson conventions and be consistent across all generators

### Nested message flattening
- Single-level flatten (`flatten = true`) promotes child message fields to parent level in JSON
- `flatten_prefix` prepends a prefix to avoid collisions (e.g., `flatten_prefix = "billing_"` produces `billing_street`)
- Recursive flattening depth is Claude's discretion — single-level is likely sufficient

### Error reporting
- Field name collisions between discriminator and variant fields → generation-time error (fatal)
- Flatten field name collisions with parent fields → generation-time error (fatal)
- No silent runtime failures — consistent with project principle ("two capital sins: breaking backward compat and inconsistencies")
- Follows Phase 4 pattern: errors/warnings at generation time, not runtime

### Annotation interaction
- Existing per-field annotations (nullable, int64_encoding, enum_encoding, timestamp_format, bytes_encoding, empty_behavior) travel with their fields when promoted/flattened
- No special interaction rules needed — annotations apply to individual fields regardless of structural position
- Composability is natural: a nullable int64 field inside a flattened message stays nullable int64 when promoted

### Claude's Discretion
- Whether `oneof_discriminator` and `oneof_flatten` are coupled or independent annotations
- Unset oneof JSON representation
- Flatten recursion depth (single-level vs recursive)
- OpenAPI representation details for discriminated unions (discriminator keyword usage) and flattened structures (allOf usage)
- Implementation approach for the structural transform in Go marshaling/unmarshaling

</decisions>

<specifics>
## Specific Ideas

- User wants to move quickly through this phase to reach language client generation (Phases 8-10)
- Keep pragmatic — favor simpler implementation that covers the common cases well over exhaustive edge case handling
- The established pattern from previous phases should guide implementation: annotation in proto, shared annotations package, identical encoding in httpgen/clientgen, TypeScript equivalent, OpenAPI schema representation, cross-generator consistency test

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-json-structural-transforms*
*Context gathered: 2026-02-06*
