# Python HTTP Client Generation

> **Generate type-safe Python HTTP clients from your protobuf services**

The `protoc-gen-py-client` plugin generates type-safe Python HTTP clients that mirror your protobuf services. Output depends only on the Python standard library; users can plug in `requests`, `httpx`, `aiohttp`, or any other HTTP library by passing a duck-typed transport at construction time.

**Minimum Python version: 3.10.** The generated code uses `X | None` syntax, `list[T]` / `dict[K, V]`, `type[Any]`, and `from __future__ import annotations`.

## Quick Start

### Installation

```bash
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-py-client@latest
```

### Configuration

Add the plugin to your `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: api
    opt: paths=source_relative
  - local: protoc-gen-go-http
    out: api
  - local: protoc-gen-py-client
    out: client/generated
    opt: paths=source_relative
```

For each `.proto` source the plugin emits a single `<file>_client.py` containing every message dataclass, enum, error class, transport, and one client class per service in that file.

### Basic Usage

```python
from note_service_client import (
    NoteServiceClient,
    NoteServiceClientOptions,
    GetNoteRequest,
)

client = NoteServiceClient(
    "http://localhost:8080",
    NoteServiceClientOptions(api_key="your-api-key"),
)

note = client.get_note(GetNoteRequest(id="note-123"))
print(note.title)
```

## Generated Components

For each input file the generator emits:

### 1. Message dataclasses

Every proto message becomes a `@dataclass` with `to_dict()` and `from_dict()` for JSON serialization that honors every JSON-mapping annotation.

```python
@dataclass
class Note:
    """Generated from proto message demo.Note."""
    id: str = ""
    title: str = ""
    content: str = ""
    priority: Priority = Priority.PRIORITY_UNSPECIFIED
    tags: list[Tag] = field(default_factory=list)
    metadata: dict[str, str] = field(default_factory=dict)
    due_date: Optional[str] = None

    def to_dict(self) -> Any: ...
    @classmethod
    def from_dict(cls, data: Any) -> "Note": ...
```

### 2. Enums

Proto enums are emitted as `IntEnum` subclasses. Variant names are preserved verbatim so that `IntEnum.name` matches the Go protojson default wire form:

```python
class Priority(IntEnum):
    """Generated from proto enum demo.Priority."""
    PRIORITY_UNSPECIFIED = 0
    PRIORITY_LOW = 1
    PRIORITY_MEDIUM = 2
    PRIORITY_HIGH = 3
    PRIORITY_URGENT = 4
```

With `(sebuf.http.enum_value)` overrides, a sibling `Priority_JSON_VALUES` dict maps members to their custom JSON strings.

### 3. Transport Protocol

The default transport is built on `urllib.request`:

```python
class HttpTransport(Protocol):
    def request(
        self,
        method: str,
        url: str,
        headers: Mapping[str, str],
        body: Optional[bytes],
        timeout: Optional[float],
    ) -> HttpResponse: ...

class UrllibTransport:
    """Default transport built on the Python standard library."""
    def request(self, method, url, headers, body, timeout) -> HttpResponse: ...
```

