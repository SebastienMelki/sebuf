# Validation Showcase Example

This example demonstrates comprehensive request validation using buf.validate (protovalidate), showcasing all validation patterns available in sebuf.

## Features Demonstrated

### String Validation

| Constraint | Description | Example |
|------------|-------------|---------|
| `min_len/max_len` | Length constraints | `string name = 1 [(buf.validate.field).string = { min_len: 2, max_len: 100 }]` |
| `email` | Email format | `string email = 1 [(buf.validate.field).string.email = true]` |
| `uuid` | UUID format | `string id = 1 [(buf.validate.field).string.uuid = true]` |
| `pattern` | Regex pattern | `string phone = 1 [(buf.validate.field).string.pattern = "^\\+[1-9]\\d{6,14}$"]` |
| `in` | Enum values | `string country = 1 [(buf.validate.field).string = { in: ["US", "CA", "MX"] }]` |

### Numeric Validation

| Constraint | Description | Example |
|------------|-------------|---------|
| `gte/lte` | Inclusive range | `int32 quantity = 1 [(buf.validate.field).int32 = { gte: 1, lte: 100 }]` |
| `gt/lt` | Exclusive range | `double price = 1 [(buf.validate.field).double = { gt: 0, lte: 10000 }]` |

### Array/Repeated Validation

| Constraint | Description | Example |
|------------|-------------|---------|
| `min_items/max_items` | Size constraints | `repeated Item items = 1 [(buf.validate.field).repeated = { min_items: 1, max_items: 50 }]` |
| `unique` | No duplicates | `repeated string codes = 1 [(buf.validate.field).repeated = { unique: true }]` |
| `items` | Element validation | `repeated string emails = 1 [(buf.validate.field).repeated = { items: { string: { email: true } } }]` |

### Map Validation

| Constraint | Description | Example |
|------------|-------------|---------|
| `max_pairs` | Size limit | `map<string, string> meta = 1 [(buf.validate.field).map = { max_pairs: 20 }]` |
| `keys` | Key validation | `map<string, string> opts = 1 [(buf.validate.field).map = { keys: { string: { pattern: "^[a-z_]+$" } } }]` |
| `values` | Value validation | `map<string, string> opts = 1 [(buf.validate.field).map = { values: { string: { max_len: 100 } } }]` |

### Nested Message Validation

| Constraint | Description | Example |
|------------|-------------|---------|
| `required` | Required nested message | `Address shipping = 1 [(buf.validate.field).required = true]` |

## Quick Start

```bash
# Generate code and run the server
make demo

# Test valid requests
make test

# Test validation errors (expect HTTP 400)
make test-validation
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/orders` | Create order with full validation |
| GET | `/api/v1/orders/{order_id}` | Get order by UUID |
| POST | `/api/v1/validate/address` | Validate shipping address |
| POST | `/api/v1/validate/coupon` | Validate coupon code |

## Example: Create Order

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "contact": {
      "email": "john.doe@example.com",
      "phone": "+14155551234"
    },
    "shipping_address": {
      "street": "123 Main Street, Apt 4B",
      "city": "San Francisco",
      "state": "CA",
      "postal_code": "94102",
      "country": "US"
    },
    "items": [{
      "product_id": "123e4567-e89b-12d3-a456-426614174000",
      "product_name": "Wireless Headphones",
      "sku": "SKU-ABC123DEF",
      "quantity": 2,
      "unit_price": 99.99,
      "discount_percent": 10.0,
      "options": {"color": "blue", "size": "large"}
    }],
    "coupon_codes": ["SAVE20"],
    "payment_method": 1
  }'
```

## Validation Error Examples

### Invalid Email Format

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{"contact": {"email": "not-an-email"}, ...}'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "contact.email",
    "description": "value must be a valid email address"
  }]
}
```

### Invalid State Code (Pattern)

```bash
curl -X POST http://localhost:8080/api/v1/validate/address \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{"address": {"state": "California", ...}}'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "address.state",
    "description": "value does not match pattern: ^[A-Z]{2}$"
  }]
}
```

### Quantity Out of Range

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{"items": [{"quantity": 500, ...}], ...}'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "items[0].quantity",
    "description": "value must be less than or equal to 100"
  }]
}
```

### Empty Items Array

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{"items": [], ...}'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "items",
    "description": "value must contain at least 1 item(s)"
  }]
}
```

## Generated Files

After running `make generate`:

```
api/
  proto/
    models/
      order.pb.go              # Order, Address, OrderItem messages
    services/
      order_service.pb.go            # Service interface
      order_service_http.pb.go       # HTTP handler registration
      order_service_http_binding.pb.go  # Request binding + validation
      order_service_http_config.pb.go   # Server options
      order_service_http_mock.pb.go     # Mock implementation
docs/
  OrderService.openapi.yaml    # OpenAPI 3.1 with validation constraints
  OrderService.openapi.json    # OpenAPI 3.1 (JSON format)
```

## Key Concepts

### Automatic Validation

All validation happens automatically in the generated `BindingMiddleware`. You don't need to write any validation code - just define the rules in your proto files.

### Validation Error Format

Validation errors return HTTP 400 with a structured response containing:
- `violations[]`: Array of field violations
  - `field`: Path to the invalid field (e.g., `items[0].quantity`)
  - `description`: Human-readable error message

### OpenAPI Integration

Validation constraints are automatically reflected in the generated OpenAPI specs:
- `minLength`/`maxLength` for string constraints
- `minimum`/`maximum` for numeric constraints
- `minItems`/`maxItems` for array constraints
- `pattern` for regex constraints
- `enum` for `in` constraints
