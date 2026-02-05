# Architecture

**Analysis Date:** 2026-02-05

## Pattern Overview

**Overall:** Plugin-based code generation using protoc plugin architecture with separate concerns for four complementary generators.

**Key Characteristics:**
- Multi-plugin design: Each generator is a standalone protoc plugin with isolated responsibility
- Protocol Buffer driven: Input is protobuf service definitions with custom HTTP annotations
- Code generation as primary function: All plugins consume protobuf definitions and emit generated code/documentation
- Type-safe output across languages: Go, TypeScript, and OpenAPI specification generation
- Custom annotation system: Uses protobuf field/method/service extensions for HTTP configuration

## Layers

**Plugin Entry Points:**
- Purpose: Accept protoc requests and coordinate generation
- Location: `cmd/protoc-gen-go-http/main.go`, `cmd/protoc-gen-go-client/main.go`, `cmd/protoc-gen-ts-client/main.go`, `cmd/protoc-gen-openapiv3/main.go`
- Contains: Main function, parameter parsing, plugin initialization
- Depends on: Standard protogen library, internal generators
- Used by: protoc compiler during code generation

**Generator Layer (HTTP):**
- Purpose: Generate HTTP server handlers, bindings, and configuration from protobuf services
- Location: `internal/httpgen/generator.go` (1550 lines)
- Contains: Service interface generation, handler registration, binding logic generation, validation code generation, mock server generation
- Depends on: `internal/httpgen/annotations.go`, `internal/httpgen/unwrap.go`, `internal/httpgen/validation.go`, `internal/httpgen/mock_generator.go`
- Used by: `cmd/protoc-gen-go-http/main.go`

**Generator Layer (Go Client):**
- Purpose: Generate type-safe HTTP client code for Go with functional options pattern
- Location: `internal/clientgen/generator.go`
- Contains: Client interface generation, call option definitions, method implementation, header helpers
- Depends on: `internal/clientgen/annotations.go`
- Used by: `cmd/protoc-gen-go-client/main.go`

**Generator Layer (TypeScript Client):**
- Purpose: Generate type-safe HTTP client code for TypeScript with typed interfaces
- Location: `internal/tsclientgen/generator.go`
- Contains: Service class generation, method implementation, error type definitions, interface definitions, header helpers
- Depends on: `internal/tsclientgen/helpers.go`, `internal/tsclientgen/types.go`, `internal/tsclientgen/annotations.go`
- Used by: `cmd/protoc-gen-ts-client/main.go`

**Generator Layer (OpenAPI):**
- Purpose: Generate comprehensive OpenAPI v3.1 specifications from protobuf services
- Location: `internal/openapiv3/generator.go` (53 lines in main, but complex generation)
- Contains: Service to OpenAPI document conversion, schema generation, header parameter documentation
- Depends on: `internal/openapiv3/types.go`, `internal/openapiv3/http_annotations.go`, `internal/openapiv3/validation.go`
- Used by: `cmd/protoc-gen-openapiv3/main.go`

**Annotation System:**
- Purpose: Define custom protobuf extensions for HTTP configuration
- Location: `proto/sebuf/http/annotations.proto` (74 lines)
- Contains: HttpConfig, HttpMethod, ServiceConfig, QueryConfig, FieldExamples, unwrap extensions
- Depends on: Google protobuf descriptor extensions
- Used by: All generators for reading configuration from proto definitions

**HTTP Package (Runtime):**
- Purpose: Provide runtime error types and interfaces for generated code
- Location: `http/errors.pb.go`, `http/annotations.pb.go`, `http/headers.pb.go`, `http/errors_impl.go`
- Contains: ValidationError, Error protobuf messages, error interface implementations, header validation definitions
- Used by: Generated HTTP handlers and client code

**Validation System:**
- Purpose: Validate HTTP configurations at generation time and request bodies at runtime
- Location: `internal/httpgen/validation.go` (172 lines)
- Contains: ValidateMethodConfig, field validation, path parameter type checking, query parameter conflict detection
- Depends on: protogen reflection
- Used by: `internal/httpgen/generator.go` during service generation

