# Project Research Summary

**Project:** protoc-gen-krakend (KrakenD API Gateway config generator for sebuf)
**Domain:** Protobuf-to-API-Gateway Configuration Generation
**Researched:** 2026-02-25
**Confidence:** HIGH

## Executive Summary

protoc-gen-krakend is a new protoc plugin that generates KrakenD API gateway endpoint configuration fragments from protobuf service definitions. It is the 6th plugin in the sebuf toolkit, and its architecture is most analogous to protoc-gen-openapiv3: it reads proto annotations, builds structured data (JSON instead of YAML), and outputs per-service files. The critical insight from research is that this plugin requires zero new Go dependencies -- the existing protogen framework, shared annotation parsing, and stdlib `encoding/json` are sufficient. KrakenD config is plain JSON, and the plugin generates endpoint fragments (not a full gateway config), which users compose via KrakenD's Flexible Config template system.

The recommended approach is typed Go structs with `json:"..."` tags marshaled via `json.MarshalIndent()`, reusing `internal/annotations/` for all HTTP routing extraction and adding a new `proto/sebuf/krakend/` package for gateway-specific annotations (rate limiting, JWT auth, circuit breaker). The key value proposition is automatic derivation of KrakenD's `input_headers` and `input_query_strings` from existing sebuf HTTP annotations -- users define headers once in proto, and all generators (HTTP server, clients, OpenAPI, and now KrakenD) stay consistent. This "define once, generate everywhere" capability is what justifies the tool over hand-writing KrakenD JSON.

The top risks are KrakenD-specific routing constraints that differ from standard HTTP routers. KrakenD requires one endpoint object per HTTP method (not per path), panics on path parameter name collisions, and cannot coexist static and parameterized routes at the same path level. All three must be validated at generation time with clear error messages. Additionally, KrakenD's zero-trust header/query forwarding model means the generator must always populate `input_headers` and `input_query_strings` from proto annotations -- omitting them silently breaks authentication and query parameters through the gateway.

## Key Findings

### Recommended Stack

No new Go dependencies. The plugin generates plain JSON using stdlib `encoding/json` with typed Go structs. It reads existing `sebuf.http` annotations via `internal/annotations/` and new `sebuf.krakend` annotations via a new Go package generated from `proto/sebuf/krakend/krakend.proto`.

**Core technologies:**
- **protogen v1.36.11 (existing)**: Plugin framework -- same as all 5 existing sebuf plugins
- **encoding/json (stdlib)**: JSON output -- typed Go structs guarantee valid JSON by construction, golden-testable
- **internal/annotations/ (existing)**: HTTP routing extraction -- reuses GetMethodHTTPConfig, GetServiceHeaders, GetQueryParams, etc.
- **proto/sebuf/krakend/ (new)**: Gateway-specific annotations -- separate package from sebuf.http, extension numbers 51000+

**Key stack decisions:**
- JSON output only (not YAML/TOML) -- KrakenD's `krakend check --lint` only validates JSON
- Per-service endpoint array files, not full krakend.json -- global config is deployment-specific
- Plugin parameters for deployment-specific values (`--krakend_opt=host=http://backend:8080`)
- No KrakenD Go library imports -- define minimal Go structs mirroring KrakenD's JSON schema

### Expected Features

**Must have (table stakes -- T1-T10):**
- **Endpoint routing from proto** (T1) -- core purpose, reuses sebuf.http.config path/method
- **Backend host/URL mapping** (T2) -- each service needs backend targets
- **Per-endpoint timeout** (T3) -- basic QoS control, KrakenD default 2s is often wrong
- **Input header forwarding** (T4) -- KrakenD blocks ALL headers by default; auto-derive from sebuf.http.service_headers/method_headers
- **Input query string forwarding** (T5) -- same zero-trust issue; auto-derive from sebuf.http.query
- **Output encoding** (T6) -- JSON default, no-op for passthrough
- **Per-service file output** (T7) -- consistent with OpenAPI generator pattern
- **Endpoint rate limiting** (T8) -- #1 reason teams adopt API gateways
- **JWT validation** (T9) -- primary open-source auth mechanism in KrakenD
- **Circuit breaker** (T10) -- essential resilience pattern

**Should have (differentiators -- D1-D17):**
- **Auto-derived input_headers/query_strings** (D1, D2) -- the "wow" moment: define once, gateway forwards automatically
- **Backend rate limiting, concurrent calls, caching** (D3, D4, D10) -- backend resilience group
- **Response shaping: allow/deny/mapping/group/target** (D5-D8) -- backend response manipulation
- **JWT claim propagation** (D12) -- forward claims as backend headers
- **Backend error handling** (D14) -- control error passthrough
- **Collection handling** (D15) -- auto-detect from unwrap annotations

