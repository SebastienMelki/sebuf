package httpgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestUnwrapFileGeneration tests that the unwrap file is generated correctly.
func TestUnwrapFileGeneration(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping unwrap tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	tempDir := t.TempDir()
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	// Build the plugin if it doesn't exist
	if _, buildStatErr := os.Stat(pluginPath); os.IsNotExist(buildStatErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	// Generate code
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+tempDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+tempDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"unwrap.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	// Read generated unwrap file
	unwrapPath := filepath.Join(tempDir, "unwrap_unwrap.pb.go")
	unwrapContent, err := os.ReadFile(unwrapPath)
	if err != nil {
		t.Fatalf("Failed to read generated unwrap file: %v", err)
	}

	content := string(unwrapContent)

	t.Run("MarshalJSON is generated for GetOptionBarsResponse", func(t *testing.T) {
		if !strings.Contains(content, "func (x *GetOptionBarsResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for GetOptionBarsResponse")
		}
	})

	t.Run("UnmarshalJSON is generated for GetOptionBarsResponse", func(t *testing.T) {
		if !strings.Contains(content, "func (x *GetOptionBarsResponse) UnmarshalJSON(data []byte) error") {
			t.Error("UnmarshalJSON not generated for GetOptionBarsResponse")
		}
	})

	t.Run("MarshalJSON handles unwrap field correctly", func(t *testing.T) {
		// Should marshal the unwrap field directly
		if !strings.Contains(content, "wrapper.GetBars()") {
			t.Error("MarshalJSON should call GetBars() on the wrapper")
		}
	})

	t.Run("UnmarshalJSON creates wrapper correctly", func(t *testing.T) {
		// Should create the wrapper with the unwrap field
		if !strings.Contains(content, "&OptionBarsList{Bars: items}") {
			t.Error("UnmarshalJSON should create OptionBarsList with Bars field")
		}
	})

	t.Run("MixedResponse handles both unwrap and regular maps", func(t *testing.T) {
		if !strings.Contains(content, "func (x *MixedResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for MixedResponse")
		}
		// Check that it handles both unwrap and regular map fields
		if !strings.Contains(content, "Handle unwrap map field: UnwrappedBars") {
			t.Error("MixedResponse should handle unwrap map field")
		}
		if !strings.Contains(content, "Handle regular map field: RegularBars") {
			t.Error("MixedResponse should handle regular map field")
		}
	})

	t.Run("ScalarMapResponse handles scalar unwrap", func(t *testing.T) {
		if !strings.Contains(content, "func (x *ScalarMapResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for ScalarMapResponse")
		}
	})
}

// TestUnwrapBindingIntegration tests that the binding file checks for json.Marshaler/Unmarshaler.
func TestUnwrapBindingIntegration(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping unwrap tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	tempDir := t.TempDir()
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	// Build the plugin if it doesn't exist
	if _, buildStatErr := os.Stat(pluginPath); os.IsNotExist(buildStatErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	// Generate code
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+tempDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+tempDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"unwrap.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	// Read generated binding file
	bindingPath := filepath.Join(tempDir, "unwrap_http_binding.pb.go")
	bindingContent, err := os.ReadFile(bindingPath)
	if err != nil {
		t.Fatalf("Failed to read generated binding file: %v", err)
	}

	content := string(bindingContent)

	t.Run("binding imports encoding/json", func(t *testing.T) {
		if !strings.Contains(content, `"encoding/json"`) {
			t.Error("Binding file should import encoding/json")
		}
	})

	t.Run("bindDataFromJSONRequest checks for json.Unmarshaler", func(t *testing.T) {
		if !strings.Contains(content, "if unmarshaler, ok := any(toBind).(json.Unmarshaler); ok {") {
			t.Error("bindDataFromJSONRequest should check for json.Unmarshaler")
		}
	})

	t.Run("marshalResponse checks for json.Marshaler", func(t *testing.T) {
		if !strings.Contains(content, "if marshaler, ok := response.(json.Marshaler); ok {") {
			t.Error("marshalResponse should check for json.Marshaler")
		}
	})
}

// TestUnwrapValidationError tests the UnwrapValidationError type.
func TestUnwrapValidationError(t *testing.T) {
	err := &UnwrapValidationError{
		MessageName: "TestMessage",
		FieldName:   "test_field",
		Reason:      "must be a repeated field",
	}

	expected := "invalid unwrap annotation on TestMessage.test_field: must be a repeated field"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

// TestUnwrapSamePackageDifferentFile tests that unwrap works when the wrapper message
// is defined in a different proto file but within the same Go package.
// This is the critical test for the bug fix.
func TestUnwrapSamePackageDifferentFile(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping unwrap tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	tempDir := t.TempDir()
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	// Build the plugin if it doesn't exist
	if _, buildStatErr := os.Stat(pluginPath); os.IsNotExist(buildStatErr) {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = projectRoot
		if buildErr := buildCmd.Run(); buildErr != nil {
			t.Fatalf("Failed to build plugin: %v", buildErr)
		}
	}

	// Generate code with BOTH proto files (same package, different files)
	// This is the critical test - both files must be passed to protoc together
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+tempDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+tempDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"same_pkg_wrapper.proto",
		"same_pkg_service.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	// The unwrap file should be generated for the SERVICE file (same_pkg_service.proto)
	// because GetBarsResponse contains a map<string, BarList> where BarList has the unwrap annotation
	unwrapPath := filepath.Join(tempDir, "same_pkg_service_unwrap.pb.go")
	unwrapContent, err := os.ReadFile(unwrapPath)
	if err != nil {
		// List files in temp dir to help debug
		files, _ := os.ReadDir(tempDir)
		t.Logf("Files in temp dir:")
		for _, f := range files {
			t.Logf("  - %s", f.Name())
		}
		t.Fatalf("Failed to read generated unwrap file (expected at %s): %v", unwrapPath, err)
	}

	content := string(unwrapContent)

	t.Run("MarshalJSON is generated for GetBarsResponse", func(t *testing.T) {
		if !strings.Contains(content, "func (x *GetBarsResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for GetBarsResponse")
			t.Logf("Generated content:\n%s", content)
		}
	})

	t.Run("UnmarshalJSON is generated for GetBarsResponse", func(t *testing.T) {
		if !strings.Contains(content, "func (x *GetBarsResponse) UnmarshalJSON(data []byte) error") {
			t.Error("UnmarshalJSON not generated for GetBarsResponse")
		}
	})

	t.Run("MarshalJSON handles unwrap field from different file", func(t *testing.T) {
		// Should marshal the unwrap field directly using GetBars()
		if !strings.Contains(content, "wrapper.GetBars()") {
			t.Error("MarshalJSON should call GetBars() on the wrapper")
			t.Logf("Generated content:\n%s", content)
		}
	})

	t.Run("UnmarshalJSON creates wrapper correctly", func(t *testing.T) {
		// Should create the wrapper with the Bars field
		if !strings.Contains(content, "&BarList{Bars: items}") {
			t.Error("UnmarshalJSON should create BarList with Bars field")
			t.Logf("Generated content:\n%s", content)
		}
	})

	t.Run("Generated code handles unwrap map field comment", func(t *testing.T) {
		// Check that it correctly identifies this as an unwrap map field
		if !strings.Contains(content, "Handle unwrap map field: Bars") {
			t.Error("Should identify Bars as an unwrap map field")
			t.Logf("Generated content:\n%s", content)
		}
	})
}
