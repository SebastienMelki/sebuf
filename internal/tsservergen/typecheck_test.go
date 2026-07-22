package tsservergen

import (
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/typecheck"
)

// TestGoldenTypecheck compiles the entire golden tree with tsc --noEmit under
// strict nodenext settings, proving the generated server modules, type
// modules, barrels, and errors.ts form a valid TypeScript program with
// resolvable .js relative imports.
func TestGoldenTypecheck(t *testing.T) {
	typecheck.Dir(t, filepath.Join("testdata", "golden"))
}