**Defer (v2+ or skip entirely):**
- **CORS configuration** (D9) -- root-level only, medium complexity, edge cases in fragment merging
- **CEL validation** (D11) -- niche, backend already validates via protovalidate
- **Sequential proxy** (D13) -- anti-pattern per KrakenD docs
- **Security headers** (D16), **No-op passthrough** (D17) -- simple but low priority

**Anti-features (do NOT build):**
- Full krakend.json generation (A1) -- infrastructure concern, not API design
- API key auth (A2) -- Enterprise-only feature
- gRPC backend integration (A3) -- Enterprise-only, contradicts sebuf's HTTP focus
- Telemetry/logging config (A5) -- operational concern
- JSON Schema validation (A9) -- redundant with protovalidate

### Architecture Approach

The plugin follows the established sebuf generator pattern: entry point in `cmd/protoc-gen-krakend/`, core logic in `internal/krakendgen/`, new proto annotations in `proto/sebuf/krakend/`. It mirrors protoc-gen-openapiv3 most closely -- structured data output, one file per service, typed Go structs for document construction. The generator reads both `sebuf.http` annotations (for routing, headers, query params) and `sebuf.krakend` annotations (for rate limits, auth, circuit breaker), building KrakenD Endpoint/Backend Go structs and marshaling to JSON.

**Major components:**
1. **proto/sebuf/krakend/krakend.proto** -- Gateway-specific annotations: EndpointConfig, ServiceGatewayConfig, RateLimitConfig, AuthConfig, CircuitBreakerConfig, with raw_extra_config escape hatches
2. **internal/krakendgen/types.go** -- Go structs matching KrakenD JSON schema: Endpoint, Backend, plus typed config structs for extra_config namespaces
3. **internal/krakendgen/generator.go** -- Core logic: iterate services/methods, extract annotations, build config structs, marshal to indented JSON
4. **internal/krakendgen/annotations.go** -- KrakenD-specific annotation extraction (reads krakend/ Go package types)
5. **cmd/protoc-gen-krakend/main.go** -- Entry point: plugin parameters, file iteration, per-service output

**Key architectural decisions:**
- Service-level `gateway_config` provides defaults; method-level `endpoint_config` overrides
- `raw_extra_config` string fields serve as escape hatches (parsed as JSON, merged into output)
- Auth lives at service level only (JWT config typically uniform across service endpoints)
- `map[string]any` for extra_config in Go structs (heterogeneous KrakenD namespace schemas)
- Extension numbers 51001-51002 for service/method options (outside sebuf.http range 50003-50020)

### Critical Pitfalls

1. **One endpoint per method requirement** -- KrakenD requires separate endpoint objects for each HTTP method, even on the same path. The generator must emit one object per RPC, never grouping by path. Fail generation if two RPCs produce identical (path, method) tuples.

2. **Silent parameter dropping (zero-trust model)** -- KrakenD forwards NO headers and NO query strings by default. The generator must always populate `input_headers` from sebuf header annotations and `input_query_strings` from query annotations. Never emit empty arrays. Never use `["*"]` wildcard.

3. **Path parameter variable name collisions** -- KrakenD panics on startup if endpoints share path prefixes with different variable names (`/users/{user_id}/posts` vs `/users/{id}/settings`). Add cross-service validation at generation time.

4. **Static vs. parameterized route conflicts** -- KrakenD cannot distinguish `/users/search` from `/users/{id}` at the same path level. Detect and reject at generation time with clear error.

5. **Extra_config namespace typos silently ignored** -- `"qos/rate-limit/router"` (wrong) vs `"qos/ratelimit/router"` (correct) produces no error. Define all namespace strings as Go constants, test against known allowlist.

## Implications for Roadmap

Based on combined research, protoc-gen-krakend should follow a 4-phase structure. The dependency chain is clear: proto annotations must exist before the generator can read them, the endpoint skeleton must work before gateway features attach to it, and tests must validate each phase before adding complexity.

### Phase 1: Proto Annotations and Endpoint Skeleton
**Rationale:** Everything depends on the proto annotation definitions and the core endpoint/backend generation. This phase produces a minimal but functional KrakenD config that actually works when loaded. The auto-derived header/query forwarding is the key value proposition that justifies the tool.
**Delivers:**
- `proto/sebuf/krakend/krakend.proto` with all annotation messages and extensions
- Generated Go code in `krakend/*.pb.go`
- `internal/krakendgen/types.go` with KrakenD config structs
- `internal/krakendgen/generator.go` with core endpoint/backend generation
- `cmd/protoc-gen-krakend/main.go` entry point
- Features T1 (endpoint routing), T2 (backend host), T3 (timeout), T4+D1 (input headers, auto-derived), T5+D2 (input query strings, auto-derived), T6 (output encoding), T7 (per-service files)
- Golden file tests for all core generation scenarios
- Generation-time validation: duplicate endpoint detection, static/parameterized route conflicts
**Avoids:** Pitfall #1 (one-endpoint-per-method), Pitfall #3 (static vs parameterized), Pitfall #4 (silent parameter dropping), Pitfall #5 (JSON template rendering -- uses raw JSON), Pitfall #9 (query string case sensitivity), Pitfall #14 (host schema prefix validation)

