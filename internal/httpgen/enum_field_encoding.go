package httpgen

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

// EnumFieldEncodingContext holds a message that needs a message-level marshaler because it either
// carries custom enum_value fields directly or nests (at any depth) a message that does. The
// generated MarshalJSONSebuf patches the direct enum fields and re-serializes the nested message
// fields through their own MarshalJSONSebuf, so custom enum strings propagate through the tree.
type EnumFieldEncodingContext struct {
	Message *protogen.Message
	// EnumFields are the message's direct custom-enum fields (may be empty for pure wrappers).
	EnumFields []*EnumFieldInfo
	// NestedFields are singular/repeated message fields whose type transitively contains a
	// custom-enum field, so they must be re-serialized via the child's custom marshaler.
	NestedFields []*protogen.Field
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

// nestedMessageChild returns the message type of a singular or repeated (non-map) message field,
// or nil if the field is not such a field.
func nestedMessageChild(field *protogen.Field) *protogen.Message {
	if field.Desc.IsMap() || field.Desc.Kind() != protoreflect.MessageKind {
		return nil
	}
	return field.Message
}

// mapMessageValueChild returns the value message type of a map<_, message> field, or nil.
func mapMessageValueChild(field *protogen.Field) *protogen.Message {
	if !field.Desc.IsMap() {
		return nil
	}
	valueField := field.Message.Fields[1]
	if valueField.Desc.Kind() != protoreflect.MessageKind {
		return nil
	}
	return valueField.Message
}

// messageTransitivelyHasCustomEnum reports whether msg, or any message it nests (singular,
// repeated, or map value) at any depth, has a direct custom enum_value field. The visited set
// guards against recursive message definitions.
func messageTransitivelyHasCustomEnum(msg *protogen.Message, visited map[string]bool) bool {
	if msg == nil {
		return false
	}
	key := string(msg.Desc.FullName())
	if visited[key] {
		return false
	}
	visited[key] = true

	for _, field := range msg.Fields {
		if customEnumForField(field) != nil {
			return true
		}
		if child := nestedMessageChild(field); child != nil &&
			messageTransitivelyHasCustomEnum(child, visited) {
			return true
		}
		if child := mapMessageValueChild(field); child != nil &&
			messageTransitivelyHasCustomEnum(child, visited) {
			return true
		}
	}
	return false
}

// fieldTransitivelyHasCustomEnum reports whether a message field's type (singular/repeated) reaches
// a custom enum at any depth.
func fieldTransitivelyHasCustomEnum(field *protogen.Field) bool {
	child := nestedMessageChild(field)
	if child == nil {
		return false
	}
	return messageTransitivelyHasCustomEnum(child, map[string]bool{})
}

// customEnumFieldInfo returns EnumFieldInfo for a direct custom-enum field whose enum is in the
// same Go package as the message (pkg), or nil otherwise. Cross-package direct enums are rejected
// by validateEnumFieldEncoding, since the marshaler references that package's private lookup maps.
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

// getCustomEnumFields returns the direct custom-enum fields of a message.
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

// getNestedEnumMessageFields returns the singular/repeated message fields of a message whose type
// transitively contains a custom enum (so they must be re-serialized through the child marshaler).
func getNestedEnumMessageFields(msg *protogen.Message) []*protogen.Field {
	var fields []*protogen.Field
	for _, field := range msg.Fields {
		if nestedMessageChild(field) != nil && fieldTransitivelyHasCustomEnum(field) {
			fields = append(fields, field)
		}
	}
	return fields
}

// hasCustomEnumFields reports whether a message directly carries any custom enum_value field.
func hasCustomEnumFields(msg *protogen.Message) bool {
	return len(getCustomEnumFields(msg)) > 0
}

// validateEnumFieldEncoding fails loudly for cases the Go generator cannot encode:
//   - a direct custom-enum field whose enum lives in a different Go package (the marshaler needs
//     that package's private lookup maps);
//   - a map<_, message> value whose message transitively carries a custom enum (nested re-
//     serialization through map values is not yet supported).
//
// Failing loudly avoids silently emitting raw proto enum names, which would contradict the docs.
func validateEnumFieldEncoding(file *protogen.File) error {
	return validateEnumFieldEncodingMessages(file.Messages)
}

func validateEnumFieldEncodingMessages(messages []*protogen.Message) error {
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			continue
		}
		if err := validateEnumFieldEncodingFields(msg); err != nil {
			return err
		}
		if err := validateEnumFieldEncodingMessages(msg.Messages); err != nil {
			return err
		}
	}
	return nil
}

