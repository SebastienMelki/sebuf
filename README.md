# sebuf

> **Modern protobuf development for HTTP APIs**

sebuf is a comprehensive toolkit that bridges protobuf definitions and HTTP API development, providing type-safe code generation, automatic documentation, and developer-friendly utilities.

[![Go Version](https://img.shields.io/github/go-mod/go-version/SebastienMelki/sebuf)](https://golang.org/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/SebastienMelki/sebuf/ci.yml?branch=main)](https://github.com/SebastienMelki/sebuf/actions)
[![Test Coverage](https://img.shields.io/badge/coverage-85%25-green)](./coverage/)
[![Go Report Card](https://goreportcard.com/badge/github.com/SebastienMelki/sebuf)](https://goreportcard.com/report/github.com/SebastienMelki/sebuf)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ What makes sebuf special?

- **🚀 Zero-dependency HTTP generation** - Build HTTP APIs directly from protobuf without additional runtime dependencies
- **📖 Automatic OpenAPI documentation** - Generate comprehensive OpenAPI v3 specifications that stay in sync
- **🛠️ Eliminate boilerplate** - Smart helpers for complex protobuf patterns like oneof fields
- **🔧 Framework agnostic** - Works with any Go HTTP framework (Gin, Echo, Chi, standard library)
- **📱 Web & mobile friendly** - JSON over HTTP APIs perfect for frontend applications

## 🚀 Quick Start

### Installation

```bash
# Install all protoc plugins
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest  
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
```

### Your first API in 60 seconds

1. **Define your service**:
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
  oneof auth_method {
    EmailAuth email = 1;
  }
}

message EmailAuth {
  string email = 1;
  string password = 2;
}

message User {
  string id = 1;
  string email = 2;
}
```

2. **Generate everything**:

#### Option A: Using Buf (Recommended)

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
    opt: 
      - paths=source_relative
  - local: protoc-gen-go-oneof-helper
    out: .
    opt: 
      - paths=source_relative
  - local: protoc-gen-go-http
    out: .
    opt: 
      - paths=source_relative
  - local: protoc-gen-openapiv3
    out: .
```

Generate code:
```bash
# First time: fetch dependencies
buf dep update

# Generate code
buf generate
```

#### Option B: Using protoc
```bash
# Clone sebuf for proto files
git clone https://github.com/SebastienMelki/sebuf.git

# Generate with correct paths
protoc --go_out=. --go-oneof-helper_out=. --go-http_out=. --openapiv3_out=. \
       --proto_path=. \
       --proto_path=./sebuf/proto \
       api.proto
```

3. **Use it**:
```go
// Use oneof helpers 
req := api.NewCreateUserRequestEmail("user@example.com", "secret")

// Register HTTP handlers
mux := http.NewServeMux()
api.RegisterUserServiceServer(userService, api.WithMux(mux))
http.ListenAndServe(":8080", mux)
```

**Done!** You now have HTTP handlers, OpenAPI docs, and helper functions.

## 🧰 Three Simple Tools

### 🔧 Oneof Helpers
Turns this:
```go
req := &CreateUserRequest{
    AuthMethod: &CreateUserRequest_Email{
        Email: &EmailAuth{Email: "user@example.com", Password: "secret"},
    },
}
```
Into this:
```go
req := NewCreateUserRequestEmail("user@example.com", "secret")
```

### 🌐 HTTP Handlers  
Generates complete HTTP servers from protobuf services. No manual routing needed.

### 📚 OpenAPI Docs
Auto-generates API documentation that stays in sync with your code.

## 📖 Documentation

- **[Getting Started Guide](./docs/getting-started.md)** - Complete tutorial from protobuf to deployed API
- **[Oneof Helpers](./docs/oneof-helpers.md)** - Eliminate boilerplate for complex protobuf types
- **[HTTP Generation](./docs/http-generation.md)** - Build HTTP APIs from protobuf services
- **[OpenAPI Generation](./docs/openapi-generation.md)** - Auto-generate API documentation
- **[Examples](./docs/examples/)** - Complete project templates and real-world examples
- **[Architecture](./docs/architecture.md)** - Technical deep-dive for contributors

## 🎯 Use Cases

### REST APIs with Type Safety
Build traditional REST APIs while leveraging protobuf's strong typing and code generation.

### Frontend API Integration  
Generate TypeScript/JavaScript clients from the same protobuf definitions used by your Go backend.

### Microservices Communication
Use HTTP for external APIs while maintaining protobuf contracts for internal service communication.

### API Documentation
Keep your API documentation perfectly synchronized with your implementation.

## 🤝 Why sebuf?

- ✅ **Type safety** from protobuf definitions
- ✅ **Direct HTTP** - no gRPC dependencies  
- ✅ **Auto-generated docs** that never go stale
- ✅ **Works with any framework** - Gin, Echo, Chi, standard library
- ✅ **Zero runtime dependencies**

## 🛠️ Development

```bash
git clone https://github.com/SebastienMelki/sebuf.git
cd sebuf
make test
```

## 🗺️ Roadmap

- ✅ **Core toolkit** - HTTP handlers, oneof helpers, OpenAPI generation
- 🚧 **Client generation** - TypeScript/JavaScript clients
- 📋 **Enhanced features** - Middleware, authentication, validation

## 🤝 Contributing

We welcome contributions! Whether it's bug reports, feature requests, documentation improvements, or code contributions.

- **[Contributing Guide](./CONTRIBUTING.md)** - How to get started
- **[Architecture Overview](./docs/architecture.md)** - Understanding the codebase
- **[Issue Templates](./github/ISSUE_TEMPLATE/)** - Report bugs or request features

## 📄 License

sebuf is released under the [MIT License](./LICENSE).

## 🙏 Acknowledgments

Built with:
- [protogen](https://pkg.go.dev/google.golang.org/protobuf/compiler/protogen) - Official protoc plugin framework
- [libopenapi](https://github.com/pb33f/libopenapi) - OpenAPI v3 document generation
- [Protocol Buffers](https://protobuf.dev/) - The foundation that makes it all possible

---

<div align="center">

**[Getting Started](./docs/getting-started.md)** • **[Documentation](./docs/)** • **[Examples](./docs/examples/)** • **[Contributing](./CONTRIBUTING.md)**

Made with ❤️ by [Sebastien](https://github.com/SebastienMelki)

</div>