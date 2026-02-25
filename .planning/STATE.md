# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-25)

**Core value:** Proto definitions are the single source of truth -- every generator (server, client, docs, gateway) must produce consistent, correct output that interoperates seamlessly.
**Current focus:** Phase 12 - Annotations and Core Endpoint Generation (v1.1 KrakenD)

## Current Position

Phase: 12 of 14 (Annotations and Core Endpoint Generation)
Plan: 4 of 4 in current phase
Status: Phase Complete
Last activity: 2026-02-25 -- Completed 12-04 (Validation and Golden Tests)

Progress: [##########] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 4 (v1.1 milestone)
- Average duration: 4min
- Total execution time: 18min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 12 | 4 | 18min | 4min |

**Recent Trend:**
- Last 5 plans: 12-01 (4min), 12-02 (6min), 12-03 (3min), 12-04 (5min)
- Trend: consistent

*Updated after each plan completion*

## Accumulated Context

### Decisions

- KrakenD annotations in separate proto package (sebuf.krakend) -- gateway config is a different concern than HTTP API shape
- Per-service endpoint fragments (not monolithic config) -- matches sebuf pattern, composable via KrakenD FC
- Reuse existing sebuf.http routing annotations -- KrakenD needs same path/method/params info
- Per-RPC config can also be set at service level; per-RPC always overrides per-service (timeouts, rate limits, circuit breakers)
- Extension numbers 51001/51002 for krakend annotations, above sebuf.http range (50003-50020)
- Plugin outputs empty JSON array per service as minimal placeholder until Plan 02
- Only require gateway_config when service has HTTP-annotated RPCs -- bare services produce empty array
- Timeout omitted from JSON via omitempty when not annotated at any level
- Nil endpoint slice normalized to empty slice for JSON [] output
- Reuse annotations.CombineHeaders for header merge in KrakenD -- method overrides service for same-name headers
- Return nil (not empty slice) for empty forwarding lists -- omitempty omits from JSON (FWD-03)
- Sort all forwarding lists for deterministic golden file comparison
- Path segment trie for route conflict detection -- simple recursive structure, efficient for typical API route counts
- Error messages reference endpoint indices (not RPC names) since Endpoint struct does not carry RPC name metadata

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-25
Stopped at: Completed 12-04-PLAN.md (Phase 12 complete)
Resume file: .planning/phases/12-annotations-and-core-endpoint-generation/12-04-SUMMARY.md
Next: Phase 13 planning
