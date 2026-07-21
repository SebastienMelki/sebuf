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

## What it proves — the full matrix

The example covers **annotated vs unannotated** enums, each **directly on the marshaled
message and nested below it**:

| | direct (on the response) | nested (below the response) |
|---|---|---|
| **annotated** (`RiskLevel`, `OptionType`) | `overall_risk` → `"low"` | `option_suggestions[].risk_level` (1 level), `...options_contract.type` (2 levels) → `"low"`, `"call"` |
| **unannotated** (`Sentiment`) | `market_sentiment` → `"SENTIMENT_BULLISH"` | `...options_contract.sentiment` → `"SENTIMENT_BULLISH"` |

The nested cases are what plain `protojson` gets wrong — it never invokes a nested Go
message's custom marshaler. Unannotated enums correctly keep their proto names at every depth.

`POST /api/v1/suggestions` with body `{"underlyingSymbol":"AAPL","requestedRisk":"low"}` returns:

```json
{
  "optionSuggestions": [
    {"optionsContract": {"symbol": "AAPL", "type": "call", "sentiment": "SENTIMENT_BULLISH", "strikePrice": 330},
     "probabilityOfProfit": 0.62, "riskLevel": "low"},
    {"optionsContract": {"symbol": "AAPL", "type": "put", "sentiment": "SENTIMENT_BEARISH", "strikePrice": 300},
     "probabilityOfProfit": 0.41, "riskLevel": "high"}
  ],
  "overallRisk": "low",
  "marketSentiment": "SENTIMENT_BULLISH"
}
```

Under `UseProtoNames: true` the same values appear under snake_case keys
(`option_suggestions`, `options_contract`, `risk_level`, `type`) — the marshaler patches
whichever key protojson emits.

- `"low"` in the request body is accepted (request parsing).
- Annotated enums serialize as custom strings at every depth — never `RISK_LEVEL_LOW` / `OPTION_TYPE_CALL`.
- Unannotated `Sentiment` keeps its proto name, direct and nested.
- Works the same under `WithMarshalOptions(protojson.MarshalOptions{UseProtoNames: true})`
  (snake_case keys) at every depth.

## Run

```bash
# from the repo root, build the plugins first:
make build

cd examples/enum-encoding
make generate      # buf generate + go mod tidy
make test          # end-to-end assertion (main_test.go)
make run           # start the server on :8080, then: make curl
```
