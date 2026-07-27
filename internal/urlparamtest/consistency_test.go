package urlparamtest

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

	"github.com/SebastienMelki/sebuf/internal/clientgen"
	"github.com/SebastienMelki/sebuf/internal/httpgen"
	"github.com/SebastienMelki/sebuf/internal/openapiv3"
	"github.com/SebastienMelki/sebuf/internal/pyclientgen"
	"github.com/SebastienMelki/sebuf/internal/tsclientgen"
	"github.com/SebastienMelki/sebuf/internal/tsservergen"
)

type generatorCase struct {
	name string
	run  func(*protogen.Plugin) error
}

// allGenerators drives each plugin from a prepared *protogen.Plugin. Every generator
// must agree on which protos are acceptable — a plugin missing from this list is a
// plugin that can silently drift back to emitting broken output.
func allGenerators() []generatorCase {
	return []generatorCase{
		{"go-http", func(p *protogen.Plugin) error { return httpgen.New(p).Generate() }},
		{"go-client", func(p *protogen.Plugin) error { return clientgen.New(p).Generate() }},
		{"py-client", func(p *protogen.Plugin) error { return pyclientgen.New(p).Generate() }},
		{"ts-client", func(p *protogen.Plugin) error { return tsclientgen.New(p).Generate() }},
		{"ts-server", func(p *protogen.Plugin) error { return tsservergen.New(p).Generate() }},
		{"openapiv3", openapiv3.ValidateFiles},
	}
}

// TestAllGeneratorsRejectNonScalarURLParams is the anti-drift test for issue #216.
//
// Before this change each generator failed differently on a message-kind query param
// — non-compiling Go, a Python dataclass repr on the wire, "[object Object]" in
// TypeScript, a runtime 400 from the Go server, and an OpenAPI document describing a
// contract none of them implemented. Now all six must fail at generation time with
// the same message.
func TestAllGeneratorsRejectNonScalarURLParams(t *testing.T) {
	requireProtoc(t)

	testCases := []struct {
		name      string
		protoFile string
		wantErr   []string // substrings every generator's error must contain
	}{
		{
			name:      "singular message query param",
			protoFile: "invalid_message_query_param.proto",
			wantErr: []string{
				"field 'client_id'",
				"is annotated with (sebuf.http.query) as parameter 'clientId'",
				"message (invalid_message_query_param.UserClientID)",
				"must be scalar types",
			},
		},
		{
			name:      "repeated message query param",
			protoFile: "invalid_repeated_message_query_param.proto",
			wantErr: []string{
				"field 'client_ids'",
				"message (invalid_repeated_message_query_param.UserClientID)",
			},
		},
		{
			name:      "map query param",
			protoFile: "invalid_map_query_param.proto",
			wantErr: []string{
				"field 'filters'",
				"unsupported type 'map'",
			},
		},
		{
			name:      "bytes query param",
			protoFile: "invalid_bytes_query_param.proto",
			wantErr: []string{
				"field 'checksum'",
				"unsupported type 'bytes'",
			},
		},
		{
			name:      "message path param",
			protoFile: "invalid_message_path_param.proto",
			wantErr: []string{
				"path variable '{client_id}'",
				"message (invalid_message_path_param.UserClientID)",
				"Change the field type or remove it from the path",
			},
		},
		{
			// The kind check alone misses this: the kind is `string`. Accepting it
			// makes the generated Go server panic on reflectMsg.Set.
			name:      "repeated scalar path param",
			protoFile: "invalid_repeated_scalar_path_param.proto",
			wantErr: []string{
				"path variable '{ids}'",
				"of type 'repeated string'",
				"cannot be repeated",
				"(sebuf.http.query)",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, gen := range allGenerators() {
				t.Run(gen.name, func(t *testing.T) {
					plugin := buildPlugin(t, []string{tc.protoFile})

					err := gen.run(plugin)
					if err == nil {
						t.Fatalf("%s accepted %s, but every generator must reject it", gen.name, tc.protoFile)
					}

					for _, want := range tc.wantErr {
						if !strings.Contains(err.Error(), want) {
							t.Errorf("%s error missing %q\ngot: %v", gen.name, want, err)
						}
					}
				})
			}
		})
	}
}

// TestAllGeneratorsAcceptScalarURLParams is the positive control: the rejection must
// not over-match. It also pins the documented issue #216 workaround — replacing the
// typed-ID wrapper with a plain string field — as something every generator accepts.
func TestAllGeneratorsAcceptScalarURLParams(t *testing.T) {
	requireProtoc(t)

	for _, gen := range allGenerators() {
		t.Run(gen.name, func(t *testing.T) {
			plugin := buildPlugin(t, []string{"valid_scalar_url_params.proto"})

			if err := gen.run(plugin); err != nil {
				t.Fatalf("%s rejected a valid scalar-only proto: %v", gen.name, err)
			}
		})
	}
}

func requireProtoc(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping URL parameter consistency tests")
	}
}

// buildPlugin compiles the fixtures to a descriptor set and wraps them in a
// *protogen.Plugin, so each generator runs in-process and its statements count
// toward coverage.
func buildPlugin(t *testing.T, protoFiles []string) *protogen.Plugin {
	t.Helper()

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

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
		t.Fatalf("failed to read descriptor set: %v", readErr)
	}

	var fds descriptorpb.FileDescriptorSet
	if unmarshalErr := proto.Unmarshal(raw, &fds); unmarshalErr != nil {
		t.Fatalf("failed to unmarshal descriptor set: %v", unmarshalErr)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: protoFiles,
		Parameter:      proto.String("paths=source_relative"),
		ProtoFile:      fds.GetFile(),
	}

	plugin, newErr := protogen.Options{}.New(req)
	if newErr != nil {
		t.Fatalf("protogen.New failed: %v", newErr)
	}
	plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

	return plugin
}
