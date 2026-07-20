# HTTP Client Generation

> **Generate type-safe HTTP clients from your protobuf services**

The `protoc-gen-go-client` plugin generates type-safe HTTP clients that mirror your server API. Clients are generated alongside your server handlers and share the same protobuf types, ensuring full type safety across your entire API.

## Quick Start

### Installation

```bash
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-client@latest
```

### Configuration

Add the client generator to your `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: api
    opt: paths=source_relative
  - local: protoc-gen-go-http
    out: api
  - local: protoc-gen-go-client
    out: api
  - local: protoc-gen-openapiv3
    out: docs
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/yourorg/api"
)

func main() {
    // Create client with options
    client := api.NewUserServiceClient(
        "http://localhost:8080",
        api.WithUserServiceHTTPClient(&http.Client{
            Timeout: 30 * time.Second,
        }),
    )

    // Make requests
    user, err := client.GetUser(context.Background(), &api.GetUserRequest{
        UserId: "user-123",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Got user: %s", user.Name)
}
```

## Generated Components

For each service, the generator creates:

### 1. Client Interface

```go
// UserServiceClient is the client API for UserService.
type UserServiceClient interface {
    GetUser(ctx context.Context, req *GetUserRequest, opts ...UserServiceCallOption) (*User, error)
    CreateUser(ctx context.Context, req *CreateUserRequest, opts ...UserServiceCallOption) (*User, error)
    UpdateUser(ctx context.Context, req *UpdateUserRequest, opts ...UserServiceCallOption) (*User, error)
    DeleteUser(ctx context.Context, req *DeleteUserRequest, opts ...UserServiceCallOption) (*DeleteResponse, error)
}
```

### 2. Client Options (Configuration)

Options for configuring the client at creation time:

```go
// Create a customized client
client := api.NewUserServiceClient(
    "http://localhost:8080",

    // Custom HTTP client with timeout
    api.WithUserServiceHTTPClient(&http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns: 100,
        },
    }),

    // Default content type (JSON or Protobuf binary)
    api.WithUserServiceContentType(api.ContentTypeProto),

    // Default headers for all requests
    api.WithUserServiceDefaultHeader("X-Tenant-ID", "tenant-123"),
)
```

### 3. Call Options (Per-Request)

Options for customizing individual requests:

```go
// Make a request with per-call options
user, err := client.GetUser(ctx, req,
    // Override content type for this request
    api.WithUserServiceCallContentType(api.ContentTypeJSON),

    // Add headers for this request only
    api.WithUserServiceHeader("X-Request-ID", "req-456"),
    api.WithUserServiceHeader("X-Custom-Header", "value"),
)
```

### 4. Header Helper Options

The generator automatically creates helper options from your header annotations:

```protobuf
service UserService {
  option (sebuf.http.service_headers) = {
    required_headers: [{
      name: "X-API-Key"
      description: "API authentication key"
      required: true
    }]
  };

  rpc DeleteUser(DeleteUserRequest) returns (DeleteResponse) {
    option (sebuf.http.method_headers) = {
      required_headers: [{
        name: "X-Confirm-Delete"
        description: "Confirmation header for delete operations"
        required: true
      }]
    };
  };
}
```

Generates these convenient helpers:

```go
// Service-level header (ClientOption) - applied to all requests
client := api.NewUserServiceClient(
    "http://localhost:8080",
    api.WithUserServiceAPIKey("your-api-key"),  // Sets X-API-Key for all requests
)

// Method-level header (CallOption) - applied to specific request
_, err := client.DeleteUser(ctx, req,
    api.WithUserServiceCallConfirmDelete("true"),  // Sets X-Confirm-Delete for this request
)
```

## Content Type Support

Clients support both JSON and binary protobuf:

```go
const (
    ContentTypeJSON  = "application/json"
    ContentTypeProto = "application/x-protobuf"
)
```

### JSON (Default)

```go
// JSON is the default
client := api.NewUserServiceClient("http://localhost:8080")
```

