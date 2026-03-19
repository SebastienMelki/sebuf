# Domain Pitfalls: KrakenD Config Generation from Protobuf

**Domain:** protoc-gen-krakend -- generating KrakenD API gateway configuration from protobuf service definitions
**Researched:** 2026-02-25
**Scope:** KrakenD-specific gotchas when generating config from proto, NOT general protoc plugin pitfalls (the team has built 5 plugins already)

---

## Critical Pitfalls

Mistakes that cause broken gateway configs, runtime panics, or silent misrouting.

### Pitfall 1: One Endpoint Per Method -- The Duplication Requirement

**What goes wrong:** The generator produces a single KrakenD endpoint object with the path `/api/v1/users` and assumes it handles both GET and POST. KrakenD silently uses only one method (defaulting to GET) or the config fails validation. The other method is unreachable through the gateway.

**Why it happens:** In sebuf's proto definitions, a service can have `ListUsers` (GET /users) and `CreateUser` (POST /users) sharing the same path with different HTTP methods. Every other sebuf generator treats path+method as a natural pair. KrakenD's architecture requires a **separate endpoint object for each HTTP method**, even on the same path. This is a hard architectural constraint (confirmed "wontfix" by KrakenD maintainers in [issue #398](https://github.com/krakend/krakend-ce/issues/398)).

**Consequences:** Routes silently fail. Users get 404s or wrong handlers. Because `krakend check` without `--test-gin-routes` may not catch this, the problem can reach production.

**Prevention:**
- The generator MUST emit one KrakenD endpoint object per RPC method, never grouping by path
- Group RPCs by `(base_path + method_path, http_method)` tuple -- if two RPCs have the same path but different methods, they produce two separate endpoint objects
- Add a generation-time assertion: if two RPCs produce identical `(endpoint_path, method)` tuples, fail generation with a clear error

**Detection:** Golden tests should include services where multiple RPCs share a path (e.g., GET /users and POST /users). Verify the output contains separate endpoint objects for each.

**Phase:** Core generation logic (Phase 1). This must be correct from the start.

---

### Pitfall 2: Path Parameter Variable Name Collisions Cause Startup Panic

**What goes wrong:** The generated KrakenD config contains endpoints like `/users/{user_id}/posts` and `/users/{id}/settings` -- same path prefix, different variable names. KrakenD panics on startup: `panic: wildcard route ':id' conflicts with existing children`.

**Why it happens:** In protobuf definitions, different service teams may name path parameters differently. The sebuf HTTP annotations use `{user_id}` in one RPC and `{id}` in another. This is fine for individual service backends, but when the generator combines multiple services into a single KrakenD config, the router (based on httprouter) requires consistent variable names across colliding path prefixes.

**Consequences:** KrakenD crashes on startup. This is not a silent failure -- it's a hard panic -- but it happens at deploy time, not generation time. Debugging requires understanding KrakenD's router internals.

**Prevention:**
- Add a generation-time validation pass: after generating all endpoints across all services, check for path prefix collisions with different variable names
- When a collision is detected, fail generation with a message like: `endpoint /users/{user_id}/posts conflicts with /users/{id}/settings -- path parameters at the same position must use the same name across all endpoints`
- Consider a KrakenD-specific annotation or generator flag to override variable names at the gateway level (the backend `url_pattern` can use the original names)

**Detection:** Test with multi-service proto files where services define overlapping paths with different param names. The generator should reject them, not produce invalid config.

**Phase:** Multi-service merging (Phase 2 or 3). This only manifests when combining endpoints from multiple services.

---

### Pitfall 3: Static Routes vs. Parameterized Routes at Same Level

**What goes wrong:** A service defines both `/users/search` (a static path) and `/users/{id}` (a parameterized path). KrakenD's router cannot distinguish these -- `search` matches as a value for `{id}`. Either the static route is unreachable, or KrakenD panics if both are registered at the same level.

**Why it happens:** KrakenD uses httprouter which has explicit matches only. You cannot register static routes and variables for the same path segment. The sebuf ecosystem freely mixes static and parameterized paths because Go's standard `net/http` router (and most other routers) handle this correctly with longest-prefix matching.

**Consequences:** Silent misrouting (requests to `/users/search` get handled as `/users/{id}` with id="search") or startup panic depending on registration order.

**Prevention:**
- Add generation-time detection: scan all endpoints for conflicts where a static segment and a variable segment occupy the same position under the same prefix
- When detected, either: (a) fail with a clear error explaining the KrakenD router limitation, or (b) emit a warning and document that the user must restructure their API paths
- Document this in the annotation guide: "KrakenD does not support `/resource/action` alongside `/resource/{id}` -- use `/resource/by-id/{id}` or similar"

**Detection:** Golden test with a proto that defines both `/notes/search` and `/notes/{id}`. Verify the generator either rejects it or handles it correctly.

**Phase:** Core generation logic (Phase 1). Must be validated from the beginning.

---

### Pitfall 4: Silent Parameter Dropping -- input_headers and input_query_strings Default to Empty

**What goes wrong:** The generated KrakenD config omits `input_headers` and `input_query_strings`, or sets them to `[]`. KrakenD's zero-trust security model means **no headers and no query strings are forwarded to backends by default**. The backend receives no Authorization header, no API keys, no query parameters -- requests fail silently or return unauthorized responses.

**Why it happens:** This is KrakenD's intentional security design. Other API gateways (nginx, Envoy, Kong) forward most headers by default. Developers and generators coming from those ecosystems assume headers pass through. The sebuf ecosystem already defines header annotations (`service_headers`, `method_headers`) for validation, but those annotations describe what the *gateway endpoint* expects from *clients* -- they don't automatically translate to what should be forwarded to *backends*.

**Consequences:** Authentication breaks silently. Custom headers (X-Request-ID, X-Tenant-ID) never reach the service. Query parameters for GET requests are stripped. Debugging is painful because the gateway returns 200 (it called the backend successfully) but the backend returns errors that get swallowed or transformed.

**Prevention:**
- The generator MUST populate `input_headers` from the union of: (a) headers defined in `service_headers`/`method_headers` annotations, (b) standard pass-through headers (Authorization, Accept, Content-Type), (c) any headers defined in a new KrakenD-specific annotation
- The generator MUST populate `input_query_strings` from: (a) fields with `(sebuf.http.query)` annotations on GET/DELETE request messages, (b) any additional query params from KrakenD annotations
- NEVER use `["*"]` wildcard -- it defeats KrakenD's security model. Always generate explicit allowlists
- Add a KrakenD annotation option to declare additional pass-through headers not in the HTTP annotation set (e.g., `X-Forwarded-For`, tracing headers)

**Detection:** Golden tests for endpoints with headers and query parameters. Verify `input_headers` and `input_query_strings` arrays are populated, not empty.

**Phase:** Core generation logic (Phase 1). This is the most likely source of "it works locally but not through the gateway" bugs.

---

### Pitfall 5: Generating Flexible Config Templates That Produce Invalid JSON

**What goes wrong:** The generator outputs Go template files (`.tmpl`) for KrakenD Flexible Config, but the rendered output has trailing commas in JSON arrays, missing commas between objects, or broken nesting. KrakenD fails to parse the rendered config.

**Why it happens:** Go templates produce text, not structured data. When using `{{ range }}` to iterate over endpoint arrays, the classic mistake is emitting a comma after the last element (trailing comma, invalid in JSON). This is compounded when multiple services contribute endpoint fragments that must be concatenated into a single `"endpoints": [...]` array.

**Consequences:** `krakend check` fails with cryptic JSON parse errors pointing at the rendered output, not the template source. Debugging requires using `FC_OUT=debug.json` to inspect the rendered config.

**Prevention:**
- **Option A (recommended): Generate raw JSON fragments, not Go templates.** Each service produces a `service_name.json` file containing its endpoint array. A thin top-level template concatenates them. This sidesteps most template-in-JSON problems.
- **Option B: If generating templates, use the `{{ if $index }},{{ end }}` pattern** at the START of each iteration, not a comma at the end. This is the official KrakenD recommendation.
- Always validate generated output by running `krakend check -tlc` on the rendered result in tests
- Consider emitting JSON via Go's `encoding/json` marshaler rather than string concatenation -- this guarantees valid JSON structure

**Detection:** Integration tests that run the actual Flexible Config rendering and validate the output JSON. Unit tests for comma handling are insufficient -- test the full render pipeline.

**Phase:** Output format design (Phase 1). The choice between raw JSON fragments vs. Go templates is an architectural decision that affects everything downstream.

---

## Moderate Pitfalls

### Pitfall 6: extra_config Namespace Typos Are Silently Ignored

**What goes wrong:** The generator emits `"qos/rate-limit/router"` instead of `"qos/ratelimit/router"` (note: no hyphen). KrakenD silently ignores the unknown namespace. The endpoint has no rate limiting, but the config appears valid.

**Why it happens:** KrakenD namespaces are magic strings with no compile-time validation. `krakend run` ignores unknown keys. Only `krakend check --lint` catches them, and that requires downloading the schema. A generator author who misremembers the namespace or uses an outdated version produces config that looks right but does nothing.

**Prevention:**
- Define all KrakenD namespace strings as Go constants in a single file (e.g., `internal/krakendgen/namespaces.go`)
- Unit test each constant against the KrakenD JSON schema (download the schema once and verify all generated namespace keys exist in it)
- Document every supported namespace in the proto annotation comments so users know what they're enabling
- Keep namespace constants versioned -- when targeting a new KrakenD version, update constants and re-validate

**Detection:** A test that parses generated config and checks every `extra_config` key against a known allowlist of valid KrakenD namespaces.

**Phase:** Feature implementation (Phase 2, when adding rate limiting/auth annotations). But the namespace constant pattern should be established in Phase 1.

---

### Pitfall 7: Timeout Cascade Misconfiguration

**What goes wrong:** The generator sets endpoint-level `timeout` to a value shorter than the backend service's actual response time, or longer than the client's timeout. KrakenD returns partial responses (HTTP 200 with incomplete data) or the client times out before KrakenD does.

**Why it happens:** KrakenD's timeout model is different from most gateways. The endpoint `timeout` is the **total pipe duration** including all backend calls, response processing, and component execution. There is no separate "backend timeout" in KrakenD CE. If the generator blindly sets a default timeout (e.g., "3s") without understanding the backend's latency profile, it creates cascading failures.

Additionally, when a timeout fires and there are multiple backends, KrakenD returns whatever data it has collected so far -- a partial 200 response, not a 504. This is surprising behavior that can propagate corrupt partial data downstream.

**Prevention:**
- Make timeout a required or prominently-defaulted annotation, not a hidden default
- Default to a conservative value (e.g., "30s") rather than KrakenD's default "2s" which is too aggressive for most real services
- Document the timeout hierarchy clearly: `Client > KrakenD endpoint > Backend service`
- For generated configs targeting single-backend endpoints (the common case in sebuf), the timeout behavior is simpler. Document that partial responses only occur with multi-backend aggregation endpoints

**Detection:** Test that the generated timeout values are present and reasonable. Warn if no timeout annotation is provided.

**Phase:** Core generation (Phase 1 for defaults, Phase 2 for annotation-driven overrides).

---

### Pitfall 8: Backend url_pattern vs. Endpoint Path Mismatch

**What goes wrong:** The generator uses the same path for both the KrakenD `endpoint` and the backend `url_pattern`, but the backend service is actually listening on a different path (e.g., the backend has a different base path, or the gateway path is a rewrite of the backend path). Requests reach the wrong backend route.

**Why it happens:** In sebuf, the proto annotations define the HTTP path as seen by clients: `base_path + method_path`. This is the correct value for the KrakenD `endpoint` field. But the backend `url_pattern` should be the path **as the backend service expects it**. In many deployments, these are the same. But when the gateway rewrites paths (e.g., gateway exposes `/v2/users` while the service expects `/api/v1/users`), they diverge.

**Consequences:** 404s from the backend, or routing to wrong handlers. Difficult to debug because the gateway config looks correct.

**Prevention:**
- Default behavior: `url_pattern` = `endpoint` path (same path, most common case)
- Support a KrakenD-specific annotation for backend path override: `(sebuf.krakend.backend_path) = "/internal/users/{id}"`
- If no override, use the full path from sebuf HTTP annotations for both `endpoint` and `url_pattern`
- Document clearly that `endpoint` is what clients see, `url_pattern` is what the backend sees

**Detection:** Golden tests with and without backend path overrides. Verify the default case uses identical paths.

**Phase:** Core generation (Phase 1 for default behavior, Phase 2 for override annotation).

---

### Pitfall 9: Query String Case Sensitivity Mismatch

**What goes wrong:** The proto field is `page_size` (snake_case), the generated `input_query_strings` entry is `page_size`, but the frontend sends `pageSize` (camelCase). KrakenD is case-sensitive on query strings -- `page_size` != `pageSize`. The parameter is silently dropped.

**Why it happens:** sebuf's `(sebuf.http.query)` annotation allows specifying a custom query parameter name, but if omitted, the generator must choose a convention. Protobuf fields use snake_case by default. JavaScript/TypeScript clients typically use camelCase. If the query string name in the KrakenD config doesn't match what the actual client sends, parameters are silently dropped.

**Consequences:** Query parameters silently stripped. Backend receives requests with missing filters, pagination, etc. No error -- just wrong results.

**Prevention:**
- Use the explicit `(sebuf.http.query).name` value if present
- If no explicit name, use the proto field's **JSON name** (which protobuf auto-generates as camelCase from snake_case)
- Document the convention clearly: "KrakenD query parameter names match the JSON names used by the TypeScript client"
- Consider generating both snake_case and camelCase variants in `input_query_strings` for compatibility (KrakenD allows multiple entries)

**Detection:** Golden test with query parameters that have explicit names and ones using defaults. Verify the generated names match what the ts-client generator would send.

**Phase:** Core generation (Phase 1). Must be consistent with the existing ts-client generator from day one.

---

### Pitfall 10: Annotation Scope Confusion -- Gateway vs. Backend Concerns

**What goes wrong:** The proto annotations conflate two different concerns: what the *gateway* should do (rate limiting, auth, CORS) and what the *backend* should do (request validation, header forwarding, response mapping). Users put rate limiting annotations on the proto RPC and expect them to apply to the backend, or put backend-specific config expecting it at the gateway endpoint level.

**Why it happens:** Existing sebuf annotations (`service_headers`, `method_headers`, query params) describe the HTTP API contract between client and service. For a direct client-to-service connection, there's one contract. When a gateway sits in between, there are *two* contracts: client-to-gateway and gateway-to-backend. The new KrakenD annotations must clearly distinguish these scopes.

**Consequences:** Rate limits applied at wrong scope (backend instead of gateway, or vice versa). Auth validation that should happen at the gateway being silently skipped. Headers forwarded when they shouldn't be, or blocked when they should pass through.

**Prevention:**
- Use a separate proto package (`sebuf.krakend`) for all gateway-specific annotations, completely separate from `sebuf.http`
- Name annotations to make scope explicit: `(sebuf.krakend.endpoint_rate_limit)` not just `(sebuf.krakend.rate_limit)`
- The generator should read `sebuf.http` annotations for path/method/headers (the API contract) and `sebuf.krakend` annotations for gateway behavior (rate limits, auth, circuit breaker)
- Document the dual-contract model prominently in the annotation guide

**Detection:** Review the annotation proto file. Every message/extension should clearly indicate its KrakenD scope (service-level, endpoint-level, or backend-level).

**Phase:** Annotation design (Phase 1). Getting the package structure right is foundational.

---

## Minor Pitfalls

### Pitfall 11: KrakenD Schema Version Drift

**What goes wrong:** The generator targets KrakenD schema v2.7 but the user runs KrakenD v2.13. New features or renamed namespaces cause silent failures or lint warnings.

**Prevention:**
- Include `$schema` in generated config pointing to the targeted KrakenD version
- Document the minimum supported KrakenD version
- Consider a generator flag `--krakend-version=2.13` to control which schema features are emitted
- When the schema URL is included, users get IDE autocompletion and validation automatically

**Phase:** Phase 1 (output format). Low effort, high value.

---

### Pitfall 12: Missing Content-Type in Backend Encoding

**What goes wrong:** The generator omits the `encoding` field on backends, defaulting to `"json"`. But the sebuf-generated Go server can serve both JSON and protobuf (Content-Type negotiation). If the backend is configured for protobuf, KrakenD's json decoder fails to parse responses.

**Prevention:**
- Default to `"json"` encoding for backends (the common case for sebuf HTTP APIs)
- For protobuf-native backends, document that `"no-op"` encoding must be used (disabling KrakenD's response manipulation)
- Provide a KrakenD annotation to override backend encoding: `(sebuf.krakend.backend_encoding) = "no-op"`

**Phase:** Phase 2 (backend configuration). Not critical for initial implementation.

---

### Pitfall 13: Automatic Redirect Behavior Breaking API Clients

**What goes wrong:** KrakenD's default router settings enable automatic redirects for trailing slashes and case mismatches. A POST to `/api/v1/users/` gets redirected to `/api/v1/users` with a 301, which browsers follow as a GET -- losing the request body.

**Prevention:**
- Generate `"disable_redirect_fixed_path": true` and `"disable_redirect_trailing_slash": true` in the router config by default
- This matches KrakenD's own recommendation for API gateways (as opposed to web servers)

**Phase:** Phase 1 (global config defaults). Simple one-liner in the generated output.

---

### Pitfall 14: Host Array Format Requires Schema Prefix

**What goes wrong:** The generator emits `"host": ["my-service:8080"]` without the `http://` or `https://` prefix. KrakenD silently fails to connect or produces garbled URLs.

**Prevention:**
- Default to `"http://"` prefix if not provided in the annotation
- Validate that host values in annotations include a schema prefix
- Support environment-variable interpolation for host: `"host": ["{{ env \"USER_SERVICE_HOST\" }}"]` when using Flexible Config

**Phase:** Phase 1 (backend configuration). Simple validation.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Core endpoint generation | One-endpoint-per-method requirement (#1) | Emit separate objects per RPC; golden tests with shared paths |
| Core endpoint generation | Static vs. parameterized route conflicts (#3) | Generation-time path conflict detection |
| Core endpoint generation | Silent parameter dropping (#4) | Always populate input_headers/input_query_strings from proto annotations |
| Output format design | JSON template rendering (#5) | Prefer raw JSON fragments over Go templates; validate with krakend check |
| Annotation design | Gateway vs. backend scope confusion (#10) | Separate sebuf.krakend package; explicit scope in annotation names |
| Multi-service merging | Variable name collisions (#2) | Cross-service path validation pass |
| Rate limiting / auth features | Namespace typos (#6) | Centralized namespace constants; schema validation tests |
| Timeout configuration | Cascade misconfiguration (#7) | Conservative defaults; clear documentation of timeout hierarchy |
| Backend path mapping | url_pattern mismatch (#8) | Default to same path; support override annotation |
| Query parameter forwarding | Case sensitivity (#9) | Use JSON names (camelCase) for query params; match ts-client behavior |

## Sources

- [KrakenD Configuration Structure](https://www.krakend.io/docs/configuration/structure/) -- HIGH confidence
- [KrakenD Endpoint Configuration](https://www.krakend.io/docs/endpoints/) -- HIGH confidence
- [KrakenD Flexible Configuration](https://www.krakend.io/docs/configuration/flexible-config/) -- HIGH confidence
- [KrakenD Parameter Forwarding](https://www.krakend.io/docs/endpoints/parameter-forwarding/) -- HIGH confidence
- [KrakenD Backend Configuration](https://www.krakend.io/docs/backends/) -- HIGH confidence
- [KrakenD Rate Limiting](https://www.krakend.io/docs/endpoints/rate-limit/) -- HIGH confidence
- [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/) -- HIGH confidence
- [KrakenD Configuration Check](https://www.krakend.io/docs/configuration/check/) -- HIGH confidence
- [KrakenD Templates](https://www.krakend.io/docs/configuration/templates/) -- HIGH confidence
- [KrakenD No-Op Encoding](https://www.krakend.io/docs/endpoints/no-op/) -- HIGH confidence
- [KrakenD Timeouts](https://www.krakend.io/docs/throttling/timeouts/) -- HIGH confidence
- [KrakenD Multiple Methods Issue #398](https://github.com/krakend/krakend-ce/issues/398) -- HIGH confidence
- [KrakenD Route Collision Panic](https://github.com/devopsfaith/krakend/issues/292) -- HIGH confidence
