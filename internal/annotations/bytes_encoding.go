package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// BytesEncodingValidationError represents an error in bytes_encoding annotation validation.
type BytesEncodingValidationError struct {
	MessageName string
	FieldName   string
	Reason      string
}

func (e *BytesEncodingValidationError) Error() string {
	return "invalid bytes_encoding annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}

// GetBytesEncoding returns the bytes encoding for a field.
// Returns BYTES_ENCODING_UNSPECIFIED if not set (callers should use protojson default: BASE64).
func GetBytesEncoding(field *protogen.Field) http.BytesEncoding {
	options := field.Desc.Options()
	if options == nil {
		return http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED
	}

	ext := proto.GetExtension(fieldOptions, http.E_BytesEncoding)
	if ext == nil {
		return http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED
	}

	encoding, ok := ext.(http.BytesEncoding)
	if !ok {
		return http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED
	}

	return encoding
}

// HasBytesEncodingAnnotation returns true if the field has any non-default bytes_encoding.
// Returns false for UNSPECIFIED and BASE64 (both use protojson default behavior).
func HasBytesEncodingAnnotation(field *protogen.Field) bool {
	encoding := GetBytesEncoding(field)
	return encoding != http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED &&
		encoding != http.BytesEncoding_BYTES_ENCODING_BASE64
}

// ValidateBytesEncodingAnnotation checks if bytes_encoding is valid for a field.
// Returns error if used on non-bytes fields.
func ValidateBytesEncodingAnnotation(field *protogen.Field, messageName string) error {
	encoding := GetBytesEncoding(field)
	if encoding == http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED {
		return nil // No annotation, nothing to validate
	}

	if field.Desc.Kind() != protoreflect.BytesKind {
		return &BytesEncodingValidationError{
			MessageName: messageName,
			FieldName:   string(field.Desc.Name()),
			Reason:      "bytes_encoding annotation is only valid on bytes fields",
		}
	}

	return nil
}
