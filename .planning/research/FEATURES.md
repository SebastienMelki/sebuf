# Feature Landscape: protoc-gen-krakend

**Domain:** KrakenD API gateway configuration generation from protobuf service definitions
**Researched:** 2026-02-25
**Prior art:** No existing `protoc-gen-krakend` tool exists. This would be novel.

---

## Table Stakes

Features users expect. Missing = the tool is not useful enough to adopt over hand-writing KrakenD JSON.

| # | Feature | Why Expected | KrakenD Config Keys | Complexity | Notes |
|---|---------|--------------|---------------------|------------|-------|
| T1 | Endpoint routing from proto | The entire point of the tool -- reuse `sebuf.http.config` (path, method) and `sebuf.http.service_config` (base_path) to generate KrakenD `endpoint` + `method` | `endpoint`, `method` | Low | Reuses existing annotations. No new proto needed. |
| T2 | Backend host/URL mapping | Each service needs at least one backend target. Without this, generated config is incomplete. | `backend[].host`, `backend[].url_pattern` | Low | New annotation: `sebuf.krakend.backend_host` at service level. URL pattern derived from `sebuf.http.config.path`. |
| T3 | Per-endpoint timeout | Timeouts are the most basic QoS control. KrakenD defaults to 2s which is often wrong. | `timeout` (endpoint level) | Low | New annotation: `sebuf.krakend.timeout` at method or service level. String duration format ("3s", "500ms"). |
| T4 | Input header forwarding | KrakenD's zero-trust model blocks ALL headers by default. Without explicit `input_headers`, the backend receives nothing -- auth tokens, content types, custom headers all get dropped. | `input_headers` (endpoint + backend level) | Low | Auto-derive from `sebuf.http.service_headers` and `sebuf.http.method_headers`. Headers declared in proto annotations automatically appear in `input_headers`. |
| T5 | Input query string forwarding | Same zero-trust issue as headers. Query params declared via `sebuf.http.query` must be forwarded. | `input_query_strings` (endpoint + backend level) | Low | Auto-derive from fields with `sebuf.http.query` annotation on request messages. |
| T6 | Output encoding | Users must control response format. JSON is default but `no-op` is needed for passthrough. | `output_encoding` (endpoint), `encoding` (backend) | Low | New annotation: `sebuf.krakend.output_encoding` at method level. Default: "json". |
| T7 | Per-service file output | Consistent with sebuf's existing per-service file pattern (OpenAPI does this). One KrakenD fragment per service. | N/A (file organization) | Low | Generate `ServiceName.krakend.json` per service. Users merge fragments into main `krakend.json` via Flexible Configuration partials. |
| T8 | Endpoint rate limiting (router level) | Rate limiting is the #1 reason teams adopt API gateways. Users explicitly requested this. | `extra_config["qos/ratelimit/router"]`: `max_rate`, `capacity`, `every`, `client_max_rate`, `client_capacity`, `strategy`, `key` | Medium | New annotation: `sebuf.krakend.rate_limit` at method or service level. |
| T9 | JWT validation | Auth is core gateway functionality. JWT is KrakenD's primary open-source auth mechanism. Users explicitly requested this. | `extra_config["auth/validator"]`: `alg`, `jwk_url`, `audience`, `issuer`, `roles`, `roles_key`, `scopes`, `scopes_key`, `cache` | Medium | New annotation: `sebuf.krakend.jwt` at method or service level. Service-level sets defaults, method-level overrides. |
| T10 | Circuit breaker | Essential resilience pattern. Users explicitly requested this. KrakenD CE includes it. | `extra_config["qos/circuit-breaker"]` (backend): `interval`, `timeout`, `max_errors`, `log_status_change` | Low | New annotation: `sebuf.krakend.circuit_breaker` at service level (applies to all backends). |

---

## Differentiators

Features that set protoc-gen-krakend apart from manual KrakenD config. Not expected, but valued.

