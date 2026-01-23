# Market Data Unwrap Example

> **Demonstrates the `unwrap` annotation for JSON/protobuf compatibility**

This example shows how to use the `(sebuf.http.unwrap)` annotation to handle a common API pattern where map values should serialize as arrays in JSON.

## The Problem

Many APIs (like financial market data APIs) return data in this format:

```json
{
  "bars": {
    "TSLA260123C00335000": [
      {"c": 143.08, "h": 143.08, "l": 143.08, "n": 1, "o": 143.08, "t": "2025-12-15T15:05:00Z", "v": 1, "vw": 143.08},
      {"c": 145.34, "h": 145.34, "l": 145.34, "n": 1, "o": 145.34, "t": "2025-12-15T18:05:00Z", "v": 20, "vw": 145.34}
    ]
  },
  "next_page_token": null
}
```

In protobuf, you **cannot** directly express `map<string, repeated Message>`. The standard workaround creates nested JSON:

```json
{
  "bars": {
    "TSLA260123C00335000": {
      "bars": [...]  // Extra nesting!
    }
  }
}
```

## The Solution: `unwrap` Annotation

The `(sebuf.http.unwrap) = true` annotation tells sebuf to collapse the wrapper during JSON serialization:

```protobuf
// Wrapper message with unwrap annotation
message OptionBarsList {
  repeated OptionBar bars = 1 [(sebuf.http.unwrap) = true];
}

// Response using the wrapper as a map value
message GetOptionBarsResponse {
  map<string, OptionBarsList> bars = 1;  // Serializes as {"symbol": [...]}
}
```

## Quick Start

```bash
# Generate code and run the server
make demo

# In another terminal, test the API
make test
```

## What Gets Generated

After running `make generate`, you'll have:

```
api/
  models/
    option_bar.pb.go              # OptionBar and OptionBarsList messages
    option_bar_unwrap.pb.go       # Custom MarshalJSON/UnmarshalJSON for unwrap
  services/
    market_data_service.pb.go           # Service interface
    market_data_service_http.pb.go      # HTTP handler registration
    market_data_service_http_binding.pb.go  # Request binding + validation
    market_data_service_http_mock.pb.go # Mock server implementation
    market_data_service_client.pb.go    # Type-safe HTTP client
docs/
  MarketDataService.openapi.yaml  # OpenAPI 3.1 spec (array schema for unwrap)
  MarketDataService.openapi.json  # OpenAPI 3.1 (JSON format)
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v2/options/bars` | Get historical option bars with pagination |
| GET | `/v2/options/bars/latest` | Get latest bar for each symbol |

## Required Headers

All endpoints require authentication headers:

| Header | Description |
|--------|-------------|
| `APCA-API-KEY-ID` | API key for authentication |
| `APCA-API-SECRET-KEY` | API secret for authentication |

## Example Requests

### Get Historical Bars

```bash
curl -X GET 'http://localhost:8080/v2/options/bars?symbols=TSLA260123C00335000&timeframe=1Day' \
  -H 'APCA-API-KEY-ID: test-key' \
  -H 'APCA-API-SECRET-KEY: test-secret'
```

**Response (with unwrap - clean arrays):**
```json
{
  "bars": {
    "TSLA260123C00335000": [
      {"c": 143.08, "h": 143.08, "l": 143.08, "n": 1, "o": 143.08, "t": "2025-12-15T15:05:00Z", "v": 1, "vw": 143.08}
    ]
  },
  "nextPageToken": "eyJwYWdlIjogMn0="
}
```

### Get Latest Bars

```bash
curl -X GET 'http://localhost:8080/v2/options/bars/latest?symbols=TSLA260123C00335000&feed=opra' \
  -H 'APCA-API-KEY-ID: test-key' \
  -H 'APCA-API-SECRET-KEY: test-secret'
```

## Validation Examples

### Missing Required Parameter

```bash
curl -X GET 'http://localhost:8080/v2/options/bars?timeframe=1Day' \
  -H 'APCA-API-KEY-ID: test-key' \
  -H 'APCA-API-SECRET-KEY: test-secret'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "symbols",
    "description": "value is required"
  }]
}
```

### Invalid Sort Value

```bash
curl -X GET 'http://localhost:8080/v2/options/bars?symbols=TSLA&timeframe=1Day&sort=invalid' \
  -H 'APCA-API-KEY-ID: test-key' \
  -H 'APCA-API-SECRET-KEY: test-secret'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "sort",
    "description": "value must be in list [asc, desc]"
  }]
}
```

