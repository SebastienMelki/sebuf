# Architecture Patterns

**Domain:** protoc-gen-krakend -- KrakenD API gateway config generation from protobuf
**Researched:** 2026-02-25

## Recommended Architecture

protoc-gen-krakend follows the same structural pattern as the existing sebuf generators, with one key difference: it produces KrakenD JSON endpoint fragments (not Go or TypeScript code). This makes it most analogous to protoc-gen-openapiv3, which also produces structured data output (YAML/JSON) rather than compilable code.

### High-Level Component Diagram

```
proto files (*.proto)
    |
    |  sebuf.http annotations (path, method, headers, query)    -- EXISTING, reused
    |  sebuf.krakend annotations (rate_limit, auth, backend)    -- NEW
    |
    v
protoc (with --krakend_out=...)
    |
    v
cmd/protoc-gen-krakend/main.go            -- NEW entry point
    |
    |  reads protogen.Plugin (services, methods, messages)
    |  delegates to generator
    |
    v
internal/krakendgen/generator.go          -- NEW generator
    |
    |  reads annotations via internal/annotations/  -- EXISTING shared package
    |  reads krakend-specific annotations via krakend/ Go package -- NEW
    |  builds KrakenD endpoint config structs
    |  marshals to JSON
    |
    v
Per-service JSON output files:
    UserService.krakend.json
    OrderService.krakend.json
    ...
```

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| `proto/sebuf/krakend/` | Define gateway-specific proto annotations (rate limiting, auth, backend config, timeout) | Compiled to Go package by protoc-gen-go |
| `krakend/` (Go package at repo root) | Generated Go code from krakend protos -- provides typed access to extension fields | Used by `internal/krakendgen/` |
| `cmd/protoc-gen-krakend/` | Entry point: reads protoc request, creates plugin, iterates files/services, delegates to generator, writes response | `internal/krakendgen/` |
| `internal/krakendgen/` | Core logic: extracts HTTP + KrakenD annotations, builds KrakenD config structs, serializes to JSON | `internal/annotations/` (HTTP config, headers, path params, query params), `krakend/` (gateway annotations) |
| `internal/annotations/` | Shared annotation parsing (already exists) -- provides HTTP config, headers, path building, query params | Used by all generators including krakendgen |

### Data Flow

```
1. protoc parses .proto files with sebuf.http.* and sebuf.krakend.* annotations
2. protoc invokes protoc-gen-krakend via stdin (CodeGeneratorRequest)
3. cmd/protoc-gen-krakend/main.go:
   a. Reads request from stdin
   b. Parses plugin parameters (e.g., backend_host, default_timeout)
   c. Creates protogen.Plugin
   d. For each file with Generate=true:
      i.  For each service in the file:
          - Extract base path via annotations.GetServiceBasePath()
          - Extract service-level KrakenD config (auth, rate limit defaults)
          - For each method:
            * Extract HTTP config via annotations.GetMethodHTTPConfig()
            * Extract headers via annotations.GetServiceHeaders/GetMethodHeaders
            * Extract query params via annotations.GetQueryParams
            * Extract KrakenD-specific annotations (rate limit, auth, backend config)
            * Build KrakenD endpoint object
          - Collect all endpoints into per-service fragment
          - Marshal to indented JSON
      ii. Write per-service JSON file via plugin.NewGeneratedFile()
4. cmd/protoc-gen-krakend/main.go writes CodeGeneratorResponse to stdout
```

### KrakenD Output Structure

Each service produces a JSON file containing an array of KrakenD endpoint objects. The output is designed to be composable via KrakenD Flexible Config templates.

**Example output (`UserService.krakend.json`):**

