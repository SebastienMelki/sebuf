package csharpgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/testutil"
)

func TestCSharpGenGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}

	testCases := []struct {
		name         string
		protoFiles   []string
		opt          string
		expectedFile string
	}{
		{
			name:         "simple contracts newtonsoft",
			protoFiles:   []string{"contracts.proto"},
			opt:          "namespace=Test.Contracts,json_lib=newtonsoft",
			expectedFile: "Contracts.g.cs",
		},
		{
			name:         "comprehensive contracts newtonsoft",
			protoFiles:   []string{"comprehensive_models.proto", "comprehensive_services.proto"},
			opt:          "namespace=Test.Contracts,json_lib=newtonsoft",
			expectedFile: "Comprehensive.Newtonsoft.g.cs",
		},
		{
			name:         "comprehensive contracts system text json",
			protoFiles:   []string{"comprehensive_models.proto", "comprehensive_services.proto"},
			opt:          "namespace=Test.Contracts,json_lib=system_text_json",
			expectedFile: "Comprehensive.SystemTextJson.g.cs",
		},
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden")
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-csharp-http")

	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = projectRoot
	if buildErr := buildCmd.Run(); buildErr != nil {
		t.Fatalf("Failed to build plugin: %v", buildErr)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			args := []string{
				"--plugin=protoc-gen-csharp-http=" + pluginPath,
				"--csharp-http_out=" + tempDir,
				"--csharp-http_opt=" + tc.opt,
				"--proto_path=" + protoDir,
				"--proto_path=" + filepath.Join(projectRoot, "proto"),
			}
			args = append(args, tc.protoFiles...)

			cmd := exec.Command("protoc", args...)
			cmd.Dir = protoDir

			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if runErr := cmd.Run(); runErr != nil {
				t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
			}

			generatedPath := filepath.Join(tempDir, "test/contracts/v1/Contracts.g.cs")
			goldenPath := filepath.Join(goldenDir, tc.expectedFile)
			generatedContent, readErr := os.ReadFile(generatedPath)
			if readErr != nil {
				t.Fatalf("Failed to read generated file %s: %v", generatedPath, readErr)
			}

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if writeErr := os.WriteFile(goldenPath, generatedContent, 0o644); writeErr != nil {
					t.Fatalf("Failed to write golden file %s: %v", goldenPath, writeErr)
				}
				return
			}

			goldenContent, goldenErr := os.ReadFile(goldenPath)
			if goldenErr != nil {
				t.Fatalf("Failed to read golden file %s: %v", goldenPath, goldenErr)
			}
			if !bytes.Equal(generatedContent, goldenContent) {
				t.Fatalf(
					"Generated file %s does not match golden file.\nDiff:\n%s",
					tc.expectedFile,
					testutil.DiffStrings(string(goldenContent), string(generatedContent)),
				)
			}
		})
	}
}
