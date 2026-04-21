---
phase: quick
plan: 260416-kwq
type: execute
wave: 1
depends_on: []
files_modified:
  - CLAUDE.md
autonomous: true
requirements: ["DOC-SSE"]
must_haves:
  truths:
    - "CLAUDE.md documents the stream annotation on HttpConfig"
    - "CLAUDE.md shows generated SSE examples for all 5 generators"
    - "CLAUDE.md annotation registry row notes stream as part of config (ext 50003)"
  artifacts:
    - path: "CLAUDE.md"
      provides: "SSE streaming documentation covering all generators"
      contains: "stream: true"
---

<objective>
Add SSE streaming documentation to CLAUDE.md covering the stream annotation and generated output examples for all 5 generators.

Purpose: The SSE streaming feature was implemented in quick task 260416-eht but CLAUDE.md has zero mention of it. Developers consulting CLAUDE.md need to see how to annotate streaming RPCs and what each generator produces.

Output: Updated CLAUDE.md with SSE streaming sections integrated into the existing documentation structure.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@CLAUDE.md
@proto/sebuf/http/annotations.proto (stream field on HttpConfig)
@internal/httpgen/testdata/proto/sse.proto (example proto usage)
@internal/httpgen/testdata/golden/sse_http.pb.go (Go server generated output)
@internal/clientgen/testdata/golden/sse_client.pb.go (Go client generated output)
@internal/tsclientgen/testdata/golden/sse_client.ts (TS client generated output)
@internal/tsservergen/testdata/golden/sse_server.ts (TS server generated output)
@internal/openapiv3/testdata/golden/yaml/SSEService.openapi.yaml (OpenAPI generated output)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add SSE streaming documentation to CLAUDE.md</name>
  <files>CLAUDE.md</files>
  <action>
Update CLAUDE.md with SSE streaming documentation in three locations:

**1. Add "SSE Streaming Annotation" section after the "Header Annotations" section (after line 220).**

Show the proto annotation usage:
```protobuf
// Mark an RPC as SSE streaming with stream: true on HttpConfig
rpc StreamEvents(StreamEventsRequest) returns (Event) {
  option (sebuf.http.config) = {
    path: "/events"
    method: HTTP_METHOD_GET
    stream: true
  };
};
```

Explain: `stream: true` on the `HttpConfig` annotation changes how all 5 generators handle the RPC. The response message becomes the type of each SSE event, not a single response. Works with path params, query params, and headers.

**2. Add SSE-specific "Generated Output Examples" for each generator, placed after the existing "TypeScript HTTP Servers" example block (after line 139) and before the "OpenAPI Specifications" example block.**

Add these subsections under the existing "Generated Output Examples" heading:

**SSE Streaming (Go Server)** - Show the key difference: SSE methods get `SSESender` parameter:
```go
type MarketDataServer interface {
    // Standard unary RPC
    GetStatus(context.Context, *GetStatusRequest) (*StatusResponse, error)
    // SSE streaming RPC - receives SSESender instead of returning response
    StreamEvents(context.Context, *StreamEventsRequest, SSESender) error
}

// SSESender interface for sending events
type SSESender interface {
    Send(event proto.Message) error
    SendWithEvent(eventType string, event proto.Message) error
    Flush()
}
```
Note: Smart error handling -- errors before first Send() return HTTP error responses; errors after Send() emit an SSE error event (since HTTP 200 is already committed).

**SSE Streaming (Go Client)** - Show EventStream pattern:
```go
// SSE methods return EventStream instead of response
stream, err := client.StreamEvents(ctx, req)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Iterate events -- follows bufio.Scanner / sql.Rows pattern
event := &Event{}
for stream.Next(event) {
    fmt.Printf("Event: %s\n", event.Id)
}
if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```
Note: Uses `bufio.Reader` (not Scanner) to avoid 64KiB token limit on large events.

