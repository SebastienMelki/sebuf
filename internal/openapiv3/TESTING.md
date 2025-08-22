# OpenAPI v3 Generator Testing Guide

This document describes the comprehensive testing strategy for the `protoc-gen-openapiv3` plugin.

## Testing Architecture

The OpenAPI v3 generator uses a multi-tier testing approach to ensure correctness, reliability, and regression detection:

### 1. Unit Tests (`*_test.go`)
- **Purpose**: Test individual components and functions in isolation
- **Scope**: Core generator functions, type mapping, schema building
- **Files**: 
  - `generator_test.go` - Core generator functionality
  - `types_test.go` - Type system and protobuf-to-OpenAPI mapping
  - `validation_test.go` - Validation rule processing
  - `http_annotations_test.go` - HTTP annotation parsing

### 2. Golden File Tests (`exhaustive_golden_test.go`)
- **Purpose**: Exhaustive regression detection via byte-for-byte comparison
- **Scope**: Complete plugin execution through protoc
- **Benefits**: 
  - Catches ANY change in generated output
  - Tests real protoc execution (not mocked)
  - Ensures consistency across formats (YAML/JSON)

### 3. Integration Tests (`integration_test.go`)
- **Purpose**: End-to-end plugin integration with protoc
- **Scope**: Plugin invocation, error handling, format options
- **Coverage**:
  - Multiple proto file scenarios
  - Format option validation
  - Error condition handling
  - Service generation verification

## Test Data Organization

```
internal/openapiv3/testdata/
├── proto/                          # Input proto files
│   ├── simple_service.proto        # Basic service with standard messages
│   ├── multiple_services.proto     # Multiple services in one file
│   ├── complex_types.proto         # Complex message types (optional fields)
│   ├── nested_messages.proto       # Nested message structures
│   ├── headers.proto               # Header validation configurations
│   ├── validation_constraints.proto # buf.validate integration
│   ├── http_annotations.proto      # HTTP method annotations
│   └── no_services.proto           # Edge case: no services defined
└── golden/                         # Expected output files
    ├── yaml/                       # YAML format golden files
    │   ├── SimpleService.openapi.yaml
    │   ├── UserService.openapi.yaml
    │   └── ...
    └── json/                       # JSON format golden files
        ├── SimpleService.openapi.json
        ├── UserService.openapi.json
        └── ...
```

## Running Tests

### All Tests
```bash
# Run complete test suite
go test -v ./internal/openapiv3/...

# Run with coverage analysis
go test -v -coverprofile=coverage.out ./internal/openapiv3/...
go tool cover -html=coverage.out
```

### Specific Test Categories

#### Unit Tests Only
```bash
# Fast unit tests
go test -v -run "^Test[^E]" ./internal/openapiv3/
```

#### Golden File Tests Only
```bash
# Comprehensive regression tests
go test -v -run TestExhaustiveGoldenFiles ./internal/openapiv3/
```

#### Integration Tests Only
```bash
# Plugin integration tests
go test -v -run TestPlugin ./internal/openapiv3/
```

### Golden File Management

#### Updating Golden Files
When intentional changes are made to the generator output:

```bash
# Update all golden files
UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles ./internal/openapiv3/

# Or use the generation script
./scripts/generate_openapi_golden_files.sh
```

#### Adding New Test Cases
1. Create new proto file in `testdata/proto/`
2. Add test case to `exhaustive_golden_test.go`
3. Generate golden files:
   ```bash
   UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles ./internal/openapiv3/
   ```
4. Verify generated files are correct
5. Commit both proto file and golden files

## Test Coverage Goals

- **Unit Tests**: >90% line coverage
- **Golden Files**: 100% of supported proto patterns
- **Integration**: All major plugin features and error paths
- **Regression**: Detect any change in output format

## Common Test Patterns

### Testing New Features

1. **Add Unit Test**: Test the feature logic in isolation
   ```go
   func TestNewFeature(t *testing.T) {
       // Test the specific function/method
   }
   ```

2. **Add Proto Test Case**: Create proto file exercising the feature
   ```protobuf
   service TestService {
       rpc TestMethod(Request) returns (Response) {
           // Feature-specific annotations
       };
   }
   ```

3. **Generate Golden File**: Let the generator create expected output
   ```bash
   UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles
   ```

4. **Verify Integration**: Ensure feature works end-to-end
   ```go
   func TestNewFeatureIntegration(t *testing.T) {
       // Test via protoc execution
   }
   ```

### Testing Error Conditions

```go
func TestErrorHandling(t *testing.T) {
    testCases := []struct {
        name        string
        input       string
        expectError bool
        errorMsg    string
    }{
        // Test cases for various error conditions
    }
    // ... test implementation
}
```

### Testing Format Variations

```go
func TestFormats(t *testing.T) {
    formats := []string{"yaml", "json"}
    for _, format := range formats {
        t.Run(format, func(t *testing.T) {
            // Test both YAML and JSON output
        })
    }
}
```

## Debugging Test Failures

### Golden File Mismatches
When golden file tests fail:

1. **Check diff output**: Test failure shows first differences
2. **Compare files**: Use generated `.generated` file
   ```bash
   diff testdata/golden/yaml/service.openapi.yaml testdata/golden/yaml/service.openapi.yaml.generated
   ```
3. **Verify changes**: Ensure changes are intentional
4. **Update if needed**: 
   ```bash
   UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles
   ```

### Integration Test Failures
When integration tests fail:

1. **Check protoc output**: Look at command execution details
2. **Verify plugin build**: Ensure plugin compiles correctly
3. **Test manually**: Run protoc command directly
   ```bash
   protoc --plugin=protoc-gen-openapiv3=./plugin \
          --openapiv3_out=./output \
          --proto_path=./proto \
          service.proto
   ```

## Test Performance

### Benchmark Tests
```go
func BenchmarkGeneration(b *testing.B) {
    // Benchmark critical paths
}
```

### Test Optimization
- **Parallel execution**: Use `t.Parallel()` where appropriate
- **Shared setup**: Reuse plugin builds across tests
- **Targeted tests**: Run specific test categories during development

## Continuous Integration

Tests run automatically on:
- **Pull requests**: Full test suite + coverage analysis
- **Main branch**: Full test suite + regression detection
- **Releases**: Extended test suite + golden file validation

### CI Configuration
- Minimum coverage threshold: 85%
- Golden file validation required
- Integration tests must pass
- No test skips allowed in CI

## Best Practices

1. **Test Independence**: Each test should be self-contained
2. **Clear Naming**: Test names should describe what they verify
3. **Comprehensive Coverage**: Test both success and failure paths
4. **Maintainable Assertions**: Use helper functions for complex validations
5. **Documentation**: Comment complex test logic
6. **Fast Execution**: Keep unit tests fast, use integration tests for E2E

## Troubleshooting

### Common Issues
- **Import path errors**: Verify proto_path settings
- **Plugin not found**: Check plugin build and path
- **Golden file encoding**: Ensure consistent line endings
- **Protoc version**: Use compatible protoc version

### Getting Help
- Check existing test patterns
- Review CLAUDE.md for project conventions
- Look at similar tests in other generators (oneofhelper, httpgen)
- Create minimal reproduction case