---
phase: quick-260416-eht
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  # Proto annotation
  - proto/sebuf/http/annotations.proto
  - http/annotations.pb.go
  # Shared annotations
  - internal/annotations/http_config.go
  - internal/annotations/method.go
  # Go HTTP server generator
  - internal/httpgen/generator.go
  - internal/httpgen/validation.go
  # Go HTTP client generator
  - internal/clientgen/generator.go
  # TS client generator
  - internal/tsclientgen/generator.go
  # TS server generator
  - internal/tsservergen/generator.go
  # OpenAPI generator
  - internal/openapiv3/generator.go
  # Test protos and golden files
  - internal/httpgen/testdata/proto/sse.proto
  - internal/httpgen/testdata/golden/sse_http.pb.go
  - internal/httpgen/testdata/golden/sse_http_binding.pb.go
  - internal/httpgen/testdata/golden/sse_http_config.pb.go
  - internal/httpgen/golden_test.go
  - internal/clientgen/testdata/proto/sse.proto
  - internal/clientgen/testdata/golden/sse_client.pb.go
  - internal/clientgen/golden_test.go
  - internal/tsclientgen/testdata/proto/sse.proto
  - internal/tsclientgen/testdata/golden/sse_client.ts
  - internal/tsclientgen/golden_test.go
  - internal/tsservergen/testdata/proto/sse.proto
  - internal/tsservergen/testdata/golden/sse_server.ts
  - internal/tsservergen/golden_test.go
  - internal/openapiv3/testdata/proto/sse.proto
  - internal/openapiv3/testdata/golden/SSEService.openapi.yaml
  - internal/openapiv3/exhaustive_golden_test.go
autonomous: true
requirements: [SSE-01]

must_haves:
  truths:
    - "An RPC can be annotated as SSE streaming via sebuf.http.config with stream=true"
    - "Go HTTP server generates an SSE handler that writes text/event-stream with Flusher support"
    - "Go HTTP server SSE interface method signature returns a send function and context, not a single response"
    - "Go HTTP client generates a streaming method that returns an event iterator, not a single response"
    - "TS client generates a streaming method that uses EventSource-style parsing of ReadableStream"
    - "TS server generates an SSE route that returns text/event-stream responses via ReadableStream"
    - "OpenAPI generator produces SSE response schema with text/event-stream content type"
    - "Non-SSE methods remain completely unchanged across all generators"
    - "Golden tests pass for all 5 generators with SSE proto"
  artifacts:
    - path: "proto/sebuf/http/annotations.proto"
      provides: "SSE streaming flag on HttpConfig message"
      contains: "bool stream"
    - path: "internal/annotations/http_config.go"
      provides: "IsSSE/IsStream helper for detecting SSE methods"
      exports: ["HTTPConfig.Stream"]
    - path: "internal/httpgen/generator.go"
      provides: "SSE handler generation for streaming methods"
    - path: "internal/clientgen/generator.go"
      provides: "SSE client method generation with event iterator"
    - path: "internal/tsclientgen/generator.go"
      provides: "SSE client method with ReadableStream parsing"
    - path: "internal/tsservergen/generator.go"
      provides: "SSE server route with ReadableStream response"
    - path: "internal/openapiv3/generator.go"
      provides: "SSE operation with text/event-stream response"
    - path: "internal/httpgen/testdata/proto/sse.proto"
      provides: "Test proto with SSE-annotated RPC methods"
    - path: "internal/httpgen/testdata/golden/sse_http.pb.go"
      provides: "Golden file for SSE HTTP handler generation"
    - path: "internal/clientgen/testdata/golden/sse_client.pb.go"
      provides: "Golden file for SSE Go client generation"
  key_links:
    - from: "proto/sebuf/http/annotations.proto"
      to: "internal/annotations/http_config.go"
      via: "GetMethodHTTPConfig reads stream field"
      pattern: "httpConfig\\.Get.*tream"
    - from: "internal/annotations/http_config.go"
      to: "internal/httpgen/generator.go"
      via: "Generator checks HTTPConfig.Stream to choose SSE vs standard handler"
      pattern: "config\\.Stream|IsSSE"
    - from: "internal/annotations/http_config.go"
      to: "internal/clientgen/generator.go"
      via: "Generator checks HTTPConfig.Stream to choose SSE vs standard client method"
      pattern: "config\\.Stream|cfg\\.isSSE"
