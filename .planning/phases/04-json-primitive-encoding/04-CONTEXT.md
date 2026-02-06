# Phase 4: JSON - Primitive Encoding - Context

**Gathered:** 2026-02-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Control how int64/uint64 and enum fields serialize to JSON via field-level annotations. All 4 generators (go-http, go-client, ts-client, openapiv3) must support these annotations consistently. This phase does NOT cover nullable fields, timestamps, bytes encoding, or structural transforms (those are Phases 5-7).

</domain>

<decisions>
## Implementation Decisions

### Annotation Granularity
- **Field-level only** — no message/service/file-level inheritance
- **Separate annotations** — `int64_encoding` for integers, `enum_encoding` for enums (not combined)
- **Collection handling** — Claude's discretion on whether repeated/map int64 fields support the annotation

### Default Behavior
- **int64/uint64 default** — Follow protojson spec: serialize as strings when no annotation present
- **enum default** — Follow protojson spec: serialize as proto name strings (e.g., "STATUS_ACTIVE")
- **Explicit defaults allowed** — Can write `int64_encoding = STRING` even though it's the default (self-documenting)
- **Backward compatible** — Code without annotations must produce identical output to current behavior

### Enum Value Mapping
- **Per-value annotation** — Each enum value can have custom JSON name via `(sebuf.http.enum_value) = "active"`
- **Partial override allowed** — Only annotate values you want to change; others use proto name
- **STRING or NUMBER** — `enum_encoding = STRING` (default) or `enum_encoding = NUMBER` (integer values)
- **Conflict is an error** — Generation fails if both `enum_encoding = NUMBER` and `enum_value` annotations are present on same enum

### Precision Warnings
- **Both warning and comment** — Print warning at generation AND add inline comment in generated code
- **No suppression** — Warning always shown; precision risk is important information
- **OpenAPI description** — Add description noting "Values > 2^53 may lose precision in JavaScript"
- **uint64 identical** — Same `int64_encoding` annotation works for both int64 and uint64 fields

### Claude's Discretion
- Collection field support (repeated/map) for int64_encoding — decide based on technical feasibility
- Exact annotation syntax and proto extension structure
- Internal implementation approach for each generator
- Warning message wording

</decisions>

<specifics>
## Specific Ideas

- int64_encoding and enum_encoding follow existing sebuf annotation patterns (field extension options)
- Precision warning should be actionable: mention STRING as the safe alternative
- OpenAPI should use format: int64 for numbers to hint at the type

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-json-primitive-encoding*
*Context gathered: 2026-02-05*
