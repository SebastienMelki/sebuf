# C# Contracts Demo

This example shows how to generate C# contracts and route metadata with `protoc-gen-csharp-http`.

## What It Covers

- flattened message fields with `flatten` and `flatten_prefix`
- discriminated oneofs with `oneof_config`
- nullable contract fields
- root unwrap collection contracts
- service route metadata
- both `newtonsoft` and `System.Text.Json` output modes

## Generate

```bash
cd examples/csharp-contracts-demo
make generate
```

Generated files:

- `gen/newtonsoft/demo/contracts/v1/Contracts.g.cs`
- `gen/system-text-json/demo/contracts/v1/Contracts.g.cs`

## Proto

The example proto lives at [proto/contracts.proto](./proto/contracts.proto).
