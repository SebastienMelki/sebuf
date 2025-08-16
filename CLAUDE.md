# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `sebuf`, a comprehensive Go protobuf toolkit for building HTTP APIs. It consists of three complementary protoc plugins that together enable modern, type-safe API development:

- **`protoc-gen-go-oneof-helper`**: Creates convenience constructors for protobuf oneof fields
- **`protoc-gen-go-http`**: Generates HTTP handlers, routing, and request/response binding
- **`protoc-gen-openapiv3`**: Creates comprehensive OpenAPI v3.1 specifications

The toolkit enables developers to build HTTP APIs directly from protobuf definitions without gRPC dependencies, targeting web and mobile API development.

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
2. **HTTP Handler Generator** (`internal/httpgen/generator.go:22`): Generates HTTP handlers, request binding, and routing configuration
3. **OpenAPI Generator** (`internal/openapiv3/generator.go:53`): Creates comprehensive OpenAPI v3.1 specifications from protobuf definitions
4. **HTTP Annotations** (`proto/sebuf/http/annotations.proto`): Custom protobuf extensions for HTTP configuration

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