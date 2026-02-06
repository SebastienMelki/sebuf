package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGoGeneratorsProduceIdenticalOneofDiscriminator verifies go-http and go-client
// produce identical oneof_discriminator encoding code.
func TestGoGeneratorsProduceIdenticalOneofDiscriminator(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	httpgenFile := filepath.Join(
		baseDir, "testdata", "golden", "oneof_discriminator_oneof_discriminator.pb.go",
	)
	clientgenFile := filepath.Join(
		baseDir,
		"..",
		"clientgen",
		"testdata",
		"golden",
		"oneof_discriminator_oneof_discriminator.pb.go",
	)

	httpgenContent, httpErr := os.ReadFile(httpgenFile)
	if httpErr != nil {
		t.Fatalf("Failed to read httpgen oneof_discriminator golden file: %v", httpErr)
	}

	clientgenContent, clientErr := os.ReadFile(clientgenFile)
	if clientErr != nil {
		t.Fatalf("Failed to read clientgen oneof_discriminator golden file: %v", clientErr)
	}

	// Normalize the source comment (generator name differs)
	httpgenNormalized := normalizeGeneratorComment(string(httpgenContent), "go-http")
	clientgenNormalized := normalizeGeneratorComment(string(clientgenContent), "go-client")

	if httpgenNormalized != clientgenNormalized {
		t.Errorf("go-http and go-client oneof_discriminator code differs after normalization")
		t.Logf("First difference:\n%s", findFirstDifference(httpgenNormalized, clientgenNormalized))
	} else {
		t.Log("go-http and go-client produce identical oneof_discriminator code")
	}
}

// TestOneofDiscriminatorTypeScriptTypes verifies TypeScript types match Go serialization
// for oneof discriminator fields (discriminated union types).
func TestOneofDiscriminatorTypeScriptTypes(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	tsFile := filepath.Join(
		baseDir, "..", "tsclientgen", "testdata", "golden", "oneof_discriminator_client.ts",
	)

	content, readErr := os.ReadFile(tsFile)
	if readErr != nil {
		t.Fatalf("Failed to read TypeScript oneof_discriminator golden file: %v", readErr)
	}

	tsContent := string(content)

	// FlattenedEvent: discriminated union with "type" discriminator (flattened)
	// Verify "text" variant with flattened fields
	if !strings.Contains(tsContent, `type: "text"`) {
		t.Error("TypeScript FlattenedEvent should have text variant with type: \"text\"")
	}

	// Verify "img" variant (custom oneof_value) with flattened fields
	if !strings.Contains(tsContent, `type: "img"`) {
		t.Error("TypeScript FlattenedEvent should have img variant with type: \"img\" (custom oneof_value)")
	}

	// Verify FlattenedEvent is a discriminated union (uses type alias with &)
	if !strings.Contains(tsContent, "export type FlattenedEvent = FlattenedEventBase & FlattenedEventContent") {
		t.Error("TypeScript FlattenedEvent should be a discriminated union type (intersection)")
	}

	// NestedEvent: discriminated union with "kind" discriminator (non-flattened)
	nestedUnionPattern := regexp.MustCompile(`kind:\s*"text"`)
	if !nestedUnionPattern.MatchString(tsContent) {
		t.Error("TypeScript NestedEvent should have text variant with kind: \"text\"")
	}

	if !strings.Contains(tsContent, `kind: "image"`) {
		t.Error("TypeScript NestedEvent should have image variant with kind: \"image\"")
	}

	if !strings.Contains(tsContent, `kind: "vid"`) {
		t.Error("TypeScript NestedEvent should have vid variant with kind: \"vid\" (custom oneof_value)")
	}

	// PlainEvent: standard interface (no discriminated union)
	if !strings.Contains(tsContent, "export interface PlainEvent") {
		t.Error("TypeScript PlainEvent should be a standard interface (no discriminated union)")
	}

	// PlainEvent should NOT have a discriminator field
	plainEventPattern := regexp.MustCompile(
		`interface PlainEvent \{[^}]*(?:type|kind)\s*:`,
	)
	if plainEventPattern.MatchString(tsContent) {
		t.Error("TypeScript PlainEvent should NOT have a discriminator field (type or kind)")
	}

	t.Log("TypeScript oneof discriminator types correctly match Go serialization")
}

