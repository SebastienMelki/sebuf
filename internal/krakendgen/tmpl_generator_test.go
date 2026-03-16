package krakendgen

import (
	"strings"
	"testing"
)

func TestServiceNameToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"UserService", "user_service"},
		{"FullGatewayService", "full_gateway_service"},
		{"JWTAuthService", "jwt_auth_service"},
		{"API", "api"},
		{"Simple", "simple"},
		{"HTTPServer", "http_server"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := serviceNameToSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("serviceNameToSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHostVarName(t *testing.T) {
	tests := []struct {
		serviceName string
		want        string
	}{
		{"UserService", "{{ .vars.user_service_host }}"},
		{"FullGatewayService", "{{ .vars.full_gateway_service_host }}"},
	}
	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			got := hostVarName(tt.serviceName)
			if got != tt.want {
				t.Errorf("hostVarName(%q) = %q, want %q", tt.serviceName, got, tt.want)
			}
		})
	}
}

func TestTemplateFileName(t *testing.T) {
	tests := []struct {
		serviceName string
		want        string
	}{
		{"UserService", "user_service_endpoints.tmpl"},
		{"FullGatewayService", "full_gateway_service_endpoints.tmpl"},
	}
	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			got := TemplateFileName(tt.serviceName)
			if got != tt.want {
				t.Errorf("TemplateFileName(%q) = %q, want %q", tt.serviceName, got, tt.want)
			}
		})
	}
}

func TestGenerateTemplateFile_Empty(t *testing.T) {
	got := GenerateTemplateFile(nil)
	if got != "" {
		t.Errorf("expected empty string for nil endpoints, got %q", got)
	}
	got = GenerateTemplateFile([]Endpoint{})
	if got != "" {
		t.Errorf("expected empty string for empty endpoints, got %q", got)
	}
}

func TestGenerateTemplateFile_SimpleEndpoint(t *testing.T) {
	eps := []Endpoint{{
		Endpoint:       "/api/v1/users",
		Method:         "POST",
		OutputEncoding: "json",
		Backend: []Backend{{
			URLPattern: "/api/v1/users",
			Host:       []string{"http://backend:8080"},
			Method:     "POST",
			Encoding:   "json",
		}},
		ServiceName: "UserService",
	}}

	got := GenerateTemplateFile(eps)

	assertContains(t, got, `"endpoint": "/api/v1/users"`)
	assertContains(t, got, `"method": "POST"`)
	assertContains(t, got, `"output_encoding": "json"`)
	assertContains(t, got, `"sd": "static"`)
	assertContains(t, got, `"disable_host_sanitize": false`)
	assertContains(t, got, `"return_error_code": true`)
	assertContains(t, got, `{{ .vars.user_service_host }}`)
	assertNotContains(t, got, `"timeout"`)
	// No endpoint-level extra_config (no JWT, no recaptcha).
	// Backend extra_config is always present (return_error_code).
	assertNotContains(t, got, `{{ template "jwt_auth_validator.tmpl"`)
	assertNotContains(t, got, `{{ include "recpatcha_validator.tmpl"`)
}

func TestGenerateTemplateFile_WithJWT(t *testing.T) {
	eps := []Endpoint{{
		Endpoint:       "/api/v1/items",
		Method:         "GET",
		OutputEncoding: "json",
		Backend: []Backend{{
			URLPattern: "/api/v1/items",
			Host:       []string{"http://backend:8080"},
			Method:     "GET",
			Encoding:   "json",
		}},
		ServiceName: "ItemService",
		HasJWT:      true,
	}}

	got := GenerateTemplateFile(eps)

	assertContains(t, got, `{{ template "jwt_auth_validator.tmpl" . }}`)
	assertContains(t, got, `"extra_config"`)
}

func TestGenerateTemplateFile_WithRecaptcha(t *testing.T) {
	eps := []Endpoint{{
		Endpoint:       "/api/v1/register",
		Method:         "POST",
		OutputEncoding: "json",
		Backend: []Backend{{
			URLPattern: "/api/v1/register",
			Host:       []string{"http://backend:8080"},
			Method:     "POST",
			Encoding:   "json",
		}},
		ServiceName:  "AuthService",
		HasRecaptcha: true,
	}}

	got := GenerateTemplateFile(eps)

	assertContains(t, got, `{{ include "recpatcha_validator.tmpl" }}`)
}

