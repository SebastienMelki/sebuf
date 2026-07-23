package tscommon

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// CheckESMessageAnnotations walks the transitive message closure reachable from
// root and returns a generation-time error if any field, message, or oneof
// carries a sebuf.http JSON-mapping annotation that protobuf-es runtime mode
// cannot honor.
//
// protobuf-es mode serializes with the runtime's canonical protojson codec
// (toJson/fromJson). sebuf's Go server, by contrast, layers an annotation-aware
// transform (MarshalJSONSebuf) on top of protojson that deliberately diverges
// from canonical protojson wherever a JSON-mapping annotation is set. That
// transform layer has not been ported to TypeScript, so any annotated proto put
// on the wire in es mode would silently disagree with the Go server (throw on
// decode or drop data). Rather than emit such code, the generator fails loud
// here for the whole 🔴/🟡 annotation set.
//
// role is a short human label for where root sits on the RPC ("request",
// "response", or "SSE event") and is only used in the error message. The walk is
// cycle-safe via a visited set keyed on message full name.
func CheckESMessageAnnotations(service, method, role string, root *protogen.Message) error {
	visited := make(map[protoreflect.FullName]bool)
	return walkESMessageClosure(service, method, role, root, visited)
}

func walkESMessageClosure(
	service, method, role string,
	msg *protogen.Message,
	visited map[protoreflect.FullName]bool,
) error {
	if msg == nil {
		return nil
	}
	name := msg.Desc.FullName()
	if visited[name] {
		return nil
	}
	visited[name] = true

	// Message-level annotations.
	if annotations.FindUnwrapField(msg) != nil {
		return unsupportedESAnnotationError(service, method, role, "unwrap", "message "+string(msg.Desc.Name()))
	}
	if annotations.HasOneofDiscriminator(msg) {
		return unsupportedESAnnotationError(
			service, method, role, "oneof_config", "message "+string(msg.Desc.Name()),
		)
	}

	for _, field := range msg.Fields {
		if err := checkESField(service, method, role, msg, field); err != nil {
			return err
		}
		// Recurse into message-typed fields. For map fields, field.Message is the
		// synthetic map-entry message, and recursing into it reaches the value
		// type (and its annotations) via the entry's value field.
		if field.Message != nil {
			if err := walkESMessageClosure(service, method, role, field.Message, visited); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkESField reports the first unsupported JSON-mapping annotation on a single
// field, or nil if the field is representable under canonical protojson.
func checkESField(
	service, method, role string,
	msg *protogen.Message,
	field *protogen.Field,
) error {
	location := fmt.Sprintf("field %s.%s", msg.Desc.Name(), field.Desc.Name())

	var annotation string
	switch {
	case annotations.HasUnwrapAnnotation(field):
		annotation = "unwrap"
	case annotations.IsFlattenField(field):
		annotation = "flatten"
	case annotations.HasTimestampFormatAnnotation(field):
		annotation = "timestamp_format"
	case annotations.HasBytesEncodingAnnotation(field):
		annotation = "bytes_encoding"
	case annotations.IsNullableField(field):
		annotation = "nullable"
	case annotations.HasEmptyBehaviorAnnotation(field):
		annotation = "empty_behavior"
	case annotations.IsInt64NumberEncoding(field):
		annotation = "int64_encoding=NUMBER"
	case field.Enum != nil && annotations.GetEnumEncoding(field) == http.EnumEncoding_ENUM_ENCODING_NUMBER:
		annotation = "enum_encoding=NUMBER"
	case field.Enum != nil && annotations.HasAnyEnumValueMapping(field.Enum):
		annotation = "enum_value"
	default:
		return nil
	}

	return unsupportedESAnnotationError(service, method, role, annotation, location)
}

// unsupportedESAnnotationError builds the generation-time error returned when a
// JSON-mapping annotation es-mode cannot honor is found in an RPC's message
// closure. location names where the annotation sits (e.g. "field User.status"
// or "message UsersResponse").
func unsupportedESAnnotationError(service, method, role, annotation, location string) error {
	return fmt.Errorf(
		"ts_runtime=protobuf-es: %s uses the %s JSON-mapping annotation (reachable from the %s of %s.%s), "+
			"which es-mode cannot honor. es-mode speaks canonical protojson and does not apply sebuf's "+
			"JSON-mapping transforms, so this proto would not be wire-compatible with a sebuf server. "+
			"Use ts_runtime=hand-rolled for this service, or remove the annotation",
		location, annotation, role, service, method,
	)
}
