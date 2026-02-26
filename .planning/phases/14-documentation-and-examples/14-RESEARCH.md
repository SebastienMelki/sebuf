# Phase 14: Documentation and Examples - Research

**Researched:** 2026-02-25
**Domain:** KrakenD documentation, Flexible Config integration, proto enum design, `krakend check` validation
**Confidence:** HIGH

## Summary

Phase 14 is the documentation and examples phase for protoc-gen-krakend. The generator is feature-complete (Phases 12-13 done) and produces valid per-service KrakenD JSON fragments. This phase needs to: (1) create a comprehensive example in `examples/` showing all KrakenD annotations, (2) write a Flexible Config integration guide, (3) add `krakend check -lc` validation as a test step, and (4) convert raw string fields to proto enums for type safety.

The critical technical findings are: KrakenD rate limit `strategy` has exactly 3 valid values (`ip`, `header`, `param`); JWT `alg` has exactly 13 valid values; KrakenD Flexible Config uses Go templates with `include` for partials and `range`/`marshal` for settings-based composition; and `krakend check -lc` is confirmed working on this machine (KrakenD 2.13.1) against existing golden files -- with one existing bug to fix (CacheConcurrentService fails lint due to `shared` + `max_items/max_size` being `oneOf` in KrakenD schema).

**Primary recommendation:** Multi-service example (2-3 services with different feature profiles), proto enums for `strategy` and `alg`, `krakend check -lc` test in golden_test.go, and a Flexible Config guide as both a markdown file and inline in the example.

<user_constraints>
## User Constraints (from discussion)

### Locked Decisions
1. Run `krakend check -lc` on ALL generated golden files as a test step -- validates output against KrakenD's official schema
2. Create an example in `examples/` that showcases ALL possible KrakenD features (gateway_config, endpoint_config, rate limiting, JWT, circuit breaker, caching, concurrent calls)
3. Document everything in both the top-level README and within the example itself
4. Use proto enums instead of raw strings for fields like rate limit `strategy`, JWT `alg`, and similar -- raw strings are not type-safe enough
5. KrakenD 2.13.1 works with `krakend check -lc` on generated files (the `-t` flag causes issues)

### Claude's Discretion
- How to structure the example (single service vs multi-service)
- README section organization for KrakenD docs
- Which specific fields should become enums vs remain strings
- Whether Flexible Config guide is a separate markdown file or inline in the example

### Deferred Ideas (OUT OF SCOPE)
- None explicitly deferred
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DOCS-01 | Example proto file demonstrating all KrakenD annotations with a working Flexible Config setup | Multi-service example structure, all annotation values researched, Flexible Config template patterns documented |
| DOCS-02 | Flexible Config integration guide showing how to compose per-service fragments into a full krakend.json | KrakenD FC_ENABLE mechanics, include/range/marshal template syntax, comma-handling pattern, directory structure all researched |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| KrakenD CLI | 2.13.1 | Validate generated configs | Installed on machine, confirmed working with `-lc` flag |
| Go templates | Built-in | KrakenD Flexible Config | Native KrakenD template engine (Go text/template + Sprig) |
| protoc | Existing | Proto compilation | Already used throughout project |
| buf | Existing | Proto build/dependency management | Already used in examples |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| `krakend check -lc` | Lint against online schema | Test step for golden files |
| `krakend check -c` (without -l) | Syntax-only check | Fallback if network unavailable |
| FC_ENABLE=1 | Enable Flexible Config mode | When composing per-service fragments |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Partials-based include | Settings + range | Range requires restructuring JSON; partials are simpler for literal JSON inclusion |
| Single comprehensive service | Multiple services | Multi-service better demonstrates compose-ability, which is the whole point |

## Architecture Patterns