```json
[
  {
    "endpoint": "/api/v1/users",
    "method": "POST",
    "backend": [
      {
        "url_pattern": "/api/v1/users",
        "host": ["http://user-service:8080"],
        "method": "POST"
      }
    ],
    "input_headers": ["Content-Type", "X-API-Key", "X-Request-ID"],
    "timeout": "3s",
    "extra_config": {
      "qos/ratelimit/router": {
        "max_rate": 100,
        "client_max_rate": 10,
        "strategy": "header",
        "key": "X-API-Key"
      },
      "auth/validator": {
        "alg": "RS256",
        "jwk_url": "https://auth.example.com/.well-known/jwks.json",
        "audience": ["api.example.com"],
        "roles_key": "roles",
        "roles": ["user", "admin"],
        "cache": true
      }
    }
  },
  {
    "endpoint": "/api/v1/users/{id}",
    "method": "GET",
    "backend": [
      {
        "url_pattern": "/api/v1/users/{id}",
        "host": ["http://user-service:8080"],
        "method": "GET",
        "extra_config": {
          "qos/circuit-breaker": {
            "interval": 60,
            "timeout": 10,
            "max_errors": 3,
            "log_status_change": true
          }
        }
      }
    ],
    "input_headers": ["Authorization", "X-API-Key"],
    "timeout": "2s"
  }
]
```

**Decision: Output per-service endpoint arrays (JSON array, not wrapped object).** This format is the simplest to compose in KrakenD Flexible Config templates. The user's `krakend.tmpl` includes fragments like:

```
{
  "version": 3,
  "endpoints": [
    {{ include "UserService.krakend.json" }}
    ,{{ include "OrderService.krakend.json" }}
  ],
  "extra_config": {
    "security/cors": { ... }
  }
}
```

### KrakenD Path Variable Compatibility

KrakenD and sebuf both use `{variable}` syntax for path parameters, so path strings pass through directly. Example: `/api/v1/users/{id}` from `sebuf.http.config.path` maps directly to the KrakenD endpoint path and backend `url_pattern`.

## Patterns to Follow

### Pattern 1: Mirror the OpenAPI Generator Structure

The openapiv3 generator is the closest analog -- it produces structured data output, one file per service, using `plugin.NewGeneratedFile()` for output. protoc-gen-krakend should follow the same pattern.

**What:** Entry point reads request, parses parameters, iterates services, delegates to generator, writes per-service output files.

**When:** Always -- this is the established sebuf convention.

**Example entry point (`cmd/protoc-gen-krakend/main.go`):**
```go
package main

import (
    "fmt"
    "io"
    "os"
    "strings"

    "google.golang.org/protobuf/compiler/protogen"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/pluginpb"

    "github.com/SebastienMelki/sebuf/internal/krakendgen"
)

func main() {
    input, err := io.ReadAll(os.Stdin)
    if err != nil {
        panic(err)
    }
    var req pluginpb.CodeGeneratorRequest
    if err := proto.Unmarshal(input, &req); err != nil {
        panic(err)
    }

    params := parseParameters(req.GetParameter())
    opts := krakendgen.Options{
        BackendHost: params["backend_host"],
        DefaultTimeout: params["timeout"],
    }

    plugin, err := protogen.Options{}.New(&req)
    if err != nil {
        panic(err)
    }

    for _, file := range plugin.Files {
        if !file.Generate {
            continue
        }
        for _, service := range file.Services {
            gen := krakendgen.NewGenerator(service, opts)
            output, err := gen.Generate()
            if err != nil {
                panic(err)
            }
            filename := fmt.Sprintf("%s.krakend.json", service.Desc.Name())
            gf := plugin.NewGeneratedFile(filename, "")
            gf.Write(output)
        }
    }

    resp := plugin.Response()
    resp.SupportedFeatures = proto.Uint64(
        uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL),
    )
    out, _ := proto.Marshal(resp)
    os.Stdout.Write(out)
}
```

### Pattern 2: Reuse Shared Annotations Package

The `internal/annotations/` package already provides everything needed for HTTP routing info. Do not duplicate this logic.

**What:** Use `annotations.GetMethodHTTPConfig()`, `annotations.GetServiceBasePath()`, `annotations.GetServiceHeaders()`, `annotations.GetMethodHeaders()`, `annotations.CombineHeaders()`, `annotations.GetQueryParams()`, `annotations.BuildHTTPPath()`, and `annotations.ExtractPathParams()`.

