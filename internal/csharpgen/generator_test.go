package csharpgen

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/contractmodel"
)

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "STATE_UNSPECIFIED", want: "StateUnspecified"},
		{input: "item_state", want: "ItemState"},
		{input: "already", want: "Already"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := pascalCase(tt.input); got != tt.want {
				t.Fatalf("pascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestJSONAttribute(t *testing.T) {
	newtonsoft := &Generator{opts: Options{JSONLib: "newtonsoft"}}
	if got := newtonsoft.jsonAttribute("owner_id"); got != `[JsonProperty("owner_id")]` {
		t.Fatalf("Newtonsoft jsonAttribute = %q", got)
	}

	systemText := &Generator{opts: Options{JSONLib: "system_text_json"}}
	if got := systemText.jsonAttribute("owner_id"); got != `[JsonPropertyName("owner_id")]` {
		t.Fatalf("System.Text.Json jsonAttribute = %q", got)
	}
}

func TestCSharpTypeMappings(t *testing.T) {
	enumType := &contractmodel.TypeRef{Kind: contractmodel.KindEnum, Name: "WidgetState"}
	if got := csharpType(
		&contractmodel.Field{Name: "state", Type: enumType, HasPresence: true},
	); got != "WidgetState?" {
		t.Fatalf("enum csharpType = %q, want %q", got, "WidgetState?")
	}

	mapType := &contractmodel.TypeRef{
		Kind: contractmodel.KindMap,
		MapKey: &contractmodel.TypeRef{
			Kind: contractmodel.KindScalar,
			Name: "string",
		},
		MapValue: &contractmodel.TypeRef{
			Kind: contractmodel.KindScalar,
			Name: "int32",
		},
	}
	if got := csharpType(&contractmodel.Field{Name: "scores", Type: mapType}); got != "Dictionary<string, int>" {
		t.Fatalf("map csharpType = %q, want %q", got, "Dictionary<string, int>")
	}

	int64String := &contractmodel.Field{
		Name: "version",
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "int64"},
	}
	if got := csharpType(int64String); got != "string" {
		t.Fatalf("default int64 csharpType = %q, want %q", got, "string")
	}

	int64Number := &contractmodel.Field{
		Name: "version",
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "int64"},
		Annotations: contractmodel.FieldAnnotations{
			Int64Encoding: sebufhttp.Int64Encoding_INT64_ENCODING_NUMBER,
		},
	}
	if got := csharpType(int64Number); got != "long" {
		t.Fatalf("number int64 csharpType = %q, want %q", got, "long")
	}

	timestampString := &contractmodel.Field{
		Name: "created_at",
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindWellKnown, WellKnown: contractmodel.WellKnownTimestamp},
	}
	if got := csharpType(timestampString); got != "string" {
		t.Fatalf("default timestamp csharpType = %q, want %q", got, "string")
	}

	timestampNumber := &contractmodel.Field{
		Name: "created_at",
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindWellKnown, WellKnown: contractmodel.WellKnownTimestamp},
		Annotations: contractmodel.FieldAnnotations{
			TimestampFormat: sebufhttp.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS,
		},
	}
	if got := csharpType(timestampNumber); got != "long" {
		t.Fatalf("unix millis timestamp csharpType = %q, want %q", got, "long")
	}

	wrapperType := &contractmodel.TypeRef{
		Kind:      contractmodel.KindWellKnown,
		Name:      "int32",
		WellKnown: contractmodel.WellKnownInt32Wrap,
	}
	if got := csharpType(&contractmodel.Field{Name: "count", Type: wrapperType}); got != "int?" {
		t.Fatalf("wrapper csharpType = %q, want %q", got, "int?")
	}

	repeatedMessage := &contractmodel.Field{
		Name:     "items",
		Repeated: true,
		Type: &contractmodel.TypeRef{
			Kind: contractmodel.KindMessage,
			Name: "WidgetDetails",
		},
	}
	if got := csharpType(repeatedMessage); got != "List<WidgetDetails>" {
		t.Fatalf("repeated message csharpType = %q, want %q", got, "List<WidgetDetails>")
	}

	nullableString := &contractmodel.Field{
		Name:        "display_name",
		Optional:    true,
		HasPresence: true,
		Type:        &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
		Annotations: contractmodel.FieldAnnotations{Nullable: true},
	}
	if got := csharpType(nullableString); got != "string?" {
		t.Fatalf("nullable string csharpType = %q, want %q", got, "string?")
	}

	emptyBehaviorNull := &contractmodel.Field{
		Name: "meta",
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindMessage, Name: "Metadata"},
		Annotations: contractmodel.FieldAnnotations{
			EmptyBehavior: sebufhttp.EmptyBehavior_EMPTY_BEHAVIOR_NULL,
		},
	}
	if got := csharpType(emptyBehaviorNull); got != "Metadata?" {
		t.Fatalf("empty_behavior null csharpType = %q, want %q", got, "Metadata?")
	}
}

