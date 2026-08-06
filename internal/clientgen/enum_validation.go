package clientgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

func validateEnumAnnotations(field *protogen.Field) error {
	if annotations.HasConflictingEnumAnnotations(field) {
		return fmt.Errorf(
			"field %s has both enum_encoding=NUMBER and enum_value annotations - this is not allowed",
			field.Desc.Name(),
		)
	}
	return nil
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
