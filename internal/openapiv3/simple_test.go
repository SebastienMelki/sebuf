package openapiv3

import (
	"strings"
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"gopkg.in/yaml.v3"
)

// Test NewGenerator constructor
func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name     string
		format   OutputFormat
		expected OutputFormat
	}{
		{
			name:     "YAML format",
			format:   FormatYAML,
			expected: FormatYAML,
		},
		{
			name:     "JSON format",
			format:   FormatJSON,
			expected: FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.format)

			// Check generator is not nil
			if gen == nil {
				t.Fatal("NewGenerator returned nil")
			}

			// Check format is set correctly
			if gen.format != tt.expected {
				t.Errorf("Expected format %v, got %v", tt.expected, gen.format)
			}

			// Check document is initialized
			if gen.doc == nil {
				t.Error("Document is nil")
			}

			// Check schemas map is initialized
			if gen.schemas == nil {
				t.Error("Schemas map is nil")
			}

			// Check document structure
			if gen.doc.Version != "3.1.0" {
				t.Errorf("Expected OpenAPI version 3.1.0, got %s", gen.doc.Version)
			}

			if gen.doc.Info == nil {
				t.Error("Info is nil")
			}

			if gen.doc.Info.Title != "Generated API" {
				t.Errorf("Expected default title 'Generated API', got %s", gen.doc.Info.Title)
			}

			if gen.doc.Info.Version != "1.0.0" {
				t.Errorf("Expected default version '1.0.0', got %s", gen.doc.Info.Version)
			}

			// Check paths are initialized
			if gen.doc.Paths == nil {
				t.Error("Paths is nil")
			}

			if gen.doc.Paths.PathItems == nil {
				t.Error("PathItems is nil")
			}

			// Check components are initialized
			if gen.doc.Components == nil {
				t.Error("Components is nil")
			}

			if gen.doc.Components.Schemas == nil {
				t.Error("Components.Schemas is nil")
			}
		})
	}
}

// Test Render method
func TestRender(t *testing.T) {
	tests := []struct {
		name       string
		format     OutputFormat
		setupFunc  func(*Generator)
		wantErr    bool
		checkFunc  func([]byte) error
	}{
		{
			name:   "YAML format",
			format: FormatYAML,
			setupFunc: func(g *Generator) {
				// Add a simple path to make output non-empty
				pathItem := &v3.PathItem{
					Post: &v3.Operation{
						OperationId: "test",
						Summary:     "Test operation",
					},
				}
				g.doc.Paths.PathItems.Set("/test", pathItem)
			},
			wantErr: false,
			checkFunc: func(data []byte) error {
				// Check that it's valid YAML by unmarshaling
				var result interface{}
				if err := yaml.Unmarshal(data, &result); err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:   "JSON format",
			format: FormatJSON,
			setupFunc: func(g *Generator) {
				// Add a simple path to make output non-empty
				pathItem := &v3.PathItem{
					Post: &v3.Operation{
						OperationId: "test",
						Summary:     "Test operation",
					},
				}
				g.doc.Paths.PathItems.Set("/test", pathItem)
			},
			wantErr: false,
			checkFunc: func(data []byte) error {
				// Check that it looks like JSON (starts with '{' and ends with '}')
				str := strings.TrimSpace(string(data))
				if !strings.HasPrefix(str, "{") || !strings.HasSuffix(str, "}") {
					t.Errorf("Output doesn't look like JSON: %s", str[:min(100, len(str))])
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.format)
			if tt.setupFunc != nil {
				tt.setupFunc(gen)
			}

			data, err := gen.Render()

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(data) == 0 {
				t.Error("Render() returned empty data")
			}

			if tt.checkFunc != nil {
				if err := tt.checkFunc(data); err != nil {
					t.Errorf("Output validation failed: %v", err)
				}
			}
		})
	}
}

// Helper function for min (not available in older Go versions)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}