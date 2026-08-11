package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// BytesEncodingFieldInfo holds field info with its bytes encoding setting.
type BytesEncodingFieldInfo struct {
	Field    *protogen.Field
	Encoding http.BytesEncoding
}

// hasBytesEncodingFields returns true if any bytes field in the message has non-default encoding.
func hasBytesEncodingFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if field.Desc.Kind() == protoreflect.BytesKind && annotations.HasBytesEncodingAnnotation(field) {
			return true
		}
	}
	return false
}

// validateBytesEncodingAnnotations validates all bytes_encoding annotations in a file.
func validateBytesEncodingAnnotations(file *protogen.File) error {
	return validateBytesEncodingInMessages(file.Messages)
}

func validateBytesEncodingInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if err := annotations.ValidateBytesEncodingAnnotation(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}
		if err := validateBytesEncodingInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// generateBytesFieldMarshal emits the field-level marshal transform for bytes_encoding.
func (g *Generator) generateBytesFieldMarshal(gf *protogen.GeneratedFile, fieldInfo *BytesEncodingFieldInfo) {
	field := fieldInfo.Field
	goName := field.GoName
	encoding := fieldInfo.Encoding

	gf.P("// Encode ", field.Desc.Name(), " with ", encoding.String())
	gf.P("if len(x.", goName, ") > 0 {")

	//exhaustive:ignore -- only non-default encodings reach here; UNSPECIFIED/BASE64 are filtered by hasBytesEncodingFields
	switch encoding {
	case http.BytesEncoding_BYTES_ENCODING_HEX:
		gf.P("data, _ = json.Marshal(hex.EncodeToString(x.", goName, "))")
	case http.BytesEncoding_BYTES_ENCODING_BASE64_RAW:
		gf.P("data, _ = json.Marshal(base64.RawStdEncoding.EncodeToString(x.", goName, "))")
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL:
		gf.P("data, _ = json.Marshal(base64.URLEncoding.EncodeToString(x.", goName, "))")
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW:
		gf.P("data, _ = json.Marshal(base64.RawURLEncoding.EncodeToString(x.", goName, "))")
	default:
		// Should not be reached since we only collect non-default encodings.
	}
	emitRawFieldSetForMarshalOptions(gf, field, "data")

	gf.P("}")
	gf.P()
}

// generateBytesFieldUnmarshal emits the field-level unmarshal transform for bytes_encoding.
func (g *Generator) generateBytesFieldUnmarshal(gf *protogen.GeneratedFile, fieldInfo *BytesEncodingFieldInfo) {
	field := fieldInfo.Field
	encoding := fieldInfo.Encoding

	gf.P("// Decode ", field.Desc.Name(), " from ", encoding.String(), " to standard base64")
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("if v, ok := raw[k]; ok {")
	gf.P("var s string")
	gf.P("if err := json.Unmarshal(v, &s); err == nil {")

	//exhaustive:ignore -- only non-default encodings reach here; UNSPECIFIED/BASE64 are filtered by hasBytesEncodingFields
	switch encoding {
	case http.BytesEncoding_BYTES_ENCODING_HEX:
		gf.P("decoded, decErr := hex.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P("raw[k], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))")
		gf.P("}")
	case http.BytesEncoding_BYTES_ENCODING_BASE64_RAW:
		gf.P("decoded, decErr := base64.RawStdEncoding.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P("raw[k], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))")
		gf.P("}")
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL:
		gf.P("decoded, decErr := base64.URLEncoding.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P("raw[k], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))")
		gf.P("}")
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW:
		gf.P("decoded, decErr := base64.RawURLEncoding.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P("raw[k], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))")
		gf.P("}")
	default:
		// Should not be reached.
	}

	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()
}
