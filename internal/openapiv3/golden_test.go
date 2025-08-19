package openapiv3_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenFiles tests the OpenAPI generator against golden files
func TestGoldenFiles(t *testing.T) {
	// Build the plugin binary
	pluginPath := buildPlugin(t)

	testCases := []struct {
		name        string
		protoFile   string
		format      string // "yaml" or "json"
		goldenFile  string
	}{
		{
			name:       "simple types YAML",
			protoFile:  "simple_types.proto",
			format:     "yaml",
			goldenFile: "simple_types.yaml",
		},
		{
			name:       "simple types JSON",
			protoFile:  "simple_types.proto",
			format:     "json",
			goldenFile: "simple_types.json",
		},
		{
			name:       "validation YAML",
			protoFile:  "validation.proto",
			format:     "yaml",
			goldenFile: "validation.yaml",
		},
		{
			name:       "validation JSON",
			protoFile:  "validation.proto",
			format:     "json",
			goldenFile: "validation.json",
		},
		{
			name:       "headers YAML",
			protoFile:  "headers.proto",
			format:     "yaml",
			goldenFile: "headers.yaml",
		},
		{
			name:       "headers JSON",
			protoFile:  "headers.proto",
			format:     "json",
			goldenFile: "headers.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp directory for output
			tempDir := t.TempDir()

			// Run protoc with our plugin
			protoPath := filepath.Join("testdata", "proto", tc.protoFile)
			generated := runProtoc(t, pluginPath, protoPath, tempDir, tc.format)

			// Read golden file
			goldenPath := filepath.Join("testdata", "golden", tc.goldenFile)
			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				if os.IsNotExist(err) && os.Getenv("UPDATE_GOLDEN") == "1" {
					// Create golden file if it doesn't exist and UPDATE_GOLDEN is set
					err = os.MkdirAll(filepath.Dir(goldenPath), 0755)
					require.NoError(t, err)
					err = os.WriteFile(goldenPath, generated, 0644)
					require.NoError(t, err)
					t.Logf("Created golden file: %s", goldenPath)
					return
				}
				t.Fatalf("Failed to read golden file: %v", err)
			}

			// Compare output
			if !bytes.Equal(generated, golden) {
				reportGoldenFileMismatch(t, tc.name, goldenPath, generated, golden)
			}
		})
	}
}

// buildPlugin builds the protoc-gen-openapiv3 plugin
func buildPlugin(t *testing.T) string {
	t.Helper()

	// Build plugin in temp directory
	tempDir := t.TempDir()
	pluginPath := filepath.Join(tempDir, "protoc-gen-openapiv3")

	cmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build plugin: %v\nOutput: %s", err, output)
	}

	return pluginPath
}

// runProtoc runs protoc with the OpenAPI generator plugin
func runProtoc(t *testing.T, pluginPath, protoPath, outputDir, format string) []byte {
	t.Helper()

	// Determine expected output filename
	outputFile := "openapi.yaml"
	if format == "json" {
		outputFile = "openapi.json"
	}

	// Build protoc command
	args := []string{
		"--plugin=protoc-gen-openapiv3=" + pluginPath,
		"--openapiv3_out=" + outputDir,
		"--openapiv3_opt=format=" + format,
		"--proto_path=testdata/proto",
		"--proto_path=../../proto", // For sebuf/http imports
		"--proto_path=.",            // For buf/validate imports (assuming it's available)
		protoPath,
	}

	// Check if buf/validate proto files are available
	validatePath := findBufValidateProtos(t)
	if validatePath != "" {
		args = append(args, "--proto_path="+validatePath)
	}

	cmd := exec.Command("protoc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("protoc failed: %v\nOutput: %s\nCommand: protoc %s", err, output, strings.Join(args, " "))
	}

	// Read generated file
	generatedPath := filepath.Join(outputDir, outputFile)
	generated, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	return generated
}

