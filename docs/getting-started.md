# Getting Started with sebuf

> **Build your first HTTP API from protobuf in under 5 minutes**

This guide shows you how to transform protobuf service definitions into production-ready HTTP APIs with automatic documentation and validation.

## 🚀 Quick start

**Already have Go and want to see it work immediately?**

```bash
# Try the working example
git clone https://github.com/SebastienMelki/sebuf.git
cd sebuf/examples/simple-api
make demo
```

This starts a complete HTTP API with authentication, validation, and OpenAPI docs.

## Step-by-step tutorial

### 1. Install tools

```bash
# Install Buf (recommended)
brew install bufbuild/buf/buf

# Install sebuf
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
```

### 2. Create your project

```bash
# Create project directory
mkdir myapi && cd myapi
go mod init github.com/yourorg/myapi
```

### 3. Create your API

Create `api.proto`:
```protobuf
syntax = "proto3";
package api;

import "sebuf/http/annotations.proto";

service UserService {
  option (sebuf.http.service_config) = { base_path: "/api/v1" };
  
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (sebuf.http.config) = { path: "/users" };
  }
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}
```

### 4. Configure generation

Create `buf.yaml`:
```yaml
version: v2
deps:
  - buf.build/sebmelki/sebuf
```

Create `buf.gen.yaml`:
```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: .
    opt: paths=source_relative
  - local: protoc-gen-go-http
    out: .
  - local: protoc-gen-openapiv3
    out: .
```

### 5. Generate everything

```bash
buf dep update
buf generate
go mod tidy
```

This creates:
- `api.pb.go` - Protobuf structs
- `api_http*.pb.go` - HTTP handlers
- `api.yaml` - OpenAPI documentation

### 6. Write your server

Create `main.go`:
```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/yourorg/myapi"
)

type UserService struct {
    users map[string]*api.User
    nextId int
}

func (s *UserService) CreateUser(ctx context.Context, req *api.CreateUserRequest) (*api.User, error) {
    s.nextId++
    user := &api.User{
        Id: fmt.Sprintf("%d", s.nextId), 
        Name: req.Name,
        Email: req.Email,
    }
    s.users[user.Id] = user
    return user, nil
}

func main() {
    service := &UserService{users: make(map[string]*api.User)}
    mux := http.NewServeMux()
    
    api.RegisterUserServiceServer(service, api.WithMux(mux))
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### 7. Test it

```bash
go run main.go
```

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "email": "john@example.com"}'
```

**That's it!** You now have a working HTTP API with OpenAPI docs.

## What you just built

- ✅ **HTTP endpoints** from protobuf services
- ✅ **JSON and binary** protobuf support  
- ✅ **OpenAPI documentation** (`api.yaml`)
- ✅ **Type safety** throughout your API
- ✅ **Zero runtime dependencies**

## Next steps

- **[Complete Tutorial](../examples/simple-api/)** - See authentication, validation, and more
- **[HTTP Generation Guide](./http-generation.md)** - Advanced HTTP features
- **[OpenAPI Guide](./openapi-generation.md)** - Documentation customization
- **[Validation Guide](./validation.md)** - Automatic request validation

## Need help?

- Try the [working example](../examples/simple-api/) first
- Check the individual guides for each tool
- Open an issue if you get stuck