func TestMessageProperties(t *testing.T) {
	gen := &Generator{opts: Options{JSONLib: "newtonsoft"}}
	profile := &contractmodel.Message{
		Name: "WidgetProfile",
		Fields: []*contractmodel.Field{
			{Name: "note", Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"}},
		},
	}
	circle := &contractmodel.Message{
		Name: "ShapeEnvelopeCircle",
		Fields: []*contractmodel.Field{
			{Name: "radius", Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "double"}},
		},
	}
	rectangle := &contractmodel.Message{
		Name: "ShapeEnvelopeRectangle",
		Fields: []*contractmodel.Field{
			{Name: "width", Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "double"}},
			{Name: "height", Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "double"}},
		},
	}
	index := map[string]*contractmodel.Message{
		profile.Name:   profile,
		circle.Name:    circle,
		rectangle.Name: rectangle,
	}

	message := &contractmodel.Message{
		Name: "Widget",
		Fields: []*contractmodel.Field{
			{Name: "id", Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"}},
			{
				Name: "profile",
				Type: &contractmodel.TypeRef{Kind: contractmodel.KindMessage, Name: "WidgetProfile"},
				Annotations: contractmodel.FieldAnnotations{
					Flatten:       true,
					FlattenPrefix: "meta_",
				},
			},
		},
		Oneofs: []*contractmodel.Oneof{
			{
				Name:          "shape",
				Discriminator: "kind",
				Flatten:       true,
				Variants: []*contractmodel.OneofVariant{
					{
						FieldName:          "circle",
						DiscriminatorValue: "circle_shape",
						Type: &contractmodel.TypeRef{
							Kind: contractmodel.KindMessage,
							Name: "ShapeEnvelopeCircle",
						},
						IsMessage: true,
					},
					{
						FieldName:          "rectangle",
						DiscriminatorValue: "rectangle",
						Type: &contractmodel.TypeRef{
							Kind: contractmodel.KindMessage,
							Name: "ShapeEnvelopeRectangle",
						},
						IsMessage: true,
					},
				},
			},
		},
	}

	properties := gen.messageProperties(message, index)
	got := make(map[string]string, len(properties))
	for _, property := range properties {
		got[property.jsonName] = property.typ
	}

	for jsonName, wantType := range map[string]string{
		"id":        "string",
		"meta_note": "string?",
		"kind":      "string?",
		"radius":    "double?",
		"width":     "double?",
		"height":    "double?",
	} {
		if got[jsonName] != wantType {
			t.Fatalf("property %q = %q, want %q (all: %#v)", jsonName, got[jsonName], wantType, got)
		}
	}
}

func TestRootUnwrapBaseType(t *testing.T) {
	message := &contractmodel.Message{
		Name: "TagList",
		Fields: []*contractmodel.Field{
			{
				Name:     "values",
				Repeated: true,
				Type:     &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
			},
		},
		Unwrap: &contractmodel.Unwrap{
			FieldName: "values",
			IsRoot:    true,
		},
	}

	if !isRootUnwrapMessage(message) {
		t.Fatalf("expected root unwrap message")
	}
	if got := rootUnwrapBaseType(message); got != "List<string>" {
		t.Fatalf("rootUnwrapBaseType() = %q, want %q", got, "List<string>")
	}
}

