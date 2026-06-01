# Marshal Options Example

This example demonstrates the `WithMarshalOptions` ServerOption, which threads `protojson.MarshalOptions` through every JSON response a generated sebuf HTTP server produces.

## The problem

Proto3 silently drops zero values from JSON. A `bool accepting_orders = false` is indistinguishable from "field never set" — both produce the same `{}` on the wire. For fields where `false` (or `0`, or `""`) carries semantic meaning, that's lossy.

```protobuf
message Status {
  bool accepting_orders = 1;   // false ≠ "unset" in app logic
  int32 subscriber_count = 2;  // 0 subscribers is a real, visible state
  string note = 3;             // empty note is a deliberate choice
}
```

## The fix

```go
import "google.golang.org/protobuf/encoding/protojson"

err := api.RegisterOfferingServiceServer(handler,
    api.WithMux(mux),
    api.WithMarshalOptions(protojson.MarshalOptions{
        EmitUnpopulated: true,
    }),
)
```

`WithMarshalOptions` is process-scoped (set once at server registration) and threads through every marshal site: standard responses, error bodies, and SSE event streams. Zero-value `MarshalOptions{}` preserves the previous wire output byte-for-byte — opt-in only.

## Run it

```bash
make demo
```

This boots two servers backed by the **same handler**, both returning a `Status{}` with proto3 zero values for every field:

- **`:8080`** — default opts
- **`:8081`** — `WithMarshalOptions(protojson.MarshalOptions{EmitUnpopulated: true})`

In another terminal:

```bash
make test
```

Expected output:

```
=== Default opts (port 8080) — zero values omitted ===
{}

=== EmitUnpopulated: true (port 8081) — zero values surfaced ===
{"acceptingOrders":false, "subscriberCount":0, "note":""}
```

Same handler. Same proto. Different wire output, controlled by one registration-time option.

## Other useful options

`protojson.MarshalOptions` exposes more knobs that flow through the same plumbing:

| Option | Effect |
|---|---|
| `EmitUnpopulated: true` | Surface proto3 zero values (this example) |
| `UseProtoNames: true` | Use snake_case field names (`accepting_orders` not `acceptingOrders`) |
| `UseEnumNumbers: true` | Emit enums as integers, not symbolic names |
| `Multiline: true` + `Indent: "  "` | Pretty-printed output |
| `Resolver: …` | Custom `Any`-type resolution |

See [protojson docs](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson#MarshalOptions) for the full list.
