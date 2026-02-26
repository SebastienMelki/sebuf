# Phase 13: Gateway Features - Research

**Researched:** 2026-02-25
**Domain:** KrakenD extra_config generation for rate limiting, JWT auth, circuit breaker, caching, and concurrent calls from protobuf annotations
**Confidence:** HIGH

## Summary

Phase 13 extends the existing `protoc-gen-krakend` plugin (built in Phase 12) to generate KrakenD `extra_config` entries from proto annotations. The plugin already produces correct endpoint/backend skeletons with routing, timeouts, host config, header forwarding, and query string forwarding. This phase adds gateway-specific features: rate limiting (endpoint-level `qos/ratelimit/router` and backend-level `qos/ratelimit/proxy`), JWT authentication (`auth/validator` with claim propagation), circuit breaker (`qos/circuit-breaker`), backend caching (`qos/http-cache`), and concurrent calls (top-level `concurrent_calls` field on endpoints).

The implementation requires three categories of changes: (1) extending the `krakend.proto` annotation definitions with new messages for each feature, (2) extending the `Endpoint` and `Backend` Go structs in `types.go` with `ExtraConfig` and `ConcurrentCalls` fields, and (3) adding generator logic to read the new annotations and build the corresponding `extra_config` maps. The existing service-level default / method-level override pattern established in Phase 12 for host and timeout applies identically to all new features. All KrakenD namespace strings must be Go constants validated against a known allowlist.

The existing Phase 12 architecture (generator.go reads annotations, builds Endpoint structs, marshals to JSON) is well-suited for this extension. No new Go dependencies are required. The main complexity is in the proto annotation message design -- balancing completeness (exposing enough KrakenD config knobs) against simplicity (not modeling every KrakenD field).

**Primary recommendation:** Add new proto messages to the existing `krakend.proto` (not new proto files), add `ExtraConfig map[string]any` fields to the existing `Endpoint` and `Backend` structs, implement resolver functions following the `resolveTimeout`/`resolveHost` pattern, and define namespace strings as Go constants in a new `namespaces.go` file.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| RLIM-01 | Endpoint-level rate limiting configurable via annotation (`qos/ratelimit/router` namespace) | KrakenD docs confirm: `qos/ratelimit/router` is endpoint-level extra_config. Fields: max_rate, capacity, every, client_max_rate, client_capacity, strategy, key. None are required -- any subset can be specified. |
| RLIM-02 | Rate limit settings include max_rate, capacity, and strategy (configurable per service and per method) | Add `RateLimitConfig` message to `GatewayConfig` (service default) and `EndpointConfig` (method override). Resolver merges: method-level replaces entire service-level config if present. |
| RLIM-03 | Backend-level rate limiting configurable via annotation (`qos/ratelimit/proxy` namespace) | KrakenD docs confirm: `qos/ratelimit/proxy` is backend-level extra_config. Fields: max_rate (required), capacity (required), every. This goes in `Backend.ExtraConfig`, not `Endpoint.ExtraConfig`. |
| AUTH-01 | JWT validation configurable via service-level annotation (`auth/validator` namespace) | KrakenD docs confirm: `auth/validator` is endpoint-level extra_config. Minimum required: `alg` + either `jwk_url` or `jwk_local_path`. Service-level only per success criteria -- makes sense since JWT config is typically uniform across a service. |
| AUTH-02 | JWT config includes JWK URL, algorithm, issuer, and audience fields | All four are confirmed KrakenD `auth/validator` fields. `alg` defaults to RS256. `audience` is an array. `issuer` is a string. `jwk_url` is a string URL. |
| AUTH-03 | JWT claim propagation configurable (forward claims as backend headers) | `propagate_claims` is an array of `[claim_name, header_name]` pairs within `auth/validator`. Propagated headers must also appear in `input_headers`. The generator must auto-add propagated header names to `input_headers`. |
| RESL-01 | Circuit breaker configurable via annotation (`qos/circuit-breaker` namespace) at service and method level | KrakenD docs confirm: `qos/circuit-breaker` is backend-level extra_config. All three fields (interval, timeout, max_errors) are required integers. Service default + method override semantics. |
| RESL-02 | Circuit breaker settings include interval, timeout, and max_errors | Confirmed from KrakenD docs. `interval` = error counting window (seconds), `timeout` = seconds before retry, `max_errors` = consecutive error threshold. Optional: `name`, `log_status_change`. |
| RESL-03 | Concurrent calls configurable per endpoint (backend `concurrent_calls` field) | KrakenD docs confirm: `concurrent_calls` is a top-level endpoint field (NOT inside extra_config), type integer, default 1. Service default + method override. |
| RESL-04 | Backend caching configurable via annotation (`qos/http-cache` namespace) | KrakenD docs confirm: `qos/http-cache` is backend-level extra_config. Fields: `shared` (bool), plus optional `max_items`/`max_size` pair. Only GET/HEAD methods are cached. Cache duration controlled by backend Cache-Control header, not KrakenD config. |
| VALD-03 | All extra_config namespace strings are Go constants validated against known KrakenD namespaces | Define constants in `internal/krakendgen/namespaces.go`. Validate at generation time that only known namespace keys appear in output. Unit test that constants match expected strings. |
| TEST-02 | Golden file tests cover gateway features (rate limiting, JWT, circuit breaker, caching, concurrent calls) | Add test protos exercising each feature individually and in combination. Follow established golden test pattern from Phase 12. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/protobuf/compiler/protogen | v1.36.11 (existing) | Plugin framework for reading proto annotations | Same framework used by Phase 12 |
| encoding/json (stdlib) | Go 1.24.7 | JSON marshaling of KrakenD config structs including extra_config | No external dependency needed |
| internal/krakendgen (existing) | N/A | Generator logic, types, validation | Phase 12 foundation -- extend, don't replace |
| proto/sebuf/krakend/krakend.proto (existing) | N/A | Gateway-specific proto annotations | Phase 12 foundation -- add new messages |
| krakend/krakend.pb.go (existing) | N/A | Generated Go code for krakend annotations | Regenerated after adding new messages |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| google.golang.org/protobuf/proto | v1.36.11 (existing) | proto.GetExtension for reading krakend annotations | Reading new fields from GatewayConfig/EndpointConfig |
| internal/annotations (existing) | N/A | CombineHeaders for merging propagated claim headers | AUTH-03 requires adding propagated claim headers to input_headers |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `map[string]any` for ExtraConfig | Typed wrapper struct per namespace | map[string]any is simpler, matches KrakenD's JSON structure directly, avoids needing a typed-struct-to-map conversion step |
| Adding fields to existing GatewayConfig/EndpointConfig | New separate extensions (51003, 51004, etc.) | Adding to existing messages is cleaner -- keeps all service-level config in one annotation, all method-level in another. Avoids annotation proliferation. |
| Modeling every KrakenD auth/validator field | Subset of most-used fields | Model the core fields (alg, jwk_url, audience, issuer, cache, propagate_claims). Advanced fields (cipher_suites, key_identify_strategy) are niche and can be added later if needed. |

