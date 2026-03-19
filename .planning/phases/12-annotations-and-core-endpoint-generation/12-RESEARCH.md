# Phase 12: Annotations and Core Endpoint Generation - Research

**Researched:** 2026-02-25
**Domain:** Protoc plugin for KrakenD API gateway endpoint generation from protobuf service definitions
**Confidence:** HIGH

## Summary

Phase 12 creates the 6th sebuf protoc plugin (`protoc-gen-krakend`) that generates KrakenD API gateway endpoint fragments from proto service definitions. The plugin reads existing `sebuf.http` annotations for routing, headers, and query parameters, plus new `sebuf.krakend` annotations for gateway-specific config (host, timeout). It outputs one JSON file per service containing a KrakenD endpoint array.

The architecture mirrors `protoc-gen-openapiv3` almost exactly: entry point in `cmd/`, core logic in `internal/krakendgen/`, new proto annotations in `proto/sebuf/krakend/`, golden file tests with real protoc execution. No new Go dependencies are required -- `encoding/json` with typed Go structs handles JSON output. The key value proposition is auto-deriving KrakenD's `input_headers` and `input_query_strings` from existing sebuf header and query annotations.

The most important implementation details are: (1) KrakenD requires one endpoint object per RPC (not per path), (2) KrakenD's zero-trust model means the generator MUST populate `input_headers` and `input_query_strings` from annotations or forwarding breaks silently, (3) static routes and parameterized routes cannot coexist at the same path level, and (4) the output JSON structure must work cleanly with KrakenD Flexible Config `{{ include }}` directives.

**Primary recommendation:** Follow the protoc-gen-openapiv3 pattern exactly -- typed Go structs with `json` tags, `json.MarshalIndent` for output, reuse `internal/annotations/` for all HTTP data extraction, add a thin `internal/krakendgen/` package for KrakenD-specific assembly and validation.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Output format and file structure
- One file per service (e.g., `UserService.krakend.json`) -- matches sebuf's existing per-service pattern (OpenAPI generator does this)
- Output JSON shape: Claude's discretion on whether to emit a bare endpoints array or a minimal wrapper -- pick what integrates cleanest with KrakenD Flexible Config `{{ include }}` directives
- Pretty-printed JSON (indented) -- readable, reviewable in PRs
- Standard protoc output via `--krakend_out=<dir>` -- follows existing sebuf plugin conventions

#### Annotation design and defaults
- Auto-include all RPCs: every RPC with `sebuf.http.config` automatically generates a KrakenD endpoint -- no extra KrakenD annotation required for basic routing
- Host configured at service level via `gateway_config` annotation, overridable at method level via `endpoint_config` -- no plugin flag for host, it lives in annotations
- Timeouts: omit if not annotated -- let KrakenD apply its own defaults, don't be opinionated
- Annotations live in a separate `sebuf.krakend` proto package -- gateway config is a different concern than HTTP API shape (confirmed from STATE.md decisions)
- Per-RPC config overrides per-service config (timeouts, host, etc.) -- consistent override semantics

#### KrakenD path mapping
- Pass proto HTTP paths through as-is -- sebuf uses `{param}` syntax which maps directly to KrakenD
- Backend path mirrors endpoint path -- gateway is a passthrough, no remapping
- Service `base_path` from `sebuf.http.service_config` is prepended to endpoint paths -- consistent with how the HTTP server generator works
- Output encoding: always `json` -- content type is a runtime client decision, not a proto definition concern; KrakenD needs JSON encoding to process payloads for its features

#### Error messages and DX
- Fail hard on route conflicts -- generation fails with a clear error, no config produced, forces fix before deploy
- Silent on success -- no output, matches protoc plugin convention; errors/warnings go to stderr only
- Rich error context -- include service name, RPC name, file location, and what to fix (e.g., "user_service.proto: UserService.GetUser and UserService.SearchUsers produce conflicting routes: GET /users/{id} vs GET /users/search")
- Validate KrakenD constraints at generation time -- catch known issues (valid namespace strings, constraint violations) before deployment rather than at KrakenD startup

### Claude's Discretion

