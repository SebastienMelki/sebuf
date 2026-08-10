package httpgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	assertHTTPGenFixtureDoesNotGenerate(t, baseDir, "backward_compat_json_mapping.pb.go", "backward_compat.proto")

	if src := readGeneratedJSONMappingGoldenFixture(t, baseDir, "empty_behavior"); !strings.Contains(src, `raw["metadataNull"] = []byte("null")`) {
		t.Error("empty_behavior.proto composed JSON mapping should write null for EMPTY_BEHAVIOR_NULL fields")
	}

	// Verify TypeScript and OpenAPI empty_behavior golden files exist; Go coverage comes from generated composed output above.
	goldenFiles := []string{
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
