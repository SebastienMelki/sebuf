# Codebase Structure

**Analysis Date:** 2026-02-05

## Directory Layout

```
sebuf/
├── cmd/                                 # Plugin entry points
│   ├── protoc-gen-go-http/             # HTTP handler generator plugin
│   ├── protoc-gen-go-client/           # Go HTTP client generator plugin
│   ├── protoc-gen-ts-client/           # TypeScript client generator plugin
│   └── protoc-gen-openapiv3/           # OpenAPI specification generator plugin
├── internal/                            # Core generation logic (not exported)
│   ├── httpgen/                        # HTTP handler generation
│   ├── clientgen/                      # Go HTTP client generation
│   ├── tsclientgen/                    # TypeScript client generation
│   └── openapiv3/                      # OpenAPI specification generation
├── http/                                # Runtime types and error definitions
│   ├── annotations.pb.go               # Generated protobuf extensions
│   ├── errors.pb.go                    # Generated error message types
│   ├── headers.pb.go                   # Generated header validation types
│   └── errors_impl.go                  # Error interface implementations
├── proto/                               # Protocol buffer definitions
│   └── sebuf/http/                     # HTTP annotation definitions
│       ├── annotations.proto           # HTTP config, query, unwrap annotations
│       ├── headers.proto               # Header validation types
│       └── errors.proto                # Error message definitions
├── examples/                            # Example projects demonstrating sebuf
│   ├── simple-api/                     # Basic CRUD API with UserService
│   ├── ts-client-demo/                 # Full TypeScript client example
│   ├── rn-client-demo/                 # React Native client example
│   ├── restful-crud/                   # RESTful CRUD patterns
│   ├── nested-resources/               # Nested resource examples
│   ├── multi-service-api/              # Multiple service coordination
│   ├── market-data-unwrap/             # Unwrap annotation example
│   ├── error-handler/                  # Error handling patterns
│   └── validation-showcase/            # Validation annotation examples
├── scripts/                             # Build and test automation
│   └── run_tests.sh                    # Test runner with coverage analysis
├── docs/                                # Documentation and guides
├── Makefile                             # Build targets
├── go.mod                               # Go module definition
└── CLAUDE.md                            # This project's Claude instructions
```

## Directory Purposes

**cmd/ - Plugin Entry Points:**
- Purpose: Each subdirectory contains a main.go that implements a protoc plugin
- Contains: main() function that sets up protogen.Options and delegates to generators
- Key files:
  - `cmd/protoc-gen-go-http/main.go`: Parses flags (generate_mock), creates httpgen.Generator
  - `cmd/protoc-gen-go-client/main.go`: Creates clientgen.Generator
  - `cmd/protoc-gen-ts-client/main.go`: Enables proto3 optional support, creates tsclientgen.Generator
  - `cmd/protoc-gen-openapiv3/main.go`: Parses format parameter, iterates services, creates openapiv3.Generator per-service

**internal/httpgen/ - HTTP Handler Generation:**
- Purpose: Generate HTTP server handlers, request/response binding, validation middleware
- Contains: Generator logic, annotation parsing, validation, unwrap support, mock generation
- Key files:
  - `generator.go` (1550 lines): Main orchestrator, generates _http.pb.go, _http_binding.pb.go, _http_config.pb.go, _http_mock.pb.go
  - `annotations.go` (392 lines): Parses HTTP config from proto extensions, extracts path parameters
  - `validation.go` (172 lines): ValidateService(), ValidateMethodConfig(), field type checking
  - `unwrap.go` (902 lines): JSON unwrapping for map values, root unwrap messages, generates MarshalJSON/UnmarshalJSON
  - `mock_generator.go` (488 lines): Generates mock server implementations for testing

**internal/clientgen/ - Go Client Generation:**
- Purpose: Generate type-safe Go HTTP clients with functional options pattern
- Contains: Client interface generation, method implementations, option types
- Key files:
  - `generator.go`: Main orchestrator, generates _client.pb.go with client interface and call options
  - `annotations.go`: Parses HTTP config from methods to determine request body inclusion

