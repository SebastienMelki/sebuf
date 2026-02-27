---
phase: quick
plan: 2
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/tscommon/types_test.go
  - internal/tscommon/helpers_test.go
  - internal/httpgen/testdata/proto/query_params.proto
autonomous: true
requirements: []

must_haves:
  truths:
    - "TSZeroCheck returns correct zero-check expressions for all proto field kinds"
    - "TSEnumUnspecifiedValue returns first enum value name, respecting custom enum_value annotations"
    - "TSZeroCheckForField handles enum fields with !== UNSPECIFIED check, repeated fields with truthy check, and int64 fields with encoding-aware checks"
    - "SnakeToLowerCamel and HeaderNameToPropertyName produce correct conversions"
    - "Custom enum_value annotations are exercised in golden file tests via an enum in query_params.proto"
  artifacts:
    - path: "internal/tscommon/types_test.go"
      provides: "Unit tests for TSZeroCheck, TSScalarType, TSScalarTypeForField, TSEnumUnspecifiedValue, TSZeroCheckForField"
      min_lines: 80
    - path: "internal/tscommon/helpers_test.go"
      provides: "Unit tests for SnakeToLowerCamel, SnakeToUpperCamel, HeaderNameToPropertyName"
      min_lines: 40
  key_links:
    - from: "internal/tscommon/types_test.go"
      to: "internal/tscommon/types.go"
      via: "direct function calls"
      pattern: "tscommon\\.TS"
    - from: "internal/httpgen/testdata/proto/query_params.proto"
      to: "internal/tsclientgen/testdata/golden/query_params_client.ts"
      via: "golden file regeneration"
      pattern: "enum_value"
---

<objective>
Add unit tests for tscommon helper functions and expand enum_value annotation coverage in test protos.

Purpose: The tscommon package has zero test coverage. The TSZeroCheck, TSZeroCheckForField, TSEnumUnspecifiedValue, SnakeToLowerCamel, and HeaderNameToPropertyName functions are used by both ts-client and ts-server generators but have no direct tests. Additionally, the query_params.proto Region enum lacks custom enum_value annotations, leaving that GetEnumValueMapping code path untested in the query param context.

Output: Two new test files (types_test.go, helpers_test.go) and updated golden files reflecting the new enum_value annotation on Region.
</objective>

<execution_context>
@/Users/sebastienmelki/.claude/get-shit-done/workflows/execute-plan.md
@/Users/sebastienmelki/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/tscommon/types.go
@internal/tscommon/helpers.go
@internal/annotations/enum_encoding.go
@internal/annotations/query.go
@internal/httpgen/testdata/proto/query_params.proto
@internal/httpgen/unwrap_test.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Unit tests for tscommon pure functions (types_test.go + helpers_test.go)</name>
  <files>internal/tscommon/types_test.go, internal/tscommon/helpers_test.go</files>
  <action>
Create two test files with table-driven tests.

**`internal/tscommon/helpers_test.go`** -- Test pure string helpers:

1. `TestSnakeToLowerCamel` -- table-driven:
   - `"user_id"` -> `"userId"`
   - `"page_number"` -> `"pageNumber"`
   - `"simple"` -> `"simple"` (no underscores)
   - `"a_b_c"` -> `"aBC"`
   - `""` -> `""` (empty string)
   - `"already_camelCase"` -> `"alreadyCamelCase"` (mixed -- just verify behavior)

2. `TestSnakeToUpperCamel` -- table-driven:
   - `"user_id"` -> `"UserId"`
   - `"page_number"` -> `"PageNumber"`
   - `"simple"` -> `"Simple"`
   - `""` -> `""`

3. `TestHeaderNameToPropertyName` -- table-driven:
   - `"X-API-Key"` -> `"apiKey"`
   - `"X-Request-ID"` -> `"requestId"`
   - `"Content-Type"` -> `"contentType"` (no X- prefix)
   - `"X-Tenant-ID"` -> `"tenantId"`

**`internal/tscommon/types_test.go`** -- Test type mapping and zero-check functions:

