// Package plugintest builds protoc plugin binaries for subprocess golden
// tests. Building into a per-test temporary directory — instead of reusing a
// prebuilt bin/ binary — keeps the tests hermetic: a stale prebuilt binary
// silently runs old generator code and reports spurious passes or failures.
package plugintest

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// Build compiles cmd/<name> from the repository root into a temporary
// directory and returns the resulting binary path. The Go build cache makes
// repeat builds cheap, so tests always exercise the current generator code.
func Build(t *testing.T, projectRoot, name string) string {
	t.Helper()

	binName := name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), binName)

	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", binPath, "./cmd/"+name)
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build %s: %v\n%s", name, err, out)
	}
	return binPath
}
