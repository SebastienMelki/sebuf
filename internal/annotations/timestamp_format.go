package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// TimestampFormatValidationError represents an error in timestamp_format annotation validation.
type TimestampFormatValidationError struct {
	MessageName string
	FieldName   string
	Reason      string
}

func (e *TimestampFormatValidationError) Error() string {
	return "invalid timestamp_format annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// GetTimestampFormat returns the timestamp format for a field.
// Returns TIMESTAMP_FORMAT_UNSPECIFIED if not set (callers should use protojson default: RFC3339).
func GetTimestampFormat(field *protogen.Field) http.TimestampFormat {
	options := field.Desc.Options()
	if options == nil {
		return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
	}

	ext := proto.GetExtension(fieldOptions, http.E_TimestampFormat)
	if ext == nil {
		return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
	}

	format, ok := ext.(http.TimestampFormat)
	if !ok {
		return http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED
	}

	return format
}

// HasTimestampFormatAnnotation returns true if the field has any non-default timestamp_format.
// Returns false for UNSPECIFIED and RFC3339 (both use protojson default behavior).
func HasTimestampFormatAnnotation(field *protogen.Field) bool {
	format := GetTimestampFormat(field)
	return format != http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED &&
		format != http.TimestampFormat_TIMESTAMP_FORMAT_RFC3339
}

// IsTimestampField returns true if the field is a google.protobuf.Timestamp message type.
func IsTimestampField(field *protogen.Field) bool {
	return field.Desc.Kind() == protoreflect.MessageKind &&
		field.Message != nil &&
		field.Message.Desc.FullName() == "google.protobuf.Timestamp"
}

// ValidateTimestampFormatAnnotation checks if timestamp_format is valid for a field.
// Returns error if used on non-Timestamp fields.
func ValidateTimestampFormatAnnotation(field *protogen.Field, messageName string) error {
	format := GetTimestampFormat(field)
	if format == http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED {
		return nil // No annotation, nothing to validate
	}

	if !IsTimestampField(field) {
		return &TimestampFormatValidationError{
			MessageName: messageName,
			FieldName:   string(field.Desc.Name()),
			Reason:      "timestamp_format annotation is only valid on google.protobuf.Timestamp fields",
		}
	}

	return nil
}