// TestOneofDiscriminatorOpenAPISchemas verifies OpenAPI schemas accurately document
// oneof discriminator (oneOf + discriminator keyword).
func TestOneofDiscriminatorOpenAPISchemas(t *testing.T) {
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
		"OneofDiscriminatorService.openapi.yaml",
	)

	content, readErr := os.ReadFile(yamlFile)
	if readErr != nil {
		t.Fatalf("Failed to read OpenAPI oneof_discriminator golden file: %v", readErr)
	}

	yamlContent := string(content)

	// FlattenedEvent: oneOf with discriminator keyword
	if !strings.Contains(yamlContent, "FlattenedEvent:\n            oneOf:") {
		t.Error("OpenAPI FlattenedEvent should use oneOf schema")
	}

	// FlattenedEvent discriminator propertyName should be "type"
	flattenedDiscIdx := strings.Index(yamlContent, "FlattenedEvent:\n            oneOf:")
	if flattenedDiscIdx >= 0 {
		window := yamlContent[flattenedDiscIdx : flattenedDiscIdx+500]
		if !strings.Contains(window, "propertyName: type") {
			t.Error("OpenAPI FlattenedEvent discriminator propertyName should be 'type'")
		}
		// Verify custom oneof_value mapping: "img" maps to FlattenedEvent_img
		if !strings.Contains(window, "img: '#/components/schemas/FlattenedEvent_img'") {
			t.Error("OpenAPI FlattenedEvent should map 'img' to FlattenedEvent_img variant schema")
		}
		if !strings.Contains(window, "text: '#/components/schemas/FlattenedEvent_text'") {
			t.Error("OpenAPI FlattenedEvent should map 'text' to FlattenedEvent_text variant schema")
		}
	}

	// NestedEvent: oneOf with discriminator keyword
	if !strings.Contains(yamlContent, "discriminator:\n                propertyName: kind") {
		t.Error("OpenAPI NestedEvent discriminator propertyName should be 'kind'")
	}

	// NestedEvent discriminator mapping should include custom "vid" value
	nestedDiscIdx := strings.Index(yamlContent, "NestedEvent:")
	if nestedDiscIdx >= 0 {
		endIdx := nestedDiscIdx + 800
		if endIdx > len(yamlContent) {
			endIdx = len(yamlContent)
		}
		window := yamlContent[nestedDiscIdx:endIdx]
		if !strings.Contains(window, "vid: '#/components/schemas/VideoContent'") {
			t.Error("OpenAPI NestedEvent should map 'vid' to VideoContent (custom oneof_value)")
		}
	}

	// PlainEvent: standard object schema (no discriminator)
	plainIdx := strings.Index(yamlContent, "PlainEvent:")
	if plainIdx >= 0 {
		// Extract a window to check there is no discriminator
		endIdx := plainIdx + 300
		if endIdx > len(yamlContent) {
			endIdx = len(yamlContent)
		}
		window := yamlContent[plainIdx:endIdx]
		if strings.Contains(window, "discriminator:") {
			t.Error("OpenAPI PlainEvent should NOT have a discriminator (no oneof annotation)")
		}
		if strings.Contains(window, "oneOf:") {
			t.Error("OpenAPI PlainEvent should NOT use oneOf (no oneof annotation)")
		}
	}

	t.Log("OpenAPI oneof discriminator schemas correctly match Go serialization")
}

// TestOneofDiscriminatorCrossGeneratorAgreement verifies all 4 generators agree on
// discriminator field names, variant values, and structural behavior for each message.
func TestOneofDiscriminatorCrossGeneratorAgreement(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	// Read all golden files
	goFile := filepath.Join(
		baseDir, "testdata", "golden", "oneof_discriminator_oneof_discriminator.pb.go",
	)
	tsFile := filepath.Join(
		baseDir, "..", "tsclientgen", "testdata", "golden", "oneof_discriminator_client.ts",
	)
	yamlFile := filepath.Join(
		baseDir, "..", "openapiv3", "testdata", "golden", "yaml",
		"OneofDiscriminatorService.openapi.yaml",
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
		message            string
		hasDiscriminator   bool
		discriminatorField string
		variantValues      []string // discriminator values
	}{
		{
			message:            "FlattenedEvent",
			hasDiscriminator:   true,
			discriminatorField: "type",
			variantValues:      []string{"text", "img"},
		},
		{
			message:            "NestedEvent",
			hasDiscriminator:   true,
			discriminatorField: "kind",
			variantValues:      []string{"text", "image", "vid"},
		},
		{
			message:            "PlainEvent",
			hasDiscriminator:   false,
			discriminatorField: "",
			variantValues:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.message, func(t *testing.T) {
			if tc.hasDiscriminator {
				verifyOneofDiscriminatorPresent(
					t,
					tc.message,
					tc.discriminatorField,
					tc.variantValues,
					goStr,
					tsStr,
					yamlStr,
				)
			} else {
				verifyOneofDiscriminatorAbsent(t, tc.message, goStr, tsStr, yamlStr)
			}
		})
	}

	// Verify all golden files exist for cross-generator coverage
	goldenFiles := []string{
		filepath.Join(
			baseDir, "testdata", "golden", "oneof_discriminator_oneof_discriminator.pb.go",
		),
		filepath.Join(
			baseDir, "..", "clientgen", "testdata", "golden",
			"oneof_discriminator_oneof_discriminator.pb.go",
		),
		filepath.Join(
			baseDir, "..", "tsclientgen", "testdata", "golden", "oneof_discriminator_client.ts",
		),
		filepath.Join(
			baseDir, "..", "openapiv3", "testdata", "golden", "yaml",
			"OneofDiscriminatorService.openapi.yaml",
		),
	}

	for _, f := range goldenFiles {
		if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
			t.Errorf("Missing golden file for cross-generator consistency: %s", f)
		}
	}

	t.Log("All 4 generators agree on oneof discriminator structure and values")
}

