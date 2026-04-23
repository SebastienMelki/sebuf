# Forward Compatibility Example

> **Runtime options for forward-compatible sebuf clients**

## The Problem

APIs evolve — providers add new response fields. By default, sebuf clients reject unknown fields:

```
failed to unmarshal response: unknown field "swap_rate"
```

## The Solution

sebuf generates `WithXxxDiscardUnknownFields` options at the client and per-call level:

```go
// Client-level: all RPCs discard unknown fields
client := services.NewQuoteServiceClient(baseURL,
    services.WithQuoteServiceDiscardUnknownFields(true),
)

// Per-call: override for a single RPC
quote, err := client.GetQuote(ctx, req,
    services.WithQuoteServiceCallDiscardUnknownFields(true),
)
```

## Run the Example

```bash
cd examples/forward-compatibility
go run .
```

Output:

```
=== 1. Default client (strict mode) ===
  Error: unknown field "region"
  Expected: strict mode rejects unknown fields like 'swap_rate'

=== 2. WithQuoteServiceDiscardUnknownFields(true) ===
  OK: symbol=AAPL price=185.50 currency=USD
  Unknown fields 'swap_rate' and 'region' silently discarded

=== 3. Per-call override: WithQuoteServiceCallDiscardUnknownFields(false) ===
  Error: unknown field "region"
  Expected: per-call override to strict rejects unknown fields
```

## Precedence

| Client-level | Per-call | Result |
|---|---|---|
| not set | not set | strict (default) |
| `true` | not set | discard unknown |
| not set | `true` | discard unknown |
| `true` | `false` | strict (per-call wins) |

## Regenerate

```bash
# From the example directory
buf dep update
buf generate
go mod tidy
```
