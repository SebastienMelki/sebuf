package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// isWrapperFullName returns true if the full name is a Google well-known wrapper type.
func isWrapperFullName(name protoreflect.FullName) bool {
	switch name {
	case "google.protobuf.DoubleValue",
		"google.protobuf.FloatValue",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.Int32Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.BoolValue",
		"google.protobuf.StringValue",
		"google.protobuf.BytesValue":
		return true
	default:
		return false
	}
}

// IsWrapperField returns true if the field is a Google well-known wrapper type
// (e.g., google.protobuf.DoubleValue, google.protobuf.StringValue).
func IsWrapperField(field *protogen.Field) bool {
	return field.Desc.Kind() == protoreflect.MessageKind &&
		field.Message != nil &&
		isWrapperFullName(field.Message.Desc.FullName())
}

// IsWrapperMessage returns true if the message is a Google well-known wrapper type.
func IsWrapperMessage(message *protogen.Message) bool {
	return message != nil && isWrapperFullName(message.Desc.FullName())
}
