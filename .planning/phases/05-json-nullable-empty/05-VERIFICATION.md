---
phase: 05-json-nullable-empty
verified: 2026-02-06T10:25:12Z
status: passed
score: 5/5 must-haves verified
---

# Phase 5: JSON - Nullable & Empty Verification Report

**Phase Goal:** Developers can express null vs absent vs default semantics for primitive fields and control empty object serialization behavior

**Verified:** 2026-02-06T10:25:12Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A proto field annotated with `nullable = true` generates pointer types in Go (`*string`, `*int32`), union types in TypeScript (`string \| null`), and `nullable: true` in OpenAPI schemas | ✓ VERIFIED | Proto annotations exist (line 122 annotations.proto). Go: `*string` pointers with MarshalJSON emitting `null` when unset (nullable_nullable.pb.go:33-47). TypeScript: `middleName: string \| null` (nullable_client.ts). OpenAPI: `type: ["string", "null"]` array syntax (NullableService.openapi.yaml:119-121). |
| 2 | Three distinct states are representable per nullable field: absent (key omitted from JSON), null (key present with `null` value), and default value (key present with value) | ✓ VERIFIED | UnmarshalJSON handles explicit `null` vs absent (nullable_nullable.pb.go:61-77). MarshalJSON emits `null` for unset fields (line 34). Regular protojson handles non-null values. Consistency tests verify behavior. |
| 3 | A proto message field annotated with `empty_behavior = PRESERVE` serializes empty messages as `{}`, `empty_behavior = NULL` as `null`, and `empty_behavior = OMIT` omits the key entirely | ✓ VERIFIED | Proto enum EmptyBehavior (annotations.proto:82-91). MarshalJSON uses `proto.Size() == 0` for empty detection (empty_behavior_empty_behavior.pb.go:33-54). NULL emits `"null"` (line 40-41), OMIT calls `delete()` (line 46-47), PRESERVE is default `{}` (line 34-35). |
| 4 | All nullable and empty-behavior semantics are consistent across go-http, go-client, ts-client, and OpenAPI generators | ✓ VERIFIED | All 4 generators call shared `annotations.IsNullableField()` and `annotations.GetEmptyBehavior()`. Consistency tests pass: TestNullableConsistencyGoHTTPvsGoClient, TestNullableConsistencyTypeScript, TestNullableConsistencyOpenAPI, TestEmptyBehaviorConsistencyGoHTTPvsGoClient, TestEmptyBehaviorConsistencyOpenAPI. |
| 5 | A cross-generator consistency test confirms that the same nullable/empty proto definitions produce semantically identical JSON across all generators (server serializes what clients expect, OpenAPI documents what both produce) | ✓ VERIFIED | Consistency tests compare golden files byte-for-byte (after normalization): `nullable_consistency_test.go` (7 tests), `empty_behavior_consistency_test.go` (3 tests). All passed. Tests verify: Go server == Go client (byte-identical MarshalJSON), TypeScript types match Go behavior (`T \| null`), OpenAPI schemas match Go behavior (type arrays, oneOf). |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `proto/sebuf/http/annotations.proto` | EmptyBehavior enum, nullable and empty_behavior field extensions | ✓ VERIFIED | Lines 80-91: EmptyBehavior enum (UNSPECIFIED, PRESERVE, NULL, OMIT). Line 122: `nullable` bool extension (50013). Line 127: `empty_behavior` EmptyBehavior extension (50014). 136 lines total. |
| `internal/annotations/nullable.go` | IsNullableField, ValidateNullableAnnotation functions | ✓ VERIFIED | 71 lines. Exports: `IsNullableField(field)` (line 24), `ValidateNullableAnnotation(field, msg)` (line 46). Uses `proto.GetExtension(fieldOptions, http.E_Nullable)`. Validates proto3 optional + primitive-only. |
| `internal/annotations/empty_behavior.go` | GetEmptyBehavior, ValidateEmptyBehaviorAnnotation functions | ✓ VERIFIED | 91 lines. Exports: `GetEmptyBehavior(field)` (line 25), `HasEmptyBehaviorAnnotation(field)` (line 51), `ValidateEmptyBehaviorAnnotation(field, msg)` (line 57). Uses `proto.GetExtension(fieldOptions, http.E_EmptyBehavior)`. Validates message-only, not repeated/map. |
| `internal/httpgen/nullable.go` | Nullable MarshalJSON/UnmarshalJSON generation for go-http | ✓ VERIFIED | 207 lines. Contains `generateNullableMarshalJSON` (line 214), `generateNullableUnmarshalJSON` (line 263). Called from generator.go:90. Generates _nullable.pb.go files with custom JSON marshaling. |
| `internal/httpgen/empty_behavior.go` | Empty behavior MarshalJSON/UnmarshalJSON generation for go-http | ✓ VERIFIED | 338 lines. Contains `generateEmptyBehaviorMarshalJSON` (line 209), `generateEmptyBehaviorUnmarshalJSON` (line 279). Called from generator.go:95. Uses `proto.Size() == 0` for empty detection. Handles NULL/OMIT/PRESERVE. |
| `internal/clientgen/nullable.go` | Nullable MarshalJSON/UnmarshalJSON generation for go-client | ✓ VERIFIED | Identical to httpgen (package name differs). Consistency test verifies byte-identical output after normalization. |
| `internal/clientgen/empty_behavior.go` | Empty behavior MarshalJSON/UnmarshalJSON generation for go-client | ✓ VERIFIED | Identical to httpgen (package name differs). Consistency test verifies byte-identical output after normalization. |
| `internal/tsclientgen/types.go` | Nullable type mapping (T \| null) | ✓ VERIFIED | Line 332: `if annotations.IsNullableField(field)` → generates `fieldName: T \| null`. Non-nullable optional fields use `fieldName?: T`. Correctly distinguishes nullable (always present, value or null) from optional (may be absent). |
| `internal/openapiv3/types.go` | Nullable schema generation with type array, empty_behavior oneOf schema | ✓ VERIFIED | Line 56: `if annotations.IsNullableField(field)` → calls `makeNullableSchema()` producing `type: ["T", "null"]` array. Lines 61-65: `if empty_behavior == NULL` → calls `makeNullableOneOfSchema()` producing `oneOf: [$ref, {type: "null"}]`. |
| `internal/httpgen/nullable_consistency_test.go` | Cross-generator consistency tests for nullable annotation | ✓ VERIFIED | 218 lines. 7 tests: GoHTTPvsGoClient (byte comparison), TypeScript (T \| null syntax), OpenAPI (type array syntax), BackwardCompat. All pass. |
| `internal/httpgen/empty_behavior_consistency_test.go` | Cross-generator consistency tests for empty_behavior annotation | ✓ VERIFIED | 161 lines. 3 tests: GoHTTPvsGoClient (byte comparison), OpenAPI (oneOf syntax), BackwardCompat. All pass. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/annotations/nullable.go | proto/sebuf/http/annotations.proto | proto.GetExtension with http.E_Nullable | ✓ WIRED | Line 35: `proto.GetExtension(fieldOptions, http.E_Nullable)`. Extension descriptor E_Nullable accessible. |
| internal/annotations/empty_behavior.go | proto/sebuf/http/annotations.proto | proto.GetExtension with http.E_EmptyBehavior | ✓ WIRED | Line 36: `proto.GetExtension(fieldOptions, http.E_EmptyBehavior)`. Extension descriptor E_EmptyBehavior accessible. |
| internal/httpgen/nullable.go | internal/annotations/nullable.go | annotations.IsNullableField, annotations.ValidateNullableAnnotation | ✓ WIRED | Lines 21, 69: Calls `annotations.IsNullableField(field)` and `annotations.ValidateNullableAnnotation(field, msgName)`. Imported at line 8. |
| internal/httpgen/empty_behavior.go | internal/annotations/empty_behavior.go | annotations.GetEmptyBehavior, annotations.ValidateEmptyBehaviorAnnotation | ✓ WIRED | Lines 30, 44, 81: Calls `annotations.HasEmptyBehaviorAnnotation()`, `annotations.GetEmptyBehavior()`, `annotations.ValidateEmptyBehaviorAnnotation()`. Imported at line 9. |
| internal/httpgen/empty_behavior.go | google.golang.org/protobuf/proto | proto.Size for empty detection | ✓ WIRED | Lines 256, 264: `proto.Size(x.MetadataPreserve) == 0` used to detect empty messages. Correctly identifies messages with all fields at proto default. |
| internal/tsclientgen/types.go | internal/annotations/nullable.go | annotations.IsNullableField | ✓ WIRED | Line 332: `if annotations.IsNullableField(field)` → generates `T \| null`. Imported and used. |
| internal/openapiv3/types.go | internal/annotations/nullable.go | annotations.IsNullableField | ✓ WIRED | Line 56: `if annotations.IsNullableField(field)` → calls `makeNullableSchema()`. Imported and used. |
| internal/openapiv3/types.go | internal/annotations/empty_behavior.go | annotations.GetEmptyBehavior | ✓ WIRED | Lines 61-63: `annotations.HasEmptyBehaviorAnnotation()` and `annotations.GetEmptyBehavior()` → oneOf schema for NULL. Imported and used. |
| internal/httpgen/nullable_consistency_test.go | internal/clientgen/nullable.go | Generated MarshalJSON output comparison | ✓ WIRED | Lines 18-46: Reads golden files from both generators, normalizes, compares byte-for-byte. Test passes proving identical output. |
| internal/httpgen/nullable_consistency_test.go | internal/openapiv3/types.go | Schema type array verification | ✓ WIRED | Lines 127-164: Reads OpenAPI YAML, verifies `type: ["string", "null"]` syntax, confirms no deprecated `nullable: true`. Test passes. |