---

<objective>
Add Server-Sent Events (SSE) support to all 5 sebuf protoc generators, enabling real-time streaming HTTP APIs from protobuf definitions. This is driven by the Alpaca Go SDK use case which requires SSE for real-time market data streaming.

Purpose: Enable protobuf-defined HTTP APIs to include streaming endpoints using the SSE protocol, which is the standard for server-to-client event streaming over HTTP. SSE is widely used for real-time data feeds (market data, notifications, activity streams) and is the protocol used by Alpaca's trading API.

Output: Updated proto annotations with `stream` flag, all 5 generators producing correct SSE code, golden tests for each generator.
</objective>

<execution_context>
@/Users/sebastienmelki/.claude/get-shit-done/workflows/execute-plan.md
@/Users/sebastienmelki/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@proto/sebuf/http/annotations.proto
@internal/annotations/http_config.go
@internal/annotations/method.go
@internal/httpgen/generator.go
@internal/httpgen/validation.go
@internal/clientgen/generator.go
@internal/tsclientgen/generator.go
@internal/tsservergen/generator.go
@internal/openapiv3/generator.go
@internal/httpgen/golden_test.go
@internal/clientgen/golden_test.go

<interfaces>
<!-- Key types and contracts the executor needs. -->

From proto/sebuf/http/annotations.proto:
```protobuf
message HttpConfig {
  string path = 1;
  HttpMethod method = 2;
  // NEW: bool stream = 3; // marks this RPC as SSE streaming
}
```

From internal/annotations/http_config.go:
```go
type HTTPConfig struct {
  Path       string
  Method     string
  PathParams []string
  // NEW: Stream bool
}

func GetMethodHTTPConfig(method *protogen.Method) *HTTPConfig
```

From internal/annotations/method.go:
```go
func HTTPMethodToString(m http.HttpMethod) string
```

From internal/httpgen/generator.go:
```go
// Current server interface pattern:
// type XxxServer interface {
//     MethodName(context.Context, *Request) (*Response, error)
// }
// SSE methods need a different signature:
// StreamMethodName(context.Context, *Request, XxxSSESender) error

func (g *Generator) generateService(gf *protogen.GeneratedFile, file *protogen.File, service *protogen.Service) error
```

