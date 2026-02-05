# JSON Mapping Features Research

Research date: 2026-02-05
Scope: Issues #87--#96 (10 proposed JSON mapping features for sebuf 1.0)
Competing tools analyzed: grpc-gateway, connect-go/connectrpc, twirp, envoy gRPC-JSON transcoder, protojson (Go canonical library), ts-proto, protobuf-es

---

## Executive Summary

Of the 10 proposed features, **3 are table stakes** (competing tools already provide them), **3 are strong differentiators** (nobody does this at the annotation level), **2 are moderate differentiators** (partial solutions exist elsewhere), and **2 are anti-features** (overlap with existing mechanisms or low practical value). All competing tools rely on global runtime options (e.g., `protojson.MarshalOptions`) rather than per-field proto annotations, which is sebuf's core differentiator: **declarative, schema-level JSON mapping that works consistently across all 4 generators**.

### Classification at a Glance

| # | Feature | Category | Complexity | Priority |
|---|---------|----------|------------|----------|
| #87 | Nullable primitives | Table stakes | Medium | High |
| #88 | int64 as string | Table stakes | Low | High |
| #89 | Enum string encoding | Table stakes | Low-Medium | High |
| #90 | Oneof discriminated union | Strong differentiator | High | Medium |
| #91 | Root-level arrays | Already solved | N/A | N/A |
| #92 | Timestamp formats | Moderate differentiator | Medium | Medium |
| #93 | Empty object handling | Moderate differentiator | Medium | Medium |
| #94 | Field name casing | Anti-feature (overlap) | Low | Low |
| #95 | Bytes encoding | Strong differentiator | Low | Low |
| #96 | Nested message flattening | Strong differentiator | High | Low |

---

## Feature-by-Feature Analysis

### #87 -- Nullable Primitives (null vs absent vs default)

**Category: TABLE STAKES**

**How competing tools handle it:**

- **Proto3 built-in**: The `optional` keyword (added in proto 3.15) generates `has_` presence methods and pointer types in Go. However, `protojson` does NOT emit `null` for unset optional fields -- it omits them entirely.
- **protojson**: `EmitUnpopulated` emits all fields including unset ones (as zero values, not null). `EmitDefaultValues` emits only set-to-default fields. Neither produces JSON `null` for scalar fields.
- **grpc-gateway**: Uses `google.protobuf.StringValue`/`Int32Value` wrapper types as the standard workaround. These are verbose and poor DX. The gateway itself uses protojson under the hood.
- **connect-go**: Delegates entirely to protojson. No custom nullable support.
- **twirp**: Uses protojson. No nullable customization.
- **ts-proto**: Wrapper types (`google.protobuf.StringValue`) map to `string | undefined`. The `useOptionals` flag controls whether fields are `field?: T` vs `field: T | undefined`.

**Why table stakes**: Every REST API framework (Rails, Django, Express, FastAPI) supports null vs absent vs default distinction. PATCH semantics require it. The protobuf ecosystem forces verbose wrapper types or `optional` + custom code. Any HTTP API toolkit must solve this cleanly.

**Sebuf's angle**: Per-field `(sebuf.http.nullable) = true` annotation is cleaner than wrapper types and goes beyond what `optional` provides (actual JSON `null` emission).

**Complexity: MEDIUM**
- Proto annotation definition: trivial
- Go server/client codegen: Must generate pointer types, custom MarshalJSON/UnmarshalJSON with null handling
- TS client codegen: Map to `T | null` type union
- OpenAPI codegen: Emit `nullable: true` (OAS 3.0) or `type: ["string", "null"]` (OAS 3.1)
- Interaction with proto3 `optional` must be clarified

