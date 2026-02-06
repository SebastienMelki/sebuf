package tsclientgen

import (
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

const (
	tsString  = "string"
	tsNumber  = "number"
	tsBoolean = "boolean"
)

// tsScalarType returns the TypeScript type for a protobuf scalar kind.
// This is the base helper that uses only kind information (no field context).
// For int64/uint64 fields, callers should use tsScalarTypeForField to check encoding annotations.
func tsScalarType(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.StringKind:
		return tsString
	case protoreflect.BoolKind:
		return tsBoolean
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.FloatKind, protoreflect.DoubleKind:
		return tsNumber
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		// proto3 JSON default: 64-bit integers as strings (safe for JavaScript)
		return tsString
	case protoreflect.BytesKind:
		// All bytes_encoding variants (BASE64, BASE64_RAW, BASE64URL, BASE64URL_RAW, HEX) serialize as strings in JSON
		return tsString
	case protoreflect.EnumKind:
		return tsString
	case protoreflect.MessageKind, protoreflect.GroupKind:
		// Handled separately via field.Message
		return "unknown"
	default:
		return "unknown"
	}
}

// tsScalarTypeForField returns the TypeScript type for a protobuf field,
// checking encoding annotations for int64/uint64 fields.
func tsScalarTypeForField(field *protogen.Field) string {
	kind := field.Desc.Kind()

	// Check for int64/uint64 encoding annotation
	//exhaustive:ignore - only int64 kinds need special handling, default covers all others
	switch kind {
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if annotations.IsInt64NumberEncoding(field) {
			return tsNumber // NUMBER encoding: JavaScript number (precision risk for > 2^53)
		}
		return tsString // Default (STRING/UNSPECIFIED): safe string encoding
	default:
		// All other types use the base helper
		return tsScalarType(kind)
	}
}

// tsZeroCheck returns the TypeScript zero-value check expression for a query param.
// Uses the proto field kind (not TS type) to determine the appropriate check.
// For int64/uint64 fields, this returns the STRING encoding check; use tsZeroCheckForField
// when the full field context is available to check encoding annotations.
func tsZeroCheck(fieldKind string) string {
	switch fieldKind {
	case "string":
		return ` !== ""`
	case "bool":
		return ""
	case "int32", "sint32", "sfixed32",
		"uint32", "fixed32",
		"float", "double":
		return " !== 0"
	case "int64", "sint64", "sfixed64",
		"uint64", "fixed64":
		// Default: 64-bit integers are encoded as strings in proto3 JSON
		return ` !== "0"`
	default:
		return ` !== ""`
	}
}

// tsZeroCheckForField returns the TypeScript zero-value check expression for a field,
// checking encoding annotations for int64/uint64 fields.
func tsZeroCheckForField(field *protogen.Field) string {
	kind := field.Desc.Kind()

	// Check for int64/uint64 encoding annotation
	//exhaustive:ignore - only int64 kinds need special handling, default covers all others
	switch kind {
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if annotations.IsInt64NumberEncoding(field) {
			return " !== 0" // NUMBER encoding: numeric zero check
		}
		return ` !== "0"` // Default (STRING/UNSPECIFIED): string zero check
	default:
		// All other types use the base helper
		return tsZeroCheck(kind.String())
	}
}

// messageSet tracks collected messages by full name to deduplicate.
type messageSet struct {
	messages map[string]*protogen.Message
	enums    map[string]*protogen.Enum
	order    []string // preserve discovery order
}

func newMessageSet() *messageSet {
	return &messageSet{
		messages: make(map[string]*protogen.Message),
		enums:    make(map[string]*protogen.Enum),
	}
}

