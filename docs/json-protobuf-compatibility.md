# JSON and Protobuf Compatibility

> Handling the semantic differences between protobuf and JSON serialization

Protobuf and JSON have different type systems and serialization behaviors. While sebuf handles most conversions automatically, some patterns require explicit annotations to produce the expected JSON output. This guide covers these edge cases and how to address them.

> **Note**: The `unwrap` annotation was added to address [issue #86](https://github.com/SebastienMelki/sebuf/issues/86) - supporting unwrapped repeated fields in map values for JSON marshaling.

## Table of Contents

- [Overview](#overview)
- [Map Value Unwrapping](#map-value-unwrapping)
- [When to Use Unwrap](#when-to-use-unwrap)
- [Limitations](#limitations)
- [Best Practices](#best-practices)

## Overview

JSON and protobuf differ in several key areas:

| Aspect | Protobuf | JSON |
|--------|----------|------|
| Map values | Must be scalar or message types | Can be any type including arrays |
| Empty collections | Omitted by default | Typically included as `[]` or `{}` |
| Field names | snake_case | camelCase (via protojson) |
| Numbers | Typed (int32, int64, float, double) | Single `number` type |

Most of these differences are handled automatically. However, **map values containing arrays** require special handling because protobuf doesn't allow `repeated` types directly as map values.

## Map Value Unwrapping

### The Problem

A common API pattern is returning a map where each value is an array:

```json
{
  "bars": {
    "AAPL": [{"price": 150.0}, {"price": 151.0}],
    "GOOG": [{"price": 2800.0}, {"price": 2810.0}]
  }
}
```

In protobuf, you cannot express this directly because map values cannot be `repeated`:

```protobuf
// INVALID: repeated types cannot be map values
message Response {
  map<string, repeated Bar> bars = 1;  // Compilation error!
}
```

The standard protobuf workaround is to wrap the array in a message:

```protobuf
message BarList {
  repeated Bar bars = 1;
}

message Response {
  map<string, BarList> bars = 1;
}
```

But this produces nested JSON that doesn't match the desired API format:

```json
{
  "bars": {
    "AAPL": {"bars": [{"price": 150.0}, {"price": 151.0}]},
    "GOOG": {"bars": [{"price": 2800.0}, {"price": 2810.0}]}
  }
}
```

### The Solution: `unwrap` Annotation

The `(sebuf.http.unwrap)` annotation tells sebuf to collapse the wrapper message during JSON serialization:

```protobuf
import "sebuf/http/annotations.proto";

message Bar {
  string symbol = 1;
  double price = 2;
  int64 volume = 3;
}

// Wrapper message with unwrap annotation
message BarList {
  repeated Bar bars = 1 [(sebuf.http.unwrap) = true];
}

message GetBarsResponse {
  map<string, BarList> bars = 1;
  string next_page_token = 2;
}
```

With `unwrap`, the JSON output matches the desired format:

```json
{
  "bars": {
    "AAPL": [{"symbol": "AAPL", "price": 150.0, "volume": 1000}],
    "GOOG": [{"symbol": "GOOG", "price": 2800.0, "volume": 500}]
  },
  "nextPageToken": "abc123"
}
```

### How It Works

When you use the `unwrap` annotation:

1. **HTTP Generation**: sebuf generates custom `MarshalJSON()` and `UnmarshalJSON()` methods for messages containing maps with unwrapped values
2. **Client Generation**: The generated client automatically uses the custom marshalers
3. **OpenAPI Generation**: The OpenAPI schema shows the unwrapped structure (array values, not wrapper objects)

### Complete Example

```protobuf
syntax = "proto3";
package marketdata.v1;

import "sebuf/http/annotations.proto";

option go_package = "github.com/yourorg/api/marketdata;marketdata";

// Single data point
message OptionBar {
  string symbol = 1;
  double open = 2;
  double high = 3;
  double low = 4;
  double close = 5;
  int64 volume = 6;
  string timestamp = 7;
}

// Wrapper with unwrap annotation
message OptionBarsList {
  repeated OptionBar bars = 1 [(sebuf.http.unwrap) = true];
}

// Request message
message GetOptionBarsRequest {
  repeated string symbols = 1;
  string start_date = 2;
  string end_date = 3;
}

// Response with map of symbol -> bars
message GetOptionBarsResponse {
  // Each symbol maps directly to an array of bars
  map<string, OptionBarsList> bars = 1;
  string next_page_token = 2;
}

service MarketDataService {
  option (sebuf.http.service_config) = {
    base_path: "/api/v1"
  };

  rpc GetOptionBars(GetOptionBarsRequest) returns (GetOptionBarsResponse) {
    option (sebuf.http.config) = {
      path: "/options/bars"
      method: HTTP_METHOD_POST
    };
  }
}
```

**JSON Request:**
```json
{
  "symbols": ["AAPL", "GOOG"],
  "startDate": "2024-01-01",
  "endDate": "2024-01-31"
}
```

**JSON Response:**
```json
{
  "bars": {
    "AAPL": [
      {"symbol": "AAPL", "open": 150.0, "high": 152.0, "low": 149.0, "close": 151.5, "volume": 10000, "timestamp": "2024-01-02T09:30:00Z"},
      {"symbol": "AAPL", "open": 151.5, "high": 153.0, "low": 150.0, "close": 152.0, "volume": 12000, "timestamp": "2024-01-03T09:30:00Z"}
    ],
    "GOOG": [
      {"symbol": "GOOG", "open": 2800.0, "high": 2850.0, "low": 2780.0, "close": 2820.0, "volume": 5000, "timestamp": "2024-01-02T09:30:00Z"}
    ]
  },
  "nextPageToken": "eyJwYWdlIjogMn0="
}
```

### Scalar Types

The `unwrap` annotation also works with scalar types:

```protobuf
message IntList {
  repeated int32 values = 1 [(sebuf.http.unwrap) = true];
}

message ScoresResponse {
  // Maps team name to array of scores
  map<string, IntList> scores = 1;
}
```

**JSON Output:**
```json
{
  "scores": {
    "TeamA": [95, 87, 92],
    "TeamB": [88, 91, 85]
  }
}
```

## When to Use Unwrap

Use the `unwrap` annotation when:

1. **You need map values to be arrays in JSON** - The most common use case
2. **You're matching an existing API contract** - When integrating with external systems that expect this format
3. **You want cleaner JSON output** - Removing unnecessary wrapper nesting

**Don't use unwrap when:**

1. The wrapper message has other fields besides the repeated field
2. You need the wrapper structure for other purposes (like additional metadata per array)
3. You're not using JSON serialization (binary protobuf doesn't need it)

## Limitations

### Constraints

1. **One unwrap field per message** - Only one field can have the `unwrap` annotation
2. **Must be a repeated field** - The annotation is only valid on `repeated` fields
3. **Wrapper message must be a map value** - The unwrap behavior only applies when the containing message is used as a map value

### Validation Errors

If constraints are violated, you'll get clear error messages:

```
unwrap annotation can only be used on repeated fields
```

```
only one field per message can have the unwrap annotation
```

## Best Practices

### 1. Name Wrapper Messages Clearly

Use descriptive names that indicate the wrapper pattern:

```protobuf
// Good: Clear that this is a list/wrapper
message OptionBarsList {
  repeated OptionBar bars = 1 [(sebuf.http.unwrap) = true];
}

// Avoid: Unclear purpose
message OptionBarsWrapper {
  repeated OptionBar items = 1 [(sebuf.http.unwrap) = true];
}
```

### 2. Keep Wrapper Messages Simple

The wrapper should only contain the unwrapped field:

```protobuf
// Good: Single purpose wrapper
message BarList {
  repeated Bar bars = 1 [(sebuf.http.unwrap) = true];
}

// Avoid: Extra fields defeat the purpose
message BarList {
  repeated Bar bars = 1 [(sebuf.http.unwrap) = true];
  string extra_info = 2;  // This field will be lost in JSON!
}
```

### 3. Document the JSON Structure

Add comments explaining the JSON behavior:

```protobuf
// GetBarsResponse contains market data bars by symbol.
// JSON serialization: {"bars": {"SYMBOL": [...bars...]}}
message GetBarsResponse {
  // Map from symbol to list of bars.
  // Note: BarsList is unwrapped, so each symbol maps directly to an array.
  map<string, BarsList> bars = 1;
}
```

### 4. Test Both Directions

Ensure your tests cover both marshaling and unmarshaling:

```go
func TestUnwrapRoundTrip(t *testing.T) {
    original := &GetBarsResponse{
        Bars: map[string]*BarsList{
            "AAPL": {Bars: []*Bar{{Symbol: "AAPL", Price: 150.0}}},
        },
    }

    // Marshal to JSON
    data, err := json.Marshal(original)
    require.NoError(t, err)

    // Verify JSON structure
    var raw map[string]interface{}
    json.Unmarshal(data, &raw)
    bars := raw["bars"].(map[string]interface{})
    assert.IsType(t, []interface{}{}, bars["AAPL"]) // Should be array, not object

    // Unmarshal back
    var restored GetBarsResponse
    err = json.Unmarshal(data, &restored)
    require.NoError(t, err)

    assert.Equal(t, original.Bars["AAPL"].Bars[0].Symbol, restored.Bars["AAPL"].Bars[0].Symbol)
}
```

---

**See also:**
- [HTTP Generation Guide](./http-generation.md) - Full HTTP handler documentation
- [OpenAPI Generation Guide](./openapi-generation.md) - API documentation generation
- [Client Generation Guide](./client-generation.md) - Type-safe client generation