From internal/clientgen/generator.go:
```go
// Current client interface pattern:
// type XxxClient interface {
//     MethodName(ctx, *Request, ...CallOption) (*Response, error)
// }
// SSE methods need:
// StreamMethodName(ctx, *Request, ...CallOption) (*XxxEventStream, error)

type rpcMethodConfig struct {
  serviceName string
  lowerName   string
  methodName  string
  httpMethod  string
  fullPath    string
  pathParams  []string
  queryParams []annotations.QueryParam
  hasBody     bool
  // NEW: isSSE bool
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Proto annotation + shared annotations + all 5 generators + golden tests</name>
  <files>
    proto/sebuf/http/annotations.proto,
    internal/annotations/http_config.go,
    internal/httpgen/generator.go,
    internal/httpgen/validation.go,
    internal/httpgen/golden_test.go,
    internal/httpgen/testdata/proto/sse.proto,
    internal/clientgen/generator.go,
    internal/clientgen/golden_test.go,
    internal/clientgen/testdata/proto/sse.proto,
    internal/tsclientgen/generator.go,
    internal/tsclientgen/golden_test.go,
    internal/tsclientgen/testdata/proto/sse.proto,
    internal/tsservergen/generator.go,
    internal/tsservergen/golden_test.go,
    internal/tsservergen/testdata/proto/sse.proto,
    internal/openapiv3/generator.go,
    internal/openapiv3/exhaustive_golden_test.go,
    internal/openapiv3/testdata/proto/sse.proto
  </files>
  <action>

## SSE Design

SSE (Server-Sent Events) is a simple protocol: the server responds with `Content-Type: text/event-stream` and sends newline-delimited events in the format `data: {json}\n\n`. The connection stays open and the client reads events as they arrive. This maps cleanly to a protobuf RPC where the response message represents a single event that gets sent repeatedly.

## Step 1: Proto Annotation

Add a `stream` field to `HttpConfig` in `proto/sebuf/http/annotations.proto`:

```protobuf
message HttpConfig {
  string path = 1;
  HttpMethod method = 2;
  // When true, this method uses Server-Sent Events (SSE) for streaming responses.
  // The server sends events with Content-Type: text/event-stream.
  // Each event is the response message serialized as JSON in the SSE data field.
  bool stream = 3;
}
```

Then regenerate the Go proto code:
```bash
cd proto/sebuf/http && buf generate
```
If `buf generate` is not configured, use protoc directly to regenerate `http/annotations.pb.go`. The key is that `HttpConfig` gains a `GetStream()` method.

## Step 2: Shared Annotations

In `internal/annotations/http_config.go`, add the `Stream` field to `HTTPConfig`:

```go
type HTTPConfig struct {
  Path       string
  Method     string
  PathParams []string
  Stream     bool
}
```

In `GetMethodHTTPConfig`, populate it:
```go
return &HTTPConfig{
  Path:       path,
  Method:     HTTPMethodToString(httpConfig.GetMethod()),
  PathParams: ExtractPathParams(path),
  Stream:     httpConfig.GetStream(),
}
```

## Step 3: Go HTTP Server Generator (httpgen)

### 3a. SSE Sender Interface and Types

In the generated `_http.pb.go` file, for each service that has SSE methods, generate:

```go
// SSESender allows sending Server-Sent Events to the client.
type SSESender interface {
    // Send sends a single SSE event with the given data.
    // The data will be serialized as JSON in the SSE "data:" field.
    Send(event proto.Message) error
    // SendWithEvent sends an SSE event with a named event type.
    SendWithEvent(eventType string, event proto.Message) error
    // Flush ensures all buffered data is sent to the client.
    // Called automatically after each Send/SendWithEvent.
    Flush()
}
```

### 3b. Server Interface

SSE methods get a different signature on the server interface:

```go
type XxxServer interface {
    // Standard unary method
    GetResource(context.Context, *GetResourceRequest) (*Resource, error)
    // SSE streaming method
    StreamEvents(context.Context, *StreamEventsRequest, SSESender) error
}
```

The SSE method receives an `SSESender` and returns only `error`. The handler is responsible for calling `sender.Send(event)` for each event to stream.

### 3c. Handler Registration

In `generateService`, detect SSE methods and generate different handler registration:

For SSE methods, instead of using `BindingMiddleware` + `genericHandler`, generate a dedicated SSE handler function:

```go
streamEventsHandler := SSEHandler[StreamEventsRequest](
    server.StreamEvents, config.errorHandler, serviceHeaders, methodHeaders,
    streamEventsPathParams, streamEventsQueryParams, "GET",
)
config.mux.Handle("GET /api/v1/events", streamEventsHandler)
```

### 3d. SSE Handler in Binding File

In `generateBindingFile`, add the `SSESender` interface, the `sseSender` implementation struct, and the `SSEHandler` function:

```go
// sseSender implements SSESender using http.ResponseWriter and http.Flusher.
type sseSender struct {
    w       http.ResponseWriter
    flusher http.Flusher
}

func (s *sseSender) Send(event proto.Message) error {
    data, err := protojson.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal SSE event: %w", err)
    }
    _, writeErr := fmt.Fprintf(s.w, "data: %s\n\n", data)
    if writeErr != nil {
        return writeErr
    }
    s.flusher.Flush()
    return nil
}

func (s *sseSender) SendWithEvent(eventType string, event proto.Message) error {
    data, err := protojson.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal SSE event: %w", err)
    }
    _, writeErr := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", eventType, data)
    if writeErr != nil {
        return writeErr
    }
    s.flusher.Flush()
    return nil
}

func (s *sseSender) Flush() {
    s.flusher.Flush()
}

