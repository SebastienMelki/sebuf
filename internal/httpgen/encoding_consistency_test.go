package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGoGeneratorsProduceIdenticalInt64Encoding verifies go-http and go-client
// produce identical int64 encoding code.
func TestGoGeneratorsProduceIdenticalInt64Encoding(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "int64_encoding_encoding.pb.go")
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"int64_encoding_encoding.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen int64 encoding golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen int64 encoding golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client int64 encoding code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical int64 encoding code")
	}
}

// TestGoGeneratorsProduceIdenticalEnumEncoding verifies go-http and go-client
// produce identical enum encoding code.
func TestGoGeneratorsProduceIdenticalEnumEncoding(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "enum_encoding_enum_encoding.pb.go")
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"enum_encoding_enum_encoding.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen enum encoding golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen enum encoding golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client enum encoding code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical enum encoding code")
	}
}

// TestTypeScriptInt64TypesMatchGoEncoding verifies TypeScript types match Go encoding.
func TestTypeScriptInt64TypesMatchGoEncoding(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "int64_encoding_client.ts")

	content, readErr := os.ReadFile(tsFile)
	if readErr != nil {
		t.Fatalf("Failed to read TypeScript int64 encoding golden file: %v", readErr)
	}

	tsContent := string(content)

	// Verify NUMBER fields use TypeScript number type
	numberFields := []string{"numberInt64", "numberUint64", "numberSint64", "numberSfixed64", "numberFixed64"}
	for _, field := range numberFields {
		// Match field: number or field?: number
		pattern := regexp.MustCompile(field + `\??:\s*number`)
		if !pattern.MatchString(tsContent) {
			t.Errorf("TypeScript int64 NUMBER field %q should have type 'number'", field)
		}
	}

	// Verify STRING (default) fields use TypeScript string type
	stringFields := []string{"defaultInt64", "stringInt64", "defaultUint64"}
	for _, field := range stringFields {
		// Match field: string or field?: string
		pattern := regexp.MustCompile(field + `\??:\s*string`)
		if !pattern.MatchString(tsContent) {
			t.Errorf("TypeScript int64 STRING field %q should have type 'string'", field)
		}
	}

	// Verify repeated NUMBER field uses number[]
	if !strings.Contains(tsContent, "repeatedNumberInt64: number[]") {
		t.Error("TypeScript repeated NUMBER int64 field should have type 'number[]'")
	}

	// Verify repeated STRING field uses string[]
	if !strings.Contains(tsContent, "repeatedDefaultInt64: string[]") {
		t.Error("TypeScript repeated STRING int64 field should have type 'string[]'")
	}

	t.Log("TypeScript int64 types correctly match Go NUMBER vs STRING encoding")
}

// TestTypeScriptEnumTypesMatchGoEncoding verifies TypeScript enum types match Go encoding.
func TestTypeScriptEnumTypesMatchGoEncoding(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "enum_encoding_client.ts")

	content, readErr := os.ReadFile(tsFile)
	if readErr != nil {
		t.Fatalf("Failed to read TypeScript enum encoding golden file: %v", readErr)
	}

	tsContent := string(content)

	// Verify Status enum uses custom enum_value mappings
	if !strings.Contains(tsContent, `type Status = "unknown" | "active" | "inactive"`) {
		t.Error("TypeScript Status enum should use custom enum_value strings")
	}

	// Verify Priority enum uses proto names (no custom values)
	if !strings.Contains(tsContent, `type Priority = "PRIORITY_LOW" | "PRIORITY_MEDIUM" | "PRIORITY_HIGH"`) {
		t.Error("TypeScript Priority enum should use proto name strings")
	}

	// Verify NUMBER-encoded enum field uses number type
	if !strings.Contains(tsContent, "priorityAsNumber: number") {
		t.Error("TypeScript enum NUMBER field should have type 'number'")
	}

	// Verify STRING-encoded enum field uses enum type
	if !strings.Contains(tsContent, "priorityAsString: Priority") {
		t.Error("TypeScript enum STRING field should have type 'Priority'")
	}

	t.Log("TypeScript enum types correctly match Go enum encoding")
}

// TestOpenAPIInt64SchemasMatchGoEncoding verifies OpenAPI schemas match Go encoding.
func TestOpenAPIInt64SchemasMatchGoEncoding(t *testing.T) {
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
		"Int64EncodingService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI int64 encoding golden file: %v", readErr)
	}

	yamlContent := string(content)

	// Verify STRING (default) fields have type: string
	if !strings.Contains(yamlContent, "defaultInt64:\n                    type: string") {
		t.Error("OpenAPI defaultInt64 should have type: string")
	}

	// Verify NUMBER fields have type: integer
	if !strings.Contains(yamlContent, "numberInt64:\n                    type: integer") {
		t.Error("OpenAPI numberInt64 should have type: integer")
	}

	// Verify NUMBER fields have precision warning in description
	if !strings.Contains(yamlContent, "Warning: Values > 2^53 may lose precision in JavaScript") {
		t.Error("OpenAPI NUMBER int64 fields should include precision warning")
	}

	t.Log("OpenAPI int64 schemas correctly match Go encoding")
}

