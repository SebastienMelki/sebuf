---
phase: quick-1
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/tsclientgen/generator.go
  - internal/tsservergen/generator.go
  - internal/tscommon/types.go
  - internal/tsclientgen/testdata/proto/query_params.proto
  - internal/tsservergen/testdata/golden/query_params_server.ts
  - internal/tsclientgen/testdata/golden/query_params_client.ts
  - internal/tsclientgen/testdata/golden/complex_features_client.ts
  - internal/tsservergen/testdata/golden/complex_features_server.ts
autonomous: true
requirements: [BUG-1, BUG-2, BUG-3, BUG-4, BUG-5, BUG-6]

must_haves:
  truths:
    - "Enum query params in TS client compare against the _UNSPECIFIED variant, not empty string"
    - "Enum query params in TS server are cast to the enum type, not defaulted to empty string"
    - "Repeated string query params in TS server use params.getAll(), not params.get()"
    - "Repeated string query params in TS client check .length > 0 and join with comma"
    - "TS server does not emit duplicate const url when route has both path params and query params"
    - "TS client prefixes unused req with _ for empty request messages"
  artifacts:
    - path: "internal/tsclientgen/generator.go"
      provides: "Fixed client query param generation for enums, repeated, and empty requests"
    - path: "internal/tsservergen/generator.go"
      provides: "Fixed server query param generation for enums, repeated, and duplicate URL"
    - path: "internal/tscommon/types.go"
      provides: "New helper functions TSZeroCheckForField for enum fields, TSEnumUnspecifiedValue"
  key_links:
    - from: "internal/tsclientgen/generator.go"
      to: "internal/tscommon/types.go"
      via: "tsZeroCheckForField and enum helpers"
      pattern: "tscommon\\.TS"
    - from: "internal/tsservergen/generator.go"
      to: "internal/tscommon/types.go"
      via: "TSScalarTypeForField and enum helpers"
      pattern: "tscommon\\.TS"
---

<objective>
Fix 6 TypeScript generation bugs affecting enum query params, repeated field query params, duplicate URL const, and unused req parameter.

Purpose: Produce type-correct TypeScript output from protoc-gen-ts-client and protoc-gen-ts-server for enum query params, repeated query params, mixed path+query routes, and empty request messages.
Output: Fixed generator code, updated golden files, passing tests.
</objective>

<execution_context>
@/Users/sebastienmelki/.claude/get-shit-done/workflows/execute-plan.md
@/Users/sebastienmelki/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/tsclientgen/generator.go
@internal/tsservergen/generator.go
@internal/tscommon/types.go
@internal/annotations/query.go
@internal/tsclientgen/types.go
@internal/tsclientgen/testdata/proto/query_params.proto
@internal/tsclientgen/testdata/golden/query_params_client.ts
@internal/tsservergen/testdata/golden/query_params_server.ts
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add enum/repeated query param test cases to proto file and add helper functions</name>
  <files>
    internal/tsclientgen/testdata/proto/query_params.proto
    internal/tscommon/types.go
  </files>
  <action>
**1a. Add test cases to `query_params.proto`:**

Add a new enum definition and a new RPC + request message that exercises all bug scenarios. Add these AFTER the existing messages in the proto file:

```protobuf
enum Region {
  REGION_UNSPECIFIED = 0;
  REGION_AMERICAS = 1;
  REGION_EUROPE = 2;
  REGION_ASIA = 3;
}

message SearchAdvancedRequest {
  // Enum query param (bugs #1 and #2)
  Region region = 1 [(sebuf.http.query) = { name: "region" }];
  // Repeated string query param (bugs #3 and #4)
  repeated string countries = 2 [(sebuf.http.query) = { name: "countries" }];
  // Normal string for baseline
  string keyword = 3 [(sebuf.http.query) = { name: "keyword" }];
}

// Empty request message (bug #6)
message EmptyRequest {}
```

Add two new RPCs to the `QueryParamService`:

