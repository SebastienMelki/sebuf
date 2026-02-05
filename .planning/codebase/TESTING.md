# Testing Patterns

**Analysis Date:** 2026-02-05

## Test Framework

**Runner:**
- Go's built-in `testing` package (standard library)
- Test execution: `go test ./...` from project root
- Test discovery: `*_test.go` files in same directory as source

**Assertion Library:**
- No external assertion library (e.g., testify/assert) used
- Manual assertions with if/else comparisons
- Test reporting: `t.Errorf()`, `t.Error()`, `t.Fatal()`, `t.Fatalf()`

**Run Commands:**
```bash
make test                # Run all tests with coverage analysis (85% threshold)
make test-fast           # Run tests without coverage (faster, uses cache)
./scripts/run_tests.sh   # Run with advanced coverage reporting
./scripts/run_tests.sh --verbose    # Verbose test output
./scripts/run_tests.sh --fast       # Fast mode without coverage
UPDATE_GOLDEN=1 go test -run TestHTTPGenGoldenFiles    # Update golden files
go test -v -run TestLowerFirst      # Run specific unit tests
```

**Coverage:**
- Required threshold: 85% for packages
- Coverage reports generated in `coverage/` directory:
  - `coverage.out`: Coverage profile
  - `coverage.html`: Interactive HTML coverage report
  - `coverage.json`: Machine-readable JSON report
  - `coverage-badge.svg`: Coverage badge for README
- Coverage mode: Runs with `-race` flag for race condition detection
- Fast mode: Skips coverage analysis entirely for faster feedback loops

## Test File Organization

**Location:**
- Co-located with source files in same directory
- Test files suffix: `_test.go`
- Golden file tests: `*_golden_test.go` (regression tests)

**Naming:**
- Test functions: `Test<FunctionName>` (e.g., `TestLowerFirst()`, `TestCamelToSnake()`)
- Golden file tests: `TestHTTPGenGoldenFiles`, `TestExhaustiveGoldenFiles`, `TestTSClientGoldenFiles`
- Helper functions in tests: Any function prefixed with lowercase (e.g., `generateTestFiles()`, `reportFirstDifference()`)

**Structure:**
```
internal/httpgen/
├── generator.go
├── generator_test.go          # Unit tests
├── golden_test.go             # Golden file regression tests
├── validation_test.go         # Validation-specific tests
├── annotations_test.go        # Annotation parsing tests
├── error_handler_test.go      # Error handling tests
├── unwrap_test.go            # Unwrap feature tests
└── testdata/
    ├── proto/
    │   ├── http_verbs_comprehensive.proto
    │   ├── query_params.proto
    │   ├── backward_compat.proto
    │   └── unwrap.proto
    └── golden/
        ├── http_verbs_comprehensive_http.pb.go
        ├── http_verbs_comprehensive_http_binding.pb.go
        ├── query_params_http.pb.go
        └── [... golden output files ...]
```

## Test Structure

**Suite Organization:**

Table-driven test pattern:
```go
func TestLowerFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Normal cases
		{"PascalCase to camelCase", "CreateUser", "createUser"},
		{"single word", "User", "user"},
		{"already lowercase", "create", "create"},

		// Edge cases
		{"empty string", "", ""},
		{"single uppercase char", "A", "a"},
		{"all uppercase", "ABC", "aBC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lowerFirst(tt.input)
			if result != tt.expected {
				t.Errorf("lowerFirst(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
```

**Patterns:**

1. **Setup Pattern (Implicit):**
   - No explicit setup/teardown for unit tests
   - Setup via test struct fields (table-driven tests)
   - Test directory creation via `t.TempDir()` for file operations

2. **Teardown Pattern:**
   - `t.TempDir()` automatically cleans up after test
   - Cleanup via `defer` for file handle closure
   - Explicit `defer os.Remove(pluginPath)` for built artifacts

