package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// NullableValidationError represents an error in nullable annotation validation.
type NullableValidationError struct {
	MessageName string
	FieldName   string
	Reason      string
}

func (e *NullableValidationError) Error() string {
	return "invalid nullable annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// IsNullableField returns true if the field has nullable=true annotation.
func IsNullableField(field *protogen.Field) bool {
	options := field.Desc.Options()
	if options == nil {
		return false
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return false
	}

	ext := proto.GetExtension(fieldOptions, http.E_Nullable)
	if ext == nil {
		return false
	}

	nullable, ok := ext.(bool)
	return ok && nullable
}

// ValidateNullableAnnotation checks if nullable annotation is valid for a field.
// Returns error if nullable=true on a non-optional field or on a message field.
func ValidateNullableAnnotation(field *protogen.Field, messageName string) error {
	if !IsNullableField(field) {
		return nil // No annotation, nothing to validate
	}

	// Nullable only valid on proto3 optional fields
	if !field.Desc.HasOptionalKeyword() {
		return &NullableValidationError{
			MessageName: messageName,
			FieldName:   string(field.Desc.Name()),
			Reason:      "nullable annotation is only valid on proto3 optional fields",
		}
	}

	// Nullable only valid on primitive types (not messages)
	if field.Desc.Kind() == protoreflect.MessageKind {
		return &NullableValidationError{
			MessageName: messageName,
			FieldName:   string(field.Desc.Name()),
			Reason:      "nullable annotation is only valid on primitive fields, not message fields",
		}
	}

	return nil
}
