package httpgen

import (
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// Int64EncodingContext holds information about messages that need custom JSON encoding
// for int64/uint64 fields with NUMBER encoding.
type Int64EncodingContext struct {
	// Message is the message that needs custom marshal/unmarshal
	Message *protogen.Message
	// NumberFields are fields with int64_encoding=NUMBER annotation
	NumberFields []*protogen.Field
	// NestedFields are message-type fields whose type transitively reaches NUMBER encoding.
	// A message can need both: patching its own fields is not enough when it also holds a
	// child that reaches an annotated field, because protojson owns the child's bytes.
	NestedFields []*protogen.Field
}

// hasInt64NumberFields returns true if any int64/uint64 field in the message has NUMBER encoding.
// This checks direct fields only (not nested messages).
func hasInt64NumberFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if isInt64Type(field) && annotations.IsInt64NumberEncoding(field) {
			return true
		}
	}
	return false
}

// getInt64NumberFields returns all int64/uint64 fields that have NUMBER encoding.
func getInt64NumberFields(message *protogen.Message) []*protogen.Field {
	var fields []*protogen.Field
	for _, field := range message.Fields {
		if isInt64Type(field) && annotations.IsInt64NumberEncoding(field) {
			fields = append(fields, field)
		}
	}
	return fields
}

// isInt64Type returns true if the field is an int64 or uint64 type (including variants).
func isInt64Type(field *protogen.Field) bool {
	kind := field.Desc.Kind().String()
	switch kind {
	case kindInt64, kindSint64, kindSfixed64, kindUint64, kindFixed64:
		return true
	default:
		return false
	}
}

// collectInt64EncodingContext analyzes messages in a file and collects int64 encoding information.
func collectInt64EncodingContext(file *protogen.File) []*Int64EncodingContext {
	var contexts []*Int64EncodingContext
	collectInt64EncodingMessages(file.Messages, &contexts)
	return contexts
}

// collectInt64EncodingMessages recursively collects messages with int64 NUMBER encoding fields.
// A message with direct NUMBER fields also carries its nested fields: patching only its own
// fields would leave a child that reaches an annotated field serialized by protojson, so the
// child's int64 would stay a quoted string. The self-referential Node{Node child; int64 id
// [NUMBER]} shape is the smallest case where both are needed at once.
func collectInt64EncodingMessages(messages []*protogen.Message, contexts *[]*Int64EncodingContext) {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}
		if hasInt64NumberFields(msg) {
			*contexts = append(*contexts, &Int64EncodingContext{
				Message:      msg,
				NumberFields: getInt64NumberFields(msg),
				NestedFields: getTransitiveInt64NestedFields(msg),
			})
		}
		// Check nested messages
		collectInt64EncodingMessages(msg.Messages, contexts)
	}
}

// getTransitiveInt64NestedFields returns the message-type fields (singular or repeated, never
// map) whose type transitively reaches an int64 NUMBER field.
func getTransitiveInt64NestedFields(msg *protogen.Message) []*protogen.Field {
	var fields []*protogen.Field
	for _, field := range msg.Fields {
		if fieldTransitivelyHasInt64Number(field) {
			fields = append(fields, field)
		}
	}
	return fields
}

// Int64WrapperContext holds information about messages that contain nested messages
// with int64 NUMBER encoding — requiring transitive MarshalJSON/UnmarshalJSON.
type Int64WrapperContext struct {
	// Message is the wrapper message that needs transitive marshal/unmarshal
	Message *protogen.Message
	// NestedFields are message-type fields whose type transitively reaches NUMBER encoding
	NestedFields []*protogen.Field
}

