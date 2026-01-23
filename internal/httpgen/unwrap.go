package httpgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
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

// collectUnwrapContext analyzes all messages in a file and collects unwrap information.
func (g *Generator) collectUnwrapContext(file *protogen.File) (*UnwrapContext, error) {
	ctx := &UnwrapContext{}

	// First, collect all messages that have unwrap fields (including nested messages)
	unwrapMessages := make(map[string]*UnwrapFieldInfo)
	var collectUnwrapFields func(messages []*protogen.Message)
	collectUnwrapFields = func(messages []*protogen.Message) {
		for _, msg := range messages {
			info, err := getUnwrapField(msg)
			if err != nil {
				// Log error but continue
				continue
			}
			if info != nil {
				unwrapMessages[string(msg.Desc.FullName())] = info
			}
			// Check nested messages too
			collectUnwrapFields(msg.Messages)
		}
	}
	collectUnwrapFields(file.Messages)

	// Now find messages that have map fields whose value type is in unwrapMessages
	var findMapFields func(messages []*protogen.Message)
	findMapFields = func(messages []*protogen.Message) {
		for _, msg := range messages {
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

				// Check if value message has an unwrap field
				unwrapInfo, ok := unwrapMessages[string(valueMsg.Desc.FullName())]
				if !ok {
					continue
				}

				mapFields = append(mapFields, &UnwrapMapField{
					Field:        field,
					ValueMessage: valueMsg,
					UnwrapField:  unwrapInfo,
				})
			}

			if len(mapFields) > 0 {
				ctx.ContainingMessages = append(ctx.ContainingMessages, &UnwrapContainingMessage{
					Message:   msg,
					MapFields: mapFields,
				})
			}

			// Check nested messages too
			findMapFields(msg.Messages)
		}
	}
	findMapFields(file.Messages)

	return ctx, nil
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
	ctx, err := g.collectUnwrapContext(file)
	if err != nil {
		return err
	}

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

		if unwrapMapField != nil {
			// This is an unwrap map field - generate unwrap logic
			g.generateUnwrapMapMarshal(gf, field, unwrapMapField, jsonName)
		} else if field.Desc.IsMap() {
			// Regular map field
			g.generateRegularMapMarshal(gf, field, jsonName)
		} else if field.Desc.IsList() {
			// Repeated field
			g.generateRepeatedFieldMarshal(gf, field, jsonName)
		} else {
			// Scalar or message field
			g.generateScalarFieldMarshal(gf, field, fieldName, jsonName)
		}
	}

	gf.P("return json.Marshal(out)")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateUnwrapMapMarshal(gf *protogen.GeneratedFile, field *protogen.Field, unwrapMapField *UnwrapMapField, jsonName string) {
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

func (g *Generator) generateScalarFieldMarshal(gf *protogen.GeneratedFile, field *protogen.Field, fieldName, jsonName string) {
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

		if unwrapMapField != nil {
			g.generateUnwrapMapUnmarshal(gf, field, unwrapMapField, jsonName)
		} else if field.Desc.IsMap() {
			g.generateRegularMapUnmarshal(gf, field, jsonName)
		} else if field.Desc.IsList() {
			g.generateRepeatedFieldUnmarshal(gf, field, jsonName)
		} else {
			g.generateScalarFieldUnmarshal(gf, field, fieldName, jsonName)
		}
	}

	gf.P("return nil")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateUnwrapMapUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field, unwrapMapField *UnwrapMapField, jsonName string) {
	fieldName := field.GoName
	valueTypeName := unwrapMapField.ValueMessage.GoIdent.GoName
	unwrapFieldName := unwrapMapField.UnwrapField.Field.GoName
	elementTypeName := ""
	if unwrapMapField.UnwrapField.ElementType != nil {
		elementTypeName = unwrapMapField.UnwrapField.ElementType.GoIdent.GoName
	}

	gf.P("// Handle unwrap map field: ", fieldName)
	gf.P(`if rawField, ok := raw["`, jsonName, `"]; ok {`)
	gf.P("var mapRaw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(rawField, &mapRaw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("x.", fieldName, " = make(map[string]*", valueTypeName, ")")
	gf.P("for k, arrayRaw := range mapRaw {")
	gf.P("var itemsRaw []json.RawMessage")
	gf.P("if err := json.Unmarshal(arrayRaw, &itemsRaw); err != nil {")
	gf.P("return err")
	gf.P("}")
	if elementTypeName != "" {
		gf.P("items := make([]*", elementTypeName, ", 0, len(itemsRaw))")
		gf.P("for _, itemRaw := range itemsRaw {")
		gf.P("item := &", elementTypeName, "{}")
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
	gf.P("x.", fieldName, "[k] = &", valueTypeName, "{", unwrapFieldName, ": items}")
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
		elementTypeName := field.Message.GoIdent.GoName
		gf.P("var itemsRaw []json.RawMessage")
		gf.P("if err := json.Unmarshal(rawField, &itemsRaw); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("x.", fieldName, " = make([]*", elementTypeName, ", 0, len(itemsRaw))")
		gf.P("for _, itemRaw := range itemsRaw {")
		gf.P("item := &", elementTypeName, "{}")
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

func (g *Generator) generateScalarFieldUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field, fieldName, jsonName string) {
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
	case "string":
		return fieldExpr + ` != ""`
	case "bool":
		return fieldExpr
	case "int32", "sint32", "sfixed32", "int64", "sint64", "sfixed64",
		"uint32", "fixed32", "uint64", "fixed64", "float", "double":
		return fieldExpr + " != 0"
	case "bytes":
		return "len(" + fieldExpr + ") > 0"
	case "enum":
		return fieldExpr + " != 0"
	default:
		return fieldExpr + " != nil"
	}
}

// getScalarTypeName returns the Go type name for a scalar field.
func getScalarTypeName(field *protogen.Field) string {
	switch field.Desc.Kind().String() {
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "int32", "sint32", "sfixed32":
		return "int32"
	case "int64", "sint64", "sfixed64":
		return "int64"
	case "uint32", "fixed32":
		return "uint32"
	case "uint64", "fixed64":
		return "uint64"
	case "float":
		return "float32"
	case "double":
		return "float64"
	case "bytes":
		return "[]byte"
	default:
		return "interface{}"
	}
}

// snakeToCamel converts snake_case to camelCase.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
