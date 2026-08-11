package httpgen

import (
	"fmt"
	"strconv"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// enumFieldShape describes how a custom-enum field is laid out in JSON.
type enumFieldShape int

const (
	enumShapeSingular enumFieldShape = iota // "field": "value"
	enumShapeRepeated                       // "field": ["value", ...]
	enumShapeMap                            // "field": {"key": "value", ...}
)

// EnumFieldInfo pairs a field with the custom-value enum it references and its JSON shape.
type EnumFieldInfo struct {
	Field *protogen.Field
	// Enum is the enum type carrying enum_value mappings. For map fields this is the
	// map value's enum type.
	Enum  *protogen.Enum
	Shape enumFieldShape
}

// customEnumForField returns the custom-value enum a field references (the map value's enum for
// map fields), regardless of Go package, or nil if the field does not reference a custom-value,
// string-encoded enum. NUMBER-encoded enums never carry custom string values.
func customEnumForField(field *protogen.Field) *protogen.Enum {
	if annotations.GetEnumEncoding(field) == http.EnumEncoding_ENUM_ENCODING_NUMBER {
		return nil
	}

	switch {
	case field.Desc.IsMap():
		if field.Desc.MapValue().Kind() != protoreflect.EnumKind {
			return nil
		}
		valueEnum := field.Message.Fields[1].Enum
		if valueEnum != nil && annotations.HasAnyEnumValueMapping(valueEnum) {
			return valueEnum
		}
	case field.Desc.Kind() == protoreflect.EnumKind:
		if field.Enum != nil && annotations.HasAnyEnumValueMapping(field.Enum) {
			return field.Enum
		}
	}
	return nil
}

// nestedMessageChild returns the message type of a singular or repeated (non-map) message field,
// or nil if the field is not such a field.
func nestedMessageChild(field *protogen.Field) *protogen.Message {
	if field.Desc.IsMap() || field.Desc.Kind() != protoreflect.MessageKind {
		return nil
	}
	return field.Message
}

// mapMessageValueChild returns the value message type of a map<_, message> field, or nil.
func mapMessageValueChild(field *protogen.Field) *protogen.Message {
	if !field.Desc.IsMap() {
		return nil
	}
	valueField := field.Message.Fields[1]
	if valueField.Desc.Kind() != protoreflect.MessageKind {
		return nil
	}
	return valueField.Message
}

// messageTransitivelyHasCustomEnum reports whether msg, or any message it nests (singular,
// repeated, or map value) at any depth, has a direct custom enum_value field. The visited set
// guards against recursive message definitions.
func messageTransitivelyHasCustomEnum(msg *protogen.Message, visited map[string]bool) bool {
	if msg == nil {
		return false
	}
	key := string(msg.Desc.FullName())
	if visited[key] {
		return false
	}
	visited[key] = true

	for _, field := range msg.Fields {
		if customEnumForField(field) != nil {
			return true
		}
		if child := nestedMessageChild(field); child != nil &&
			messageTransitivelyHasCustomEnum(child, visited) {
			return true
		}
		if child := mapMessageValueChild(field); child != nil &&
			messageTransitivelyHasCustomEnum(child, visited) {
			return true
		}
	}
	return false
}

// customEnumFieldInfo returns EnumFieldInfo for a direct custom-enum field whose enum is in the
// same Go package as the message (pkg), or nil otherwise. Cross-package direct enums are rejected
// by validateEnumFieldEncoding, since the marshaler references that package's private lookup maps.
func customEnumFieldInfo(field *protogen.Field, pkg protogen.GoImportPath) *EnumFieldInfo {
	enum := customEnumForField(field)
	if enum == nil || enum.GoIdent.GoImportPath != pkg {
		return nil
	}

	shape := enumShapeSingular
	switch {
	case field.Desc.IsMap():
		shape = enumShapeMap
	case field.Desc.IsList():
		shape = enumShapeRepeated
	}
	return &EnumFieldInfo{Field: field, Enum: enum, Shape: shape}
}

// getCustomEnumFields returns the direct custom-enum fields of a message.
func getCustomEnumFields(msg *protogen.Message) []*EnumFieldInfo {
	pkg := msg.GoIdent.GoImportPath
	var fields []*EnumFieldInfo
	for _, field := range msg.Fields {
		if info := customEnumFieldInfo(field, pkg); info != nil {
			fields = append(fields, info)
		}
	}
	return fields
}

// hasCustomEnumFields reports whether a message directly carries any custom enum_value field.
func hasCustomEnumFields(msg *protogen.Message) bool {
	return len(getCustomEnumFields(msg)) > 0
}

// validateEnumFieldEncoding fails loudly for cases the Go generator cannot encode:
//   - a direct custom-enum field whose enum lives in a different Go package (the marshaler needs
//     that package's private lookup maps);
//   - a map<_, message> value whose message transitively carries a custom enum (nested re-
//     serialization through map values is not yet supported).
//
// Failing loudly avoids silently emitting raw proto enum names, which would contradict the docs.
func validateEnumFieldEncoding(file *protogen.File) error {
	return validateEnumFieldEncodingMessages(file.Messages)
}

func validateEnumFieldEncodingMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}
		if err := validateEnumFieldEncodingFields(msg); err != nil {
			return err
		}
		if err := validateEnumFieldEncodingMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

