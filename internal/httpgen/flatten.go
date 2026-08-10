package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// FlattenFieldInfo holds field info with its flatten prefix.
type FlattenFieldInfo struct {
	Field  *protogen.Field
	Prefix string // flatten_prefix value (may be empty)
}

// hasFlattenFields returns true if any field in the message has flatten annotation.
func hasFlattenFields(message *protogen.Message) bool {
	return annotations.HasFlattenFields(message)
}

// validateFlattenAnnotations validates all flatten annotations in a file.
// It keeps semantic validation only: field validity and flattened-name collisions.
func validateFlattenAnnotations(file *protogen.File) error {
	return validateFlattenInMessages(file.Messages)
}

func validateFlattenInMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if err := annotations.ValidateFlattenField(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}

		if hasFlattenFields(msg) {
			if err := annotations.ValidateFlattenCollisions(msg); err != nil {
				return err
			}
		}

		if err := validateFlattenInMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// generateFlattenFieldMarshal emits the field-level marshal transform for flatten.
// Forwards opts to child's MarshalJSONSebuf when available (annotation composability),
// otherwise uses opts.Marshal so server-configured options reach plain messages too.
func (g *Generator) generateFlattenFieldMarshal(gf *protogen.GeneratedFile, info *FlattenFieldInfo) {
	field := info.Field
	goName := field.GoName
	prefix := info.Prefix

	gf.P("// Flatten field: ", field.Desc.Name())
	gf.P("if x.", goName, " != nil {")
	emitRawFieldDeleteAllJSONKeys(gf, field)
	gf.P("// Forward opts to child's MarshalJSONSebuf when available (annotation composability)")
	gf.P("var childData []byte")
	gf.P("var childErr error")
	gf.P(
		"if m, ok := any(x.",
		goName,
		").(interface{ MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error) }); ok {",
	)
	gf.P("childData, childErr = m.MarshalJSONSebuf(opts)")
	gf.P("} else {")
	gf.P("childData, childErr = opts.Marshal(x.", goName, ")")
	gf.P("}")
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
