# protoc-gen-go-whatif: LLM-Powered Test Scenario Generation

## Overview

`protoc-gen-go-whatif` is a protoc plugin that generates type-safe, scenario-based mock servers using AI. It leverages Large Language Models (LLMs) to automatically create realistic "what if" test scenarios for your gRPC/HTTP APIs, reducing the manual effort required to build comprehensive test suites.

## Vision & Goals

### Primary Vision
Transform API testing by making it **effortless to generate realistic edge cases** that developers wouldn't think of manually. Instead of writing dozens of mock scenarios by hand, developers can leverage AI to automatically generate intelligent test cases that exercise real-world failure modes.

### Key Goals
1. **Type-Safe API**: Generate clean Go functions like `WhatIf.CreateUserDuplicateEmail()` 
2. **LLM Intelligence**: Use AI to suggest realistic scenarios based on proto message structures
3. **Proto-Aware**: Understand request/response fields and generate appropriate test data
4. **Robust Fallbacks**: Graceful degradation when LLM calls fail
5. **Developer Experience**: Simple configuration and intuitive usage

## Architecture

### Core Components

```
cmd/protoc-gen-go-whatif/
├── main.go                    # Plugin entry point, CLI argument parsing

internal/whatif/
├── generator.go               # Main orchestration and code generation
├── llm.go                     # Two-step LLM integration
└── scenarios.go               # Core data structures
```

### Two-Step LLM Approach

The plugin uses a sophisticated two-step approach to generate intelligent scenarios:

#### Step 1: Abstract Scenario Generation
- **Input**: Service and method definitions, semantics analysis
- **Output**: Creative scenario names and descriptions
- **LLM Prompt**: High-level business logic understanding
- **Example**: "Generate scenarios for CreateUser method that test edge cases"

#### Step 2: Proto-Aware Field Value Generation  
- **Input**: Abstract scenario + proto message structures
- **Output**: Specific field values and error codes
- **LLM Prompt**: Detailed proto field descriptions with validation rules
- **Example**: Generate `ERROR: 409 Email already exists` for duplicate scenarios

### Generated Code Structure

For each service, the plugin generates three files:

1. **`api_whatif.pb.go`**: Core types and server structure
2. **`api_whatif_scenarios.pb.go`**: Type-safe scenario functions
3. **`api_whatif_mock.pb.go`**: Mock server implementation

## Usage

### Configuration

Add to your `buf.gen.yaml`:

```yaml
plugins:
  - local: protoc-gen-go-whatif
    out: api
    opt:
      - paths=source_relative
      - openrouter_api_key=YOUR_OPENROUTER_API_KEY_HERE
      - model=openai/gpt-4o-mini
      - debug=true
```

### Generated API

The plugin generates type-safe functions for each scenario:

```go
// Service-level scenarios (affect all methods)
server := api.NewWhatIfUserServiceServer(
    api.WhatIf.DatabaseDown(),
    api.WhatIf.MaintenanceMode(),
)

// Method-specific scenarios  
server := api.NewWhatIfUserServiceServer(
    api.WhatIf.CreateUserDuplicateEmail(),
    api.WhatIf.LoginExpiredSocialToken(),
    api.WhatIf.GetUserRateLimitExceeded(),
)
```

### Example Generated Scenarios

The LLM generates contextually appropriate scenarios:

**CreateUser Method:**
- `CreateUserDuplicateEmail()` → `409 Email already exists`
- `CreateUserInvalidEmailFormat()` → `400 Invalid email format`
- `CreateUserExceedingNameLength()` → `400 Name exceeds maximum length`

**Login Method:**
- `LoginExpiredSocialToken()` → `401 Token has expired`
- `LoginLockedUserAccount()` → `403 Account is locked`
- `LoginMissingAuthToken()` → `401 Missing authentication credentials`

## Implementation Details

### LLM Integration

**OpenRouter Configuration:**
- Uses OpenAI Go SDK with OpenRouter base URL
- Structured JSON outputs with schema validation
- Graceful fallback when API calls fail

**Prompt Engineering:**
- Proto message structure analysis
- Business semantics inference (CRUD operations, auth patterns)
- Validation rule extraction from field names
- Context-aware error code selection

### Conflict Resolution

**Method Prefixing:**
Method-specific scenarios are prefixed to avoid naming conflicts:
- `CreateUserInvalidEmailFormat()` vs `LoginInvalidEmailFormat()`
- Ensures unique function names across all methods

