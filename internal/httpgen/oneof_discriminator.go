package httpgen

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// OneofDiscriminatorContext holds information about a message that needs custom JSON encoding
// for oneof fields with discriminator annotations.
type OneofDiscriminatorContext struct {
	// Message is the message that needs custom marshal/unmarshal
	Message *protogen.Message
	// Oneofs are the annotated oneof fields with their resolved discriminator info
	Oneofs []*annotations.OneofDiscriminatorInfo
}

// hasOneofDiscriminator returns true if any oneof in the message has a discriminator annotation.
func hasOneofDiscriminator(message *protogen.Message) bool {
	return annotations.HasOneofDiscriminator(message)
}

// collectOneofDiscriminatorContext analyzes messages in a file and collects oneof discriminator info.
func collectOneofDiscriminatorContext(file *protogen.File) []*OneofDiscriminatorContext {
	var contexts []*OneofDiscriminatorContext
	collectOneofDiscriminatorMessages(file.Messages, &contexts)
	return contexts
}

// collectOneofDiscriminatorMessages recursively collects messages with oneof discriminator annotations.
func collectOneofDiscriminatorMessages(messages []*protogen.Message, contexts *[]*OneofDiscriminatorContext) {
	for _, msg := range messages {
		if hasOneofDiscriminator(msg) {
			var oneofs []*annotations.OneofDiscriminatorInfo
			for _, oneof := range msg.Oneofs {
				info := annotations.GetOneofDiscriminatorInfo(oneof)
				if info != nil {
					oneofs = append(oneofs, info)
				}
			}
			if len(oneofs) > 0 {
				*contexts = append(*contexts, &OneofDiscriminatorContext{
					Message: msg,
					Oneofs:  oneofs,
				})
			}
		}
		// Check nested messages
		collectOneofDiscriminatorMessages(msg.Messages, contexts)
	}
}

// validateOneofDiscriminatorAnnotations validates all oneof discriminator annotations in a file.
// Returns the first validation error encountered, or nil if all valid.
func validateOneofDiscriminatorAnnotations(file *protogen.File) error {
	return validateOneofDiscriminatorInMessages(file.Messages)
}

// validateOneofDiscriminatorInMessages recursively validates oneof discriminator annotations.
func validateOneofDiscriminatorInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, oneof := range msg.Oneofs {
			config := annotations.GetOneofConfig(oneof)
			if config == nil {
				continue
			}
			if err := annotations.ValidateOneofDiscriminator(msg, oneof, config); err != nil {
				return err
			}
		}
		if err := validateOneofDiscriminatorInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// checkMarshalJSONConflict checks whether a message that needs oneof MarshalJSON
// also requires MarshalJSON from another encoding feature (conflict = generation error).
func checkMarshalJSONConflict(message *protogen.Message) error {
	var conflicts []string

	if hasInt64NumberFields(message) {
		conflicts = append(conflicts, "int64_encoding=NUMBER")
	}
	if hasNullableFields(message) {
		conflicts = append(conflicts, "nullable")
	}
	if hasEmptyBehaviorFields(message) {
		conflicts = append(conflicts, "empty_behavior")
	}
	if hasTimestampFormatFields(message) {
		conflicts = append(conflicts, "timestamp_format")
	}
	if hasBytesEncodingFields(message) {
		conflicts = append(conflicts, "bytes_encoding")
	}

	if len(conflicts) > 0 {
		return fmt.Errorf(
			"message %s: oneof_config requires MarshalJSON but conflicts with %s (also requires MarshalJSON)",
			message.GoIdent.GoName, strings.Join(conflicts, ", "),
		)
	}

	return nil
}

// generateOneofDiscriminatorFile generates the *_oneof_discriminator.pb.go file if needed.
//

func (g *Generator) generateOneofDiscriminatorFile(file *protogen.File) error {
	// Validate annotations first
	if err := validateOneofDiscriminatorAnnotations(file); err != nil {
		return err
	}

	contexts := collectOneofDiscriminatorContext(file)
	if len(contexts) == 0 {
		return nil
	}

	// Check for MarshalJSON conflicts
	for _, ctx := range contexts {
		if err := checkMarshalJSONConflict(ctx.Message); err != nil {
			return err
		}
	}

	filename := file.GeneratedFilenamePrefix + "_oneof_discriminator.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeOneofDiscriminatorImports(gf)

	for _, ctx := range contexts {
		g.generateOneofMarshalJSON(gf, ctx)
		g.generateOneofUnmarshalJSON(gf, ctx)
	}

	return nil
}

