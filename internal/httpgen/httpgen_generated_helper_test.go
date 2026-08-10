package httpgen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func readGeneratedJSONMappingGoldenFixture(t *testing.T, baseDir, fixtureBase string, extraProtoFiles ...string) string {
	t.Helper()

	protoFiles := append([]string{fixtureBase + ".proto"}, extraProtoFiles...)
	genDir := generateHTTPGenGoldenFixture(t, baseDir, protoFiles...)
	return readGeneratedHTTPGenFile(t, filepath.Join(genDir, fixtureBase+"_json_mapping.pb.go"))
}

func assertHTTPGenFixtureDoesNotGenerate(t *testing.T, baseDir, forbiddenFile string, protoFiles ...string) {
	t.Helper()

	genDir := generateHTTPGenGoldenFixture(t, baseDir, protoFiles...)
	if _, statErr := os.Stat(filepath.Join(genDir, forbiddenFile)); statErr == nil {
		t.Fatalf("%s should not be generated", forbiddenFile)
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("stat generated file %s: %v", forbiddenFile, statErr)
	}
}

func generateHTTPGenGoldenFixture(t *testing.T, baseDir string, protoFiles ...string) string {
	t.Helper()

	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping generated fixture tests")
	}
	if _, err := exec.LookPath("protoc-gen-go"); err != nil {
		t.Skip("protoc-gen-go not found, skipping generated fixture tests")
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	genDir := t.TempDir()
	pluginPath := buildHTTPGenPluginForFixture(t, projectRoot, t.TempDir())

	args := []string{
		"--plugin=protoc-gen-go-http=" + pluginPath,
		"--go_out=" + genDir,
		"--go_opt=paths=source_relative",
		"--go-http_out=" + genDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path=" + protoDir,
		"--proto_path=" + filepath.Join(projectRoot, "proto"),
	}
	args = append(args, protoFiles...)

	cmd := exec.Command("protoc", args...)
	cmd.Dir = protoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("protoc generated fixture failed for %v: %v\n%s", protoFiles, err, out)
	}

	return genDir
}

func readGeneratedHTTPGenFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated HTTP file %s: %v", path, err)
	}
	return string(content)
}

func buildHTTPGenPluginForFixture(t *testing.T, projectRoot, tempDir string) string {
	t.Helper()

	pluginPath := filepath.Join(tempDir, "protoc-gen-go-http")
	cmd := exec.Command("go", "build", "-o", pluginPath, "./cmd/protoc-gen-go-http")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build protoc-gen-go-http: %v\n%s", err, out)
	}
	return pluginPath
}

func buildGeneratedHTTPGenGoldenFixture(t *testing.T, baseDir, modulePath string, protoFiles ...string) {
	t.Helper()

	genDir := generateHTTPGenGoldenFixture(t, baseDir, protoFiles...)
	projectRoot := filepath.Join(baseDir, "..", "..")
	goMod := fmt.Sprintf(`module %s

go 1.26.0

require (
	github.com/SebastienMelki/sebuf v0.0.0
	google.golang.org/protobuf %s
)

replace github.com/SebastienMelki/sebuf => %s
`, modulePath, extractProtobufVersionFromModFile(t, projectRoot), projectRoot)
	if err := os.WriteFile(filepath.Join(genDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write generated fixture go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = genDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy generated fixture: %v\n%s", err, out)
	}

	buildCmd := exec.Command("go", "test", "./...", "-count=1")
	buildCmd.Dir = genDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("generated fixture go test failed: %v\n%s", err, out)
	}
}
