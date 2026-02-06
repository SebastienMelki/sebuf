# Phase 6: JSON - Data Encoding - Research

**Researched:** 2026-02-06
**Domain:** Timestamp format options and bytes encoding options across all 4 generators
**Confidence:** HIGH

## Summary

This phase implements two per-field annotations controlling how `google.protobuf.Timestamp` fields and `bytes` fields serialize to JSON. These are the last two "data encoding" annotations before Phase 7 (structural transforms).

**Timestamp formats:** protojson hardcodes `google.protobuf.Timestamp` to RFC 3339 strings (`"2024-01-15T09:30:00Z"`). No protobuf tool offers alternatives. sebuf adds per-field `timestamp_format` annotation supporting RFC3339 (default/protojson), UNIX_SECONDS (integer), UNIX_MILLIS (integer), and DATE (date-only string "2024-01-15"). This requires custom MarshalJSON/UnmarshalJSON because protojson handles Timestamp as a well-known type with no override mechanism.

**Bytes encoding:** protojson hardcodes `bytes` fields to standard base64 (RFC 4648). No protobuf tool offers alternatives. sebuf adds per-field `bytes_encoding` annotation supporting BASE64 (default), BASE64_RAW, BASE64URL, BASE64URL_RAW, and HEX. This requires custom MarshalJSON/UnmarshalJSON because protojson provides no per-field encoding control.

The implementation follows the established Phase 4/5 pattern exactly: proto extension -> shared annotation parsing -> per-generator implementation -> consistency tests. The critical insight is that both annotations target fields that protojson handles with well-known type magic (Timestamp) or built-in encoding (bytes), so the customization must intercept protojson output at the map-level, same as int64 NUMBER encoding.

**Primary recommendation:** Follow the Phase 4/5 pattern. Use protojson for base serialization, parse into `map[string]json.RawMessage`, then replace annotated fields with custom-encoded values. Identical .go files in httpgen and clientgen. Annotation validation must check that `timestamp_format` is only applied to `google.protobuf.Timestamp` fields and `bytes_encoding` only to `bytes` fields.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| protojson | v1.36.x | JSON marshaling baseline | Official proto3 JSON spec implementation; handles Timestamp WKT |
| encoding/json | stdlib | Custom marshal/unmarshal, map manipulation | Same pattern as Phase 4/5 |
| encoding/base64 | stdlib | BASE64, BASE64_RAW, BASE64URL, BASE64URL_RAW encoding | Standard library, all 4 variants built in |
| encoding/hex | stdlib | HEX encoding for bytes | Standard library |
| time | stdlib | Timestamp conversion (AsTime, Unix, Format) | Standard library |
| timestamppb | google.golang.org/protobuf | Timestamp well-known type | Already a transitive dependency |
| protogen | v1.36.x | Code generation framework | Official protoc plugin framework |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| strconv | stdlib | FormatInt for Unix timestamps | Unix seconds/millis as numeric JSON |
| math | stdlib | Precision validation (if needed) | Unix millis conversion |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Post-protojson map patching | Full custom marshal from scratch | Would lose protojson's handling of ALL other fields and well-known types |
| runtime format dispatch | Generation-time format baking | Runtime dispatch adds overhead and complexity; generation-time is consistent with Phase 4/5 |

**Installation:**
No new dependencies required. `timestamppb` is already a transitive dependency of google.golang.org/protobuf.

## Architecture Patterns

### Recommended Proto Extension Structure

