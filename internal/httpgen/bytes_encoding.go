package httpgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// BytesEncodingContext holds information about messages that need custom JSON encoding
// for bytes fields with non-default encoding (HEX, BASE64_RAW, BASE64URL, BASE64URL_RAW).
type BytesEncodingContext struct {
	// Message is the message that needs custom marshal/unmarshal
	Message *protogen.Message
	// BytesFields are fields with non-default bytes_encoding annotation
	BytesFields []*BytesEncodingFieldInfo
}

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

// getBytesEncodingFields returns all bytes fields with non-default encoding annotation.
func getBytesEncodingFields(message *protogen.Message) []*BytesEncodingFieldInfo {
	var fields []*BytesEncodingFieldInfo
	for _, field := range message.Fields {
		if field.Desc.Kind() == protoreflect.BytesKind && annotations.HasBytesEncodingAnnotation(field) {
			fields = append(fields, &BytesEncodingFieldInfo{
				Field:    field,
				Encoding: annotations.GetBytesEncoding(field),
			})
		}
	}
	return fields
}

// collectBytesEncodingContext analyzes messages in a file and collects bytes encoding information.
func collectBytesEncodingContext(file *protogen.File) []*BytesEncodingContext {
	var contexts []*BytesEncodingContext
	collectBytesEncodingMessages(file.Messages, &contexts)
	return contexts
}

// collectBytesEncodingMessages recursively collects messages with non-default bytes encoding fields.
func collectBytesEncodingMessages(messages []*protogen.Message, contexts *[]*BytesEncodingContext) {
	for _, msg := range messages {
		if hasBytesEncodingFields(msg) {
			*contexts = append(*contexts, &BytesEncodingContext{
				Message:     msg,
				BytesFields: getBytesEncodingFields(msg),
			})
		}
		// Check nested messages
		collectBytesEncodingMessages(msg.Messages, contexts)
	}
}

// validateBytesEncodingAnnotations validates all bytes_encoding annotations in a file.
// Returns the first validation error encountered, or nil if all valid.
func validateBytesEncodingAnnotations(file *protogen.File) error {
	return validateBytesEncodingInMessages(file.Messages)
}

// validateBytesEncodingInMessages recursively validates bytes_encoding annotations.
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

// generateBytesEncodingFile generates the *_bytes_encoding.pb.go file if needed.
func (g *Generator) generateBytesEncodingFile(file *protogen.File) error {
	if err := validateBytesEncodingAnnotations(file); err != nil {
		return err
	}

	contexts := collectBytesEncodingContext(file)
	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_bytes_encoding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeBytesEncodingImports(gf, contexts)

	for _, ctx := range contexts {
		g.generateBytesMarshalJSON(gf, ctx)
		g.generateBytesUnmarshalJSON(gf, ctx)
	}

	return nil
}

