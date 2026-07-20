package tsclientgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/typecheck"
)

// TestTSClientGenESGoldenTypecheck runs tsc --noEmit over the checked-in es
// goldens (testdata/golden/es). Byte-comparing goldens locks down what sebuf
// emits; this locks down that it compiles against the real @bufbuild/protobuf
// types and the symbols protoc-gen-es actually exported — the machine check
// that ESQualifiedName still matches protoc-gen-es naming. Skips (never fails)
// when tsc/npx or a @bufbuild/protobuf install is unavailable.
func TestTSClientGenESGoldenTypecheck(t *testing.T) {
	goldenES := filepath.Join("testdata", "golden", "es")
	linkESNodeModules(t, goldenES)
	typecheck.Dir(t, goldenES)
}

// TestTSClientGenESResultGoldenTypecheck runs tsc --noEmit over the es +
// ts_error_handling=result goldens (testdata/golden/es-result). It is the
// machine check that the Result return type, the ClientError union (built-ins +
// proto *Error types), the structural registry, and decodeError all compile
// against the real @bufbuild/protobuf types. Skips when the toolchain is absent.
func TestTSClientGenESResultGoldenTypecheck(t *testing.T) {
	goldenES := filepath.Join("testdata", "golden", "es-result")
	linkESNodeModules(t, goldenES)
	typecheck.Dir(t, goldenES)
}

// linkESNodeModules symlinks a node_modules that resolves @bufbuild/protobuf
// into dir so tsc (nodenext resolution) can resolve the runtime imports. It
// mirrors the discovery in conformance_test.go: SEBUF_ES_NODE_MODULES overrides,
// else the git-ignored .scratch/es-spike install; skips when neither exists.
// The symlink is created only if absent and removed on cleanup, leaving a
// developer's own symlink untouched.
func linkESNodeModules(t *testing.T, dir string) {
	t.Helper()

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	nodeModules := os.Getenv("SEBUF_ES_NODE_MODULES")
	if nodeModules == "" {
		nodeModules = filepath.Join(baseDir, "..", "..", ".scratch", "es-spike", "node_modules")
	}
	if _, statErr := os.Stat(filepath.Join(nodeModules, "@bufbuild", "protobuf", "package.json")); statErr != nil {
		t.Skipf("@bufbuild/protobuf not found under %s, skipping es typecheck", nodeModules)
	}
	absNodeModules, err := filepath.Abs(nodeModules)
	if err != nil {
		t.Fatalf("Failed to resolve node_modules path: %v", err)
	}

	linkPath := filepath.Join(dir, "node_modules")
	if _, linkErr := os.Lstat(linkPath); os.IsNotExist(linkErr) {
		if symErr := os.Symlink(absNodeModules, linkPath); symErr != nil {
			t.Fatalf("Failed to symlink node_modules for es typecheck: %v", symErr)
		}
		t.Cleanup(func() { _ = os.Remove(linkPath) })
	}
}
