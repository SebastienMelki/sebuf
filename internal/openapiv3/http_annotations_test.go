package openapiv3

import (
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

func TestGetMethodHTTPConfig(t *testing.T) {
	tests := []struct {
		name     string
		options  *descriptorpb.MethodOptions
		expected *HTTPConfig
	}{
		{
			name:     "nil options",
			options:  nil,
			expected: nil,
		},
		{
			name:     "empty options",
			options:  &descriptorpb.MethodOptions{},
			expected: nil,
		},
		{
			name: "with HTTP config",
			options: func() *descriptorpb.MethodOptions {
				opts := &descriptorpb.MethodOptions{}
				proto.SetExtension(opts, http.E_Config, &http.HttpConfig{
					Path: "/api/users",
				})
				return opts
			}(),
			expected: &HTTPConfig{
				Path: "/api/users",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := &protogen.Method{
				Desc: &mockMethodDescriptor{
					options: tt.options,
				},
			}
			
			result := getMethodHTTPConfig(method)
			
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Path, result.Path)
			}
		})
	}
}

func TestGetServiceHTTPConfig(t *testing.T) {
	tests := []struct {
		name     string
		options  *descriptorpb.ServiceOptions
		expected *ServiceHTTPConfig
	}{
		{
			name:     "nil options",
			options:  nil,
			expected: nil,
		},
		{
			name:     "empty options",
			options:  &descriptorpb.ServiceOptions{},
			expected: nil,
		},
		{
			name: "with service config",
			options: func() *descriptorpb.ServiceOptions {
				opts := &descriptorpb.ServiceOptions{}
				proto.SetExtension(opts, http.E_ServiceConfig, &http.ServiceConfig{
					BasePath: "/api/v1",
				})
				return opts
			}(),
			expected: &ServiceHTTPConfig{
				BasePath: "/api/v1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &protogen.Service{
				Desc: &mockServiceDescriptor{
					options: tt.options,
				},
			}
			
			result := getServiceHTTPConfig(service)
			
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.BasePath, result.BasePath)
			}
		})
	}
}

func TestBuildHTTPPath(t *testing.T) {
	tests := []struct {
		name         string
		servicePath  string
		methodPath   string
		expectedPath string
	}{
		{
			name:         "both empty",
			servicePath:  "",
			methodPath:   "",
			expectedPath: "/",
		},
		{
			name:         "only service path",
			servicePath:  "/api/v1",
			methodPath:   "",
			expectedPath: "/api/v1",
		},
		{
			name:         "only method path",
			servicePath:  "",
			methodPath:   "/users",
			expectedPath: "/users",
		},
		{
			name:         "both paths",
			servicePath:  "/api/v1",
			methodPath:   "/users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "service path with trailing slash",
			servicePath:  "/api/v1/",
			methodPath:   "/users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "method path without leading slash",
			servicePath:  "/api/v1",
			methodPath:   "users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "paths without leading slashes",
			servicePath:  "api/v1",
			methodPath:   "users",
			expectedPath: "/api/v1/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHTTPPath(tt.servicePath, tt.methodPath)
			assert.Equal(t, tt.expectedPath, result)
		})
	}
}

