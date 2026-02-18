package tsclientgen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

// collectServiceMessages collects all messages transitively referenced by services in a file.
func collectServiceMessages(file *protogen.File) *tscommon.MessageSet {
	return tscommon.CollectServiceMessages(file)
}

// generateEnumType writes a TypeScript string union type for a protobuf enum.
func generateEnumType(p printer, enum *protogen.Enum) {
	tscommon.GenerateEnumType(tscommon.Printer(p), enum)
}

// generateInterface writes a TypeScript interface for a protobuf message.
func generateInterface(p printer, msg *protogen.Message) {
	tscommon.GenerateInterface(tscommon.Printer(p), msg)
}

// rootUnwrapTSType returns the TypeScript type for a root-unwrapped message.
func rootUnwrapTSType(msg *protogen.Message) string {
	return tscommon.RootUnwrapTSType(msg)
}

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
