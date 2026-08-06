package tsclientgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

// TestTSClientGenHandRolledNeverImportsProtobufES asserts that hand-rolled
// output never imports @bufbuild/protobuf, even when an RPC is literally named
// Create (rendering `async create(` in the body). Hand-rolled consumers do not
// install the protobuf-es runtime, so any such import breaks their build.
func TestTSClientGenHandRolledNeverImportsProtobufES(t *testing.T) {
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
	if genErr := New(plugin, tscommon.MessageRuntimeHandRolled, tscommon.ErrorHandlingThrow).Generate(); genErr != nil {
		t.Fatalf("Generate() failed: %v", genErr)
	}
	for _, f := range plugin.Response().GetFile() {
		if strings.Contains(f.GetContent(), "@bufbuild/protobuf") {
			t.Errorf("hand-rolled output %s imports @bufbuild/protobuf:\n%s", f.GetName(), f.GetContent())
		}
	}
}

// TestTSClientGenESImportsOnlyUsedRuntimeSymbols asserts that an es-mode client
// imports only the protobuf-es helpers it actually calls. The fixture is a
// GET-only method whose query field is named create, so `req.create` appears in
// the body; the import must still be just fromJson + MessageInitShape — an
// unused `create` import fails consumers compiling with noUnusedLocals.
func TestTSClientGenESImportsOnlyUsedRuntimeSymbols(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping in-process runtime-symbol test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

	plugin := buildInProcessPlugin(t, protoDir, projectRoot, []string{"runtime_symbol_field.proto"})
	if genErr := New(plugin, tscommon.MessageRuntimeES, tscommon.ErrorHandlingThrow).Generate(); genErr != nil {
		t.Fatalf("Generate() failed: %v", genErr)
	}

	var clientContent string
	for _, f := range plugin.Response().GetFile() {
		if strings.HasSuffix(f.GetName(), "_client.ts") {
			clientContent = f.GetContent()
		}
	}
	if clientContent == "" {
		t.Fatal("generator emitted no _client.ts file")
	}

	var importLine string
	for _, line := range strings.Split(clientContent, "\n") {
		if strings.Contains(line, `from "@bufbuild/protobuf"`) {
			importLine = line
			break
		}
	}
	if importLine == "" {
		t.Fatalf("es-mode client has no @bufbuild/protobuf import:\n%s", clientContent)
	}
	for _, unused := range []string{"create", "toJson"} {
		if strings.Contains(importLine, unused) {
			t.Errorf("GET-only es client imports unused runtime symbol %q: %s", unused, importLine)
		}
	}
	for _, used := range []string{"fromJson", "MessageInitShape"} {
		if !strings.Contains(importLine, used) {
			t.Errorf("es client import is missing %q: %s", used, importLine)
		}
	}
}