Any object with a matching `.request(...)` method is accepted — no inheritance required. See [Transport Injection](#transport-injection) below.

### 4. Error hierarchy

```python
class ApiError(Exception):
    status: int
    body: bytes
    headers: Mapping[str, str]

class ValidationError(ApiError):
    violations: list[FieldViolation]

# One subclass per proto message ending in "Error" — populated from the
# response body by the client when the JSON shape matches.
class NotFoundError(ApiError):
    resource_type: str
    resource_id: str
```

See [Error Handling](#error-handling).

### 5. Client options and call options

```python
@dataclass
class NoteServiceClientOptions:
    """Construct-time options."""
    transport: Optional[HttpTransport] = None
    default_headers: Optional[Mapping[str, str]] = None
    timeout: Optional[float] = None
    content_type: str = "application/json"
    # Typed kwarg per service-level header annotation:
    api_key: Optional[str] = None      # from X-API-Key
    tenant_id: Optional[str] = None    # from X-Tenant-ID

@dataclass
class NoteServiceCallOptions:
    """Per-call options passed as the 2nd arg of every RPC method."""
    headers: Optional[Mapping[str, str]] = None
    timeout: Optional[float] = None
    content_type: Optional[str] = None
    # Typed kwarg per service- AND method-level header annotation.
    api_key: Optional[str] = None       # service-level (override per call)
    tenant_id: Optional[str] = None     # service-level (override per call)
    request_id: Optional[str] = None    # method-level (e.g. CreateNote)
    idempotency_key: Optional[str] = None  # method-level (e.g. UpdateNote)
```

### 6. Client class

```python
class NoteServiceClient:
    def __init__(self, base_url: str, options: Optional[NoteServiceClientOptions] = None) -> None: ...

    def list_notes(self, req: ListNotesRequest, options: Optional[NoteServiceCallOptions] = None) -> ListNotesResponse: ...
    def get_note(self, req: GetNoteRequest, options: Optional[NoteServiceCallOptions] = None) -> Note: ...
    def create_note(self, req: CreateNoteRequest, options: Optional[NoteServiceCallOptions] = None) -> Note: ...
    # ...
```

RPC methods are snake_cased automatically (`CreateNote` → `create_note`).

## Transport Injection

The generated client has no third-party HTTP dependencies. Swap in `requests`, `httpx`, or your own middleware by passing any object with the right `.request()` shape:

```python
import requests
from note_service_client import HttpResponse, NoteServiceClient, NoteServiceClientOptions

class RequestsTransport:
    def __init__(self, session: requests.Session | None = None):
        self._session = session or requests.Session()

    def request(self, method, url, headers, body, timeout):
        resp = self._session.request(method, url, headers=dict(headers), data=body, timeout=timeout)
        return HttpResponse(status=resp.status_code, headers=dict(resp.headers), body=resp.content)

client = NoteServiceClient(
    "http://localhost:3000",
    NoteServiceClientOptions(transport=RequestsTransport()),
)
```

Middleware wraps any transport. The python-client-demo includes a logging interceptor:

```python
class LoggingTransport:
    def __init__(self, inner: HttpTransport):
        self._inner = inner
    def request(self, method, url, headers, body, timeout):
        print(f"[LOG] -> {method} {url}")
        return self._inner.request(method, url, headers, body, timeout)
```

## URL Building

### Path Parameters

Substituted directly from request fields and URL-encoded:

```protobuf
rpc GetNote(GetNoteRequest) returns (Note) {
  option (sebuf.http.config) = {
    path: "/notes/{id}"
    method: HTTP_METHOD_GET
  };
}
```

```python
client.get_note(GetNoteRequest(id="note-123"))
# GET /notes/note-123
```

### Query Parameters

For `GET` and `DELETE`, fields annotated with `(sebuf.http.query)` become URL query parameters. Repeated fields use `urlencode(..., doseq=True)`:

```protobuf
message ListNotesRequest {
  string status = 1 [(sebuf.http.query) = {name: "status"}];
  int32 limit  = 2 [(sebuf.http.query) = {name: "limit"}];
  int32 offset = 3 [(sebuf.http.query) = {name: "offset"}];
}
```

```python
client.list_notes(ListNotesRequest(status="pending", limit=10, offset=0))
# GET /notes?status=pending&limit=10&offset=0
```

## Header Management

Headers come from three layers, in increasing priority:

1. **`default_headers`** on `*ClientOptions` — applied to every request.
2. **Service headers** declared with `(sebuf.http.service_headers)` — typed kwargs on both client and call options (call wins).
3. **Method headers** declared with `(sebuf.http.method_headers)` — typed kwargs on call options only.
4. **Per-call `headers` field** on call options — last write wins.

```protobuf
service NoteService {
  option (sebuf.http.service_headers) = {
    required_headers: [
      { name: "X-API-Key"   type: "string" required: true format: "uuid" },
      { name: "X-Tenant-ID" type: "integer" required: true }
    ]
  };

  rpc CreateNote(CreateNoteRequest) returns (Note) {
    option (sebuf.http.config) = { path: "/notes" method: HTTP_METHOD_POST };
    option (sebuf.http.method_headers) = {
      required_headers: [{ name: "X-Request-ID" type: "string" required: true }]
    };
  }
}
```

```python
client = NoteServiceClient(
    "http://localhost:3000",
    NoteServiceClientOptions(
        api_key="550e8400-...",
        tenant_id="42",
        default_headers={"X-Custom-Header": "demo"},
    ),
)

# Method header passed as a typed kwarg on call options:
client.create_note(req, NoteServiceCallOptions(request_id="req-001"))

# Per-call service header override + arbitrary extra headers:
client.get_note(
    GetNoteRequest(id="note-1"),
    NoteServiceCallOptions(tenant_id="99", headers={"X-Trace-ID": "trace-abc"}),
)
```

HTTP header names follow PEP-8-style snake_case kwargs: `X-API-Key` → `api_key`, `X-Request-ID` → `request_id`. The original header name is always sent on the wire.

## Error Handling

The client raises typed exceptions on non-2xx responses:

| Status | Body shape | Exception |
|--------|------------|-----------|
| 400 | `{"violations": [...]}` | `ValidationError` (with `.violations: list[FieldViolation]`) |
| any | matches a registered `*Error` proto shape | the matching subclass (e.g. `NotFoundError`) |
| any | anything else | `ApiError` |

### Custom proto errors

Any proto message whose name ends in `Error` becomes a generated `ApiError` subclass:

```protobuf
message NotFoundError {
  string resource_type = 1;
  string resource_id = 2;
}
```

```python
try:
    client.get_note(GetNoteRequest(id="does-not-exist"))
except NotFoundError as e:
    print(f"status={e.status} type={e.resource_type} id={e.resource_id}")
except ApiError as e:
    # Catch-all for unknown error shapes.
    print(f"unexpected error: status={e.status} body={e.body!r}")
```

Selection between subclasses uses a `_ERROR_CLASSES` registry keyed by the JSON field-name set of each error message. The client picks the first registered class whose required keys are all present in the response body. Empty-field error messages match nothing automatically — they require an explicit `except` clause.

### Validation errors

`buf.validate` violations from the Go HTTP server are parsed into typed `FieldViolation`s:

```python
from note_service_client import ValidationError

try:
    client.create_note(CreateNoteRequest(title=""), call_opts)
except ValidationError as e:
    for v in e.violations:
        print(f"  {v.field}: {v.description}")
```

The same shape covers missing required headers — the violation's `field` is the header name.

## JSON Mapping Annotations

Every `(sebuf.http.*)` JSON-mapping annotation is honored by the Python generator. The wire format matches the Go and TypeScript generators byte-for-byte. Full reference: [JSON / Protobuf Compatibility](./json-protobuf-compatibility.md).

Supported annotations: `int64_encoding`, `enum_encoding` + `enum_value`, `nullable`, `empty_behavior`, `timestamp_format`, `bytes_encoding`, `oneof_config` + `oneof_value`, `flatten` + `flatten_prefix`, `unwrap` (map-value and root).

### Timestamps

`google.protobuf.Timestamp` always types as `datetime`. `timestamp_format` affects the wire encoding only:

```protobuf
message Event {
  google.protobuf.Timestamp created_at  = 1;
  google.protobuf.Timestamp unix_ts     = 2 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS];
  google.protobuf.Timestamp unix_millis = 3 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_MILLIS];
  google.protobuf.Timestamp date_only   = 4 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_DATE];
}
```

```python
@dataclass
class Event:
    created_at:  Optional[datetime] = None  # RFC3339 wire format
    unix_ts:     Optional[datetime] = None  # int seconds wire format
    unix_millis: Optional[datetime] = None  # int millis wire format
    date_only:   Optional[datetime] = None  # "YYYY-MM-DD" wire format
```

### int64 encoding

`int64` and `uint64` default to a string wire form (JS-safe). The Python field type follows the wire:

```protobuf
message Order {
  int64 amount = 1;  // default: STRING encoding
  uint64 id    = 2 [(sebuf.http.int64_encoding) = INT64_ENCODING_NUMBER];
}
```

```python
@dataclass
class Order:
    amount: str = "0"  # JSON: "12345"
    id:     int = 0    # JSON: 12345
```

### bytes encoding

```python
data = b"Hello"
# Default: base64 "SGVsbG8="
# BYTES_ENCODING_HEX: "48656c6c6f"
# BYTES_ENCODING_BASE64URL: URL-safe base64
# BYTES_ENCODING_BASE64_RAW / _BASE64URL_RAW: no padding
```

### Discriminated oneofs

A oneof carrying `(sebuf.http.oneof_config)` deserializes into Optional fields on the parent dataclass; the discriminator key selects which variant is populated. The `flatten: true` option promotes the variant's fields into the parent.

## Python Keywords

Field, method, and header names that collide with Python keywords are escaped with a trailing underscore (`from` → `from_`, `class` → `class_`). Both hard (`return`, `def`, …) and soft (`match`, `case`) Python 3.10 keywords are escaped. The wire JSON name is always preserved.

## Server-Sent Events

SSE streaming RPCs (`stream: true` on `HttpConfig`) are detected and emit method stubs that raise `NotImplementedError`. Full SSE support is tracked in a follow-up issue — when implemented, SSE methods will return an `Iterator[T]` (sync) and / or `AsyncIterator[T]` (async).

```python
# Generated for an SSE RPC:
def stream_events(self, req: StreamEventsRequest, options: Optional[CallOptions] = None) -> Iterator[Event]:
    """SSE streaming is not yet supported by protoc-gen-py-client."""
    raise NotImplementedError("...")
```

## Known Limitations

- **Content-Type**: JSON only. Calling an RPC with `content_type="application/x-protobuf"` raises `NotImplementedError`. Protobuf binary support would force a `google-protobuf` dependency on every consumer and is deferred to a follow-up.
- **SSE streaming**: detection only — see [Server-Sent Events](#server-sent-events).
- **Async transport**: only a sync `HttpTransport` shape today. An async variant returning `Awaitable[HttpResponse]` is tracked separately.
- **Generated docstrings**: a stub comment per class only; proto comments are not currently surfaced.
- **`__init__.py` ergonomics**: imports today are per-file (`from note_service_client import ...`). A follow-up will emit `__init__.py` re-exports.

## Complete Example

The repository ships an end-to-end example at `examples/python-client-demo/`. It uses the same Go HTTP server as the TypeScript demo so the two client surfaces can be compared directly.

```bash
cd examples/python-client-demo
make demo
```

The demo exercises every feature documented above in seven labelled sections (configuration, CRUD, query params, header layering, validation errors, typed proto errors, custom transport + unwrap response).

## See Also

- [Go HTTP Client Generation](./client-generation.md) — the Go client surface this Python client mirrors.
- [JSON / Protobuf Compatibility](./json-protobuf-compatibility.md) — wire format reference shared by every sebuf generator.
- [HTTP Handler Generation](./http-generation.md) — the Go server side that pairs with this client.
- [Validation](./validation.md) — `buf.validate` rules surfaced as `ValidationError`.
