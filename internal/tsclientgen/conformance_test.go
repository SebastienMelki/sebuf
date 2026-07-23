package tsclientgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestESWireConformance runs the protobuf-es wire-conformance proof
// (testdata/es/conformance.test.mjs) under node. It is the executable guarantee
// that an es-mode client decodes the Go server's default protojson correctly:
//
//   - zero-valued scalars/bools/int64 and empty lists that the server OMITS are
//     MATERIALIZED after fromJson ("" / 0 / false / 0n / []);
//   - toJson re-emits the same canonical (zero-values-omitted) form;
//   - unknown server fields are tolerated when ignoreUnknownFields is set.
//
// The .mjs imports @bufbuild/protobuf via bare specifiers (through the generated
// conformance_pb.js), so node needs a node_modules with @bufbuild/protobuf in
// testdata/es's directory tree. This test symlinks one in (git-ignored) from the
// Task-1 spike install, runs node, and cleans up. It SKIPS cleanly — never fails
// — when node or @bufbuild/protobuf is unavailable, so CI stays green. Mirrors
// the protoc/protoc-gen-es LookPath skips in golden_test.go.
//
// To run manually, see the header of testdata/es/conformance.test.mjs.
func TestESWireConformance(t *testing.T) {
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found, skipping protobuf-es wire-conformance test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(baseDir, "..", "..")
	esDir := filepath.Join(baseDir, "testdata", "es")

	// Locate a node_modules that resolves @bufbuild/protobuf. Allow an override
	// for environments that install it elsewhere; otherwise reuse the git-ignored
	// Task-1 spike install.
	nodeModules := os.Getenv("SEBUF_ES_NODE_MODULES")
	if nodeModules == "" {
		nodeModules = filepath.Join(projectRoot, ".scratch", "es-spike", "node_modules")
	}
	if _, statErr := os.Stat(filepath.Join(nodeModules, "@bufbuild", "protobuf", "package.json")); statErr != nil {
		t.Skipf("@bufbuild/protobuf not found under %s, skipping protobuf-es wire-conformance test", nodeModules)
	}

	// node resolves bare specifiers from a node_modules in the file's directory
	// tree, so symlink one into testdata/es. Only create/remove it if absent, so
	// a developer's existing symlink is left untouched.
	linkPath := filepath.Join(esDir, "node_modules")
	if _, linkErr := os.Lstat(linkPath); os.IsNotExist(linkErr) {
		if symErr := os.Symlink(nodeModules, linkPath); symErr != nil {
			t.Fatalf("Failed to symlink node_modules for conformance test: %v", symErr)
		}
		t.Cleanup(func() { _ = os.Remove(linkPath) })
	}

	cmd := exec.Command(nodeBin, "conformance.test.mjs")
	cmd.Dir = esDir
	output, runErr := cmd.CombinedOutput()
	t.Logf("conformance.test.mjs output:\n%s", output)
	if runErr != nil {
		t.Fatalf("protobuf-es wire-conformance check failed: %v", runErr)
	}
}
