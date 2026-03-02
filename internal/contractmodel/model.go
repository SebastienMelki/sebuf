package contractmodel

import (
	"sort"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	StructFullName    protoreflect.FullName = "google.protobuf.Struct"
	TimestampFullName protoreflect.FullName = "google.protobuf.Timestamp"
)

type Kind int

const (
	KindScalar Kind = iota
	KindEnum
	KindMessage
	KindStruct
	KindTimestamp
	KindMap
)

type TypeRef struct {
	Kind     Kind
	Name     string
	MapKey   *TypeRef
	MapValue *TypeRef
}

type Field struct {
	Name     string
	Type     *TypeRef
	Repeated bool
}

type Enum struct {
	Name   string
	Values []string
}

type Message struct {
	Name   string
	Fields []*Field
}

type Method struct {
	Name         string
	InputType    string
	ResponseType string
}

type Service struct {
	Name    string
	Methods []*Method
}

type Package struct {
	Name     string
	Files    []*protogen.File
	Enums    []*Enum
	Messages []*Message
	Services []*Service
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
		pkg := &Package{Name: name, Files: filesForPackage}
		pkg.Enums = collectEnums(filesForPackage, table)
		pkg.Messages = collectMessages(filesForPackage, table)
		pkg.Services = collectServices(filesForPackage, table)
		packages = append(packages, pkg)
	}

	return packages
}

func buildSymbols(files []*protogen.File) *symbols {
	table := &symbols{
		messages: make(map[protoreflect.FullName]string),
		enums:    make(map[protoreflect.FullName]string),
	}
	for _, file := range files {
		for _, enum := range file.Enums {
			table.enums[enum.Desc.FullName()] = enum.GoIdent.GoName
		}
		for _, msg := range file.Messages {
			walkMessageSymbols(msg, "", table)
		}
	}
	return table
}

func walkMessageSymbols(msg *protogen.Message, prefix string, table *symbols) {
	name := prefix + msg.GoIdent.GoName
	if !msg.Desc.IsMapEntry() {
		table.messages[msg.Desc.FullName()] = name
	}
	for _, enum := range msg.Enums {
		table.enums[enum.Desc.FullName()] = name + enum.GoIdent.GoName
	}
	for _, nested := range msg.Messages {
		walkMessageSymbols(nested, name+"__", table)
	}
}

func collectEnums(files []*protogen.File, table *symbols) []*Enum {
	var result []*Enum
	for _, file := range files {
		for _, enum := range file.Enums {
			result = append(result, &Enum{Name: table.enums[enum.Desc.FullName()], Values: enumValues(enum)})
		}
		for _, msg := range file.Messages {
			collectNestedEnums(msg, table, &result)
		}
	}
	return result
}

func collectNestedEnums(msg *protogen.Message, table *symbols, out *[]*Enum) {
	for _, enum := range msg.Enums {
		*out = append(*out, &Enum{Name: table.enums[enum.Desc.FullName()], Values: enumValues(enum)})
	}
	for _, nested := range msg.Messages {
		collectNestedEnums(nested, table, out)
	}
}

func enumValues(enum *protogen.Enum) []string {
	values := make([]string, 0, len(enum.Values))
	for _, value := range enum.Values {
		values = append(values, string(value.Desc.Name()))
	}
	return values
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
		fields := make([]*Field, 0, len(msg.Fields))
		for _, field := range msg.Fields {
			fields = append(fields, &Field{
				Name:     string(field.Desc.Name()),
				Type:     resolveType(field, table),
				Repeated: field.Desc.IsList() && !field.Desc.IsMap(),
			})
		}
		*out = append(*out, &Message{Name: table.messages[msg.Desc.FullName()], Fields: fields})
	}
	for _, nested := range msg.Messages {
		collectMessage(nested, table, out)
	}
}

func collectServices(files []*protogen.File, table *symbols) []*Service {
	var result []*Service
	for _, file := range files {
		for _, service := range file.Services {
			methods := make([]*Method, 0, len(service.Methods))
			for _, method := range service.Methods {
				methods = append(methods, &Method{
					Name:         method.GoName,
					InputType:    resolveMessageName(method.Input, table),
					ResponseType: resolveMessageName(method.Output, table),
				})
			}
			result = append(result, &Service{Name: service.GoName, Methods: methods})
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
	return message.GoIdent.GoName
}

func resolveType(field *protogen.Field, table *symbols) *TypeRef {
	if field.Desc.IsMap() {
		keyField := field.Message.Fields[0]
		valueField := field.Message.Fields[1]
		return &TypeRef{
			Kind:     KindMap,
			MapKey:   scalarTypeRef(keyField.Desc.Kind()),
			MapValue: resolveMapValueType(valueField, table),
		}
	}

	//nolint:exhaustive // scalar kinds intentionally fall through to scalarTypeRef in the default case
	switch field.Desc.Kind() {
	case protoreflect.EnumKind:
		if name, ok := table.enums[field.Enum.Desc.FullName()]; ok {
			return &TypeRef{Kind: KindEnum, Name: name}
		}
		return &TypeRef{Kind: KindEnum, Name: field.Enum.GoIdent.GoName}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		fullName := field.Message.Desc.FullName()
		switch fullName {
		case StructFullName:
			return &TypeRef{Kind: KindStruct}
		case TimestampFullName:
			return &TypeRef{Kind: KindTimestamp}
		default:
			return &TypeRef{Kind: KindMessage, Name: resolveMessageName(field.Message, table)}
		}
	default:
		return scalarTypeRef(field.Desc.Kind())
	}
}

func resolveMapValueType(field *protogen.Field, table *symbols) *TypeRef {
	//nolint:exhaustive // scalar kinds intentionally fall through to scalarTypeRef in the default case
	switch field.Desc.Kind() {
	case protoreflect.EnumKind:
		if name, ok := table.enums[field.Enum.Desc.FullName()]; ok {
			return &TypeRef{Kind: KindEnum, Name: name}
		}
		return &TypeRef{Kind: KindEnum, Name: field.Enum.GoIdent.GoName}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		fullName := field.Message.Desc.FullName()
		switch fullName {
		case StructFullName:
			return &TypeRef{Kind: KindStruct}
		case TimestampFullName:
			return &TypeRef{Kind: KindTimestamp}
		default:
			return &TypeRef{Kind: KindMessage, Name: resolveMessageName(field.Message, table)}
		}
	default:
		return scalarTypeRef(field.Desc.Kind())
	}
}

func scalarTypeRef(kind protoreflect.Kind) *TypeRef {
	return &TypeRef{Kind: KindScalar, Name: strings.ToLower(kind.String())}
}
