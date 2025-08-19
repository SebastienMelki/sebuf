# OpenAPI v3 Generator Testing Strategy

## Overview

The OpenAPI v3 generator test suite provides comprehensive coverage through a two-tier testing approach that ensures both code correctness and output stability.

## Test Structure

### 1. Unit Tests
Located in `internal/openapiv3/*_test.go`

#### `types_test.go`
- **Purpose**: Tests type conversion logic
- **Coverage**:
  - Scalar field conversion (bool, int32/64, uint32/64, float, double, string, bytes)
  - Repeated field handling (arrays)
  - Map field conversion
  - Enum field processing
  - Optional field handling
  - Schema extraction utilities

#### `validation_test.go`
- **Purpose**: Tests buf.validate integration
- **Coverage**:
  - String constraints (min/max length, patterns, formats)
  - Numeric constraints (min/max, exclusive bounds)
  - Collection constraints (arrays, maps)
  - Required field detection
  - Constraint application to schemas
  - All validation rule types

#### `http_annotations_test.go`
- **Purpose**: Tests HTTP annotation processing
- **Coverage**:
  - Service HTTP configuration extraction
  - Method HTTP configuration extraction
  - Path building and combination
  - Header configuration (service & method level)
  - Header merging logic
  - Header to OpenAPI parameter conversion
  - Type mapping for headers

#### `generator_test.go`
- **Purpose**: Tests core generation logic
- **Coverage**:
  - Generator initialization
  - File processing
  - Message to schema conversion
  - Service to path conversion
  - Document rendering (YAML/JSON)
  - Required field handling
  - Header parameter generation

### 2. Golden File Tests
Located in `internal/openapiv3/golden_test.go`

#### Purpose
- Detect any regression in generated output
- Validate complete end-to-end generation
- Ensure consistent output across changes

#### Test Data Structure
```
internal/openapiv3/testdata/
├── proto/                  # Input proto files
│   ├── simple_types.proto  # Basic type testing
│   ├── validation.proto    # buf.validate scenarios
│   └── headers.proto       # Header parameter testing
└── golden/                 # Expected outputs
    ├── simple_types.yaml
    ├── simple_types.json
    ├── validation.yaml
    ├── validation.json
    ├── headers.yaml
    └── headers.json
```

## Test Scenarios

### Simple Types (`simple_types.proto`)
Tests fundamental protobuf-to-OpenAPI mapping:
- Scalar types (string, int32, bool, double, bytes)
- Optional fields (proto3 optional)
- Repeated fields (arrays)
- Map fields
- Enum types
- Nested messages
- Service definitions with HTTP annotations

### Validation (`validation.proto`)
Tests buf.validate constraint mapping:
- **String validation**: length, email, UUID, patterns, URI, hostname, IP formats
- **Numeric validation**: min/max, exclusive bounds, const, enum values
- **Collection validation**: min/max items, unique items, min/max properties
- **Required fields**: field-level requirements
- **Complex validation**: combined constraints, nested validated messages

### Headers (`headers.proto`)
Tests header parameter generation:
- Service-level headers (apply to all methods)
- Method-level headers (specific to method)
- Header override (method overrides service)
- Various header types (string, integer, boolean, number, array)
- Header formats (UUID, email, date, date-time)
- Optional vs required headers
- Deprecated headers

## Running Tests

### Run All Tests
```bash
# Run with coverage analysis (85% threshold)
./scripts/run_tests.sh

# Run without coverage (faster)
./scripts/run_tests.sh --fast

# Run with verbose output
./scripts/run_tests.sh --verbose
```

### Run Specific Tests
```bash
# Unit tests only
go test -v ./internal/openapiv3

# Golden file tests only
go test -v -run TestGoldenFiles ./internal/openapiv3

# Exhaustive tests (comprehensive scenarios)
EXHAUSTIVE_TEST=1 go test -v -run TestExhaustiveGoldenFiles ./internal/openapiv3
```

### Update Golden Files
```bash
# When output changes are intentional
UPDATE_GOLDEN=1 go test -run TestGoldenFiles ./internal/openapiv3
```

## Coverage Goals

Target: **85% code coverage**

Current coverage areas:
- ✅ Type conversion functions
- ✅ Validation constraint mapping
- ✅ HTTP annotation parsing
- ✅ Header parameter generation
- ✅ Document generation
- ✅ Schema building
- ✅ Path construction

## Testing Best Practices

### 1. Unit Test Guidelines
- Test individual functions in isolation
- Use mock objects for protogen types
- Focus on edge cases and error conditions
- Validate all constraint mappings

### 2. Golden File Guidelines
- Cover all major feature combinations
- Include both YAML and JSON outputs
- Test real protoc execution
- Validate complete document structure

### 3. Adding New Tests
When adding new features:
1. Add unit tests for new functions
2. Update test proto files with new scenarios
3. Generate new golden files
4. Verify output manually before committing

### 4. Debugging Failed Tests
- Check `*.generated` files for actual output
- Compare with golden files using diff tools
- Run with `--verbose` for detailed output
- Check protoc plugin build logs

## Integration with CI

The test suite integrates with the project's CI pipeline:
- Runs on every pull request
- Enforces 85% coverage threshold
- Validates golden files haven't changed
- Checks for race conditions

## Mock Helpers

The test suite includes comprehensive mock implementations:
- `mockFieldDescriptor`: Simulates protobuf field descriptors
- `mockMessageDescriptor`: Simulates message descriptors
- `mockServiceDescriptor`: Simulates service descriptors
- `mockMethodDescriptor`: Simulates method descriptors

These mocks allow testing without real protobuf compilation.

## Known Limitations

1. **buf.validate proto availability**: Tests attempt to find buf.validate protos automatically but may require manual configuration
2. **protoc availability**: Golden tests require protoc to be installed
3. **Cross-platform paths**: Some path handling may need adjustment on Windows

## Future Improvements

- [ ] Add performance benchmarks
- [ ] Test streaming RPC support
- [ ] Add mutation testing
- [ ] Test error handling scenarios
- [ ] Add fuzz testing for edge cases
- [ ] Test with complex real-world proto files