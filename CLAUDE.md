# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `sebuf`, a Go protobuf code generator plugin that creates helper functions for protobuf messages with oneof fields. The main binary is `protoc-gen-go-oneof-helper`, which generates convenience constructors for oneof field assignments.

## Architecture

The project follows a clean Go protoc plugin architecture with separated concerns:

- **cmd/protoc-gen-go-oneof-helper/main.go**: Minimal plugin entry point (protoc interface only)
- **internal/oneofhelper/**: Core generation logic and utilities  
- **internal/oneofhelper/**: Comprehensive test suite with golden file testing
- **scripts/**: Test automation scripts

### Core Components

1. **Plugin Entry Point** (`main()` in cmd/protoc-gen-go-oneof-helper/main.go:14): Minimal orchestration - reads protobuf CodeGeneratorRequest from stdin, delegates to oneofhelper package, writes response to stdout
2. **Code Generation** (`GenerateHelpers()` at internal/oneofhelper/generator.go:13): Iterates through protobuf files and generates helper functions for messages with oneofs
3. **Helper Generation** (`GenerateMessageHelpers()` at generator.go:30): Creates constructor functions for oneof fields, handling complex parameter mapping from nested message fields
4. **Type System** (`getFieldType()` at generator.go:118): Comprehensive protobuf-to-Go type mapping including maps, arrays, optionals, and message types

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
go build -o protoc-gen-go-oneof-helper ./cmd/protoc-gen-go-oneof-helper

# Format code
go fmt ./...
```

### Manual Testing
```bash
# Test plugin with sample proto file
protoc --plugin=protoc-gen-go-oneof-helper=./protoc-gen-go-oneof-helper \
       --go-helpers_out=. \
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