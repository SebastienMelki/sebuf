package clientgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// EnumEncodingContext holds information about enums that need custom JSON encoding.
type EnumEncodingContext struct {
	Enum              *protogen.Enum
	HasCustomValues   bool
	HasNumberEncoding bool
}

func validateEnumAnnotations(field *protogen.Field) error {
	if annotations.HasConflictingEnumAnnotations(field) {
		return fmt.Errorf(
			"field %s has both enum_encoding=NUMBER and enum_value annotations - this is not allowed",
			field.Desc.Name(),
		)
	}
	return nil
}

func collectEnumsWithCustomValues(file *protogen.File) []*EnumEncodingContext {
	var contexts []*EnumEncodingContext
	seen := make(map[string]bool)

	for _, enum := range file.Enums {
		if hasCustomEnumValues(enum) {
			fullName := string(enum.Desc.FullName())
			if !seen[fullName] {
				seen[fullName] = true
				contexts = append(contexts, &EnumEncodingContext{
					Enum:            enum,
					HasCustomValues: true,
				})
			}
		}
	}

	for _, msg := range file.Messages {
		collectEnumsFromMessage(msg, &contexts, seen)
	}

	return contexts
}

func collectEnumsFromMessage(msg *protogen.Message, contexts *[]*EnumEncodingContext, seen map[string]bool) {
	for _, enum := range msg.Enums {
		if hasCustomEnumValues(enum) {
			fullName := string(enum.Desc.FullName())
			if !seen[fullName] {
				seen[fullName] = true
				*contexts = append(*contexts, &EnumEncodingContext{
					Enum:            enum,
					HasCustomValues: true,
				})
			}
		}
	}

	for _, nested := range msg.Messages {
		collectEnumsFromMessage(nested, contexts, seen)
	}
}

func hasCustomEnumValues(enum *protogen.Enum) bool {
	return annotations.HasAnyEnumValueMapping(enum)
}

func (g *Generator) generateEnumEncodingFile(file *protogen.File) error {
	contexts := collectEnumsWithCustomValues(file)

	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_enum_encoding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeEncodingHeader(gf, file)
	g.writeEnumEncodingImports(gf)

	for _, ctx := range contexts {
		g.generateEnumLookupMaps(gf, ctx.Enum)
		g.generateEnumMarshalJSON(gf, ctx.Enum)
		g.generateEnumUnmarshalJSON(gf, ctx.Enum)
	}

	return nil
}

func (g *Generator) writeEnumEncodingImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P("\"encoding/json\"")
	gf.P("\"fmt\"")
	gf.P(")")
	gf.P()
}

func (g *Generator) generateEnumLookupMaps(gf *protogen.GeneratedFile, enum *protogen.Enum) {
	enumName := enum.GoIdent.GoName
	lowerName := annotations.LowerFirst(enumName)

	gf.P("var ", lowerName, "ToJSON = map[", enumName, "]string{")
	for _, value := range enum.Values {
		customValue := annotations.GetEnumValueMapping(value)
		jsonValue := customValue
		if jsonValue == "" {
			jsonValue = string(value.Desc.Name())
		}
		gf.P(value.GoIdent.GoName, ": \"", jsonValue, "\",")
	}
	gf.P("}")
	gf.P()

	gf.P("var ", lowerName, "FromJSON = map[string]", enumName, "{")
	for _, value := range enum.Values {
		customValue := annotations.GetEnumValueMapping(value)
		jsonValue := customValue
		if jsonValue == "" {
			jsonValue = string(value.Desc.Name())
		}
		gf.P("\"", jsonValue, "\": ", value.GoIdent.GoName, ",")
	}
	for _, value := range enum.Values {
		customValue := annotations.GetEnumValueMapping(value)
		if customValue != "" {
			protoName := string(value.Desc.Name())
			gf.P("\"", protoName, "\": ", value.GoIdent.GoName, ",")
		}
	}
	gf.P("}")
	gf.P()
}

func (g *Generator) generateEnumMarshalJSON(gf *protogen.GeneratedFile, enum *protogen.Enum) {
	enumName := enum.GoIdent.GoName
	lowerName := annotations.LowerFirst(enumName)

	gf.P("func (x ", enumName, ") MarshalJSON() ([]byte, error) {")
	gf.P("if s, ok := ", lowerName, "ToJSON[x]; ok {")
	gf.P("return json.Marshal(s)")
	gf.P("}")
	gf.P("return json.Marshal(x.String())")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateEnumUnmarshalJSON(gf *protogen.GeneratedFile, enum *protogen.Enum) {
	enumName := enum.GoIdent.GoName
	lowerName := annotations.LowerFirst(enumName)

	gf.P("func (x *", enumName, ") UnmarshalJSON(data []byte) error {")
	gf.P("var s string")
	gf.P("if err := json.Unmarshal(data, &s); err == nil {")
	gf.P("if v, ok := ", lowerName, "FromJSON[s]; ok {")
	gf.P("*x = v")
	gf.P("return nil")
	gf.P("}")
	gf.P("return fmt.Errorf(\"unknown ", enumName, " value: %q\", s)")
	gf.P("}")
	gf.P()
	gf.P("var n int32")
	gf.P("if err := json.Unmarshal(data, &n); err == nil {")
	gf.P("*x = ", enumName, "(n)")
	gf.P("return nil")
	gf.P("}")
	gf.P()
	gf.P("return fmt.Errorf(\"cannot unmarshal %s into ", enumName, "\", string(data))")
	gf.P("}")
	gf.P()
}

func (g *Generator) validateEnumAnnotationsInFile(file *protogen.File) error {
	for _, msg := range file.Messages {
		if err := g.validateEnumAnnotationsInMessage(msg); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) validateEnumAnnotationsInMessage(msg *protogen.Message) error {
	for _, field := range msg.Fields {
		if field.Desc.Kind() == protoreflect.EnumKind {
			if err := validateEnumAnnotations(field); err != nil {
				return err
			}
		}
	}

	for _, nested := range msg.Messages {
		if err := g.validateEnumAnnotationsInMessage(nested); err != nil {
			return err
		}
	}

	return nil
}
