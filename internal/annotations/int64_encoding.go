package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// GetInt64Encoding returns the int64 encoding for a field.
// Returns INT64_ENCODING_UNSPECIFIED if not set (callers should use protojson default: STRING).
// This annotation is valid on int64, sint64, sfixed64, uint64, and fixed64 fields.
func GetInt64Encoding(field *protogen.Field) http.Int64Encoding {
	options := field.Desc.Options()
	if options == nil {
		return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
	}

	ext := proto.GetExtension(fieldOptions, http.E_Int64Encoding)
	if ext == nil {
		return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
	}

	encoding, ok := ext.(http.Int64Encoding)
	if !ok {
		return http.Int64Encoding_INT64_ENCODING_UNSPECIFIED
	}

	return encoding
}

// IsInt64NumberEncoding returns true if the field should encode int64/uint64 as JSON number.
// Returns false for UNSPECIFIED or STRING (both use protojson default string encoding).
func IsInt64NumberEncoding(field *protogen.Field) bool {
	return GetInt64Encoding(field) == http.Int64Encoding_INT64_ENCODING_NUMBER
}