The client automatically handles special JSON serialization, including messages with `unwrap` annotations for map values. See [JSON/Protobuf Compatibility](./json-protobuf-compatibility.md) for details.

### Binary Protobuf

For better performance with large payloads:

```go
// Set as default for all requests
client := api.NewUserServiceClient(
    "http://localhost:8080",
    api.WithUserServiceContentType(api.ContentTypeProto),
)

// Or per-request
user, err := client.GetUser(ctx, req,
    api.WithUserServiceCallContentType(api.ContentTypeProto),
)
```

## URL Building

### Path Parameters

Path parameters are automatically substituted from request fields:

```protobuf
rpc GetUser(GetUserRequest) returns (User) {
  option (sebuf.http.config) = {
    path: "/users/{user_id}"
    method: HTTP_METHOD_GET
  };
}

message GetUserRequest {
  string user_id = 1;
}
```

```go
// user_id is automatically inserted into the URL path
user, err := client.GetUser(ctx, &api.GetUserRequest{
    UserId: "user-123",  // Results in GET /users/user-123
})
```

### Nested Path Parameters

Multiple path parameters are supported:

```protobuf
rpc GetTeamMember(GetTeamMemberRequest) returns (Member) {
  option (sebuf.http.config) = {
    path: "/orgs/{org_id}/teams/{team_id}/members/{member_id}"
    method: HTTP_METHOD_GET
  };
}
```

```go
member, err := client.GetTeamMember(ctx, &api.GetTeamMemberRequest{
    OrgId:    "org-123",
    TeamId:   "team-456",
    MemberId: "member-789",
})
// Results in GET /orgs/org-123/teams/team-456/members/member-789
```

### Query Parameters

For GET and DELETE methods, fields are encoded as query parameters:

```protobuf
rpc ListProducts(ListProductsRequest) returns (ListProductsResponse) {
  option (sebuf.http.config) = {
    path: "/products"
    method: HTTP_METHOD_GET
  };
}

message ListProductsRequest {
  int32 page = 1 [(sebuf.http.query) = {name: "page"}];
  int32 limit = 2 [(sebuf.http.query) = {name: "limit"}];
  string category = 3 [(sebuf.http.query) = {name: "category"}];
  double min_price = 4 [(sebuf.http.query) = {name: "min_price"}];
  ProductStatus status = 5 [(sebuf.http.query) = {name: "status"}]; // enum — accepts name or number
}
```

```go
products, err := client.ListProducts(ctx, &api.ListProductsRequest{
    Page:     1,
    Limit:    20,
    Category: "electronics",
    MinPrice: 50.0,
})
// Results in GET /products?page=1&limit=20&category=electronics&min_price=50
```

## Error Handling

### Typed Errors

The client automatically handles error responses:

```go
user, err := client.GetUser(ctx, req)
if err != nil {
    // Check for validation errors (HTTP 400)
    var validationErr *sebufhttp.ValidationError
    if errors.As(err, &validationErr) {
        for _, violation := range validationErr.Violations {
            log.Printf("Field %s: %s", violation.Field, violation.Message)
        }
        return
    }

    // Check for generic errors
    var genericErr *sebufhttp.Error
    if errors.As(err, &genericErr) {
        log.Printf("Error: %s", genericErr.Message)
        return
    }

    // Network or other errors
    log.Printf("Request failed: %v", err)
}
```

### Custom Error Types

If your server returns custom proto error types, you can handle them:

```go
user, err := client.GetUser(ctx, req)
if err != nil {
    // Unmarshal to your custom error type
    var notFoundErr api.NotFoundError
    if errors.As(err, &notFoundErr) {
        log.Printf("Resource not found: %s %s",
            notFoundErr.ResourceType,
            notFoundErr.ResourceId)
        return
    }
}
```

## Complete Example

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "time"

    "github.com/yourorg/api"
    sebufhttp "github.com/SebastienMelki/sebuf/http"
)

