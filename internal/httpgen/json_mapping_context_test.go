package httpgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestJSONMappingContextCollectsSameMessageTransforms(t *testing.T) {
	file := buildJSONMappingContextTestFile(t, `
message NullableTimestamp {
  optional string note = 1 [(sebuf.http.nullable) = true];
  google.protobuf.Timestamp at = 2 [(sebuf.http.timestamp_format) = TIMESTAMP_FORMAT_UNIX_SECONDS];
}
`)

	contexts, err := collectJSONMappingContexts(file)
	if err != nil {
		t.Fatalf("collectJSONMappingContexts: %v", err)
	}
	if len(contexts) != 1 {
		t.Fatalf("contexts len = %d, want 1", len(contexts))
	}

	ctx := requireJSONMappingContext(t, contexts, "NullableTimestamp")
	if len(ctx.FieldTransforms) != 2 {
		t.Fatalf("FieldTransforms len = %d, want 2", len(ctx.FieldTransforms))
	}
	assertFieldTransform(t, ctx, "note", TransformNullable)
	assertFieldTransform(t, ctx, "at", TransformTimestampFormat)
	if len(ctx.NestedDelegationFields) != 0 {
		t.Fatalf("NestedDelegationFields len = %d, want 0", len(ctx.NestedDelegationFields))
	}
	if ctx.RootUnwrap != nil {
		t.Fatalf("RootUnwrap = %#v, want nil", ctx.RootUnwrap)
	}
}

func TestJSONMappingContextCollectsNestedDelegation(t *testing.T) {
	file := buildJSONMappingContextTestFile(t, `
message HexChild {
  bytes payload = 1 [(sebuf.http.bytes_encoding) = BYTES_ENCODING_HEX];
}

message Parent {
  HexChild child = 1;
  string name = 2;
}
`)

	contexts, err := collectJSONMappingContexts(file)
	if err != nil {
		t.Fatalf("collectJSONMappingContexts: %v", err)
	}

	parent := requireJSONMappingContext(t, contexts, "Parent")
	if len(parent.FieldTransforms) != 0 {
		t.Fatalf("Parent FieldTransforms len = %d, want 0", len(parent.FieldTransforms))
	}
	assertNestedDelegationField(t, parent, "child")

	child := requireJSONMappingContext(t, contexts, "HexChild")
	assertFieldTransform(t, child, "payload", TransformBytesEncoding)

	childField := fieldByName(t, parent.Message, "child")
	if !fieldNeedsNestedJSONDelegation(childField) {
		t.Fatalf("fieldNeedsNestedJSONDelegation(child) = false, want true")
	}
}

func TestJSONMappingContextCollectsMapValueUnwrapTransform(t *testing.T) {
	file := buildJSONMappingContextTestFile(t, `
message MapItem {
  string id = 1;
}

message MapItemList {
  repeated MapItem items = 1 [(sebuf.http.unwrap) = true];
}

message MapParent {
  map<string, MapItemList> lists = 1;
}
`)

	contexts, err := collectJSONMappingContexts(file)
	if err != nil {
		t.Fatalf("collectJSONMappingContexts: %v", err)
	}

	parent := requireJSONMappingContext(t, contexts, "MapParent")
	assertFieldTransform(t, parent, "lists", TransformMapValueUnwrap)
}

func TestJSONMappingContextCollectsRootUnwrapDocumentTransform(t *testing.T) {
	file := buildJSONMappingContextTestFile(t, `
message RootItem {
  string id = 1;
}

message RootList {
  repeated RootItem items = 1 [(sebuf.http.unwrap) = true];
}
`)

	contexts, err := collectJSONMappingContexts(file)
	if err != nil {
		t.Fatalf("collectJSONMappingContexts: %v", err)
	}

	ctx := requireJSONMappingContext(t, contexts, "RootList")
	if ctx.RootUnwrap == nil {
		t.Fatalf("RootUnwrap = nil, want document transform")
	}
	if got := string(ctx.RootUnwrap.UnwrapField.Desc.Name()); got != "items" {
		t.Fatalf("RootUnwrap field = %q, want items", got)
	}
	if len(ctx.FieldTransforms) != 0 {
		t.Fatalf("FieldTransforms len = %d, want 0 for root unwrap", len(ctx.FieldTransforms))
	}
	if !messageNeedsJSONMapping(ctx.Message) {
		t.Fatalf("messageNeedsJSONMapping(RootList) = false, want true")
	}
}

