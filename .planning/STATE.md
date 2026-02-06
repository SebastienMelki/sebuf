# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-05)

**Core value:** Proto definitions are the single source of truth -- every generator must produce consistent, correct output that interoperates seamlessly.
**Current focus:** Phase 6 complete -- JSON Data Encoding (timestamp formats, bytes encoding, cross-generator consistency).

## Current Position

Phase: 6 of 11 (JSON - Data Encoding)
Plan: 4 of 4 in current phase
Status: Phase complete
Last activity: 2026-02-06 -- Completed 06-04-PLAN.md (cross-generator consistency tests)

Progress: [#########################] 100% (25 plans of ~25 estimated total)

## Performance Metrics

**Velocity:**
- Total plans completed: 25
- Average duration: ~6.2m
- Total execution time: ~2.6 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 - Foundation Quick Wins | 2/2 | ~17m | ~8.5m |
| 02 - Shared Annotations | 4/4 | ~26m | ~6.5m |
| 03 - Existing Client Review | 6/6 | ~36m | ~6.0m |
| 04 - JSON Primitive Encoding | 5/5 | ~65m | ~13.0m |
| 05 - JSON Nullable & Empty | 4/4 | ~21m | ~5.3m |
| 06 - JSON Data Encoding | 4/4 | ~30m | ~7.5m |

**Recent Trend:**
- Last 5 plans: 05-04 (4m), 06-01 (3m), 06-02 (15m), 06-03 (8m), 06-04 (4m)
- Trend: Consistency-only plans (no impl) complete fastest; cross-generator impl plans take longer

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: Foundation refactoring (shared annotations) must precede all JSON mapping work
- Roadmap: Existing Go and TS clients must be reviewed/polished before any new features (Phase 3)
- Roadmap: Cross-generator consistency validation is mandatory in every JSON mapping and language phase
- Roadmap: Two capital sins: breaking backward compatibility, and inconsistencies between docs/clients/servers
- Roadmap: JSON-01 (nullable) must precede JSON-06 (empty objects) due to dependency
- Roadmap: Language clients (Phases 8-10) parallelizable after Phase 7 completes
- Roadmap: JSON-08 (nested flattening) kept in v1 scope despite research suggesting deferral
- D-01-01-01: Two-pass generation pattern for cross-file unwrap (collect all unwrap info globally first, then generate per-file)
- D-01-01-02: Preserve root unwrap functionality while adding cross-file resolution
- D-02-01-01: Transparent structs with protogen parameters -- all exported structs have exported fields, all functions accept protogen types
- D-02-01-02: Unified QueryParam struct with all 7 fields from all 4 generators (FieldName, FieldGoName, FieldJSONName, ParamName, Required, FieldKind, Field)
- D-02-01-03: Two unwrap APIs -- GetUnwrapField (full validation) and FindUnwrapField (simple lookup) for different generator needs
- D-02-01-04: Convention-based extensibility -- one file per annotation concept, GetXxx() function signatures
- D-02-02-01: Dead code removal -- parseExistingAnnotation removed during migration (always returned empty string)
- D-02-02-02: Test deduplication -- httpgen annotation tests removed since covered by shared package
- D-02-03-01: BuildHTTPPath safe for both generators -- httpPath always initialized before path building
- D-02-03-02: Generator-specific naming helpers kept in respective packages (snakeToUpperCamel, snakeToLowerCamel, headerNameToPropertyName)
- D-02-04-01: Lowercase HTTP method constants in openapiv3 -- OpenAPI requires lowercase, shared package returns uppercase, resolved with strings.ToLower() + local constants
- D-02-04-02: OpenAPI-specific functions (convertHeadersToParameters, mapHeaderTypeToOpenAPI) stay in openapiv3/types.go, not shared package
- D-02-04-03: Cross-file error propagation -- 5 functions changed to return errors, fail-hard up to Generator.Generate()
- D-02-04-04: Serialization audit confirmed no changes needed -- encoding/json correctly used for interface checks only
- D-03-02-01: JSON default for unknown content types everywhere -- bindDataBasedOnContentType, marshalResponse, writeProtoMessageResponse, writeResponseBody all default to JSON
- D-03-02-02: Content-Type set in three response-writing functions covering all paths: writeProtoMessageResponse, genericHandler success path, writeResponseBody
- D-03-01-01: Added UnwrapService to httpgen unwrap.proto (alongside OptionDataService) for cross-generator root-level unwrap testing
- D-03-01-02: Root-level unwrap RPCs use POST method (not GET) to satisfy httpgen GET-with-body validation
- D-03-01-03: Proto3 optional support added to go-http and go-client plugins via SupportedFeatures declaration
- D-03-03-01: Go client already consistent with server - no fixes needed (audit verified 6 key areas: query params, Content-Type, errors, path params, headers, unwrap)
- D-03-04-01: TS client already consistent with Go server - no fixes needed (int64 as string, query encoding, FieldViolation fields, header handling, all 4 unwrap variants)
- D-03-04-02: No JSDoc generation by design - minimalist generated code
- D-03-05-01: Error schema uses single 'message' field matching sebuf.http.Error proto (not error+code)
- D-03-05-02: int64/uint64 mapped to type:string per proto3 JSON spec for JavaScript precision safety
- D-03-05-03: Added headerTypeUint64 constant and removed minimum constraint since uint64 is now string type
- D-03-06-01: Default path inconsistency for services without HTTP annotations is accepted (backward compat fallback mode only)
- D-03-06-02: Cross-generator consistency verified for all 10 key areas (paths, methods, params, schemas, errors, headers, unwrap)
- D-04-01-01: Extension numbers 50010-50012 continue sequence from existing 50009 (unwrap)
- D-04-01-02: UNSPECIFIED (0) always means "use protojson default" - explicit STRING value available for documentation
- D-04-01-03: GetEnumValueMapping returns empty string (not nil) for consistency with Go string semantics
- D-04-03-01: tsScalarTypeForField pattern - keep base tsScalarType unchanged, add encoding-aware variant
- D-04-03-02: appendInt64PrecisionWarning called after description set - ensures comment text + warning combined
- D-04-03-03: nolint directives for valid lint warnings - exhaustive (has default), funlen (big switch), nestif (existing pattern)
- D-04-02-01: Use protojson for base serialization, then modify map for NUMBER fields - preserves all other field handling
- D-04-02-02: Print precision warning to stderr during generation, not at runtime - developer sees during build
- D-04-02-03: Identical encoding.go implementation in httpgen and clientgen - guarantees server/client JSON match
- D-04-04-01: Separate enum_encoding.go files in httpgen/clientgen to avoid import conflicts with int64 encoding.go
- D-04-04-02: Both proto name and custom value accepted in UnmarshalJSON for backward compatibility
- D-04-04-03: NUMBER encoding returns 'number' type in TypeScript, 'integer' type in OpenAPI
- D-04-05-01: Split TestEncodingConsistencyAcrossGenerators into separate test functions for linting compliance
- D-04-05-02: Use normalizeGeneratorComment to allow byte-level comparison between go-http and go-client
- D-04-05-03: Convert openapiv3 enum_encoding.proto from duplicate file to symlink for consistency
- D-05-01-01: Extension numbers 50013 (nullable) and 50014 (empty_behavior) continue sequence from 50012
- D-05-01-02: UNSPECIFIED (0) means default behavior (same as PRESERVE for empty_behavior)
- D-05-02-01: Identical nullable.go in httpgen and clientgen for server/client JSON consistency
- D-05-02-02: Nullable TypeScript fields use T | null (not optional ?) - always present with value or null
- D-05-02-03: OpenAPI 3.1 type array syntax [T, null] instead of deprecated nullable: true
- D-05-02-04: Nullable encoding placed before service check in clientgen (like httpgen) for message-only files
- D-05-03-01: Identical empty_behavior.go in httpgen and clientgen for server/client JSON consistency
- D-05-03-02: OpenAPI oneOf schema for NULL fields ({$ref} | {type: null}) instead of deprecated nullable:true
- D-05-03-03: OMIT fields use standard $ref in OpenAPI (serialization-only behavior, schema unchanged)
- D-05-03-04: Exhaustive switch for EmptyBehavior enum to satisfy linter
- D-05-04-01: Added empty_behavior test proto to clientgen (Rule 3 deviation) to enable byte-level golden file comparison
- D-06-01-01: Extension numbers 50015-50016 continue sequence from existing 50014 (empty_behavior)
- D-06-01-02: UNSPECIFIED (0) always means protojson default -- RFC3339 for timestamps, BASE64 for bytes
- D-06-01-03: HasTimestampFormatAnnotation excludes both UNSPECIFIED and RFC3339 (both produce default behavior)
- D-06-01-04: HasBytesEncodingAnnotation excludes both UNSPECIFIED and BASE64 (both produce default behavior)
- D-06-02-01: Timestamp detected before generic MessageKind in type switches to prevent $ref generation
- D-06-02-02: google.protobuf.Timestamp skipped from tsclientgen messageSet (primitive, not nested object)
- D-06-02-03: convertTimestampField helper in openapiv3 for clean format-to-schema mapping
- D-06-02-04: nolint:exhaustive on tsTimestampType switch -- default handles RFC3339/DATE/UNSPECIFIED
- D-06-03-01: HEX UnmarshalJSON needs both encoding/hex AND encoding/base64 imports (re-encodes decoded hex as standard base64 for protojson)
- D-06-03-02: nolint:dupl on MarshalJSON/UnmarshalJSON across empty_behavior, timestamp_format, bytes_encoding (three similar files trigger dupl threshold)
- D-06-03-03: OpenAPI HEX uses format:hex with regex pattern ^[0-9a-fA-F]*$ for validation
- D-06-03-04: OpenAPI BASE64URL uses format:base64url (not base64 with modifier) for clarity

### Pending Todos

None.

### Blockers/Concerns

- Research flags Phase 7 JSON-04 (oneof discriminated union) as HIGH complexity -- may need deeper research during planning

## Session Continuity

Last session: 2026-02-06
Stopped at: Completed 06-04-PLAN.md (Phase 6 complete)
Resume file: None
Next: Phase 7 planning (JSON Complex Types)