func main() {
    // Create a configured client
    client := api.NewProductServiceClient(
        "http://localhost:8080",

        // Custom HTTP client
        api.WithProductServiceHTTPClient(&http.Client{
            Timeout: 30 * time.Second,
        }),

        // API key for all requests
        api.WithProductServiceAPIKey("your-api-key"),
    )

    ctx := context.Background()

    // List products with query parameters
    list, err := client.ListProducts(ctx, &api.ListProductsRequest{
        Page:     1,
        Limit:    10,
        Category: "electronics",
    })
    if err != nil {
        log.Fatalf("Failed to list products: %v", err)
    }
    log.Printf("Found %d products", len(list.Products))

    // Create a product
    product, err := client.CreateProduct(ctx, &api.CreateProductRequest{
        Name:        "New Product",
        Description: "A great product",
        Price:       99.99,
        CategoryId:  "electronics",
        Tags:        []string{"new", "featured"},
    })
    if err != nil {
        var validationErr *sebufhttp.ValidationError
        if errors.As(err, &validationErr) {
            log.Printf("Validation failed:")
            for _, v := range validationErr.Violations {
                log.Printf("  %s: %s", v.Field, v.Message)
            }
            return
        }
        log.Fatalf("Failed to create product: %v", err)
    }
    log.Printf("Created product: %s", product.Id)

    // Get a product
    retrieved, err := client.GetProduct(ctx, &api.GetProductRequest{
        ProductId: product.Id,
    })
    if err != nil {
        log.Fatalf("Failed to get product: %v", err)
    }
    log.Printf("Retrieved: %s - $%.2f", retrieved.Name, retrieved.Price)

    // Update with binary protobuf for better performance
    updated, err := client.UpdateProduct(ctx, &api.UpdateProductRequest{
        ProductId:   product.Id,
        Name:        "Updated Product",
        Description: "Even better",
        Price:       149.99,
        CategoryId:  "electronics",
        Tags:        []string{"updated", "premium"},
    },
        api.WithProductServiceCallContentType(api.ContentTypeProto),
    )
    if err != nil {
        log.Fatalf("Failed to update product: %v", err)
    }
    log.Printf("Updated: %s", updated.Name)

    // Delete with confirmation header
    _, err = client.DeleteProduct(ctx, &api.DeleteProductRequest{
        ProductId: product.Id,
    },
        api.WithProductServiceCallConfirmDelete("true"),
    )
    if err != nil {
        log.Fatalf("Failed to delete product: %v", err)
    }
    log.Printf("Product deleted")
}
```

## Best Practices

### 1. Reuse Clients

Create clients once and reuse them:

```go
// Good: Create once, reuse
var productClient api.ProductServiceClient

func init() {
    productClient = api.NewProductServiceClient(
        os.Getenv("API_URL"),
        api.WithProductServiceAPIKey(os.Getenv("API_KEY")),
    )
}

func GetProduct(id string) (*api.Product, error) {
    return productClient.GetProduct(context.Background(), &api.GetProductRequest{
        ProductId: id,
    })
}
```

### 2. Set Timeouts

Always configure timeouts:

```go
client := api.NewUserServiceClient(
    "http://localhost:8080",
    api.WithUserServiceHTTPClient(&http.Client{
        Timeout: 30 * time.Second,
    }),
)
```

### 3. Use Context for Cancellation

Pass contexts for cancellation support:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

user, err := client.GetUser(ctx, req)
```

### 4. Handle Errors Properly

Always check for specific error types:

```go
user, err := client.GetUser(ctx, req)
if err != nil {
    var validationErr *sebufhttp.ValidationError
    if errors.As(err, &validationErr) {
        // Handle validation error
        return
    }
    // Handle other errors
}
```

## TypeScript Client Generation

For TypeScript/JavaScript projects, sebuf also provides `protoc-gen-ts-client` which generates TypeScript HTTP clients with full type safety. See the [ts-client-demo example](../examples/ts-client-demo/) for a complete walkthrough.

Add it to your `buf.gen.yaml`. The TypeScript generators build the type modules
in a single pass over the full set of files — resolving cross-file references
into relative imports and emitting one shared `errors.ts` — so they **require
`strategy: all`**. The default `strategy: directory` runs the plugin once per
directory in isolation, which re-emits the shared type modules and `errors.ts`
from each pass and cannot resolve references into one consistent module tree:

