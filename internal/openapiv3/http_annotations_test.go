package openapiv3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"

	"github.com/SebastienMelki/sebuf/http"
)

// Test buildHTTPPath function
func TestBuildHTTPPath(t *testing.T) {
	tests := []struct {
		name        string
		servicePath string
		methodPath  string
		expected    string
	}{
		{
			name:        "Both paths provided",
			servicePath: "/api/v1",
			methodPath:  "/users",
			expected:    "/api/v1/users",
		},
		{
			name:        "Service path with trailing slash",
			servicePath: "/api/v1/",
			methodPath:  "/users",
			expected:    "/api/v1/users",
		},
		{
			name:        "Method path without leading slash",
			servicePath: "/api/v1",
			methodPath:  "users",
			expected:    "/api/v1/users",
		},
		{
			name:        "Both paths with slashes",
			servicePath: "/api/v1/",
			methodPath:  "/users",
			expected:    "/api/v1/users",
		},
		{
			name:        "Only service path",
			servicePath: "/api/v1",
			methodPath:  "",
			expected:    "/api/v1",
		},
		{
			name:        "Only method path",
			servicePath: "",
			methodPath:  "/users",
			expected:    "/users",
		},
		{
			name:        "Both empty",
			servicePath: "",
			methodPath:  "",
			expected:    "/",
		},
		{
			name:        "Service path without leading slash",
			servicePath: "api/v1",
			methodPath:  "/users",
			expected:    "/api/v1/users",
		},
		{
			name:        "Complex nested path",
			servicePath: "/api/v2/admin",
			methodPath:  "/users/management",
			expected:    "/api/v2/admin/users/management",
		},
		{
			name:        "Root paths",
			servicePath: "/",
			methodPath:  "/",
			expected:    "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHTTPPath(tt.servicePath, tt.methodPath)
			if result != tt.expected {
				t.Errorf("buildHTTPPath(%q, %q) = %q, expected %q",
					tt.servicePath, tt.methodPath, result, tt.expected)
			}
		})
	}
}

// Test ensureLeadingSlash function
func TestEnsureLeadingSlash(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Path with leading slash",
			path:     "/api/v1",
			expected: "/api/v1",
		},
		{
			name:     "Path without leading slash",
			path:     "api/v1",
			expected: "/api/v1",
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "/",
		},
		{
			name:     "Root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "Single character",
			path:     "a",
			expected: "/a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureLeadingSlash(tt.path)
			if result != tt.expected {
				t.Errorf("ensureLeadingSlash(%q) = %q, expected %q",
					tt.path, result, tt.expected)
			}
		})
	}
}

// Test mapHeaderTypeToOpenAPI function
func TestMapHeaderTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		name       string
		headerType string
		expected   string
	}{
		{
			name:       "String type",
			headerType: "string",
			expected:   "string",
		},
		{
			name:       "Empty type defaults to string",
			headerType: "",
			expected:   "string",
		},
		{
			name:       "Integer type",
			headerType: "integer",
			expected:   "integer",
		},
		{
			name:       "Int type",
			headerType: "int",
			expected:   "integer",
		},
		{
			name:       "Int32 type",
			headerType: "int32",
			expected:   "integer",
		},
		{
			name:       "Int64 type",
			headerType: "int64",
			expected:   "integer",
		},
		{
			name:       "Number type",
			headerType: "number",
			expected:   "number",
		},
		{
			name:       "Float type",
			headerType: "float",
			expected:   "number",
		},
		{
			name:       "Double type",
			headerType: "double",
			expected:   "number",
		},
		{
			name:       "Boolean type",
			headerType: "boolean",
			expected:   "boolean",
		},
		{
			name:       "Bool type",
			headerType: "bool",
			expected:   "boolean",
		},
		{
			name:       "Array type",
			headerType: "array",
			expected:   "array",
		},
		{
			name:       "Case insensitive - STRING",
			headerType: "STRING",
			expected:   "string",
		},
		{
			name:       "Case insensitive - INTEGER",
			headerType: "INTEGER",
			expected:   "integer",
		},
		{
			name:       "Unknown type defaults to string",
			headerType: "unknown",
			expected:   "string",
		},
		{
			name:       "Custom type defaults to string",
			headerType: "MyCustomType",
			expected:   "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapHeaderTypeToOpenAPI(tt.headerType)
			if result != tt.expected {
				t.Errorf("mapHeaderTypeToOpenAPI(%q) = %q, expected %q",
					tt.headerType, result, tt.expected)
			}
		})
	}
}

