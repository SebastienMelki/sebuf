package openapiv3_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestJSONMappingCompositionFlattensCommonFieldsAndPreservesOneofDiscriminator(t *testing.T) {
	pluginPath := "./protoc-gen-openapiv3-composition-test"
	buildCmd := exec.Command("go", "build", "-o", pluginPath, "../../cmd/protoc-gen-openapiv3")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build plugin: %v\nOutput: %s", err, string(output))
	}
	defer os.Remove(pluginPath)

	tempDir := t.TempDir()
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-openapiv3="+pluginPath,
		"--openapiv3_out="+tempDir,
		"--openapiv3_opt=format=json",
		"--proto_path=testdata/proto",
		"--proto_path=../../proto",
		"testdata/proto/json_mapping_composition.proto",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("protoc failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	generatedPath := filepath.Join(tempDir, "JSONMappingCompositionService.openapi.json")
	content, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("failed to read generated OpenAPI file: %v", err)
	}

	var doc map[string]any
	if unmarshalErr := json.Unmarshal(content, &doc); unmarshalErr != nil {
		t.Fatalf("generated OpenAPI JSON is invalid: %v", unmarshalErr)
	}

	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)

	nestedSchema := schemas["FlattenWithNestedOneof"].(map[string]any)
	assertSchemaHasDiscriminator(t, nestedSchema, "kind")
	assertAllOfProperties(t, nestedSchema, []string{"id", "kind", "audit_createdBy", "audit_version"})
	assertAllOfDoesNotHaveProperties(t, nestedSchema, []string{"audit"})

	flatSchema := schemas["FlattenWithFlatOneof"].(map[string]any)
	assertSchemaHasDiscriminator(t, flatSchema, "type")
	assertOneOfVariantHasProperties(t, schemas, flatSchema, []string{"id", "type", "audit_createdBy", "audit_version"})
	assertOneOfVariantDoesNotHaveProperties(t, schemas, flatSchema, []string{"audit"})
}

func assertSchemaHasDiscriminator(t *testing.T, schema map[string]any, propertyName string) {
	t.Helper()
	discriminator, ok := schema["discriminator"].(map[string]any)
	if !ok {
		t.Fatalf("schema missing discriminator: %#v", schema)
	}
	if got := discriminator["propertyName"]; got != propertyName {
		t.Fatalf("discriminator propertyName = %v, want %q", got, propertyName)
	}
}

func assertAllOfProperties(t *testing.T, schema map[string]any, want []string) {
	t.Helper()
	props := collectAllOfProperties(t, schema)
	for _, name := range want {
		if _, ok := props[name]; !ok {
			t.Fatalf("schema allOf missing property %q; properties: %v", name, mapKeys(props))
		}
	}
}

func assertAllOfDoesNotHaveProperties(t *testing.T, schema map[string]any, unwanted []string) {
	t.Helper()
	props := collectAllOfProperties(t, schema)
	for _, name := range unwanted {
		if _, ok := props[name]; ok {
			t.Fatalf("schema allOf unexpectedly has property %q; properties: %v", name, mapKeys(props))
		}
	}
}

func collectAllOfProperties(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	allOf, ok := schema["allOf"].([]any)
	if !ok {
		t.Fatalf("schema missing allOf: %#v", schema)
	}
	props := map[string]any{}
	for _, item := range allOf {
		itemMap := item.(map[string]any)
		itemProps, _ := itemMap["properties"].(map[string]any)
		for name, prop := range itemProps {
			props[name] = prop
		}
	}
	return props
}

func assertOneOfVariantHasProperties(t *testing.T, schemas map[string]any, schema map[string]any, want []string) {
	t.Helper()
	props := collectFirstOneOfVariantProperties(t, schemas, schema)
	for _, name := range want {
		if _, ok := props[name]; !ok {
			t.Fatalf("oneOf variant missing property %q; properties: %v", name, mapKeys(props))
		}
	}
}

func assertOneOfVariantDoesNotHaveProperties(
	t *testing.T, schemas map[string]any, schema map[string]any, unwanted []string,
) {
	t.Helper()
	props := collectFirstOneOfVariantProperties(t, schemas, schema)
	for _, name := range unwanted {
		if _, ok := props[name]; ok {
			t.Fatalf("oneOf variant unexpectedly has property %q; properties: %v", name, mapKeys(props))
		}
	}
}

func collectFirstOneOfVariantProperties(t *testing.T, schemas map[string]any, schema map[string]any) map[string]any {
	t.Helper()
	oneOf, ok := schema["oneOf"].([]any)
	if !ok || len(oneOf) == 0 {
		t.Fatalf("schema missing oneOf variants: %#v", schema)
	}
	ref := oneOf[0].(map[string]any)["$ref"].(string)
	const prefix = "#/components/schemas/"
	if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
		t.Fatalf("unexpected oneOf ref: %q", ref)
	}
	variant := schemas[ref[len(prefix):]].(map[string]any)
	props, ok := variant["properties"].(map[string]any)
	if !ok {
		t.Fatalf("oneOf variant missing properties: %#v", variant)
	}
	return props
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