**Unwrap System:**
- Purpose: Handle JSON serialization of wrapped repeated fields in map values
- Location: `internal/httpgen/unwrap.go` (902 lines)
- Contains: UnwrapContext, RootUnwrapMessage, map value unwrapping logic, JSON method generation
- Depends on: protogen reflection
- Used by: `internal/httpgen/generator.go` for special JSON serialization

## Data Flow

**Protoc Plugin Invocation (HTTP Handler Generation):**

1. protoc passes CodeGeneratorRequest to stdin → `cmd/protoc-gen-go-http/main.go`
2. Main.go creates protogen.Plugin from request → calls `httpgen.NewWithOptions()`
3. httpgen.Generator.Generate() iterates all files
4. For each file:
   - Calls ValidateService() from `internal/httpgen/validation.go` (fail-fast)
   - Calls generateErrorImplFile() to create Error interface implementations
   - Calls generateUnwrapFile() to create JSON unwrap methods
   - Calls generateHTTPFile() → generates `*_http.pb.go` with service interfaces and registration functions
   - Calls generateBindingFile() → generates `*_http_binding.pb.go` with BindingMiddleware, request/response binding logic
   - Calls generateConfigFile() → generates `*_http_config.pb.go` with path/query parameter configs and header validation
   - Optionally calls generateMockFile() → generates mock server implementation
5. Generated files written to protogen.GeneratedFile
6. Response marshaled to stdout as protobuf

**Protoc Plugin Invocation (Go Client Generation):**

1. protoc passes CodeGeneratorRequest to stdin → `cmd/protoc-gen-go-client/main.go`
2. Main.go creates protogen.Plugin → calls `clientgen.New()`
3. clientgen.Generator.Generate() iterates all files
4. For each file with services:
   - Calls generateClientFile() → generates `*_client.pb.go`
   - For each service: generates ClientInterface, NewClient() constructor, functional option types
   - For each method: generates call wrapper with option application, HTTP verb detection, request body serialization
5. Generated files written to protogen.GeneratedFile
6. Response marshaled to stdout

**Protoc Plugin Invocation (TypeScript Client Generation):**

1. protoc passes CodeGeneratorRequest to stdin → `cmd/protoc-gen-ts-client/main.go`
2. Main.go creates protogen.Plugin with proto3 optional support → calls `tsclientgen.New()`
3. tsclientgen.Generator.Generate() iterates all files
4. For each file with services:
   - Calls generateClientFile() → generates `*_client.ts`
   - Collects all message interfaces and enums used by services
   - Generates TypeScript interfaces for each message
   - Generates enum type definitions
   - Generates error types (FieldViolation, ValidationError, ApiError)
   - For each service: generates service client class
   - For each method: generates method implementation with path substitution, query parameter encoding, header helpers
5. Generated TypeScript files written to protogen.GeneratedFile
6. Response marshaled to stdout

**Protoc Plugin Invocation (OpenAPI Generation):**

1. protoc passes CodeGeneratorRequest to stdin → `cmd/protoc-gen-openapiv3/main.go`
2. Main.go parses format parameter (yaml/json), creates protogen.Plugin
3. For each service in each file:
   - Creates openapiv3.Generator → calls CollectReferencedMessages() to gather all types
   - Calls ProcessService() to convert service to OpenAPI paths
   - Calls Render() to serialize to YAML or JSON
   - Writes one file per service: `{ServiceName}.openapi.{yaml|json}`
4. Response marshaled to stdout

**Request Handling (Runtime - HTTP Handler):**

1. HTTP request arrives at mux → routed to generated handler
2. BindingMiddleware wrapper intercepts
3. Path parameters extracted from URL, unmarshaled to request message fields
4. Query parameters parsed from URL, unmarshaled to request message fields
5. Request body read from HTTP request, unmarshaled (JSON or protobuf) to request message
6. Header validation middleware checks required headers (service-level + method-level)
7. Request body validation via protovalidate (buf.validate rules)
8. If validation passes: handler function called with context and request
9. Handler returns response message or error
10. Response serialized to JSON or protobuf based on Accept header
11. HTTP response written with status code and serialized body

