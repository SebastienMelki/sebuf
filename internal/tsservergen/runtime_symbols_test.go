package tsservergen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

// TestTSServerGenHandRolledNeverImportsProtobufES asserts that hand-rolled
// output never imports @bufbuild/protobuf, even when an RPC is literally named
// Create (rendering a handler method `create(` in the body). Hand-rolled
// consumers do not install the protobuf-es runtime, so any such import breaks
// their build.
func TestTSServerGenHandRolledNeverImportsProtobufES(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping in-process runtime-symbol test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

	plugin := buildInProcessPlugin(t, protoDir, projectRoot, []string{"runtime_symbol_rpc.proto"})
	if genErr := New(plugin, tscommon.MessageRuntimeHandRolled).Generate(); genErr != nil {
		t.Fatalf("Generate() failed: %v", genErr)
	}
	for _, f := range plugin.Response().GetFile() {
		if strings.Contains(f.GetContent(), "@bufbuild/protobuf") {
			t.Errorf("hand-rolled output %s imports @bufbuild/protobuf:\n%s", f.GetName(), f.GetContent())
		}
	}
}
