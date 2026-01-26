package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"
)

// Proto field kind constants for type checking.
const (
	kindString   = "string"
	kindBool     = "bool"
	kindInt32    = "int32"
	kindUint32   = "uint32"
	kindSint32   = "sint32"
	kindSfixed32 = "sfixed32"
	kindInt64    = "int64"
	kindSint64   = "sint64"
	kindSfixed64 = "sfixed64"
	kindUint64   = "uint64"
	kindFixed32  = "fixed32"
	kindFixed64  = "fixed64"
	kindFloat    = "float"
	kindDouble   = "double"
	kindBytes    = "bytes"
	kindEnum     = "enum"
)

// UnwrapContext holds information about messages that need unwrap JSON methods.
type UnwrapContext struct {
	// Messages that contain map fields whose value type has an unwrap field
	ContainingMessages []*UnwrapContainingMessage
}

// UnwrapContainingMessage represents a message that contains map fields with unwrap values.
type UnwrapContainingMessage struct {
	Message   *protogen.Message
	MapFields []*UnwrapMapField
}

// UnwrapMapField represents a map field whose value type has an unwrap field.
type UnwrapMapField struct {
	Field        *protogen.Field   // The map field
	ValueMessage *protogen.Message // The map value message type
	UnwrapField  *UnwrapFieldInfo  // The unwrap field info from the value message
}

// GlobalUnwrapInfo holds unwrap field information collected from all files.
// This enables cross-file unwrap resolution within the same Go package.
type GlobalUnwrapInfo struct {
	// UnwrapFields maps message full names to their unwrap field info.
	UnwrapFields map[string]*UnwrapFieldInfo
}

// NewGlobalUnwrapInfo creates a new GlobalUnwrapInfo instance.
func NewGlobalUnwrapInfo() *GlobalUnwrapInfo {
	return &GlobalUnwrapInfo{
		UnwrapFields: make(map[string]*UnwrapFieldInfo),
	}
}

// CollectGlobalUnwrapInfo scans all files to be generated and collects unwrap field information.
// This enables the generator to find unwrap annotations on messages defined in other files
// within the same Go package.
func CollectGlobalUnwrapInfo(files []*protogen.File) *GlobalUnwrapInfo {
	global := NewGlobalUnwrapInfo()
	for _, file := range files {
		if !file.Generate {
			continue
		}
		collectFileUnwrapFields(file.Messages, global)
	}
	return global
}

// collectFileUnwrapFields recursively collects unwrap fields from messages.
func collectFileUnwrapFields(messages []*protogen.Message, global *GlobalUnwrapInfo) {
	for _, msg := range messages {
		info, err := getUnwrapField(msg)
		if err != nil {
			// Log error but continue
			continue
		}
		if info != nil {
			global.UnwrapFields[string(msg.Desc.FullName())] = info
		}
		// Check nested messages too
		collectFileUnwrapFields(msg.Messages, global)
	}
}

// collectUnwrapContext analyzes all messages in a file and collects unwrap information.
func (g *Generator) collectUnwrapContext(file *protogen.File) *UnwrapContext {
	ctx := &UnwrapContext{}

	// Use global unwrap map if available (two-pass mode), otherwise fall back to single-file mode
	var globalUnwrapMap map[string]*UnwrapFieldInfo
	if g.globalUnwrap != nil {
		globalUnwrapMap = g.globalUnwrap.UnwrapFields
	} else {
		// Fallback for direct calls (e.g., in tests without full Generate() flow)
		globalUnwrapMap = collectAllUnwrapFields(file.Messages)
	}

	// Now find messages that have map fields whose value type is in the unwrap map
	findMapFieldsWithUnwrap(file.Messages, globalUnwrapMap, ctx)

	return ctx
}

