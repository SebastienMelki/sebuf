# Forward Compatibility Example

> **Runtime options for forward-compatible sebuf clients**

## The Problem

By default, `protojson.Unmarshal` rejects unknown fields. When an API provider adds a new response field, the client fails:

```
failed to unmarshal response: proto: (line 1:45): unknown field "swap_rate"
```

## The Solution: `WithXxxDiscardUnknownFields`

sebuf generates client and call options to opt into forward compatibility at runtime:

```go
// Service-level: all RPCs discard unknown fields
client := services.NewMarketDataServiceClient(baseURL,
    services.WithMarketDataServiceDiscardUnknownFields(true),
)

// Per-call: override for a single RPC
resp, err := client.GetQuote(ctx, req,
    services.WithMarketDataServiceCallDiscardUnknownFields(true),
)
```

### Precedence

| Client-level | Per-call | Result |
|---|---|---|
| not set | not set | strict (default) |
| `true` | not set | discard unknown |
| not set | `true` | discard unknown |
| `true` | `false` | strict (per-call wins) |
| `false` | `true` | discard unknown (per-call wins) |

## Run the Example

```bash
go run ./examples/forward-compatibility/
```

## When to Use

Use `WithXxxDiscardUnknownFields(true)` when:
- You're consuming a third-party API that may add new fields
- You want forward compatibility without regenerating your SDK

Keep the default (strict mode) when:
- You control both client and server
- You want to catch schema drift early