// addMessage adds a message and recursively adds all referenced messages.
func (ms *messageSet) addMessage(msg *protogen.Message) {
	fullName := string(msg.Desc.FullName())
	if _, exists := ms.messages[fullName]; exists {
		return
	}

	// Skip google.protobuf.Timestamp — serialized as primitive (string/number), not nested object
	if fullName == "google.protobuf.Timestamp" {
		return
	}

	// Skip map entry messages — they're synthetic and handled inline
	if msg.Desc.IsMapEntry() {
		// Still recurse into value type if it's a message
		for _, field := range msg.Fields {
			if field.Desc.Kind() == protoreflect.MessageKind && field.Message != nil {
				ms.addMessage(field.Message)
			}
			if field.Desc.Kind() == protoreflect.EnumKind && field.Enum != nil {
				ms.addEnum(field.Enum)
			}
		}
		return
	}

	ms.messages[fullName] = msg
	ms.order = append(ms.order, fullName)

	// Recurse into all fields
	for _, field := range msg.Fields {
		if field.Desc.Kind() == protoreflect.MessageKind && field.Message != nil {
			ms.addMessage(field.Message)
		}
		if field.Desc.Kind() == protoreflect.EnumKind && field.Enum != nil {
			ms.addEnum(field.Enum)
		}
	}
}

func (ms *messageSet) addEnum(enum *protogen.Enum) {
	fullName := string(enum.Desc.FullName())
	ms.enums[fullName] = enum
}

// orderedMessages returns messages in discovery order.
func (ms *messageSet) orderedMessages() []*protogen.Message {
	result := make([]*protogen.Message, 0, len(ms.order))
	for _, name := range ms.order {
		if msg, ok := ms.messages[name]; ok {
			result = append(result, msg)
		}
	}
	return result
}

// orderedEnums returns enums sorted by full name.
func (ms *messageSet) orderedEnums() []*protogen.Enum {
	names := make([]string, 0, len(ms.enums))
	for name := range ms.enums {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]*protogen.Enum, 0, len(names))
	for _, name := range names {
		result = append(result, ms.enums[name])
	}
	return result
}

// collectServiceMessages collects all messages transitively referenced by services in a file.
func collectServiceMessages(file *protogen.File) *messageSet {
	ms := newMessageSet()
	for _, service := range file.Services {
		for _, method := range service.Methods {
			ms.addMessage(method.Input)
			ms.addMessage(method.Output)
		}
	}
	return ms
}

// tsFieldType returns the TypeScript type string for a protobuf field.
func tsFieldType(field *protogen.Field) string {
	// Handle map fields
	if field.Desc.IsMap() {
		valueField := field.Message.Fields[1] // map value is always second field of map entry
		valueType := tsFieldType(valueField)

		// Check if the map value is a message with unwrap annotation
		if valueField.Desc.Kind() == protoreflect.MessageKind && valueField.Message != nil {
			unwrapField := annotations.FindUnwrapField(valueField.Message)
			if unwrapField != nil && !unwrapField.Desc.IsMap() {
				// Map-value unwrap: collapse wrapper to inner type array
				// Use tsElementType since unwrapField is always repeated
				valueType = tsElementType(unwrapField) + "[]"
			}
		}

		return fmt.Sprintf("Record<string, %s>", valueType)
	}

	// Handle repeated fields
	if field.Desc.IsList() {
		elemType := tsElementType(field)
		return elemType + "[]"
	}

	// Handle google.protobuf.Timestamp fields (serialized as primitive, not as nested object)
	if annotations.IsTimestampField(field) {
		return tsTimestampType(field)
	}

	// Handle message fields
	if field.Desc.Kind() == protoreflect.MessageKind && field.Message != nil {
		return string(field.Message.Desc.Name())
	}

	// Handle enum fields
	if field.Desc.Kind() == protoreflect.EnumKind && field.Enum != nil {
		// Check for NUMBER encoding - return number type instead of enum name
		encoding := annotations.GetEnumEncoding(field)
		if encoding == http.EnumEncoding_ENUM_ENCODING_NUMBER {
			return tsNumber
		}
		return string(field.Enum.Desc.Name())
	}

	// Scalar types (use field-aware function for encoding annotations)
	return tsScalarTypeForField(field)
}

