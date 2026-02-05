package httpgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// Proto field kind constants for type checking.
const (
	kindString    = "string"
	kindBool      = "bool"
	kindInt32     = "int32"
	kindUint32    = "uint32"
	kindSint32    = "sint32"
	kindSfixed32  = "sfixed32"
	kindInt64     = "int64"
	kindSint64    = "sint64"
	kindSfixed64  = "sfixed64"
	kindUint64    = "uint64"
	kindFixed32   = "fixed32"
	kindFixed64   = "fixed64"
	kindFloat     = "float"
	kindDouble    = "double"
	kindBytes     = "bytes"
	kindEnum      = "enum"
	kindInterface = "interface{}"
)

// UnwrapContext holds information about messages that need unwrap JSON methods.
type UnwrapContext struct {
	// Messages that contain map fields whose value type has an unwrap field
	ContainingMessages []*UnwrapContainingMessage
	// Messages that have root-level unwrap (single field with unwrap on map or repeated)
	RootUnwrapMessages []*RootUnwrapMessage
}

// RootUnwrapMessage represents a message that should unwrap at the root level.
// This means the entire message serializes to just the unwrap field's value.
type RootUnwrapMessage struct {
	Message      *protogen.Message            // The message with root unwrap
	UnwrapField  *protogen.Field              // The single field with unwrap=true
	IsMap        bool                         // True if the unwrap field is a map
	ValueMessage *protogen.Message            // For maps: the map value message type
	ValueUnwrap  *annotations.UnwrapFieldInfo // For maps: if the value message also has an unwrap field
}

// UnwrapContainingMessage represents a message that contains map fields with unwrap values.
type UnwrapContainingMessage struct {
	Message   *protogen.Message
	MapFields []*UnwrapMapField
}

// UnwrapMapField represents a map field whose value type has an unwrap field.
type UnwrapMapField struct {
	Field        *protogen.Field              // The map field
	ValueMessage *protogen.Message            // The map value message type
	UnwrapField  *annotations.UnwrapFieldInfo // The unwrap field info from the value message
}

// GlobalUnwrapInfo holds unwrap field information collected from all files.
// This enables cross-file unwrap resolution within the same Go package.
type GlobalUnwrapInfo struct {
	// UnwrapFields maps message full names to their unwrap field info.
	UnwrapFields map[string]*annotations.UnwrapFieldInfo
}

// NewGlobalUnwrapInfo creates a new GlobalUnwrapInfo instance.
func NewGlobalUnwrapInfo() *GlobalUnwrapInfo {
	return &GlobalUnwrapInfo{
		UnwrapFields: make(map[string]*annotations.UnwrapFieldInfo),
	}
}

// CollectGlobalUnwrapInfo scans all files to be generated and collects unwrap field information.
// This enables the generator to find unwrap annotations on messages defined in other files
// within the same Go package. Returns an error if any unwrap annotation is invalid.
func CollectGlobalUnwrapInfo(files []*protogen.File) (*GlobalUnwrapInfo, error) {
	global := NewGlobalUnwrapInfo()
	for _, file := range files {
		if !file.Generate {
			continue
		}
		if err := collectFileUnwrapFields(file.Messages, global); err != nil {
			return nil, fmt.Errorf("file %s: %w", file.Desc.Path(), err)
		}
	}
	return global, nil
}

// collectFileUnwrapFields recursively collects unwrap fields from messages.
// Returns an error if any unwrap annotation is invalid (fail-hard).
func collectFileUnwrapFields(messages []*protogen.Message, global *GlobalUnwrapInfo) error {
	for _, msg := range messages {
		info, err := annotations.GetUnwrapField(msg)
		if err != nil {
			return fmt.Errorf("collecting unwrap fields for message %s: %w", msg.Desc.FullName(), err)
		}
		if info != nil {
			global.UnwrapFields[string(msg.Desc.FullName())] = info
		}
		// Check nested messages too
		if err = collectFileUnwrapFields(msg.Messages, global); err != nil {
			return err
		}
	}
	return nil
}

// collectUnwrapContext analyzes all messages in a file and collects unwrap information.
// Returns an error if any unwrap annotation is invalid (fail-hard).
func (g *Generator) collectUnwrapContext(file *protogen.File) (*UnwrapContext, error) {
	ctx := &UnwrapContext{}

	// Use global unwrap map if available (two-pass mode), otherwise fall back to single-file mode
	var globalUnwrapMap map[string]*annotations.UnwrapFieldInfo
	if g.globalUnwrap != nil {
		globalUnwrapMap = g.globalUnwrap.UnwrapFields
	} else {
		// Fallback for direct calls (e.g., in tests without full Generate() flow)
		var err error
		globalUnwrapMap, err = collectAllUnwrapFields(file.Messages)
		if err != nil {
			return nil, err
		}
	}

	// Collect root unwrap messages (single field with unwrap on map/repeated)
	collectRootUnwrapMessages(file.Messages, globalUnwrapMap, ctx)

	// Now find messages that have map fields whose value type is in the unwrap map
	// Only include messages that are NOT root unwrap messages (those are handled separately)
	findMapFieldsWithUnwrap(file.Messages, globalUnwrapMap, ctx)

	return ctx, nil
}

