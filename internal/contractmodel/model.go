package contractmodel

import (
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

const (
	AnyFullName       protoreflect.FullName = "google.protobuf.Any"
	DurationFullName  protoreflect.FullName = "google.protobuf.Duration"
	EmptyFullName     protoreflect.FullName = "google.protobuf.Empty"
	FieldMaskFullName protoreflect.FullName = "google.protobuf.FieldMask"
	ListValueFullName protoreflect.FullName = "google.protobuf.ListValue"
	StructFullName    protoreflect.FullName = "google.protobuf.Struct"
	TimestampFullName protoreflect.FullName = "google.protobuf.Timestamp"
	ValueFullName     protoreflect.FullName = "google.protobuf.Value"
	DoubleValueName   protoreflect.FullName = "google.protobuf.DoubleValue"
	FloatValueName    protoreflect.FullName = "google.protobuf.FloatValue"
	Int64ValueName    protoreflect.FullName = "google.protobuf.Int64Value"
	UInt64ValueName   protoreflect.FullName = "google.protobuf.UInt64Value"
	Int32ValueName    protoreflect.FullName = "google.protobuf.Int32Value"
	UInt32ValueName   protoreflect.FullName = "google.protobuf.UInt32Value"
	BoolValueName     protoreflect.FullName = "google.protobuf.BoolValue"
	StringValueName   protoreflect.FullName = "google.protobuf.StringValue"
	BytesValueName    protoreflect.FullName = "google.protobuf.BytesValue"
)

type Kind int

const (
	KindScalar Kind = iota
	KindEnum
	KindMessage
	KindWellKnown
	KindMap
)

type WellKnownType string

const (
	WellKnownAny        WellKnownType = "any"
	WellKnownDuration   WellKnownType = "duration"
	WellKnownEmpty      WellKnownType = "empty"
	WellKnownFieldMask  WellKnownType = "field_mask"
	WellKnownListValue  WellKnownType = "list_value"
	WellKnownStruct     WellKnownType = "struct"
	WellKnownTimestamp  WellKnownType = "timestamp"
	WellKnownValue      WellKnownType = "value"
	WellKnownDoubleWrap WellKnownType = "double_wrapper"
	WellKnownFloatWrap  WellKnownType = "float_wrapper"
	WellKnownInt64Wrap  WellKnownType = "int64_wrapper"
	WellKnownUInt64Wrap WellKnownType = "uint64_wrapper"
	WellKnownInt32Wrap  WellKnownType = "int32_wrapper"
	WellKnownUInt32Wrap WellKnownType = "uint32_wrapper"
	WellKnownBoolWrap   WellKnownType = "bool_wrapper"
	WellKnownStringWrap WellKnownType = "string_wrapper"
	WellKnownBytesWrap  WellKnownType = "bytes_wrapper"
)

type TypeRef struct {
	Kind      Kind
	Name      string
	WellKnown WellKnownType
	MapKey    *TypeRef
	MapValue  *TypeRef
}

type Query struct {
	Name     string
	Required bool
}

type FieldAnnotations struct {
	Query           *Query
	Unwrap          bool
	Int64Encoding   sebufhttp.Int64Encoding
	EnumEncoding    sebufhttp.EnumEncoding
	Nullable        bool
	EmptyBehavior   sebufhttp.EmptyBehavior
	TimestampFormat sebufhttp.TimestampFormat
	BytesEncoding   sebufhttp.BytesEncoding
	Flatten         bool
	FlattenPrefix   string
	OneofValue      string
}

type Field struct {
	Name           string
	JSONName       string
	Type           *TypeRef
	Repeated       bool
	Optional       bool
	HasPresence    bool
	IsMap          bool
	IsOneofVariant bool
	OneofName      string
	Annotations    FieldAnnotations
}

type EnumValue struct {
	Name      string
	JSONValue string
	Number    int32
}

type Enum struct {
	Name      string
	ProtoName string
	Values    []*EnumValue
}

type OneofVariant struct {
	FieldName          string
	DiscriminatorValue string
	Type               *TypeRef
	IsMessage          bool
}

type Oneof struct {
	Name          string
	Discriminator string
	Flatten       bool
	Variants      []*OneofVariant
}

type Unwrap struct {
	FieldName   string
	IsRoot      bool
	IsMapField  bool
	ElementType *TypeRef
}

type Message struct {
	Name      string
	ProtoName string
	Fields    []*Field
	Oneofs    []*Oneof
	Unwrap    *Unwrap
}

type Method struct {
	Name         string
	InputType    string
	ResponseType string
	HTTPMethod   string
	Path         string
	PathParams   []string
}

type Service struct {
	Name     string
	BasePath string
	Methods  []*Method
}

type Package struct {
	Name        string
	SourceFiles []string
	Enums       []*Enum
	Messages    []*Message
	Services    []*Service
}

type symbols struct {
	messages map[protoreflect.FullName]string
	enums    map[protoreflect.FullName]string
}

func Packages(files []*protogen.File) []*Package {
	byPackage := make(map[string][]*protogen.File)
	for _, file := range files {
		if !file.Generate {
			continue
		}
		pkg := string(file.Desc.Package())
		if pkg == "" {
			pkg = "default"
		}
		byPackage[pkg] = append(byPackage[pkg], file)
	}

	names := make([]string, 0, len(byPackage))
	for name := range byPackage {
		names = append(names, name)
	}
	sort.Strings(names)

	packages := make([]*Package, 0, len(names))
	for _, name := range names {
		filesForPackage := byPackage[name]
		sort.Slice(filesForPackage, func(i, j int) bool {
			return filesForPackage[i].Desc.Path() < filesForPackage[j].Desc.Path()
		})

		table := buildSymbols(filesForPackage)
		pkg := &Package{
			Name:        name,
			SourceFiles: sourceFiles(filesForPackage),
		}
		pkg.Enums = collectEnums(filesForPackage, table)
		pkg.Messages = collectMessages(filesForPackage, table)
		pkg.Services = collectServices(filesForPackage, table)
		packages = append(packages, pkg)
	}

	return packages
}

func sourceFiles(files []*protogen.File) []string {
	result := make([]string, 0, len(files))
	for _, file := range files {
		result = append(result, file.Desc.Path())
	}
	return result
}

func buildSymbols(files []*protogen.File) *symbols {
	table := &symbols{
		messages: make(map[protoreflect.FullName]string),
		enums:    make(map[protoreflect.FullName]string),
	}
	for _, file := range files {
		for _, enum := range file.Enums {
			table.enums[enum.Desc.FullName()] = string(enum.Desc.Name())
		}
		for _, msg := range file.Messages {
			walkMessageSymbols(msg, nil, table)
		}
	}
	return table
}

func walkMessageSymbols(msg *protogen.Message, parents []string, table *symbols) {
	name := append(append([]string{}, parents...), string(msg.Desc.Name()))
	symbol := strings.Join(name, "")
	if !msg.Desc.IsMapEntry() {
		table.messages[msg.Desc.FullName()] = symbol
	}
	for _, enum := range msg.Enums {
		table.enums[enum.Desc.FullName()] = symbol + string(enum.Desc.Name())
	}
	for _, nested := range msg.Messages {
		walkMessageSymbols(nested, name, table)
	}
}

func collectEnums(files []*protogen.File, table *symbols) []*Enum {
	var result []*Enum
	for _, file := range files {
		for _, enum := range file.Enums {
			result = append(result, buildEnum(enum, table))
		}
		for _, msg := range file.Messages {
			collectNestedEnums(msg, table, &result)
		}
	}
	return result
}

func collectNestedEnums(msg *protogen.Message, table *symbols, out *[]*Enum) {
	for _, enum := range msg.Enums {
		*out = append(*out, buildEnum(enum, table))
	}
	for _, nested := range msg.Messages {
		collectNestedEnums(nested, table, out)
	}
}

func buildEnum(enum *protogen.Enum, table *symbols) *Enum {
	values := make([]*EnumValue, 0, len(enum.Values))
	for _, value := range enum.Values {
		jsonValue := annotations.GetEnumValueMapping(value)
		if jsonValue == "" {
			jsonValue = string(value.Desc.Name())
		}
		values = append(values, &EnumValue{
			Name:      string(value.Desc.Name()),
			JSONValue: jsonValue,
			Number:    int32(value.Desc.Number()),
		})
	}
	return &Enum{
		Name:      table.enums[enum.Desc.FullName()],
		ProtoName: string(enum.Desc.Name()),
		Values:    values,
	}
}

func collectMessages(files []*protogen.File, table *symbols) []*Message {
	var result []*Message
	for _, file := range files {
		for _, msg := range file.Messages {
			collectMessage(msg, table, &result)
		}
	}
	return result
}

func collectMessage(msg *protogen.Message, table *symbols, out *[]*Message) {
	if !msg.Desc.IsMapEntry() {
		queryByField := make(map[string]annotations.QueryParam)
		for _, param := range annotations.GetQueryParams(msg) {
			queryByField[param.FieldName] = param
		}

		fields := make([]*Field, 0, len(msg.Fields))
		for _, field := range msg.Fields {
			fields = append(fields, &Field{
				Name:           string(field.Desc.Name()),
				JSONName:       field.Desc.JSONName(),
				Type:           resolveType(field, table),
				Repeated:       field.Desc.IsList() && !field.Desc.IsMap(),
				Optional:       field.Desc.HasOptionalKeyword(),
				HasPresence:    field.Desc.HasPresence(),
				IsMap:          field.Desc.IsMap(),
				IsOneofVariant: field.Oneof != nil && !field.Oneof.Desc.IsSynthetic(),
				OneofName:      oneofName(field),
				Annotations:    fieldAnnotations(field, queryByField[string(field.Desc.Name())]),
			})
		}

		message := &Message{
			Name:      table.messages[msg.Desc.FullName()],
			ProtoName: string(msg.Desc.Name()),
			Fields:    fields,
			Oneofs:    collectOneofs(msg, table),
			Unwrap:    collectUnwrap(msg, table),
		}
		*out = append(*out, message)
	}
	for _, nested := range msg.Messages {
		collectMessage(nested, table, out)
	}
}

func oneofName(field *protogen.Field) string {
	if field.Oneof == nil || field.Oneof.Desc.IsSynthetic() {
		return ""
	}
	return string(field.Oneof.Desc.Name())
}

func fieldAnnotations(field *protogen.Field, query annotations.QueryParam) FieldAnnotations {
	result := FieldAnnotations{
		Unwrap:          annotations.HasUnwrapAnnotation(field),
		Int64Encoding:   annotations.GetInt64Encoding(field),
		EnumEncoding:    annotations.GetEnumEncoding(field),
		Nullable:        annotations.IsNullableField(field),
		EmptyBehavior:   annotations.GetEmptyBehavior(field),
		TimestampFormat: annotations.GetTimestampFormat(field),
		BytesEncoding:   annotations.GetBytesEncoding(field),
		Flatten:         annotations.IsFlattenField(field),
		FlattenPrefix:   annotations.GetFlattenPrefix(field),
		OneofValue:      annotations.GetOneofVariantValue(field),
	}
	if query.FieldName != "" {
		result.Query = &Query{Name: query.ParamName, Required: query.Required}
	}
	return result
}

func collectUnwrap(msg *protogen.Message, table *symbols) *Unwrap {
	info, err := annotations.GetUnwrapField(msg)
	if err != nil || info == nil {
		return nil
	}

	var elementType *TypeRef
	if info.ElementType != nil {
		elementType = resolveMessageType(info.ElementType, table)
	}

	return &Unwrap{
		FieldName:   string(info.Field.Desc.Name()),
		IsRoot:      info.IsRootUnwrap,
		IsMapField:  info.IsMapField,
		ElementType: elementType,
	}
}

func collectOneofs(msg *protogen.Message, table *symbols) []*Oneof {
	var result []*Oneof
	for _, oneof := range msg.Oneofs {
		if oneof.Desc.IsSynthetic() {
			continue
		}

		model := &Oneof{Name: string(oneof.Desc.Name())}
		if info := annotations.GetOneofDiscriminatorInfo(oneof); info != nil {
			model.Discriminator = info.Discriminator
			model.Flatten = info.Flatten
			for _, variant := range info.Variants {
				model.Variants = append(model.Variants, &OneofVariant{
					FieldName:          string(variant.Field.Desc.Name()),
					DiscriminatorValue: variant.DiscriminatorVal,
					Type:               resolveType(variant.Field, table),
					IsMessage:          variant.IsMessage,
				})
			}
		} else {
			for _, field := range oneof.Fields {
				model.Variants = append(model.Variants, &OneofVariant{
					FieldName:          string(field.Desc.Name()),
					DiscriminatorValue: string(field.Desc.Name()),
					Type:               resolveType(field, table),
					IsMessage:          field.Message != nil,
				})
			}
		}

		result = append(result, model)
	}
	return result
}

func collectServices(files []*protogen.File, table *symbols) []*Service {
	var result []*Service
	for _, file := range files {
		for _, service := range file.Services {
			basePath := annotations.GetServiceBasePath(service)
			methods := make([]*Method, 0, len(service.Methods))
			for _, method := range service.Methods {
				httpConfig := annotations.GetMethodHTTPConfig(method)
				httpMethod := "POST"
				fullPath := annotations.BuildHTTPPath(basePath, "")
				var pathParams []string
				if httpConfig != nil {
					httpMethod = httpConfig.Method
					fullPath = annotations.BuildHTTPPath(basePath, httpConfig.Path)
					pathParams = append(pathParams, httpConfig.PathParams...)
				}

				methods = append(methods, &Method{
					Name:         method.GoName,
					InputType:    resolveMessageName(method.Input, table),
					ResponseType: resolveMessageName(method.Output, table),
					HTTPMethod:   httpMethod,
					Path:         fullPath,
					PathParams:   pathParams,
				})
			}
			result = append(result, &Service{Name: service.GoName, BasePath: basePath, Methods: methods})
		}
	}
	return result
}

func resolveMessageName(message *protogen.Message, table *symbols) string {
	if message == nil {
		return ""
	}
	if name, ok := table.messages[message.Desc.FullName()]; ok {
		return name
	}
	return string(message.Desc.Name())
}

func resolveMessageType(message *protogen.Message, table *symbols) *TypeRef {
	return &TypeRef{Kind: KindMessage, Name: resolveMessageName(message, table)}
}

func resolveType(field *protogen.Field, table *symbols) *TypeRef {
	if field.Desc.IsMap() {
		keyField := field.Message.Fields[0]
		valueField := field.Message.Fields[1]
		return &TypeRef{
			Kind:     KindMap,
			MapKey:   scalarTypeRef(keyField.Desc.Kind()),
			MapValue: resolveNonMapType(valueField, table),
		}
	}
	return resolveNonMapType(field, table)
}

func resolveNonMapType(field *protogen.Field, table *symbols) *TypeRef {
	//nolint:exhaustive // scalar kinds intentionally fall through to scalarTypeRef in the default case
	switch field.Desc.Kind() {
	case protoreflect.EnumKind:
		if name, ok := table.enums[field.Enum.Desc.FullName()]; ok {
			return &TypeRef{Kind: KindEnum, Name: name}
		}
		return &TypeRef{Kind: KindEnum, Name: string(field.Enum.Desc.Name())}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if ref := wellKnownTypeRef(field.Message.Desc.FullName()); ref != nil {
			return ref
		}
		return resolveMessageType(field.Message, table)
	default:
		return scalarTypeRef(field.Desc.Kind())
	}
}

func wellKnownTypeRef(fullName protoreflect.FullName) *TypeRef {
	switch fullName {
	case AnyFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownAny), WellKnown: WellKnownAny}
	case DurationFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownDuration), WellKnown: WellKnownDuration}
	case EmptyFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownEmpty), WellKnown: WellKnownEmpty}
	case FieldMaskFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownFieldMask), WellKnown: WellKnownFieldMask}
	case ListValueFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownListValue), WellKnown: WellKnownListValue}
	case StructFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownStruct), WellKnown: WellKnownStruct}
	case TimestampFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownTimestamp), WellKnown: WellKnownTimestamp}
	case ValueFullName:
		return &TypeRef{Kind: KindWellKnown, Name: csharpFriendlyName(WellKnownValue), WellKnown: WellKnownValue}
	case DoubleValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "double", WellKnown: WellKnownDoubleWrap}
	case FloatValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "float", WellKnown: WellKnownFloatWrap}
	case Int64ValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "int64", WellKnown: WellKnownInt64Wrap}
	case UInt64ValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "uint64", WellKnown: WellKnownUInt64Wrap}
	case Int32ValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "int32", WellKnown: WellKnownInt32Wrap}
	case UInt32ValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "uint32", WellKnown: WellKnownUInt32Wrap}
	case BoolValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "bool", WellKnown: WellKnownBoolWrap}
	case StringValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "string", WellKnown: WellKnownStringWrap}
	case BytesValueName:
		return &TypeRef{Kind: KindWellKnown, Name: "bytes", WellKnown: WellKnownBytesWrap}
	default:
		return nil
	}
}

func csharpFriendlyName(kind WellKnownType) string {
	switch kind {
	case WellKnownAny:
		return "Any"
	case WellKnownDuration:
		return "Duration"
	case WellKnownEmpty:
		return "Empty"
	case WellKnownFieldMask:
		return "FieldMask"
	case WellKnownListValue:
		return "ListValue"
	case WellKnownStruct:
		return "Struct"
	case WellKnownTimestamp:
		return "Timestamp"
	case WellKnownValue:
		return "Value"
	default:
		return string(kind)
	}
}

func scalarTypeRef(kind protoreflect.Kind) *TypeRef {
	return &TypeRef{Kind: KindScalar, Name: strings.ToLower(kind.String())}
}