// tsElementType returns the TypeScript type for the element of a repeated field.
func tsElementType(field *protogen.Field) string {
	// Handle google.protobuf.Timestamp (serialized as primitive, not as nested object)
	if annotations.IsTimestampField(field) {
		return tsTimestampType(field)
	}
	if field.Desc.Kind() == protoreflect.MessageKind && field.Message != nil {
		return string(field.Message.Desc.Name())
	}
	if field.Desc.Kind() == protoreflect.EnumKind && field.Enum != nil {
		// Check for NUMBER encoding - return number type instead of enum name
		encoding := annotations.GetEnumEncoding(field)
		if encoding == http.EnumEncoding_ENUM_ENCODING_NUMBER {
			return tsNumber
		}
		return string(field.Enum.Desc.Name())
	}
	// Use field-aware function for encoding annotations
	return tsScalarTypeForField(field)
}

// rootUnwrapTSType returns the TypeScript type for a root-unwrapped message.
func rootUnwrapTSType(msg *protogen.Message) string {
	field := msg.Fields[0]

	if field.Desc.IsMap() {
		valueField := field.Message.Fields[1]
		valueType := tsFieldType(valueField)

		// Check for combined unwrap: root map + value unwrap
		if valueField.Desc.Kind() == protoreflect.MessageKind && valueField.Message != nil {
			unwrapField := annotations.FindUnwrapField(valueField.Message)
			if unwrapField != nil {
				// Use tsElementType since unwrapField is always repeated
				valueType = tsElementType(unwrapField) + "[]"
			}
		}

		return fmt.Sprintf("Record<string, %s>", valueType)
	}

	if field.Desc.IsList() {
		return tsElementType(field) + "[]"
	}

	return tsFieldType(field)
}

// generateEnumType writes a TypeScript string union type for a protobuf enum.
// Uses custom enum_value annotations if present, otherwise uses proto names.
func generateEnumType(p printer, enum *protogen.Enum) {
	name := string(enum.Desc.Name())
	values := enum.Values

	if len(values) == 0 {
		p("export type %s = string;", name)
		p("")
		return
	}

	var parts []string
	for _, v := range values {
		// Check for custom enum_value annotation
		customValue := annotations.GetEnumValueMapping(v)
		if customValue != "" {
			parts = append(parts, fmt.Sprintf(`"%s"`, customValue))
		} else {
			parts = append(parts, fmt.Sprintf(`"%s"`, string(v.Desc.Name())))
		}
	}

	p("export type %s = %s;", name, strings.Join(parts, " | "))
	p("")
}

// generateInterface writes a TypeScript interface for a protobuf message.
// If the message has discriminated oneofs, it generates appropriate union types.
//

func generateInterface(p printer, msg *protogen.Message) {
	name := string(msg.Desc.Name())

	// Collect discriminated oneof info
	var discriminatedOneofs []*annotations.OneofDiscriminatorInfo
	for _, oneof := range msg.Oneofs {
		info := annotations.GetOneofDiscriminatorInfo(oneof)
		if info != nil {
			discriminatedOneofs = append(discriminatedOneofs, info)
		}
	}

	// Check if any are flattened (requires type alias with intersection)
	hasFlattenedOneof := false
	for _, info := range discriminatedOneofs {
		if info.Flatten {
			hasFlattenedOneof = true
			break
		}
	}

	// Generate discriminated union types before the message
	for _, info := range discriminatedOneofs {
		generateOneofDiscriminatedUnionType(p, name, info)
	}

	if hasFlattenedOneof {
		generateFlattenedOneofInterface(p, msg, name, discriminatedOneofs)
	} else {
		generateStandardInterface(p, msg, name, discriminatedOneofs)
	}
}

