package openapiv3

import (
	"testing"
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

func BenchmarkMapHeaderTypeToOpenAPI(b *testing.B) {
	types := []string{"string", "integer", "number", "boolean", "array", "", "unknown"}
	for i := range b.N {
		mapHeaderTypeToOpenAPI(types[i%len(types)])
	}
}