- Exact JSON structure (bare array vs minimal wrapper) -- pick what's cleanest for FC integration
- Annotation proto message design (field names, nesting)
- Plugin architecture and code organization within internal/krakendgen/
- Golden test structure and coverage strategy
- How to detect and report static vs parameterized route conflicts

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| ANNO-01 | New proto package `proto/sebuf/krakend/` with gateway-specific annotations (extension numbers 51000+) | Codebase analysis confirms extension number registry (50003-50020 used by sebuf.http), 51000+ is reserved. Pattern established by `proto/sebuf/http/`. Go package generated to `krakend/*.pb.go`. |
| ANNO-02 | Service-level `gateway_config` annotation for service-wide defaults (host, timeout) | Phase 12 scope is host+timeout only (rate limit, auth, circuit breaker deferred to Phase 13). ServiceOptions extension at 51001. |
| ANNO-03 | Method-level `endpoint_config` annotation for per-RPC overrides (timeout) | MethodOptions extension at 51002. Same fields as gateway_config but method-level override semantics. Host override is also useful here. |
| ANNO-04 | Method-level config always overrides service-level config for the same setting | Merge logic: build default from service-level, overlay method-level non-zero values. Mirrors sebuf.http.service_headers/method_headers override pattern in `annotations.CombineHeaders()`. |
| CORE-01 | Plugin reads `sebuf.http.config` annotations to extract HTTP path and method for each RPC | Existing `annotations.GetMethodHTTPConfig()` returns path, method, path params. Existing `annotations.GetServiceBasePath()` returns base path. Both reusable directly. |
| CORE-02 | Plugin generates one JSON file per proto service (`{ServiceName}.krakend.json`) containing array of KrakenD endpoint objects | Mirrors `cmd/protoc-gen-openapiv3/main.go` pattern: iterate file.Services, generate per-service, write via plugin.NewGeneratedFile. |
| CORE-03 | Backend host is configurable via plugin parameter (`--krakend_opt=host=http://backend:8080`) | **NOTE: CONTEXT.md overrides this.** User decided "no plugin flag for host, it lives in annotations." Host comes from `gateway_config` annotation only. CORE-03 is superseded. |
| CORE-04 | Backend host is configurable via service-level annotation (overrides plugin parameter) | Host from `gateway_config` at service level, overridable at method level via `endpoint_config`. No plugin parameter needed per CONTEXT.md. |
| CORE-05 | Per-endpoint timeout is configurable via service-level default and method-level override | KrakenD timeout is a duration string (e.g., "2s", "500ms"). Omit from output if not annotated (user decision: let KrakenD apply its defaults). |
| CORE-06 | Output encoding defaults to JSON for all endpoints | Hardcode `"output_encoding": "json"` on every endpoint. User decision: always JSON. |
| FWD-01 | `input_headers` auto-populated from `sebuf.http.service_headers` and `sebuf.http.method_headers` annotations | Reuse `annotations.GetServiceHeaders()`, `annotations.GetMethodHeaders()`, `annotations.CombineHeaders()`. Extract header names from the merged list. |
| FWD-02 | `input_query_strings` auto-populated from `sebuf.http.query` annotations on request message fields | Reuse `annotations.GetQueryParams(method.Input)`. Extract `ParamName` from each QueryParam. |
| FWD-03 | Auto-derived headers and query strings are never empty arrays (KrakenD zero-trust model) | Omit `input_headers`/`input_query_strings` fields entirely when no annotations exist. KrakenD treats absent as "forward nothing" (same as empty array). Never emit `[]` or `["*"]`. |
| VALD-01 | Generation fails with clear error if two RPCs produce identical (path, method) tuples | Build a map of `(resolved_path, http_method)` -> RPC name during generation. Duplicate entry = error. |
| VALD-02 | Generation fails with clear error if static and parameterized routes conflict at the same path level | After resolving all paths, build a trie of path segments. At each level, if both static segments and a `{param}` segment exist, it is a conflict. |
| TEST-01 | Golden file tests cover all core generation scenarios (endpoint routing, backend mapping, timeouts) | Follow openapiv3 exhaustive_golden_test.go pattern: build plugin binary, run protoc, compare output byte-for-byte with golden files. |
| TEST-03 | Golden file tests cover auto-derived header and query string forwarding | Test protos with service_headers, method_headers, and query annotations. Verify `input_headers` and `input_query_strings` appear correctly in output. |
| TEST-04 | Golden file tests cover generation-time validation errors (duplicate endpoints, path conflicts) | Follow tsservergen TestTSServerGenValidationErrors pattern: run protoc expecting failure, check stderr contains expected error substring. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| google.golang.org/protobuf/compiler/protogen | v1.36.11 (existing) | Plugin framework: reads CodeGeneratorRequest, provides File/Service/Method/Field access | Same framework used by all 5 existing sebuf plugins |
| encoding/json (stdlib) | Go 1.24.7 | JSON marshaling of KrakenD config structs | No external dependency needed; typed Go structs with json tags guarantee valid output |
| internal/annotations (existing) | N/A | Extract HTTP config, headers, query params from sebuf.http annotations | Already used by all generators; provides GetMethodHTTPConfig, GetServiceHeaders, GetQueryParams, CombineHeaders |
| proto/sebuf/krakend/ (new) | N/A | Gateway-specific proto annotations (host, timeout) | New proto package following sebuf pattern; generated Go code in krakend/*.pb.go |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| google.golang.org/protobuf/proto | v1.36.11 (existing) | proto.GetExtension for reading krakend annotations | Reading new sebuf.krakend extensions from ServiceOptions/MethodOptions |
| google.golang.org/protobuf/types/descriptorpb | v1.36.11 (existing) | Type assertions on ServiceOptions/MethodOptions | Same pattern as annotations/http_config.go and annotations/headers.go |
| google.golang.org/protobuf/types/pluginpb | v1.36.11 (existing) | CodeGeneratorResponse with FEATURE_PROTO3_OPTIONAL | Entry point pattern from cmd/protoc-gen-openapiv3/main.go |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| encoding/json | encoding/json/v2 | v2 is not yet in stdlib; json/v1 is sufficient for this use case |
| Typed Go structs | map[string]any | Loses type safety, harder to test, matches KrakenD but not sebuf patterns |
| Per-service files | Monolithic krakend.json | User decision: per-service matches sebuf pattern, composable via FC |

**Installation:**
```bash
# No new dependencies needed. Just:
go mod tidy  # After adding proto/sebuf/krakend/ generated Go code
```

## Architecture Patterns

### Recommended Project Structure
```
cmd/protoc-gen-krakend/
  main.go                    # Entry point: read request, parse params, iterate files/services, write response
proto/sebuf/krakend/
  krakend.proto              # Gateway annotations: GatewayConfig, EndpointConfig
krakend/
  krakend.pb.go              # Generated Go code from krakend.proto
internal/krakendgen/
  types.go                   # KrakenD JSON struct types: Endpoint, Backend
  generator.go               # Core logic: service -> endpoints, annotation merging
  validation.go              # Route conflict detection (duplicate endpoints, static vs param)
  golden_test.go             # Exhaustive golden file tests
  testdata/
    proto/                   # Test proto files (some symlinked from httpgen/testdata/proto)
    golden/                  # Expected JSON output files
```

### Pattern 1: Entry Point (mirrors protoc-gen-openapiv3)
**What:** Minimal main.go that reads protoc request, iterates files/services, calls generator, writes output
**When to use:** Every sebuf protoc plugin follows this pattern
**Example:**
```go
// Source: cmd/protoc-gen-openapiv3/main.go (verified from codebase)
func main() {
    req := readRequest()     // Read CodeGeneratorRequest from stdin
    plugin := createPlugin(req)
    generateFiles(plugin)    // Iterate files, call generator per service
    writeResponse(plugin)    // Write CodeGeneratorResponse to stdout
}

func generateFiles(plugin *protogen.Plugin) {
    for _, file := range plugin.Files {
        if !file.Generate {
            continue
        }
        for _, service := range file.Services {
            output := generateServiceEndpoints(service)
            filename := fmt.Sprintf("%s.krakend.json", service.Desc.Name())
            gf := plugin.NewGeneratedFile(filename, "")
            gf.Write(output)
        }
    }
}
```

### Pattern 2: Typed KrakenD Structs with JSON Tags
**What:** Go structs that mirror KrakenD's endpoint/backend JSON schema
**When to use:** Core output construction
**Example:**
```go
// Source: KrakenD endpoint configuration docs (https://www.krakend.io/docs/endpoints/)
type Endpoint struct {
    Endpoint         string   `json:"endpoint"`
    Method           string   `json:"method"`
    OutputEncoding   string   `json:"output_encoding"`
    Timeout          string   `json:"timeout,omitempty"`
    InputHeaders     []string `json:"input_headers,omitempty"`
    InputQueryStrings []string `json:"input_query_strings,omitempty"`
    Backend          []Backend `json:"backend"`
}

type Backend struct {
    URLPattern string   `json:"url_pattern"`
    Host       []string `json:"host"`
    Method     string   `json:"method"`
    Encoding   string   `json:"encoding"`
}
```

### Pattern 3: Annotation Merge (service defaults + method overrides)
**What:** Build endpoint config by starting with service-level defaults and overlaying method-level values
**When to use:** For every field that supports service/method annotation levels
**Example:**
```go
// Source: Mirrors annotations.CombineHeaders() pattern from annotations/headers.go
func resolveTimeout(serviceConfig *krakend.GatewayConfig, methodConfig *krakend.EndpointConfig) string {
    timeout := ""
    if serviceConfig != nil && serviceConfig.Timeout != "" {
        timeout = serviceConfig.Timeout
    }
    if methodConfig != nil && methodConfig.Timeout != "" {
        timeout = methodConfig.Timeout  // Method overrides service
    }
    return timeout
}
```

### Pattern 4: Golden Test with protoc Execution
**What:** Build plugin binary, run protoc with test protos, compare JSON output byte-for-byte against golden files
**When to use:** Primary testing strategy for all sebuf generators
**Example:**
```go
// Source: internal/openapiv3/exhaustive_golden_test.go (verified from codebase)
func TestKrakenDGoldenFiles(t *testing.T) {
    pluginPath := buildPlugin(t, "../../cmd/protoc-gen-krakend")
    defer os.Remove(pluginPath)

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            tempDir := t.TempDir()
            cmd := exec.Command("protoc",
                "--plugin=protoc-gen-krakend="+pluginPath,
                "--krakend_out="+tempDir,
                "--proto_path=testdata/proto",
                "--proto_path=../../proto",
                tc.protoFile,
            )
            // ... run, read output, compare with golden file
        })
    }
}
```

### Pattern 5: Validation Error Tests (expect protoc failure)
**What:** Test protos that should cause generation-time errors; verify protoc exits non-zero with expected stderr message
**When to use:** Route conflict detection, duplicate endpoint detection
**Example:**
```go
// Source: internal/tsservergen/golden_test.go TestTSServerGenValidationErrors (verified)
testCases := []struct {
    name      string
    protoFile string
    wantErr   string // substring expected in stderr
}{
    {
        name:      "duplicate GET endpoints",
        protoFile: "invalid_duplicate_routes.proto",
        wantErr:   "duplicate route: GET /api/v1/users",
    },
    {
        name:      "static vs parameterized conflict",
        protoFile: "invalid_route_conflict.proto",
        wantErr:   "route conflict: GET /users/search vs GET /users/{id}",
    },
}
```

### Anti-Patterns to Avoid
- **Generating full krakend.json:** The plugin generates endpoint fragments only. Global config (port, TLS, telemetry) is deployment-specific. Users compose fragments via KrakenD Flexible Config.
- **Using map[string]any for endpoint structs:** Loses type safety. Use typed Go structs with `json` tags. This makes golden tests reliable and catches schema drift at compile time.
- **Emitting empty `input_headers: []` or `input_query_strings: []`:** Use `omitempty` so these fields are absent when no annotations exist. KrakenD treats absent identically to empty array (forwards nothing).
- **Plugin flag for host:** User decided host lives in annotations only. No `--krakend_opt=host=...`.
- **Emitting timeout when not annotated:** User decided to let KrakenD apply its own defaults. Only include timeout when explicitly set via annotation.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP path/method extraction | Custom annotation parser | `annotations.GetMethodHTTPConfig()` | Already handles all sebuf.http.config parsing, path param extraction, method conversion |
| Base path prepending | Manual string concat | `annotations.BuildHTTPPath()` | Already handles slash normalization, empty paths, leading slashes |
| Header list extraction | Read proto extensions directly | `annotations.GetServiceHeaders()` + `annotations.GetMethodHeaders()` + `annotations.CombineHeaders()` | Handles merge semantics, deduplication, sorting |
| Query param extraction | Parse field options manually | `annotations.GetQueryParams()` | Returns structured QueryParam with param name, field kind, required flag |
| Path param extraction | Regex on path strings | `annotations.ExtractPathParams()` | Already compiled regex, returns []string of param names |
| HTTP method to string | Switch on enum | `annotations.HTTPMethodToString()` | Handles all enum values including UNSPECIFIED default |
| JSON output formatting | fmt.Sprintf JSON | `json.MarshalIndent(endpoints, "", "  ")` | Correct escaping, valid JSON guaranteed |
| Golden file update workflow | Custom test helpers | Follow UPDATE_GOLDEN=1 pattern from openapiv3 | Established pattern, consistent DX |

**Key insight:** The existing `internal/annotations/` package covers >80% of the data extraction needed for KrakenD endpoint generation. The new code is assembly logic (combining extracted data into KrakenD JSON structs) and validation logic (detecting route conflicts). Almost no new annotation parsing is needed except for the new `sebuf.krakend` extensions.

## Common Pitfalls

### Pitfall 1: KrakenD Requires One Endpoint Per Method (Not Per Path)
**What goes wrong:** Generator groups RPCs by path and produces one endpoint object for `/api/v1/users`. KrakenD silently routes only GET (its default), making POST unreachable.
**Why it happens:** Other tools treat path as the primary key with method as a property. KrakenD requires separate endpoint objects for each HTTP method on the same path.
**How to avoid:** Emit one endpoint object per RPC. Two RPCs on the same path with different methods = two separate endpoint objects. This is already how protoc-gen-openapiv3 works (one operation per method per path), so following the same iteration pattern naturally produces correct output.
**Warning signs:** Golden test with GET /users and POST /users shows only one endpoint object in output.

### Pitfall 2: Zero-Trust Parameter Dropping
**What goes wrong:** Generated config omits `input_headers` and `input_query_strings`. KrakenD forwards NO headers and NO query strings to backends. Authentication and filtering break silently.
**Why it happens:** KrakenD's zero-trust model blocks all forwarding by default. Other gateways (nginx, Kong) forward most headers by default. Developers assume headers pass through.
**How to avoid:** Always populate `input_headers` from service_headers + method_headers annotations. Always populate `input_query_strings` from query annotations. Use `omitempty` so fields are absent (not empty array) when no annotations exist. Never emit `["*"]`.
**Warning signs:** Backend logs show missing Authorization header or missing query params when requests go through gateway.

### Pitfall 3: Static vs Parameterized Route Conflicts
**What goes wrong:** Service defines `/users/search` (GET, static) and `/users/{id}` (GET, parameterized). KrakenD's httprouter cannot distinguish these -- `search` matches as a value for `{id}`. Either route is unreachable or KrakenD panics.
**Why it happens:** KrakenD uses httprouter which requires explicit matches only. Go's net/http mux and most other routers handle this with longest-prefix matching. sebuf protos freely mix these because the server-side router handles it fine.
**How to avoid:** Build a trie of path segments per HTTP method. At each level, if both literal segments (like "search") and a parameterized segment (like "{id}") exist as siblings, report a conflict. Include both paths and RPC names in the error message.
**Warning signs:** A golden test with `/notes/by-tag` and `/notes/{id}` in the same service should trigger validation failure.

### Pitfall 4: Backend Host Must Include Scheme
**What goes wrong:** Annotation sets host as `backend:8080`. KrakenD fails to connect because it requires `http://backend:8080` with scheme prefix.
**Why it happens:** KrakenD's host field expects a full URL base (with scheme). Developers may omit `http://` or `https://`.
**How to avoid:** Consider validation at generation time: if host does not start with `http://` or `https://`, emit a warning or error. However, this may be too strict -- some users use service mesh with custom schemes. At minimum, document the requirement clearly.
**Warning signs:** KrakenD logs connection errors or "no such host" when host lacks scheme.

### Pitfall 5: CORE-03 vs CONTEXT.md Host Decision Conflict
**What goes wrong:** Implementation follows CORE-03 (plugin parameter for host) but user decided in CONTEXT.md that host lives in annotations only.
**Why it happens:** REQUIREMENTS.md and success criteria were written before the CONTEXT.md discussion session refined decisions. The CONTEXT.md user decisions take precedence.
**How to avoid:** Host comes from `gateway_config` annotation at service level, overridable at method level via `endpoint_config`. No `--krakend_opt=host=...` flag. If a service has no host annotation, generation should fail with a clear error (host is required for backend config).
**Warning signs:** Tests or code that parse a "host" plugin parameter.

### Pitfall 6: KrakenD Method Field Defaults to GET
**What goes wrong:** Generator omits the `method` field on a POST endpoint. KrakenD defaults to GET, routing incorrectly.
**Why it happens:** KrakenD's default method is GET. If omitted, all endpoints become GET regardless of the proto annotation.
**How to avoid:** Always emit the `method` field on every endpoint. Never rely on KrakenD defaults for HTTP method.
**Warning signs:** POST/PUT/PATCH/DELETE endpoints responding to GET requests through the gateway.

## Code Examples

Verified patterns from the existing codebase:

### Reading sebuf.http.config for Endpoint Routing
```go
// Source: internal/annotations/http_config.go (verified)
config := annotations.GetMethodHTTPConfig(method)  // Returns *HTTPConfig or nil
if config == nil {
    return // RPC has no HTTP annotation, skip it
}
basePath := annotations.GetServiceBasePath(service)
fullPath := annotations.BuildHTTPPath(basePath, config.Path)
httpMethod := config.Method  // "GET", "POST", "PUT", "DELETE", "PATCH"
```

### Auto-Deriving input_headers from Header Annotations
```go
// Source: internal/annotations/headers.go (verified)
serviceHeaders := annotations.GetServiceHeaders(service)
methodHeaders := annotations.GetMethodHeaders(method)
combined := annotations.CombineHeaders(serviceHeaders, methodHeaders)

var inputHeaders []string
for _, h := range combined {
    inputHeaders = append(inputHeaders, h.GetName())
}
// inputHeaders is now ["X-API-Key", "X-Request-ID", ...] for KrakenD
```

### Auto-Deriving input_query_strings from Query Annotations
```go
// Source: internal/annotations/query.go (verified)
queryParams := annotations.GetQueryParams(method.Input)

var inputQueryStrings []string
for _, qp := range queryParams {
    inputQueryStrings = append(inputQueryStrings, qp.ParamName)
}
// inputQueryStrings is now ["page", "page_size", "filter", ...] for KrakenD
```

### Per-Service File Output Pattern
```go
// Source: cmd/protoc-gen-openapiv3/main.go (verified)
for _, service := range file.Services {
    // Generate JSON for this service
    endpoints := generator.GenerateService(service)
    output, _ := json.MarshalIndent(endpoints, "", "  ")

    // Write as ServiceName.krakend.json
    filename := fmt.Sprintf("%s.krakend.json", service.Desc.Name())
    gf := plugin.NewGeneratedFile(filename, "")
    gf.Write(output)
    gf.Write([]byte("\n"))  // Trailing newline
}
```

### Reading New KrakenD Annotations
```go
// Pattern from: internal/annotations/http_config.go (adapted for krakend package)
func GetGatewayConfig(service *protogen.Service) *krakend.GatewayConfig {
    options := service.Desc.Options()
    if options == nil {
        return nil
    }
    serviceOptions, ok := options.(*descriptorpb.ServiceOptions)
    if !ok {
        return nil
    }
    ext := proto.GetExtension(serviceOptions, krakend.E_GatewayConfig)
    if ext == nil {
        return nil
    }
    config, ok := ext.(*krakend.GatewayConfig)
    if !ok || config == nil {
        return nil
    }
    return config
}
```

## Discretion Recommendations

### JSON Output Structure: Bare Array

**Recommendation:** Emit a bare JSON array of endpoint objects, not a wrapper object.

**Rationale:** KrakenD Flexible Config integrates endpoint fragments using the `{{ include }}` or `{{ marshal }}` directives within a `range` loop over settings files. The cleanest integration pattern is:

```json
// UserService.krakend.json -- bare array
[
  {
    "endpoint": "/api/v1/users",
    "method": "GET",
    ...
  },
  {
    "endpoint": "/api/v1/users",
    "method": "POST",
    ...
  }
]
```

In the KrakenD main template:
```
"endpoints": [
  {{ $first := true }}
  {{ range $file := .settings_files }}
    {{ $endpoints := include $file }}
    {{ range $ep := $endpoints }}
      {{ if not $first }},{{ end }}
      {{ marshal $ep }}
      {{ $first = false }}
    {{ end }}
  {{ end }}
]
```

A wrapper object (`{"endpoints": [...]}`) would require extra unwrapping in the template. A bare array is simpler to consume.

However, there is one complication: `{{ include }}` inserts plain text, so including a bare array means the user must handle comma separation between files. An alternative is a settings-file approach where endpoint data goes in a `.json` settings file and the template iterates it. But since `include` is the most common pattern and the comma handling is a well-documented KrakenD pattern, a bare array is the cleanest choice.

### Annotation Proto Message Design

**Recommendation:** Minimal messages for Phase 12, with extension points for Phase 13.

```protobuf
syntax = "proto3";
package sebuf.krakend;

import "google/protobuf/descriptor.proto";

option go_package = "github.com/SebastienMelki/sebuf/krakend;krakend";

// GatewayConfig sets service-wide KrakenD defaults.
// Applied via (sebuf.krakend.gateway_config) on a service.
message GatewayConfig {
  // Backend host URL (e.g., "http://backend:8080"). Required.
  // Must include scheme (http:// or https://).
  repeated string host = 1;

  // Default timeout for all endpoints in this service.
  // KrakenD duration format: "2s", "500ms", "1m".
  // Omit to use KrakenD's built-in default.
  string timeout = 2;
}

// EndpointConfig overrides service-level defaults for a specific RPC.
// Applied via (sebuf.krakend.endpoint_config) on an rpc method.
message EndpointConfig {
  // Override backend host for this specific endpoint.
  repeated string host = 1;

  // Override timeout for this specific endpoint.
  string timeout = 2;
}

extend google.protobuf.ServiceOptions {
  GatewayConfig gateway_config = 51001;
}

extend google.protobuf.MethodOptions {
  EndpointConfig endpoint_config = 51002;
}
```

**Design notes:**
- `host` is `repeated string` because KrakenD supports load balancing across multiple hosts
- Timeout is a `string` (not duration) because KrakenD uses Go duration format ("2s", "500ms") and we pass it through as-is
- Extension numbers 51001-51002 leave room for 51003+ in Phase 13 (rate limiting, auth, circuit breaker)
- Messages are intentionally minimal for Phase 12; Phase 13 will add `extra_config`-related fields

### Plugin Architecture

**Recommendation:** Three-file structure in `internal/krakendgen/`:

1. **types.go** -- KrakenD JSON struct types (Endpoint, Backend). Pure data, no logic.
2. **generator.go** -- Core generation: iterate RPCs, extract annotations, build Endpoint structs, merge service/method configs. Single `GenerateService(*protogen.Service) ([]Endpoint, error)` entry point.
3. **validation.go** -- Route conflict detection: duplicate (path, method) tuples, static vs parameterized conflicts. Called before marshaling. Returns structured errors.

Plus test infrastructure:
4. **golden_test.go** -- Exhaustive golden file tests (build plugin, run protoc, byte-compare)
5. **validation_test.go** -- Unit tests for conflict detection functions
6. **testdata/proto/** -- Test proto files
7. **testdata/golden/** -- Expected JSON output

### Golden Test Coverage Strategy

**Recommendation:** Follow the openapiv3 pattern with these test categories:

**Core generation (TEST-01):**
- `simple_service.proto` -- Single service with GET/POST/PUT/DELETE RPCs, base_path
- `timeout_config.proto` -- Service-level timeout, method-level override, no timeout (omitted)
- `host_config.proto` -- Service-level host, method-level host override, multiple hosts
- `multiple_services.proto` -- Two services in one file, separate output files

**Auto-derived forwarding (TEST-03):**
- `headers_forwarding.proto` -- Service headers only, method headers only, combined (method overrides service)
- `query_forwarding.proto` -- Query params on GET method, multiple params, no params (field omitted)
- `combined_forwarding.proto` -- Headers + query params together

**Validation errors (TEST-04):**
- `invalid_duplicate_routes.proto` -- Two RPCs with same (path, method)
- `invalid_route_conflict.proto` -- Static route and parameterized route at same level
- `invalid_no_host.proto` -- Service with no gateway_config (missing host)

### Static vs Parameterized Route Conflict Detection

**Recommendation:** Path segment trie approach.

```go
// Build a trie of path segments, keyed by HTTP method.
// For each endpoint, split path into segments.
// At each trie level, track whether we have literal segments and/or a param segment.
// Conflict exists when a literal and a param coexist at the same level under the same parent.

type routeNode struct {
    children   map[string]*routeNode  // literal segment -> child
    paramChild *routeNode             // {param} child (at most one)
    paramName  string                 // name of the param if paramChild is set
    endpoints  []string               // RPC names registered at this node (leaf)
}

func detectConflicts(endpoints []endpointInfo) []error {
    // Group by HTTP method first
    byMethod := map[string][]endpointInfo{}
    for _, ep := range endpoints {
        byMethod[ep.method] = append(byMethod[ep.method], ep)
    }

    var errors []error
    for method, eps := range byMethod {
        root := &routeNode{children: map[string]*routeNode{}}
        for _, ep := range eps {
            if err := root.insert(ep); err != nil {
                errors = append(errors, err)
            }
        }
    }
    return errors
}
```

At each node during insertion: if a new literal segment is being inserted and `paramChild` already exists (or vice versa), report a conflict. If two endpoints register at the exact same leaf node, report a duplicate.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| KrakenD v2.x with YAML/TOML config | KrakenD v2.13+ with JSON config (lint only validates JSON) | KrakenD v2.0+ | JSON is the only lint-validated format; generate JSON |
| Manual KrakenD config | p3ym4n/krakend-generator (Go structs + json.Marshal) | Prior art | Validates the typed-structs approach for generating KrakenD config |
| Monolithic krakend.json | Flexible Config with endpoint fragments | KrakenD v1.0+ | Per-service fragments composed via templates is the recommended pattern |

**Deprecated/outdated:**
- KrakenD v1.x config format: Version 3 is current (since KrakenD v2.0)
- YAML/TOML KrakenD config: While supported, only JSON gets `krakend check --lint` validation

## Open Questions

1. **Should host be required or optional?**
   - What we know: KrakenD backend objects require a `host` array. Without it, the endpoint is invalid. The user decided host lives in annotations (not plugin flag).
   - What's unclear: Should the generator fail if no `gateway_config` annotation exists (no host), or should it be possible to generate endpoints without a host (for use cases where the host is injected by FC templates)?
   - Recommendation: Require host in `gateway_config`. If absent, fail generation with a clear error: "service X has no (sebuf.krakend.gateway_config) annotation -- backend host is required." Users can always add an FC template variable as the host value if needed. This catches missing annotations early rather than producing invalid KrakenD config.

2. **Trailing newline in JSON output**
   - What we know: Pretty-printed JSON from `json.MarshalIndent` does not include a trailing newline. Some tools and CI checks expect files to end with newline.
   - Recommendation: Append `\n` after the JSON output, consistent with how most text files work. The openapiv3 generator's YAML output naturally ends with newline; keep the same convention.

3. **How should RPCs without sebuf.http.config be handled?**
   - What we know: User decided "every RPC with sebuf.http.config automatically generates a KrakenD endpoint." RPCs without this annotation are skipped.
   - Recommendation: Skip silently. This is consistent with how protoc-gen-openapiv3 handles RPCs without HTTP config -- it generates a default path but we should NOT do that for KrakenD because the default path format (`/ServiceName/MethodName`) is not useful for a gateway.

## Sources

### Primary (HIGH confidence)
- [KrakenD Endpoint Configuration](https://www.krakend.io/docs/endpoints/) -- Endpoint fields, method default (GET), timeout format ("2s"), output_encoding
- [KrakenD Backend Configuration](https://www.krakend.io/docs/backends/) -- Backend fields: url_pattern, host (array), method, encoding
- [KrakenD Parameter Forwarding](https://www.krakend.io/docs/endpoints/parameter-forwarding/) -- Zero-trust model: input_headers defaults to [], input_query_strings defaults to [], case sensitivity rules
- [KrakenD Flexible Config](https://www.krakend.io/docs/configuration/flexible-config/) -- FC_ENABLE, include/marshal functions, settings files, template syntax
- [KrakenD Configuration Structure](https://www.krakend.io/docs/configuration/structure/) -- Version 3, endpoints array, root-level fields
- [KrakenD Templates](https://www.krakend.io/docs/configuration/templates/) -- include/marshal functions, range iteration, comma handling pattern
- Existing sebuf codebase (verified 2026-02-25):
  - `internal/annotations/` -- All shared annotation parsing functions
  - `internal/openapiv3/generator.go` -- Reference architecture for per-service output generation
  - `internal/openapiv3/exhaustive_golden_test.go` -- Golden test pattern
  - `internal/tsservergen/golden_test.go` -- Validation error test pattern
  - `cmd/protoc-gen-openapiv3/main.go` -- Plugin entry point pattern
  - `proto/sebuf/http/annotations.proto` -- Extension number registry (50003-50020 used)
  - `proto/sebuf/http/headers.proto` -- Header annotation structure
  - `Makefile` -- Auto-discovers cmd/ directories, builds to bin/

### Secondary (MEDIUM confidence)
- [KrakenD Multiple Methods Issue #398](https://github.com/krakend/krakend-ce/issues/398) -- Confirms one endpoint per method requirement
- [KrakenD Route Collision Issue #292](https://github.com/devopsfaith/krakend/issues/292) -- Path variable collision behavior
- `.planning/research/SUMMARY.md` -- Project-level research (verified 2026-02-25)
- `.planning/research/PITFALLS.md` -- KrakenD-specific pitfalls documentation

### Tertiary (LOW confidence)
- Backend host validation (scheme requirement): Inferred from KrakenD docs showing all examples with `http://` or `https://` prefix. No official statement that scheme is strictly required; KrakenD may infer it. Needs runtime validation.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- Zero new dependencies; existing protogen + stdlib json + existing annotations package. Verified against codebase.
- Architecture: HIGH -- Mirrors protoc-gen-openapiv3 exactly. All patterns verified from existing codebase source code.
- Pitfalls: HIGH -- KrakenD-specific constraints verified from official docs and GitHub issues. Route conflict detection approach is straightforward trie-based algorithm.
- Annotation design: MEDIUM -- Proto message structure is reasonable but field naming and ergonomics may need refinement during implementation. Extension numbers 51001-51002 confirmed available.

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (30 days -- stable domain, KrakenD releases are incremental)
