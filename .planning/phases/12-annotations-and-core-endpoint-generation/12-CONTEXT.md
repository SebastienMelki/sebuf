# Phase 12: Annotations and Core Endpoint Generation - Context

**Gathered:** 2026-02-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Generate KrakenD API gateway endpoint fragments from proto service definitions via a new `protoc-gen-krakend` plugin. Covers: proto annotation package (sebuf.krakend), plugin scaffold, endpoint/backend generation with host and timeout config, auto-derived header and query string forwarding from existing sebuf.http annotations, and route conflict validation. Gateway features (rate limiting, JWT, circuit breaker) are Phase 13.

</domain>

<decisions>
## Implementation Decisions

### Output format and file structure
- One file per service (e.g., `UserService.krakend.json`) — matches sebuf's existing per-service pattern (OpenAPI generator does this)
- Output JSON shape: Claude's discretion on whether to emit a bare endpoints array or a minimal wrapper — pick what integrates cleanest with KrakenD Flexible Config `{{ include }}` directives
- Pretty-printed JSON (indented) — readable, reviewable in PRs
- Standard protoc output via `--krakend_out=<dir>` — follows existing sebuf plugin conventions

### Annotation design and defaults
- Auto-include all RPCs: every RPC with `sebuf.http.config` automatically generates a KrakenD endpoint — no extra KrakenD annotation required for basic routing
- Host configured at service level via `gateway_config` annotation, overridable at method level via `endpoint_config` — no plugin flag for host, it lives in annotations
- Timeouts: omit if not annotated — let KrakenD apply its own defaults, don't be opinionated
- Annotations live in a separate `sebuf.krakend` proto package — gateway config is a different concern than HTTP API shape (confirmed from STATE.md decisions)
- Per-RPC config overrides per-service config (timeouts, host, etc.) — consistent override semantics

### KrakenD path mapping
- Pass proto HTTP paths through as-is — sebuf uses `{param}` syntax which maps directly to KrakenD
- Backend path mirrors endpoint path — gateway is a passthrough, no remapping
- Service `base_path` from `sebuf.http.service_config` is prepended to endpoint paths — consistent with how the HTTP server generator works
- Output encoding: always `json` — content type is a runtime client decision, not a proto definition concern; KrakenD needs JSON encoding to process payloads for its features

### Error messages and DX
- Fail hard on route conflicts — generation fails with a clear error, no config produced, forces fix before deploy
- Silent on success — no output, matches protoc plugin convention; errors/warnings go to stderr only
- Rich error context — include service name, RPC name, file location, and what to fix (e.g., "user_service.proto: UserService.GetUser and UserService.SearchUsers produce conflicting routes: GET /users/{id} vs GET /users/search")
- Validate KrakenD constraints at generation time — catch known issues (valid namespace strings, constraint violations) before deployment rather than at KrakenD startup

### Claude's Discretion
- Exact JSON structure (bare array vs minimal wrapper) — pick what's cleanest for FC integration
- Annotation proto message design (field names, nesting)
- Plugin architecture and code organization within internal/krakendgen/
- Golden test structure and coverage strategy
- How to detect and report static vs parameterized route conflicts

</decisions>

<specifics>
## Specific Ideas

- "My head of devops loves KrakenD — I want to blow his mind but not go overboard"
- Keep it tight and practical: zero-config should produce useful output from existing sebuf.http annotations alone
- Override semantics should be consistent: service-level sets defaults, method-level overrides, just like the rest of sebuf

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-annotations-and-core-endpoint-generation*
*Context gathered: 2026-02-25*