// verifyOneofDiscriminatorPresent checks that all 4 generators agree on a discriminated message.
func verifyOneofDiscriminatorPresent(
	t *testing.T,
	message, discriminatorField string,
	variantValues []string,
	goStr, tsStr, yamlStr string,
) {
	t.Helper()

	// Go: should have MarshalJSON with discriminator
	goMarshalPattern := regexp.MustCompile(`func \(x \*` + message + `\) MarshalJSON\(\)`)
	if !goMarshalPattern.MatchString(goStr) {
		t.Errorf("Go should generate MarshalJSON for %s", message)
	}

	// Go: should reference the discriminator field name
	goDiscPattern := regexp.MustCompile(`raw\["` + discriminatorField + `"\]`)
	if !goDiscPattern.MatchString(goStr) {
		t.Errorf("Go MarshalJSON for %s should reference discriminator field %q", message, discriminatorField)
	}

	// Go: should include all variant values
	for _, val := range variantValues {
		valPattern := regexp.MustCompile(`json\.Marshal\("` + val + `"\)`)
		if !valPattern.MatchString(goStr) {
			t.Errorf("Go MarshalJSON for %s should include variant value %q", message, val)
		}
	}

	// TypeScript: should have discriminated union or variant references
	for _, val := range variantValues {
		tsValPattern := regexp.MustCompile(discriminatorField + `:\s*"` + val + `"`)
		if !tsValPattern.MatchString(tsStr) {
			t.Errorf("TypeScript %s should include variant %s: %q", message, discriminatorField, val)
		}
	}

	// OpenAPI: should have discriminator keyword with propertyName
	if !strings.Contains(yamlStr, "propertyName: "+discriminatorField) {
		t.Errorf("OpenAPI %s should have discriminator propertyName: %s", message, discriminatorField)
	}

	// OpenAPI: should have mapping entries for each variant value
	for _, val := range variantValues {
		if !strings.Contains(yamlStr, val+": '#/components/schemas/") {
			t.Errorf("OpenAPI %s discriminator mapping should include %q", message, val)
		}
	}
}

// verifyOneofDiscriminatorAbsent checks that a non-discriminated message has no discriminator.
func verifyOneofDiscriminatorAbsent(
	t *testing.T,
	message string,
	goStr, tsStr, yamlStr string,
) {
	t.Helper()

	// Go: should NOT have MarshalJSON
	goMarshalPattern := regexp.MustCompile(`func \(x \*` + message + `\) MarshalJSON\(\)`)
	if goMarshalPattern.MatchString(goStr) {
		t.Errorf("Go should NOT generate MarshalJSON for %s (no discriminator)", message)
	}

	// TypeScript: should be a standard interface
	tsInterfacePattern := regexp.MustCompile(`export interface ` + message + ` \{`)
	if !tsInterfacePattern.MatchString(tsStr) {
		t.Errorf("TypeScript %s should be a standard interface", message)
	}

	// OpenAPI: should NOT have discriminator
	plainIdx := strings.Index(yamlStr, message+":")
	if plainIdx >= 0 {
		endIdx := plainIdx + 300
		if endIdx > len(yamlStr) {
			endIdx = len(yamlStr)
		}
		window := yamlStr[plainIdx:endIdx]
		if strings.Contains(window, "discriminator:") {
			t.Errorf("OpenAPI %s should NOT have discriminator", message)
		}
	}
}
