package pyclientgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// encodeScalarExpr returns a Python expression that converts a scalar source
// value into the form expected by JSON. `src` is the source Python expression
// (e.g. "self.name", "v", "_x").
//
// For most scalars this is the identity ("v"), but several proto types need
// transformation: bytes → base64/hex string, int64 → str/int depending on
// int64_encoding, enum → JSON name or custom enum_value, Timestamp → RFC3339
// string or unix int per timestamp_format.
func encodeScalarExpr(field *protogen.Field, src string) string {
	if isWellKnown(field) {
		return encodeWKTExpr(field, src)
	}

	switch field.Desc.Kind() {
	case protoreflect.BytesKind:
		return encodeBytesExpr(field, src)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if annotations.IsInt64NumberEncoding(field) {
			return src // emit as JSON number
		}
		return fmt.Sprintf("str(%s)", src) // protojson default: emit as string
	case protoreflect.EnumKind:
		return encodeEnumExpr(field, src)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return fmt.Sprintf("%s.to_dict()", src)
	}
	return src
}

// decodeScalarExpr returns a Python expression that decodes a JSON-side value
// into the Python representation, inverse of encodeScalarExpr.
func decodeScalarExpr(field *protogen.Field, src string) string {
	if isWellKnown(field) {
		return decodeWKTExpr(field, src)
	}

	switch field.Desc.Kind() {
	case protoreflect.BytesKind:
		return decodeBytesExpr(field, src)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if annotations.IsInt64NumberEncoding(field) {
			return fmt.Sprintf("int(%s)", src)
		}
		// Wire is string; we keep it as string locally to preserve precision.
		return fmt.Sprintf("str(%s)", src)
	case protoreflect.EnumKind:
		return decodeEnumExpr(field, src)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return fmt.Sprintf("%s.from_dict(%s)", pythonTypeName(field.Message), src)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return fmt.Sprintf("float(%s)", src)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return fmt.Sprintf("int(%s)", src)
	case protoreflect.BoolKind:
		return fmt.Sprintf("bool(%s)", src)
	case protoreflect.StringKind:
		return fmt.Sprintf("str(%s)", src)
	}
	return src
}

// encodeBytesExpr returns a Python expression that encodes a bytes value to JSON
// per the field's bytes_encoding annotation.
func encodeBytesExpr(field *protogen.Field, src string) string {
	switch annotations.GetBytesEncoding(field) {
	case sebufhttp.BytesEncoding_BYTES_ENCODING_HEX:
		return fmt.Sprintf("%s.hex()", src)
	case sebufhttp.BytesEncoding_BYTES_ENCODING_BASE64URL:
		return fmt.Sprintf(`base64.urlsafe_b64encode(%s).decode("ascii")`, src)
	case sebufhttp.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW:
		return fmt.Sprintf(`base64.urlsafe_b64encode(%s).decode("ascii").rstrip("=")`, src)
	case sebufhttp.BytesEncoding_BYTES_ENCODING_BASE64_RAW:
		return fmt.Sprintf(`base64.b64encode(%s).decode("ascii").rstrip("=")`, src)
	default:
		// BASE64 / UNSPECIFIED: standard base64 with padding (protojson default)
		return fmt.Sprintf(`base64.b64encode(%s).decode("ascii")`, src)
	}
}

// decodeBytesExpr inverts encodeBytesExpr.
func decodeBytesExpr(field *protogen.Field, src string) string {
	switch annotations.GetBytesEncoding(field) {
	case sebufhttp.BytesEncoding_BYTES_ENCODING_HEX:
		return fmt.Sprintf("bytes.fromhex(%s)", src)
	case sebufhttp.BytesEncoding_BYTES_ENCODING_BASE64URL,
		sebufhttp.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW:
		// urlsafe_b64decode tolerates missing padding when input length is a multiple
		// of 4 already; pad to a multiple of 4 so RAW variants decode cleanly.
		return fmt.Sprintf(`base64.urlsafe_b64decode(%s + "=" * (-len(%s) %% 4))`, src, src)
	case sebufhttp.BytesEncoding_BYTES_ENCODING_BASE64_RAW:
		return fmt.Sprintf(`base64.b64decode(%s + "=" * (-len(%s) %% 4))`, src, src)
	default:
		return fmt.Sprintf("base64.b64decode(%s)", src)
	}
}