// messageTransitivelyHasInt64Number reports whether msg, or any message it nests (singular or
// repeated) at any depth, has a direct int64/uint64 field with int64_encoding=NUMBER. Walking
// field.Message resolves across proto files, so imported types are covered — a per-file name set
// is not (issue #217). The visited set guards against recursive message definitions.
//
// Unlike messageTransitivelyHasCustomEnum, this deliberately does NOT descend through map values.
// The emitters cannot re-serialize a map field, so counting a map path as "reachable" would mark a
// parent as a wrapper whose child never gets a marshaler: the parent falls back to protojson and
// the map values stay quoted, and the useless wrapper can trip the MarshalJSON conflict check
// against an annotation the message legitimately carries. The predicate stays aligned with what
// the emitters can actually traverse.
func messageTransitivelyHasInt64Number(msg *protogen.Message, visited map[string]bool) bool {
	if msg == nil {
		return false
	}
	key := string(msg.Desc.FullName())
	if visited[key] {
		return false
	}
	visited[key] = true

	if hasInt64NumberFields(msg) {
		return true
	}

	for _, field := range msg.Fields {
		if child := nestedMessageChild(field); child != nil &&
			messageTransitivelyHasInt64Number(child, visited) {
			return true
		}
	}
	return false
}

// fieldTransitivelyHasInt64Number reports whether a message field's type (singular or repeated,
// never a map) reaches an int64 NUMBER field at any depth.
func fieldTransitivelyHasInt64Number(field *protogen.Field) bool {
	child := nestedMessageChild(field)
	if child == nil {
		return false
	}
	return messageTransitivelyHasInt64Number(child, map[string]bool{})
}

// collectWrapperContexts finds messages that contain fields whose message type transitively
// reaches int64 NUMBER encoding, at any depth and in any file.
func collectWrapperContexts(
	file *protogen.File,
	unwrapMsgNames map[string]bool,
) []*Int64WrapperContext {
	var contexts []*Int64WrapperContext
	collectWrapperMessages(file.Messages, unwrapMsgNames, &contexts)
	return contexts
}

// collectWrapperMessages recursively collects wrapper messages.
func collectWrapperMessages(
	messages []*protogen.Message,
	unwrapMsgNames map[string]bool, // messages to exclude (already have unwrap MarshalJSON)
	contexts *[]*Int64WrapperContext,
) {
	for _, msg := range messages {
		// Bug fix: Skip synthetic proto3 map-entry messages.
		// proto3 map<K,V> fields create implicit nested message types (e.g. Foo_BarEntry)
		// that are never emitted as exported Go struct types by protoc-gen-go.
		if msg.Desc.IsMapEntry() {
			continue
		}

		// Skip messages that already have direct NUMBER fields (handled by existing logic)
		if hasInt64NumberFields(msg) {
			collectWrapperMessages(msg.Messages, unwrapMsgNames, contexts)
			continue
		}

		// Bug fix: Skip messages that already have unwrap-generated MarshalJSON.
		// Both generators cannot emit MarshalJSON for the same type.
		if unwrapMsgNames[string(msg.Desc.FullName())] {
			collectWrapperMessages(msg.Messages, unwrapMsgNames, contexts)
			continue
		}

		// Map fields are excluded: the emitted wrapper cannot traverse them.
		nestedFields := getTransitiveInt64NestedFields(msg)

		if len(nestedFields) > 0 {
			*contexts = append(*contexts, &Int64WrapperContext{
				Message:      msg,
				NestedFields: nestedFields,
			})
		}

		collectWrapperMessages(msg.Messages, unwrapMsgNames, contexts)
	}
}

// checkInt64WrapperMarshalJSONConflict returns an error if a message that needs a transitive
// int64 wrapper marshaler also carries another MarshalJSON-generating annotation. Only one
// feature can own a message's MarshalJSON/UnmarshalJSON methods, so combining them would
// produce duplicate method declarations. Fail fast with a clear message (matching
// enum/flatten/oneof behavior).
func checkInt64WrapperMarshalJSONConflict(msg *protogen.Message) error {
	var conflicts []string

	if hasNullableFields(msg) {
		conflicts = append(conflicts, "nullable")
	}
	if hasEmptyBehaviorFields(msg) {
		conflicts = append(conflicts, "empty_behavior")
	}
	if hasTimestampFormatFields(msg) {
		conflicts = append(conflicts, "timestamp_format")
	}
	if hasBytesEncodingFields(msg) {
		conflicts = append(conflicts, "bytes_encoding")
	}
	if hasCustomEnumFields(msg) {
		conflicts = append(conflicts, "enum_value")
	}
	if hasFlattenFields(msg) {
		conflicts = append(conflicts, "flatten")
	}
	if hasOneofDiscriminator(msg) {
		conflicts = append(conflicts, "oneof_config")
	}

	if len(conflicts) > 0 {
		return fmt.Errorf(
			"message %s: nested int64_encoding=NUMBER requires MarshalJSON but conflicts with %s "+
				"(also requires MarshalJSON) -- "+
				"only one MarshalJSON-generating feature is supported per message",
			msg.GoIdent.GoName, strings.Join(conflicts, ", "),
		)
	}

	return nil
}

