package httpgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmptyBehaviorConsistencyGoHTTPvsGoClient verifies go-http and go-client
// produce identical empty_behavior encoding code.
func TestEmptyBehaviorConsistencyGoHTTPvsGoClient(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "empty_behavior_empty_behavior.pb.go")
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"empty_behavior_empty_behavior.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen empty_behavior golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen empty_behavior golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client empty_behavior encoding code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical empty_behavior encoding code")
	}
}

// TestEmptyBehaviorConsistencyOpenAPI verifies OpenAPI uses oneOf for empty_behavior=NULL fields.
func TestEmptyBehaviorConsistencyOpenAPI(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	yamlFile := filepath.Join(
		baseDir,
		"..",
		"openapiv3",
		"testdata",
		"golden",
		"yaml",
		"EmptyBehaviorService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI empty_behavior golden file: %v", readErr)
	}

	yamlContent := string(content)

	// Verify NULL fields use oneOf with null type
	// metadataNull should use oneOf: [$ref Metadata, type: "null"]
	if !strings.Contains(yamlContent, "metadataNull:\n                    oneOf:") {
		t.Error("OpenAPI empty_behavior=NULL metadataNull should use oneOf schema")
	}
	if !strings.Contains(
		yamlContent,
		"- $ref: '#/components/schemas/Metadata'\n                        - type: \"null\"",
	) {
		t.Error("OpenAPI empty_behavior=NULL metadataNull oneOf should include $ref and type: null")
	}

	// settings should also use oneOf with null type
	if !strings.Contains(yamlContent, "settings:\n                    oneOf:") {
		t.Error("OpenAPI empty_behavior=NULL settings should use oneOf schema")
	}
	if !strings.Contains(
		yamlContent,
		"- $ref: '#/components/schemas/Settings'\n                        - type: \"null\"",
	) {
		t.Error("OpenAPI empty_behavior=NULL settings oneOf should include $ref and type: null")
	}

	// Verify PRESERVE fields use standard $ref (no oneOf)
	if !strings.Contains(yamlContent, "metadataPreserve:\n                    $ref: '#/components/schemas/Metadata'") {
		t.Error("OpenAPI empty_behavior=PRESERVE metadataPreserve should use standard $ref")
	}
	if strings.Contains(yamlContent, "metadataPreserve:\n                    oneOf:") {
		t.Error("OpenAPI empty_behavior=PRESERVE metadataPreserve should NOT use oneOf")
	}

	// Verify OMIT fields use standard $ref (no oneOf)
	if !strings.Contains(yamlContent, "metadataOmit:\n                    $ref: '#/components/schemas/Metadata'") {
		t.Error("OpenAPI empty_behavior=OMIT metadataOmit should use standard $ref")
	}
	if strings.Contains(yamlContent, "metadataOmit:\n                    oneOf:") {
		t.Error("OpenAPI empty_behavior=OMIT metadataOmit should NOT use oneOf")
	}

	// Verify default (no annotation) fields use standard $ref
	if !strings.Contains(yamlContent, "metadataDefault:\n                    $ref: '#/components/schemas/Metadata'") {
		t.Error("OpenAPI default metadataDefault should use standard $ref")
	}

	t.Log("OpenAPI empty_behavior schemas correctly use oneOf for NULL fields")
}

// TestEmptyBehaviorConsistencyBackwardCompat verifies protos without empty_behavior are unaffected.
func TestEmptyBehaviorConsistencyBackwardCompat(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	// backward_compat.proto should NOT generate an empty_behavior file
	emptyBehaviorFile := filepath.Join(baseDir, "testdata", "golden", "backward_compat_empty_behavior.pb.go")
	if _, statErr := os.Stat(emptyBehaviorFile); statErr == nil {
		t.Error("backward_compat.proto should not generate an empty_behavior file (no empty_behavior annotations)")
	}

	// Verify all generators have empty_behavior golden files
	goldenFiles := []string{
		// Go httpgen
		filepath.Join(baseDir, "testdata", "golden", "empty_behavior_empty_behavior.pb.go"),
		// Go clientgen
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "empty_behavior_empty_behavior.pb.go"),
		// TypeScript
		filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "empty_behavior_client.ts"),
		// OpenAPI
		filepath.Join(baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "EmptyBehaviorService.openapi.yaml"),
	}

	for _, f := range goldenFiles {
		if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
			t.Errorf("Missing empty_behavior golden file for cross-generator consistency: %s", f)
		}
	}

	t.Log("Backward compatibility verified - protos without empty_behavior are unaffected")
}