**Installation:**
```bash
# No new dependencies. After modifying krakend.proto:
cd proto && buf generate   # Regenerate krakend.pb.go
go mod tidy
```

## Architecture Patterns

### Recommended Project Structure
```
proto/sebuf/krakend/
  krakend.proto              # MODIFIED: add RateLimitConfig, JWTConfig, CircuitBreakerConfig, CacheConfig messages
krakend/
  krakend.pb.go              # REGENERATED after proto changes
internal/krakendgen/
  types.go                   # MODIFIED: add ExtraConfig fields to Endpoint and Backend, add ConcurrentCalls to Endpoint
  generator.go               # MODIFIED: add resolver functions for each feature, build extra_config maps
  namespaces.go              # NEW: Go constants for KrakenD namespace strings + allowlist
  namespaces_test.go         # NEW: Unit tests for namespace constants
  golden_test.go             # MODIFIED: add test cases for gateway features
  testdata/
    proto/
      rate_limit_service.proto       # NEW: rate limiting test cases
      jwt_auth_service.proto         # NEW: JWT auth test cases
      circuit_breaker_service.proto  # NEW: circuit breaker test cases
      cache_concurrent_service.proto # NEW: caching + concurrent calls test cases
      full_gateway_service.proto     # NEW: all features combined
    golden/
      RateLimitService.krakend.json              # NEW
      JWTAuthService.krakend.json                # NEW
      CircuitBreakerService.krakend.json         # NEW
      CacheConcurrentService.krakend.json        # NEW
      FullGatewayService.krakend.json            # NEW
```

### Pattern 1: ExtraConfig as map[string]any
**What:** Use `map[string]any` with `omitempty` for the `extra_config` JSON field on both Endpoint and Backend structs.
**When to use:** For all extra_config output. KrakenD's extra_config is a heterogeneous map where each key is a namespace string and each value has a different schema.
**Why:** Typed Go structs (RateLimitConfig, etc.) are used internally for annotation parsing and config building. They get converted to `map[string]any` entries before marshaling. This avoids needing a custom JSON marshaler.
**Example:**
```go
// types.go -- updated
type Endpoint struct {
    Endpoint          string         `json:"endpoint"`
    Method            string         `json:"method"`
    OutputEncoding    string         `json:"output_encoding"`
    Timeout           string         `json:"timeout,omitempty"`
    ConcurrentCalls   int32          `json:"concurrent_calls,omitempty"`
    InputHeaders      []string       `json:"input_headers,omitempty"`
    InputQueryStrings []string       `json:"input_query_strings,omitempty"`
    Backend           []Backend      `json:"backend"`
    ExtraConfig       map[string]any `json:"extra_config,omitempty"`
}

type Backend struct {
    URLPattern  string         `json:"url_pattern"`
    Host        []string       `json:"host"`
    Method      string         `json:"method"`
    Encoding    string         `json:"encoding"`
    ExtraConfig map[string]any `json:"extra_config,omitempty"`
}
```

