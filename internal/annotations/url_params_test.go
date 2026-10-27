package annotations

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/http"
)

func TestIsURLParamKindCompatible(t *testing.T) {
	compatible := []protoreflect.Kind{
		protoreflect.StringKind,
		protoreflect.BoolKind,
		protoreflect.EnumKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind,
		protoreflect.FloatKind, protoreflect.DoubleKind,
	}
	for _, kind := range compatible {
		if !IsURLParamKindCompatible(kind) {
			t.Errorf("kind %v should be URL-param compatible", kind)
		}
	}

	// These are the kinds behind issue #216: every generator used to fall through to
	// a string-shaped default for them.
	incompatible := []protoreflect.Kind{
		protoreflect.MessageKind,
		protoreflect.GroupKind,
		protoreflect.BytesKind,
	}
	for _, kind := range incompatible {
		if IsURLParamKindCompatible(kind) {
			t.Errorf("kind %v should NOT be URL-param compatible", kind)
		}
	}

	if IsURLParamKindCompatible(protoreflect.Kind(0)) {
		t.Error("unknown kind should not be URL-param compatible")
	}
}

func TestURLParamValidationError_Query(t *testing.T) {
	err := &URLParamValidationError{
		MessageName: "ListClientRestrictionsRequest",
		FieldName:   "client_id",
		ParamName:   "clientId",
		Location:    URLParamLocationQuery,
		TypeName:    "message (core.v1.UserClientID)",
	}

	msg := err.Error()
	for _, want := range []string{
		"field 'client_id'",
		"message 'ListClientRestrictionsRequest'",
		"(sebuf.http.query) as parameter 'clientId'",
		"unsupported type 'message (core.v1.UserClientID)'",
		URLParamScalarKinds,
		"POST/PUT/PATCH",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("query error missing %q, got: %s", want, msg)
		}
	}
}

func TestURLParamValidationError_Path(t *testing.T) {
	err := &URLParamValidationError{
		MessageName: "GetClientRequest",
		FieldName:   "client_id",
		ParamName:   "client_id",
		Location:    URLParamLocationPath,
		TypeName:    "message (core.v1.UserClientID)",
	}

	msg := err.Error()
	for _, want := range []string{
		"path variable '{client_id}'",
		"message 'GetClientRequest'",
		URLParamScalarKinds,
		"Change the field type or remove it from the path",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("path error missing %q, got: %s", want, msg)
		}
	}

	if strings.Contains(msg, "sebuf.http.query") {
		t.Errorf("path error should not mention the query annotation, got: %s", msg)
	}
}

// TestValidateFileURLParams drives the validator over synthesized descriptors, so it
// runs without protoc. The cross-generator behaviour is covered separately in
// internal/urlparamtest.
func TestValidateFileURLParams(t *testing.T) {
	tests := []struct {
		name    string
		file    *descriptorpb.FileDescriptorProto
		wantErr string // empty means "must be accepted"
	}{
		{
			name: "scalar query param is accepted",
			file: buildTestFile(
				queryField("client_id", descriptorpb.FieldDescriptorProto_TYPE_STRING, "", "clientId"), ""),
		},
		{
			name: "repeated scalar query param is accepted",
			file: buildTestFile(
				repeated(queryField("tags", descriptorpb.FieldDescriptorProto_TYPE_STRING, "", "tags")), ""),
		},
		{
			name: "scalar path param is accepted",
			file: buildTestFile(plainField("id", descriptorpb.FieldDescriptorProto_TYPE_STRING, ""), "/items/{id}"),
		},
		{
			name: "message query param is rejected",
			file: buildTestFile(
				queryField("client_id", descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
					".test.UserClientID", "clientId"),
				""),
			wantErr: "unsupported type 'message (test.UserClientID)'",
		},
		{
			name: "repeated message query param is rejected",
			file: buildTestFile(
				repeated(queryField("ids", descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
					".test.UserClientID", "ids")),
				""),
			wantErr: "field 'ids'",
		},
		{
			name: "bytes query param is rejected",
			file: buildTestFile(
				queryField("checksum", descriptorpb.FieldDescriptorProto_TYPE_BYTES, "", "checksum"), ""),
			wantErr: "unsupported type 'bytes'",
		},
		{
			name: "message path param is rejected",
			file: buildTestFile(
				plainField("client_id", descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".test.UserClientID"),
				"/clients/{client_id}"),
			wantErr: "path variable '{client_id}'",
		},
		{
			// Kind is `string`, so only the IsList check catches this.
			name: "repeated scalar path param is rejected",
			file: buildTestFile(
				repeated(plainField("ids", descriptorpb.FieldDescriptorProto_TYPE_STRING, "")),
				"/items/{ids}",
			),
			wantErr: "cannot be repeated",
		},
		{
			// Kind is reported ahead of cardinality: dropping `repeated` alone would
			// still leave this unbindable.
			name: "repeated message path param reports the kind, not the repetition",
			file: buildTestFile(
				repeated(plainField("ids", descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
					".test.UserClientID")),
				"/items/{ids}",
			),
			wantErr: "must be scalar types",
		},
		{
			name: "path variable with no matching field is left to httpgen",
			file: buildTestFile(
				plainField("id", descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				"/items/{missing}",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := loadTestFile(t, tt.file)

			err := ValidateFileURLParams(file)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected the proto to be accepted, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error to contain %q, got: %v", tt.wantErr, err)
			}
			// Every error is prefixed with the offending service and method.
			if !strings.Contains(err.Error(), "ItemService.GetItem:") {
				t.Errorf("expected error to name the service and method, got: %v", err)
			}
		})
	}
}

