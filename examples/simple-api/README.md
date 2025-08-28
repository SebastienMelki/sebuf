# Simple API Tutorial

> **Learn sebuf by building a complete HTTP API from scratch**

This tutorial walks you through creating a working HTTP API with authentication, user management, and automatic documentation - all generated from a protobuf definition.

## 🚀 Just want to see it work?

```bash
make demo
```

This runs the complete workflow and starts the API server. Skip to [Testing the API](#testing-the-api) to try it out.

## What you'll build

A user management API with:
- ✅ **HTTP endpoints** for creating users and authentication
- ✅ **Mock server generation** with realistic data from field examples
- ✅ **Header validation** with type checking and format validation (UUID, email, datetime)
- ✅ **Automatic request validation** using buf.validate annotations
- ✅ **Structured error responses** with field-level validation details
- ✅ **Multiple auth methods** (email, token, social) using oneof fields
- ✅ **Helper functions** that eliminate protobuf boilerplate  
- ✅ **OpenAPI documentation** that stays in sync automatically, including header parameters and examples
- ✅ **JSON and binary** protobuf support

## Step-by-step walkthrough

### 1. Install dependencies

```bash
# Install Buf for protobuf management
brew install bufbuild/buf/buf

# Install sebuf tools
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
```

### 2. Understanding the protobuf definition

Look at `api.proto` - notice how we define services with HTTP annotations, header validation, body validation rules, and field examples:

```protobuf
service UserService {
  option (sebuf.http.service_config) = { base_path: "/api/v1" };
  
  // Service-level headers apply to all methods
  option (sebuf.http.service_headers) = {
    required_headers: [
      {
        name: "X-API-Key"
        description: "API authentication key"
        type: "string"
        format: "uuid"
        required: true
      }
    ]
  };
  
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (sebuf.http.config) = { path: "/users" };
    // Method-specific headers
    option (sebuf.http.method_headers) = {
      required_headers: [
        {
          name: "X-Request-ID"
          type: "string"
          format: "uuid"
          required: false
        }
      ]
    };
  }
  
  rpc Login(LoginRequest) returns (LoginResponse) {
    option (sebuf.http.config) = { path: "/auth/login" };
  }
}
```

**Field examples** provide realistic data for documentation and mock servers:

```protobuf
message CreateUserRequest {
  // Name is required and must be between 2 and 100 characters
  string name = 1 [
    (buf.validate.field).string = {
      min_len: 2,
      max_len: 100
    },
    (sebuf.http.field_examples) = {
      values: ["Alice Johnson", "Bob Smith", "Charlie Davis"]
    }
  ];
  
  // Email is required and must be a valid email address
  string email = 2 [
    (buf.validate.field).string.email = true,
    (sebuf.http.field_examples) = {
      values: ["alice@example.com", "bob@example.com", "charlie@example.com"]
    }
  ];
}
```

**Automatic validation** is built in for both headers and request bodies using buf.validate annotations shown above.

The `LoginRequest` uses a oneof field for different authentication methods:

```protobuf
message LoginRequest {
  oneof auth_method {
    EmailAuth email = 1;
    TokenAuth token = 2;
    SocialAuth social = 3;
  }
}
```

### 3. Generate all the code

```bash
# Fetch dependencies (first time only)
buf dep update

# Generate HTTP handlers, helper functions, and OpenAPI docs
buf generate

# Update Go module dependencies
go mod tidy
```

This creates:
- `api/api_http*.pb.go` - HTTP server code
- `api/api_http_mock.pb.go` - Mock server implementation (if enabled)
- `api/api_helpers.pb.go` - Helper functions for oneof fields
- `docs/UserService.openapi.yaml` - Complete API documentation with examples

### 4. Run the server

```bash
go run main.go
```

The server starts on port 8080 with these endpoints:
- `POST /api/v1/users` - Create a user
- `POST /api/v1/users/get` - Get a user by ID  
- `POST /api/v1/auth/login` - Login with different methods

**Note:** The example now uses a mock server (`api.NewMockUserServiceServer()`) that generates realistic responses based on the field examples in your protobuf definition.

## Testing the API

### Testing Header Validation

#### Valid request with required headers
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -H "X-Request-ID: 987fcdeb-51a2-43d1-9012-345678901234" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'
```

#### Missing required header (returns 400)
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'
# Response: 400 Bad Request
# Body:
{
  "violations": [{
    "field": "X-API-Key",
    "description": "required header 'X-API-Key' is missing"
  }]
}
```

#### Invalid header format (returns 400)
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-API-Key: not-a-valid-uuid" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'
# Response: 400 Bad Request
# Body:
{
  "violations": [{
    "field": "X-API-Key",
    "description": "header 'X-API-Key' validation failed: invalid UUID format"
  }]
}
```

### Testing Body Validation
Try creating a user with invalid data to see body validation in action (remember to include valid headers):

```bash
# Invalid email (returns 400)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "name": "John Doe",
    "email": "not-an-email"
  }'
