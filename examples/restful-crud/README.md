# RESTful CRUD Example

This example demonstrates a complete RESTful CRUD API using sebuf, showcasing:

- **All HTTP verbs**: GET, POST, PUT, PATCH, DELETE
- **Path parameters**: `/products/{product_id}`
- **Query parameters**: Pagination, filtering, sorting, and search
- **PUT vs PATCH semantics**: Full replacement vs partial update

## Features Demonstrated

| Feature | Description |
|---------|-------------|
| `HTTP_METHOD_GET` | List and retrieve operations |
| `HTTP_METHOD_POST` | Create operations with request body |
| `HTTP_METHOD_PUT` | Full resource replacement |
| `HTTP_METHOD_PATCH` | Partial resource update with optional fields |
| `HTTP_METHOD_DELETE` | Delete operations with confirmation header |
| Path parameters | `{product_id}` bound from URL |
| Query parameters | `page`, `limit`, `category`, `min_price`, `max_price`, `sort`, `desc`, `q` |
| Request validation | buf.validate rules for all fields |
| Header validation | X-API-Key required, X-Confirm-Delete for deletes |

## Quick Start

```bash
# Generate code and run the server
make demo

# Or step by step:
make generate
make run
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/products` | List products with pagination/filtering |
| GET | `/api/v1/products/{product_id}` | Get a single product |
| POST | `/api/v1/products` | Create a new product |
| PUT | `/api/v1/products/{product_id}` | Full update (replace all fields) |
| PATCH | `/api/v1/products/{product_id}` | Partial update (only provided fields) |
| DELETE | `/api/v1/products/{product_id}` | Delete a product |

## Query Parameters

The `GET /api/v1/products` endpoint supports:

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page number (default: 1) |
| `limit` | int | Items per page (default: 20, max: 100) |
| `category` | string | Filter by category ID |
| `min_price` | double | Minimum price filter |
| `max_price` | double | Maximum price filter |
| `sort` | string | Sort field (name, price, created_at) |
| `desc` | bool | Sort descending (default: false) |
| `q` | string | Search in name/description |

## Example Requests

### List Products with Filtering

```bash
curl -X GET "http://localhost:8080/api/v1/products?page=1&limit=10&category=cat-electronics&min_price=50&sort=price" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000"
```

### Create a Product

```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "name": "Wireless Headphones",
    "description": "Noise-canceling Bluetooth headphones",
    "price": 99.99,
    "stock_quantity": 100,
    "category_id": "cat-electronics",
    "tags": ["audio", "wireless"]
  }'
```

### Full Update (PUT)

```bash
curl -X PUT http://localhost:8080/api/v1/products/123e4567-e89b-12d3-a456-426614174001 \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{
    "name": "Premium Wireless Headphones",
    "description": "Updated premium headphones",
    "price": 149.99,
    "stock_quantity": 50,
    "category_id": "cat-electronics",
    "tags": ["audio", "wireless", "premium"]
  }'
```

### Partial Update (PATCH)

```bash
curl -X PATCH http://localhost:8080/api/v1/products/123e4567-e89b-12d3-a456-426614174001 \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -d '{"price": 129.99}'
```

### Delete a Product

```bash
curl -X DELETE http://localhost:8080/api/v1/products/123e4567-e89b-12d3-a456-426614174001 \
  -H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
  -H "X-Confirm-Delete: true"
```

## Generated Files

After running `make generate`:

```
api/
  proto/
    models/
      product.pb.go           # Product message types
    services/
      product_service.pb.go         # Service interface
      product_service_http.pb.go    # HTTP handler registration
      product_service_http_binding.pb.go  # Request/response binding
      product_service_http_config.pb.go   # Server options
      product_service_http_mock.pb.go     # Mock implementation
docs/
  ProductService.openapi.yaml   # OpenAPI 3.1 spec (YAML)
  ProductService.openapi.json   # OpenAPI 3.1 spec (JSON)
```

## Key Concepts

### PUT vs PATCH

- **PUT**: Full replacement. All fields must be provided. Missing fields are set to defaults.
- **PATCH**: Partial update. Only provided fields are updated. Uses `optional` proto3 fields.

### Path Parameters

Path parameters like `{product_id}` are automatically bound from the URL path to the corresponding request field with the same name.

### Query Parameters

Fields annotated with `(sebuf.http.query)` are parsed from URL query string instead of request body. This is useful for GET requests that shouldn't have a body.
