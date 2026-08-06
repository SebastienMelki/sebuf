package openapiv3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	yaml "go.yaml.in/yaml/v4"
)

func TestMapHeaderTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		name       string
		headerType string
		expected   string
	}{
		// String types
		{"string", "string", "string"},
		{"empty defaults to string", "", "string"},
		{"STRING uppercase", "STRING", "string"},
		{"String mixed case", "String", "string"},

		// Integer types
		{"integer", "integer", "integer"},
		{"int", "int", "integer"},
		{"int32", "int32", "integer"},
		{"int64", "int64", "integer"},
		{"INTEGER uppercase", "INTEGER", "integer"},

		// Number types
		{"number", "number", "number"},
		{"float", "float", "number"},
		{"double", "double", "number"},
		{"NUMBER uppercase", "NUMBER", "number"},

		// Boolean types
		{"boolean", "boolean", "boolean"},
		{"bool", "bool", "boolean"},
		{"BOOLEAN uppercase", "BOOLEAN", "boolean"},

		// Array type
		{"array", "array", "array"},
		{"ARRAY uppercase", "ARRAY", "array"},

		// Unknown types default to string
		{"unknown type", "unknown", "string"},
		{"custom type", "custom", "string"},
		{"object type", "object", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapHeaderTypeToOpenAPI(tt.headerType)
			if result != tt.expected {
				t.Errorf("mapHeaderTypeToOpenAPI(%q) = %q, expected %q", tt.headerType, result, tt.expected)
			}
		})
	}
}

func TestNewExampleNodeForSchemaTagsStringExamples(t *testing.T) {
	schema := &base.Schema{Type: []string{"string"}}

	tests := []struct {
		name          string
		example       string
		wantMarshaled string
	}{
		{
			name:          "numeric-looking string remains a YAML string",
			example:       "50000.00",
			wantMarshaled: "\"50000.00\"",
		},
		{
			name:          "timestamp-looking string remains a YAML string",
			example:       "2024-01-01T00:00:00Z",
			wantMarshaled: "\"2024-01-01T00:00:00Z\"",
		},
		{
			name:          "unambiguous string is not over-quoted",
			example:       "ACTIVE",
			wantMarshaled: "ACTIVE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := newExampleNodeForSchema(schema, tt.example)
			if node.Tag != "!!str" {
				t.Fatalf("node.Tag = %q, want %q", node.Tag, "!!str")
			}

			marshaled, err := yaml.Marshal(node)
			if err != nil {
				t.Fatalf("yaml.Marshal returned error: %v", err)
			}
			got := strings.TrimSpace(string(marshaled))
			if got != tt.wantMarshaled {
				t.Fatalf("marshaled example = %q, want %q", got, tt.wantMarshaled)
			}
		})
	}
}

func TestNewExampleNodeForSchemaTagsNativeScalarExamples(t *testing.T) {
	tests := []struct {
		name     string
		schema   *base.Schema
		example  string
		wantTag  string
		wantYAML string
	}{
		{
			name:     "integer example remains numeric",
			schema:   &base.Schema{Type: []string{"integer"}},
			example:  "42",
			wantTag:  "!!int",
			wantYAML: "42",
		},
		{
			name:     "number example remains numeric",
			schema:   &base.Schema{Type: []string{"number"}},
			example:  "42.50",
			wantTag:  "!!float",
			wantYAML: "42.50",
		},
		{
			name:     "boolean example remains boolean",
			schema:   &base.Schema{Type: []string{"boolean"}},
			example:  "true",
			wantTag:  "!!bool",
			wantYAML: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := newExampleNodeForSchema(tt.schema, tt.example)
			if node.Tag != tt.wantTag {
				t.Fatalf("node.Tag = %q, want %q", node.Tag, tt.wantTag)
			}

			marshaled, err := yaml.Marshal(node)
			if err != nil {
				t.Fatalf("yaml.Marshal returned error: %v", err)
			}
			got := strings.TrimSpace(string(marshaled))
			if got != tt.wantYAML {
				t.Fatalf("marshaled example = %q, want %q", got, tt.wantYAML)
			}
		})
	}
}

func BenchmarkMapHeaderTypeToOpenAPI(b *testing.B) {
	types := []string{"string", "integer", "number", "boolean", "array", "", "unknown"}
	for i := range b.N {
		mapHeaderTypeToOpenAPI(types[i%len(types)])
	}
}