```yaml
version: v2
plugins:
  - local: protoc-gen-ts-client
    out: ./client/generated
    opt: paths=source_relative
    strategy: all
```

The TypeScript client generates, using a per-proto **modules layout**:
- One type module per source proto (`<proto>.ts`) holding the request/response
  interfaces and enums for that file's messages
- A slim client module per proto (`<proto>_client.ts`) with the client class,
  importing its request/response types from the sibling type module
- A shared `errors.ts` at the output root exporting the `ValidationError` and
  `ApiError` classes (and `FieldViolation`); every client module imports these
  via a relative specifier
- Cross-package references become relative type-only imports between modules
  (e.g. `import type { ItemID } from "../../common/v1/types";`), so types
  defined in one proto package are reused, not re-declared
- Service-level headers as constructor options (e.g., `apiKey` from `X-API-Key`)
- Method-level headers as call options (e.g., `requestId` from `X-Request-ID`)
- Automatic query parameter encoding and path parameter substitution

## TypeScript Server Generation

For TypeScript server-side code generation, sebuf provides `protoc-gen-ts-server` which generates framework-agnostic HTTP server handlers using the Web Fetch API. See the [ts-fullstack-demo example](../examples/ts-fullstack-demo/) for a complete TS client + TS server working together from the same proto.

Add it to your `buf.gen.yaml`. Like the client generator, it emits a per-proto
modules layout and therefore **requires `strategy: all`**:

```yaml
version: v2
plugins:
  - local: protoc-gen-ts-server
    out: ./server/generated
    opt: paths=source_relative
    strategy: all
```

The TypeScript server generates, using the same per-proto **modules layout** as
the client (type module `<proto>.ts` per source proto, slim server module
`<proto>_server.ts` importing its types from the sibling type module, and the
shared `errors.ts` at the output root):
- Handler interface (`{Service}Handler`) with methods for each RPC
- Route descriptors (`RouteDescriptor[]`) for wiring into any framework
- `create{Service}Routes(handler, options)` factory function
- `ServerContext` with headers, path params, and raw request
- Header validation, query/body parsing, and error handling
- Proto-defined error interfaces (messages ending with "Error"), emitted into
  their proto's type module
- Works natively in Node 18+, Deno, Bun, and Cloudflare Workers

### TypeScript Custom Error Handling

Both TypeScript generators (client and server) automatically include TypeScript interfaces for any protobuf message whose name ends with "Error". This mirrors Go's convention where error messages automatically implement the `error` interface.

**Define error messages in proto:**
```protobuf
message NotFoundError {
  string resource_type = 1;
  string resource_id = 2;
}

message LoginError {
  string reason = 1;
  string email = 2;
  int32 retry_after_seconds = 3;
}
```