| # | Feature | Value Proposition | KrakenD Config Keys | Complexity | Notes |
|---|---------|-------------------|---------------------|------------|-------|
| D1 | Auto-derived input_headers from proto | Zero manual header config. Headers declared in `sebuf.http.service_headers` / `sebuf.http.method_headers` automatically become KrakenD `input_headers`. No other tool does this. | `input_headers` | Low | Pure derivation from existing annotations. The generator reads header names from the proto and emits them in `input_headers`. This is the "magic" moment -- define headers once in proto, get gateway forwarding for free. |
| D2 | Auto-derived input_query_strings from proto | Query params from `sebuf.http.query` annotations automatically forwarded. | `input_query_strings` | Low | Same pattern as D1 but for query strings. |
| D3 | Backend rate limiting (proxy level) | Controls how fast KrakenD calls your backend, protecting it from overload. Different from router rate limit which controls client access. | `extra_config["qos/ratelimit/proxy"]` (backend): `max_rate`, `capacity`, `every` | Low | New annotation: `sebuf.krakend.backend_rate_limit` at service level. |
| D4 | Concurrent calls | KrakenD's performance optimization: send N parallel requests to backend, return first success. Reduces latency and error rates. | `concurrent_calls` (endpoint level) | Low | New annotation: `sebuf.krakend.concurrent_calls` at method level. Integer value. |
| D5 | Backend response filtering (allow/deny) | Control which fields from backend response reach the client. Useful for removing internal fields. | `backend[].allow`, `backend[].deny` | Medium | New annotation: `sebuf.krakend.response_allow` / `sebuf.krakend.response_deny` at method level. Array of field names. Supports dot notation for nested fields. |
| D6 | Backend response mapping (field rename) | Rename fields in backend response before returning to client. | `backend[].mapping` | Low | New annotation: `sebuf.krakend.response_mapping` at method level. Map of old_name -> new_name. |
| D7 | Backend response grouping | Wrap backend response under a named key. Essential when aggregating multiple backends. | `backend[].group` | Low | New annotation: `sebuf.krakend.response_group` at method level (per-backend). String value. |
| D8 | Backend response target extraction | Extract nested field as root (e.g., unwrap `{"data": {...}}` to just `{...}`). | `backend[].target` | Low | New annotation: `sebuf.krakend.response_target` at method level. String field name. |
| D9 | CORS configuration | Generate service-level CORS config. While CORS is global in KrakenD (root extra_config), generating the default from service definitions saves boilerplate. | `extra_config["security/cors"]`: `allow_origins`, `allow_methods`, `allow_headers`, `expose_headers`, `allow_credentials`, `max_age` | Medium | New annotation: `sebuf.krakend.cors` at file or service level. Generates root-level CORS config. |
| D10 | Backend caching | Enable in-memory response caching at the backend level. Reduces load on upstream services. | `extra_config["qos/http-cache"]` (backend): `shared`, `max_items`, `max_size` | Low | New annotation: `sebuf.krakend.cache` at method or service level. |
| D11 | CEL request validation expressions | Add request-time validation expressions (beyond what proto validation does). Gateway-level validation before hitting backend. | `extra_config["validation/cel"]` (endpoint): array of `{check_expr}` | Medium | New annotation: `sebuf.krakend.cel_validation` at method level. Array of CEL expression strings. |
| D12 | JWT claim propagation | Forward specific JWT claims as backend request headers. Enables backend to receive user identity without parsing JWT itself. | `propagate_claims` within `auth/validator` config | Low | Extension of T9 (JWT). Add `propagate_claims` field to the JWT annotation message. |
| D13 | Sequential proxy | Enable backend call chaining where output of one call feeds the next. | `extra_config["proxy"]["sequential"]`: `true` | Medium | New annotation: `sebuf.krakend.sequential` at method level. Boolean. Only meaningful for endpoints with multiple backends. |
| D14 | Backend error handling | Control whether backend errors pass through to client (return_error_code, return_error_details, return_error_msg). | `extra_config["backend/http"]`: `return_error_code`, `return_error_details`, `return_error_msg` (router) | Low | New annotation: `sebuf.krakend.error_handling` at method or service level. |
| D15 | Collection handling (is_collection) | Declare when backend returns an array instead of an object. | `backend[].is_collection` | Low | Auto-derive from response message type: if the response has `sebuf.http.unwrap` on a repeated field, set `is_collection: true` or use `output_encoding: "json-collection"`. |
| D16 | Security HTTP headers | Generate service-level security headers (HSTS, XSS protection, clickjacking prevention). | `extra_config["security/http"]`: `sts_seconds`, `frame_deny`, `content_type_nosniff`, etc. | Low | New annotation: `sebuf.krakend.security_headers` at file level. One-time global config. |
| D17 | No-op passthrough mode | Mark specific endpoints as pure proxy passthrough (no request/response inspection). | `output_encoding: "no-op"` | Low | New annotation: `sebuf.krakend.passthrough` at method level. Boolean. Overrides output_encoding to "no-op". |

---

## Anti-Features

Features to explicitly NOT build. These would hurt the tool or create maintenance burden disproportionate to value.