func TestGenerateTemplateFile_JWTAndRecaptcha(t *testing.T) {
	eps := []Endpoint{{
		Endpoint:       "/api/v1/register",
		Method:         "POST",
		OutputEncoding: "json",
		Backend: []Backend{{
			URLPattern: "/api/v1/register",
			Host:       []string{"http://backend:8080"},
			Method:     "POST",
			Encoding:   "json",
		}},
		ServiceName:  "AuthService",
		HasJWT:       true,
		HasRecaptcha: true,
	}}

	got := GenerateTemplateFile(eps)

	assertContains(t, got, `{{ include "recpatcha_validator.tmpl" }}`)
	assertContains(t, got, `{{ template "jwt_auth_validator.tmpl" . }}`)

	// Recaptcha should come before JWT.
	recIdx := strings.Index(got, "recpatcha_validator")
	jwtIdx := strings.Index(got, "jwt_auth_validator")
	if recIdx >= jwtIdx {
		t.Error("recaptcha should appear before JWT in extra_config")
	}
}

func TestGenerateTemplateFile_HeaderPartial(t *testing.T) {
	eps := []Endpoint{{
		Endpoint:       "/api/v1/trade",
		Method:         "POST",
		OutputEncoding: "json",
		InputHeaders:   []string{"X-API-Key", "Authorization"},
		Backend: []Backend{{
			URLPattern: "/api/v1/trade",
			Host:       []string{"http://backend:8080"},
			Method:     "POST",
			Encoding:   "json",
		}},
		ServiceName:   "TradeService",
		HeaderPartial: "trading_input_headers.tmpl",
		HasJWT:        true,
	}}

	got := GenerateTemplateFile(eps)

	assertContains(t, got, `{{ include "trading_input_headers.tmpl" }}`)
	assertNotContains(t, got, `"input_headers"`)
}

func TestGenerateTemplateFile_MethodTimeout(t *testing.T) {
	eps := []Endpoint{{
		Endpoint:       "/api/v1/report",
		Method:         "GET",
		OutputEncoding: "json",
		Timeout:        "90s",
		Backend: []Backend{{
			URLPattern: "/api/v1/report",
			Host:       []string{"http://backend:8080"},
			Method:     "GET",
			Encoding:   "json",
		}},
		ServiceName:     "ReportService",
		IsMethodTimeout: true,
	}}

	got := GenerateTemplateFile(eps)
	assertContains(t, got, `"timeout": "90s"`)
}

func TestGenerateTemplateFile_ServiceTimeout_Omitted(t *testing.T) {
	eps := []Endpoint{{
		Endpoint:       "/api/v1/items",
		Method:         "GET",
		OutputEncoding: "json",
		Timeout:        "3s",
		Backend: []Backend{{
			URLPattern: "/api/v1/items",
			Host:       []string{"http://backend:8080"},
			Method:     "GET",
			Encoding:   "json",
		}},
		ServiceName:     "ItemService",
		IsMethodTimeout: false,
	}}

	got := GenerateTemplateFile(eps)
	assertNotContains(t, got, `"timeout"`)
}

func TestGenerateTemplateFile_MultipleEndpoints(t *testing.T) {
	eps := []Endpoint{
		{
			Endpoint:       "/api/v1/items",
			Method:         "POST",
			OutputEncoding: "json",
			Backend: []Backend{
				{URLPattern: "/api/v1/items", Host: []string{"h"}, Method: "POST", Encoding: "json"},
			},
			ServiceName: "ItemService",
		},
		{
			Endpoint:       "/api/v1/items/{id}",
			Method:         "GET",
			OutputEncoding: "json",
			Backend: []Backend{
				{URLPattern: "/api/v1/items/{id}", Host: []string{"h"}, Method: "GET", Encoding: "json"},
			},
			ServiceName: "ItemService",
		},
	}

	got := GenerateTemplateFile(eps)

	// Should be comma-separated fragments, not wrapped in a JSON array.
	assertContains(t, got, "},\n{")
	if strings.Count(got, `"endpoint"`) != 2 {
		t.Errorf("expected 2 endpoints, got %d", strings.Count(got, `"endpoint"`))
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q, but it doesn't.\nOutput:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected output NOT to contain %q, but it does.\nOutput:\n%s", needle, haystack)
	}
}
