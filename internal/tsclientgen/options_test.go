package tsclientgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/plugintest"
)

var allMessageFixture = []string{
	"allmessages/v1/messages.proto",
	"allmessages/v1/service.proto",
}

func TestGenerateEmitsEveryDeclarationFromMessageOnlyCrossFile(t *testing.T) {
	protoDir, projectRoot := tsClientTestDirs(t)
	plugin := buildInProcessPlugin(t, protoDir, projectRoot, allMessageFixture)

	if err := New(plugin).Generate(); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	content := generatedFileContent(t, plugin, "allmessages/v1/messages.ts")
	for _, want := range []string{
		"export interface Item {",
		"export interface UnusedState {",
		"export interface UnusedStateDetails {",
		"export interface UnusedStateOrphan {",
		"export type UnusedKind =",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("message-only module missing %q\n---\n%s", want, content)
		}
	}
}

func TestFieldNamesProtoCoversTypesPathAndQueryAccess(t *testing.T) {
	protoDir, projectRoot := tsClientTestDirs(t)
	plugin := buildInProcessPlugin(t, protoDir, projectRoot, allMessageFixture)

	gen := NewWithOptions(plugin, Options{FieldNames: protoFieldNames})
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	messages := generatedFileContent(t, plugin, "allmessages/v1/messages.ts")
	serviceTypes := generatedFileContent(t, plugin, "allmessages/v1/service.ts")
	client := generatedFileContent(t, plugin, "allmessages/v1/service_client.ts")
	for _, check := range []struct {
		name    string
		content string
		want    string
	}{
		{name: "cross-file response field", content: messages, want: "display_name: string;"},
		{name: "unused message field", content: messages, want: "owner_id: string;"},
		{name: "request type field", content: serviceTypes, want: "resource_id: string;"},
		{name: "path access", content: client, want: "req.resource_id"},
		{name: "query access", content: client, want: "req.page_size"},
	} {
		if !strings.Contains(check.content, check.want) {
			t.Errorf("%s missing %q\n---\n%s", check.name, check.want, check.content)
		}
	}
	for _, forbidden := range []string{"displayName: string;", "ownerId: string;", "req.resourceId", "req.pageSize"} {
		if strings.Contains(messages+serviceTypes+client, forbidden) {
			t.Errorf("field_names=proto output unexpectedly contains JSON-name access %q", forbidden)
		}
	}
}

func TestFieldNamesRejectsUnknownValue(t *testing.T) {
	protoDir, projectRoot := tsClientTestDirs(t)
	plugin := buildInProcessPlugin(t, protoDir, projectRoot, allMessageFixture)

	err := NewWithOptions(plugin, Options{FieldNames: "invalid"}).Generate()
	if err == nil || !strings.Contains(err.Error(), "field_names must be json or proto") {
		t.Fatalf("Generate() error = %v, want field_names validation error", err)
	}
}

func TestFieldNamesProtoPluginOption(t *testing.T) {
	protoDir, projectRoot := tsClientTestDirs(t)
	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-ts-client")
	outDir := t.TempDir()
	args := []string{
		"--plugin=protoc-gen-ts-client=" + pluginPath,
		"--ts-client_out=" + outDir,
		"--ts-client_opt=paths=source_relative,field_names=proto",
		"--proto_path=" + protoDir,
		"--proto_path=" + filepath.Join(projectRoot, "proto"),
	}
	args = append(args, allMessageFixture...)
	cmd := exec.Command("protoc", args...)
	cmd.Dir = protoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("protoc failed: %v\noutput: %s", err, output)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "allmessages", "v1", "service_client.ts"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	if !strings.Contains(string(content), "req.resource_id") || !strings.Contains(string(content), "req.page_size") {
		t.Fatalf("field_names=proto plugin option was not applied:\n%s", content)
	}
}

func tsClientTestDirs(t *testing.T) (protoDir, projectRoot string) {
	t.Helper()
	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	return filepath.Join(baseDir, "testdata", "proto"), filepath.Join(baseDir, "..", "..")
}