**When:** For extracting HTTP routing information from proto service definitions.

**Example:**
```go
func (g *Generator) buildEndpoint(service *protogen.Service, method *protogen.Method) *Endpoint {
    // Reuse existing annotations -- same functions used by openapiv3, httpgen, etc.
    basePath := annotations.GetServiceBasePath(service)
    httpConfig := annotations.GetMethodHTTPConfig(method)

    var path, httpMethod string
    if httpConfig != nil {
        path = annotations.BuildHTTPPath(basePath, httpConfig.Path)
        httpMethod = httpConfig.Method
    } else {
        path = fmt.Sprintf("/%s/%s", service.Desc.Name(), method.Desc.Name())
        httpMethod = "POST"
    }

    // Headers from sebuf.http annotations -> KrakenD input_headers
    serviceHeaders := annotations.GetServiceHeaders(service)
    methodHeaders := annotations.GetMethodHeaders(method)
    combined := annotations.CombineHeaders(serviceHeaders, methodHeaders)
    inputHeaders := extractHeaderNames(combined)

    // Query params from sebuf.http annotations -> KrakenD input_query_strings
    queryParams := annotations.GetQueryParams(method.Input)
    inputQueryStrings := extractQueryParamNames(queryParams)

    return &Endpoint{
        Endpoint:          path,
        Method:            httpMethod,
        InputHeaders:      inputHeaders,
        InputQueryStrings: inputQueryStrings,
        // ... backend, extra_config from krakend annotations
    }
}
```

### Pattern 3: Typed Config Structs with JSON Tags

Build KrakenD config as Go structs with `json:"..."` tags, then marshal. This ensures valid JSON output, enables unit testing of the struct-building logic, and follows Go conventions.

**What:** Define Go structs matching KrakenD's JSON schema, build them from proto annotations, marshal to JSON.

**When:** Always -- do not use string concatenation or template-based JSON generation.

**Example (`internal/krakendgen/types.go`):**
```go
package krakendgen

// Endpoint represents a single KrakenD endpoint.
type Endpoint struct {
    Endpoint          string         `json:"endpoint"`
    Method            string         `json:"method"`
    Backend           []*Backend     `json:"backend"`
    InputHeaders      []string       `json:"input_headers,omitempty"`
    InputQueryStrings []string       `json:"input_query_strings,omitempty"`
    Timeout           string         `json:"timeout,omitempty"`
    ExtraConfig       map[string]any `json:"extra_config,omitempty"`
}

// Backend represents a KrakenD backend.
type Backend struct {
    URLPattern  string         `json:"url_pattern"`
    Host        []string       `json:"host"`
    Method      string         `json:"method,omitempty"`
    Encoding    string         `json:"encoding,omitempty"`
    ExtraConfig map[string]any `json:"extra_config,omitempty"`
}

// RateLimitConfig represents qos/ratelimit/router config.
type RateLimitConfig struct {
    MaxRate        int    `json:"max_rate,omitempty"`
    Capacity       int    `json:"capacity,omitempty"`
    Every          string `json:"every,omitempty"`
    ClientMaxRate  int    `json:"client_max_rate,omitempty"`
    ClientCapacity int    `json:"client_capacity,omitempty"`
    Strategy       string `json:"strategy,omitempty"`
    Key            string `json:"key,omitempty"`
}

// AuthValidatorConfig represents auth/validator config.
type AuthValidatorConfig struct {
    Alg       string   `json:"alg"`
    JWKURL    string   `json:"jwk_url,omitempty"`
    Audience  []string `json:"audience,omitempty"`
    Issuer    string   `json:"issuer,omitempty"`
    RolesKey  string   `json:"roles_key,omitempty"`
    Roles     []string `json:"roles,omitempty"`
    ScopesKey string   `json:"scopes_key,omitempty"`
    Scopes    []string `json:"scopes,omitempty"`
    Cache     bool     `json:"cache,omitempty"`
}

// CircuitBreakerConfig represents qos/circuit-breaker config.
type CircuitBreakerConfig struct {
    Interval        int    `json:"interval"`
    Timeout         int    `json:"timeout"`
    MaxErrors       int    `json:"max_errors"`
    Name            string `json:"name,omitempty"`
    LogStatusChange bool   `json:"log_status_change,omitempty"`
}
```

