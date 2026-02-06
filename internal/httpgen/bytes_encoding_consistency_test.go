package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGoGeneratorsProduceIdenticalBytesEncoding verifies go-http and go-client
// produce identical bytes_encoding code.
func TestGoGeneratorsProduceIdenticalBytesEncoding(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "bytes_encoding_bytes_encoding.pb.go")
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"bytes_encoding_bytes_encoding.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen bytes_encoding golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen bytes_encoding golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client bytes_encoding code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical bytes_encoding code")
	}
}

// TestBytesEncodingTypeScriptTypes verifies all bytes encoding variants produce
// string type in TypeScript (bytes are always strings regardless of encoding).
func TestBytesEncodingTypeScriptTypes(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "bytes_encoding_client.ts")

	content, readErr := os.ReadFile(tsFile)
	if readErr != nil {
		t.Fatalf("Failed to read TypeScript bytes_encoding golden file: %v", readErr)
	}

	tsContent := string(content)

	// ALL bytes encoding variants should produce string type in TypeScript
	bytesFields := []string{
		"defaultData",
		"base64Data",
		"base64RawData",
		"base64urlData",
		"base64urlRawData",
		"hexData",
	}

	for _, field := range bytesFields {
		// Match field: string (not optional -- bytes fields are always present)
		pattern := regexp.MustCompile(field + `:\s*string`)
		if !pattern.MatchString(tsContent) {
			t.Errorf("TypeScript bytes field %q should have type 'string'", field)
		}

		// Verify no field uses number type (bytes should never be numeric)
		numberPattern := regexp.MustCompile(field + `:\s*number`)
		if numberPattern.MatchString(tsContent) {
			t.Errorf("TypeScript bytes field %q should NOT have type 'number'", field)
		}
	}

	t.Log("TypeScript bytes encoding types correctly use string for all variants")
}

// TestBytesEncodingOpenAPISchemas verifies OpenAPI schemas accurately document
// bytes encoding formats and patterns.
func TestBytesEncodingOpenAPISchemas(t *testing.T) {
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
		"BytesEncodingService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI bytes_encoding golden file: %v", readErr)
	}

	yamlContent := string(content)

	// Verify all bytes fields have type: string
	bytesFields := []string{
		"defaultData",
		"base64Data",
		"base64RawData",
		"base64urlData",
		"base64urlRawData",
		"hexData",
	}
	for _, field := range bytesFields {
		if !strings.Contains(yamlContent, field+":\n                    type: string") {
			t.Errorf("OpenAPI bytes field %q should have type: string", field)
		}
	}

	// Verify default (BASE64) has format: byte
	if !strings.Contains(
		yamlContent,
		"defaultData:\n                    type: string\n                    format: byte",
	) {
		t.Error("OpenAPI defaultData should have format: byte")
	}

	// Verify explicit BASE64 has format: byte
	if !strings.Contains(
		yamlContent,
		"base64Data:\n                    type: string\n                    format: byte",
	) {
		t.Error("OpenAPI base64Data should have format: byte")
	}

	// Verify BASE64_RAW has format: byte (same encoding, just no padding)
	if !strings.Contains(
		yamlContent,
		"base64RawData:\n                    type: string\n                    format: byte",
	) {
		t.Error("OpenAPI base64RawData should have format: byte")
	}

	// Verify BASE64URL has format: base64url
	if !strings.Contains(
		yamlContent,
		"base64urlData:\n                    type: string\n                    format: base64url",
	) {
		t.Error("OpenAPI base64urlData should have format: base64url")
	}

	// Verify BASE64URL_RAW has format: base64url
	if !strings.Contains(
		yamlContent,
		"base64urlRawData:\n                    type: string\n                    format: base64url",
	) {
		t.Error("OpenAPI base64urlRawData should have format: base64url")
	}

	// Verify HEX has format: hex and hex validation pattern
	if !strings.Contains(
		yamlContent,
		"hexData:\n                    type: string\n                    pattern: ^[0-9a-fA-F]*$",
	) {
		t.Error("OpenAPI hexData should have hex validation pattern")
	}
	if !strings.Contains(yamlContent, "format: hex") {
		t.Error("OpenAPI hexData should have format: hex")
	}

	t.Log("OpenAPI bytes encoding schemas correctly match Go serialization")
}