```protobuf
// In proto/sebuf/http/annotations.proto (extend existing)

// TimestampFormat controls how google.protobuf.Timestamp fields serialize to JSON
enum TimestampFormat {
  // Follow protojson default (RFC 3339 string: "2024-01-15T09:30:00Z")
  TIMESTAMP_FORMAT_UNSPECIFIED = 0;
  // Explicit RFC 3339 (same as default but documented)
  TIMESTAMP_FORMAT_RFC3339 = 1;
  // Unix seconds as integer: 1705312200
  TIMESTAMP_FORMAT_UNIX_SECONDS = 2;
  // Unix milliseconds as integer: 1705312200000
  TIMESTAMP_FORMAT_UNIX_MILLIS = 3;
  // Date-only string: "2024-01-15"
  TIMESTAMP_FORMAT_DATE = 4;
}

// BytesEncoding controls how bytes fields serialize to JSON
enum BytesEncoding {
  // Follow protojson default (standard base64 with padding)
  BYTES_ENCODING_UNSPECIFIED = 0;
  // Standard base64 with padding (RFC 4648): "SGVsbG8="
  BYTES_ENCODING_BASE64 = 1;
  // Base64 without padding: "SGVsbG8"
  BYTES_ENCODING_BASE64_RAW = 2;
  // URL-safe base64 with padding: "SGVsbG8="
  BYTES_ENCODING_BASE64URL = 3;
  // URL-safe base64 without padding: "SGVsbG8"
  BYTES_ENCODING_BASE64URL_RAW = 4;
  // Hexadecimal encoding: "48656c6c6f"
  BYTES_ENCODING_HEX = 5;
}

extend google.protobuf.FieldOptions {
  // ... existing extensions (unwrap=50009, int64_encoding=50010, enum_encoding=50011,
  //   nullable=50013, empty_behavior=50014) ...

  // Controls timestamp JSON encoding for this field.
  // Valid on: google.protobuf.Timestamp fields only.
  // Default: RFC3339 (protojson default).
  optional TimestampFormat timestamp_format = 50015;

  // Controls bytes JSON encoding for this field.
  // Valid on: bytes fields only.
  // Default: BASE64 (protojson default).
  optional BytesEncoding bytes_encoding = 50016;
}
```

### Recommended Project Structure (additions)

```
internal/annotations/
    timestamp_format.go      # GetTimestampFormat(field), IsTimestampField(field), ValidateTimestampFormatAnnotation(field)
    bytes_encoding.go        # GetBytesEncoding(field), IsBytesField(field), ValidateBytesEncodingAnnotation(field)

internal/httpgen/
    timestamp_format.go      # TimestampFormatContext, generateTimestampFormatEncodingFile()
    bytes_encoding.go        # BytesEncodingContext, generateBytesEncodingFile()

internal/clientgen/
    timestamp_format.go      # Identical to httpgen (server/client consistency)
    bytes_encoding.go        # Identical to httpgen (server/client consistency)
```

### Pattern 1: Timestamp Format MarshalJSON

**What:** Generate MarshalJSON that intercepts protojson output for Timestamp fields and re-encodes them in the specified format.

**When to use:** Any message with a `google.protobuf.Timestamp` field annotated with non-default `timestamp_format`.

**Example (generated code):**
```go
// MarshalJSON implements json.Marshaler for EventWithTimestamps.
// This method handles timestamp_format fields: created_at, event_date
func (x *EventWithTimestamps) MarshalJSON() ([]byte, error) {
    if x == nil {
        return []byte("null"), nil
    }

    // Use protojson for base serialization (handles all other fields correctly)
    data, err := protojson.Marshal(x)
    if err != nil {
        return nil, err
    }

    // Parse into a map to modify timestamp fields
    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, err
    }

    // Handle timestamp_format=UNIX_SECONDS for field: created_at
    if x.CreatedAt != nil {
        t := x.CreatedAt.AsTime()
        raw["createdAt"], _ = json.Marshal(t.Unix())
    }

    // Handle timestamp_format=DATE for field: event_date
    if x.EventDate != nil {
        t := x.EventDate.AsTime()
        raw["eventDate"], _ = json.Marshal(t.Format("2006-01-02"))
    }

    return json.Marshal(raw)
}
```

### Pattern 2: Bytes Encoding MarshalJSON

**What:** Generate MarshalJSON that intercepts protojson output for bytes fields and re-encodes them.

**When to use:** Any message with a `bytes` field annotated with non-default `bytes_encoding`.