### Pattern 4: Plugin Parameters for Deployment-Specific Values

Backend host addresses, default timeouts, and environment-specific settings should come from protoc plugin parameters, not proto annotations. Proto annotations define the API shape; plugin parameters configure deployment.

**What:** Use `--krakend_opt=backend_host=http://user-service:8080,timeout=3s` for values that vary per deployment.

**When:** For any value that is deployment-specific rather than API-shape-specific.

**Example invocation:**
```bash
protoc \
  --krakend_out=./gateway/endpoints \
  --krakend_opt=backend_host=http://user-service:8080,timeout=3s \
  --proto_path=proto \
  proto/services/user_service.proto
```

### Pattern 5: Golden File Tests for JSON Output

Follow the exact same golden file test pattern as openapiv3. Build the plugin binary, run protoc against test protos, compare output byte-for-byte with golden files. Support `UPDATE_GOLDEN=1` for updating.

**What:** Exhaustive golden file tests in `internal/krakendgen/golden_test.go` with test protos in `internal/krakendgen/testdata/proto/` and golden outputs in `internal/krakendgen/testdata/golden/`.

**When:** For every generated output variation.

**Test directory structure:**
```
internal/krakendgen/
    generator.go
    types.go
    annotations.go          -- KrakenD-specific annotation extraction
    golden_test.go
    generator_test.go       -- unit tests for individual functions
    testdata/
        proto/
            simple_service.proto       -- basic routing only
            rate_limited_service.proto  -- rate limiting annotations
            auth_service.proto         -- JWT auth annotations
            backend_config.proto       -- circuit breaker, timeouts
            full_featured.proto        -- all features combined
            no_annotations.proto       -- fallback behavior
            headers_and_query.proto    -- input_headers, input_query_strings
        golden/
            SimpleService.krakend.json
            RateLimitedService.krakend.json
            AuthService.krakend.json
            ...
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Monolithic Config Output

**What:** Generating a single `krakend.json` with all services merged.
**Why bad:** Breaks the per-service output convention, makes Flexible Config composition harder, and does not scale when different services are compiled separately.
**Instead:** Output one `ServiceName.krakend.json` per service, each containing that service's endpoint array.

### Anti-Pattern 2: Duplicating HTTP Annotation Parsing

**What:** Re-implementing HTTP config extraction in krakendgen instead of using `internal/annotations/`.
**Why bad:** Creates divergence risk -- if annotations change, krakendgen could generate endpoints with paths/methods that don't match the Go HTTP handlers or OpenAPI specs.
**Instead:** Always use `internal/annotations/` for HTTP routing info. Only add new code in krakendgen for KrakenD-specific annotations.

### Anti-Pattern 3: Embedding Deployment Config in Proto Annotations

**What:** Putting backend host URLs, environment-specific timeouts, or infrastructure-specific values in proto annotations.
**Why bad:** Proto files define the API contract, not deployment topology. Backend hosts change per environment; proto files should not.
**Instead:** Use plugin parameters (`--krakend_opt=backend_host=...`) for deployment-specific values. Proto annotations should only define API-shape concerns (rate limits, auth requirements, circuit breaker thresholds).

### Anti-Pattern 4: String Concatenation for JSON

**What:** Building JSON output via `fmt.Sprintf` or string concatenation.
**Why bad:** Fragile, hard to test, prone to escaping bugs, no type safety.
**Instead:** Use typed Go structs with `json:"..."` tags and `json.MarshalIndent()`.

### Anti-Pattern 5: Modeling Every KrakenD Config Field in Proto

**What:** Trying to model every KrakenD extra_config namespace and field in proto annotations.
**Why bad:** KrakenD config surface is enormous (100+ namespaces). Proto annotations should cover the 80% use case.
**Instead:** Model the top features (rate limiting, JWT auth, circuit breaker, timeouts). Provide a `raw_extra_config` string field on both endpoint and backend config that gets parsed as JSON and merged into the output. This is the escape hatch for any KrakenD feature not directly modeled.

## Component Build Order

The build order matters because of dependency chains.

### Phase 1: Proto Annotations (no code dependencies)
1. **`proto/sebuf/krakend/krakend.proto`** -- Define the proto annotation messages and extensions
2. **`krakend/*.pb.go`** -- Run `protoc --go_out` to generate Go code from the proto
3. **Update `Makefile` proto target** to include the new proto files
4. **Update `proto/buf.yaml`** if needed for the new package

This must come first because everything downstream imports the generated Go types for extension field access.

### Phase 2: Generator Foundation (depends on Phase 1)
5. **`internal/krakendgen/types.go`** -- Go structs matching KrakenD JSON schema
6. **`internal/krakendgen/annotations.go`** -- KrakenD-specific annotation extraction functions (reads `krakend/` Go package extension types)
7. **`internal/krakendgen/generator.go`** -- Core logic: iterate methods, extract annotations, build config structs, marshal JSON
8. **`cmd/protoc-gen-krakend/main.go`** -- Entry point wiring

### Phase 3: Tests (depends on Phase 2)
9. **Test protos** in `internal/krakendgen/testdata/proto/` (symlink shared protos from httpgen where applicable)
10. **Golden files** in `internal/krakendgen/testdata/golden/`
11. **`internal/krakendgen/golden_test.go`** -- Golden file tests
12. **`internal/krakendgen/generator_test.go`** -- Unit tests for individual builder functions

### Phase 4: Documentation and Distribution
13. **Update CLAUDE.md** with krakendgen architecture and test instructions
14. **Update `.goreleaser.yaml`** to include protoc-gen-krakend binary
15. **Update README.md** with KrakenD generator documentation
16. **Publish updated proto to BSR** via `make publish`

## Integration Points with Existing sebuf Code

### 1. `internal/annotations/` (read-only dependency)

The krakendgen package imports and calls existing annotation functions. No changes needed to the shared annotations package.

| Function | What krakendgen uses it for |
|----------|-----------------------------|
| `GetMethodHTTPConfig(method)` | Endpoint path, HTTP method, path params |
| `GetServiceBasePath(service)` | Service-level path prefix |
| `BuildHTTPPath(base, method)` | Full endpoint path construction |
| `GetServiceHeaders(service)` | Service-level headers -> `input_headers` |
| `GetMethodHeaders(method)` | Method-level headers -> `input_headers` |
| `CombineHeaders(svc, method)` | Merged headers with method override |
| `GetQueryParams(message)` | Query params -> `input_query_strings` |
| `ExtractPathParams(path)` | Path variable extraction (for validation) |
| `HTTPMethodToString(method)` | HTTP method enum to string |

### 2. `proto/sebuf/http/` (read-only dependency)

Existing HTTP annotations are consumed but not modified. Test protos in krakendgen will import `sebuf/http/annotations.proto` alongside the new `sebuf/krakend/krakend.proto`.

### 3. `proto/sebuf/krakend/` (new package)

New proto package with `go_package = "github.com/SebastienMelki/sebuf/krakend;krakend"`. This generates to `krakend/*.pb.go` at the repo root, matching the `http/*.pb.go` pattern.

### 4. Makefile Auto-Discovery

The Makefile auto-discovers `cmd/*` directories via `$(wildcard $(CMD_DIR)/*)`. Adding `cmd/protoc-gen-krakend/` automatically includes it in `make build` with zero Makefile changes. The proto target needs updating to include the new proto files.

### 5. `proto/buf.yaml` Registry

The buf.yaml module (`buf.build/sebmelki/sebuf`) needs to include the new `sebuf/krakend/` package so downstream users can import the KrakenD annotations via `deps: [buf.build/sebmelki/sebuf]`. The `make publish` target handles pushing to BSR.

### 6. `.goreleaser.yaml`

The release configuration needs a new binary entry for `protoc-gen-krakend` so it is distributed via Homebrew, Docker, and package managers alongside the existing plugins.

## Proto Annotation Design

The `proto/sebuf/krakend/krakend.proto` defines gateway-specific annotations at service and method levels. It lives in a separate proto package from `sebuf.http` because gateway configuration is a fundamentally different concern than HTTP API shape.

```protobuf
syntax = "proto3";

package sebuf.krakend;

import "google/protobuf/descriptor.proto";

option go_package = "github.com/SebastienMelki/sebuf/krakend;krakend";

// EndpointConfig defines KrakenD endpoint-level configuration.
// Applied to individual RPC methods via (sebuf.krakend.endpoint_config).
message EndpointConfig {
  // Endpoint timeout (e.g., "3s", "1500ms"). Overrides service default.
  string timeout = 1;

  // Rate limiting configuration for this endpoint.
  RateLimitConfig rate_limit = 2;

  // Backend configuration for this endpoint.
  BackendConfig backend = 3;

  // Raw JSON to merge into the endpoint's extra_config.
  // Escape hatch for any KrakenD feature not directly modeled.
  string raw_extra_config = 10;
}

// ServiceGatewayConfig defines KrakenD configuration defaults for all RPCs in a service.
// Applied to a service via (sebuf.krakend.gateway_config).
message ServiceGatewayConfig {
  // Default timeout for all endpoints in this service.
  string default_timeout = 1;

  // Auth/JWT configuration applied to all endpoints.
  AuthConfig auth = 2;

  // Default rate limiting for all endpoints (method-level overrides).
  RateLimitConfig default_rate_limit = 3;

  // Default backend configuration for all endpoints.
  BackendConfig default_backend = 4;

  // Raw JSON to merge into every endpoint's extra_config in this service.
  string raw_extra_config = 10;
}

// RateLimitConfig maps to KrakenD's qos/ratelimit/router namespace.
message RateLimitConfig {
  int32 max_rate = 1;          // Max requests all users in time frame
  int32 capacity = 2;          // Max tokens in bucket (defaults to max_rate)
  string every = 3;            // Time period: "1s", "10m", "1h"
  int32 client_max_rate = 4;   // Max requests per individual client
  int32 client_capacity = 5;   // Max tokens per client bucket
  string strategy = 6;         // "ip", "header", or "param"
  string key = 7;              // Header or param name for client identification
}

// AuthConfig maps to KrakenD's auth/validator namespace.
message AuthConfig {
  string alg = 1;              // Hashing algorithm: RS256, HS256, ES256, etc.
  string jwk_url = 2;          // Remote JWK endpoint URL
  repeated string audience = 3;
  string issuer = 4;
  string roles_key = 5;
  repeated string roles = 6;
  string scopes_key = 7;
  repeated string scopes = 8;
  bool cache = 9;              // Enable JWK key caching
}

// BackendConfig defines backend-level KrakenD configuration.
message BackendConfig {
  // Circuit breaker configuration.
  CircuitBreakerConfig circuit_breaker = 1;
  // Backend encoding (default: "json").
  string encoding = 2;
  // Raw JSON to merge into the backend's extra_config.
  string raw_extra_config = 10;
}

// CircuitBreakerConfig maps to KrakenD's qos/circuit-breaker namespace.
message CircuitBreakerConfig {
  int32 interval = 1;          // Error counting window in seconds
  int32 timeout = 2;           // Seconds before retesting backend
  int32 max_errors = 3;        // Consecutive errors to trigger open state
  bool log_status_change = 4;  // Log circuit breaker state changes
}

// Extension for method options (extension number 51001, outside sebuf.http range)
extend google.protobuf.MethodOptions {
  optional EndpointConfig endpoint_config = 51001;
}

// Extension for service options (extension number 51002, outside sebuf.http range)
extend google.protobuf.ServiceOptions {
  optional ServiceGatewayConfig gateway_config = 51002;
}
```

**Extension number range:** 51000+ to avoid collision with the existing sebuf.http range (50003-50020). This provides clean separation between the two proto packages.

**Key design choices:**
- Service-level `gateway_config` provides defaults; method-level `endpoint_config` overrides.
- `raw_extra_config` fields serve as escape hatches for any KrakenD feature not directly modeled (parsed as JSON and merged into extra_config).
- Auth lives at service level only -- JWT config is typically uniform across all endpoints in a service.
- Rate limiting can be set at both levels -- service default with per-endpoint overrides.

## Key Design Decision: What Stays in Proto vs Plugin Parameters

| Concern | In Proto Annotations | In Plugin Parameters | Rationale |
|---------|---------------------|---------------------|-----------|
| Endpoint path/method | Yes (sebuf.http) | No | API contract, same across all environments |
| Rate limit thresholds | Yes (sebuf.krakend) | No | Part of API design intent |
| Auth requirements (algorithm, audience, roles) | Yes (sebuf.krakend) | No | Part of API security contract |
| JWK URL | Yes (sebuf.krakend) | Override via raw_extra_config | Usually stable, but may vary per env |
| Circuit breaker thresholds | Yes (sebuf.krakend) | No | Part of resilience design |
| Timeout | Yes (sebuf.krakend) | Also as default via plugin param | API has opinion, deployment can override |
| Backend host | No | Yes (`backend_host`) | Deployment-specific, changes per env |
| Backend encoding | Optionally (sebuf.krakend) | Default in plugin | Usually same across deployment |

## Scalability Considerations

| Concern | At 5 services | At 50 services | At 200+ services |
|---------|--------------|----------------|------------------|
| Output file count | 5 JSON files, trivial | 50 JSON files, FC templates get longer | Use `range` loops in FC templates over a directory |
| Compilation time | Negligible | Negligible (protoc is per-file) | Consider buf workspace with parallel compilation |
| Config correctness | Manual review feasible | Need `krakend check` in CI | Need automated consistency checks |
| FC template complexity | Simple includes | Template loops with settings | Need structured FC with per-service settings files |
| Annotation consistency | Easy to verify manually | Need CI checks for missing annotations | Lint rules for annotation completeness |

## Sources

- [KrakenD Configuration Structure](https://www.krakend.io/docs/configuration/structure/) -- root config format, version 3
- [KrakenD Endpoint Configuration](https://www.krakend.io/docs/endpoints/) -- endpoint fields, input_headers, input_query_strings
- [KrakenD Backend Configuration](https://www.krakend.io/docs/backends/) -- backend fields, url_pattern, host, encoding
- [KrakenD Flexible Config](https://www.krakend.io/docs/configuration/flexible-config/) -- FC_ENABLE, templates, partials, settings
- [KrakenD Templates](https://www.krakend.io/docs/configuration/templates/) -- Go template syntax, include/template directives
- [KrakenD Rate Limiting](https://www.krakend.io/docs/endpoints/rate-limit/) -- qos/ratelimit/router namespace
- [KrakenD JWT Validation](https://www.krakend.io/docs/authorization/jwt-validation/) -- auth/validator namespace
- [KrakenD Circuit Breaker](https://www.krakend.io/docs/backends/circuit-breaker/) -- qos/circuit-breaker namespace
- [KrakenD Timeouts](https://www.krakend.io/docs/throttling/timeouts/) -- endpoint and service timeout configuration
- [KrakenD CORS](https://www.krakend.io/docs/service-settings/cors/) -- security/cors namespace (service-level, not per-endpoint)
- Existing sebuf codebase: `internal/annotations/`, `internal/openapiv3/`, `cmd/protoc-gen-openapiv3/` -- patterns and conventions (HIGH confidence, read directly from source)
