package pyclientgen

// Python type-name constants. Centralised here so future renames (e.g. moving
// from `str` to `String`) only need to touch one location, and to satisfy
// goconst's complaint about repeated string literals.
const (
	pyStr  = "str"
	pyInt  = "int"
	pyBool = "bool"
	pyNone = "None"
	pyAny  = "Any"

	pyDictStrAny = "dict[str, Any]"
)

// Well-known type proto names. These are the FullName() strings we match
// against to apply the WKT-specific Python representation.
const (
	wktTimestamp   = "google.protobuf.Timestamp"
	wktDuration    = "google.protobuf.Duration"
	wktAny         = "google.protobuf.Any"
	wktFieldMask   = "google.protobuf.FieldMask"
	wktEmpty       = "google.protobuf.Empty"
	wktStruct      = "google.protobuf.Struct"
	wktValue       = "google.protobuf.Value"
	wktListValue   = "google.protobuf.ListValue"
	wktStringValue = "google.protobuf.StringValue"
	wktBoolValue   = "google.protobuf.BoolValue"
	wktInt32Value  = "google.protobuf.Int32Value"
	wktUInt32Value = "google.protobuf.UInt32Value"
	wktInt64Value  = "google.protobuf.Int64Value"
	wktUInt64Value = "google.protobuf.UInt64Value"
	wktFloatValue  = "google.protobuf.FloatValue"
	wktDoubleValue = "google.protobuf.DoubleValue"
	wktBytesValue  = "google.protobuf.BytesValue"
)