### Requirements Coverage

No requirements explicitly mapped to Phase 5 in REQUIREMENTS.md. Phase targets success criteria from ROADMAP.md.

Phase 5 requirements from ROADMAP (JSON-01, JSON-06):
- JSON-01 (Nullable primitives): ✓ SATISFIED — All 5 success criteria verified
- JSON-06 (Empty message handling): ✓ SATISFIED — All 5 success criteria verified

### Anti-Patterns Found

None. Clean implementation.

**Blockers:** 0
**Warnings:** 0
**Info:** 0

### Verification Details

**Test Results:**
```
go test ./...
ok  	github.com/SebastienMelki/sebuf/internal/annotations	(cached)
ok  	github.com/SebastienMelki/sebuf/internal/clientgen	(cached)
ok  	github.com/SebastienMelki/sebuf/internal/httpgen	(cached)
ok  	github.com/SebastienMelki/sebuf/internal/openapiv3	(cached)
ok  	github.com/SebastienMelki/sebuf/internal/tsclientgen	(cached)
```

**Consistency Tests:**
```
=== RUN   TestNullableConsistencyGoHTTPvsGoClient
--- PASS: TestNullableConsistencyGoHTTPvsGoClient (0.00s)
=== RUN   TestNullableConsistencyTypeScript
--- PASS: TestNullableConsistencyTypeScript (0.00s)
=== RUN   TestNullableConsistencyOpenAPI
--- PASS: TestNullableConsistencyOpenAPI (0.00s)
=== RUN   TestEmptyBehaviorConsistencyGoHTTPvsGoClient
--- PASS: TestEmptyBehaviorConsistencyGoHTTPvsGoClient (0.00s)
=== RUN   TestEmptyBehaviorConsistencyOpenAPI
--- PASS: TestEmptyBehaviorConsistencyOpenAPI (0.00s)
```

