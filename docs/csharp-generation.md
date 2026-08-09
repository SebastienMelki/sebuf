# C# HTTP Client Generation

`protoc-gen-csharp-http` generates nullable-aware C# DTOs and typed HTTP clients
from Sebuf service and JSON-mapping annotations. It is intended for C# callers
of a Sebuf HTTP API; it does not generate an ASP.NET server or replace the
protobuf C# runtime.

The generator supports two JSON libraries:

- `newtonsoft` (the default), using Newtonsoft.Json;
- `system_text_json`, using System.Text.Json and no Newtonsoft dependency.

Generated request and response DTOs are used by the generated clients. If an
application serializes DTOs independently, it is responsible for selecting and
configuring the same serializer behavior required by its API. In particular,
the client's protocol normalization is part of the generated client path, not
a promise that arbitrary direct DTO serialization has identical wire behavior.

## Install and configure

Install the plugin where `buf`/`protoc` can find it:

```sh
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-csharp-http@latest
```

Add it to `buf.gen.yaml`. The `namespace` option is the root C# namespace; the
generator appends the protobuf package when more than one package is generated
or packages reference each other. `json_lib` accepts `newtonsoft` or
`system_text_json`.

```yaml
version: v2
plugins:
  - local: protoc-gen-csharp-http
    out: generated/csharp
    opt:
      - namespace=Acme.Contracts
      - json_lib=system_text_json
```

Run generation with:

```sh
buf generate
```

For a protobuf package such as `acme.billing.v1`, output is written under
`generated/csharp/acme/billing/v1/Contracts.g.cs`. With a single independent
package, generated declarations use the configured namespace directly. For
multiple packages or cross-package contracts, declarations are placed under
package-derived namespace segments to keep same-named messages unambiguous.
Generated files are derived artifacts and must not be edited by hand.

For `newtonsoft`, add a compatible Newtonsoft.Json package reference to the
consumer project. `system_text_json` uses the .NET JSON APIs included with
modern .NET targets.

## Generated surface

Each package file can contain:

- DTO classes and enums for the package's contracts;
- `I<Service>Client` and `<Service>Client` for annotated services;
- `<Service>ClientOptions` and `<Service>CallOptions` for transport, headers,
  timeout, and content type;
- `ApiException`, `ValidationException`, and typed exception classes for
  protobuf messages whose names end in `Error`.

Create a client with an application-owned `HttpClient` when possible:

```csharp
using var httpClient = new HttpClient();
var client = new WidgetServiceClient("https://api.example.test", new WidgetServiceClientOptions
{
    HttpClient = httpClient,
    Timeout = TimeSpan.FromSeconds(20),
    DefaultHeaders = new Dictionary<string, string>
    {
        ["X-Client-Version"] = "1.2.0"
    },
    ApiKey = "service-header-value",
});

var widget = await client.GetWidgetAsync(
    new GetWidgetRequest { Id = "w-123" },
    new WidgetServiceCallOptions
    {
        RequestId = "request-123",
        Headers = new Dictionary<string, string> { ["X-Trace-ID"] = "trace-123" }
    },
    cancellationToken);
```

The generated client accepts an injected `HttpClient`; otherwise it constructs
one. Service header annotations become client-option properties, and service or
method header annotations become call-option properties. Generic default and
per-call headers are merged, with the per-call value taking precedence.

Only `application/json` content is currently supported. Supplying another
content type fails explicitly with `NotSupportedException`. A call timeout can
be set at client or call scope and is combined with the supplied
`CancellationToken`.

## HTTP and JSON behavior

For every unary annotated RPC, the generated method uses the configured HTTP
verb and path. Path parameters are URI-escaped. Annotated query fields on GET
and DELETE requests are encoded as query values (including repeated values).
Methods with a request body serialize the request as JSON; bodyless methods do
not send a JSON body.

The client implements the Sebuf JSON mapping needed at the HTTP boundary,
including:

- configurable bytes and enum encodings;
- protobuf null and empty-value semantics;
- flattened fields and root unwrap behavior;
- oneof discriminator inference and mutually-exclusive variant cleanup;
- cross-package contract references with correct generated namespaces.

Successful empty responses become a default response DTO. Non-successful
responses retain status, body, and response headers in `ApiException`. A 400
body containing `violations` becomes `ValidationException`; a JSON error body
matching a generated protobuf `*Error` message becomes that message's typed
exception. Malformed or unrecognized error bodies safely fall back to
`ApiException`.

## Streaming limitation

SSE/streaming annotations are recognized, but C# SSE consumption is not yet
implemented. Calling a generated streaming method throws
`NotSupportedException` immediately. Do not use the C# client generator for an
SSE endpoint until streaming support is added; use a dedicated streaming client
instead.

## Testing changes

The generator has focused golden, compile, and runtime tests. From the
repository root:

```sh
go test ./internal/csharpgen
```

The compile and runtime coverage needs `protoc` and the .NET SDK. The runtime
test exercises both JSON library choices against an injected fake
`HttpMessageHandler`, covering routing, headers, query/path/body encoding,
error dispatch, cancellation, and JSON normalization. The test skips only the
environment-dependent portions when those tools are unavailable; run it in CI
with both tools installed for complete coverage.
