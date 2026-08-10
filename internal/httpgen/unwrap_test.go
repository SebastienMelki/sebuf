package httpgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// TestUnwrapFileGeneration tests that unwrap handling is generated in the composed JSON mapping file.
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

	buildHTTPPluginForUnwrapTest(t, projectRoot, pluginPath)

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

	if _, statErr := os.Stat(filepath.Join(tempDir, "unwrap_unwrap.pb.go")); !os.IsNotExist(statErr) {
		t.Fatalf("legacy standalone unwrap file should not be generated, stat error: %v", statErr)
	}

	// Read generated composed JSON mapping file
	mappingPath := filepath.Join(tempDir, "unwrap_json_mapping.pb.go")
	mappingContent, err := os.ReadFile(mappingPath)
	if err != nil {
		t.Fatalf("Failed to read generated JSON mapping file: %v", err)
	}

	content := string(mappingContent)

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

	t.Run("UnmarshalJSON rewraps values before protojson", func(t *testing.T) {
		if !strings.Contains(content, "Re-wrap map values for unwrap field: bars") {
			t.Error("UnmarshalJSON should re-wrap map values in the composed pipeline")
		}
		if !strings.Contains(content, `"bars": arrayRaw`) {
			t.Error("UnmarshalJSON should wrap arrays under the Bars JSON field")
		}
	})

	t.Run("MixedResponse handles unwrap map through composed transform", func(t *testing.T) {
		if !strings.Contains(content, "func (x *MixedResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for MixedResponse")
		}
		if !strings.Contains(content, "Handle unwrap map field: UnwrappedBars") {
			t.Error("MixedResponse should handle unwrap map field")
		}
	})

	t.Run("ScalarMapResponse handles scalar unwrap", func(t *testing.T) {
		if !strings.Contains(content, "func (x *ScalarMapResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for ScalarMapResponse")
		}
	})
}

func buildHTTPPluginForUnwrapTest(t *testing.T, projectRoot, pluginPath string) {
	t.Helper()

	buildCmd := exec.Command("go", "build", "-o", pluginPath, "./cmd/protoc-gen-go-http")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build plugin: %v\n%s", err, out)
	}
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

	buildHTTPPluginForUnwrapTest(t, projectRoot, pluginPath)

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

	t.Run("marshalJSONWithOpts checks sebufMarshaler before json.Marshaler", func(t *testing.T) {
		if !strings.Contains(content, "if m, ok := msg.(sebufMarshaler); ok {") {
			t.Error("marshalJSONWithOpts should check for sebufMarshaler")
		}
		if !strings.Contains(content, "if m, ok := msg.(json.Marshaler); ok {") {
			t.Error("marshalJSONWithOpts should also check for json.Marshaler as a fallback")
		}
	})
}

// TestUnwrapValidationError tests the UnwrapValidationError type.
func TestUnwrapValidationError(t *testing.T) {
	err := &annotations.UnwrapValidationError{
		MessageName: "TestMessage",
		FieldName:   "test_field",
		Reason:      "must be a repeated field",
	}

	expected := "invalid unwrap annotation on TestMessage.test_field: must be a repeated field"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

// TestRootUnwrapFileGeneration tests that root unwrap methods are generated correctly.
func TestRootUnwrapFileGeneration(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping root unwrap tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	tempDir := t.TempDir()
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	buildHTTPPluginForUnwrapTest(t, projectRoot, pluginPath)

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

	if _, statErr := os.Stat(filepath.Join(tempDir, "unwrap_unwrap.pb.go")); !os.IsNotExist(statErr) {
		t.Fatalf("legacy standalone unwrap file should not be generated, stat error: %v", statErr)
	}

	// Read generated composed JSON mapping file
	mappingPath := filepath.Join(tempDir, "unwrap_json_mapping.pb.go")
	mappingContent, err := os.ReadFile(mappingPath)
	if err != nil {
		t.Fatalf("Failed to read generated JSON mapping file: %v", err)
	}

	content := string(mappingContent)

	t.Run("RootMapResponse MarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *RootMapResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for RootMapResponse")
		}
	})

	t.Run("RootMapResponse UnmarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *RootMapResponse) UnmarshalJSON(data []byte) error") {
			t.Error("UnmarshalJSON not generated for RootMapResponse")
		}
	})

	t.Run("RootRepeatedResponse MarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *RootRepeatedResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for RootRepeatedResponse")
		}
	})

	t.Run("RootRepeatedResponse UnmarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *RootRepeatedResponse) UnmarshalJSON(data []byte) error") {
			t.Error("UnmarshalJSON not generated for RootRepeatedResponse")
		}
	})

	t.Run("RootMapWithValueUnwrapResponse MarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *RootMapWithValueUnwrapResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for RootMapWithValueUnwrapResponse")
		}
	})

	t.Run("ScalarRootMapResponse MarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *ScalarRootMapResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for ScalarRootMapResponse")
		}
	})

	t.Run("ScalarRootRepeatedResponse MarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *ScalarRootRepeatedResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for ScalarRootRepeatedResponse")
		}
	})

	t.Run("Root unwrap marshal returns transformed raw field", func(t *testing.T) {
		if !strings.Contains(content, "// Apply root-level unwrap last.") {
			t.Error("root unwrap should run as the final composed marshal transform")
		}
		if !strings.Contains(content, "return rootRaw, nil") {
			t.Error("root unwrap should return the transformed raw field value")
		}
	})
}