// encodeEnumExpr returns the JSON-side encoding for an enum field. By default
// enums serialize as the proto enum name (STRING) or the integer (NUMBER); if
// any variant declares (sebuf.http.enum_value), the JSON_VALUES table overrides.
func encodeEnumExpr(field *protogen.Field, src string) string {
	switch annotations.GetEnumEncoding(field) {
	case sebufhttp.EnumEncoding_ENUM_ENCODING_NUMBER:
		return fmt.Sprintf("int(%s)", src)
	default:
		// STRING / UNSPECIFIED: use JSON_VALUES override or the variant name.
		enumName := pythonEnumName(field.Enum)
		return fmt.Sprintf("%s_JSON_VALUES.get(%s, %s.name)", enumName, src, src)
	}
}

// decodeEnumExpr returns the Python-side decoding for an enum field.
func decodeEnumExpr(field *protogen.Field, src string) string {
	enumName := pythonEnumName(field.Enum)
	switch annotations.GetEnumEncoding(field) {
	case sebufhttp.EnumEncoding_ENUM_ENCODING_NUMBER:
		return fmt.Sprintf("%s(int(%s))", enumName, src)
	default:
		// Accept either string (name or json_value) or int from the wire.
		return fmt.Sprintf("_decode_enum_%s(%s)", enumName, src)
	}
}

// encodeWKTExpr handles JSON encoding for well-known types.
func encodeWKTExpr(field *protogen.Field, src string) string {
	switch field.Message.Desc.FullName() {
	case "google.protobuf.Timestamp":
		switch annotations.GetTimestampFormat(field) {
		case sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS:
			return fmt.Sprintf("int(%s.timestamp())", src)
		case sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS:
			return fmt.Sprintf("int(%s.timestamp() * 1000)", src)
		case sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_DATE:
			return fmt.Sprintf(`%s.strftime("%%Y-%%m-%%d")`, src)
		default:
			// RFC3339: ensure trailing Z when UTC, .isoformat() otherwise.
			return fmt.Sprintf(`(%s.astimezone(timezone.utc).strftime("%%Y-%%m-%%dT%%H:%%M:%%SZ") if %s.tzinfo else %s.isoformat() + "Z")`, src, src, src)
		}
	case "google.protobuf.Empty":
		return "{}"
	case "google.protobuf.StringValue", "google.protobuf.BoolValue",
		"google.protobuf.Int32Value", "google.protobuf.UInt32Value",
		"google.protobuf.FloatValue", "google.protobuf.DoubleValue",
		"google.protobuf.Duration", "google.protobuf.Any", "google.protobuf.FieldMask",
		"google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.ListValue":
		// Already in the appropriate JSON-ready Python form (str/int/float/bool/dict/list).
		return src
	case "google.protobuf.Int64Value", "google.protobuf.UInt64Value":
		if annotations.IsInt64NumberEncoding(field) {
			return src
		}
		return fmt.Sprintf("str(%s)", src)
	case "google.protobuf.BytesValue":
		return fmt.Sprintf(`base64.b64encode(%s).decode("ascii")`, src)
	}
	return src
}

// decodeWKTExpr inverts encodeWKTExpr.
func decodeWKTExpr(field *protogen.Field, src string) string {
	switch field.Message.Desc.FullName() {
	case "google.protobuf.Timestamp":
		switch annotations.GetTimestampFormat(field) {
		case sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS:
			return fmt.Sprintf("datetime.fromtimestamp(int(%s), tz=timezone.utc)", src)
		case sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS:
			return fmt.Sprintf("datetime.fromtimestamp(int(%s) / 1000, tz=timezone.utc)", src)
		case sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_DATE:
			return src // keep as date string
		default:
			return fmt.Sprintf(`datetime.fromisoformat(%s.replace("Z", "+00:00"))`, src)
		}
	case "google.protobuf.Empty":
		return "{}"
	case "google.protobuf.BytesValue":
		return fmt.Sprintf("base64.b64decode(%s)", src)
	}
	return src
}
