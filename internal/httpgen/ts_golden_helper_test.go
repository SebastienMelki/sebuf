package httpgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readCombinedTSGolden reads a fixture's TypeScript output. In the modules
// layout the output is split across the canonical type module (<name>.ts,
// holding interfaces/enums/unions) and the service module (<name>_client.ts,
// holding the client class and encoding logic). It returns their concatenation
// so cross-generator consistency checks can locate a symbol regardless of which
// module it landed in.
func readCombinedTSGolden(t *testing.T, baseDir, name string) string {
	t.Helper()
	dir := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden")
	var sb strings.Builder
	for _, suffix := range []string{".ts", "_client.ts"} {
		b, err := os.ReadFile(filepath.Join(dir, name+suffix))
		if err != nil {
			t.Fatalf("Failed to read TypeScript golden %s%s: %v", name, suffix, err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}
