# KrakenD Gateway Example

Generate KrakenD API gateway configuration from annotated protobuf definitions and compose multi-service fragments into a complete gateway config using Flexible Config.

## Quick Start

```bash
make all    # generate -> partials -> validate -> compose
```

Prerequisites:
- `protoc` (Protocol Buffers compiler)
- `krakend` CLI (for validation and Flexible Config)
- `jq` (for extracting partials)
- `protoc-gen-krakend` binary (build from repo root with `make build`)

## Directory Structure

```
examples/krakend-gateway/
  proto/
    models/
      common.proto              # Shared messages (Pagination)
    services/
      user_service.proto        # JWT, rate limiting, headers, query params
      product_service.proto     # Circuit breaker, caching, concurrent calls
  gateway/
    krakend.tmpl                # Flexible Config template (composes services)
    settings/
      service_files.json        # Service registry for the template
    partials/                   # (generated) Extracted endpoint arrays
  generated/                    # (generated) Per-service .krakend.json files
  Makefile                      # Workflow: generate, partials, validate, compose
  buf.yaml                      # Buf config (for linting only)
  buf.gen.yaml                  # Reference config (generation uses protoc directly)
```

## Proto Annotations Reference

Each annotation is demonstrated with inline comments in the proto files. Open the proto files to see the exact usage.

### gateway_config (Service-Level)

Sets defaults for all endpoints in a service. Applied via `option (sebuf.krakend.gateway_config)`.

```protobuf
option (sebuf.krakend.gateway_config) = {
  host: ["http://users-backend:8080"]  // Backend host(s)
  timeout: "3s"                        // Request timeout
  // ... rate_limit, jwt, circuit_breaker, cache, concurrent_calls
};
```

See: `proto/services/user_service.proto` lines 66-105

### endpoint_config (Method-Level Overrides)

Overrides service defaults for a specific RPC. Applied via `option (sebuf.krakend.endpoint_config)`.

```protobuf
rpc UpdateUser(UpdateUserRequest) returns (User) {
  option (sebuf.krakend.endpoint_config) = {
    rate_limit: { max_rate: 50, strategy: RATE_LIMIT_STRATEGY_HEADER, key: "X-API-Key" }
  };
}
```

See: `proto/services/user_service.proto` UpdateUser RPC

### Rate Limiting

**Endpoint-level** (`qos/ratelimit/router`) -- limits client requests to the gateway:

| Field | Description |
|-------|-------------|
| `max_rate` | Global requests/second across all clients |
| `client_max_rate` | Per-client requests/second |
| `strategy` | How to identify clients: `RATE_LIMIT_STRATEGY_IP`, `RATE_LIMIT_STRATEGY_HEADER`, `RATE_LIMIT_STRATEGY_PARAM` |
| `key` | Header/param name for `HEADER`/`PARAM` strategy |

**Backend-level** (`qos/ratelimit/proxy`) -- limits gateway requests to backends:

| Field | Description |
|-------|-------------|
| `max_rate` | Max requests/second to the backend |
| `capacity` | Burst allowance |

See: UserService (IP strategy) and ProductService (header strategy)

### JWT Authentication

Service-level only -- all endpoints share the same auth config.

| Field | Description |
|-------|-------------|
| `alg` | Signing algorithm: `JWT_ALGORITHM_RS256`, `JWT_ALGORITHM_HS256`, `JWT_ALGORITHM_ES256`, etc. |
| `jwk_url` | JWKS endpoint for public keys |
| `audience` | Expected `aud` claim(s) |
| `issuer` | Expected `iss` claim |
| `cache` | Cache JWKS responses (recommended) |
| `propagate_claims` | Forward claims as headers: `{claim: "sub", header: "X-User"}` |

Propagated claim headers are automatically added to `input_headers`.

See: `proto/services/user_service.proto` jwt block

### Circuit Breaker

Prevents cascading failures by opening the circuit when backends fail.

| Field | Description |
|-------|-------------|
| `interval` | Error sampling window (seconds) |
| `timeout` | How long circuit stays open before probing (seconds) |
| `max_errors` | Errors within interval that trigger opening |
| `name` | Label for logs and metrics |

See: ProductService (service-level) and CreateProduct (aggressive override)

### Caching

HTTP response caching with two mutually exclusive modes:

**Shared mode** -- uses KrakenD's global shared cache:
```protobuf
cache: { shared: true }
```

**Sized mode** -- per-endpoint cache with limits:
```protobuf
cache: { max_items: 500, max_size: 5242880 }  // 5 MB
```

These modes are **mutually exclusive**. Setting `shared: true` with `max_items` or `max_size` causes a generation-time validation error.

See: ProductService (shared at service level, sized override on GetProduct)

### Concurrent Calls

Send N identical requests to backends, return the fastest response. Useful for latency-sensitive reads.