func TestGeneratePackage(t *testing.T) {
	plugin := newCSharpTestPlugin(t)
	gen := New(plugin, Options{Namespace: "Test.Contracts", JSONLib: "newtonsoft"})

	pkg := &contractmodel.Package{
		Name: "test.contracts.v1",
		Enums: []*contractmodel.Enum{
			{
				Name: "WidgetState",
				Values: []*contractmodel.EnumValue{
					{Name: "STATE_UNSPECIFIED", JSONValue: "STATE_UNSPECIFIED", Number: 0},
					{Name: "STATE_READY", JSONValue: "ready", Number: 1},
				},
			},
		},
		Messages: []*contractmodel.Message{
			{
				Name: "WidgetProfile",
				Fields: []*contractmodel.Field{
					{
						Name: "note",
						Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
					},
				},
			},
			{
				Name: "Widget",
				Fields: []*contractmodel.Field{
					{
						Name: "id",
						Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
					},
					{
						Name:        "display_name",
						Optional:    true,
						HasPresence: true,
						Type:        &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
						Annotations: contractmodel.FieldAnnotations{
							Nullable: true,
						},
					},
					{
						Name:        "state",
						HasPresence: true,
						Type:        &contractmodel.TypeRef{Kind: contractmodel.KindEnum, Name: "WidgetState"},
						Annotations: contractmodel.FieldAnnotations{
							EnumEncoding: sebufhttp.EnumEncoding_ENUM_ENCODING_STRING,
						},
					},
					{
						Name: "meta",
						Type: &contractmodel.TypeRef{
							Kind:      contractmodel.KindWellKnown,
							WellKnown: contractmodel.WellKnownStruct,
						},
						Repeated: false,
					},
					{
						Name: "profile",
						Type: &contractmodel.TypeRef{Kind: contractmodel.KindMessage, Name: "WidgetProfile"},
						Annotations: contractmodel.FieldAnnotations{
							Flatten:       true,
							FlattenPrefix: "meta_",
						},
					},
				},
				Oneofs: []*contractmodel.Oneof{
					{
						Name:          "shape",
						Discriminator: "kind",
						Flatten:       true,
						Variants: []*contractmodel.OneofVariant{
							{
								FieldName:          "circle",
								DiscriminatorValue: "circle_shape",
								Type: &contractmodel.TypeRef{
									Kind: contractmodel.KindMessage,
									Name: "ShapeEnvelopeCircle",
								},
								IsMessage: true,
							},
						},
					},
				},
			},
			{
				Name: "ShapeEnvelopeCircle",
				Fields: []*contractmodel.Field{
					{
						Name: "radius",
						Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "double"},
					},
				},
			},
			{
				Name: "TagList",
				Fields: []*contractmodel.Field{
					{
						Name:     "values",
						Repeated: true,
						Type:     &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
					},
				},
				Unwrap: &contractmodel.Unwrap{
					FieldName: "values",
					IsRoot:    true,
				},
			},
		},
		Services: []*contractmodel.Service{
			{
				Name:     "WidgetService",
				BasePath: "/api/v1",
				Methods: []*contractmodel.Method{
					{
						Name:         "GetWidget",
						HTTPMethod:   "GET",
						Path:         "/api/v1/widgets/{id}",
						InputType:    "GetWidgetRequest",
						ResponseType: "Widget",
					},
				},
			},
		},
	}

	if err := gen.generatePackage(pkg); err != nil {
		t.Fatalf("generatePackage() error = %v", err)
	}

	output := generatedCSharpContent(t, plugin, "test/contracts/v1/Contracts.g.cs")
	for _, want := range []string{
		"public enum WidgetState",
		`[EnumMember(Value = "ready")]`,
		"StateUnspecified = 0",
		`[JsonConverter(typeof(StringEnumConverter))]`,
		"public WidgetState? State { get; set; }",
		"public string? DisplayName { get; set; }",
		`[JsonProperty("meta")]`,
		"public Dictionary<string, object> Meta { get; set; }",
		`[JsonProperty("meta_note")]`,
		"public string? MetaNote { get; set; }",
		`[JsonProperty("kind")]`,
		"public string? Kind { get; set; }",
		`[JsonProperty("radius")]`,
		"public double? Radius { get; set; }",
		"public sealed class TagList : List<string>",
		"public static class WidgetService",
		`public const string Path = "/api/v1/widgets/{id}";`,
		`public const string RequestType = "GetWidgetRequest";`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("generated output missing %q:\n%s", want, output)
		}
	}
}

func newCSharpTestPlugin(t *testing.T) *protogen.Plugin {
	t.Helper()
	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"placeholder.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("placeholder.proto"),
				Package: proto.String("test.contracts.v1"),
				Syntax:  proto.String("proto3"),
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/testdata/csharp;csharptest"),
				},
			},
		},
	}

	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}
	return plugin
}

func generatedCSharpContent(t *testing.T, plugin *protogen.Plugin, filename string) string {
	t.Helper()
	resp := plugin.Response()
	for _, file := range resp.GetFile() {
		if file.GetName() == filename {
			return file.GetContent()
		}
	}
	t.Fatalf("generated file %q not found", filename)
	return ""
}
