package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// UnwrapFieldInfo contains information about an unwrap field in a message.
type UnwrapFieldInfo struct {
	Field        *protogen.Field   // The field with unwrap=true
	ElementType  *protogen.Message // The element type of the repeated field (if message type)
	IsRootUnwrap bool              // True if this is a root-level unwrap (single field in message)
	IsMapField   bool              // True if the unwrap field is a map (only for root unwrap)
}

// UnwrapValidationError represents an error in unwrap annotation validation.
type UnwrapValidationError struct {
	MessageName string
	FieldName   string
	Reason      string
}

func (e *UnwrapValidationError) Error() string {
	return "invalid unwrap annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// HasUnwrapAnnotation checks if a field has the unwrap=true annotation.
func HasUnwrapAnnotation(field *protogen.Field) bool {
	options := field.Desc.Options()
	if options == nil {
		return false
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return false
	}

	ext := proto.GetExtension(fieldOptions, http.E_Unwrap)
	if ext == nil {
		return false
	}

	unwrap, ok := ext.(bool)
	return ok && unwrap
}

// GetUnwrapField returns the unwrap field info for a message, or nil if none exists.
// Returns an error if the annotation is invalid (e.g., on non-repeated/non-map field,
// multiple unwrap fields, or map-without-root-unwrap).
//
// Root-level unwrap: When a message has exactly one field with unwrap=true on a map or
// repeated field, the entire message serializes to just that field's value.
//
// Map-value unwrap: When a repeated field has unwrap=true and the message is used as a
// map value, the wrapper is collapsed to just the array.
func GetUnwrapField(message *protogen.Message) (*UnwrapFieldInfo, error) {
	var unwrapField *protogen.Field

	for _, field := range message.Fields {
		if !HasUnwrapAnnotation(field) {
			continue
		}

		// Validate: must be a repeated field or a map field
		isMap := field.Desc.IsMap()
		isList := field.Desc.IsList()
		if !isList && !isMap {
			return nil, &UnwrapValidationError{
				MessageName: string(message.Desc.Name()),
				FieldName:   string(field.Desc.Name()),
				Reason:      "unwrap annotation can only be used on repeated or map fields",
			}
		}

		// Validate: only one unwrap field per message
		if unwrapField != nil {
			return nil, &UnwrapValidationError{
				MessageName: string(message.Desc.Name()),
				FieldName:   string(field.Desc.Name()),
				Reason:      "only one field per message can have the unwrap annotation",
			}
		}

		unwrapField = field
	}

	if unwrapField == nil {
		return nil, nil //nolint:nilnil // nil,nil is intentional: no unwrap field exists, not an error
	}

	isMapField := unwrapField.Desc.IsMap()

	// Check for root-level unwrap: single field with unwrap annotation
	// Root unwrap is only valid when the message has exactly one field
	isRootUnwrap := len(message.Fields) == 1

	// For root unwrap on maps, validate that we're dealing with a map field
	// For non-root unwrap (map-value unwrap), the field must be a repeated field (not a map)
	if !isRootUnwrap && isMapField {
		return nil, &UnwrapValidationError{
			MessageName: string(message.Desc.Name()),
			FieldName:   string(unwrapField.Desc.Name()),
			Reason:      "map fields with unwrap annotation require the message to have exactly one field (root unwrap)",
		}
	}

	info := &UnwrapFieldInfo{
		Field:        unwrapField,
		IsRootUnwrap: isRootUnwrap,
		IsMapField:   isMapField,
	}

	// If the element type is a message, capture it
	// For repeated fields, this is the element message type
	// For map fields, we don't set ElementType here (handled separately in root unwrap logic)
	if unwrapField.Message != nil && !isMapField {
		info.ElementType = unwrapField.Message
	}

	return info, nil
}

// FindUnwrapField returns the unwrap-annotated repeated field in a message, or nil.
// This is the simple version without validation, used by tsclientgen and openapiv3
// when only the repeated unwrap field is needed (not maps or root unwrap).
func FindUnwrapField(message *protogen.Message) *protogen.Field {
	for _, field := range message.Fields {
		if HasUnwrapAnnotation(field) && field.Desc.IsList() {
			return field
		}
	}
	return nil
}

// IsRootUnwrap checks if a message has a single field with unwrap=true.
// A root unwrap means the entire message serializes as just the field's value.
func IsRootUnwrap(message *protogen.Message) bool {
	if len(message.Fields) != 1 {
		return false
	}
	return HasUnwrapAnnotation(message.Fields[0])
}
