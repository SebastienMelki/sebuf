# Requirements: protoc-gen-krakend (v1.1)

**Defined:** 2026-02-25
**Core Value:** Proto definitions are the single source of truth — KrakenD gateway config stays in sync with service definitions automatically.

## v1.1 Requirements

Requirements for the KrakenD config generator milestone. Each maps to roadmap phases.

### Proto Annotations

- [ ] **ANNO-01**: New proto package `proto/sebuf/krakend/` with gateway-specific annotations (extension numbers 51000+)
- [ ] **ANNO-02**: Service-level `gateway_config` annotation for service-wide defaults (host, timeout, rate limit, auth, circuit breaker)
- [ ] **ANNO-03**: Method-level `endpoint_config` annotation for per-RPC overrides (timeout, rate limit, circuit breaker)
- [ ] **ANNO-04**: Method-level config always overrides service-level config for the same setting

### Core Generation

- [ ] **CORE-01**: Plugin reads `sebuf.http.config` annotations to extract HTTP path and method for each RPC
- [ ] **CORE-02**: Plugin generates one JSON file per proto service (`{ServiceName}.krakend.json`) containing an array of KrakenD endpoint objects
- [ ] **CORE-03**: Backend host is configurable via plugin parameter (`--krakend_opt=host=http://backend:8080`)
- [ ] **CORE-04**: Backend host is configurable via service-level annotation (overrides plugin parameter)
- [ ] **CORE-05**: Per-endpoint timeout is configurable via service-level default and method-level override
- [ ] **CORE-06**: Output encoding defaults to JSON for all endpoints

### Auto-Derived Forwarding

- [ ] **FWD-01**: `input_headers` auto-populated from `sebuf.http.service_headers` and `sebuf.http.method_headers` annotations
- [ ] **FWD-02**: `input_query_strings` auto-populated from `sebuf.http.query` annotations on request message fields
- [ ] **FWD-03**: Auto-derived headers and query strings are never empty arrays (KrakenD zero-trust model)

### Rate Limiting

- [ ] **RLIM-01**: Endpoint-level rate limiting configurable via annotation (`qos/ratelimit/router` namespace)
- [ ] **RLIM-02**: Rate limit settings include max_rate, capacity, and strategy (configurable per service and per method)
- [ ] **RLIM-03**: Backend-level rate limiting configurable via annotation (`qos/ratelimit/proxy` namespace)

### Authentication

- [ ] **AUTH-01**: JWT validation configurable via service-level annotation (`auth/validator` namespace)
- [ ] **AUTH-02**: JWT config includes JWK URL, algorithm, issuer, and audience fields
- [ ] **AUTH-03**: JWT claim propagation configurable (forward claims as backend headers)

### Resilience

- [ ] **RESL-01**: Circuit breaker configurable via annotation (`qos/circuit-breaker` namespace) at service and method level
- [ ] **RESL-02**: Circuit breaker settings include interval, timeout, and max_errors
- [ ] **RESL-03**: Concurrent calls configurable per endpoint (backend `concurrent_calls` field)
- [ ] **RESL-04**: Backend caching configurable via annotation (`qos/http-cache` namespace)

### Validation

- [ ] **VALD-01**: Generation fails with clear error if two RPCs produce identical (path, method) tuples
- [ ] **VALD-02**: Generation fails with clear error if static and parameterized routes conflict at the same path level
- [ ] **VALD-03**: All extra_config namespace strings are Go constants validated against known KrakenD namespaces

### Testing

- [ ] **TEST-01**: Golden file tests cover all core generation scenarios (endpoint routing, backend mapping, timeouts)
- [ ] **TEST-02**: Golden file tests cover gateway features (rate limiting, JWT, circuit breaker, caching, concurrent calls)
- [ ] **TEST-03**: Golden file tests cover auto-derived header and query string forwarding
- [ ] **TEST-04**: Golden file tests cover generation-time validation errors (duplicate endpoints, path conflicts)

### Documentation

- [ ] **DOCS-01**: Example proto file demonstrating all KrakenD annotations with a working Flexible Config setup
- [ ] **DOCS-02**: Flexible Config integration guide showing how to compose per-service fragments into a full krakend.json

## Future Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Response Shaping

- **RESP-01**: Response field filtering via allow/deny lists
- **RESP-02**: Response field mapping (rename fields at gateway level)
- **RESP-03**: Response grouping (nest response under key)
- **RESP-04**: Response target extraction (unwrap nested field)
- **RESP-05**: Collection handling (auto-derive from unwrap annotations)

### Advanced Features

- **ADV-01**: raw_extra_config escape hatch for arbitrary KrakenD JSON injection
- **ADV-02**: CORS configuration from proto annotations
- **ADV-03**: Security headers (HSTS, X-Frame-Options, etc.)
- **ADV-04**: No-op passthrough mode (proxy without transformation)
- **ADV-05**: Cross-service path variable collision detection

### Distribution

- **DIST-01**: GoReleaser integration for protoc-gen-krakend binary
- **DIST-02**: BSR publishing for proto/sebuf/krakend/ package
- **DIST-03**: CLAUDE.md updates with krakendgen architecture

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full krakend.json generation | Infrastructure concern — global config (port, TLS, telemetry) is deployment-specific |
| API key authentication | KrakenD Enterprise-only feature |
| gRPC backend integration | Enterprise-only, contradicts sebuf's HTTP focus |
| Telemetry/logging config | Operational concern, not API shape |
| CEL request validation | Niche, backend already validates via protovalidate |
| JSON Schema validation | Redundant with protovalidate |
| Sequential proxy | Anti-pattern per KrakenD docs |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| ANNO-01 | — | Pending |
| ANNO-02 | — | Pending |
| ANNO-03 | — | Pending |
| ANNO-04 | — | Pending |
| CORE-01 | — | Pending |
| CORE-02 | — | Pending |
| CORE-03 | — | Pending |
| CORE-04 | — | Pending |
| CORE-05 | — | Pending |
| CORE-06 | — | Pending |
| FWD-01 | — | Pending |
| FWD-02 | — | Pending |
| FWD-03 | — | Pending |
| RLIM-01 | — | Pending |
| RLIM-02 | — | Pending |
| RLIM-03 | — | Pending |
| AUTH-01 | — | Pending |
| AUTH-02 | — | Pending |
| AUTH-03 | — | Pending |
| RESL-01 | — | Pending |
| RESL-02 | — | Pending |
| RESL-03 | — | Pending |
| RESL-04 | — | Pending |
| VALD-01 | — | Pending |
| VALD-02 | — | Pending |
| VALD-03 | — | Pending |
| TEST-01 | — | Pending |
| TEST-02 | — | Pending |
| TEST-03 | — | Pending |
| TEST-04 | — | Pending |
| DOCS-01 | — | Pending |
| DOCS-02 | — | Pending |

**Coverage:**
- v1.1 requirements: 32 total
- Mapped to phases: 0
- Unmapped: 32

---
*Requirements defined: 2026-02-25*
*Last updated: 2026-02-25 after initial definition*
