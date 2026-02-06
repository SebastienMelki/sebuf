package clientgen

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// FlattenContext holds information about messages that need custom JSON encoding
// for flatten-annotated fields.
type FlattenContext struct {
	// Message is the message that needs custom marshal/unmarshal
	Message *protogen.Message
	// FlattenInfos holds all flatten-annotated fields with their prefixes
	FlattenInfos []*FlattenFieldInfo
}

// FlattenFieldInfo holds field info with its flatten prefix.
type FlattenFieldInfo struct {
	Field  *protogen.Field
	Prefix string // flatten_prefix value (may be empty)
}

// hasFlattenFields returns true if any field in the message has flatten annotation.
func hasFlattenFields(message *protogen.Message) bool {
	return annotations.HasFlattenFields(message)
}

// getFlattenFieldInfos returns all flatten fields with their prefixes.
func getFlattenFieldInfos(message *protogen.Message) []*FlattenFieldInfo {
	var infos []*FlattenFieldInfo
	for _, field := range message.Fields {
		if annotations.IsFlattenField(field) {
			infos = append(infos, &FlattenFieldInfo{
				Field:  field,
				Prefix: annotations.GetFlattenPrefix(field),
			})
		}
	}
	return infos
}

// collectFlattenContexts collects flatten contexts from all messages in a file.
func collectFlattenContexts(file *protogen.File) []*FlattenContext {
	var contexts []*FlattenContext
	collectFlattenMessages(file.Messages, &contexts)
	return contexts
}

// collectFlattenMessages recursively collects messages with flatten fields.
func collectFlattenMessages(messages []*protogen.Message, contexts *[]*FlattenContext) {
	for _, msg := range messages {
		if hasFlattenFields(msg) {
			*contexts = append(*contexts, &FlattenContext{
				Message:      msg,
				FlattenInfos: getFlattenFieldInfos(msg),
			})
		}
		// Check nested messages
		collectFlattenMessages(msg.Messages, contexts)
	}
}

// validateFlattenAnnotations validates all flatten annotations in a file.
// Checks field validity, name collisions, and MarshalJSON conflicts.
func validateFlattenAnnotations(file *protogen.File) error {
	return validateFlattenInMessages(file.Messages)
}

// validateFlattenInMessages recursively validates flatten annotations.
func validateFlattenInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		// Validate each field
		for _, field := range msg.Fields {
			if err := annotations.ValidateFlattenField(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}

		// Validate collisions for messages with flatten fields
		if hasFlattenFields(msg) {
			if err := annotations.ValidateFlattenCollisions(msg); err != nil {
				return err
			}
			// Check for MarshalJSON conflicts
			if err := validateFlattenMarshalJSONConflict(msg); err != nil {
				return err
			}
		}

		// Validate nested messages
		if err := validateFlattenInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// validateFlattenMarshalJSONConflict checks if a message with flatten also has other
// encoding features that generate MarshalJSON. Only one feature can own MarshalJSON.
func validateFlattenMarshalJSONConflict(msg *protogen.Message) error {
	conflicts := detectMarshalJSONConflicts(msg)
	if len(conflicts) > 0 {
		return fmt.Errorf(
			"message %s has both flatten and %s -- "+
				"only one MarshalJSON-generating feature is supported per message",
			msg.GoIdent.GoName, strings.Join(conflicts, ", "),
		)
	}
	return nil
}

// detectMarshalJSONConflicts returns a list of other MarshalJSON-generating features
// present on the same message. Returns nil if no conflicts.
func detectMarshalJSONConflicts(msg *protogen.Message) []string {
	var conflicts []string

	for _, field := range msg.Fields {
		if annotations.IsFlattenField(field) {
			continue
		}
		if isInt64Type(field) && annotations.IsInt64NumberEncoding(field) {
			conflicts = append(conflicts, "int64_encoding=NUMBER")
		}
		if annotations.IsNullableField(field) {
			conflicts = append(conflicts, "nullable")
		}
		if annotations.HasEmptyBehaviorAnnotation(field) {
			conflicts = append(conflicts, "empty_behavior")
		}
		if annotations.IsTimestampField(field) && annotations.HasTimestampFormatAnnotation(field) {
			conflicts = append(conflicts, "timestamp_format")
		}
		if annotations.HasBytesEncodingAnnotation(field) {
			conflicts = append(conflicts, "bytes_encoding")
		}
	}

	return conflicts
}

// generateFlattenFile generates the *_flatten.pb.go file if needed.
func (g *Generator) generateFlattenFile(file *protogen.File) error {
	if err := validateFlattenAnnotations(file); err != nil {
		return err
	}

	contexts := collectFlattenContexts(file)
	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_flatten.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeFlattenImports(gf)

	for _, ctx := range contexts {
		g.generateFlattenMarshalJSON(gf, ctx)
		g.generateFlattenUnmarshalJSON(gf, ctx)
	}

	return nil
}

// writeFlattenImports writes the imports needed for flatten encoding.
func (g *Generator) writeFlattenImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
}