### Pattern 2: Service/Method Override for Each Feature
**What:** For every feature (rate limit, circuit breaker, caching, concurrent calls), read service-level default from GatewayConfig, then check for method-level override in EndpointConfig. Method-level replaces service-level entirely (not field-by-field merge).
**When to use:** For all features with both service and method annotations.
**Why:** Consistent with Phase 12's `resolveTimeout`/`resolveHost` pattern. Whole-object replacement is simpler and less surprising than field-by-field merge.
**Example:**
```go
// generator.go
func resolveRateLimit(gwConfig *krakend.GatewayConfig, epConfig *krakend.EndpointConfig) *krakend.RateLimitConfig {
    if epConfig != nil && epConfig.GetRateLimit() != nil {
        return epConfig.GetRateLimit()
    }
    if gwConfig != nil {
        return gwConfig.GetRateLimit()
    }
    return nil
}
```

### Pattern 3: Namespace Constants
**What:** Define all KrakenD namespace strings as Go constants in a single file. Use these constants when building extra_config maps. Never use string literals for namespaces.
**When to use:** Every time a namespace string is referenced in generator code or tests.
**Example:**
```go
// namespaces.go
package krakendgen

// KrakenD extra_config namespace constants.
const (
    NamespaceRateLimitRouter = "qos/ratelimit/router"
    NamespaceRateLimitProxy  = "qos/ratelimit/proxy"
    NamespaceAuthValidator   = "auth/validator"
    NamespaceCircuitBreaker  = "qos/circuit-breaker"
    NamespaceHTTPCache       = "qos/http-cache"
)

// KnownNamespaces is the allowlist of valid KrakenD extra_config keys
// that this generator may produce.
var KnownNamespaces = []string{
    NamespaceRateLimitRouter,
    NamespaceRateLimitProxy,
    NamespaceAuthValidator,
    NamespaceCircuitBreaker,
    NamespaceHTTPCache,
}
```

### Pattern 4: JWT Claim Propagation Auto-Adds Headers
**What:** When JWT claim propagation is configured, the propagated header names must be automatically added to `input_headers` so KrakenD forwards them to backends.
**When to use:** Whenever `propagate_claims` is set in the auth config.
**Why:** KrakenD's zero-trust model requires explicit `input_headers`. If claim propagation creates headers like `X-User` but `X-User` is not in `input_headers`, the backend never receives it. The generator must handle this automatically.
**Example:**
```go
// After deriving inputHeaders from sebuf.http annotations:
if jwtConfig != nil && len(jwtConfig.GetPropagateClaims()) > 0 {
    for _, claim := range jwtConfig.GetPropagateClaims() {
        headerName := claim.GetHeaderName()
        inputHeaders = appendIfMissing(inputHeaders, headerName)
    }
    sort.Strings(inputHeaders)
}
```

### Pattern 5: Typed Config Structs for extra_config Values
**What:** Build typed Go maps for each namespace's config values, then insert them into the `map[string]any` ExtraConfig.
**When to use:** When constructing the extra_config for each feature.
**Why:** Typed maps (using specific keys) ensure correct JSON field names. Building a `map[string]any` directly is cleaner than marshaling a struct then unmarshaling to map.
**Example:**
```go
func buildRateLimitRouterConfig(rl *krakend.RateLimitConfig) map[string]any {
    config := map[string]any{}
    if rl.GetMaxRate() > 0 {
        config["max_rate"] = rl.GetMaxRate()
    }
    if rl.GetCapacity() > 0 {
        config["capacity"] = rl.GetCapacity()
    }
    if rl.GetStrategy() != "" {
        config["strategy"] = rl.GetStrategy()
    }
    if rl.GetKey() != "" {
        config["key"] = rl.GetKey()
    }
    if rl.GetClientMaxRate() > 0 {
        config["client_max_rate"] = rl.GetClientMaxRate()
    }
    if rl.GetClientCapacity() > 0 {
        config["client_capacity"] = rl.GetClientCapacity()
    }
    if rl.GetEvery() != "" {
        config["every"] = rl.GetEvery()
    }
    return config
}
```

