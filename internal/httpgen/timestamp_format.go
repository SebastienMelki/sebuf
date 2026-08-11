package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

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

// validateTimestampFormatAnnotations validates all timestamp_format annotations in a file.
func validateTimestampFormatAnnotations(file *protogen.File) error {
	return validateTimestampFormatInMessages(file.Messages)
}

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

// generateTimestampFieldMarshal emits the field-level marshal transform for timestamp_format.
//
//nolint:exhaustive // Only non-default formats need handling; default/RFC3339 are excluded by HasTimestampFormatAnnotation
func (g *Generator) generateTimestampFieldMarshal(gf *protogen.GeneratedFile, fieldInfo *TimestampFormatFieldInfo) {
	field := fieldInfo.Field
	goName := field.GoName
	format := fieldInfo.Format

	gf.P("// Convert ", field.Desc.Name(), " to ", format.String(), " format")
	gf.P("if x.", goName, " != nil {")
	gf.P("t := x.", goName, ".AsTime()")

	switch format {
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS:
		gf.P("data, _ = json.Marshal(t.Unix())")
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS:
		gf.P("data, _ = json.Marshal(t.UnixMilli())")
	case http.TimestampFormat_TIMESTAMP_FORMAT_DATE:
		gf.P(`data, _ = json.Marshal(t.Format("2006-01-02"))`)
	}
	emitRawFieldSetForMarshalOptions(gf, field, "data")

	gf.P("}")
	gf.P()
}

// generateTimestampFieldUnmarshal emits the field-level unmarshal transform for timestamp_format.
//
//nolint:exhaustive // Only non-default formats need handling; default/RFC3339 are excluded by HasTimestampFormatAnnotation
func (g *Generator) generateTimestampFieldUnmarshal(gf *protogen.GeneratedFile, fieldInfo *TimestampFormatFieldInfo) {
	field := fieldInfo.Field
	format := fieldInfo.Format

	gf.P("// Convert ", field.Desc.Name(), " from ", format.String(), " to RFC 3339 for protojson")
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("if v, ok := raw[k]; ok {")

	switch format {
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS:
		gf.P("var n int64")
		gf.P("if err := json.Unmarshal(v, &n); err == nil {")
		gf.P("t := time.Unix(n, 0)")
		gf.P("raw[k], _ = json.Marshal(t.Format(time.RFC3339Nano))")
		gf.P("}")
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS:
		gf.P("var n int64")
		gf.P("if err := json.Unmarshal(v, &n); err == nil {")
		gf.P("t := time.UnixMilli(n)")
		gf.P("raw[k], _ = json.Marshal(t.Format(time.RFC3339Nano))")
		gf.P("}")
	case http.TimestampFormat_TIMESTAMP_FORMAT_DATE:
		gf.P("var s string")
		gf.P("if err := json.Unmarshal(v, &s); err == nil {")
		gf.P(`t, parseErr := time.Parse("2006-01-02", s)`)
		gf.P("if parseErr == nil {")
		gf.P("raw[k], _ = json.Marshal(t.Format(time.RFC3339Nano))")
		gf.P("}")
		gf.P("}")
	}

	gf.P("}")
	gf.P("}")
	gf.P()
}
