package openapiv3_test

import (
	"strings"
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"gopkg.in/yaml.v3"

	"github.com/SebastienMelki/sebuf/internal/openapiv3"
)

// Test NewGenerator constructor.
func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name     string
		format   openapiv3.OutputFormat
		expected openapiv3.OutputFormat
	}{
		{
			name:     "YAML format",
			format:   openapiv3.FormatYAML,
			expected: openapiv3.FormatYAML,
		},
		{
			name:     "JSON format",
			format:   openapiv3.FormatJSON,
			expected: openapiv3.FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := openapiv3.NewGenerator(tt.format)

			// Check generator is not nil
			if gen == nil {
				t.Fatal("NewGenerator returned nil")
			}

			// Check format is set correctly
			if gen.Format() != tt.expected {
				t.Errorf("Expected format %v, got %v", tt.expected, gen.Format())
			}

			// Check document is initialized
			if gen.Doc() == nil {
				t.Error("Document is nil")
			}

			// Check schemas map is initialized
			if gen.Schemas() == nil {
				t.Error("Schemas map is nil")
			}

			// Check document structure
			if gen.Doc().Version != "3.1.0" {
				t.Errorf("Expected OpenAPI version 3.1.0, got %s", gen.Doc().Version)
			}

			if gen.Doc().Info == nil {
				t.Error("Info is nil")
			}

			if gen.Doc().Info.Title != "Generated API" {
				t.Errorf("Expected default title 'Generated API', got %s", gen.Doc().Info.Title)
			}

			if gen.Doc().Info.Version != "1.0.0" {
				t.Errorf("Expected default version '1.0.0', got %s", gen.Doc().Info.Version)
			}

			// Check paths are initialized
			if gen.Doc().Paths == nil {
				t.Error("Paths is nil")
			}

			if gen.Doc().Paths.PathItems == nil {
				t.Error("PathItems is nil")
			}

			// Check components are initialized
			if gen.Doc().Components == nil {
				t.Error("Components is nil")
			}

			if gen.Doc().Components.Schemas == nil {
				t.Error("Components.Schemas is nil")
			}
		})
	}
}

// Test Render method.
func TestRender(t *testing.T) {
	tests := []struct {
		name      string
		format    openapiv3.OutputFormat
		setupFunc func(*openapiv3.Generator)
		wantErr   bool
		checkFunc func([]byte) error
	}{
		{
			name:   "YAML format",
			format: openapiv3.FormatYAML,
			setupFunc: func(g *openapiv3.Generator) {
				// Add a simple path to make output non-empty
				pathItem := &v3.PathItem{
					Post: &v3.Operation{
						OperationId: "test",
						Summary:     "Test operation",
					},
				}
				g.Doc().Paths.PathItems.Set("/test", pathItem)
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
			format: openapiv3.FormatJSON,
			setupFunc: func(g *openapiv3.Generator) {
				// Add a simple path to make output non-empty
				pathItem := &v3.PathItem{
					Post: &v3.Operation{
						OperationId: "test",
						Summary:     "Test operation",
					},
				}
				g.Doc().Paths.PathItems.Set("/test", pathItem)
			},
			wantErr: false,
			checkFunc: func(data []byte) error {
				// Check that it looks like JSON (starts with '{' and ends with '}')
				str := strings.TrimSpace(string(data))
				if !strings.HasPrefix(str, "{") || !strings.HasSuffix(str, "}") {
					length := 100
					if len(str) < length {
						length = len(str)
					}
					t.Errorf("Output doesn't look like JSON: %s", str[:length])
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := openapiv3.NewGenerator(tt.format)
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
				if checkErr := tt.checkFunc(data); checkErr != nil {
					t.Errorf("Output validation failed: %v", checkErr)
				}
			}
		})
	}
}
