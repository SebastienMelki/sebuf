package clientgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// NullableContext holds information about messages that need custom JSON encoding
// for nullable primitive fields.
type NullableContext struct {
	Message        *protogen.Message
	NullableFields []*protogen.Field
}

// hasNullableFields returns true if any field in the message has nullable=true.
func hasNullableFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if annotations.IsNullableField(field) {
			return true
		}
	}
	return false
}

// getNullableFields returns all fields with nullable=true annotation.
func getNullableFields(message *protogen.Message) []*protogen.Field {
	var fields []*protogen.Field
	for _, field := range message.Fields {
		if annotations.IsNullableField(field) {
			fields = append(fields, field)
		}
	}
	return fields
}

// collectNullableContext analyzes messages in a file and collects nullable field information.
func collectNullableContext(file *protogen.File) []*NullableContext {
	var contexts []*NullableContext
	collectNullableMessages(file.Messages, &contexts)
	return contexts
}

// collectNullableMessages recursively collects messages with nullable fields.
func collectNullableMessages(messages []*protogen.Message, contexts *[]*NullableContext) {
	for _, msg := range messages {
		if hasNullableFields(msg) {
			*contexts = append(*contexts, &NullableContext{
				Message:        msg,
				NullableFields: getNullableFields(msg),
			})
		}
		// Check nested messages
		collectNullableMessages(msg.Messages, contexts)
	}
}

// validateNullableAnnotations validates all nullable annotations in a file.
// Returns the first validation error encountered, or nil if all valid.
func validateNullableAnnotations(file *protogen.File) error {
	return validateNullableInMessages(file.Messages)
}

func validateNullableInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if err := annotations.ValidateNullableAnnotation(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}
		// Validate nested messages
		if err := validateNullableInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// generateNullableEncodingFile generates the *_nullable.pb.go file if needed.
func (g *Generator) generateNullableEncodingFile(file *protogen.File) error {
	// First validate all nullable annotations
	if err := validateNullableAnnotations(file); err != nil {
		return err
	}

	contexts := collectNullableContext(file)
	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_nullable.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeNullableImports(gf)

	for _, ctx := range contexts {
		g.generateNullableMarshalJSON(gf, ctx)
		g.generateNullableUnmarshalJSON(gf, ctx)
	}

	return nil
}

func (g *Generator) writeNullableImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
}

// generateNullableMarshalJSON generates MarshalJSON that emits null for unset nullable fields.
func (g *Generator) generateNullableMarshalJSON(gf *protogen.GeneratedFile, ctx *NullableContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.NullableFields {
		fieldNames = append(fieldNames, string(f.Desc.Name()))
	}

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles nullable fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()

	gf.P("// Use protojson for base serialization")
	gf.P("data, err := protojson.Marshal(x)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	gf.P("// Parse into a map to handle nullable fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	// For each nullable field, emit null when not set
	for _, field := range ctx.NullableFields {
		jsonName := field.Desc.JSONName()
		goName := field.GoName

		gf.P("// Handle nullable field: ", field.Desc.Name())
		gf.P("// proto3 optional + nullable=true: emit null when not set")
		gf.P("if x.", goName, " == nil {")
		gf.P(`raw["`, jsonName, `"] = []byte("null")`)
		gf.P("}")
		gf.P()
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()
}

// generateNullableUnmarshalJSON generates UnmarshalJSON that accepts null for nullable fields.
func (g *Generator) generateNullableUnmarshalJSON(gf *protogen.GeneratedFile, ctx *NullableContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.NullableFields {
		fieldNames = append(fieldNames, string(f.Desc.Name()))
	}

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles nullable fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("// Parse to check for explicit null values on nullable fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	// For nullable fields, remove explicit nulls before protojson unmarshal
	// protojson doesn't handle null for scalar optionals, so we remove them
	for _, field := range ctx.NullableFields {
		jsonName := field.Desc.JSONName()

		gf.P("// Handle nullable field: ", field.Desc.Name())
		gf.P("// Remove explicit null so protojson leaves field unset")
		gf.P(`if rawVal, ok := raw["`, jsonName, `"]; ok && string(rawVal) == "null" {`)
		gf.P(`delete(raw, "`, jsonName, `")`)
		gf.P("}")
		gf.P()
	}

	gf.P("// Re-marshal without nulls for protojson")
	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("return protojson.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()
}
