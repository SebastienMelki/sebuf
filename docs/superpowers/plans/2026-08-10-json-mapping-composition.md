# JSON Mapping Composition Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace httpgen's per-feature JSON-mapping method emitters with one composed `MarshalJSONSebuf` / `UnmarshalJSONSebuf` emitter per message, backed by a regression test matrix.

**Architecture:** Introduce a central `internal/httpgen/json_mapping*.go` model that collects every JSON-mapping transform needed by a message, then emits exactly one generated file per proto file containing one method pair per affected message. The generated marshal path uses four stages: protojson base object, nested child delegation, field/document-local transforms, then optional root unwrap document replacement. The unmarshal path mirrors the inverse transforms before delegating to `protojson.UnmarshalOptions`.

**Tech Stack:** Go protoc plugin using `google.golang.org/protobuf/compiler/protogen`, `protojson`, existing sebuf annotations in `internal/annotations`, Go tests, generated-code compile tests, and golden tests.

## Global Constraints

- Scope is `protoc-gen-go-http` / `internal/httpgen` first; do not implement parallel clientgen changes unless a task explicitly says so.
- Generated Go APIs must keep `MarshalJSONSebuf(opts protojson.MarshalOptions) ([]byte, error)`, `MarshalJSON() ([]byte, error)`, `UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error`, and `UnmarshalJSON(data []byte) error`.
- A generated message type must never receive more than one method with the same name.
- Single-feature message behavior should remain byte-compatible unless the existing output is proven wrong by the new nested/composition semantics.
- `protojson` is still the base serializer/deserializer for unannotated fields.
- Custom child message mappings must be delegated explicitly; do not assume `protojson` calls generated methods.
- Prefer focused new files over growing the largest existing emitters further.
- Run `go fmt ./...` after Go changes.
- Do not update broad golden files until targeted behavior tests and generated-code build tests prove the new behavior.

---

## Task 1: Requirements and Test Matrix Document

**Purpose:** Give every later fresh-context agent a precise contract and acceptance matrix before code changes.

**Files:**
- Create: `docs/superpowers/specs/2026-08-10-json-mapping-composition-requirements.md`
- Read-only context: `internal/httpgen/*_consistency_test.go`, `internal/httpgen/unwrap_test.go`, `internal/httpgen/golden_test.go`, issue #254

**Interfaces:**
- Produces: a requirements/spec document with named test cases that later tasks must reference.
- Consumes: no code from later tasks.

- [ ] **Step 1: Re-read the issue and existing tests**

Run:
```bash
gh issue view 254 --repo SebastienMelki/sebuf --json title,body,labels,url
rg -n "MarshalJSONSebuf|UnmarshalJSONSebuf|int64_encoding|nullable|empty_behavior|timestamp_format|bytes_encoding|enum_value|flatten|oneof_config|unwrap" internal/httpgen
```

- [ ] **Step 2: Write the requirements document**

Create `docs/superpowers/specs/2026-08-10-json-mapping-composition-requirements.md` with these sections:

