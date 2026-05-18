# Sebuf Rust Plugins

This directory contains Rust implementations of the sebuf protobuf plugins, providing the same functionality as the Go versions but targeting the Rust ecosystem.

## Overview

The Rust plugins generate Rust code from protobuf definitions, enabling type-safe HTTP API development using modern Rust frameworks like Axum.

### Plugins

- **`protoc-gen-rust-oneof-helper`** - Generates convenience constructors for oneof fields
- **`protoc-gen-rust-http`** - Generates HTTP handlers using Axum framework  
- **`protoc-gen-rust-openapiv3`** - Generates OpenAPI 3.1 specifications

### Core Library

- **`sebuf-core`** - Shared utilities for protobuf parsing, code generation, and plugin infrastructure

## Building

```bash
# Build all Rust plugins
make build-rust

# Or build with cargo directly
cargo build --release

# Install to PATH
make install-rust
```

## Testing  

```bash
# Run comprehensive test suite
make test-rust

# Run tests without coverage (faster)
make test-rust-fast

# Update golden test files
make test-rust-update-golden
```

The test suite includes:

- **Unit tests** for core utilities
- **Integration tests** for each plugin
- **Golden file tests** for regression detection

## Usage

Once built, the plugins work with `protoc` like any other protobuf plugin:

```bash
# Generate oneof helpers
protoc --rust-oneof-helper_out=./generated \
       --proto_path=./proto \
       user.proto

# Generate HTTP handlers
protoc --rust-http_out=./generated \
       --proto_path=./proto \
       user.proto

# Generate OpenAPI specs  
protoc --rust-openapiv3_out=./docs \
       --proto_path=./proto \
       user.proto
```

## Generated Code Examples

### Oneof Helpers

For a protobuf with oneof fields:

```protobuf
message LoginRequest {
  oneof auth_method {
    EmailAuth email = 1;
    PhoneAuth phone = 2;
  }
  
  message EmailAuth {
    string email = 1;
    string password = 2;
  }
  
  message PhoneAuth {
    string phone = 1;
    string code = 2;
  }
}
```

Generates convenience constructors:

```rust
pub fn new_login_request_email(email: String, password: String) -> LoginRequest {
    LoginRequest {
        auth_method: Some(LoginRequest::Email(EmailAuth {
            email,
            password,
        })),
        ..Default::default()
    }
}

pub fn new_login_request_phone(phone: String, code: String) -> LoginRequest {
    LoginRequest {
        auth_method: Some(LoginRequest::Phone(PhoneAuth {
            phone,
            code,
        })),
        ..Default::default()
    }
}
```

### HTTP Handlers

For a service definition:

```protobuf
service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
}
```

Generates:

```rust
#[async_trait::async_trait]
pub trait UserServiceServer: Send + Sync + 'static {
    async fn create_user(&self, request: CreateUserRequest) -> Result<User, StatusCode>;
    async fn get_user(&self, request: GetUserRequest) -> Result<User, StatusCode>;
}

pub fn register_user_service_server<S: UserServiceServer>(server: Arc<S>) -> Router {
    Router::new()
        .route("/api/v1/create_user", post(create_user_handler::<S>))
        .route("/api/v1/get_user", post(get_user_handler::<S>))
        .layer(ServiceBuilder::new().layer(CorsLayer::permissive()).into_inner())
        .with_state(server)
}

// Handler functions generated automatically
async fn create_user_handler<S: UserServiceServer>(
    State(server): State<Arc<S>>,
    Json(request): Json<CreateUserRequest>,
) -> impl IntoResponse {
    match server.create_user(request).await {
        Ok(response) => (StatusCode::OK, Json(response)).into_response(),
        Err(status) => (status, Json(serde_json::json!({
            "error": status.to_string()
        }))).into_response(),
    }
}
```

### OpenAPI Specifications

Generates complete OpenAPI 3.1 YAML files:

```yaml
openapi: 3.1.0
info:
  title: UserService API  
  version: 1.0.0
paths:
  /api/v1/create_user:
    post:
      summary: CreateUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
```

## Architecture

The plugins follow a clean architecture:

### sebuf-core

Provides shared functionality:

- `Plugin` trait for protobuf plugin implementation
- `ProtoParser` for protobuf reflection and type mapping
- `CodeGenerator` for Rust AST generation using `quote`
- `TypeMapper` for protobuf-to-Rust type conversion

### Plugin Structure

Each plugin follows the same pattern:

1. **Main binary** - Handles protoc protocol communication
2. **Generator** - Core generation logic using sebuf-core utilities  
3. **Integration tests** - End-to-end testing with real protoc execution
4. **Golden file tests** - Regression testing of generated output

## Dependencies

- **prost** - Protobuf implementation for Rust
- **quote** - Rust AST generation
- **syn** - Rust syntax parsing
- **axum** - HTTP framework for generated handlers
- **serde** - JSON serialization
- **heck** - String case conversion

## Development

### Running Tests

```bash
# Run specific plugin tests
cargo test -p protoc-gen-rust-oneof-helper
cargo test -p protoc-gen-rust-http  
cargo test -p protoc-gen-rust-openapiv3

# Run with output for debugging
cargo test -- --nocapture

# Update golden files after changes
UPDATE_GOLDEN=1 cargo test --test golden_test
```

### Code Quality

```bash
# Format code
cargo fmt --all

# Run clippy
cargo clippy --all -- -D warnings

# Check for security issues
cargo audit
```

### Adding New Plugins

To add a new plugin:

1. Create new directory under `rust/`
2. Add to workspace in root `Cargo.toml`
3. Implement `Plugin` trait from `sebuf-core`
4. Add integration tests following existing patterns
5. Update `Makefile` build targets

## Comparison with Go Plugins

| Feature | Go Plugins | Rust Plugins |
|---------|------------|--------------|
| **Performance** | Fast compilation | Faster runtime |
| **Type Safety** | Strong | Stronger with ownership |
| **HTTP Framework** | net/http | Axum |
| **JSON Handling** | encoding/json | serde |
| **Async Support** | Goroutines | async/await |
| **Memory Safety** | GC | Zero-cost ownership |
| **Ecosystem** | Mature | Growing rapidly |

## Contributing

1. Follow existing code patterns
2. Add comprehensive tests for new features
3. Update golden files when output changes
4. Ensure all lints pass
5. Document public APIs

The Rust plugins maintain feature parity with the Go versions while leveraging Rust's strengths in performance, safety, and modern async programming.