### Anti-Patterns to Avoid
- **Field-by-field merge for overrides:** Do NOT merge individual fields from service-level and method-level configs. If a method specifies a rate limit, it replaces the entire service-level rate limit. Partial merges create confusing semantics ("I set max_rate at method level but capacity came from service level?").
- **Inline namespace strings:** Never write `epExtraConfig["qos/ratelimit/router"] = ...` with a string literal. Always use the constant: `epExtraConfig[NamespaceRateLimitRouter] = ...`.
- **Modeling all KrakenD fields in proto:** Do not try to model every field from the KrakenD auth/validator docs (e.g., cipher_suites, key_identify_strategy). Model the core fields users need. Advanced config can be added in future phases.
- **Emitting empty extra_config:** Use `omitempty` on the ExtraConfig field. If no gateway features are annotated, the field should be absent from JSON, not `"extra_config": {}`.
- **concurrent_calls inside extra_config:** `concurrent_calls` is a top-level endpoint field, NOT nested inside extra_config. Do not put it in ExtraConfig.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Input header derivation | Custom header extraction | Existing `deriveInputHeaders()` + append propagated claim headers | Already handles service/method merge, sorting, nil-for-empty |
| Proto extension reading | Manual proto descriptor parsing | `proto.GetExtension(options, krakend.E_GatewayConfig)` | Established pattern from Phase 12 |
| JSON output formatting | String concatenation for extra_config | `json.MarshalIndent` with `map[string]any` ExtraConfig field | Correct escaping, valid JSON guaranteed |
| Namespace validation | Runtime string comparison | Compile-time constants + test-time allowlist check | Catches typos at test time, not deploy time |
| Service/method override logic | Custom merge framework | Simple `if methodConfig != nil { return methodConfig } else { return serviceConfig }` pattern | Phase 12 established this; no framework needed for simple override |

**Key insight:** Phase 12 already solved the hard architecture problems (plugin structure, annotation reading, golden testing, service/method override semantics). Phase 13 is incremental: add new annotation messages, add resolver functions following established patterns, add ExtraConfig fields to existing structs. The generator structure does not need to change.

## Common Pitfalls

### Pitfall 1: Namespace String Typos Are Silently Ignored
**What goes wrong:** Generator emits `"qos/rate-limit/router"` (with hyphen) instead of `"qos/ratelimit/router"` (no hyphen). KrakenD silently ignores the unknown namespace. No rate limiting is applied.
**Why it happens:** KrakenD namespaces are magic strings with no compile-time validation. They are easy to mistype.
**How to avoid:** Define all namespaces as Go constants in `namespaces.go`. Use constants in all generator code. Add a unit test that verifies each constant matches the expected string value. Never use string literals.
**Warning signs:** A golden test shows extra_config but KrakenD doesn't apply the feature at runtime.

### Pitfall 2: Propagated Claims Not in input_headers
**What goes wrong:** JWT claim propagation is configured (e.g., `"sub"` -> `"X-User"`), but `X-User` is not in `input_headers`. KrakenD creates the `X-User` header from the JWT claim but then strips it before forwarding to the backend (zero-trust model).
**Why it happens:** `propagate_claims` creates headers within the auth/validator component. Those headers exist inside KrakenD's pipeline but are subject to the same `input_headers` allowlist as any other header.
**How to avoid:** When the generator processes JWT config with `propagate_claims`, it must automatically add each propagated header name to the endpoint's `input_headers` array. This should happen before the final sort and dedup of input_headers.
**Warning signs:** Backend receives JWT-validated requests but with no user identity headers.

### Pitfall 3: Circuit Breaker at Wrong Config Level
**What goes wrong:** Generator puts `qos/circuit-breaker` in endpoint-level `extra_config`. KrakenD ignores it because circuit breaker is a backend-level component.
**Why it happens:** It is easy to confuse which features go at endpoint level vs backend level. KrakenD's documentation is clear but the scoping is non-obvious.
**How to avoid:** Maintain a clear mapping: rate limit router -> endpoint extra_config, auth/validator -> endpoint extra_config, rate limit proxy -> backend extra_config, circuit breaker -> backend extra_config, http-cache -> backend extra_config.
**Warning signs:** Golden test shows circuit breaker config but it is inside the endpoint object, not inside the backend object.

### Pitfall 4: concurrent_calls Placed in extra_config
**What goes wrong:** Generator puts `concurrent_calls` inside `extra_config`. KrakenD ignores it because `concurrent_calls` is a top-level endpoint field.
**Why it happens:** Most KrakenD features use extra_config. Developers assume concurrent_calls works the same way.
**How to avoid:** Add `ConcurrentCalls` as a direct field on the `Endpoint` struct with `json:"concurrent_calls,omitempty"`. Do not put it in ExtraConfig.
**Warning signs:** Golden test output shows `"extra_config": {"concurrent_calls": 3}` instead of `"concurrent_calls": 3` at endpoint level.

