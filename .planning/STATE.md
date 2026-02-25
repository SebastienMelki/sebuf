# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-25)

**Core value:** Proto definitions are the single source of truth -- every generator (server, client, docs, gateway) must produce consistent, correct output that interoperates seamlessly.
**Current focus:** Milestone v1.1 -- KrakenD Config Generator

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-02-25 — Milestone v1.1 started

## Accumulated Context

### Decisions

- KrakenD annotations in separate proto package (sebuf.krakend) -- gateway config is a different concern than HTTP API shape
- Per-service endpoint fragments (not monolithic config) -- matches sebuf pattern, composable via KrakenD FC
- Reuse existing sebuf.http routing annotations -- KrakenD needs same path/method/params info

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-25
Stopped at: Milestone v1.1 initialization
Resume file: None
Next: Define requirements, create roadmap
