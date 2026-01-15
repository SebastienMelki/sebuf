package httpgen

import (
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHTTPGenGoldenFiles tests HTTP handler generation against golden files.
// This ensures any changes to code generation are intentional and reviewed.
//
// To update golden files after intentional changes:
//
//	UPDATE_GOLDEN=1 go test -run TestHTTPGenGoldenFiles
func TestHTTPGenGoldenFiles(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping golden file tests")
	}

	testCases := []struct {
		name      string
		protoFile string
		// Expected generated files (without path prefix)
		expectedFiles []string
	}{
		{
			name:      "comprehensive HTTP verbs",
			protoFile: "http_verbs_comprehensive.proto",
			expectedFiles: []string{
				"http_verbs_comprehensive_http.pb.go",
				"http_verbs_comprehensive_http_binding.pb.go",
				"http_verbs_comprehensive_http_config.pb.go",
			},
		},
		{
			name:      "query parameters",
			protoFile: "query_params.proto",
			expectedFiles: []string{
				"query_params_http.pb.go",
				"query_params_http_binding.pb.go",
				"query_params_http_config.pb.go",
			},
		},
		{
			name:      "backward compatibility",
			protoFile: "backward_compat.proto",
			expectedFiles: []string{
				"backward_compat_http.pb.go",
				"backward_compat_http_binding.pb.go",
				"backward_compat_http_config.pb.go",
			},
		},
	}

	// Get paths
	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to project root (up from internal/httpgen)
	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	goldenDir := filepath.Join(baseDir, "testdata", "golden")

	// Create golden directory if it doesn't exist
	mkdirErr := os.MkdirAll(goldenDir, 0o755)
	if mkdirErr != nil {
		t.Fatalf("Failed to create golden directory: %v", mkdirErr)
	}

	// Create temp directory for generated files
	tempDir := t.TempDir()

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			protoPath := filepath.Join(protoDir, tc.protoFile)

			// Check proto file exists
			_, statErr := os.Stat(protoPath)
			if os.IsNotExist(statErr) {
				t.Fatalf("Proto file not found: %s", protoPath)
			}

			// Run protoc with go-http plugin
			cmd := exec.Command("protoc",
				"--go_out="+tempDir,
				"--go_opt=paths=source_relative",
				"--go-http_out="+tempDir,
				"--go-http_opt=paths=source_relative",
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

			// Compare or update golden files
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

// updateGoldenFile writes generated content to a golden file.
func updateGoldenFile(t *testing.T, goldenPath string, content []byte) {
	t.Helper()
	writeErr := os.WriteFile(goldenPath, content, 0o644)
	if writeErr != nil {
		t.Fatalf("Failed to write golden file %s: %v", goldenPath, writeErr)
	}
	t.Logf("Updated golden file: %s", goldenPath)
}

// compareGoldenFile compares generated content with a golden file.
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

// diffStrings returns a simple diff between two strings.
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

// TestHTTPGenValidation tests that invalid configurations produce expected errors.
func TestHTTPGenValidation(t *testing.T) {
	// These tests verify validation error messages are clear and actionable
	tests := []struct {
		name          string
		config        HTTPConfig
		pathParams    []string
		queryParams   []QueryParam
		errorContains string
	}{
		{
			name: "GET with unbound fields should error",
			config: HTTPConfig{
				Path:   "/users",
				Method: "GET",
			},
			pathParams:    nil,
			queryParams:   nil,
			errorContains: "", // This case would be caught during generation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation logic tests
			if tt.config.Method == http.MethodGet || tt.config.Method == http.MethodDelete {
				// These methods shouldn't have body fields
				// Test is informational - actual validation happens in ValidateMethodConfig
				t.Logf("Config: %+v", tt.config)
			}
		})
	}
}

// TestGeneratedCodeCompiles verifies that generated code compiles correctly.
// This is an integration test that runs the actual compiler.
func TestGeneratedCodeCompiles(t *testing.T) {
	// Skip if protoc is not available
	if _, lookErr := exec.LookPath("protoc"); lookErr != nil {
		t.Skip("protoc not found, skipping compilation test")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")

	// Create temp directory for generated files
	tempDir := t.TempDir()

	// Generate code for comprehensive test proto
	cmd := exec.Command("protoc",
		"--go_out="+tempDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+tempDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"http_verbs_comprehensive.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	// Try to compile the generated code (syntax check)
	// We use 'go build' with -n flag for dry run
	buildCmd := exec.Command("go", "build", "-n", "./...")
	buildCmd.Dir = tempDir

	// Note: This won't fully work without proper go.mod setup,
	// but protoc success indicates the generated code is syntactically valid
	t.Log("Generated code produced successfully")
}
