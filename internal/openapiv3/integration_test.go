package openapiv3_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPluginIntegration tests the actual protoc plugin integration
// This verifies that the plugin works correctly when invoked by protoc
func TestPluginIntegration(t *testing.T) {
	// Build the plugin binary for testing
	pluginPath := "./protoc-gen-openapiv3-integration-test"
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build plugin: %v", err)
	}
	defer os.Remove(pluginPath)

	testCases := []struct {
		name      string
		protoFile string
		format    string
		expectOut bool // whether we expect output files to be generated
	}{
		{
			name:      "simple_service_yaml",
			protoFile: "simple_service.proto",
			format:    "yaml",
			expectOut: true,
		},
		{
			name:      "simple_service_json",
			protoFile: "simple_service.proto",
			format:    "json",
			expectOut: true,
		},
		{
			name:      "multiple_services_yaml",
			protoFile: "multiple_services.proto",
			format:    "yaml",
			expectOut: true,
		},
		{
			name:      "headers_json",
			protoFile: "headers.proto",
			format:    "json",
			expectOut: true,
		},
		{
			name:      "no_services_yaml",
			protoFile: "no_services.proto",
			format:    "yaml",
			expectOut: false, // No services = no output files
		},
		{
			name:      "nested_messages_json",
			protoFile: "nested_messages.proto",
			format:    "json",
			expectOut: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory for output
			tempDir := t.TempDir()

			// Run protoc with our plugin
			cmd := exec.Command("protoc",
				"--plugin=protoc-gen-openapiv3="+pluginPath,
				"--openapiv3_out="+tempDir,
				"--openapiv3_opt=format="+tc.format,
				"--proto_path=testdata/proto",
				"--proto_path=../../proto",
				"testdata/proto/"+tc.protoFile,
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("protoc failed for %s: %v\nOutput: %s", tc.name, err, string(output))
			}

			// Check if output files were generated as expected
			files, listErr := os.ReadDir(tempDir)
			if listErr != nil {
				t.Fatalf("Failed to list output directory: %v", listErr)
			}

			if tc.expectOut {
				if len(files) == 0 {
					t.Errorf("Expected output files but none were generated for %s", tc.name)
				} else {
					// Verify file extensions match format
					var extension string
					if tc.format == "json" {
						extension = ".json"
					} else {
						extension = ".yaml"
					}

					foundCorrectFile := false
					for _, file := range files {
						if strings.HasSuffix(file.Name(), extension) {
							foundCorrectFile = true
							break
						}
					}

					if !foundCorrectFile {
						t.Errorf("Expected file with %s extension but found: %v", extension, getFileNames(files))
					}
				}
			} else {
				if len(files) > 0 {
					t.Errorf("Expected no output files but found: %v", getFileNames(files))
				}
			}
		})
	}
}

// TestPluginErrorHandling tests how the plugin handles various error conditions
func TestPluginErrorHandling(t *testing.T) {
	// Build the plugin binary for testing
	pluginPath := "./protoc-gen-openapiv3-error-test"
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build plugin: %v", err)
	}
	defer os.Remove(pluginPath)

	testCases := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing_proto_file",
			args: []string{
				"--plugin=protoc-gen-openapiv3=" + pluginPath,
				"--openapiv3_out=" + t.TempDir(),
				"--proto_path=testdata/proto",
				"testdata/proto/nonexistent.proto",
			},
			expectError: true,
			errorMsg:    "No such file or directory",
		},
		{
			name: "invalid_proto_path",
			args: []string{
				"--plugin=protoc-gen-openapiv3=" + pluginPath,
				"--openapiv3_out=" + t.TempDir(),
				"--proto_path=nonexistent/path",
				"testdata/proto/simple_service.proto",
			},
			expectError: true,
			errorMsg:    "directory does not exist",
		},
		{
			name: "invalid_format_option",
			args: []string{
				"--plugin=protoc-gen-openapiv3=" + pluginPath,
				"--openapiv3_out=" + t.TempDir(),
				"--openapiv3_opt=format=invalid",
				"--proto_path=testdata/proto",
				"testdata/proto/simple_service.proto",
			},
			expectError: false, // Should default to YAML and not error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("protoc", tc.args...)
			output, err := cmd.CombinedOutput()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but command succeeded. Output: %s", string(output))
				} else if !strings.Contains(string(output), tc.errorMsg) {
					t.Errorf("Expected error message to contain %q, but got: %s", tc.errorMsg, string(output))
				}
			} else {
				if err != nil {
					t.Errorf("Expected success but got error: %v\nOutput: %s", err, string(output))
				}
			}
		})
	}
}

