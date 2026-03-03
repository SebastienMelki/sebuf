package contractmodel

import (
	"slices"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"google.golang.org/protobuf/types/pluginpb"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
)

func TestPackagesBuildsRichModel(t *testing.T) {
	plugin := newContractModelPlugin(t)

	pkgs := Packages(plugin.Files)
	if len(pkgs) != 1 {
		t.Fatalf("Packages() returned %d packages, want 1", len(pkgs))
	}

	pkg := pkgs[0]
	if got, want := pkg.Name, "test.contracts.v1"; got != want {
		t.Fatalf("Package.Name = %q, want %q", got, want)
	}
	if got, want := pkg.SourceFiles, []string{"widget.proto", "widget_service.proto"}; !slices.Equal(got, want) {
		t.Fatalf("Package.SourceFiles = %v, want %v", got, want)
	}

	widgetState := findEnum(t, pkg, "WidgetState")
	if widgetState.ProtoName != "State" {
		t.Fatalf("Enum.ProtoName = %q, want %q", widgetState.ProtoName, "State")
	}
	if got := widgetState.Values[1].JSONValue; got != "ready" {
		t.Fatalf("Enum JSON mapping = %q, want %q", got, "ready")
	}

	widgetDetails := findMessage(t, pkg, "WidgetDetails")
	if widgetDetails.ProtoName != "Details" {
		t.Fatalf("Nested message proto name = %q, want %q", widgetDetails.ProtoName, "Details")
	}

	widget := findMessage(t, pkg, "Widget")
	displayName := findField(t, widget, "display_name")
	if !displayName.Optional || !displayName.HasPresence {
		t.Fatalf("display_name field should preserve optional presence: %+v", displayName)
	}
	if !displayName.Annotations.Nullable {
		t.Fatalf("display_name field should carry nullable annotation")
	}

	ownerID := findField(t, widget, "owner_id")
	if ownerID.Annotations.Query == nil || ownerID.Annotations.Query.Name != "owner" || !ownerID.Annotations.Query.Required {
		t.Fatalf("owner_id query annotation = %+v, want owner/required", ownerID.Annotations.Query)
	}

	createdAt := findField(t, widget, "created_at")
	if got := createdAt.Type.WellKnown; got != WellKnownTimestamp {
		t.Fatalf("created_at WellKnown = %q, want %q", got, WellKnownTimestamp)
	}
	if got := createdAt.Annotations.TimestampFormat; got != sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS {
		t.Fatalf("created_at TimestampFormat = %v, want UNIX_MILLIS", got)
	}

	payload := findField(t, widget, "payload")
	if got := payload.Annotations.BytesEncoding; got != sebufhttp.BytesEncoding_BYTES_ENCODING_HEX {
		t.Fatalf("payload BytesEncoding = %v, want HEX", got)
	}

	version := findField(t, widget, "version")
	if got := version.Annotations.Int64Encoding; got != sebufhttp.Int64Encoding_INT64_ENCODING_NUMBER {
		t.Fatalf("version Int64Encoding = %v, want NUMBER", got)
	}

	state := findField(t, widget, "state")
	if got := state.Type.Name; got != "WidgetState" {
		t.Fatalf("state type = %q, want %q", got, "WidgetState")
	}
	if got := state.Annotations.EnumEncoding; got != sebufhttp.EnumEncoding_ENUM_ENCODING_NUMBER {
		t.Fatalf("state EnumEncoding = %v, want NUMBER", got)
	}

	profile := findField(t, widget, "profile")
	if !profile.Annotations.Flatten || profile.Annotations.FlattenPrefix != "meta_" {
		t.Fatalf("profile flatten annotations = %+v", profile.Annotations)
	}

	shapeHolder := findMessage(t, pkg, "ShapeHolder")
	if len(shapeHolder.Oneofs) != 1 {
		t.Fatalf("ShapeHolder.Oneofs = %d, want 1", len(shapeHolder.Oneofs))
	}
	shape := shapeHolder.Oneofs[0]
	if shape.Name != "shape" || shape.Discriminator != "kind" || !shape.Flatten {
		t.Fatalf("ShapeHolder oneof = %+v, want named discriminated flatten oneof", shape)
	}
	if len(shape.Variants) != 2 || shape.Variants[0].DiscriminatorValue != "circle_shape" {
		t.Fatalf("ShapeHolder variants = %+v", shape.Variants)
	}

	tags := findMessage(t, pkg, "Tags")
	if tags.Unwrap == nil || !tags.Unwrap.IsRoot || tags.Unwrap.FieldName != "items" {
		t.Fatalf("Tags unwrap = %+v, want root unwrap on items", tags.Unwrap)
	}

	service := findService(t, pkg, "WidgetService")
	if service.BasePath != "/api/v1" {
		t.Fatalf("Service.BasePath = %q, want %q", service.BasePath, "/api/v1")
	}
	getWidget := findMethod(t, service, "GetWidget")
	if getWidget.HTTPMethod != "GET" || getWidget.Path != "/api/v1/widgets/{id}" {
		t.Fatalf("GetWidget metadata = %+v", getWidget)
	}
	if got, want := getWidget.PathParams, []string{"id"}; !slices.Equal(got, want) {
		t.Fatalf("GetWidget.PathParams = %v, want %v", got, want)
	}
}

