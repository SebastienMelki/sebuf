package httpgen

import (
	"io"
	"os"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// Int64EncodingContext holds information about messages that need custom JSON encoding
// for int64/uint64 fields with NUMBER encoding.
type Int64EncodingContext struct {
	// Message is the message that needs custom marshal/unmarshal
	Message *protogen.Message
	// NumberFields are fields with int64_encoding=NUMBER annotation
	NumberFields []*protogen.Field
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
func collectInt64EncodingMessages(messages []*protogen.Message, contexts *[]*Int64EncodingContext) {
	for _, msg := range messages {
		if hasInt64NumberFields(msg) {
			*contexts = append(*contexts, &Int64EncodingContext{
				Message:      msg,
				NumberFields: getInt64NumberFields(msg),
			})
		}
		// Check nested messages
		collectInt64EncodingMessages(msg.Messages, contexts)
	}
}

// Int64WrapperContext holds information about messages that contain nested messages
// with int64 NUMBER encoding — requiring transitive MarshalJSON/UnmarshalJSON.
type Int64WrapperContext struct {
	// Message is the wrapper message that needs transitive marshal/unmarshal
	Message *protogen.Message
	// NestedFields are message-type fields whose type has direct NUMBER encoding
	NestedFields []*protogen.Field
}

// collectWrapperContexts finds messages that contain fields whose message type
// has direct int64 NUMBER encoding (i.e., types already in directMsgNames).
func collectWrapperContexts(file *protogen.File, directMsgNames map[string]bool) []*Int64WrapperContext {
	var contexts []*Int64WrapperContext
	collectWrapperMessages(file.Messages, directMsgNames, &contexts)
	return contexts
}

// collectWrapperMessages recursively collects wrapper messages.
func collectWrapperMessages(
	messages []*protogen.Message,
	directMsgNames map[string]bool,
	contexts *[]*Int64WrapperContext,
) {
	for _, msg := range messages {
		// Skip messages that already have direct NUMBER fields (handled by existing logic)
		if directMsgNames[string(msg.Desc.FullName())] {
			collectWrapperMessages(msg.Messages, directMsgNames, contexts)
			continue
		}

		var nestedFields []*protogen.Field
		for _, field := range msg.Fields {
			if field.Desc.Kind() == protoreflect.MessageKind &&
				!field.Desc.IsMap() &&
				field.Message != nil &&
				directMsgNames[string(field.Message.Desc.FullName())] {
				nestedFields = append(nestedFields, field)
			}
		}

		if len(nestedFields) > 0 {
			*contexts = append(*contexts, &Int64WrapperContext{
				Message:      msg,
				NestedFields: nestedFields,
			})
		}

		collectWrapperMessages(msg.Messages, directMsgNames, contexts)
	}
}

// printInt64PrecisionWarning prints a generation-time warning for fields with NUMBER encoding.
func printInt64PrecisionWarning(w io.Writer, field *protogen.Field, messageName string) {
	_, _ = w.Write([]byte(
		"Warning: Field " + messageName + "." + string(field.Desc.Name()) +
			" uses int64_encoding=NUMBER. Values > 2^53 may lose precision in JavaScript.\n",
	))
}

// generateInt64EncodingFile generates the *_encoding.pb.go file if needed.
func (g *Generator) generateInt64EncodingFile(file *protogen.File) error {
	contexts := collectInt64EncodingContext(file)

	// Build set of message full names that have direct NUMBER fields
	directMsgNames := make(map[string]bool, len(contexts))
	for _, ctx := range contexts {
		directMsgNames[string(ctx.Message.Desc.FullName())] = true
	}

	// Collect wrapper messages whose fields reference messages with NUMBER fields
	wrapperContexts := collectWrapperContexts(file, directMsgNames)

	// If no messages need int64 encoding, skip generation
	if len(contexts) == 0 && len(wrapperContexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_encoding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeInt64EncodingImports(gf)

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

func (g *Generator) writeInt64EncodingImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P(`"strconv"`)
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

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles int64_encoding=NUMBER fields: ", strings.Join(numberFieldNames, ", "))
	gf.P("// Warning: int64 fields with NUMBER encoding may lose precision for values > 2^53 in JavaScript.")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()

	// First, marshal using protojson to get the base JSON
	gf.P("// Use protojson for base serialization (handles all other fields correctly)")
	gf.P("data, err := protojson.Marshal(x)")
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

	gf.P("return json.Marshal(raw)")
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

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles int64_encoding=NUMBER fields: ", strings.Join(numberFieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
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

	gf.P("// Re-marshal to JSON with string values for protojson")
	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("// Use protojson to unmarshal the rest")
	gf.P("return protojson.Unmarshal(modified, x)")
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

// generateWrapperMarshalJSON generates a MarshalJSON that re-marshals nested
// messages via json.Marshal, so their custom MarshalJSON methods are called.
func (g *Generator) generateWrapperMarshalJSON(gf *protogen.GeneratedFile, ctx *Int64WrapperContext) {
	msgName := ctx.Message.GoIdent.GoName

	var nestedFieldNames []string
	for _, f := range ctx.NestedFields {
		nestedFieldNames = append(nestedFieldNames, string(f.Desc.Name()))
	}

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P(
		"// This method re-marshals nested messages that have int64_encoding=NUMBER fields: ",
		strings.Join(nestedFieldNames, ", "),
	)
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()
	gf.P("// Use protojson for base serialization (handles all other fields correctly)")
	gf.P("data, err := protojson.Marshal(x)")
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

	for _, field := range ctx.NestedFields {
		jsonName := field.Desc.JSONName()
		gf.P("// Re-serialize \"", jsonName, "\" using its custom MarshalJSON")
		gf.P("if x.", field.GoName, " != nil {")
		gf.P("raw[\"", jsonName, "\"], err = json.Marshal(x.", field.GoName, ")")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P("}")
		gf.P()
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()
}

// generateWrapperUnmarshalJSON generates an UnmarshalJSON that delegates nested
// message parsing to their custom UnmarshalJSON, then converts back for protojson.
func (g *Generator) generateWrapperUnmarshalJSON(gf *protogen.GeneratedFile, ctx *Int64WrapperContext) {
	msgName := ctx.Message.GoIdent.GoName

	var nestedFieldNames []string
	for _, f := range ctx.NestedFields {
		nestedFieldNames = append(nestedFieldNames, string(f.Desc.Name()))
	}

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P(
		"// This method handles nested messages that have int64_encoding=NUMBER fields: ",
		strings.Join(nestedFieldNames, ", "),
	)
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	for _, field := range ctx.NestedFields {
		jsonName := field.Desc.JSONName()
		gf.P("// Handle \"", jsonName, "\" using its custom UnmarshalJSON")
		gf.P("if rawVal, ok := raw[\"", jsonName, "\"]; ok {")
		gf.P("inner := &", gf.QualifiedGoIdent(field.Message.GoIdent), "{}")
		gf.P("if err := json.Unmarshal(rawVal, inner); err != nil {")
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

	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("return protojson.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()
}
