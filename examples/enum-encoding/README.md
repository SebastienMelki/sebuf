# enum-encoding

Demonstrates the `(sebuf.http.enum_value)` annotation end-to-end on the generated
Go HTTP server.

The `RiskLevel` enum maps its values to custom JSON strings:

```protobuf
enum RiskLevel {
  RISK_LEVEL_UNSPECIFIED = 0;                                  // no mapping -> proto name
  RISK_LEVEL_LOW    = 1 [(sebuf.http.enum_value) = "low"];
  RISK_LEVEL_MEDIUM = 2 [(sebuf.http.enum_value) = "medium"];
  RISK_LEVEL_HIGH   = 3 [(sebuf.http.enum_value) = "high"];
}
```

The generated server serializes responses and parses requests through `protojson`,
which by itself emits the raw proto names (`RISK_LEVEL_LOW`). sebuf generates a
message-level marshaler (`*_enum_field_encoding.pb.go`) that rewrites enum fields to
their custom strings on the way out and back on the way in, so the wire format matches
the OpenAPI docs and the TypeScript/Python clients.

## What it proves

`POST /api/v1/suggestion` with body `{"symbol":"AAPL","requestedRisk":"low"}` returns:

```json
{
  "symbol": "AAPL",
  "probabilityOfProfit": 0.62,
  "riskLevel": "low",
  "alternateRiskLevels": ["medium", "high"],
  "riskBySymbol": {"AAPL": "low", "TSLA": "high"}
}
```

- `"low"` in the request body is accepted (request parsing).
- `riskLevel`, the repeated `alternateRiskLevels`, and the `riskBySymbol` map values
  all serialize as custom strings — never `RISK_LEVEL_LOW`.

## Run

```bash
# from the repo root, build the plugins first:
make build

cd examples/enum-encoding
make generate      # buf generate + go mod tidy
make test          # end-to-end assertion (main_test.go)
make run           # start the server on :8080, then: make curl
```