// TestValidateFileURLParams_MapField pins the error wording for map fields: they
// report MessageKind, but naming the synthetic "GetItemRequestFiltersEntry" type in
// the error would be noise, so URLParamTypeName collapses them to "map".
func TestValidateFileURLParams_MapField(t *testing.T) {
	fd := buildTestFile(
		repeated(queryField("filters", descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			".test.GetItemRequest.FiltersEntry", "filters")),
		"",
	)
	// Attach the synthetic map-entry type protoc would generate.
	fd.MessageType[1].NestedType = []*descriptorpb.DescriptorProto{{
		Name: proto.String("FiltersEntry"),
		Field: []*descriptorpb.FieldDescriptorProto{
			plainField("key", descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
			numbered(plainField("value", descriptorpb.FieldDescriptorProto_TYPE_STRING, ""), 2),
		},
		Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
	}}

	err := ValidateFileURLParams(loadTestFile(t, fd))
	if err == nil {
		t.Fatal("expected a map query param to be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported type 'map'") {
		t.Errorf("expected the error to name the type as 'map', got: %v", err)
	}
}

func TestFindFieldByProtoName(t *testing.T) {
	file := loadTestFile(t, buildTestFile(plainField("id", descriptorpb.FieldDescriptorProto_TYPE_STRING, ""), ""))
	request := file.Services[0].Methods[0].Input

	if got := FindFieldByProtoName(request, "id"); got == nil {
		t.Error("expected to find field 'id'")
	}
	if got := FindFieldByProtoName(request, "nope"); got != nil {
		t.Errorf("expected nil for a missing field, got %v", got.Desc.Name())
	}
}

// --- descriptor builders -----------------------------------------------------

func plainField(
	name string,
	kind descriptorpb.FieldDescriptorProto_Type,
	typeName string,
) *descriptorpb.FieldDescriptorProto {
	f := &descriptorpb.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(1),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:   kind.Enum(),
	}
	if typeName != "" {
		f.TypeName = proto.String(typeName)
	}
	return f
}

func queryField(
	name string,
	kind descriptorpb.FieldDescriptorProto_Type,
	typeName, paramName string,
) *descriptorpb.FieldDescriptorProto {
	f := plainField(name, kind, typeName)
	opts := &descriptorpb.FieldOptions{}
	proto.SetExtension(opts, http.E_Query, &http.QueryConfig{
		Name:     paramName,
		Required: true,
	})
	f.Options = opts
	return f
}

func repeated(f *descriptorpb.FieldDescriptorProto) *descriptorpb.FieldDescriptorProto {
	f.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	return f
}

func numbered(f *descriptorpb.FieldDescriptorProto, n int32) *descriptorpb.FieldDescriptorProto {
	f.Number = proto.Int32(n)
	return f
}

// buildTestFile assembles a single-service proto around one request field. An empty
// path means "no HTTP config annotation at all".
func buildTestFile(field *descriptorpb.FieldDescriptorProto, path string) *descriptorpb.FileDescriptorProto {
	method := &descriptorpb.MethodDescriptorProto{
		Name:       proto.String("GetItem"),
		InputType:  proto.String(".test.GetItemRequest"),
		OutputType: proto.String(".test.Item"),
	}
	if path != "" {
		opts := &descriptorpb.MethodOptions{}
		proto.SetExtension(opts, http.E_Config, &http.HttpConfig{
			Path:   path,
			Method: http.HttpMethod_HTTP_METHOD_GET,
		})
		method.Options = opts
	}

	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/annotations/testpb;testpb"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("UserClientID"),
				Field: []*descriptorpb.FieldDescriptorProto{
					plainField("value", descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				},
			},
			{Name: proto.String("GetItemRequest"), Field: []*descriptorpb.FieldDescriptorProto{field}},
			{Name: proto.String("Item")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{Name: proto.String("ItemService"), Method: []*descriptorpb.MethodDescriptorProto{method}},
		},
	}
}

func loadTestFile(t *testing.T, fd *descriptorpb.FileDescriptorProto) *protogen.File {
	t.Helper()

	plugin, err := protogen.Options{}.New(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{fd.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("protogen.New failed: %v", err)
	}

	return plugin.Files[0]
}