```protobuf
  // Advanced search with enum + repeated params
  rpc SearchAdvanced(SearchAdvancedRequest) returns (SearchResponse) {
    option (sebuf.http.config) = {
      path: "/search/advanced"
      method: HTTP_METHOD_GET
    };
  }

  // RPC with empty request message
  rpc GetDefaults(EmptyRequest) returns (SearchResponse) {
    option (sebuf.http.config) = {
      path: "/defaults"
      method: HTTP_METHOD_GET
    };
  }
```

Note: The `tsservergen/testdata/proto/query_params.proto` is a symlink to `httpgen/testdata/proto/query_params.proto` (or tsclientgen). Check with `ls -la` first. If it's NOT a symlink, it's a copy -- update the tsclientgen version (the canonical one shared via symlink) or both files.

**1b. Add helper functions to `internal/tscommon/types.go`:**

Add a function `TSEnumUnspecifiedValue` that returns the first enum value name (the UNSPECIFIED variant) given a `*protogen.Field`:

```go
// TSEnumUnspecifiedValue returns the first enum value (UNSPECIFIED variant) as a quoted string.
// For enum fields, this is the zero-value equivalent used in TS zero checks.
func TSEnumUnspecifiedValue(field *protogen.Field) string {
    if field.Desc.Kind() != protoreflect.EnumKind || field.Enum == nil {
        return `""`
    }
    values := field.Enum.Values
    if len(values) == 0 {
        return `""`
    }
    // Check for custom enum_value annotation on the first value
    customValue := annotations.GetEnumValueMapping(values[0])
    if customValue != "" {
        return fmt.Sprintf(`"%s"`, customValue)
    }
    return fmt.Sprintf(`"%s"`, string(values[0].Desc.Name()))
}
```

Update `TSZeroCheckForField` to handle enum kind:

Currently it only handles int64 kinds specially and falls back to `TSZeroCheck(kind.String())` for everything else. The problem is that `kind.String()` returns `"enum"` for enum fields, which falls through to the default `!== ""` case. This generates `!== ""` but the TS type is a string union like `"REGION_UNSPECIFIED" | "REGION_AMERICAS"` where `""` is not a valid value.

Add an `EnumKind` case BEFORE the default:

```go
case protoreflect.EnumKind:
    return " !== " + TSEnumUnspecifiedValue(field)
```

Also update `TSZeroCheck` (the non-field version) to handle the `"enum"` kind string -- return `""` (empty string = use bool check, same as bool) since without the field context we cannot know the UNSPECIFIED value. Note: this fallback path is only used when `qp.Field` is nil, which is rare.

**1c. Add `IsRepeatedField` check to `TSZeroCheckForField`:**

Before the switch statement in `TSZeroCheckForField`, add an early return for repeated (list) fields:

```go
// Repeated fields use length check
if field.Desc.IsList() {
    return ""  // empty string = use the truthy check pattern (like bool)
}
```

This ensures that repeated fields get a simple truthy check rather than a string comparison.
  </action>
  <verify>
Run `go build ./internal/tscommon/...` to verify compilation. Run `go test ./internal/tscommon/...` if unit tests exist.
  </verify>
  <done>
`TSEnumUnspecifiedValue` returns the first enum value name. `TSZeroCheckForField` returns ` !== "REGION_UNSPECIFIED"` for enum fields and `""` (truthy check) for repeated fields.
  </done>
</task>

<task type="auto">
  <name>Task 2: Fix all 6 bugs in ts-client and ts-server generators</name>
  <files>
    internal/tsclientgen/generator.go
    internal/tsservergen/generator.go
  </files>
  <action>
**Bug #1 -- protoc-gen-ts-client: enum fields compared to ""**

In `generateURLBuilding` in `internal/tsclientgen/generator.go`, the zero check logic already calls `tsZeroCheckForField(qp.Field)` when `qp.Field != nil`. The fix from Task 1 handles this -- `TSZeroCheckForField` now returns ` !== "REGION_UNSPECIFIED"` for enum fields. No additional changes needed in the client generator for this bug, as long as Task 1's `TSZeroCheckForField` changes are correct.

