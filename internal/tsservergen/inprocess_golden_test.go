package tsservergen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// TestTSServerGenInProcess drives the server generator in-process against the
// same fixtures and golden files as TestTSServerGenGoldenFiles. Running the
// generator inside the test binary (instead of as a subprocess protoc plugin)
// makes its statements count toward Go coverage while still asserting that the
// emitted TypeScript is byte-identical to the checked-in golden files.
func TestTSServerGenInProcess(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping in-process golden tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden")

	for _, tc := range inProcessFixtures() {
		t.Run(tc.name, func(t *testing.T) {
			plugin := buildInProcessPlugin(t, protoDir, projectRoot, tc.protoFiles)

			gen := New(plugin)
			if genErr := gen.Generate(); genErr != nil {
				t.Fatalf("Generate() failed: %v", genErr)
			}

			assertResponseMatchesGolden(t, plugin, goldenDir)
		})
	}
}

// TestTSServerGenInProcessValidationErrors drives the invalid-proto fixtures
// in-process and asserts Generate() returns an error mentioning the offending
// field, covering the error branches in buildRPCRouteConfig / validateFieldCoverage.
func TestTSServerGenInProcessValidationErrors(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping in-process validation tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

	testCases := []struct {
		name      string
		protoFile string
		wantErr   string
	}{
		{
			name:      "path param without matching field",
			protoFile: "invalid_path_param.proto",
			wantErr:   "path parameter {id} has no matching field on request message GetItemRequest",
		},
		{
			name:      "unreachable field on GET method",
			protoFile: "invalid_uncovered_field.proto",
			wantErr:   "fields [category] on request message GetItemRequest are not reachable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := buildInProcessPlugin(t, protoDir, projectRoot, []string{tc.protoFile})
			genErr := New(plugin).Generate()
			if genErr == nil {
				t.Fatalf("expected Generate() to fail for %s, but it succeeded", tc.protoFile)
			}
			if !strings.Contains(genErr.Error(), tc.wantErr) {
				t.Errorf("expected error to contain %q, got: %v", tc.wantErr, genErr)
			}
		})
	}
}

// TestTSServerGenInProcessReservedName drives a fixture whose message and enum
// collide with the shared error-helper names (ValidationError, ApiError) and
// asserts the server module imports them under deterministic aliases while the
// non-colliding hoisted nested name (WrapperValidationError) stays unaliased.
func TestTSServerGenInProcessReservedName(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping in-process reserved-name test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

	plugin := buildInProcessPlugin(t, protoDir, projectRoot, []string{"reserved_name.proto"})
	if genErr := New(plugin).Generate(); genErr != nil {
		t.Fatalf("Generate() failed: %v", genErr)
	}

	content := generatedFileContent(t, plugin, "reserved_name_server.ts")
	for _, want := range []string{
		`ApiError as ApiError_1`,
		`ValidationError as ValidationError_1`,
		`Promise<ValidationError_1>`,
		`Promise<ApiError_1[]>`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("reserved_name_server.ts missing %q\n---\n%s", want, content)
		}
	}
	if strings.Contains(content, "WrapperValidationError as") {
		t.Errorf("WrapperValidationError must not be aliased\n---\n%s", content)
	}
}

// generatedFileContent returns the content of the named file from the plugin
// response, failing the test if it was not emitted.
func generatedFileContent(t *testing.T, plugin *protogen.Plugin, name string) string {
	t.Helper()
	for _, f := range plugin.Response().GetFile() {
		if f.GetName() == name {
			return f.GetContent()
		}
	}
	t.Fatalf("generator did not emit %s", name)
	return ""
}

// inProcessFixture names a set of proto files generated together.
type inProcessFixture struct {
	name       string
	protoFiles []string
}

