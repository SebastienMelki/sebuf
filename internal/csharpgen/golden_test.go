package csharpgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCSharpGenGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden")
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-csharp-http")

	if _, statErr := os.Stat(pluginPath); os.IsNotExist(statErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	tempDir := t.TempDir()
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-csharp-http="+pluginPath,
		"--csharp-http_out="+tempDir,
		"--csharp-http_opt=namespace=Test.Contracts,json_lib=newtonsoft",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"contracts.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	expectedFile := "test/contracts/v1/Contracts.g.cs"
	generatedPath := filepath.Join(tempDir, expectedFile)
	goldenPath := filepath.Join(goldenDir, filepath.Base(expectedFile))
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
			expectedFile,
			diffStrings(string(goldenContent), string(generatedContent)),
		)
	}
}

func diffStrings(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	var diff strings.Builder
	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}
	for i := range maxLines {
		var expectedLine, actualLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}
		if expectedLine != actualLine {
			diff.WriteString("Line ")
			diff.WriteString(strconv.Itoa(i + 1))
			diff.WriteString(":\n  expected: ")
			diff.WriteString(expectedLine)
			diff.WriteString("\n  actual:   ")
			diff.WriteString(actualLine)
			diff.WriteString("\n")
		}
	}
	return diff.String()
}
