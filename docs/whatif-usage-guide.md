# protoc-gen-go-whatif Usage Guide

## Quick Start

### 1. Installation

The plugin is built automatically with `make build`:

```bash
make build
# Creates bin/protoc-gen-go-whatif
```

### 2. Configuration

Add to your `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - local: protoc-gen-go-whatif
    out: api
    opt:
      - paths=source_relative
      - openrouter_api_key=YOUR_OPENROUTER_API_KEY_HERE
      - model=openai/gpt-4o-mini
      - debug=true
```

**Get an OpenRouter API Key:**
1. Visit [openrouter.ai](https://openrouter.ai)
2. Sign up and get your API key
3. Replace `YOUR_OPENROUTER_API_KEY_HERE` with your actual key

### 3. Generate Code

```bash
# Make sure the plugin is in your PATH
export PATH=$PATH:path/to/sebuf/bin

# Generate code
buf generate
```

### 4. Use Generated Scenarios

```go
package main

import (
    "context"
    "fmt"
    "your-module/api"
)

func main() {
    // Create a mock server with LLM-generated scenarios
    server := api.NewWhatIfUserServiceServer(
        api.WhatIf.DatabaseDown(),              // Service-level: affects all methods
        api.WhatIf.CreateUserDuplicateEmail(),  // Method-specific: only CreateUser
        api.WhatIf.LoginExpiredSocialToken(),   // Method-specific: only Login
    )
    
    // Test CreateUser with duplicate email scenario
    ctx := context.Background()
    req := &api.CreateUserRequest{
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    // This will trigger the "duplicate email" scenario
    user, err := server.CreateUser(ctx, req)
    if err != nil {
        fmt.Printf("Expected error: %v\n", err)
        // Output: "email already exists in system"
    }
}
```

## Available Scenarios

### Service-Level Scenarios
These affect **all methods** in a service:

```go
api.WhatIf.DatabaseDown()                    // Database connection failed
api.WhatIf.MaintenanceMode()                 // Service in maintenance
api.WhatIf.AuthenticationServiceUnavailable() // Auth service down
```

### Method-Specific Scenarios

#### CreateUser Scenarios
```go
api.WhatIf.CreateUserDuplicateEmail()        // 409: Email already exists
api.WhatIf.CreateUserInvalidEmailFormat()    // 400: Invalid email format
api.WhatIf.CreateUserEmptyName()             // 400: Name field empty
api.WhatIf.CreateUserExceedingNameLength()   // 400: Name too long
api.WhatIf.CreateUserMissingRequiredFields() // 400: Missing required fields
```

#### GetUser Scenarios
```go
api.WhatIf.GetUserUserNotFound()             // 404: User not found
api.WhatIf.GetUserInvalidUserIdFormat()      // 400: Invalid ID format
api.WhatIf.GetUserExpiredUserSession()       // 401: Session expired
api.WhatIf.GetUserUserDeleted()              // 404: User deleted
api.WhatIf.GetUserRateLimitExceeded()        // 429: Rate limit hit
```

#### Login Scenarios
```go
api.WhatIf.LoginInvalidEmailFormat()         // 400: Invalid email
api.WhatIf.LoginMissingAuthToken()           // 401: Missing credentials
api.WhatIf.LoginExpiredSocialToken()         // 401: Token expired
api.WhatIf.LoginLockedUserAccount()          // 403: Account locked
api.WhatIf.LoginRateLimitExceeded()          // 429: Too many attempts
```

## Testing Patterns

### Unit Tests

```go
func TestCreateUser_DuplicateEmail(t *testing.T) {
    server := api.NewWhatIfUserServiceServer(
        api.WhatIf.CreateUserDuplicateEmail(),
    )
    
    req := &api.CreateUserRequest{
        Name:  "Test User",
        Email: "existing@example.com",
    }
    
    _, err := server.CreateUser(context.Background(), req)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "email already exists")
}
```

### Integration Tests

```go
func TestServiceResilience(t *testing.T) {
    // Test service behavior when database is down
    server := api.NewWhatIfUserServiceServer(
        api.WhatIf.DatabaseDown(),
    )
    
    // All methods should fail gracefully
    testCases := []struct {
        name string
        fn   func() error
    }{
        {"CreateUser", func() error {
            _, err := server.CreateUser(ctx, &api.CreateUserRequest{})
            return err
        }},
        {"GetUser", func() error {
            _, err := server.GetUser(ctx, &api.GetUserRequest{})
            return err
        }},
        {"Login", func() error {
            _, err := server.Login(ctx, &api.LoginRequest{})
            return err
        }},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := tc.fn()
            assert.Error(t, err)
            assert.Contains(t, err.Error(), "database connection failed")
        })
    }
}
```

### Scenario Combinations

```go
func TestComplexScenarios(t *testing.T) {
    // Combine multiple scenarios
    server := api.NewWhatIfUserServiceServer(
        api.WhatIf.DatabaseDown(),                // Service-level
        api.WhatIf.CreateUserDuplicateEmail(),    // CreateUser-specific
        api.WhatIf.LoginExpiredSocialToken(),     // Login-specific
    )
    
    // Database down affects all methods
    // Plus method-specific scenarios when database is working
}
```

## Configuration Options

### Model Selection

Different models have different strengths:

```yaml
# Fast and cost-effective (recommended)
- model=openai/gpt-4o-mini

# More creative scenarios
- model=openai/gpt-4o

# Alternative providers
- model=anthropic/claude-3-haiku
```

### Debug Mode

Enable debug output to see LLM interactions:

```yaml
- debug=true
```

Debug output shows:
- LLM prompts and responses
- Schema validation
- Scenario generation steps
- Error details

### Example Debug Output

```
Debug: Generating method scenarios for UserService.CreateUser (Step 1: Abstract scenarios)
Debug: LLM call successful
Debug: Response: {"scenarios":[{"name":"duplicate_email","description":"Attempt to create a user with an email that already exists"...}]}
Debug: Generating field values for scenario duplicate_email (Step 2)
Debug: Field values response: {"field_instructions":"ERROR: 409 Email already exists","is_error":true}
```

## Troubleshooting

### Plugin Not Found

```bash
# Make sure plugin is built and in PATH
make build
export PATH=$PATH:$(pwd)/bin
which protoc-gen-go-whatif
```

### LLM API Errors

```bash
# Check your API key
curl -H "Authorization: Bearer YOUR_KEY" https://openrouter.ai/api/v1/models

# Enable debug mode to see detailed errors
- debug=true
```

### Generated Code Issues

```bash
# Check if generated code compiles
go build ./api/

# View generated files
ls -la api/api_whatif*.pb.go
```

### Common Issues

1. **Missing API Key**: Set your OpenRouter API key in `buf.gen.yaml`
2. **Network Issues**: LLM calls require internet connection
3. **Rate Limits**: OpenRouter has rate limits, plugin includes retries
4. **Model Errors**: Some models work better than others, try `gpt-4o-mini`

## Best Practices

### 1. Start Simple

Begin with a few scenarios to understand the generated API:

```go
server := api.NewWhatIfUserServiceServer(
    api.WhatIf.DatabaseDown(),
)
```

### 2. Use in CI/CD

Add whatif scenarios to your test suite:

```go
func TestAllErrorScenarios(t *testing.T) {
    scenarios := []api.WhatIfOption{
        api.WhatIf.CreateUserDuplicateEmail(),
        api.WhatIf.GetUserUserNotFound(),
        api.WhatIf.LoginExpiredSocialToken(),
    }
    
    for _, scenario := range scenarios {
        t.Run(scenario.Name(), func(t *testing.T) {
            server := api.NewWhatIfUserServiceServer(scenario)
            // Test the scenario...
        })
    }
}
```

### 3. Document Scenarios

The generated scenarios serve as documentation of edge cases:

```go
// Document what scenarios your API handles
var DocumentedScenarios = []api.WhatIfOption{
    api.WhatIf.CreateUserDuplicateEmail(),    // Handles email uniqueness
    api.WhatIf.LoginExpiredSocialToken(),     // Validates token freshness
    api.WhatIf.GetUserRateLimitExceeded(),    // Implements rate limiting
}
```

### 4. Monitor LLM Quality

Review generated scenarios periodically:
- Are they realistic for your domain?
- Do they cover important edge cases?
- Are error messages appropriate?

The LLM learns from your proto definitions, so well-documented protos generate better scenarios.

## Advanced Usage

### Custom Scenarios (Future)

While the plugin focuses on LLM generation, you can extend it:

```go
// Future: Custom scenario implementation
type CustomScenario struct {
    name string
    handler func(context.Context, string, interface{}) (interface{}, error)
}
```

### Scenario Composition (Future)

```go
// Future: Combine scenarios
server := api.NewWhatIfUserServiceServer(
    api.WhatIf.Compose(
        api.WhatIf.DatabaseDown(),
        api.WhatIf.CreateUserDuplicateEmail(),
    ),
)
```

---

*This usage guide covers the current implementation. As the plugin evolves, more features will be added to enhance the developer experience.*