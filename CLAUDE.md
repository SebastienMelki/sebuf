# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `sebuf`, a comprehensive Go protobuf toolkit for building HTTP APIs. It consists of three complementary protoc plugins that together enable modern, type-safe API development:

- **`protoc-gen-go-oneof-helper`**: Creates convenience constructors for protobuf oneof fields
- **`protoc-gen-go-http`**: Generates HTTP handlers, routing, request/response binding, and automatic validation
- **`protoc-gen-openapiv3`**: Creates comprehensive OpenAPI v3.1 specifications

The toolkit enables developers to build HTTP APIs directly from protobuf definitions without gRPC dependencies, targeting web and mobile API development with built-in request validation.

## Architecture

The project follows a clean Go protoc plugin architecture with separated concerns across three main components:

### Plugin Structure
- **cmd/protoc-gen-go-oneof-helper/**: Oneof helper generator entry point
- **cmd/protoc-gen-go-http/**: HTTP handler generator entry point
- **cmd/protoc-gen-openapiv3/**: OpenAPI specification generator entry point
- **internal/oneofhelper/**: Oneof helper generation logic and tests
- **internal/httpgen/**: HTTP handler generation logic and annotations
- **internal/openapiv3/**: OpenAPI generation logic and type mapping
- **proto/sebuf/http/**: HTTP annotation definitions
- **scripts/**: Test automation and build scripts

### Core Components

1. **Oneof Helper Generator** (`internal/oneofhelper/generator.go:27`): Creates convenience constructors for oneof fields containing message types
2. **HTTP Handler Generator** (`internal/httpgen/generator.go:22`): Generates HTTP handlers, request binding, routing configuration, and automatic validation
3. **OpenAPI Generator** (`internal/openapiv3/generator.go:53`): Creates comprehensive OpenAPI v3.1 specifications from protobuf definitions
4. **HTTP Annotations** (`proto/sebuf/http/annotations.proto`): Custom protobuf extensions for HTTP configuration
5. **Validation Annotations** (`proto/sebuf/validate/validate.proto`): Alias for buf.validate enabling sebuf.validate annotations

### Generated Output Examples

**Oneof Helpers** - Convenience constructors:
```go
// NewLoginRequestEmail creates a new LoginRequest with Email set
func NewLoginRequestEmail(email string, password string) *LoginRequest {
    return &LoginRequest{
        AuthMethod: &LoginRequest_Email{
            Email: &LoginRequest_EmailAuth{
                Email:    email,
                Password: password,
            },
        },
    }
}
```

**HTTP Handlers** - Complete HTTP server infrastructure:
```go
// UserServiceServer is the server API for UserService
type UserServiceServer interface {
    CreateUser(context.Context, *CreateUserRequest) (*User, error)
}

// RegisterUserServiceServer registers HTTP handlers for UserService
func RegisterUserServiceServer(server UserServiceServer, opts ...ServerOption) error
```

**OpenAPI Specifications** - Comprehensive API documentation:
```yaml
openapi: 3.1.0
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

**Automatic Validation** - Built-in request validation:
```go
// Generated validation code automatically validates requests
func BindingMiddleware[Req any](next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // ... binding logic ...
    
    // Automatic validation happens here
    if msg, ok := any(toBind).(proto.Message); ok {
      if err := ValidateMessage(msg); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
    }
    
    // ... continue to handler ...
  })
}
```

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
go build -o protoc-gen-go-oneof-helper ./cmd/protoc-gen-go-oneof-helper
go build -o protoc-gen-go-http ./cmd/protoc-gen-go-http
go build -o protoc-gen-openapiv3 ./cmd/protoc-gen-openapiv3

# Format code
go fmt ./...
```

### Manual Testing
```bash
# Test all plugins with sample proto file
protoc --go_out=. --go_opt=module=github.com/SebastienMelki/sebuf \
       --go-oneof-helper_out=. \
       --go-http_out=. \
       --openapiv3_out=./docs \
       --proto_path=internal/oneofhelper/testdata/proto \
       internal/oneofhelper/testdata/proto/simple_oneof.proto

# Test specific plugin
protoc --go-oneof-helper_out=. \
       --proto_path=internal/oneofhelper/testdata/proto \
       internal/oneofhelper/testdata/proto/simple_oneof.proto
```

## Testing Strategy

The project uses a comprehensive two-tier testing approach:

### Golden File Tests (Primary)
- **Exhaustive regression detection**: Catches ANY change in generated output down to single characters
- **Real protoc execution**: Tests actual plugin behavior, not mocked components
- **File locations**: internal/oneofhelper/exhaustive_golden_test.go, internal/oneofhelper/golden_test.go
- **Test data**: internal/oneofhelper/testdata/proto/*.proto → internal/oneofhelper/testdata/golden/*_helpers.pb.go

### Unit Tests (Secondary)
- **Function-level testing**: Tests individual functions like `lowerFirst()`, `getFieldType()`
- **Mocked components**: Uses protogen mocks for isolated testing
- **File locations**: internal/oneofhelper/simple_test.go, internal/oneofhelper/comprehensive_test.go

## Validation System

The HTTP generator automatically includes request validation using protovalidate:

### sebuf.validate Annotations
- **Alias for buf.validate**: Use `(sebuf.validate.field)` instead of `(buf.validate.field)`
- **Full compatibility**: All buf.validate rules work identically
- **Automatic validation**: No configuration required - validation happens automatically
- **Performance optimized**: Validator instance is cached and reused

### Supported Validation Rules
```protobuf
message CreateUserRequest {
  // String validation
  string name = 1 [(sebuf.validate.field).string = {
    min_len: 2,
    max_len: 100
  }];
  
  // Email validation
  string email = 2 [(sebuf.validate.field).string.email = true];
  
  // UUID validation
  string id = 3 [(sebuf.validate.field).string.uuid = true];
  
  // Enum validation (in constraint)
  string status = 4 [(sebuf.validate.field).string = {
    in: ["active", "inactive", "pending"]
  }];
  
  // Numeric validation
  int32 age = 5 [(sebuf.validate.field).int32 = {
    gte: 18,
    lte: 120
  }];
}
```

### Error Handling
- **HTTP 400 responses**: Validation errors return Bad Request with error message
- **Detailed errors**: Full validation error details from protovalidate
- **Fail-fast**: Validation stops request processing immediately on failure

## Type System

The plugin handles comprehensive protobuf-to-Go type mapping in `getFieldType()` (generator.go:118):

- **Scalar types**: string, bool, int32/64, uint32/64, float32/64, bytes
- **Complex types**: repeated fields (slices), map fields, optional fields (pointers)
- **Message types**: Nested messages with proper import handling via protogen.GeneratedFile
- **Enum types**: With fallback to int32

## Key Implementation Details

### Oneof Detection Logic
- Only generates helpers for oneof fields that contain message types (not scalar types)
- Recursively processes nested messages to find all oneofs (internal/oneofhelper/generator.go:26)
- Uses protogen reflection to inspect field properties

### Parameter Generation  
- Flattens nested message fields into function parameters
- Converts protobuf field names to Go parameter names using `lowerFirst()` (generator.go:118)
- Maintains type safety through protogen's type system

### Import Management
- Uses protogen.GeneratedFile's automatic import handling
- Calls `g.QualifiedGoIdent()` for proper type references across packages

## Project Structure

The repository contains:
- **cmd/protoc-gen-go-oneof-helper/**: Minimal plugin entry point (47 lines)
- **internal/oneofhelper/**: Core generation logic and comprehensive test suite
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
- **[buf.validate](https://buf.build/bufbuild/protovalidate)** - The proto definitions that provide the validation annotations we alias as `sebuf.validate`.

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