// collectAllUnwrapFields recursively collects all messages with unwrap fields.
func collectAllUnwrapFields(messages []*protogen.Message) map[string]*UnwrapFieldInfo {
	result := make(map[string]*UnwrapFieldInfo)
	collectUnwrapFieldsRecursive(messages, result)
	return result
}

// collectUnwrapFieldsRecursive is a helper that recursively collects unwrap fields.
func collectUnwrapFieldsRecursive(messages []*protogen.Message, result map[string]*UnwrapFieldInfo) {
	for _, msg := range messages {
		info, err := getUnwrapField(msg)
		if err != nil {
			// Log error but continue
			continue
		}
		if info != nil {
			result[string(msg.Desc.FullName())] = info
		}
		// Check nested messages too
		collectUnwrapFieldsRecursive(msg.Messages, result)
	}
}

// findMapFieldsWithUnwrap finds messages with map fields whose values have unwrap fields.
func findMapFieldsWithUnwrap(
	messages []*protogen.Message,
	unwrapMessages map[string]*UnwrapFieldInfo,
	ctx *UnwrapContext,
) {
	for _, msg := range messages {
		mapFields := collectUnwrapMapFields(msg, unwrapMessages)

		if len(mapFields) > 0 {
			ctx.ContainingMessages = append(ctx.ContainingMessages, &UnwrapContainingMessage{
				Message:   msg,
				MapFields: mapFields,
			})
		}

		// Check nested messages too
		findMapFieldsWithUnwrap(msg.Messages, unwrapMessages, ctx)
	}
}

// collectUnwrapMapFields collects map fields whose value types have unwrap fields.
// This checks the value message directly, which works for both local and imported messages.
func collectUnwrapMapFields(msg *protogen.Message, unwrapMessages map[string]*UnwrapFieldInfo) []*UnwrapMapField {
	var mapFields []*UnwrapMapField

	for _, field := range msg.Fields {
		if !field.Desc.IsMap() {
			continue
		}

		// Get the value type of the map
		valueMsg := getMapValueMessage(field)
		if valueMsg == nil {
			continue
		}

		valueMsgFullName := string(valueMsg.Desc.FullName())

		// First check global/local cache, then check the message directly (for cross-package imports)
		unwrapInfo, ok := unwrapMessages[valueMsgFullName]
		if !ok {
			// Check imported message directly for unwrap annotation
			var err error
			unwrapInfo, err = getUnwrapField(valueMsg)
			if err != nil || unwrapInfo == nil {
				continue
			}
		}

		mapFields = append(mapFields, &UnwrapMapField{
			Field:        field,
			ValueMessage: valueMsg,
			UnwrapField:  unwrapInfo,
		})
	}

	return mapFields
}

// getMapValueMessage returns the message type of a map field's value, or nil if not a message.
func getMapValueMessage(field *protogen.Field) *protogen.Message {
	if field.Message == nil || len(field.Message.Fields) < 2 {
		return nil
	}

	// Map entry messages have exactly 2 fields: key (field 1) and value (field 2)
	const mapValueFieldNumber = 2
	for _, f := range field.Message.Fields {
		if f.Desc.Number() == mapValueFieldNumber {
			return f.Message
		}
	}
	return nil
}

// generateUnwrapFile generates the *_unwrap.pb.go file if needed.
func (g *Generator) generateUnwrapFile(file *protogen.File) error {
	ctx := g.collectUnwrapContext(file)

	// If no messages need unwrap methods, skip generation
	if len(ctx.ContainingMessages) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_unwrap.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeUnwrapImports(gf)

	for _, containing := range ctx.ContainingMessages {
		g.generateUnwrapMarshalJSON(gf, containing)
		g.generateUnwrapUnmarshalJSON(gf, containing)
	}

	return nil
}

func (g *Generator) writeUnwrapImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"encoding/json"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()
}