func validateEnumFieldEncodingFields(msg *protogen.Message) error {
	pkg := msg.GoIdent.GoImportPath
	for _, field := range msg.Fields {
		if enum := customEnumForField(field); enum != nil &&
			!field.Desc.IsMap() && enum.GoIdent.GoImportPath != pkg {
			return fmt.Errorf(
				"message %s field %q references enum %s with (sebuf.http.enum_value) mappings from a "+
					"different Go package (%s); cross-package custom enum JSON encoding is not supported "+
					"by the Go generator",
				msg.GoIdent.GoName, field.Desc.Name(), enum.GoIdent.GoName, enum.GoIdent.GoImportPath,
			)
		}
		if child := mapMessageValueChild(field); child != nil &&
			messageTransitivelyHasCustomEnum(child, map[string]bool{}) {
			return fmt.Errorf(
				"message %s field %q is a map whose value message %s carries (sebuf.http.enum_value) "+
					"mappings; custom enum JSON encoding inside map values is not yet supported",
				msg.GoIdent.GoName, field.Desc.Name(), child.GoIdent.GoName,
			)
		}
	}
	return nil
}

// enumFieldJSONKeys returns the Go slice-literal contents of the JSON keys a field may appear
// under: the camelCase JSON name and, when different, the proto (snake_case) name. protojson emits
// the proto name when MarshalOptions.UseProtoNames is set, so both must be patched.
func enumFieldJSONKeys(field *protogen.Field) string {
	jsonName := field.Desc.JSONName()
	protoName := string(field.Desc.Name())
	if jsonName == protoName {
		return strconv.Quote(jsonName)
	}
	return strconv.Quote(jsonName) + ", " + strconv.Quote(protoName)
}

// generateEnumFieldMarshal emits the field-level marshal transform for enum_value.
func (g *Generator) generateEnumFieldMarshal(gf *protogen.GeneratedFile, info *EnumFieldInfo) {
	lower := annotations.LowerFirst(info.Enum.GoIdent.GoName)
	toJSON := lower + "ToJSON"
	fromJSON := lower + "FromJSON"

	gf.P("// Rewrite ", info.Field.Desc.Name(), " to custom enum_value strings")
	gf.P("for _, k := range []string{", enumFieldJSONKeys(info.Field), "} {")
	gf.P("v, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")

	switch info.Shape {
	case enumShapeSingular:
		gf.P("var s string")
		gf.P("if err := json.Unmarshal(v, &s); err != nil {")
		gf.P("continue")
		gf.P("}")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("raw[k], _ = json.Marshal(", toJSON, "[e])")
		gf.P("}")
	case enumShapeRepeated:
		gf.P("var arr []string")
		gf.P("if err := json.Unmarshal(v, &arr); err != nil {")
		gf.P("continue")
		gf.P("}")
		gf.P("for i, s := range arr {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("arr[i] = ", toJSON, "[e]")
		gf.P("}")
		gf.P("}")
		gf.P("raw[k], _ = json.Marshal(arr)")
	case enumShapeMap:
		gf.P("var m map[string]string")
		gf.P("if err := json.Unmarshal(v, &m); err != nil {")
		gf.P("continue")
		gf.P("}")
		gf.P("for mk, s := range m {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("m[mk] = ", toJSON, "[e]")
		gf.P("}")
		gf.P("}")
		gf.P("raw[k], _ = json.Marshal(m)")
	}

	gf.P("}")
	gf.P()
}

// generateEnumFieldUnmarshal emits the field-level unmarshal transform for enum_value.
func (g *Generator) generateEnumFieldUnmarshal(gf *protogen.GeneratedFile, info *EnumFieldInfo) {
	lower := annotations.LowerFirst(info.Enum.GoIdent.GoName)
	fromJSON := lower + "FromJSON"

	gf.P("// Rewrite ", info.Field.Desc.Name(), " from custom enum_value strings to proto names")
	gf.P("for _, k := range []string{", enumFieldJSONKeys(info.Field), "} {")
	gf.P("v, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")

	switch info.Shape {
	case enumShapeSingular:
		gf.P("var s string")
		gf.P("if err := json.Unmarshal(v, &s); err != nil {")
		gf.P("continue")
		gf.P("}")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("raw[k], _ = json.Marshal(e.String())")
		gf.P("}")
	case enumShapeRepeated:
		gf.P("var arr []string")
		gf.P("if err := json.Unmarshal(v, &arr); err != nil {")
		gf.P("continue")
		gf.P("}")
		gf.P("for i, s := range arr {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("arr[i] = e.String()")
		gf.P("}")
		gf.P("}")
		gf.P("raw[k], _ = json.Marshal(arr)")
	case enumShapeMap:
		gf.P("var m map[string]string")
		gf.P("if err := json.Unmarshal(v, &m); err != nil {")
		gf.P("continue")
		gf.P("}")
		gf.P("for mk, s := range m {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("m[mk] = e.String()")
		gf.P("}")
		gf.P("}")
		gf.P("raw[k], _ = json.Marshal(m)")
	}

	gf.P("}")
	gf.P()
}
