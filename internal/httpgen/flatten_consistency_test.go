package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGoGeneratorsProduceIdenticalFlatten verifies go-http and go-client
// produce identical flatten encoding code.
func TestGoGeneratorsProduceIdenticalFlatten(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(baseDir, "testdata", "golden", "flatten_flatten.pb.go")
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"flatten_flatten.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen flatten golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen flatten golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client flatten code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical flatten code")
	}
}

// TestFlattenTypeScriptTypes verifies TypeScript types match Go serialization
// for flatten fields (inlined child fields with prefixes).
func TestFlattenTypeScriptTypes(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "flatten_client.ts")

	content, readErr := os.ReadFile(tsFile)
	if readErr != nil {
		t.Fatalf("Failed to read TypeScript flatten golden file: %v", readErr)
	}

	tsContent := string(content)

	// SimpleFlatten: street, city, zip at top level (no nested address)
	simpleFlattenFields := []string{"street: string", "city: string", "zip: string"}
	for _, field := range simpleFlattenFields {
		if !containsInInterface(tsContent, "SimpleFlatten", field) {
			t.Errorf("TypeScript SimpleFlatten should contain %q at top level", field)
		}
	}

	// SimpleFlatten should NOT have a nested address property
	if containsInInterface(tsContent, "SimpleFlatten", "address") {
		t.Error("TypeScript SimpleFlatten should NOT have nested 'address' property")
	}

	// DualFlatten: billing_street, billing_city, etc. with prefixes
	dualFlattenFields := []string{
		"billing_street: string",
		"billing_city: string",
		"billing_zip: string",
		"shipping_street: string",
		"shipping_city: string",
		"shipping_zip: string",
	}
	for _, field := range dualFlattenFields {
		if !containsInInterface(tsContent, "DualFlatten", field) {
			t.Errorf("TypeScript DualFlatten should contain %q", field)
		}
	}

	// MixedFlatten: flattened fields AND a nested contact property
	mixedFlattenFields := []string{"street: string", "city: string", "zip: string"}
	for _, field := range mixedFlattenFields {
		if !containsInInterface(tsContent, "MixedFlatten", field) {
			t.Errorf("TypeScript MixedFlatten should contain flattened field %q", field)
		}
	}
	if !containsInInterface(tsContent, "MixedFlatten", "contact") {
		t.Error("TypeScript MixedFlatten should have nested 'contact' property")
	}

	// PlainNested: standard nested address property
	if !containsInInterface(tsContent, "PlainNested", "address") {
		t.Error("TypeScript PlainNested should have nested 'address' property")
	}

	t.Log("TypeScript flatten types correctly match Go serialization")
}

// TestFlattenOpenAPISchemas verifies OpenAPI schemas accurately document flatten
// (correct property names and types using allOf composition).
func TestFlattenOpenAPISchemas(t *testing.T) {
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
		"FlattenService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI flatten golden file: %v", readErr)
	}

	yamlContent := string(content)

	// SimpleFlatten: should have street, city, zip as direct properties (via allOf)
	verifyOpenAPIFlattenedProperties(t, yamlContent, "SimpleFlatten",
		[]string{"street", "city", "zip"})

	// SimpleFlatten: should use allOf for flattening
	if !strings.Contains(yamlContent, "SimpleFlatten:\n            allOf:") {
		t.Error("OpenAPI SimpleFlatten should use allOf for flattened schema")
	}

	// DualFlatten: should have billing_street, billing_city, etc.
	verifyOpenAPIFlattenedProperties(t, yamlContent, "DualFlatten",
		[]string{"billing_street", "billing_city", "billing_zip",
			"shipping_street", "shipping_city", "shipping_zip"})

	// DualFlatten: should use allOf
	if !strings.Contains(yamlContent, "DualFlatten:\n            allOf:") {
		t.Error("OpenAPI DualFlatten should use allOf for flattened schema")
	}

	// MixedFlatten: should have both flattened and nested properties
	verifyOpenAPIFlattenedProperties(t, yamlContent, "MixedFlatten",
		[]string{"street", "city", "zip"})

	// MixedFlatten: should also have contact as $ref
	mixedIdx := strings.Index(yamlContent, "MixedFlatten:")
	if mixedIdx >= 0 {
		endIdx := mixedIdx + 500
		if endIdx > len(yamlContent) {
			endIdx = len(yamlContent)
		}
		window := yamlContent[mixedIdx:endIdx]
		if !strings.Contains(window, "$ref: '#/components/schemas/ContactInfo'") {
			t.Error("OpenAPI MixedFlatten should have $ref to ContactInfo for non-flattened contact")
		}
	}

	// PlainNested: should have $ref to Address (no flattening)
	plainIdx := strings.Index(yamlContent, "PlainNested:")
	if plainIdx >= 0 {
		endIdx := plainIdx + 300
		if endIdx > len(yamlContent) {
			endIdx = len(yamlContent)
		}
		window := yamlContent[plainIdx:endIdx]
		if !strings.Contains(window, "$ref: '#/components/schemas/Address'") {
			t.Error("OpenAPI PlainNested should have $ref to Address (no flattening)")
		}
		if strings.Contains(window, "allOf:") {
			t.Error("OpenAPI PlainNested should NOT use allOf (no flatten annotation)")
		}
	}

	t.Log("OpenAPI flatten schemas correctly match Go serialization")
}

