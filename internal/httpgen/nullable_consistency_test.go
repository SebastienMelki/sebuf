package httpgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNullableConsistencyGoHTTPvsGoClient verifies go-http and go-client
// produce identical nullable encoding code.
func TestNullableConsistencyGoHTTPvsGoClient(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "nullable_nullable.pb.go")
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"nullable_nullable.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen nullable golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen nullable golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client nullable encoding code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical nullable encoding code")
	}
}

// TestNullableConsistencyTypeScript verifies TypeScript uses T | null for nullable fields.
func TestNullableConsistencyTypeScript(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "nullable_client.ts")

	content, readErr := os.ReadFile(tsFile)
	if readErr != nil {
		t.Fatalf("Failed to read TypeScript nullable golden file: %v", readErr)
	}

	tsContent := string(content)

	// Verify nullable fields use T | null syntax (not optional ?)
	nullableFields := []struct {
		field    string
		baseType string
	}{
		{"middleName", "string"},
		{"age", "number"},
		{"isVerified", "boolean"},
	}

	for _, nf := range nullableFields {
		expectedPattern := nf.field + ": " + nf.baseType + " | null"
		if !strings.Contains(tsContent, expectedPattern) {
			t.Errorf("TypeScript nullable field %q should have type '%s | null', expected pattern: %q",
				nf.field, nf.baseType, expectedPattern)
		}

		// Verify it does NOT use optional ? syntax for nullable fields
		optionalPattern := nf.field + "?: " + nf.baseType
		if strings.Contains(tsContent, optionalPattern) {
			t.Errorf("TypeScript nullable field %q should use 'T | null' (not optional '?')", nf.field)
		}
	}

	// Verify non-nullable optional field uses ? syntax (not | null)
	if !strings.Contains(tsContent, "nickname?: string") {
		t.Error("TypeScript non-nullable optional field 'nickname' should use '?' syntax")
	}

	// Verify non-nullable optional field does NOT use | null
	if strings.Contains(tsContent, "nickname: string | null") {
		t.Error("TypeScript non-nullable optional field 'nickname' should NOT use '| null'")
	}

	t.Log("TypeScript nullable types correctly use T | null syntax")
}

// TestNullableConsistencyOpenAPI verifies OpenAPI uses type array ["T", "null"] for nullable fields.
func TestNullableConsistencyOpenAPI(t *testing.T) {
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
		"NullableService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI nullable golden file: %v", readErr)
	}

	yamlContent := string(content)

	// Verify nullable string field uses type array with "null"
	if !strings.Contains(
		yamlContent,
		"middleName:\n                    type:\n                        - string\n                        - \"null\"",
	) {
		t.Error("OpenAPI nullable middleName should use type array [string, null]")
	}

	// Verify nullable integer field uses type array with "null"
	if !strings.Contains(
		yamlContent,
		"age:\n                    type:\n                        - integer\n                        - \"null\"",
	) {
		t.Error("OpenAPI nullable age should use type array [integer, null]")
	}

	// Verify nullable boolean field uses type array with "null"
	if !strings.Contains(
		yamlContent,
		"isVerified:\n                    type:\n                        - boolean\n                        - \"null\"",
	) {
		t.Error("OpenAPI nullable isVerified should use type array [boolean, null]")
	}

	// Verify non-nullable field does NOT use type array
	if strings.Contains(
		yamlContent,
		"nickname:\n                    type:\n                        - string\n                        - \"null\"",
	) {
		t.Error("OpenAPI non-nullable nickname should NOT use type array with null")
	}

	// Verify id field (required, non-nullable) is simple type
	if !strings.Contains(yamlContent, "id:\n                    type: string") {
		t.Error("OpenAPI non-nullable id should use simple type: string")
	}

	t.Log("OpenAPI nullable schemas correctly use type array [T, null] syntax")
}

// TestNullableConsistencyBackwardCompat verifies protos without nullable annotations are unaffected.
func TestNullableConsistencyBackwardCompat(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	// backward_compat.proto should NOT generate a nullable file
	nullableFile := filepath.Join(baseDir, "testdata", "golden", "backward_compat_nullable.pb.go")
	if _, statErr := os.Stat(nullableFile); statErr == nil {
		t.Error("backward_compat.proto should not generate a nullable file (no nullable annotations)")
	}

	// Verify all generators have nullable golden files
	goldenFiles := []string{
		// Go httpgen
		filepath.Join(baseDir, "testdata", "golden", "nullable_nullable.pb.go"),
		// Go clientgen
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "nullable_nullable.pb.go"),
		// TypeScript
		filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "nullable_client.ts"),
		// OpenAPI
		filepath.Join(baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "NullableService.openapi.yaml"),
	}

	for _, f := range goldenFiles {
		if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
			t.Errorf("Missing nullable golden file for cross-generator consistency: %s", f)
		}
	}

	t.Log("Backward compatibility verified - protos without nullable are unaffected")
}