| # | Anti-Feature | Why Avoid | What to Do Instead |
|---|--------------|-----------|-------------------|
| A1 | Full krakend.json generation | KrakenD's root config includes service-level settings (port, TLS, telemetry, logging, plugin loading) that are infrastructure concerns, not API design concerns. Generating the full file couples proto definitions to deployment config. | Generate per-service endpoint fragment JSON files. Users compose these into their krakend.json using KrakenD Flexible Configuration (`FC_PARTIALS`). Document the merge pattern. |
| A2 | API key authentication config | API keys are KrakenD Enterprise-only (`auth/api-keys`). Generating Enterprise-only config from open-source proto annotations creates confusion and lock-in. | Support JWT validation (open-source `auth/validator`). Document that API keys require Enterprise and can be added to the generated fragments manually. |
| A3 | gRPC backend integration | KrakenD's gRPC client is Enterprise-only. Generating gRPC backend config targets a paid feature. Also, sebuf is an HTTP toolkit -- gRPC backends contradict the core use case. | Generate HTTP backend config only. Users wanting gRPC backends can add `backend/grpc` extra_config manually. |
| A4 | Service Discovery configuration | KrakenD SD (`sd: "dns"`) is deployment-infrastructure config that varies by environment (Docker, K8s, bare metal). Proto definitions should not encode deployment topology. | Generate `sd: "static"` (default). Document how to override `host` arrays and `sd` mode per environment using Flexible Configuration settings files. |
| A5 | Telemetry/metrics/logging config | These are operational concerns (OpenTelemetry, Prometheus, Datadog, etc.) that vary per deployment. Proto definitions have no business encoding observability stack choices. | Do not generate any `telemetry/*` config. Document recommended telemetry setup separately. |
| A6 | Plugin/middleware registration | KrakenD plugins (`plugin/http-server`, `plugin/http-client`, `plugin/middleware`) are custom Go code deployed alongside KrakenD. Referencing them from proto annotations is nonsensical. | Do not generate any `plugin/*` config. |
| A7 | Response Go templates (Enterprise) | Enterprise-only response transformation via Go templates. Same issue as A2. | Use backend `allow`/`deny`/`mapping`/`target` for response shaping (open-source features). |
| A8 | Workflows (Enterprise) | Enterprise-only multi-step request orchestration. | Support basic `sequential` proxy (open-source) instead. |
| A9 | JSON Schema request validation | KrakenD's `validation/json-schema` duplicates what sebuf's backend already does via `buf.validate` / protovalidate. Generating gateway-level JSON Schema from proto is redundant -- the backend already validates. | Let backend handle request validation via protovalidate. If users want gateway-level validation, use CEL expressions (D11) for lightweight checks. |
| A10 | Flatmap array manipulation | KrakenD's `proxy/flatmap_filter` is a complex response transformation engine for arrays. It is rarely needed when the backend already shapes its response correctly (which sebuf ensures). | Backend response shaping is handled by the generated HTTP server code. Use `allow`/`deny`/`target` for simple filtering. |

---

## Feature Dependencies

```
T1 (Endpoint routing) --> T2 (Backend host) --> ALL OTHER FEATURES
   The endpoint + backend skeleton is the foundation everything else attaches to.

T4 (Input headers) <-- D1 (Auto-derive headers)
   D1 is the automatic version of T4. T4 provides manual override capability.

T5 (Input query strings) <-- D2 (Auto-derive query strings)
   D2 is the automatic version of T5.

T9 (JWT validation) --> D12 (Claim propagation)
   Claim propagation is a sub-feature of JWT config.

T8 (Rate limiting) --> D3 (Backend rate limiting)
   Router rate limit should come before proxy rate limit (different scopes, same conceptual area).

D5 (Allow/deny) + D6 (Mapping) + D7 (Group) + D8 (Target) form a cohesive "response shaping" group.
   All are backend-level response manipulation. Implement together.

T10 (Circuit breaker) + D3 (Backend rate limiting) + D10 (Caching) form "backend resilience" group.
   All are backend extra_config. Implement together.
```

---

## MVP Recommendation

### Phase 1: Skeleton (must ship first)

Prioritize:
1. **T1 - Endpoint routing** -- The core translation of proto HTTP annotations to KrakenD endpoints
2. **T2 - Backend host mapping** -- Without backends, endpoints are empty shells
3. **T7 - Per-service file output** -- Consistent with OpenAPI generator pattern
4. **T4 + D1 - Input header forwarding (auto-derived)** -- KrakenD blocks all headers by default; without this, nothing works. Auto-derivation from existing `sebuf.http.service_headers`/`sebuf.http.method_headers` is the key value proposition.
5. **T5 + D2 - Input query string forwarding (auto-derived)** -- Same rationale as headers.
6. **T3 - Timeouts** -- Trivial to add, critical for correctness.
7. **T6 - Output encoding** -- Simple, necessary for non-JSON backends.

