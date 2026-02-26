# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-25)

**Core value:** Proto definitions are the single source of truth -- every generator (server, client, docs, gateway) must produce consistent, correct output that interoperates seamlessly.
**Current focus:** Phase 14 - Documentation and Examples (v1.1 KrakenD)

## Current Position

Phase: 14 of 14 (Documentation and Examples)
Plan: 3 of 3 in current phase
Status: All Plans Complete
Last activity: 2026-02-25 -- Completed 14-02 (KrakenD Gateway Example)

Progress: [##########] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 10 (v1.1 milestone)
- Average duration: 4min
- Total execution time: 47min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 12 | 4 | 18min | 4min |
| 13 | 3 | 12min | 4min |
| 14 | 3 | 17min | 5min |

**Recent Trend:**
- Last 5 plans: 13-02 (3min), 13-03 (4min), 14-01 (5min), 14-03 (4min), 14-02 (8min)
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
- KrakenDConfig wrapper struct lives in types.go alongside Endpoint/Backend -- keeps all output types co-located
- ExtraConfig is map[string]any with omitempty -- nil maps omitted from JSON so existing golden files unaffected
- resolve/build pattern for extra_config: resolveX picks service or method level, buildXConfig creates map for namespace
- Rate limit int32 fields stored as int32 in map (not float64) -- Go json.Marshal handles correctly
- JWT is service-level only -- same auth config on every endpoint, no method-level override
- Propagated claim headers auto-added to input_headers with dedup and sort for KrakenD zero-trust model
- propagate_claims serialized as array-of-arrays per KrakenD spec, not array of objects
- JWT cache field only included when true (false is default, omitted)
- Circuit breaker and cache configs validated before endpoint loop for fail-fast on invalid service-level config
- Method-level circuit breaker and cache validated per-endpoint inside the loop
- Circuit breaker int32 fields stored as int32 in config map, consistent with rate limit pattern
- Enum-to-string mapping via explicit switch statements for clarity and compile-time safety
- Cache shared+max_items/max_size validation added before existing pairing check for fail-fast
- krakend check -lc test skips gracefully when CLI not installed for CI compatibility
- KrakenD section in README placed after How It Works and before Quick Setup for natural reading flow
- README kept concise as teaser driving users to krakend-gateway example
- Use protoc directly in krakend-gateway example because krakend proto not yet on BSR
- FC partials extracted via jq + sed to strip outer array brackets for Flexible Config include
- krakend check -d output is human-readable debug, not JSON dump -- compose validates only

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-25
Stopped at: Completed 14-02-PLAN.md (all phase 14 plans complete)
Resume file: .planning/phases/14-documentation-and-examples/14-02-SUMMARY.md
Next: Phase 14 complete. All v1.1 milestone plans executed.
