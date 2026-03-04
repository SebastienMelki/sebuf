# Python HTTP Client Generation

> **Generate Python contracts and HTTP clients from protobuf services**

`protoc-gen-py-client` generates Python dataclasses, enum types, typed service descriptors, and `urllib`-based HTTP clients from annotated protobuf packages. It is intended for typed SDKs, integration scripts, and Python applications that need the same JSON-facing contract behavior as the other sebuf generators.

## Quick Start

### Installation

```bash
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-py-client@latest
```

### Buf Configuration

Add the plugin to `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - local: protoc-gen-py-client
    out: gen/python
    opt:
      - package=acme/contracts/v1
```

### Protoc Usage

```bash
protoc \
  --plugin=protoc-gen-py-client="$(go env GOPATH)/bin/protoc-gen-py-client" \
  --py-client_out=gen/python \
  --py-client_opt=package=acme/contracts/v1 \
  --proto_path=. \
  --proto_path=./proto \
  proto/example/v1/service.proto
```

## Generated Output

For each generated protobuf package, the plugin emits:

- Python `IntEnum` types for protobuf enums
- Python `@dataclass` contract types for protobuf messages
- `to_dict()` and `from_dict()` helpers for JSON payload conversion
- `ServiceDescriptor` and `MethodDescriptor` metadata
- `{Service}Client` HTTP clients built on `urllib.request`
- `ApiException` for non-2xx responses

Nested messages and enums are flattened into Python-friendly names such as `WidgetProfile` and `WidgetState`.

## Supported Options

### Generator Options

- `package`
  Overrides the output package path. Example:
  - `package=acme/contracts/v1`
  - output file: `gen/python/acme/contracts/v1/contracts.py`

## JSON Contract Behavior

The generated Python contracts reflect the JSON-facing shape for the supported annotations below.

### Field and Message Shape

- `nullable`
  Nullable JSON fields become `T | None`
- `flatten`
  Preserves flattened JSON field naming from the shared contract model
- `oneof_config`
  Emits discriminator-aware serialization for configured oneofs
- `unwrap`
  Supports root unwrap messages and unwrap-aware collection serialization
- `empty_behavior`
  Preserves `NULL` and `OMIT` semantics when serializing empty child messages

### Value Encoding

- `int64_encoding`
  Uses `int` for numeric encoding and `str` otherwise
- `enum_encoding`
  Supports numeric enum payloads or string enum payloads
- `enum_value`
  Applies custom string values during JSON serialization and parsing
- `timestamp_format`
  Maps timestamp fields to `int` or `str` depending on configured format
- `bytes_encoding`
  Supports `base64`, `base64_raw`, `base64url`, `base64url_raw`, and `hex`

## Client Runtime

For each protobuf service, the generator emits a `{Service}Client` with:

- path parameter replacement
- query parameter extraction from annotated request fields
- JSON request body serialization
- JSON response parsing into generated dataclasses
- `ApiException` on non-success HTTP responses

Example:

```python
client = WidgetServiceClient("https://api.example.com", headers={"Authorization": "Bearer token"})
widget = client.get_widget(GetWidgetRequest(id="w_123"))
```

## Example

Proto:

```proto
message Widget {
  string id = 1;
  optional string display_name = 2 [(sebuf.http.nullable) = true];
  bytes payload = 3 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_HEX];
}
```

Generated Python:

```python
@dataclass(slots=True)
class Widget:
    id: str = ""
    display_name: str | None = None
    payload: bytes = b""
```

See [examples/python-client-demo](../examples/python-client-demo/) for a working generation example.
