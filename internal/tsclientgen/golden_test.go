package tsclientgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTSClientGenGoldenFiles tests TypeScript client generation against golden files.
// This ensures any changes to code generation are intentional and reviewed.
//
// To update golden files after intentional changes:
//
//	UPDATE_GOLDEN=1 go test -run TestTSClientGenGoldenFiles
func TestTSClientGenGoldenFiles(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}

	testCases := []struct {
		name          string
		protoFile     string
		expectedFiles []string
	}{
		{
			name:      "comprehensive HTTP verbs",
			protoFile: "http_verbs_comprehensive.proto",
			expectedFiles: []string{
				"http_verbs_comprehensive_client.ts",
			},
		},
		{
			name:      "query parameters",
			protoFile: "query_params.proto",
			expectedFiles: []string{
				"query_params_client.ts",
			},
		},
		{
			name:      "backward compatibility",
			protoFile: "backward_compat.proto",
			expectedFiles: []string{
				"backward_compat_client.ts",
			},
		},
		{
			name:      "complex features",
			protoFile: "complex_features.proto",
			expectedFiles: []string{
				"complex_features_client.ts",
			},
		},
		{
			name:      "unwrap variants",
			protoFile: "unwrap.proto",
			expectedFiles: []string{
				"unwrap_client.ts",
			},
		},
		{
			name:      "int64 encoding",
			protoFile: "int64_encoding.proto",
			expectedFiles: []string{
				"int64_encoding_client.ts",
			},
		},
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden")

	mkdirErr := os.MkdirAll(goldenDir, 0o755)
	if mkdirErr != nil {
		t.Fatalf("Failed to create golden directory: %v", mkdirErr)
	}

	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-ts-client")

	// Build the plugin if it doesn't exist
	if _, buildStatErr := os.Stat(pluginPath); os.IsNotExist(buildStatErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	tempDir := t.TempDir()

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			protoPath := filepath.Join(protoDir, tc.protoFile)

			_, statErr := os.Stat(protoPath)
			if os.IsNotExist(statErr) {
				t.Fatalf("Proto file not found: %s", protoPath)
			}

			// Run protoc with ts-client plugin
			cmd := exec.Command("protoc",
				"--plugin=protoc-gen-ts-client="+pluginPath,
				"--ts-client_out="+tempDir,
				"--ts-client_opt=paths=source_relative",
				"--proto_path="+protoDir,
				"--proto_path="+filepath.Join(projectRoot, "proto"),
				tc.protoFile,
			)
			cmd.Dir = protoDir

			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			runErr := cmd.Run()
			if runErr != nil {
				t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
			}

			for _, expectedFile := range tc.expectedFiles {
				generatedPath := filepath.Join(tempDir, expectedFile)
				goldenPath := filepath.Join(goldenDir, expectedFile)

				generatedContent, readErr := os.ReadFile(generatedPath)
				if readErr != nil {
					t.Fatalf("Failed to read generated file %s: %v", generatedPath, readErr)
				}

				if updateGolden {
					updateGoldenFile(t, goldenPath, generatedContent)
					continue
				}
				compareGoldenFile(t, expectedFile, goldenPath, generatedContent)
			}
		})
	}
}

func updateGoldenFile(t *testing.T, goldenPath string, content []byte) {
	t.Helper()
	writeErr := os.WriteFile(goldenPath, content, 0o644)
	if writeErr != nil {
		t.Fatalf("Failed to write golden file %s: %v", goldenPath, writeErr)
	}
	t.Logf("Updated golden file: %s", goldenPath)
}

func compareGoldenFile(t *testing.T, expectedFile, goldenPath string, generatedContent []byte) {
	t.Helper()
	goldenContent, goldenReadErr := os.ReadFile(goldenPath)
	if goldenReadErr != nil {
		if os.IsNotExist(goldenReadErr) {
			t.Fatalf("Golden file not found: %s\nRun with UPDATE_GOLDEN=1 to create it", goldenPath)
		}
		t.Fatalf("Failed to read golden file %s: %v", goldenPath, goldenReadErr)
	}

	if !bytes.Equal(generatedContent, goldenContent) {
		t.Errorf("Generated file %s does not match golden file.\n"+
			"Run with UPDATE_GOLDEN=1 to update golden files after reviewing changes.\n"+
			"Diff:\n%s",
			expectedFile,
			diffStrings(string(goldenContent), string(generatedContent)))
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

	diffCount := 0
	const maxDiffs = 20

	for i := 0; i < maxLines && diffCount < maxDiffs; i++ {
		var expLine, actLine string
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actLine = actualLines[i]
		}

		if expLine != actLine {
			diff.WriteString("Line ")
			diff.WriteRune(rune('0' + i/100))
			diff.WriteRune(rune('0' + (i/10)%10))
			diff.WriteRune(rune('0' + i%10))
			diff.WriteString(":\n")
			diff.WriteString("  expected: ")
			diff.WriteString(expLine)
			diff.WriteString("\n  actual:   ")
			diff.WriteString(actLine)
			diff.WriteString("\n")
			diffCount++
		}
	}

	if diffCount >= maxDiffs {
		diff.WriteString("... (more differences truncated)\n")
	}

	return diff.String()
}
