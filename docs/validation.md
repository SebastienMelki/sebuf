# Request Validation

sebuf provides automatic request validation powered by [protovalidate](https://github.com/bufbuild/protovalidate), giving you production-ready validation with zero configuration.

## Quick Start

Add validation rules to your protobuf messages using `sebuf.validate` annotations:

```protobuf
syntax = "proto3";

import "sebuf/validate/validate.proto";

message CreateUserRequest {
  // Name must be between 2 and 100 characters
  string name = 1 [(sebuf.validate.field).string = {
    min_len: 2,
    max_len: 100
  }];
  
  // Email must be valid
  string email = 2 [(sebuf.validate.field).string.email = true];
  
  // Age must be between 18 and 120
  int32 age = 3 [(sebuf.validate.field).int32 = {
    gte: 18,
    lte: 120
  }];
}
```

That's it! Validation happens automatically in your HTTP handlers.

## Features

- ✅ **Zero configuration** - Validation works automatically
- ✅ **All protovalidate rules** - Full compatibility with buf.validate ecosystem
- ✅ **sebuf.validate namespace** - Use `(sebuf.validate.field)` consistently
- ✅ **Performance optimized** - Cached validator instances
- ✅ **Clear error messages** - HTTP 400 with detailed validation errors
- ✅ **No code generation** - Pure runtime validation

## Validation Rules

### String Validation

```protobuf
message StringValidationExample {
  // Length constraints
  string name = 1 [(sebuf.validate.field).string = {
    min_len: 1,
    max_len: 50
  }];
  
  // Email validation
  string email = 2 [(sebuf.validate.field).string.email = true];
  
  // UUID validation
  string id = 3 [(sebuf.validate.field).string.uuid = true];
  
  // Pattern matching (regex)
  string phone = 4 [(sebuf.validate.field).string.pattern = "^\\+?[1-9]\\d{1,14}$"];
  
  // Enum-like validation (allowed values)
  string status = 5 [(sebuf.validate.field).string = {
    in: ["active", "inactive", "pending"]
  }];
  
  // URL validation
  string website = 6 [(sebuf.validate.field).string.uri = true];
}
```

### Numeric Validation

```protobuf
message NumericValidationExample {
  // Integer range
  int32 age = 1 [(sebuf.validate.field).int32 = {
    gte: 0,
    lte: 150
  }];
  
  // Exact value
  int32 version = 2 [(sebuf.validate.field).int32.const = 1];
  
  // List of allowed values
  int32 priority = 3 [(sebuf.validate.field).int32 = {
    in: [1, 2, 3, 4, 5]
  }];
  
  // Float validation
  float score = 4 [(sebuf.validate.field).float = {
    gte: 0.0,
    lte: 100.0
  }];
}
```

### Collection Validation

```protobuf
message CollectionValidationExample {
  // Repeated field size
  repeated string tags = 1 [(sebuf.validate.field).repeated = {
    min_items: 1,
    max_items: 10
  }];
  
  // Map validation
  map<string, string> metadata = 2 [(sebuf.validate.field).map = {
    min_pairs: 1,
    max_pairs: 20
  }];
  
  // Nested message validation
  repeated UserInfo users = 3 [(sebuf.validate.field).repeated.min_items = 1];
}
```

### Message Validation

```protobuf
message MessageValidationExample {
  // Required field (non-zero/non-empty)
  string required_field = 1 [(sebuf.validate.field).required = true];
  
  // Skip validation for this field
  string internal_field = 2 [(sebuf.validate.field).ignore = IGNORE_ALWAYS];
}
```

## Error Handling

When validation fails, sebuf returns an HTTP 400 Bad Request with the validation error message:

```bash
# Invalid email
curl -X POST /api/users -d '{"email": "invalid"}'
# Returns: 400 Bad Request
# Body: "validation error: field 'email' with value 'invalid' failed rule 'string.email'"

# Name too short  
curl -X POST /api/users -d '{"name": "J"}'
# Returns: 400 Bad Request
# Body: "validation error: field 'name' with value 'J' failed rule 'string.min_len'"
```

## Advanced Usage

### Custom Error Messages

Use CEL expressions for custom validation logic:

```protobuf
message AdvancedValidation {
  string password = 1 [(sebuf.validate.field).string = {
    min_len: 8,
    pattern: "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).*$"
  }];
  
  // Custom CEL validation
  string username = 2 [(sebuf.validate.field).cel = {
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
  string contact = 2 [(sebuf.validate.field).cel = {
    id: "contact.conditional",
    expression: "this.type != 'email' || this.contact.isEmail()"
  }];
}
```

## Performance

Validation is highly optimized:

- **Cached validators**: Validator instances are created once and reused
- **No reflection overhead**: Validation rules are pre-compiled
- **Minimal allocations**: Only allocates on validation errors
- **Sub-microsecond latency**: After initial warm-up

## Compatibility

sebuf validation is fully compatible with the protovalidate ecosystem:

- **buf CLI**: Use buf validate commands
- **IDE support**: Validation rules show in proto IDE plugins  
- **Other languages**: Same rules work with protovalidate for Python, Java, etc.
- **Migration**: Easy to migrate from buf.validate to sebuf.validate

## Best Practices

1. **Validate at the boundary**: Add validation to your API request messages
2. **Be specific**: Use the most specific validation rule (email vs pattern)
3. **Consider UX**: Validation errors are shown to users - make them helpful
4. **Test edge cases**: Test validation with boundary values
5. **Document constraints**: Include validation info in API documentation

## Troubleshooting

**Validation not working?**
- Ensure you're importing `"sebuf/validate/validate.proto"`
- Check that your message fields have validation annotations
- Regenerate your code after adding validation rules

**Performance concerns?**
- Validation overhead is minimal (<1μs per request after warm-up)
- Validators are cached automatically
- No code generation required

**Need help?**
- Check the [protovalidate documentation](https://github.com/bufbuild/protovalidate)
- See the [examples/simple-api](../examples/simple-api) for working examples
- Open an issue on GitHub