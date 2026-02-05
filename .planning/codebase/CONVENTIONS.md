# Coding Conventions

**Analysis Date:** 2026-02-05

## Naming Patterns

**Files:**
- Go implementation files: `lowercase_snake_case.go` (e.g., `generator.go`, `annotations.go`, `error_handler.go`)
- Test files: `*_test.go` suffix with same base name (e.g., `generator_test.go`, `validation_test.go`)
- Golden file tests: `*_golden_test.go` for regression testing (e.g., `golden_test.go` in httpgen, tsclientgen, openapiv3)
- Generated output files: `*_http.pb.go`, `*_http_binding.pb.go`, `*_http_config.pb.go`, `*_client.ts`

**Functions:**
- Package-scoped helper functions: `camelCase` with lowercase first letter (e.g., `lowerFirst()`, `camelToSnake()`, `extractPathParams()`)
- Exported functions: `PascalCase` (e.g., `New()`, `Generate()`, `ProcessService()`)
- Methods: `PascalCase` (e.g., `GenerateFile()`, `generateHTTPFile()`)
- Constructor functions: `New` or `NewWithOptions()` pattern in `internal/*gen/generator.go`

**Variables:**
- Local variables: `camelCase` (e.g., `pathParams`, `httpConfig`, `baseDir`)
- Constants: `camelCase` for unexported package constants (e.g., `httpMethodGET`, `tsString`)
- Struct fields: `PascalCase` (e.g., `Path`, `Method`, `FieldName`)
- Regular expression patterns: `pathParamRegex`, `lowerCaseRegex` - lowercase_snake_case with Regex suffix

**Types:**
- Struct types: `PascalCase` (e.g., `HTTPConfig`, `QueryParam`, `ValidationError`, `Generator`)
- Interface types: `PascalCase` (e.g., `Printer`)
- Constants for enum-like strings: `camelCase` with Type prefix (e.g., `httpMethodGET`, `httpMethodPOST`)

## Code Style

**Formatting:**
- Go formatter: `go fmt` (standard Go formatting with tabs, 4-space indent)
- Enforced via EditorConfig at `/.editorconfig`:
  - Go files: tab indent, 4 spaces per tab
  - Proto files: space indent, 2 spaces
  - YAML/JSON: space indent, 2 spaces
  - Makefiles: tab indent (required by make syntax)

**Linting:**
- Tool: `golangci-lint` (if installed)
- Run with: `make lint` or `make lint-fix` (auto-fix mode)
- Not yet configured with .golangci.yml, uses defaults
- Check available at `/.editorconfig` for code style expectations

**Line Length:**
- No explicit enforced limit observed
- Typical patterns suggest preference for readability over strict column limits

## Import Organization

**Order:**
1. Standard library imports (e.g., `fmt`, `strings`, `bytes`, `os`)
2. External third-party imports (e.g., `google.golang.org/protobuf`, `github.com/pb33f/libopenapi`)
3. Local package imports (e.g., `github.com/SebastienMelki/sebuf/internal/...`)

**Path Aliases:**
- Used for clarity in imports: `v3 "github.com/pb33f/libopenapi/datamodel/high/v3"`
- Used for standard rewrites: `yaml "go.yaml.in/yaml/v4"`, `k8syaml "sigs.k8s.io/yaml"`
- Module imports: `"github.com/SebastienMelki/sebuf/http"` for custom annotations

**Bare Imports:**
- Avoid bare imports; all imports have explicit references in code

## Error Handling

**Patterns:**
- Early return on error: `if err != nil { return err }` for error propagation up the stack
- Error wrapping with context: `fmt.Errorf("validation error: %w", err)` to add context while preserving underlying error
- Test error patterns: `if err := cmd.Run(); err != nil { t.Fatalf("Failed to build plugin: %v", err) }`
- Custom validation errors: `ValidationError` struct with `Service`, `Method`, and `Message` fields
- Helper functions in tests: `t.Helper()` to mark test helper functions
- Assertions for nil checks: Tests check explicit nil comparisons for nil values

**Error Messages:**
- Descriptive messages: "Failed to get working directory: %v"
- Context-specific: Include what operation failed and the underlying error
- No automatic error wrapping unless adding value with context

## Logging

**Framework:** `testing.T` (t.Log, t.Logf, t.Error, t.Errorf, t.Fatal, t.Fatalf)

