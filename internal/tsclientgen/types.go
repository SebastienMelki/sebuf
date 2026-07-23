package tsclientgen

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

// tsZeroCheck returns the TypeScript zero-value check expression for a query param.
func tsZeroCheck(fieldKind string) string {
	return tscommon.TSZeroCheck(fieldKind)
}

// tsZeroCheckForField returns the TypeScript zero-value check expression for a field.
func tsZeroCheckForField(field *protogen.Field) string {
	return tscommon.TSZeroCheckForField(field)
}

// esZeroCheckForField returns the query-param zero-value check for protobuf-es
// mode. protobuf-es types 64-bit integer fields as bigint (regardless of the
// sebuf int64_encoding annotation, which only affects JSON wire format), so the
// string "0" check that hand-rolled mode uses would compare bigint to string.
// Only the 64-bit case differs; every other kind delegates unchanged.
func esZeroCheckForField(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return " !== 0n"
	default:
		return tsZeroCheckForField(field)
	}
}

// printer is a function that prints a formatted line.
type printer func(format string, args ...interface{})