func validateEnumFieldEncodingFields(msg *protogen.Message) error {
	pkg := msg.GoIdent.GoImportPath
	for _, field := range msg.Fields {
		if enum := customEnumForField(field); enum != nil &&
			!field.Desc.IsMap() && enum.GoIdent.GoImportPath != pkg {
			return fmt.Errorf(
				"message %s field %q references enum %s with (sebuf.http.enum_value) mappings from a "+
					"different Go package (%s); cross-package custom enum JSON encoding is not supported "+
					"by the Go generator",
				msg.GoIdent.GoName, field.Desc.Name(), enum.GoIdent.GoName, enum.GoIdent.GoImportPath,
			)
		}
		if child := mapMessageValueChild(field); child != nil &&
			messageTransitivelyHasCustomEnum(child, map[string]bool{}) {
			return fmt.Errorf(
				"message %s field %q is a map whose value message %s carries (sebuf.http.enum_value) "+
					"mappings; custom enum JSON encoding inside map values is not yet supported",
				msg.GoIdent.GoName, field.Desc.Name(), child.GoIdent.GoName,
			)
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
		enumFields := getCustomEnumFields(msg)
		nestedFields := getNestedEnumMessageFields(msg)
		if len(enumFields) > 0 || len(nestedFields) > 0 {
			*contexts = append(*contexts, &EnumFieldEncodingContext{
				Message:      msg,
				EnumFields:   enumFields,
				NestedFields: nestedFields,
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
	if nestsInt64NumberMessage(msg) {
		conflicts = append(conflicts, "int64_encoding=NUMBER (nested)")
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

// nestsInt64NumberMessage reports whether msg directly nests a message with int64 NUMBER fields,
// which would make it an int64 wrapper (also generating MarshalJSONSebuf).
func nestsInt64NumberMessage(msg *protogen.Message) bool {
	for _, field := range msg.Fields {
		if child := nestedMessageChild(field); child != nil && hasInt64NumberFields(child) {
			return true
		}
	}
	return false
}

// generateEnumFieldEncodingFile generates the *_enum_field_encoding.pb.go file if needed.
// It emits message-level MarshalJSON/UnmarshalJSON that translate enum fields between the raw
// proto value names protojson uses and the custom enum_value strings (reusing the lookup maps in
// *_enum_encoding.pb.go), and re-serialize nested messages so custom strings propagate through
// the whole message tree.
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

// marshalerNames lists the direct enum fields and nested message fields a marshaler handles,
// for the method doc comment.
func marshalerNames(ctx *EnumFieldEncodingContext) string {
	var names []string
	for _, f := range ctx.EnumFields {
		names = append(names, string(f.Field.Desc.Name()))
	}
	for _, f := range ctx.NestedFields {
		names = append(names, string(f.Desc.Name()))
	}
	return strings.Join(names, ", ")
}

// generateEnumFieldMarshalJSON emits MarshalJSONSebuf (+ MarshalJSON wrapper) that rewrites direct
// enum fields to their custom strings and re-serializes nested messages via their marshaler.
func (g *Generator) generateEnumFieldMarshalJSON(gf *protogen.GeneratedFile, ctx *EnumFieldEncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	gf.P("// MarshalJSONSebuf implements sebufMarshaler for ", msgName, ".")
	gf.P("// This method handles enum_value fields and nested messages: ", marshalerNames(ctx))
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

	gf.P("// Parse into a map to rewrite enum fields and nested messages")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.EnumFields {
		g.generateEnumFieldMarshal(gf, info)
	}
	for _, field := range ctx.NestedFields {
		g.generateNestedMessageMarshal(gf, field)
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

// generateNestedMessageMarshal re-serializes a nested message field through the child's
// MarshalJSONSebuf (forwarding opts) so custom enum strings propagate, mirroring the int64 wrapper.
// The result is written under whichever JSON key protojson emitted (camelCase, or the proto
// snake_case name under UseProtoNames).
func (g *Generator) generateNestedMessageMarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	jsonName := field.Desc.JSONName()
	keys := enumFieldJSONKeys(field)

	if field.Desc.IsList() {
		gf.P("// Re-serialize repeated \"", jsonName, "\" forwarding opts to each element")
		gf.P("if len(x.", field.GoName, ") > 0 {")
		gf.P("items := make([]json.RawMessage, 0, len(x.", field.GoName, "))")
		gf.P("for _, item := range x.", field.GoName, " {")
		gf.P("if m, ok := any(item).(interface{ MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error) }); ok {")
		gf.P("itemData, itemErr := m.MarshalJSONSebuf(opts)")
		gf.P("if itemErr != nil {")
		gf.P("return nil, itemErr")
		gf.P("}")
		gf.P("items = append(items, itemData)")
		gf.P("} else {")
		gf.P("itemData, itemErr := opts.Marshal(item)")
		gf.P("if itemErr != nil {")
		gf.P("return nil, itemErr")
		gf.P("}")
		gf.P("items = append(items, itemData)")
		gf.P("}")
		gf.P("}")
		gf.P("listData, listErr := json.Marshal(items)")
		gf.P("if listErr != nil {")
		gf.P("return nil, listErr")
		gf.P("}")
		gf.P("for _, k := range []string{", keys, "} {")
		gf.P("if _, ok := raw[k]; ok {")
		gf.P("raw[k] = listData")
		gf.P("}")
		gf.P("}")
		gf.P("}")
		gf.P()
		return
	}

	gf.P("// Re-serialize \"", jsonName, "\" forwarding opts when child supports MarshalJSONSebuf")
	gf.P("if x.", field.GoName, " != nil {")
	gf.P("if m, ok := any(x.", field.GoName,
		").(interface{ MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error) }); ok {")
	gf.P("childData, childErr := m.MarshalJSONSebuf(opts)")
	gf.P("if childErr != nil {")
	gf.P("return nil, childErr")
	gf.P("}")
	gf.P("for _, k := range []string{", keys, "} {")
	gf.P("if _, ok := raw[k]; ok {")
	gf.P("raw[k] = childData")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateEnumFieldUnmarshalJSON emits UnmarshalJSONSebuf (+ UnmarshalJSON wrapper) that rewrites
// incoming custom enum_value strings back to proto value names and delegates nested message parsing.
func (g *Generator) generateEnumFieldUnmarshalJSON(gf *protogen.GeneratedFile, ctx *EnumFieldEncodingContext) {
	msgName := ctx.Message.GoIdent.GoName

	gf.P("// UnmarshalJSONSebuf implements sebufUnmarshaler for ", msgName, ".")
	gf.P("// This method handles enum_value fields and nested messages: ", marshalerNames(ctx))
	gf.P("func (x *", msgName, ") UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {")
	gf.P("// Parse the raw JSON to rewrite custom enum_value strings and nested messages")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	for _, info := range ctx.EnumFields {
		g.generateEnumFieldUnmarshal(gf, info)
	}
	for _, field := range ctx.NestedFields {
		g.generateNestedMessageUnmarshal(gf, field)
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

// generateNestedMessageUnmarshal delegates nested message parsing to the child's UnmarshalJSONSebuf
// (forwarding opts), then converts back to protojson form, mirroring the int64 wrapper. It handles
// whichever JSON key the request used (camelCase or the proto snake_case name).
func (g *Generator) generateNestedMessageUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	jsonName := field.Desc.JSONName()
	keys := enumFieldJSONKeys(field)
	childIdent := gf.QualifiedGoIdent(field.Message.GoIdent)

	gf.P("// Handle \"", jsonName, "\" using its custom unmarshaler")
	gf.P("for _, k := range []string{", keys, "} {")
	gf.P("rawVal, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")

	if field.Desc.IsList() {
		gf.P("var rawItems []json.RawMessage")
		gf.P("if err := json.Unmarshal(rawVal, &rawItems); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("protoItems := make([]json.RawMessage, len(rawItems))")
		gf.P("for i, itemRaw := range rawItems {")
		gf.P("inner := &", childIdent, "{}")
		gf.P("if u, ok := any(inner).(interface{ UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error }); ok {")
		gf.P("if err := u.UnmarshalJSONSebuf(itemRaw, opts); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("} else if err := json.Unmarshal(itemRaw, inner); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("itemJSON, marshalErr := protojson.Marshal(inner)")
		gf.P("if marshalErr != nil {")
		gf.P("return marshalErr")
		gf.P("}")
		gf.P("protoItems[i] = itemJSON")
		gf.P("}")
		gf.P("protoJSON, marshalErr := json.Marshal(protoItems)")
		gf.P("if marshalErr != nil {")
		gf.P("return marshalErr")
		gf.P("}")
		gf.P("raw[k] = protoJSON")
	} else {
		gf.P("inner := &", childIdent, "{}")
		gf.P("if u, ok := any(inner).(interface{ UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error }); ok {")
		gf.P("if err := u.UnmarshalJSONSebuf(rawVal, opts); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("} else if err := json.Unmarshal(rawVal, inner); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("innerJSON, marshalErr := protojson.Marshal(inner)")
		gf.P("if marshalErr != nil {")
		gf.P("return marshalErr")
		gf.P("}")
		gf.P("raw[k] = innerJSON")
	}

	gf.P("}")
	gf.P()
}