// SSEHandler creates an HTTP handler for SSE streaming methods.
func SSEHandler[Req any](
    handler func(context.Context, *Req, SSESender) error,
    errorHandler ErrorHandler,
    serviceHeaders, methodHeaders []*sebufhttp.Header,
    pathParams []PathParamConfig,
    queryParams []QueryParamConfig,
    httpMethod string,
) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validate headers
        if validationErr := validateHeaders(r, serviceHeaders, methodHeaders); validationErr != nil {
            writeErrorWithHandler(w, r, validationErr, errorHandler)
            return
        }

        // Bind request
        req := new(Req)
        if msg, ok := any(req).(proto.Message); ok {
            if err := bindPathParams(r, msg, pathParams); err != nil {
                writeErrorWithHandler(w, r, err, errorHandler)
                return
            }
            if err := bindQueryParams(r, msg, queryParams); err != nil {
                writeErrorWithHandler(w, r, err, errorHandler)
                return
            }
        }

        // Bind body for POST/PUT/PATCH
        if httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH" {
            if err := bindDataBasedOnContentType(r, req); err != nil {
                validationErr := &sebufhttp.ValidationError{
                    Violations: []*sebufhttp.FieldViolation{{
                        Field:       "body",
                        Description: fmt.Sprintf("failed to parse request body: %v", err),
                    }},
                }
                writeErrorWithHandler(w, r, validationErr, errorHandler)
                return
            }
        }

        // Validate request body
        if msg, ok := any(req).(proto.Message); ok {
            if err := ValidateMessage(msg); err != nil {
                writeErrorWithHandler(w, r, err, errorHandler)
                return
            }
        }

        // Check Flusher support
        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "streaming not supported", http.StatusInternalServerError)
            return
        }

        // Set SSE headers
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")

        sender := &sseSender{w: w, flusher: flusher}

        // Call handler -- blocks until stream completes or context cancels
        if err := handler(r.Context(), req, sender); err != nil {
            // If headers already sent, we can't change status code.
            // Send an SSE error event instead.
            fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
            flusher.Flush()
        }
    })
}
```

### 3e. Validation

In `validation.go`, SSE methods should NOT trigger the "GET with body fields" validation error, because SSE GET methods may have query params but the validation logic is already fine (it only validates GET/DELETE). No changes needed to validation logic itself -- the existing validation covers SSE GET methods correctly.

However, SSE methods should typically be GET. Add a **warning** (not error) if an SSE method uses POST/PUT/PATCH -- SSE is typically GET-based but don't block it.

## Step 4: Go HTTP Client Generator (clientgen)

### 4a. rpcMethodConfig

Add `isSSE bool` to `rpcMethodConfig` struct, set from `httpConfig.Stream`.

### 4b. Client Interface

SSE methods get different signatures:

```go
type XxxClient interface {
    GetResource(ctx context.Context, req *GetResourceRequest, opts ...XxxCallOption) (*Resource, error)
    StreamEvents(ctx context.Context, req *StreamEventsRequest, opts ...XxxCallOption) (*XxxEventStream[*StreamEventsResponse], error)
}
```

### 4c. EventStream Type

Generate a generic `EventStream` type in the client file (once per service that has SSE methods):

```go
// XxxEventStream reads Server-Sent Events from a streaming endpoint.
type XxxEventStream[T proto.Message] struct {
    resp    *http.Response
    scanner *bufio.Scanner
    err     error
}

// Next reads the next event from the stream.
// Returns false when the stream ends or an error occurs.
func (s *XxxEventStream[T]) Next(event T) bool {
    for s.scanner.Scan() {
        line := s.scanner.Text()
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        data := strings.TrimPrefix(line, "data: ")
        if err := protojson.Unmarshal([]byte(data), event); err != nil {
            s.err = fmt.Errorf("failed to unmarshal SSE event: %w", err)
            return false
        }
        return true
    }
    if err := s.scanner.Err(); err != nil {
        s.err = err
    }
    return false
}

// Err returns any error encountered during streaming.
func (s *XxxEventStream[T]) Err() error {
    return s.err
}