### Pitfall 5: Backend Rate Limit Confused with Endpoint Rate Limit
**What goes wrong:** Generator puts `qos/ratelimit/proxy` at endpoint level or `qos/ratelimit/router` at backend level. Wrong scope = silent no-op.
**Why it happens:** Both are rate limiting but at different scopes. Router = client-to-gateway (endpoint level). Proxy = gateway-to-backend (backend level).
**How to avoid:** Two separate proto messages: `RateLimitConfig` for router (on endpoint) and `BackendRateLimitConfig` for proxy (on backend). Two separate resolver functions. Two separate ExtraConfig insertions.
**Warning signs:** Rate limiting works in one direction but not the other.

### Pitfall 6: JSON Key Ordering in extra_config
**What goes wrong:** Golden tests fail because `map[string]any` produces non-deterministic key ordering in JSON output.
**Why it happens:** Go maps do not have guaranteed iteration order. `json.Marshal` iterates map keys in the order Go's runtime decides.
**How to avoid:** Go's `encoding/json` marshals map keys in sorted order (since Go 1.12). This is deterministic and matches golden file comparison. No special handling needed. BUT: if using `map[string]any` with mixed types (string and int keys), ensure consistency. Since KrakenD namespaces are all strings, sorted order is reliable.
**Warning signs:** N/A -- Go guarantees sorted map key output since Go 1.12.

### Pitfall 7: Proto Field Defaults Emitting Zero Values
**What goes wrong:** A `GatewayConfig` with no rate limit annotation causes the generated code to read `gwConfig.GetRateLimit()` which returns a zero-value `RateLimitConfig` (not nil). The generator then emits `"qos/ratelimit/router": {}` with all zero values.
**Why it happens:** Proto3 defaults non-message fields to zero. For message fields, `GetXxx()` on a nil receiver returns nil, but on a non-nil parent with an absent field, it may return a zero-value struct depending on how protobuf generates the code.
**How to avoid:** Always nil-check the return of `GetRateLimit()`, `GetCircuitBreaker()`, etc. before building the extra_config map. If the proto message field is absent, `GetXxx()` returns nil for message types in proto3. Test this explicitly.
**Warning signs:** Golden test output contains `"extra_config": {"qos/ratelimit/router": {}}` when no rate limit was annotated.

## Code Examples

Verified patterns from the existing codebase and KrakenD documentation:

### Extending krakend.proto with Rate Limiting
```protobuf
// Source: Derived from KrakenD docs + existing krakend.proto pattern
message RateLimitConfig {
  // Maximum requests per time period (all clients combined).
  int32 max_rate = 1;
  // Token bucket capacity (defaults to max_rate if 0).
  int32 capacity = 2;
  // Time period: "1s", "1m", "1h". Default: "1s".
  string every = 3;
  // Maximum requests per client per time period.
  int32 client_max_rate = 4;
  // Per-client token bucket capacity.
  int32 client_capacity = 5;
  // Client identification: "ip", "header", or "param".
  string strategy = 6;
  // Header name or param for client identification (when strategy is "header" or "param").
  string key = 7;
}

message BackendRateLimitConfig {
  // Maximum requests per time period to this backend.
  int32 max_rate = 1;
  // Token bucket capacity.
  int32 capacity = 2;
  // Time period. Default: "1s".
  string every = 3;
}
```

### JWT Auth Config Proto Message
```protobuf
// Source: Derived from KrakenD auth/validator docs
message JWTConfig {
  // Hashing algorithm: RS256, HS256, ES256, etc.
  string alg = 1;
  // Remote JWK endpoint URL.
  string jwk_url = 2;
  // Expected token audiences (all must match).
  repeated string audience = 3;
  // Expected token issuer.
  string issuer = 4;
  // Enable JWK key caching.
  bool cache = 5;
  // JWT claim to header propagation rules.
  repeated ClaimToHeader propagate_claims = 6;
}

message ClaimToHeader {
  // JWT claim name (supports dot notation for nested: "realm_access.role").
  string claim = 1;
  // HTTP header name to forward the claim value as.
  string header = 2;
}
```

### Building Endpoint ExtraConfig
```go
// Source: Derived from Phase 12 generator.go pattern
func buildEndpointExtraConfig(
    gwConfig *krakend.GatewayConfig,
    epConfig *krakend.EndpointConfig,
) map[string]any {
    extraConfig := map[string]any{}

    // Rate limit (endpoint level = qos/ratelimit/router)
    rl := resolveRateLimit(gwConfig, epConfig)
    if rl != nil {
        extraConfig[NamespaceRateLimitRouter] = buildRateLimitRouterConfig(rl)
    }

    // JWT auth (service level only)
    jwt := gwConfig.GetJwt()
    if jwt != nil {
        extraConfig[NamespaceAuthValidator] = buildAuthValidatorConfig(jwt)
    }

    if len(extraConfig) == 0 {
        return nil  // omitempty will omit the field
    }
    return extraConfig
}
```