### Phase 2: Gateway Features (Rate Limiting, Auth, Resilience)
**Rationale:** Rate limiting and JWT auth are the #1 and #2 requested features that justify using an API gateway. Circuit breaker completes the resilience story. These features attach to the endpoint/backend skeleton from Phase 1 via extra_config namespaces.
**Delivers:**
- T8 (endpoint rate limiting) -- `qos/ratelimit/router` namespace
- T9 (JWT validation) -- `auth/validator` namespace
- T10 (circuit breaker) -- `qos/circuit-breaker` namespace
- D3 (backend rate limiting) -- `qos/ratelimit/proxy` namespace
- D4 (concurrent calls) -- trivial, high value
- D10 (backend caching) -- `qos/http-cache` namespace
- D12 (JWT claim propagation) -- extension of JWT config
- D14 (backend error handling) -- control error passthrough
- All extra_config namespace strings as Go constants
- Cross-service path variable collision detection (Pitfall #2)
**Avoids:** Pitfall #6 (namespace typos), Pitfall #7 (timeout cascade -- conservative defaults, clear docs), Pitfall #10 (scope confusion -- separate sebuf.krakend package)

### Phase 3: Response Shaping and Advanced Features
**Rationale:** Response manipulation features (allow/deny/mapping/group/target) form a cohesive group that should be implemented together. Collection handling auto-derives from existing unwrap annotations. These are differentiators that add polish.
**Delivers:**
- D5-D8 (response filtering, mapping, grouping, target extraction)
- D15 (collection handling, auto-derived from unwrap)
- D17 (no-op passthrough mode)
- Backend path override annotation (Pitfall #8 mitigation)
**Avoids:** Pitfall #8 (url_pattern vs endpoint path mismatch -- support override annotation)

### Phase 4: Documentation, Distribution, and Polish
**Rationale:** After features are complete, integrate into the build/release pipeline and provide comprehensive documentation.
**Delivers:**
- Updated CLAUDE.md with krakendgen architecture and commands
- Updated `.goreleaser.yaml` with protoc-gen-krakend binary
- Updated `proto/buf.yaml` for BSR publishing
- Example usage in `examples/` directory
- Flexible Config integration guide with recommended patterns
- Comma handling documentation for multi-service includes

### Phase Ordering Rationale

- **Annotations before generator**: The proto definitions must be compiled to Go before the generator can import them. This is a hard dependency.
- **Skeleton before features**: Rate limiting, JWT, and circuit breaker all attach to the endpoint/backend structs as extra_config entries. The skeleton must exist and be tested first.
- **Core routing before response shaping**: Response manipulation (allow/deny/mapping) is a nice-to-have that requires the backend object to already exist and be correct.
- **Features before documentation**: Documentation describes what exists. Shipping docs for incomplete features creates confusion.
- **Path validation in Phase 1, cross-service validation in Phase 2**: Single-service path conflicts (static vs parameterized) can be detected immediately. Cross-service variable name collisions only matter when combining multiple services, which is a Phase 2 concern.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (JWT validation):** The `auth/validator` namespace has many fields. Need to determine the minimal useful subset for the annotation message vs. what should go in `raw_extra_config`. Also need to verify JWK URL handling and cache_duration semantics.
- **Phase 2 (cross-service validation):** Path variable collision detection across services generated in separate protoc runs is architecturally challenging. May need a separate validation tool rather than in-generator detection.
- **Phase 3 (response shaping):** The interaction between `allow`/`deny`, `mapping`, `group`, and `target` has ordering semantics in KrakenD. Need to verify the exact processing order.

Phases with standard patterns (skip research-phase):
- **Phase 1 (endpoint skeleton):** Well-documented KrakenD endpoint/backend config. Follows exact same pattern as protoc-gen-openapiv3. Existing `internal/annotations/` provides all HTTP routing data.
- **Phase 4 (documentation):** Standard distribution and documentation work. GoReleaser and BSR publishing patterns already established.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Zero new dependencies. Existing protogen + stdlib json. KrakenD config schema well-documented. Validated by prior art (p3ym4n/krakend-generator). |
| Features | HIGH | KrakenD feature matrix verified from official docs. Open Source vs Enterprise boundary clear. 10 table stakes + 17 differentiators well-categorized. |
| Architecture | HIGH | Mirrors established protoc-gen-openapiv3 pattern. Component boundaries, data flow, and integration points with existing code all verified against source. |
| Pitfalls | HIGH | 14 pitfalls documented. Critical ones (one-endpoint-per-method, zero-trust forwarding, path collisions) confirmed from KrakenD GitHub issues and official docs. |

**Overall confidence:** HIGH

The research is thorough across all four dimensions. Source quality is consistently high: official KrakenD documentation (verified 2026-02-25), KrakenD GitHub issues confirming architectural constraints, existing sebuf codebase patterns, and prior art validation. The plugin's scope is well-bounded (endpoint fragments, not full config), and every major decision has clear rationale.

### Gaps to Address

- **Exact annotation message ergonomics:** The proto annotation messages (EndpointConfig, ServiceGatewayConfig, etc.) are well-defined structurally but may need refinement during implementation for usability. The `raw_extra_config` escape hatch mitigates this -- users can work around annotation gaps without waiting for new proto fields.

- **Cross-service path validation scope:** Pitfall #2 (path variable collisions) only manifests when endpoints from multiple services are loaded into one KrakenD instance. The generator processes services independently (one protoc invocation per proto file). Detection may require a post-generation validation step or a separate `krakend check` wrapper. Decide during Phase 2 planning.

- **KrakenD Flexible Config comma handling:** The `{{ include }}` approach requires manual commas between service includes. This is a UX papercut. Consider generating a helper template or documenting the `range`-with-settings-file pattern. Address in Phase 4.

- **Backend caching TTL semantics:** KrakenD's `qos/http-cache` TTL is controlled by backend Cache-Control headers, not KrakenD config. The annotation semantics for D10 need clarification -- it may be a simple boolean enable/disable rather than a TTL annotation.

- **CORS as root-level config:** CORS in KrakenD is root-level `extra_config`, not per-endpoint. Generating CORS from per-service proto annotations requires careful merging strategy. Deferred to post-Phase 3 or handled via documentation only.

## Sources

### Primary (HIGH confidence)
- [KrakenD Configuration Guide](https://www.krakend.io/docs/configuration/) -- Config structure, version 3, schema reference
- [KrakenD Endpoint Configuration](https://www.krakend.io/docs/endpoints/) -- Endpoint fields, input_headers, input_query_strings
- [KrakenD Backend Configuration](https://www.krakend.io/docs/backends/) -- Backend fields, url_pattern, host, encoding
- [KrakenD Flexible Config](https://www.krakend.io/docs/configuration/flexible-config/) -- FC_ENABLE, templates, partials, settings
- [KrakenD Parameter Forwarding](https://www.krakend.io/docs/endpoints/parameter-forwarding/) -- Zero-trust header/query forwarding policy
- [KrakenD Rate Limiting](https://www.krakend.io/docs/endpoints/rate-limit/) -- qos/ratelimit/router namespace
- [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/) -- auth/validator namespace
- [KrakenD Circuit Breaker](https://www.krakend.io/docs/backends/circuit-breaker/) -- qos/circuit-breaker namespace
- [KrakenD CE Releases](https://github.com/krakend/krakend-ce/releases) -- v2.13.1 latest (2026-02-18)
- [KrakenD Schema Repo](https://github.com/krakend/krakend-schema) -- v2.1-v2.13 schemas, JSON only
- [KrakenD Features Comparison](https://www.krakend.io/features/) -- Open Source vs Enterprise matrix
- [KrakenD Multiple Methods Issue #398](https://github.com/krakend/krakend-ce/issues/398) -- One endpoint per method confirmed

### Secondary (MEDIUM confidence)
- [p3ym4n/krakend-generator](https://github.com/p3ym4n/krakend-generator) -- Prior art: Go structs + json.Marshal pattern
- [KrakenD Route Collision Panic](https://github.com/devopsfaith/krakend/issues/292) -- Path variable collision behavior
- [KrakenD Supported Formats](https://www.krakend.io/docs/configuration/supported-formats/) -- JSON recommended, only format with --lint
- Existing sebuf codebase: `internal/annotations/`, `internal/openapiv3/`, `cmd/protoc-gen-openapiv3/` -- patterns and conventions

### Tertiary (LOW confidence)
- KrakenD CEL validation behavior -- verified namespace exists but expression syntax needs runtime testing
- Backend caching TTL semantics -- `qos/http-cache` documented but TTL control mechanism needs clarification

---
*Research completed: 2026-02-25*
*Ready for roadmap: yes*