**Rationale:** This phase produces a minimal but functional KrakenD config that actually works when loaded. The auto-derived header/query forwarding is the "wow" moment that justifies using the tool over manual config.

### Phase 2: Gateway Features (rate limiting, auth, resilience)

Prioritize:
1. **T8 - Endpoint rate limiting** -- #1 requested feature
2. **T9 - JWT validation** -- #2 requested feature
3. **T10 - Circuit breaker** -- Simple, high value
4. **D3 - Backend rate limiting** -- Completes the rate limiting story
5. **D4 - Concurrent calls** -- Trivial, high value
6. **D10 - Backend caching** -- Simple, high value
7. **D12 - JWT claim propagation** -- Extension of JWT, low incremental cost
8. **D14 - Backend error handling** -- Simple config, important for debugging

**Rationale:** This phase adds the gateway-specific value that KrakenD provides beyond simple reverse proxying. Rate limiting and JWT are the two features users explicitly requested.

### Phase 3: Response Shaping and Advanced Features

Prioritize:
1. **D5-D8 - Response filtering/mapping/grouping/target** -- Cohesive response shaping group
2. **D15 - Collection handling** -- Auto-derived from existing unwrap annotations
3. **D9 - CORS configuration** -- Common need, medium complexity
4. **D16 - Security headers** -- Low effort, good defaults
5. **D17 - No-op passthrough** -- Niche but simple

Defer:
- **D11 - CEL validation**: Medium complexity, niche use case (backend already validates)
- **D13 - Sequential proxy**: Anti-pattern per KrakenD docs, rarely needed

---

## KrakenD Open Source vs Enterprise Feature Matrix

Critical for protoc-gen-krakend: only generate config for open-source features by default.

| Feature | Community (Free) | Enterprise (Paid) | protoc-gen-krakend Support |
|---------|-----------------|-------------------|---------------------------|
| Endpoint rate limit | Yes | Yes | T8 - Generate |
| Backend rate limit | Yes | Yes | D3 - Generate |
| JWT validation | Yes | Yes | T9 - Generate |
| API keys | No | Yes | A2 - Do NOT generate |
| CORS | Yes | Yes | D9 - Generate |
| Circuit breaker | Yes | Yes | T10 - Generate |
| Concurrent calls | Yes | Yes | D4 - Generate |
| Caching | Yes | Yes | D10 - Generate |
| CEL validation | Yes | Yes | D11 - Generate (Phase 3) |
| JSON Schema validation | Yes | Yes | A9 - Do NOT generate (redundant) |
| Security headers | Yes | Yes | D16 - Generate |
| Sequential proxy | Yes | Yes | D13 - Defer |
| Response manipulation (allow/deny/mapping/target/group) | Yes | Yes | D5-D8 - Generate |
| Response Go templates | No | Yes | A7 - Do NOT generate |
| Workflows | No | Yes | A8 - Do NOT generate |
| gRPC backend | No | Yes | A3 - Do NOT generate |
| Security Policies | No | Yes | Do NOT generate |
| Tiered rate limit | No | Yes | Do NOT generate |
| Stateful rate limit (Redis) | No | Yes | Do NOT generate |

---

## Annotation Extension Number Planning

Existing sebuf annotations use extension numbers 50003-50020. KrakenD annotations should use a separate range to avoid collisions and keep concerns separated.

**Recommendation:** Use range 51000-51099 for `sebuf.krakend.*` extensions in a new `proto/sebuf/krakend/` package.

| Ext # | Name | Target | Purpose |
|-------|------|--------|---------|
| 51000 | krakend_endpoint | MethodOptions | Per-method KrakenD endpoint config (timeout, concurrent_calls, output_encoding, passthrough) |
| 51001 | krakend_service | ServiceOptions | Per-service KrakenD config (backend_host, default timeout, circuit_breaker, backend_rate_limit, cache, error_handling) |
| 51002 | krakend_rate_limit | MethodOptions | Per-method router rate limit config |
| 51003 | krakend_jwt | MethodOptions | Per-method JWT validation config |
| 51004 | krakend_jwt_service | ServiceOptions | Service-level JWT defaults |
| 51005 | krakend_response | MethodOptions | Per-method response shaping (allow, deny, mapping, group, target) |
| 51006 | krakend_cors | FileOptions | File-level CORS config |
| 51007 | krakend_security | FileOptions | File-level security headers config |
| 51008 | krakend_cel | MethodOptions | Per-method CEL validation expressions |

