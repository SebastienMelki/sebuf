package tsclientgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/plugintest"
	"github.com/SebastienMelki/sebuf/internal/tscommon/typecheck"
)

// TestWirePresenceContract pins how the TypeScript output types the four
// shapes a hand-built wire contract depends on (an unset message field, an
// unset optional scalar, an explicit JSON null and an empty repeated field)
// and proves that (sebuf.http.enum_value) makes the enum's type the given
// strings. It generates with the option field_names=proto.
// The generated client sends JSON.stringify(request), so the generated types
// are the wire. testdata/golden/wire_presence/wire_presence_usage.ts is a
// hand-written consumer that the compiler checks against them, with
// @ts-expect-error lines for what must not compile.
//
// To update the golden after an intentional change:
//
//	UPDATE_GOLDEN=1 go test -run TestWirePresenceContract
func TestWirePresenceContract(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}
	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden", "wire_presence")
	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-ts-client")

	outDir := t.TempDir()
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-ts-client="+pluginPath,
		"--ts-client_out="+outDir,
		"--ts-client_opt=paths=source_relative,field_names=proto",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"wire_presence.proto",
	)
	cmd.Dir = protoDir
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("protoc failed: %v\n%s", runErr, out)
	}

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"
	for _, rel := range generatedTSFiles(t, outDir) {
		generated, readErr := os.ReadFile(filepath.Join(outDir, rel))
		if readErr != nil {
			t.Fatalf("Failed to read generated file %s: %v", rel, readErr)
		}
		goldenPath := filepath.Join(goldenDir, rel)
		if updateGolden {
			updateGoldenFile(t, goldenPath, generated)
			continue
		}
		compareGoldenFile(t, rel, goldenPath, generated)
	}

	typecheck.Dir(t, goldenDir)
}