**Dependencies**: None. Foundational feature that other features (#93) depend on.

**Implementation risk**: Must decide semantics clearly: `nullable` annotation + `optional` keyword = what behavior? Recommend: `nullable` controls JSON null emission, `optional` controls field presence tracking. They compose: a nullable optional field can be null, absent, or set.

---

### #88 -- int64 as String Encoding

**Category: TABLE STAKES**

**How competing tools handle it:**

- **Proto3 canonical JSON spec**: int64, uint64, fixed64, sfixed64 are **always** encoded as strings in canonical protojson. This is the official specification. The rationale: JavaScript Number.MAX_SAFE_INTEGER is 2^53, so 64-bit integers lose precision.
- **protojson (Go)**: Always encodes int64/uint64 as JSON strings. There is an open issue (#1414 on golang/protobuf) requesting `Write64KindsAsInteger` to emit them as numbers, but it has not been implemented.
- **grpc-gateway**: Inherits protojson behavior. int64 always serialized as string. Open issue #438 requesting number format.
- **connect-go**: Uses protojson. int64 is always string.
- **protobuf-es/connect-es**: Uses BigInt in JavaScript runtime; JSON wire format uses strings per spec. Connect clients automatically convert.
- **ts-proto**: int64 fields map to `number` by default (with precision warning) or `string` with `forceLong=string`. BigInt support available with `forceLong=bigint`.
- **envoy transcoder**: Follows protobuf spec (strings for 64-bit).

**Why table stakes**: The protobuf spec mandates string encoding for int64. But many existing REST APIs (especially non-protobuf ones) use numeric int64 in JSON. If sebuf targets teams building new APIs, string is correct default. If targeting interop with existing APIs, number encoding must be an option.

**Sebuf's angle**: Per-field control (`STRING` vs `NUMBER`) and file-level default. More granular than protojson's all-or-nothing behavior.

**Complexity: LOW**
- Proto annotation: Simple enum option
- Go codegen: `strconv.FormatInt`/`strconv.ParseInt` for string mode; direct numeric for number mode
- TS codegen: `string` type for string mode; `number` for number mode (with BigInt consideration)
- OpenAPI: `type: string, format: int64` vs `type: integer, format: int64`

**Dependencies**: None. Self-contained.

**Implementation risk**: Low. Well-understood problem. Main decision is default behavior (sebuf should default to `NUMBER` for HTTP API friendliness, unlike protojson which defaults to `STRING`).

---

### #89 -- Enum String Encoding with Custom Values

**Category: TABLE STAKES**

**How competing tools handle it:**

- **Proto3 canonical JSON**: Enums are serialized as their proto name strings (e.g., `"STATUS_ACTIVE"`). Parsers must also accept numeric values.
- **protojson**: `UseEnumNumbers` option switches globally to integer encoding. No per-enum or per-value customization.
- **grpc-gateway**: `enums_as_ints` option in OpenAPI generation. Global toggle via `MarshalOptions.UseEnumNumbers`.
- **connect-go**: Delegates to protojson options. Global enum number toggle.
- **ts-proto**: `stringEnums=true` generates string-based TypeScript enums. `useNumericEnumForJson=true` for JSON encoding as integers.
- **envoy**: `preserve_proto_field_names` preserves original field names (related but not enum-specific).

**What nobody does**: Custom enum value strings (e.g., `STATUS_ACTIVE` -> `"active"`). All tools use either the raw proto name or the integer. The `json_name` option exists for fields but NOT for enum values.

**Why table stakes** (with differentiating element): String vs number encoding for enums is table stakes. Custom value strings are a differentiator, but a commonly requested one -- REST APIs universally use lowercase strings like `"active"`, `"pending"`, not `"STATUS_ACTIVE"`.

**Sebuf's angle**: Both the encoding mode (string/number) and custom value mapping are per-enum proto annotations. This is unique.

**Complexity: LOW-MEDIUM**
- Proto annotation: Enum-level encoding mode + per-value custom string
- Go codegen: Bidirectional lookup maps, custom MarshalJSON/UnmarshalJSON on enum type
- TS codegen: String literal union type with custom values
- OpenAPI: `enum: ["active", "inactive"]` with custom values

**Dependencies**: None. Self-contained.

**Implementation risk**: Low. Must handle unknown enum values gracefully (fail? use numeric fallback?).

---

### #90 -- Oneof as Discriminated Union (Flattened with Type Field)

**Category: STRONG DIFFERENTIATOR**

**How competing tools handle it:**

- **Proto3 canonical JSON**: Oneof fields serialize as the set field's name with its value. `{"click": {"x": 100, "y": 200}}`. No flattening. No discriminator field.
- **protojson**: No support for discriminated unions or flattening.
- **grpc-gateway**: GitHub issue #82 (2016) requested oneof support in query parameters. Issue #585 requested discriminator support in OpenAPI. Neither fully addressed. Standard oneof JSON output only.
- **connect-go**: Standard protojson oneof handling. No flattening.
- **ts-proto**: `oneof=unions` generates ADT types: `{ $case: 'click'; click: ClickEvent } | { $case: 'purchase'; purchase: PurchaseEvent }`. This is a TypeScript type-level discriminated union but the JSON wire format is still standard protobuf (`{"click": {...}}`). The discriminator field `$case` exists only in the TypeScript type system, not in serialized JSON.
- **OpenAPI**: The `discriminator` keyword with `oneOf` is a standard OAS pattern, but no protobuf tool generates this from oneof today.

**Why this is a strong differentiator**: Discriminated unions with flattened fields are the standard REST API pattern (Stripe, AWS, GitHub webhooks all use `"type"` discriminator fields). No protobuf tool generates this. ts-proto's `$case` is the closest but is TypeScript-only and not a wire format feature.

**Complexity: HIGH**
- Proto annotations: Oneof-level discriminator name + flatten flag, per-field discriminator value
- Go codegen: Complex MarshalJSON merging parent fields with child fields, UnmarshalJSON with type-switch on raw JSON. Field collision detection at generation time.
- TS codegen: TypeScript discriminated union types with shared interface
- OpenAPI: `discriminator` with `oneOf` schemas, flattened property definitions
- Edge cases: Field name collisions between parent and child, nested oneofs, optional fields in variants

**Dependencies**: Benefits from #87 (nullable) for optional discriminator fields, but not strictly required.

**Implementation risk**: HIGH. Field collision detection, deeply nested messages, interaction with validation. Recommend phased approach: Phase 1 = discriminator only (no flatten), Phase 2 = flatten.

---

### #91 -- Root-Level Arrays in Responses

**Category: ALREADY SOLVED**

**Current state**: The existing `(sebuf.http.unwrap) = true` annotation on a repeated field inside a response message already handles this case. The codebase has comprehensive unwrap support:
- Root repeated unwrap: `repeated Item items = 1 [(sebuf.http.unwrap) = true]` -> `[{...}, {...}]`
- Root map unwrap: `map<string, Item> items = 1 [(sebuf.http.unwrap) = true]` -> `{"key": {...}}`
- Combined root + value unwrap: nested unwrapping

This is confirmed by test files at:
- `internal/httpgen/testdata/proto/unwrap.proto` (lines 91-94 for `RootRepeatedResponse`)
- `internal/tsclientgen/testdata/proto/complex_features.proto` (lines 211-235)
- `internal/openapiv3/testdata/proto/unwrap.proto` (lines 87-96)

**Action**: Close issue #91. Document existing unwrap functionality as the solution. Consider adding a prominent example in documentation.

**Dependencies**: None (already implemented).

---

### #92 -- Multiple Timestamp Formats

**Category: MODERATE DIFFERENTIATOR**

**How competing tools handle it:**

- **Proto3 canonical JSON**: `google.protobuf.Timestamp` always serializes as RFC 3339 string (`"2024-01-15T09:30:00Z"`). This is the specification. No options.
- **protojson**: Hardcoded RFC 3339 for Timestamp well-known type. No customization.
- **grpc-gateway**: Inherits protojson behavior. RFC 3339 only.
- **connect-go**: RFC 3339 only.
- **twirp**: RFC 3339 only.
- **protobuf-es**: RFC 3339 only. Timestamp maps to `{ seconds: bigint, nanos: number }` internally.
- **ts-proto**: Timestamp can map to `Date` object or `Timestamp` message type, but JSON format is always RFC 3339.
- **envoy**: RFC 3339 only.

**Why moderate differentiator**: The entire protobuf ecosystem is locked to RFC 3339 for Timestamps. Many REST APIs use Unix timestamps (Stripe uses Unix seconds, JavaScript APIs often use Unix milliseconds). Supporting multiple formats is genuinely useful for interop, but it's not a widely expected feature because protobuf users accept RFC 3339.

**Complexity: MEDIUM**
- Proto annotation: Per-field enum (RFC3339, UNIX_SECONDS, UNIX_MILLIS, DATE)
- Go codegen: Format-specific marshal/unmarshal in custom JSON methods
- TS codegen: Type changes from `string` to `number` for unix formats; `Date` handling per format
- OpenAPI: `format: date-time` vs `format: date` vs custom `format: unix-timestamp`
- Precision: Unix millis requires nanosecond-to-millisecond conversion

**Dependencies**: None. Self-contained.

**Implementation risk**: Medium. DATE format lossy (drops time). Must document timezone semantics for non-RFC3339 formats. Only applies to `google.protobuf.Timestamp` fields.

---

### #93 -- Empty Object Handling (Preserve vs Omit vs Null)

**Category: MODERATE DIFFERENTIATOR**

**How competing tools handle it:**

- **Proto3 default**: Zero-valued message fields are omitted from JSON output. An empty message `{}` is the zero value, so it's omitted.
- **protojson**: `EmitUnpopulated` emits all fields (empty messages as `{}`). `EmitDefaultValues` emits set-to-default primitives but not unset message fields. Neither provides null for empty messages.
- **grpc-gateway**: Configurable via `MarshalOptions.EmitUnpopulated`. Global toggle only.
- **connect-go**: Global toggle via protojson options.
- **ts-proto**: `useOptionals` controls whether unset fields are `undefined` or omitted.

**What nobody does**: Per-field control over empty object behavior. All tools use a single global toggle.

**Why moderate differentiator**: PATCH semantics (RFC 7396 JSON Merge Patch) give different meaning to `{}`, `null`, and absent. Per-field control is useful for APIs that mix PATCH and non-PATCH endpoints. But the use case is narrow -- most APIs use a consistent approach.

**Complexity: MEDIUM**
- Proto annotation: Per-field enum (PRESERVE, NULL, OMIT) + `omit_empty` boolean
- Go codegen: Conditional emission logic in MarshalJSON with `proto.Size()` zero-check
- TS codegen: Optional chaining and null type handling
- OpenAPI: Required array manipulation, nullable annotation
- Interaction with #87 (nullable): A nullable field with empty_behavior=NULL means two sources of null. Must define precedence.

**Dependencies**: Partially overlaps with #87 (nullable primitives). Should be designed together.

**Implementation risk**: Medium. Complex interaction with nullable. The `omit_empty` option overlaps with Go's `omitempty` and protojson's `EmitUnpopulated`. Must clearly differentiate sebuf's behavior.

**Recommendation**: Simplify to just `omit_empty = true` for 1.0. Defer `empty_behavior` enum to post-1.0. The simpler version covers 90% of use cases.

---

### #94 -- Field Name Casing Options

**Category: ANTI-FEATURE (significant overlap with existing mechanisms)**

**How competing tools handle it:**

- **Proto3 built-in**: `json_name` field option allows explicit override per field. Default behavior: proto compiler auto-generates camelCase `json_name` from snake_case field names. JSON parsers MUST accept both the camelCase json_name and the original proto field name.
- **protojson**: `UseProtoNames` option uses original proto field names (snake_case) instead of json_name. Global toggle.
- **grpc-gateway**: Supports `UseProtoNames` via MarshalOptions.
- **envoy**: `preserve_proto_field_names` option to keep original names.
- **ts-proto**: `snakeToCamel` option with granular control (`keys`, `json`, `keys_json`).
- **protobuf-es**: Uses json_name by default (camelCase).

**Why anti-feature**: Proto3's built-in `json_name` field option already provides per-field override. The `UseProtoNames` pattern is global and handles the most common case (snake_case preservation). Adding a sebuf-specific `json_naming` file option duplicates built-in functionality. Users who want specific field names can already use `json_name`.

**Complexity: LOW**
- If implemented: snake-to-camel conversion function, file-level option, field-level override
- But the existing `json_name` proto option already does this

**Dependencies**: None.

**Implementation risk**: Low technically, but high in terms of confusion. Users will wonder why they have both `json_name` and `json_name_override`.

**Recommendation**: Do NOT implement. Document proto3's built-in `json_name` option instead. If file-level default is essential, implement only the file-level `json_naming` option (SNAKE_CASE vs CAMEL_CASE) and explicitly state it controls the default, with proto3 `json_name` taking precedence.

---

### #95 -- Bytes Encoding Options (base64, hex)

**Category: STRONG DIFFERENTIATOR**

**How competing tools handle it:**

- **Proto3 canonical JSON**: bytes fields always encode as standard base64 (RFC 4648). No options.
- **protojson**: Hardcoded base64. Open issue (#1030 on golang/protobuf) requesting hex support -- not implemented.
- **grpc-gateway**: Base64 only.
- **connect-go**: Base64 only.
- **All other tools**: Base64 only.
- **Cosmos SDK**: Issue #12994 requested hex display for bytes in JSON. Workaround: change field type to string and convert manually.

**Why strong differentiator**: No protobuf tool supports alternative bytes encoding. Crypto/blockchain APIs universally use hex encoding. JWT/URL contexts use base64url. The standard workaround (use string field type + manual conversion) loses type safety and proto semantics.

**Complexity: LOW**
- Proto annotation: Per-field enum (BASE64, BASE64_RAW, BASE64URL, BASE64URL_RAW, HEX)
- Go codegen: Swap encoding function in MarshalJSON/UnmarshalJSON (`base64.StdEncoding` vs `base64.RawURLEncoding` vs `hex.EncodeToString`)
- TS codegen: Corresponding encode/decode functions
- OpenAPI: `format: byte` vs `pattern: "^[0-9a-fA-F]+$"` for hex

**Dependencies**: None. Self-contained.

**Implementation risk**: Very low. Well-understood encoding. Each variant is a simple function swap.

---

### #96 -- Nested Message Flattening

**Category: STRONG DIFFERENTIATOR (but niche)**

**How competing tools handle it:**

- **No protobuf tool supports this.** Nested messages always serialize as nested JSON objects. There is no flatten or embed option in any competing tool.
- **JSON Schema**: No flatten concept.
- **OpenAPI**: `allOf` can compose schemas but doesn't flatten at the wire level.
- **Go encoding/json**: Supports anonymous struct embedding (which flattens), but protobuf messages are never anonymous.

**Why strong differentiator (but niche)**: This is genuinely novel -- no protobuf tool does it. However, the use cases are narrow: legacy API compatibility, HTML form interop, CSV representations. Most modern APIs use nested objects and this is the expected JSON pattern.

**Complexity: HIGH**
- Proto annotations: Per-field `flatten = true` + `flatten_prefix` string
- Go codegen: Complex MarshalJSON that walks nested fields and emits them at parent level with prefix. UnmarshalJSON must reverse. Recursive flattening (flatten inside flatten) multiplies complexity.
- TS codegen: Flattened interface types, mapping logic
- OpenAPI: Inlined properties with prefix in schema
- Collision detection at generation time (flattened field name matches parent field name)
- Deeply nested messages: how many levels of flatten are supported?

**Dependencies**: Interacts with #90 (oneof flattening uses similar machinery). Shared implementation possible.

**Implementation risk**: HIGH. Recursive flattening, collision detection, interaction with validation (which field validates?), interaction with oneof. The prefix mechanism is fragile with proto field renaming.

**Recommendation**: Defer to post-1.0. The use case is primarily legacy interop, and those teams can use custom MarshalJSON overrides. If implemented, limit to 1 level of flattening initially.

---

## Dependency Graph

```
#87 (Nullable) <-------- #93 (Empty objects) [nullable semantics overlap]
       |
       v
#90 (Oneof discriminated) ---> #96 (Flattening) [shared flatten machinery]
       |
       v
  (OpenAPI discriminator support)

#88 (int64 string) ---- independent
#89 (Enum encoding) ---- independent
#91 (Root arrays) ------ DONE (existing unwrap)
#92 (Timestamps) ------- independent
#94 (Field casing) ----- anti-feature, skip
#95 (Bytes encoding) --- independent
```

Key dependency chains:
1. **#87 must precede #93**: Empty object behavior depends on nullable semantics being defined
2. **#90 informs #96**: Oneof flattening and message flattening share flatten+merge machinery
3. **All others are independent** and can be implemented in any order

---

## Recommended Implementation Order

### Phase 1: Table Stakes (required for 1.0 credibility)
1. **#88 int64 as string** -- Low complexity, high value, independent
2. **#89 Enum encoding** -- Low-medium complexity, high value, independent
3. **#87 Nullable primitives** -- Medium complexity, high value, foundational

### Phase 2: High-Value Differentiators
4. **#95 Bytes encoding** -- Low complexity, strong differentiator
5. **#92 Timestamp formats** -- Medium complexity, useful for interop
6. **#93 Empty object handling** -- Medium complexity, depends on #87

### Phase 3: Complex Differentiators (post-1.0 candidates)
7. **#90 Oneof discriminated union** -- High complexity, strong differentiator, phased delivery
8. **#96 Nested message flattening** -- High complexity, niche use case

### Skip
- **#91**: Already solved by existing unwrap annotation
- **#94**: Overlaps with proto3's built-in `json_name`; document existing mechanism instead

---

## How Competing Tools Compare (Summary Matrix)

| Feature | protojson | grpc-gateway | connect-go | twirp | envoy | ts-proto | sebuf (proposed) |
|---------|-----------|--------------|------------|-------|-------|----------|------------------|
| Nullable primitives | Wrappers only | Wrappers only | Wrappers only | Wrappers only | Wrappers only | Wrappers | Per-field annotation |
| int64 encoding | Always string | Always string | Always string | Always string | Always string | Configurable global | Per-field annotation |
| Enum custom values | No | No | No | No | No | No | Per-value annotation |
| Enum string/number | Global toggle | Global toggle | Global toggle | String only | String only | Global toggle | Per-enum annotation |
| Oneof discriminated | No | No | No | No | No | TS types only | Proto annotation |
| Root-level arrays | No | No | No | No | No | No | Already solved (unwrap) |
| Timestamp formats | RFC3339 only | RFC3339 only | RFC3339 only | RFC3339 only | RFC3339 only | RFC3339 only | Per-field annotation |
| Empty object control | Global toggle | Global toggle | Global toggle | No | No | Partial | Per-field annotation |
| Field name casing | Global toggle | Global toggle | Global toggle | No | Global toggle | Global toggle | Skip (use json_name) |
| Bytes encoding | Base64 only | Base64 only | Base64 only | Base64 only | Base64 only | Base64 only | Per-field annotation |
| Nested flattening | No | No | No | No | No | No | Per-field annotation |

**Sebuf's consistent differentiator**: Every feature is a per-field/per-enum/per-message proto annotation, not a global runtime option. This means:
1. The schema is self-documenting (the proto file describes the JSON format)
2. Different fields in the same message can have different behaviors
3. All 4 generators (go-http, go-client, ts-client, openapiv3) read the same annotations and produce consistent output

---

## Cross-Cutting Concerns

### Custom MarshalJSON/UnmarshalJSON Generation
Features #87, #88, #89, #90, #92, #93, #95, #96 all require generating custom `MarshalJSON`/`UnmarshalJSON` methods on proto messages. The existing unwrap implementation already does this (`internal/httpgen/unwrap.go`). Key consideration: **only one MarshalJSON per message type is allowed**. Multiple features on the same message must be composed into a single method.

**Recommendation**: Build a `MessageJSONOverride` abstraction that collects all field-level customizations for a message and generates a single composite MarshalJSON/UnmarshalJSON.

### Interaction with protojson
sebuf currently uses `protojson.Marshal`/`protojson.Unmarshal` for messages without custom annotations. Messages WITH custom JSON mapping annotations need custom marshal/unmarshal code. The generator must detect which messages need custom code and which can use protojson defaults.

### TypeScript Client Impact
For the TS client generator, each JSON mapping feature changes the TypeScript type definitions AND the runtime serialization logic. The TS client must embed or reference encode/decode helpers for each feature used.

### OpenAPI Impact
Each JSON mapping feature changes the OpenAPI schema output. Features like nullable, int64-as-string, enum custom values, and timestamp formats all affect `type`, `format`, `enum`, and `nullable` properties in the generated OpenAPI spec.

---

## Sources

- [ProtoJSON Format specification](https://protobuf.dev/programming-guides/json/)
- [protojson Go package documentation](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson)
- [gRPC-Gateway customization docs](https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_your_gateway/)
- [gRPC-Gateway int64 string issue #438](https://github.com/grpc-ecosystem/grpc-gateway/issues/438)
- [gRPC-Gateway oneof support issue #82](https://github.com/grpc-ecosystem/grpc-gateway/issues/82)
- [gRPC-Gateway discriminator issue #585](https://github.com/grpc-ecosystem/grpc-gateway/issues/585)
- [gRPC-Gateway null handling issue #1681](https://github.com/grpc-ecosystem/grpc-gateway/issues/1681)
- [gRPC-Gateway PATCH feature docs](https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/patch_feature/)
- [Connect-go serialization docs](https://connectrpc.com/docs/go/serialization-and-compression/)
- [Connect-ES 2.0 announcement](https://buf.build/blog/connect-es-v2)
- [protobuf-es GitHub](https://github.com/bufbuild/protobuf-es)
- [ts-proto GitHub and README](https://github.com/stephenh/ts-proto)
- [ts-proto oneof unions issue #314](https://github.com/stephenh/ts-proto/issues/314)
- [Twirp serialization docs](https://twitchtv.github.io/twirp/docs/proto_and_json.html)
- [Envoy gRPC-JSON transcoder filter docs](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/grpc_json_transcoder_filter)
- [golang/protobuf int64 number option issue #1414](https://github.com/golang/protobuf/issues/1414)
- [golang/protobuf hex bytes issue #1030](https://github.com/golang/protobuf/issues/1030)
- [protobuf int64 as string issue #2679](https://github.com/protocolbuffers/protobuf/issues/2679)
- [protoc-gen-go-json by mfridman](https://github.com/mfridman/protoc-gen-go-json)
- [OpenAPI discriminator guide](https://redocly.com/learn/openapi/discriminator)
- [Google AIP-203 Field Behavior](https://google.aip.dev/203)
- [Stainless: null values in REST APIs](https://www.stainless.com/sdk-api-best-practices/how-to-pass-null-value-in-rest-api-post-put-and-patch)
- [Speakeasy: null in OpenAPI](https://www.speakeasy.com/openapi/schemas/null)
