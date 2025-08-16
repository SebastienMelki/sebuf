# sebuf

> **Build HTTP APIs from protobuf definitions**

Transform your protobuf services into production-ready HTTP APIs with automatic documentation and helper functions.

[![Go Version](https://img.shields.io/github/go-mod/go-version/SebastienMelki/sebuf)](https://golang.org/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/SebastienMelki/sebuf/ci.yml?branch=main)](https://github.com/SebastienMelki/sebuf/actions)
[![Test Coverage](https://img.shields.io/badge/coverage-85%25-green)](./coverage/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 Try it in 30 seconds

```bash
# Clone and run the working example
git clone https://github.com/SebastienMelki/sebuf.git
cd sebuf/examples/simple-api
make demo
```

This starts a working HTTP API with JSON endpoints, OpenAPI docs, and helper functions - all generated from a simple `.proto` file.

## What you get

- **HTTP handlers** from protobuf services (JSON + binary support)
- **Automatic request validation** using protovalidate with sebuf.validate annotations
- **OpenAPI v3.1 docs** that stay in sync with your code  
- **Helper functions** that eliminate protobuf boilerplate
- **Zero runtime dependencies** - works with any Go HTTP framework

## How it works

From this protobuf definition:
```protobuf
service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
}

message CreateUserRequest {
  // Automatic validation with sebuf.validate
  string name = 1 [(sebuf.validate.field).string = {
    min_len: 2, max_len: 100
  }];
  string email = 2 [(sebuf.validate.field).string.email = true];
  
  oneof auth_method {
    EmailAuth email = 3;
    TokenAuth token = 4;
  }
}
```

sebuf generates:
```go
// HTTP handlers with automatic validation
api.RegisterUserServiceServer(userService, api.WithMux(mux))

// Helper functions  
req := api.NewCreateUserRequestEmail("user@example.com", "secret")
req := api.NewCreateUserRequestToken("auth-token")

// Validation happens automatically - returns HTTP 400 for invalid requests
// OpenAPI docs (api.yaml) - automatically generated with validation rules
```

## Quick setup

```bash
# Install the tools
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest  
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest

# Try the complete example
cd examples/simple-api && make demo
```

## Next steps

- **[Complete Tutorial](./examples/simple-api/)** - Full walkthrough with working code
- **[Documentation](./docs/)** - Comprehensive guides and API reference  
- **[More Examples](./docs/examples/)** - Additional patterns and use cases

## What's this good for?

- **Web & mobile APIs** - JSON/HTTP endpoints from protobuf definitions
- **API documentation** - OpenAPI specs that never get out of sync
- **Type-safe development** - Leverage protobuf's type system for HTTP APIs
- **Client generation** - Generate clients for any language from your API spec

## Built on Great Tools

sebuf stands on the shoulders of giants, integrating with an incredible ecosystem:

- **[Protocol Buffers](https://protobuf.dev/)** by Google - The foundation for everything
- **[protovalidate](https://github.com/bufbuild/protovalidate)** by Buf - Powers our automatic validation  
- **[Buf CLI](https://buf.build/)** - Modern protobuf tooling and dependency management
- **[OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0)** - Industry standard API documentation
- **[Common Expression Language (CEL)](https://github.com/google/cel-go)** by Google - Flexible validation rules

We're grateful to all maintainers of these projects that make sebuf possible.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](./LICENSE)