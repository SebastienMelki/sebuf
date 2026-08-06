package tsclientgen

import (
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/typecheck"
)

// TestGoldenTypecheck compiles the hand-rolled golden tree with tsc --noEmit
// under strict nodenext settings, proving the generated client modules, type
// modules, barrels, and errors.ts form a valid TypeScript program with
// resolvable .js relative imports. The protobuf-es goldens are excluded
// because they import @bufbuild/protobuf, which needs a linked node_modules to
// resolve; TestTSClientGenESGoldenTypecheck covers them with that link in
// place.
func TestGoldenTypecheck(t *testing.T) {
	typecheck.DirExcluding(t, filepath.Join("testdata", "golden"), typecheck.ESGoldenDirs()...)
}