### Recommended Example Structure
```
examples/krakend-gateway/
  README.md                     # Example docs + Flexible Config guide
  proto/
    models/
      common.proto              # Shared messages
    services/
      user_service.proto        # JWT + rate limiting + headers
      product_service.proto     # Circuit breaker + caching + concurrent calls
  gateway/
    krakend.tmpl                # Flexible Config template
    partials/
      user_service.json         # Generated: UserService.krakend.json endpoints
      product_service.json      # Generated: ProductService.krakend.json endpoints
    settings/
      dev.json                  # Dev environment settings
  Makefile                      # generate + compose + validate workflow
  buf.gen.yaml                  # Buf generation config with krakend plugin
  buf.yaml                      # Buf module config
  go.mod / go.sum               # Go module for example
```

### Pattern 1: Multi-Service Example with Feature Differentiation
**What:** Two services that exercise different subsets of KrakenD features, so every annotation appears at least once across the example, and the Flexible Config shows multi-service composition.
**When to use:** This is the recommended pattern for the example.
**Feature Distribution:**

| Feature | UserService | ProductService |
|---------|-------------|----------------|
| gateway_config (host, timeout) | Yes | Yes |
| endpoint_config (overrides) | Yes (timeout override) | Yes (circuit breaker override) |
| Rate limiting (endpoint) | Yes (strategy: ip) | Yes (strategy: header) |
| Rate limiting (backend) | Yes | No |
| JWT auth + propagate_claims | Yes | No |
| Circuit breaker | No | Yes |
| Caching | No | Yes |
| Concurrent calls | No | Yes |
| Headers (service + method) | Yes | Yes |
| Query params | Yes (list endpoint) | No |

### Pattern 2: Flexible Config Template (krakend.tmpl)
**What:** Go template that composes per-service JSON fragments into a complete krakend.json.
**Two approaches (recommend approach A):**

**Approach A: Partials-based include (simple, recommended)**
```
{
  "$schema": "https://www.krakend.io/schema/krakend.json",
  "version": 3,
  "endpoints": [
    {{ include "user_service.json" }},
    {{ include "product_service.json" }}
  ]
}
```
The partials are the generated `endpoints` array content (extracted from the full KrakenD config), comma-separated in the template. Each partial contains the endpoint objects for one service (without the outer `$schema`/`version`/`endpoints` wrapper).

**Problem with approach A:** The generated files are complete KrakenD configs with `$schema`, `version`, and `endpoints` wrapper. Including them directly would embed full configs inside the endpoints array, which is invalid.

**Solution: Use settings + marshal (approach B, actually recommended)**
Since protoc-gen-krakend generates complete standalone configs (not bare endpoint arrays), the Flexible Config guide should explain how to either:
1. Post-process generated files to extract just the endpoints array (e.g., `jq '.endpoints[]'`)
2. Use KrakenD settings to load the endpoint arrays and merge them with `range`

**Approach B: Settings-based range (handles generated file format)**
```json
// settings/services.json
{
  "service_files": ["UserService.krakend.json", "ProductService.krakend.json"]
}
```
```
// krakend.tmpl
{
  "$schema": "https://www.krakend.io/schema/krakend.json",
  "version": 3,
  "endpoints": [
    {{ $first := true }}
    {{ range $file := .services.service_files }}
      {{ $config := include $file | fromJson }}
      {{ range $idx, $ep := $config.endpoints }}
        {{ if not $first }},{{ end }}
        {{ $ep | marshal }}
        {{ $first = false }}
      {{ end }}
    {{ end }}
  ]
}
```

**Actually simplest approach (recommended): Direct jq extraction + partials**

The guide should show: `jq '.endpoints' UserService.krakend.json > partials/user_endpoints.json`

Then the template uses:
```
{
  "version": 3,
  "endpoints": [
    {{ include "user_endpoints.json" }}
    ,
    {{ include "product_endpoints.json" }}
  ]
}
```

Where each partial is a bare JSON array of endpoint objects (without the `[` `]` brackets -- just the comma-separated objects).

**Recommended final approach:** The Makefile extracts endpoints from generated files using `jq` and creates partials, then the krakend.tmpl includes them. This is the cleanest pattern.