1. `TestTSScalarType` -- table-driven covering ALL protoreflect.Kind values:
   - StringKind -> "string"
   - BoolKind -> "boolean"
   - Int32Kind, Uint32Kind, FloatKind, DoubleKind -> "number"
   - Int64Kind, Uint64Kind -> "string" (proto3 JSON default)
   - BytesKind -> "string"
   - EnumKind -> "string"
   - MessageKind, GroupKind -> "unknown"

2. `TestTSZeroCheck` -- table-driven for the string-based function:
   - `"string"` -> ` !== ""`
   - `"bool"` -> `""` (truthy check)
   - `"int32"`, `"uint32"`, `"float"`, `"double"` -> ` !== 0`
   - `"int64"`, `"uint64"` -> ` !== "0"`
   - `"enum"` -> `""` (truthy check, no field context)
   - `"unknown_kind"` -> ` !== ""` (default fallback)
   - Also test sint32, sfixed32, sint64, sfixed64, fixed32, fixed64

3. `TestTSEnumUnspecifiedValue` -- Uses protoc to generate real protogen fields. Follow the unwrap_test.go pattern:
   - Run protoc on `enum_encoding.proto` (which has Status with custom enum_value "unknown" on first value, and Priority without custom values)
   - Parse the generated TS client output to verify:
     a. Status enum first value uses custom annotation: verify the generated TS contains `"unknown"` as first value in Status type
     b. Priority enum first value uses proto name: verify `"PRIORITY_LOW"` as first value in Priority type
   - This is an integration-style test since we cannot easily construct protogen.Field mocks with extension options. The golden files already cover this, but this test explicitly validates the TSEnumUnspecifiedValue logic via its output in the generated code.

   ALTERNATIVELY (simpler, preferred): Since TSEnumUnspecifiedValue requires a real *protogen.Field with populated Enum and extension options, and mocking protogen is complex, instead write a test that validates the behavior THROUGH the golden file output. Specifically:
   - Read `internal/tsclientgen/testdata/golden/enum_encoding_client.ts`
   - Verify Status type line contains `"unknown"` (custom enum_value) as first value
   - Verify Priority type line contains `"PRIORITY_LOW"` (proto name) as first value
   - This confirms TSEnumUnspecifiedValue + GetEnumValueMapping work correctly

   Name this test `TestTSEnumUnspecifiedValue_ViaGoldenOutput` with a comment explaining why it validates through golden output rather than direct function calls (protogen.Field requires real protoc extensions).

Important: The package for types_test.go and helpers_test.go is `tscommon` (same package, internal tests). Import `protoreflect` for kind constants in types_test.go.
  </action>
  <verify>
Run: `cd /Users/sebastienmelki/Documents/documents_sebastiens_mac_mini/Workspace/kompani/sebuf.nosync && go test ./internal/tscommon/ -v -count=1`

All tests pass. Expect:
- TestSnakeToLowerCamel (6+ subtests)
- TestSnakeToUpperCamel (4+ subtests)
- TestHeaderNameToPropertyName (4+ subtests)
- TestTSScalarType (10+ subtests)
- TestTSZeroCheck (12+ subtests)
- TestTSEnumUnspecifiedValue_ViaGoldenOutput (2 subtests)
  </verify>
  <done>All tscommon unit tests pass. Pure functions (TSScalarType, TSZeroCheck, SnakeToLowerCamel, HeaderNameToPropertyName) have table-driven tests covering all branches. TSEnumUnspecifiedValue behavior validated through golden file output for both custom and default enum values.</done>
</task>

<task type="auto">
  <name>Task 2: Add custom enum_value to Region enum in query_params.proto and update golden files</name>
  <files>internal/httpgen/testdata/proto/query_params.proto</files>
  <action>
Modify the `Region` enum in `query_params.proto` to add custom `enum_value` annotations on some values, exercising the GetEnumValueMapping branch in TSEnumUnspecifiedValue when used as a query parameter.

Change the Region enum from:
```protobuf
enum Region {
  REGION_UNSPECIFIED = 0;
  REGION_AMERICAS = 1;
  REGION_EUROPE = 2;
  REGION_ASIA = 3;
}
```

To:
```protobuf
enum Region {
  REGION_UNSPECIFIED = 0 [(sebuf.http.enum_value) = "unspecified"];
  REGION_AMERICAS = 1 [(sebuf.http.enum_value) = "americas"];
  REGION_EUROPE = 2 [(sebuf.http.enum_value) = "europe"];
  REGION_ASIA = 3 [(sebuf.http.enum_value) = "asia"];
}
```

