// Package typecheck runs the TypeScript compiler over generated output in
// tests, verifying that the emitted module tree typechecks under the strict
// nodenext settings the modules layout targets. Byte-comparing golden files
// catches regressions in what we emit; this catches emitting something that
// was never valid TypeScript in the first place (duplicate identifiers,
// shadowed globals, unused imports, wrong casts).
package typecheck

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// tsVersion pins the compiler fetched by the npx fallback so results don't
// drift with whatever "latest" is on the machine running the tests.
const tsVersion = "5.9.3"

// Dir typechecks every .ts file under dir with tsc --noEmit. The test is
// skipped when no TypeScript toolchain is available (neither tsc nor npx on
// PATH); any compile error fails the test with the compiler output.
func Dir(t *testing.T, dir string) {
	t.Helper()

	tsc := tscCommand(t)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve %s: %v", dir, err)
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
  "include": [%q]
}
`, filepath.ToSlash(absDir)+"/**/*.ts")

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