// writeBytesEncodingImports writes the imports needed for bytes encoding.
func (g *Generator) writeBytesEncodingImports(gf *protogen.GeneratedFile, contexts []*BytesEncodingContext) {
	needsBase64 := false
	needsHex := false

	for _, ctx := range contexts {
		for _, f := range ctx.BytesFields {
			//exhaustive:ignore -- only non-default encodings need imports; UNSPECIFIED/BASE64 are filtered out
			switch f.Encoding {
			case http.BytesEncoding_BYTES_ENCODING_BASE64_RAW,
				http.BytesEncoding_BYTES_ENCODING_BASE64URL,
				http.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW:
				needsBase64 = true
			case http.BytesEncoding_BYTES_ENCODING_HEX:
				needsHex = true
				needsBase64 = true // UnmarshalJSON re-encodes to standard base64 for protojson
			default:
				// No extra import needed
			}
		}
	}

	gf.P("import (")
	if needsBase64 {
		gf.P(`"encoding/base64"`)
	}
	if needsHex {
		gf.P(`"encoding/hex"`)
	}
	gf.P(`"encoding/json"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
}

// generateBytesMarshalJSON generates a MarshalJSON method that encodes bytes fields
// with the configured encoding (HEX, BASE64_RAW, BASE64URL, BASE64URL_RAW).
//
//nolint:dupl // Code generation patterns naturally have similar structure across encoding types
func (g *Generator) generateBytesMarshalJSON(gf *protogen.GeneratedFile, ctx *BytesEncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.BytesFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles bytes_encoding fields: ", strings.Join(fieldNames, ", "))
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

	gf.P("// Parse into a map to modify bytes-encoded fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	for _, fieldInfo := range ctx.BytesFields {
		g.generateBytesFieldMarshal(gf, fieldInfo)
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()
}

// generateBytesFieldMarshal generates marshaling code for a single bytes field.
func (g *Generator) generateBytesFieldMarshal(gf *protogen.GeneratedFile, fieldInfo *BytesEncodingFieldInfo) {
	field := fieldInfo.Field
	goName := field.GoName
	jsonName := field.Desc.JSONName()
	encoding := fieldInfo.Encoding

	gf.P("// Encode ", field.Desc.Name(), " with ", encoding.String())
	gf.P("if len(x.", goName, ") > 0 {")

	//exhaustive:ignore -- only non-default encodings reach here; UNSPECIFIED/BASE64 are filtered by hasBytesEncodingFields
	switch encoding {
	case http.BytesEncoding_BYTES_ENCODING_HEX:
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(hex.EncodeToString(x.`, goName, `))`)
	case http.BytesEncoding_BYTES_ENCODING_BASE64_RAW:
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(base64.RawStdEncoding.EncodeToString(x.`, goName, `))`)
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL:
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(base64.URLEncoding.EncodeToString(x.`, goName, `))`)
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW:
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(base64.RawURLEncoding.EncodeToString(x.`, goName, `))`)
	default:
		// Should not be reached since we only collect non-default encodings
	}

	gf.P("}")
	gf.P()
}

// generateBytesUnmarshalJSON generates an UnmarshalJSON method that decodes bytes fields
// from the configured encoding back to standard base64 for protojson.
//
//nolint:dupl // Code generation patterns naturally have similar structure across encoding types
func (g *Generator) generateBytesUnmarshalJSON(gf *protogen.GeneratedFile, ctx *BytesEncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.BytesFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles bytes_encoding fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("// Parse the raw JSON to extract bytes-encoded fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	for _, fieldInfo := range ctx.BytesFields {
		g.generateBytesFieldUnmarshal(gf, fieldInfo)
	}

	gf.P("// Re-marshal with standard base64 values for protojson")
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

// generateBytesFieldUnmarshal generates unmarshaling code for a single bytes field.
// It decodes from the configured encoding, then re-encodes as standard base64 for protojson.
func (g *Generator) generateBytesFieldUnmarshal(gf *protogen.GeneratedFile, fieldInfo *BytesEncodingFieldInfo) {
	field := fieldInfo.Field
	jsonName := field.Desc.JSONName()
	encoding := fieldInfo.Encoding

	gf.P("// Decode ", field.Desc.Name(), " from ", encoding.String(), " to standard base64")
	gf.P(`if v, ok := raw["`, jsonName, `"]; ok {`)
	gf.P("var s string")
	gf.P("if err := json.Unmarshal(v, &s); err == nil {")

	//exhaustive:ignore -- only non-default encodings reach here; UNSPECIFIED/BASE64 are filtered by hasBytesEncodingFields
	switch encoding {
	case http.BytesEncoding_BYTES_ENCODING_HEX:
		gf.P("decoded, decErr := hex.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))`)
		gf.P("}")
	case http.BytesEncoding_BYTES_ENCODING_BASE64_RAW:
		gf.P("decoded, decErr := base64.RawStdEncoding.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))`)
		gf.P("}")
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL:
		gf.P("decoded, decErr := base64.URLEncoding.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))`)
		gf.P("}")
	case http.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW:
		gf.P("decoded, decErr := base64.RawURLEncoding.DecodeString(s)")
		gf.P("if decErr == nil {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(base64.StdEncoding.EncodeToString(decoded))`)
		gf.P("}")
	default:
		// Should not be reached
	}

	gf.P("}")
	gf.P("}")
	gf.P()
}