// findBufValidateProtos attempts to find buf/validate proto files
func findBufValidateProtos(t *testing.T) string {
	t.Helper()

	// Common locations to check
	possiblePaths := []string{
		// Go module cache
		filepath.Join(os.Getenv("GOPATH"), "pkg", "mod", "buf.build", "gen", "go", "bufbuild", "protovalidate"),
		// Local vendor directory
		"vendor/buf.build/gen/go/bufbuild/protovalidate",
		// buf cache directory
		filepath.Join(os.Getenv("HOME"), ".cache", "buf", "v1", "module", "data"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try to find it using go list
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "buf.build/gen/go/bufbuild/protovalidate")
	output, err := cmd.Output()
	if err == nil {
		dir := strings.TrimSpace(string(output))
		if dir != "" {
			// The proto files are typically in a subdirectory
			protoDir := filepath.Join(dir, "protocolbuffers", "go")
			if _, err := os.Stat(protoDir); err == nil {
				return protoDir
			}
		}
	}

	t.Log("Warning: Could not find buf/validate proto files. Tests may fail if they use validation.")
	return ""
}

// reportGoldenFileMismatch reports differences between generated and golden files
func reportGoldenFileMismatch(t *testing.T, testName, goldenFile string, generatedContent, goldenContent []byte) {
	t.Helper()

	generated := string(generatedContent)
	golden := string(goldenContent)

	t.Errorf("Generated output does not match golden file for %s", testName)
	t.Errorf("Generated file size: %d bytes", len(generatedContent))
	t.Errorf("Golden file size: %d bytes", len(goldenContent))

	// Report first difference
	reportFirstDifference(t, generated, golden)

	// Report line differences
	reportLineDifferences(t, generated, golden)

	// Handle golden file update
	handleGoldenFileUpdate(t, goldenFile, generatedContent)

	// Write temporary generated file for inspection
	writeTemporaryGeneratedFile(t, goldenFile, generatedContent)
}

func reportFirstDifference(t *testing.T, generated, golden string) {
	t.Helper()

	minLen := len(generated)
	if len(golden) < minLen {
		minLen = len(golden)
	}

	for i := 0; i < minLen; i++ {
		if generated[i] != golden[i] {
			t.Errorf("First difference at byte position %d", i)
			t.Errorf("Generated byte: %d (%c)", generated[i], generated[i])
			t.Errorf("Golden byte: %d (%c)", golden[i], golden[i])
			
			// Show context around the difference
			start := i - 20
			if start < 0 {
				start = 0
			}
			end := i + 20
			if end > minLen {
				end = minLen
			}
			
			t.Errorf("Context (generated): %q", generated[start:end])
			t.Errorf("Context (golden): %q", golden[start:end])
			return
		}
	}

	if len(generated) != len(golden) {
		t.Errorf("Files have same prefix but different lengths")
	}
}

func reportLineDifferences(t *testing.T, generated, golden string) {
	t.Helper()

	generatedLines := strings.Split(generated, "\n")
	goldenLines := strings.Split(golden, "\n")

	maxLines := len(generatedLines)
	if len(goldenLines) > maxLines {
		maxLines = len(goldenLines)
	}

	diffCount := 0
	maxDiffs := 10

	for i := 0; i < maxLines && diffCount < maxDiffs; i++ {
		var generatedLine, goldenLine string
		if i < len(generatedLines) {
			generatedLine = generatedLines[i]
		}
		if i < len(goldenLines) {
			goldenLine = goldenLines[i]
		}

		if generatedLine != goldenLine {
			t.Errorf("Line %d differs:", i+1)
			t.Errorf("  Generated: %q", generatedLine)
			t.Errorf("  Golden:    %q", goldenLine)
			diffCount++
		}
	}

	if diffCount >= maxDiffs {
		t.Errorf("... (showing first %d differences only)", maxDiffs)
	}

	t.Errorf("Total lines - Generated: %d, Golden: %d", len(generatedLines), len(goldenLines))
}

func handleGoldenFileUpdate(t *testing.T, goldenFile string, generatedContent []byte) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenFile, generatedContent, 0644); err != nil {
			t.Logf("Failed to update golden file: %v", err)
		} else {
			t.Logf("Updated golden file: %s", goldenFile)
		}
	} else {
		t.Log("To update golden files, run: UPDATE_GOLDEN=1 go test -run TestGoldenFiles")
	}
}

func writeTemporaryGeneratedFile(t *testing.T, goldenFile string, generatedContent []byte) {
	t.Helper()

	tempGenFile := goldenFile + ".generated"
	if err := os.WriteFile(tempGenFile, generatedContent, 0644); err == nil {
		t.Logf("Generated content written to: %s", tempGenFile)
		t.Logf("To compare: diff %s %s", goldenFile, tempGenFile)
	}
}

// TestExhaustiveGoldenFiles runs comprehensive tests with all proto features
func TestExhaustiveGoldenFiles(t *testing.T) {
	// Skip if not in exhaustive mode
	if os.Getenv("EXHAUSTIVE_TEST") != "1" && testing.Short() {
		t.Skip("Skipping exhaustive tests. Set EXHAUSTIVE_TEST=1 to run.")
	}

	// This test would include more complex scenarios
	// For now, it runs the same tests as TestGoldenFiles
	TestGoldenFiles(t)
}

// TestGeneratorOutput validates that the generator produces valid OpenAPI documents
func TestGeneratorOutput(t *testing.T) {
	pluginPath := buildPlugin(t)
	tempDir := t.TempDir()

	testCases := []struct {
		name      string
		protoFile string
		format    string
		validate  func(t *testing.T, content []byte)
	}{
		{
			name:      "YAML format validity",
			protoFile: "simple_types.proto",
			format:    "yaml",
			validate: func(t *testing.T, content []byte) {
				// Check YAML starts correctly
				assert.True(t, bytes.HasPrefix(content, []byte("openapi:")), "YAML should start with 'openapi:'")
				assert.Contains(t, string(content), "openapi: 3.1.0", "Should have OpenAPI version")
				assert.Contains(t, string(content), "info:", "Should have info section")
				assert.Contains(t, string(content), "paths:", "Should have paths section")
			},
		},
		{
			name:      "JSON format validity",
			protoFile: "simple_types.proto",
			format:    "json",
			validate: func(t *testing.T, content []byte) {
				// Check JSON structure
				assert.True(t, bytes.HasPrefix(bytes.TrimSpace(content), []byte("{")), "JSON should start with '{'")
				assert.True(t, bytes.HasSuffix(bytes.TrimSpace(content), []byte("}")), "JSON should end with '}'")
				assert.Contains(t, string(content), `"openapi":"3.1.0"`, "Should have OpenAPI version")
				assert.Contains(t, string(content), `"info":`, "Should have info section")
				assert.Contains(t, string(content), `"paths":`, "Should have paths section")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			protoPath := filepath.Join("testdata", "proto", tc.protoFile)
			generated := runProtoc(t, pluginPath, protoPath, tempDir, tc.format)
			tc.validate(t, generated)
		})
	}
}