// generateOneofDiscriminatedUnionType generates a TypeScript discriminated union type for a oneof.
func generateOneofDiscriminatedUnionType(p printer, msgName string, info *annotations.OneofDiscriminatorInfo) {
	unionName := msgName + snakeToUpperCamel(string(info.Oneof.Desc.Name()))

	var branches []string
	for _, variant := range info.Variants {
		var branch string
		switch {
		case info.Flatten && variant.IsMessage:
			// Flattened: { discriminator: "value", ...variant fields }
			branch = fmt.Sprintf("{ %s: \"%s\"", info.Discriminator, variant.DiscriminatorVal)
			var sb strings.Builder
			for _, childField := range variant.Field.Message.Fields {
				jsonName := childField.Desc.JSONName()
				tsType := tsFieldType(childField)
				sb.WriteString(fmt.Sprintf("; %s: %s", jsonName, tsType))
			}
			branch += sb.String()
			branch += " }"
		case variant.IsMessage:
			// Non-flattened message: { discriminator: "value", fieldName?: MessageType }
			fieldJSONName := variant.Field.Desc.JSONName()
			msgType := string(variant.Field.Message.Desc.Name())
			branch = fmt.Sprintf(
				"{ %s: \"%s\"; %s?: %s }",
				info.Discriminator,
				variant.DiscriminatorVal,
				fieldJSONName,
				msgType,
			)
		default:
			// Non-flattened scalar: { discriminator: "value", fieldName?: scalarType }
			fieldJSONName := variant.Field.Desc.JSONName()
			tsType := tsScalarTypeForField(variant.Field)
			branch = fmt.Sprintf(
				"{ %s: \"%s\"; %s?: %s }",
				info.Discriminator,
				variant.DiscriminatorVal,
				fieldJSONName,
				tsType,
			)
		}
		branches = append(branches, branch)
	}

	p("export type %s =", unionName)
	for i, branch := range branches {
		if i < len(branches)-1 {
			p("  | %s", branch)
		} else {
			p("  | %s;", branch)
		}
	}
	p("")
}

// generateFlattenedOneofInterface generates a type alias with intersection for messages
// with flattened discriminated oneofs.
func generateFlattenedOneofInterface(
	p printer,
	msg *protogen.Message,
	name string,
	discriminatedOneofs []*annotations.OneofDiscriminatorInfo,
) {
	// Build set of fields that belong to discriminated oneofs
	oneofFields := buildOneofFieldSet(discriminatedOneofs)

	// Generate base fields interface
	p("export interface %sBase {", name)
	for _, field := range msg.Fields {
		if oneofFields[field] {
			continue
		}
		if annotations.IsFlattenField(field) && field.Message != nil {
			prefix := annotations.GetFlattenPrefix(field)
			generateFlattenedFields(p, field.Message, prefix)
			continue
		}
		generateFieldDeclaration(p, field)
	}
	p("}")
	p("")

	// Generate type alias as intersection of base and all discriminated union types
	parts := []string{fmt.Sprintf("%sBase", name)}
	for _, info := range discriminatedOneofs {
		unionName := name + snakeToUpperCamel(string(info.Oneof.Desc.Name()))
		parts = append(parts, unionName)
	}
	p("export type %s = %s;", name, strings.Join(parts, " & "))
	p("")
}

