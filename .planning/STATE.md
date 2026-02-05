# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-05)

**Core value:** Proto definitions are the single source of truth -- every generator must produce consistent, correct output that interoperates seamlessly.
**Current focus:** Phase 2 - Shared Annotations (plan 02 complete, httpgen migrated)

## Current Position

Phase: 2 of 11 (Foundation - Shared Annotations)
Plan: 2 of 4 in current phase
Status: In progress
Last activity: 2026-02-05 -- Completed 02-02-PLAN.md (httpgen migration to shared annotations)

Progress: [####.......] 18% (4 plans of ~22 estimated total)

## Performance Metrics

**Velocity:**
- Total plans completed: 4
- Average duration: ~7m
- Total execution time: ~0.5 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 - Foundation Quick Wins | 2/2 | ~17m | ~8.5m |
| 02 - Shared Annotations | 2/4 | ~11m | ~5.5m |

**Recent Trend:**
- Last 5 plans: 01-01 (7m), 01-02 (~10m), 02-01 (5m), 02-02 (6m)
- Trend: Consistent, accelerating

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

### Pending Todos

None.

### Blockers/Concerns

- Research flags Phase 7 JSON-04 (oneof discriminated union) as HIGH complexity -- may need deeper research during planning

## Session Continuity

Last session: 2026-02-05
Stopped at: Completed 02-02-PLAN.md. Ready for 02-03-PLAN.md (clientgen + tsclientgen + openapiv3 migration).
Resume file: None
