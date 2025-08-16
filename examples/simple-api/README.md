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
- ✅ **Multiple auth methods** (email, token, social) using oneof fields
- ✅ **Helper functions** that eliminate protobuf boilerplate  
- ✅ **OpenAPI documentation** that stays in sync automatically
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

Look at `api.proto` - notice how we define services with HTTP annotations:

```protobuf
service UserService {
  option (sebuf.http.service_config) = { base_path: "/api/v1" };
  
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (sebuf.http.config) = { path: "/users" };
  }
  
  rpc Login(LoginRequest) returns (LoginResponse) {
    option (sebuf.http.config) = { path: "/auth/login" };
  }
}
```

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
- `api/api_helpers.pb.go` - Helper functions for oneof fields
- `openapi.yaml` - Complete API documentation

### 4. Run the server

```bash
go run main.go
```

The server starts on port 8080 with these endpoints:
- `POST /api/v1/users` - Create a user
- `POST /api/v1/users/get` - Get a user by ID  
- `POST /api/v1/auth/login` - Login with different methods

## Testing the API

### Create a user
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'
```

### Login with email (demonstrates oneof helpers)
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": {
      "email": "john@example.com",
      "password": "secret123"
    }
  }'
```

### Login with token
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "token": {
      "token": "my-auth-token"
    }
  }'
```

### Login with social auth
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "social": {
      "provider": "google",
      "access_token": "oauth-token"
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

After running `buf generate`, open `openapi.yaml` in your favorite OpenAPI viewer:

```bash
# Quick view with Swagger UI
docker run -p 8081:8080 -v $(pwd):/app swaggerapi/swagger-ui
# Then visit http://localhost:8081/?url=/app/openapi.yaml
```

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