// collectAllUnwrapFields recursively collects all messages with unwrap fields.
// Returns an error if any unwrap annotation is invalid (fail-hard).
func collectAllUnwrapFields(messages []*protogen.Message) (map[string]*annotations.UnwrapFieldInfo, error) {
	result := make(map[string]*annotations.UnwrapFieldInfo)
	if err := collectUnwrapFieldsRecursive(messages, result); err != nil {
		return nil, err
	}
	return result, nil
}

// collectUnwrapFieldsRecursive is a helper that recursively collects unwrap fields.
// Returns an error if any unwrap annotation is invalid (fail-hard).
func collectUnwrapFieldsRecursive(messages []*protogen.Message, result map[string]*annotations.UnwrapFieldInfo) error {
	for _, msg := range messages {
		info, err := annotations.GetUnwrapField(msg)
		if err != nil {
			return fmt.Errorf("collecting unwrap fields for message %s: %w", msg.Desc.FullName(), err)
		}
		if info != nil {
			fullName := string(msg.Desc.FullName())
			result[fullName] = info
		}
		// Check nested messages too
		if err = collectUnwrapFieldsRecursive(msg.Messages, result); err != nil {
			return err
		}
	}
	return nil
}

// collectRootUnwrapMessages finds messages that have root-level unwrap.
// Root unwrap means the message has exactly one field with unwrap=true on a map or repeated field.
func collectRootUnwrapMessages(
	messages []*protogen.Message,
	unwrapMessages map[string]*annotations.UnwrapFieldInfo,
	ctx *UnwrapContext,
) {
	for _, msg := range messages {
		info, ok := unwrapMessages[string(msg.Desc.FullName())]
		if !ok || !info.IsRootUnwrap {
			// Check nested messages
			collectRootUnwrapMessages(msg.Messages, unwrapMessages, ctx)
			continue
		}

		rootUnwrap := &RootUnwrapMessage{
			Message:     msg,
			UnwrapField: info.Field,
			IsMap:       info.IsMapField,
		}

		// For map fields, get the value message and check if it also has unwrap
		if info.IsMapField {
			valueMsg := getMapValueMessage(info.Field)
			if valueMsg != nil {
				rootUnwrap.ValueMessage = valueMsg
				// Check if the value message also has an unwrap field (combined unwrap)
				if valueUnwrap, valueErr := annotations.GetUnwrapField(
					valueMsg,
				); valueErr == nil &&
					valueUnwrap != nil {
					rootUnwrap.ValueUnwrap = valueUnwrap
				}
			}
		}

		ctx.RootUnwrapMessages = append(ctx.RootUnwrapMessages, rootUnwrap)

		// Check nested messages
		collectRootUnwrapMessages(msg.Messages, unwrapMessages, ctx)
	}
}