// TestOpenAPIEnumSchemasMatchGoEncoding verifies OpenAPI enum schemas match Go encoding.
func TestOpenAPIEnumSchemasMatchGoEncoding(t *testing.T) {
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
		"EnumEncodingService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI enum encoding golden file: %v", readErr)
	}

	yamlContent := string(content)

	// Verify Status enum uses custom values
	if !strings.Contains(yamlContent, "- unknown") ||
		!strings.Contains(yamlContent, "- active") ||
		!strings.Contains(yamlContent, "- inactive") {
		t.Error("OpenAPI Status enum should use custom enum_value strings")
	}

	// Verify Priority enum uses proto names
	if !strings.Contains(yamlContent, "- PRIORITY_LOW") ||
		!strings.Contains(yamlContent, "- PRIORITY_MEDIUM") ||
		!strings.Contains(yamlContent, "- PRIORITY_HIGH") {
		t.Error("OpenAPI Priority enum should use proto name strings")
	}

	// Verify NUMBER-encoded enum has type: integer
	if !strings.Contains(yamlContent, "priorityAsNumber:\n                    type: integer") {
		t.Error("OpenAPI NUMBER enum field should have type: integer")
	}

	// Verify STRING-encoded enum has type: string
	if !strings.Contains(yamlContent, "priorityAsString:\n                    type: string") {
		t.Error("OpenAPI STRING enum field should have type: string")
	}

	t.Log("OpenAPI enum schemas correctly match Go encoding")
}

// TestPhase4SuccessCriteria explicitly verifies each Phase 4 success criterion from ROADMAP.md.
//nolint:funlen // This test function covers all 6 Phase 4 success criteria
func TestPhase4SuccessCriteria(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	t.Run("Criterion 1: int64_encoding=STRING produces JSON strings", func(t *testing.T) {
		verifyCriterion1Int64String(t, baseDir)
	})

	t.Run("Criterion 2: int64_encoding=NUMBER produces JSON numbers with warning", func(t *testing.T) {
		verifyCriterion2Int64Number(t, baseDir)
	})

	t.Run("Criterion 3: enum_encoding=STRING produces proto name strings", func(t *testing.T) {
		verifyCriterion3EnumString(t, baseDir)
	})

	t.Run("Criterion 4: enum_value annotations produce custom JSON strings", func(t *testing.T) {
		verifyCriterion4EnumValue(t, baseDir)
	})

	t.Run("Criterion 5: OpenAPI schemas accurately reflect encoding", func(t *testing.T) {
		verifyCriterion5OpenAPISchemas(t, baseDir)
	})

	t.Run("Criterion 6: Cross-generator consistency verified", func(t *testing.T) {
		verifyCriterion6CrossGenerator(t, baseDir)
	})
}

func verifyCriterion1Int64String(t *testing.T, baseDir string) {
	t.Helper()

	// Go: default behavior via protojson (no custom MarshalJSON for STRING fields)
	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "int64_encoding_encoding.pb.go")
	content, readErr := os.ReadFile(httpgenFile)
	if readErr != nil {
		t.Fatalf("Failed to read file: %v", readErr)
	}

	// The MarshalJSON function should NOT modify STRING-encoded fields
	// It only modifies NUMBER fields listed in the comment
	if !strings.Contains(string(content), "int64_encoding=NUMBER fields:") {
		t.Error("Go encoding file should document which fields have NUMBER encoding")
	}

	// Verify STRING fields are not in the NUMBER modification list
	if strings.Contains(string(content), "DefaultInt64") &&
		strings.Contains(string(content), "raw[\"defaultInt64\"]") {
		t.Error("STRING-encoded defaultInt64 should not be modified in MarshalJSON")
	}

	// TypeScript: string type
	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "int64_encoding_client.ts")
	tsContent, _ := os.ReadFile(tsFile)
	if !strings.Contains(string(tsContent), "defaultInt64: string") {
		t.Error("TypeScript STRING int64 should be 'string' type")
	}

	// OpenAPI: type: string
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "Int64EncodingService.openapi.yaml",
	)
	yamlContent, _ := os.ReadFile(yamlFile)
	if !strings.Contains(string(yamlContent), "defaultInt64:\n                    type: string") {
		t.Error("OpenAPI STRING int64 should be 'type: string'")
	}

	t.Log("PASS: Criterion 1 verified - int64_encoding=STRING produces JSON strings")
}

