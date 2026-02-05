# Pitfalls: Protobuf Code Generation, Custom JSON Serialization, and Multi-Language Generators

**Research Date:** 2026-02-05
**Scope:** sebuf toolkit -- 10 JSON mapping annotations across 4 generators, 8 new language client generators
**Audience:** Roadmap/planning for v1.0 (JSON mapping) and v2.0 (multi-language clients)

---

## 1. Cross-Generator Annotation Inconsistency

### The Pitfall

When the same annotation must produce semantically equivalent behavior across multiple generators (go-http, go-client, ts-client, openapiv3), implementations drift. Each generator has its own copy of annotation parsing logic (`hasUnwrapAnnotation`, `getMethodHTTPConfig`, `httpMethodToString`) duplicated across four `internal/*/annotations.go` files. A fix applied to one generator is easily forgotten in others.

**sebuf is already experiencing this.** The codebase contains four independent copies of `hasUnwrapAnnotation` (in `internal/httpgen/annotations.go:286`, `internal/openapiv3/types.go:252`, `internal/tsclientgen/annotations.go:231`, and `internal/tsclientgen/types.go:213`). Similarly, `getMethodHTTPConfig` and `httpMethodToString` are duplicated in all four generator packages. When PR #98 (cross-file unwrap resolution) fixes a bug in `httpgen`, the same fix must be manually replicated to `openapiv3` and `tsclientgen`.

### Warning Signs

- A bug fix merged for one generator that does not touch the other three
- Golden file tests pass for the fixed generator but no corresponding test data exists for the others
- Different return types for the same conceptual function (e.g., httpgen's `getUnwrapField` returns `(*UnwrapFieldInfo, error)` while openapiv3's returns `*protogen.Field`)
- Code review comments like "did you update the other generators too?"

### Prevention Strategy

1. **Extract shared annotation parsing into a common package.** Create `internal/annotations/` (or `internal/shared/`) containing canonical implementations of `hasUnwrapAnnotation`, `getMethodHTTPConfig`, `httpMethodToString`, and all future annotation parsers. Each generator imports and calls these. This is the single most impactful structural change before adding 10 new annotations.

2. **Create a cross-generator conformance test suite.** For each annotation, define a single `.proto` test fixture and verify that all generators produce semantically equivalent output (e.g., the go-http server marshals the same JSON that the ts-client expects to receive, and the OpenAPI spec documents the same schema).

