# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `sebuf`, a specialized Go protobuf toolkit for building HTTP APIs. It consists of five complementary protoc plugins that together enable modern, type-safe API development:

- **`protoc-gen-go-http`**: Generates HTTP handlers, routing, request/response binding, and automatic validation
- **`protoc-gen-go-client`**: Generates type-safe Go HTTP clients with functional options pattern
- **`protoc-gen-ts-client`**: Generates TypeScript HTTP clients with full type safety, header helpers, and error handling
- **`protoc-gen-ts-server`**: Generates TypeScript HTTP server handlers using the Web Fetch API (Request/Response), framework-agnostic
- **`protoc-gen-openapiv3`**: Creates comprehensive OpenAPI v3.1 specifications

The toolkit enables developers to build HTTP APIs directly from protobuf definitions without gRPC dependencies, targeting web and mobile API development with built-in request validation.

## Architecture

The project follows a clean Go protoc plugin architecture with separated concerns across two main components:

### Plugin Structure
- **cmd/protoc-gen-go-http/**: HTTP handler generator entry point
- **cmd/protoc-gen-go-client/**: Go HTTP client generator entry point
- **cmd/protoc-gen-ts-client/**: TypeScript HTTP client generator entry point
- **cmd/protoc-gen-ts-server/**: TypeScript HTTP server generator entry point
- **cmd/protoc-gen-openapiv3/**: OpenAPI specification generator entry point
- **internal/httpgen/**: HTTP handler generation logic, annotations, and header validation middleware
- **internal/clientgen/**: Go HTTP client generation logic and annotations
- **internal/tscommon/**: Shared TypeScript type mapping and generation (used by ts-client and ts-server)
- **internal/tsclientgen/**: TypeScript HTTP client generation logic
- **internal/tsservergen/**: TypeScript HTTP server generation logic, header validation, route creation
- **internal/openapiv3/**: OpenAPI generation logic, type mapping, and header parameter generation
- **proto/sebuf/http/**: HTTP annotation definitions including headers.proto for header validation
- **scripts/**: Test automation and build scripts

### Core Components

1. **HTTP Handler Generator** (`internal/httpgen/generator.go:22`): Generates HTTP handlers, request binding, routing configuration, automatic body validation, and header validation middleware
2. **Go HTTP Client Generator** (`internal/clientgen/generator.go:13`): Generates type-safe Go HTTP clients with functional options pattern, automatic request/response marshaling, and error handling
3. **TypeScript HTTP Client Generator** (`internal/tsclientgen/generator.go`): Generates TypeScript HTTP clients with typed interfaces, service/method header helpers, query parameter encoding, path parameter substitution, and structured error handling (ValidationError/ApiError)
4. **TypeScript HTTP Server Generator** (`internal/tsservergen/generator.go`): Generates framework-agnostic TypeScript HTTP server handlers using the Web Fetch API (`Request` → `Promise<Response>`), with route descriptors, header validation, query/body parsing, and error handling
5. **OpenAPI Generator** (`internal/openapiv3/generator.go:53`): Creates comprehensive OpenAPI v3.1 specifications from protobuf definitions with full header parameter support, generating one file per service for better organization
6. **Shared TypeScript Types** (`internal/tscommon/`): Shared TypeScript type mapping, interface generation, error types, and proto-defined error message collection (messages ending with "Error") used by both ts-client and ts-server generators
7. **HTTP Annotations** (`proto/sebuf/http/annotations.proto`): Custom protobuf extensions for HTTP configuration
5. **Header Validation** (`proto/sebuf/http/headers.proto`): Protobuf definitions for service and method-level header validation
6. **Validation System**: Automatic request body validation via buf.validate/protovalidate and header validation middleware

### Generated Output Examples

**HTTP Handlers** - Complete HTTP server infrastructure:
```go
// UserServiceServer is the server API for UserService
type UserServiceServer interface {
    CreateUser(context.Context, *CreateUserRequest) (*User, error)
}

// RegisterUserServiceServer registers HTTP handlers for UserService
func RegisterUserServiceServer(server UserServiceServer, opts ...ServerOption) error
```

**HTTP Clients** - Type-safe HTTP client with functional options:
```go
// UserServiceClient is the client API for UserService
type UserServiceClient interface {
    CreateUser(ctx context.Context, req *CreateUserRequest, opts ...UserServiceCallOption) (*User, error)
}

// Create a client with options
client := NewUserServiceClient(
    "http://localhost:8080",
    WithUserServiceHTTPClient(&http.Client{Timeout: 30 * time.Second}),
    WithUserServiceAPIKey("your-api-key"),  // From service_headers annotation
)

// Make requests with per-call options
user, err := client.CreateUser(ctx, req,
    WithUserServiceHeader("X-Request-ID", "req-123"),
    WithUserServiceCallContentType(ContentTypeProto),
)
```

**TypeScript HTTP Clients** - Type-safe client with header helpers:
```typescript
// Generated client with typed interfaces
const client = new UserServiceClient("http://localhost:8080", {
  apiKey: "your-api-key",  // From service_headers annotation
});

// Make requests with per-call options
const user = await client.createUser(
  { name: "John", email: "john@example.com" },
  { requestId: "req-123" },  // From method_headers annotation
);

// Error handling with typed errors
try {
  await client.getUser({ id: "not-found" });
} catch (e) {
  if (e instanceof ValidationError) {
    console.log(e.violations);  // Field-level validation errors
  } else if (e instanceof ApiError) {
    // Parse proto-defined custom errors using generated interfaces
    const body = JSON.parse(e.body) as NotFoundError;
    console.log(body.resourceType, body.resourceId);
  }
}
```

**TypeScript HTTP Servers** - Framework-agnostic server with Web Fetch API:
```typescript
// Generated handler interface (like Go's XxxServer)
export interface UserServiceHandler {
  createUser(ctx: ServerContext, req: CreateUserRequest): Promise<User>;
  getUser(ctx: ServerContext, req: GetUserRequest): Promise<User>;
}

// Route creation — wire into any framework (Express, Hono, Bun, etc.)
const routes: RouteDescriptor[] = createUserServiceRoutes(handler, {
  onError: (err, req) => new Response("Internal error", { status: 500 }),
  validateRequest: (method, body) => myValidator(method, body),
});

// Each route: { method: "POST", path: "/api/v1/users", handler: (req) => Response }
// Handlers do: validate headers → parse body/query → optional validation → call handler → JSON response

// Works natively in Node 18+, Deno, Bun, Cloudflare Workers
// Example with Bun:
Bun.serve({
  fetch(req) {
    const url = new URL(req.url);
    for (const route of routes) {
      if (req.method === route.method && matchPath(url.pathname, route.path)) {
        return route.handler(req);
      }
    }
    return new Response("Not Found", { status: 404 });
  },
});
```

**OpenAPI Specifications** - Comprehensive API documentation (one file per service):
```yaml
# UserService.openapi.yaml
openapi: 3.1.0
info:
  title: UserService API
  version: 1.0.0
paths:
  /api/v1/users:
    post:
      summary: CreateUser
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
```

**Automatic Validation** - Built-in request and header validation:
```go
// Generated validation code automatically validates requests
func BindingMiddleware[Req any](next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // ... binding logic ...
    
    // Automatic body validation happens here
    if msg, ok := any(toBind).(proto.Message); ok {
      if err := ValidateMessage(msg); err != nil {
        writeValidationError(w, r, err)
        return
      }
    }
    
    // ... continue to handler ...
  })
}

// Generated header validation middleware
func HeaderValidationMiddleware(requiredHeaders []HeaderConfig) func(http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      // Validate required headers
      if validationErr := validateHeaders(r, serviceHeaders, methodHeaders); validationErr != nil {
        writeValidationErrorResponse(w, r, validationErr)
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}
```

**Header Annotations** - Service and method-level header configuration:
```protobuf
service UserService {
  option (sebuf.http.service_headers) = {
    required_headers: [
      {
        name: "X-API-Key"
        description: "API authentication key"
        type: "string"
        required: true
        format: "uuid"
      }
    ]
  };
  
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (sebuf.http.method_headers) = {
      required_headers: [
        {
          name: "X-Request-ID"
          type: "string"
          format: "uuid"
          required: true
        }
      ]
    };
  }
}
```

**Unwrap Annotation** - For map values that should serialize as arrays in JSON, or for root-level unwrapping:

**Map-value unwrap** - Collapses wrapper when used as map value:
```protobuf
// Wrapper message with unwrap annotation
message BarList {
  repeated Bar bars = 1 [(sebuf.http.unwrap) = true];
}

message Response {
  // JSON: {"bars": {"AAPL": [...], "GOOG": [...]}}
  // Without unwrap: {"bars": {"AAPL": {"bars": [...]}, ...}}
  map<string, BarList> bars = 1;
}
```

**Root-level unwrap** - Message with single field unwraps at root:
```protobuf
// Root map unwrap - entire message becomes the map
message UsersResponse {
  map<string, User> users = 1 [(sebuf.http.unwrap) = true];
}
// JSON: {"user-1": {...}, "user-2": {...}}
// Without unwrap: {"users": {"user-1": {...}, "user-2": {...}}}

// Root repeated unwrap - entire message becomes the array
message UserList {
  repeated User users = 1 [(sebuf.http.unwrap) = true];
}
// JSON: [{...}, {...}]
// Without unwrap: {"users": [{...}, {...}]}

// Combined unwrap - root map + value unwrap for clean map-of-arrays
message BarsResponse {
  map<string, BarList> data = 1 [(sebuf.http.unwrap) = true];
}
// JSON: {"AAPL": [...], "GOOG": [...]}
// Without unwrap: {"data": {"AAPL": {"bars": [...]}, ...}}
```

**JSON Mapping Annotations** - Control how protobuf fields serialize to JSON across all generators:

**int64_encoding** - Controls int64/uint64 JSON encoding (ext 50010):
```protobuf
message Order {
  // Serializes as JSON number: 12345 (precision warning for values > 2^53)
  int64 amount = 1 [(sebuf.http.int64_encoding) = INT64_ENCODING_NUMBER];
  // Serializes as JSON string: "12345" (default, safe for JavaScript)
  uint64 id = 2 [(sebuf.http.int64_encoding) = INT64_ENCODING_STRING];
}
```

**enum_encoding / enum_value** - Controls enum JSON encoding (ext 50011, 50012):
```protobuf
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1 [(sebuf.http.enum_value) = "active"];
  STATUS_INACTIVE = 2 [(sebuf.http.enum_value) = "inactive"];
}

message User {
  // With enum_value: serializes as "active" instead of "STATUS_ACTIVE"
  Status status = 1 [(sebuf.http.enum_encoding) = ENUM_ENCODING_STRING];
  // Serializes as number: 1
  Status role = 2 [(sebuf.http.enum_encoding) = ENUM_ENCODING_NUMBER];
}
```

**nullable** - Explicit null semantics for primitive fields (ext 50013):
```protobuf
message Profile {
  // Three states: absent (omitted), null, or "value"
  // Requires proto3 optional keyword
  optional string bio = 1 [(sebuf.http.nullable) = true];
}
// Set: {"bio": "hello"}  |  Null: {"bio": null}  |  Absent: {}
```

**empty_behavior** - Controls empty message field serialization (ext 50014):
```protobuf
message Response {
  Metadata meta = 1 [(sebuf.http.empty_behavior) = EMPTY_BEHAVIOR_PRESERVE]; // {}
  Metadata audit = 2 [(sebuf.http.empty_behavior) = EMPTY_BEHAVIOR_NULL];    // null
  Metadata debug = 3 [(sebuf.http.empty_behavior) = EMPTY_BEHAVIOR_OMIT];    // omitted
}
```

**timestamp_format** - Controls google.protobuf.Timestamp serialization (ext 50015):
```protobuf
message Event {
  google.protobuf.Timestamp created_at = 1; // Default: "2024-01-15T09:30:00Z"
  google.protobuf.Timestamp unix_ts = 2 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS]; // 1705312200
  google.protobuf.Timestamp unix_ms = 3 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_MILLIS];  // 1705312200000
  google.protobuf.Timestamp date = 4 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_DATE];             // "2024-01-15"
}
```

**bytes_encoding** - Controls bytes field serialization (ext 50016):
```protobuf
message Document {
  bytes data = 1;  // Default: standard base64 "SGVsbG8="
  bytes hash = 2 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_HEX];          // "48656c6c6f"
  bytes token = 3 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_BASE64URL];    // URL-safe base64
  bytes raw = 4 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_BASE64_RAW];     // No padding
}
```

**oneof_config / oneof_value** - Discriminated unions for oneof fields (ext 50017, 50018):
```protobuf
message Event {
  string id = 1;
  oneof payload {
    option (sebuf.http.oneof_config) = {
      discriminator: "type"
      flatten: true
    };
    TextPayload text = 2 [(sebuf.http.oneof_value) = "text"];
    ImagePayload image = 3 [(sebuf.http.oneof_value) = "image"];
  }
}
// Flattened: {"id": "1", "type": "text", "body": "hello"}
// Not flattened: {"id": "1", "type": "text", "text": {"body": "hello"}}
```

**flatten / flatten_prefix** - Promote nested message fields to parent (ext 50019, 50020):
```protobuf
message Order {
  string id = 1;
  Address billing = 2 [
    (sebuf.http.flatten) = true,
    (sebuf.http.flatten_prefix) = "billing_"
  ];
  Address shipping = 3 [
    (sebuf.http.flatten) = true,
    (sebuf.http.flatten_prefix) = "shipping_"
  ];
}
// JSON: {"id": "1", "billing_street": "123 Main", "shipping_street": "456 Oak"}
// Without flatten: {"id": "1", "billing": {"street": "123 Main"}, "shipping": {"street": "456 Oak"}}
```

### Annotation Extension Number Registry

All custom annotations live in `proto/sebuf/http/annotations.proto`:

| Ext # | Name | Target | Purpose |
|-------|------|--------|---------|
| 50003 | config | MethodOptions | HTTP path and method |
| 50004 | service_config | ServiceOptions | Service base path |
| 50007 | field_examples | FieldOptions | Example values for docs |
| 50008 | query | FieldOptions | Query parameter config |
| 50009 | unwrap | FieldOptions | Map value / root unwrapping |
| 50010 | int64_encoding | FieldOptions | int64/uint64 JSON encoding |
| 50011 | enum_encoding | FieldOptions | Enum JSON encoding |
| 50012 | enum_value | EnumValueOptions | Custom enum value string |
| 50013 | nullable | FieldOptions | Nullable primitive fields |
| 50014 | empty_behavior | FieldOptions | Empty message handling |
| 50015 | timestamp_format | FieldOptions | Timestamp JSON format |
| 50016 | bytes_encoding | FieldOptions | Bytes JSON encoding |
| 50017 | oneof_config | OneofOptions | Discriminated union config |
| 50018 | oneof_value | FieldOptions | Custom discriminator value |
| 50019 | flatten | FieldOptions | Nested message flattening |
| 50020 | flatten_prefix | FieldOptions | Prefix for flattened fields |

## Development Commands

### Testing
```bash
# Run all tests with coverage analysis (85% threshold)
./scripts/run_tests.sh

# Run tests without coverage (faster)
./scripts/run_tests.sh --fast

# Run with verbose output
./scripts/run_tests.sh --verbose

# Update golden files after intentional changes
UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles

# Run specific test categories
go test -v -run TestLowerFirst              # Unit tests
go test -v -run TestExhaustiveGoldenFiles   # Golden file tests
```

### Building
```bash
# Build all plugin binaries
make build

# Build individual plugins
go build -o protoc-gen-go-http ./cmd/protoc-gen-go-http
go build -o protoc-gen-ts-client ./cmd/protoc-gen-ts-client
go build -o protoc-gen-openapiv3 ./cmd/protoc-gen-openapiv3

# Format code
go fmt ./...
```

### Manual Testing
```bash
# Test plugins with sample proto file
protoc --go_out=. --go_opt=module=github.com/SebastienMelki/sebuf \
       --go-http_out=. \
       --openapiv3_out=./docs \
       --proto_path=examples/simple-api/proto \
       examples/simple-api/proto/services/user_service.proto
```

## Testing Strategy

The project uses a comprehensive two-tier testing approach:

### Golden File Tests (Primary)
- **Exhaustive regression detection**: Catches ANY change in generated output down to single characters
- **Real protoc execution**: Tests actual plugin behavior, not mocked components
- **File locations**: internal/openapiv3/exhaustive_golden_test.go, internal/tsclientgen/golden_test.go
- **Test data**: internal/openapiv3/testdata/ for OpenAPI, internal/tsclientgen/testdata/ for TypeScript client

### Unit Tests (Secondary)
- **Function-level testing**: Tests individual functions for HTTP, OpenAPI, and TypeScript client generators
- **Mocked components**: Uses protogen mocks for isolated testing
- **File locations**: internal/httpgen/, internal/openapiv3/, and internal/tsclientgen/ test files
- **Unwrap tests**: internal/httpgen/unwrap_test.go for map value unwrapping

## Validation System

The HTTP generator automatically includes comprehensive validation for both request bodies and headers:

### Request Body Validation (buf.validate Integration)
- **Direct buf.validate support**: Use standard `(buf.validate.field)` annotations
- **Full protovalidate compatibility**: All buf.validate rules work identically
- **Automatic validation**: No configuration required - validation happens automatically
- **Performance optimized**: Validator instance is cached and reused

### Header Validation
- **Service-level headers**: Applied to all RPCs in a service via `(sebuf.http.service_headers)`
- **Method-level headers**: Applied to specific RPCs via `(sebuf.http.method_headers)`
- **Type validation**: Support for string, integer, number, boolean, and array types
- **Format validation**: Built-in validators for UUID, email, date-time, date, time formats
- **Required headers**: Automatic HTTP 400 responses for missing required headers
- **Header merging**: Method headers override service headers with the same name

### Supported Validation Rules

**Request Body Validation:**
```protobuf
message CreateUserRequest {
  // String validation
  string name = 1 [(buf.validate.field).string = {
    min_len: 2,
    max_len: 100
  }];
  
  // Email validation
  string email = 2 [(buf.validate.field).string.email = true];
  
  // UUID validation
  string id = 3 [(buf.validate.field).string.uuid = true];
  
  // Enum validation (in constraint)
  string status = 4 [(buf.validate.field).string = {
    in: ["active", "inactive", "pending"]
  }];
  
  // Numeric validation
  int32 age = 5 [(buf.validate.field).int32 = {
    gte: 18,
    lte: 120
  }];
}
```

**Header Validation:**
```protobuf
service UserService {
  option (sebuf.http.service_headers) = {
    required_headers: [
      {
        name: "X-API-Key"
        description: "API authentication key"
        type: "string"
        required: true
        format: "uuid"
        example: "123e4567-e89b-12d3-a456-426614174000"
      },
      {
        name: "X-Tenant-ID"
        type: "integer"
        required: true
      }
    ]
  };
}
```

### Error Handling
- **Structured Error Responses**: All errors use protobuf messages for consistent API responses
- **Automatic Go Error Interface**: Any protobuf message ending with "Error" automatically implements Go's error interface for `errors.As()` and `errors.Is()` support
- **Automatic TypeScript Error Interfaces**: Both TS generators (`protoc-gen-ts-client`, `protoc-gen-ts-server`) generate TypeScript interfaces for proto messages ending with "Error", enabling type-safe custom error handling across server and client
- **Proto Message Error Preservation**: Custom proto error messages returned from handlers are serialized directly, preserving their structure (not wrapped in a generic Error message)
- **Validation Errors (HTTP 400)**: ValidationError with field-level violations for body and header validation failures
- **Handler Errors (HTTP 500)**: Error messages for service implementation failures with custom messages
- **Content-Type Aware**: Error responses serialized as JSON or protobuf based on request Content-Type
- **Client-side Error Handling**: Error types can be unmarshaled from HTTP responses and used with standard Go error patterns
- **Detailed validation errors**: Full validation error details from protovalidate for body validation
- **Header validation errors**: Clear messages indicating which header failed validation and why
- **Fail-fast**: Validation stops request processing immediately on failure (headers validated before body)

**Custom Proto Error Example:**
```protobuf
// Define custom error messages — works across Go, TS server, and TS client
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

```go
// Go: Return it from your handler - it will be serialized directly
func (s *Server) GetUser(ctx context.Context, req *GetUserRequest) (*User, error) {
    user, err := s.db.FindUser(req.Id)
    if err != nil {
        return nil, &NotFoundError{
            ResourceType: "user",
            ResourceId:   req.Id,
        }
    }
    return user, nil
}
// Response: {"resourceType":"user","resourceId":"123"}
// NOT: {"message":"{\"resourceType\":\"user\",\"resourceId\":\"123\"}"}
```

```typescript
// TS Server: implement generated interface, serialize in onError hook
class NotFoundError extends Error implements NotFoundErrorType {
  resourceType: string;
  resourceId: string;
  // ... constructor
}

// TS Client: parse ApiError.body using generated interface
const body = JSON.parse(e.body) as NotFoundError;
console.log(body.resourceType, body.resourceId);
```

## Type System

The plugin handles comprehensive protobuf-to-Go type mapping in `getFieldType()` (generator.go:118):

- **Scalar types**: string, bool, int32/64, uint32/64, float32/64, bytes
- **Complex types**: repeated fields (slices), map fields, optional fields (pointers)
- **Message types**: Nested messages with proper import handling via protogen.GeneratedFile
- **Enum types**: With fallback to int32

## Key Implementation Details

### HTTP Handler Generation
- Generates complete HTTP handlers with automatic request/response binding
- Implements comprehensive validation for both headers and request bodies
- Uses protogen reflection to generate type-safe handlers

### OpenAPI Generation
- Creates comprehensive OpenAPI v3.1 specifications
- Supports header parameter generation and validation rules
- Generates one file per service for better organization

### Import Management
- Uses protogen.GeneratedFile's automatic import handling
- Calls `g.QualifiedGoIdent()` for proper type references across packages

## Project Structure

The repository contains:
- **cmd/protoc-gen-go-http/**: HTTP handler plugin entry point
- **cmd/protoc-gen-go-client/**: Go HTTP client plugin entry point
- **cmd/protoc-gen-ts-client/**: TypeScript HTTP client plugin entry point
- **cmd/protoc-gen-ts-server/**: TypeScript HTTP server plugin entry point
- **cmd/protoc-gen-openapiv3/**: OpenAPI generation plugin entry point
- **internal/annotations/**: Shared annotation parsing used by all 5 generators (unwrap, query params, headers, JSON mapping)
- **internal/httpgen/**: HTTP handler generation logic and tests
- **internal/clientgen/**: Go HTTP client generation logic and tests
- **internal/tscommon/**: Shared TypeScript type mapping and generation (interfaces, enums, error types)
- **internal/tsclientgen/**: TypeScript HTTP client generation logic and tests
- **internal/tsservergen/**: TypeScript HTTP server generation logic and tests
- **internal/openapiv3/**: OpenAPI generation logic and comprehensive test suite
- **examples/ts-client-demo/**: End-to-end TypeScript client example with NoteService CRUD API
- **scripts/run_tests.sh**: Advanced test runner with coverage analysis and reporting

## Acknowledgments & Ecosystem

sebuf stands on the shoulders of giants. We build upon and integrate with an incredible ecosystem of tools and libraries:

### Core Foundation
- **[Protocol Buffers](https://protobuf.dev/)** by Google - The foundation that makes everything possible. Proto3 syntax, rich type system, and cross-language compatibility.
- **[protoc](https://grpc.io/docs/protoc-installation/)** - The official Protocol Buffer compiler that powers our plugin architecture.
- **[protogen](https://pkg.go.dev/google.golang.org/protobuf/compiler/protogen)** - Go's official protoc plugin framework that provides the foundation for all our generators.

### Validation Ecosystem  
- **[protovalidate](https://github.com/bufbuild/protovalidate)** by Buf - The modern validation framework that powers our automatic request validation. Built on CEL for flexibility and performance.
- **[Common Expression Language (CEL)](https://github.com/google/cel-go)** by Google - The expression language that enables powerful custom validation rules in protovalidate.
- **[buf.validate](https://buf.build/bufbuild/protovalidate)** - The proto definitions that provide the validation annotations used directly in sebuf (e.g., `buf.validate.field`).

### API Documentation
- **[OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0)** - The industry standard for REST API documentation that our OpenAPI generator targets.
- **[JSON Schema](https://json-schema.org/)** - The schema definition language that OpenAPI 3.1 uses and that we generate for protobuf messages.

### Development Tooling
- **[Buf CLI](https://buf.build/)** - The modern protobuf build system that replaces protoc for dependency management and code generation.
- **[Go Modules](https://go.dev/blog/using-go-modules)** - Go's dependency management system that ensures reproducible builds.

### HTTP & JSON Standards
- **[net/http](https://pkg.go.dev/net/http)** - Go's standard HTTP library that provides the foundation for our generated HTTP handlers.
- **[encoding/json](https://pkg.go.dev/encoding/json)** - Go's standard JSON library for request/response serialization.
- **[protojson](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson)** - Google's canonical JSON mapping for Protocol Buffers.

### Testing & Quality
- **[Golden File Testing](https://en.wikipedia.org/wiki/Characterization_test)** - The testing pattern we use for regression detection in code generation.
- **[Go Testing](https://pkg.go.dev/testing)** - Go's built-in testing framework that powers our comprehensive test suite.

This ecosystem approach means:
- **Standards compliance**: We follow established protocols and specifications
- **Interoperability**: Generated APIs work with existing tools and frameworks  
- **Community support**: Leverage documentation, tools, and knowledge from these mature projects
- **Future-proofing**: Built on stable, widely-adopted technologies

We're grateful to all the maintainers and contributors of these projects that make sebuf possible.