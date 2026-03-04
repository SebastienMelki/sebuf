package pyclientgen

import (
	"reflect"
	"testing"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/contractmodel"
)

func TestInitFiles(t *testing.T) {
	t.Parallel()

	got := initFiles("test/contracts/v1")
	want := []string{
		"test/__init__.py",
		"test/contracts/__init__.py",
		"test/contracts/v1/__init__.py",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("initFiles() = %v, want %v", got, want)
	}
}

func TestOutputPackagePath(t *testing.T) {
	t.Parallel()

	gen := &Generator{opts: Options{Package: "custom.client.v1"}}
	if got, want := gen.outputPackagePath("ignored.package"), "custom/client/v1"; got != want {
		t.Fatalf("outputPackagePath() = %q, want %q", got, want)
	}
}

func TestPythonFieldTypeRequiredVsOptional(t *testing.T) {
	t.Parallel()

	required := &contractmodel.Field{
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
	}
	optional := &contractmodel.Field{
		Optional: true,
		Type:     &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
	}

	if got, want := pythonFieldType(required), "str"; got != want {
		t.Fatalf("pythonFieldType(required) = %q, want %q", got, want)
	}
	if got, want := pythonFieldType(optional), "str | None"; got != want {
		t.Fatalf("pythonFieldType(optional) = %q, want %q", got, want)
	}
}

func TestPythonScalarRespectsAnnotations(t *testing.T) {
	t.Parallel()

	int64String := &contractmodel.Field{
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "int64"},
	}
	int64Number := &contractmodel.Field{
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "int64"},
		Annotations: contractmodel.FieldAnnotations{
			Int64Encoding: sebufhttp.Int64Encoding_INT64_ENCODING_NUMBER,
		},
	}
	bytesHex := &contractmodel.Field{
		Type: &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "bytes"},
		Annotations: contractmodel.FieldAnnotations{
			BytesEncoding: sebufhttp.BytesEncoding_BYTES_ENCODING_HEX,
		},
	}

	if got, want := pythonScalar(int64String), "str"; got != want {
		t.Fatalf("pythonScalar(int64 string) = %q, want %q", got, want)
	}
	if got, want := pythonScalar(int64Number), "int"; got != want {
		t.Fatalf("pythonScalar(int64 number) = %q, want %q", got, want)
	}
	if got, want := serializeBaseExpr(bytesHex, "payload"), `_encode_bytes(payload, "hex")`; got != want {
		t.Fatalf("serializeBaseExpr(bytes hex) = %q, want %q", got, want)
	}
}

func TestPythonMethodName(t *testing.T) {
	t.Parallel()

	if got, want := pythonMethodName("GetWidget"), "get_widget"; got != want {
		t.Fatalf("pythonMethodName() = %q, want %q", got, want)
	}
}

func TestEffectiveFieldsFlattensMessageFields(t *testing.T) {
	t.Parallel()

	profile := &contractmodel.Message{
		Name: "WidgetProfile",
		Fields: []*contractmodel.Field{
			{
				Name:     "note",
				JSONName: "note",
				Type:     &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "string"},
			},
		},
	}
	widget := &contractmodel.Message{
		Name: "Widget",
		Fields: []*contractmodel.Field{
			{
				Name:     "profile",
				JSONName: "profile",
				Type:     &contractmodel.TypeRef{Kind: contractmodel.KindMessage, Name: "WidgetProfile"},
				Annotations: contractmodel.FieldAnnotations{
					Flatten:       true,
					FlattenPrefix: "meta_",
				},
			},
		},
	}

	got := effectiveFields(widget, map[string]*contractmodel.Message{"WidgetProfile": profile})
	if len(got) != 1 {
		t.Fatalf("effectiveFields() len = %d, want 1", len(got))
	}
	if got[0].Name != "meta_note" || got[0].JSONName != "meta_note" {
		t.Fatalf("effectiveFields() flatten = %+v, want meta_note field", got[0])
	}
}

func TestEffectiveFieldsFlattensOneofVariants(t *testing.T) {
	t.Parallel()

	circle := &contractmodel.Message{
		Name: "ShapeEnvelopeCircle",
		Fields: []*contractmodel.Field{
			{
				Name:     "radius",
				JSONName: "radius",
				Type:     &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "double"},
			},
		},
	}
	rectangle := &contractmodel.Message{
		Name: "ShapeEnvelopeRectangle",
		Fields: []*contractmodel.Field{
			{
				Name:     "width",
				JSONName: "width",
				Type:     &contractmodel.TypeRef{Kind: contractmodel.KindScalar, Name: "double"},
			},
		},
	}
	envelope := &contractmodel.Message{
		Name: "ShapeEnvelope",
		Fields: []*contractmodel.Field{
			{
				Name:           "circle",
				JSONName:       "circle",
				Type:           &contractmodel.TypeRef{Kind: contractmodel.KindMessage, Name: "ShapeEnvelopeCircle"},
				IsOneofVariant: true,
				OneofName:      "shape",
			},
			{
				Name:           "rectangle",
				JSONName:       "rectangle",
				Type:           &contractmodel.TypeRef{Kind: contractmodel.KindMessage, Name: "ShapeEnvelopeRectangle"},
				IsOneofVariant: true,
				OneofName:      "shape",
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

	got := effectiveFields(envelope, map[string]*contractmodel.Message{
		"ShapeEnvelopeCircle":    circle,
		"ShapeEnvelopeRectangle": rectangle,
	})

	names := []string{got[0].Name, got[1].Name, got[2].Name}
	want := []string{"kind", "radius", "width"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("effectiveFields() names = %v, want %v", names, want)
	}
}