Verify this by tracing the code path: `generateURLBuilding` -> `tsZeroCheckForField(qp.Field)` -> `tscommon.TSZeroCheckForField(field)` -> new `EnumKind` case.

**Bug #2 -- protoc-gen-ts-server: enum fields assigned from params.get() ?? ""**

In `generateQueryParamField` in `internal/tsservergen/generator.go`, the `default:` case in the `tsType` switch emits `params.get("region") ?? ""`. For enum fields, the type is a string union where `""` is not valid.

Fix: Add a new case BEFORE the default in the `if qp.Field != nil` branch. After the `case tscommon.TSBoolean:` case and before `default:`, detect enum fields:

```go
// Check if it's an enum field
if qp.Field != nil && qp.Field.Desc.Kind() == protoreflect.EnumKind && qp.Field.Enum != nil {
    unspecified := tscommon.TSEnumUnspecifiedValue(qp.Field)
    p(`            %s: (params.get("%s") ?? %s) as %s,`, jsonName, paramName, unspecified, string(qp.Field.Enum.Desc.Name()))
} else {
    // default string
    p(`            %s: params.get("%s") ?? "",`, jsonName, paramName)
}
```

This needs to replace the existing `default:` case in the `qp.Field != nil` branch. The enum field gets a cast to the enum type and defaults to the UNSPECIFIED value.

Similarly in the `qp.Field == nil` fallback branch (the `else` with FieldKind switch), add a case for `"enum"` kind:
```go
case "enum":
    p(`            %s: params.get("%s") ?? "",`, jsonName, paramName)
```
(Without field context, we can only default to `""` and let TS figure it out.)

**Bug #3 -- protoc-gen-ts-server: repeated string fields read as single string**

In `generateQueryParamField` in `internal/tsservergen/generator.go`, add an early check at the top of the function for repeated fields:

```go
func (g *Generator) generateQueryParamField(p tscommon.Printer, qp annotations.QueryParam) {
    jsonName := qp.FieldJSONName
    paramName := qp.ParamName

    // Handle repeated fields (bug #3)
    if qp.Field != nil && qp.Field.Desc.IsList() {
        p(`            %s: params.getAll("%s"),`, jsonName, paramName)
        return
    }

    // ... existing code ...
}
```

This uses `params.getAll()` which returns `string[]`, matching the TS interface type for repeated string fields.

**Bug #4 -- protoc-gen-ts-client: repeated string fields compared to ""**

In `generateURLBuilding` in `internal/tsclientgen/generator.go`, the zero check for repeated fields returns `""` (truthy check) from Task 1. But the `params.set` call uses `String(req.countries)` which would stringify the array wrong.

The fix needs to be in the generation code. After the zero-check guard, when the field is a repeated/list field, use a different serialization strategy. Modify the query param generation loop in `generateURLBuilding`:

After computing `check`, add detection for repeated fields:

```go
// Handle repeated fields specially
if qp.Field != nil && qp.Field.Desc.IsList() {
    p("    if (req.%s && req.%s.length > 0) req.%s.forEach(v => params.append(\"%s\", v));",
        qp.FieldJSONName, qp.FieldJSONName, qp.FieldJSONName, qp.ParamName)
    continue  // Skip the normal params.set logic below
}
```

Place this BEFORE the `if check == ""` block, inside the `for _, qp := range cfg.queryParams` loop. Use `continue` to skip the standard set logic. This uses `params.append` for each value so the URL gets `?countries=US&countries=UK`.

**Bug #5 -- protoc-gen-ts-server: duplicate const url**

In `internal/tsservergen/generator.go`, the `generatePathParamExtraction` function emits `const url = new URL(req.url, "http://localhost");` when there are path params (line 548). Then `generateQueryParamParsing` ALSO emits `const url = new URL(req.url, "http://localhost");` (line 595).