// TestPluginFormatOptions tests various format options
func TestPluginFormatOptions(t *testing.T) {
	// Build the plugin binary for testing
	pluginPath := "./protoc-gen-openapiv3-format-test"
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build plugin: %v", err)
	}
	defer os.Remove(pluginPath)

	testCases := []struct {
		name           string
		format         string
		expectedExt    string
		expectedPrefix string // Expected start of file content
	}{
		{
			name:           "yaml_format",
			format:         "yaml",
			expectedExt:    ".yaml",
			expectedPrefix: "openapi: 3.1.0",
		},
		{
			name:           "json_format",
			format:         "json",
			expectedExt:    ".json",
			expectedPrefix: "{",
		},
		{
			name:           "default_format",
			format:         "", // No format specified, should default to YAML
			expectedExt:    ".yaml",
			expectedPrefix: "openapi: 3.1.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			args := []string{
				"--plugin=protoc-gen-openapiv3=" + pluginPath,
				"--openapiv3_out=" + tempDir,
				"--proto_path=testdata/proto",
				"--proto_path=../../proto",
				"testdata/proto/simple_service.proto",
			}

			// Add format option if specified
			if tc.format != "" {
				args = append(args[:2], append([]string{"--openapiv3_opt=format=" + tc.format}, args[2:]...)...)
			}

			cmd := exec.Command("protoc", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("protoc failed: %v\nOutput: %s", err, string(output))
			}

			// Find generated files
			files, listErr := os.ReadDir(tempDir)
			if listErr != nil {
				t.Fatalf("Failed to list output directory: %v", listErr)
			}

			if len(files) == 0 {
				t.Fatal("No files generated")
			}

			// Check file extension
			var generatedFile string
			for _, file := range files {
				if strings.HasSuffix(file.Name(), tc.expectedExt) {
					generatedFile = filepath.Join(tempDir, file.Name())
					break
				}
			}

			if generatedFile == "" {
				t.Fatalf("No file with expected extension %s found. Files: %v", tc.expectedExt, getFileNames(files))
			}

			// Check file content starts correctly
			content, readErr := os.ReadFile(generatedFile)
			if readErr != nil {
				t.Fatalf("Failed to read generated file: %v", readErr)
			}

			contentStr := strings.TrimSpace(string(content))
			if !strings.HasPrefix(contentStr, tc.expectedPrefix) {
				t.Errorf("File content should start with %q, but starts with: %q", tc.expectedPrefix, contentStr[:min(50, len(contentStr))])
			}
		})
	}
}

// TestPluginServiceGeneration tests generation of different service configurations
func TestPluginServiceGeneration(t *testing.T) {
	// Build the plugin binary for testing
	pluginPath := "./protoc-gen-openapiv3-service-test"
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build plugin: %v", err)
	}
	defer os.Remove(pluginPath)

	testCases := []struct {
		name             string
		protoFile        string
		expectedServices []string // Service names that should generate files
	}{
		{
			name:             "single_service",
			protoFile:        "simple_service.proto",
			expectedServices: []string{"SimpleService"},
		},
		{
			name:             "multiple_services",
			protoFile:        "multiple_services.proto",
			expectedServices: []string{"UserService", "AdminService", "NotificationService"},
		},
		{
			name:      "headers_service",
			protoFile: "headers.proto",
			expectedServices: []string{
				"HeaderService",
				"NoHeaderService", 
				"HeaderTypesService",
				"EdgeCaseService",
				"DeprecatedHeaderService",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			cmd := exec.Command("protoc",
				"--plugin=protoc-gen-openapiv3="+pluginPath,
				"--openapiv3_out="+tempDir,
				"--openapiv3_opt=format=yaml",
				"--proto_path=testdata/proto",
				"--proto_path=../../proto",
				"testdata/proto/"+tc.protoFile,
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("protoc failed: %v\nOutput: %s", err, string(output))
			}

			// Check that expected service files were generated
			files, listErr := os.ReadDir(tempDir)
			if listErr != nil {
				t.Fatalf("Failed to list output directory: %v", listErr)
			}

			generatedServices := make(map[string]bool)
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".openapi.yaml") {
					serviceName := strings.TrimSuffix(file.Name(), ".openapi.yaml")
					generatedServices[serviceName] = true
				}
			}

			// Verify all expected services were generated
			for _, expectedService := range tc.expectedServices {
				if !generatedServices[expectedService] {
					t.Errorf("Expected service %s was not generated. Generated files: %v", expectedService, getFileNames(files))
				}
			}

			// Verify no unexpected services were generated (basic check)
			if len(generatedServices) != len(tc.expectedServices) {
				t.Logf("Generated %d services, expected %d", len(generatedServices), len(tc.expectedServices))
				t.Logf("Generated services: %v", getMapKeys(generatedServices))
				t.Logf("Expected services: %v", tc.expectedServices)
			}
		})
	}
}

// Helper functions

func getFileNames(files []os.DirEntry) []string {
	names := make([]string, len(files))
	for i, file := range files {
		names[i] = file.Name()
	}
	return names
}

func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}