# Technology Stack: protoc-gen-krakend

**Project:** protoc-gen-krakend (KrakenD API Gateway config generator for sebuf)
**Researched:** 2026-02-25
**Scope:** Stack additions for a new protoc plugin that generates KrakenD endpoint config fragments from proto service definitions. Does NOT cover existing sebuf infrastructure (protogen, annotations, golden tests).

---

## Recommended Stack

### KrakenD Target Version and Config Schema

| Property | Value | Confidence | Rationale |
|----------|-------|------------|-----------|
| KrakenD CE target | v2.13.x | HIGH | Latest stable release (v2.13.1, released 2026-02-18). Config is backward-compatible within all v2.x releases. |
| Config `version` field | `3` (integer) | HIGH | The only active config format version since KrakenD v2.0. Versions 1 and 2 are deprecated (v1 since 2016). This is the config format version, not KrakenD software version. |
| JSON Schema reference | `https://www.krakend.io/schema/v2.13/krakend.json` | HIGH | Published at [krakend/krakend-schema](https://github.com/krakend/krakend-schema). Enables IDE autocomplete and `krakend check --lint` validation. Schema versions exist for v2.1 through v2.13. |
| Output file format | JSON | HIGH | KrakenD's recommended and default format. JSON is the ONLY format supporting `krakend check --lint`, the only format compatible with KrakenDesigner, and the format used in all official documentation. YAML/TOML/HCL are supported at runtime but lack linting. |

**Sources:** [KrakenD Configuration Guide](https://www.krakend.io/docs/configuration/), [KrakenD Schema Repo](https://github.com/krakend/krakend-schema), [KrakenD CE Releases](https://github.com/krakend/krakend-ce/releases), [KrakenD Supported Formats](https://www.krakend.io/docs/configuration/supported-formats/)

---

### Core Framework (Zero New Dependencies)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `google.golang.org/protobuf/compiler/protogen` | v1.36.11 (existing) | Protoc plugin framework | Already used by all 5 sebuf plugins. Provides CodeGeneratorRequest/Response, typed proto access, `plugin.NewGeneratedFile()` for output. |
| `google.golang.org/protobuf/proto` | v1.36.11 (existing) | Proto extension extraction | Already used in `internal/annotations/` for reading `sebuf.http` annotations via `proto.GetExtension()`. |
| `encoding/json` | Go 1.24 stdlib | JSON output generation | Produces valid, deterministic JSON from Go structs via `json.MarshalIndent()`. `json:"field,omitempty"` handles optional fields cleanly. No external dependency needed. |

**No new entries in `go.mod`.** The KrakenD plugin generates plain JSON files. It does not import any KrakenD-specific Go packages, does not link against KrakenD, and does not need a JSON schema validation library (validation is done externally via `krakend check --lint`).

---

### JSON Generation Approach

**Use `encoding/json.MarshalIndent()` with typed Go structs. Do NOT use `text/template`.**

| Approach | Verdict | Rationale |
|----------|---------|-----------|
| `encoding/json` + Go structs | **USE THIS** | Type-safe at compile time. Produces guaranteed-valid JSON. Testable via golden files. Consistent with sebuf's pattern (OpenAPI generator uses typed Go structs for document construction). `json:"field,omitempty"` handles optional fields naturally. |
| `text/template` | **DO NOT USE** | Error-prone for JSON generation (comma handling between array elements, string escaping, nested object formatting). No compile-time type checking. Hard to maintain and debug. Not used by any existing sebuf generator. |
| Third-party JSON builder library | **DO NOT USE** | Unnecessary dependency. `encoding/json` is sufficient for the output complexity. |

**Confidence:** HIGH. The `p3ym4n/krakend-generator` Go project uses the identical approach (Go structs marshaled to JSON) for generating KrakenD configs, validating this pattern.

**Implementation -- define Go structs that mirror KrakenD config objects:**

```go
// internal/krakendgen/types.go

// Endpoint represents a KrakenD endpoint configuration object.
type Endpoint struct {
    Endpoint          string                 `json:"endpoint"`
    Method            string                 `json:"method,omitempty"`
    OutputEncoding    string                 `json:"output_encoding,omitempty"`
    Timeout           string                 `json:"timeout,omitempty"`
    CacheTTL          string                 `json:"cache_ttl,omitempty"`
    InputHeaders      []string               `json:"input_headers,omitempty"`
    InputQueryStrings []string               `json:"input_query_strings,omitempty"`
    Backend           []Backend              `json:"backend"`
    ExtraConfig       map[string]interface{} `json:"extra_config,omitempty"`
}

// Backend represents a KrakenD backend configuration object.
type Backend struct {
    URLPattern  string                 `json:"url_pattern"`
    Method      string                 `json:"method,omitempty"`
    Host        []string               `json:"host,omitempty"`
    Encoding    string                 `json:"encoding,omitempty"`
    Group       string                 `json:"group,omitempty"`
    Allow       []string               `json:"allow,omitempty"`
    Deny        []string               `json:"deny,omitempty"`
    Target      string                 `json:"target,omitempty"`
    ExtraConfig map[string]interface{} `json:"extra_config,omitempty"`
}
```

**Why `map[string]interface{}` for `extra_config`:** KrakenD extra_config namespaces have heterogeneous schemas (rate limiting differs from JWT validation differs from circuit breaker). Using `map[string]interface{}` with typed helper constructors is the pragmatic approach. The alternative (a discriminated union type per namespace) adds complexity without benefit since the output is JSON.

---

### Output File Structure

**Per-service JSON files, each containing an array of KrakenD endpoint objects.**

| Output | Filename Pattern | Content |
|--------|-----------------|---------|
| Service endpoint fragment | `{ServiceName}.krakend.json` | JSON array of endpoint objects for one protobuf service |

**Why per-service files (not one monolithic `krakend.json`):**
1. Matches the existing sebuf pattern -- `protoc-gen-openapiv3` outputs `{ServiceName}.openapi.yaml`, one file per service
2. Natural fit for KrakenD Flexible Config `{{ include }}` composition
3. Enables independent service deployment and selective gateway config assembly
4. Avoids conflicts when protoc processes multiple .proto files

**Why a JSON array of endpoints (not a full KrakenD config with `$schema`, `version`, `extra_config`):**
- The plugin generates **endpoint fragments**, not a deployable gateway config
- Global settings (CORS, telemetry, TLS, service-level rate limits) are deployment-specific and belong in the user's Flexible Config base template
- Fragment approach composes cleanly: `{{ include "UserService.krakend.json" }}`

**Example output (`UserService.krakend.json`):**
```json
[
  {
    "endpoint": "/api/v1/users",
    "method": "POST",
    "output_encoding": "json",
    "input_headers": [
      "Content-Type",
      "X-API-Key"
    ],
    "backend": [
      {
        "url_pattern": "/api/v1/users",
        "method": "POST",
        "host": [
          "http://user-service:8080"
        ],
        "encoding": "json"
      }
    ],
    "extra_config": {
      "qos/ratelimit/router": {
        "max_rate": 100,
        "every": "1s"
      }
    }
  },
  {
    "endpoint": "/api/v1/users/{id}",
    "method": "GET",
    "output_encoding": "json",
    "input_headers": [
      "X-API-Key"
    ],
    "input_query_strings": [
      "include_deleted"
    ],
    "backend": [
      {
        "url_pattern": "/api/v1/users/{id}",
        "method": "GET",
        "host": [
          "http://user-service:8080"
        ],
        "encoding": "json"
      }
    ]
  }
]
```

**Confidence:** HIGH -- follows established sebuf output patterns and KrakenD Flexible Config documentation.

---

### KrakenD Flexible Config Integration

The generated JSON fragments are designed for consumption via KrakenD's Flexible Config template system. Users compose a full gateway config from these fragments.

**Flexible Config essentials:**

| Concept | Description |
|---------|-------------|
| Base template | `krakend.tmpl` -- the main config file treated as a Go template when `FC_ENABLE=1` |
| Partials (`FC_PARTIALS`) | Text files inserted **as-is** via `{{ include "file.json" }}`. No template evaluation. |
| Templates (`FC_TEMPLATES`) | `.tmpl` files evaluated as Go templates via `{{ template "file.tmpl" . }}` |
| Settings (`FC_SETTINGS`) | JSON files providing data accessible as `.variable` in templates |
| Output debug (`FC_OUT`) | Saves rendered config for debugging, not required at runtime |

**Why partials (not templates) for generated fragments:**
Generated JSON fragments are complete, valid JSON. They should be included as-is via `{{ include }}` -- no template evaluation needed. This is simpler and avoids template syntax conflicts in the JSON.

**Example user `krakend.tmpl`:**
```
{
  "$schema": "https://www.krakend.io/schema/v2.13/krakend.json",
  "version": 3,
  "endpoints": [
    {{ include "UserService.krakend.json" }},
    {{ include "OrderService.krakend.json" }}
  ],
  "extra_config": {
    {{ include "global_config.json" }}
  }
}
```

**Note on comma handling:** The `{{ include }}` approach requires manual commas between service includes. This is a known Flexible Config pattern. Users can alternatively use `range` with settings files listing services, using `{{if $idx}},{{end}}` for comma separation. The plugin's documentation should show both patterns.

**Confidence:** HIGH -- [KrakenD Flexible Config docs](https://www.krakend.io/docs/configuration/flexible-config/) and [KrakenD Templates docs](https://www.krakend.io/docs/configuration/templates/) confirm this approach.

---

### Proto Annotations (New `sebuf.krakend` Package)

New KrakenD-specific annotations live in `proto/sebuf/krakend/annotations.proto` under a **separate** `sebuf.krakend` package.

**Why a separate package, not extending `sebuf.http`:**
1. KrakenD configuration is a gateway concern, not HTTP semantics. Rate limiting, circuit breaking, and JWT validation are gateway behaviors that don't belong in HTTP routing annotations.
2. Users who don't use KrakenD should never encounter these annotations.
3. Clean extension number space: existing `sebuf.http` uses 50003-50020. A new package avoids numbering conflicts.
4. Follows the principle of optional dependencies: importing `sebuf/krakend/annotations.proto` is opt-in.

**Annotation design:**

| Level | Annotation | Extension # | Purpose |
|-------|-----------|-------------|---------|
| Service | `krakend_service` on `ServiceOptions` | 51001 | Default backend host(s), default timeout, service-wide extra_config |
| Method | `krakend_endpoint` on `MethodOptions` | 51002 | Per-endpoint: timeout, cache_ttl, rate limiting, auth/JWT config |
| Method | `krakend_backend` on `MethodOptions` | 51003 | Per-endpoint backend overrides: encoding, host override, allow/deny lists, circuit breaker |

**What NOT to annotate (reuse from `sebuf.http`):**
- HTTP path and method -- `sebuf.http.config` (ext 50003)
- Service base path -- `sebuf.http.service_config` (ext 50004)
- Query parameters -- `sebuf.http.query` (ext 50008)
- Required headers -- `sebuf.http.service_headers` / `sebuf.http.method_headers`

The KrakenD generator reads BOTH `sebuf.http` and `sebuf.krakend` annotations, using the existing `internal/annotations` package for HTTP routing and a new package for KrakenD-specific config.

**Confidence:** HIGH for separation approach. MEDIUM for exact annotation message structure (will be refined during implementation based on ergonomics).

---

### KrakenD Extra Config Namespace Coverage

The plugin should support generating these KrakenD `extra_config` namespaces, driven by annotation values:

**Endpoint-level namespaces (in endpoint `extra_config`):**

| Namespace | Purpose | KrakenD Scope | Priority | Confidence |
|-----------|---------|---------------|----------|------------|
| `qos/ratelimit/router` | Rate limiting (max_rate, client_max_rate, every, strategy) | Endpoint | P0 | HIGH |
| `auth/validator` | JWT validation (alg, jwk_url, audience, roles, scopes, propagate_claims) | Endpoint | P0 | HIGH |
| `validation/json-schema` | Request body JSON schema validation | Endpoint | P2 | LOW |

**Backend-level namespaces (in backend `extra_config`):**

| Namespace | Purpose | KrakenD Scope | Priority | Confidence |
|-----------|---------|---------------|----------|------------|
| `qos/circuit-breaker/http` | Circuit breaker (interval, timeout, max_errors) | Backend | P1 | HIGH |
| `qos/ratelimit/proxy` | Backend-specific rate limiting | Backend | P2 | MEDIUM |

**NOT generated by the plugin (user provides in global config):**

| Namespace | Why Not Generate It | Where It Belongs |
|-----------|-------------------|-----------------|
| `telemetry/logging` | Global infrastructure concern | Root `extra_config` in base template |
| `telemetry/opentelemetry` | Global infrastructure concern | Root `extra_config` in base template |
| `security/cors` | Global or per-environment setting | Root `extra_config` in base template |
| `security/http` | Global security policy | Root `extra_config` in base template |
| `auth/api-keys` | Enterprise-only feature | Root `extra_config` in base template |
| `server/static-filesystem` | Unrelated to API endpoints | Root `extra_config` in base template |

**Sources:** [KrakenD Rate Limiting](https://www.krakend.io/docs/endpoints/rate-limit/), [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/), [KrakenD Circuit Breaker](https://www.krakend.io/docs/backends/circuit-breaker/)

---

### Input Headers and Query Strings Mapping

KrakenD uses a **zero-trust forwarding policy**: no headers or query strings are forwarded to backends by default. The plugin must explicitly populate `input_headers` and `input_query_strings` from existing sebuf annotations.

| sebuf Annotation | KrakenD Field | Mapping Logic |
|-----------------|---------------|---------------|
| `sebuf.http.service_headers` required headers | `input_headers` | All header names from service-level headers |
| `sebuf.http.method_headers` required headers | `input_headers` | Merged with service headers via existing `annotations.CombineHeaders()` |
| `sebuf.http.query` fields on request message | `input_query_strings` | Query param names from annotated fields |
| (implicit for POST/PUT/PATCH) | `input_headers: ["Content-Type"]` | Methods with request bodies always need Content-Type forwarded |
| (configurable via annotation) | `input_headers: ["Authorization"]` | JWT-protected endpoints should forward Authorization header |

**Confidence:** HIGH -- direct mapping from existing annotation data structures in `internal/annotations/`.

---

### Plugin Parameters

Following the existing pattern (`protoc-gen-openapiv3` accepts `format=json|yaml` via `--openapiv3_opt`), the KrakenD plugin accepts protoc parameters:

| Parameter | Default | Purpose | Example |
|-----------|---------|---------|---------|
| `host` | (empty `[]`) | Default backend host(s), comma-separated | `host=http://localhost:8080` |
| `encoding` | `json` | Default backend response encoding | `encoding=json` |
| `timeout` | (omitted -- KrakenD defaults to 2s) | Default endpoint timeout | `timeout=5s` |

**Usage:**
```bash
protoc --krakend_out=./gateway/partials \
       --krakend_opt=host=http://user-service:8080,timeout=5s \
       proto/services/user_service.proto
```

**Why host is a plugin parameter (not only an annotation):** Backend hosts are environment-specific (localhost in dev, service DNS in k8s). Plugin parameters allow CI/CD to inject the correct host without modifying proto files. The annotation can override the parameter per-service when needed.

**Confidence:** MEDIUM -- parameter set may evolve during implementation.

---

### Config Validation Strategy

**Use `krakend check --lint` externally. Do NOT implement in-process JSON Schema validation.**

| Approach | Verdict | Rationale |
|----------|---------|-----------|
| `krakend check --lint` (CLI) | **USE THIS** | Official validation tool. Uses the published JSON schema. Works against rendered full config (after Flexible Config composition). |
| In-process Go schema validation | **DO NOT USE** | No Go library exists for KrakenD config validation. The [krakend-schema](https://github.com/krakend/krakend-schema) repo contains only JSON Schema files and a Makefile, no Go packages. Building one would be fragile and drift from official schema. |
| Golden file tests | **USE THIS (for generator correctness)** | Test that the generator produces structurally correct JSON matching expected output. Does not validate KrakenD semantic correctness, but catches regressions. |

**Validation workflow for users:**
1. Generate fragments: `protoc --krakend_out=./gateway/partials ...`
2. Render full config: `FC_ENABLE=1 FC_PARTIALS=./gateway/partials FC_OUT=krakend.json krakend check -c krakend.tmpl`
3. The `check --lint` validates the rendered config against the JSON schema

**Confidence:** HIGH -- this is how KrakenD's own documentation recommends validation.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Output format | JSON | YAML | KrakenD recommends JSON; only format with `--lint` support; all official docs use JSON |
| JSON generation | `encoding/json` structs | `text/template` | Templates are error-prone for JSON (commas, escaping); structs are type-safe and golden-testable |
| Output scope | Per-service endpoint array | Full `krakend.json` | Global config is deployment-specific; fragments compose via Flexible Config |
| Output scope | Per-service endpoint array | Per-method individual files | Too granular; per-service matches existing `protoc-gen-openapiv3` pattern |
| Annotation package | `sebuf.krakend` (new) | Extend `sebuf.http` | KrakenD concepts are gateway-specific, not HTTP semantics; clean separation |
| Backend host config | Plugin parameter + annotation override | Hardcoded in proto | Hosts vary per environment; must be injectable without changing proto files |
| Config validation | `krakend check --lint` (external) | In-process Go validation | No Go library exists; CLI check is the official approach |
| KrakenD Go library dependency | None (generate raw JSON) | Import `krakend-ce` or `krakend-config` | Massive dependency tree; not designed for external use; raw JSON is simpler and decoupled |

---

## What NOT to Use and Why

### Do NOT import any KrakenD Go packages
KrakenD CE (`github.com/krakendio/krakend-ce`) is a large application with many transitive dependencies. Importing it for config struct definitions would bloat sebuf's dependency tree. Instead, define minimal Go structs mirroring KrakenD's JSON schema. The struct definitions are small and stable (KrakenD maintains backward compatibility within v2.x).

### Do NOT generate YAML output
KrakenD's `krakend check --lint` only validates JSON. YAML config support exists at runtime but lacks validation tooling. JSON is the documented, recommended, validated format.

### Do NOT generate a complete `krakend.json`
A complete config includes global settings (CORS, telemetry, TLS, server settings) that are deployment-specific and have nothing to do with proto service definitions. The plugin should generate only what it knows from proto: endpoint definitions with their backends and per-endpoint extra_config.

### Do NOT use Go text/template for JSON generation
Template-based JSON generation is the most common source of bugs in config generators: missing commas between array items, incorrect string escaping, broken nesting. Go structs + `json.Marshal` produce valid JSON by construction.

### Do NOT implement custom JSON Schema validation
The [krakend-schema](https://github.com/krakend/krakend-schema) repository contains only schema JSON files, not Go code. Building a Go validator from these schemas would be fragile, version-coupling, and redundant with `krakend check --lint`.

---

## Project Structure (New Files)

Following existing sebuf conventions:

```
cmd/protoc-gen-krakend/
    main.go                          # Plugin entry point (pattern: cmd/protoc-gen-openapiv3/main.go)

internal/krakendgen/
    generator.go                     # Core generator: iterates services/methods, builds Endpoint structs
    types.go                         # KrakenD config types: Endpoint, Backend, RateLimit, etc.
    extra_config.go                  # Builders for extra_config namespaces (rate limit, JWT, circuit breaker)
    generator_test.go                # Unit tests
    golden_test.go                   # Golden file tests
    testdata/
        proto/                       # Test .proto files (symlink or copy from httpgen/testdata/proto)
        golden/                      # Expected output .krakend.json files

proto/sebuf/krakend/
    annotations.proto                # KrakenD-specific annotations (sebuf.krakend package)
```

**Confidence:** HIGH -- follows established patterns from `cmd/protoc-gen-openapiv3` and `internal/openapiv3/`.

---

## Installation

```bash
# No new Go dependencies. Build alongside existing plugins:
make build
# Outputs: bin/protoc-gen-krakend (plus existing 5 plugins)

# Usage with protoc:
protoc --krakend_out=./gateway/partials \
       --krakend_opt=host=http://backend:8080 \
       --proto_path=proto \
       proto/services/user_service.proto

# Usage with buf (buf.gen.yaml):
# plugins:
#   - name: krakend
#     out: gateway/partials
#     opt: host=http://backend:8080
```

---

## KrakenD Extra Config Reference (Quick Reference for Implementation)

### Rate Limiting (`qos/ratelimit/router`)

```json
{
  "qos/ratelimit/router": {
    "max_rate": 100,
    "client_max_rate": 10,
    "every": "1s",
    "strategy": "ip",
    "key": "",
    "capacity": 100,
    "client_capacity": 10
  }
}
```

Fields: `max_rate` (number, all-users rate), `client_max_rate` (number, per-client), `every` (string duration: ns/us/ms/s/m/h, default "1s"), `strategy` ("ip"/"header"/"param"), `key` (header/param name when strategy is header/param), `capacity` (bucket size, defaults to max_rate), `client_capacity` (per-client bucket).

**Source:** [KrakenD Rate Limiting](https://www.krakend.io/docs/endpoints/rate-limit/)

### JWT Validation (`auth/validator`)

```json
{
  "auth/validator": {
    "alg": "RS256",
    "jwk_url": "https://auth.example.com/.well-known/jwks.json",
    "audience": ["https://api.example.com"],
    "issuer": "https://auth.example.com",
    "roles_key": "roles",
    "roles": ["admin", "user"],
    "scopes_key": "scope",
    "scopes": ["read:users"],
    "scopes_matcher": "any",
    "propagate_claims": [["sub", "X-User-ID"]],
    "cache": true,
    "cache_duration": 900
  }
}
```

Algorithms: EdDSA, HS256/384/512, RS256/384/512, ES256/384/512, PS256/384/512. Minimum required: `alg` + (`jwk_url` OR `jwk_local_path`).

**Source:** [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/)

### Circuit Breaker (`qos/circuit-breaker/http`)

```json
{
  "qos/circuit-breaker/http": {
    "interval": 60,
    "timeout": 10,
    "max_errors": 3,
    "name": "cb-endpoint-name",
    "log_status_change": true
  }
}
```

Fields: `interval` (seconds, error counting window), `timeout` (seconds, open-state wait), `max_errors` (consecutive failures before opening), `name` (identifier), `log_status_change` (boolean).

**Source:** [KrakenD Circuit Breaker](https://www.krakend.io/docs/backends/circuit-breaker/)

---

## Sources

- [KrakenD Configuration Guide](https://www.krakend.io/docs/configuration/) -- Config structure, version field, schema reference
- [KrakenD Configuration Structure](https://www.krakend.io/docs/configuration/structure/) -- Top-level fields, extra_config namespace patterns
- [KrakenD Endpoint Configuration](https://www.krakend.io/docs/endpoints/) -- Endpoint object fields, input_headers, input_query_strings, output_encoding, timeout
- [KrakenD Backend Configuration](https://www.krakend.io/docs/backends/) -- Backend object fields, url_pattern, host, encoding, allow/deny, group, target
- [KrakenD Flexible Config](https://www.krakend.io/docs/configuration/flexible-config/) -- FC_ENABLE, FC_PARTIALS, FC_TEMPLATES, FC_SETTINGS, FC_OUT
- [KrakenD Templates](https://www.krakend.io/docs/configuration/templates/) -- include, template, marshal, range functions
- [KrakenD Supported Formats](https://www.krakend.io/docs/configuration/supported-formats/) -- JSON recommended, only format with --lint
- [KrakenD Schema Repo](https://github.com/krakend/krakend-schema) -- v2.1-v2.13 schemas, JSON only, no Go packages
- [KrakenD CE Releases](https://github.com/krakend/krakend-ce/releases) -- v2.13.1 latest (2026-02-18)
- [KrakenD Rate Limiting](https://www.krakend.io/docs/endpoints/rate-limit/) -- qos/ratelimit/router namespace, all fields
- [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/) -- auth/validator namespace, all fields, algorithms
- [KrakenD Circuit Breaker](https://www.krakend.io/docs/backends/circuit-breaker/) -- qos/circuit-breaker/http namespace
- [KrakenD Parameter Forwarding](https://www.krakend.io/docs/endpoints/parameter-forwarding/) -- Zero-trust header/query forwarding policy
- [p3ym4n/krakend-generator](https://github.com/p3ym4n/krakend-generator) -- Prior art: Go structs + json.Marshal for KrakenD config generation
