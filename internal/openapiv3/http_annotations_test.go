package openapiv3

import (
	"reflect"
	"testing"

	"github.com/SebastienMelki/sebuf/http"
)

func TestHttpMethodToString(t *testing.T) {
	tests := []struct {
		name     string
		method   http.HttpMethod
		expected string
	}{
		// Standard HTTP methods (lowercase for OpenAPI)
		{"GET method", http.HttpMethod_HTTP_METHOD_GET, "get"},
		{"POST method", http.HttpMethod_HTTP_METHOD_POST, "post"},
		{"PUT method", http.HttpMethod_HTTP_METHOD_PUT, "put"},
		{"DELETE method", http.HttpMethod_HTTP_METHOD_DELETE, "delete"},
		{"PATCH method", http.HttpMethod_HTTP_METHOD_PATCH, "patch"},

		// Backward compatibility - UNSPECIFIED defaults to POST
		{"UNSPECIFIED defaults to post", http.HttpMethod_HTTP_METHOD_UNSPECIFIED, "post"},

		// Edge cases - unknown values default to POST
		{"unknown positive value defaults to post", http.HttpMethod(999), "post"},
		{"unknown negative value defaults to post", http.HttpMethod(-1), "post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := httpMethodToString(tt.method)
			if result != tt.expected {
				t.Errorf("httpMethodToString(%v) = %q, expected %q", tt.method, result, tt.expected)
			}
		})
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		// Normal cases
		{"single param", "/users/{id}", []string{"id"}},
		{"single param with underscore", "/users/{user_id}", []string{"user_id"}},
		{"multiple params", "/users/{user_id}/posts/{post_id}", []string{"user_id", "post_id"}},
		{"three params", "/orgs/{org_id}/teams/{team_id}/members/{member_id}", []string{"org_id", "team_id", "member_id"}},

		// Empty/missing cases
		{"no params", "/users", nil},
		{"empty path", "", nil},
		{"just slash", "/", nil},

		// Edge cases
		{"unclosed brace", "/users/{id", nil},
		{"empty braces", "/users/{}", nil},
		{"hyphenated param", "/users/{user-id}", []string{"user-id"}},
		{"consecutive params", "/users/{type}/{id}", []string{"type", "id"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPathParams(tt.path)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractPathParams(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestBuildHTTPPath(t *testing.T) {
	tests := []struct {
		name        string
		servicePath string
		methodPath  string
		expected    string
	}{
		// Normal cases
		{"both paths with leading slashes", "/api/v1", "/users", "/api/v1/users"},
		{"service path only", "/api/v1", "", "/api/v1"},
		{"method path only", "", "/users", "/users"},

		// Empty cases
		{"both empty", "", "", "/"},

		// Slash handling
		{"service with trailing slash", "/api/v1/", "/users", "/api/v1/users"},
		{"method without leading slash", "/api/v1", "users", "/api/v1/users"},
		{"both without slashes", "api", "users", "/api/users"},
		{"service trailing + method leading", "/api/v1/", "/users", "/api/v1/users"},

		// Complex paths
		{"nested service path", "/api/v1/admin", "/users/list", "/api/v1/admin/users/list"},
		{"path with params", "/api/v1", "/users/{user_id}", "/api/v1/users/{user_id}"},
		{"both with params", "/orgs/{org_id}", "/teams/{team_id}", "/orgs/{org_id}/teams/{team_id}"},

		// Edge cases
		{"service only no leading slash", "api", "", "/api"},
		{"method only no leading slash", "", "users", "/users"},
		{"multiple trailing slashes on service", "/api///", "/users", "/api///users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHTTPPath(tt.servicePath, tt.methodPath)
			if result != tt.expected {
				t.Errorf("buildHTTPPath(%q, %q) = %q, expected %q", tt.servicePath, tt.methodPath, result, tt.expected)
			}
		})
	}
}

func TestEnsureLeadingSlash(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"already has slash", "/users", "/users"},
		{"no slash", "users", "/users"},
		{"empty string", "", "/"},
		{"just slash", "/", "/"},
		{"double slash already has leading slash", "//users", "//users"},
		{"path with params", "users/{id}", "/users/{id}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureLeadingSlash(tt.path)
			if result != tt.expected {
				t.Errorf("ensureLeadingSlash(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		})
	}
}

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

func TestCombineHeaders(t *testing.T) {
	tests := []struct {
		name           string
		serviceHeaders []*http.Header
		methodHeaders  []*http.Header
		expectedNames  []string
	}{
		{
			name:           "service only",
			serviceHeaders: []*http.Header{{Name: "X-API-Key"}},
			methodHeaders:  nil,
			expectedNames:  []string{"X-API-Key"},
		},
		{
			name:           "method only",
			serviceHeaders: nil,
			methodHeaders:  []*http.Header{{Name: "X-Request-ID"}},
			expectedNames:  []string{"X-Request-ID"},
		},
		{
			name:           "both empty",
			serviceHeaders: nil,
			methodHeaders:  nil,
			expectedNames:  nil,
		},
		{
			name:           "method overrides service with same name",
			serviceHeaders: []*http.Header{{Name: "X-API-Key", Description: "service level"}},
			methodHeaders:  []*http.Header{{Name: "X-API-Key", Description: "method level"}},
			expectedNames:  []string{"X-API-Key"},
		},
		{
			name:           "different headers combined",
			serviceHeaders: []*http.Header{{Name: "X-API-Key"}},
			methodHeaders:  []*http.Header{{Name: "X-Request-ID"}},
			expectedNames:  []string{"X-API-Key", "X-Request-ID"},
		},
		{
			name: "multiple headers sorted",
			serviceHeaders: []*http.Header{
				{Name: "Z-Header"},
				{Name: "A-Header"},
			},
			methodHeaders: []*http.Header{
				{Name: "M-Header"},
			},
			expectedNames: []string{"A-Header", "M-Header", "Z-Header"},
		},
		{
			name:           "empty name headers skipped during merge",
			serviceHeaders: []*http.Header{{Name: ""}, {Name: "X-Valid"}},
			methodHeaders:  []*http.Header{{Name: "X-Method"}},
			expectedNames:  []string{"X-Method", "X-Valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := combineHeaders(tt.serviceHeaders, tt.methodHeaders)

			// Check count
			if len(result) != len(tt.expectedNames) {
				t.Errorf("combineHeaders() returned %d headers, expected %d", len(result), len(tt.expectedNames))
				return
			}

			// Check names (should be sorted)
			for i, name := range tt.expectedNames {
				if result[i].GetName() != name {
					t.Errorf("combineHeaders()[%d].Name = %q, expected %q", i, result[i].GetName(), name)
				}
			}
		})
	}
}

func TestCombineHeaders_MethodOverridesService(t *testing.T) {
	// Test that method headers override service headers with the same name
	serviceHeaders := []*http.Header{
		{Name: "X-API-Key", Description: "Service API Key", Required: true},
	}
	methodHeaders := []*http.Header{
		{Name: "X-API-Key", Description: "Method API Key", Required: false},
	}

	result := combineHeaders(serviceHeaders, methodHeaders)

	if len(result) != 1 {
		t.Fatalf("Expected 1 header, got %d", len(result))
	}

	// Method header should win
	if result[0].GetDescription() != "Method API Key" {
		t.Errorf("Method header should override service header, got description: %q", result[0].GetDescription())
	}
	if result[0].GetRequired() != false {
		t.Error("Method header should override service header's Required field")
	}
}

func TestHTTPConfig_Struct(t *testing.T) {
	config := HTTPConfig{
		Path:       "/users/{id}",
		Method:     "get",
		PathParams: []string{"id"},
	}

	if config.Path != "/users/{id}" {
		t.Errorf("HTTPConfig.Path = %q, expected %q", config.Path, "/users/{id}")
	}
	if config.Method != "get" {
		t.Errorf("HTTPConfig.Method = %q, expected %q", config.Method, "get")
	}
	if len(config.PathParams) != 1 || config.PathParams[0] != "id" {
		t.Errorf("HTTPConfig.PathParams = %v, expected [id]", config.PathParams)
	}
}

func TestQueryParam_Struct(t *testing.T) {
	param := QueryParam{
		FieldName: "page_number",
		ParamName: "page",
		Required:  true,
		Field:     nil, // protogen.Field can be nil for this test
	}

	if param.FieldName != "page_number" {
		t.Errorf("QueryParam.FieldName = %q, expected %q", param.FieldName, "page_number")
	}
	if param.ParamName != "page" {
		t.Errorf("QueryParam.ParamName = %q, expected %q", param.ParamName, "page")
	}
	if !param.Required {
		t.Error("QueryParam.Required = false, expected true")
	}
}

func TestServiceHTTPConfig_Struct(t *testing.T) {
	config := ServiceHTTPConfig{
		BasePath: "/api/v1",
	}

	if config.BasePath != "/api/v1" {
		t.Errorf("ServiceHTTPConfig.BasePath = %q, expected %q", config.BasePath, "/api/v1")
	}
}

// Benchmark tests
func BenchmarkBuildHTTPPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buildHTTPPath("/api/v1", "/users/{user_id}")
	}
}

func BenchmarkEnsureLeadingSlash(b *testing.B) {
	paths := []string{"/users", "users", "", "/", "api/v1/users"}
	for i := 0; i < b.N; i++ {
		ensureLeadingSlash(paths[i%len(paths)])
	}
}

func BenchmarkMapHeaderTypeToOpenAPI(b *testing.B) {
	types := []string{"string", "integer", "number", "boolean", "array", "", "unknown"}
	for i := 0; i < b.N; i++ {
		mapHeaderTypeToOpenAPI(types[i%len(types)])
	}
}

func BenchmarkCombineHeaders(b *testing.B) {
	serviceHeaders := []*http.Header{
		{Name: "X-API-Key"},
		{Name: "X-Tenant-ID"},
	}
	methodHeaders := []*http.Header{
		{Name: "X-Request-ID"},
		{Name: "X-API-Key"}, // Override
	}
	for i := 0; i < b.N; i++ {
		combineHeaders(serviceHeaders, methodHeaders)
	}
}

func BenchmarkExtractPathParams(b *testing.B) {
	paths := []string{
		"/users/{user_id}",
		"/orgs/{org_id}/teams/{team_id}/members/{member_id}",
		"/api/v1/users",
	}
	for i := 0; i < b.N; i++ {
		extractPathParams(paths[i%len(paths)])
	}
}