```markdown
# JSON Mapping Composition Requirements

## Problem

httpgen currently emits separate JSON methods per feature. That makes same-message JSON-mapping combinations conflict, makes some combinations compile-broken, and lets annotated child messages lose their custom mapping when encoded through an annotated parent.

## In-Scope Features

Field/document transforms handled by the composed emitter:
- `int64_encoding=NUMBER`
- `enum_value` / field enum string encoding
- `bytes_encoding`
- `timestamp_format`
- `nullable`
- `empty_behavior`
- `flatten`
- `oneof_config`
- `unwrap`, split into map-value unwrap and root unwrap

## Required Marshal Pipeline

1. Return `null` for nil receiver.
2. Use `opts.Marshal(x)` as the unannotated base.
3. Decode the base object into `map[string]json.RawMessage`, except root unwrap emitters may decode/construct the final root document as needed.
4. Re-marshal child message fields that implement `MarshalJSONSebuf`, including singular message fields, repeated message fields, and map values where the generated code can safely address the value type.
5. Apply field transforms for annotated fields. Transforms that touch different JSON keys must compose.
6. Apply root unwrap last if the message is root-unwrapped.
7. Return `json.Marshal` of the final raw document.

## Required Unmarshal Pipeline

1. Parse incoming JSON into raw JSON structures.
2. Invert root unwrap first when present so protojson receives the normal object shape.
3. Invert field transforms: number-to-string int64, enum custom string-to-proto enum value, nullable presence/null handling, timestamp/bytes custom formats, oneof discriminator shape, flatten expansion, and map-value unwrap rewrapping.
4. Delegate child raw JSON to `UnmarshalJSONSebuf` when the child implements it, then put protojson-compatible JSON back into the parent raw object.
5. Call `opts.Unmarshal` for the final proto-compatible JSON object.

## Test Matrix

### Generation and Build Matrix
For each pair among these feature families, generate a message containing both features on different fields, run protoc/buf generation, and run `go test` or `go test ./...` on the generated package:

- int64_number
- enum_value
- bytes_encoding
- timestamp_format
- nullable
- empty_behavior
- flatten
- oneof_config
- map_value_unwrap
- root_unwrap

Expected: all valid pairs generate and build. Root unwrap pairs are valid when the root unwrap field can be structurally combined with sibling annotations by the composed document transform. Invalid annotation shapes should fail only for semantic annotation errors, not duplicate method declarations.

### Runtime Marshal Cases
- nullable + timestamp_format on one message emits `null` and unix timestamp simultaneously.
- enum_value + nullable emits custom enum strings and explicit null.
- bytes_encoding child nested under nullable parent emits child hex/base64url/raw format, not protojson default base64.
- int64_number child nested under another annotated parent emits JSON numbers at every nested level.
- flatten parent delegates child custom mappings before promoting fields.
- map-value unwrap composes with sibling nullable/timestamp/enum fields.
- root unwrap composes with child/map-value transforms.

### Runtime Unmarshal Cases
- The JSON emitted by each marshal runtime case unmarshals into the expected proto values using `UnmarshalJSONSebuf`.
- Child custom JSON accepted through a parent does not rely on protojson accepting the custom representation directly.

### Regression Invariants
- Generated code contains at most one `MarshalJSONSebuf` and one `UnmarshalJSONSebuf` per Go message type.
- Previous conflict errors containing `only one MarshalJSON-generating feature is supported per message` disappear for composable cases.
- No generated file imports unused packages.
- Existing single-feature golden behavior is preserved unless explicitly covered by a corrected nested/composed runtime assertion.
```

- [ ] **Step 3: Self-review the requirements document**

