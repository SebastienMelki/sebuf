package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// hasOneofDiscriminator returns true if any oneof in the message has a discriminator annotation.
func hasOneofDiscriminator(message *protogen.Message) bool {
	return annotations.HasOneofDiscriminator(message)
}

// validateOneofDiscriminatorAnnotations validates all oneof discriminator annotations in a file.
func validateOneofDiscriminatorAnnotations(file *protogen.File) error {
	return validateOneofDiscriminatorInMessages(file.Messages)
}

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

// generateOneofMarshalVariants emits the field-level marshal transform for a discriminated oneof.
func (g *Generator) generateOneofMarshalVariants(gf *protogen.GeneratedFile, info *annotations.OneofDiscriminatorInfo) {
	oneofGoName := info.Oneof.GoName

	gf.P("// Handle oneof ", info.Oneof.Desc.Name(), " with discriminator \"", info.Discriminator, "\"")
	gf.P("switch x.Get", oneofGoName, "().(type) {")

	for _, variant := range info.Variants {
		wrapperType := variant.Field.GoIdent.GoName
		gf.P("case *", wrapperType, ":")
		gf.P(`raw["`, info.Discriminator, `"], _ = json.Marshal("`, variant.DiscriminatorVal, `")`)

		if info.Flatten && variant.IsMessage {
			g.generateFlattenedMarshal(gf, variant)
		}
	}

	gf.P("default:")
	gf.P("// Oneof not set: omit discriminator entirely")
	gf.P("}")
	gf.P()
}

// generateFlattenedMarshal emits flattening for a single oneof variant.
func (g *Generator) generateFlattenedMarshal(
	gf *protogen.GeneratedFile,
	variant annotations.OneofVariant,
) {
	fieldGoName := variant.Field.GoName
	fieldJSONName := variant.Field.Desc.JSONName()

	gf.P("// Flatten: forward opts to variant via MarshalJSONSebuf when available")
	gf.P("if inner := x.Get", fieldGoName, "(); inner != nil {")
	gf.P("var variantData []byte")
	gf.P("var varErr error")
	gf.P(
		"if m, ok := any(inner).(interface{ MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error) }); ok {",
	)
	gf.P("variantData, varErr = m.MarshalJSONSebuf(opts)")
	gf.P("} else {")
	gf.P("variantData, varErr = opts.Marshal(inner)")
	gf.P("}")
	gf.P("if varErr == nil {")
	gf.P("var variantMap map[string]json.RawMessage")
	gf.P("if json.Unmarshal(variantData, &variantMap) == nil {")
	gf.P("// Merge variant fields into parent")
	gf.P("for fk, fv := range variantMap {")
	gf.P("raw[fk] = fv")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P("delete(raw, \"", fieldJSONName, "\")")
	gf.P("}")
}

// generateOneofUnmarshalVariants emits the field-level unmarshal transform for a discriminated oneof.
func (g *Generator) generateOneofUnmarshalVariants(
	gf *protogen.GeneratedFile,
	info *annotations.OneofDiscriminatorInfo,
) {
	gf.P("// Read discriminator for oneof ", info.Oneof.Desc.Name())
	gf.P(`if discRaw, ok := raw["`, info.Discriminator, `"]; ok {`)
	gf.P("var disc string")
	gf.P("if err := json.Unmarshal(discRaw, &disc); err != nil {")
	gf.P(`return fmt.Errorf("invalid discriminator %q: %w", "`, info.Discriminator, `", err)`)
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
	}

	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateFlattenedUnmarshal emits flatten inversion for a single oneof variant.
func (g *Generator) generateFlattenedUnmarshal(
	gf *protogen.GeneratedFile,
	variant annotations.OneofVariant,
	info *annotations.OneofDiscriminatorInfo,
) {
	fieldGoName := variant.Field.GoName
	wrapperType := variant.Field.GoIdent.GoName
	msgType := variant.Field.Message.GoIdent.GoName
	fieldJSONName := variant.Field.Desc.JSONName()

	var childJSONNames []string
	for _, childField := range variant.Field.Message.Fields {
		childJSONNames = append(childJSONNames, childField.Desc.JSONName())
	}

	gf.P("// Flatten unmarshal: extract ", fieldGoName, " fields from flat map")
	gf.P("variantMap := make(map[string]json.RawMessage)")

	for _, childJSON := range childJSONNames {
		gf.P(`if fv, exists := raw["`, childJSON, `"]; exists {`)
		gf.P(`variantMap["`, childJSON, `"] = fv`)
		gf.P(`delete(raw, "`, childJSON, `")`)
		gf.P("}")
	}

	gf.P("variantData, _ := json.Marshal(variantMap)")
	gf.P("variant := &", msgType, "{}")
	gf.P("if err := json.Unmarshal(variantData, variant); err != nil {")
	gf.P(`return fmt.Errorf("failed to unmarshal variant %s: %w", "`, fieldGoName, `", err)`)
	gf.P("}")
	gf.P("x.", info.Oneof.GoName, " = &", wrapperType, "{", fieldGoName, ": variant}")
	gf.P(`raw["`, fieldJSONName, `"], _ = json.Marshal(variant)`)
}

// generateNestedUnmarshal emits non-flattened inversion for a message oneof variant.
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
	gf.P(`return fmt.Errorf("failed to unmarshal variant %s: %w", "`, fieldGoName, `", err)`)
	gf.P("}")
	gf.P("x.", info.Oneof.GoName, " = &", wrapperType, "{", fieldGoName, ": variant}")
	gf.P("}")
}
