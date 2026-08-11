package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// hasInt64NumberFields returns true if any int64/uint64 field in the message has NUMBER encoding.
// This checks direct fields only (not nested messages).
func hasInt64NumberFields(message *protogen.Message) bool {
	for _, field := range message.Fields {
		if isInt64Type(field) && annotations.IsInt64NumberEncoding(field) {
			return true
		}
	}
	return false
}

// isInt64Type returns true if the field is an int64 or uint64 type (including variants).
func isInt64Type(field *protogen.Field) bool {
	kind := field.Desc.Kind().String()
	switch kind {
	case kindInt64, kindSint64, kindSfixed64, kindUint64, kindFixed64:
		return true
	default:
		return false
	}
}

// generateInt64FieldMarshal emits the field-level marshal transform for int64_encoding=NUMBER.
func (g *Generator) generateInt64FieldMarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	if field.Desc.IsList() {
		g.generateRepeatedInt64FieldMarshal(gf, field)
	} else {
		g.generateSingularInt64FieldMarshal(gf, field)
	}
}

func (g *Generator) generateSingularInt64FieldMarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	fieldName := field.GoName

	gf.P("// Convert ", fieldName, " from string to number")
	gf.P("if x.", fieldName, " != 0 {")
	gf.P("data, _ = json.Marshal(x.", fieldName, ")")
	emitRawFieldSetForMarshalOptions(gf, field, "data")
	gf.P("} else {")
	gf.P("// Remove the field if zero (proto3 default behavior)")
	emitRawFieldDeleteAllJSONKeys(gf, field)
	gf.P("}")
	gf.P()
}

func (g *Generator) generateRepeatedInt64FieldMarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	fieldName := field.GoName

	gf.P("// Convert repeated ", fieldName, " from strings to numbers")
	gf.P("if len(x.", fieldName, ") > 0 {")
	gf.P("data, _ = json.Marshal(x.", fieldName, ")")
	emitRawFieldSetForMarshalOptions(gf, field, "data")
	gf.P("}")
	gf.P()
}

// generateInt64FieldUnmarshal emits the field-level unmarshal transform for int64_encoding=NUMBER.
func (g *Generator) generateInt64FieldUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	if field.Desc.IsList() {
		g.generateRepeatedInt64FieldUnmarshal(gf, field)
	} else {
		g.generateSingularInt64FieldUnmarshal(gf, field)
	}
}

func (g *Generator) generateSingularInt64FieldUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	isUnsigned := isUint64Type(field)

	gf.P("// Convert ", field.Desc.Name(), " from number to string for protojson")
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("if rawVal, ok := raw[k]; ok {")
	if isUnsigned {
		gf.P("var num uint64")
		gf.P("if err := json.Unmarshal(rawVal, &num); err == nil {")
		gf.P("raw[k], _ = json.Marshal(strconv.FormatUint(num, 10))")
	} else {
		gf.P("var num int64")
		gf.P("if err := json.Unmarshal(rawVal, &num); err == nil {")
		gf.P("raw[k], _ = json.Marshal(strconv.FormatInt(num, 10))")
	}
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateRepeatedInt64FieldUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	isUnsigned := isUint64Type(field)

	gf.P("// Convert repeated ", field.Desc.Name(), " from numbers to strings for protojson")
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("if rawVal, ok := raw[k]; ok {")
	if isUnsigned {
		gf.P("var nums []uint64")
		gf.P("if err := json.Unmarshal(rawVal, &nums); err == nil {")
		gf.P("strs := make([]string, len(nums))")
		gf.P("for i, n := range nums {")
		gf.P("strs[i] = strconv.FormatUint(n, 10)")
		gf.P("}")
		gf.P("raw[k], _ = json.Marshal(strs)")
	} else {
		gf.P("var nums []int64")
		gf.P("if err := json.Unmarshal(rawVal, &nums); err == nil {")
		gf.P("strs := make([]string, len(nums))")
		gf.P("for i, n := range nums {")
		gf.P("strs[i] = strconv.FormatInt(n, 10)")
		gf.P("}")
		gf.P("raw[k], _ = json.Marshal(strs)")
	}
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()
}

// isUint64Type returns true if the field is an unsigned 64-bit type.
func isUint64Type(field *protogen.Field) bool {
	kind := field.Desc.Kind().String()
	return kind == kindUint64 || kind == kindFixed64
}