// Close closes the underlying HTTP response body.
func (s *XxxEventStream[T]) Close() error {
    return s.resp.Body.Close()
}
```

### 4d. SSE Client Method

Generate a different method body for SSE RPCs:

```go
func (c *xxxClient) StreamEvents(ctx context.Context, req *StreamEventsRequest, opts ...XxxCallOption) (*XxxEventStream[*StreamEventsResponse], error) {
    // ... same call options, URL building, headers as standard ...

    httpReq.Header.Set("Accept", "text/event-stream")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to execute request: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        defer resp.Body.Close()
        // ... standard error handling ...
    }

    return &XxxEventStream[*StreamEventsResponse]{
        resp:    resp,
        scanner: bufio.NewScanner(resp.Body),
    }, nil
}
```

Note: The caller is responsible for calling `Close()` on the EventStream. The `defer resp.Body.Close()` is only used in the error path.

## Step 5: TypeScript Client Generator (tsclientgen)

### 5a. rpcMethodConfig

Add `isSSE bool` to the TS client's `rpcMethodConfig`, set from `httpConfig.Stream`.

### 5b. Client Method Signature

SSE methods return `AsyncIterable<OutputType>` instead of `Promise<OutputType>`:

```typescript
async *streamEvents(req: StreamEventsRequest, options?: XxxCallOptions): AsyncGenerator<StreamEventsResponse> {
```

### 5c. SSE Client Method Body

Generate a method that uses the Fetch API ReadableStream to parse SSE:

```typescript
async *streamEvents(req: StreamEventsRequest, options?: XxxCallOptions): AsyncGenerator<StreamEventsResponse> {
    let path = "/api/v1/events";
    const url = this.baseURL + path;

    const headers: Record<string, string> = {
      "Accept": "text/event-stream",
      ...this.defaultHeaders,
      ...options?.headers,
    };

    const resp = await this.fetchFn(url, {
      method: "GET",
      headers,
      signal: options?.signal,
    });

    if (!resp.ok) {
      return this.handleError(resp);
    }

    const reader = resp.body!.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.slice(6);
            yield JSON.parse(data) as StreamEventsResponse;
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
}
```

## Step 6: TypeScript Server Generator (tsservergen)

### 6a. Handler Interface

SSE methods use a different signature on the handler interface:

```typescript
export interface XxxServiceHandler {
    getResource(ctx: ServerContext, req: GetResourceRequest): Promise<Resource>;
    streamEvents(ctx: ServerContext, req: StreamEventsRequest): ReadableStream<StreamEventsResponse>;
}
```

The handler returns a `ReadableStream` of the event type.

### 6b. Route Handler

The SSE route handler wraps the ReadableStream into an SSE-formatted Response:

```typescript
handler: async (req: Request): Promise<Response> => {
    // ... validate headers, parse query/body/path params as usual ...

    const stream = impl.streamEvents(ctx, body);

    // Convert ReadableStream<T> to SSE text stream
    const sseStream = new ReadableStream({
        async start(controller) {
            const reader = stream.getReader();
            const encoder = new TextEncoder();
            try {
                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;
                    controller.enqueue(encoder.encode(`data: ${JSON.stringify(value)}\n\n`));
                }
                controller.close();
            } catch (err) {
                controller.enqueue(encoder.encode(`event: error\ndata: ${JSON.stringify({message: String(err)})}\n\n`));
                controller.close();
            }
        }
    });

    return new Response(sseStream, {
        headers: {
            "Content-Type": "text/event-stream",
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
        },
    });
}
```

## Step 7: OpenAPI Generator (openapiv3)

### 7a. SSE Response Schema

For SSE methods, instead of the standard JSON response, generate:

```yaml
responses:
  "200":
    description: Server-Sent Events stream
    content:
      text/event-stream:
        schema:
          type: string
          description: "SSE stream. Each event contains a JSON-encoded StreamEventsResponse in the data field."
```

Add a `x-sse-event-schema` vendor extension pointing to the response message schema for tooling:

```yaml
      x-sse-event-schema:
        $ref: '#/components/schemas/StreamEventsResponse'
