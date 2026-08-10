package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// JSONMappingContext models all custom JSON mapping work needed for one message.
type JSONMappingContext struct {
	Message                *protogen.Message
	FieldTransforms        []*JSONMappingFieldTransform
	NestedDelegationFields []*protogen.Field
	RootUnwrap             *RootUnwrapMessage
}

// JSONMappingFieldTransform records a direct field-level JSON mapping transform.
type JSONMappingFieldTransform struct {
	Field *protogen.Field
	Kind  JSONMappingTransformKind
}

// JSONMappingTransformKind identifies the custom JSON mapping feature a field needs.
type JSONMappingTransformKind string

const (
	TransformInt64Number        JSONMappingTransformKind = "int64_number"
	TransformEnumValue          JSONMappingTransformKind = "enum_value"
	TransformBytesEncoding      JSONMappingTransformKind = "bytes_encoding"
	TransformTimestampFormat    JSONMappingTransformKind = "timestamp_format"
	TransformNullable           JSONMappingTransformKind = "nullable"
	TransformEmptyBehavior      JSONMappingTransformKind = "empty_behavior"
	TransformFlatten            JSONMappingTransformKind = "flatten"
	TransformOneofDiscriminator JSONMappingTransformKind = "oneof_config"
	TransformMapValueUnwrap     JSONMappingTransformKind = "map_value_unwrap"
)

// collectJSONMappingContexts analyzes a file and returns one context for each message that needs
// direct JSON transforms, nested delegation to a child message, or root-level unwrap handling.
func collectJSONMappingContexts(file *protogen.File) ([]*JSONMappingContext, error) {
	if err := validateJSONMappingCollectorAnnotations(file); err != nil {
		return nil, err
	}

	unwrapFields, err := collectAllUnwrapFields(file.Messages)
	if err != nil {
		return nil, err
	}

	var contexts []*JSONMappingContext
	collectJSONMappingContextMessages(file.Messages, unwrapFields, &contexts)
	return contexts, nil
}

// messageNeedsJSONMapping reports whether a message needs a central JSON mapping context either
// directly or because one of its child message fields needs mapping transitively.
func messageNeedsJSONMapping(msg *protogen.Message) bool {
	return messageNeedsJSONMappingRecursive(msg, map[string]bool{})
}

// fieldNeedsNestedJSONDelegation reports whether a message-typed field's value type needs custom
// JSON mapping directly or transitively.
func fieldNeedsNestedJSONDelegation(field *protogen.Field) bool {
	return fieldNeedsNestedJSONDelegationRecursive(field, map[string]bool{})
}

func collectJSONMappingContextMessages(
	messages []*protogen.Message,
	unwrapFields map[string]*annotations.UnwrapFieldInfo,
	contexts *[]*JSONMappingContext,
) {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}

		ctx := buildJSONMappingContext(msg, unwrapFields)
		if len(ctx.FieldTransforms) > 0 || len(ctx.NestedDelegationFields) > 0 || ctx.RootUnwrap != nil {
			*contexts = append(*contexts, ctx)
		}

		collectJSONMappingContextMessages(msg.Messages, unwrapFields, contexts)
	}
}

func buildJSONMappingContext(
	msg *protogen.Message,
	unwrapFields map[string]*annotations.UnwrapFieldInfo,
) *JSONMappingContext {
	ctx := &JSONMappingContext{
		Message:         msg,
		FieldTransforms: directJSONMappingFieldTransforms(msg, unwrapFields),
		RootUnwrap:      rootUnwrapMessageFor(msg, unwrapFields),
	}

	for _, field := range msg.Fields {
		if fieldNeedsNestedJSONDelegation(field) {
			ctx.NestedDelegationFields = append(ctx.NestedDelegationFields, field)
		}
	}

	return ctx
}

func directJSONMappingFieldTransforms(
	msg *protogen.Message,
	unwrapFields map[string]*annotations.UnwrapFieldInfo,
) []*JSONMappingFieldTransform {
	var transforms []*JSONMappingFieldTransform

	for _, field := range msg.Fields {
		if isInt64Type(field) && annotations.IsInt64NumberEncoding(field) {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformInt64Number})
		}
		if customEnumForField(field) != nil {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformEnumValue})
		}
		if field.Desc.Kind() == protoreflect.BytesKind && annotations.HasBytesEncodingAnnotation(field) {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformBytesEncoding})
		}
		if annotations.IsTimestampField(field) && annotations.HasTimestampFormatAnnotation(field) {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformTimestampFormat})
		}
		if annotations.IsNullableField(field) {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformNullable})
		}
		if annotations.HasEmptyBehaviorAnnotation(field) {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformEmptyBehavior})
		}
		if annotations.IsFlattenField(field) {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformFlatten})
		}
		if fieldHasMapValueUnwrap(field, unwrapFields) {
			transforms = append(transforms, &JSONMappingFieldTransform{Field: field, Kind: TransformMapValueUnwrap})
		}
	}

	for _, oneof := range msg.Oneofs {
		if annotations.GetOneofConfig(oneof) == nil {
			continue
		}
		transforms = append(transforms, &JSONMappingFieldTransform{
			Field: firstOneofField(oneof),
			Kind:  TransformOneofDiscriminator,
		})
	}

	return transforms
}

