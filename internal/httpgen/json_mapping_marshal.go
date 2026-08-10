package httpgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// generateJSONMappingFile emits the composed JSON mapping implementation for every message in
// the file that needs direct transforms, nested delegation, or root unwrap handling.
func (g *Generator) generateJSONMappingFile(file *protogen.File) error {
	contexts, err := collectJSONMappingContexts(file)
	if err != nil {
		return fmt.Errorf("collecting JSON mapping contexts for %s: %w", file.Desc.Path(), err)
	}
	if len(contexts) == 0 {
		return nil
	}

	filename := file.GeneratedFilenamePrefix + "_json_mapping.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.writeJSONMappingImports(gf, contexts)

	for _, ctx := range contexts {
		g.generateJSONMappingMarshalJSON(gf, ctx)
		g.generateJSONMappingUnmarshalJSON(gf, ctx)
	}

	return nil
}

func (g *Generator) writeJSONMappingImports(gf *protogen.GeneratedFile, contexts []*JSONMappingContext) {
	needsBase64 := false
	needsHex := false
	needsFmt := false
	needsProto := false
	needsStrconv := false
	needsTime := false

	for _, ctx := range contexts {
		mapValueUnwrapFields := make(map[*protogen.Field]bool)
		for _, transform := range ctx.FieldTransforms {
			if transform.Kind == TransformMapValueUnwrap {
				mapValueUnwrapFields[transform.Field] = true
			}
		}
		for _, field := range ctx.NestedDelegationFields {
			if field.Desc.IsMap() && !mapValueUnwrapFields[field] {
				needsFmt = true
			}
		}
		for _, transform := range ctx.FieldTransforms {
			switch transform.Kind {
			case TransformInt64Number:
				needsStrconv = true
			case TransformBytesEncoding:
				// Unmarshal always re-encodes custom bytes to standard base64 for protojson.
				needsBase64 = true
				switch annotations.GetBytesEncoding(transform.Field) {
				case http.BytesEncoding_BYTES_ENCODING_HEX:
					needsHex = true
				default:
					// No extra import needed.
				}
			case TransformTimestampFormat:
				needsTime = true
			case TransformMapValueUnwrap:
				if valueMsg := getMapValueMessage(transform.Field); valueMsg != nil {
					unwrapInfo := unwrapInfoForMessage(valueMsg, nil)
					if unwrapInfo != nil && (unwrapInfo.ElementType != nil || transform.Field.Desc.MapKey().Kind() != protoreflect.StringKind) {
						needsFmt = true
					}
				}
			case TransformOneofDiscriminator:
				needsFmt = true
			case TransformEmptyBehavior:
				needsProto = true
			default:
				// No extra import needed.
			}
		}
	}

	gf.P("import (")
	if needsBase64 {
		gf.P(`"encoding/base64"`)
	}
	if needsHex {
		gf.P(`"encoding/hex"`)
	}
	gf.P(`"encoding/json"`)
	if needsFmt {
		gf.P(`"fmt"`)
	}
	if needsStrconv {
		gf.P(`"strconv"`)
	}
	if needsTime {
		gf.P(`"time"`)
	}
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	if needsProto {
		gf.P(`"google.golang.org/protobuf/proto"`)
	}
	gf.P(")")
	gf.P()
}

// generateJSONMappingMarshalJSON emits one composed MarshalJSONSebuf/MarshalJSON pair for a
// mapped message.
func (g *Generator) generateJSONMappingMarshalJSON(gf *protogen.GeneratedFile, ctx *JSONMappingContext) {
	msgName := ctx.Message.GoIdent.GoName

	gf.P("// MarshalJSONSebuf implements sebufMarshaler for ", msgName, ".")
	gf.P("// This method composes sebuf JSON mapping annotations and nested message delegation.")
	gf.P("func (x *", msgName, ") MarshalJSONSebuf(opts protojson.MarshalOptions) ([]byte, error) {")
	gf.P("if x == nil {")
	gf.P("return []byte(\"null\"), nil")
	gf.P("}")
	gf.P("data, err := opts.Marshal(x)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("var raw map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(data, &raw); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P()

	g.generateJSONMappingNestedDelegation(gf, ctx)

	for _, transform := range ctx.FieldTransforms {
		g.generateJSONMappingFieldMarshal(gf, ctx, transform)
	}

	if ctx.RootUnwrap != nil {
		g.generateJSONMappingRootUnwrapMarshal(gf, ctx.RootUnwrap)
	} else {
		gf.P("return json.Marshal(raw)")
	}
	gf.P("}")
	gf.P()

	gf.P("// MarshalJSON implements json.Marshaler for ", msgName, ".")
	gf.P("func (x *", msgName, ") MarshalJSON() ([]byte, error) {")
	gf.P("return x.MarshalJSONSebuf(protojson.MarshalOptions{})")
	gf.P("}")
	gf.P()

}

func (g *Generator) generateJSONMappingNestedDelegation(
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
			g.generateJSONMappingMapNestedDelegation(gf, field)
		case field.Desc.IsList():
			g.generateJSONMappingRepeatedNestedDelegation(gf, field)
		default:
			g.generateJSONMappingSingularNestedDelegation(gf, field)
		}
	}
}