// generateFlattenMarshalJSON generates MarshalJSON that promotes flattened child fields to the parent level.
//
//nolint:dupl // Intentionally similar to oneof_discriminator MarshalJSON — both use protojson-then-manipulate pattern
func (g *Generator) generateFlattenMarshalJSON(gf *protogen.GeneratedFile, ctx *FlattenContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, info := range ctx.FlattenInfos {
		fieldNames = append(fieldNames, string(info.Field.Desc.Name()))
	}

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles flatten fields: ", strings.Join(fieldNames, ", "))
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

	gf.P("// Parse into a map to promote flattened child fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.FlattenInfos {
		g.generateFlattenFieldMarshal(gf, info)
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()
}

// generateFlattenFieldMarshal generates marshaling code for a single flattened field.
// Uses json.Marshal for the child to respect its own MarshalJSON (annotation composability).
func (g *Generator) generateFlattenFieldMarshal(gf *protogen.GeneratedFile, info *FlattenFieldInfo) {
	field := info.Field
	goName := field.GoName
	jsonName := field.Desc.JSONName()
	prefix := info.Prefix

	gf.P("// Flatten field: ", field.Desc.Name())
	gf.P("if x.", goName, " != nil {")
	gf.P(`delete(raw, "`, jsonName, `")`)
	gf.P("// Use json.Marshal to invoke child's MarshalJSON (annotation composability)")
	gf.P("childData, childErr := json.Marshal(x.", goName, ")")
	gf.P("if childErr != nil {")
	gf.P("return nil, childErr")
	gf.P("}")
	gf.P("var childRaw map[string]json.RawMessage")
	gf.P("if childErr = json.Unmarshal(childData, &childRaw); childErr != nil {")
	gf.P("return nil, childErr")
	gf.P("}")
	gf.P("for k, v := range childRaw {")
	if prefix != "" {
		gf.P(`raw["`, prefix, `" + k] = v`)
	} else {
		gf.P("raw[k] = v")
	}
	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateFlattenUnmarshalJSON generates UnmarshalJSON that extracts prefixed child fields
// and reconstructs the nested message.
func (g *Generator) generateFlattenUnmarshalJSON(gf *protogen.GeneratedFile, ctx *FlattenContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, info := range ctx.FlattenInfos {
		fieldNames = append(fieldNames, string(info.Field.Desc.Name()))
	}

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles flatten fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.FlattenInfos {
		g.generateFlattenFieldUnmarshal(gf, info)
	}

	gf.P("// Re-marshal remaining fields for protojson")
	gf.P("remaining, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("return protojson.Unmarshal(remaining, x)")
	gf.P("}")
	gf.P()
}

// generateFlattenFieldUnmarshal generates unmarshaling code for a single flattened field.
// It enumerates all child fields at generation time and extracts them from the parent map.
func (g *Generator) generateFlattenFieldUnmarshal(gf *protogen.GeneratedFile, info *FlattenFieldInfo) {
	field := info.Field
	goName := field.GoName
	prefix := info.Prefix

	if field.Message == nil {
		return
	}

	childMsg := field.Message
	childTypeName := childMsg.GoIdent.GoName

	gf.P("// Extract flattened child fields for: ", field.Desc.Name())
	gf.P("{")
	gf.P("childRaw := make(map[string]json.RawMessage)")

	// Enumerate all child fields at generation time
	for _, childField := range childMsg.Fields {
		childJSONName := childField.Desc.JSONName()
		flattenedKey := prefix + childJSONName

		gf.P(`if v, ok := raw["`, flattenedKey, `"]; ok {`)
		gf.P(`childRaw["`, childJSONName, `"] = v`)
		gf.P(`delete(raw, "`, flattenedKey, `")`)
		gf.P("}")
	}

	gf.P("if len(childRaw) > 0 {")
	gf.P("childData, childErr := json.Marshal(childRaw)")
	gf.P("if childErr != nil {")
	gf.P("return childErr")
	gf.P("}")
	gf.P("x.", goName, " = &", childTypeName, "{}")
	gf.P("// Use json.Unmarshal to invoke child's UnmarshalJSON (annotation composability)")
	gf.P("if childErr = json.Unmarshal(childData, x.", goName, "); childErr != nil {")
	gf.P("return childErr")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()
}
