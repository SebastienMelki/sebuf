package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// generateJSONMappingUnmarshalJSON emits the inverse composed UnmarshalJSONSebuf/UnmarshalJSON
// pair for a mapped message.
func (g *Generator) generateJSONMappingUnmarshalJSON(gf *protogen.GeneratedFile, ctx *JSONMappingContext) {
	msgName := ctx.Message.GoIdent.GoName

	gf.P("// UnmarshalJSONSebuf implements sebufUnmarshaler for ", msgName, ".")
	gf.P("// This method composes inverse sebuf JSON mapping annotations and nested message delegation.")
	gf.P("func (x *", msgName, ") UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error {")
	gf.P("var raw map[string]json.RawMessage")
	if ctx.RootUnwrap != nil {
		g.generateJSONMappingRootUnwrapUnmarshal(gf, ctx.RootUnwrap)
	} else {
		gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
		gf.P("return err")
		gf.P("}")
	}
	gf.P()

	for _, transform := range ctx.FieldTransforms {
		g.generateJSONMappingFieldUnmarshal(gf, ctx, transform)
	}

	g.generateJSONMappingNestedUnmarshalDelegation(gf, ctx)

	gf.P("modified, err := json.Marshal(raw)")
	gf.P("if err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("return opts.Unmarshal(modified, x)")
	gf.P("}")
	gf.P()

	gf.P("// UnmarshalJSON implements json.Unmarshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") UnmarshalJSON(data []byte) error {")
	gf.P("return x.UnmarshalJSONSebuf(data, protojson.UnmarshalOptions{})")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingRootUnwrapUnmarshal(
	gf *protogen.GeneratedFile,
	rootUnwrap *RootUnwrapMessage,
) {
	jsonName := rootUnwrap.UnwrapField.Desc.JSONName()

	gf.P("// Invert root-level unwrap before other transforms.")
	gf.P("raw = make(map[string]json.RawMessage, 1)")
	gf.P(`raw["`, jsonName, `"] = data`)
}

func (g *Generator) generateJSONMappingFieldUnmarshal(
	gf *protogen.GeneratedFile,
	ctx *JSONMappingContext,
	transform *JSONMappingFieldTransform,
) {
	field := transform.Field
	if field == nil {
		return
	}

	switch transform.Kind {
	case TransformInt64Number:
		g.generateInt64FieldUnmarshal(gf, field)
	case TransformEnumValue:
		if info := customEnumFieldInfo(field, ctx.Message.GoIdent.GoImportPath); info != nil {
			g.generateEnumFieldUnmarshal(gf, info)
		}
	case TransformBytesEncoding:
		g.generateBytesFieldUnmarshal(gf, &BytesEncodingFieldInfo{
			Field:    field,
			Encoding: annotations.GetBytesEncoding(field),
		})
	case TransformTimestampFormat:
		g.generateTimestampFieldUnmarshal(gf, &TimestampFormatFieldInfo{
			Field:  field,
			Format: annotations.GetTimestampFormat(field),
		})
	case TransformNullable:
		g.generateJSONMappingNullableFieldUnmarshal(gf, field)
	case TransformEmptyBehavior:
		g.generateJSONMappingEmptyBehaviorFieldUnmarshal(gf, &EmptyBehaviorFieldInfo{
			Field:    field,
			Behavior: annotations.GetEmptyBehavior(field),
		})
	case TransformFlatten:
		g.generateJSONMappingFlattenFieldUnmarshal(gf, &FlattenFieldInfo{
			Field:  field,
			Prefix: annotations.GetFlattenPrefix(field),
		})
	case TransformOneofDiscriminator:
		if info := oneofDiscriminatorInfoForField(ctx.Message, field); info != nil {
			g.generateOneofUnmarshalVariants(gf, info)
			gf.P(`delete(raw, "`, info.Discriminator, `")`)
			gf.P()
		}
	case TransformMapValueUnwrap:
		g.generateJSONMappingMapValueUnwrapUnmarshal(gf, field)
	}
}

func (g *Generator) generateJSONMappingNullableFieldUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	gf.P("// Handle nullable field: ", field.Desc.Name())
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("if rawVal, ok := raw[k]; ok && string(rawVal) == \"null\" {")
	gf.P("delete(raw, k)")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingEmptyBehaviorFieldUnmarshal(
	gf *protogen.GeneratedFile,
	fieldInfo *EmptyBehaviorFieldInfo,
) {
	if fieldInfo.Behavior != http.EmptyBehavior_EMPTY_BEHAVIOR_NULL {
		return
	}

	gf.P("// Handle empty_behavior=NULL for field: ", fieldInfo.Field.Desc.Name())
	gf.P("for _, k := range []string{", enumFieldJSONKeys(fieldInfo.Field), "} {")
	gf.P("if rawVal, ok := raw[k]; ok && string(rawVal) == \"null\" {")
	gf.P("raw[k] = []byte(\"{}\")")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingFlattenFieldUnmarshal(gf *protogen.GeneratedFile, info *FlattenFieldInfo) {
	field := info.Field
	if field.Message == nil {
		return
	}

	jsonName := field.Desc.JSONName()
	prefix := info.Prefix

	gf.P("// Reconstruct flattened child object for field: ", field.Desc.Name())
	gf.P("{")
	gf.P("childRaw := make(map[string]json.RawMessage)")
	for _, childField := range field.Message.Fields {
		childJSONName := childField.Desc.JSONName()
		childProtoName := string(childField.Desc.Name())
		gf.P(`if v, ok := raw["`, prefix+childJSONName, `"]; ok {`)
		gf.P(`childRaw["`, childJSONName, `"] = v`)
		gf.P(`delete(raw, "`, prefix+childJSONName, `")`)
		gf.P("}")
		if childProtoName != childJSONName {
			gf.P(`if v, ok := raw["`, prefix+childProtoName, `"]; ok {`)
			gf.P(`childRaw["`, childProtoName, `"] = v`)
			gf.P(`delete(raw, "`, prefix+childProtoName, `")`)
			gf.P("}")
		}
	}
	gf.P("if len(childRaw) > 0 {")
	gf.P("childData, childErr := json.Marshal(childRaw)")
	gf.P("if childErr != nil {")
	gf.P("return childErr")
	gf.P("}")
	gf.P(`raw["`, jsonName, `"] = childData`)
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingMapValueUnwrapUnmarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	valueMsg := getMapValueMessage(field)
	unwrapInfo := unwrapInfoForMessage(valueMsg, nil)
	if valueMsg == nil || unwrapInfo == nil {
		return
	}

	g.generateJSONMappingUnwrapMapFieldUnmarshal(gf, field, &UnwrapMapField{
		Field:        field,
		ValueMessage: valueMsg,
		UnwrapField:  unwrapInfo,
	})
}

func (g *Generator) generateJSONMappingUnwrapMapFieldUnmarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	unwrapMapField *UnwrapMapField,
) {
	unwrapJSONName := unwrapMapField.UnwrapField.Field.Desc.JSONName()
	elementType := unwrapMapField.UnwrapField.ElementType

	gf.P("// Re-wrap map values for unwrap field: ", field.Desc.Name())
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("rawField, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")
	gf.P("var mapRaw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(rawField, &mapRaw); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("for mapKey, arrayRaw := range mapRaw {")
	if elementType != nil && messageNeedsJSONMapping(elementType) {
		childIdent := gf.QualifiedGoIdent(elementType.GoIdent)
		gf.P("var itemsRaw []json.RawMessage")
		gf.P("if err := json.Unmarshal(arrayRaw, &itemsRaw); err != nil {")
		gf.P("return err")
		gf.P("}")
		gf.P("protoItems := make([]json.RawMessage, len(itemsRaw))")
		gf.P("for i, itemRaw := range itemsRaw {")
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
		gf.P("rewrittenArray, marshalErr := json.Marshal(protoItems)")
		gf.P("if marshalErr != nil {")
		gf.P("return marshalErr")
		gf.P("}")
		gf.P("arrayRaw = rewrittenArray")
	}
	gf.P("wrapperRaw := map[string]json.RawMessage{")
	gf.P(`"`, unwrapJSONName, `": arrayRaw,`)
	gf.P("}")
	gf.P("wrapperData, marshalErr := json.Marshal(wrapperRaw)")
	gf.P("if marshalErr != nil {")
	gf.P("return marshalErr")
	gf.P("}")
	gf.P("mapRaw[mapKey] = wrapperData")
	gf.P("}")
	gf.P("rewrapped, marshalErr := json.Marshal(mapRaw)")
	gf.P("if marshalErr != nil {")
	gf.P("return marshalErr")
	gf.P("}")
	gf.P("raw[k] = rewrapped")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingNestedUnmarshalDelegation(
	gf *protogen.GeneratedFile,
	ctx *JSONMappingContext,
) {
	mapValueUnwrapFields := make(map[*protogen.Field]bool)
	for _, transform := range ctx.FieldTransforms {
		if transform.Kind == TransformMapValueUnwrap {
			mapValueUnwrapFields[transform.Field] = true
		}
	}

	for _, field := range ctx.NestedDelegationFields {
		if field.Desc.Kind() == protoreflect.BytesKind || mapValueUnwrapFields[field] {
			continue
		}
		switch {
		case field.Desc.IsMap():
			g.generateJSONMappingMapNestedUnmarshalDelegation(gf, field)
		case field.Desc.IsList():
			g.generateJSONMappingRepeatedNestedUnmarshalDelegation(gf, field)
		default:
			g.generateJSONMappingSingularNestedUnmarshalDelegation(gf, field)
		}
	}
}

func (g *Generator) generateJSONMappingSingularNestedUnmarshalDelegation(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
) {
	childIdent := gf.QualifiedGoIdent(field.Message.GoIdent)

	gf.P("// Delegate nested JSON unmarshal for field: ", field.Desc.Name())
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("rawVal, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")
	gf.P("inner := &", childIdent, "{}")
	emitJSONMappingUnmarshalChild(gf, "inner", "rawVal")
	gf.P("innerJSON, marshalErr := protojson.Marshal(inner)")
	gf.P("if marshalErr != nil {")
	gf.P("return marshalErr")
	gf.P("}")
	gf.P("raw[k] = innerJSON")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingRepeatedNestedUnmarshalDelegation(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
) {
	childIdent := gf.QualifiedGoIdent(field.Message.GoIdent)

	gf.P("// Delegate nested JSON unmarshal for repeated field: ", field.Desc.Name())
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("rawVal, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")
	gf.P("var rawItems []json.RawMessage")
	gf.P("if err := json.Unmarshal(rawVal, &rawItems); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("protoItems := make([]json.RawMessage, len(rawItems))")
	gf.P("for i, itemRaw := range rawItems {")
	gf.P("inner := &", childIdent, "{}")
	emitJSONMappingUnmarshalChild(gf, "inner", "itemRaw")
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
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingMapNestedUnmarshalDelegation(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
) {
	child := mapMessageValueChild(field)
	if child == nil {
		return
	}
	childIdent := gf.QualifiedGoIdent(child.GoIdent)

	gf.P("// Delegate nested JSON unmarshal for map field: ", field.Desc.Name())
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("rawVal, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")
	gf.P("var rawItems map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(rawVal, &rawItems); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("protoItems := make(map[string]json.RawMessage, len(rawItems))")
	gf.P("for itemKey, itemRaw := range rawItems {")
	gf.P("inner := &", childIdent, "{}")
	emitJSONMappingUnmarshalChild(gf, "inner", "itemRaw")
	gf.P("itemJSON, marshalErr := protojson.Marshal(inner)")
	gf.P("if marshalErr != nil {")
	gf.P("return marshalErr")
	gf.P("}")
	gf.P("protoItems[itemKey] = itemJSON")
	gf.P("}")
	gf.P("protoJSON, marshalErr := json.Marshal(protoItems)")
	gf.P("if marshalErr != nil {")
	gf.P("return marshalErr")
	gf.P("}")
	gf.P("raw[k] = protoJSON")
	gf.P("}")
	gf.P()
}

func emitJSONMappingUnmarshalChild(gf *protogen.GeneratedFile, valueExpr, dataExpr string) {
	gf.P("if u, ok := any(", valueExpr, ").(interface{ UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error }); ok {")
	gf.P("if err := u.UnmarshalJSONSebuf(", dataExpr, ", opts); err != nil {")
	gf.P("return err")
	gf.P("}")
	gf.P("} else if err := json.Unmarshal(", dataExpr, ", ", valueExpr, "); err != nil {")
	gf.P("return err")
	gf.P("}")
}