**Example (generated code):**
```go
// MarshalJSON implements json.Marshaler for FileWithHashes.
// This method handles bytes_encoding fields: sha256_hash, data_url_safe
func (x *FileWithHashes) MarshalJSON() ([]byte, error) {
    if x == nil {
        return []byte("null"), nil
    }

    // Use protojson for base serialization
    data, err := protojson.Marshal(x)
    if err != nil {
        return nil, err
    }

    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, err
    }

    // Handle bytes_encoding=HEX for field: sha256_hash
    if len(x.Sha256Hash) > 0 {
        raw["sha256Hash"], _ = json.Marshal(hex.EncodeToString(x.Sha256Hash))
    }

    // Handle bytes_encoding=BASE64URL_RAW for field: data_url_safe
    if len(x.DataUrlSafe) > 0 {
        raw["dataUrlSafe"], _ = json.Marshal(base64.RawURLEncoding.EncodeToString(x.DataUrlSafe))
    }

    return json.Marshal(raw)
}
```

### Pattern 3: TypeScript Type Mapping

**What:** Adjust TypeScript types based on timestamp and bytes encoding annotations.

**When to use:** All Timestamp and bytes field type generation in ts-client.

**Example:**
```typescript
// Timestamp with UNIX_SECONDS -> number
// Timestamp with UNIX_MILLIS -> number
// Timestamp with RFC3339 -> string (default)
// Timestamp with DATE -> string
export interface EventWithTimestamps {
  defaultTimestamp?: string;      // RFC3339 (default, Timestamp is message = optional)
  unixCreatedAt?: number;         // UNIX_SECONDS
  unixMillisCreatedAt?: number;   // UNIX_MILLIS
  eventDate?: string;             // DATE

  // bytes with HEX -> string (always string, just different encoding)
  sha256Hash: string;             // HEX
  dataUrlSafe: string;            // BASE64URL_RAW
}
```

### Pattern 4: OpenAPI Schema for Timestamp/Bytes Formats

**What:** Generate OpenAPI schemas that document the actual format used.

**When to use:** All timestamp and bytes fields in OpenAPI schema generation.

**Example:**
```yaml
# Timestamp with UNIX_SECONDS
createdAt:
  type: integer
  format: unix-timestamp
  description: "Unix timestamp in seconds"

# Timestamp with UNIX_MILLIS
createdAtMs:
  type: integer
  format: unix-timestamp-ms
  description: "Unix timestamp in milliseconds"

# Timestamp with DATE
eventDate:
  type: string
  format: date

# Timestamp with RFC3339 (default)
updatedAt:
  type: string
  format: date-time

# bytes with HEX
sha256Hash:
  type: string
  format: hex
  pattern: "^[0-9a-fA-F]*$"

# bytes with BASE64URL_RAW
token:
  type: string
  format: base64url
```

### Anti-Patterns to Avoid