3. **Enforce "all-or-nothing" annotation implementation.** When filing an issue for a new annotation (like #87-#96), require implementation across all 4 generators before closing. Use a checklist in the issue template.

### Phase Mapping

- **Before v1.0 JSON work begins:** Extract shared annotation code. This prevents 10x duplication (10 annotations times 4 generators).
- **During v1.0:** Conformance tests for each annotation.
- **Before v2.0:** The shared annotation package becomes the contract that new language generators must implement against.

### Real Examples

- **grpc-gateway** experienced this exact problem: [Issue #298](https://github.com/grpc-ecosystem/grpc-gateway/issues/298) documented that integration with non-Go languages was "partially broken" because the gateway's JSON marshaling assumptions did not match what other language clients expected.
- **connect-rpc** solved this by defining a formal protocol specification that all language implementations (Go, TypeScript, Swift, Kotlin) must conform to, with a shared conformance test suite (`connectrpc/conformance`).

---

## 2. int64/uint64 String-vs-Number JSON Encoding

### The Pitfall

The proto3 canonical JSON mapping encodes `int64`, `uint64`, `sint64`, `fixed64`, `sfixed64` as **JSON strings** (not numbers), because JavaScript's `Number` type loses precision beyond 2^53. This is one of the most misunderstood aspects of protobuf JSON serialization. sebuf's planned feature #88 (int64/uint64 as string encoding) must handle this correctly, but the real danger is the cascade of consequences.

The TypeScript client (`internal/tsclientgen/types.go:29-31`) already maps 64-bit integers to `string` in TypeScript, which is correct for proto3 JSON. But the Go server (`internal/httpgen/generator.go`) uses `protojson` which encodes them as strings by default. If sebuf introduces a custom annotation to optionally encode int64 as a JSON number (for APIs that know their values fit in 53 bits), the generated Go code must use `encoding/json` instead of `protojson` for that field, and the TypeScript client must switch the type from `string` to `number`. Every generator must agree on when to use which encoding.

### Warning Signs

- TypeScript client receives a string `"123"` but expects a number `123` (or vice versa)
- Go client sends an int64 as a JSON number but the server's protojson unmarshaler rejects it
- OpenAPI spec shows `type: integer` but the actual API returns `type: string` for int64 fields
- Round-trip tests pass within a single language but fail cross-language
- Values above 2^53 silently lose precision in JavaScript

### Prevention Strategy

1. **Default to proto3 canonical behavior** (int64 as string). The custom annotation should be opt-in: `[(sebuf.http.json_int64) = NUMBER]`.

2. **Warn at generation time** if a field uses the number encoding without validation constraints limiting the range to safe integer bounds (`-(2^53 - 1)` to `2^53 - 1`). Combine with buf.validate constraints: `int64 value = 1 [(buf.validate.field).int64 = { gte: -9007199254740991, lte: 9007199254740991 }, (sebuf.http.json_int64) = NUMBER];`

3. **Test with boundary values.** Golden file test fixtures must include values at the boundaries: `0`, `1`, `-1`, `9007199254740991` (MAX_SAFE_INTEGER), `9007199254740992` (first unsafe), `9223372036854775807` (int64 max).

4. **Document the TypeScript type change.** When `json_int64 = NUMBER` is set, the TypeScript type must change from `string` to `number`, and the OpenAPI schema must change from `type: string` to `type: integer, format: int64`.

### Phase Mapping

- **v1.0 (#88):** Implement with opt-in annotation. Default remains proto3 canonical (string). Enforce cross-generator consistency in golden file tests.

### Real Examples

- [protobuf issue #2679](https://github.com/protocolbuffers/protobuf/issues/2679): "Why uint64, int64 Protobuf-Type be serialize as String JSON-Type?" -- this has caused confusion for hundreds of developers.
- [protobuf issue #8331](https://github.com/protocolbuffers/protobuf/issues/8331): Feature request for `int64 as JSON number marshal option`, still debated years later.
- The gRPC-Gateway's `JSONPb` marshaler had to add an `EmitUnpopulated` option because the int64 string encoding interacted poorly with zero-value omission.

---

## 3. Proto3 Field Presence: Null vs. Absent vs. Default

### The Pitfall

Proto3 implicit-presence fields cannot distinguish between "field was not set" and "field was explicitly set to the zero value." This is sebuf's issue #87 (nullable primitives). The pitfall is that different generators may handle this differently, leading to data loss or semantic errors during round-trips.

Consider a `PATCH /users/{id}` endpoint with `optional string name = 1`. The Go server must distinguish:
- Field absent from JSON (`name` key missing) -> do not update
- Field set to `""` -> clear the name
- Field set to `null` -> clear the name (different from absent!)

Proto3 `optional` keyword gives explicit presence (has/clear semantics), but sebuf currently maps this to TypeScript `?` optional property (`internal/tsclientgen/types.go:298`), which conflates null and absent. The Go client and server have different behavior: Go uses pointer types for optional fields, while TypeScript uses `undefined`.

### Warning Signs

- PATCH endpoints silently overwrite fields with zero values when the client omits them
- TypeScript `undefined` and `null` produce different JSON but the server treats them the same
- OpenAPI schema lacks `nullable: true` for optional fields (already noted in `internal/openapiv3/types.go:42-44`)
- Tests only cover "field present with value" but not "field absent" and "field null"

### Prevention Strategy

1. **Define three-state semantics explicitly in the annotation.** For issue #87, create an annotation like `[(sebuf.http.nullable) = true]` that maps to:
   - Go: `*string` (nil = absent, pointer to "" = null/cleared)
   - TypeScript: `string | null | undefined` (undefined = absent, null = explicit null)
   - OpenAPI: `nullable: true` with explicit documentation
   - JSON: omitted key = absent; `"name": null` = null; `"name": ""` = empty string

2. **Add PATCH-specific test fixtures.** Test all three states (absent, null, zero) for every scalar type. Verify round-trip: client sends partial JSON -> server receives correct has/clear state -> server responds -> client parses correctly.

3. **Review the comment in `internal/openapiv3/types.go:42-44`.** It says "For proto3 optional fields, we could add nullable: true but OpenAPI 3.1 handles this differently than 3.0." This is a known gap -- OpenAPI 3.1 uses JSON Schema's `type: ["string", "null"]` instead of `nullable: true`.

### Phase Mapping

- **v1.0 (#87):** Implement with explicit three-state semantics. This blocks PATCH endpoints from being reliable.

### Real Examples

- [protobuf field_presence.md](https://github.com/protocolbuffers/protobuf/blob/main/docs/field_presence.md): Official documentation acknowledging the complexity.
- The `google.protobuf.FieldMask` approach used by Google APIs avoids this by explicitly listing which fields to update, but it adds API complexity that sebuf users may not want.

---

## 4. Enum String Encoding and Zero Value Semantics

### The Pitfall

Issue #89 (enum string encoding with custom values) intersects with proto3's requirement that enum zero value must be `_UNSPECIFIED`. Proto3 JSON encodes enums as their **name strings** (e.g., `"ACTIVE"` not `0`). But several edge cases break cross-language consistency:

- **Unknown enum values:** Go's protojson silently preserves unknown enum values as integers. TypeScript and other clients may not.
- **Enum aliases:** `allow_alias = true` creates multiple names for the same numeric value. JSON serialization picks one name, but which one varies by language.
- **Enum renaming:** Renaming an enum value is a breaking change for JSON serialization (but not for binary protobuf). This is subtle and easy to miss.
- **Custom string values:** If sebuf allows `[(sebuf.http.enum_value) = "active"]` to map `STATUS_ACTIVE` to `"active"` in JSON, every generator must apply this mapping during both serialization and deserialization.

The TypeScript generator (`internal/tsclientgen/types.go:256-273`) currently emits enum values as string union types using the proto name. The OpenAPI generator (`internal/openapiv3/types.go:142-170`) documents them the same way. But neither handles the case where a custom JSON value differs from the proto name.

### Warning Signs

- TypeScript client sends `"active"` but Go server expects `"STATUS_ACTIVE"`
- Adding a new enum value causes JSON deserialization failures in clients that haven't been regenerated
- OpenAPI spec shows proto enum names but API returns custom JSON values
- Default zero value (`_UNSPECIFIED`) appears in API responses when the field was not set, confusing API consumers

### Prevention Strategy

1. **Make custom enum values a mapping table, not per-value annotations.** Define a message-level annotation that maps all values at once: `option (sebuf.http.enum_json) = { values: [{ proto: "STATUS_ACTIVE", json: "active" }] }`. This ensures completeness -- missing mappings are caught at generation time.

2. **Validate completeness at generation time.** If custom enum JSON is configured, every non-UNSPECIFIED value must have a mapping. Missing mappings should be a generation-time error.

3. **Test unknown values.** Send a JSON string that does not match any known enum value. Verify each generator's behavior: Go protojson returns an error; generated custom code should too.

4. **Document the breaking change risk.** Renaming enum values or adding aliases with JSON serialization active is a wire-format breaking change.

### Phase Mapping

- **v1.0 (#89):** Implement with completeness validation. Cross-generator golden file tests.

### Real Examples

- [golang/protobuf issue #636](https://github.com/golang/protobuf/issues/636): "jsonpb: forward and backward compatible enums" -- debate about how to handle unknown enum values in JSON.
- [protobuf issue #6355](https://github.com/protocolbuffers/protobuf/issues/6355): "jsonpb: allow invalid enums to be unmarshalled to zero value int32" -- demonstrates that different languages handle unknown enum values differently.

---

## 5. Oneof as Discriminated Union: The Flattening Trap

### The Pitfall

Issue #90 (oneof as discriminated union) is architecturally the most complex JSON mapping feature. Proto3 oneof fields serialize as the set field's JSON name. But sebuf wants to support **flattened discriminated unions** with a `type` field. This creates a fundamental tension with protobuf's type system.

Example proto:
```protobuf
oneof content {
  TextContent text = 1;
  ImageContent image = 2;
}
```

Proto3 canonical JSON: `{"text": {"body": "hello"}}` (the field name IS the discriminator)
Discriminated union JSON: `{"type": "text", "body": "hello"}` (flattened, explicit discriminator)

The flattening approach requires:
1. **Go server:** Custom `MarshalJSON`/`UnmarshalJSON` that reads the `type` field first, then deserializes the remaining fields into the correct type. This is exactly the pattern used in `internal/httpgen/unwrap.go` but significantly more complex because the discriminated fields can have overlapping JSON names.
2. **TypeScript client:** A TypeScript discriminated union type: `type Content = { type: "text"; body: string } | { type: "image"; url: string }`. The client must use a type guard.
3. **OpenAPI spec:** `oneOf` with `discriminator.propertyName: "type"` and `discriminator.mapping`.
4. **Go client:** Same custom JSON as the server.

The trap is that overlapping field names between oneof variants make flattening ambiguous. If both `TextContent` and `ImageContent` have a `title` field, the flattened JSON `{"type": "text", "title": "hello"}` works, but `{"type": "image", "title": "photo"}` also works. But what if they have different types for `title`?

### Warning Signs

- Compilation succeeds but runtime JSON unmarshaling picks the wrong oneof variant
- TypeScript union type is too narrow (missing fields) or too wide (allows invalid combinations)
- OpenAPI discriminator mapping doesn't match actual generated JSON
- Performance regression from reflection-based type checking in generated unmarshal code

### Prevention Strategy

1. **Validate at generation time** that flattened oneof variants have no conflicting field names (same JSON name, different types). Emit a clear error.

2. **Start with the non-flattened (canonical proto3) approach** for v1.0. The flattened discriminated union should be opt-in via annotation. The canonical approach already works because protojson handles it.

3. **Study how connect-rpc handles this.** Connect does not support flattened oneofs -- it uses proto3 canonical JSON, which is simpler and less error-prone.

4. **Generate a type guard function in TypeScript** rather than relying on developers to write runtime checks. The generated code should include `isTextContent(content: Content): content is TextContent & { type: "text" }`.

### Phase Mapping

- **v1.0 (#90):** Implement opt-in annotation. Validate field name conflicts at generation time. Support canonical (non-flattened) by default.

### Real Examples

- **protobuf.js** [issue #839](https://github.com/protobufjs/protobuf.js/issues/839): Well-known types support (Struct, Value) -- demonstrates how oneof-like dynamic types cause major implementation complexity in JavaScript.
- **OpenAPI 3.1 discriminator** semantics differ from proto3 oneof semantics, leading to documentation that doesn't match runtime behavior.

---

## 6. Golden File Test Explosion and Maintenance Burden

### The Pitfall

Golden file tests are sebuf's primary regression detection mechanism. Currently there are golden files for 4 generators, each with multiple test proto fixtures. Adding 10 JSON mapping annotations means each annotation needs test coverage across all 4 generators. With 8 new language generators in v2.0, the golden file count explodes:

- **Current:** ~4 proto fixtures x 4 generators x ~3 output files = ~48 golden files
- **After v1.0:** ~14 proto fixtures x 4 generators x ~3 output files = ~168 golden files
- **After v2.0:** ~14 proto fixtures x 12 generators x ~3 output files = ~504 golden files

The maintenance cost is not just file count -- it is **churn**. Any change to the code generation template (even whitespace or comment changes) requires updating ALL golden files for that generator. The `UPDATE_GOLDEN=1` mechanism updates them, but reviewing 50+ changed golden files in a PR makes code review nearly impossible.

### Warning Signs

- PRs that change 100+ golden files with a one-line generator change
- Reviewers rubber-stamping golden file changes without reading them
- Golden file updates failing because the test environment has a different protoc version
- Contributors submitting PRs without running golden file updates (CI catches it, but wastes time)

### Prevention Strategy

1. **Separate structural golden files from content golden files.** Use golden files only for structural/interface verification (does the generated code compile? does it have the right methods?). Use unit tests for content verification (does the int64 field serialize as a string?).

2. **Use template-based golden files** where possible. Instead of storing the entire generated file, store a template with placeholders for version-dependent parts (e.g., protoc version comments, import paths).

3. **Organize golden files by feature, not by generator.** Instead of `internal/httpgen/testdata/golden/unwrap_http.pb.go`, use `testdata/features/unwrap/go-http.golden`, `testdata/features/unwrap/ts-client.golden`, `testdata/features/unwrap/openapi.golden`. This makes it clear which files a feature change should affect.

4. **Add a golden file diff summary to CI output.** When golden files change, CI should print a human-readable summary of what changed (new methods added, type changes, etc.), not just "files differ."

5. **For v2.0 language generators**, consider parameterized golden file tests where the same proto fixture is run through all generators and each output is compared independently.

### Phase Mapping

- **Before v1.0:** Restructure golden file organization. Add diff summary tooling.
- **During v1.0:** One golden file fixture per annotation, covering all 4 generators.
- **Before v2.0:** Design parameterized golden file framework for N generators.

### Real Examples

- **Protobuf-ES** (Buf's TypeScript runtime) restructured from per-message golden files to per-feature test suites specifically to manage golden file explosion as they expanded coverage.
- **python-betterproto** uses a combination of golden files for structural tests and runtime assertion tests for serialization behavior, avoiding the explosion problem.

---

## 7. Well-Known Type Special Casing

### The Pitfall

Protobuf well-known types (`google.protobuf.Timestamp`, `google.protobuf.Duration`, `google.protobuf.Struct`, `google.protobuf.Value`, `google.protobuf.Any`, wrapper types) have **special JSON serialization rules** that differ from regular messages. For example:

- `Timestamp` serializes as an RFC 3339 string, not as `{"seconds": 123, "nanos": 456}`
- `Duration` serializes as `"3.5s"`, not as `{"seconds": 3, "nanos": 500000000}`
- `Struct` serializes as a plain JSON object, not as `{"fields": {...}}`
- `Value` serializes as the contained JSON value directly
- Wrapper types (`StringValue`, `Int32Value`) serialize as the unwrapped value or `null`

Issue #92 (multiple timestamp formats) touches this directly. If sebuf generates custom JSON marshal/unmarshal code, it must replicate or call protojson's well-known type handling. Failing to do so means that a message containing a `Timestamp` field will serialize incorrectly when processed through sebuf's custom marshal code but correctly through standard protojson.

### Warning Signs

- Timestamps appear as `{"seconds":1234567890,"nanos":0}` instead of `"2009-02-13T23:31:30Z"`
- `google.protobuf.Struct` fields break TypeScript interface generation (they're dynamic, not typed)
- Custom marshal/unmarshal code works for regular messages but panics on well-known types
- OpenAPI schema for a Timestamp field shows the message structure instead of `type: string, format: date-time`

### Prevention Strategy

1. **Delegate to protojson for well-known types.** Generated custom MarshalJSON code should detect well-known types and call `protojson.Marshal()` for those fields, not attempt custom handling.

2. **Detect well-known type usage in TypeScript.** Map `google.protobuf.Timestamp` to `string` (ISO 8601) in TypeScript interfaces, not to a `Timestamp` interface with `seconds` and `nanos` fields.

3. **Add well-known type test fixtures.** Create a test proto that uses every well-known type as both a field and a repeated field, and verify all 4 generators handle them correctly.

4. **For issue #92 (timestamp formats):** The annotation should only affect fields that are `google.protobuf.Timestamp`. Applying a timestamp format annotation to a `string` field should be a generation-time error.

### Phase Mapping

- **v1.0 (#92):** Implement timestamp format annotation with well-known type detection. Add well-known type test fixtures.
- **v2.0:** Each new language generator must handle well-known types correctly from the start.

### Real Examples

- [protobuf issue #4549](https://github.com/protocolbuffers/protobuf/issues/4549): "Ruby JSON serialization is incompatible when using well known types" -- Ruby's protobuf library encoded Timestamps as objects instead of strings.
- **prost-wkt** (Rust) exists solely to add well-known type JSON serialization to Rust's prost library, because the base library does not handle it: [fdeantoni/prost-wkt](https://github.com/fdeantoni/prost-wkt).

---

## 8. Cross-File Type Resolution in Custom Annotations

### The Pitfall

PR #98 (cross-file unwrap resolution) exposed a fundamental issue: when a message in file A references a message in file B that has a `sebuf.http.unwrap` annotation, the generator processing file A must be able to "see" the annotation on file B's message. This is especially tricky because protoc plugins receive `FileDescriptorProto` objects for all transitive imports, but the plugin only generates output for explicitly requested files.

The silent error suppression in `internal/httpgen/unwrap.go:87-90` (documented in CONCERNS.md) is a direct symptom of this. When `getUnwrapField()` fails for a cross-file message, the error is swallowed and the unwrap behavior silently degrades.

Every new JSON mapping annotation (#87-#96) will face this same problem: the annotation is on a message definition in one file, but the message is used (as a field type, map value, or method parameter) in another file. The generator must resolve annotations across file boundaries.

### Warning Signs

- Annotations work when the annotated message is in the same file as the service, but fail when in a different file
- Silent degradation: no error, but the JSON serialization falls back to default behavior
- `collectUnwrapFieldsRecursive` (or future annotation collectors) only traverse messages in the current file, missing imported messages
- Test fixtures all use single-file protos, hiding cross-file issues

### Prevention Strategy

1. **Never suppress annotation resolution errors.** Replace the `continue` at `internal/httpgen/unwrap.go:87-90` with proper error reporting. At minimum, log a warning. Ideally, return an error that causes generation to fail.

2. **Add multi-file test fixtures.** For every annotation, create a test case where the annotated message is in a separate imported file. This catches cross-file resolution bugs before release.

3. **Build annotation resolution as a preprocessing pass.** Before any generator processes its files, run a pass that collects all annotations from all files (including non-generated imports). Store the results in a map keyed by message full name. This is how protoc itself resolves type names -- it processes all files in topological order.

4. **Use `protoreflect.FileDescriptor.FullName()` for annotation lookup**, not file-scoped iteration. The CodeGeneratorRequest provides `FileDescriptorProto` for all transitive dependencies, so annotations are always available -- the generator just needs to look in the right place.

### Phase Mapping

- **Before v1.0:** Fix error suppression. Add multi-file test fixtures. This blocks PR #98 merge.
- **During v1.0:** Implement preprocessing pass for annotation collection.

### Real Examples

- **grpc-gateway** had the same issue with custom options on imported messages. Their solution was to process all files, not just the requested ones, during the annotation collection phase.
- **protoc-gen-validate** (PGV, predecessor to protovalidate) had cross-file validation rule resolution bugs that were only caught when users started using multi-file proto organizations.

---

## 9. Multi-Language Generator Idiom Mismatch

### The Pitfall

v2.0 plans 8 new language generators: Python, Rust, Swift, Kotlin, Java, C#, Ruby, Dart. Each language has fundamentally different idioms for the patterns sebuf generates:

| Pattern | Go | TypeScript | Python | Rust | Swift | Kotlin |
|---------|-----|-----------|--------|------|-------|--------|
| Optional | `*T` pointer | `T \| undefined` | `Optional[T]` | `Option<T>` | `T?` | `T?` |
| Error handling | `(T, error)` | `throw/catch` | `raise/except` | `Result<T, E>` | `throw/catch` | `throw/catch` |
| Enum | `iota` const | string union | `Enum` class | `enum` | `enum` | `enum class` |
| Functional options | `With*()` funcs | options object | `**kwargs` | builder pattern | builder/trailing closure | builder/DSL |
| HTTP client | `*http.Client` | `fetch` | `requests`/`httpx` | `reqwest` | `URLSession` | `OkHttp`/`Ktor` |
| JSON marshal | `encoding/json` | built-in | `json` module | `serde` | `Codable` | `kotlinx.serialization` |

The pitfall is trying to force one language's idioms onto another. The Go client uses the functional options pattern (`WithUserServiceHTTPClient()`, `WithUserServiceAPIKey()`), which is idiomatic in Go but awkward in Python or Kotlin. If each generator author makes independent idiom choices, the APIs become inconsistent in "feel" even if they're consistent in behavior.

### Warning Signs

- A Swift generator PR (like #72) that uses Go-style patterns instead of Swift idioms
- API surface documentation that describes Go patterns but links to a Python client that uses different patterns
- Community contributors who are experts in their language but unfamiliar with protobuf serialization edge cases
- Generated code that compiles/runs but feels "foreign" to native developers of that language

### Prevention Strategy

1. **Define a language-agnostic "client contract" document.** Specify what every client must do (methods, error types, configuration, serialization behavior) without specifying HOW. Each language adapter implements the contract idiomatically.

2. **Require a "language expert review"** for each new generator. The Go authors of sebuf should not be the sole reviewers of a Swift or Kotlin client generator. Recruit at least one reviewer who is expert in the target language.

3. **Study connect-rpc's multi-language approach.** Connect has Go, TypeScript, Swift, and Kotlin clients. Each follows language idioms (Swift uses `async/await`, Kotlin uses coroutines) while maintaining behavioral consistency. Their [connect-swift](https://github.com/connectrpc/connect-swift) and [connect-kotlin](https://github.com/connectrpc/connect-kotlin) repos are excellent references.

4. **Start with two languages that are maximally different.** Python (dynamic, duck-typed) and Rust (static, ownership model) will surface the widest range of idiom challenges. If the client contract works for both, it will work for the others.

### Phase Mapping

- **Before v2.0:** Write the language-agnostic client contract. Study connect-rpc implementations.
- **v2.0 phase 1:** Python + Rust (maximally different). Validate the contract.
- **v2.0 phase 2:** Swift + Kotlin (mobile). Leverage connect-rpc patterns.
- **v2.0 phase 3:** Java + C# + Ruby + Dart (fill remaining).

### Real Examples

- **GoGo Protobuf** (deprecated in favor of google.golang.org/protobuf) tried to add Go-specific optimizations to protobuf serialization, and the result was API incompatibility with the canonical implementation. [Post-mortem](https://jbrandhorst.com/post/gogoproto/).
- **connect-swift** initially used a builder pattern for configuration but switched to trailing closures after Swift community feedback, demonstrating the importance of language-expert review.

---

## 10. Custom MarshalJSON/UnmarshalJSON Conflicts with protojson

### The Pitfall

sebuf's unwrap feature already generates custom `MarshalJSON` and `UnmarshalJSON` methods on protobuf messages (`internal/httpgen/unwrap.go`, 902 lines). The v1.0 JSON mapping features (#87-#96) will require generating more custom JSON methods. But proto Go structs already have `protojson.Marshal`/`protojson.Unmarshal` which handle the canonical proto3 JSON mapping.

The conflict: if a message has both a custom `MarshalJSON` (for unwrap or field name casing) and is also passed to `protojson.Marshal` (for standard proto serialization), the behavior depends on which serializer is called. `encoding/json.Marshal()` will use the custom `MarshalJSON`, but `protojson.Marshal()` will ignore it. If the generated HTTP handler uses `protojson` for response serialization but the custom `MarshalJSON` changes the field names, the two paths produce different JSON.

The current unwrap code generates `MarshalJSON` on wrapper messages. If the HTTP handler serializes the response using `protojson.Marshal(response)`, the unwrap behavior is bypassed. The handler must use `json.Marshal(response)` instead. This requirement is implicit and fragile.

### Warning Signs

- Response JSON has proto-style field names (`field_name`) in some code paths and custom names (`fieldName`) in others
- Unwrap works when using `json.Marshal` but fails when using `protojson.Marshal`
- Generated code that calls `protojson.Marshal` on a message that has custom `MarshalJSON` -- the custom method is silently ignored
- Tests pass because they test `MarshalJSON` directly but the HTTP handler uses `protojson`

### Prevention Strategy

1. **Decide once: `json.Marshal` or `protojson.Marshal`.** The generated HTTP handler and client should consistently use one serialization path. If custom JSON behavior is needed, the handler MUST use `encoding/json` (which calls `MarshalJSON`). Document this prominently.

2. **Generate a `MarshalHTTPJSON` method** that wraps the correct serialization behavior. The handler calls this method instead of `json.Marshal` or `protojson.Marshal`. This decouples the generated code from the choice of JSON library.

3. **Test the actual handler serialization path.** Don't just test `MarshalJSON` in isolation -- test the full path: create a handler, send a request, read the response, verify the JSON. This catches protojson/encoding-json conflicts.

4. **Add a compile-time check.** Generate a test file that ensures the custom `MarshalJSON` is actually called during handler response serialization. Something like: `var _ json.Marshaler = (*MyMessage)(nil)` to verify the interface is satisfied.

### Phase Mapping

- **Before v1.0:** Audit the HTTP handler's serialization path. Ensure it uses `encoding/json` when custom JSON methods exist.
- **During v1.0:** Generate `MarshalHTTPJSON` wrapper for consistent serialization.

### Real Examples

- The gRPC-Gateway's `JSONPb` marshaler had to choose between `protojson` and `encoding/json` and explicitly documents that [custom MarshalJSON is not supported](https://pkg.go.dev/github.com/grpc-ecosystem/grpc-gateway/v2) when using protojson mode.

---

## 11. Recursive Message and Circular Reference Handling

### The Pitfall

Protobuf allows recursive message definitions:
```protobuf
message TreeNode {
  string value = 1;
  repeated TreeNode children = 2;
}
```

The CONCERNS.md already identifies this (`internal/httpgen/unwrap.go` recursive processing without cycle detection). But the problem extends beyond unwrap: every generator that walks message hierarchies (TypeScript interface generation in `internal/tsclientgen/types.go:82`, OpenAPI schema generation in `internal/openapiv3/types.go`, Go type generation) must handle cycles.

For JSON mapping features like #96 (nested message flattening), recursive messages make flattening impossible -- you cannot flatten an infinitely deep structure. For #93 (empty object handling), recursive messages can be empty at any depth.

### Warning Signs

- Stack overflow during code generation with deeply nested or recursive messages
- Generated TypeScript interfaces that reference themselves without proper handling
- OpenAPI `$ref` cycles that crash documentation renderers
- Infinite loops in annotation collection passes

### Prevention Strategy

1. **Add a `visited map[string]bool` to all recursive message walkers.** This is called out in CONCERNS.md but not yet implemented. Do it before adding more recursive walkers for new annotations.

2. **Emit a generation-time error for annotations that conflict with recursion.** If `[(sebuf.http.flatten) = true]` is applied to a field whose type is recursive, fail with: "cannot flatten recursive message type TreeNode."

3. **Use protobuf's full name as the cycle detection key.** `msg.Desc.FullName()` is globally unique and handles cross-file references.

4. **Test with recursive messages.** Add `TreeNode`-style messages to test fixtures for every generator.

### Phase Mapping

- **Before v1.0:** Add cycle detection to all recursive walkers. Add recursive message test fixtures.
- **v1.0 (#96):** Validate flattening annotations against recursive types.

### Real Examples

- **OpenAPI Generator** (the popular multi-language codegen tool for OpenAPI specs) had to add explicit cycle detection after users reported stack overflows with recursive schemas. Their fix was a `visited` set.

---

## 12. Backward Compatibility and Annotation Evolution

### The Pitfall

sebuf's annotations are proto extensions with specific field numbers (e.g., `unwrap = 50009`, `query = 50008`). Once released, these field numbers cannot change. The planned 10 JSON mapping annotations will each need a field number. If the annotation design changes after release (e.g., `json_int64` needs to become a message with sub-options instead of a simple enum), the old field number is burned and a new one is needed, or backward compatibility is broken.

Additionally, each new annotation must be backward-compatible with existing proto files that don't use it. The generators must handle the absence of new annotations gracefully (fall back to default behavior). This is complicated by the fact that proto extension defaults are language-dependent -- in Go, a missing bool extension defaults to `false`, but a missing message extension defaults to `nil`.

### Warning Signs

- An annotation that seemed like a boolean but needed to become a message with options
- Old proto files that worked before v1.0 fail after upgrading because a new annotation changes default behavior
- Generated code that checks `ext == nil` but the extension returns a zero-value struct instead of nil
- Extension field number conflicts with other proto ecosystems using the same range

### Prevention Strategy

1. **Use message types for all new annotations, even if they currently have a single field.** `[(sebuf.http.json_int64) = { encoding: NUMBER }]` instead of `[(sebuf.http.json_int64) = NUMBER]`. Messages can be extended without breaking changes; enums and scalars cannot.

2. **Reserve a field number range.** The current annotations use 50003-50009. Reserve 50010-50099 for v1.0 JSON mapping annotations. Document which numbers are assigned and which are reserved.

3. **Test backward compatibility explicitly.** Keep a "v0.x" proto file in test fixtures that uses only pre-v1.0 annotations. Run it through every generator on every release to verify it still works.

4. **Default to no-op for missing annotations.** Every annotation parser must handle the nil/zero-value case and produce the same behavior as before the annotation existed.

### Phase Mapping

- **Before v1.0:** Reserve field number range. Design all 10 annotations as messages. Review annotation design with community (RFC period).
- **v1.0:** Ship annotations. Commit to backward compatibility.
- **v1.1+:** Extend annotations by adding fields to existing messages (non-breaking).

### Real Examples

- **Google's API design guide** requires that all extensions use message types for this exact reason. Google APIs have been stable for 10+ years because of this discipline.
- **buf.validate** (protovalidate) uses message-typed extensions exclusively, allowing them to add validation rules without breaking existing proto files.

---

## 13. Generated Code Import and Dependency Management

### The Pitfall

Each new language generator produces code that depends on runtime libraries (HTTP clients, JSON serializers, error types). If the generated code imports a specific version of a library that conflicts with the user's project, the generator becomes unusable. This is especially problematic for:

- **Go:** The `sebuf/http` runtime package is a dependency. If the user's `go.mod` pins a different version, compilation fails.
- **TypeScript:** Generated code currently has zero dependencies (pure fetch-based). New features may require polyfills or utility libraries.
- **Python:** The generated client will need `requests` or `httpx`. Which one? What version constraint?
- **Rust:** Generated code will need `reqwest` and `serde`. Version conflicts with the user's `Cargo.toml` are common.

The current Go generators use `protogen.GeneratedFile.QualifiedGoIdent()` for import management (`internal/httpgen/generator.go`), which handles Go imports correctly. But TypeScript and future language generators do not have equivalent tooling -- imports are generated as raw strings.

### Warning Signs

- Users report "version conflict" errors when adding sebuf-generated code to their project
- Generated TypeScript code uses a `fetch` API that doesn't exist in Node.js without a polyfill
- Python generated code imports `requests` but user's project uses `httpx` (or vice versa)
- Generated code assumes a specific directory structure that doesn't match the user's project

### Prevention Strategy

1. **Minimize runtime dependencies.** Generated code should depend on the language's standard library as much as possible. For TypeScript, use `fetch` (available everywhere since Node 18). For Python, use `urllib` standard library with an option to use `requests`.

2. **Make the HTTP client injectable.** The Go client already does this with `WithUserServiceHTTPClient()`. Every language generator should follow this pattern -- the generated code accepts an HTTP client interface, not a concrete implementation.

3. **Version-pin runtime libraries in the generated code header.** Include a comment like `// Requires: sebuf-runtime >= 1.0.0, < 2.0.0` so users know what version to install.

4. **Test generated code in isolation.** Create a CI job that generates code, drops it into a fresh project (with only standard library dependencies), and verifies it compiles and runs.

### Phase Mapping

- **v2.0:** Design runtime interface for each language before writing generators. Test in fresh project environments.

### Real Examples

- **grpc-gateway** generates code that depends on `runtime.MarshalerForRequest()` from its own runtime package. Version mismatches between the generator and runtime are a common source of issues.
- **connect-rpc** keeps its runtime package minimal and stable, with a clear stability promise.

---

## 14. Community Contributor Generator Quality Control

### The Pitfall

Community contributors are starting to submit PRs (Swift generator draft #72). When 8 new language generators are open for contribution, each contributor brings deep knowledge of their language but limited knowledge of protobuf serialization edge cases. A Swift expert may not know about proto3 field presence semantics, int64 string encoding, or well-known type special casing.

The result is generators that work for simple cases (basic CRUD, scalar fields) but break on edge cases (maps of maps with unwrap, recursive messages, int64 boundary values, enum aliases). These bugs are discovered by users, not during review.

### Warning Signs

- A new generator PR that has no test cases for well-known types, int64, or optional fields
- Generated code that uses language-native JSON serialization without proto3-specific handling
- PR reviewers (Go experts) approving code in a language they don't know well
- Generated code that works with the example protos but fails with the user's complex real-world protos

### Prevention Strategy

1. **Create a "generator conformance checklist"** that every new generator PR must satisfy:
   - [ ] Scalar type mapping for all 15 proto kinds
   - [ ] Optional/presence handling (explicit presence with `optional` keyword)
   - [ ] int64/uint64 string encoding in JSON
   - [ ] Enum string encoding
   - [ ] Map field handling (string keys, message values)
   - [ ] Repeated field handling
   - [ ] Nested message handling
   - [ ] Well-known type handling (Timestamp, Duration, Struct, Value)
   - [ ] Unwrap annotation support
   - [ ] Error type generation (ValidationError, ApiError)
   - [ ] Cross-file type resolution
   - [ ] Recursive message handling

2. **Provide a canonical test proto** that exercises ALL of the above. Every generator must produce output for this proto that passes its golden file test. This proto should be `internal/shared/testdata/conformance.proto`.

3. **Require runnable example code** with each generator PR. Not just generated code, but a working client that calls a test server and verifies responses.

4. **Pair community contributors with core maintainers** for review. The contributor handles language idioms; the maintainer handles protobuf correctness.

### Phase Mapping

- **Before v2.0:** Create conformance checklist and canonical test proto. Document contribution guide for new generators.
- **v2.0:** Enforce checklist for all PRs. Pair review process.

### Real Examples

- **OpenAPI Generator** has 40+ language generators, many contributed by the community. Quality varies significantly, with some generators missing enum support or having broken nullable handling. They now use a "feature matrix" to track which features each generator supports.
- **gRPC** itself requires a conformance test suite for each new language implementation.

---

## 15. Field Name Casing Inconsistency Between Serialization Layers

### The Pitfall

Issue #94 (field name casing options) touches the most pervasive aspect of JSON serialization: field names. Proto3 defines a canonical JSON mapping where field names use `lowerCamelCase` (the `json_name` option in the field descriptor). But REST APIs commonly use `snake_case`, and some use `PascalCase`.

The danger is that sebuf has multiple serialization layers, and each must agree on the field name:

1. **Proto descriptor:** `json_name` (e.g., `userId` for field `user_id`)
2. **Go struct tag:** `json:"userId"` (generated by protoc-gen-go)
3. **Custom MarshalJSON:** Must use the same name
4. **TypeScript interface:** Property name derived from `json_name`
5. **OpenAPI schema:** Property name from `json_name`
6. **HTTP query parameters:** Often `snake_case` regardless of JSON casing
7. **Path parameters:** From URL pattern, often `snake_case`

If sebuf allows overriding the JSON field name, it creates a divergence between the proto `json_name` and the actual serialized name. The Go struct tag (generated by Google's protoc-gen-go, not by sebuf) will still use the proto `json_name`, so `encoding/json.Marshal` will use one name while sebuf's custom code uses another.

### Warning Signs

- Query parameter named `page_size` but JSON body field named `pageSize`
- TypeScript client sends `userId` but server expects `user_id` (or vice versa)
- OpenAPI spec shows one casing but the actual API uses another
- Protojson and encoding/json produce different field names for the same message

### Prevention Strategy

1. **Use proto's `json_name` as the authoritative source.** If the user wants `snake_case` JSON, they should set `json_name` in the proto definition: `string user_id = 1 [json_name = "user_id"];`. This ensures all serialization layers agree.

2. **If file-level casing override is implemented (#94)**, it must override `json_name` at generation time, not at runtime. The TypeScript interface, OpenAPI schema, and custom JSON code must all use the overridden name.

3. **Never mix casing within a single message.** If a file-level annotation sets `snake_case`, ALL fields in that file must use `snake_case`. Per-field overrides should be the exception, not the rule.

4. **Test round-trips between Go server and TypeScript client** with custom casing. Verify that the Go server can unmarshal what the TypeScript client sends, and vice versa.

### Phase Mapping

- **v1.0 (#94):** Implement file-level casing with per-field override. Test cross-generator round-trips.

### Real Examples

- **gRPC-Gateway** allows a `json_name` option override but warns that it can cause inconsistencies with direct protojson usage.
- **OpenAPI Generator** has separate `naming convention` options for models and properties, which frequently cause mismatches.

---

## Summary: Priority Matrix

| Pitfall | Severity | Likelihood | Phase | Effort |
|---------|----------|------------|-------|--------|
| 1. Cross-generator annotation inconsistency | Critical | Already happening | Before v1.0 | Medium |
| 2. int64/uint64 string-vs-number | High | High | v1.0 (#88) | Medium |
| 3. Null vs absent vs default | Critical | High | v1.0 (#87) | High |
| 4. Enum string encoding | High | Medium | v1.0 (#89) | Medium |
| 5. Oneof discriminated union | High | Medium | v1.0 (#90) | High |
| 6. Golden file explosion | Medium | Certain | Before v1.0 | Medium |
| 7. Well-known type special casing | High | High | v1.0 (#92) | Medium |
| 8. Cross-file type resolution | Critical | Already happening | Before v1.0 | Medium |
| 9. Multi-language idiom mismatch | High | High | Before v2.0 | Low (planning) |
| 10. MarshalJSON vs protojson conflict | High | Medium | Before v1.0 | Medium |
| 11. Recursive message handling | Medium | Medium | Before v1.0 | Low |
| 12. Backward compatibility | High | Medium | Before v1.0 | Low (design) |
| 13. Generated code dependencies | Medium | High | v2.0 | Medium |
| 14. Community contributor quality | Medium | Certain | Before v2.0 | Low (process) |
| 15. Field name casing inconsistency | High | High | v1.0 (#94) | Medium |

### Top 5 Actions Before Starting v1.0 JSON Work

1. Extract shared annotation parsing into `internal/shared/` package (prevents pitfall #1)
2. Fix silent error suppression in cross-file resolution (prevents pitfall #8)
3. Audit HTTP handler serialization path for protojson vs encoding/json consistency (prevents pitfall #10)
4. Add recursive message and multi-file test fixtures (prevents pitfalls #8, #11)
5. Design all 10 annotations as message types with reserved field number range (prevents pitfall #12)

---

*Research completed: 2026-02-05*

**Sources:**
- [Proto Best Practices](https://protobuf.dev/best-practices/dos-donts/)
- [ProtoJSON Format](https://protobuf.dev/programming-guides/json/)
- [Field Presence](https://protobuf.dev/programming-guides/field_presence/)
- [protobuf int64 JSON string issue #2679](https://github.com/protocolbuffers/protobuf/issues/2679)
- [protobuf int64 JSON number option #8331](https://github.com/protocolbuffers/protobuf/issues/8331)
- [protobuf oneof issue #15777](https://github.com/protocolbuffers/protobuf/issues/15777)
- [grpc-gateway cross-language issue #298](https://github.com/grpc-ecosystem/grpc-gateway/issues/298)
- [golang/protobuf enum compat #636](https://github.com/golang/protobuf/issues/636)
- [protobuf enum zero value #6355](https://github.com/protocolbuffers/protobuf/issues/6355)
- [Ruby well-known types incompatibility #4549](https://github.com/protocolbuffers/protobuf/issues/4549)
- [protobuf.js well-known types #839](https://github.com/protobufjs/protobuf.js/issues/839)
- [prost-wkt Rust well-known types](https://github.com/fdeantoni/prost-wkt)
- [GoGo Protobuf lessons](https://jbrandhorst.com/post/gogoproto/)
- [connect-rpc multi-language](https://connectrpc.com/)
- [connect-swift](https://github.com/connectrpc/connect-swift)
- [connect-kotlin](https://github.com/connectrpc/connect-kotlin)
- [Avoiding Protobuf Pitfalls with Buf](https://earthly.dev/blog/buf-protobuf/)
- [python-betterproto](https://github.com/danielgtaylor/python-betterproto)
- [Protobuf-ES](https://buf.build/blog/protobuf-es-the-protocol-buffers-typescript-javascript-runtime-we-all-deserve)
