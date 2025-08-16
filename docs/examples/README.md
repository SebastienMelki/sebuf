# sebuf Examples

We're building practical examples to help you get started with sebuf.

## Quick Demo

Want to try sebuf right now? Here's the simplest possible example:

```bash
# Install the tools
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest

# Create a simple service definition
cat > api.proto << 'EOF'
syntax = "proto3";
package demo;

import "sebuf/http/annotations.proto";

service GreetService {
  option (sebuf.http.service_config) = { base_path: "/api" };
  
  rpc SayHello(HelloRequest) returns (HelloResponse) {
    option (sebuf.http.config) = { path: "/hello" };
  }
}

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
}
EOF

# Generate everything
protoc --go_out=. --go-oneof-helper_out=. --go-http_out=. --openapiv3_out=. api.proto

# That's it! You now have HTTP handlers, helper functions, and OpenAPI docs.
```

## Working Examples

Currently available:
- **Simple API Tutorial** - Step-by-step guide above ✅
- **More examples coming soon** - Help us build them! 🚧

## Want to Contribute?

Help us create examples that matter! We need:
- Real-world API scenarios
- Framework integration guides  
- Deployment examples

See [Contributing Guidelines](../../CONTRIBUTING.md) to get started.

---

**Keep it simple!** 🚀