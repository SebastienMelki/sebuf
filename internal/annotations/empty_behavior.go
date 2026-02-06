package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// EmptyBehaviorValidationError represents an error in empty_behavior annotation validation.
type EmptyBehaviorValidationError struct {
	MessageName string
	FieldName   string
	Reason      string
}

func (e *EmptyBehaviorValidationError) Error() string {
	return "invalid empty_behavior annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// GetEmptyBehavior returns the empty behavior for a field.
// Returns EMPTY_BEHAVIOR_UNSPECIFIED if not set (callers should treat as PRESERVE).
func GetEmptyBehavior(field *protogen.Field) http.EmptyBehavior {
	options := field.Desc.Options()
	if options == nil {
		return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
	}

	ext := proto.GetExtension(fieldOptions, http.E_EmptyBehavior)
	if ext == nil {
		return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
	}

	behavior, ok := ext.(http.EmptyBehavior)
	if !ok {
		return http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
	}

	return behavior
}

// HasEmptyBehaviorAnnotation returns true if the field has any empty_behavior annotation set
// (including explicit PRESERVE, NULL, or OMIT - not just UNSPECIFIED).
func HasEmptyBehaviorAnnotation(field *protogen.Field) bool {
	return GetEmptyBehavior(field) != http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED
}

// ValidateEmptyBehaviorAnnotation checks if empty_behavior annotation is valid for a field.
// Returns error if used on primitive, repeated, or map fields.
func ValidateEmptyBehaviorAnnotation(field *protogen.Field, messageName string) error {
	if !HasEmptyBehaviorAnnotation(field) {
		return nil // No annotation, nothing to validate
	}

	// Empty behavior only valid on message fields
	if field.Desc.Kind() != protoreflect.MessageKind {
		return &EmptyBehaviorValidationError{
			MessageName: messageName,
			FieldName:   string(field.Desc.Name()),
			Reason:      "empty_behavior annotation is only valid on message fields",
		}
	}

	// Not valid on repeated fields
	if field.Desc.IsList() {
		return &EmptyBehaviorValidationError{
			MessageName: messageName,
			FieldName:   string(field.Desc.Name()),
			Reason:      "empty_behavior annotation is not valid on repeated fields",
		}
	}

	// Not valid on map fields
	if field.Desc.IsMap() {
		return &EmptyBehaviorValidationError{
			MessageName: messageName,
			FieldName:   string(field.Desc.Name()),
			Reason:      "empty_behavior annotation is not valid on map fields",
		}
	}

	return nil
}
