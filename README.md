<div align="center">
  <img src="docs/sebuf-logo.png" alt="sebuf logo" width="200">
  
  
  > **Build HTTP APIs from protobuf definitions**
  
  Transform your protobuf services into production-ready HTTP APIs with automatic documentation and validation.
</div>

<div align="center">

[![Go Version](https://img.shields.io/github/go-mod/go-version/SebastienMelki/sebuf)](https://golang.org/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/SebastienMelki/sebuf/ci.yml?branch=main)](https://github.com/SebastienMelki/sebuf/actions)
[![Test Coverage](https://img.shields.io/badge/coverage-85%25-green)](./coverage/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

## 🚀 Try it in 30 seconds

```bash
# Clone and run the working example
git clone https://github.com/SebastienMelki/sebuf.git
cd sebuf/examples/simple-api
make demo
```

This starts a working HTTP API with JSON endpoints and OpenAPI docs - all generated from a simple `.proto` file.

## What you get

**Five generators from one `.proto` file:**

| Generator | Output |
|-----------|--------|
| `protoc-gen-go-http` | Go HTTP handlers with routing, request binding, and validation |
| `protoc-gen-go-client` | Type-safe Go HTTP clients with functional options and per-call customization |
| `protoc-gen-ts-client` | TypeScript HTTP clients with full type safety, header helpers, and error handling |
| `protoc-gen-ts-server` | TypeScript HTTP servers with request and header validation, using the Web Fetch API — works with Node, Deno, Bun, Cloudflare Workers |
| `protoc-gen-openapiv3` | OpenAPI v3.1 specs that stay in sync with your code, one file per service |

**Validation and error handling — built in, not bolted on:**

- Automatic request body validation via [buf.validate](https://github.com/bufbuild/protovalidate) annotations
- HTTP header validation with type checking and format validation (UUID, email, datetime)
- Structured error responses with field-level details in JSON or protobuf
- Proto messages ending with "Error" automatically implement Go's `errors.As()` / `errors.Is()`

**Developer experience:**

- Mock server generation with realistic field examples for rapid prototyping
- Zero runtime dependencies — works with any Go HTTP framework

## How it works

From this protobuf definition:
```protobuf
service UserService {
  // Header validation at service level
  option (sebuf.http.service_headers) = {
    required_headers: [{
      name: "X-API-Key"
      type: "string"
      format: "uuid"
      required: true
    }]
  };
  
  rpc CreateUser(CreateUserRequest) returns (User);
}

message CreateUserRequest {
  // Automatic validation with buf.validate
  string name = 1 [
    (buf.validate.field).string = {
      min_len: 2, max_len: 100
    },
    (sebuf.http.field_examples) = {
      values: ["Alice Johnson", "Bob Smith", "Charlie Davis"]
    }
  ];
  string email = 2 [
    (buf.validate.field).string.email = true,
    (sebuf.http.field_examples) = {
      values: ["alice@example.com", "bob@example.com"]
    }
  ];
  
  oneof auth_method {
    EmailAuth email = 3;
    TokenAuth token = 4;
  }
}
```

sebuf generates:

**Go** — handlers, clients, and mocks:
```go
// HTTP handlers with automatic validation (headers + body)
api.RegisterUserServiceServer(userService, api.WithMux(mux))

// Type-safe HTTP client with functional options
client := api.NewUserServiceClient("http://localhost:8080",
    api.WithUserServiceAPIKey("your-api-key"),
)
user, err := client.CreateUser(ctx, req)

// Mock server with realistic data
mockService := api.NewMockUserServiceServer()
api.RegisterUserServiceServer(mockService, api.WithMux(mux))
```

**TypeScript** — clients and servers:
```typescript
// HTTP client with full type safety
const client = new UserServiceClient("http://localhost:8080", {
  apiKey: "your-api-key",
});
const user = await client.createUser({ name: "John", email: "john@example.com" });

// HTTP server (framework-agnostic, Web Fetch API)
const routes = createUserServiceRoutes(handler);
// Wire into any framework: Bun.serve, Deno.serve, Express, Hono, etc.
```

**OpenAPI** — validation rules, headers, and examples included automatically:
```
UserService.openapi.yaml
```

## Quick setup

```bash
# Install the tools
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-client@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-ts-client@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-ts-server@latest

# Try the complete example
cd examples/simple-api && make demo
```

## Used in Production

<table>
<tr>
<td width="50%" valign="top">

### [WorldMonitor](https://github.com/koala73/worldmonitor)
Real-time global intelligence dashboard tracking seismology, cyber threats, markets, aviation, and more. Full-stack TypeScript with sebuf — generated TS clients, TS server handlers, and OpenAPI docs all from the same proto definitions. Actively battle-testing sebuf across every generator. See [the integration PR](https://github.com/koala73/worldmonitor/pull/106).

</td>
<td width="50%" valign="top">

### [alpaca-go](https://github.com/SebastienMelki/alpaca-go)
Type-safe Go SDK for the [Alpaca Trading API](https://alpaca.markets/) — 100+ endpoints across trading, market data, brokerage, and auth, all generated from protobuf definitions. The entire Alpaca REST API modeled as proto files with sebuf annotations. Clients, validation, and OpenAPI docs that can never drift from the actual API.

</td>
</tr>
<tr>
<td width="50%" valign="top">

### [Anghami](https://www.anghami.com/) & [OSN+](https://osnplus.com/)
sebuf powers API services at [Anghami](https://www.anghami.com/), the leading music streaming platform in the Middle East and North Africa, and at [OSN+](https://osnplus.com/), the region's premium streaming service featuring HBO, Paramount+, and OSN Originals.

</td>
<td width="50%" valign="top">

### [Sarwa](https://www.sarwa.co/)
sebuf is used at [Sarwa](https://www.sarwa.co/), the fastest-growing investment and personal finance platform in the MENA region, powering type-safe API contracts across their trading, investing, and savings services.

</td>
</tr>
</table>

## Next steps

- **[Complete Tutorial](./examples/simple-api/)** - Full walkthrough with working code
- **[Documentation](./docs/)** - Comprehensive guides and API reference
- **[More Examples](./docs/examples/)** - Additional patterns and use cases

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

## Star History

<a href="https://star-history.com/#SebastienMelki/sebuf&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=SebastienMelki/sebuf&type=Date&theme=dark" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=SebastienMelki/sebuf&type=Date" />
 </picture>
</a>
