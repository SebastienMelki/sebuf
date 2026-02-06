package clientgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// TimestampFormatContext holds information about messages that need custom JSON encoding
// for Timestamp fields with non-default format annotations.
type TimestampFormatContext struct {
	// Message is the message that needs custom marshal/unmarshal
	Message *protogen.Message
	// TimestampFields are fields with timestamp_format annotation (non-default)
	TimestampFields []*TimestampFormatFieldInfo
}

// TimestampFormatFieldInfo holds field info with its timestamp format setting.
type TimestampFormatFieldInfo struct {
	Field  *protogen.Field
	Format http.TimestampFormat
}

// hasTimestampFormatFields returns true if any Timestamp field in the message has a non-default format.
func hasTimestampFormatFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if annotations.IsTimestampField(field) && annotations.HasTimestampFormatAnnotation(field) {
			return true
		}
	}
	return false
}

// getTimestampFormatFields returns all Timestamp fields with non-default format annotations.
func getTimestampFormatFields(message *protogen.Message) []*TimestampFormatFieldInfo {
	var fields []*TimestampFormatFieldInfo
	for _, field := range message.Fields {
		if annotations.IsTimestampField(field) && annotations.HasTimestampFormatAnnotation(field) {
			fields = append(fields, &TimestampFormatFieldInfo{
				Field:  field,
				Format: annotations.GetTimestampFormat(field),
			})
		}
	}
	return fields
}

// collectTimestampFormatContext analyzes messages in a file and collects timestamp format info.
func collectTimestampFormatContext(file *protogen.File) []*TimestampFormatContext {
	var contexts []*TimestampFormatContext
	collectTimestampFormatMessages(file.Messages, &contexts)
	return contexts
}

// collectTimestampFormatMessages recursively collects messages with timestamp format fields.
func collectTimestampFormatMessages(messages []*protogen.Message, contexts *[]*TimestampFormatContext) {
	for _, msg := range messages {
		if hasTimestampFormatFields(msg) {
			*contexts = append(*contexts, &TimestampFormatContext{
				Message:         msg,
				TimestampFields: getTimestampFormatFields(msg),
			})
		}
		// Check nested messages
		collectTimestampFormatMessages(msg.Messages, contexts)
	}
}

// validateTimestampFormatAnnotations validates all timestamp_format annotations in a file.
// Returns the first validation error encountered, or nil if all valid.
func validateTimestampFormatAnnotations(file *protogen.File) error {
	return validateTimestampFormatInMessages(file.Messages)
}

// validateTimestampFormatInMessages recursively validates timestamp_format annotations.
func validateTimestampFormatInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if err := annotations.ValidateTimestampFormatAnnotation(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}
		if err := validateTimestampFormatInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// generateTimestampFormatEncodingFile generates the *_timestamp_format.pb.go file if needed.
func (g *Generator) generateTimestampFormatEncodingFile(file *protogen.File) error {
	// First validate all timestamp_format annotations
	if err := validateTimestampFormatAnnotations(file); err != nil {
		return err
	}

	contexts := collectTimestampFormatContext(file)
	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_timestamp_format.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeTimestampFormatImports(gf)

	for _, ctx := range contexts {
		g.generateTimestampFormatMarshalJSON(gf, ctx)
		g.generateTimestampFormatUnmarshalJSON(gf, ctx)
	}

	return nil
}

// writeTimestampFormatImports writes the imports needed for timestamp format encoding.
func (g *Generator) writeTimestampFormatImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P(`"time"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
}

// generateTimestampFormatMarshalJSON generates MarshalJSON that converts Timestamp fields to the specified format.
//
//nolint:dupl // Code generation patterns naturally have similar structure across encoding types
func (g *Generator) generateTimestampFormatMarshalJSON(gf *protogen.GeneratedFile, ctx *TimestampFormatContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.TimestampFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles timestamp_format fields: ", strings.Join(fieldNames, ", "))
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

	gf.P("// Parse into a map to modify timestamp format fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	for _, fieldInfo := range ctx.TimestampFields {
		g.generateTimestampFieldMarshal(gf, fieldInfo)
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()
}

// generateTimestampFieldMarshal generates marshal code for a single Timestamp field.
//
//nolint:exhaustive // Only non-default formats need handling; default/RFC3339 are excluded by HasTimestampFormatAnnotation
func (g *Generator) generateTimestampFieldMarshal(gf *protogen.GeneratedFile, fieldInfo *TimestampFormatFieldInfo) {
	field := fieldInfo.Field
	goName := field.GoName
	jsonName := field.Desc.JSONName()
	format := fieldInfo.Format

	gf.P("// Convert ", field.Desc.Name(), " to ", format.String(), " format")
	gf.P("if x.", goName, " != nil {")
	gf.P("t := x.", goName, ".AsTime()")

	switch format {
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS:
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(t.Unix())`)
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS:
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(t.UnixMilli())`)
	case http.TimestampFormat_TIMESTAMP_FORMAT_DATE:
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(t.Format("2006-01-02"))`)
	}

	gf.P("}")
	gf.P()
}

// generateTimestampFormatUnmarshalJSON generates UnmarshalJSON that converts timestamp formats back to RFC 3339.
//
//nolint:dupl // Code generation patterns naturally have similar structure across encoding types
func (g *Generator) generateTimestampFormatUnmarshalJSON(gf *protogen.GeneratedFile, ctx *TimestampFormatContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.TimestampFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles timestamp_format fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("// Parse the raw JSON to extract timestamp format fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	for _, fieldInfo := range ctx.TimestampFields {
		g.generateTimestampFieldUnmarshal(gf, fieldInfo)
	}

	gf.P("// Re-marshal with RFC 3339 values for protojson")
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

// generateTimestampFieldUnmarshal generates unmarshal code for a single Timestamp field.
//
//nolint:exhaustive // Only non-default formats need handling; default/RFC3339 are excluded by HasTimestampFormatAnnotation
func (g *Generator) generateTimestampFieldUnmarshal(gf *protogen.GeneratedFile, fieldInfo *TimestampFormatFieldInfo) {
	field := fieldInfo.Field
	jsonName := field.Desc.JSONName()
	format := fieldInfo.Format

	gf.P("// Convert ", jsonName, " from ", format.String(), " to RFC 3339 for protojson")
	gf.P(`if v, ok := raw["`, jsonName, `"]; ok {`)

	switch format {
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS:
		gf.P("var n int64")
		gf.P("if err := json.Unmarshal(v, &n); err == nil {")
		gf.P("t := time.Unix(n, 0)")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(t.Format(time.RFC3339Nano))`)
		gf.P("}")
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS:
		gf.P("var n int64")
		gf.P("if err := json.Unmarshal(v, &n); err == nil {")
		gf.P("t := time.UnixMilli(n)")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(t.Format(time.RFC3339Nano))`)
		gf.P("}")
	case http.TimestampFormat_TIMESTAMP_FORMAT_DATE:
		gf.P("var s string")
		gf.P("if err := json.Unmarshal(v, &s); err == nil {")
		gf.P(`t, parseErr := time.Parse("2006-01-02", s)`)
		gf.P("if parseErr == nil {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(t.Format(time.RFC3339Nano))`)
		gf.P("}")
		gf.P("}")
	}

	gf.P("}")
	gf.P()
}