Run:
```bash
rg -n "TBD|TODO|later|appropriate|similar" docs/superpowers/specs/2026-08-10-json-mapping-composition-requirements.md
```
Expected: no matches except intentional prose quoted from issue text, if any.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-08-10-json-mapping-composition-requirements.md
git commit -m "docs: specify json mapping composition requirements"
```

---

## Task 2: Add Failing Composition Test Bed

**Purpose:** Add tests that demonstrate the desired behavior before implementation. These tests are expected to fail against the old per-feature emitter.

**Files:**
- Create: `internal/httpgen/json_mapping_composition_test.go`
- Create fixtures as needed under: `internal/httpgen/testdata/json_mapping_composition/`
- Modify only if reusable helpers already exist: `internal/httpgen/golden_test.go` or local test helpers

**Interfaces:**
- Produces: tests named `TestJSONMappingFeaturePairsGenerateAndBuild`, `TestJSONMappingNestedDelegationRuntime`, `TestJSONMappingComposedMarshalRuntime`, and `TestJSONMappingComposedUnmarshalRuntime`.
- Consumes: requirements document from Task 1.

- [ ] **Step 1: Inspect existing protoc/golden helper style**

Run:
```bash
rg -n "protoc|buf generate|GeneratedFile|testdata|UPDATE_GOLDEN|exec.Command" internal/httpgen internal/urlparamtest
```

- [ ] **Step 2: Add a pairwise generation/build test**

In `internal/httpgen/json_mapping_composition_test.go`, create a table-driven test with feature names matching Task 1. The test should generate proto files dynamically or from fixtures, run the go-http plugin, then run `go test` in the generated module/package.

The minimum pair set must include:
```go
[]struct{ left, right string }{
    {"nullable", "timestamp_format"},
    {"nullable", "enum_value"},
    {"nullable", "bytes_encoding"},
    {"nullable", "empty_behavior"},
    {"nullable", "flatten"},
    {"nullable", "oneof_config"},
    {"nullable", "map_value_unwrap"},
    {"timestamp_format", "bytes_encoding"},
    {"enum_value", "bytes_encoding"},
    {"int64_number", "nullable"},
    {"int64_number", "timestamp_format"},
    {"map_value_unwrap", "timestamp_format"},
    {"root_unwrap", "bytes_encoding"},
}
```

- [ ] **Step 3: Add runtime nested delegation tests**

Add runtime assertions for these exact shapes:

```protobuf
message HexInner {
  bytes b = 1 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_HEX];
}
message NullableOuter {
  HexInner inner = 1;
  optional string n = 2 [(sebuf.http.nullable) = true];
}
```

Expected marshal JSON contains:
```json
{"inner":{"b":"48656c6c6f"},"n":null}
```

Also add an int64 nested shape:
```protobuf
message NumberInner {
  int64 id = 1 [(sebuf.http.int64_encoding) = INT64_ENCODING_NUMBER];
}
message TimestampOuter {
  NumberInner inner = 1;
  google.protobuf.Timestamp at = 2 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS];
}
```

Expected marshal JSON contains a numeric `inner.id` and numeric `at`.

- [ ] **Step 4: Add runtime composed unmarshal tests**

For each runtime marshal case, unmarshal the expected custom JSON using `UnmarshalJSONSebuf` and assert the Go message fields contain the expected bytes, int64, optional presence, and timestamp values.

- [ ] **Step 5: Run tests and confirm failure on current main**

Run:
```bash
go test ./internal/httpgen -run 'TestJSONMapping' -count=1 -v
```
Expected before implementation: at least one failure from duplicate method generation, generation rejection, build failure, or wrong runtime JSON.

- [ ] **Step 6: Commit failing test bed**

```bash
git add internal/httpgen/json_mapping_composition_test.go internal/httpgen/testdata/json_mapping_composition
git commit -m "test(httpgen): cover json mapping composition matrix"
```

---

## Task 3: Add Central JSON Mapping Collector

**Purpose:** Model per-message JSON mapping once, independent of how individual features used to emit methods.

**Files:**
- Create: `internal/httpgen/json_mapping_context.go`
- Test: `internal/httpgen/json_mapping_context_test.go`
- Modify: no generator switch yet, except adding isolated helper calls if needed for tests

**Interfaces:**
- Produces:
  - `type JSONMappingContext struct`
  - `type JSONMappingFieldTransform struct`
  - `func collectJSONMappingContexts(file *protogen.File) ([]*JSONMappingContext, error)`
  - `func messageNeedsJSONMapping(msg *protogen.Message) bool`
  - `func fieldNeedsNestedJSONDelegation(field *protogen.Field) bool`
- Consumes: existing annotation helpers in `internal/annotations` and existing feature detector functions in `internal/httpgen`.

- [ ] **Step 1: Write collector unit tests**

Create tests covering:
- nullable + timestamp on same message returns one context with two field transforms.
- parent with child bytes_encoding returns a nested delegation field.
- map-value unwrap appears as a field transform.
- root unwrap appears as document transform.
- no annotated fields returns no context.

- [ ] **Step 2: Implement context structs**

Use names like:
```go
type JSONMappingContext struct {
    Message *protogen.Message
    FieldTransforms []*JSONMappingFieldTransform
    NestedDelegationFields []*protogen.Field
    RootUnwrap *RootUnwrapMessage
}

type JSONMappingFieldTransform struct {
    Field *protogen.Field
    Kind JSONMappingTransformKind
}

type JSONMappingTransformKind string
```

Required `Kind` constants:
```go
const (
    TransformInt64Number JSONMappingTransformKind = "int64_number"
    TransformEnumValue JSONMappingTransformKind = "enum_value"
    TransformBytesEncoding JSONMappingTransformKind = "bytes_encoding"
    TransformTimestampFormat JSONMappingTransformKind = "timestamp_format"
    TransformNullable JSONMappingTransformKind = "nullable"
    TransformEmptyBehavior JSONMappingTransformKind = "empty_behavior"
    TransformFlatten JSONMappingTransformKind = "flatten"
    TransformOneofDiscriminator JSONMappingTransformKind = "oneof_config"
    TransformMapValueUnwrap JSONMappingTransformKind = "map_value_unwrap"
)
```

- [ ] **Step 3: Implement collector traversal**

Traversal rules:
- Skip synthetic map-entry messages.
- Recurse into nested messages.
- Add one context per message needing any transform, nested delegation, or root unwrap.
- Reuse existing annotation validation; do not delete conflict checks in this task.
- Determine nested delegation generically: a message field delegates if its message type needs JSON mapping directly or transitively.

- [ ] **Step 4: Run collector tests**

```bash
go test ./internal/httpgen -run 'TestJSONMappingContext|TestJSONMappingCollector' -count=1 -v
```
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/httpgen/json_mapping_context.go internal/httpgen/json_mapping_context_test.go
git commit -m "feat(httpgen): collect composed json mapping contexts"
```