func TestJSONMappingContextSkipsUnannotatedMessages(t *testing.T) {
	file := buildJSONMappingContextTestFile(t, `
message PlainChild {
  string id = 1;
}

message PlainParent {
  PlainChild child = 1;
  repeated PlainChild children = 2;
  map<string, PlainChild> child_by_key = 3;
}
`)

	contexts, err := collectJSONMappingContexts(file)
	if err != nil {
		t.Fatalf("collectJSONMappingContexts: %v", err)
	}
	if len(contexts) != 0 {
		t.Fatalf("contexts len = %d, want 0", len(contexts))
	}
}

func buildJSONMappingContextTestFile(t *testing.T, body string) *protogen.File {
	t.Helper()

	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping JSON mapping context collector tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")

	tempDir := t.TempDir()
	protoPath := filepath.Join(tempDir, "context.proto")
	if err := os.WriteFile(protoPath, []byte(jsonMappingContextTestProto(body)), 0o644); err != nil {
		t.Fatalf("write proto fixture: %v", err)
	}

	descPath := filepath.Join(tempDir, "descriptors.pb")
	cmd := exec.Command("protoc",
		"--descriptor_set_out="+descPath,
		"--include_imports",
		"--proto_path="+tempDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"context.proto",
	)
	cmd.Dir = tempDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("protoc descriptor_set_out failed: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(descPath)
	if err != nil {
		t.Fatalf("read descriptor set: %v", err)
	}

	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &fds); err != nil {
		t.Fatalf("unmarshal descriptor set: %v", err)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"context.proto"},
		Parameter:      proto.String("paths=source_relative"),
		ProtoFile:      fds.GetFile(),
	}
	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("protogen.New: %v", err)
	}

	for _, file := range plugin.Files {
		if file.Desc.Path() == "context.proto" {
			return file
		}
	}
	t.Fatalf("generated protogen file context.proto not found")
	return nil
}

func jsonMappingContextTestProto(body string) string {
	return `syntax = "proto3";

package contexttest;

import "google/protobuf/timestamp.proto";
import "sebuf/http/annotations.proto";

option go_package = "github.com/SebastienMelki/sebuf/internal/httpgen/contexttest;contexttest";

` + body
}

func requireJSONMappingContext(
	t *testing.T,
	contexts []*JSONMappingContext,
	goName string,
) *JSONMappingContext {
	t.Helper()
	for _, ctx := range contexts {
		if ctx.Message.GoIdent.GoName == goName {
			return ctx
		}
	}
	t.Fatalf("context for %s not found; got %v", goName, contextMessageNames(contexts))
	return nil
}

func contextMessageNames(contexts []*JSONMappingContext) []string {
	names := make([]string, 0, len(contexts))
	for _, ctx := range contexts {
		names = append(names, ctx.Message.GoIdent.GoName)
	}
	return names
}

func assertFieldTransform(
	t *testing.T,
	ctx *JSONMappingContext,
	fieldName string,
	kind JSONMappingTransformKind,
) {
	t.Helper()
	for _, transform := range ctx.FieldTransforms {
		if transform.Field != nil && string(transform.Field.Desc.Name()) == fieldName && transform.Kind == kind {
			return
		}
	}
	t.Fatalf("transform %s on field %s not found in %#v", kind, fieldName, ctx.FieldTransforms)
}

func assertNestedDelegationField(t *testing.T, ctx *JSONMappingContext, fieldName string) {
	t.Helper()
	for _, field := range ctx.NestedDelegationFields {
		if string(field.Desc.Name()) == fieldName {
			return
		}
	}
	t.Fatalf("nested delegation field %s not found in %v", fieldName, fieldNames(ctx.NestedDelegationFields))
}

func fieldByName(t *testing.T, msg *protogen.Message, fieldName string) *protogen.Field {
	t.Helper()
	for _, field := range msg.Fields {
		if string(field.Desc.Name()) == fieldName {
			return field
		}
	}
	t.Fatalf("field %s not found on %s", fieldName, msg.GoIdent.GoName)
	return nil
}

func fieldNames(fields []*protogen.Field) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, string(field.Desc.Name()))
	}
	return names
}
