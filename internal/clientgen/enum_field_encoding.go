package clientgen

import (
	"fmt"
	"strconv"
	"strings"

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

// customEnumFieldInfo returns EnumFieldInfo for a field whose enum type carries custom
// enum_value mappings and is generated in the same Go package as the message (pkg), or nil
// otherwise. The marshaler references the package-private xToJSON/xFromJSON lookup maps from
// *_enum_encoding.pb.go, so cross-package enums are handled separately by validateEnumFieldEncoding
// (which fails loudly rather than silently emitting proto names).
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

// getCustomEnumFields returns the custom-enum fields of a message (direct fields only).
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

// hasCustomEnumFields reports whether a message has any custom-enum field needing a marshaler.
func hasCustomEnumFields(msg *protogen.Message) bool {
	return len(getCustomEnumFields(msg)) > 0
}

// validateEnumFieldEncoding fails loudly when a message field references a custom enum_value enum
// from a different Go package. The generated marshaler relies on the package-private lookup maps
// emitted alongside the enum, so a cross-package enum cannot be patched; emitting proto names
// silently would contradict the OpenAPI docs. Cross-package support is tracked as a follow-up.
func validateEnumFieldEncoding(file *protogen.File) error {
	return validateEnumFieldEncodingMessages(file.Messages)
}

func validateEnumFieldEncodingMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}
		pkg := msg.GoIdent.GoImportPath
		for _, field := range msg.Fields {
			enum := customEnumForField(field)
			if enum == nil || enum.GoIdent.GoImportPath == pkg {
				continue
			}
			return fmt.Errorf(
				"message %s field %q references enum %s with (sebuf.http.enum_value) mappings from a "+
					"different Go package (%s); cross-package custom enum JSON encoding is not supported "+
					"by the Go generator",
				msg.GoIdent.GoName, field.Desc.Name(), enum.GoIdent.GoName, enum.GoIdent.GoImportPath,
			)
		}
		if err := validateEnumFieldEncodingMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

// collectEnumFieldEncodingContext gathers messages that need a custom enum-field marshaler.
func collectEnumFieldEncodingContext(file *protogen.File) []*EnumFieldEncodingContext {
	var contexts []*EnumFieldEncodingContext
	collectEnumFieldEncodingMessages(file.Messages, &contexts)
	return contexts
}

func collectEnumFieldEncodingMessages(
	messages []*protogen.Message,
	contexts *[]*EnumFieldEncodingContext,
) {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}
		if enumFields := getCustomEnumFields(msg); len(enumFields) > 0 {
			*contexts = append(*contexts, &EnumFieldEncodingContext{
				Message:    msg,
				EnumFields: enumFields,
			})
		}
		collectEnumFieldEncodingMessages(msg.Messages, contexts)
	}
}

// EnumFieldEncodingContext holds a message that needs a message-level marshaler to apply
// custom enum_value strings to its enum fields (protojson emits the raw proto value names).
type EnumFieldEncodingContext struct {
	Message    *protogen.Message
	EnumFields []*EnumFieldInfo
}

// checkEnumMarshalJSONConflict returns an error if a message that needs a custom enum-field
// marshaler also carries another MarshalJSON-generating annotation. Only one feature can own a
// message's MarshalJSON/UnmarshalJSON methods, so combining them would produce duplicate method
// declarations. Fail fast with a clear message (matching flatten/oneof behavior).
func checkEnumMarshalJSONConflict(msg *protogen.Message) error {
	var conflicts []string

	if hasInt64NumberFields(msg) {
		conflicts = append(conflicts, "int64_encoding=NUMBER")
	}
	if hasBytesEncodingFields(msg) {
		conflicts = append(conflicts, "bytes_encoding")
	}
	if hasNullableFields(msg) {
		conflicts = append(conflicts, "nullable")
	}
	if hasEmptyBehaviorFields(msg) {
		conflicts = append(conflicts, "empty_behavior")
	}
	if hasTimestampFormatFields(msg) {
		conflicts = append(conflicts, "timestamp_format")
	}
	if hasFlattenFields(msg) {
		conflicts = append(conflicts, "flatten")
	}
	if hasOneofDiscriminator(msg) {
		conflicts = append(conflicts, "oneof_config")
	}
	if annotations.IsRootUnwrap(msg) {
		conflicts = append(conflicts, "unwrap")
	}

	if len(conflicts) > 0 {
		return fmt.Errorf(
			"message %s: enum_value requires MarshalJSON but conflicts with %s (also requires MarshalJSON) -- "+
				"only one MarshalJSON-generating feature is supported per message",
			msg.GoIdent.GoName, strings.Join(conflicts, ", "),
		)
	}

	return nil
}

