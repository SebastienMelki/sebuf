package tsservergen

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestESWireConformance runs the protobuf-es server-side wire-conformance proof
// (testdata/es/conformance.test.mjs) under node. It is the ENCODE-side mirror of
// tsclientgen's TestESWireConformance: where that one proves an es-mode client
// decodes the Go server's protojson correctly, this proves an es-mode server
// ENCODES a body the Go server would have produced, which is what every sebuf
// client then has to read.
//
//   - zero-valued scalars/bools/lists/maps and zero enums are OMITTED;
//   - 64-bit ints leave as JSON strings, exact past 2^53;
//   - enums leave as their proto NAME string;
//   - a partially-populated nested message emits only its set fields;
//   - the canonical body is stable through a decode/re-encode cycle.
//
// The .mjs imports @bufbuild/protobuf via bare specifiers (through the generated
// conformance_pb.js), so node needs a node_modules with @bufbuild/protobuf in
// testdata/es's directory tree. linkESNodeModules (es_typecheck_test.go) handles
// that and SKIPS cleanly when the install is absent; this test additionally
// skips when node itself is missing, so CI stays green either way.
//
// To run manually, see the header of testdata/es/conformance.test.mjs.
func TestESWireConformance(t *testing.T) {
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found, skipping protobuf-es wire-conformance test")
	}

	esDir := filepath.Join("testdata", "es")
	linkESNodeModules(t, esDir)

	cmd := exec.Command(nodeBin, "conformance.test.mjs")
	cmd.Dir = esDir
	output, runErr := cmd.CombinedOutput()
	t.Logf("conformance.test.mjs output:\n%s", output)
	if runErr != nil {
		t.Fatalf("protobuf-es server wire-conformance check failed: %v", runErr)
	}
}