// TestFlattenCrossGeneratorAgreement verifies all 4 generators agree on flatten
// field names and structural behavior for each message.
func TestFlattenCrossGeneratorAgreement(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	// Read all golden files
	goFile := filepath.Join(baseDir, "testdata", "golden", "flatten_flatten.pb.go")
	tsFile := filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "flatten_client.ts")
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "FlattenService.openapi.yaml",
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
		message        string
		hasFlatten     bool
		flattenedNames []string // expected flattened field names in JSON
	}{
		{
			message:        "SimpleFlatten",
			hasFlatten:     true,
			flattenedNames: []string{"street", "city", "zip"},
		},
		{
			message:    "DualFlatten",
			hasFlatten: true,
			flattenedNames: []string{
				"billing_street",
				"billing_city",
				"billing_zip",
				"shipping_street",
				"shipping_city",
				"shipping_zip",
			},
		},
		{
			message:    "MixedFlatten",
			hasFlatten: true,
			flattenedNames: []string{
				"street", "city", "zip",
			},
		},
		{
			message:        "PlainNested",
			hasFlatten:     false,
			flattenedNames: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.message, func(t *testing.T) {
			if tc.hasFlatten {
				verifyFlattenPresent(t, tc.message, tc.flattenedNames, goStr, tsStr, yamlStr)
			} else {
				verifyFlattenAbsent(t, tc.message, goStr, tsStr, yamlStr)
			}
		})
	}

	// Verify all golden files exist for cross-generator coverage
	goldenFiles := []string{
		filepath.Join(baseDir, "testdata", "golden", "flatten_flatten.pb.go"),
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "flatten_flatten.pb.go"),
		filepath.Join(baseDir, "..", "tsclientgen", "testdata", "golden", "flatten_client.ts"),
		filepath.Join(
			baseDir, "..", "openapiv3", "testdata", "golden", "yaml", "FlattenService.openapi.yaml",
		),
	}

	for _, f := range goldenFiles {
		if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
			t.Errorf("Missing golden file for cross-generator consistency: %s", f)
		}
	}

	t.Log("All 4 generators agree on flatten structure and field names")
}

// verifyFlattenPresent checks that all 4 generators agree on a flattened message.
func verifyFlattenPresent(
	t *testing.T,
	message string,
	flattenedNames []string,
	goStr, tsStr, yamlStr string,
) {
	t.Helper()

	// Go: should have MarshalJSON for flatten messages
	goMarshalPattern := regexp.MustCompile(`func \(x \*` + message + `\) MarshalJSON\(\)`)
	if !goMarshalPattern.MatchString(goStr) {
		t.Errorf("Go should generate MarshalJSON for %s", message)
	}

	// Go: should reference expected flattened field names
	for _, name := range flattenedNames {
		if !strings.Contains(goStr, `"`+name+`"`) {
			t.Errorf("Go flatten code for %s should reference field name %q", message, name)
		}
	}

	// TypeScript: should have flattened fields at interface level
	for _, name := range flattenedNames {
		tsFieldPattern := regexp.MustCompile(name + `:\s*\w+`)
		if !tsFieldPattern.MatchString(tsStr) {
			t.Errorf("TypeScript %s should have flattened field %q at interface level", message, name)
		}
	}

	// OpenAPI: should have flattened fields as properties
	for _, name := range flattenedNames {
		if !strings.Contains(yamlStr, name+":") {
			t.Errorf("OpenAPI %s should have flattened property %q", message, name)
		}
	}
}

// verifyFlattenAbsent checks that a non-flattened message has standard nesting.
func verifyFlattenAbsent(
	t *testing.T,
	message string,
	goStr, tsStr, yamlStr string,
) {
	t.Helper()

	// Go: should NOT have MarshalJSON
	goMarshalPattern := regexp.MustCompile(`func \(x \*` + message + `\) MarshalJSON\(\)`)
	if goMarshalPattern.MatchString(goStr) {
		t.Errorf("Go should NOT generate MarshalJSON for %s (no flatten)", message)
	}

	// TypeScript: should have nested address property (not flattened)
	if !containsInInterface(tsStr, message, "address") {
		t.Errorf("TypeScript %s should have nested 'address' property (not flattened)", message)
	}

	// OpenAPI: should NOT use allOf (no flatten)
	plainIdx := strings.Index(yamlStr, message+":")
	if plainIdx >= 0 {
		endIdx := plainIdx + 300
		if endIdx > len(yamlStr) {
			endIdx = len(yamlStr)
		}
		window := yamlStr[plainIdx:endIdx]
		if strings.Contains(window, "allOf:") {
			t.Errorf("OpenAPI %s should NOT use allOf (no flatten annotation)", message)
		}
	}
}

// containsInInterface checks if a TypeScript interface contains a field.
// It finds the interface declaration and checks within its body.
func containsInInterface(tsContent, interfaceName, field string) bool {
	// Find the interface
	pattern := regexp.MustCompile(`export interface ` + interfaceName + ` \{([^}]*)`)
	match := pattern.FindString(tsContent)
	if match == "" {
		return false
	}
	return strings.Contains(match, field)
}

// verifyOpenAPIFlattenedProperties checks that an OpenAPI schema contains expected
// flattened property names.
func verifyOpenAPIFlattenedProperties(t *testing.T, yamlContent, schemaName string, properties []string) {
	t.Helper()

	schemaIdx := strings.Index(yamlContent, schemaName+":")
	if schemaIdx < 0 {
		t.Errorf("OpenAPI schema %s not found", schemaName)
		return
	}

	// Extract a generous window after the schema definition
	endIdx := schemaIdx + 800
	if endIdx > len(yamlContent) {
		endIdx = len(yamlContent)
	}
	window := yamlContent[schemaIdx:endIdx]

	for _, prop := range properties {
		if !strings.Contains(window, prop+":") {
			t.Errorf("OpenAPI %s should have flattened property %q", schemaName, prop)
		}
	}
}