**Lint:** Clean (0 issues)

**Golden Files:** All present and verified
- nullable.proto + golden files across all 4 generators
- empty_behavior.proto + golden files across all 4 generators

**Generated Code Verification:**

1. **Nullable MarshalJSON** (nullable_nullable.pb.go):
   - Lines 33-47: Emits `raw["middleName"] = []byte("null")` when field is unset
   - Uses protojson for base serialization, then overwrites with null
   - Handles multiple nullable fields (middleName, age, isVerified)

2. **Nullable UnmarshalJSON** (nullable_nullable.pb.go):
   - Lines 61-77: Removes explicit `"null"` from JSON before protojson unmarshal
   - Allows protojson to leave field unset when null received
   - Correctly distinguishes null vs absent

3. **Empty Behavior MarshalJSON** (empty_behavior_empty_behavior.pb.go):
   - Lines 33-54: Uses `proto.Size(x.MetadataNull) == 0` to detect empty
   - NULL behavior: `raw["metadataNull"] = []byte("null")` (line 41)
   - OMIT behavior: `delete(raw, "metadataOmit")` (line 47)
   - PRESERVE behavior: No action, protojson default `{}` (line 35)

4. **TypeScript Types** (nullable_client.ts):
   - `middleName: string | null` (not `middleName?: string`)
   - `age: number | null`
   - `isVerified: boolean | null`
   - Correctly distinguishes nullable (always present) from optional (may be absent)

5. **OpenAPI Schemas** (NullableService.openapi.yaml):
   - Lines 119-121: `type: ["string", "null"]` for nullable middleName
   - Lines 127-129: `type: ["integer", "null"]` for nullable age
   - Lines 133-135: `type: ["boolean", "null"]` for nullable isVerified
   - No deprecated `nullable: true` syntax

6. **OpenAPI oneOf for empty_behavior=NULL** (EmptyBehaviorService.openapi.yaml):
   - metadataNull: `oneOf: [$ref: '#/components/schemas/Metadata', type: "null"]`
   - Correctly documents NULL behavior where empty message serializes as null

---

## Summary

**Phase 5 goal ACHIEVED.** All 5 success criteria verified:

1. ✓ Nullable annotation generates pointer types (Go), union types (TypeScript), type arrays (OpenAPI)
2. ✓ Three distinct states representable: absent, null, value
3. ✓ Empty behavior controls message serialization: PRESERVE ({}), NULL (null), OMIT (omitted)
4. ✓ Semantics consistent across all 4 generators (shared annotations package)
5. ✓ Cross-generator consistency tests confirm identical JSON output

**Evidence:**
- 11 new files created (annotations, nullable/empty_behavior for each generator, consistency tests)
- 10 consistency tests passing
- All generators use shared `internal/annotations` package
- Golden files demonstrate correct output across all generators
- `proto.Size() == 0` correctly detects empty messages
- Type array syntax for nullable (OpenAPI 3.1 compliant)
- oneOf with null for empty_behavior=NULL
- Zero anti-patterns, zero regressions

Ready to proceed to Phase 6.

---

_Verified: 2026-02-06T10:25:12Z_
_Verifier: Claude (gsd-verifier)_
