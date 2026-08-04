package httpgen

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
)

// readInt64Golden reads a golden file produced by TestHTTPGenGoldenFiles, skipping the test if
// it has not been generated yet.
func readInt64Golden(t *testing.T, name string) string {
	t.Helper()

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	goldenFile := filepath.Join(baseDir, "testdata", "golden", name)
	content, readErr := os.ReadFile(goldenFile)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			t.Skipf("Golden file not found: %s — run with UPDATE_GOLDEN=1 to generate it", goldenFile)
		}
		t.Fatalf("Failed to read golden file: %v", readErr)
	}
	return string(content)
}

// TestCrossFileInt64WrapperGenerated covers issue #217: when the message carrying
// int64_encoding=NUMBER is declared in an imported file, the importing message must still get a
// transitive MarshalJSONSebuf. Before the fix, wrapper detection tested membership in a name set
// built from the file being generated, so imported types never qualified, no wrapper was emitted,
// and protojson serialized the int64 as a quoted string.
func TestCrossFileInt64WrapperGenerated(t *testing.T) {
	src := readInt64Golden(t, "int64_cross_file_response_encoding.pb.go")

	// Both the singular and the repeated case are reported in the issue.
	for _, method := range []string{
		"func (x *GetSensorReadingResponse) MarshalJSONSebuf(",
		"func (x *GetSensorReadingResponse) UnmarshalJSONSebuf(",
		"func (x *GetSensorReadingsResponse) MarshalJSONSebuf(",
		"func (x *GetSensorReadingsResponse) UnmarshalJSONSebuf(",
	} {
		if !strings.Contains(src, method) {
			t.Errorf("missing %q -- the nested type is imported from another file, so wrapper "+
				"detection fell back to the per-file name set (issue #217)", method)
		}
	}

	// The wrapper must forward opts to the child rather than letting protojson own the field.
	if !strings.Contains(src, "m.MarshalJSONSebuf(opts)") {
		t.Error("cross-file wrapper does not forward opts to the nested message's MarshalJSONSebuf: " +
			"int64 NUMBER fields will serialize as quoted strings")
	}

	// The repeated field must be decoded element-by-element, not as a single message.
	if !strings.Contains(src, "var rawItems []json.RawMessage") {
		t.Error("repeated nested field is not decoded per-element on the unmarshal side: " +
			"unmarshaling a JSON array into a single message will fail at runtime")
	}
}

// TestDeepNestedInt64WrapperGenerated covers the depth > 1 half of issue #217. Wrapper messages
// were never fed back into the qualifying set, so given Outer -> Middle -> Leaf only Middle got a
// marshaler and Outer silently bypassed it — even within a single file.
func TestDeepNestedInt64WrapperGenerated(t *testing.T) {
	src := readInt64Golden(t, "int64_deep_nested_encoding_encoding.pb.go")

	// Every message in the chain gets a marshaler: the leaf directly, the rest transitively.
	for _, method := range []string{
		"func (x *Leaf) MarshalJSONSebuf(",
		"func (x *Middle) MarshalJSONSebuf(",
		"func (x *Outer) MarshalJSONSebuf(",
		"func (x *OuterList) MarshalJSONSebuf(",
	} {
		if !strings.Contains(src, method) {
			t.Errorf("missing %q -- nesting deeper than one level was not detected (issue #217)", method)
		}
	}

	// The visited-set cycle guard must let a self-referential message terminate: Node holds a
	// Node, and NodeHolder reaches int64 only through that cyclic type.
	for _, method := range []string{
		"func (x *Node) MarshalJSONSebuf(",
		"func (x *NodeHolder) MarshalJSONSebuf(",
	} {
		if !strings.Contains(src, method) {
			t.Errorf("missing %q -- recursive message definitions are not handled", method)
		}
	}
}

// TestInt64WrapperMarshalJSONConflict verifies that a message needing a transitive int64 wrapper
// which also owns MarshalJSON via another annotation is rejected at generation time. Widening
// wrapper detection widens the set of messages that can collide: a silent skip would serialize
// the int64 as a quoted string, and a duplicate emit would not compile.
//
// Both Go generators carry their own copy of the check, so both are driven here — the two must
// never disagree about which protos they accept.
func TestInt64WrapperMarshalJSONConflict(t *testing.T) {
	requireProtocForInt64Tests(t)

	generators := []struct {
		name string
		run  func(*protogen.Plugin) error
	}{
		{"go-http", func(p *protogen.Plugin) error { return New(p).Generate() }},
		{"go-client", func(p *protogen.Plugin) error { return clientgen.New(p).Generate() }},
	}

	t.Run("conflict is rejected and names both features", func(t *testing.T) {
		for _, gen := range generators {
			t.Run(gen.name, func(t *testing.T) {
				err := gen.run(buildInt64TestPlugin(t, []string{"int64_wrapper_conflict.proto"}))
				if err == nil {
					t.Fatal("expected generation to fail for a message that is both an int64 " +
						"wrapper and carries flatten -- emitting both would declare " +
						"MarshalJSONSebuf twice on the same Go type")
				}
				for _, want := range []string{"ConflictingResponse", "flatten", "only one MarshalJSON"} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("conflict error should mention %q, got: %v", want, err)
					}
				}
			})
		}
	})

	// Positive control: the rejection must not over-match. A plain wrapper chain, including
	// the cyclic Node/NodeHolder pair, has to keep generating cleanly.
	t.Run("plain wrapper chains are still accepted", func(t *testing.T) {
		for _, gen := range generators {
			t.Run(gen.name, func(t *testing.T) {
				err := gen.run(buildInt64TestPlugin(t, []string{"int64_deep_nested_encoding.proto"}))
				if err != nil {
					t.Fatalf("generator rejected a valid int64 wrapper chain: %v", err)
				}
			})
		}
	})

	// A message reaching int64 only through a map value must NOT be treated as a wrapper: the
	// emitters cannot traverse maps, so the marshaler would be dead code, and here it would also
	// collide with a flatten annotation the message legitimately carries.
	t.Run("map-only reachability is not a wrapper and not a conflict", func(t *testing.T) {
		for _, gen := range generators {
			t.Run(gen.name, func(t *testing.T) {
				err := gen.run(buildInt64TestPlugin(t, []string{"int64_map_path_only.proto"}))
				if err != nil {
					t.Fatalf("generator reported a conflict for a message that only reaches int64 "+
						"through a map value -- the wrapper it would emit cannot serialize that "+
						"path anyway: %v", err)
				}
			})
		}
	})
}

func requireProtocForInt64Tests(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping int64 wrapper conflict tests")
	}
}

// buildInt64TestPlugin compiles fixtures to a descriptor set and wraps them in a
// *protogen.Plugin, so each generator runs in-process and its statements count toward coverage.
// Mirrors buildPlugin in internal/urlparamtest.
func buildInt64TestPlugin(t *testing.T, protoFiles []string) *protogen.Plugin {
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