```protobuf
concurrent_calls: 2  // Service default
// Override per-RPC:
option (sebuf.krakend.endpoint_config) = { concurrent_calls: 3 };
```

See: ProductService (2 at service level, 3 for GetProduct)

## Feature Distribution

| Feature | UserService | ProductService |
|---------|:-----------:|:--------------:|
| JWT Authentication | x | |
| Rate Limit (IP strategy) | x | |
| Rate Limit (Header strategy) | x (UpdateUser) | x |
| Backend Rate Limit | x | |
| Circuit Breaker | | x |
| Cache (shared) | | x |
| Cache (sized, override) | | x (GetProduct) |
| Concurrent Calls | | x |
| Query Parameters | x (ListUsers) | |
| Method Headers | x (GetUser) | |
| Service Headers | x | x |

## Flexible Config Integration Guide

KrakenD's Flexible Config lets you compose a complete gateway configuration from per-service fragments. This is the recommended approach for multi-service architectures where each team owns their API definition.

### Why Flexible Config?

Without Flexible Config, you would need to manually merge all service endpoint definitions into a single monolithic `krakend.json`. With sebuf's per-service generation + Flexible Config:

1. Each service defines its own gateway behavior in proto annotations
2. `protoc-gen-krakend` generates per-service configs (standalone, independently validatable)
3. Flexible Config composes them into a single gateway config at deployment time

### Step 1: Generate Per-Service Configs

```bash
make generate
```

Runs protoc with `protoc-gen-krakend` to produce per-service `.krakend.json` files in `generated/`:

```bash
protoc \
  --plugin=protoc-gen-krakend=../../bin/protoc-gen-krakend \
  --krakend_out=./generated \
  --proto_path=proto \
  --proto_path=../../proto \
  proto/services/user_service.proto proto/services/product_service.proto
```

Each generated file is a complete, standalone KrakenD config with `$schema` and `version`. You can validate them independently.

### Step 2: Extract Endpoint Partials

```bash
make partials
```

Extracts the `endpoints` array from each generated file, stripping the outer JSON envelope (`$schema`, `version`, array brackets). The result is bare endpoint objects suitable for `{{ include }}`:

```bash
# Extract endpoints array and strip outer [ ] brackets
jq '.endpoints' generated/UserService.krakend.json | sed '1d;$d' > gateway/partials/user_endpoints.json
```

### Step 3: Create the Gateway Template

The template (`gateway/krakend.tmpl`) uses `{{ include }}` to pull in partials:

```json
{
  "$schema": "https://www.krakend.io/schema/krakend.json",
  "version": 3,
  "endpoints": [
    {{ include "user_endpoints.json" }},
    {{ include "product_endpoints.json" }}
  ]
}
```

Each partial contains comma-separated endpoint objects. The comma between includes handles the join between services.

### Step 4: Validate the Composed Config

```bash
make compose
```

Runs `krakend check` with Flexible Config environment variables:

```bash
FC_ENABLE=1 \
  FC_PARTIALS=gateway/partials \
  FC_SETTINGS=gateway/settings \
  krakend check -l -c gateway/krakend.tmpl
```

| Variable | Purpose |
|----------|---------|
| `FC_ENABLE=1` | Enable Flexible Config template processing |
| `FC_PARTIALS` | Directory for `{{ include }}` file lookup |
| `FC_SETTINGS` | Directory for `{{ marshal }}` settings files |

### Validate Per-Service Configs Independently

Each per-service generated file is standalone and can be validated independently:

```bash
krakend check -l -c generated/UserService.krakend.json
krakend check -l -c generated/ProductService.krakend.json
```

This is useful for CI pipelines where you want to validate a single service's config without composing the full gateway.

## Adding a New Service

1. **Define the proto** with HTTP and KrakenD annotations:

   ```protobuf
   service OrderService {
     option (sebuf.http.service_config) = { base_path: "/api/v1" };
     option (sebuf.krakend.gateway_config) = {
       host: ["http://orders-backend:8080"]
       timeout: "3s"
     };

     rpc CreateOrder(CreateOrderRequest) returns (Order) {
       option (sebuf.http.config) = { path: "/orders", method: HTTP_METHOD_POST };
     };
   }
   ```

2. **Add the proto file** to the Makefile `PROTOS` variable:

   ```makefile
   PROTOS := proto/services/user_service.proto proto/services/product_service.proto proto/services/order_service.proto
   ```

3. **Add the partial extraction** to the `partials` target:

   ```makefile
   @jq '.endpoints' generated/OrderService.krakend.json | sed '1d;$$d' > gateway/partials/order_endpoints.json
   ```

4. **Add the include** to `gateway/krakend.tmpl`:

   ```
   "endpoints": [
     {{ include "user_endpoints.json" }},
     {{ include "product_endpoints.json" }},
     {{ include "order_endpoints.json" }}
   ]
   ```

5. **Run the workflow:**

   ```bash
   make all
   ```