// TestBytesEncodingCrossGeneratorAgreement verifies all 4 generators agree on
// the type and format for each bytes encoding variant.
func TestBytesEncodingCrossGeneratorAgreement(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	// Read all golden files
	goFile := filepath.Join(baseDir, "testdata", "golden", "bytes_encoding_bytes_encoding.pb.go")
	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "bytes_encoding_client.ts")
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "BytesEncodingService.openapi.yaml",
	)

	goContent, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatalf("Failed to read Go golden file: %v", err)
	}
	tsContent, err := os.ReadFile(tsFile)
	if err != nil {
		t.Fatalf("Failed to read TypeScript golden file: %v", err)
	}
	yamlContent, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to read OpenAPI golden file: %v", err)
	}

	goStr := string(goContent)
	tsStr := string(tsContent)
	yamlStr := string(yamlContent)

	testCases := []struct {
		encoding      string
		jsonField     string
		goHasMarshal  bool   // Whether MarshalJSON modifies this field
		tsType        string // always "string" for bytes
		openapiType   string // always "string" for bytes
		openapiFormat string // "byte", "hex", "base64url"
	}{
		{"HEX", "hexData", true, "string", "string", "hex"},
		{"BASE64_RAW", "base64RawData", true, "string", "string", "byte"},
		{"BASE64URL", "base64urlData", true, "string", "string", "base64url"},
		{"BASE64URL_RAW", "base64urlRawData", true, "string", "string", "base64url"},
		{"BASE64", "base64Data", false, "string", "string", "byte"},
		{"default", "defaultData", false, "string", "string", "byte"},
	}

	for _, tc := range testCases {
		t.Run(tc.encoding, func(t *testing.T) {
			// Verify Go MarshalJSON modifies (or not) the field
			goFieldPattern := regexp.MustCompile(`raw\["` + tc.jsonField + `"\]`)
			goHasMarshalModification := goFieldPattern.MatchString(goStr)
			if tc.goHasMarshal != goHasMarshalModification {
				t.Errorf("Go MarshalJSON %s field %q: expected modified=%v, got modified=%v",
					tc.encoding, tc.jsonField, tc.goHasMarshal, goHasMarshalModification)
			}

			// Verify TypeScript type matches expected (always string for bytes)
			tsFieldPattern := regexp.MustCompile(tc.jsonField + `:\s*` + tc.tsType)
			if !tsFieldPattern.MatchString(tsStr) {
				t.Errorf("TypeScript %s field %q: expected type %q", tc.encoding, tc.jsonField, tc.tsType)
			}

			// Verify OpenAPI type matches expected (always string for bytes)
			openapiTypePattern := tc.jsonField + ":\n                    type: " + tc.openapiType
			if !strings.Contains(yamlStr, openapiTypePattern) {
				t.Errorf("OpenAPI %s field %q: expected type %q", tc.encoding, tc.jsonField, tc.openapiType)
			}

			// Verify OpenAPI format matches expected by checking the field's YAML block
			// Fields in the schema are at 20-space indent, format follows on next lines
			expectedFormatLine := "format: " + tc.openapiFormat
			fieldIdx := strings.Index(yamlStr, tc.jsonField+":\n                    type: "+tc.openapiType)
			if fieldIdx < 0 {
				t.Errorf("OpenAPI %s field %q: field definition not found", tc.encoding, tc.jsonField)
			} else {
				// Extract a window after the field to check for format
				endIdx := fieldIdx + 200
				if endIdx > len(yamlStr) {
					endIdx = len(yamlStr)
				}
				fieldWindow := yamlStr[fieldIdx:endIdx]
				if !strings.Contains(fieldWindow, expectedFormatLine) {
					t.Errorf("OpenAPI %s field %q: expected format %q in:\n%s",
						tc.encoding, tc.jsonField, tc.openapiFormat, fieldWindow)
				}
			}
		})
	}

	// Verify all golden files exist for cross-generator coverage
	goldenFiles := []string{
		filepath.Join(baseDir, "testdata", "golden", "bytes_encoding_bytes_encoding.pb.go"),
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "bytes_encoding_bytes_encoding.pb.go"),
		filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "bytes_encoding_client.ts"),
		filepath.Join(
			baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "BytesEncodingService.openapi.yaml",
		),
	}

	for _, f := range goldenFiles {
		if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
			t.Errorf("Missing golden file for cross-generator consistency: %s", f)
		}
	}

	t.Log("All 4 generators agree on bytes encoding types and formats")
}