---

## Task 4: Implement Composed Marshal Emitter and Switch Generation

**Purpose:** Emit one `MarshalJSONSebuf` / `MarshalJSON` per mapped message and stop old marshal emitters from generating duplicate methods.

**Files:**
- Create: `internal/httpgen/json_mapping_marshal.go`
- Modify: `internal/httpgen/generator.go`
- Modify: old emitter entry points in `encoding.go`, `enum_field_encoding.go`, `nullable.go`, `empty_behavior.go`, `timestamp_format.go`, `bytes_encoding.go`, `flatten.go`, `oneof_discriminator.go`, `unwrap.go`
- Test: `internal/httpgen/json_mapping_composition_test.go`

**Interfaces:**
- Consumes: `collectJSONMappingContexts` from Task 3.
- Produces:
  - `func (g *Generator) generateJSONMappingFile(file *protogen.File) error`
  - `func (g *Generator) generateJSONMappingMarshalJSON(gf *protogen.GeneratedFile, ctx *JSONMappingContext)`

- [ ] **Step 1: Add `generateJSONMappingFile` without deleting old code**

Generate file name:
```go
filename := file.GeneratedFilenamePrefix + "_json_mapping.pb.go"
```

Imports should be computed conservatively at first, then tightened before commit. Start with:
```go
import (
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "strconv"
    "strings"
    "time"

    "google.golang.org/protobuf/encoding/protojson"
)
```
Remove unused imports once actual emitted code determines needs.

- [ ] **Step 2: Emit marshal skeleton**

Each generated method must follow this shape:
```go
func (x *Msg) MarshalJSONSebuf(opts protojson.MarshalOptions) ([]byte, error) {
    if x == nil {
        return []byte("null"), nil
    }
    data, err := opts.Marshal(x)
    if err != nil {
        return nil, err
    }
    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return nil, err
    }
    // nested delegation
    // field transforms
    // optional root unwrap document transform
    return json.Marshal(raw)
}

func (x *Msg) MarshalJSON() ([]byte, error) {
    return x.MarshalJSONSebuf(protojson.MarshalOptions{})
}
```

- [ ] **Step 3: Move direct field marshal transform emission into the composed emitter**

Reuse the existing single-feature block generators where possible by calling them from the new emitter rather than duplicating logic. Required marshal coverage:
- `generateInt64FieldMarshal`
- enum field mapping logic from `enum_field_encoding.go`
- nullable field logic from `nullable.go`
- empty behavior logic from `empty_behavior.go`
- timestamp format logic from `timestamp_format.go`
- bytes encoding logic from `bytes_encoding.go`
- flatten promotion logic from `flatten.go`
- oneof discriminator logic from `oneof_discriminator.go`
- map-value unwrap logic from `unwrap.go`

- [ ] **Step 4: Emit generic nested delegation before field transforms**

Support:
- singular message fields: if non-nil and child implements `MarshalJSONSebuf`, put child bytes into `raw[jsonName]`.
- repeated message fields: build `[]json.RawMessage` with per-item delegation.
- map fields with message values: build `map[string]json.RawMessage` with per-value delegation when possible.

Do not delegate scalar fields or bytes fields.

- [ ] **Step 5: Switch `generateFile` to use new emitter**

In `internal/httpgen/generator.go`, replace the calls that emit JSON mapping method files with one call:
```go
if err := g.generateJSONMappingFile(file); err != nil {
    return err
}
```

Keep non-method files such as enum lookup map generation if they are still needed by composed transforms.

- [ ] **Step 6: Disable old method-generating entry points**

Old entry points must not emit `MarshalJSONSebuf` / `MarshalJSON` anymore when the composed emitter is enabled. Prefer deleting calls from `generateFile`; leave helper functions temporarily only if the new emitter reuses them.

