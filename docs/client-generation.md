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
from each pass and cannot resolve references into one consistent module tree.
They also **require `paths=source_relative`**: the module tree mirrors the
proto source tree, and the generators fail loudly under any other path mode
(such as protoc's default `paths=import`) rather than scatter type and service
modules across different directories:

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
  via a relative specifier. A proto type whose emitted TS name collides with
  one of these helpers keeps its name in its own type module and is imported
  into service modules under a deterministic alias (e.g. `ApiError_1`)
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

When generating both a TS client and a TS server, give each its own `out:`
directory (as in the examples: `./client/generated` and `./server/generated`).
The type modules and `errors.ts` are byte-identical between the two
generators, but each writes its own per-package `index.ts` barrel re-exporting
its service module — pointing both generators at one directory would leave
only the last writer's barrel, silently dropping the other's re-exports.

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

### Oneofs in TypeScript

Both TypeScript generators render a protobuf `oneof` as a discriminated union that matches protojson exactly. The clients do no conversion — requests are `JSON.stringify`-ed and responses are cast with `as T` — so the generated types are the wire contract.

- **Un-annotated oneof (default)** — presence-discriminated union: exactly the set member's JSON key appears on the parent. Each arm carries one variant key with a non-optional payload and types every sibling `?: never`, with a final all-`never` arm for the unset oneof. A message becomes the intersection of a base interface and one union per oneof.
- **Annotated `oneof_config`, `flatten: false`** — the discriminator and the set variant's key both sit flat on the parent, with `?: never` guards on the sibling keys.
- **Annotated `oneof_config`, `flatten: true`** — the variant's child fields are spread onto the parent next to the discriminator; the variant key itself is not emitted.

```typescript
// message Event { string id = 1; oneof content { TextContent text = 2; int32 count = 3; } }
export type EventContent =
  | { text: TextContent; count?: never }
  | { count: number; text?: never }
  | { text?: never; count?: never };
export type Event = EventBase & EventContent;

// Construct exactly one variant:
const e: Event = { id: "1", text: { body: "hi" } };

// Narrow on presence, NOT truthiness (count can legitimately be 0):
if ("text" in e && e.text !== undefined) {
  console.log(e.text.body);
} else if ("count" in e && e.count !== undefined) {
  console.log(e.count); // narrowed to number, correct even when 0
}
```

**Caveats:**

1. **Breaking change.** Consumers written against the old shapes break: the previous un-annotated "flattened bag" of optional fields, and the `event.content` wrapper property for `flatten: false`, are both gone. In particular, setting two members of the same oneof is now a compile error rather than a silently-tolerated object.
2. **Narrow scalar variants by presence, not truthiness.** Use `"count" in e` or `e.count !== undefined` — a truthiness check (`if (e.count)`) misfires on the zero values `0`, `""`, and `false`, which are valid payloads.
3. **Exactly-one is enforced only at object-literal construction.** That is where TypeScript rejects setting more than one member; the TS types are therefore stricter than the Python and OpenAPI outputs for the same proto.
4. **`?: never` is a compile-time guard, not runtime exclusivity.** Under non-strict TypeScript, `{ text: ..., image: undefined }` still type-checks, and reads go through an unchecked `as T` cast — nothing validates at runtime that exactly one member is set.
5. **The unset state is modeled differently across outputs.** TypeScript represents "no member set" with the all-`never` arm of the union, whereas OpenAPI represents it via `oneOf` / a discriminator — so the unset case is not equally visible when comparing the two generated surfaces.

## See Also

- **[HTTP Generation Guide](./http-generation.md)** - Go server-side handler generation
- **[Validation Guide](./validation.md)** - Request validation
- **[Examples](./examples/)** - Working examples