func firstOneofField(oneof *protogen.Oneof) *protogen.Field {
	if len(oneof.Fields) == 0 {
		return nil
	}
	return oneof.Fields[0]
}

func fieldHasMapValueUnwrap(
	field *protogen.Field,
	unwrapFields map[string]*annotations.UnwrapFieldInfo,
) bool {
	valueMsg := getMapValueMessage(field)
	if valueMsg == nil {
		return false
	}

	if unwrapInfo := unwrapInfoForMessage(valueMsg, unwrapFields); unwrapInfo != nil {
		return true
	}
	return false
}

func rootUnwrapMessageFor(
	msg *protogen.Message,
	unwrapFields map[string]*annotations.UnwrapFieldInfo,
) *RootUnwrapMessage {
	info := unwrapInfoForMessage(msg, unwrapFields)
	if info == nil || !info.IsRootUnwrap {
		return nil
	}

	rootUnwrap := &RootUnwrapMessage{
		Message:     msg,
		UnwrapField: info.Field,
		IsMap:       info.IsMapField,
	}

	if info.IsMapField {
		valueMsg := getMapValueMessage(info.Field)
		if valueMsg != nil {
			rootUnwrap.ValueMessage = valueMsg
			rootUnwrap.ValueUnwrap = unwrapInfoForMessage(valueMsg, unwrapFields)
		}
	}

	return rootUnwrap
}

func unwrapInfoForMessage(
	msg *protogen.Message,
	unwrapFields map[string]*annotations.UnwrapFieldInfo,
) *annotations.UnwrapFieldInfo {
	if msg == nil {
		return nil
	}
	if unwrapInfo := unwrapFields[string(msg.Desc.FullName())]; unwrapInfo != nil {
		return unwrapInfo
	}
	unwrapInfo, err := annotations.GetUnwrapField(msg)
	if err != nil {
		return nil
	}
	return unwrapInfo
}

func messageNeedsJSONMappingRecursive(msg *protogen.Message, visited map[string]bool) bool {
	if msg == nil || msg.Desc.IsMapEntry() {
		return false
	}
	if messageDirectlyNeedsJSONMapping(msg) {
		return true
	}

	key := string(msg.Desc.FullName())
	if visited[key] {
		return false
	}
	visited[key] = true

	for _, field := range msg.Fields {
		if fieldNeedsNestedJSONDelegationRecursive(field, visited) {
			return true
		}
	}
	return false
}

func messageDirectlyNeedsJSONMapping(msg *protogen.Message) bool {
	if hasInt64NumberFields(msg) || hasCustomEnumFields(msg) || hasBytesEncodingFields(msg) ||
		hasTimestampFormatFields(msg) || hasNullableFields(msg) || hasEmptyBehaviorFields(msg) ||
		hasFlattenFields(msg) || hasOneofDiscriminator(msg) {
		return true
	}

	if unwrapInfo, err := annotations.GetUnwrapField(msg); err == nil && unwrapInfo != nil && unwrapInfo.IsRootUnwrap {
		return true
	}

	for _, field := range msg.Fields {
		if fieldHasMapValueUnwrap(field, nil) {
			return true
		}
	}
	return false
}

func fieldNeedsNestedJSONDelegationRecursive(field *protogen.Field, visited map[string]bool) bool {
	child := nestedJSONMappingChild(field)
	if child == nil {
		return false
	}
	return messageNeedsJSONMappingRecursive(child, visited)
}

func nestedJSONMappingChild(field *protogen.Field) *protogen.Message {
	if child := mapMessageValueChild(field); child != nil {
		return child
	}
	return nestedMessageChild(field)
}

func validateJSONMappingCollectorAnnotations(file *protogen.File) error {
	if err := validateNullableAnnotations(file); err != nil {
		return err
	}
	if err := validateTimestampFormatAnnotations(file); err != nil {
		return err
	}
	if err := validateBytesEncodingAnnotations(file); err != nil {
		return err
	}
	if err := validateEmptyBehaviorAnnotations(file); err != nil {
		return err
	}
	if err := validateFlattenAnnotationsForJSONMappingCollector(file.Messages); err != nil {
		return err
	}
	if err := validateOneofDiscriminatorAnnotations(file); err != nil {
		return err
	}
	if err := validateEnumAnnotationsForJSONMappingCollector(file.Messages); err != nil {
		return err
	}
	if err := validateEnumFieldEncoding(file); err != nil {
		return err
	}
	_, err := collectAllUnwrapFields(file.Messages)
	return err
}

func validateFlattenAnnotationsForJSONMappingCollector(messages []*protogen.Message) error {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}
		for _, field := range msg.Fields {
			if err := annotations.ValidateFlattenField(field, msg.GoIdent.GoName); err != nil {
				return err
			}
		}
		if hasFlattenFields(msg) {
			if err := annotations.ValidateFlattenCollisions(msg); err != nil {
				return err
			}
		}
		if err := validateFlattenAnnotationsForJSONMappingCollector(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

func validateEnumAnnotationsForJSONMappingCollector(messages []*protogen.Message) error {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}
		for _, field := range msg.Fields {
			if field.Desc.Kind() == protoreflect.EnumKind {
				if err := validateEnumAnnotations(field); err != nil {
					return err
				}
			}
		}
		if err := validateEnumAnnotationsForJSONMappingCollector(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}
