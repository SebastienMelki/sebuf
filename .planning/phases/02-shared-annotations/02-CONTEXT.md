# Phase 2: Foundation - Shared Annotations - Context

**Gathered:** 2026-02-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Extract all duplicated annotation parsing code (~1,289 lines across httpgen, clientgen, tsclientgen, openapiv3) into a single `internal/annotations` package. All 4 generators import this shared package for annotation parsing (HTTPConfig, QueryParam, UnwrapInfo, HeaderConfig). Zero behavior change — all existing golden file tests must pass unchanged. The shared package must be designed for extensibility since 8 new JSON mapping annotations arrive in Phases 4-7.

</domain>

<decisions>
## Implementation Decisions

### Package API surface
- Separate functions per annotation type (GetHTTPConfig, GetUnwrapInfo, GetHeaders, etc.) — each generator calls only what it needs
- Convention-based extensibility: each annotation type is a Go file with standard `GetXxx()` / `ParseXxx()` function signatures following a naming pattern, making it straightforward to add new annotation types in Phases 4-7
- Include common helper utilities that multiple generators currently duplicate (e.g., shared functions beyond just annotation parsing) — extract these alongside annotations
- Claude's Discretion: transparent structs vs opaque types — pick what fits the codebase style
- Claude's Discretion: whether to accept protogen types or protoreflect types — pick based on current code patterns
- Claude's Discretion: whether cross-file annotation resolution lives in the shared package or remains in generators — decide based on where duplication actually exists

### Migration approach
- Incremental migration: one generator at a time
- Order: httpgen first (most complex, defines widest API surface), then clientgen, then tsclientgen, then openapiv3
- Delete old annotation parsing code immediately after each generator migrates — no temporary coexistence
- One atomic git commit per generator migration — easy to review and bisect
- Always run `make lint-fix` after each migration step to catch linting issues early

### Error handling contract
- Fail hard on cross-file annotation resolution failures — no partial output, generation stops with a clear error
- Validate annotations at parse time in the shared package — errors caught early with clear messages pointing to the proto file/line
- Every error message must include proto file path + message/field name so the user can jump straight to the problem
- Claude's Discretion: whether to use Go errors or protogen's error reporting — pick based on best developer experience

### Serialization consistency
- Use protojson exclusively — no encoding/json for proto message serialization
- Audit and fix any accidental encoding/json usage for proto messages in the HTTP handler generator

</decisions>

<specifics>
## Specific Ideas

- User emphasized: always run `make lint-fix` periodically during execution to stay clean on linting
- The shared package's convention-based pattern should make it obvious how to add a new annotation type (add a file, follow the naming pattern, done)
- httpgen goes first because it has the widest annotation surface area — the API it demands will cover most of what other generators need

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-shared-annotations*
*Context gathered: 2026-02-05*
