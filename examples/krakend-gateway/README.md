# KrakenD Gateway Example

End-to-end demo: annotate protos with `sebuf.krakend` annotations, generate a ready-to-use `krakend.json`, and run a real KrakenD gateway in front of a Go backend.

## Quick Start

```bash
# 1. Build plugins (from repo root)
make build

# 2. Generate code + start gateway
cd examples/krakend-gateway
make demo

# 3. Test the gateway
make test

# 4. Shut down
make docker-down
```

## Architecture

```
    Client (curl)
         |
    :8080 (host)
         |
   +-----------+
   |  KrakenD  |  docker: gateway
   |  Gateway  |  reads gateway/krakend.json
   +-----+-----+
         |
   users-backend:8080  /  products-backend:8080
         |                      |
         +----------+-----------+
                    |
             +-------------+
             | Go Backend  |  docker: backend
             | UserService |  (both network aliases
             | ProductSvc  |   point to same container)
             +-------------+
                 :8080
```

Single Go binary serves both services. Docker Compose runs it with two network aliases (`users-backend`, `products-backend`) matching the proto `host` annotations.

## What Gets Generated

```bash
make generate
```

Runs two protoc invocations:

1. `protoc-gen-krakend` → `gateway/krakend.json` (single combined config, ready for KrakenD)
2. `protoc-gen-go` + `protoc-gen-go-http` → `api/` (Go types + HTTP handlers)

No jq, no templates, no manual stitching. One command, everything works.

## Directory Structure

```
examples/krakend-gateway/
  proto/
    models/common.proto              Shared messages (Pagination)
    services/
      user_service.proto             JWT, rate limiting, headers, query params
      product_service.proto          Circuit breaker, caching, concurrent calls
  gateway/
    krakend.json                     (generated) Ready-to-use KrakenD config
  api/                               (generated) Go protobuf types + HTTP handlers
  main.go                            Backend server (UserService + ProductService)
  Dockerfile                         Multi-stage Go build
  docker-compose.yml                 Gateway + backend orchestration
  Makefile                           generate, run, demo, test, clean
```

## Running Locally (No Docker)

```bash
make generate
make run
# Backend on :8080 — no gateway, direct access
curl http://localhost:8080/api/v1/products
```

## Proto Annotations Reference

Each annotation is demonstrated with inline comments in the proto files.

### gateway_config (Service-Level)

Sets defaults for all endpoints in a service:

```protobuf
option (sebuf.krakend.gateway_config) = {
  host: ["http://users-backend:8080"]
  timeout: "3s"
  rate_limit: { max_rate: 100, strategy: RATE_LIMIT_STRATEGY_IP }
  jwt: { alg: JWT_ALGORITHM_RS256, jwk_url: "..." }
  circuit_breaker: { interval: 60, timeout: 10, max_errors: 3 }
  cache: { shared: true }
  concurrent_calls: 2
};
```

### endpoint_config (Method-Level Overrides)

Override service defaults per-RPC:

```protobuf
rpc UpdateUser(UpdateUserRequest) returns (User) {
  option (sebuf.krakend.endpoint_config) = {
    rate_limit: { max_rate: 50, strategy: RATE_LIMIT_STRATEGY_HEADER, key: "X-API-Key" }
  };
}
```

### Rate Limiting

**Endpoint-level** (`qos/ratelimit/router`):

| Field | Description |
|-------|-------------|
| `max_rate` | Global requests/second |
| `client_max_rate` | Per-client requests/second |
| `strategy` | `RATE_LIMIT_STRATEGY_IP`, `_HEADER`, `_PARAM` |
| `key` | Header/param name (required for HEADER/PARAM) |

**Backend-level** (`qos/ratelimit/proxy`):

| Field | Description |
|-------|-------------|
| `max_rate` | Max requests/second to backend |
| `capacity` | Burst allowance |

### JWT Authentication

| Field | Description |
|-------|-------------|
| `alg` | `JWT_ALGORITHM_RS256`, `_HS256`, `_ES256`, etc. |
| `jwk_url` | JWKS endpoint |
| `audience` | Expected `aud` claim(s) |
| `issuer` | Expected `iss` claim |
| `cache` | Cache JWKS responses |
| `propagate_claims` | Forward claims as headers (auto-added to `input_headers`) |

### Circuit Breaker

| Field | Description |
|-------|-------------|
| `interval` | Error sampling window (seconds) |
| `timeout` | Open duration before probe (seconds) |
| `max_errors` | Errors to trigger opening |

### Caching

Two mutually exclusive modes:

- **Shared**: `cache: { shared: true }` — global shared cache
- **Sized**: `cache: { max_items: 500, max_size: 5242880 }` — per-endpoint

### Concurrent Calls

```protobuf
concurrent_calls: 2  // Send N requests, return fastest
```

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

## Auto-Derived Forwarding

The generator reads `sebuf.http` annotations to automatically populate KrakenD's zero-trust header/query forwarding:

- **`input_headers`**: Derived from `service_headers` + `method_headers` annotations, plus JWT `propagate_claims` headers
- **`input_query_strings`**: Derived from `query` field annotations

No manual `input_headers` lists needed — the proto is the single source of truth.