// collectDirectEncodingMsgNames returns the set of message full names that will have
// custom MarshalJSON/UnmarshalJSON from the encoding generator (direct NUMBER fields only).
// This is used by the unwrap generator to call json.Marshal instead of protojson.Marshal
// for item types that implement json.Marshaler via the encoding generator.
func collectDirectEncodingMsgNames(file *protogen.File) map[string]bool {
	contexts := collectInt64EncodingContext(file)
	result := make(map[string]bool, len(contexts))
	for _, ctx := range contexts {
		result[string(ctx.Message.Desc.FullName())] = true
	}
	return result
}

// printInt64PrecisionWarning prints a generation-time warning for fields with NUMBER encoding.
func printInt64PrecisionWarning(w io.Writer, field *protogen.Field, messageName string) {
	_, _ = w.Write([]byte(
		"Warning: Field " + messageName + "." + string(field.Desc.Name()) +
			" uses int64_encoding=NUMBER. Values > 2^53 may lose precision in JavaScript.\n",
	))
}

// generateInt64EncodingFile generates the *_encoding.pb.go file if needed.
func (g *Generator) generateInt64EncodingFile(file *protogen.File, unwrapMsgNames map[string]bool) error {
	contexts := collectInt64EncodingContext(file)

	// Collect wrapper messages, excluding those with unwrap-generated MarshalJSON
	wrapperContexts := collectWrapperContexts(file, unwrapMsgNames)

	// If no messages need int64 encoding, skip generation
	if len(contexts) == 0 && len(wrapperContexts) == 0 {
		return nil
	}

	// A wrapper message must own MarshalJSON exclusively — fail fast if another
	// annotation on the same message also generates it.
	for _, ctx := range wrapperContexts {
		if err := checkInt64WrapperMarshalJSONConflict(ctx.Message); err != nil {
			return err
		}
	}

	filename := file.GeneratedFilenamePrefix + "_encoding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeInt64EncodingImports(gf, len(contexts) > 0)

	// Generate marshal/unmarshal for messages with direct NUMBER fields
	for _, ctx := range contexts {
		for _, field := range ctx.NumberFields {
			printInt64PrecisionWarning(os.Stderr, field, ctx.Message.GoIdent.GoName)
		}

		g.generateInt64MarshalJSON(gf, ctx)
		g.generateInt64UnmarshalJSON(gf, ctx)
	}

	// Generate transitive marshal/unmarshal for wrapper messages
	for _, ctx := range wrapperContexts {
		g.generateWrapperMarshalJSON(gf, ctx)
		g.generateWrapperUnmarshalJSON(gf, ctx)
	}

	return nil
}

// writeInt64EncodingImports emits the import block. strconv is only used by the direct-field
// unmarshalers, so a file holding nothing but transitive wrappers — which happens whenever the
// annotated message is declared in an imported file (issue #217) — must not import it, or the
// generated code fails to compile with "strconv imported and not used".
func (g *Generator) writeInt64EncodingImports(gf *protogen.GeneratedFile, needsStrconv bool) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	if needsStrconv {
		gf.P(`"strconv"`)
	}
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
}