// generateStandardInterface generates a standard interface, handling non-flattened
// discriminated oneofs as optional union properties.
func generateStandardInterface(
	p printer,
	msg *protogen.Message,
	name string,
	discriminatedOneofs []*annotations.OneofDiscriminatorInfo,
) {
	// Build set of fields that belong to discriminated oneofs
	oneofFields := buildOneofFieldSet(discriminatedOneofs)

	// Build map of oneof -> union type name for non-flattened
	oneofUnionNames := make(map[*protogen.Oneof]string)
	for _, info := range discriminatedOneofs {
		unionName := name + snakeToUpperCamel(string(info.Oneof.Desc.Name()))
		oneofUnionNames[info.Oneof] = unionName
	}

	p("export interface %s {", name)
	// Track which oneofs we've already emitted
	emittedOneofs := make(map[*protogen.Oneof]bool)

	for _, field := range msg.Fields {
		if oneofFields[field] {
			// For discriminated oneof fields, emit the union type once for the oneof
			if field.Oneof != nil && !emittedOneofs[field.Oneof] {
				if unionName, ok := oneofUnionNames[field.Oneof]; ok {
					oneofJSONName := string(field.Oneof.Desc.Name())
					p("  %s?: %s;", oneofJSONName, unionName)
					emittedOneofs[field.Oneof] = true
				}
			}
			continue
		}

		if annotations.IsFlattenField(field) && field.Message != nil {
			prefix := annotations.GetFlattenPrefix(field)
			generateFlattenedFields(p, field.Message, prefix)
			continue
		}

		generateFieldDeclaration(p, field)
	}
	p("}")
	p("")
}

// buildOneofFieldSet returns a set of fields that belong to discriminated oneofs.
func buildOneofFieldSet(discriminatedOneofs []*annotations.OneofDiscriminatorInfo) map[*protogen.Field]bool {
	oneofFields := make(map[*protogen.Field]bool)
	for _, info := range discriminatedOneofs {
		for _, variant := range info.Variants {
			oneofFields[variant.Field] = true
		}
	}
	return oneofFields
}

// generateFieldDeclaration generates a single TypeScript field declaration line.
func generateFieldDeclaration(p printer, field *protogen.Field) {
	jsonName := field.Desc.JSONName()
	tsType := tsFieldType(field)

	//nolint:gocritic // if-else chain is clearer than switch for distinct boolean checks
	if annotations.IsNullableField(field) {
		p("  %s: %s | null;", jsonName, tsType)
	} else if isOptionalField(field) {
		p("  %s?: %s;", jsonName, tsType)
	} else {
		p("  %s: %s;", jsonName, tsType)
	}
}

// snakeToUpperCamel converts snake_case to UpperCamelCase.
func snakeToUpperCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// generateFlattenedFields inlines child message fields at the parent level with optional prefix.
func generateFlattenedFields(p printer, childMsg *protogen.Message, prefix string) {
	for _, childField := range childMsg.Fields {
		jsonName := prefix + childField.Desc.JSONName()
		tsType := tsFieldType(childField)

		//nolint:gocritic // if-else chain is clearer than switch for distinct boolean checks
		if annotations.IsNullableField(childField) {
			p("  %s: %s | null;", jsonName, tsType)
		} else if isOptionalField(childField) {
			p("  %s?: %s;", jsonName, tsType)
		} else {
			p("  %s: %s;", jsonName, tsType)
		}
	}
}

// isOptionalField returns true if the field should be optional in TypeScript.
func isOptionalField(field *protogen.Field) bool {
	// Explicit proto3 optional
	if field.Desc.HasOptionalKeyword() {
		return true
	}
	// Message-typed fields are nullable in proto3
	if field.Desc.Kind() == protoreflect.MessageKind && !field.Desc.IsList() && !field.Desc.IsMap() {
		return true
	}
	return false
}

// tsTimestampType returns the TypeScript type for a google.protobuf.Timestamp field
// based on its timestamp_format annotation.
// UNIX_SECONDS and UNIX_MILLIS serialize as integers -> number
// RFC3339, DATE, and default serialize as strings -> string.
//
//nolint:exhaustive // Only non-default formats need explicit handling; default covers RFC3339/DATE/UNSPECIFIED
func tsTimestampType(field *protogen.Field) string {
	format := annotations.GetTimestampFormat(field)
	switch format {
	case http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS,
		http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS:
		return tsNumber
	default:
		// RFC3339, DATE, UNSPECIFIED -> string
		return tsString
	}
}

// printer is a function that prints a formatted line.
type printer func(format string, args ...interface{})
