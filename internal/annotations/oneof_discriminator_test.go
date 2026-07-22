package annotations

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/http"
)

// --- protogen harness -------------------------------------------------------
//
// ValidateOneofDiscriminator operates on real *protogen.Message / *protogen.Oneof
// values, which cannot be hand-mocked. These helpers compile a hand-built proto3
// FileDescriptorProto into a *protogen.Plugin so the validator can be exercised on
// genuine protogen inputs. The oneof_config is passed to the validator directly, so
// the descriptor needs no sebuf.http extension options.

const validateTestPkg = "test.validate.v1"

// validateJSONName mirrors protoc's default lowerCamelCase json_name derivation.
func validateJSONName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if i > 0 && len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// scalarField builds a singular proto3 string field descriptor.
func scalarField(name string, number int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(number),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		JsonName: proto.String(validateJSONName(name)),
	}
}

// oneofMsgField builds a singular message-typed field bound to the oneof at index 0.
func oneofMsgField(name string, number int32, msgName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:       proto.String(name),
		Number:     proto.Int32(number),
		Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:       descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName:   proto.String("." + validateTestPkg + "." + msgName),
		JsonName:   proto.String(validateJSONName(name)),
		OneofIndex: proto.Int32(0),
	}
}

// validateOneofFile builds a proto3 file with an Event message carrying a base
// field plus a two-variant oneof (text -> TextContent, image -> ImageContent).
func validateOneofFile() *descriptorpb.FileDescriptorProto {
	textContent := &descriptorpb.DescriptorProto{
		Name:  proto.String("TextContent"),
		Field: []*descriptorpb.FieldDescriptorProto{scalarField("body", 1)},
	}
	imageContent := &descriptorpb.DescriptorProto{
		Name:  proto.String("ImageContent"),
		Field: []*descriptorpb.FieldDescriptorProto{scalarField("url", 1)},
	}
	event := &descriptorpb.DescriptorProto{
		Name: proto.String("Event"),
		Field: []*descriptorpb.FieldDescriptorProto{
			scalarField("id", 1),
			oneofMsgField("text", 2, "TextContent"),
			oneofMsgField("image", 3, "ImageContent"),
		},
		OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: proto.String("content")}},
	}

	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("validate_oneof.proto"),
		Package: proto.String(validateTestPkg),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/annotations/validatev1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{textContent, imageContent, event},
	}
}

func buildValidatePlugin(t *testing.T, fd *descriptorpb.FileDescriptorProto) *protogen.Plugin {
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

func findValidateMessage(t *testing.T, plugin *protogen.Plugin, name string) *protogen.Message {
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

func findValidateOneof(t *testing.T, msg *protogen.Message, name string) *protogen.Oneof {
	t.Helper()
	for _, oneof := range msg.Oneofs {
		if string(oneof.Desc.Name()) == name {
			return oneof
		}
	}
	t.Fatalf("oneof %q not found on message %q", name, msg.Desc.Name())
	return nil
}

// TestValidateOneofDiscriminator_VariantCollision covers the non-flatten
// discriminator/variant-key collision guard: on the non-flatten path the
// discriminator and the set variant's key share the parent object, so a
// discriminator equal to a variant's JSON name must be rejected. The flatten path
// is exempt because it spreads the variant's child fields and never emits the
// variant key itself.
func TestValidateOneofDiscriminator_VariantCollision(t *testing.T) {
	plugin := buildValidatePlugin(t, validateOneofFile())
	msg := findValidateMessage(t, plugin, "Event")
	oneof := findValidateOneof(t, msg, "content")

	t.Run("non_flatten_discriminator_equals_variant_name_errors", func(t *testing.T) {
		config := &http.OneofConfig{Discriminator: "text", Flatten: false}
		err := ValidateOneofDiscriminator(msg, oneof, config)
		if err == nil {
			t.Fatal("expected error for discriminator colliding with variant JSON name, got nil")
		}
		for _, want := range []string{"content", "text", "variant"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q missing expected substring %q", err.Error(), want)
			}
		}
	})

	t.Run("non_flatten_safe_discriminator_no_error", func(t *testing.T) {
		config := &http.OneofConfig{Discriminator: "kind", Flatten: false}
		if err := ValidateOneofDiscriminator(msg, oneof, config); err != nil {
			t.Errorf("expected no error for safe discriminator, got: %v", err)
		}
	})

	t.Run("flatten_discriminator_equals_variant_name_allowed", func(t *testing.T) {
		config := &http.OneofConfig{Discriminator: "text", Flatten: true}
		if err := ValidateOneofDiscriminator(msg, oneof, config); err != nil {
			t.Errorf("expected no collision error on flatten path, got: %v", err)
		}
	})
}
