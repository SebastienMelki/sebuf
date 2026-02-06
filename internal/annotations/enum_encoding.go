package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// GetEnumEncoding returns the enum encoding for a field.
// Returns ENUM_ENCODING_UNSPECIFIED if not set (callers should use protojson default: STRING names).
// This annotation is only valid on enum fields.
func GetEnumEncoding(field *protogen.Field) http.EnumEncoding {
	options := field.Desc.Options()
	if options == nil {
		return http.EnumEncoding_ENUM_ENCODING_UNSPECIFIED
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return http.EnumEncoding_ENUM_ENCODING_UNSPECIFIED
	}

	ext := proto.GetExtension(fieldOptions, http.E_EnumEncoding)
	if ext == nil {
		return http.EnumEncoding_ENUM_ENCODING_UNSPECIFIED
	}

	encoding, ok := ext.(http.EnumEncoding)
	if !ok {
		return http.EnumEncoding_ENUM_ENCODING_UNSPECIFIED
	}

	return encoding
}

// GetEnumValueMapping returns the custom JSON value for an enum value, or empty string if not set.
// When set, this value should be used instead of the proto name for JSON serialization.
func GetEnumValueMapping(value *protogen.EnumValue) string {
	options := value.Desc.Options()
	if options == nil {
		return ""
	}

	enumValueOptions, ok := options.(*descriptorpb.EnumValueOptions)
	if !ok {
		return ""
	}

	ext := proto.GetExtension(enumValueOptions, http.E_EnumValue)
	if ext == nil {
		return ""
	}

	customValue, ok := ext.(string)
	if !ok {
		return ""
	}

	return customValue
}

// HasAnyEnumValueMapping returns true if any value in the enum has a custom JSON mapping.
func HasAnyEnumValueMapping(enum *protogen.Enum) bool {
	for _, value := range enum.Values {
		if GetEnumValueMapping(value) != "" {
			return true
		}
	}
	return false
}

// HasConflictingEnumAnnotations checks if a field has both enum_encoding=NUMBER and
// enum_value annotations on its enum values, which is an error per CONTEXT.md.
func HasConflictingEnumAnnotations(field *protogen.Field) bool {
	if field.Enum == nil {
		return false
	}

	encoding := GetEnumEncoding(field)
	if encoding != http.EnumEncoding_ENUM_ENCODING_NUMBER {
		return false
	}

	return HasAnyEnumValueMapping(field.Enum)
}