// generateEnumFieldEncodingFile generates the *_enum_field_encoding.pb.go file if needed.
// It emits message-level MarshalJSON/UnmarshalJSON that translate enum fields between the raw
// proto value names protojson uses and the custom enum_value strings, reusing the lookup maps
// generated in *_enum_encoding.pb.go.
func (g *Generator) generateEnumFieldEncodingFile(file *protogen.File) error {
	if err := validateEnumFieldEncoding(file); err != nil {
		return err
	}

	contexts := collectEnumFieldEncodingContext(file)
	if len(contexts) == 0 {
		return nil
	}

	for _, ctx := range contexts {
		if err := checkEnumMarshalJSONConflict(ctx.Message); err != nil {
			return err
		}
	}

	filename := file.GeneratedFilenamePrefix + "_enum_field_encoding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeEnumFieldEncodingImports(gf)

	for _, ctx := range contexts {
		g.generateEnumFieldMarshalJSON(gf, ctx)
		g.generateEnumFieldUnmarshalJSON(gf, ctx)
	}

	return nil
}

func (g *Generator) writeEnumFieldEncodingImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
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

// generateEnumFieldMarshalJSON emits MarshalJSONSebuf (+ MarshalJSON wrapper) that rewrites
// enum fields from proto value names to custom enum_value strings.
//
//nolint:dupl // Code generation patterns naturally have similar structure across encoding types
func (g *Generator) generateEnumFieldMarshalJSON(gf *protogen.GeneratedFile, ctx *EnumFieldEncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.EnumFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// MarshalJSONSebuf implements sebufMarshaler for ", msgName, ".")
	gf.P("// This method handles enum_value fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") MarshalJSONSebuf(opts protojson.MarshalOptions) ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()

	gf.P("// Use protojson for base serialization (handles all other fields correctly)")
	gf.P("data, err := opts.Marshal(x)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	gf.P("// Parse into a map to rewrite enum fields to their custom enum_value strings")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.EnumFields {
		g.generateEnumFieldMarshal(gf, info)
	}

	gf.P("return json.Marshal(raw)")
	gf.P("}")
	gf.P()

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("return x.MarshalJSONSebuf(protojson.MarshalOptions{})")
	gf.P("}")
	gf.P()
}

// generateEnumFieldMarshal emits the map-patching code for one enum field (proto name -> custom).
// It patches both the JSON name and proto name keys so UseProtoNames output is handled.
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

// generateEnumFieldUnmarshalJSON emits UnmarshalJSONSebuf (+ UnmarshalJSON wrapper) that rewrites
// incoming custom enum_value strings back to proto value names so protojson accepts them.
//
//nolint:dupl // Code generation patterns naturally have similar structure across encoding types
func (g *Generator) generateEnumFieldUnmarshalJSON(gf *protogen.GeneratedFile, ctx *EnumFieldEncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	var fieldNames []string
	for _, f := range ctx.EnumFields {
		fieldNames = append(fieldNames, string(f.Field.Desc.Name()))
	}

	gf.P("// UnmarshalJSONSebuf implements sebufUnmarshaler for ", msgName, ".")
	gf.P("// This method handles enum_value fields: ", strings.Join(fieldNames, ", "))
	gf.P("func (x *", msgName, ") UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {")
	gf.P("// Parse the raw JSON to rewrite custom enum_value strings back to proto names")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.EnumFields {
		g.generateEnumFieldUnmarshal(gf, info)
	}

	gf.P("// Re-marshal with proto value names for protojson")
	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()
	gf.P("// Use protojson to unmarshal the rest")
	gf.P("return opts.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("return x.UnmarshalJSONSebuf(data, protojson.UnmarshalOptions{})")
	gf.P("}")
	gf.P()
}

// generateEnumFieldUnmarshal emits the map-patching code for one enum field (custom -> proto name).
// It patches both the JSON name and proto name keys (protojson.Unmarshal accepts either), and the
// lookup accepts both custom values and proto names, so it is idempotent for clients that already
// send proto names; numeric values fall through untouched.
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