func (g *Generator) generateUnwrapMarshalJSON(gf *protogen.GeneratedFile, containing *UnwrapContainingMessage) {
	msgName := containing.Message.GoIdent.GoName

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method handles unwrap field serialization for map values.")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()
	gf.P("out := make(map[string]json.RawMessage)")
	gf.P()

	// Handle each field in the message
	for _, field := range containing.Message.Fields {
		fieldName := field.GoName
		jsonName := getJSONFieldName(field)

		// Check if this is one of our unwrap map fields
		var unwrapMapField *UnwrapMapField
		for _, mf := range containing.MapFields {
			if mf.Field == field {
				unwrapMapField = mf
				break
			}
		}

		switch {
		case unwrapMapField != nil:
			// This is an unwrap map field - generate unwrap logic
			g.generateUnwrapMapMarshal(gf, field, unwrapMapField, jsonName)
		case field.Desc.IsMap():
			// Regular map field
			g.generateRegularMapMarshal(gf, field, jsonName)
		case field.Desc.IsList():
			// Repeated field
			g.generateRepeatedFieldMarshal(gf, field, jsonName)
		default:
			// Scalar or message field
			g.generateScalarFieldMarshal(gf, field, fieldName, jsonName)
		}
	}

	gf.P("return json.Marshal(out)")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateUnwrapMapMarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	unwrapMapField *UnwrapMapField,
	jsonName string,
) {
	fieldName := field.GoName
	unwrapFieldName := unwrapMapField.UnwrapField.Field.GoName
	isMessageType := unwrapMapField.UnwrapField.ElementType != nil

	gf.P("// Handle unwrap map field: ", fieldName)
	gf.P("if x.", fieldName, " != nil {")
	gf.P("mapData := make(map[string]json.RawMessage)")
	gf.P("for k, wrapper := range x.", fieldName, " {")
	gf.P("if wrapper != nil {")

	if isMessageType {
		// For message types, marshal each item with protojson
		gf.P("// Marshal the unwrap field directly (the array)")
		gf.P("items := make([]json.RawMessage, 0, len(wrapper.Get", unwrapFieldName, "()))")
		gf.P("for _, item := range wrapper.Get", unwrapFieldName, "() {")
		gf.P("data, err := protojson.Marshal(item)")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P("items = append(items, data)")
		gf.P("}")
		gf.P("arrayData, err := json.Marshal(items)")
	} else {
		// For scalar types, marshal the array directly with json
		gf.P("// Marshal the unwrap field directly (the array of scalars)")
		gf.P("arrayData, err := json.Marshal(wrapper.Get", unwrapFieldName, "())")
	}

	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("mapData[k] = arrayData")
	gf.P("}")
	gf.P("}")
	gf.P("data, err := json.Marshal(mapData)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P(`out["`, jsonName, `"] = data`)
	gf.P("}")
	gf.P()
}

func (g *Generator) generateRegularMapMarshal(gf *protogen.GeneratedFile, field *protogen.Field, jsonName string) {
	fieldName := field.GoName

	gf.P("// Handle regular map field: ", fieldName)
	gf.P("if len(x.", fieldName, ") > 0 {")
	gf.P("data, err := json.Marshal(x.", fieldName, ")")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P(`out["`, jsonName, `"] = data`)
	gf.P("}")
	gf.P()
}

func (g *Generator) generateRepeatedFieldMarshal(gf *protogen.GeneratedFile, field *protogen.Field, jsonName string) {
	fieldName := field.GoName

	gf.P("// Handle repeated field: ", fieldName)
	gf.P("if len(x.", fieldName, ") > 0 {")
	// Check if it's a message type
	if field.Message != nil {
		gf.P("items := make([]json.RawMessage, 0, len(x.", fieldName, "))")
		gf.P("for _, item := range x.", fieldName, " {")
		gf.P("data, err := protojson.Marshal(item)")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P("items = append(items, data)")
		gf.P("}")
		gf.P("data, err := json.Marshal(items)")
	} else {
		gf.P("data, err := json.Marshal(x.", fieldName, ")")
	}
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P(`out["`, jsonName, `"] = data`)
	gf.P("}")
	gf.P()
}

func (g *Generator) generateScalarFieldMarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	fieldName, jsonName string,
) {
	// Check if this is an optional field or message
	if field.Message != nil {
		gf.P("// Handle message field: ", fieldName)
		gf.P("if x.", fieldName, " != nil {")
		gf.P("data, err := protojson.Marshal(x.", fieldName, ")")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P(`out["`, jsonName, `"] = data`)
		gf.P("}")
	} else {
		// Scalar field - only include if non-zero
		zeroCheck := getZeroValueCheck(field, "x."+fieldName)
		gf.P("// Handle scalar field: ", fieldName)
		gf.P("if ", zeroCheck, " {")
		gf.P("data, err := json.Marshal(x.", fieldName, ")")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P(`out["`, jsonName, `"] = data`)
		gf.P("}")
	}
	gf.P()
}