This ensures:
- TSEnumUnspecifiedValue returns `"unspecified"` (custom) instead of `"REGION_UNSPECIFIED"` (proto name)
- TSZeroCheckForField for enum query params generates ` !== "unspecified"` instead of ` !== "REGION_UNSPECIFIED"`
- The TS type union becomes `"unspecified" | "americas" | "europe" | "asia"` instead of proto names
- Both tsclientgen and tsservergen golden files will update (they symlink to same proto)

After modifying the proto:
1. `rm -rf bin && make build` (rebuild plugins)
2. `UPDATE_GOLDEN=1 go test -run TestTSClientGenGoldenFiles ./internal/tsclientgen/` (update ts-client golden)
3. `UPDATE_GOLDEN=1 go test -run TestTSServerGenGoldenFiles ./internal/tsservergen/` (update ts-server golden)
4. `UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles ./internal/httpgen/` (update go-http golden)
5. `UPDATE_GOLDEN=1 go test -run TestExhaustiveGoldenFiles ./internal/openapiv3/` (update openapi golden)
6. `UPDATE_GOLDEN=1 go test -run TestGoClientGoldenFiles ./internal/clientgen/` (update go-client golden if it has query_params)

Verify the updated golden files show the custom enum values ("unspecified", "americas", etc.) instead of proto names.

Then update the TestTSEnumUnspecifiedValue_ViaGoldenOutput test from Task 1 if needed -- it reads enum_encoding golden (not query_params), so it should not need changes. But add a second golden validation test that reads the query_params golden to verify Region uses custom values:
- Read `internal/tsclientgen/testdata/golden/query_params_client.ts`
- Verify Region type line contains `"unspecified"` | `"americas"` | `"europe"` | `"asia"`
- Verify the searchAdvanced method's query param encoding uses ` !== "unspecified"` for the region zero check

Run `make lint-fix` after all changes.
  </action>
  <verify>
Run all tests:
```
cd /Users/sebastienmelki/Documents/documents_sebastiens_mac_mini/Workspace/kompani/sebuf.nosync
go test ./... -count=1
```

All 8 test packages pass. Specifically:
- `go test ./internal/tsclientgen/ -v -run TestTSClientGenGoldenFiles/query_parameters` passes
- `go test ./internal/tsservergen/ -v -run TestTSServerGenGoldenFiles/query_parameters` passes
- `go test ./internal/tscommon/ -v` passes (including new golden validation for custom enum_value on Region)
- `make lint-fix` reports 0 issues
  </verify>
  <done>Region enum in query_params.proto uses custom enum_value annotations. All golden files across all 5 generators are updated to reflect the custom values. The GetEnumValueMapping code path is now exercised in the query parameter context (not just the enum_encoding.proto context). Full test suite passes.</done>
</task>

</tasks>

<verification>
1. `go test ./internal/tscommon/ -v -count=1` -- all new unit tests pass
2. `go test ./... -count=1` -- full test suite passes (no regressions from proto changes)
3. `make lint-fix` -- 0 lint issues
4. Verify `internal/tscommon/types_test.go` exists with tests for TSScalarType, TSZeroCheck, and golden-based TSEnumUnspecifiedValue validation
5. Verify `internal/tscommon/helpers_test.go` exists with tests for SnakeToLowerCamel, SnakeToUpperCamel, HeaderNameToPropertyName
6. Verify `internal/httpgen/testdata/proto/query_params.proto` Region enum has `enum_value` annotations
</verification>

<success_criteria>
- Two new test files in internal/tscommon/ with comprehensive table-driven tests
- All pure function branches covered (all proto kinds in TSScalarType, all field kinds in TSZeroCheck)
- TSEnumUnspecifiedValue behavior validated for both custom and default enum values
- Region enum in query_params.proto exercises custom enum_value in query param context
- All golden files updated consistently across all generators
- Full test suite passes with 0 lint issues
</success_criteria>

<output>
After completion, create `.planning/quick/2-add-unit-tests-for-tscommon-helpers-and-/2-SUMMARY.md`
</output>