func TestPackagesResolveWellKnownTypesAndCrossFileMessages(t *testing.T) {
	plugin := newContractModelPlugin(t)
	pkg := Packages(plugin.Files)[0]

	holder := findMessage(t, pkg, "WellKnownHolder")
	cases := map[string]WellKnownType{
		"meta":      WellKnownStruct,
		"any_value": WellKnownAny,
		"ttl":       WellKnownDuration,
		"raw_value": WellKnownValue,
		"items":     WellKnownListValue,
		"mask":      WellKnownFieldMask,
		"label":     WellKnownStringWrap,
	}
	for fieldName, want := range cases {
		if got := findField(t, holder, fieldName).Type.WellKnown; got != want {
			t.Fatalf("%s WellKnown = %q, want %q", fieldName, got, want)
		}
	}

	reset := findMethod(t, findService(t, pkg, "WidgetService"), "ResetWidget")
	if reset.ResponseType != "Empty" {
		t.Fatalf("ResetWidget.ResponseType = %q, want %q", reset.ResponseType, "Empty")
	}

	getWidget := findMethod(t, findService(t, pkg, "WidgetService"), "GetWidget")
	if getWidget.ResponseType != "Widget" {
		t.Fatalf("cross-file response type = %q, want %q", getWidget.ResponseType, "Widget")
	}
}

func newContractModelPlugin(t *testing.T) *protogen.Plugin {
	t.Helper()

	widgetFile := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("widget.proto"),
		Package:    proto.String("test.contracts.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/protobuf/struct.proto", "google/protobuf/timestamp.proto", "proto/sebuf/http/annotations.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/testcontracts/widget;widgetpb"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			widgetDescriptor(t),
			{
				Name: proto.String("Tags"),
				Field: []*descriptorpb.FieldDescriptorProto{
					repeatedScalarField("items", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, withFieldOption(t, sebufhttp.E_Unwrap, true)),
				},
			},
		},
	}

	serviceFile := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("widget_service.proto"),
		Package: proto.String("test.contracts.v1"),
		Syntax:  proto.String("proto3"),
		Dependency: []string{
			"widget.proto",
			"google/protobuf/any.proto",
			"google/protobuf/duration.proto",
			"google/protobuf/empty.proto",
			"google/protobuf/field_mask.proto",
			"google/protobuf/struct.proto",
			"google/protobuf/wrappers.proto",
			"proto/sebuf/http/annotations.proto",
		},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/testcontracts/widgetservice;widgetservicepb"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			shapeHolderDescriptor(t),
			wellKnownHolderDescriptor(),
			{
				Name: proto.String("GetWidgetRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					scalarField("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name:    proto.String("WidgetService"),
				Options: withServiceOption(t, sebufhttp.E_ServiceConfig, &sebufhttp.ServiceConfig{BasePath: "/api/v1"}),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("GetWidget"),
						InputType:  proto.String(".test.contracts.v1.GetWidgetRequest"),
						OutputType: proto.String(".test.contracts.v1.Widget"),
						Options: withMethodOption(t, sebufhttp.E_Config, &sebufhttp.HttpConfig{
							Path:   "/widgets/{id}",
							Method: sebufhttp.HttpMethod_HTTP_METHOD_GET,
						}),
					},
					{
						Name:       proto.String("ResetWidget"),
						InputType:  proto.String(".test.contracts.v1.GetWidgetRequest"),
						OutputType: proto.String(".google.protobuf.Empty"),
						Options: withMethodOption(t, sebufhttp.E_Config, &sebufhttp.HttpConfig{
							Path:   "/widgets/{id}:reset",
							Method: sebufhttp.HttpMethod_HTTP_METHOD_POST,
						}),
					},
				},
			},
		},
	}

	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"widget.proto", "widget_service.proto"},
		ProtoFile: append(
			[]*descriptorpb.FileDescriptorProto{widgetFile, serviceFile},
			testDependencyProtos()...,
		),
	}

	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func widgetDescriptor(t *testing.T) *descriptorpb.DescriptorProto {
	t.Helper()
	return &descriptorpb.DescriptorProto{
		Name: proto.String("Widget"),
		Field: []*descriptorpb.FieldDescriptorProto{
			scalarField("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			scalarField("owner_id", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING,
				withFieldOption(t, sebufhttp.E_Query, &sebufhttp.QueryConfig{Name: "owner", Required: true})),
			optionalScalarField("display_name", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING,
				withFieldOption(t, sebufhttp.E_Nullable, true)),
			messageField("created_at", 4, ".google.protobuf.Timestamp",
				withFieldOption(t, sebufhttp.E_TimestampFormat, sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS)),
			scalarField("payload", 5, descriptorpb.FieldDescriptorProto_TYPE_BYTES,
				withFieldOption(t, sebufhttp.E_BytesEncoding, sebufhttp.BytesEncoding_BYTES_ENCODING_HEX)),
			scalarField("version", 6, descriptorpb.FieldDescriptorProto_TYPE_INT64,
				withFieldOption(t, sebufhttp.E_Int64Encoding, sebufhttp.Int64Encoding_INT64_ENCODING_NUMBER)),
			enumField("state", 7, ".test.contracts.v1.Widget.State",
				withFieldOption(t, sebufhttp.E_EnumEncoding, sebufhttp.EnumEncoding_ENUM_ENCODING_NUMBER)),
			messageField("details", 8, ".test.contracts.v1.Widget.Details"),
			messageField("profile", 9, ".test.contracts.v1.Widget.Profile",
				withFieldOption(t, sebufhttp.E_Flatten, true),
				withFieldOption(t, sebufhttp.E_FlattenPrefix, "meta_")),
		},
		NestedType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Details"),
				Field: []*descriptorpb.FieldDescriptorProto{
					scalarField("note", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				},
			},
			{
				Name: proto.String("Profile"),
				Field: []*descriptorpb.FieldDescriptorProto{
					scalarField("label", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("State"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("STATE_UNSPECIFIED"), Number: proto.Int32(0)},
					{
						Name:    proto.String("STATE_READY"),
						Number:  proto.Int32(1),
						Options: withEnumValueOption(t, sebufhttp.E_EnumValue, "ready"),
					},
				},
			},
		},
		OneofDecl: []*descriptorpb.OneofDescriptorProto{
			{Name: proto.String("_display_name")},
		},
	}
}

