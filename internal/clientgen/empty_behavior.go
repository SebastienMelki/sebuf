package clientgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// EmptyBehaviorContext holds information about messages that need custom JSON encoding
// for empty_behavior fields.
type EmptyBehaviorContext struct {
	// Message is the message that needs custom marshal/unmarshal
	Message *protogen.Message
	// EmptyBehaviorFields are fields with empty_behavior annotation
	EmptyBehaviorFields []*EmptyBehaviorFieldInfo
}

// EmptyBehaviorFieldInfo holds field info with its empty behavior setting.
type EmptyBehaviorFieldInfo struct {
	Field    *protogen.Field
	Behavior http.EmptyBehavior
}

// hasEmptyBehaviorFields returns true if any message field has empty_behavior annotation.
func hasEmptyBehaviorFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if annotations.HasEmptyBehaviorAnnotation(field) {
			return true
		}
	}
	return false
}

// getEmptyBehaviorFields returns all message fields with empty_behavior annotation.
func getEmptyBehaviorFields(message *protogen.Message) []*EmptyBehaviorFieldInfo {
	var fields []*EmptyBehaviorFieldInfo
	for _, field := range message.Fields {
		if annotations.HasEmptyBehaviorAnnotation(field) {
			fields = append(fields, &EmptyBehaviorFieldInfo{
				Field:    field,
				Behavior: annotations.GetEmptyBehavior(field),
			})
		}
	}
	return fields
}

// collectEmptyBehaviorContext analyzes messages in a file and collects empty_behavior info.
func collectEmptyBehaviorContext(file *protogen.File) []*EmptyBehaviorContext {
	var contexts []*EmptyBehaviorContext
	collectEmptyBehaviorMessages(file.Messages, &contexts)
	return contexts
}

// collectEmptyBehaviorMessages recursively collects messages with empty_behavior fields.
func collectEmptyBehaviorMessages(messages []*protogen.Message, contexts *[]*EmptyBehaviorContext) {
	for _, msg := range messages {
		if hasEmptyBehaviorFields(msg) {
			*contexts = append(*contexts, &EmptyBehaviorContext{
				Message:             msg,
				EmptyBehaviorFields: getEmptyBehaviorFields(msg),
			})
		}
		// Check nested messages
		collectEmptyBehaviorMessages(msg.Messages, contexts)
	}
}

// validateEmptyBehaviorAnnotations validates all empty_behavior annotations in a file.
func validateEmptyBehaviorAnnotations(file *protogen.File) error {
	return validateEmptyBehaviorInMessages(file.Messages)
}

// validateEmptyBehaviorInMessages recursively validates empty_behavior annotations.
func validateEmptyBehaviorInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if err := annotations.ValidateEmptyBehaviorAnnotation(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}
		if err := validateEmptyBehaviorInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// generateEmptyBehaviorEncodingFile generates the *_empty_behavior.pb.go file if needed.
func (g *Generator) generateEmptyBehaviorEncodingFile(file *protogen.File) error {
	if err := validateEmptyBehaviorAnnotations(file); err != nil {
		return err
	}

	contexts := collectEmptyBehaviorContext(file)
	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_empty_behavior.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeEmptyBehaviorImports(gf)

	for _, ctx := range contexts {
		g.generateEmptyBehaviorMarshalJSON(gf, ctx)
		g.generateEmptyBehaviorUnmarshalJSON(gf, ctx)
	}

	return nil
}

// writeEmptyBehaviorImports writes the imports needed for empty_behavior encoding.
func (g *Generator) writeEmptyBehaviorImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(`"google.golang.org/protobuf/proto"`)
	gf.P(")")
	gf.P()
}

// generateEmptyBehaviorMarshalJSON generates MarshalJSON that handles empty message fields.
func (g *Generator) generateEmptyBehaviorMarshalJSON(gf *protogen.GeneratedFile, ctx *EmptyBehaviorContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.EmptyBehaviorFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles empty_behavior fields: ", strings.Join(fieldNames, ", "))
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

	gf.P("// Parse into a map to handle empty_behavior fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	// For each empty_behavior field, apply the configured behavior
	for _, fieldInfo := range ctx.EmptyBehaviorFields {
		g.generateEmptyBehaviorFieldMarshal(gf, fieldInfo)
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()
}

// generateEmptyBehaviorFieldMarshal generates marshaling code for a single empty_behavior field.
func (g *Generator) generateEmptyBehaviorFieldMarshal(gf *protogen.GeneratedFile, fieldInfo *EmptyBehaviorFieldInfo) {
	field := fieldInfo.Field
	jsonName := field.Desc.JSONName()
	goName := field.GoName
	behavior := fieldInfo.Behavior

	gf.P("// Handle empty_behavior for field: ", field.Desc.Name())
	gf.P("if x.", goName, " != nil && proto.Size(x.", goName, ") == 0 {")

	switch behavior {
	case http.EmptyBehavior_EMPTY_BEHAVIOR_NULL:
		gf.P("// EMPTY_BEHAVIOR_NULL: serialize empty message as null")
		gf.P(`raw["`, jsonName, `"] = []byte("null")`)
	case http.EmptyBehavior_EMPTY_BEHAVIOR_OMIT:
		gf.P("// EMPTY_BEHAVIOR_OMIT: remove field when message is empty")
		gf.P(`delete(raw, "`, jsonName, `")`)
	case http.EmptyBehavior_EMPTY_BEHAVIOR_PRESERVE:
		gf.P("// EMPTY_BEHAVIOR_PRESERVE: keep as {} (default protojson behavior)")
		gf.P("// No action needed - protojson already emits {}")
	case http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED:
		// UNSPECIFIED treated as PRESERVE
		gf.P("// EMPTY_BEHAVIOR_UNSPECIFIED: use default (PRESERVE)")
	}

	gf.P("}")
	gf.P()
}

// generateEmptyBehaviorUnmarshalJSON generates UnmarshalJSON that handles empty_behavior.
// For NULL behavior, accept null as empty message. For OMIT, missing field means empty.
func (g *Generator) generateEmptyBehaviorUnmarshalJSON(gf *protogen.GeneratedFile, ctx *EmptyBehaviorContext) {
	msgName := ctx.Message.GoIdent.GoName

	// Check if any field has NULL behavior (needs special handling)
	hasNullBehavior := false
	for _, f := range ctx.EmptyBehaviorFields {
		if f.Behavior == http.EmptyBehavior_EMPTY_BEHAVIOR_NULL {
			hasNullBehavior = true
			break
		}
	}

	if !hasNullBehavior {
		// No special unmarshal needed for PRESERVE/OMIT
		return
	}

	var fieldNames []string
	for _, f := range ctx.EmptyBehaviorFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles empty_behavior fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("// Parse to check for explicit null values on empty_behavior=NULL fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	// For NULL fields, convert null to empty object for protojson
	for _, fieldInfo := range ctx.EmptyBehaviorFields {
		if fieldInfo.Behavior == http.EmptyBehavior_EMPTY_BEHAVIOR_NULL {
			field := fieldInfo.Field
			jsonName := field.Desc.JSONName()

			gf.P("// Handle empty_behavior=NULL: convert null to {} for protojson")
			gf.P(`if rawVal, ok := raw["`, jsonName, `"]; ok && string(rawVal) == "null" {`)
			gf.P(`raw["`, jsonName, `"] = []byte("{}")`)
			gf.P("}")
			gf.P()
		}
	}

	gf.P("// Re-marshal for protojson")
	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("return protojson.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()
}