```

In `processMethod`, detect `isSSE` from the config and call a new `buildSSEResponses` helper instead of the standard `buildResponses`.

## Step 8: Test Proto

Create `sse.proto` in each generator's testdata/proto/ directory (use symlinks from httpgen's copy where possible per project convention). The test proto should cover:

```protobuf
syntax = "proto3";
package test.sse;
option go_package = "github.com/SebastienMelki/sebuf/internal/{gen}/testdata/generated;generated";

import "sebuf/http/annotations.proto";
import "sebuf/http/headers.proto";

service SSEService {
  option (sebuf.http.service_config) = {
    base_path: "/api/v1"
  };

  // Standard unary RPC (should be unaffected)
  rpc GetStatus(GetStatusRequest) returns (StatusResponse) {
    option (sebuf.http.config) = {
      path: "/status"
      method: HTTP_METHOD_GET
    };
  }

  // SSE streaming RPC
  rpc StreamEvents(StreamEventsRequest) returns (Event) {
    option (sebuf.http.config) = {
      path: "/events"
      method: HTTP_METHOD_GET
      stream: true
    };
  }

  // SSE with path params
  rpc StreamResourceEvents(StreamResourceEventsRequest) returns (ResourceEvent) {
    option (sebuf.http.config) = {
      path: "/resources/{resource_id}/events"
      method: HTTP_METHOD_GET
      stream: true
    };
  }

  // SSE with query params
  rpc StreamFilteredEvents(StreamFilteredEventsRequest) returns (Event) {
    option (sebuf.http.config) = {
      path: "/events/filtered"
      method: HTTP_METHOD_GET
      stream: true
    };
  }
}

message GetStatusRequest {}

message StatusResponse {
  string status = 1;
  int64 uptime_seconds = 2;
}

message StreamEventsRequest {}

message Event {
  string id = 1;
  string type = 2;
  string payload = 3;
  int64 timestamp = 4;
}

message StreamResourceEventsRequest {
  string resource_id = 1;
}

message ResourceEvent {
  string resource_id = 1;
  string event_type = 2;
  string data = 3;
}

message StreamFilteredEventsRequest {
  string event_type = 1 [(sebuf.http.query) = { name: "type" }];
  int32 limit = 2 [(sebuf.http.query) = { name: "limit" }];
}
```

## Step 9: Golden Tests

Add the SSE test case to each generator's golden test file:

**httpgen/golden_test.go** -- add test case:
```go
{
    name:      "SSE streaming",
    protoFile: "sse.proto",
    expectedFiles: []string{
        "sse_http.pb.go",
        "sse_http_binding.pb.go",
        "sse_http_config.pb.go",
    },
},
```

**clientgen/golden_test.go** -- add analogous test case for `sse_client.pb.go`.

**tsclientgen/golden_test.go** -- add test case for `sse_client.ts`.

**tsservergen/golden_test.go** -- add test case for `sse_server.ts`.

**openapiv3/exhaustive_golden_test.go** -- add test case for `SSEService.openapi.yaml`.

Then run with `UPDATE_GOLDEN=1` to generate the initial golden files, review them for correctness, and commit.

## Step 10: Build and Test

```bash
# Regenerate proto Go types
cd proto/sebuf/http && protoc --go_out=../../../ --go_opt=module=github.com/SebastienMelki/sebuf annotations.proto

# Clean build
rm -rf bin && make build

# Generate golden files
UPDATE_GOLDEN=1 go test ./internal/httpgen/ -run TestHTTPGenGoldenFiles
UPDATE_GOLDEN=1 go test ./internal/clientgen/ -run TestClientGenGoldenFiles
UPDATE_GOLDEN=1 go test ./internal/tsclientgen/ -run TestTSClientGoldenFiles
UPDATE_GOLDEN=1 go test ./internal/tsservergen/ -run TestTSServerGoldenFiles
UPDATE_GOLDEN=1 go test ./internal/openapiv3/ -run TestExhaustiveGoldenFiles

# Run all tests
./scripts/run_tests.sh

