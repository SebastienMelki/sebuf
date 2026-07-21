package tscommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

func TestTSScalarType(t *testing.T) {
	tests := []struct {
		name string
		kind protoreflect.Kind
		want string
	}{
		{name: "StringKind", kind: protoreflect.StringKind, want: "string"},
		{name: "BoolKind", kind: protoreflect.BoolKind, want: "boolean"},
		{name: "Int32Kind", kind: protoreflect.Int32Kind, want: "number"},
		{name: "Sint32Kind", kind: protoreflect.Sint32Kind, want: "number"},
		{name: "Sfixed32Kind", kind: protoreflect.Sfixed32Kind, want: "number"},
		{name: "Uint32Kind", kind: protoreflect.Uint32Kind, want: "number"},
		{name: "Fixed32Kind", kind: protoreflect.Fixed32Kind, want: "number"},
		{name: "FloatKind", kind: protoreflect.FloatKind, want: "number"},
		{name: "DoubleKind", kind: protoreflect.DoubleKind, want: "number"},
		{name: "Int64Kind", kind: protoreflect.Int64Kind, want: "string"},
		{name: "Sint64Kind", kind: protoreflect.Sint64Kind, want: "string"},
		{name: "Sfixed64Kind", kind: protoreflect.Sfixed64Kind, want: "string"},
		{name: "Uint64Kind", kind: protoreflect.Uint64Kind, want: "string"},
		{name: "Fixed64Kind", kind: protoreflect.Fixed64Kind, want: "string"},
		{name: "BytesKind", kind: protoreflect.BytesKind, want: "string"},
		{name: "EnumKind", kind: protoreflect.EnumKind, want: "string"},
		{name: "MessageKind", kind: protoreflect.MessageKind, want: "unknown"},
		{name: "GroupKind", kind: protoreflect.GroupKind, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TSScalarType(tt.kind)
			if got != tt.want {
				t.Errorf("TSScalarType(%v) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestTSZeroCheck(t *testing.T) {
	tests := []struct {
		name      string
		fieldKind string
		want      string
	}{
		{name: "string", fieldKind: "string", want: ` !== ""`},
		{name: "bool", fieldKind: "bool", want: ""},
		{name: "int32", fieldKind: "int32", want: " !== 0"},
		{name: "sint32", fieldKind: "sint32", want: " !== 0"},
		{name: "sfixed32", fieldKind: "sfixed32", want: " !== 0"},
		{name: "uint32", fieldKind: "uint32", want: " !== 0"},
		{name: "fixed32", fieldKind: "fixed32", want: " !== 0"},
		{name: "float", fieldKind: "float", want: " !== 0"},
		{name: "double", fieldKind: "double", want: " !== 0"},
		{name: "int64", fieldKind: "int64", want: ` !== "0"`},
		{name: "sint64", fieldKind: "sint64", want: ` !== "0"`},
		{name: "sfixed64", fieldKind: "sfixed64", want: ` !== "0"`},
		{name: "uint64", fieldKind: "uint64", want: ` !== "0"`},
		{name: "fixed64", fieldKind: "fixed64", want: ` !== "0"`},
		{name: "enum", fieldKind: "enum", want: ""},
		{name: "unknown_kind", fieldKind: "unknown_kind", want: ` !== ""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TSZeroCheck(tt.fieldKind)
			if got != tt.want {
				t.Errorf("TSZeroCheck(%q) = %q, want %q", tt.fieldKind, got, tt.want)
			}
		})
	}
}

// TestNonFlattenedOneofBranch verifies the arm rendered for a single variant of a
// non-flattened annotated oneof: the discriminator with its value, the set
// variant's key carrying a non-optional payload, and `?: never` guards for every
// sibling variant key. nonFlattenedOneofBranch reads only info.Discriminator, so
// it can be exercised directly without a *protogen.Oneof.
func TestNonFlattenedOneofBranch(t *testing.T) {
	// Three message variants keyed by their JSON names; render the "text" arm.
	info := &annotations.OneofDiscriminatorInfo{Discriminator: "kind"}
	jsonNames := []string{"text", "image", "video"}

	got := nonFlattenedOneofBranch(info, "text", jsonNames, 0, "TextContent")
	want := `{ kind: "text"; text: TextContent; image?: never; video?: never }`
	if got != want {
		t.Errorf("nonFlattenedOneofBranch message variant:\n  got:  %s\n  want: %s", got, want)
	}

	// A middle variant guards both its siblings (before and after its index).
	gotMid := nonFlattenedOneofBranch(info, "image", jsonNames, 1, "ImageContent")
	wantMid := `{ kind: "image"; image: ImageContent; text?: never; video?: never }`
	if gotMid != wantMid {
		t.Errorf("nonFlattenedOneofBranch middle variant:\n  got:  %s\n  want: %s", gotMid, wantMid)
	}

	// A scalar variant: the discriminator value, the field name, and the scalar
	// TS type land correctly, with the sibling key guarded.
	scalarInfo := &annotations.OneofDiscriminatorInfo{Discriminator: "type"}
	gotScalar := nonFlattenedOneofBranch(scalarInfo, "count", []string{"count", "label"}, 0, "number")
	wantScalar := `{ type: "count"; count: number; label?: never }`
	if gotScalar != wantScalar {
		t.Errorf("nonFlattenedOneofBranch scalar variant:\n  got:  %s\n  want: %s", gotScalar, wantScalar)
	}
}

// readGoldenFile reads a golden file relative to the project root and returns its content.
func readGoldenFile(t *testing.T, projectRoot, relPath string) string {
	t.Helper()
	goldenPath := filepath.Join(projectRoot, relPath)
	content, readErr := os.ReadFile(goldenPath)
	if readErr != nil {
		t.Fatalf("Failed to read golden file %s: %v", relPath, readErr)
	}
	return string(content)
}

// TestTSEnumUnspecifiedValue_ViaGoldenOutput validates TSEnumUnspecifiedValue behavior
// through the golden file output rather than direct function calls.
// TSEnumUnspecifiedValue requires a real *protogen.Field with populated Enum and
// extension options, which cannot be easily mocked. Instead, we verify the generated
// output captures the correct behavior: custom enum_value annotations produce custom
// strings, while enums without annotations use the proto name.
func TestTSEnumUnspecifiedValue_ViaGoldenOutput(t *testing.T) {
	// Find project root from the test's working directory (internal/tscommon/)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")

	t.Run("custom_enum_value_Status", func(t *testing.T) {
		s := readGoldenFile(
			t, projectRoot,
			"internal/tsclientgen/testdata/golden/enum_encoding.ts",
		)

		// Status enum has custom enum_value annotations: "unknown", "active", "inactive"
		// TSEnumUnspecifiedValue should return "unknown" (custom) for first value
		expected := `export type Status = "unknown" | "active" | "inactive";`
		if !strings.Contains(s, expected) {
			t.Errorf("Expected Status type with custom enum_value annotations:\n  %s", expected)
		}
	})

	t.Run("default_enum_value_Priority", func(t *testing.T) {
		s := readGoldenFile(
			t, projectRoot,
			"internal/tsclientgen/testdata/golden/enum_encoding.ts",
		)

		// Priority enum has NO custom enum_value annotations
		// TSEnumUnspecifiedValue should return "PRIORITY_LOW" (proto name) for first value
		expected := `export type Priority = "PRIORITY_LOW" | "PRIORITY_MEDIUM" | "PRIORITY_HIGH";`
		if !strings.Contains(s, expected) {
			t.Errorf("Expected Priority type with proto names:\n  %s", expected)
		}
	})

	t.Run("custom_enum_value_Region_query_params", func(t *testing.T) {
		// The Region enum type now lives in the canonical type module...
		types := readGoldenFile(
			t, projectRoot,
			"internal/tsclientgen/testdata/golden/query_params.ts",
		)

		// Region enum has custom enum_value annotations
		expected := `export type Region = "unspecified" | "americas" | "europe" | "asia";`
		if !strings.Contains(types, expected) {
			t.Errorf("Expected Region type with custom enum_value annotations:\n  %s", expected)
		}

		// ...while the zero check lives in the service module.
		client := readGoldenFile(
			t, projectRoot,
			"internal/tsclientgen/testdata/golden/query_params_client.ts",
		)

		// TSZeroCheckForField for enum query params should use custom unspecified value
		if !strings.Contains(client, `req.region !== "unspecified"`) {
			t.Error(
				`Expected zero check to use custom enum_value "unspecified" for Region query param`,
			)
		}
	})
}

// --- protogen harness -------------------------------------------------------
//
// The presence-union emitter and interface routing operate on real
// *protogen.Message / *protogen.Oneof values, which cannot be hand-mocked.
// These helpers compile a hand-built proto3 FileDescriptorProto into a
// *protogen.Plugin so the emitters can be exercised on genuine protogen inputs
// without pulling in a proto compiler.

const testProtoPkg = "test.oneof.v1"

// msgFieldProto builds a singular proto3 field descriptor.
func msgFieldProto(
	name string,
	number int32,
	typ descriptorpb.FieldDescriptorProto_Type,
) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(number),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     typ.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
}

// msgRefFieldProto builds a singular proto3 message-typed field descriptor.
func msgRefFieldProto(name string, number int32, msgName string) *descriptorpb.FieldDescriptorProto {
	f := msgFieldProto(name, number, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	f.TypeName = proto.String("." + testProtoPkg + "." + msgName)
	return f
}

// jsonName mirrors protoc's default lowerCamelCase json_name derivation.
func jsonName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if i > 0 && len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// buildTestPlugin compiles the descriptor into a *protogen.Plugin.
func buildTestPlugin(t *testing.T, fd *descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{fd.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("protogen.Options{}.New: %v", err)
	}
	return plugin
}

// findMessage returns the top-level message with the given name.
func findMessage(t *testing.T, plugin *protogen.Plugin, name string) *protogen.Message {
	t.Helper()
	for _, file := range plugin.Files {
		for _, msg := range file.Messages {
			if string(msg.Desc.Name()) == name {
				return msg
			}
		}
	}
	t.Fatalf("message %q not found in plugin", name)
	return nil
}

// findOneof returns the named oneof on a message.
func findOneof(t *testing.T, msg *protogen.Message, name string) *protogen.Oneof {
	t.Helper()
	for _, oneof := range msg.Oneofs {
		if string(oneof.Desc.Name()) == name {
			return oneof
		}
	}
	t.Fatalf("oneof %q not found on message %q", name, msg.Desc.Name())
	return nil
}

// capturePrinter returns a Printer that accumulates emitted lines and the
// buffer holding them.
func capturePrinter() (Printer, *strings.Builder) {
	var sb strings.Builder
	p := func(format string, args ...interface{}) {
		fmt.Fprintf(&sb, format+"\n", args...)
	}
	return p, &sb
}

// oneofTestFile builds a proto3 file exercising the un-annotated oneof shapes
// the presence-union emitter targets.
func oneofTestFile() *descriptorpb.FileDescriptorProto {
	str := descriptorpb.FieldDescriptorProto_TYPE_STRING
	msg := func(
		name string,
		fields []*descriptorpb.FieldDescriptorProto,
		oneofs []*descriptorpb.OneofDescriptorProto,
	) *descriptorpb.DescriptorProto {
		return &descriptorpb.DescriptorProto{
			Name:      proto.String(name),
			Field:     fields,
			OneofDecl: oneofs,
		}
	}
	oneof := func(name string) *descriptorpb.OneofDescriptorProto {
		return &descriptorpb.OneofDescriptorProto{Name: proto.String(name)}
	}

	textContent := msg("TextContent", []*descriptorpb.FieldDescriptorProto{
		msgFieldProto("text", 1, str),
	}, nil)
	imageContent := msg("ImageContent", []*descriptorpb.FieldDescriptorProto{
		msgFieldProto("url", 1, str),
	}, nil)
	videoContent := msg("VideoContent", []*descriptorpb.FieldDescriptorProto{
		msgFieldProto("url", 1, str),
	}, nil)

	// Base field + two-message oneof.
	plainEvent := msg("PlainEvent",
		[]*descriptorpb.FieldDescriptorProto{
			msgFieldProto("id", 1, str),
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("text", 2, "TextContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("image", 3, "ImageContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
		},
		[]*descriptorpb.OneofDescriptorProto{oneof("content")},
	)

	// Every field belongs to a single oneof (no base field).
	allOneofEvent := msg("AllOneofEvent",
		[]*descriptorpb.FieldDescriptorProto{
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("text", 1, "TextContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("image", 2, "ImageContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
		},
		[]*descriptorpb.OneofDescriptorProto{oneof("body")},
	)

	// Three-variant message oneof.
	threeVariantEvent := msg("ThreeVariantEvent",
		[]*descriptorpb.FieldDescriptorProto{
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("text", 1, "TextContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("image", 2, "ImageContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("video", 3, "VideoContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
		},
		[]*descriptorpb.OneofDescriptorProto{oneof("pick")},
	)

	// Oneof mixing a scalar member (with a snake_case name) and a message member.
	scalarOneofEvent := msg("ScalarOneofEvent",
		[]*descriptorpb.FieldDescriptorProto{
			func() *descriptorpb.FieldDescriptorProto {
				f := msgFieldProto("raw_text", 1, str)
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
			func() *descriptorpb.FieldDescriptorProto {
				f := msgRefFieldProto("detail", 2, "TextContent")
				f.OneofIndex = proto.Int32(0)
				return f
			}(),
		},
		[]*descriptorpb.OneofDescriptorProto{oneof("value")},
	)

	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("oneof_harness.proto"),
		Package: proto.String(testProtoPkg),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/tscommon/testoneofv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			textContent, imageContent, videoContent,
			plainEvent, allOneofEvent, threeVariantEvent, scalarOneofEvent,
		},
	}
}

func TestSynthesizeOneofInfo(t *testing.T) {
	plugin := buildTestPlugin(t, oneofTestFile())

	t.Run("message_variants", func(t *testing.T) {
		msg := findMessage(t, plugin, "PlainEvent")
		info := synthesizeOneofInfo(findOneof(t, msg, "content"))

		if info.Discriminator != "" {
			t.Errorf("Discriminator = %q, want empty", info.Discriminator)
		}
		if info.Flatten {
			t.Error("Flatten = true, want false")
		}
		if len(info.Variants) != 2 {
			t.Fatalf("len(Variants) = %d, want 2", len(info.Variants))
		}
		wantJSON := []string{"text", "image"}
		for i, v := range info.Variants {
			if v.DiscriminatorVal != wantJSON[i] {
				t.Errorf("Variants[%d].DiscriminatorVal = %q, want %q", i, v.DiscriminatorVal, wantJSON[i])
			}
			if v.Field.Desc.JSONName() != wantJSON[i] {
				t.Errorf("Variants[%d] JSONName = %q, want %q", i, v.Field.Desc.JSONName(), wantJSON[i])
			}
			if !v.IsMessage {
				t.Errorf("Variants[%d].IsMessage = false, want true", i)
			}
		}
	})

	t.Run("scalar_variant_json_name_and_ismessage", func(t *testing.T) {
		msg := findMessage(t, plugin, "ScalarOneofEvent")
		info := synthesizeOneofInfo(findOneof(t, msg, "value"))

		if len(info.Variants) != 2 {
			t.Fatalf("len(Variants) = %d, want 2", len(info.Variants))
		}
		// raw_text is a scalar member; its JSON name is camelCased.
		if got := info.Variants[0].DiscriminatorVal; got != "rawText" {
			t.Errorf("Variants[0].DiscriminatorVal = %q, want %q", got, "rawText")
		}
		if info.Variants[0].IsMessage {
			t.Error("scalar variant IsMessage = true, want false")
		}
		// detail is a message member.
		if got := info.Variants[1].DiscriminatorVal; got != "detail" {
			t.Errorf("Variants[1].DiscriminatorVal = %q, want %q", got, "detail")
		}
		if !info.Variants[1].IsMessage {
			t.Error("message variant IsMessage = false, want true")
		}
	})
}

func TestGeneratePresenceOneofUnionType(t *testing.T) {
	plugin := buildTestPlugin(t, oneofTestFile())

	tests := []struct {
		name      string
		message   string
		oneof     string
		unionName string
		want      string
	}{
		{
			name:      "three_variant_message_oneof",
			message:   "ThreeVariantEvent",
			oneof:     "pick",
			unionName: "ThreeVariantEventPick",
			want: `export type ThreeVariantEventPick =
  | { text: TextContent; image?: never; video?: never }
  | { image: ImageContent; text?: never; video?: never }
  | { video: VideoContent; text?: never; image?: never }
  | { text?: never; image?: never; video?: never };

`,
		},
		{
			name:      "scalar_and_message_oneof",
			message:   "ScalarOneofEvent",
			oneof:     "value",
			unionName: "ScalarOneofEventValue",
			want: `export type ScalarOneofEventValue =
  | { rawText: string; detail?: never }
  | { detail: TextContent; rawText?: never }
  | { rawText?: never; detail?: never };

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := findMessage(t, plugin, tt.message)
			info := synthesizeOneofInfo(findOneof(t, msg, tt.oneof))
			p, sb := capturePrinter()
			generatePresenceOneofUnionType(nil, p, tt.unionName, info)
			if sb.String() != tt.want {
				t.Errorf("generatePresenceOneofUnionType output mismatch:\n got:\n%s\nwant:\n%s", sb.String(), tt.want)
			}
		})
	}
}

func TestGenerateInterface_PresenceOneof(t *testing.T) {
	plugin := buildTestPlugin(t, oneofTestFile())

	t.Run("base_field_emits_base_interface_and_intersection", func(t *testing.T) {
		msg := findMessage(t, plugin, "PlainEvent")
		p, sb := capturePrinter()
		GenerateInterface(p, msg)
		out := sb.String()

		want := `export type PlainEventContent =
  | { text: TextContent; image?: never }
  | { image: ImageContent; text?: never }
  | { text?: never; image?: never };

export interface PlainEventBase {
  id: string;
}

export type PlainEvent = PlainEventBase & PlainEventContent;

`
		if out != want {
			t.Errorf("GenerateInterface(PlainEvent) mismatch:\n got:\n%s\nwant:\n%s", out, want)
		}
	})

	t.Run("all_fields_in_oneof_skips_base_interface", func(t *testing.T) {
		msg := findMessage(t, plugin, "AllOneofEvent")
		p, sb := capturePrinter()
		GenerateInterface(p, msg)
		out := sb.String()

		// No XBase interface when every field belongs to the oneof.
		if strings.Contains(out, "AllOneofEventBase") {
			t.Errorf("expected no AllOneofEventBase interface, got:\n%s", out)
		}
		if strings.Contains(out, "export interface AllOneofEvent") {
			t.Errorf("expected no standalone AllOneofEvent interface, got:\n%s", out)
		}
		// The alias points directly at the union.
		if !strings.Contains(out, "export type AllOneofEvent = AllOneofEventBody;") {
			t.Errorf("expected direct union alias, got:\n%s", out)
		}
	})
}
