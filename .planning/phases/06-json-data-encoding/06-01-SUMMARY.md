---
phase: 06-json-data-encoding
plan: 01
subsystem: annotations
tags: [protobuf, annotations, timestamp, bytes, encoding]
dependency-graph:
  requires: [05-01]
  provides: [timestamp_format annotation, bytes_encoding annotation, shared parsing functions]
  affects: [06-02, 06-03, 06-04]
tech-stack:
  added: []
  patterns: [annotation-per-file, GetXxx/HasXxx/ValidateXxx convention]
key-files:
  created:
    - internal/annotations/timestamp_format.go
    - internal/annotations/bytes_encoding.go
  modified:
    - proto/sebuf/http/annotations.proto
    - http/annotations.pb.go
    - http/errors.pb.go
    - http/headers.pb.go
    - internal/annotations/annotations_test.go
decisions:
  - id: D-06-01-01
    title: Extension numbers 50015-50016 continue sequence from existing 50014 (empty_behavior)
  - id: D-06-01-02
    title: UNSPECIFIED (0) always means protojson default -- RFC3339 for timestamps, BASE64 for bytes
  - id: D-06-01-03
    title: HasTimestampFormatAnnotation excludes both UNSPECIFIED and RFC3339 (both produce default behavior)
  - id: D-06-01-04
    title: HasBytesEncodingAnnotation excludes both UNSPECIFIED and BASE64 (both produce default behavior)
metrics:
  duration: 3m25s
  completed: 2026-02-06
---

# Phase 6 Plan 1: Timestamp Format and Bytes Encoding Annotations Summary

**One-liner:** Proto annotations and shared parsing for timestamp_format (RFC3339/Unix/Date) and bytes_encoding (base64/hex variants) with validation.

## What Was Done

### Task 1: Proto Annotations (c143646)

Added two new enums and field extensions to `proto/sebuf/http/annotations.proto`:

**TimestampFormat enum** (5 values):
- `TIMESTAMP_FORMAT_UNSPECIFIED` (0) -- protojson default (RFC 3339)
- `TIMESTAMP_FORMAT_RFC3339` (1) -- explicit RFC 3339, self-documenting
- `TIMESTAMP_FORMAT_UNIX_SECONDS` (2) -- integer seconds
- `TIMESTAMP_FORMAT_UNIX_MILLIS` (3) -- integer milliseconds
- `TIMESTAMP_FORMAT_DATE` (4) -- date-only string "2024-01-15"

**BytesEncoding enum** (6 values):
- `BYTES_ENCODING_UNSPECIFIED` (0) -- protojson default (standard base64)
- `BYTES_ENCODING_BASE64` (1) -- explicit standard base64 with padding
- `BYTES_ENCODING_BASE64_RAW` (2) -- base64 without padding
- `BYTES_ENCODING_BASE64URL` (3) -- URL-safe base64 with padding
- `BYTES_ENCODING_BASE64URL_RAW` (4) -- URL-safe base64 without padding
- `BYTES_ENCODING_HEX` (5) -- hexadecimal lowercase

**Field extensions:**
- `timestamp_format` (50015) -- valid on google.protobuf.Timestamp fields only
- `bytes_encoding` (50016) -- valid on bytes fields only

### Task 2: Shared Annotation Parsing (dc11725)

Created two Go files following the established annotation-per-file pattern:

**timestamp_format.go** exports:
- `GetTimestampFormat(field)` -- returns TimestampFormat enum value
- `HasTimestampFormatAnnotation(field)` -- true for non-default, non-RFC3339 formats
- `IsTimestampField(field)` -- detects google.protobuf.Timestamp by FullName
- `ValidateTimestampFormatAnnotation(field, messageName)` -- rejects non-Timestamp fields
- `TimestampFormatValidationError` -- structured error type

**bytes_encoding.go** exports:
- `GetBytesEncoding(field)` -- returns BytesEncoding enum value
- `HasBytesEncodingAnnotation(field)` -- true for non-default, non-BASE64 encodings
- `ValidateBytesEncodingAnnotation(field, messageName)` -- rejects non-bytes fields
- `BytesEncodingValidationError` -- structured error type

**Tests added** to `annotations_test.go`:
- Extension descriptor tests (numbers 50015, 50016)
- Enum value tests (all 5 TimestampFormat + all 6 BytesEncoding values)
- String representation tests for debugging
- Validation error message format tests

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| D-06-01-01 | Extension numbers 50015-50016 | Continue sequential numbering from 50014 (empty_behavior) |
| D-06-01-02 | UNSPECIFIED = protojson default | Consistent with all prior annotations (int64, enum, empty_behavior) |
| D-06-01-03 | Has* excludes RFC3339/BASE64 | Both RFC3339 and UNSPECIFIED produce identical protojson behavior; same for BASE64 |
| D-06-01-04 | Has* excludes BASE64 default | Parallel to HasTimestampFormatAnnotation excluding RFC3339 |

## Deviations from Plan

None -- plan executed exactly as written.

## Verification Results

| Check | Result |
|-------|--------|
| Proto compiles (`buf build`) | PASS |
| Go packages compile | PASS |
| Annotation tests pass | PASS (all new tests green) |
| Lint clean (`make lint-fix`) | PASS (0 issues) |
| Extension numbers unique (50015, 50016) | PASS |
| Full test suite (`go test ./...`) | PASS (zero regression) |

## Next Phase Readiness

Plans 06-02 through 06-04 can now use the shared annotation functions:
- `GetTimestampFormat()` and `GetBytesEncoding()` for encoding decisions
- `HasTimestampFormatAnnotation()` and `HasBytesEncodingAnnotation()` for detection
- `IsTimestampField()` for type checking
- `ValidateTimestampFormatAnnotation()` and `ValidateBytesEncodingAnnotation()` for generation-time validation

No blockers identified.