**Generated TypeScript interfaces (in the proto's type module `my_service.ts`, shared by both server and client):**
```typescript
export interface NotFoundError {
  resourceType: string;
  resourceId: string;
}

export interface LoginError {
  reason: string;
  email: string;
  retryAfterSeconds: number;
}
```

**Server — implement the generated interface and handle in `onError`:**
```typescript
import { type NotFoundError as NotFoundErrorType } from "./generated/proto/my_service.ts";

class NotFoundError extends Error implements NotFoundErrorType {
  resourceType: string;
  resourceId: string;
  constructor(resourceType: string, resourceId: string) {
    super(`${resourceType} '${resourceId}' not found`);
    this.resourceType = resourceType;
    this.resourceId = resourceId;
  }
}

const routes = createMyServiceRoutes(handler, {
  onError: (err, req) => {
    if (err instanceof NotFoundError) {
      const body: NotFoundErrorType = { resourceType: err.resourceType, resourceId: err.resourceId };
      return new Response(JSON.stringify(body), {
        status: 404,
        headers: { "Content-Type": "application/json" },
      });
    }
    // ... handle other errors
  },
});
```

**Client — parse `ApiError.body` using the generated interface:**
```typescript
import { ApiError } from "./generated/errors.ts";
import { type NotFoundError } from "./generated/proto/my_service.ts";

try {
  await client.getUser({ id: "not-found" });
} catch (e) {
  if (e instanceof ApiError && e.statusCode === 404) {
    const body = JSON.parse(e.body) as NotFoundError;
    console.log(body.resourceType); // "user"
    console.log(body.resourceId);   // "not-found"
  }
}
```

The proto definition serves as the single source of truth for error shapes — both server and client use the same generated interface for type safety across the wire.

## protobuf-es runtime (TypeScript)

By default the TypeScript generators emit their own plain-interface types (the
per-proto type module `<proto>.ts` described above). Passing
`ts_runtime=protobuf-es` switches both the client and server generators into
**protobuf-es transport mode**: instead of declaring their own interfaces, they
consume the message types and schemas emitted by
[`protoc-gen-es`](https://github.com/bufbuild/protobuf-es) (the `<proto>_pb.ts`
files) and serialize on the wire through protobuf-es's `fromJson`/`toJson`. This
gives you protobuf-es's spec-compliant **canonical proto3 JSON** encoding
(defaults, `bigint`, oneofs, well-known types) for free, shared across your whole
app.

> **Wire contract — read this before enabling es-mode.** es-mode speaks
> **canonical protojson** and applies **none** of sebuf's `sebuf.http`
> JSON-mapping annotations (`unwrap`, `oneof_config` flatten, `flatten` /
> `flatten_prefix`, `enum_value`, `timestamp_format`, `bytes_encoding`,
> `nullable`, `empty_behavior`, `enum_encoding`, `int64_encoding`). A sebuf Go
> server, by contrast, layers an annotation-aware transform
> (`MarshalJSONSebuf`) on top of protojson that deliberately diverges from
> canonical protojson wherever one of those annotations is set. es-mode does
> **not** yet have the TypeScript equivalent of that transform layer, so for any
> annotated proto it is on a **different, incompatible wire** than the sebuf Go
> server (and than the default hand-rolled TS output). It is therefore only
> wire-compatible with a sebuf server for protos that use **none** of those
> annotations. This is not left to chance: the generator **walks each RPC's
> request/response message closure and fails loud at generation time** if it
> finds any such annotation (see `CheckESMessageAnnotations` in
> `internal/tscommon/es_guard.go`). Use the default hand-rolled runtime for
> services that rely on JSON-mapping annotations.

### buf.gen.yaml shape

In this mode you run `protoc-gen-es` **and** the sebuf ts-client (or ts-server)
plugin together, into the same output directory. protoc-gen-es emits the
`<proto>_pb.ts` message schemas; the sebuf plugin emits the transport
client/server that imports them. Both sebuf plugins still require
`strategy: all` (see the sections above):

```yaml
version: v2
plugins:
  # 1. protoc-gen-es: emits <proto>_pb.ts message classes + schemas
  - local: protoc-gen-es
    out: ./generated
    opt:
      - target=ts
      - import_extension=js
  # 2. sebuf ts-client in protobuf-es transport mode
  - local: protoc-gen-ts-client
    out: ./generated
    opt:
      - paths=source_relative
      - ts_runtime=protobuf-es
    strategy: all
  # (and/or) sebuf ts-server in the same mode
  - local: protoc-gen-ts-server
    out: ./generated
    opt:
      - paths=source_relative
      - ts_runtime=protobuf-es
    strategy: all
```

The exact plugin options mirror what the generator's golden tests invoke:
`--es_opt=target=ts,import_extension=js` for protoc-gen-es, and
`--ts-client_opt=paths=source_relative,ts_runtime=protobuf-es` /
`--ts-server_opt=paths=source_relative,ts_runtime=protobuf-es` for the sebuf
plugins.

### Runtime dependency

protobuf-es transport mode has a runtime dependency on **`@bufbuild/protobuf`
v2** (the `<proto>_pb.ts` files and the generated transport both import
`create`, `fromJson`, `toJson`, `MessageInitShape`, and the message/`*Schema`
symbols from it). Install it in the consuming project:

```bash
npm install @bufbuild/protobuf
```

The generated goldens were produced with `@bufbuild/protoc-gen-es@2.12.1`.

The generated transport import only pulls in the `@bufbuild/protobuf` symbols a
given file actually uses (e.g. a GET-only client imports just `fromJson` and
`MessageInitShape`, not `create`/`toJson`), so the output compiles cleanly under
strict `noUnusedLocals`.

### Consumer-facing differences

Compared to the default plain-interface mode, code that uses the generated
client/server sees protobuf-es's types and conventions:

- **Branded messages.** Message types are protobuf-es branded types
  (`Message<"pkg.Name"> & {...}`), not structural interfaces. You cannot pass an
  arbitrary object literal where a full message is expected.
- **`create()` for construction; `MessageInitShape` on the boundaries.** To
  build a message value use protobuf-es's `create(SchemaSymbol, {...})`. Client
  methods and server handlers do not force you to call `create()` yourself,
  though: client methods accept `MessageInitShape<typeof RequestSchema>` (the
  loose init shape), and server handler methods **return**
  `MessageInitShape<typeof ResponseSchema>` — the generated code calls
  `create()` for you before encoding. So handlers can `return { ... }` with a
  plain init object.
- **Oneofs as `{ case, value }`.** A protobuf oneof is a discriminated union
  (`{ case: "bigText"; value: TextContent } | { case: "bigImage"; value:
  ImageContent } | { case: undefined; value?: undefined }`), not sibling
  optional fields.
- **int64 as `bigint`.** 64-bit integer fields (`int64`, `uint64`, `sint64`,
  `fixed64`, `sfixed64`) are represented as `bigint`, following protobuf-es.
- **Standalone functions, not a client class.** Each RPC is emitted as a
  top-level `export async function` that takes its config per call as a required
  options argument (`baseURL` required; `fetch`, `headers`, `signal` optional).
  The default plain-interface mode still emits a `class` you `new` and configure
  once.

  ```ts
  import { getUser } from "./gen/user_service_client.js";

  const opts = { baseURL: "https://api.example.com", headers: { apiKey } };
  const user = await getUser({ id: "123" }, opts);
  ```

  Because every RPC is an independent top-level function, bundlers **tree-shake**
  the ones you don't import — using a single method doesn't pull the whole
  service into your bundle. There is no client object to construct or pass
  around; hold the options object (plain data) if you want to reuse config.

  The options type is a **shared `RequestOptions`** (emitted once in the shared
  `client.ts`), so it is not duplicated per service. A service that declares
  typed headers instead gets a `{Service}RequestOptions` that **`extends
  RequestOptions`** and adds only its typed header properties:

  ```ts
  // client.ts (shared)
  export interface RequestOptions { baseURL: string; fetch?: typeof fetch; headers?: Record<string, string>; signal?: AbortSignal; }
  // token_service_client.ts (declares X-API-Key + X-Request-ID headers)
  export interface TokenServiceRequestOptions extends RequestOptions { apiKey?: string; requestId?: string; }
  ```

### Server-streaming (SSE)

Server-streaming RPCs **are** supported in protobuf-es mode. On the client, a
streaming RPC is an `async function*` returning `AsyncGenerator<T>` that yields
each event decoded with `fromJson(...)`; on the server, the handler returns a
`ReadableStream<MessageInitShape<typeof EventSchema>>` and the generated route
encodes each value with `toJson(create(EventSchema, value))` into an SSE
`data:` frame. Both directions go through protobuf-es's canonical JSON.

### Server `EmitDefaultValues` is not required for TS correctness

In protobuf-es transport mode the client decodes every response with
`fromJson(Schema, ..., { ignoreUnknownFields: true })`. protobuf-es's `fromJson`
fills in proto3 default values for any field the server omitted, so the
consumer always gets a fully-populated message even when the server does **not**
set the `EmitDefaultValues` flag. That flag is therefore not needed for
TypeScript correctness in this mode (unknown/extra fields on the wire are also
ignored rather than causing an error).

### Typed Result returns (`ts_error_handling=result`)

By default a generated es-mode client method returns the decoded message and
**throws** on failure (`ValidationError` / `ApiError`). Passing the additional
plugin option **`ts_error_handling=result`** (es-mode only — it fails loud
without `ts_runtime=protobuf-es`) switches **unary** methods to **return a typed
discriminated `Result`** instead of throwing, so error handling is checked by the
compiler:

```yaml
  - local: protoc-gen-ts-client
    out: ./generated
    opt:
      - paths=source_relative
      - ts_runtime=protobuf-es
      - ts_error_handling=result
    strategy: all
```

Each unary method returns `Promise<Result<T, ClientError>>`:

```ts
export type Result<T, E> =
  | { ok: true; data: T; error?: undefined }
  | { ok: false; data?: undefined; error: E };
```

`ok` is the discriminant, and `data` / `error` are always present (the inactive
one `undefined`), so you can narrow on `r.ok` **or** destructure
`const { data, error }`.

`ClientError` is the union of the two built-ins plus **every proto-defined
`*Error` message** in the schema. sebuf has no per-RPC error declaration (any
handler may return any `*Error`), so — mirroring the Python client — a shared
`result.ts` carries a structural registry and a `decodeError` that picks the
first `*Error` whose JSON marker keys are all present in the body (400-with-
`violations` → `ValidationError`; no match → `ApiError`). Proto errors are
protobuf-es messages, so they carry a `$typeName` discriminant:

```ts
const r = await getAccount({ id }, { baseURL: "https://api.example.com" });
if (!r.ok) {
  const e = r.error;
  if (e instanceof ValidationError) {
    console.log(e.violations);
  } else if (e instanceof ApiError) {
    console.log(e.statusCode, e.body);
  } else {
    switch (e.$typeName) {
      case "test.resulterr.NotFoundError":
        console.log(e.resourceType, e.resourceId);
        break;
      case "test.resulterr.LoginError":
        console.log(e.reason, e.retryAfterSeconds);
        break;
    }
  }
  return;
}
r.data; // Account (typed, non-null)
```

**Scope:** unary methods only. **SSE** methods keep their `AsyncGenerator<T>`
shape and still throw on a failed stream-open (the single-`Result` shape doesn't
fit a stream). Hand-rolled mode and the default (throwing) es-mode are
unchanged.

### Known limitations

- **`sebuf.http` JSON-mapping annotations are not honored (fail-loud).** As
  described in the wire-contract note above, es-mode emits canonical protojson
  and does not apply any JSON-mapping annotation. The generator rejects, at
  generation time, any RPC whose request/response message closure carries one —
  with an error such as `ts_runtime=protobuf-es: field User.status uses the
  enum_value JSON-mapping annotation (reachable from the response of
  UserService.GetUser), which es-mode cannot honor`. Porting the sebuf transform
  layer to TypeScript (so annotated protos round-trip byte-for-byte with the Go
  server) is tracked as future work; until then, use hand-rolled mode for those
  services.
- **Enum path AND query parameters are not yet supported.** protobuf-es enums
  are numeric, but path- and query-parameter values arrive as strings on the
  wire; safely bridging the two requires a name↔number conversion that is not
  yet implemented. This applies to enum path parameters, enum query parameters,
  and repeated enum query parameters. Rather than emit code that fails a
  downstream `tsc` (with a confusing `TS2307: Cannot find module './<proto>.js'`,
  because the hand-rolled enum type module is deliberately not emitted in
  es-mode), **the generator now fails loud at generation time** with a clear
  error, e.g. `ts_runtime=protobuf-es: enum query parameter "region" on
  Service.Method is not yet supported`. Both the client and server generators
  enforce this (see `checkNoEnumParamsES` in `internal/tsclientgen/generator.go`
  and `internal/tsservergen/generator.go`). **String, integer, boolean, and
  other scalar** path and query parameters work fine under
  `ts_runtime=protobuf-es`.
- **Prefer top-level messages for RPC input/output.** Nested-message local
  names are reconciled to protoc-gen-es's underscore form (`Outer_Inner` /
  `Outer_InnerSchema`, via `ESQualifiedName`), so nested messages used as RPC
  I/O do resolve — but this was verified against
  `@bufbuild/protoc-gen-es@2.12.1`; if you use a different protoc-gen-es version
  and hit a naming mismatch, promoting the message to top level is the safe
  workaround. The es golden typecheck tests (`es_typecheck_test.go` in both
  generators) run `tsc --noEmit` over the goldens against the real
  `@bufbuild/protobuf` package, so a naming divergence fails CI rather than a
  consumer build.
