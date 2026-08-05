// Package typecheck runs the TypeScript compiler over generated output in
// tests, verifying that the emitted module tree typechecks under the strict
// nodenext settings the modules layout targets. Byte-comparing golden files
// catches regressions in what we emit; this catches emitting something that
// was never valid TypeScript in the first place (duplicate identifiers,
// shadowed globals, unused imports, wrong casts).
package typecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// tsVersion pins the compiler fetched by the npx fallback so results don't
// drift with whatever "latest" is on the machine running the tests.
const tsVersion = "5.9.3"

// ESGoldenDirs returns the golden subdirectory patterns holding protobuf-es
// output ("es", plus any "es-*" variant such as "es-result"). Callers that
// typecheck a whole testdata/golden tree pass these to DirExcluding: the es
// goldens import @bufbuild/protobuf, which only resolves when a node_modules
// providing that package is linked into scope, so they cannot be compiled by a
// bare whole-tree pass. Each TS generator covers them with a dedicated es
// typecheck test that links node_modules first (and skips when the install is
// absent). Matching by pattern rather than by name keeps new es-* variants
// covered without touching this list.
//
// The tradeoff of the "es-*" glob is that a future golden dir whose name merely
// starts with "es-" but has nothing to do with protobuf-es would be silently
// dropped from the whole-tree typecheck. Name such a directory something else,
// or switch to an explicit list if that ever stops being the rarer case.
func ESGoldenDirs() []string {
	return []string{"es", "es-*"}
}

// Dir typechecks every .ts file under dir with tsc --noEmit. The test is
// skipped when no TypeScript toolchain is available (neither tsc nor npx on
// PATH); any compile error fails the test with the compiler output.
func Dir(t *testing.T, dir string) {
	t.Helper()
	DirExcluding(t, dir)
}

// DirExcluding behaves like Dir but drops every .ts file under the given
// subdirectory patterns (relative to dir, tsconfig glob syntax) from the
// compilation. Use it when part of a tree needs a compilation scope the shared
// tsconfig cannot provide — e.g. third-party module resolution — and is
// typechecked by its own test instead.
func DirExcluding(t *testing.T, dir string, excludeDirs ...string) {
	t.Helper()

	tsc := tscCommand(t)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve %s: %v", dir, err)
	}

	// tsconfig exclude entries are matched against the include glob, so each
	// excluded subdirectory is spelled out as everything beneath it.
	excludes := make([]string, 0, len(excludeDirs))
	for _, excludeDir := range excludeDirs {
		excludes = append(excludes, filepath.ToSlash(absDir)+"/"+excludeDir+"/**/*")
	}
	excludeJSON, err := json.Marshal(excludes)
	if err != nil {
		t.Fatalf("failed to encode tsconfig excludes: %v", err)
	}

	// noUnusedLocals is deliberate: a generated module importing a symbol it
	// never uses is a generator bug (and breaks consumers with strict configs).
	config := fmt.Sprintf(`{
  "compilerOptions": {
    "module": "nodenext",
    "moduleResolution": "nodenext",
    "target": "es2020",
    "lib": ["es2020", "dom", "dom.iterable", "dom.asynciterable"],
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "noUnusedLocals": true
  },
  "include": [%q],
  "exclude": %s
}
`, filepath.ToSlash(absDir)+"/**/*.ts", excludeJSON)

	tsconfigPath := filepath.Join(t.TempDir(), "tsconfig.json")
	if writeErr := os.WriteFile(tsconfigPath, []byte(config), 0o600); writeErr != nil {
		t.Fatalf("failed to write tsconfig: %v", writeErr)
	}

	args := make([]string, 0, len(tsc)+1)
	args = append(args, tsc[1:]...)
	args = append(args, "-p", tsconfigPath)
	//nolint:gosec // test-only helper invoking the compiler found on PATH
	cmd := exec.CommandContext(context.Background(), tsc[0], args...)
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Errorf("tsc --noEmit failed for %s: %v\n%s", dir, runErr, out)
	}
}

// tscCommand returns the command (argv prefix) that invokes the TypeScript
// compiler: tsc from PATH when installed, otherwise a pinned compiler via
// npx. Skips the test when neither is available.
func tscCommand(t *testing.T) []string {
	t.Helper()
	if path, err := exec.LookPath("tsc"); err == nil {
		return []string{path}
	}
	if npx, err := exec.LookPath("npx"); err == nil {
		return []string{npx, "--yes", "--package=typescript@" + tsVersion, "tsc"}
	}
	t.Skip("neither tsc nor npx found on PATH, skipping TypeScript typecheck")
	return nil
}
