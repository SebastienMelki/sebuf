# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-05)

**Core value:** Proto definitions are the single source of truth -- every generator must produce consistent, correct output that interoperates seamlessly.
**Current focus:** Phase 3 (Existing Client Review) -- shared test proto infrastructure established, fixing client correctness issues.

## Current Position

Phase: 3 of 11 (Existing Client Review)
Plan: 3 of 6 in current phase (03-01, 03-02, and 03-03 complete)
Status: In progress
Last activity: 2026-02-05 -- Completed 03-03-PLAN.md (Go Client Consistency Audit)

Progress: [#########..] 41% (9 plans of ~22 estimated total)

## Performance Metrics

**Velocity:**
- Total plans completed: 9
- Average duration: ~6.3m
- Total execution time: ~0.95 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 - Foundation Quick Wins | 2/2 | ~17m | ~8.5m |
| 02 - Shared Annotations | 4/4 | ~26m | ~6.5m |
| 03 - Existing Client Review | 3/6 | ~17m | ~5.7m |

**Recent Trend:**
- Last 5 plans: 02-04 (10m), 03-02 (4m), 03-01 (10m), 03-03 (3m)
- Trend: Consistent, audit/verification tasks fastest

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

### Pending Todos

None.

### Blockers/Concerns

- Research flags Phase 7 JSON-04 (oneof discriminated union) as HIGH complexity -- may need deeper research during planning

## Session Continuity

Last session: 2026-02-05
Stopped at: Completed 03-03-PLAN.md (Go Client Consistency Audit)
Resume file: None