- **URL parameters do not route through the protobuf-es codec.** Only request
  bodies, response bodies, and SSE events go through
  `create`/`toJson`/`fromJson`. Path and query parameter values are derived
  directly from strings: on the client they are string-coerced into the URL
  (`encodeURIComponent(String(...))` / `URLSearchParams`); on the server they are
  read off the URL and coerced back to the field's protobuf-es type before being
  placed on the request message (numeric fields via `Number(...)`, 64-bit via
  `BigInt(...)`, booleans via `=== "true"`, strings verbatim). URL parameters
  have no protojson canonical form, so this coercion — rather than the codec — is
  by design; it means codec guarantees (e.g. int64 string/number normalization)
  do not apply to them.
- **Path parameters are not compile-time required.** `MessageInitShape<...>`
  makes every request field optional at the type level, so omitting a path
  parameter field compiles and produces a URL containing the string
  `"undefined"` at runtime. Hand-rolled mode's request interfaces mark such
  fields required; in es-mode this check is deferred to the server's 4xx.

### Roadmap — lifting the guard

es-mode ships as a **preview**: correct and safe for protos that use no
`sebuf.http` JSON-mapping annotation, and fail-loud for the rest. The plan to
make it a true drop-in replacement for annotated protos — matching what the Go
server does with `MarshalJSONSebuf` — is staged so the fail-loud guard can be
lifted **one annotation at a time**, each only after it is proven wire-correct.

