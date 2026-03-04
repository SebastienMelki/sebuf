# Python HTTP Client Demo

This example shows how to generate Python contracts and HTTP clients with `protoc-gen-py-client`.

## What It Covers

- required vs optional fields
- flattened nested type names
- `to_dict()` / `from_dict()` helpers
- typed service descriptors
- generated `urllib` service clients
- `query`, `nullable`, `enum_encoding`, `enum_value`, `bytes_encoding`, `timestamp_format`, `empty_behavior`, and `unwrap`

## Generate

```bash
cd examples/python-client-demo
make generate
```

Generated file:

- `gen/demo/contracts/v1/contracts.py`

The generated module includes:

- message and enum contracts
- `ServiceDescriptor` / `MethodDescriptor`
- `ApiException`
- `CatalogServiceClient`

## Proto

The example proto lives at [proto/contracts.proto](./proto/contracts.proto).
