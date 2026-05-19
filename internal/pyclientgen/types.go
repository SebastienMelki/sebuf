package pyclientgen

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// pythonScalarType maps a protobuf scalar kind to the Python builtin used in
// dataclass type annotations. int64/uint64 default to `str` to match protojson's
// JavaScript-safe encoding; the int64_encoding=NUMBER annotation flips the type
// to `int`.
func pythonScalarType(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return pyBool
	case protoreflect.StringKind:
		return pyStr
	case protoreflect.BytesKind:
		return "bytes"
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return "float"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return pyInt
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if annotations.IsInt64NumberEncoding(field) {
			return pyInt
		}
		return pyStr
	case protoreflect.EnumKind:
		if field.Enum != nil {
			return pythonEnumName(field.Enum)
		}
		return pyInt
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return pythonTypeName(field.Message)
	default:
		return pyAny
	}
}

// pythonFieldType returns the Python type annotation for a field, accounting
// for repeated/map/optional modifiers and well-known type rewrites.
func pythonFieldType(field *protogen.Field) string {
	if isWellKnown(field) {
		if t := wellKnownPythonType(field); t != "" {
			return wrapModifiers(field, t)
		}
	}

	base := pythonScalarType(field)
	return wrapModifiers(field, base)
}

// wrapModifiers adds list/dict/Optional wrappers to a base type expression.
func wrapModifiers(field *protogen.Field, base string) string {
	if field.Desc.IsMap() {
		keyType := pythonScalarType(field.Message.Fields[0])
		valField := field.Message.Fields[1]
		valType := pythonFieldType(valField)
		// Strip the optional/list wrappers that pythonFieldType applies for
		// repeated/optional contexts — map values are intrinsically non-repeated.
		valType = stripOptional(valType)
		return fmt.Sprintf("dict[%s, %s]", keyType, valType)
	}
	if field.Desc.IsList() {
		return fmt.Sprintf("list[%s]", base)
	}
	if field.Desc.HasOptionalKeyword() || annotations.IsNullableField(field) {
		return fmt.Sprintf("Optional[%s]", base)
	}
	// Message fields are inherently nullable in proto3 (unset == None).
	if field.Desc.Kind() == protoreflect.MessageKind && !field.Desc.IsList() && !field.Desc.IsMap() {
		return fmt.Sprintf("Optional[%s]", base)
	}
	return base
}

func stripOptional(t string) string {
	if strings.HasPrefix(t, "Optional[") && strings.HasSuffix(t, "]") {
		return t[len("Optional[") : len(t)-1]
	}
	return t
}

// pythonFieldDefault returns the default expression placed after the type
// annotation in the dataclass field declaration. Mutable defaults must use
// dataclasses.field(default_factory=...) per Python's contract.
func pythonFieldDefault(field *protogen.Field) string {
	if field.Desc.IsMap() {
		return "field(default_factory=dict)"
	}
	if field.Desc.IsList() {
		return "field(default_factory=list)"
	}
	if field.Desc.HasOptionalKeyword() || annotations.IsNullableField(field) {
		return pyNone
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return pyFalse
	case protoreflect.StringKind:
		return `""`
	case protoreflect.BytesKind:
		return `b""`
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return "0.0"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "0"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if annotations.IsInt64NumberEncoding(field) {
			return "0"
		}
		return `"0"`
	case protoreflect.EnumKind:
		if field.Enum != nil && len(field.Enum.Values) > 0 {
			zero := field.Enum.Values[0]
			return fmt.Sprintf("%s.%s", pythonEnumName(field.Enum), variantPythonName(field.Enum, zero))
		}
		return "0"
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return pyNone
	default:
		return pyNone
	}
}

// pythonFieldName returns the Python attribute name for a field, escaping
// Python keywords. We snake_case the proto field name (already lowercase in
// proto convention) but preserve the original on the wire via the JSON name.
func pythonFieldName(field *protogen.Field) string {
	return escapePyKeyword(string(field.Desc.Name()))
}

// jsonFieldName returns the JSON wire name for a field. We default to the
// proto-declared json_name (which protoc populates with lowerCamelCase unless
// the user overrides). This matches protojson on the server side.
func jsonFieldName(field *protogen.Field) string {
	return field.Desc.JSONName()
}

// fieldIsMessage reports whether a singular field is a non-WKT message field
// requiring nested to_dict / from_dict handling.
func fieldIsMessage(field *protogen.Field) bool {
	if field.Desc.IsMap() {
		return false
	}
	return field.Desc.Kind() == protoreflect.MessageKind && !isWellKnown(field)
}

// isWellKnown reports whether a message field references a google.protobuf WKT.
func isWellKnown(field *protogen.Field) bool {
	if field.Message == nil {
		return false
	}
	full := string(field.Message.Desc.FullName())
	return strings.HasPrefix(full, "google.protobuf.")
}

// wellKnownPythonType maps WKT proto names to the Python representation we emit.
// Returns "" when no rewrite applies (caller falls back to dataclass).
func wellKnownPythonType(field *protogen.Field) string {
	if field.Message == nil {
		return ""
	}
	switch field.Message.Desc.FullName() {
	case wktTimestamp:
		return wellKnownTimestampType(field)
	case wktDuration:
		return pyStr
	case wktAny, wktEmpty, wktStruct:
		return pyDictStrAny
	case wktFieldMask:
		return pyListStr
	case wktValue:
		return pyAny
	case wktListValue:
		return "list[Any]"
	case wktStringValue:
		return pyStr
	case wktBoolValue:
		return pyBool
	case wktInt32Value, wktUInt32Value:
		return pyInt
	case wktInt64Value, wktUInt64Value:
		if annotations.IsInt64NumberEncoding(field) {
			return pyInt
		}
		return pyStr
	case wktFloatValue, wktDoubleValue:
		return "float"
	case wktBytesValue:
		return "bytes"
	}
	return ""
}

// wellKnownTimestampType returns the Python annotation type for Timestamp
// fields. The user-facing type is always `datetime`; `timestamp_format` only
// affects the wire encoding (the encoder/decoder in encoding.go convert
// to/from int seconds, int millis, date string, or RFC3339 string).
func wellKnownTimestampType(_ *protogen.Field) string {
	return "datetime"
}