**SSE Streaming (TypeScript Client)** - Show AsyncGenerator pattern:
```typescript
// SSE methods are async generators
for await (const event of client.streamEvents({})) {
  console.log(event.id, event.type, event.payload);
}

// With abort signal for cancellation
const controller = new AbortController();
for await (const event of client.streamEvents({}, { signal: controller.signal })) {
  if (shouldStop) controller.abort();
}
```
Note: Uses Fetch API ReadableStream with proper line buffering.

**SSE Streaming (TypeScript Server)** - Show ReadableStream return type:
```typescript
export interface MarketDataHandler {
  // Standard unary RPC
  getStatus(ctx: ServerContext, req: GetStatusRequest): Promise<StatusResponse>;
  // SSE streaming RPC - returns ReadableStream instead of Promise
  streamEvents(ctx: ServerContext, req: StreamEventsRequest): ReadableStream<Event>;
}

// Implementation example
const handler: MarketDataHandler = {
  streamEvents(ctx, req) {
    return new ReadableStream({
      start(controller) {
        // Push events
        controller.enqueue({ id: "1", type: "update", payload: "...", timestamp: "..." });
        // Close when done
        controller.close();
      },
    });
  },
};
```
Note: Framework-agnostic via Web Streams API. Works natively in Node 18+, Deno, Bun, Cloudflare Workers.

**SSE Streaming (OpenAPI)** - Show the text/event-stream content type and vendor extension:
```yaml
/api/v1/events:
  get:
    summary: StreamEvents
    responses:
      "200":
        description: Server-Sent Events stream
        content:
          text/event-stream:
            schema:
              type: string
              description: SSE stream. Each event contains a JSON-encoded Event in the data field.
        x-sse-event-schema:
          $ref: '#/components/schemas/Event'
```
Note: Uses `text/event-stream` content type with `x-sse-event-schema` vendor extension pointing to the actual event schema for tooling that supports it.

**3. Update the Annotation Extension Number Registry table.**

The `stream` field is field 3 on `HttpConfig`, which is extension 50003 (`config`). Update the existing row for ext 50003 to clarify it includes stream:

| 50003 | config | MethodOptions | HTTP path, method, and SSE streaming flag |

Change "HTTP path and method" to "HTTP path, method, and SSE streaming flag".
  </action>
  <verify>
    <automated>grep -c "SSE\|stream: true\|SSESender\|EventStream\|AsyncGenerator\|ReadableStream\|text/event-stream" CLAUDE.md | xargs test 7 -le</automated>
  </verify>
  <done>
    - CLAUDE.md contains SSE streaming annotation usage example showing `stream: true`
    - CLAUDE.md contains generated output examples for all 5 generators (Go server SSESender, Go client EventStream, TS client AsyncGenerator, TS server ReadableStream, OpenAPI text/event-stream)
    - Annotation registry row for ext 50003 mentions SSE streaming flag
    - All examples match the actual generated code from golden files
    - Documentation follows the existing CLAUDE.md style and structure
  </done>
</task>

</tasks>

<verification>
- `grep -c "stream: true" CLAUDE.md` returns at least 2 (annotation example + one generated example)
- `grep "SSESender" CLAUDE.md` finds the Go server SSE interface
- `grep "EventStream" CLAUDE.md` finds the Go client pattern
- `grep "AsyncGenerator" CLAUDE.md` finds the TS client pattern
- `grep "ReadableStream" CLAUDE.md` finds the TS server pattern
- `grep "text/event-stream" CLAUDE.md` finds the OpenAPI example
- `grep "SSE streaming flag" CLAUDE.md` finds the updated registry row
</verification>

<success_criteria>
CLAUDE.md fully documents the SSE streaming feature with annotation usage and generated output examples for all 5 generators, integrated naturally into the existing documentation structure.
</success_criteria>

<output>
After completion, create `.planning/quick/260416-kwq-streaming-clients-docs/260416-kwq-SUMMARY.md`
</output>