# Lint
make lint-fix
```

## Key Design Decisions

1. **`stream: true` on HttpConfig** -- not a separate annotation. Keeps the config together (path + method + stream). Extension number stays the same since it's a field addition to an existing message.

2. **Go server SSE pattern**: Handler receives `SSESender` interface. This follows the same inversion-of-control pattern as gRPC streaming (where handler receives a stream object). The handler loops, calling `sender.Send()` and checking `ctx.Done()`.

3. **Go client iterator pattern**: Returns `EventStream[T]` with `Next(event T) bool` / `Err()` / `Close()`. This follows Go's `sql.Rows` / `bufio.Scanner` pattern. Caller must `defer stream.Close()`.

4. **TS client async generator**: Uses `async *method()` returning `AsyncGenerator<T>`. This is idiomatic TypeScript for async iteration (`for await (const event of stream) { ... }`).

5. **TS server ReadableStream**: Handler returns `ReadableStream<T>`, the generated route wraps it into SSE format. This is framework-agnostic (Web Streams API).

6. **No separate SSE-specific generated file**: SSE handler code goes into the same `_http_binding.pb.go` / `_client.pb.go` files as the existing code. The types (SSESender, EventStream) are generated once per file that contains SSE methods.

7. **Event format**: Standard SSE `data: {json}\n\n`. Named events supported via `SendWithEvent` on server side. The response message type IS the event type -- each Send() serializes one instance of the response message.
  </action>
  <verify>
    <automated>cd /Users/sebastienmelki/Documents/documents_sebastiens_mac_mini/Workspace/kompani/sebuf.nosync && ./scripts/run_tests.sh --fast</automated>
  </verify>
  <done>
    - `stream: true` field exists on HttpConfig in annotations.proto and compiled to Go
    - HTTPConfig struct in shared annotations has Stream field populated from proto
    - Go HTTP server generates SSESender interface, sseSender struct, SSEHandler function for SSE methods
    - Go HTTP server interface has `Method(ctx, *Req, SSESender) error` for SSE methods, `Method(ctx, *Req) (*Resp, error)` for standard methods
    - Go client generates EventStream type with Next/Err/Close for SSE methods, standard response for non-SSE
    - TS client generates async generator methods for SSE, standard async methods for non-SSE
    - TS server generates ReadableStream-based handlers for SSE, standard Promise handlers for non-SSE
    - OpenAPI generates text/event-stream response for SSE methods, application/json for standard
    - All existing golden tests still pass (no regression)
    - New SSE golden tests pass for all 5 generators
    - `make lint-fix` reports 0 issues
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| client -> SSE endpoint | Untrusted client connects to streaming endpoint |
| SSE data serialization | Server serializes proto messages into SSE data field |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-sse-01 | D (Denial of Service) | SSE handler | mitigate | SSE handler uses request context -- when client disconnects, ctx.Done() fires and handler should return. Generated code checks ctx in the SSEHandler wrapper. |
| T-sse-02 | I (Information Disclosure) | SSE error event | accept | On handler error after headers sent, error message is sent as SSE event. This matches standard SSE error handling. Handler implementors should not leak sensitive info in errors. |
| T-sse-03 | T (Tampering) | SSE client parsing | mitigate | Client EventStream parses only `data:` prefixed lines, ignoring other SSE fields. Uses protojson.Unmarshal which validates against schema. |
</threat_model>

<verification>
1. `./scripts/run_tests.sh --fast` passes (all test packages, including new SSE golden tests)
2. `make lint-fix` reports 0 issues
3. Golden files reviewed: SSE methods produce different generated code, non-SSE methods unchanged
4. Test proto has both SSE and non-SSE methods in same service to verify no interference
</verification>

<success_criteria>
- All 5 generators support `stream: true` annotation on HttpConfig
- Generated Go server SSE handler uses text/event-stream, Flusher, proper headers
- Generated Go client provides iterator-style event reading with Close()
- Generated TS client uses async generator with ReadableStream parsing
- Generated TS server returns ReadableStream wrapped as SSE Response
- Generated OpenAPI uses text/event-stream content type for SSE operations
- All existing tests pass (zero regression)
- New SSE golden tests pass for all 5 generators
</success_criteria>

<output>
After completion, create `.planning/quick/260416-eht-implement-sse/260416-eht-SUMMARY.md`
</output>
