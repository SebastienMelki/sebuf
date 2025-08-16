# sebuf Architecture

> Technical deep-dive into the sebuf protobuf toolkit design and implementation

This document provides a comprehensive technical overview of sebuf's architecture, designed for contributors, maintainers, and developers who want to understand how the toolkit works internally.

## Table of Contents

- [System Overview](#system-overview)
- [Plugin Architecture](#plugin-architecture)
- [Code Generation Pipeline](#code-generation-pipeline)
- [Component Deep Dive](#component-deep-dive)
- [Type System](#type-system)
- [Testing Strategy](#testing-strategy)
- [Performance Considerations](#performance-considerations)
- [Extension Points](#extension-points)

## System Overview

sebuf is a collection of three specialized protoc plugins that work together to enable modern HTTP API development from protobuf definitions:

```
┌─────────────────────────────────────────────────────────────────┐
│                        sebuf Toolkit                            │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  protoc-gen-    │  protoc-gen-    │    protoc-gen-              │
│  go-oneof-      │  go-http        │    openapiv3                │
│  helper         │                 │                             │
│                 │                 │                             │
│ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────────────────┐ │
│ │ Convenience │ │ │HTTP Handlers│ │ │   OpenAPI v3.1          │ │
│ │Constructors │ │ │   + Binding │ │ │  Specifications         │ │
│ │             │ │ │   + Routing │ │ │                         │ │
│ └─────────────┘ │ └─────────────┘ │ └─────────────────────────┘ │
└─────────────────┴─────────────────┴─────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Standard Go HTTP Stack                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   net/http  │  │ Gin/Echo/   │  │  Client Libraries       │  │
│  │             │  │ Chi/Fiber   │  │  (Generated from        │  │
│  │             │  │             │  │   OpenAPI specs)        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Plugin Architecture

Each sebuf plugin follows the standard protoc plugin interface while maintaining clean separation of concerns:

### Common Plugin Structure

```go
// All plugins follow this pattern:
func main() {
    // 1. Read CodeGeneratorRequest from stdin
    input := readStdin()
    
    // 2. Parse into protogen structures
    plugin := protogen.New(input)
    
    // 3. Delegate to specialized generator
    generator.Process(plugin)
    
    // 4. Write CodeGeneratorResponse to stdout
    writeStdout(plugin.Response())
}
```

### Plugin Responsibilities

| Plugin | Primary Function | Output | Dependencies |
|--------|-----------------|---------|--------------|
| `protoc-gen-go-oneof-helper` | Generate convenience constructors | `*_helpers.pb.go` | `protoc-gen-go` |
| `protoc-gen-go-http` | Generate HTTP handlers & routing | `*_http*.pb.go` | `protoc-gen-go`, sebuf annotations |
| `protoc-gen-openapiv3` | Generate OpenAPI specifications | `*.yaml`, `*.json` | None (standalone) |

## Code Generation Pipeline

### Phase 1: Protobuf Compilation

```mermaid
graph TD
    A[.proto files] --> B[protoc compiler]
    B --> C[CodeGeneratorRequest]
    C --> D[protoc-gen-go]
    C --> E[protoc-gen-go-oneof-helper]
    C --> F[protoc-gen-go-http]
    C --> G[protoc-gen-openapiv3]
    
    D --> H[.pb.go files]
    E --> I[*_helpers.pb.go]
    F --> J[*_http*.pb.go]
    G --> K[OpenAPI specs]
```

### Phase 2: Code Generation Flow

```go
// Simplified generation pipeline
type GenerationPipeline struct {
    input  *pluginpb.CodeGeneratorRequest
    plugin *protogen.Plugin
}

func (p *GenerationPipeline) Process() error {
    // 1. Parse and validate input
    if err := p.parseInput(); err != nil {
        return err
    }
    
    // 2. Process each file
    for _, file := range p.plugin.Files {
        if !file.Generate {
            continue
        }
        
        // 3. Generate code for each component
        if err := p.processFile(file); err != nil {
            return err
        }
    }
    
    return nil
}

func (p *GenerationPipeline) processFile(file *protogen.File) error {
    // File-level processing
    // - Extract metadata
    // - Process services and messages
    // - Generate output files
    return nil
}
```

## Component Deep Dive

### 1. Oneof Helper Generator

**Location**: `internal/oneofhelper/`

**Core Algorithm**:
```go
func GenerateHelpers(plugin *protogen.Plugin, file *protogen.File) {
    // 1. Traverse all messages in file
    for _, message := range file.Messages {
        // 2. Find oneof fields
        for _, oneof := range message.Oneofs {
            // 3. Process each oneof field
            for _, field := range oneof.Fields {
                // 4. Generate helper only for message types
                if field.Message != nil {
                    GenerateOneofHelper(message, oneof, field)
                }
            }
        }
        
        // 5. Recurse into nested messages
        for _, nested := range message.Messages {
            // Process recursively...
        }
    }
}
```

**Helper Generation Pattern**:
```go
// Generated function pattern: New{MessageName}{FieldName}
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

### 2. HTTP Handler Generator

**Location**: `internal/httpgen/`

**Architecture**:
```go
type HTTPGenerator struct {
    plugin *protogen.Plugin
}

func (g *HTTPGenerator) Generate() error {
    for _, file := range g.plugin.Files {
        // Generate three types of files:
        // 1. Main HTTP handlers (*_http.pb.go)
        // 2. Request binding logic (*_http_binding.pb.go)  
        // 3. Configuration options (*_http_config.pb.go)
        
        g.generateHTTPFile(file)
        g.generateBindingFile(file)
        g.generateConfigFile(file)
    }
}
```

**Generated Components**:

1. **Service Interface** - Type-safe server contract
2. **Registration Function** - HTTP handler setup
3. **Binding Middleware** - Request/response transformation
4. **Configuration Options** - Customization points

**Request Processing Flow**:
```go
// Generated request processing pipeline
HTTP Request → Content-Type Detection → Binding Middleware → Service Method → Response Marshaling → HTTP Response
```

### 3. OpenAPI Generator

**Location**: `internal/openapiv3/`

**Document Structure**:
```go
type Generator struct {
    doc     *v3.Document    // OpenAPI document
    schemas *SchemaMap      // Component schemas
    format  OutputFormat    // YAML or JSON
}

func (g *Generator) ProcessFile(file *protogen.File) {
    // 1. Extract document metadata
    g.updateDocumentInfo(file)
    
    // 2. Process all messages → schemas
    for _, message := range file.Messages {
        g.processMessage(message)
    }
    
    // 3. Process all services → paths
    for _, service := range file.Services {
        g.processService(service)
    }
}
```

**Type Mapping System**:
```go
func (g *Generator) convertField(field *protogen.Field) *Schema {
    switch {
    case field.Desc.IsList():
        return g.createArraySchema(field)
    case field.Desc.IsMap():
        return g.createMapSchema(field)
    case field.Message != nil:
        return g.createMessageReference(field)
    default:
        return g.convertScalarType(field)
    }
}
```

## Type System

sebuf implements a comprehensive type mapping system that handles the full spectrum of protobuf types:

### Scalar Type Mapping

| Protobuf Type | Go Type | OpenAPI Type | JSON Type |
|---------------|---------|--------------|-----------|
| `string` | `string` | `string` | `string` |
| `int32` | `int32` | `integer/int32` | `number` |
| `int64` | `int64` | `integer/int64` | `string` |
| `bool` | `bool` | `boolean` | `boolean` |
| `bytes` | `[]byte` | `string/byte` | `string` (base64) |
| `double` | `float64` | `number/double` | `number` |

### Complex Type Handling

**Repeated Fields (Arrays)**:
```protobuf
repeated string tags = 1;
```
```go
// Go: []string
// OpenAPI: {"type": "array", "items": {"type": "string"}}
```

**Map Fields**:
```protobuf
map<string, string> metadata = 1;
```
```go
// Go: map[string]string  
// OpenAPI: {"type": "object", "additionalProperties": {"type": "string"}}
```

**Message References**:
```protobuf
User user = 1;
```
```go
// Go: *User
// OpenAPI: {"$ref": "#/components/schemas/User"}
```

**Oneof Fields**:
```protobuf
oneof auth_method {
  EmailAuth email = 1;
  TokenAuth token = 2;
}
```
```go
// Generated helpers:
func NewLoginRequestEmail(email, password string) *LoginRequest
func NewLoginRequestToken(token string) *LoginRequest
```

## Testing Strategy

sebuf employs a multi-layered testing approach to ensure reliability:

### 1. Golden File Testing

**Purpose**: Detect unintended changes in generated code
**Location**: `internal/*/testdata/`

```go
func TestExhaustiveGoldenFiles(t *testing.T) {
    testCases := []string{
        "simple_oneof",
        "complex_types", 
        "nested_messages",
    }
    
    for _, testCase := range testCases {
        t.Run(testCase, func(t *testing.T) {
            // Generate code from test proto
            generated := generateCode(testCase + ".proto")
            
            // Compare with golden file
            golden := readGoldenFile(testCase + "_helpers.pb.go")
            assert.Equal(t, golden, generated)
        })
    }
}
```

### 2. Unit Testing

**Purpose**: Test individual functions and components
**Coverage**: 85%+ target

```go
func TestFieldTypeMapping(t *testing.T) {
    tests := []struct {
        field    *protogen.Field
        expected string
    }{
        {stringField, "string"},
        {repeatedStringField, "[]string"},
        {mapField, "map[string]string"},
    }
    // Test implementation...
}
```

### 3. Integration Testing

**Purpose**: Test complete workflows end-to-end

```go
func TestHTTPGenerationWorkflow(t *testing.T) {
    // 1. Create test proto file
    protoContent := `...`
    
    // 2. Run protoc with sebuf plugins
    runProtoc(protoContent)
    
    // 3. Compile generated Go code
    compileGenerated()
    
    // 4. Test HTTP endpoints
    testEndpoints()
}
```

### Test Data Organization

```
internal/oneofhelper/testdata/
├── proto/                    # Input proto files
│   ├── simple_oneof.proto
│   ├── complex_types.proto
│   └── nested_messages.proto
└── golden/                   # Expected outputs
    ├── simple_oneof_helpers.pb.go
    ├── complex_types_helpers.pb.go
    └── nested_messages_helpers.pb.go
```

## Performance Considerations

### Memory Management

**Efficient Protogen Usage**:
```go
// ✅ Good: Process files sequentially
for _, file := range plugin.Files {
    if !file.Generate {
        continue // Skip non-target files
    }
    processFile(file)
}

// ❌ Avoid: Loading all files into memory
allFiles := loadAllFiles(plugin.Files) // Memory intensive
```

**String Building Optimization**:
```go
// ✅ Good: Use GeneratedFile for output
g.P("func ", functionName, "(", parameters, ") {")
g.P("    return &", structName, "{")
g.P("        Field: value,")
g.P("    }")
g.P("}")

// ❌ Avoid: String concatenation
result := "func " + functionName + "(" + parameters + ") {\n" + ...
```

### Generation Speed

**Benchmarks** (for reference):
- Simple service (5 methods): ~2ms
- Complex service (20 methods, nested types): ~15ms
- Large API (100+ methods): ~100ms

**Optimization Strategies**:
1. **Lazy Evaluation** - Only process files marked for generation
2. **Incremental Generation** - Cache unchanged components
3. **Parallel Processing** - Independent files can be processed concurrently

## Extension Points

sebuf is designed to be extensible for future enhancements:

### 1. New Plugin Types

**Adding a New Generator**:
```go
// cmd/protoc-gen-new-feature/main.go
func main() {
    protogen.Options{}.Run(func(gen *protogen.Plugin) error {
        // Your custom generation logic
        return newfeature.Generate(gen)
    })
}
```

### 2. Custom Annotations

**Extending HTTP Annotations**:
```protobuf
// proto/sebuf/http/annotations.proto
extend google.protobuf.MethodOptions {
  AuthConfig auth_config = 50005;  // New annotation
}

message AuthConfig {
  repeated string required_roles = 1;
  bool require_authentication = 2;
}
```

### 3. Framework Integrations

**Plugin Interface for Frameworks**:
```go
type FrameworkAdapter interface {
    GenerateHandlers(service *protogen.Service) error
    GenerateMiddleware(options MiddlewareOptions) error
    GenerateRouting(paths []HTTPPath) error
}

// Implementations:
type GinAdapter struct{}
type EchoAdapter struct{}
type ChiAdapter struct{}
```

### 4. Output Format Extensions

**Adding New Output Formats**:
```go
type OutputFormat string

const (
    FormatYAML     OutputFormat = "yaml"
    FormatJSON     OutputFormat = "json"
    FormatTypeScript OutputFormat = "typescript"  // New format
)
```

## Design Principles

### 1. Separation of Concerns

Each plugin has a single, well-defined responsibility:
- **Oneof Helper**: Convenience constructors only
- **HTTP Generator**: HTTP protocol handling only  
- **OpenAPI Generator**: Documentation generation only

### 2. Zero Runtime Dependencies

Generated code has no sebuf runtime dependencies:
- Uses only standard Go libraries
- Compatible with any HTTP framework
- No vendor lock-in

### 3. Backwards Compatibility

- Generated code is stable across sebuf versions
- API changes follow semantic versioning
- Migration guides for breaking changes

### 4. Developer Experience

- Clear error messages with actionable guidance
- Comprehensive documentation and examples
- Predictable generation patterns

## Future Enhancements

### Planned Features

1. **Advanced HTTP Features**
   - Custom middleware generation
   - Authentication/authorization integration
   - Rate limiting support

2. **Additional Output Formats**
   - TypeScript client generation
   - Swagger/OpenAPI 2.0 support
   - GraphQL schema generation

3. **Performance Optimizations**
   - Incremental compilation
   - Parallel generation
   - Memory usage optimization

4. **Enhanced Tooling**
   - IDE integration
   - Live reload during development
   - Configuration validation

---

This architecture enables sebuf to provide a cohesive, yet modular toolkit for modern protobuf-to-HTTP development while maintaining flexibility for future enhancements and integrations.