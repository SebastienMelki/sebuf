package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGoGeneratorsProduceIdenticalTimestampFormat verifies go-http and go-client
// produce identical timestamp_format encoding code.
func TestGoGeneratorsProduceIdenticalTimestampFormat(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "timestamp_format_timestamp_format.pb.go")
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"timestamp_format_timestamp_format.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen timestamp_format golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen timestamp_format golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client timestamp_format encoding code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical timestamp_format encoding code")
	}
}

// TestTimestampFormatTypeScriptTypes verifies TypeScript types match Go serialization
// for timestamp format fields.
func TestTimestampFormatTypeScriptTypes(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "timestamp_format_client.ts")

	content, readErr := os.ReadFile(tsFile)
	if readErr != nil {
		t.Fatalf("Failed to read TypeScript timestamp_format golden file: %v", readErr)
	}

	tsContent := string(content)

	// Verify UNIX_SECONDS and UNIX_MILLIS fields use TypeScript number type
	numberFields := []string{"unixSecondsTs", "unixMillisTs"}
	for _, field := range numberFields {
		// Match field: number or field?: number
		pattern := regexp.MustCompile(field + `\??:\s*number`)
		if !pattern.MatchString(tsContent) {
			t.Errorf("TypeScript timestamp NUMBER field %q should have type 'number'", field)
		}
	}

	// Verify RFC3339 and DATE fields use TypeScript string type
	stringFields := []string{"defaultTs", "rfc3339Ts", "dateTs"}
	for _, field := range stringFields {
		// Match field: string or field?: string
		pattern := regexp.MustCompile(field + `\??:\s*string`)
		if !pattern.MatchString(tsContent) {
			t.Errorf("TypeScript timestamp STRING field %q should have type 'string'", field)
		}
	}

	t.Log("TypeScript timestamp types correctly match Go serialization")
}

// TestTimestampFormatOpenAPISchemas verifies OpenAPI schemas accurately document
// timestamp format types and formats.
func TestTimestampFormatOpenAPISchemas(t *testing.T) {
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
		"TimestampFormatService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI timestamp_format golden file: %v", readErr)
	}

	yamlContent := string(content)

	// Verify UNIX_SECONDS field has type: integer, format: unix-timestamp
	if !strings.Contains(
		yamlContent,
		"unixSecondsTs:\n                    type: integer\n                    format: unix-timestamp",
	) {
		t.Error("OpenAPI unixSecondsTs should have type: integer, format: unix-timestamp")
	}

	// Verify UNIX_MILLIS field has type: integer, format: unix-timestamp-ms
	if !strings.Contains(
		yamlContent,
		"unixMillisTs:\n                    type: integer\n                    format: unix-timestamp-ms",
	) {
		t.Error("OpenAPI unixMillisTs should have type: integer, format: unix-timestamp-ms")
	}

	// Verify DATE field has type: string, format: date
	if !strings.Contains(yamlContent, "dateTs:\n                    type: string\n                    format: date") {
		t.Error("OpenAPI dateTs should have type: string, format: date")
	}

	// Verify default (RFC3339) field has type: string, format: date-time
	if !strings.Contains(
		yamlContent,
		"defaultTs:\n                    type: string\n                    format: date-time",
	) {
		t.Error("OpenAPI defaultTs should have type: string, format: date-time")
	}

	// Verify explicit RFC3339 field has type: string, format: date-time
	if !strings.Contains(
		yamlContent,
		"rfc3339Ts:\n                    type: string\n                    format: date-time",
	) {
		t.Error("OpenAPI rfc3339Ts should have type: string, format: date-time")
	}

	t.Log("OpenAPI timestamp format schemas correctly match Go serialization")
}

// TestTimestampFormatCrossGeneratorAgreement verifies all 4 generators agree on
// the type and format for each timestamp format variant.
func TestTimestampFormatCrossGeneratorAgreement(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	// Read all golden files
	goFile := filepath.Join(baseDir, "testdata", "golden", "timestamp_format_timestamp_format.pb.go")
	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "timestamp_format_client.ts")
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "TimestampFormatService.openapi.yaml",
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
		format        string
		jsonField     string
		goHasMarshal  bool   // Whether MarshalJSON modifies this field
		tsType        string // "number" or "string"
		openapiType   string // "integer" or "string"
		openapiFormat string // "unix-timestamp", "date-time", etc.
	}{
		{"UNIX_SECONDS", "unixSecondsTs", true, "number", "integer", "unix-timestamp"},
		{"UNIX_MILLIS", "unixMillisTs", true, "number", "integer", "unix-timestamp-ms"},
		{"DATE", "dateTs", true, "string", "string", "date"},
		{"RFC3339", "rfc3339Ts", false, "string", "string", "date-time"},
		{"default", "defaultTs", false, "string", "string", "date-time"},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			// Verify Go MarshalJSON modifies (or not) the field
			goFieldPattern := regexp.MustCompile(`raw\["` + tc.jsonField + `"\]`)
			goHasMarshalModification := goFieldPattern.MatchString(goStr)
			if tc.goHasMarshal != goHasMarshalModification {
				t.Errorf("Go MarshalJSON %s field %q: expected modified=%v, got modified=%v",
					tc.format, tc.jsonField, tc.goHasMarshal, goHasMarshalModification)
			}

			// Verify TypeScript type matches expected
			tsFieldPattern := regexp.MustCompile(tc.jsonField + `\??:\s*` + tc.tsType)
			if !tsFieldPattern.MatchString(tsStr) {
				t.Errorf("TypeScript %s field %q: expected type %q", tc.format, tc.jsonField, tc.tsType)
			}

			// Verify OpenAPI type and format match expected
			openapiTypePattern := tc.jsonField + ":\n                    type: " + tc.openapiType
			if !strings.Contains(yamlStr, openapiTypePattern) {
				t.Errorf("OpenAPI %s field %q: expected type %q", tc.format, tc.jsonField, tc.openapiType)
			}

			openapiFormatPattern := tc.jsonField + ":\n                    type: " + tc.openapiType +
				"\n                    format: " + tc.openapiFormat
			if !strings.Contains(yamlStr, openapiFormatPattern) {
				t.Errorf("OpenAPI %s field %q: expected format %q", tc.format, tc.jsonField, tc.openapiFormat)
			}
		})
	}

	// Verify all golden files exist for cross-generator coverage
	goldenFiles := []string{
		filepath.Join(baseDir, "testdata", "golden", "timestamp_format_timestamp_format.pb.go"),
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "timestamp_format_timestamp_format.pb.go"),
		filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "timestamp_format_client.ts"),
		filepath.Join(
			baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "TimestampFormatService.openapi.yaml",
		),
	}

	for _, f := range goldenFiles {
		if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
			t.Errorf("Missing golden file for cross-generator consistency: %s", f)
		}
	}

	t.Log("All 4 generators agree on timestamp format types and formats")
}