- [ ] **Step 7: Run marshal-focused tests**

```bash
go test ./internal/httpgen -run 'TestJSONMapping.*Marshal|TestJSONMappingNestedDelegation|TestJSONMappingFeaturePairsGenerateAndBuild' -count=1 -v
```
Expected: marshal and build portions pass or fail only on unmarshal-specific assertions scheduled for Task 5.

- [ ] **Step 8: Commit**

```bash
git add internal/httpgen/json_mapping_marshal.go internal/httpgen/generator.go internal/httpgen/*.go
git commit -m "feat(httpgen): emit composed json marshal methods"
```

---

## Task 5: Implement Composed Unmarshal Emitter

**Purpose:** Mirror marshal composition so clients and servers can parse the custom JSON they emit.

**Files:**
- Create: `internal/httpgen/json_mapping_unmarshal.go`
- Modify: `internal/httpgen/json_mapping_marshal.go` only if shared helpers should move to `json_mapping_emit.go`
- Modify: old unmarshal helper code as needed
- Test: `internal/httpgen/json_mapping_composition_test.go`

**Interfaces:**
- Consumes: `JSONMappingContext` from Task 3 and file generation hook from Task 4.
- Produces:
  - `func (g *Generator) generateJSONMappingUnmarshalJSON(gf *protogen.GeneratedFile, ctx *JSONMappingContext)`

- [ ] **Step 1: Emit unmarshal skeleton**

Each generated method must follow this shape:
```go
func (x *Msg) UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {
    var raw map[string]json.RawMessage
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }
    // root unwrap inverse, if any
    // field transform inverse
    // nested child delegation inverse
    modified, err := json.Marshal(raw)
    if err != nil {
        return err
    }
    return opts.Unmarshal(modified, x)
}

func (x *Msg) UnmarshalJSON(data []byte) error {
    return x.UnmarshalJSONSebuf(data, protojson.UnmarshalOptions{})
}
```

- [ ] **Step 2: Move inverse field transform emission into composed emitter**

Required inverse coverage:
- `int64_encoding=NUMBER`: number(s) to protojson string(s).
- enum custom values: custom string(s) to proto enum name/value expected by protojson.
- bytes custom encoding: hex/base64url/raw to standard base64 bytes accepted by protojson.
- timestamp custom formats: unix seconds/millis/date to RFC3339 or protojson-compatible timestamp.
- nullable: preserve explicit null for optional nullable fields using existing semantics.
- empty_behavior: invert only where existing unmarshal behavior already defines semantics; do not invent lossy behavior.
- flatten: collect prefixed/promoted keys back under the child object before protojson.
- oneof discriminator: convert discriminated union JSON back into protojson oneof shape.
- map-value unwrap: wrap map values back into the annotated wrapper field.

- [ ] **Step 3: Emit nested child unmarshal delegation**

For fields with child custom mapping:
- singular: allocate child if needed, call `UnmarshalJSONSebuf`, then replace `raw[jsonName]` with `opts.Marshal(child)` or `json.Marshal(child)` output that protojson can consume.
- repeated: unmarshal raw array, delegate per item, rebuild array.
- map message values: unmarshal raw object, delegate per value, rebuild object.

- [ ] **Step 4: Run unmarshal tests**