3. **Assertion Pattern:**
   - Comparison: `if result != expected { t.Errorf(...) }`
   - Nil checks: `if gen == nil { t.Fatal("NewGenerator returned nil") }`
   - Error checks: `if err != nil { t.Fatalf("Failed: %v", err) }`
   - String comparisons: `if !strings.Contains(result, expectedPart) { t.Errorf(...) }`

## Mocking

**Framework:**
- Protocol Buffers code generation reflection (protogen package)
- No external mocking library (e.g., testify/mock, GoMock)

**Patterns:**

Manual test stub for protogen types:
```go
// internal/httpgen/mock_generator.go
// Contains manual mock implementations of protogen interfaces
// Used for testing generator functions with custom field/service configurations
```

Real protoc execution in golden file tests:
```go
// Build the plugin binary for testing
buildCmd := exec.Command("make", "build")
buildCmd.Dir = projectRoot
if buildErr := buildCmd.Run(); buildErr != nil {
    t.Fatalf("Failed to build plugin: %v", buildErr)
}

// Run protoc with the built plugin
cmd := exec.Command("protoc",
    "--plugin=protoc-gen-go-http="+pluginPath,
    "--go_out="+tempDir,
    "--go-http_out="+tempDir,
    "--proto_path="+protoDir,
    protoFile,
)
```

**What to Mock:**
- Use real protogen types when testing code generation (more reliable than mocks)
- Only mock when testing error conditions or behavior with specific protobuf descriptor values

**What NOT to Mock:**
- Protobuf descriptors (test with real .proto files)
- Code generation output (golden file tests validate real protoc output)
- External tooling like protoc (tests skip if protoc unavailable)

## Fixtures and Factories

**Test Data:**

Stored in `testdata/proto/` directories:
- `internal/httpgen/testdata/proto/http_verbs_comprehensive.proto`
- `internal/tsclientgen/testdata/proto/*.proto`
- `internal/openapiv3/testdata/proto/*.proto`

Pattern structure:
```protobuf
// Simple message for testing
message User {
    string id = 1;
    string name = 2;
}

// Service with HTTP configuration
service UserService {
    rpc CreateUser(CreateUserRequest) returns (User) {
        option (sebuf.http.config) = {
            path: "/users"
            method: HTTP_METHOD_POST
        };
    }
}
```

**Location:**
- Protocol buffer test files: `internal/*gen/testdata/proto/`
- Generated golden files: `internal/*gen/testdata/golden/`
- Test helper functions: In the `*_test.go` file itself (e.g., `generateTestFiles()`)

## Coverage

**Requirements:**
- Threshold: 85% per package
- Coverage mode: Enabled by default with `go test`
- Enforced in CI via `./scripts/run_tests.sh`
- Fast mode: Can skip coverage for rapid feedback during development

**View Coverage:**
```bash
# Generate and view HTML report
go tool cover -html=coverage/coverage.out -o coverage/coverage.html
open coverage/coverage.html

# View command-line report
go tool cover -func=coverage/coverage.out

# View JSON report
cat coverage/coverage.json

# View coverage badge
cat coverage/coverage-badge.svg
```

## Test Types

**Unit Tests:**
- Location: `*_test.go` files in same package
- Scope: Individual functions and helper methods
- Examples:
  - `TestLowerFirst()`: Tests name conversion function
  - `TestCamelToSnake()`: Tests case conversion
  - `TestValidationError_Struct()`: Tests error struct fields
  - `TestIsPathParamCompatibleByKind()`: Tests type compatibility logic
  - `TestHeaderNameToPropertyName()`: Tests header name to property conversion

**Integration Tests:**
- Location: `*_integration_test.go` suffix
- Scope: Multi-component interactions
- Examples:
  - `internal/openapiv3/integration_test.go`: Tests full OpenAPI document generation
  - Golden file tests (see below)