**Handler Naming:**
- Function: `CreateUserDuplicateEmail()`
- Handler: `createuserduplicateemailHandler`
- Scenario: `duplicate_email`

### Error Intelligence

The LLM generates appropriate HTTP status codes:
- `400` - Validation errors (invalid format, missing fields)
- `401` - Authentication failures (expired tokens, missing credentials)
- `403` - Authorization failures (locked accounts, insufficient permissions)
- `404` - Resource not found (deleted users, invalid IDs)
- `409` - Conflicts (duplicate email, resource exists)
- `429` - Rate limiting (too many requests)

## Current Status

### ✅ Implemented
- [x] Two-step LLM scenario generation
- [x] Type-safe Go function API
- [x] OpenRouter integration with structured outputs
- [x] Proto-aware prompts and field analysis
- [x] Automatic conflict resolution
- [x] Intelligent error code generation
- [x] Graceful LLM failure handling
- [x] Method and service level scenarios

### 🚧 In Progress
- [ ] Proto reflection for dynamic response building
- [ ] Scenario caching to reduce LLM calls
- [ ] More sophisticated field value parsing

### 🎯 Future Enhancements
- [ ] Success scenario generation (not just errors)

## Technical Challenges Solved

### OpenAI Structured Output Validation
**Problem**: OpenAI requires strict JSON schemas with `additionalProperties: false`
**Solution**: Simplified schema design, avoiding complex map structures

### LLM Reliability
**Problem**: LLMs can fail or return unexpected formats
**Solution**: Two-step approach with graceful degradation to basic scenarios

### Naming Conflicts
**Problem**: Methods can have similar scenario names (e.g., "InvalidEmail")
**Solution**: Automatic method prefixing for unique function names

### Proto Message Understanding
**Problem**: LLM needs to understand proto field structure and validation
**Solution**: Rich prompt engineering with field descriptions and inferred validation rules

## Performance Considerations

### LLM Call Optimization
- **Batching**: Generate multiple scenarios per API call
- **Caching**: Future implementation to cache scenarios by proto hash
- **Timeout**: 30-second timeout with fallback to basic scenarios

### Generated Code Size
- **Minimal overhead**: Each scenario generates ~10 lines of Go code
- **Type safety**: No reflection in hot paths, compile-time verification

## Integration Patterns

### With Existing sebuf Plugins
The whatif plugin complements existing sebuf tools:
- **protoc-gen-go-http**: Generates real HTTP handlers
- **protoc-gen-openapiv3**: Documents API contracts
- **protoc-gen-go-whatif**: Provides test scenarios

### Testing Workflow
1. Define proto services and methods
2. Generate code with `buf generate`
3. Use whatif scenarios in tests
4. Verify both success and failure paths

## Configuration Options

```yaml
opt:
  - openrouter_api_key=sk-or-...     # Required: OpenRouter API key
  - model=openai/gpt-4o-mini         # Optional: LLM model selection
  - debug=true                       # Optional: Enable debug output
  - cache_scenarios=true             # Future: Enable scenario caching
  - max_scenarios_per_method=5       # Future: Limit scenario count
```

## Examples

See `examples/simple-api/` for a complete working example demonstrating:
- Multi-method service (CreateUser, GetUser, Login)
- Generated scenario functions
- Test suite using whatif scenarios
- Integration with other sebuf plugins

## Contributing

The whatif plugin is designed for extensibility:
- **New LLM providers**: Implement the LLMClient interface
- **Custom scenarios**: Extend the Scenario data structure
- **Field value parsing**: Enhance the parseFieldValues function
- **Prompt engineering**: Improve scenario quality via prompt optimization

## Roadmap

### Short Term (Next Month)
- [ ] Success scenario generation
- [ ] Enhanced field value parsing with proto reflection
- [ ] Scenario caching implementation

### Medium Term (Next Quarter)  
- [ ] Support for more LLM providers (Anthropic, local models)
- [ ] Advanced prompt engineering for better scenarios
- [ ] Integration with popular testing frameworks

### Long Term (Next Year)
- [ ] Scenario composition and dependencies
- [ ] Streaming RPC support
- [ ] Automatic performance test generation
- [ ] Observability and metrics integration

---

*This plugin represents a new paradigm in API testing: leveraging AI to generate comprehensive, realistic test scenarios automatically. The combination of type safety, proto awareness, and LLM intelligence creates a powerful tool for building robust APIs.*