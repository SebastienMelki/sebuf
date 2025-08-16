# Simple API Example

This example demonstrates how to use all three sebuf plugins to build a complete HTTP API with generated helpers and OpenAPI documentation.

## Quick Start

### Step 1: Install Dependencies

```bash
# Install Buf (if not already installed)
brew install bufbuild/buf/buf

# Install sebuf plugins
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
```

### Step 2: Generate Code

```bash
# First time: fetch the sebuf dependency
buf dep update

# Generate all the code
buf generate

# Update Go dependencies
go mod tidy
```

### Step 3: Run the Server

```bash
go run main.go
```

### Step 4: Test the API

```bash
# Create a user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'

# With authentication (using oneof helper)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": {
      "email": "john@example.com",
      "password": "secret123"
    }
  }'
```

## Files in This Example

- `buf.yaml` - Buf module configuration with sebuf dependency
- `buf.gen.yaml` - Code generation configuration for all three plugins
- `api.proto` - Service definition with HTTP annotations
- `main.go` - Server implementation using generated code
- `api.yaml` - Generated OpenAPI documentation (after running `buf generate`)

## What This Demonstrates

1. **Oneof Helpers**: The login endpoint uses a oneof field for different authentication methods. The generated helpers make it easy to construct these requests.

2. **HTTP Generation**: The protobuf service is automatically exposed as HTTP endpoints with JSON support.

3. **OpenAPI Documentation**: Complete API documentation is generated and kept in sync with your protobuf definitions.

## Next Steps

- Try modifying the proto file and regenerating
- Add more services and messages
- Integrate with your favorite Go HTTP framework
- Import the OpenAPI spec into Postman or Insomnia