func (g *Generator) generateJSONMappingSingularNestedDelegation(gf *protogen.GeneratedFile, field *protogen.Field) {
	fieldName := field.GoName

	gf.P("// Delegate nested JSON mapping for field: ", field.Desc.Name())
	gf.P("if x.", fieldName, " != nil {")
	emitInlineMarshalChild(gf, "x."+fieldName)
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	emitRawFieldSetForMarshalOptions(gf, field, "data")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingRepeatedNestedDelegation(gf *protogen.GeneratedFile, field *protogen.Field) {
	fieldName := field.GoName

	gf.P("// Delegate nested JSON mapping for repeated field: ", field.Desc.Name())
	gf.P("if len(x.", fieldName, ") > 0 {")
	gf.P("items := make([]json.RawMessage, 0, len(x.", fieldName, "))")
	gf.P("for _, item := range x.", fieldName, " {")
	emitInlineMarshalChild(gf, "item")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("items = append(items, data)")
	gf.P("}")
	gf.P("data, err := json.Marshal(items)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	emitRawFieldSetForMarshalOptions(gf, field, "data")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingMapNestedDelegation(gf *protogen.GeneratedFile, field *protogen.Field) {
	fieldName := field.GoName

	gf.P("// Delegate nested JSON mapping for map field: ", field.Desc.Name())
	gf.P("if len(x.", fieldName, ") > 0 {")
	gf.P("items := make(map[string]json.RawMessage, len(x.", fieldName, "))")
	gf.P("for k, item := range x.", fieldName, " {")
	gf.P("if item == nil {")
	gf.P("continue")
	gf.P("}")
	emitInlineMarshalChild(gf, "item")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("items[fmt.Sprint(k)] = data")
	gf.P("}")
	gf.P("data, err := json.Marshal(items)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	emitRawFieldSetForMarshalOptions(gf, field, "data")
	gf.P("}")
	gf.P()
}

func emitRawFieldSetForMarshalOptions(gf *protogen.GeneratedFile, field *protogen.Field, valueExpr string) {
	jsonName := field.Desc.JSONName()
	protoName := string(field.Desc.Name())
	if protoName == jsonName {
		gf.P(`raw["`, jsonName, `"] = `, valueExpr)
		return
	}

	gf.P("if opts.UseProtoNames {")
	gf.P(`raw["`, protoName, `"] = `, valueExpr)
	gf.P(`delete(raw, "`, jsonName, `")`)
	gf.P("} else {")
	gf.P(`raw["`, jsonName, `"] = `, valueExpr)
	gf.P(`delete(raw, "`, protoName, `")`)
	gf.P("}")
}

func (g *Generator) generateJSONMappingFieldMarshal(
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
		g.generateInt64FieldMarshal(gf, field)
	case TransformEnumValue:
		if info := customEnumFieldInfo(field, ctx.Message.GoIdent.GoImportPath); info != nil {
			g.generateEnumFieldMarshal(gf, info)
		}
	case TransformBytesEncoding:
		g.generateBytesFieldMarshal(gf, &BytesEncodingFieldInfo{
			Field:    field,
			Encoding: annotations.GetBytesEncoding(field),
		})
	case TransformTimestampFormat:
		g.generateTimestampFieldMarshal(gf, &TimestampFormatFieldInfo{
			Field:  field,
			Format: annotations.GetTimestampFormat(field),
		})
	case TransformNullable:
		g.generateNullableFieldMarshal(gf, field)
	case TransformEmptyBehavior:
		g.generateEmptyBehaviorFieldMarshal(gf, &EmptyBehaviorFieldInfo{
			Field:    field,
			Behavior: annotations.GetEmptyBehavior(field),
		})
	case TransformFlatten:
		g.generateFlattenFieldMarshal(gf, &FlattenFieldInfo{
			Field:  field,
			Prefix: annotations.GetFlattenPrefix(field),
		})
	case TransformOneofDiscriminator:
		if info := oneofDiscriminatorInfoForField(ctx.Message, field); info != nil {
			g.generateOneofMarshalVariants(gf, info)
		}
	case TransformMapValueUnwrap:
		g.generateJSONMappingMapValueUnwrapMarshal(gf, field)
	}
}

func (g *Generator) generateNullableFieldMarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	jsonName := field.Desc.JSONName()
	goName := field.GoName

	gf.P("// Handle nullable field: ", field.Desc.Name())
	gf.P("// proto3 optional + nullable=true: emit null when not set")
	gf.P("if x.", goName, " == nil {")
	gf.P(`raw["`, jsonName, `"] = []byte("null")`)
	gf.P("}")
	gf.P()
}

func oneofDiscriminatorInfoForField(
	msg *protogen.Message,
	field *protogen.Field,
) *annotations.OneofDiscriminatorInfo {
	for _, oneof := range msg.Oneofs {
		if firstOneofField(oneof) != field {
			continue
		}
		if info := annotations.GetOneofDiscriminatorInfo(oneof); info != nil {
			return info
		}
	}
	return nil
}

func (g *Generator) generateJSONMappingMapValueUnwrapMarshal(gf *protogen.GeneratedFile, field *protogen.Field) {
	valueMsg := getMapValueMessage(field)
	unwrapInfo := unwrapInfoForMessage(valueMsg, nil)
	if valueMsg == nil || unwrapInfo == nil {
		return
	}

	g.generateJSONMappingUnwrapMapMarshal(gf, field, &UnwrapMapField{
		Field:        field,
		ValueMessage: valueMsg,
		UnwrapField:  unwrapInfo,
	})
}

func (g *Generator) generateJSONMappingUnwrapMapMarshal(
	gf *protogen.GeneratedFile,
	field *protogen.Field,
	unwrapMapField *UnwrapMapField,
) {
	fieldName := field.GoName
	unwrapFieldName := unwrapMapField.UnwrapField.Field.GoName
	unwrapJSONName := unwrapMapField.UnwrapField.Field.Desc.JSONName()
	unwrapProtoName := string(unwrapMapField.UnwrapField.Field.Desc.Name())
	isMessageType := unwrapMapField.UnwrapField.ElementType != nil

	gf.P("// Handle unwrap map field: ", fieldName)
	gf.P("for _, k := range []string{", enumFieldJSONKeys(field), "} {")
	gf.P("rawField, ok := raw[k]")
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")
	gf.P("var mapData map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(rawField, &mapData); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("for mapKey, wrapperRaw := range mapData {")
	gf.P("if string(wrapperRaw) == \"null\" {")
	gf.P("continue")
	gf.P("}")
	gf.P("var wrapperObject map[string]json.RawMessage")
	gf.P("if err := json.Unmarshal(wrapperRaw, &wrapperObject); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P(`arrayData, ok := wrapperObject["`, unwrapJSONName, `"]`)
	if unwrapProtoName != unwrapJSONName {
		gf.P("if !ok {")
		gf.P(`arrayData, ok = wrapperObject["`, unwrapProtoName, `"]`)
		gf.P("}")
	}
	if isMessageType {
		gf.P("for goKey, wrapper := range x.", fieldName, " {")
		gf.P("if fmt.Sprint(goKey) != mapKey || wrapper == nil {")
		gf.P("continue")
		gf.P("}")
		gf.P("items := make([]json.RawMessage, 0, len(wrapper.Get", unwrapFieldName, "()))")
		gf.P("for _, item := range wrapper.Get", unwrapFieldName, "() {")
		emitInlineMarshalChild(gf, "item")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P("items = append(items, data)")
		gf.P("}")
		gf.P("rewrittenArray, err := json.Marshal(items)")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P("arrayData = rewrittenArray")
		gf.P("ok = true")
		gf.P("break")
		gf.P("}")
	} else {
		gf.P("if !ok {")
		gf.P("for goKey, wrapper := range x.", fieldName, " {")
		if field.Desc.MapKey().Kind() == protoreflect.StringKind {
			gf.P("if goKey != mapKey || wrapper == nil {")
		} else {
			gf.P("if fmt.Sprint(goKey) != mapKey || wrapper == nil {")
		}
		gf.P("continue")
		gf.P("}")
		gf.P("arrayData, err = json.Marshal(wrapper.Get", unwrapFieldName, "())")
		gf.P("if err != nil {")
		gf.P("return nil, err")
		gf.P("}")
		gf.P("ok = true")
		gf.P("break")
		gf.P("}")
		gf.P("}")
	}
	gf.P("if !ok {")
	gf.P("continue")
	gf.P("}")

	gf.P("mapData[mapKey] = arrayData")
	gf.P("}")
	gf.P("data, err := json.Marshal(mapData)")
	gf.P("if err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("raw[k] = data")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateJSONMappingRootUnwrapMarshal(
	gf *protogen.GeneratedFile,
	rootUnwrap *RootUnwrapMessage,
) {
	defaultJSON := "[]"
	if rootUnwrap.IsMap {
		defaultJSON = "{}"
	}

	gf.P("// Apply root-level unwrap last.")
	gf.P("for _, k := range []string{", enumFieldJSONKeys(rootUnwrap.UnwrapField), "} {")
	gf.P("if rootRaw, ok := raw[k]; ok {")
	gf.P("return rootRaw, nil")
	gf.P("}")
	gf.P("}")
	gf.P(`return []byte("`, defaultJSON, `"), nil`)
}
