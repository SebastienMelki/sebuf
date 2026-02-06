package annotations

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// IsFlattenField returns true if the field has flatten=true annotation.
func IsFlattenField(field *protogen.Field) bool {
	options := field.Desc.Options()
	if options == nil {
		return false
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return false
	}

	if !proto.HasExtension(fieldOptions, http.E_Flatten) {
		return false
	}

	ext := proto.GetExtension(fieldOptions, http.E_Flatten)

	flatten, ok := ext.(bool)
	return ok && flatten
}

// GetFlattenPrefix returns the flatten prefix for a field, or empty string if not set.
func GetFlattenPrefix(field *protogen.Field) string {
	options := field.Desc.Options()
	if options == nil {
		return ""
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return ""
	}

	if !proto.HasExtension(fieldOptions, http.E_FlattenPrefix) {
		return ""
	}

	ext := proto.GetExtension(fieldOptions, http.E_FlattenPrefix)

	prefix, ok := ext.(string)
	if !ok {
		return ""
	}

	return prefix
}

// HasFlattenFields returns true if any field in the message has a flatten annotation.
func HasFlattenFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if IsFlattenField(field) {
			return true
		}
	}
	return false
}

// ValidateFlattenField validates that flatten is used correctly on a field.
// Returns error if flatten is used on repeated, map, scalar, or oneof variant fields.
// Also returns error if flatten_prefix is set without flatten=true.
func ValidateFlattenField(field *protogen.Field, messageName string) error {
	isFlatten := IsFlattenField(field)
	prefix := GetFlattenPrefix(field)

	if !isFlatten && prefix != "" {
		return fmt.Errorf(
			"field %s.%s: flatten_prefix=%q set without flatten=true",
			messageName, field.Desc.Name(), prefix,
		)
	}

	if !isFlatten {
		return nil
	}

	if field.Desc.IsList() {
		return fmt.Errorf(
			"field %s.%s: flatten is not valid on repeated fields",
			messageName, field.Desc.Name(),
		)
	}

	if field.Desc.IsMap() {
		return fmt.Errorf(
			"field %s.%s: flatten is not valid on map fields",
			messageName, field.Desc.Name(),
		)
	}

	if field.Desc.Kind() != protoreflect.MessageKind {
		return fmt.Errorf(
			"field %s.%s: flatten is only valid on message fields (got %s)",
			messageName, field.Desc.Name(), field.Desc.Kind(),
		)
	}

	if field.Oneof != nil {
		return fmt.Errorf(
			"field %s.%s: flatten is not valid on oneof variant fields (use oneof_config.flatten instead)",
			messageName, field.Desc.Name(),
		)
	}

	return nil
}

// ValidateFlattenCollisions checks for field name collisions when multiple fields
// are flattened at the same level.
func ValidateFlattenCollisions(message *protogen.Message) error {
	usedNames := make(map[string]string) // json_name -> source description

	// First, register all non-flattened field JSON names
	for _, field := range message.Fields {
		if IsFlattenField(field) {
			continue
		}

		usedNames[field.Desc.JSONName()] = fmt.Sprintf("parent field %q", field.Desc.Name())
	}

	// Then check each flattened field's children
	for _, field := range message.Fields {
		if !IsFlattenField(field) || field.Message == nil {
			continue
		}

		prefix := GetFlattenPrefix(field)

		for _, childField := range field.Message.Fields {
			flattenedName := prefix + childField.Desc.JSONName()
			if source, exists := usedNames[flattenedName]; exists {
				return fmt.Errorf(
					"field %s.%s: flattened child %q (JSON: %q) collides with %s",
					string(message.Desc.Name()), field.Desc.Name(),
					childField.Desc.Name(), flattenedName, source,
				)
			}

			usedNames[flattenedName] = fmt.Sprintf(
				"flattened from %s.%s", field.Desc.Name(), childField.Desc.Name(),
			)
		}
	}

	return nil
}
