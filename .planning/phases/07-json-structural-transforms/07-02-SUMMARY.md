---
phase: 07-json-structural-transforms
plan: 02
subsystem: api
tags: [oneof, discriminator, protobuf, json, codegen, marshal, openapi, typescript]

# Dependency graph
requires:
  - phase: 07-01
    provides: "Oneof discriminator annotation parsing (GetOneofConfig, GetOneofVariantValue, GetOneofDiscriminatorInfo, ValidateOneofDiscriminator)"
provides:
  - "Oneof discriminator MarshalJSON/UnmarshalJSON generation in go-http and go-client"
  - "TypeScript discriminated union type generation"
  - "OpenAPI oneOf + discriminator keyword schema representation"
  - "Golden test coverage for oneof discriminator across all 4 generators"
affects: [07-04, phase-08, phase-09, phase-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Oneof MarshalJSON: protojson base -> delete variant keys -> add discriminator + variant fields (flattened) or just discriminator (nested)"
    - "Oneof UnmarshalJSON: read discriminator -> route to variant -> protojson for base fields"
    - "OpenAPI oneOf + discriminator keyword with mapping for discriminated unions"
    - "TypeScript discriminated union types: export type FooContent = { type: 'text'; body: string } | { type: 'img'; url: string }"

key-files:
  created:
    - internal/httpgen/oneof_discriminator.go
    - internal/clientgen/oneof_discriminator.go
    - internal/httpgen/testdata/proto/oneof_discriminator.proto
    - internal/httpgen/testdata/golden/oneof_discriminator_*.pb.go
    - internal/clientgen/testdata/golden/oneof_discriminator_*.pb.go
    - internal/tsclientgen/testdata/golden/oneof_discriminator_client.ts
    - internal/openapiv3/testdata/golden/*/OneofDiscriminatorService.openapi.*
  modified:
    - internal/httpgen/generator.go
    - internal/clientgen/generator.go
    - internal/tsclientgen/types.go
    - internal/openapiv3/generator.go
    - internal/*/golden_test.go

key-decisions:
  - "D-07-02-01: Use json.Marshal for variant messages in oneof MarshalJSON for annotation composability (variant's own MarshalJSON invoked)"
  - "D-07-02-02: Flattened mode promotes variant fields alongside discriminator; non-flattened keeps variant nested under field name"
  - "D-07-02-03: OpenAPI uses per-variant schemas with oneOf+discriminator for flattened, object schema with discriminator property for nested"
  - "D-07-02-04: nolint:gocognit on openapiv3 functions refactored into helpers (buildFlattenedVariantSchemas, buildNestedOneofVariants, etc.)"

patterns-established:
  - "Oneof MarshalJSON/UnmarshalJSON: protojson-then-manipulate-raw-JSON with json.Marshal for child composability"
  - "OpenAPI discriminator: oneOf keyword with mapping for flattened, discriminator property + oneOf for nested"
  - "TypeScript discriminated unions: type alias with | branches, each branch has literal discriminator + variant fields"

# Metrics
duration: ~25min
completed: 2026-02-06
---

# Phase 7 Plan 2: Oneof Discriminated Unions Summary

**MarshalJSON/UnmarshalJSON oneof discriminator generation across go-http/go-client, TypeScript discriminated union types, and OpenAPI oneOf+discriminator schemas with full golden test coverage**

## Performance

- **Duration:** ~25 min (interrupted by rate limits, completed across sessions)
- **Tasks:** 2
- **Files modified:** 18+

## Accomplishments
- Go generators (httpgen + clientgen) produce custom MarshalJSON/UnmarshalJSON for messages with discriminated oneofs (flattened and non-flattened modes)
- TypeScript generator creates discriminated union types with literal discriminator values
- OpenAPI generator uses oneOf + discriminator keyword with mapping for accurate schema representation
- Golden tests created and verified for all 4 generators covering FlattenedEvent (flat + custom value), NestedEvent (nested + custom value), and PlainEvent (backward compat)
- Annotation composability: json.Marshal for variant messages ensures variant's own encoding annotations are respected

## Task Commits

1. **Task 1: Implement oneof discriminator in Go generators** - `485be6e` (feat)
2. **Task 2: Complete golden files, lint fixes, TS/OpenAPI verification** - `7708d25` (feat)

## Files Created/Modified
- `internal/httpgen/oneof_discriminator.go` - OneofDiscriminatorContext, MarshalJSON/UnmarshalJSON generation for go-http
- `internal/clientgen/oneof_discriminator.go` - Identical oneof encoding for go-client
- `internal/tsclientgen/types.go` - generateOneofDiscriminatedUnionType, generateFlattenedOneofInterface for TS
- `internal/openapiv3/generator.go` - buildFlattenedOneofSchema, buildNestedOneofSchema with helper extraction
- `internal/httpgen/testdata/proto/oneof_discriminator.proto` - Test proto with FlattenedEvent, NestedEvent, PlainEvent
- `internal/*/testdata/proto/oneof_discriminator.proto` - Symlinks to httpgen source
- `internal/*/testdata/golden/oneof_discriminator_*` - Golden files for all 4 generators

## Decisions Made
- D-07-02-01: json.Marshal for variant messages enables annotation composability
- D-07-02-02: Flattened mode promotes variant fields to parent; nested keeps variant under field name
- D-07-02-03: OpenAPI per-variant schemas for flattened, object+discriminator for nested
- D-07-02-04: Extracted buildFlattenedVariantSchemas, buildFlattenedDiscriminator, buildNestedOneofVariants, buildNestedDiscriminator to satisfy gocognit limits

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Lint: gocritic if-else-chain in tsclientgen**
- Converted if-else chain to switch statement in generateOneofDiscriminatedUnionType
- Committed in: 7708d25

**2. [Rule 3 - Blocking] Lint: gocognit in openapiv3 generator**
- Extracted 4 helper functions from buildFlattenedOneofSchema and buildNestedOneofSchema
- Committed in: 7708d25

## Issues Encountered
- Rate limit interruption during Task 2 execution -- resumed in new session
- Parallel agent (07-03: flatten) worked on same branch, handled by atomic commits of feature-specific files

## Next Phase Readiness
- Oneof discriminator implementation complete across all 4 generators
- Golden tests verify correct output for all discriminator scenarios
- Plan 07-04 (cross-generator consistency tests) can validate oneof consistency

---
*Phase: 07-json-structural-transforms*
*Completed: 2026-02-06*
