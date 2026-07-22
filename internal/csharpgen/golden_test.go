package csharpgen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/tscommon/plugintest"
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
	pluginPath := plugintest.Build(t, projectRoot, "protoc-gen-csharp-http")

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
					diffStrings(string(goldenContent), string(generatedContent)),
				)
			}
		})
	}
}

func diffStrings(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	maxLines := max(len(expectedLines), len(actualLines))

	var diff strings.Builder
	diffCount := 0
	const maxDiffs = 20
	for i := 0; i < maxLines && diffCount < maxDiffs; i++ {
		var expectedLine, actualLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}
		if expectedLine == actualLine {
			continue
		}
		fmt.Fprintf(
			&diff,
			"Line %d:\n  expected: %s\n  actual:   %s\n",
			i+1,
			expectedLine,
			actualLine,
		)
		diffCount++
	}
	if diffCount == maxDiffs {
		diff.WriteString("... (more differences truncated)\n")
	}
	return diff.String()
}