### Building Backend ExtraConfig
```go
func buildBackendExtraConfig(
    gwConfig *krakend.GatewayConfig,
    epConfig *krakend.EndpointConfig,
) map[string]any {
    extraConfig := map[string]any{}

    // Circuit breaker (backend level)
    cb := resolveCircuitBreaker(gwConfig, epConfig)
    if cb != nil {
        extraConfig[NamespaceCircuitBreaker] = buildCircuitBreakerConfig(cb)
    }

    // Backend rate limit (backend level = qos/ratelimit/proxy)
    brl := resolveBackendRateLimit(gwConfig, epConfig)
    if brl != nil {
        extraConfig[NamespaceRateLimitProxy] = buildBackendRateLimitConfig(brl)
    }

    // HTTP cache (backend level)
    cache := resolveCache(gwConfig, epConfig)
    if cache != nil {
        extraConfig[NamespaceHTTPCache] = buildHTTPCacheConfig(cache)
    }

    if len(extraConfig) == 0 {
        return nil
    }
    return extraConfig
}
```

### KrakenD extra_config Scope Reference

Verified from KrakenD official documentation:

| Namespace | Scope | Go Struct Field |
|-----------|-------|-----------------|
| `qos/ratelimit/router` | endpoint | `Endpoint.ExtraConfig` |
| `auth/validator` | endpoint | `Endpoint.ExtraConfig` |
| `qos/ratelimit/proxy` | backend | `Backend.ExtraConfig` |
| `qos/circuit-breaker` | backend | `Backend.ExtraConfig` |
| `qos/http-cache` | backend | `Backend.ExtraConfig` |

**NOT in extra_config:**
| Field | Scope | Go Struct Field |
|-------|-------|-----------------|
| `concurrent_calls` | endpoint (top-level) | `Endpoint.ConcurrentCalls` |

### Expected JSON Output Example (All Features)
```json
[
  {
    "endpoint": "/api/v1/users",
    "method": "POST",
    "output_encoding": "json",
    "timeout": "3s",
    "concurrent_calls": 2,
    "input_headers": [
      "Authorization",
      "X-API-Key",
      "X-User"
    ],
    "extra_config": {
      "auth/validator": {
        "alg": "RS256",
        "jwk_url": "https://auth.example.com/.well-known/jwks.json",
        "audience": ["api.example.com"],
        "issuer": "https://auth.example.com",
        "cache": true,
        "propagate_claims": [
          ["sub", "X-User"]
        ]
      },
      "qos/ratelimit/router": {
        "max_rate": 100,
        "client_max_rate": 10,
        "strategy": "header",
        "key": "X-API-Key"
      }
    },
    "backend": [
      {
        "url_pattern": "/api/v1/users",
        "host": ["http://backend:8080"],
        "method": "POST",
        "encoding": "json",
        "extra_config": {
          "qos/circuit-breaker": {
            "interval": 60,
            "timeout": 10,
            "max_errors": 3,
            "log_status_change": true
          },
          "qos/http-cache": {
            "shared": true
          },
          "qos/ratelimit/proxy": {
            "max_rate": 50,
            "capacity": 50,
            "every": "1s"
          }
        }
      }
    ]
  }
]
```

## KrakenD Feature Config Reference

### qos/ratelimit/router (Endpoint Level)
**Scope:** endpoint extra_config
**Required fields:** None (any subset is valid)
**All fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| max_rate | number | - | Max requests (all clients) per time period |
| capacity | integer | 1 | Token bucket capacity (all clients) |
| every | string | "1s" | Time period (e.g., "1s", "1m", "1h") |
| client_max_rate | number | - | Max requests per individual client |
| client_capacity | integer | 1 | Per-client token bucket capacity |
| strategy | string | - | Client identification: "ip", "header", "param" |
| key | string | - | Header/param name for client ID |

### qos/ratelimit/proxy (Backend Level)
**Scope:** backend extra_config
**Required fields:** max_rate, capacity
**All fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| max_rate | number | - | Max requests per time period to backend |
| capacity | integer | 1 | Token bucket capacity |
| every | string | "1s" | Time period |

### auth/validator (Endpoint Level)
**Scope:** endpoint extra_config
**Required fields:** alg + (jwk_url or jwk_local_path)
**Core fields for Phase 13:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| alg | string | "RS256" | Hashing algorithm |
| jwk_url | string | - | Remote JWK endpoint URL |
| audience | array | - | Expected audiences (all must match) |
| issuer | string | - | Expected issuer |
| cache | boolean | false | Enable JWK caching |
| propagate_claims | array of [claim, header] pairs | - | Forward claims as backend headers |

