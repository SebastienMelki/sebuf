package tsclientgen

import (
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

func TestGenerateClientFileForMessageOnlyProto(t *testing.T) {
	plugin, file := newTSClientTestPlugin(t)

	gen := New(plugin, Options{})
	if err := gen.generateClientFile(file); err != nil {
		t.Fatalf("generateClientFile() error = %v", err)
	}

	output := generatedContent(t, plugin, "tsclient_test_client.ts")
	if !strings.Contains(output, "export interface DashboardState {") {
		t.Fatalf("generated output missing DashboardState interface:\n%s", output)
	}
	if !strings.Contains(output, "ownerId: string;") {
		t.Fatalf("generated output missing JSON-style ownerId field:\n%s", output)
	}
	if strings.Contains(output, "export class TestServiceClient") {
		t.Fatalf("message-only proto should not generate a client class:\n%s", output)
	}
}

func TestGenerateClientFileUsesProtoFieldNamesWhenRequested(t *testing.T) {
	plugin, file := newTSClientTestPlugin(t)

	gen := New(plugin, Options{FieldNames: protoFieldNames})
	if err := gen.generateClientFile(file); err != nil {
		t.Fatalf("generateClientFile() error = %v", err)
	}

	output := generatedContent(t, plugin, "tsclient_test_client.ts")
	if !strings.Contains(output, "owner_id: string;") {
		t.Fatalf("generated output missing proto owner_id field:\n%s", output)
	}
	if strings.Contains(output, "ownerId: string;") {
		t.Fatalf("generated output unexpectedly used JSON field names:\n%s", output)
	}
}

func TestGenerateURLBuildingRespectsConfiguredFieldNames(t *testing.T) {
	_, file := newTSClientTestPlugin(t)
	reqMsg := findTSClientMessage(t, file, "SearchWorkersRequest")

	cfg := &rpcMethodConfig{
		fullPath:   "/v1/workers/{worker_id}",
		httpMethod: "GET",
		pathParams: []string{"worker_id"},
		queryParams: []annotations.QueryParam{
			{
				ParamName:     "tag_ids",
				FieldJSONName: "tagIds",
				Field:         findTSClientField(t, reqMsg, "tag_ids"),
			},
			{
				ParamName:     "include_inactive",
				FieldJSONName: "includeInactive",
				Field:         findTSClientField(t, reqMsg, "include_inactive"),
			},
		},
	}

	var jsonLines []string
	(&Generator{}).generateURLBuilding(tsClientPrinter(&jsonLines), cfg)
	jsonOutput := strings.Join(jsonLines, "\n")
	if !strings.Contains(jsonOutput, "req.workerId") {
		t.Fatalf("JSON URL output missing workerId access:\n%s", jsonOutput)
	}
	if !strings.Contains(jsonOutput, "req.tagIds") {
		t.Fatalf("JSON URL output missing tagIds access:\n%s", jsonOutput)
	}
	if !strings.Contains(jsonOutput, "req.includeInactive") {
		t.Fatalf("JSON URL output missing includeInactive access:\n%s", jsonOutput)
	}

	var protoLines []string
	(&Generator{opts: Options{FieldNames: protoFieldNames}}).generateURLBuilding(tsClientPrinter(&protoLines), cfg)
	protoOutput := strings.Join(protoLines, "\n")
	if !strings.Contains(protoOutput, "req.worker_id") {
		t.Fatalf("proto URL output missing worker_id access:\n%s", protoOutput)
	}
	if !strings.Contains(protoOutput, "req.tag_ids") {
		t.Fatalf("proto URL output missing tag_ids access:\n%s", protoOutput)
	}
	if !strings.Contains(protoOutput, "req.include_inactive") {
		t.Fatalf("proto URL output missing include_inactive access:\n%s", protoOutput)
	}
}

func newTSClientTestPlugin(t *testing.T) (*protogen.Plugin, *protogen.File) {
	t.Helper()

	fileDescriptor := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("tsclient_test.proto"),
		Package: proto.String("test.tsclient.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("github.com/SebastienMelki/sebuf/internal/testdata/tsclient;tsclientpb"),
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
						TypeName: proto.String(".test.tsclient.v1.Worker"),
					},
					{
						Name:     proto.String("primary_role"),
						Number:   proto.Int32(3),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".test.tsclient.v1.Role"),
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
					{
						Name:   proto.String("tag_ids"),
						Number: proto.Int32(2),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:   proto.String("include_inactive"),
						Number: proto.Int32(3),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
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
	return plugin, file
}

func generatedContent(t *testing.T, plugin *protogen.Plugin, filename string) string {
	t.Helper()
	resp := plugin.Response()
	for _, file := range resp.File {
		if file.GetName() == filename {
			return file.GetContent()
		}
	}
	t.Fatalf("generated file %q not found", filename)
	return ""
}

func findTSClientMessage(t *testing.T, file *protogen.File, name string) *protogen.Message {
	t.Helper()
	for _, msg := range file.Messages {
		if string(msg.Desc.Name()) == name {
			return msg
		}
	}
	t.Fatalf("message %q not found", name)
	return nil
}

func findTSClientField(t *testing.T, msg *protogen.Message, name string) *protogen.Field {
	t.Helper()
	for _, field := range msg.Fields {
		if string(field.Desc.Name()) == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return nil
}

func tsClientPrinter(lines *[]string) printer {
	return func(format string, args ...interface{}) {
		if len(args) == 0 {
			*lines = append(*lines, format)
			return
		}
		*lines = append(*lines, fmt.Sprintf(format, args...))
	}
}