### Pattern 3: krakend check -lc in Golden Tests
**What:** Add a test function that runs `krakend check -lc` on every golden file.
**Implementation:**
```go
func TestKrakenDSchemaValidation(t *testing.T) {
    if _, err := exec.LookPath("krakend"); err != nil {
        t.Skip("krakend CLI not found, skipping schema validation tests")
    }
    // Loop over all *.krakend.json files in golden dir
    // Run: krakend check -lc <file>
    // Assert exit code 0
}
```

### Anti-Patterns to Avoid
- **Single monolithic proto with every feature:** Makes the example confusing and doesn't demonstrate multi-service composition
- **Hand-editing generated JSON for partials:** Fragile; should be scripted with `jq`
- **Using `krakend check -t` flag:** Known to cause issues per user confirmation; only use `-lc`
- **Using `krakend check -n` instead of `-l`:** The `-n` uses embedded schema which may be outdated; `-l` downloads current schema

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Config validation | Custom JSON schema checker | `krakend check -lc` | Official tool, always current schema |
| Endpoint extraction | Manual copy-paste | `jq '.endpoints'` | Reliable, scriptable |
| Template composition | Custom merger tool | KrakenD Flexible Config | Native KrakenD feature, well-documented |
| Proto enum validation | String validation in generator | Proto enum types | Compile-time safety, IDE autocomplete |

## Common Pitfalls

### Pitfall 1: CacheConcurrentService Golden File KrakenD Lint Error
**What goes wrong:** `krakend check -lc` fails on `CacheConcurrentService.krakend.json` with `'oneOf' failed, subschemas 0, 1 matched` for `qos/http-cache`.
**Why it happens:** KrakenD schema enforces `oneOf`: either (`shared`) alone, or (`max_items` + `max_size`), or neither. The golden file has `shared: true` AND `max_items/max_size` together.
**How to avoid:** Fix the test proto `cache_concurrent_service.proto` to remove `shared: true` from the override that also has `max_items/max_size`. Also consider adding validation in the generator for this constraint.
**Warning signs:** This is an existing bug that MUST be fixed before adding `krakend check -lc` as a test step, otherwise the test will fail.

### Pitfall 2: Flexible Config Partial Format Mismatch
**What goes wrong:** Generated `.krakend.json` files are complete configs with `$schema`/`version`/`endpoints` wrapper. They cannot be directly included as partials in a krakend.tmpl `endpoints` array.
**Why it happens:** The generator produces standalone validatable configs (which is correct for `krakend check`).
**How to avoid:** The guide must show how to extract just the endpoint objects for composition. Use `jq '.endpoints[]'` to extract bare endpoint objects.
**Warning signs:** If the guide shows `{{ include "UserService.krakend.json" }}` inside endpoints array, it will produce invalid JSON.

### Pitfall 3: Proto Enum Backward Compatibility
**What goes wrong:** Changing `string strategy = 6` to `RateLimitStrategy strategy = 6` breaks existing users who pass string values.
**Why it happens:** Proto enum wire format is integer, not string. Existing proto text format `strategy: "ip"` would need to become `strategy: RATE_LIMIT_STRATEGY_IP`.
**How to avoid:** This IS a breaking change in proto text format but NOT in JSON wire format (protojson maps enum names). Since this is a new feature (v1.1, no existing users), it's safe. Document clearly in the example.
**Warning signs:** All existing test protos using string values need updating.

### Pitfall 4: Trailing Comma in Flexible Config Template
**What goes wrong:** JSON syntax error from trailing comma when the last service has no comma after it.
**Why it happens:** Go templates don't auto-handle JSON comma separation.
**How to avoid:** Use the `{{if $idx}},{{end}}` pattern or hardcode commas between known includes.
**Warning signs:** `krakend check` will catch this with a syntax error.

