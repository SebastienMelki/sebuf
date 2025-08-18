# Validation

sebuf provides comprehensive automatic validation for both HTTP headers and request bodies, powered by [protovalidate](https://github.com/bufbuild/protovalidate) for body validation and custom middleware for header validation, giving you production-ready validation with zero configuration.

## Quick Start

### Request Body Validation

Add validation rules to your protobuf messages using `buf.validate` annotations:

```protobuf
syntax = "proto3";

import "buf/validate/validate.proto";

message CreateUserRequest {
  // Name must be between 2 and 100 characters
  string name = 1 [(buf.validate.field).string = {
    min_len: 2,
    max_len: 100
  }];
  
  // Email must be valid
  string email = 2 [(buf.validate.field).string.email = true];
  
  // Age must be between 18 and 120
  int32 age = 3 [(buf.validate.field).int32 = {
    gte: 18,
    lte: 120
  }];
}
```

### Header Validation

Add header validation to your services using sebuf annotations:

```protobuf
import "sebuf/http/headers.proto";

service UserService {
  option (sebuf.http.service_headers) = {
    required_headers: [
      {
        name: "X-API-Key"
        type: "string"
        format: "uuid"
        required: true
        description: "API authentication key"
      }
    ]
  };
  
  rpc CreateUser(CreateUserRequest) returns (User);
}
```

That's it! Both header and body validation happen automatically in your HTTP handlers.

## Features

- ✅ **Zero configuration** - Validation works automatically
- ✅ **Comprehensive coverage** - Both headers and request bodies validated
- ✅ **All protovalidate rules** - Full compatibility with buf.validate ecosystem for body validation
- ✅ **Header type validation** - Support for string, integer, number, boolean, array types
- ✅ **Header format validation** - Built-in validators for UUID, email, datetime formats
- ✅ **Performance optimized** - Cached validator instances
- ✅ **Clear error messages** - HTTP 400 with detailed validation errors
- ✅ **Fail-fast validation** - Headers validated before body for efficiency

## Request Body Validation Rules

### String Validation

```protobuf
message StringValidationExample {
  // Length constraints
  string name = 1 [(buf.validate.field).string = {
    min_len: 1,
    max_len: 50
  }];
  
  // Email validation
  string email = 2 [(buf.validate.field).string.email = true];
  
  // UUID validation
  string id = 3 [(buf.validate.field).string.uuid = true];
  
  // Pattern matching (regex)
  string phone = 4 [(buf.validate.field).string.pattern = "^\\+?[1-9]\\d{1,14}$"];
  
  // Enum-like validation (allowed values)
  string status = 5 [(buf.validate.field).string = {
    in: ["active", "inactive", "pending"]
  }];
  
  // URL validation
  string website = 6 [(buf.validate.field).string.uri = true];
}
```

### Numeric Validation

```protobuf
message NumericValidationExample {
  // Integer range
  int32 age = 1 [(buf.validate.field).int32 = {
    gte: 0,
    lte: 150
  }];
  
  // Exact value
  int32 version = 2 [(buf.validate.field).int32.const = 1];
  
  // List of allowed values
  int32 priority = 3 [(buf.validate.field).int32 = {
    in: [1, 2, 3, 4, 5]
  }];
  
  // Float validation
  float score = 4 [(buf.validate.field).float = {
    gte: 0.0,
    lte: 100.0
  }];
}
```

### Collection Validation

```protobuf
message CollectionValidationExample {
  // Repeated field size
  repeated string tags = 1 [(buf.validate.field).repeated = {
    min_items: 1,
    max_items: 10
  }];
  
  // Map validation
  map<string, string> metadata = 2 [(buf.validate.field).map = {
    min_pairs: 1,
    max_pairs: 20
  }];
  
  // Nested message validation
  repeated UserInfo users = 3 [(buf.validate.field).repeated.min_items = 1];
}
```

### Message Validation

```protobuf
message MessageValidationExample {
  // Required field (non-zero/non-empty)
  string required_field = 1 [(buf.validate.field).required = true];
  
  // Skip validation for this field
  string internal_field = 2 [(buf.validate.field).ignore = IGNORE_ALWAYS];
}
```

## Header Validation

### Service-Level Headers

Headers defined at the service level apply to all RPCs in that service:

```protobuf
service APIService {
  option (sebuf.http.service_headers) = {
    required_headers: [
      {
        name: "X-API-Key"
        description: "API authentication key"
        type: "string"
        format: "uuid"
        required: true
        example: "123e4567-e89b-12d3-a456-426614174000"
      },
      {
        name: "X-Tenant-ID"
        description: "Tenant identifier"
        type: "integer"
        required: true
        minimum: 1
        maximum: 999999
      },
      {
        name: "X-Debug-Mode"
        description: "Enable debug mode"
        type: "boolean"
        required: false
        default: "false"
      }
    ]
  };
}
```

### Method-Level Headers

Headers can be specified per RPC method, overriding service-level headers with the same name:

```protobuf
rpc CreateResource(CreateResourceRequest) returns (Resource) {
  option (sebuf.http.method_headers) = {
    required_headers: [
      {
        name: "X-Request-ID"
        type: "string"
        format: "uuid"
        required: true
      },
      {
        name: "X-Idempotency-Key"
        type: "string"
        required: true
        min_length: 16
        max_length: 64
      }
    ]
  };
}
```

### Supported Header Types and Formats

| Type | Formats | Description |
|------|---------|-------------|
| `string` | `uuid`, `email`, `date-time`, `date`, `time` | Text with optional format validation |
| `integer` | - | Whole numbers with optional min/max constraints |
| `number` | - | Decimal numbers with optional min/max constraints |
| `boolean` | - | `true` or `false` values |
| `array` | - | Comma-separated values |

### Header Validation Examples

```protobuf
// Comprehensive header validation example
service SecureAPI {
  option (sebuf.http.service_headers) = {
    required_headers: [
      // UUID validation
      {
        name: "X-Trace-ID"
        type: "string"
        format: "uuid"
        required: true
      },
      // Email validation
      {
        name: "X-Admin-Email"
        type: "string"
        format: "email"
        required: false
      },
      // Date-time validation
      {
        name: "X-Request-Time"
        type: "string"
        format: "date-time"
        required: true
      },
      // Integer with constraints
      {
        name: "X-Rate-Limit"
        type: "integer"
        required: false
        minimum: 1
        maximum: 1000
        default: "100"
      },
      // Enum-like validation
      {
        name: "X-API-Version"
        type: "string"
        required: false
        enum: ["v1", "v2", "v3"]
        default: "v2"
      },
      // Array type
      {
        name: "X-Features"
        type: "array"
        required: false
        description: "Comma-separated feature flags"
      }
    ]
  };
}
```

## Error Handling

When validation fails, sebuf returns an HTTP 400 Bad Request with the validation error message. Headers are validated before the request body.

### Header Validation Errors

```bash
# Missing required header
curl -X POST /api/users -d '{"name": "John"}'
# Returns: 400 Bad Request
# Body: "Missing required header: X-API-Key"

# Invalid header format (UUID)
curl -X POST /api/users \
  -H "X-API-Key: not-a-uuid" \
  -d '{"name": "John"}'
# Returns: 400 Bad Request
# Body: "Invalid header X-API-Key: invalid UUID format"

# Invalid header type (expecting integer)
curl -X POST /api/users \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -H "X-Tenant-ID: abc" \
  -d '{"name": "John"}'
# Returns: 400 Bad Request
# Body: "Invalid header X-Tenant-ID: must be an integer"
```

### Body Validation Errors

```bash
# Invalid email
curl -X POST /api/users \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{"email": "invalid"}'
# Returns: 400 Bad Request
# Body: "validation error: field 'email' with value 'invalid' failed rule 'string.email'"

# Name too short  
curl -X POST /api/users \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{"name": "J"}'
# Returns: 400 Bad Request
# Body: "validation error: field 'name' with value 'J' failed rule 'string.min_len'"
```

## Advanced Usage

### Custom Error Messages

Use CEL expressions for custom validation logic:

```protobuf
message AdvancedValidation {
  string password = 1 [(buf.validate.field).string = {
    min_len: 8,
    pattern: "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).*$"
  }];
  
  // Custom CEL validation
  string username = 2 [(buf.validate.field).cel = {
    id: "username.unique",
    message: "Username must be unique and start with letter",
    expression: "this.matches('^[a-zA-Z][a-zA-Z0-9_]*$')"
  }];
}
```

### Conditional Validation

```protobuf
message ConditionalValidation {
  string type = 1;
  
  // Only validate email if type is "email"
  string contact = 2 [(buf.validate.field).cel = {
    id: "contact.conditional",
    expression: "this.type != 'email' || this.contact.isEmail()"
  }];
}
```

## Performance

Validation is highly optimized:

- **Cached validators**: Validator instances for body validation are created once and reused
- **Efficient header checking**: Headers validated in a single pass before body processing
- **No reflection overhead**: Validation rules are pre-compiled
- **Minimal allocations**: Only allocates on validation errors
- **Sub-microsecond latency**: After initial warm-up
- **Fail-fast**: Headers validated first to avoid unnecessary body parsing

## Compatibility

sebuf validation is fully compatible with the protovalidate ecosystem:

- **buf CLI**: Use buf validate commands for body validation rules
- **IDE support**: Validation rules show in proto IDE plugins  
- **Other languages**: Same body validation rules work with protovalidate for Python, Java, etc.
- **OpenAPI**: Header validations automatically appear in generated OpenAPI specs
- **Migration**: Uses standard buf.validate annotations for body validation

## Best Practices

1. **Validate at the boundary**: Add validation to both headers and request messages
2. **Be specific**: Use the most specific validation rule (email vs pattern, UUID format vs string)
3. **Layer validation**: Use headers for auth/metadata, body for business data
4. **Consider UX**: Validation errors are shown to users - make them helpful
5. **Test edge cases**: Test validation with boundary values and missing headers
6. **Document constraints**: Include validation info in API documentation
7. **Use service-level headers**: Define common headers once at service level
8. **Override when needed**: Use method-level headers for specific requirements

## Troubleshooting

**Body validation not working?**
- Ensure you're importing `"buf/validate/validate.proto"`
- Check that your message fields have validation annotations
- Regenerate your code after adding validation rules

**Header validation not working?**
- Ensure you're importing `"sebuf/http/headers.proto"`
- Check that headers are defined in service or method options
- Verify header names match exactly (case-sensitive)
- Regenerate your code after adding header annotations

**Performance concerns?**
- Validation overhead is minimal (<1μs per request after warm-up)
- Header validation is done in a single pass
- Body validators are cached automatically
- No code generation required for validation logic

**Debugging validation issues?**
- Headers are validated first - check header errors before body errors
- Use curl with -v flag to see all headers being sent
- Check generated OpenAPI spec to verify header requirements

**Need help?**
- Check the [protovalidate documentation](https://github.com/bufbuild/protovalidate) for body validation
- See the [examples/simple-api](../examples/simple-api) for working examples
- Review [http-generation.md](./http-generation.md#header-validation) for header details
- Open an issue on GitHub