func shapeHolderDescriptor(t *testing.T) *descriptorpb.DescriptorProto {
	t.Helper()
	return &descriptorpb.DescriptorProto{
		Name: proto.String("ShapeHolder"),
		Field: []*descriptorpb.FieldDescriptorProto{
			messageFieldWithOneof("circle", 1, ".test.contracts.v1.ShapeHolder.Circle", 0,
				withFieldOption(t, sebufhttp.E_OneofValue, "circle_shape")),
			messageFieldWithOneof("rectangle", 2, ".test.contracts.v1.ShapeHolder.Rectangle", 0),
		},
		OneofDecl: []*descriptorpb.OneofDescriptorProto{
			{
				Name:    proto.String("shape"),
				Options: withOneofOption(t, sebufhttp.E_OneofConfig, &sebufhttp.OneofConfig{Discriminator: "kind", Flatten: true}),
			},
		},
		NestedType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Circle"),
				Field: []*descriptorpb.FieldDescriptorProto{
					scalarField("radius", 1, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
				},
			},
			{
				Name: proto.String("Rectangle"),
				Field: []*descriptorpb.FieldDescriptorProto{
					scalarField("width", 1, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
					scalarField("height", 2, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
				},
			},
		},
	}
}

func wellKnownHolderDescriptor() *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String("WellKnownHolder"),
		Field: []*descriptorpb.FieldDescriptorProto{
			messageField("meta", 1, ".google.protobuf.Struct"),
			messageField("any_value", 2, ".google.protobuf.Any"),
			messageField("ttl", 3, ".google.protobuf.Duration"),
			messageField("raw_value", 4, ".google.protobuf.Value"),
			messageField("items", 5, ".google.protobuf.ListValue"),
			messageField("mask", 6, ".google.protobuf.FieldMask"),
			messageField("label", 7, ".google.protobuf.StringValue"),
		},
	}
}

func scalarField(
	name string,
	number int32,
	kind descriptorpb.FieldDescriptorProto_Type,
	options ...*descriptorpb.FieldOptions,
) *descriptorpb.FieldDescriptorProto {
	field := &descriptorpb.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(number),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:   kind.Enum(),
	}
	if len(options) > 0 {
		field.Options = mergeFieldOptions(options...)
	}
	return field
}

func repeatedScalarField(
	name string,
	number int32,
	kind descriptorpb.FieldDescriptorProto_Type,
	options ...*descriptorpb.FieldOptions,
) *descriptorpb.FieldDescriptorProto {
	field := scalarField(name, number, kind, options...)
	field.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	return field
}