// writeOneofDiscriminatorImports writes the imports needed for oneof discriminator encoding.
func (g *Generator) writeOneofDiscriminatorImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P(`"fmt"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
}

// generateOneofMarshalJSON generates MarshalJSON that adds discriminator fields for annotated oneofs.
//
//nolint:dupl // Code generation patterns naturally have similar structure across encoding types
func (g *Generator) generateOneofMarshalJSON(gf *protogen.GeneratedFile, ctx *OneofDiscriminatorContext) {
	msgName := ctx.Message.GoIdent.GoName

	var oneofNames []string
	for _, info := range ctx.Oneofs {
		oneofNames = append(oneofNames, string(info.Oneof.Desc.Name()))
	}

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles oneof discriminator fields: ", strings.Join(oneofNames, ", "))
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

	gf.P("// Parse into a map to add discriminator fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.Oneofs {
		g.generateOneofMarshalVariants(gf, info)
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()
}

// generateOneofMarshalVariants generates the marshal switch logic for a single discriminated oneof.
func (g *Generator) generateOneofMarshalVariants(gf *protogen.GeneratedFile, info *annotations.OneofDiscriminatorInfo) {
	oneofGoName := info.Oneof.GoName

	gf.P("// Handle oneof ", info.Oneof.Desc.Name(), " with discriminator \"", info.Discriminator, "\"")
	gf.P("switch x.Get", oneofGoName, "().(type) {")

	for _, variant := range info.Variants {
		wrapperType := variant.Field.GoIdent.GoName
		gf.P("case *", wrapperType, ":")

		// Add discriminator value
		gf.P(`raw["`, info.Discriminator, `"], _ = json.Marshal("`, variant.DiscriminatorVal, `")`)

		if info.Flatten && variant.IsMessage {
			g.generateFlattenedMarshal(gf, variant)
		}
		// Non-flattened: protojson already puts variant under its field name, just add discriminator
	}

	gf.P("default:")
	gf.P("// Oneof not set: omit discriminator entirely")
	gf.P("}")
	gf.P()
}

// generateFlattenedMarshal generates flattened marshal code for a single variant.
// It merges the variant's child fields into the parent map and removes the wrapper key.
func (g *Generator) generateFlattenedMarshal(
	gf *protogen.GeneratedFile,
	variant annotations.OneofVariant,
) {
	fieldGoName := variant.Field.GoName
	fieldJSONName := variant.Field.Desc.JSONName()

	gf.P("// Flatten: marshal variant via json.Marshal to invoke child MarshalJSON")
	gf.P("if inner := x.Get", fieldGoName, "(); inner != nil {")
	gf.P("variantData, varErr := json.Marshal(inner)")
	gf.P("if varErr == nil {")
	gf.P("var variantMap map[string]json.RawMessage")
	gf.P("if json.Unmarshal(variantData, &variantMap) == nil {")
	gf.P("// Merge variant fields into parent")
	gf.P("for fk, fv := range variantMap {")
	gf.P("raw[fk] = fv")
	gf.P("}")
	gf.P("}")
	gf.P("}")

	// Remove the wrapper key that protojson added
	gf.P("delete(raw, \"", fieldJSONName, "\")")
	gf.P("}")
}

// generateOneofUnmarshalJSON generates UnmarshalJSON that reads discriminator fields
// and routes to the correct variant.
//

func (g *Generator) generateOneofUnmarshalJSON(gf *protogen.GeneratedFile, ctx *OneofDiscriminatorContext) {
	msgName := ctx.Message.GoIdent.GoName

	var oneofNames []string
	for _, info := range ctx.Oneofs {
		oneofNames = append(oneofNames, string(info.Oneof.Desc.Name()))
	}

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles oneof discriminator fields: ", strings.Join(oneofNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("// Parse into a map to read discriminator fields")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.Oneofs {
		g.generateOneofUnmarshalVariants(gf, info)
	}

	gf.P("// Remove discriminator fields before protojson unmarshal")
	for _, info := range ctx.Oneofs {
		gf.P(`delete(raw, "`, info.Discriminator, `")`)
	}
	gf.P()

	gf.P("// Re-marshal remaining fields for protojson")
	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("return protojson.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()
}

// generateOneofUnmarshalVariants generates the unmarshal switch logic for a single discriminated oneof.
//

func (g *Generator) generateOneofUnmarshalVariants(
	gf *protogen.GeneratedFile,
	info *annotations.OneofDiscriminatorInfo,
) {
	gf.P("// Read discriminator for oneof ", info.Oneof.Desc.Name())
	gf.P(`if discRaw, ok := raw["`, info.Discriminator, `"]; ok {`)
	gf.P("var disc string")
	gf.P("if err := json.Unmarshal(discRaw, &disc); err != nil {")
	gf.P(`return fmt.Errorf("invalid discriminator %q: %%w", "`, info.Discriminator, `", err)`)
	gf.P("}")
	gf.P()

	gf.P("switch disc {")

	for _, variant := range info.Variants {
		gf.P(`case "`, variant.DiscriminatorVal, `":`)

		if info.Flatten && variant.IsMessage {
			g.generateFlattenedUnmarshal(gf, variant, info)
		} else if variant.IsMessage {
			g.generateNestedUnmarshal(gf, variant, info)
		}
		// Scalar variants in non-flattened mode: protojson handles them
	}

	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateFlattenedUnmarshal generates flattened unmarshal code for a single variant.
// It extracts variant fields from the flat map, constructs the variant, and sets it.
func (g *Generator) generateFlattenedUnmarshal(
	gf *protogen.GeneratedFile,
	variant annotations.OneofVariant,
	info *annotations.OneofDiscriminatorInfo,
) {
	fieldGoName := variant.Field.GoName
	wrapperType := variant.Field.GoIdent.GoName
	msgType := variant.Field.Message.GoIdent.GoName
	fieldJSONName := variant.Field.Desc.JSONName()

	// Collect all child field JSON names for this variant
	var childJSONNames []string
	for _, childField := range variant.Field.Message.Fields {
		childJSONNames = append(childJSONNames, childField.Desc.JSONName())
	}

	gf.P("// Flatten unmarshal: extract ", fieldGoName, " fields from flat map")
	gf.P("variantMap := make(map[string]json.RawMessage)")

	// Move child fields from the parent map into the variant map
	for _, childJSON := range childJSONNames {
		gf.P(`if fv, exists := raw["`, childJSON, `"]; exists {`)
		gf.P(`variantMap["`, childJSON, `"] = fv`)
		gf.P(`delete(raw, "`, childJSON, `")`)
		gf.P("}")
	}

	gf.P("variantData, _ := json.Marshal(variantMap)")
	gf.P("variant := &", msgType, "{}")
	gf.P("if err := json.Unmarshal(variantData, variant); err != nil {")
	gf.P(`return fmt.Errorf("failed to unmarshal variant %s: %%w", "`, fieldGoName, `", err)`)
	gf.P("}")
	gf.P("x.", info.Oneof.GoName, " = &", wrapperType, "{", fieldGoName, ": variant}")

	// Add the variant back to raw under its original field name for protojson
	// (protojson expects the oneof wrapper format)
	gf.P(`raw["`, fieldJSONName, `"], _ = json.Marshal(variant)`)
}

// generateNestedUnmarshal generates non-flattened unmarshal code for a message variant.
// For non-flattened mode, the variant is already nested under its field name.
// We use json.Unmarshal to invoke the child's UnmarshalJSON if it has one.
func (g *Generator) generateNestedUnmarshal(
	gf *protogen.GeneratedFile,
	variant annotations.OneofVariant,
	info *annotations.OneofDiscriminatorInfo,
) {
	fieldGoName := variant.Field.GoName
	fieldJSONName := variant.Field.Desc.JSONName()
	wrapperType := variant.Field.GoIdent.GoName
	msgType := variant.Field.Message.GoIdent.GoName

	gf.P("// Non-flattened unmarshal: use json.Unmarshal for child UnmarshalJSON support")
	gf.P(`if variantRaw, exists := raw["`, fieldJSONName, `"]; exists {`)
	gf.P("variant := &", msgType, "{}")
	gf.P("if err := json.Unmarshal(variantRaw, variant); err != nil {")
	gf.P(`return fmt.Errorf("failed to unmarshal variant %s: %%w", "`, fieldGoName, `", err)`)
	gf.P("}")
	gf.P("x.", info.Oneof.GoName, " = &", wrapperType, "{", fieldGoName, ": variant}")
	gf.P("}")
}
