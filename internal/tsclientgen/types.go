package tsclientgen

import (
	"google.golang.org/protobuf/compiler/protogen"

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

// printer is a function that prints a formatted line.
type printer func(format string, args ...interface{})