**internal/tsclientgen/ - TypeScript Client Generation:**
- Purpose: Generate type-safe TypeScript HTTP clients with service classes
- Contains: Client class generation, interface definitions, error types, method implementation
- Key files:
  - `generator.go`: Main orchestrator, generates _client.ts with interfaces, enums, error types, service client
  - `helpers.go`: Utility functions for TypeScript generation (path substitution, query encoding, header handling)
  - `types.go`: Message and service metadata collection, dependency tracking
  - `annotations.go`: Parses HTTP config from methods

**internal/openapiv3/ - OpenAPI Specification Generation:**
- Purpose: Generate comprehensive OpenAPI v3.1 specifications from protobuf services
- Contains: Schema generation, path generation, validation rule documentation
- Key files:
  - `generator.go`: Main orchestrator, processes service to OpenAPI doc, generates one file per service
  - `types.go`: Protobuf to OpenAPI schema conversion, field type mapping
  - `http_annotations.go`: Extracts HTTP config for path and method documentation
  - `validation.go`: Extracts buf.validate rules to OpenAPI constraints

**http/ - Runtime Types and Error Handling:**
- Purpose: Provide error types and interfaces used by generated code and clients
- Contains: Protobuf error messages, error interface implementations, header definitions
- Key files:
  - `errors.pb.go`: Generated ValidationError and Error message types
  - `headers.pb.go`: Generated header validation configuration types
  - `errors_impl.go`: Error() method implementations for ValidationError and Error, enables errors.As()/errors.Is()
  - `annotations.pb.go`: Generated extension definitions used throughout

**proto/sebuf/http/ - Protobuf Annotation Definitions:**
- Purpose: Define custom protobuf extensions for HTTP configuration
- Contains: Extension definitions for methods, services, and fields
- Key files:
  - `annotations.proto` (74 lines): HttpConfig, ServiceConfig, QueryConfig, FieldExamples, unwrap
  - `headers.proto`: HeaderConfig for service and method-level header validation
  - `errors.proto`: ErrorField and FieldViolation message definitions

**examples/ - Demonstration Projects:**
- Purpose: Show sebuf usage patterns and serve as integration tests
- Contains: Proto definitions, generated code, client code, API implementations
- Key directories:
  - `simple-api/`: Minimal UserService CRUD example
  - `ts-client-demo/`: Full TypeScript client with NoteService
  - `rn-client-demo/`: React Native client example
  - `validation-showcase/`: buf.validate annotation patterns
  - `market-data-unwrap/`: Unwrap annotation for map values

**scripts/ - Build and Test Automation:**
- Purpose: Automate testing, code generation, and coverage analysis
- Contains: Shell scripts for common development tasks
- Key files:
  - `run_tests.sh`: Test runner supporting --fast, --verbose, UPDATE_GOLDEN=1 modes

## Key File Locations

**Entry Points:**
- `cmd/protoc-gen-go-http/main.go`: HTTP handler plugin invocation point
- `cmd/protoc-gen-go-client/main.go`: Go client plugin invocation point
- `cmd/protoc-gen-ts-client/main.go`: TypeScript client plugin invocation point
- `cmd/protoc-gen-openapiv3/main.go`: OpenAPI plugin invocation point

**Configuration:**
- `proto/sebuf/http/annotations.proto`: HTTP method, service, and field configuration
- `proto/sebuf/http/headers.proto`: Header validation configuration
- `Makefile`: Build targets and common commands
- `go.mod`: Go module dependencies

**Core Logic:**
- `internal/httpgen/generator.go`: HTTP handler generation orchestrator
- `internal/clientgen/generator.go`: Go client generation orchestrator
- `internal/tsclientgen/generator.go`: TypeScript client generation orchestrator
- `internal/openapiv3/generator.go`: OpenAPI generation orchestrator
- `internal/httpgen/validation.go`: HTTP configuration validation
- `internal/httpgen/unwrap.go`: JSON unwrap serialization

**Testing:**
- `internal/httpgen/golden_test.go`: Golden file regression tests for HTTP handlers
- `internal/clientgen/golden_test.go`: Golden file regression tests for Go clients
- `internal/tsclientgen/golden_test.go`: Golden file regression tests for TypeScript clients
- `internal/openapiv3/exhaustive_golden_test.go`: Golden file regression tests for OpenAPI
- `internal/httpgen/testdata/proto/`: Test proto definitions
- `internal/httpgen/testdata/golden/`: Expected generated output for HTTP handlers