func messageField(name string, number int32, typeName string, options ...*descriptorpb.FieldOptions) *descriptorpb.FieldDescriptorProto {
	field := &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(number),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: proto.String(typeName),
	}
	if len(options) > 0 {
		field.Options = mergeFieldOptions(options...)
	}
	return field
}

func enumField(name string, number int32, typeName string, options ...*descriptorpb.FieldOptions) *descriptorpb.FieldDescriptorProto {
	field := &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(number),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
		TypeName: proto.String(typeName),
	}
	if len(options) > 0 {
		field.Options = mergeFieldOptions(options...)
	}
	return field
}

func optionalScalarField(
	name string,
	number int32,
	kind descriptorpb.FieldDescriptorProto_Type,
	options ...*descriptorpb.FieldOptions,
) *descriptorpb.FieldDescriptorProto {
	field := scalarField(name, number, kind, options...)
	field.OneofIndex = proto.Int32(0)
	field.Proto3Optional = proto.Bool(true)
	return field
}

func messageFieldWithOneof(
	name string,
	number int32,
	typeName string,
	oneofIndex int32,
	options ...*descriptorpb.FieldOptions,
) *descriptorpb.FieldDescriptorProto {
	field := messageField(name, number, typeName, options...)
	field.OneofIndex = proto.Int32(oneofIndex)
	return field
}

func mergeFieldOptions(options ...*descriptorpb.FieldOptions) *descriptorpb.FieldOptions {
	merged := &descriptorpb.FieldOptions{}
	for _, option := range options {
		proto.Merge(merged, option)
	}
	return merged
}

func withFieldOption(t *testing.T, ext protoreflect.ExtensionType, value any) *descriptorpb.FieldOptions {
	t.Helper()
	opts := &descriptorpb.FieldOptions{}
	proto.SetExtension(opts, ext, value)
	return opts
}

func withEnumValueOption(t *testing.T, ext protoreflect.ExtensionType, value any) *descriptorpb.EnumValueOptions {
	t.Helper()
	opts := &descriptorpb.EnumValueOptions{}
	proto.SetExtension(opts, ext, value)
	return opts
}

func withMethodOption(t *testing.T, ext protoreflect.ExtensionType, value any) *descriptorpb.MethodOptions {
	t.Helper()
	opts := &descriptorpb.MethodOptions{}
	proto.SetExtension(opts, ext, value)
	return opts
}

func withServiceOption(t *testing.T, ext protoreflect.ExtensionType, value any) *descriptorpb.ServiceOptions {
	t.Helper()
	opts := &descriptorpb.ServiceOptions{}
	proto.SetExtension(opts, ext, value)
	return opts
}

func withOneofOption(t *testing.T, ext protoreflect.ExtensionType, value any) *descriptorpb.OneofOptions {
	t.Helper()
	opts := &descriptorpb.OneofOptions{}
	proto.SetExtension(opts, ext, value)
	return opts
}

func testDependencyProtos() []*descriptorpb.FileDescriptorProto {
	return []*descriptorpb.FileDescriptorProto{
		protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto),
		protodesc.ToFileDescriptorProto(anypb.File_google_protobuf_any_proto),
		protodesc.ToFileDescriptorProto(durationpb.File_google_protobuf_duration_proto),
		protodesc.ToFileDescriptorProto(emptypb.File_google_protobuf_empty_proto),
		protodesc.ToFileDescriptorProto(fieldmaskpb.File_google_protobuf_field_mask_proto),
		protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
		protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto),
		protodesc.ToFileDescriptorProto(wrapperspb.File_google_protobuf_wrappers_proto),
		protodesc.ToFileDescriptorProto(sebufhttp.File_proto_sebuf_http_annotations_proto),
	}
}

func findEnum(t *testing.T, pkg *Package, name string) *Enum {
	t.Helper()
	for _, enum := range pkg.Enums {
		if enum.Name == name {
			return enum
		}
	}
	t.Fatalf("enum %q not found", name)
	return nil
}

func findMessage(t *testing.T, pkg *Package, name string) *Message {
	t.Helper()
	for _, message := range pkg.Messages {
		if message.Name == name {
			return message
		}
	}
	t.Fatalf("message %q not found", name)
	return nil
}

func findField(t *testing.T, msg *Message, name string) *Field {
	t.Helper()
	for _, field := range msg.Fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return nil
}

func findService(t *testing.T, pkg *Package, name string) *Service {
	t.Helper()
	for _, service := range pkg.Services {
		if service.Name == name {
			return service
		}
	}
	t.Fatalf("service %q not found", name)
	return nil
}

func findMethod(t *testing.T, service *Service, name string) *Method {
	t.Helper()
	for _, method := range service.Methods {
		if method.Name == name {
			return method
		}
	}
	t.Fatalf("method %q not found", name)
	return nil
}
