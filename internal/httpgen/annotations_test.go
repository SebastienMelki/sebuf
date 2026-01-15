package httpgen

import (
	nethttp "net/http"
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
		// Standard HTTP methods
		{"GET method", http.HttpMethod_HTTP_METHOD_GET, "GET"},
		{"POST method", http.HttpMethod_HTTP_METHOD_POST, "POST"},
		{"PUT method", http.HttpMethod_HTTP_METHOD_PUT, "PUT"},
		{"DELETE method", http.HttpMethod_HTTP_METHOD_DELETE, "DELETE"},
		{"PATCH method", http.HttpMethod_HTTP_METHOD_PATCH, "PATCH"},

		// Backward compatibility - UNSPECIFIED defaults to POST
		{"UNSPECIFIED defaults to POST", http.HttpMethod_HTTP_METHOD_UNSPECIFIED, "POST"},

		// Edge cases - unknown values default to POST
		{"unknown positive value defaults to POST", http.HttpMethod(999), "POST"},
		{"unknown negative value defaults to POST", http.HttpMethod(-1), "POST"},
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
		{
			"three params",
			"/orgs/{org_id}/teams/{team_id}/members/{member_id}",
			[]string{"org_id", "team_id", "member_id"},
		},
		{"camelCase param", "/items/{itemId}", []string{"itemId"}},

		// Empty/missing cases
		{"no params", "/users", nil},
		{"empty path", "", nil},
		{"just slash", "/", nil},
		{"only static segments", "/api/v1/users/list", nil},

		// Edge cases - malformed
		{"unclosed brace should not match", "/users/{id", nil},
		{"unopened brace should not match", "/users/id}", nil},
		{"empty braces returns nil", "/users/{}", nil},

		// Edge cases - special characters in param names
		{"hyphenated param", "/users/{user-id}", []string{"user-id"}},
		{"param with numbers", "/users/{id123}", []string{"id123"}},
		{"param starting with number", "/users/{123id}", []string{"123id"}},
		{"param with underscore prefix", "/users/{_private}", []string{"_private"}},

		// Complex paths
		{
			"deeply nested path",
			"/api/v1/orgs/{org_id}/teams/{team_id}/members/{member_id}/roles/{role_id}",
			[]string{"org_id", "team_id", "member_id", "role_id"},
		},
		{"param at start", "/{version}/users", []string{"version"}},
		{"param at end", "/users/{id}", []string{"id"}},
		{"consecutive params", "/users/{type}/{id}", []string{"type", "id"}},
		{"duplicate param names", "/users/{id}/items/{id}", []string{"id", "id"}},

		// With query string (path params should still be extracted)
		{"path with trailing content", "/users/{id}/profile", []string{"id"}},
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

func TestHTTPConfig_Struct(t *testing.T) {
	// Test that HTTPConfig struct has expected fields
	config := HTTPConfig{
		Path:       "/users/{id}",
		Method:     "GET",
		PathParams: []string{"id"},
	}

	if config.Path != "/users/{id}" {
		t.Errorf("HTTPConfig.Path = %q, expected %q", config.Path, "/users/{id}")
	}
	if config.Method != nethttp.MethodGet {
		t.Errorf("HTTPConfig.Method = %q, expected %q", config.Method, nethttp.MethodGet)
	}
	if len(config.PathParams) != 1 || config.PathParams[0] != "id" {
		t.Errorf("HTTPConfig.PathParams = %v, expected [id]", config.PathParams)
	}
}

func TestQueryParam_Struct(t *testing.T) {
	// Test that QueryParam struct has expected fields
	param := QueryParam{
		FieldName:   "page_number",
		FieldGoName: "PageNumber",
		ParamName:   "page",
		Required:    true,
	}

	if param.FieldName != "page_number" {
		t.Errorf("QueryParam.FieldName = %q, expected %q", param.FieldName, "page_number")
	}
	if param.FieldGoName != "PageNumber" {
		t.Errorf("QueryParam.FieldGoName = %q, expected %q", param.FieldGoName, "PageNumber")
	}
	if param.ParamName != "page" {
		t.Errorf("QueryParam.ParamName = %q, expected %q", param.ParamName, "page")
	}
	if !param.Required {
		t.Error("QueryParam.Required = false, expected true")
	}
}

func TestServiceConfigImpl_Struct(t *testing.T) {
	// Test that ServiceConfigImpl struct has expected fields
	config := ServiceConfigImpl{
		BasePath: "/api/v1",
	}

	if config.BasePath != "/api/v1" {
		t.Errorf("ServiceConfigImpl.BasePath = %q, expected %q", config.BasePath, "/api/v1")
	}
}

// Benchmark tests for performance-critical functions.
func BenchmarkExtractPathParams_SingleParam(b *testing.B) {
	path := "/users/{user_id}"
	for range b.N {
		extractPathParams(path)
	}
}

func BenchmarkExtractPathParams_MultipleParams(b *testing.B) {
	path := "/orgs/{org_id}/teams/{team_id}/members/{member_id}"
	for range b.N {
		extractPathParams(path)
	}
}

func BenchmarkExtractPathParams_NoParams(b *testing.B) {
	path := "/api/v1/users/list"
	for range b.N {
		extractPathParams(path)
	}
}

func BenchmarkHttpMethodToString(b *testing.B) {
	methods := []http.HttpMethod{
		http.HttpMethod_HTTP_METHOD_GET,
		http.HttpMethod_HTTP_METHOD_POST,
		http.HttpMethod_HTTP_METHOD_PUT,
		http.HttpMethod_HTTP_METHOD_DELETE,
		http.HttpMethod_HTTP_METHOD_PATCH,
	}
	for i := range b.N {
		httpMethodToString(methods[i%len(methods)])
	}
}
