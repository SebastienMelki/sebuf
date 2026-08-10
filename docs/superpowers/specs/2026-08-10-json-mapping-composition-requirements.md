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

## Named Test Case Reference

### Generation and Build Cases
- `GBM-ALL-PAIRS`: exercise every valid pair among the feature families in the Generation and Build Matrix.
- `GBM-ROOT-STRUCTURAL`: include root unwrap pairs only when the root unwrap field can be structurally combined with sibling annotations by the composed document transform.
- `GBM-INVALID-SEMANTIC-ONLY`: invalid annotation shapes fail only for semantic annotation errors, not duplicate method declarations.

### Runtime Marshal Cases
- `RM-NULLABLE-TIMESTAMP`: nullable + timestamp_format on one message emits `null` and unix timestamp simultaneously.
- `RM-ENUM-NULLABLE`: enum_value + nullable emits custom enum strings and explicit null.
- `RM-BYTES-CHILD-NULLABLE-PARENT`: bytes_encoding child nested under nullable parent emits child hex/base64url/raw format, not protojson default base64.
- `RM-INT64-NESTED-ANNOTATED-PARENT`: int64_number child nested under another annotated parent emits JSON numbers at every nested level.
- `RM-FLATTEN-DELEGATES-CHILD`: flatten parent delegates child custom mappings before promoting fields.
- `RM-MAP-VALUE-UNWRAP-SIBLINGS`: map-value unwrap composes with sibling nullable/timestamp/enum fields.
- `RM-ROOT-UNWRAP-CHILD-MAP-VALUE`: root unwrap composes with child/map-value transforms.

### Runtime Unmarshal Cases
- `RU-MARSHAL-ROUND-TRIP`: the JSON emitted by each marshal runtime case unmarshals into the expected proto values using `UnmarshalJSONSebuf`.
- `RU-CHILD-CUSTOM-THROUGH-PARENT`: child custom JSON accepted through a parent does not rely on protojson accepting the custom representation directly.

### Regression Invariants
- `RI-SINGLE-METHOD-PER-TYPE`: generated code contains at most one `MarshalJSONSebuf` and one `UnmarshalJSONSebuf` per Go message type.
- `RI-CONFLICT-ERRORS-REMOVED`: previous conflict errors containing `only one MarshalJSON-generating feature is supported per message` disappear for composable cases.
- `RI-NO-UNUSED-IMPORTS`: no generated file imports unused packages.
- `RI-SINGLE-FEATURE-GOLDEN-COMPAT`: existing single-feature golden behavior is preserved unless explicitly covered by a corrected nested/composed runtime assertion.
