package clientgen

import (
	"fmt"
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

// EnumFieldEncodingContext holds a message that needs a message-level marshaler to apply
// custom enum_value strings to its enum fields (protojson emits the raw proto value names).
type EnumFieldEncodingContext struct {
	Message    *protogen.Message
	EnumFields []*EnumFieldInfo
}

// customEnumFieldInfo returns EnumFieldInfo for a field whose enum type carries custom
// enum_value mappings (and is not NUMBER-encoded), or nil if the field is not such a field.
// Only enums generated in the same Go package as the message are handled, since the marshaler
// references the package-private xToJSON/xFromJSON lookup maps from *_enum_encoding.pb.go.
func customEnumFieldInfo(field *protogen.Field, pkg protogen.GoImportPath) *EnumFieldInfo {
	// NUMBER encoding never carries custom string values (rejected by validateEnumAnnotations).
	if annotations.GetEnumEncoding(field) == http.EnumEncoding_ENUM_ENCODING_NUMBER {
		return nil
	}

	switch {
	case field.Desc.IsMap():
		if field.Desc.MapValue().Kind() != protoreflect.EnumKind {
			return nil
		}
		valueEnum := field.Message.Fields[1].Enum
		if !isSamePackageCustomEnum(valueEnum, pkg) {
			return nil
		}
		return &EnumFieldInfo{Field: field, Enum: valueEnum, Shape: enumShapeMap}
	case field.Desc.Kind() == protoreflect.EnumKind:
		if !isSamePackageCustomEnum(field.Enum, pkg) {
			return nil
		}
		shape := enumShapeSingular
		if field.Desc.IsList() {
			shape = enumShapeRepeated
		}
		return &EnumFieldInfo{Field: field, Enum: field.Enum, Shape: shape}
	default:
		return nil
	}
}

// isSamePackageCustomEnum reports whether the enum has custom values and is generated in pkg.
func isSamePackageCustomEnum(enum *protogen.Enum, pkg protogen.GoImportPath) bool {
	if enum == nil || !annotations.HasAnyEnumValueMapping(enum) {
		return false
	}
	return enum.GoIdent.GoImportPath == pkg
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

// generateEnumFieldMarshalJSON emits MarshalJSONSebuf (+ MarshalJSON wrapper) that rewrites
// enum fields from proto value names to custom enum_value strings.
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
func (g *Generator) generateEnumFieldMarshal(gf *protogen.GeneratedFile, info *EnumFieldInfo) {
	jsonName := info.Field.Desc.JSONName()
	lower := annotations.LowerFirst(info.Enum.GoIdent.GoName)
	toJSON := lower + "ToJSON"
	fromJSON := lower + "FromJSON"

	gf.P("// Rewrite ", info.Field.Desc.Name(), " to custom enum_value strings")
	gf.P(`if v, ok := raw["`, jsonName, `"]; ok {`)

	switch info.Shape {
	case enumShapeSingular:
		gf.P("var s string")
		gf.P("if err := json.Unmarshal(v, &s); err == nil {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(`, toJSON, `[e])`)
		gf.P("}")
		gf.P("}")
	case enumShapeRepeated:
		gf.P("var arr []string")
		gf.P("if err := json.Unmarshal(v, &arr); err == nil {")
		gf.P("for i, s := range arr {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("arr[i] = ", toJSON, "[e]")
		gf.P("}")
		gf.P("}")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(arr)`)
		gf.P("}")
	case enumShapeMap:
		gf.P("var m map[string]string")
		gf.P("if err := json.Unmarshal(v, &m); err == nil {")
		gf.P("for k, s := range m {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("m[k] = ", toJSON, "[e]")
		gf.P("}")
		gf.P("}")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(m)`)
		gf.P("}")
	}

	gf.P("}")
	gf.P()
}

// generateEnumFieldUnmarshalJSON emits UnmarshalJSONSebuf (+ UnmarshalJSON wrapper) that rewrites
// incoming custom enum_value strings back to proto value names so protojson accepts them.
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
// The lookup accepts both custom values and proto names (both are keys in xFromJSON), so it is
// idempotent for clients that already send proto names; numeric values fall through untouched.
func (g *Generator) generateEnumFieldUnmarshal(gf *protogen.GeneratedFile, info *EnumFieldInfo) {
	jsonName := info.Field.Desc.JSONName()
	lower := annotations.LowerFirst(info.Enum.GoIdent.GoName)
	fromJSON := lower + "FromJSON"

	gf.P("// Rewrite ", info.Field.Desc.Name(), " from custom enum_value strings to proto names")
	gf.P(`if v, ok := raw["`, jsonName, `"]; ok {`)

	switch info.Shape {
	case enumShapeSingular:
		gf.P("var s string")
		gf.P("if err := json.Unmarshal(v, &s); err == nil {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(e.String())`)
		gf.P("}")
		gf.P("}")
	case enumShapeRepeated:
		gf.P("var arr []string")
		gf.P("if err := json.Unmarshal(v, &arr); err == nil {")
		gf.P("for i, s := range arr {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("arr[i] = e.String()")
		gf.P("}")
		gf.P("}")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(arr)`)
		gf.P("}")
	case enumShapeMap:
		gf.P("var m map[string]string")
		gf.P("if err := json.Unmarshal(v, &m); err == nil {")
		gf.P("for k, s := range m {")
		gf.P("if e, ok := ", fromJSON, "[s]; ok {")
		gf.P("m[k] = e.String()")
		gf.P("}")
		gf.P("}")
		gf.P(`raw["`, jsonName, `"], _ = json.Marshal(m)`)
		gf.P("}")
	}

	gf.P("}")
	gf.P()
}