### Pitfall 5: `every` Field is KrakenD Duration, Not Proto Duration
**What goes wrong:** Confusion between Go duration format and proto duration well-known type.
**Why it happens:** KrakenD uses Go's `time.ParseDuration` format (e.g., "1s", "500ms", "1h").
**How to avoid:** Keep `every` as a string field in proto (don't use google.protobuf.Duration). Document valid suffixes.
**Warning signs:** N/A -- current implementation is already correct.

## Code Examples

### Enum Definitions for Proto (New)

These enums should be added to `proto/sebuf/krakend/krakend.proto`:

```protobuf
// RateLimitStrategy identifies the client for per-client rate limiting.
enum RateLimitStrategy {
  RATE_LIMIT_STRATEGY_UNSPECIFIED = 0;
  RATE_LIMIT_STRATEGY_IP = 1;       // Identify client by IP address
  RATE_LIMIT_STRATEGY_HEADER = 2;   // Identify client by HTTP header (requires key)
  RATE_LIMIT_STRATEGY_PARAM = 3;    // Identify client by URL path parameter (requires key)
}

// JWTAlgorithm specifies the JWT signing algorithm.
enum JWTAlgorithm {
  JWT_ALGORITHM_UNSPECIFIED = 0;
  JWT_ALGORITHM_RS256 = 1;
  JWT_ALGORITHM_RS384 = 2;
  JWT_ALGORITHM_RS512 = 3;
  JWT_ALGORITHM_HS256 = 4;
  JWT_ALGORITHM_HS384 = 5;
  JWT_ALGORITHM_HS512 = 6;
  JWT_ALGORITHM_ES256 = 7;
  JWT_ALGORITHM_ES384 = 8;
  JWT_ALGORITHM_ES512 = 9;
  JWT_ALGORITHM_PS256 = 10;
  JWT_ALGORITHM_PS384 = 11;
  JWT_ALGORITHM_PS512 = 12;
  JWT_ALGORITHM_EDDSA = 13;
}
```

Source: Official KrakenD documentation at https://www.krakend.io/docs/endpoints/rate-limit/ and https://www.krakend.io/docs/authorization/jwt-validation/

### Generator Enum-to-String Mapping

The generator must map proto enum values to KrakenD JSON string values:

```go
// Source: KrakenD official docs
func rateLimitStrategyToString(s krakend.RateLimitStrategy) string {
    switch s {
    case krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_IP:
        return "ip"
    case krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_HEADER:
        return "header"
    case krakend.RateLimitStrategy_RATE_LIMIT_STRATEGY_PARAM:
        return "param"
    default:
        return ""
    }
}

func jwtAlgorithmToString(a krakend.JWTAlgorithm) string {
    switch a {
    case krakend.JWTAlgorithm_JWT_ALGORITHM_RS256:
        return "RS256"
    // ... etc for all 13 algorithms
    default:
        return ""
    }
}
```

### krakend check Test Integration

```go
// Source: verified with krakend 2.13.1 on this machine
func TestKrakenDSchemaValidation(t *testing.T) {
    krakendPath, err := exec.LookPath("krakend")
    if err != nil {
        t.Skip("krakend CLI not found in PATH, skipping schema validation")
    }

    goldenDir := filepath.Join("testdata", "golden")
    entries, err := os.ReadDir(goldenDir)
    if err != nil {
        t.Fatalf("Failed to read golden dir: %v", err)
    }

    for _, entry := range entries {
        if !strings.HasSuffix(entry.Name(), ".krakend.json") {
            continue
        }
        t.Run(entry.Name(), func(t *testing.T) {
            filePath := filepath.Join(goldenDir, entry.Name())
            cmd := exec.Command(krakendPath, "check", "-lc", filePath)
            var stderr bytes.Buffer
            cmd.Stderr = &stderr
            if err := cmd.Run(); err != nil {
                t.Errorf("krakend check -lc failed for %s: %v\n%s",
                    entry.Name(), err, stderr.String())
            }
        })
    }
}
```

### Flexible Config Template (krakend.tmpl)

```
{
  "$schema": "https://www.krakend.io/schema/krakend.json",
  "version": 3,
  "endpoints": [
    {{ include "user_endpoints.json" }}
    ,
    {{ include "product_endpoints.json" }}
  ]
}
```

Where `user_endpoints.json` is extracted via:
```bash
jq '.endpoints | map(tostring) | join(",\n")' UserService.krakend.json
# Or more practically:
jq -c '.endpoints[]' UserService.krakend.json | paste -sd ',' -
```

### Makefile Workflow for Example

```makefile
generate:
    buf generate

# Extract endpoint arrays from generated per-service configs
partials:
    mkdir -p gateway/partials
    jq '.endpoints' generated/UserService.krakend.json > gateway/partials/user_endpoints.json
    jq '.endpoints' generated/ProductService.krakend.json > gateway/partials/product_endpoints.json

# Compose full krakend.json using Flexible Config
compose: partials
    FC_ENABLE=1 \
    FC_PARTIALS=gateway/partials \
    krakend check -lc gateway/krakend.tmpl

validate:
    krakend check -lc generated/UserService.krakend.json
    krakend check -lc generated/ProductService.krakend.json
```

## Fields to Convert to Enums vs Keep as Strings

| Field | Current Type | Recommendation | Rationale |
|-------|-------------|----------------|-----------|
| `RateLimitConfig.strategy` | string | **ENUM** (RateLimitStrategy) | Fixed set: ip, header, param |
| `JWTConfig.alg` | string | **ENUM** (JWTAlgorithm) | Fixed set: 13 algorithms |
| `RateLimitConfig.key` | string | **KEEP STRING** | Free-form header name or param name |
| `RateLimitConfig.every` | string | **KEEP STRING** | Go duration format, infinite valid values |
| `BackendRateLimitConfig.every` | string | **KEEP STRING** | Same as above |
| `GatewayConfig.timeout` | string | **KEEP STRING** | Go duration format |
| `EndpointConfig.timeout` | string | **KEEP STRING** | Go duration format |
| `CircuitBreakerConfig.name` | string | **KEEP STRING** | User-defined name |

**Confidence: HIGH** -- Based on official KrakenD documentation confirming the exact valid values for strategy and alg.

## KrakenD Configuration Field Reference

### Rate Limit Strategy Values (Confirmed)
| Value | Description | Requires `key` |
|-------|-------------|-----------------|
| `ip` | Identify client by IP address | No |
| `header` | Identify client by HTTP header value | Yes (header name) |
| `param` | Identify client by URL path parameter | Yes (param name) |

Source: https://www.krakend.io/docs/endpoints/rate-limit/

### JWT Algorithm Values (Confirmed)
All 13 supported algorithms:
`EdDSA`, `HS256`, `HS384`, `HS512`, `RS256`, `RS384`, `RS512`, `ES256`, `ES384`, `ES512`, `PS256`, `PS384`, `PS512`

Default: `RS256`

Source: https://www.krakend.io/docs/authorization/jwt-validation/

### KrakenD Cache `oneOf` Constraint
The `qos/http-cache` configuration enforces a `oneOf` schema constraint:
- Option 1: `shared` alone
- Option 2: `max_items` + `max_size` together
- Option 3: Neither (empty `{}` for uncapped cache)

**Cannot combine `shared` with `max_items`/`max_size`.** This is enforced by KrakenD's JSON schema validation.

Source: https://www.krakend.io/docs/backends/caching/

### KrakenD Flexible Config Environment Variables
| Variable | Required | Description |
|----------|----------|-------------|
| `FC_ENABLE=1` | Yes | Activates Flexible Configuration |
| `FC_PARTIALS` | No | Path to partials directory (inserted as-is) |
| `FC_TEMPLATES` | No | Path to templates directory (evaluated as Go templates) |
| `FC_SETTINGS` | No | Path to settings directory (JSON files for template variables) |
| `FC_OUT` | No | Save rendered config to file (useful for debugging) |

Source: https://www.krakend.io/docs/configuration/flexible-config/

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| String strategy field | Proto enum | This phase | Type safety, IDE autocomplete, compile-time validation |
| String alg field | Proto enum | This phase | Same benefits |
| No schema validation | `krakend check -lc` in tests | This phase | Every golden file validated against official KrakenD schema |
| Manual krakend.json | Flexible Config composition | This phase (docs) | Users learn to compose generated fragments |

## Existing Bug to Fix

**CacheConcurrentService golden file fails `krakend check -lc`:**
```
ERROR linting the configuration file:
- at '/endpoints/1/backend/0/extra_config/qos~1http-cache':
  'oneOf' failed, subschemas 0, 1 matched
```

The test proto `cache_concurrent_service.proto` sets `shared: true` alongside `max_items: 1000, max_size: 10485760` in the method-level override. This violates KrakenD's `oneOf` schema constraint. Fix by removing `shared: true` from the override.

All other 11 golden files pass `krakend check -lc` successfully (verified on this machine).

## Open Questions

1. **Flexible Config partial extraction approach**
   - What we know: Generated files are complete KrakenD configs. Partials need bare endpoint objects.
   - What's unclear: Best way to extract -- `jq` is cleanest but adds a dependency on the user's machine. Alternative: modify generator to also output bare endpoint arrays.
   - Recommendation: Use `jq` in the example Makefile. It's universally available and the approach is simple. Document as optional -- users can also manually copy endpoint objects.

2. **Should generator add cache oneOf validation?**
   - What we know: KrakenD schema rejects `shared` + `max_items/max_size`. Our generator currently doesn't enforce this.
   - What's unclear: Whether to add this to generator validation or let `krakend check` catch it.
   - Recommendation: Add generator-time validation to `validateCache()` to reject this combination. Catches errors earlier and makes the proto contract clearer.

3. **BSR publishing of krakend proto package**
   - What we know: `buf.build/sebmelki/sebuf` exists and includes the HTTP annotations. The krakend proto package needs publishing too.
   - What's unclear: Whether to publish in this phase or defer to DIST-02.
   - Recommendation: Defer to DIST-02. Example can use local paths for now.

## Sources

### Primary (HIGH confidence)
- KrakenD Rate Limiting docs: https://www.krakend.io/docs/endpoints/rate-limit/ -- strategy values (ip, header, param), every defaults, key usage
- KrakenD JWT Validation docs: https://www.krakend.io/docs/authorization/jwt-validation/ -- 13 algorithm values, propagate_claims format
- KrakenD Flexible Config docs: https://www.krakend.io/docs/configuration/flexible-config/ -- FC_ENABLE, partials, templates, settings
- KrakenD Template docs: https://www.krakend.io/docs/configuration/templates/ -- include, marshal, range, comma pattern
- KrakenD Check docs: https://www.krakend.io/docs/configuration/check/ -- -l, -c, -n, -t flags
- KrakenD Caching docs: https://www.krakend.io/docs/backends/caching/ -- shared/max_items oneOf constraint
- KrakenD Config Structure docs: https://www.krakend.io/docs/configuration/structure/ -- minimum valid config fields
- Local verification: `krakend check -lc` tested on all 12 golden files (11 pass, 1 known bug)

### Secondary (MEDIUM confidence)
- GitHub issue krakend/krakend-ce#403: Multiple endpoint files pattern -- shows range + comma-guard approach

### Tertiary (LOW confidence)
- None -- all findings verified with official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- KrakenD 2.13.1 confirmed installed and working, all values verified against official docs
- Architecture: HIGH -- Example structure follows existing patterns in `examples/`, Flexible Config patterns from official docs
- Pitfalls: HIGH -- Cache oneOf bug verified locally, Flexible Config gotchas documented from official sources
- Enum values: HIGH -- All strategy (3 values) and algorithm (13 values) confirmed from official KrakenD documentation

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (KrakenD schema is stable, enum values don't change between minor versions)