- **timestamp_format on non-Timestamp fields:** Must validate at generation time. A `string` field with timestamp_format is invalid.
- **bytes_encoding on non-bytes fields:** Must validate at generation time. An `int32` field with bytes_encoding is invalid.
- **Mixing custom MarshalJSON with protojson for Timestamp:** protojson has special WKT handling that produces RFC 3339. Custom format MUST replace the protojson output, not supplement it.
- **Nanos truncation without warning:** DATE format drops time AND nanos. UNIX_SECONDS drops nanos. Document this.
- **Lowercase hex vs uppercase hex:** Use lowercase hex consistently (Go's hex.EncodeToString uses lowercase). Document this.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Timestamp to Unix seconds | Manual second extraction | `timestamppb.Timestamp.AsTime().Unix()` | Handles nanos correctly |
| Timestamp to Unix millis | Manual calculation | `ts.AsTime().UnixMilli()` | Standard Go method (Go 1.17+) |
| Timestamp to date string | Custom date formatting | `ts.AsTime().Format("2006-01-02")` | Standard Go date formatting |
| Base64 standard encoding | Custom encoder | `base64.StdEncoding.EncodeToString()` | Handles padding correctly |
| Base64 raw encoding | Strip padding manually | `base64.RawStdEncoding.EncodeToString()` | Standard library variant |
| Base64 URL-safe encoding | Replace +/ manually | `base64.URLEncoding.EncodeToString()` | Standard library variant |
| Base64 URL-safe raw | Combine above | `base64.RawURLEncoding.EncodeToString()` | Standard library variant |
| Hex encoding | fmt.Sprintf("%x") | `hex.EncodeToString()` | Handles even-length padding |
| Timestamp field detection | String comparison | `field.Message.Desc.FullName() == "google.protobuf.Timestamp"` | Canonical WKT detection |

**Key insight:** Go's `encoding/base64` package has exactly 4 built-in encodings that map 1:1 to our 4 base64 variants: `StdEncoding`, `RawStdEncoding`, `URLEncoding`, `RawURLEncoding`. No custom encoding logic needed.

## Common Pitfalls

### Pitfall 1: Timestamp is a Message Type, Not a Scalar

**What goes wrong:** Treating `google.protobuf.Timestamp` like a scalar in annotation validation or type detection. It is `protoreflect.MessageKind`, not a scalar kind.

**Why it happens:** All previous annotations (int64_encoding, nullable, etc.) target scalar fields or generic messages. Timestamp is a specific message type that must be detected by its full name.

**How to avoid:** Use `field.Desc.Kind() == protoreflect.MessageKind && field.Message.Desc.FullName() == "google.protobuf.Timestamp"` for detection. The annotation validation must check this exact condition.

**Warning signs:** timestamp_format annotation silently ignored on Timestamp fields because the validation only checks scalar kinds.

### Pitfall 2: protojson WKT Handling Produces RFC 3339 String, Not Object

**What goes wrong:** Developer expects protojson to produce `{"seconds":1234,"nanos":0}` but protojson produces `"2009-02-13T23:31:30Z"`.

**Why it happens:** protojson has special well-known type handling for Timestamp. It serializes as an RFC 3339 string, not as the message structure.

**How to avoid:** The MarshalJSON code replaces the RFC 3339 string in the JSON map, not a nested object. The code pattern is: `raw["fieldName"]` contains `"\"2009-02-13T23:31:30Z\""` (a JSON string), and we replace it with `1234567890` (a JSON number for UNIX_SECONDS) or `"\"2009-02-13\""` (a JSON string for DATE).

**Warning signs:** MarshalJSON output contains `{"seconds":1234}` instead of a formatted string/number.

### Pitfall 3: DATE Format is Lossy

**What goes wrong:** Round-tripping through DATE format loses time information. A Timestamp of `2024-01-15T14:30:00Z` becomes `"2024-01-15"` which round-trips to `2024-01-15T00:00:00Z`.

**Why it happens:** DATE format intentionally drops time component.

**How to avoid:** Document that DATE is lossy. For UnmarshalJSON, parse `"2024-01-15"` as midnight UTC. This is acceptable and intentional for date-only fields (e.g., birthday, event date).

**Warning signs:** Tests expecting exact Timestamp round-trip with DATE format.

### Pitfall 4: UNIX_SECONDS Drops Nanos

**What goes wrong:** A Timestamp with nanos (e.g., 500000000 nanos = 0.5 seconds) loses the sub-second precision when serialized as UNIX_SECONDS.

**Why it happens:** `time.Time.Unix()` returns integer seconds only.

**How to avoid:** Document that UNIX_SECONDS truncates nanos. For UNIX_MILLIS, use `time.Time.UnixMilli()` which preserves millisecond precision (but truncates sub-millisecond nanos). Accept this as intentional behavior.

**Warning signs:** Precision loss in cross-generator tests.

### Pitfall 5: Bytes Zero Value Handling

**What goes wrong:** An empty `bytes` field (nil/empty slice) is omitted by protojson. With custom encoding, the field might appear as `""` or be inconsistently handled.

**Why it happens:** protojson omits zero-value fields. An empty bytes field has no base64/hex representation in the protojson output map.

**How to avoid:** Follow the same pattern as int64 encoding: only replace the field in the map if the Go field is non-empty (`len(x.Field) > 0`). If empty, protojson already omits it, so no action needed.

**Warning signs:** Empty bytes fields appearing as empty strings in JSON.

### Pitfall 6: UnmarshalJSON for Timestamp Formats

**What goes wrong:** Server receives `1705312200` (UNIX_SECONDS) but protojson expects `"2024-01-15T09:30:00Z"`.

**Why it happens:** protojson only accepts RFC 3339 strings for Timestamp fields. Custom format data must be converted back before passing to protojson.

**How to avoid:** In UnmarshalJSON, detect the format (number = unix seconds/millis, date string = DATE), convert to RFC 3339, then pass to protojson. For UNIX_SECONDS: parse number, create `time.Unix(n, 0)`, format as RFC 3339, replace in map.

**Warning signs:** 400 errors when sending UNIX timestamp format to the server.

### Pitfall 7: UnmarshalJSON for Bytes Encoding

**What goes wrong:** Server receives a hex string but protojson expects base64.

**Why it happens:** protojson only accepts standard base64 for bytes fields.

**How to avoid:** In UnmarshalJSON, decode the custom encoding (hex, base64url, etc.), re-encode as standard base64, replace in the map, then pass to protojson.

**Warning signs:** Invalid bytes data after deserialization.

## Code Examples

Verified patterns from codebase analysis and official documentation:

### Annotation Parsing (following established Phase 4/5 pattern)

```go
// Source: Pattern from internal/annotations/int64_encoding.go
package annotations

import (
    "google.golang.org/protobuf/compiler/protogen"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/reflect/protoreflect"
    "google.golang.org/protobuf/types/descriptorpb"

    "github.com/SebastienMelki/sebuf/http"
)

// TimestampFormatValidationError represents an error in timestamp_format annotation validation.
type TimestampFormatValidationError struct {
    MessageName string
    FieldName   string
    Reason      string
}

func (e *TimestampFormatValidationError) Error() string {
    return "invalid timestamp_format annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// GetTimestampFormat returns the timestamp format for a field.
// Returns TIMESTAMP_FORMAT_UNSPECIFIED if not set (callers should use protojson default: RFC3339).
func GetTimestampFormat(field *protogen.Field) http.TimestampFormat {
    options := field.Desc.Options()
    if options == nil {
        return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
    }

    fieldOptions, ok := options.(*descriptorpb.FieldOptions)
    if !ok {
        return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
    }

    ext := proto.GetExtension(fieldOptions, http.E_TimestampFormat)
    if ext == nil {
        return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
    }

    format, ok := ext.(http.TimestampFormat)
    if !ok {
        return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
    }

    return format
}

// HasTimestampFormatAnnotation returns true if the field has any non-default timestamp_format.
func HasTimestampFormatAnnotation(field *protogen.Field) bool {
    format := GetTimestampFormat(field)
    return format != http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED &&
        format != http.TimestampFormat_TIMESTAMP_FORMAT_RFC3339
}

// IsTimestampField returns true if the field is google.protobuf.Timestamp.
func IsTimestampField(field *protogen.Field) bool {
    return field.Desc.Kind() == protoreflect.MessageKind &&
        field.Message != nil &&
        field.Message.Desc.FullName() == "google.protobuf.Timestamp"
}

// ValidateTimestampFormatAnnotation checks if timestamp_format is valid for a field.
func ValidateTimestampFormatAnnotation(field *protogen.Field, messageName string) error {
    format := GetTimestampFormat(field)
    if format == http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED {
        return nil // No annotation, nothing to validate
    }

    if !IsTimestampField(field) {
        return &TimestampFormatValidationError{
            MessageName: messageName,
            FieldName:   string(field.Desc.Name()),
            Reason:      "timestamp_format annotation is only valid on google.protobuf.Timestamp fields",
        }
    }

    return nil
}
```

### Bytes Encoding Annotation Parsing

```go
// Source: Pattern from internal/annotations/empty_behavior.go
package annotations

// BytesEncodingValidationError represents an error in bytes_encoding annotation validation.
type BytesEncodingValidationError struct {
    MessageName string
    FieldName   string
    Reason      string
}

func (e *BytesEncodingValidationError) Error() string {
    return "invalid bytes_encoding annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// GetBytesEncoding returns the bytes encoding for a field.
func GetBytesEncoding(field *protogen.Field) http.BytesEncoding {
    // Same pattern as GetInt64Encoding, GetEmptyBehavior, etc.
    // ...
}

// HasBytesEncodingAnnotation returns true if the field has any non-default bytes_encoding.
func HasBytesEncodingAnnotation(field *protogen.Field) bool {
    encoding := GetBytesEncoding(field)
    return encoding != http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED &&
        encoding != http.BytesEncoding_BYTES_ENCODING_BASE64
}

// ValidateBytesEncodingAnnotation checks if bytes_encoding is valid for a field.
func ValidateBytesEncodingAnnotation(field *protogen.Field, messageName string) error {
    encoding := GetBytesEncoding(field)
    if encoding == http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED {
        return nil
    }

    if field.Desc.Kind() != protoreflect.BytesKind {
        return &BytesEncodingValidationError{
            MessageName: messageName,
            FieldName:   string(field.Desc.Name()),
            Reason:      "bytes_encoding annotation is only valid on bytes fields",
        }
    }

    return nil
}
```

### Go Encoding Functions (base64/hex mapping)

```go
// Source: Go standard library encoding packages
import (
    "encoding/base64"
    "encoding/hex"
)

// Go base64 package provides exactly our 4 base64 variants:
// BASE64       -> base64.StdEncoding.EncodeToString(data)
// BASE64_RAW   -> base64.RawStdEncoding.EncodeToString(data)
// BASE64URL    -> base64.URLEncoding.EncodeToString(data)
// BASE64URL_RAW -> base64.RawURLEncoding.EncodeToString(data)

// HEX -> hex.EncodeToString(data)  (lowercase)

// Corresponding decode:
// BASE64       -> base64.StdEncoding.DecodeString(s)
// BASE64_RAW   -> base64.RawStdEncoding.DecodeString(s)
// BASE64URL    -> base64.URLEncoding.DecodeString(s)
// BASE64URL_RAW -> base64.RawURLEncoding.DecodeString(s)
// HEX          -> hex.DecodeString(s)
```

### Timestamp Conversion Functions

```go
// Source: google.golang.org/protobuf/types/known/timestamppb
import "google.golang.org/protobuf/types/known/timestamppb"

ts := &timestamppb.Timestamp{Seconds: 1705312200, Nanos: 0}
t := ts.AsTime() // time.Time in UTC

// RFC3339 (protojson default, no action needed)
rfc3339 := t.Format(time.RFC3339Nano) // "2024-01-15T09:30:00Z"

// UNIX_SECONDS
unixSec := t.Unix() // int64: 1705312200

// UNIX_MILLIS
unixMs := t.UnixMilli() // int64: 1705312200000

// DATE
dateStr := t.Format("2006-01-02") // "2024-01-15"

// UnmarshalJSON reverse:
// UNIX_SECONDS -> time.Unix(n, 0) -> timestamppb.New(t)
// UNIX_MILLIS -> time.UnixMilli(n) -> timestamppb.New(t)
// DATE -> time.Parse("2006-01-02", s) -> timestamppb.New(t)
```

### TypeScript Timestamp Type Mapping

```typescript
// Source: Pattern from internal/tsclientgen/types.go analysis

// Current: google.protobuf.Timestamp -> string (via message name "Timestamp")
// Note: Timestamp is currently treated as a regular message reference

// With timestamp_format annotation:
// RFC3339 -> string (date-time string)
// UNIX_SECONDS -> number (integer)
// UNIX_MILLIS -> number (integer)
// DATE -> string (date string)
```

### OpenAPI Timestamp Schema Mapping

```yaml
# RFC3339 (default):
field:
  type: string
  format: date-time

# UNIX_SECONDS:
field:
  type: integer
  format: unix-timestamp
  description: "Unix timestamp in seconds"

# UNIX_MILLIS:
field:
  type: integer
  format: unix-timestamp-ms
  description: "Unix timestamp in milliseconds"

# DATE:
field:
  type: string
  format: date
```

### OpenAPI Bytes Schema Mapping

```yaml
# BASE64 (default):
field:
  type: string
  format: byte

# BASE64_RAW:
field:
  type: string
  format: byte
  description: "Base64 encoded without padding"

# BASE64URL:
field:
  type: string
  format: byte
  description: "URL-safe base64 encoded"

# BASE64URL_RAW:
field:
  type: string
  format: byte
  description: "URL-safe base64 encoded without padding"

# HEX:
field:
  type: string
  format: hex
  pattern: "^[0-9a-fA-F]*$"
```

## Implementation Strategy by Generator

### go-http (HTTP Server)

**Touch points:**
- `proto/sebuf/http/annotations.proto`: Add TimestampFormat, BytesEncoding enums and extension fields 50015-50016
- `internal/annotations/timestamp_format.go`: GetTimestampFormat, IsTimestampField, ValidateTimestampFormatAnnotation
- `internal/annotations/bytes_encoding.go`: GetBytesEncoding, ValidateBytesEncodingAnnotation
- `internal/httpgen/timestamp_format.go`: TimestampFormatContext, generateTimestampFormatEncodingFile
- `internal/httpgen/bytes_encoding.go`: BytesEncodingContext, generateBytesEncodingFile
- `internal/httpgen/generator.go`: Call generateTimestampFormatEncodingFile and generateBytesEncodingFile in generateFile()

**Key considerations:**
- Timestamp fields are `protoreflect.MessageKind`, not scalars. Detection uses FullName comparison.
- protojson emits Timestamp as RFC 3339 string in the JSON map. Custom format replaces that string/number.
- Generated file naming: `*_timestamp_format.pb.go`, `*_bytes_encoding.pb.go`
- Imports needed for timestamp: `encoding/json`, `time`, `google.golang.org/protobuf/encoding/protojson`, `google.golang.org/protobuf/types/known/timestamppb`
- Imports needed for bytes: `encoding/json`, `encoding/base64`, `encoding/hex`, `google.golang.org/protobuf/encoding/protojson`

### go-client (HTTP Client)

**Touch points:**
- `internal/clientgen/timestamp_format.go`: Identical to httpgen
- `internal/clientgen/bytes_encoding.go`: Identical to httpgen
- `internal/clientgen/generator.go`: Call generateTimestampFormatEncodingFile and generateBytesEncodingFile

**Key consideration:** Identical implementation to go-http guarantees server/client JSON match (D-04-02-03, D-05-02-01).

### ts-client (TypeScript Client)

**Touch points:**
- `internal/tsclientgen/types.go`: Add Timestamp field detection and type mapping based on format. Add bytes encoding awareness (all variants map to `string` except UNIX_SECONDS/UNIX_MILLIS which map to `number` for Timestamp).

**Key considerations:**
- Timestamp with UNIX_SECONDS or UNIX_MILLIS -> TypeScript `number` type
- Timestamp with RFC3339 or DATE -> TypeScript `string` type
- Bytes with any encoding -> TypeScript `string` type (all are string representations)
- Must detect `google.protobuf.Timestamp` message by full name in type mapping
- Current `tsFieldType` for MessageKind returns `string(field.Message.Desc.Name())` which would return "Timestamp" -- need to intercept before generic message handling

### openapiv3 (OpenAPI Spec)

**Touch points:**
- `internal/openapiv3/types.go`: In `convertScalarField`, before the generic MessageKind handling, detect Timestamp fields and apply format-specific schema. In the BytesKind case, check for custom encoding.

**Key considerations:**
- Timestamp with UNIX_SECONDS: `type: integer, format: unix-timestamp`
- Timestamp with UNIX_MILLIS: `type: integer, format: unix-timestamp-ms`
- Timestamp with DATE: `type: string, format: date`
- Timestamp default (RFC3339): `type: string, format: date-time`
- Bytes with HEX: `type: string, format: hex, pattern: "^[0-9a-fA-F]*$"`
- Bytes default (BASE64): `type: string, format: byte` (existing behavior)
- Must intercept before the generic `$ref: '#/components/schemas/Timestamp'` handling

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| protojson hardcoded RFC 3339 | Per-field timestamp_format | This phase | Enables Unix timestamps, date-only for REST APIs |
| protojson hardcoded base64 | Per-field bytes_encoding | This phase | Enables hex for crypto/blockchain, base64url for URLs |
| Timestamp -> $ref: Timestamp schema | Timestamp -> inline format-specific schema | This phase | OpenAPI documents actual wire format |

**Deprecated/outdated:**
- No alternative approaches existed. These are genuinely new capabilities.

## Open Questions

Things that couldn't be fully resolved:

1. **MarshalJSON conflict with other annotations on same message**
   - What we know: A message might have int64 NUMBER fields AND timestamp format fields AND bytes encoding fields AND nullable fields AND empty_behavior fields. Each generates its own `*_X.pb.go` file with MarshalJSON.
   - What's unclear: Can a message have MarshalJSON from multiple files? No -- Go only allows one MarshalJSON per type.
   - Recommendation: If a message has annotations from multiple features, only ONE file should contain MarshalJSON/UnmarshalJSON. This is the same issue Phases 4/5 face. The established pattern seems to be that each feature generates its own file, which means each message can only use ONE feature's MarshalJSON. This is a pre-existing limitation. For Phase 6, follow the same pattern. If a message needs both timestamp_format and int64_encoding, the planner should determine whether to combine them in a unified encoding file or document the limitation. **This is the most important open question.**
   - Mitigation: In practice, messages typically use one or two encoding features. The golden file test infrastructure would catch any Go compilation error from duplicate MarshalJSON methods. If it becomes an issue, a unified `_custom_json.pb.go` file could combine all features.

2. **Timestamp nanos precision for UNIX_MILLIS**
   - What we know: `time.Time.UnixMilli()` returns int64. Sub-millisecond nanos are truncated.
   - What's unclear: Should we emit a warning for nanos precision loss like int64 NUMBER encoding?
   - Recommendation: No warning needed. UNIX_MILLIS implies millisecond precision. Sub-millisecond nanos are rare in practice.

3. **Timestamp field detection in TypeScript**
   - What we know: Current tsclientgen returns `string(field.Message.Desc.Name())` for message types, which would give "Timestamp".
   - What's unclear: How Timestamp is currently handled in TypeScript types -- is there an existing Timestamp interface?
   - Recommendation: Check if `Timestamp` gets collected in the messageSet. If so, the type would be `Timestamp` (an interface with `seconds` and `nanos`). With the annotation, it should be inlined as `string` or `number` instead of referencing the Timestamp interface. This needs verification during implementation.

## Sources

### Primary (HIGH confidence)

- Codebase: `internal/annotations/*.go` - Established annotation parsing patterns (verified)
- Codebase: `internal/httpgen/encoding.go` - Phase 4 MarshalJSON map-patching pattern (verified)
- Codebase: `internal/httpgen/nullable.go` - Phase 5 MarshalJSON pattern (verified)
- Codebase: `internal/httpgen/empty_behavior.go` - Phase 5 empty behavior pattern (verified)
- Codebase: `internal/openapiv3/types.go` - Current BytesKind handling (format: byte) (verified)
- Codebase: `internal/tsclientgen/types.go` - Current type mapping (verified)
- Codebase: `proto/sebuf/http/annotations.proto` - Current extensions up to 50014 (verified)
- [Go encoding/base64](https://pkg.go.dev/encoding/base64) - StdEncoding, RawStdEncoding, URLEncoding, RawURLEncoding
- [Go encoding/hex](https://pkg.go.dev/encoding/hex) - EncodeToString, DecodeString
- [Go timestamppb](https://pkg.go.dev/google.golang.org/protobuf/types/known/timestamppb) - AsTime, Unix, UnixMilli
- [Proto3 JSON Mapping](https://protobuf.dev/programming-guides/json/) - Timestamp as RFC 3339, bytes as base64

### Secondary (MEDIUM confidence)

- `.planning/research/FEATURES.md` #92 - Timestamp format ecosystem analysis
- `.planning/research/FEATURES.md` #95 - Bytes encoding ecosystem analysis
- `.planning/research/PITFALLS.md` #7 - Well-known type special casing pitfall
- Phase 4 RESEARCH.md - Established patterns for this type of feature

### Tertiary (LOW confidence)

- None. All findings verified against codebase and official documentation.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - uses existing stdlib packages (encoding/base64, encoding/hex, time), no new dependencies
- Architecture: HIGH - follows established Phase 4/5 patterns exactly (identical file structure, annotation parsing, map-patching MarshalJSON)
- Pitfalls: HIGH - Timestamp WKT behavior verified against protojson documentation; bytes encoding variants are well-understood stdlib functions
- Open question (MarshalJSON conflict): MEDIUM - this is a pre-existing architectural concern from Phase 4, not new to Phase 6

**Research date:** 2026-02-06
**Valid until:** 90 days (stable domain, proto3 JSON spec is mature, Go stdlib encoding packages are stable)