func verifyCriterion2Int64Number(t *testing.T, baseDir string) {
	t.Helper()

	// Go: MarshalJSON converts to number
	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "int64_encoding_encoding.pb.go")
	content, _ := os.ReadFile(httpgenFile)
	goContent := string(content)

	// Should have MarshalJSON for NUMBER fields
	if !strings.Contains(goContent, "func (x *Int64EncodingTest) MarshalJSON()") {
		t.Error("Go should generate MarshalJSON for NUMBER int64 fields")
	}

	// Should contain precision warning comment
	if !strings.Contains(goContent, "may lose precision for values > 2^53") {
		t.Error("Go should include precision warning for NUMBER encoding")
	}

	// TypeScript: number type
	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "int64_encoding_client.ts")
	tsContent, _ := os.ReadFile(tsFile)
	if !strings.Contains(string(tsContent), "numberInt64: number") {
		t.Error("TypeScript NUMBER int64 should be 'number' type")
	}

	// OpenAPI: type: integer with warning
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "Int64EncodingService.openapi.yaml",
	)
	yamlContent, _ := os.ReadFile(yamlFile)
	if !strings.Contains(string(yamlContent), "numberInt64:\n                    type: integer") {
		t.Error("OpenAPI NUMBER int64 should be 'type: integer'")
	}
	if !strings.Contains(string(yamlContent), "Warning: Values > 2^53 may lose precision") {
		t.Error("OpenAPI NUMBER int64 should include precision warning")
	}

	t.Log("PASS: Criterion 2 verified - int64_encoding=NUMBER produces JSON numbers with warning")
}

func verifyCriterion3EnumString(t *testing.T, baseDir string) {
	t.Helper()

	// TypeScript: Priority enum uses proto names
	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "enum_encoding_client.ts")
	tsContent, _ := os.ReadFile(tsFile)
	if !strings.Contains(string(tsContent), `"PRIORITY_LOW" | "PRIORITY_MEDIUM" | "PRIORITY_HIGH"`) {
		t.Error("TypeScript Priority enum should use proto names")
	}

	// OpenAPI: uses proto names
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "EnumEncodingService.openapi.yaml",
	)
	yamlContent, _ := os.ReadFile(yamlFile)
	if !strings.Contains(string(yamlContent), "- PRIORITY_LOW") {
		t.Error("OpenAPI Priority enum should use proto names")
	}

	t.Log("PASS: Criterion 3 verified - enum_encoding=STRING produces proto name strings")
}

func verifyCriterion4EnumValue(t *testing.T, baseDir string) {
	t.Helper()

	// Go: lookup maps with custom values
	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "enum_encoding_enum_encoding.pb.go")
	content, _ := os.ReadFile(httpgenFile)
	goContent := string(content)

	if !strings.Contains(goContent, `Status_STATUS_UNSPECIFIED: "unknown"`) {
		t.Error("Go should map STATUS_UNSPECIFIED to custom value 'unknown'")
	}
	if !strings.Contains(goContent, `Status_STATUS_ACTIVE:      "active"`) {
		t.Error("Go should map STATUS_ACTIVE to custom value 'active'")
	}

	// TypeScript: custom values in union type
	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "enum_encoding_client.ts")
	tsContent, _ := os.ReadFile(tsFile)
	if !strings.Contains(string(tsContent), `"unknown" | "active" | "inactive"`) {
		t.Error("TypeScript Status enum should use custom values")
	}

	// OpenAPI: custom values in enum
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "EnumEncodingService.openapi.yaml",
	)
	yamlContent, _ := os.ReadFile(yamlFile)
	if !strings.Contains(string(yamlContent), "- unknown") ||
		!strings.Contains(string(yamlContent), "- active") {
		t.Error("OpenAPI Status enum should use custom values")
	}

	t.Log("PASS: Criterion 4 verified - enum_value annotations produce custom JSON strings")
}

