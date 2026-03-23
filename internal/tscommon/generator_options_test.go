package tscommon

import (
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestCollectFileMessagesIncludesTopLevelMessagesAndEnums(t *testing.T) {
	file := newTSCommonTestFile(t)

	ms := CollectFileMessages(file)

	messages := ms.OrderedMessages()
	if len(messages) != 3 {
		t.Fatalf("CollectFileMessages returned %d messages, want 3", len(messages))
	}

	var messageNames []string
	for _, msg := range messages {
		messageNames = append(messageNames, string(msg.Desc.Name()))
	}

	if got, want := strings.Join(messageNames, ","), "Worker,DashboardState,SearchWorkersRequest"; got != want {
		t.Fatalf("CollectFileMessages names = %q, want %q", got, want)
	}

	enums := ms.OrderedEnums()
	if len(enums) != 1 {
		t.Fatalf("CollectFileMessages returned %d enums, want 1", len(enums))
	}
	if got, want := string(enums[0].Desc.Name()), "Role"; got != want {
		t.Fatalf("CollectFileMessages enum = %q, want %q", got, want)
	}
}

func TestGenerateInterfaceWithOptionsUsesConfiguredFieldNames(t *testing.T) {
	file := newTSCommonTestFile(t)
	msg := findMessageByName(t, file, "DashboardState")

	var jsonLines []string
	GenerateInterfaceWithOptions(
		testPrinter(&jsonLines),
		msg,
		GenerateOptions{},
	)
	jsonOutput := strings.Join(jsonLines, "\n")
	if !strings.Contains(jsonOutput, "ownerId: string;") {
		t.Fatalf("JSON naming output missing ownerId field:\n%s", jsonOutput)
	}
	if strings.Contains(jsonOutput, "owner_id: string;") {
		t.Fatalf("JSON naming output unexpectedly used proto field name:\n%s", jsonOutput)
	}

	var protoLines []string
	GenerateInterfaceWithOptions(
		testPrinter(&protoLines),
		msg,
		GenerateOptions{UseProtoFieldNames: true},
	)
	protoOutput := strings.Join(protoLines, "\n")
	if !strings.Contains(protoOutput, "owner_id: string;") {
		t.Fatalf("proto naming output missing owner_id field:\n%s", protoOutput)
	}
	if strings.Contains(protoOutput, "ownerId: string;") {
		t.Fatalf("proto naming output unexpectedly used JSON field name:\n%s", protoOutput)
	}
}

func TestGenerateFieldAndFlattenedFieldsUseConfiguredFieldNames(t *testing.T) {
	file := newTSCommonTestFile(t)
	worker := findMessageByName(t, file, "Worker")
	field := findFieldByName(t, worker, "display_name")

	var jsonFieldLines []string
	GenerateFieldDeclaration(testPrinter(&jsonFieldLines), field, GenerateOptions{})
	if got := strings.Join(jsonFieldLines, "\n"); !strings.Contains(got, "displayName: string;") {
		t.Fatalf("GenerateFieldDeclaration JSON output = %q, want displayName", got)
	}

	var protoFieldLines []string
	GenerateFieldDeclaration(
		testPrinter(&protoFieldLines),
		field,
		GenerateOptions{UseProtoFieldNames: true},
	)
	if got := strings.Join(protoFieldLines, "\n"); !strings.Contains(got, "display_name: string;") {
		t.Fatalf("GenerateFieldDeclaration proto output = %q, want display_name", got)
	}

	var jsonFlattenLines []string
	GenerateFlattenedFields(testPrinter(&jsonFlattenLines), worker, "member_", GenerateOptions{})
	jsonFlattened := strings.Join(jsonFlattenLines, "\n")
	if !strings.Contains(jsonFlattened, "member_displayName: string;") {
		t.Fatalf("GenerateFlattenedFields JSON output missing member_displayName:\n%s", jsonFlattened)
	}

	var protoFlattenLines []string
	GenerateFlattenedFields(
		testPrinter(&protoFlattenLines),
		worker,
		"member_",
		GenerateOptions{UseProtoFieldNames: true},
	)
	protoFlattened := strings.Join(protoFlattenLines, "\n")
	if !strings.Contains(protoFlattened, "member_display_name: string;") {
		t.Fatalf("GenerateFlattenedFields proto output missing member_display_name:\n%s", protoFlattened)
	}
}

func newTSCommonTestFile(t *testing.T) *protogen.File {
	t.Helper()

	fileDescriptor := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("tscommon_test.proto"),
		Package: proto.String("test.tscommon.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/testdata/tscommon;tscommonpb"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Role"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("ROLE_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("ROLE_ADMIN"), Number: proto.Int32(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Worker"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("worker_id"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:   proto.String("display_name"),
						Number: proto.Int32(2),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
			{
				Name: proto.String("DashboardState"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("owner_id"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     proto.String("workers"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".test.tscommon.v1.Worker"),
					},
					{
						Name:     proto.String("primary_role"),
						Number:   proto.Int32(3),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".test.tscommon.v1.Role"),
					},
				},
			},
			{
				Name: proto.String("SearchWorkersRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("worker_id"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}

	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{fileDescriptor.GetName()},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fileDescriptor},
	}

	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("protogen.Options.New() error = %v", err)
	}

	file := plugin.FilesByPath[fileDescriptor.GetName()]
	if file == nil {
		t.Fatalf("generated protogen file not found for %s", fileDescriptor.GetName())
	}
	return file
}

func findMessageByName(t *testing.T, file *protogen.File, name string) *protogen.Message {
	t.Helper()
	for _, msg := range file.Messages {
		if string(msg.Desc.Name()) == name {
			return msg
		}
	}
	t.Fatalf("message %q not found", name)
	return nil
}

func findFieldByName(t *testing.T, msg *protogen.Message, name string) *protogen.Field {
	t.Helper()
	for _, field := range msg.Fields {
		if string(field.Desc.Name()) == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return nil
}

func testPrinter(lines *[]string) Printer {
	return func(format string, args ...interface{}) {
		if len(args) == 0 {
			*lines = append(*lines, format)
			return
		}
		*lines = append(*lines, fmt.Sprintf(format, args...))
	}
}