// findMapFieldsWithUnwrap finds messages with map fields whose values have unwrap fields.
func findMapFieldsWithUnwrap(
	messages []*protogen.Message,
	unwrapMessages map[string]*annotations.UnwrapFieldInfo,
	ctx *UnwrapContext,
) {
	for _, msg := range messages {
		// Skip if this message is already a root unwrap message
		if isRootUnwrapMessage(msg, ctx) {
			// Check nested messages too
			findMapFieldsWithUnwrap(msg.Messages, unwrapMessages, ctx)
			continue
		}

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

// isRootUnwrapMessage checks if a message is already in the root unwrap list.
func isRootUnwrapMessage(msg *protogen.Message, ctx *UnwrapContext) bool {
	for _, rootUnwrap := range ctx.RootUnwrapMessages {
		if rootUnwrap.Message == msg {
			return true
		}
	}
	return false
}

// collectUnwrapMapFields collects map fields whose value types have unwrap fields.
// This checks the value message directly, which works for both local and imported messages.
func collectUnwrapMapFields(
	msg *protogen.Message,
	unwrapMessages map[string]*annotations.UnwrapFieldInfo,
) []*UnwrapMapField {
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
			unwrapInfo, err = annotations.GetUnwrapField(valueMsg)
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
	ctx, err := g.collectUnwrapContext(file)
	if err != nil {
		return fmt.Errorf("collecting unwrap context for %s: %w", file.Desc.Path(), err)
	}

	// If no messages need unwrap methods, skip generation
	if len(ctx.ContainingMessages) == 0 && len(ctx.RootUnwrapMessages) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_unwrap.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeUnwrapImports(gf)

	// Generate root unwrap methods first
	for _, rootUnwrap := range ctx.RootUnwrapMessages {
		if rootUnwrap.IsMap {
			g.generateRootMapUnwrapMarshalJSON(gf, rootUnwrap)
			g.generateRootMapUnwrapUnmarshalJSON(gf, rootUnwrap)
		} else {
			g.generateRootRepeatedUnwrapMarshalJSON(gf, rootUnwrap)
			g.generateRootRepeatedUnwrapUnmarshalJSON(gf, rootUnwrap)
		}
	}

	// Generate map-value unwrap methods
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
		return kindInterface
	}
}

// =============================================================================
// Root Unwrap Generation Methods
// =============================================================================

// generateRootMapUnwrapMarshalJSON generates MarshalJSON for root-level map unwrap.
// The message serializes to {"key1": ..., "key2": ...} instead of {"fieldName": {"key1": ..., ...}}.
func (g *Generator) generateRootMapUnwrapMarshalJSON(gf *protogen.GeneratedFile, rootUnwrap *RootUnwrapMessage) {
	msgName := rootUnwrap.Message.GoIdent.GoName
	fieldName := rootUnwrap.UnwrapField.GoName

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method performs root-level unwrap, serializing the message as just the map value.")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()

	// Check if we have combined unwrap (root map + value unwrap)
	switch {
	case rootUnwrap.ValueUnwrap != nil:
		// Combined unwrap: root map with value that also has unwrap
		g.generateRootMapWithValueUnwrapMarshal(gf, rootUnwrap, fieldName)
	case rootUnwrap.ValueMessage != nil:
		// Root map with message values (no value unwrap)
		g.generateRootMapMessageValueMarshal(gf, rootUnwrap, fieldName)
	default:
		// Root map with scalar values
		gf.P("return json.Marshal(x.", fieldName, ")")
	}

	gf.P("}")
	gf.P()
}

// generateRootMapWithValueUnwrapMarshal handles the case where root map value also has unwrap.
// Example: {"AAPL": [...], "GOOG": [...]} where each value is an unwrapped array.
func (g *Generator) generateRootMapWithValueUnwrapMarshal(
	gf *protogen.GeneratedFile,
	rootUnwrap *RootUnwrapMessage,
	fieldName string,
) {
	unwrapFieldName := rootUnwrap.ValueUnwrap.Field.GoName
	isMessageType := rootUnwrap.ValueUnwrap.ElementType != nil

	gf.P("out := make(map[string]json.RawMessage)")
	gf.P("for k, wrapper := range x.", fieldName, " {")
	gf.P("if wrapper != nil {")

	if isMessageType {
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
		gf.P("arrayData, err := json.Marshal(wrapper.Get", unwrapFieldName, "())")
	}

	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("out[k] = arrayData")
	gf.P("}")
	gf.P("}")
	gf.P("return json.Marshal(out)")
}

// generateRootMapMessageValueMarshal handles root map with message values (no value unwrap).
func (g *Generator) generateRootMapMessageValueMarshal(
	gf *protogen.GeneratedFile,
	_ *RootUnwrapMessage,
	fieldName string,
) {
	gf.P("out := make(map[string]json.RawMessage)")
	gf.P("for k, v := range x.", fieldName, " {")
	gf.P("if v != nil {")
	gf.P("data, err := protojson.Marshal(v)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("out[k] = data")
	gf.P("}")
	gf.P("}")
	gf.P("return json.Marshal(out)")
}

// generateRootMapUnwrapUnmarshalJSON generates UnmarshalJSON for root-level map unwrap.
func (g *Generator) generateRootMapUnwrapUnmarshalJSON(gf *protogen.GeneratedFile, rootUnwrap *RootUnwrapMessage) {
	msgName := rootUnwrap.Message.GoIdent.GoName
	fieldName := rootUnwrap.UnwrapField.GoName

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method performs root-level unwrap, deserializing from just the map value.")
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")

	// Check if we have combined unwrap (root map + value unwrap)
	switch {
	case rootUnwrap.ValueUnwrap != nil:
		g.generateRootMapWithValueUnwrapUnmarshal(gf, rootUnwrap, fieldName)
	case rootUnwrap.ValueMessage != nil:
		g.generateRootMapMessageValueUnmarshal(gf, rootUnwrap, fieldName)
	default:
		// Root map with scalar values
		gf.P("return json.Unmarshal(data, &x.", fieldName, ")")
	}

	gf.P("}")
	gf.P()
}

// generateRootMapWithValueUnwrapUnmarshal handles unmarshaling combined unwrap.
func (g *Generator) generateRootMapWithValueUnwrapUnmarshal(
	gf *protogen.GeneratedFile,
	rootUnwrap *RootUnwrapMessage,
	fieldName string,
) {
	valueTypeIdent := rootUnwrap.ValueMessage.GoIdent
	unwrapFieldName := rootUnwrap.ValueUnwrap.Field.GoName
	var elementTypeIdent *protogen.GoIdent
	if rootUnwrap.ValueUnwrap.ElementType != nil {
		ident := rootUnwrap.ValueUnwrap.ElementType.GoIdent
		elementTypeIdent = &ident
	}

	gf.P("var mapRaw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &mapRaw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("x.", fieldName, " = make(map[string]*", gf.QualifiedGoIdent(valueTypeIdent), ")")
	gf.P("for k, arrayRaw := range mapRaw {")

	if elementTypeIdent != nil {
		gf.P("var itemsRaw []json.RawMessage")
		gf.P("if err := json.Unmarshal(arrayRaw, &itemsRaw); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("items := make([]*", gf.QualifiedGoIdent(*elementTypeIdent), ", 0, len(itemsRaw))")
		gf.P("for _, itemRaw := range itemsRaw {")
		gf.P("item := &", *elementTypeIdent, "{}")
		gf.P("if err := protojson.Unmarshal(itemRaw, item); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("items = append(items, item)")
		gf.P("}")
	} else {
		gf.P("var items []", getScalarTypeName(rootUnwrap.ValueUnwrap.Field))
		gf.P("if err := json.Unmarshal(arrayRaw, &items); err != nil {")
		gf.P("return err")
		gf.P("}")
	}

	gf.P("x.", fieldName, "[k] = &", valueTypeIdent, "{", unwrapFieldName, ": items}")
	gf.P("}")
	gf.P("return nil")
}

// generateRootMapMessageValueUnmarshal handles root map with message values (no value unwrap).
func (g *Generator) generateRootMapMessageValueUnmarshal(
	gf *protogen.GeneratedFile,
	rootUnwrap *RootUnwrapMessage,
	fieldName string,
) {
	valueTypeIdent := rootUnwrap.ValueMessage.GoIdent

	gf.P("var mapRaw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &mapRaw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("x.", fieldName, " = make(map[string]*", gf.QualifiedGoIdent(valueTypeIdent), ")")
	gf.P("for k, v := range mapRaw {")
	gf.P("item := &", valueTypeIdent, "{}")
	gf.P("if err := protojson.Unmarshal(v, item); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("x.", fieldName, "[k] = item")
	gf.P("}")
	gf.P("return nil")
}

// generateRootRepeatedUnwrapMarshalJSON generates MarshalJSON for root-level repeated unwrap.
// The message serializes to [...] instead of {"fieldName": [...]}.
func (g *Generator) generateRootRepeatedUnwrapMarshalJSON(gf *protogen.GeneratedFile, rootUnwrap *RootUnwrapMessage) {
	msgName := rootUnwrap.Message.GoIdent.GoName
	fieldName := rootUnwrap.UnwrapField.GoName

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("// This method performs root-level unwrap, serializing the message as just the array value.")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P()

	// Check if element is a message type
	if rootUnwrap.UnwrapField.Message != nil {
		elementTypeIdent := rootUnwrap.UnwrapField.Message.GoIdent
		gf.P("items := make([]json.RawMessage, 0, len(x.", fieldName, "))")
		gf.P("for _, item := range x.", fieldName, " {")
		gf.P("data, err := protojson.Marshal(item)")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P("items = append(items, data)")
		gf.P("}")
		gf.P("return json.Marshal(items)")
		gf.P()
		// Suppress unused variable warning
		_ = elementTypeIdent
	} else {
		// Scalar type - marshal directly
		gf.P("return json.Marshal(x.", fieldName, ")")
	}

	gf.P("}")
	gf.P()
}

// generateRootRepeatedUnwrapUnmarshalJSON generates UnmarshalJSON for root-level repeated unwrap.
func (g *Generator) generateRootRepeatedUnwrapUnmarshalJSON(gf *protogen.GeneratedFile, rootUnwrap *RootUnwrapMessage) {
	msgName := rootUnwrap.Message.GoIdent.GoName
	fieldName := rootUnwrap.UnwrapField.GoName

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("// This method performs root-level unwrap, deserializing from just the array value.")
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")

	// Check if element is a message type
	if rootUnwrap.UnwrapField.Message != nil {
		elementTypeIdent := rootUnwrap.UnwrapField.Message.GoIdent
		gf.P("var itemsRaw []json.RawMessage")
		gf.P("if err := json.Unmarshal(data, &itemsRaw); err != nil {")
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
		gf.P("return nil")
	} else {
		// Scalar type - unmarshal directly
		gf.P("return json.Unmarshal(data, &x.", fieldName, ")")
	}

	gf.P("}")
	gf.P()
}