**Patterns:**
- Test logging: `t.Logf("Updated golden file: %s", goldenFile)`
- Test errors: `t.Errorf("Expected format %v, got %v", expected, actual)`
- Fatal errors: `t.Fatalf("Failed to read file: %v", err)` for test setup errors
- Helper functions: `t.Helper()` at start of test helper functions to report errors at caller location
- No printf-style formatting for simple messages: use `t.Log()` without format args

**Console Output in Scripts:**
- Scripts use colored output: `${RED}`, `${GREEN}`, `${YELLOW}`, `${BLUE}`, `${NC}` (no color)
- Status indicators: ✅ (success), ❌ (failure), ⚠️ (warning), 📊 (metrics), 💡 (info)

## Comments

**When to Comment:**
- Function-level documentation: Every exported function starts with `// FunctionName <description>` comment
- Complex logic: Explain WHY, not WHAT (code shows WHAT)
- Examples in comments: `// Example: "/users/{user_id}/posts/{post_id}" -> ["user_id", "post_id"]`
- Backward compatibility notes: `// HTTP_METHOD_UNSPECIFIED defaults to POST for backward compatibility`
- Important behavior: Comments explain enum handling, default values, and special cases

**JSDoc/TSDoc:**
- TypeScript generated code includes JSDoc-style comments for exported functions
- Not consistently used in Go code but present for critical helpers

**Test Comments:**
- Test helper comments explain: `// Test [what] tests [goal]`
- Golden file test comments explain update mechanism: `// UPDATE_GOLDEN=1 go test -run TestHTTPGenGoldenFiles`

## Function Design

**Size:**
- Single responsibility principle: Each function handles one concern
- Medium-sized functions (30-60 lines): Common for generator methods
- Small helper functions (5-15 lines): For name conversions, type mapping, common patterns

**Parameters:**
- Receiver methods on generators: `func (g *Generator) generateFile(file *protogen.File) error`
- Functional options pattern: Constructor options passed via `Options` struct (e.g., `NewWithOptions(plugin, opts)`)
- Test parameters: Simple `tt` struct pattern: `name`, `input`, `expected`/`want`
- Helper functions receive `*testing.T` first: `func generateTestFiles(t *testing.T, protoFile string)`

**Return Values:**
- Single return: `string`, `*HTTPConfig`, `error` (error last)
- Multiple returns: `(*generatedFiles, error)` pattern
- Pointer returns for complex types: `*Generator`, `*HTTPConfig`, `*protogen.Message`
- Void helpers: Some helpers return nothing: `func (g *Generator) writeHeader(p printer, file *protogen.File)`

## Module Design

**Exports:**
- Unexported by default: Functions and types start lowercase unless part of public API
- Exported for cross-package use: `New()`, `Generate()`, `ProcessService()`, `Format()`, `Doc()`, `Schemas()`
- Package constants: Exported for configuration (e.g., `FormatYAML`, `FormatJSON`)

**Barrel Files:**
- Not used in this codebase
- Each `internal/*gen/` package has focused purpose with related functions

**Package Organization:**
- `internal/httpgen/`: HTTP handler generation
- `internal/clientgen/`: Go HTTP client generation
- `internal/tsclientgen/`: TypeScript HTTP client generation
- `internal/openapiv3/`: OpenAPI v3.1 specification generation

**Initialization:**
- Generators initialized with `New(plugin)` or `NewWithOptions(plugin, opts)`
- Entry point via `(g *Generator).Generate()` method that processes all files

## Type Conversions

**Patterns:**
- Type assertions with nil checks: `httpConfig, ok := ext.(*http.HttpConfig); if !ok || httpConfig == nil`
- Enum conversions: `httpMethodToString()` for enum to string with defaults
- String case conversions: `lowerFirst()`, `camelToSnake()`, `snakeToLowerCamel()`, `headerNameToPropertyName()`

## Testing Patterns

**Table-driven tests:**
```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {"PascalCase to camelCase", "CreateUser", "createUser"},
    {"single word", "User", "user"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := lowerFirst(tt.input)
        if result != tt.expected {
            t.Errorf("lowerFirst(%q) = %q, expected %q", tt.input, result, tt.expected)
        }
    })
}
```

**Test Helper Pattern:**
- Mark with `t.Helper()` at function start
- Accept `*testing.T` as first parameter
- Use `t.Fatalf()` for setup errors, `t.Errorf()` for assertion failures

---

*Convention analysis: 2026-02-05*