**Golden File Regression Tests:**
- Location: `*_golden_test.go` files (e.g., `golden_test.go`)
- Scope: Exhaustive byte-for-byte comparison of generated code
- Pattern: Execute real protoc with built plugin, compare output to golden reference files
- Purpose: Catch ANY unintended changes to generated output
- Test files:
  - `internal/httpgen/golden_test.go`: Tests HTTP handler generation
  - `internal/clientgen/golden_test.go`: Tests Go HTTP client generation
  - `internal/tsclientgen/golden_test.go`: Tests TypeScript client generation
  - `internal/openapiv3/exhaustive_golden_test.go`: Tests OpenAPI document generation

**E2E Tests:**
- Not explicitly implemented
- Golden file tests serve as E2E validation (full pipeline from proto to generated code)
- Real protoc execution provides end-to-end testing

## Common Patterns

**Async Testing:**
- Not applicable (Go is single-goroutine for test functions)
- Some tests use `t.TempDir()` which returns immediately (no async cleanup)

**Error Testing:**

Testing validation errors:
```go
func TestValidationError_Struct(t *testing.T) {
	err := ValidationError{
		Service: "UserService",
		Method:  "GetUser",
		Message: "path variable '{user_id}' has no matching field",
	}

	if err.Service != "UserService" {
		t.Errorf("ValidationError.Service = %q, expected %q", err.Service, "UserService")
	}
}
```

Testing error types and nil checks:
```go
func TestNewGenerator(t *testing.T) {
	gen := openapiv3.NewGenerator(openapiv3.FormatYAML)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}

	if gen.Doc() == nil {
		t.Error("Document is nil")
	}
}
```

**Golden File Testing Pattern:**

```go
// Skip if protoc unavailable
if _, err := exec.LookPath("protoc"); err != nil {
    t.Skip("protoc not found, skipping golden file tests")
}

// Define test cases with proto files and expected outputs
testCases := []struct {
    name          string
    protoFile     string
    expectedFiles []string
}{
    {
        name:      "comprehensive HTTP verbs",
        protoFile: "http_verbs_comprehensive.proto",
        expectedFiles: []string{
            "http_verbs_comprehensive_http.pb.go",
            "http_verbs_comprehensive_http_binding.pb.go",
            "http_verbs_comprehensive_http_config.pb.go",
        },
    },
}

// For each test case:
// 1. Build plugin binary
// 2. Run protoc with built plugin
// 3. Compare generated output to golden files byte-for-byte
// 4. On mismatch:
//    - Report first byte difference
//    - Report line-by-line differences (first 10 only)
//    - Support UPDATE_GOLDEN=1 env var to update golden files
//    - Write diff to .generated file for manual inspection
```

**Golden File Update Mechanism:**

```bash
# Update golden files after intentional code generation changes
UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles

# Or via Makefile
UPDATE_GOLDEN=1 go test -run TestHTTPGenGoldenFiles
```

When `UPDATE_GOLDEN=1`:
- Mismatched golden files are overwritten with new generated content
- Logs indicate which files were updated
- `.generated` files written for comparison even on update

**Golden File Helper Functions:**

```go
// reportFirstDifference: Locates first byte difference
// reportLineDifferences: Shows line-by-line diffs (limited to 10)
// handleGoldenFileUpdate: Updates golden file if UPDATE_GOLDEN=1
// writeTemporaryGeneratedFile: Writes .generated file for diff inspection
// tryCreateGoldenFile: Creates new golden file if missing and UPDATE_GOLDEN=1
```

## Test Coverage Gaps

**Known Focus Areas:**
- Unit tests: Strong coverage of name conversion functions, validation logic, type mapping
- Golden file tests: Comprehensive regression detection for all code generators
- Integration tests: Limited (only in openapiv3 package)
- Error path coverage: Validation and error handling tested, but some edge cases may lack tests

**Running Tests with Specific Scope:**
```bash
# Run tests for single package
go test -v ./internal/httpgen

# Run tests matching pattern
go test -v -run TestLower

# Run with race detector
go test -race ./...

# Run with verbose output
go test -v ./...
```

---

*Testing analysis: 2026-02-05*