// Test combineHeaders function
func TestCombineHeaders(t *testing.T) {
	tests := []struct {
		name           string
		serviceHeaders []*http.Header
		methodHeaders  []*http.Header
		expectedCount  int
		expectedNames  []string
		description    string
	}{
		{
			name:           "Both empty",
			serviceHeaders: nil,
			methodHeaders:  nil,
			expectedCount:  0,
			expectedNames:  []string{},
			description:    "Should return empty slice when both are empty",
		},
		{
			name:           "Only service headers",
			serviceHeaders: []*http.Header{
				{Name: proto.String("X-Service-Header"), Type: proto.String("string")},
			},
			methodHeaders: nil,
			expectedCount: 1,
			expectedNames: []string{"X-Service-Header"},
			description:   "Should return only service headers",
		},
		{
			name:          "Only method headers",
			serviceHeaders: nil,
			methodHeaders: []*http.Header{
				{Name: proto.String("X-Method-Header"), Type: proto.String("string")},
			},
			expectedCount: 1,
			expectedNames: []string{"X-Method-Header"},
			description:   "Should return only method headers",
		},
		{
			name: "No overlapping headers",
			serviceHeaders: []*http.Header{
				{Name: proto.String("X-Service-Header"), Type: proto.String("string")},
			},
			methodHeaders: []*http.Header{
				{Name: proto.String("X-Method-Header"), Type: proto.String("string")},
			},
			expectedCount: 2,
			expectedNames: []string{"X-Service-Header", "X-Method-Header"},
			description:   "Should combine when no overlaps",
		},
		{
			name: "Overlapping headers - method overrides service",
			serviceHeaders: []*http.Header{
				{Name: proto.String("X-Override"), Type: proto.String("string"), Description: proto.String("Service")},
			},
			methodHeaders: []*http.Header{
				{Name: proto.String("X-Override"), Type: proto.String("integer"), Description: proto.String("Method")},
			},
			expectedCount: 1,
			expectedNames: []string{"X-Override"},
			description:   "Method header should override service header with same name",
		},
		{
			name: "Multiple headers with some overlaps",
			serviceHeaders: []*http.Header{
				{Name: proto.String("X-Service-1"), Type: proto.String("string")},
				{Name: proto.String("X-Common"), Type: proto.String("string"), Description: proto.String("Service")},
				{Name: proto.String("X-Service-2"), Type: proto.String("integer")},
			},
			methodHeaders: []*http.Header{
				{Name: proto.String("X-Method-1"), Type: proto.String("boolean")},
				{Name: proto.String("X-Common"), Type: proto.String("number"), Description: proto.String("Method")},
			},
			expectedCount: 4,
			expectedNames: []string{"X-Service-1", "X-Common", "X-Service-2", "X-Method-1"},
			description:   "Should combine all with method overriding common name",
		},
		{
			name: "Headers with empty names (should be filtered)",
			serviceHeaders: []*http.Header{
				{Name: proto.String(""), Type: proto.String("string")},
				{Name: proto.String("X-Valid"), Type: proto.String("string")},
			},
			methodHeaders: []*http.Header{
				{Name: proto.String(""), Type: proto.String("integer")},
			},
			expectedCount: 1,
			expectedNames: []string{"X-Valid"},
			description:   "Should filter out headers with empty names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := combineHeaders(tt.serviceHeaders, tt.methodHeaders)

			// Check count
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d headers, got %d", tt.expectedCount, len(result))
			}

			// Check that all expected names are present
			resultNames := make(map[string]bool)
			for _, header := range result {
				if header.GetName() != "" {
					resultNames[header.GetName()] = true
				}
			}

			for _, expectedName := range tt.expectedNames {
				if !resultNames[expectedName] {
					t.Errorf("Expected header %q not found in result", expectedName)
				}
			}

			// For overlap test, verify that method header overrode service header
			if tt.name == "Overlapping headers - method overrides service" && len(result) > 0 {
				overrideHeader := result[0]
				if overrideHeader.GetDescription() != "Method" {
					t.Errorf("Expected method header to override service header, got description: %s", 
						overrideHeader.GetDescription())
				}
				if overrideHeader.GetType() != "integer" {
					t.Errorf("Expected method header type 'integer', got: %s", 
						overrideHeader.GetType())
				}
			}
		})
	}
}