// TestCrossFileUnwrapResolution tests that unwrap annotations are resolved across
// proto files in the same Go package. This tests the GlobalUnwrapInfo feature.
func TestCrossFileUnwrapResolution(t *testing.T) {
	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping cross-file unwrap tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	tempDir := t.TempDir()
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	buildHTTPPluginForUnwrapTest(t, projectRoot, pluginPath)

	// Generate code for BOTH proto files simultaneously (same package, different files)
	// This is the key: protoc processes both files together, and our generator must
	// resolve unwrap annotations across them
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+tempDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+tempDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"same_pkg_service.proto",
		"same_pkg_wrapper.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	if _, statErr := os.Stat(filepath.Join(tempDir, "same_pkg_service_unwrap.pb.go")); !os.IsNotExist(statErr) {
		t.Fatalf("legacy standalone unwrap file should not be generated, stat error: %v", statErr)
	}

	// The JSON mapping file should be generated for same_pkg_service.proto because
	// GetBarsResponse has a map<string, BarList> where BarList (from same_pkg_wrapper.proto)
	// has an unwrap field.
	mappingPath := filepath.Join(tempDir, "same_pkg_service_json_mapping.pb.go")
	mappingContent, err := os.ReadFile(mappingPath)
	if err != nil {
		t.Fatalf("Failed to read generated JSON mapping file (cross-file resolution failed): %v", err)
	}

	content := string(mappingContent)

	t.Run("GetBarsResponse MarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *GetBarsResponse) MarshalJSON() ([]byte, error)") {
			t.Error("MarshalJSON not generated for GetBarsResponse - cross-file unwrap resolution failed")
		}
	})

	t.Run("GetBarsResponse UnmarshalJSON is generated", func(t *testing.T) {
		if !strings.Contains(content, "func (x *GetBarsResponse) UnmarshalJSON(data []byte) error") {
			t.Error("UnmarshalJSON not generated for GetBarsResponse - cross-file unwrap resolution failed")
		}
	})

	t.Run("MarshalJSON accesses wrapper's Bars field", func(t *testing.T) {
		if !strings.Contains(content, "wrapper.GetBars()") {
			t.Error("MarshalJSON should call GetBars() on the wrapper from the other file")
		}
	})

	t.Run("UnmarshalJSON rewraps arrays under BarList Bars field", func(t *testing.T) {
		if !strings.Contains(content, `"bars": arrayRaw`) {
			t.Error("UnmarshalJSON should wrap arrays under the BarList Bars JSON field")
		}
	})
}

// TestCrossFileInt64EncodingUnwrap proves that when Bar (with int64_encoding=NUMBER)
// is defined in a separate proto file from GetBarsResponse (which has a map<string,BarList>
// unwrap field), the unwrap generator must emit json.Marshal(item) — not protojson.Marshal(item).
//
// If this test fails with "uses protojson.Marshal instead of json.Marshal", it means
// collectDirectEncodingMsgNames only scans the current file and misses Bar from the imported file,
// causing Bar.MarshalJSON (from the encoding generator) to be bypassed at runtime.
func TestCrossFileInt64EncodingUnwrap(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	tempDir := t.TempDir()
	pluginPath := filepath.Join(projectRoot, "bin", "protoc-gen-go-http")

	buildHTTPPluginForUnwrapTest(t, projectRoot, pluginPath)

	// Both files are passed together — Bar is in cross_int64_bar.proto,
	// GetBarsResponse is in cross_int64_service.proto.
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http="+pluginPath,
		"--go_out="+tempDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+tempDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		"cross_int64_bar.proto",
		"cross_int64_service.proto",
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	if _, statErr := os.Stat(filepath.Join(tempDir, "cross_int64_service_unwrap.pb.go")); !os.IsNotExist(statErr) {
		t.Fatalf("legacy standalone unwrap file should not be generated, stat error: %v", statErr)
	}

	// The JSON mapping file is generated for cross_int64_service.proto (it owns GetBarsResponse).
	mappingContent, readErr := os.ReadFile(filepath.Join(tempDir, "cross_int64_service_json_mapping.pb.go"))
	if readErr != nil {
		t.Fatalf("Failed to read generated JSON mapping file: %v", readErr)
	}
	content := string(mappingContent)

	// Bar.MarshalJSONSebuf (from cross_int64_bar_encoding.pb.go) converts volume to a number.
	// The unwrap generator emits an inline MarshalJSONSebuf type assertion for each item
	// so Bar.MarshalJSONSebuf is invoked. Direct protojson.Marshal(item) would bypass it
	// and volume would be serialized as a quoted string — wrong.
	t.Run("forwards opts via inline MarshalJSONSebuf so Bar.MarshalJSONSebuf is called", func(t *testing.T) {
		if strings.Contains(content, "protojson.Marshal(item)") {
			t.Error("unwrap generator must not call protojson.Marshal(item) directly: " +
				"Bar.MarshalJSONSebuf from the encoding generator would be bypassed, " +
				"and int64 fields with NUMBER encoding would serialize as quoted strings")
		}
		if !strings.Contains(content, "m.MarshalJSONSebuf(opts)") {
			t.Error("expected inline MarshalJSONSebuf forwarding in generated unwrap file")
		}
	})
}
