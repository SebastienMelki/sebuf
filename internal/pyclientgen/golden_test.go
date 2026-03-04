package pyclientgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/testutil"
)

func TestPyClientGenGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = projectRoot
	if out, buildErr := buildCmd.CombinedOutput(); buildErr != nil {
		t.Fatalf("failed to build plugin: %v\n%s", buildErr, out)
	}

	tests := []struct {
		name          string
		protoDir      string
		inputs        []string
		expectedFiles map[string]string
	}{
		{
			name:     "minimal",
			protoDir: filepath.Join(baseDir, "testdata", "proto"),
			inputs:   []string{"contracts.proto"},
			expectedFiles: map[string]string{
				"test/contracts/v1/__init__.py":  "__init__.py",
				"test/contracts/v1/contracts.py": "contracts.py",
			},
		},
		{
			name:     "comprehensive",
			protoDir: filepath.Join(baseDir, "testdata", "proto"),
			inputs:   []string{"comprehensive_models.proto", "comprehensive_services.proto"},
			expectedFiles: map[string]string{
				"test/contracts/v1/__init__.py":  "comprehensive.__init__.py",
				"test/contracts/v1/contracts.py": "comprehensive.contracts.py",
			},
		},
	}

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-py-client")
	goldenDir := filepath.Join(baseDir, "testdata", "golden")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			args := []string{
				"--plugin=protoc-gen-py-client=" + pluginPath,
				"--py-client_out=" + tempDir,
				"--proto_path=" + tt.protoDir,
				"--proto_path=" + filepath.Join(projectRoot, "proto"),
			}
			args = append(args, tt.inputs...)

			cmd := exec.Command("protoc", args...)
			cmd.Dir = tt.protoDir

			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if runErr := cmd.Run(); runErr != nil {
				t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
			}

			for generatedFile, goldenName := range tt.expectedFiles {
				generatedPath := filepath.Join(tempDir, generatedFile)
				goldenPath := filepath.Join(goldenDir, goldenName)

				generatedContent, readErr := os.ReadFile(generatedPath)
				if readErr != nil {
					t.Fatalf("failed to read generated file %s: %v", generatedPath, readErr)
				}

				if updateGolden {
					if writeErr := os.WriteFile(goldenPath, generatedContent, 0o644); writeErr != nil {
						t.Fatalf("failed to write golden file %s: %v", goldenPath, writeErr)
					}
					continue
				}

				goldenContent, goldenErr := os.ReadFile(goldenPath)
				if goldenErr != nil {
					t.Fatalf("failed to read golden file %s: %v", goldenPath, goldenErr)
				}
				if !bytes.Equal(generatedContent, goldenContent) {
					t.Fatalf(
						"generated file %s does not match golden file.\nDiff:\n%s",
						generatedFile,
						testutil.DiffStrings(string(goldenContent), string(generatedContent)),
					)
				}
			}
		})
	}
}