1. **Conformance harness (prerequisite for everything below).** A cross-runtime
   test that asserts es-mode's on-the-wire bytes are **byte-equivalent** to a
   sebuf Go server for a given `(message, annotation)` case — encode *and*
   decode. The Go marshal pipeline already has per-annotation consistency tests
   (`internal/httpgen/*_consistency_test.go`) that pin the expected wire; those
   become the oracle. Start with captured golden wire fixtures (deterministic, no
   network), add a handful of live Go-server round-trips for the trickiest
   annotations. This harness is what lets us trust each transform and is the gate
   for removing an annotation from the guard.
2. **Transform layer (the TS analog of `MarshalJSONSebuf`).** The Go layer is not
   a bespoke serializer — it post-processes canonical protojson: `protojson`
   marshal → rewrite only the annotated JSON keys → emit. The same shape ports to
   TypeScript over `toJson`/`fromJson`. Because protobuf-es exposes the message
   `Schema` (`DescMessage`) at runtime, this can be a single hand-written runtime
   shim (`sebufToJson`/`sebufFromJson`) driven by a small per-field annotation
   table the plugin emits — not per-message codegen. Implement annotation by
   annotation; as each passes the conformance harness, drop it from
   `CheckESMessageAnnotations`.
3. **Suggested order** (cheapest / highest-value first): `int64_encoding`,
   `enum_value`, `enum_encoding`, `timestamp_format`, `bytes_encoding`,
   `nullable`, `empty_behavior`, then the structural ones — `flatten` /
   `flatten_prefix`, `unwrap`, and `oneof_config` flatten. Each lands with its
   own conformance case and a guard-lift in the same change.

Until an annotation reaches step 2 with green conformance, es-mode rejects it at
generation time — no silent wire divergence.

## See Also

- **[HTTP Generation Guide](./http-generation.md)** - Go server-side handler generation
- **[Validation Guide](./validation.md)** - Request validation
- **[Examples](./examples/)** - Working examples