### qos/circuit-breaker (Backend Level)
**Scope:** backend extra_config
**Required fields:** interval, timeout, max_errors
**All fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| interval | integer | - | Error counting window (seconds) |
| timeout | integer | - | Seconds before retry |
| max_errors | integer | - | Consecutive errors to open circuit |
| name | string | - | Friendly name for logging |
| log_status_change | boolean | false | Log state transitions |

### qos/http-cache (Backend Level)
**Scope:** backend extra_config
**Required fields:** None (can declare with just shared:true, or max_items+max_size, or empty)
**All fields:**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| shared | boolean | false | Share cache across backends with same request |
| max_items | integer | - | Max items in LRU cache (must pair with max_size) |
| max_size | integer | - | Max bytes in LRU cache (must pair with max_items) |

### concurrent_calls (Endpoint Top-Level)
**Scope:** endpoint (direct field, NOT extra_config)
**Type:** integer
**Default:** 1
**Description:** Number of parallel requests to backend; returns first response.

## Proto Annotation Design

### Design Decision: Add to Existing Messages vs New Extensions

**Recommendation: Add new fields to existing `GatewayConfig` and `EndpointConfig` messages.**

Rationale:
- Phase 12 established `GatewayConfig` (service-level, ext 51001) and `EndpointConfig` (method-level, ext 51002)
- Adding fields to existing messages keeps all service-level config in one annotation and all method-level config in another
- No new extension numbers needed
- Proto3 is backward-compatible -- adding fields does not break existing protos
- Existing test protos that don't use the new fields will still work

**Updated krakend.proto structure:**
```protobuf
message GatewayConfig {
  repeated string host = 1;
  string timeout = 2;

  // Phase 13 additions:
  RateLimitConfig rate_limit = 3;
  BackendRateLimitConfig backend_rate_limit = 4;
  JWTConfig jwt = 5;
  CircuitBreakerConfig circuit_breaker = 6;
  CacheConfig cache = 7;
  int32 concurrent_calls = 8;
}

message EndpointConfig {
  repeated string host = 1;
  string timeout = 2;

  // Phase 13 additions:
  RateLimitConfig rate_limit = 3;
  BackendRateLimitConfig backend_rate_limit = 4;
  CircuitBreakerConfig circuit_breaker = 5;
  CacheConfig cache = 6;
  int32 concurrent_calls = 7;
}
```

**Key design notes:**
- JWT is service-level only (on GatewayConfig, not EndpointConfig) -- JWT config is uniform per service
- Rate limiting (router) is on both -- service default + method override
- Backend rate limiting (proxy) is on both -- service default + method override
- Circuit breaker is on both -- service default + method override
- Cache is on both -- service default + method override
- Concurrent calls is on both -- service default + method override
- JWT is NOT on EndpointConfig -- per success criteria, JWT is service-level annotation only

### propagate_claims Proto Design

The KrakenD `propagate_claims` format is `[[claim, header], ...]`. In proto, model this as a repeated message:

```protobuf
message ClaimToHeader {
  string claim = 1;   // JWT claim name (dot notation for nested)
  string header = 2;  // Header name to forward as
}
```

When serializing to JSON, convert to the array-of-arrays format KrakenD expects:
```go
func buildPropagateClaims(claims []*krakend.ClaimToHeader) [][]string {
    result := make([][]string, len(claims))
    for i, c := range claims {
        result[i] = []string{c.GetClaim(), c.GetHeader()}
    }
    return result
}
```

### CacheConfig Proto Design

KrakenD's `qos/http-cache` has three optional fields. Model as:
```protobuf
message CacheConfig {
  bool shared = 1;
  int32 max_items = 2;  // 0 = not set
  int32 max_size = 3;   // 0 = not set (bytes)
}
```