func verifyCriterion5OpenAPISchemas(t *testing.T, baseDir string) {
	t.Helper()

	// Int64 schemas
	int64YamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "Int64EncodingService.openapi.yaml",
	)
	int64Content, _ := os.ReadFile(int64YamlFile)
	int64Yaml := string(int64Content)

	// STRING -> type: string, format: int64
	if !strings.Contains(int64Yaml, "type: string\n                    format: int64") {
		t.Error("OpenAPI STRING int64 should have type: string, format: int64")
	}

	// NUMBER -> type: integer, format: int64
	if !strings.Contains(int64Yaml, "type: integer\n                    format: int64") {
		t.Error("OpenAPI NUMBER int64 should have type: integer, format: int64")
	}

	// Enum schemas
	enumYamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "EnumEncodingService.openapi.yaml",
	)
	enumContent, _ := os.ReadFile(enumYamlFile)
	enumYaml := string(enumContent)

	// NUMBER enum -> type: integer with numeric enum values
	expectedNumberEnum := "priorityAsNumber:\n                    type: integer\n                    enum:\n                        - 0\n                        - 1\n                        - 2"
	if !strings.Contains(enumYaml, expectedNumberEnum) {
		t.Error("OpenAPI NUMBER enum should have type: integer with numeric values")
	}

	// STRING enum -> type: string with string enum values
	if !strings.Contains(
		enumYaml,
		"priorityAsString:\n                    type: string\n                    enum:\n                        - PRIORITY_LOW",
	) {
		t.Error("OpenAPI STRING enum should have type: string with proto names")
	}

	t.Log("PASS: Criterion 5 verified - OpenAPI schemas accurately reflect encoding")
}

func verifyCriterion6CrossGenerator(t *testing.T, baseDir string) {
	t.Helper()

	// This criterion is verified by the other tests (TestGoGenerators*, TestTypeScript*, TestOpenAPI*)
	// Here we just verify all golden files exist
	goldenFiles := []string{
		// Go httpgen
		filepath.Join(baseDir, "testdata", "golden", "int64_encoding_encoding.pb.go"),
		filepath.Join(baseDir, "testdata", "golden", "enum_encoding_enum_encoding.pb.go"),
		// Go clientgen
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "int64_encoding_encoding.pb.go"),
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "enum_encoding_enum_encoding.pb.go"),
		// TypeScript
		filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "int64_encoding_client.ts"),
		filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "enum_encoding_client.ts"),
		// OpenAPI
		filepath.Join(baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "Int64EncodingService.openapi.yaml"),
		filepath.Join(baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "EnumEncodingService.openapi.yaml"),
	}

	for _, f := range goldenFiles {
		if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
			t.Errorf("Missing golden file for cross-generator consistency: %s", f)
		}
	}

	t.Log("PASS: Criterion 6 verified - All generators have encoding golden files")
}

// normalizeGeneratorComment replaces the generator name in the "Code generated by" comment
// to allow comparison between go-http and go-client output.
func normalizeGeneratorComment(content, generatorName string) string {
	// Replace "Code generated by protoc-gen-go-http" or "protoc-gen-go-client"
	// with a normalized placeholder
	content = strings.ReplaceAll(content, "protoc-gen-"+generatorName, "protoc-gen-NORMALIZED")
	return content
}

// findFirstDifference returns a description of the first difference between two strings.
func findFirstDifference(a, b string) string {
	lines1 := strings.Split(a, "\n")
	lines2 := strings.Split(b, "\n")

	for i := 0; i < len(lines1) && i < len(lines2); i++ {
		if lines1[i] != lines2[i] {
			return "Line " + formatLineNumber(i) +
				":\n  Expected: " + lines1[i] + "\n  Actual:   " + lines2[i]
		}
	}

	if len(lines1) != len(lines2) {
		return "Different number of lines: " + formatLineNumber(len(lines1)) +
			" vs " + formatLineNumber(len(lines2))
	}

	return "No difference found"
}

func formatLineNumber(n int) string {
	return string(rune('0'+n/100)) + string(rune('0'+(n/10)%10)) + string(rune('0'+n%10))
}

// TestBackwardCompatibility verifies protos without encoding annotations are unchanged.
func TestBackwardCompatibility(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	t.Run("Proto without encoding annotations produces no encoding file", func(t *testing.T) {
		// Check that backward_compat.proto (no encoding annotations) doesn't generate
		// an encoding file
		encodingFile := filepath.Join(baseDir, "testdata", "golden", "backward_compat_encoding.pb.go")
		if _, statErr := os.Stat(encodingFile); statErr == nil {
			t.Error("backward_compat.proto should not generate an encoding file (no encoding annotations)")
		}

		t.Log("PASS: Protos without encoding annotations are unchanged")
	})

	t.Run("Proto without encoding annotations produces standard golden files", func(t *testing.T) {
		// Verify backward_compat.proto still generates the standard files
		standardFiles := []string{
			filepath.Join(baseDir, "testdata", "golden", "backward_compat_http.pb.go"),
			filepath.Join(baseDir, "testdata", "golden", "backward_compat_http_binding.pb.go"),
			filepath.Join(baseDir, "testdata", "golden", "backward_compat_http_config.pb.go"),
		}

		for _, f := range standardFiles {
			if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
				t.Errorf("backward_compat.proto should still generate: %s", filepath.Base(f))
			}
		}

		t.Log("PASS: Backward-compatible protos generate standard files")
	})
}