**State Management:**

- No persistent state: Each plugin invocation is isolated
- Configuration read from proto annotations at generation time
- Generated code is pure and stateless
- Runtime state managed by application code using generated handlers/clients

## Key Abstractions

**HTTPConfig:**
- Purpose: Represents HTTP method configuration (path, verb, extracted path parameters)
- Examples: `internal/httpgen/annotations.go:26-30`
- Pattern: Extracted from method options via proto.GetExtension()

**QueryParam:**
- Purpose: Represents query parameter binding (field name, parameter name, required flag)
- Examples: `internal/httpgen/annotations.go:32-38`
- Pattern: Extracted from field options via proto.GetExtension()

**ValidationError:**
- Purpose: Represents generation-time validation errors (service, method, message)
- Examples: `internal/httpgen/validation.go:11-15`
- Pattern: Collected during ValidateService() and reported to user

**UnwrapContext:**
- Purpose: Tracks messages requiring JSON unwrapping for map value collapse
- Examples: `internal/httpgen/unwrap.go:28-43`
- Pattern: Analyzed during file processing, used to generate MarshalJSON/UnmarshalJSON

**Generator:**
- Purpose: Base abstraction for code generation from protobuf definitions
- Pattern: Implements Generate() method iterating files, creates protogen.GeneratedFile for output

## Entry Points

**HTTP Handler Plugin:**
- Location: `cmd/protoc-gen-go-http/main.go`
- Triggers: `protoc --go-http_out=. filename.proto`
- Responsibilities: Register plugins with protoc, delegate to httpgen.Generator

**Go Client Plugin:**
- Location: `cmd/protoc-gen-go-client/main.go`
- Triggers: `protoc --go-client_out=. filename.proto`
- Responsibilities: Register plugins with protoc, delegate to clientgen.Generator

**TypeScript Client Plugin:**
- Location: `cmd/protoc-gen-ts-client/main.go`
- Triggers: `protoc --ts-client_out=. filename.proto`
- Responsibilities: Register plugins with protoc, delegate to tsclientgen.Generator, enable proto3 optional support

**OpenAPI Plugin:**
- Location: `cmd/protoc-gen-openapiv3/main.go`
- Triggers: `protoc --openapiv3_out=format=yaml:.  filename.proto`
- Responsibilities: Parse format parameter, register plugins with protoc, delegate to openapiv3.Generator per-service

## Error Handling

**Strategy:** Fail-fast validation at generation time, with comprehensive error messages. Runtime validation via ValidationError messages.

**Patterns:**

- **Generation-Time Validation** (`internal/httpgen/validation.go`):
  - ValidateService() checks all methods before generating any code
  - Returns ValidationError structs with service, method, and detailed message
  - Checks path parameter compatibility, query parameter conflicts, field existence
  - Prevents invalid configurations from producing broken code

- **Runtime Request Validation** (`internal/httpgen/generator.go` binding generation):
  - Body validation via protovalidate (buf.validate rules)
  - Header validation via HeaderValidationMiddleware
  - Both return HTTP 400 with ValidationError message if validation fails
  - ValidationError includes field-level violation details

- **Client-Side Error Handling** (TypeScript and Go clients):
  - ApiError and ValidationError types for distinction
  - ApiError contains statusCode and message for HTTP errors
  - ValidationError contains violations array for field-level errors
  - Can be used with errors.As() / errors.Is() in Go, instanceof in TypeScript

## Cross-Cutting Concerns

**Logging:** Not generated. Application layer responsible for request/response logging via middleware.

**Validation:**
- Automatic via buf.validate annotations in request messages
- Header validation via sebuf.http.service_headers and method_headers annotations
- Generation-time validation of HTTP configurations via ValidateService()

**Authentication:**
- Service-level required headers defined in service_headers annotation
- Method-level required headers defined in method_headers annotation
- Generated code enforces presence; application layer handles verification
- Example: X-API-Key and X-Request-ID headers in example proto

---

*Architecture analysis: 2026-02-05*
