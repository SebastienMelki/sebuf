# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `sebuf`, a Go protobuf code generator plugin that creates helper functions for protobuf messages with oneof fields. The main binary is `protoc_gen_go_helpers`, which generates convenience constructors for oneof field assignments.

## Architecture

The project follows a clean Go protoc plugin architecture with separated concerns:

- **cmd/protoc_gen_go_helpers/main.go**: Minimal plugin entry point (protoc interface only)
- **internal/generator/**: Core generation logic and utilities  
- **internal/**: Comprehensive test suite with golden file testing
- **scripts/**: Test automation scripts

### Core Components

1. **Plugin Entry Point** (`main()` in cmd/protoc_gen_go_helpers/main.go:14): Minimal orchestration - reads protobuf CodeGeneratorRequest from stdin, delegates to generator package, writes response to stdout
2. **Code Generation** (`GenerateHelpers()` at internal/generator/generator.go:12): Iterates through protobuf files and generates helper functions for messages with oneofs
3. **Helper Generation** (`GenerateOneofHelper()` at generator.go:35): Creates constructor functions for oneof fields, handling complex parameter mapping from nested message fields
4. **Type System** (`GetFieldType()` at generator.go:76): Comprehensive protobuf-to-Go type mapping including maps, arrays, optionals, and message types

### Generated Output Pattern

For each oneof field with a nested message, generates helpers like:
```go
// NewSimpleMessageEmail creates a new SimpleMessage with Email set
func NewSimpleMessageEmail(email string, password string) *SimpleMessage {
    return &SimpleMessage{
        AuthMethod: &SimpleMessage_Email{
            Email: &SimpleMessage_EmailAuth{
                Email:    email,
                Password: password,
            },
        },
    }
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
# Build the plugin binary
go build -o protoc_gen_go_helpers ./cmd/protoc_gen_go_helpers

# Format code
go fmt ./...
```

### Manual Testing
```bash
# Test plugin with sample proto file
protoc --plugin=protoc_gen_go_helpers=./protoc_gen_go_helpers \
       --go-helpers_out=. \
       --proto_path=internal/testdata/proto \
       internal/testdata/proto/simple_oneof.proto
```

## Testing Strategy

The project uses a comprehensive two-tier testing approach:

### Golden File Tests (Primary)
- **Exhaustive regression detection**: Catches ANY change in generated output down to single characters
- **Real protoc execution**: Tests actual plugin behavior, not mocked components
- **File locations**: internal/exhaustive_golden_test.go, internal/golden_test.go
- **Test data**: internal/testdata/proto/*.proto → internal/testdata/golden/*_helpers.pb.go

### Unit Tests (Secondary)
- **Function-level testing**: Tests individual functions like `lowerFirst()`, `getFieldType()`
- **Mocked components**: Uses protogen mocks for isolated testing
- **File locations**: internal/simple_test.go, internal/comprehensive_test.go

## Type System

The plugin handles comprehensive protobuf-to-Go type mapping in `getFieldType()` (main.go:132):

- **Scalar types**: string, bool, int32/64, uint32/64, float32/64, bytes
- **Complex types**: repeated fields (slices), map fields, optional fields (pointers)
- **Message types**: Nested messages with proper import handling via protogen.GeneratedFile
- **Enum types**: With fallback to int32

## Key Implementation Details

### Oneof Detection Logic
- Only generates helpers for oneof fields that contain message types (not scalar types)
- Recursively processes nested messages to find all oneofs (internal/generator/generator.go:25)
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
- **cmd/protoc_gen_go_helpers/**: Minimal plugin entry point (47 lines)
- **internal/generator/**: Core generation logic (all business logic moved here)
- **internal/**: Comprehensive test suite with golden file testing
- **scripts/run_tests.sh**: Advanced test runner with coverage analysis and reporting
- **testdata/**: Proto files and expected generated output for testing