// inProcessFixtures returns the fixtures exercised by the in-process runner. It
// mirrors the golden-file test cases, including the two-package crosspkg fixture
// that exercises cross-package relative imports.
func inProcessFixtures() []inProcessFixture {
	return []inProcessFixture{
		{name: "comprehensive HTTP verbs", protoFiles: []string{"http_verbs_comprehensive.proto"}},
		{name: "query parameters", protoFiles: []string{"query_params.proto"}},
		{name: "backward compatibility", protoFiles: []string{"backward_compat.proto"}},
		{name: "complex features", protoFiles: []string{"complex_features.proto"}},
		{name: "unwrap variants", protoFiles: []string{"unwrap.proto"}},
		{name: "int64 encoding", protoFiles: []string{"int64_encoding.proto"}},
		{name: "enum encoding", protoFiles: []string{"enum_encoding.proto"}},
		{name: "nullable fields", protoFiles: []string{"nullable.proto"}},
		{name: "empty behavior", protoFiles: []string{"empty_behavior.proto"}},
		{name: "timestamp format", protoFiles: []string{"timestamp_format.proto"}},
		{name: "bytes encoding", protoFiles: []string{"bytes_encoding.proto"}},
		{name: "flatten", protoFiles: []string{"flatten.proto"}},
		{name: "oneof discriminator", protoFiles: []string{"oneof_discriminator.proto"}},
		{name: "SSE streaming", protoFiles: []string{"sse.proto"}},
		{name: "empty request body", protoFiles: []string{"empty_request_body.proto"}},
		{name: "reserved error-helper names", protoFiles: []string{"reserved_name.proto"}},
		{
			name:       "cross-package imports",
			protoFiles: []string{"crosspkg/common/v1/types.proto", "crosspkg/shop/v1/service.proto"},
		},
	}
}

// TestTSServerGenInProcessRequiresSourceRelative drives a fixture under
// protoc's default path mode (paths=import) and asserts Generate() fails
// loudly: type modules and service modules would otherwise land in different
// directories, silently breaking the per-package barrels.
func TestTSServerGenInProcessRequiresSourceRelative(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping in-process path-mode test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

	plugin := buildInProcessPluginParams(t, protoDir, projectRoot, []string{"query_params.proto"}, "")
	genErr := New(plugin).Generate()
	if genErr == nil {
		t.Fatal("expected Generate() to fail under default path mode, but it succeeded")
	}
	if !strings.Contains(genErr.Error(), "paths=source_relative") {
		t.Errorf("expected error to mention paths=source_relative, got: %v", genErr)
	}
}

// buildInProcessPlugin compiles the given fixtures into a FileDescriptorSet with
// protoc and constructs a protogen.Plugin from it, so the generator can run in
// the test process. Parameters mirror the golden test (paths=source_relative).
func buildInProcessPlugin(
	t *testing.T,
	protoDir, projectRoot string,
	protoFiles []string,
) *protogen.Plugin {
	t.Helper()
	return buildInProcessPluginParams(t, protoDir, projectRoot, protoFiles, "paths=source_relative")
}

// buildInProcessPluginParams is buildInProcessPlugin with an explicit plugin
// parameter string (empty means protoc's default path mode).
func buildInProcessPluginParams(
	t *testing.T,
	protoDir, projectRoot string,
	protoFiles []string,
	parameter string,
) *protogen.Plugin {
	t.Helper()

	descPath := filepath.Join(t.TempDir(), "descriptors.pb")
	args := []string{
		"--descriptor_set_out=" + descPath,
		"--include_imports",
		"--proto_path=" + protoDir,
		"--proto_path=" + filepath.Join(projectRoot, "proto"),
	}
	args = append(args, protoFiles...)
	cmd := exec.Command("protoc", args...)
	cmd.Dir = protoDir
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("protoc descriptor_set_out failed: %v\noutput: %s", runErr, out)
	}

	raw, readErr := os.ReadFile(descPath)
	if readErr != nil {
		t.Fatalf("Failed to read descriptor set: %v", readErr)
	}
	var fds descriptorpb.FileDescriptorSet
	if unmarshalErr := proto.Unmarshal(raw, &fds); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal descriptor set: %v", unmarshalErr)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: protoFiles,
		Parameter:      proto.String(parameter),
		ProtoFile:      fds.GetFile(),
	}
	plugin, newErr := protogen.Options{}.New(req)
	if newErr != nil {
		t.Fatalf("protogen.New failed: %v", newErr)
	}
	plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
	return plugin
}

// assertResponseMatchesGolden compares every file the generator emitted against
// its checked-in golden file, byte for byte.
func assertResponseMatchesGolden(t *testing.T, plugin *protogen.Plugin, goldenDir string) {
	t.Helper()

	resp := plugin.Response()
	if resp.GetError() != "" {
		t.Fatalf("generator reported error in response: %s", resp.GetError())
	}
	if len(resp.GetFile()) == 0 {
		t.Fatal("generator emitted no files")
	}

	for _, f := range resp.GetFile() {
		rel := f.GetName()
		goldenPath := filepath.Join(goldenDir, filepath.FromSlash(rel))
		compareGoldenFile(t, rel, goldenPath, []byte(f.GetContent()))
	}
}
