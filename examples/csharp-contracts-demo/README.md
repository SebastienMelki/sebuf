# C# HTTP Client Demo

This example shows how to generate C# contracts and `HttpClient` service clients with `protoc-gen-csharp-http`.

## What It Covers

- flattened message fields with `flatten` and `flatten_prefix`
- discriminated oneofs with `oneof_config`
- nullable contract fields
- root unwrap collection contracts
- generated `I{Service}Client` / `{Service}Client` types
- request/response JSON handling for `unwrap` and `bytes_encoding`
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

Each generated file includes:

- message and enum contracts
- service clients and per-call options
- `ApiException`
- `ServiceContracts` metadata

## Proto

The example proto lives at [proto/contracts.proto](./proto/contracts.proto).
