# C# HTTP Client Generation

> **Generate C# contracts and `HttpClient` service clients from protobuf services**

`protoc-gen-csharp-http` generates C# contract types and `HttpClient`-based service clients from annotated protobuf packages. It is designed for SDKs, typed API integrations, and shared contracts where C# needs the same JSON-facing shape and HTTP calling surface as other sebuf generators.

## Quick Start

### Installation

```bash
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-csharp-http@latest
```

### Buf Configuration

Add the plugin to `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - local: protoc-gen-csharp-http
    out: gen/csharp
    opt:
      - namespace=Acme.Contracts
      - json_lib=system_text_json
```

### Protoc Usage

```bash
protoc \
  --plugin=protoc-gen-csharp-http="$(go env GOPATH)/bin/protoc-gen-csharp-http" \
  --csharp-http_out=gen/csharp \
  --csharp-http_opt=namespace=Acme.Contracts,json_lib=newtonsoft \
  --proto_path=. \
  --proto_path=./proto \
  proto/example/v1/service.proto
```

## Generated Output

For each generated package, the plugin emits one `Contracts.g.cs` file containing:

- C# `enum` types for protobuf enums
- C# classes for protobuf messages
- `I{Service}Client` and `{Service}Client` types built on `HttpClient`
- `{Service}ClientOptions` and `{Service}CallOptions` for headers and transport configuration
- `ApiException` for non-success responses
- `ServiceContracts` metadata with service name, base path, HTTP method, route, request type, and response type per RPC

Nested protobuf messages and enums are flattened into idiomatic C# names such as `WidgetProfile` and `WidgetState`.

## Supported Options

### Generator Options

- `namespace`
  Sets the C# namespace. Default: `Sebuf.Generated`
- `json_lib`
  Chooses JSON attributes and converters. Supported values:
  - `newtonsoft`
  - `system_text_json`

## JSON Contract Behavior

The generator reflects the JSON-facing contract shape for the supported annotations below.

### Field and Message Shape

- `flatten`
  Flattens child message fields into the parent contract, honoring `flatten_prefix`
- `oneof_config`
  Emits discriminator properties and flattened discriminated-union fields when configured
- `unwrap`
  Root unwrap messages generate collection-shaped contracts such as `List<T>`, and map-value unwrap is preserved during client request/response serialization
- `nullable`
  Uses nullable C# reference/value types where the JSON contract can be `null`
- `empty_behavior`
  Uses nullable contract fields for `NULL` and `OMIT` empty-message behavior

### Value Encoding

- `int64_encoding`
  Maps `int64` JSON number encoding to `long`; otherwise uses `string`
- `enum_encoding`
  Supports numeric enums or string enums with JSON converters
- `enum_value`
  Applies custom string values via `[EnumMember(Value = "...")]`
- `timestamp_format`
  Maps timestamp fields to `string` or `long` depending on configured format
- `bytes_encoding`
  Represents bytes fields as `byte[]` and re-encodes on the wire for `hex`, `base64_raw`, `base64url`, and `base64url_raw`

## Example

Proto:

```proto
message Widget {
  optional string display_name = 1 [(sebuf.http.nullable) = true];
  Profile profile = 2 [(sebuf.http.flatten) = true, (sebuf.http.flatten_prefix) = "meta_"];

  message Profile {
    string note = 1;
  }
}
```

Generated C#:

```csharp
public sealed class Widget
{
    [JsonProperty("display_name")]
    public string? DisplayName { get; set; }

    [JsonProperty("meta_note")]
    public string? MetaNote { get; set; }
}
```

## Client Runtime

For each protobuf service, the generator emits:

- `IWidgetServiceClient`
- `WidgetServiceClient`
- `WidgetServiceClientOptions`
- `WidgetServiceCallOptions`

Generated clients:

- use `HttpClient`
- build paths from annotated route params
- add annotated query params for `GET` / `DELETE`
- apply service-level and method-level headers
- serialize request bodies as JSON
- deserialize JSON responses into generated contracts
- preserve `unwrap` and `bytes_encoding` wire behavior
- throw `ApiException` for non-2xx responses

See [examples/csharp-contracts-demo](../examples/csharp-contracts-demo/) for a working generation example.
