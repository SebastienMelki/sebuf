# Error Handler Example

This example demonstrates the `WithErrorHandler` ServerOption feature of sebuf, which allows you to customize how errors are handled and returned to clients.

## Overview

The `ErrorHandler` function is called whenever an error occurs during request processing. You can use it to:

1. **Log errors** without modifying the response
2. **Add custom headers** to error responses
3. **Set custom HTTP status codes** for specific error types
4. **Return custom error response bodies** (proto.Message)
5. **Write directly to the response** for full control

## Quick Start

```bash
# Generate code and build
make all

# Run the server
make run
```

## Error Handler Examples

### 1. Logging Only

Just log errors without changing the response:

```go
func LoggingErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
    log.Printf("[%s] Error: %v", r.Header.Get("X-Request-ID"), err)
    return nil // Use default response
}
```

### 2. Add Custom Headers

Add tracking headers while keeping default response:

```go
func HeaderErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
    w.Header().Set("X-Error-ID", uuid.NewString())
    w.Header().Set("X-Error-Timestamp", time.Now().UTC().Format(time.RFC3339))
    return nil // Use default response with custom headers
}
```

### 3. Custom Status Codes

Override the default HTTP status codes:

```go
func StatusCodeErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
    var valErr *sebufhttp.ValidationError
    if errors.As(err, &valErr) {
        w.WriteHeader(http.StatusUnprocessableEntity) // 422 instead of 400
    }
    return nil // Framework marshals default error with custom status
}
```

### 4. Custom Response Body

Return a custom protobuf message as the error response:

```go
func CustomBodyErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
    return &models.CustomError{
        Code:      "ERR001",
        Message:   err.Error(),
        RequestId: r.Header.Get("X-Request-ID"),
        Timestamp: time.Now().Unix(),
    }
}
```

### 5. Full Control

Write directly to the response for complete control:

```go
func FullControlErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte(`{"error": "custom format"}`))
    return nil // Framework won't write anything since we already called Write()
}
```

## Error Types

The handler receives the error and can inspect it using `errors.As()`:

```go
// Check for validation errors (HTTP 400)
var valErr *sebufhttp.ValidationError
if errors.As(err, &valErr) {
    // Handle validation error
    for _, v := range valErr.Violations {
        fmt.Printf("Field %s: %s\n", v.Field, v.Description)
    }
}

// Check for handler errors (HTTP 500)
var handlerErr *sebufhttp.Error
if errors.As(err, &handlerErr) {
    // Handle service error
    fmt.Printf("Error: %s\n", handlerErr.Message)
}

// Check for custom error types
var notFound *ErrNotFound
if errors.As(err, &notFound) {
    // Handle not found error
}
```

## Registration

Register your service with a custom error handler:

```go
services.RegisterUserServiceServer(server,
    services.WithMux(mux),
    services.WithErrorHandler(myErrorHandler),
)
```

## Test Commands

```bash
# Create a user (success)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: test-123" \
  -d '{"name": "John Doe", "email": "john@example.com"}'

# Validation error (short name)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "J", "email": "invalid-email"}'

# Not found error
curl -v http://localhost:8080/api/v1/users/non-existent-id

# With request ID header
curl -v http://localhost:8080/api/v1/users/non-existent-id \
  -H "X-Request-ID: my-trace-id-456"
```

## Default Behavior

When no `ErrorHandler` is configured, or when the handler returns `nil`:

- **ValidationError**: Returns HTTP 400 with the ValidationError protobuf message
- **Other errors**: Returns HTTP 500 with the Error protobuf message

The response format (JSON or protobuf) matches the request's Content-Type header.
