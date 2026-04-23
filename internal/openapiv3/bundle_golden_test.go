package openapiv3_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBundleGoldenFiles verifies that bundle mode emits a single merged OpenAPI
// document covering every service in the protoc invocation. The bundled output
// uses proto-package-qualified schema names for collision safety and populates
// info / servers from plugin options.
func TestBundleGoldenFiles(t *testing.T) {
	pluginPath := "./protoc-gen-openapiv3-bundle-test"
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build plugin: %v", err)
	}
	defer os.Remove(pluginPath)

	// Shared option string exercising every bundle_* knob.
	bundleOpt := strings.Join([]string{
		"bundle=true",
		"bundle_only=true",
		"bundle_output=origin.openapi.yaml",
		"bundle_title=Multi API",
		"bundle_version=2.0.0",
		"bundle_description=Origin-level bundle spanning multiple services.",
		"bundle_server=https://api.example.com",
		"bundle_server=https://staging.example.com",
		"bundle_contact_name=API Team",
		"bundle_contact_email=api@example.com",
		"bundle_license_name=Apache-2.0",
		"bundle_license_url=https://www.apache.org/licenses/LICENSE-2.0",
	}, ",")

	testCases := []struct {
		name       string
		protoFile  string
		outputName string
		goldenFile string
		format     string
	}{
		{
			name:       "bundle_multiple_services_yaml",
			protoFile:  "testdata/proto/multiple_services.proto",
			outputName: "origin.openapi.yaml",
			goldenFile: "testdata/golden/yaml/bundle_multiple_services.openapi.yaml",
			format:     "yaml",
		},
		{
			name:       "bundle_multiple_services_json",
			protoFile:  "testdata/proto/multiple_services.proto",
			outputName: "origin.openapi.json",
			goldenFile: "testdata/golden/json/bundle_multiple_services.openapi.json",
			format:     "json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Override bundle_output extension to match the format being tested.
			outputName := tc.outputName
			opt := strings.Replace(bundleOpt, "bundle_output=origin.openapi.yaml", "bundle_output="+outputName, 1)
			opt = fmt.Sprintf("format=%s,%s", tc.format, opt)

			cmd := exec.Command("protoc",
				"--plugin=protoc-gen-openapiv3="+pluginPath,
				"--openapiv3_out="+tempDir,
				"--openapiv3_opt="+opt,
				"--proto_path=testdata/proto",
				"--proto_path=../../proto",
				tc.protoFile,
			)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if runErr := cmd.Run(); runErr != nil {
				t.Fatalf("protoc failed for %s: %v\nStdout: %s\nStderr: %s",
					tc.name, runErr, stdout.String(), stderr.String())
			}

			// bundle_only=true means no per-service files should have been written.
			entries, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatalf("Failed to read temp dir: %v", err)
			}
			for _, e := range entries {
				if e.Name() != outputName {
					t.Errorf("bundle_only=true should have suppressed per-service file %q", e.Name())
				}
			}

			generatedFile := filepath.Join(tempDir, outputName)
			generatedContent, err := os.ReadFile(generatedFile)
			if err != nil {
				t.Fatalf("Failed to read generated bundle %s: %v", generatedFile, err)
			}

			goldenContent, err := os.ReadFile(tc.goldenFile)
			if err != nil {
				if created := tryCreateGoldenFile(t, tc.goldenFile, generatedContent, err); created {
					return
				}
				t.Fatalf("Failed to read golden file %s: %v", tc.goldenFile, err)
			}

			if !bytes.Equal(generatedContent, goldenContent) {
				reportGoldenFileMismatch(t, tc.name, tc.goldenFile, generatedContent, goldenContent)
			} else {
				t.Logf("✓ Perfect match for %s (%d bytes)", tc.name, len(generatedContent))
			}
		})
	}
}

// TestBundleAlongsidePerService verifies that bundle=true (without bundle_only)
// emits both the bundle and the per-service files.
func TestBundleAlongsidePerService(t *testing.T) {
	pluginPath := "./protoc-gen-openapiv3-alongside-test"
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build plugin: %v", err)
	}
	defer os.Remove(pluginPath)

	tempDir := t.TempDir()

	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-openapiv3="+pluginPath,
		"--openapiv3_out="+tempDir,
		"--openapiv3_opt=format=yaml,bundle=true,bundle_title=Alongside",
		"--proto_path=testdata/proto",
		"--proto_path=../../proto",
		"testdata/proto/multiple_services.proto",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nStderr: %s", runErr, stderr.String())
	}

	expected := []string{
		"openapi.yaml",
		"UserService.openapi.yaml",
		"AdminService.openapi.yaml",
		"NotificationService.openapi.yaml",
	}
	for _, name := range expected {
		if _, err := os.Stat(filepath.Join(tempDir, name)); err != nil {
			t.Errorf("expected file %q to be emitted alongside bundle, got err: %v", name, err)
		}
	}
}
