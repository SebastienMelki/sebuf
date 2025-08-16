# HTTP Generation

> Transform protobuf services into production-ready HTTP APIs

The `protoc-gen-go-http` plugin generates complete HTTP server infrastructure from protobuf service definitions, enabling you to build JSON/HTTP APIs with the type safety and code generation benefits of protobuf.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [HTTP Annotations](#http-annotations)
- [Generated Code Structure](#generated-code-structure)
- [Framework Integration](#framework-integration)
- [Request/Response Handling](#requestresponse-handling)
- [Configuration Options](#configuration-options)
- [Advanced Examples](#advanced-examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The HTTP generation plugin creates three main components from your protobuf service definitions:

1. **Service Interface** - Type-safe server interface matching your protobuf service
2. **HTTP Handlers** - Complete HTTP request/response handling with JSON and binary protobuf support
3. **Registration Functions** - Easy integration with Go HTTP frameworks and standard library

### Key Features

- **Multiple Content Types** - Automatic JSON and binary protobuf support
- **Framework Agnostic** - Works with any Go HTTP framework or standard library
- **Type Safe** - Full protobuf type checking and validation
- **Customizable Routing** - Control HTTP paths through annotations
- **Middleware Ready** - Built-in hooks for authentication, logging, etc.

## Installation

```bash
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
```

Verify installation:
```bash
protoc-gen-go-http --version
```

## Quick Start

### 1. Define Your Service

Create `user_service.proto`:
```protobuf
syntax = "proto3";
package userapi;

import "sebuf/http/annotations.proto";

option go_package = "github.com/yourorg/userapi;userapi";

// User management service
service UserService {
  // Configure base path for all endpoints
  option (sebuf.http.service_config) = {
    base_path: "/api/v1"
  };
  
  // Create a new user
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (sebuf.http.config) = {
      path: "/users"
    };
  }
  
  // Get user by ID
  rpc GetUser(GetUserRequest) returns (User) {
    option (sebuf.http.config) = {
      path: "/users/get"
    };
  }
  
  // List all users
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse) {
    option (sebuf.http.config) = {
      path: "/users"
    };
  }
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
  string department = 3;
}

message GetUserRequest {
  string id = 1;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
  string department_filter = 3;
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
  string department = 4;
  int64 created_at = 5;
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

### 2. Generate HTTP Code

```bash
protoc --go_out=. --go_opt=module=github.com/yourorg/userapi \
       --go-http_out=. \
       user_service.proto
```

### 3. Implement Your Service

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    
    "github.com/yourorg/userapi"
)

// Implement the generated service interface
type UserServiceImpl struct {
    users map[string]*userapi.User
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, req *userapi.CreateUserRequest) (*userapi.User, error) {
    user := &userapi.User{
        Id:         generateID(),
        Name:       req.Name,
        Email:      req.Email,
        Department: req.Department,
        CreatedAt:  time.Now().Unix(),
    }
    
    s.users[user.Id] = user
    return user, nil
}

func (s *UserServiceImpl) GetUser(ctx context.Context, req *userapi.GetUserRequest) (*userapi.User, error) {
    user, exists := s.users[req.Id]
    if !exists {
        return nil, fmt.Errorf("user not found: %s", req.Id)
    }
    return user, nil
}

func (s *UserServiceImpl) ListUsers(ctx context.Context, req *userapi.ListUsersRequest) (*userapi.ListUsersResponse, error) {
    var filteredUsers []*userapi.User
    
    for _, user := range s.users {
        if req.DepartmentFilter == "" || user.Department == req.DepartmentFilter {
            filteredUsers = append(filteredUsers, user)
        }
    }
    
    return &userapi.ListUsersResponse{
        Users:      filteredUsers,
        TotalCount: int32(len(filteredUsers)),
    }, nil
}

func main() {
    // Create service implementation
    userService := &UserServiceImpl{
        users: make(map[string]*userapi.User),
    }
    
    // Register HTTP handlers
    mux := http.NewServeMux()
    err := userapi.RegisterUserServiceServer(userService, userapi.WithMux(mux))
    if err != nil {
        log.Fatal(err)
    }
    
    // Start server
    fmt.Println("Server starting on :8080")
    fmt.Println("Endpoints:")
    fmt.Println("  POST   /api/v1/users      - Create user")
    fmt.Println("  POST   /api/v1/users/get - Get user")
    fmt.Println("  POST   /api/v1/users      - List users")
    
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### 4. Test Your API

```bash
# Create a user (JSON)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com", 
    "department": "Engineering"
  }'

# Get a user  
curl -X POST http://localhost:8080/api/v1/users/get \
  -H "Content-Type: application/json" \
  -d '{"id": "123"}'

# List users with filter
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "page_size": 10,
    "department_filter": "Engineering"
  }'
```

## HTTP Annotations

Control HTTP routing and behavior using protobuf options:

### Service-Level Configuration

```protobuf
service MyService {
  option (sebuf.http.service_config) = {
    base_path: "/api/v1";
  };
  
  // ... methods
}
```

**Options:**
- `base_path`: URL prefix for all methods in this service

### Method-Level Configuration  

```protobuf
rpc CreateUser(CreateUserRequest) returns (User) {
  option (sebuf.http.config) = {
    path: "/users";
  };
}
```

**Options:**
- `path`: Custom HTTP path for this method

### Path Resolution

The final HTTP path is determined by:

1. **Custom path with base path**: `base_path + path`
   ```protobuf
   // Results in: POST /api/v1/users
   option (sebuf.http.service_config) = { base_path: "/api/v1" };
   option (sebuf.http.config) = { path: "/users" };
   ```

2. **Custom path only**: Uses `path` directly
   ```protobuf
   // Results in: POST /custom/endpoint  
   option (sebuf.http.config) = { path: "/custom/endpoint" };
   ```

3. **Base path only**: Generates path from method name
   ```protobuf
   // Results in: POST /api/v1/create_user
   option (sebuf.http.service_config) = { base_path: "/api/v1" };
   ```

4. **Default**: Uses package and method name
   ```protobuf
   // Results in: POST /userapi/create_user (no annotations)
   ```

## Generated Code Structure

The plugin generates three files for each protobuf file containing services:

### 1. Main HTTP File (`*_http.pb.go`)

**Service Interface:**
```go
// UserServiceServer is the server API for UserService service.
type UserServiceServer interface {
    CreateUser(context.Context, *CreateUserRequest) (*User, error)
    GetUser(context.Context, *GetUserRequest) (*User, error)
    ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error)
}
```

**Registration Function:**
```go
// RegisterUserServiceServer registers HTTP handlers for UserService
func RegisterUserServiceServer(server UserServiceServer, opts ...ServerOption) error
```

### 2. Binding File (`*_http_binding.pb.go`)

Contains middleware and request/response handling:

- **Content Type Support** - JSON and binary protobuf
- **Request Binding** - Automatic deserialization from HTTP requests  
- **Response Marshaling** - Automatic serialization to HTTP responses
- **Error Handling** - Structured error responses

### 3. Config File (`*_http_config.pb.go`)

Provides configuration options:

```go
// ServerOption configures HTTP server behavior
type ServerOption func(c *serverConfiguration)

// WithMux configures a custom HTTP ServeMux
func WithMux(mux *http.ServeMux) ServerOption
```

## Framework Integration

The generated code works with any Go HTTP framework:

### Standard Library

```go
mux := http.NewServeMux()
userapi.RegisterUserServiceServer(userService, userapi.WithMux(mux))
http.ListenAndServe(":8080", mux)
```

### Gin Framework

```go
import "github.com/gin-gonic/gin"

r := gin.Default()

// Convert gin router to http.ServeMux for sebuf
mux := http.NewServeMux()
userapi.RegisterUserServiceServer(userService, userapi.WithMux(mux))

// Mount sebuf handlers on gin
r.Any("/api/*path", gin.WrapH(mux))

r.Run(":8080")
```

### Echo Framework

```go
import "github.com/labstack/echo/v4"

e := echo.New()

// Create dedicated mux for sebuf
mux := http.NewServeMux() 
userapi.RegisterUserServiceServer(userService, userapi.WithMux(mux))

// Mount on echo
e.Any("/api/*", echo.WrapHandler(mux))

e.Start(":8080")
```

### Chi Router

```go
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()

// sebuf handlers
mux := http.NewServeMux()
userapi.RegisterUserServiceServer(userService, userapi.WithMux(mux))

// Mount on chi
r.Mount("/api/", http.StripPrefix("/api", mux))

http.ListenAndServe(":8080", r)
```

## Request/Response Handling

### Content Type Support

The generated handlers automatically support multiple content types:

**JSON (default):**
```bash
curl -X POST /api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "email": "john@example.com"}'
```

**Binary Protobuf:**
```bash
# Using protobuf binary format
curl -X POST /api/v1/users \
  -H "Content-Type: application/x-protobuf" \
  --data-binary @user_request.pb
```

### Request Processing Flow

1. **Content Type Detection** - Checks `Content-Type` header
2. **Request Binding** - Deserializes based on content type
3. **Validation** - Protobuf validation (required fields, types)
4. **Service Call** - Invokes your service implementation
5. **Response Marshaling** - Serializes response in same format as request

### Error Handling

Generated handlers provide structured error responses:

```go
// Service implementation error
func (s *UserService) GetUser(ctx context.Context, req *GetUserRequest) (*User, error) {
    if req.Id == "" {
        return nil, fmt.Errorf("user ID is required")
    }
    
    user, exists := s.users[req.Id]
    if !exists {
        return nil, fmt.Errorf("user not found: %s", req.Id)
    }
    
    return user, nil
}
```

**Error Response (JSON):**
```json
{
  "error": "user not found: 123",
  "status": 500
}
```

## Configuration Options

### Server Options

```go
// Use custom ServeMux
mux := http.NewServeMux()
RegisterUserServiceServer(service, WithMux(mux))

// Use default ServeMux (http.DefaultServeMux)
RegisterUserServiceServer(service)
```

### Custom Middleware

Add middleware by wrapping the generated handlers:

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

// Apply middleware to the mux
mux := http.NewServeMux()
RegisterUserServiceServer(service, WithMux(mux))

// Wrap with middleware  
handler := loggingMiddleware(mux)
http.ListenAndServe(":8080", handler)
```

## Advanced Examples

### Authentication Service

```protobuf
service AuthService {
  option (sebuf.http.service_config) = {
    base_path: "/auth"
  };
  
  rpc Login(LoginRequest) returns (LoginResponse) {
    option (sebuf.http.config) = { path: "/login" };
  }
  
  rpc RefreshToken(RefreshRequest) returns (TokenResponse) {
    option (sebuf.http.config) = { path: "/refresh" };
  }
  
  rpc Logout(LogoutRequest) returns (LogoutResponse) {
    option (sebuf.http.config) = { path: "/logout" };
  }
}
```

### E-commerce API

```protobuf
service ProductService {
  option (sebuf.http.service_config) = {
    base_path: "/api/v1/products"
  };
  
  rpc CreateProduct(CreateProductRequest) returns (Product) {
    option (sebuf.http.config) = { path: "" };  // POST /api/v1/products
  }
  
  rpc GetProduct(GetProductRequest) returns (Product) {
    option (sebuf.http.config) = { path: "/get" };  // POST /api/v1/products/get
  }
  
  rpc SearchProducts(SearchRequest) returns (SearchResponse) {
    option (sebuf.http.config) = { path: "/search" };  // POST /api/v1/products/search
  }
}
```

### File Upload Service

```protobuf
service FileService {
  rpc UploadFile(UploadFileRequest) returns (UploadFileResponse) {
    option (sebuf.http.config) = { path: "/files/upload" };
  }
}

message UploadFileRequest {
  string filename = 1;
  bytes content = 2;
  string content_type = 3;
  map<string, string> metadata = 4;
}
```

## Best Practices

### 1. Consistent URL Design

```protobuf
// Good: RESTful paths
service UserService {
  option (sebuf.http.service_config) = { base_path: "/api/v1" };
  
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (sebuf.http.config) = { path: "/users" };
  }
  
  rpc GetUser(GetUserRequest) returns (User) {
    option (sebuf.http.config) = { path: "/users/get" };
  }
  
  rpc UpdateUser(UpdateUserRequest) returns (User) {
    option (sebuf.http.config) = { path: "/users/update" };
  }
  
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse) {
    option (sebuf.http.config) = { path: "/users/delete" };
  }
}
```

### 2. Error Handling Strategy

```go
type UserService struct {
    repo UserRepository
}

func (s *UserService) GetUser(ctx context.Context, req *GetUserRequest) (*User, error) {
    // Validate input
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }
    
    // Business logic
    user, err := s.repo.FindByID(req.Id)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        return nil, status.Error(codes.Internal, "failed to retrieve user")
    }
    
    return user, nil
}
```

### 3. Request Validation

```protobuf
message CreateUserRequest {
  string name = 1;           // Validate: non-empty, max length
  string email = 2;          // Validate: email format
  string department = 3;     // Validate: enum or predefined list
  repeated string roles = 4; // Validate: valid role names
}
```

```go
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    // Custom validation beyond protobuf
    if err := validateCreateUserRequest(req); err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }
    
    // Business logic...
}

func validateCreateUserRequest(req *CreateUserRequest) error {
    if req.Name == "" {
        return fmt.Errorf("name is required")
    }
    
    if len(req.Name) > 100 {
        return fmt.Errorf("name too long (max 100 characters)")
    }
    
    if !isValidEmail(req.Email) {
        return fmt.Errorf("invalid email format")
    }
    
    return nil
}
```

### 4. Testing Generated Handlers

```go
func TestUserServiceHTTP(t *testing.T) {
    // Create service implementation
    service := &UserServiceImpl{
        users: make(map[string]*User),
    }
    
    // Setup HTTP handlers
    mux := http.NewServeMux()
    err := RegisterUserServiceServer(service, WithMux(mux))
    require.NoError(t, err)
    
    // Test server
    server := httptest.NewServer(mux)
    defer server.Close()
    
    t.Run("CreateUser", func(t *testing.T) {
        reqBody := `{
            "name": "Test User",
            "email": "test@example.com",
            "department": "Engineering"
        }`
        
        resp, err := http.Post(
            server.URL+"/api/v1/users",
            "application/json",
            strings.NewReader(reqBody),
        )
        require.NoError(t, err)
        defer resp.Body.Close()
        
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        
        var user User
        err = json.NewDecoder(resp.Body).Decode(&user)
        require.NoError(t, err)
        
        assert.Equal(t, "Test User", user.Name)
        assert.Equal(t, "test@example.com", user.Email)
    })
}
```

## Troubleshooting

### Common Issues

#### 1. Plugin Not Found
```
protoc-gen-go-http: program not found or is not executable
```

**Solution:**
```bash
# Ensure plugin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Reinstall plugin
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
```

#### 2. Import Errors
```
cannot find package "github.com/SebastienMelki/sebuf/http"
```

**Solution:**
Ensure the annotations are available:
```bash
# Option 1: Use from Buf Schema Registry
echo 'deps: [buf.build/sebmelki/sebuf]' >> buf.yaml

# Option 2: Include in your module
go get github.com/SebastienMelki/sebuf/http
```

#### 3. No Handlers Generated
Check that:
- Your proto file contains `service` definitions
- Services have at least one `rpc` method
- You're using the correct plugin (`--go-http_out`)

#### 4. Path Conflicts
```
pattern /api/v1/users conflicts with pattern /api/
```

**Solution:**
Ensure path patterns don't overlap:
```protobuf
// Good: Specific paths
option (sebuf.http.config) = { path: "/users" };
option (sebuf.http.config) = { path: "/users/get" };

// Avoid: Overlapping patterns  
option (sebuf.http.config) = { path: "/users" };
option (sebuf.http.config) = { path: "/users/" };  // Conflicts
```

### Getting Help

- **Examples**: Check the [examples directory](../examples/)
- **Test Cases**: Review tests in `internal/httpgen/`
- **Issues**: File a GitHub issue with your proto definition
- **Discussions**: Join GitHub Discussions for questions

## Integration with Other sebuf Tools

### With Oneof Helpers

```go
// Use oneof helpers in HTTP handlers
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
    // Create requests easily with generated helpers
    var authReq *AuthRequest
    
    if email := getEmailFromHTTPContext(ctx); email != "" {
        authReq = NewAuthRequestEmail(email.Email, email.Password)
    } else if token := getTokenFromHTTPContext(ctx); token != "" {
        authReq = NewAuthRequestToken(token.Token)
    }
    
    return s.authenticate(authReq)
}
```

### With OpenAPI Generation

Generate both HTTP handlers and OpenAPI documentation:

```bash
# Generate both HTTP handlers and OpenAPI spec
protoc --go_out=. --go_opt=module=github.com/yourorg/api \
       --go-http_out=. \
       --openapiv3_out=./docs \
       api.proto
```

The OpenAPI spec will automatically reflect your HTTP annotations and routing.

---

**Next:** Learn how to generate comprehensive API documentation with [OpenAPI Generation](./openapi-generation.md)

**See also:**
- [Getting Started Guide](./getting-started.md)
- [Oneof Helpers](./oneof-helpers.md)
- [Examples](./examples/)