func (g *Generator) generateUnwrapUnmarshalJSON(gf *protogen.GeneratedFile, containing *UnwrapContainingMessage) {
	msgName := containing.Message.GoIdent.GoName

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method handles unwrap field deserialization for map values.")
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P()

	// Handle each field
	for _, field := range containing.Message.Fields {
		fieldName := field.GoName
		jsonName := getJSONFieldName(field)

		// Check if this is one of our unwrap map fields
		var unwrapMapField *UnwrapMapField
		for _, mf := range containing.MapFields {
			if mf.Field == field {
				unwrapMapField = mf
				break
			}
		}

		switch {
		case unwrapMapField != nil:
			g.generateUnwrapMapUnmarshal(gf, field, unwrapMapField, jsonName)
		case field.Desc.IsMap():
			g.generateRegularMapUnmarshal(gf, field, jsonName)
		case field.Desc.IsList():
			g.generateRepeatedFieldUnmarshal(gf, field, jsonName)
		default:
			g.generateScalarFieldUnmarshal(gf, field, fieldName, jsonName)
		}
	}

	gf.P("return nil")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateUnwrapMapUnmarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	unwrapMapField *UnwrapMapField,
	jsonName string,
) {
	fieldName := field.GoName
	valueTypeIdent := unwrapMapField.ValueMessage.GoIdent
	unwrapFieldName := unwrapMapField.UnwrapField.Field.GoName
	var elementTypeIdent *protogen.GoIdent
	if unwrapMapField.UnwrapField.ElementType != nil {
		ident := unwrapMapField.UnwrapField.ElementType.GoIdent
		elementTypeIdent = &ident
	}

	gf.P("// Handle unwrap map field: ", fieldName)
	gf.P(`if rawField, ok := raw["`, jsonName, `"]; ok {`)
	gf.P("var mapRaw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(rawField, &mapRaw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("x.", fieldName, " = make(map[string]*", gf.QualifiedGoIdent(valueTypeIdent), ")")
	gf.P("for k, arrayRaw := range mapRaw {")
	gf.P("var itemsRaw []json.RawMessage")
	gf.P("if err := json.Unmarshal(arrayRaw, &itemsRaw); err != nil {")
	gf.P("return err")
	gf.P("}")
	if elementTypeIdent != nil {
		gf.P("items := make([]*", gf.QualifiedGoIdent(*elementTypeIdent), ", 0, len(itemsRaw))")
		gf.P("for _, itemRaw := range itemsRaw {")
		gf.P("item := &", *elementTypeIdent, "{}")
		gf.P("if err := protojson.Unmarshal(itemRaw, item); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("items = append(items, item)")
		gf.P("}")
	} else {
		// Scalar type - need different handling
		gf.P("var items []", getScalarTypeName(unwrapMapField.UnwrapField.Field))
		gf.P("if err := json.Unmarshal(arrayRaw, &items); err != nil {")
		gf.P("return err")
		gf.P("}")
	}
	gf.P("x.", fieldName, "[k] = &", valueTypeIdent, "{", unwrapFieldName, ": items}")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateRegularMapUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field, jsonName string) {
	fieldName := field.GoName

	gf.P("// Handle regular map field: ", fieldName)
	gf.P(`if rawField, ok := raw["`, jsonName, `"]; ok {`)
	gf.P("if err := json.Unmarshal(rawField, &x.", fieldName, "); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateRepeatedFieldUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field, jsonName string) {
	fieldName := field.GoName

	gf.P("// Handle repeated field: ", fieldName)
	gf.P(`if rawField, ok := raw["`, jsonName, `"]; ok {`)
	if field.Message != nil {
		elementTypeIdent := field.Message.GoIdent
		gf.P("var itemsRaw []json.RawMessage")
		gf.P("if err := json.Unmarshal(rawField, &itemsRaw); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("x.", fieldName, " = make([]*", gf.QualifiedGoIdent(elementTypeIdent), ", 0, len(itemsRaw))")
		gf.P("for _, itemRaw := range itemsRaw {")
		gf.P("item := &", elementTypeIdent, "{}")
		gf.P("if err := protojson.Unmarshal(itemRaw, item); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("x.", fieldName, " = append(x.", fieldName, ", item)")
		gf.P("}")
	} else {
		gf.P("if err := json.Unmarshal(rawField, &x.", fieldName, "); err != nil {")
		gf.P("return err")
		gf.P("}")
	}
	gf.P("}")
	gf.P()
}

func (g *Generator) generateScalarFieldUnmarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	fieldName, jsonName string,
) {
	gf.P("// Handle field: ", fieldName)
	gf.P(`if rawField, ok := raw["`, jsonName, `"]; ok {`)
	if field.Message != nil {
		gf.P("x.", fieldName, " = &", field.Message.GoIdent.GoName, "{}")
		gf.P("if err := protojson.Unmarshal(rawField, x.", fieldName, "); err != nil {")
		gf.P("return err")
		gf.P("}")
	} else {
		gf.P("if err := json.Unmarshal(rawField, &x.", fieldName, "); err != nil {")
		gf.P("return err")
		gf.P("}")
	}
	gf.P("}")
	gf.P()
}

// getJSONFieldName returns the JSON field name for a protobuf field.
func getJSONFieldName(field *protogen.Field) string {
	// Use the proto JSON name (which is camelCase version of the proto field name)
	return field.Desc.JSONName()
}

// getZeroValueCheck returns a condition that checks if a field is non-zero.
func getZeroValueCheck(field *protogen.Field, fieldExpr string) string {
	switch field.Desc.Kind().String() {
	case kindString:
		return fieldExpr + ` != ""`
	case kindBool:
		return fieldExpr
	case kindInt32, kindSint32, kindSfixed32, kindInt64, kindSint64, kindSfixed64,
		kindUint32, kindFixed32, kindUint64, kindFixed64, kindFloat, kindDouble:
		return fieldExpr + " != 0"
	case kindBytes:
		return "len(" + fieldExpr + ") > 0"
	case kindEnum:
		return fieldExpr + " != 0"
	default:
		return fieldExpr + " != nil"
	}
}

// getScalarTypeName returns the Go type name for a scalar field.
func getScalarTypeName(field *protogen.Field) string {
	switch field.Desc.Kind().String() {
	case kindString:
		return "string"
	case kindBool:
		return "bool"
	case kindInt32, kindSint32, kindSfixed32:
		return "int32"
	case kindInt64, kindSint64, kindSfixed64:
		return "int64"
	case kindUint32, kindFixed32:
		return "uint32"
	case kindUint64, kindFixed64:
		return "uint64"
	case kindFloat:
		return "float32"
	case kindDouble:
		return "float64"
	case kindBytes:
		return "[]byte"
	default:
		return "interface{}"
	}
}
