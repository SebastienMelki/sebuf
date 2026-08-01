package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// IsStructMessage returns true if the message is google.protobuf.Struct.
func IsStructMessage(message *protogen.Message) bool {
	return message != nil && message.Desc.FullName() == "google.protobuf.Struct"
}

// IsValueMessage returns true if the message is google.protobuf.Value.
func IsValueMessage(message *protogen.Message) bool {
	return message != nil && message.Desc.FullName() == "google.protobuf.Value"
}

// IsListValueMessage returns true if the message is google.protobuf.ListValue.
func IsListValueMessage(message *protogen.Message) bool {
	return message != nil && message.Desc.FullName() == "google.protobuf.ListValue"
}

// IsStructWellKnownMessage returns true if the message is Struct, Value, or ListValue.
func IsStructWellKnownMessage(message *protogen.Message) bool {
	return IsStructMessage(message) || IsValueMessage(message) || IsListValueMessage(message)
}

// IsStructWellKnownField returns true if the field is a Struct, Value, or ListValue message.
func IsStructWellKnownField(field *protogen.Field) bool {
	return field.Desc.Kind() == protoreflect.MessageKind &&
		field.Message != nil &&
		IsStructWellKnownMessage(field.Message)
}