When building the JSON, only include fields that are non-zero:
```go
func buildHTTPCacheConfig(cache *krakend.CacheConfig) map[string]any {
    config := map[string]any{}
    if cache.GetShared() {
        config["shared"] = true
    }
    if cache.GetMaxItems() > 0 {
        config["max_items"] = cache.GetMaxItems()
    }
    if cache.GetMaxSize() > 0 {
        config["max_size"] = cache.GetMaxSize()
    }
    return config
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hand-writing KrakenD extra_config JSON | Generating from proto annotations via protoc-gen-krakend | Phase 13 (new) | Proto becomes single source of truth for gateway config |
| Separate gateway config files per environment | Proto annotations for feature intent, FC for environment values | Phase 12+13 | API design intent (rate limits, auth) lives with the API definition |
| KrakenD Designer UI for config | Code-generation from proto (DevOps-as-code) | Phase 13 (new) | Config is version-controlled, reviewable, diff-able |

**Deprecated/outdated:**
- N/A -- Phase 13 builds on Phase 12 patterns which are current

## Open Questions

1. **Should JWT config support method-level overrides?**
   - What we know: Success criteria #2 says "A service annotated with JWT validation settings produces endpoint-level extra_config" -- phrasing suggests service-level only. The FEATURES.md research also says "Auth lives at service level only -- JWT config is typically uniform across all endpoints in a service."
   - Recommendation: Keep JWT on `GatewayConfig` only. If a method needs different JWT rules, it is likely a different service. This avoids complexity and matches KrakenD best practices where auth is typically uniform per service.

2. **Should CacheConfig require both max_items and max_size together?**
   - What we know: KrakenD docs say max_items and max_size must be paired. Setting one without the other is ambiguous.
   - Recommendation: Add a generation-time validation that if either max_items or max_size is non-zero, both must be non-zero. Emit a clear error message if they are mismatched. This catches proto annotation mistakes early.

3. **Should the generator validate that circuit breaker interval/timeout/max_errors are all set?**
   - What we know: KrakenD docs list all three as required for `qos/circuit-breaker`.
   - Recommendation: Yes -- validate at generation time. If a `CircuitBreakerConfig` is present but any of the three fields is zero, emit an error. All three are semantically required; zero values are not meaningful.

4. **How should existing golden files be affected?**
   - What we know: Adding `ExtraConfig map[string]any` with `omitempty` to Endpoint and Backend structs will NOT change existing golden files because the field is omitted when nil/empty. Adding `ConcurrentCalls int32` with `omitempty` will also not change existing golden files because 0 is omitted.
   - Recommendation: Verify this assumption in the first plan before adding features. Run existing golden tests after struct changes to confirm no regressions.

## Sources

### Primary (HIGH confidence)
- [KrakenD Rate Limiting (Router)](https://www.krakend.io/docs/endpoints/rate-limit/) -- `qos/ratelimit/router` fields, types, and defaults. Verified 2026-02-25.
- [KrakenD Rate Limiting (Proxy)](https://www.krakend.io/docs/backends/rate-limit/) -- `qos/ratelimit/proxy` fields. max_rate and capacity are required. Verified 2026-02-25.
- [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/) -- `auth/validator` fields, propagate_claims format (array of [claim, header] pairs). Verified 2026-02-25.
- [KrakenD Circuit Breaker](https://www.krakend.io/docs/backends/circuit-breaker/) -- `qos/circuit-breaker` fields. interval, timeout, max_errors all required integers. Verified 2026-02-25.
- [KrakenD Backend Caching](https://www.krakend.io/docs/backends/caching/) -- `qos/http-cache` fields. shared, max_items/max_size pair. Only GET/HEAD cached. Verified 2026-02-25.
- [KrakenD Endpoint Configuration](https://www.krakend.io/docs/endpoints/) -- concurrent_calls is top-level endpoint field, NOT extra_config. Verified 2026-02-25.
- [KrakenD Configuration Structure](https://www.krakend.io/docs/configuration/structure/) -- extra_config scoping: service, endpoint, backend levels. Verified 2026-02-25.
- Existing sebuf codebase (verified 2026-02-25):
  - `proto/sebuf/krakend/krakend.proto` -- Current annotation definitions (host, timeout only)
  - `krakend/krakend.pb.go` -- Generated Go code with E_GatewayConfig (51001), E_EndpointConfig (51002)
  - `internal/krakendgen/types.go` -- Current Endpoint/Backend structs (no ExtraConfig yet)
  - `internal/krakendgen/generator.go` -- resolveHost, resolveTimeout, deriveInputHeaders patterns
  - `internal/krakendgen/golden_test.go` -- 6 success cases + 3 validation error cases
  - `.planning/research/FEATURES.md` -- Feature landscape with KrakenD namespace reference
  - `.planning/research/PITFALLS.md` -- Pitfall #6 on namespace typos
  - `.planning/research/ARCHITECTURE.md` -- Architecture patterns including ExtraConfig map[string]any recommendation

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` -- Phase 13 requirement definitions (RLIM-01 through TEST-02)
- `.planning/ROADMAP.md` -- Phase 13 success criteria and plan structure
- `.planning/phases/12-annotations-and-core-endpoint-generation/12-RESEARCH.md` -- Phase 12 research with annotation design notes

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- Zero new dependencies; extends existing Phase 12 patterns with identical tools
- Architecture: HIGH -- ExtraConfig as map[string]any is the only viable approach; proto message design follows established patterns; all scoping (endpoint vs backend) verified from official KrakenD docs
- Pitfalls: HIGH -- Namespace typos, scope confusion, propagate_claims + input_headers interaction all documented in prior research and verified against official docs
- Proto annotation design: HIGH -- Adding fields to existing messages is backward-compatible; no new extension numbers needed; JWT service-level-only matches success criteria

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (30 days -- stable domain, KrakenD API is mature)