// generateInt64MarshalJSON generates a MarshalJSON method that encodes int64 NUMBER fields as numbers.
func (g *Generator) generateInt64MarshalJSON(gf *protogen.GeneratedFile, ctx *Int64EncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	// Build list of NUMBER field names for the comment
	var numberFieldNames []string
	for _, f := range ctx.NumberFields {
		numberFieldNames = append(numberFieldNames, string(f.Desc.Name()))
	}

	gf.P("// MarshalJSONSebuf implements sebufMarshaler for ", msgName, ".")
	gf.P("// This method handles int64_encoding=NUMBER fields: ", strings.Join(numberFieldNames, ", "))
	if len(ctx.NestedFields) > 0 {
		var nestedFieldNames []string
		for _, f := range ctx.NestedFields {
			nestedFieldNames = append(nestedFieldNames, string(f.Desc.Name()))
		}
		gf.P(
			"// It also re-marshals nested messages that reach int64_encoding=NUMBER fields: ",
			strings.Join(nestedFieldNames, ", "),
		)
	}
	gf.P("// Warning: int64 fields with NUMBER encoding may lose precision for values > 2^53 in JavaScript.")
	gf.P(
		"func (x *",
		msgName,
		") MarshalJSONSebuf(opts protojson.MarshalOptions) ([]byte, error) {",
	)
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()

	// First, marshal using protojson to get the base JSON
	gf.P("// Use protojson for base serialization (handles all other fields correctly)")
	gf.P("data, err := opts.Marshal(x)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	// Unmarshal into a map to modify the NUMBER fields
	gf.P("// Parse into a map to modify NUMBER-encoded int64 fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	// For each NUMBER field, replace the string representation with a number
	for _, field := range ctx.NumberFields {
		g.generateInt64FieldMarshal(gf, field)
	}

	// Patching this message's own fields is not enough: a child that reaches an annotated field
	// is still owned by protojson in the base output, so its int64 would stay a quoted string.
	emitNestedFieldsMarshal(gf, ctx.NestedFields)

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()

	// Backward-compatible MarshalJSON wrapper for stdlib encoding/json.
	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("return x.MarshalJSONSebuf(protojson.MarshalOptions{})")
	gf.P("}")
	gf.P()
}

// generateInt64FieldMarshal generates code to marshal a single int64 NUMBER field.
func (g *Generator) generateInt64FieldMarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	fieldName := field.GoName
	jsonName := field.Desc.JSONName()

	if field.Desc.IsList() {
		// Handle repeated int64 fields
		g.generateRepeatedInt64FieldMarshal(gf, fieldName, jsonName)
	} else {
		// Handle singular int64 field
		g.generateSingularInt64FieldMarshal(gf, fieldName, jsonName)
	}
}

// generateSingularInt64FieldMarshal generates marshal code for a singular int64 NUMBER field.
func (g *Generator) generateSingularInt64FieldMarshal(
	gf *protogen.GeneratedFile,
	fieldName, jsonName string,
) {
	gf.P("// Convert ", fieldName, " from string to number")
	gf.P("if x.", fieldName, " != 0 {")
	gf.P(`raw["`, jsonName, `"], _ = json.Marshal(x.`, fieldName, `)`)
	gf.P("} else {")
	gf.P("// Remove the field if zero (proto3 default behavior)")
	gf.P(`delete(raw, "`, jsonName, `")`)
	gf.P("}")
	gf.P()
}

// generateRepeatedInt64FieldMarshal generates marshal code for a repeated int64 NUMBER field.
func (g *Generator) generateRepeatedInt64FieldMarshal(
	gf *protogen.GeneratedFile,
	fieldName, jsonName string,
) {
	gf.P("// Convert repeated ", fieldName, " from strings to numbers")
	gf.P("if len(x.", fieldName, ") > 0 {")
	gf.P(`raw["`, jsonName, `"], _ = json.Marshal(x.`, fieldName, `)`)
	gf.P("}")
	gf.P()
}

// generateInt64UnmarshalJSON generates an UnmarshalJSON method that decodes int64 NUMBER fields from numbers.
func (g *Generator) generateInt64UnmarshalJSON(gf *protogen.GeneratedFile, ctx *Int64EncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	// Build list of NUMBER field names for the comment
	var numberFieldNames []string
	for _, f := range ctx.NumberFields {
		numberFieldNames = append(numberFieldNames, string(f.Desc.Name()))
	}

	gf.P("// UnmarshalJSONSebuf implements sebufUnmarshaler for ", msgName, ".")
	gf.P("// This method handles int64_encoding=NUMBER fields: ", strings.Join(numberFieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {")
	gf.P("// First, parse the raw JSON to extract NUMBER-encoded fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	// For each NUMBER field, convert number to string for protojson
	for _, field := range ctx.NumberFields {
		g.generateInt64FieldUnmarshal(gf, field)
	}

	// Nested children need the same treatment on the way in: protojson would reject the bare
	// JSON numbers their own annotated fields are encoded as.
	emitNestedFieldsUnmarshal(gf, ctx.NestedFields)

	gf.P("// Re-marshal to JSON with string values for protojson")
	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("// Use protojson to unmarshal the rest")
	gf.P("return opts.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()

	// Backward-compatible UnmarshalJSON wrapper for stdlib encoding/json
	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("return x.UnmarshalJSONSebuf(data, protojson.UnmarshalOptions{})")
	gf.P("}")
	gf.P()
}

// generateInt64FieldUnmarshal generates code to unmarshal a single int64 NUMBER field.
func (g *Generator) generateInt64FieldUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	jsonName := field.Desc.JSONName()

	if field.Desc.IsList() {
		// Handle repeated int64 fields
		g.generateRepeatedInt64FieldUnmarshal(gf, field, jsonName)
	} else {
		// Handle singular int64 field
		g.generateSingularInt64FieldUnmarshal(gf, field, jsonName)
	}
}

// generateSingularInt64FieldUnmarshal generates unmarshal code for a singular int64 NUMBER field.
func (g *Generator) generateSingularInt64FieldUnmarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	jsonName string,
) {
	isUnsigned := isUint64Type(field)

	gf.P("// Convert ", jsonName, " from number to string for protojson")
	gf.P(`if rawVal, ok := raw["`, jsonName, `"]; ok {`)
	if isUnsigned {
		gf.P("var num uint64")
		gf.P("if err := json.Unmarshal(rawVal, &num); err == nil {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(strconv.FormatUint(num, 10))`)
	} else {
		gf.P("var num int64")
		gf.P("if err := json.Unmarshal(rawVal, &num); err == nil {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(strconv.FormatInt(num, 10))`)
	}
	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateRepeatedInt64FieldUnmarshal generates unmarshal code for a repeated int64 NUMBER field.
func (g *Generator) generateRepeatedInt64FieldUnmarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	jsonName string,
) {
	isUnsigned := isUint64Type(field)

	gf.P("// Convert repeated ", jsonName, " from numbers to strings for protojson")
	gf.P(`if rawVal, ok := raw["`, jsonName, `"]; ok {`)
	if isUnsigned {
		gf.P("var nums []uint64")
		gf.P("if err := json.Unmarshal(rawVal, &nums); err == nil {")
		gf.P("strs := make([]string, len(nums))")
		gf.P("for i, n := range nums {")
		gf.P("strs[i] = strconv.FormatUint(n, 10)")
		gf.P("}")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(strs)`)
	} else {
		gf.P("var nums []int64")
		gf.P("if err := json.Unmarshal(rawVal, &nums); err == nil {")
		gf.P("strs := make([]string, len(nums))")
		gf.P("for i, n := range nums {")
		gf.P("strs[i] = strconv.FormatInt(n, 10)")
		gf.P("}")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(strs)`)
	}
	gf.P("}")
	gf.P("}")
	gf.P()
}

// isUint64Type returns true if the field is an unsigned 64-bit type.
func isUint64Type(field *protogen.Field) bool {
	kind := field.Desc.Kind().String()
	return kind == kindUint64 || kind == kindFixed64
}

// emitNestedFieldsMarshal emits the re-serialization of message-typed fields whose type reaches
// an int64 NUMBER field, forwarding opts so the child's MarshalJSONSebuf is invoked. Shared by the
// direct and wrapper marshalers: a message with its own NUMBER fields still needs this for its
// children. Assumes `raw`, `opts` and `err` are in scope.
func emitNestedFieldsMarshal(gf *protogen.GeneratedFile, nestedFields []*protogen.Field) {
	for _, field := range nestedFields {
		jsonName := field.Desc.JSONName()
		if field.Desc.IsList() {
			// Repeated field: per-element opts forwarding so child MarshalJSONSebuf receives opts.
			gf.P("// Re-serialize repeated \"", jsonName, "\" forwarding opts to each element")
			gf.P("if len(x.", field.GoName, ") > 0 {")
			gf.P("items := make([]json.RawMessage, 0, len(x.", field.GoName, "))")
			gf.P("for _, item := range x.", field.GoName, " {")
			gf.P(
				"if m, ok := any(item).(interface{ MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error) }); ok {",
			)
			gf.P("itemData, itemErr := m.MarshalJSONSebuf(opts)")
			gf.P("if itemErr != nil {")
			gf.P("return nil, itemErr")
			gf.P("}")
			gf.P("items = append(items, itemData)")
			gf.P("} else {")
			gf.P("itemData, itemErr := opts.Marshal(item)")
			gf.P("if itemErr != nil {")
			gf.P("return nil, itemErr")
			gf.P("}")
			gf.P("items = append(items, itemData)")
			gf.P("}")
			gf.P("}")
			gf.P("raw[\"", jsonName, "\"], err = json.Marshal(items)")
			gf.P("if err != nil {")
			gf.P("return nil, err")
			gf.P("}")
			gf.P("}")
			gf.P()
		} else {
			// Singular field: nil check then re-serialize forwarding opts when possible.
			gf.P("// Re-serialize \"", jsonName, "\" forwarding opts when child supports MarshalJSONSebuf")
			gf.P("if x.", field.GoName, " != nil {")
			gf.P(
				"if m, ok := any(x.",
				field.GoName,
				").(interface{ MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error) }); ok {",
			)
			gf.P("raw[\"", jsonName, "\"], err = m.MarshalJSONSebuf(opts)")
			gf.P("} else {")
			gf.P("raw[\"", jsonName, "\"], err = opts.Marshal(x.", field.GoName, ")")
			gf.P("}")
			gf.P("if err != nil {")
			gf.P("return nil, err")
			gf.P("}")
			gf.P("}")
			gf.P()
		}
	}
}

// generateWrapperMarshalJSON generates a MarshalJSONSebuf that re-marshals nested
// messages via the sebuf opts pipeline, so their custom MarshalJSONSebuf methods are called.
func (g *Generator) generateWrapperMarshalJSON(gf *protogen.GeneratedFile, ctx *Int64WrapperContext) {
	msgName := ctx.Message.GoIdent.GoName

	var nestedFieldNames []string
	for _, f := range ctx.NestedFields {
		nestedFieldNames = append(nestedFieldNames, string(f.Desc.Name()))
	}

	gf.P("// MarshalJSONSebuf implements sebufMarshaler for ", msgName, ".")
	gf.P(
		"// This method re-marshals nested messages that have int64_encoding=NUMBER fields: ",
		strings.Join(nestedFieldNames, ", "),
	)
	gf.P(
		"func (x *",
		msgName,
		") MarshalJSONSebuf(opts protojson.MarshalOptions) ([]byte, error) {",
	)
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()
	gf.P("// Use protojson for base serialization (handles all other fields correctly)")
	gf.P("data, err := opts.Marshal(x)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()
	gf.P("// Parse into a map to re-serialize nested messages with custom MarshalJSON")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	emitNestedFieldsMarshal(gf, ctx.NestedFields)

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()

	// Backward-compatible MarshalJSON wrapper for stdlib encoding/json.
	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("return x.MarshalJSONSebuf(protojson.MarshalOptions{})")
	gf.P("}")
	gf.P()
}

// emitNestedFieldsUnmarshal emits per-field decoding of message-typed fields whose type reaches
// an int64 NUMBER field, dispatching through the child's UnmarshalJSONSebuf so opts propagate.
// Shared by the direct and wrapper unmarshalers. Assumes `raw` and `opts` are in scope.
func emitNestedFieldsUnmarshal(gf *protogen.GeneratedFile, nestedFields []*protogen.Field) {
	for _, field := range nestedFields {
		jsonName := field.Desc.JSONName()
		if field.Desc.IsList() {
			// Repeated field: decode as raw items so we can dispatch each element
			// through UnmarshalJSONSebuf (opts propagation) or json.Unmarshaler fallback.
			gf.P("// Handle repeated \"", jsonName, "\" using its custom unmarshaler")
			gf.P("if rawVal, ok := raw[\"", jsonName, "\"]; ok {")
			gf.P("var rawItems []json.RawMessage")
			gf.P("if err := json.Unmarshal(rawVal, &rawItems); err != nil {")
			gf.P("return err")
			gf.P("}")
			gf.P("protoItems := make([]json.RawMessage, len(rawItems))")
			gf.P("for i, itemRaw := range rawItems {")
			gf.P("inner := &", gf.QualifiedGoIdent(field.Message.GoIdent), "{}")
			gf.P(
				"if u, ok := any(inner).(interface{ UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error }); ok {",
			)
			gf.P("if err := u.UnmarshalJSONSebuf(itemRaw, opts); err != nil {")
			gf.P("return err")
			gf.P("}")
			gf.P("} else if err := json.Unmarshal(itemRaw, inner); err != nil {")
			gf.P("return err")
			gf.P("}")
			gf.P("itemJSON, marshalErr := protojson.Marshal(inner)")
			gf.P("if marshalErr != nil {")
			gf.P("return marshalErr")
			gf.P("}")
			gf.P("protoItems[i] = itemJSON")
			gf.P("}")
			gf.P("protoJSON, marshalErr := json.Marshal(protoItems)")
			gf.P("if marshalErr != nil {")
			gf.P("return marshalErr")
			gf.P("}")
			gf.P("raw[\"", jsonName, "\"] = protoJSON")
			gf.P("}")
			gf.P()
		} else {
			// Singular field: dispatch through UnmarshalJSONSebuf or json.Unmarshaler fallback.
			gf.P("// Handle \"", jsonName, "\" using its custom unmarshaler")
			gf.P("if rawVal, ok := raw[\"", jsonName, "\"]; ok {")
			gf.P("inner := &", gf.QualifiedGoIdent(field.Message.GoIdent), "{}")
			gf.P(
				"if u, ok := any(inner).(interface{ UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error }); ok {",
			)
			gf.P("if err := u.UnmarshalJSONSebuf(rawVal, opts); err != nil {")
			gf.P("return err")
			gf.P("}")
			gf.P("} else if err := json.Unmarshal(rawVal, inner); err != nil {")
			gf.P("return err")
			gf.P("}")
			gf.P("innerJSON, err := protojson.Marshal(inner)")
			gf.P("if err != nil {")
			gf.P("return err")
			gf.P("}")
			gf.P("raw[\"", jsonName, "\"] = innerJSON")
			gf.P("}")
			gf.P()
		}
	}
}

// generateWrapperUnmarshalJSON generates an UnmarshalJSONSebuf that delegates nested
// message parsing via the sebufUnmarshaler interface (propagating opts), then converts
// back for protojson. Also emits a backward-compatible UnmarshalJSON wrapper.
func (g *Generator) generateWrapperUnmarshalJSON(gf *protogen.GeneratedFile, ctx *Int64WrapperContext) {
	msgName := ctx.Message.GoIdent.GoName

	var nestedFieldNames []string
	for _, f := range ctx.NestedFields {
		nestedFieldNames = append(nestedFieldNames, string(f.Desc.Name()))
	}

	gf.P("// UnmarshalJSONSebuf implements sebufUnmarshaler for ", msgName, ".")
	gf.P(
		"// This method handles nested messages that have int64_encoding=NUMBER fields: ",
		strings.Join(nestedFieldNames, ", "),
	)
	gf.P("func (x *", msgName, ") UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	emitNestedFieldsUnmarshal(gf, ctx.NestedFields)

	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("return opts.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()

	// Backward-compatible UnmarshalJSON wrapper for stdlib encoding/json
	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("return x.UnmarshalJSONSebuf(data, protojson.UnmarshalOptions{})")
	gf.P("}")
	gf.P()
}