---

## Confidence Assessment

| Feature Area | Confidence | Reason |
|--------------|------------|--------|
| Endpoint routing (T1-T2) | HIGH | KrakenD endpoint/backend config is well-documented and stable |
| Header/query forwarding (T4-T5, D1-D2) | HIGH | KrakenD zero-trust model is well-documented; derivation from existing annotations is straightforward |
| Rate limiting (T8, D3) | HIGH | `qos/ratelimit/router` and `qos/ratelimit/proxy` namespaces verified from official docs |
| JWT validation (T9, D12) | HIGH | `auth/validator` namespace fully documented with all config keys |
| Circuit breaker (T10) | HIGH | `qos/circuit-breaker` namespace verified from official docs |
| Response shaping (D5-D8) | HIGH | `allow`/`deny`/`mapping`/`group`/`target` are core KrakenD backend features, well-documented |
| CORS (D9) | MEDIUM | `security/cors` is root-level only -- need to verify how per-service fragments merge into root config |
| Caching (D10) | MEDIUM | `qos/http-cache` works but TTL is controlled by backend Cache-Control headers, not KrakenD config. Annotation semantics need clarification. |
| CEL validation (D11) | MEDIUM | `validation/cel` verified but CEL expression syntax needs validation against actual KrakenD behavior |
| Enterprise feature boundaries | HIGH | Feature comparison matrix from official KrakenD features page |

---

## Sources

- [KrakenD Endpoint Configuration](https://www.krakend.io/docs/endpoints/) -- Endpoint config keys, verified 2026-02-25
- [KrakenD Backend Configuration](https://www.krakend.io/docs/backends/) -- Backend config keys, verified 2026-02-25
- [KrakenD Rate Limiting (Router)](https://www.krakend.io/docs/endpoints/rate-limit/) -- `qos/ratelimit/router` namespace, verified 2026-02-25
- [KrakenD Rate Limiting (Proxy)](https://www.krakend.io/docs/backends/rate-limit/) -- `qos/ratelimit/proxy` namespace, verified 2026-02-25
- [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/) -- `auth/validator` namespace, verified 2026-02-25
- [KrakenD Circuit Breaker](https://www.krakend.io/docs/backends/circuit-breaker/) -- `qos/circuit-breaker` namespace, verified 2026-02-25
- [KrakenD CORS](https://www.krakend.io/docs/service-settings/cors/) -- `security/cors` namespace, verified 2026-02-25
- [KrakenD Parameter Forwarding](https://www.krakend.io/docs/endpoints/parameter-forwarding/) -- input_headers, input_query_strings, verified 2026-02-25
- [KrakenD Data Manipulation](https://www.krakend.io/docs/backends/data-manipulation/) -- allow/deny/mapping/group/target, verified 2026-02-25
- [KrakenD Caching](https://www.krakend.io/docs/backends/caching/) -- `qos/http-cache` namespace, verified 2026-02-25
- [KrakenD CEL Validation](https://www.krakend.io/docs/endpoints/common-expression-language-cel/) -- `validation/cel` namespace, verified 2026-02-25
- [KrakenD Timeouts](https://www.krakend.io/docs/throttling/timeouts/) -- Timeout configuration, verified 2026-02-25
- [KrakenD Configuration Structure](https://www.krakend.io/docs/configuration/structure/) -- extra_config scoping, verified 2026-02-25
- [KrakenD Features Comparison](https://www.krakend.io/features/) -- Open Source vs Enterprise matrix, verified 2026-02-25
- [KrakenD Flexible Configuration](https://www.krakend.io/docs/configuration/flexible-config/) -- Template/partial system for composing config, verified 2026-02-25
- [KrakenD No-Op Encoding](https://www.krakend.io/docs/endpoints/no-op/) -- Passthrough proxy mode, verified 2026-02-25
- [KrakenD Sequential Proxy](https://www.krakend.io/docs/endpoints/sequential-proxy/) -- Backend call chaining, verified 2026-02-25
- [KrakenD Error Handling](https://www.krakend.io/docs/backends/detailed-errors/) -- return_error_code, return_error_details, verified 2026-02-25
- [KrakenD Security Headers](https://www.krakend.io/docs/service-settings/security/) -- HTTP security config, verified 2026-02-25
- [KrakenD API Keys (Enterprise)](https://www.krakend.io/docs/enterprise/authentication/api-keys/) -- Enterprise-only feature, verified 2026-02-25
- [KrakenD JSON Schema Validation](https://www.krakend.io/docs/endpoints/json-schema/) -- validation/json-schema namespace, verified 2026-02-25