// Test convertHeadersToParameters function
func TestConvertHeadersToParameters(t *testing.T) {
	tests := []struct {
		name     string
		headers  []*http.Header
		expected int
		checkFn  func([]*v3.Parameter) error
	}{
		{
			name:     "Empty headers",
			headers:  []*http.Header{},
			expected: 0,
			checkFn:  nil,
		},
		{
			name:     "Nil headers",
			headers:  nil,
			expected: 0,
			checkFn:  nil,
		},
		{
			name: "Single string header",
			headers: []*http.Header{
				{
					Name:        proto.String("X-API-Key"),
					Description: proto.String("API key for authentication"),
					Type:        proto.String("string"),
					Required:    true,
					Format:      proto.String("uuid"),
					Example:     proto.String("123e4567-e89b-12d3-a456-426614174000"),
				},
			},
			expected: 1,
			checkFn: func(params []*v3.Parameter) error {
				param := params[0]
				if param.Name != "X-API-Key" {
					t.Errorf("Expected name 'X-API-Key', got %s", param.Name)
				}
				if param.In != "header" {
					t.Errorf("Expected in 'header', got %s", param.In)
				}
				if param.Description != "API key for authentication" {
					t.Errorf("Expected description, got %s", param.Description)
				}
				if param.Required == nil || !*param.Required {
					t.Error("Expected required to be true")
				}

				// Check schema
				if param.Schema == nil {
					t.Error("Schema is nil")
					return nil
				}

				schema := param.Schema.Schema()
				if len(schema.Type) != 1 || schema.Type[0] != "string" {
					t.Errorf("Expected schema type string, got %v", schema.Type)
				}
				if schema.Format != "uuid" {
					t.Errorf("Expected format 'uuid', got %s", schema.Format)
				}

				return nil
			},
		},
		{
			name: "Multiple headers with different types",
			headers: []*http.Header{
				{
					Name:     proto.String("X-String-Header"),
					Type:     proto.String("string"),
					Required: true,
				},
				{
					Name:     proto.String("X-Integer-Header"),
					Type:     proto.String("integer"),
					Required: false,
				},
				{
					Name:     proto.String("X-Boolean-Header"),
					Type:     proto.String("boolean"),
					Required: false,
				},
			},
			expected: 3,
			checkFn: func(params []*v3.Parameter) error {
				// Check that we have correct types for each parameter
				typeMap := make(map[string]string)
				requiredMap := make(map[string]bool)

				for _, param := range params {
					if param.Schema != nil && param.Schema.Schema() != nil {
						schema := param.Schema.Schema()
						if len(schema.Type) > 0 {
							typeMap[param.Name] = schema.Type[0]
						}
					}
					if param.Required != nil {
						requiredMap[param.Name] = *param.Required
					}
				}

				if typeMap["X-String-Header"] != "string" {
					t.Errorf("Expected X-String-Header type string, got %s", typeMap["X-String-Header"])
				}
				if typeMap["X-Integer-Header"] != "integer" {
					t.Errorf("Expected X-Integer-Header type integer, got %s", typeMap["X-Integer-Header"])
				}
				if typeMap["X-Boolean-Header"] != "boolean" {
					t.Errorf("Expected X-Boolean-Header type boolean, got %s", typeMap["X-Boolean-Header"])
				}

				// Check required flags
				if !requiredMap["X-String-Header"] {
					t.Error("Expected X-String-Header to be required")
				}
				if requiredMap["X-Integer-Header"] {
					t.Error("Expected X-Integer-Header to be optional")
				}

				return nil
			},
		},
		{
			name: "Header with deprecated flag",
			headers: []*http.Header{
				{
					Name:       proto.String("X-Legacy-Header"),
					Type:       proto.String("string"),
					Required:   false,
					Deprecated: true,
				},
			},
			expected: 1,
			checkFn: func(params []*v3.Parameter) error {
				param := params[0]
				if !param.Deprecated {
					t.Error("Expected parameter to be marked as deprecated")
				}
				return nil
			},
		},
		{
			name: "Header with empty name (should be filtered)",
			headers: []*http.Header{
				{
					Name: proto.String(""),
					Type: proto.String("string"),
				},
				{
					Name: proto.String("X-Valid-Header"),
					Type: proto.String("string"),
				},
			},
			expected: 1,
			checkFn: func(params []*v3.Parameter) error {
				if params[0].Name != "X-Valid-Header" {
					t.Errorf("Expected valid header, got %s", params[0].Name)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHeadersToParameters(tt.headers)

			if len(result) != tt.expected {
				t.Errorf("Expected %d parameters, got %d", tt.expected, len(result))
			}

			if tt.checkFn != nil {
				if err := tt.checkFn(result); err != nil {
					t.Errorf("Check function failed: %v", err)
				}
			}
		})
	}
}

// Test with mock service/method (requires more setup)
func TestGetServiceHTTPConfig(t *testing.T) {
	// Create a mock service with HTTP configuration
	service := &protogen.Service{
		Desc: &mockServiceDescWithHTTP{
			name: "TestService",
			hasServiceConfig: true,
			basePath: "/api/v1",
		},
	}

	config := getServiceHTTPConfig(service)

	if config == nil {
		t.Error("Expected service HTTP config, got nil")
	} else {
		if config.BasePath != "/api/v1" {
			t.Errorf("Expected base path '/api/v1', got %s", config.BasePath)
		}
	}
}

func TestGetServiceHTTPConfigNil(t *testing.T) {
	// Create a mock service without HTTP configuration
	service := &protogen.Service{
		Desc: &mockServiceDescWithHTTP{
			name: "TestService",
			hasServiceConfig: false,
		},
	}

	config := getServiceHTTPConfig(service)

	if config != nil {
		t.Errorf("Expected nil config for service without HTTP config, got %+v", config)
	}
}

// === Mock implementations for HTTP config testing ===

type mockServiceDescWithHTTP struct {
	name             string
	hasServiceConfig bool
	basePath         string
}

func (m *mockServiceDescWithHTTP) Name() protoreflect.Name         { return protoreflect.Name(m.name) }
func (m *mockServiceDescWithHTTP) FullName() protoreflect.FullName { return protoreflect.FullName(m.name) }
func (m *mockServiceDescWithHTTP) IsPlaceholder() bool            { return false }
func (m *mockServiceDescWithHTTP) Options() protoreflect.ProtoMessage {
	options := &descriptorpb.ServiceOptions{}
	if m.hasServiceConfig {
		// In a real implementation, this would set the extension
		// For now, we'll return basic options
	}
	return options
}
func (m *mockServiceDescWithHTTP) Index() int                   { return 0 }
func (m *mockServiceDescWithHTTP) Syntax() protoreflect.Syntax { return protoreflect.Proto3 }
func (m *mockServiceDescWithHTTP) Methods() protoreflect.MethodDescriptors { return nil }
func (m *mockServiceDescWithHTTP) Parent() protoreflect.Descriptor          { return nil }
func (m *mockServiceDescWithHTTP) ParentFile() protoreflect.FileDescriptor  { return nil }