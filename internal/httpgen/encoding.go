package httpgen

import (
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

	// If no messages need int64 encoding, skip generation
	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_encoding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeInt64EncodingImports(gf)

	// Generate marshal/unmarshal for each message
	for _, ctx := range contexts {
		// Print warnings during generation
		for _, field := range ctx.NumberFields {
			printInt64PrecisionWarning(os.Stderr, field, ctx.Message.GoIdent.GoName)
		}

		g.generateInt64MarshalJSON(gf, ctx)
		g.generateInt64UnmarshalJSON(gf, ctx)
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