Fix: In `generateQueryParamParsing`, check if path params already exist. If they do, the `url` variable is already declared -- just access `url.searchParams` without re-declaring `url`.

Modify `generateQueryParamParsing` to accept the `cfg *rpcRouteConfig` it already receives, and check `len(cfg.pathParams) > 0`:

```go
if len(cfg.pathParams) > 0 {
    // url already declared in path param extraction
    p("          const params = url.searchParams;")
} else {
    p("          const url = new URL(req.url, \"http://localhost\");")
    p("          const params = url.searchParams;")
}
```

Replace the current two lines:
```go
p("          const url = new URL(req.url, \"http://localhost\");")
p("          const params = url.searchParams;")
```

**Bug #6 -- protoc-gen-ts-client: unused req parameter for empty request messages**

In `generateRPCMethod` in `internal/tsclientgen/generator.go`, at line 272 the signature is:
```go
p("  async %s(req: %s, options?: %sCallOptions): Promise<%s> {", ...)
```

Check if the input message has zero fields. If so, prefix with `_`:

```go
reqParam := "req"
if len(method.Input.Fields) == 0 {
    reqParam = "_req"
}
p("  async %s(%s: %s, options?: %sCallOptions): Promise<%s> {",
    tsMethodName, reqParam, inputType, cfg.serviceName, outputType)
```

This only affects the parameter name in the signature. The body already won't reference `req` since there are no fields to build path/query params from and no body to send for GET methods with empty input.

**After all fixes, rebuild and update golden files:**

```bash
rm -rf bin && make build
UPDATE_GOLDEN=1 go test -run TestTSClientGenGoldenFiles ./internal/tsclientgen/
UPDATE_GOLDEN=1 go test -run TestTSServerGenGoldenFiles ./internal/tsservergen/
```

Then run the full test suite to make sure nothing else broke:

```bash
go test ./internal/tsclientgen/ ./internal/tsservergen/ ./internal/tscommon/
```

Then run lint:

```bash
make lint-fix
```
  </action>
  <verify>
1. `rm -rf bin && make build` succeeds
2. `go test ./internal/tsclientgen/ ./internal/tsservergen/ ./internal/tscommon/` all pass
3. `make lint-fix` reports 0 issues
4. Inspect the updated golden files to manually verify:
   - `query_params_client.ts`: enum field uses `!== "REGION_UNSPECIFIED"`, repeated field uses `forEach` with `params.append`, `getDefaults` uses `_req`
   - `query_params_server.ts`: enum field uses `as Region` cast with UNSPECIFIED default, repeated field uses `params.getAll`, `getWithFilters` handler has only ONE `const url` declaration, `getDefaults` handler uses `{} as EmptyRequest`
  </verify>
  <done>
All 6 bugs fixed: (1) client enum zero check uses UNSPECIFIED variant, (2) server enum default uses UNSPECIFIED + type cast, (3) server repeated uses getAll(), (4) client repeated uses forEach+append, (5) server emits url const only once for mixed path+query routes, (6) client prefixes unused req with _ for empty messages. All golden files updated and tests pass.
  </done>
</task>

</tasks>

<verification>
1. `go test ./...` passes (all packages, not just the two modified)
2. Golden files show correct TypeScript for all 6 bug scenarios
3. Existing golden files for other test protos are unchanged (no regressions)
4. `make lint-fix` clean
</verification>

<success_criteria>
- TypeScript generated for enum query params compiles without type errors (no `""` comparison against string union)
- TypeScript generated for repeated query params uses array operations (getAll/forEach+append)
- No duplicate variable declarations in generated server code for mixed path+query routes
- No unused parameter warnings in generated client code for empty request messages
- All existing tests continue to pass (no regressions)
</success_criteria>

<output>
After completion, create `.planning/quick/1-fix-6-ts-generator-bugs-enum-query-param/1-SUMMARY.md`
</output>