## Naming Conventions

**Files:**

- Plugin entry points: `protoc-gen-{language}-{feature}/main.go`
  - Examples: `protoc-gen-go-http`, `protoc-gen-ts-client`

- Generated code files:
  - HTTP handlers: `{service}_http.pb.go`, `{service}_http_binding.pb.go`, `{service}_http_config.pb.go`, `{service}_http_mock.pb.go`
  - Go clients: `{service}_client.pb.go`
  - TypeScript clients: `{service}_client.ts`
  - OpenAPI specs: `{ServiceName}.openapi.yaml` or `{ServiceName}.openapi.json`

- Test files: `{module}_test.go` (unit tests) or `{feature}_golden_test.go` (regression tests)

- Proto files: `*.proto` with package names matching directory structure (e.g., `sebuf/http/annotations.proto`)

**Directories:**

- Package directories: lowercase, hyphenated for multiple words
  - Examples: `protoc-gen-go-http`, `protoc-gen-ts-client`

- Proto package paths: dot-separated, lowercase
  - Examples: `sebuf.http`, `api.services`, `api.models`

- Test data: `testdata/` with subdirectories for `proto/`, `golden/`, `coverage/`

## Where to Add New Code

**New HTTP Handler Feature:**
- Primary code: `internal/httpgen/generator.go` (extend Generate() or generateHTTPFile())
- Tests: `internal/httpgen/{feature}_test.go` and update `internal/httpgen/testdata/proto/` with test proto
- Golden files: `internal/httpgen/testdata/golden/` for expected output

**New Go Client Feature:**
- Implementation: `internal/clientgen/generator.go` (extend generateClientFile() or generateServiceClient())
- Tests: `internal/clientgen/golden_test.go` with new test data in `internal/clientgen/testdata/proto/`
- Golden files: `internal/clientgen/testdata/golden/`

**New TypeScript Client Feature:**
- Implementation: `internal/tsclientgen/generator.go` (extend generateClientFile() or generateServiceClient())
- Helpers: `internal/tsclientgen/helpers.go` for utility functions
- Tests: `internal/tsclientgen/golden_test.go` with test data in `internal/tsclientgen/testdata/proto/`
- Golden files: `internal/tsclientgen/testdata/golden/`

**New OpenAPI Feature:**
- Implementation: `internal/openapiv3/generator.go` (extend ProcessService() or convertField())
- Type mapping: `internal/openapiv3/types.go` for schema conversion
- Tests: `internal/openapiv3/exhaustive_golden_test.go` with test data in `internal/openapiv3/testdata/proto/`
- Golden files: `internal/openapiv3/testdata/golden/{yaml|json}/`

**New Proto Annotation:**
- Definition: `proto/sebuf/http/{feature}.proto` or extend existing proto file
- Runtime code: `http/{feature}.go` if new error types or interfaces needed
- Parser: `internal/*/annotations.go` in relevant generator packages

**Utilities and Helpers:**
- Shared functions: Create in relevant generator's helpers file (e.g., `internal/tsclientgen/helpers.go`)
- Cross-package: Consider `http/` package if runtime-critical

## Special Directories

**internal/httpgen/testdata/:**
- Purpose: Test data for HTTP handler generation
- Generated: Yes (by protoc from proto files)
- Committed: Yes (golden files are committed to track regressions)
- Contents:
  - `proto/`: Proto definitions used in tests
  - `golden/`: Expected generated output files

**internal/openapiv3/testdata/:**
- Purpose: Test data for OpenAPI generation
- Generated: Yes (by protoc and test generation)
- Committed: Yes (golden files track regressions)
- Contents:
  - `proto/`: Proto definitions used in tests
  - `golden/yaml/`: Expected YAML OpenAPI output
  - `golden/json/`: Expected JSON OpenAPI output

**examples/:**
- Purpose: Demonstration projects and integration tests
- Generated: Yes (code generated by protoc from proto files)
- Committed: Yes (example protos and generated code)
- Note: Represents real usage patterns, sometimes run through test suite

**bin/:**
- Purpose: Built plugin binaries
- Generated: Yes (by make build)
- Committed: No (generated by build process)

---

*Structure analysis: 2026-02-05*