# Response: {"violations": [{"field": "email", "description": "value must be a valid email address"}]}

# Name too short (returns 400)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "name": "J",
    "email": "john@example.com"
  }'
# Response: {"violations": [{"field": "name", "description": "value length must be at least 2 runes"}]}

# Multiple validation failures (returns 400)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "name": "J",
    "email": "invalid"
  }'
# Response:
{
  "violations": [
    {"field": "name", "description": "value length must be at least 2 runes"},
    {"field": "email", "description": "value must be a valid email address"}
  ]
}
```

### Login with email (demonstrates oneof helpers and validation)
```bash
# Valid login request
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "email": {
      "email": "john@example.com",
      "password": "secret123"
    }
  }'

# Test password validation (too short - should return 400)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "email": {
      "email": "john@example.com",
      "password": "short"
    }
  }'
```

### Login with token
```bash
# Valid token login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "token": {
      "token": "my-auth-token-1234567890"
    }
  }'

# Invalid token (too short - should return 400)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "token": {
      "token": "short"
    }
  }'
```

### Login with social auth
```bash
# Valid social login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "social": {
      "provider": "google",
      "access_token": "oauth-token-1234567890123456789012345"
    }
  }'

# Invalid provider (should return 400)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "social": {
      "provider": "invalid-provider",
      "access_token": "oauth-token-1234567890123456789012345"
    }
  }'
```

## See the generated helpers in action

In Go code, instead of this verbose protobuf construction:
```go
req := &api.LoginRequest{
  AuthMethod: &api.LoginRequest_Email{
    Email: &api.EmailAuth{
      Email: "user@example.com", 
      Password: "secret",
    },
  },
}
```

You can use the generated helpers:
```go
req := api.NewLoginRequestEmail("user@example.com", "secret")
req := api.NewLoginRequestToken("auth-token")  
req := api.NewLoginRequestSocial("google", "oauth-token")
```

## View the API documentation

After running `buf generate`, you'll find the OpenAPI documentation in the `docs/` directory:
- `docs/UserService.openapi.yaml` - Complete API documentation with all endpoints

The documentation includes:
- All API endpoints with their paths
- Required and optional header parameters with validation rules
- Request and response schemas
- Validation constraints from buf.validate annotations

```bash
# Quick view with Swagger UI
docker run -p 8081:8080 -v $(pwd)/docs:/app swaggerapi/swagger-ui
# Then visit http://localhost:8081/?url=/app/UserService.openapi.yaml
```

The OpenAPI spec shows:
- Service-level headers (e.g., `X-API-Key` required for all endpoints)
- Method-specific headers (e.g., `X-Request-ID` for CreateUser)
- Complete validation rules for request bodies
- Field examples for all fields with `(sebuf.http.field_examples)` annotations
- Error response schemas for validation failures

## Explore the generated code

- **`api/api.pb.go`** - Standard protobuf structs
- **`api/api_helpers.pb.go`** - Helper functions for oneof fields
- **`api/api_http.pb.go`** - HTTP server interface and registration
- **`api/api_http_binding.pb.go`** - Request/response binding logic
- **`api/api_http_config.pb.go`** - Configuration options

## Make it your own

1. **Add more endpoints** - Edit `api.proto` and regenerate
2. **Try different frameworks** - The generated handlers work with Gin, Echo, Chi, etc.
3. **Generate clients** - Use the OpenAPI spec to generate clients for any language
4. **Add middleware** - Wrap the generated handlers with your own logic

## Troubleshooting

**Command not found errors?**
- Make sure `$(go env GOPATH)/bin` is in your PATH
- Try reinstalling: `go install github.com/SebastienMelki/sebuf/cmd/...@latest`

**Import errors?**
- Run `go mod tidy` after generating code
- Check that all required tools are installed

**Need help?** 
- Check the [main documentation](../../docs/)
- Open an issue on GitHub
- Join the discussions