```bash
go test ./internal/httpgen -run 'TestJSONMapping.*Unmarshal|TestJSONMappingComposed' -count=1 -v
```
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/httpgen/json_mapping_unmarshal.go internal/httpgen/*.go
git commit -m "feat(httpgen): emit composed json unmarshal methods"
```

---

## Task 6: Port Root and Map-Value Unwrap Fully into the Composed Pipeline

**Purpose:** Remove unwrap's special hand-rolled method ownership and make unwrap compose with sibling annotations.

**Files:**
- Modify: `internal/httpgen/unwrap.go`
- Modify: `internal/httpgen/json_mapping_context.go`
- Modify: `internal/httpgen/json_mapping_marshal.go`
- Modify: `internal/httpgen/json_mapping_unmarshal.go`
- Test: `internal/httpgen/unwrap_test.go`, `internal/httpgen/json_mapping_composition_test.go`

**Interfaces:**
- Consumes: composed emitter hooks from Tasks 4 and 5.
- Produces: no standalone unwrap-generated `MarshalJSONSebuf` / `UnmarshalJSONSebuf` methods.

- [ ] **Step 1: Split unwrap classification**

Ensure the collector distinguishes:
- `TransformMapValueUnwrap` for fields whose map values are wrapper messages with an unwrap field.
- `RootUnwrap` on the context for messages where `annotations.IsRootUnwrap` is true.

- [ ] **Step 2: Delete or bypass standalone unwrap method generation**

`generateUnwrapFile` must no longer emit method declarations. If non-method unwrap metadata remains useful, keep only helper collection code.

- [ ] **Step 3: Marshal map-value unwrap as a field transform**

Starting from `raw[jsonName]`, replace each map value object with its inner unwrap field JSON.

- [ ] **Step 4: Unmarshal map-value unwrap as an inverse field transform**

Starting from incoming map JSON, wrap each map value into the wrapper object's annotated field JSON before protojson sees it.

- [ ] **Step 5: Marshal root unwrap as the final document transform**

For root map unwrap, return the raw map field value instead of the containing object.
For root repeated unwrap, return the raw repeated field value instead of the containing object.
This must run after nested delegation and field transforms so child/value annotations survive.

- [ ] **Step 6: Unmarshal root unwrap as the first inverse document transform**

Wrap the incoming root map/array JSON under the annotated field's JSON name before applying other inverse transforms and `opts.Unmarshal`.

- [ ] **Step 7: Run unwrap and composition tests**

```bash
go test ./internal/httpgen -run 'Test.*Unwrap|TestJSONMapping.*unwrap|TestJSONMappingComposed' -count=1 -v
```
Expected: pass.

- [ ] **Step 8: Commit**

```bash
git add internal/httpgen/unwrap.go internal/httpgen/json_mapping_*.go internal/httpgen/*test.go
git commit -m "feat(httpgen): compose unwrap with json mapping pipeline"
```

---

## Task 7: Delete Conflict Machinery and Old Emitters

**Purpose:** Finish the architectural migration by removing code that preserved the old one-feature-per-message model.

**Files:**
- Modify/delete as appropriate: `internal/httpgen/encoding.go`, `enum_field_encoding.go`, `nullable.go`, `empty_behavior.go`, `timestamp_format.go`, `bytes_encoding.go`, `flatten.go`, `oneof_discriminator.go`, `unwrap.go`
- Modify tests expecting old conflict errors: `internal/httpgen/validation_test.go`, feature consistency tests
- Test: all httpgen tests

**Interfaces:**
- Consumes: all composed emitter behavior from Tasks 3-6.
- Produces: simpler old feature files containing only reusable helpers/validators, or removes obsolete files if empty.

- [ ] **Step 1: Locate old conflict code**

Run:
```bash
rg -n "conflict|only one MarshalJSON|detectMarshalJSONConflicts|checkInt64WrapperMarshalJSONConflict|collectWrapperContexts|unwrapMsgNames|directEncodingMsgNames" internal/httpgen
```

- [ ] **Step 2: Delete obsolete conflict validation**

Remove checks whose only purpose was preventing duplicate method generation. Keep semantic annotation validation such as invalid flatten targets, invalid oneof discriminator collisions, invalid URL params, invalid enum annotations, etc.

- [ ] **Step 3: Delete obsolete method emitters**

Remove or stop compiling functions that emitted standalone JSON method pairs and are no longer called. If a helper emits only a field transform block used by the composed emitter, keep and rename it to make the smaller responsibility clear.

- [ ] **Step 4: Update tests that asserted old conflicts**

Change old conflict tests into successful generation/build tests, or delete them if Task 2 covers the behavior more directly.

- [ ] **Step 5: Run focused tests**

```bash
go test ./internal/httpgen -count=1
```
Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/httpgen
git commit -m "refactor(httpgen): remove json mapping conflict emitters"
```

---

## Task 8: Golden Files, Full Test Suite, and Cross-Generator Check

**Purpose:** Normalize generated output after behavior is proven and ensure go-client integration still sees httpgen-owned JSON mapping methods correctly.

**Files:**
- Modify generated golden files under `internal/httpgen/testdata` and possibly `internal/clientgen/testdata/golden` only if test outputs intentionally changed.
- Modify: `internal/clientgen/golden_test.go` only if file naming expectations must change from old `_encoding.pb.go`/`_unwrap.pb.go` to `_json_mapping.pb.go`.

**Interfaces:**
- Consumes: completed composed emitter.
- Produces: updated golden fixtures and compatibility with clientgen tests.

- [ ] **Step 1: Run targeted httpgen tests**

```bash
go test ./internal/httpgen -count=1
```
Expected: pass or only golden mismatch failures caused by intentional generated output changes.

- [ ] **Step 2: Update goldens only after targeted tests pass**

Use existing golden update workflow. If the repository uses `UPDATE_GOLDEN=1`, run:
```bash
UPDATE_GOLDEN=1 go test ./internal/httpgen -count=1
```

- [ ] **Step 3: Run clientgen compatibility tests**

```bash
go test ./internal/clientgen -run 'TestCombinedGoHTTPAndGoClientGenerationDoesNotDuplicateJSONMappingFiles|Test.*ForwardCompat' -count=1 -v
```
Expected: pass. If failures refer only to old generated filename expectations, update tests to expect the new httpgen-owned `_json_mapping.pb.go` file.

- [ ] **Step 4: Run broader tests**

```bash
go test ./internal/annotations ./internal/httpgen ./internal/clientgen -count=1
```
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/httpgen internal/clientgen
git commit -m "test: update json mapping generated output"
```

---

## Task 9: Final Verification and Documentation Update

**Purpose:** Prove the branch is ready for review and document the new composition behavior.

**Files:**
- Modify: relevant docs mentioning JSON mapping conflicts, likely `docs/http-generation.md` and/or annotation docs discovered by search.
- Modify: issue notes or plan/spec only if final implementation differs from plan.

**Interfaces:**
- Consumes: all prior tasks.
- Produces: final verified branch ready for PR/review.

- [ ] **Step 1: Find stale docs**

```bash
rg -n "only one MarshalJSON|conflict|int64_encoding|nullable|empty_behavior|timestamp_format|bytes_encoding|enum_value|flatten|oneof_config|unwrap" docs proto internal -g '*.md' -g '*.proto'
```

- [ ] **Step 2: Update docs**

Docs must state:
- JSON-mapping annotations compose within one message.
- Child message custom mappings are preserved when nested under another custom-mapped message.
- Root unwrap is a document transform and runs after field/child transforms.
- Query/path scalar restrictions are unchanged.

- [ ] **Step 3: Run formatting and full test suite**

```bash
go fmt ./...
./scripts/run_tests.sh --fast
```
Expected: pass.

If time and runtime allow, also run:
```bash
./scripts/run_tests.sh
```
Expected: pass with coverage threshold.

- [ ] **Step 4: Final self-review**

Run:
```bash
git diff --stat
rg -n "only one MarshalJSON-generating feature|TODO|TBD|panic\(" internal/httpgen docs/superpowers/specs/2026-08-10-json-mapping-composition-requirements.md docs
```
Expected: no stale conflict text except in historical issue/spec context, and no TODO/TBD placeholders.

- [ ] **Step 5: Commit**

```bash
git add docs internal/httpgen internal/clientgen
git commit -m "docs: describe composed json mapping behavior"
```

---

## Suggested Agent Handoff Order

Use one fresh-context worker per task, sequentially. Each worker should start by reading:

1. `AGENTS.md`
2. `docs/superpowers/plans/2026-08-10-json-mapping-composition.md`
3. The requirements document from Task 1, once it exists
4. The files listed in that task's **Files** section

Recommended sequence:

1. Requirements/spec worker
2. Test-bed worker
3. Collector worker
4. Marshal emitter worker
5. Unmarshal emitter worker
6. Unwrap integration worker
7. Cleanup worker
8. Golden/clientgen worker
9. Final docs/verification worker

Do not parallelize Tasks 3-7. They share the same emitter surface and will conflict. Task 9 can be done only after Task 8 passes.

## Plan Self-Review

- Spec coverage: Task 1 creates the feature requirements and matrix; Task 2 adds the regression net; Tasks 3-6 implement the four-stage composed pipeline; Task 7 removes old conflict machinery; Task 8 updates goldens and clientgen compatibility; Task 9 verifies and documents.
- Placeholder scan: no intentional `TBD`/`TODO` placeholders are present.
- Type consistency: central names are consistently `JSONMappingContext`, `JSONMappingFieldTransform`, `JSONMappingTransformKind`, `collectJSONMappingContexts`, and `generateJSONMappingFile`.
