---
phase: quick-260416-eht
status: complete
date: 2026-04-16
commit: 8759337
---

# Quick Task 260416-eht: Implement SSE Streaming Support

## What Was Done

Added Server-Sent Events (SSE) streaming support across all 5 sebuf protoc generators, enabling real-time streaming HTTP APIs from protobuf definitions.

### Changes

1. **Proto annotation** (`proto/sebuf/http/annotations.proto`): Added `bool stream = 3` to `HttpConfig` message, regenerated Go proto code
2. **Shared annotations** (`internal/annotations/http_config.go`): Added `Stream bool` field to `HTTPConfig` struct, populated from proto
3. **Go HTTP server** (`internal/httpgen/generator.go`): Added `SSESender` interface, `sseSender` struct, and `SSEHandler[Req]` generic handler with `text/event-stream` headers, `http.Flusher` support, and request binding (path params, query params, body, validation)
4. **Go HTTP client** (`internal/clientgen/generator.go`): Added `EventStream[T]` generic type with `Next(T) bool` / `Err()` / `Close()` following `bufio.Scanner` pattern, and SSE-specific method generation
5. **TS client** (`internal/tsclientgen/generator.go`): Added `async *method()` returning `AsyncGenerator<T>` with `ReadableStream` SSE parsing
6. **TS server** (`internal/tsservergen/generator.go`): Added SSE route generation where handler returns `ReadableStream<T>`, route wraps into SSE-formatted `Response`
7. **OpenAPI** (`internal/openapiv3/generator.go`): Added `text/event-stream` content type with `x-sse-event-schema` vendor extension for SSE operations

### Test Coverage

- SSE test proto covers: standard unary RPC + SSE streaming + SSE with path params + SSE with query params (all in same service)
- Golden tests pass for all 5 generators (14 test cases each for tsclientgen/tsservergen, 9+ for others)
- Non-SSE methods verified unchanged across all generators
- All 8 test packages pass, 0 lint issues

## Outcome

SSE streaming is now a first-class feature in sebuf. Users can annotate RPCs with `stream: true` on `sebuf.http.config` to generate streaming endpoints across all supported languages and documentation formats.
