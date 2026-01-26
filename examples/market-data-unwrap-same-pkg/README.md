# Market Data Unwrap - Same Package Example

> **Demonstrates cross-file unwrap when wrapper and service are in separate proto files**

This example builds on `market-data-unwrap` to show that the `(sebuf.http.unwrap)` annotation works when the wrapper message is defined in a **different proto file** than the response message that uses it.

## When to Use This Pattern

Use separate proto files for wrapper messages when:

- **Shared wrappers** - Multiple services need the same wrapper type
- **Domain separation** - Keep data models separate from service definitions
- **Team organization** - Different teams own different parts of the API

## Project Structure

```
proto/
  stock_bar.proto       # Defines Bar and BarList (with unwrap)
  stock_service.proto   # Imports and uses BarList in GetBarsResponse
```

### stock_bar.proto

```protobuf
// Data model file - defines shared types
message Bar {
  string symbol = 1;
  double price = 2;
  int64 volume = 3;
}

// Wrapper with unwrap annotation
message BarList {
  repeated Bar bars = 1 [(sebuf.http.unwrap) = true];
}
```

### stock_service.proto

```protobuf
// Service file - imports wrapper from another file
import "stock_bar.proto";

message GetBarsResponse {
  // BarList is from a different file, unwrap still works
  map<string, BarList> bars = 1;
}
```

## Quick Start

```bash
# Generate and run
buf generate
go run main.go

# Test the API
curl -X POST 'http://localhost:8080/api/v1/bars' \
  -H 'Content-Type: application/json' \
  -d '{"symbols": ["AAPL", "GOOG"]}'
```

## Key Point

The unwrap feature works across proto file boundaries **within the same Go package**. This is handled automatically - no special configuration needed.

## Comparison with market-data-unwrap

| Example | Wrapper Location | Use Case |
|---------|-----------------|----------|
| `market-data-unwrap` | Same file as response | Simple, single-file APIs |
| `market-data-unwrap-same-pkg` | Separate file | Shared types, modular design |

Both produce identical JSON output:

```json
{
  "bars": {
    "AAPL": [{"symbol": "AAPL", "price": 150.0, "volume": 1000}],
    "GOOG": [{"symbol": "GOOG", "price": 2800.0, "volume": 500}]
  }
}
```

## See Also

- [market-data-unwrap](../market-data-unwrap/) - Single-file unwrap example
- [JSON/Protobuf Compatibility Guide](../../docs/json-protobuf-compatibility.md) - Full unwrap documentation