### Missing Required Header

```bash
curl -X GET 'http://localhost:8080/v2/options/bars?symbols=TSLA&timeframe=1Day'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "APCA-API-KEY-ID",
    "description": "required header 'APCA-API-KEY-ID' is missing"
  }]
}
```

### Limit Out of Range

```bash
curl -X GET 'http://localhost:8080/v2/options/bars?symbols=TSLA&timeframe=1Day&limit=50000' \
  -H 'APCA-API-KEY-ID: test-key' \
  -H 'APCA-API-SECRET-KEY: test-secret'
```

Response (HTTP 400):
```json
{
  "violations": [{
    "field": "limit",
    "description": "value must be less than or equal to 10000"
  }]
}
```

## Using the Generated Client

Run the full client example:

```bash
# Start the server first
go run main.go

# In another terminal, run the client example
go run client_example.go
```

The generated client handles all the unwrap serialization automatically:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/SebastienMelki/sebuf/examples/market-data-unwrap/api/services"
)

func main() {
    // Create client with authentication headers
    client := services.NewMarketDataServiceClient(
        "http://localhost:8080",
        services.WithMarketDataServiceAPCAAPIKEYID("your-api-key"),
        services.WithMarketDataServiceAPCAAPISECRETKEY("your-secret"),
    )

    // Get option bars
    resp, err := client.GetOptionBars(context.Background(), &services.GetOptionBarsRequest{
        Symbols:   "TSLA260123C00335000,AAPL240119C00150000",
        Timeframe: "1Day",
        Start:     "2025-12-01",
        End:       "2025-12-31",
        Limit:     100,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Access the data - unwrap is handled transparently
    for symbol, barsList := range resp.Bars {
        fmt.Printf("Symbol: %s, Bars: %d\n", symbol, len(barsList.Bars))
        for _, bar := range barsList.Bars {
            fmt.Printf("  Time: %s, Close: %.2f, Volume: %d\n", bar.T, bar.C, bar.V)
        }
    }
}
```

## OpenAPI Documentation

The generated OpenAPI spec correctly shows array schemas for unwrapped map values:

```yaml
# docs/MarketDataService.openapi.yaml
components:
  schemas:
    GetOptionBarsResponse:
      type: object
      properties:
        bars:
          type: object
          additionalProperties:
            type: array  # Array, not object wrapper!
            items:
              $ref: '#/components/schemas/OptionBar'
```

View the docs:
```bash
# With Swagger UI
docker run -p 8081:8080 -v $(pwd)/docs:/app swaggerapi/swagger-ui
# Then visit http://localhost:8081/?url=/app/MarketDataService.openapi.yaml
```

## Key Concepts

### How Unwrap Works

1. **Proto definition**: Mark one repeated field in a message with `[(sebuf.http.unwrap) = true]`
2. **Code generation**: sebuf generates custom `MarshalJSON()` and `UnmarshalJSON()` methods
3. **Runtime**: When the message is a map value, JSON serialization collapses the wrapper

### Constraints

- Only **one field per message** can have the unwrap annotation
- The field **must be a repeated type**
- Unwrap only applies when the message is used as a **map value**

### Files Generated

| File | Description |
|------|-------------|
| `*_unwrap.pb.go` | Custom JSON marshaling for messages with unwrap fields |
| `*_client.pb.go` | HTTP client that uses the custom marshalers |
| `*.openapi.yaml` | OpenAPI spec with correct array schemas |

## Troubleshooting

**API returns nested objects instead of arrays?**
- Ensure `[(sebuf.http.unwrap) = true]` is on the repeated field
- Run `make clean && make generate` to regenerate code

**Client not handling unwrap correctly?**
- The client uses custom marshalers automatically
- Check that you're using the generated client, not manual HTTP calls

**OpenAPI shows object instead of array for map values?**
- Regenerate the OpenAPI spec with `make generate`
- Check that the proto files have the unwrap annotation

## See Also

- [JSON/Protobuf Compatibility Guide](../../docs/json-protobuf-compatibility.md) - Full documentation
- [HTTP Generation Guide](../../docs/http-generation.md) - HTTP handler features
- [Client Generation Guide](../../docs/client-generation.md) - Client features
