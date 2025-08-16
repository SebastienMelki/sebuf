# sebuf Examples

We're building practical examples to help you get started with sebuf.

## Quick Demo

Want to try sebuf right now? Here's the simplest possible example:

### Using Buf (Recommended - Easiest)

```bash
# Install Buf if you haven't already
brew install bufbuild/buf/buf  # Or see https://docs.buf.build/installation

# Install the sebuf plugins
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest

# Step 1: Create your API definition
```

Create `api.proto`:
```protobuf
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
```

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

```bash

# First time: fetch dependencies
buf dep update

# Generate everything
buf generate

# Update Go modules
go mod tidy

# That's it! You now have HTTP handlers, helper functions, and OpenAPI docs.
```

### Using protoc (Traditional)

```bash
# Install the tools
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Clone sebuf to get the proto files
git clone https://github.com/SebastienMelki/sebuf.git

# Create your service definition
# Use the same api.proto file from above

# Generate with correct proto paths
protoc --go_out=. --go-oneof-helper_out=. --go-http_out=. --openapiv3_out=. \
       --proto_path=. \
       --proto_path=./sebuf/proto \
       api.proto
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