func TestEnsureLeadingSlash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "/",
		},
		{
			name:     "with leading slash",
			input:    "/api",
			expected: "/api",
		},
		{
			name:     "without leading slash",
			input:    "api",
			expected: "/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureLeadingSlash(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCombineHeaders(t *testing.T) {
	tests := []struct {
		name           string
		serviceHeaders []*http.Header
		methodHeaders  []*http.Header
		expectedCount  int
		checkResult    func(t *testing.T, result []*http.Header)
	}{
		{
			name:           "both empty",
			serviceHeaders: nil,
			methodHeaders:  nil,
			expectedCount:  0,
		},
		{
			name: "only service headers",
			serviceHeaders: []*http.Header{
				{Name: "X-API-Key", Type: "string", Required: true},
				{Name: "X-Tenant-ID", Type: "integer", Required: true},
			},
			methodHeaders: nil,
			expectedCount: 2,
		},
		{
			name:          "only method headers",
			serviceHeaders: nil,
			methodHeaders: []*http.Header{
				{Name: "X-Request-ID", Type: "string", Required: true},
			},
			expectedCount: 1,
		},
		{
			name: "method overrides service header",
			serviceHeaders: []*http.Header{
				{Name: "X-Tenant-ID", Type: "integer", Required: true},
			},
			methodHeaders: []*http.Header{
				{Name: "X-Tenant-ID", Type: "string", Required: false},
			},
			expectedCount: 1,
			checkResult: func(t *testing.T, result []*http.Header) {
				// Find the X-Tenant-ID header
				var tenantHeader *http.Header
				for _, h := range result {
					if h.GetName() == "X-Tenant-ID" {
						tenantHeader = h
						break
					}
				}
				require.NotNil(t, tenantHeader)
				assert.Equal(t, "string", tenantHeader.GetType())
				assert.False(t, tenantHeader.GetRequired())
			},
		},
		{
			name: "combined unique headers",
			serviceHeaders: []*http.Header{
				{Name: proto.String("X-API-Key"), Type: proto.String("string"), Required: true},
			},
			methodHeaders: []*http.Header{
				{Name: "X-Request-ID", Type: "string", Required: true},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := combineHeaders(tt.serviceHeaders, tt.methodHeaders)
			assert.Len(t, result, tt.expectedCount)
			
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestConvertHeadersToParameters(t *testing.T) {
	tests := []struct {
		name          string
		headers       []*http.Header
		expectedCount int
		checkParams   func(t *testing.T, params []*v3.Parameter)
	}{
		{
			name:          "nil headers",
			headers:       nil,
			expectedCount: 0,
		},
		{
			name:          "empty headers",
			headers:       []*http.Header{},
			expectedCount: 0,
		},
		{
			name: "single header",
			headers: []*http.Header{
				{
					Name:        "X-API-Key",
					Description: "API authentication key",
					Type:        "string",
					Required:    true,
					Format:      "uuid",
					Example:     "123e4567-e89b-12d3-a456-426614174000",
				},
			},
			expectedCount: 1,
			checkParams: func(t *testing.T, params []*v3.Parameter) {
				param := params[0]
				assert.Equal(t, "X-API-Key", param.Name)
				assert.Equal(t, "header", param.In)
				assert.Equal(t, "API authentication key", param.Description)
				assert.True(t, *param.Required)
				
				// Check schema
				schema := extractSchemaFromProxy(t, param.Schema)
				assert.Equal(t, []string{"string"}, schema.Type)
				assert.Equal(t, "uuid", schema.Format)
				assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", schema.Example.Value)
			},
		},
		{
			name: "multiple headers with different types",
			headers: []*http.Header{
				{
					Name:     "X-String-Header",
					Type:     "string",
					Required: true,
				},
				{
					Name:     "X-Integer-Header",
					Type:     "integer",
					Required: false,
				},
				{
					Name:     "X-Boolean-Header",
					Type:     "boolean",
					Required: false,
				},
				{
					Name:     "X-Number-Header",
					Type:     "number",
					Required: false,
				},
				{
					Name:     "X-Array-Header",
					Type:     "array",
					Required: false,
				},
			},
			expectedCount: 5,
			checkParams: func(t *testing.T, params []*v3.Parameter) {
				typeMap := make(map[string]string)
				for _, param := range params {
					schema := extractSchemaFromProxy(t, param.Schema)
					typeMap[param.Name] = schema.Type[0]
				}
				
				assert.Equal(t, "string", typeMap["X-String-Header"])
				assert.Equal(t, "integer", typeMap["X-Integer-Header"])
				assert.Equal(t, "boolean", typeMap["X-Boolean-Header"])
				assert.Equal(t, "number", typeMap["X-Number-Header"])
				assert.Equal(t, "array", typeMap["X-Array-Header"])
			},
		},
		{
			name: "deprecated header",
			headers: []*http.Header{
				{
					Name:       "X-Legacy-Header",
					Type:       "string",
					Required:   false,
					Deprecated: true,
				},
			},
			expectedCount: 1,
			checkParams: func(t *testing.T, params []*v3.Parameter) {
				assert.True(t, params[0].Deprecated)
			},
		},
		{
			name: "header without name (should be skipped)",
			headers: []*http.Header{
				{
					Type:     proto.String("string"),
					Required: false,
				},
				{
					Name:     proto.String("X-Valid-Header"),
					Type:     proto.String("string"),
					Required: false,
				},
			},
			expectedCount: 1,
			checkParams: func(t *testing.T, params []*v3.Parameter) {
				assert.Equal(t, "X-Valid-Header", params[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHeadersToParameters(tt.headers)
			
			if tt.expectedCount == 0 {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
				
				if tt.checkParams != nil {
					tt.checkParams(t, result)
				}
			}
		})
	}
}

func TestMapHeaderTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"String", "string"},
		{"STRING", "string"},
		{"", "string"}, // Default case
		{"integer", "integer"},
		{"int", "integer"},
		{"int32", "integer"},
		{"int64", "integer"},
		{"number", "number"},
		{"float", "number"},
		{"double", "number"},
		{"boolean", "boolean"},
		{"bool", "boolean"},
		{"array", "array"},
		{"unknown", "string"}, // Unknown type defaults to string
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapHeaderTypeToOpenAPI(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock implementations for testing

type mockMethodDescriptor struct {
	protoreflect.MethodDescriptor
	name    protoreflect.Name
	options *descriptorpb.MethodOptions
}

func (m *mockMethodDescriptor) Options() protoreflect.ProtoMessage {
	if m.options != nil {
		return m.options
	}
	return &descriptorpb.MethodOptions{}
}

func (m *mockMethodDescriptor) Name() protoreflect.Name {
	if m.name != "" {
		return m.name
	}
	return "TestMethod"
}

type mockServiceDescriptor struct {
	protoreflect.ServiceDescriptor
	options *descriptorpb.ServiceOptions
	name    protoreflect.Name
}

func (m *mockServiceDescriptor) Options() protoreflect.ProtoMessage {
	if m.options != nil {
		return m.options
	}
	return &descriptorpb.ServiceOptions{}
}

func (m *mockServiceDescriptor) Name() protoreflect.Name {
	if m.name != "" {
		return m.name
	}
	return "TestService"
}