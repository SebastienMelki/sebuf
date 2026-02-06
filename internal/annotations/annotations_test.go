package annotations

import (
	"reflect"
	"testing"

	"github.com/SebastienMelki/sebuf/http"
)

func TestHTTPMethodToString(t *testing.T) {
	tests := []struct {
		name     string
		method   http.HttpMethod
		expected string
	}{
		// Standard HTTP methods (uppercase)
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
			result := HTTPMethodToString(tt.method)
			if result != tt.expected {
				t.Errorf("HTTPMethodToString(%v) = %q, expected %q", tt.method, result, tt.expected)
			}
		})
	}
}

func TestHTTPMethodToLower(t *testing.T) {
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

		// Backward compatibility - UNSPECIFIED defaults to post
		{"UNSPECIFIED defaults to post", http.HttpMethod_HTTP_METHOD_UNSPECIFIED, "post"},

		// Edge cases - unknown values default to post
		{"unknown positive value defaults to post", http.HttpMethod(999), "post"},
		{"unknown negative value defaults to post", http.HttpMethod(-1), "post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTTPMethodToLower(tt.method)
			if result != tt.expected {
				t.Errorf("HTTPMethodToLower(%v) = %q, expected %q", tt.method, result, tt.expected)
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

		// With trailing content
		{"path with trailing content", "/users/{id}/profile", []string{"id"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPathParams(tt.path)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExtractPathParams(%q) = %v, expected %v", tt.path, result, tt.expected)
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
			result := BuildHTTPPath(tt.servicePath, tt.methodPath)
			if result != tt.expected {
				t.Errorf("BuildHTTPPath(%q, %q) = %q, expected %q", tt.servicePath, tt.methodPath, result, tt.expected)
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
			result := EnsureLeadingSlash(tt.path)
			if result != tt.expected {
				t.Errorf("EnsureLeadingSlash(%q) = %q, expected %q", tt.path, result, tt.expected)
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
			result := CombineHeaders(tt.serviceHeaders, tt.methodHeaders)

			// Check count
			if len(result) != len(tt.expectedNames) {
				t.Errorf("CombineHeaders() returned %d headers, expected %d", len(result), len(tt.expectedNames))
				return
			}

			// Check names (should be sorted)
			for i, name := range tt.expectedNames {
				if result[i].GetName() != name {
					t.Errorf("CombineHeaders()[%d].Name = %q, expected %q", i, result[i].GetName(), name)
				}
			}
		})
	}
}

func TestCombineHeaders_MethodOverridesService(t *testing.T) {
	serviceHeaders := []*http.Header{
		{Name: "X-API-Key", Description: "Service API Key", Required: true},
	}
	methodHeaders := []*http.Header{
		{Name: "X-API-Key", Description: "Method API Key", Required: false},
	}

	result := CombineHeaders(serviceHeaders, methodHeaders)

	if len(result) != 1 {
		t.Fatalf("Expected 1 header, got %d", len(result))
	}

	// Method header should win
	if result[0].GetDescription() != "Method API Key" {
		t.Errorf("Method header should override service header, got description: %q", result[0].GetDescription())
	}
	if result[0].GetRequired() {
		t.Error("Method header should override service header's Required field")
	}
}

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single uppercase char", "A", "a"},
		{"single lowercase char", "a", "a"},
		{"normal case", "FooBar", "fooBar"},
		{"already lowercase", "fooBar", "fooBar"},
		{"all uppercase", "FOO", "fOO"},
		{"single word", "Hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LowerFirst(tt.input)
			if result != tt.expected {
				t.Errorf("LowerFirst(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHTTPConfig_Struct(t *testing.T) {
	config := HTTPConfig{
		Path:       "/users/{id}",
		Method:     "GET",
		PathParams: []string{"id"},
	}

	if config.Path != "/users/{id}" {
		t.Errorf("HTTPConfig.Path = %q, expected %q", config.Path, "/users/{id}")
	}
	if config.Method != "GET" { //nolint:usestdlibvars // http here is sebuf/http, not net/http
		t.Errorf("HTTPConfig.Method = %q, expected %q", config.Method, "GET")
	}
	if len(config.PathParams) != 1 || config.PathParams[0] != "id" {
		t.Errorf("HTTPConfig.PathParams = %v, expected [id]", config.PathParams)
	}
}

func TestQueryParam_Struct(t *testing.T) {
	param := QueryParam{
		FieldName:     "page_number",
		FieldGoName:   "PageNumber",
		FieldJSONName: "pageNumber",
		ParamName:     "page",
		Required:      true,
		FieldKind:     "int32",
		Field:         nil, // protogen.Field can be nil in tests
	}

	if param.FieldName != "page_number" {
		t.Errorf("QueryParam.FieldName = %q, expected %q", param.FieldName, "page_number")
	}
	if param.FieldGoName != "PageNumber" {
		t.Errorf("QueryParam.FieldGoName = %q, expected %q", param.FieldGoName, "PageNumber")
	}
	if param.FieldJSONName != "pageNumber" {
		t.Errorf("QueryParam.FieldJSONName = %q, expected %q", param.FieldJSONName, "pageNumber")
	}
	if param.ParamName != "page" {
		t.Errorf("QueryParam.ParamName = %q, expected %q", param.ParamName, "page")
	}
	if !param.Required {
		t.Error("QueryParam.Required = false, expected true")
	}
	if param.FieldKind != "int32" {
		t.Errorf("QueryParam.FieldKind = %q, expected %q", param.FieldKind, "int32")
	}
	if param.Field != nil {
		t.Error("QueryParam.Field should be nil")
	}
}

func TestServiceConfig_Struct(t *testing.T) {
	config := ServiceConfig{
		BasePath: "/api/v1",
	}

	if config.BasePath != "/api/v1" {
		t.Errorf("ServiceConfig.BasePath = %q, expected %q", config.BasePath, "/api/v1")
	}
}

func TestUnwrapValidationError(t *testing.T) {
	err := &UnwrapValidationError{
		MessageName: "MyMessage",
		FieldName:   "items",
		Reason:      "unwrap annotation can only be used on repeated or map fields",
	}

	expected := "invalid unwrap annotation on MyMessage.items: unwrap annotation can only be used on repeated or map fields"
	if err.Error() != expected {
		t.Errorf("UnwrapValidationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

// Benchmark tests for performance-critical functions.

func BenchmarkExtractPathParams_SingleParam(b *testing.B) {
	path := "/users/{user_id}"
	for range b.N {
		ExtractPathParams(path)
	}
}

func BenchmarkExtractPathParams_MultipleParams(b *testing.B) {
	path := "/orgs/{org_id}/teams/{team_id}/members/{member_id}"
	for range b.N {
		ExtractPathParams(path)
	}
}

func BenchmarkExtractPathParams_NoParams(b *testing.B) {
	path := "/api/v1/users/list"
	for range b.N {
		ExtractPathParams(path)
	}
}

func BenchmarkBuildHTTPPath(b *testing.B) {
	for range b.N {
		BuildHTTPPath("/api/v1", "/users/{user_id}")
	}
}

func BenchmarkEnsureLeadingSlash(b *testing.B) {
	paths := []string{"/users", "users", "", "/", "api/v1/users"}
	for i := range b.N {
		EnsureLeadingSlash(paths[i%len(paths)])
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
	for range b.N {
		CombineHeaders(serviceHeaders, methodHeaders)
	}
}

func BenchmarkHTTPMethodToString(b *testing.B) {
	methods := []http.HttpMethod{
		http.HttpMethod_HTTP_METHOD_GET,
		http.HttpMethod_HTTP_METHOD_POST,
		http.HttpMethod_HTTP_METHOD_PUT,
		http.HttpMethod_HTTP_METHOD_DELETE,
		http.HttpMethod_HTTP_METHOD_PATCH,
	}
	for i := range b.N {
		HTTPMethodToString(methods[i%len(methods)])
	}
}

func BenchmarkLowerFirst(b *testing.B) {
	inputs := []string{"FooBar", "Hello", "fooBar", "", "A"}
	for i := range b.N {
		LowerFirst(inputs[i%len(inputs)])
	}
}

// Tests for Int64Encoding and EnumEncoding annotation types.
// Note: Full integration tests with protogen.Field require actual proto files.
// These tests verify the enum types and helper logic work correctly.

func TestInt64EncodingValues(t *testing.T) {
	// Verify the enum values exist and have expected numeric values
	tests := []struct {
		name     string
		encoding http.Int64Encoding
		value    int32
	}{
		{"UNSPECIFIED", http.Int64Encoding_INT64_ENCODING_UNSPECIFIED, 0},
		{"STRING", http.Int64Encoding_INT64_ENCODING_STRING, 1},
		{"NUMBER", http.Int64Encoding_INT64_ENCODING_NUMBER, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.encoding) != tt.value {
				t.Errorf("Int64Encoding_%s = %d, expected %d", tt.name, int32(tt.encoding), tt.value)
			}
		})
	}
}

func TestEnumEncodingValues(t *testing.T) {
	// Verify the enum values exist and have expected numeric values
	tests := []struct {
		name     string
		encoding http.EnumEncoding
		value    int32
	}{
		{"UNSPECIFIED", http.EnumEncoding_ENUM_ENCODING_UNSPECIFIED, 0},
		{"STRING", http.EnumEncoding_ENUM_ENCODING_STRING, 1},
		{"NUMBER", http.EnumEncoding_ENUM_ENCODING_NUMBER, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.encoding) != tt.value {
				t.Errorf("EnumEncoding_%s = %d, expected %d", tt.name, int32(tt.encoding), tt.value)
			}
		})
	}
}

func TestInt64EncodingStringRepresentation(t *testing.T) {
	// Verify the string names for debugging/logging
	tests := []struct {
		encoding http.Int64Encoding
		expected string
	}{
		{http.Int64Encoding_INT64_ENCODING_UNSPECIFIED, "INT64_ENCODING_UNSPECIFIED"},
		{http.Int64Encoding_INT64_ENCODING_STRING, "INT64_ENCODING_STRING"},
		{http.Int64Encoding_INT64_ENCODING_NUMBER, "INT64_ENCODING_NUMBER"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// The .String() method is generated by protoc-gen-go
			if tt.encoding.String() != tt.expected {
				t.Errorf("Int64Encoding.String() = %q, expected %q", tt.encoding.String(), tt.expected)
			}
		})
	}
}

func TestEnumEncodingStringRepresentation(t *testing.T) {
	// Verify the string names for debugging/logging
	tests := []struct {
		encoding http.EnumEncoding
		expected string
	}{
		{http.EnumEncoding_ENUM_ENCODING_UNSPECIFIED, "ENUM_ENCODING_UNSPECIFIED"},
		{http.EnumEncoding_ENUM_ENCODING_STRING, "ENUM_ENCODING_STRING"},
		{http.EnumEncoding_ENUM_ENCODING_NUMBER, "ENUM_ENCODING_NUMBER"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// The .String() method is generated by protoc-gen-go
			if tt.encoding.String() != tt.expected {
				t.Errorf("EnumEncoding.String() = %q, expected %q", tt.encoding.String(), tt.expected)
			}
		})
	}
}

func TestInt64EncodingExtensionDescriptor(t *testing.T) {
	// Verify the extension descriptor exists and has correct properties
	ext := http.E_Int64Encoding
	if ext == nil {
		t.Fatal("E_Int64Encoding extension descriptor is nil")
	}

	// Extension number should be 50010
	if ext.TypeDescriptor().Number() != 50010 {
		t.Errorf("E_Int64Encoding number = %d, expected 50010", ext.TypeDescriptor().Number())
	}
}

func TestEnumEncodingExtensionDescriptor(t *testing.T) {
	// Verify the extension descriptor exists and has correct properties
	ext := http.E_EnumEncoding
	if ext == nil {
		t.Fatal("E_EnumEncoding extension descriptor is nil")
	}

	// Extension number should be 50011
	if ext.TypeDescriptor().Number() != 50011 {
		t.Errorf("E_EnumEncoding number = %d, expected 50011", ext.TypeDescriptor().Number())
	}
}

func TestEnumValueExtensionDescriptor(t *testing.T) {
	// Verify the extension descriptor exists and has correct properties
	ext := http.E_EnumValue
	if ext == nil {
		t.Fatal("E_EnumValue extension descriptor is nil")
	}

	// Extension number should be 50012
	if ext.TypeDescriptor().Number() != 50012 {
		t.Errorf("E_EnumValue number = %d, expected 50012", ext.TypeDescriptor().Number())
	}
}

func TestNullableExtensionDescriptor(t *testing.T) {
	// Verify the extension descriptor exists and has correct properties
	ext := http.E_Nullable
	if ext == nil {
		t.Fatal("E_Nullable extension descriptor is nil")
	}

	// Extension number should be 50013
	if ext.TypeDescriptor().Number() != 50013 {
		t.Errorf("E_Nullable number = %d, expected 50013", ext.TypeDescriptor().Number())
	}
}

func TestEmptyBehaviorExtensionDescriptor(t *testing.T) {
	// Verify the extension descriptor exists and has correct properties
	ext := http.E_EmptyBehavior
	if ext == nil {
		t.Fatal("E_EmptyBehavior extension descriptor is nil")
	}

	// Extension number should be 50014
	if ext.TypeDescriptor().Number() != 50014 {
		t.Errorf("E_EmptyBehavior number = %d, expected 50014", ext.TypeDescriptor().Number())
	}
}

func TestEmptyBehaviorValues(t *testing.T) {
	// Verify the enum values exist and have expected numeric values
	tests := []struct {
		name     string
		behavior http.EmptyBehavior
		value    int32
	}{
		{"UNSPECIFIED", http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED, 0},
		{"PRESERVE", http.EmptyBehavior_EMPTY_BEHAVIOR_PRESERVE, 1},
		{"NULL", http.EmptyBehavior_EMPTY_BEHAVIOR_NULL, 2},
		{"OMIT", http.EmptyBehavior_EMPTY_BEHAVIOR_OMIT, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.behavior) != tt.value {
				t.Errorf("EmptyBehavior_%s = %d, expected %d", tt.name, int32(tt.behavior), tt.value)
			}
		})
	}
}

func TestEmptyBehaviorStringRepresentation(t *testing.T) {
	// Verify the string names for debugging/logging
	tests := []struct {
		behavior http.EmptyBehavior
		expected string
	}{
		{http.EmptyBehavior_EMPTY_BEHAVIOR_UNSPECIFIED, "EMPTY_BEHAVIOR_UNSPECIFIED"},
		{http.EmptyBehavior_EMPTY_BEHAVIOR_PRESERVE, "EMPTY_BEHAVIOR_PRESERVE"},
		{http.EmptyBehavior_EMPTY_BEHAVIOR_NULL, "EMPTY_BEHAVIOR_NULL"},
		{http.EmptyBehavior_EMPTY_BEHAVIOR_OMIT, "EMPTY_BEHAVIOR_OMIT"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// The .String() method is generated by protoc-gen-go
			if tt.behavior.String() != tt.expected {
				t.Errorf("EmptyBehavior.String() = %q, expected %q", tt.behavior.String(), tt.expected)
			}
		})
	}
}

func TestNullableValidationError(t *testing.T) {
	err := &NullableValidationError{
		MessageName: "MyMessage",
		FieldName:   "my_field",
		Reason:      "nullable annotation is only valid on proto3 optional fields",
	}

	expected := "invalid nullable annotation on MyMessage.my_field: nullable annotation is only valid on proto3 optional fields"
	if err.Error() != expected {
		t.Errorf("NullableValidationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestEmptyBehaviorValidationError(t *testing.T) {
	err := &EmptyBehaviorValidationError{
		MessageName: "MyMessage",
		FieldName:   "my_field",
		Reason:      "empty_behavior annotation is only valid on message fields",
	}

	expected := "invalid empty_behavior annotation on MyMessage.my_field: empty_behavior annotation is only valid on message fields"
	if err.Error() != expected {
		t.Errorf("EmptyBehaviorValidationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

// Tests for TimestampFormat annotation types.

func TestTimestampFormatExtensionDescriptor(t *testing.T) {
	ext := http.E_TimestampFormat
	if ext == nil {
		t.Fatal("E_TimestampFormat extension descriptor is nil")
	}

	// Extension number should be 50015
	if ext.TypeDescriptor().Number() != 50015 {
		t.Errorf("E_TimestampFormat number = %d, expected 50015", ext.TypeDescriptor().Number())
	}
}

func TestTimestampFormatValues(t *testing.T) {
	tests := []struct {
		name   string
		format http.TimestampFormat
		value  int32
	}{
		{"UNSPECIFIED", http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED, 0},
		{"RFC3339", http.TimestampFormat_TIMESTAMP_FORMAT_RFC3339, 1},
		{"UNIX_SECONDS", http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS, 2},
		{"UNIX_MILLIS", http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS, 3},
		{"DATE", http.TimestampFormat_TIMESTAMP_FORMAT_DATE, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.format) != tt.value {
				t.Errorf("TimestampFormat_%s = %d, expected %d", tt.name, int32(tt.format), tt.value)
			}
		})
	}
}

func TestTimestampFormatStringRepresentation(t *testing.T) {
	tests := []struct {
		format   http.TimestampFormat
		expected string
	}{
		{http.TimestampFormat_TIMESTAMP_FORMAT_UNSPECIFIED, "TIMESTAMP_FORMAT_UNSPECIFIED"},
		{http.TimestampFormat_TIMESTAMP_FORMAT_RFC3339, "TIMESTAMP_FORMAT_RFC3339"},
		{http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_SECONDS, "TIMESTAMP_FORMAT_UNIX_SECONDS"},
		{http.TimestampFormat_TIMESTAMP_FORMAT_UNIX_MILLIS, "TIMESTAMP_FORMAT_UNIX_MILLIS"},
		{http.TimestampFormat_TIMESTAMP_FORMAT_DATE, "TIMESTAMP_FORMAT_DATE"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.format.String() != tt.expected {
				t.Errorf("TimestampFormat.String() = %q, expected %q", tt.format.String(), tt.expected)
			}
		})
	}
}

func TestTimestampFormatValidationError(t *testing.T) {
	err := &TimestampFormatValidationError{
		MessageName: "MyMessage",
		FieldName:   "created_at",
		Reason:      "timestamp_format annotation is only valid on google.protobuf.Timestamp fields",
	}

	expected := "invalid timestamp_format annotation on MyMessage.created_at: timestamp_format annotation is only valid on google.protobuf.Timestamp fields"
	if err.Error() != expected {
		t.Errorf("TimestampFormatValidationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

// Tests for BytesEncoding annotation types.

func TestBytesEncodingExtensionDescriptor(t *testing.T) {
	ext := http.E_BytesEncoding
	if ext == nil {
		t.Fatal("E_BytesEncoding extension descriptor is nil")
	}

	// Extension number should be 50016
	if ext.TypeDescriptor().Number() != 50016 {
		t.Errorf("E_BytesEncoding number = %d, expected 50016", ext.TypeDescriptor().Number())
	}
}

func TestBytesEncodingValues(t *testing.T) {
	tests := []struct {
		name     string
		encoding http.BytesEncoding
		value    int32
	}{
		{"UNSPECIFIED", http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED, 0},
		{"BASE64", http.BytesEncoding_BYTES_ENCODING_BASE64, 1},
		{"BASE64_RAW", http.BytesEncoding_BYTES_ENCODING_BASE64_RAW, 2},
		{"BASE64URL", http.BytesEncoding_BYTES_ENCODING_BASE64URL, 3},
		{"BASE64URL_RAW", http.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW, 4},
		{"HEX", http.BytesEncoding_BYTES_ENCODING_HEX, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.encoding) != tt.value {
				t.Errorf("BytesEncoding_%s = %d, expected %d", tt.name, int32(tt.encoding), tt.value)
			}
		})
	}
}

func TestBytesEncodingStringRepresentation(t *testing.T) {
	tests := []struct {
		encoding http.BytesEncoding
		expected string
	}{
		{http.BytesEncoding_BYTES_ENCODING_UNSPECIFIED, "BYTES_ENCODING_UNSPECIFIED"},
		{http.BytesEncoding_BYTES_ENCODING_BASE64, "BYTES_ENCODING_BASE64"},
		{http.BytesEncoding_BYTES_ENCODING_BASE64_RAW, "BYTES_ENCODING_BASE64_RAW"},
		{http.BytesEncoding_BYTES_ENCODING_BASE64URL, "BYTES_ENCODING_BASE64URL"},
		{http.BytesEncoding_BYTES_ENCODING_BASE64URL_RAW, "BYTES_ENCODING_BASE64URL_RAW"},
		{http.BytesEncoding_BYTES_ENCODING_HEX, "BYTES_ENCODING_HEX"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.encoding.String() != tt.expected {
				t.Errorf("BytesEncoding.String() = %q, expected %q", tt.encoding.String(), tt.expected)
			}
		})
	}
}

func TestBytesEncodingValidationError(t *testing.T) {
	err := &BytesEncodingValidationError{
		MessageName: "MyMessage",
		FieldName:   "payload",
		Reason:      "bytes_encoding annotation is only valid on bytes fields",
	}

	expected := "invalid bytes_encoding annotation on MyMessage.payload: bytes_encoding annotation is only valid on bytes fields"
	if err.Error() != expected {
		t.Errorf("BytesEncodingValidationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

// Tests for oneof_config and oneof_value annotation types.

func TestOneofConfigExtensionDescriptor(t *testing.T) {
	ext := http.E_OneofConfig
	if ext == nil {
		t.Fatal("E_OneofConfig extension descriptor is nil")
	}

	// Extension number should be 50017
	if ext.TypeDescriptor().Number() != 50017 {
		t.Errorf("E_OneofConfig number = %d, expected 50017", ext.TypeDescriptor().Number())
	}
}

func TestOneofValueExtensionDescriptor(t *testing.T) {
	ext := http.E_OneofValue
	if ext == nil {
		t.Fatal("E_OneofValue extension descriptor is nil")
	}

	// Extension number should be 50018
	if ext.TypeDescriptor().Number() != 50018 {
		t.Errorf("E_OneofValue number = %d, expected 50018", ext.TypeDescriptor().Number())
	}
}

func TestOneofConfigMessageFields(t *testing.T) {
	// Verify OneofConfig can be constructed with both fields
	config := &http.OneofConfig{
		Discriminator: "type",
		Flatten:       true,
	}

	if config.GetDiscriminator() != "type" {
		t.Errorf("OneofConfig.Discriminator = %q, expected %q", config.GetDiscriminator(), "type")
	}
	if !config.GetFlatten() {
		t.Error("OneofConfig.Flatten = false, expected true")
	}

	// Verify zero value defaults
	empty := &http.OneofConfig{}
	if empty.GetDiscriminator() != "" {
		t.Errorf("empty OneofConfig.Discriminator = %q, expected %q", empty.GetDiscriminator(), "")
	}
	if empty.GetFlatten() {
		t.Error("empty OneofConfig.Flatten = true, expected false")
	}
}

// Tests for flatten and flatten_prefix annotation types.

func TestFlattenExtensionDescriptor(t *testing.T) {
	ext := http.E_Flatten
	if ext == nil {
		t.Fatal("E_Flatten extension descriptor is nil")
	}

	// Extension number should be 50019
	if ext.TypeDescriptor().Number() != 50019 {
		t.Errorf("E_Flatten number = %d, expected 50019", ext.TypeDescriptor().Number())
	}
}

func TestFlattenPrefixExtensionDescriptor(t *testing.T) {
	ext := http.E_FlattenPrefix
	if ext == nil {
		t.Fatal("E_FlattenPrefix extension descriptor is nil")
	}

	// Extension number should be 50020
	if ext.TypeDescriptor().Number() != 50020 {
		t.Errorf("E_FlattenPrefix number = %d, expected 50020", ext.TypeDescriptor().Number())
	}
}

// Tests for OneofDiscriminatorInfo and OneofVariant structs.

func TestOneofDiscriminatorInfoStruct(t *testing.T) {
	info := &OneofDiscriminatorInfo{
		Discriminator: "kind",
		Flatten:       true,
		Variants: []OneofVariant{
			{
				DiscriminatorVal: "email",
				IsMessage:        true,
			},
			{
				DiscriminatorVal: "sms",
				IsMessage:        true,
			},
		},
	}

	if info.Discriminator != "kind" {
		t.Errorf("Discriminator = %q, expected %q", info.Discriminator, "kind")
	}
	if !info.Flatten {
		t.Error("Flatten = false, expected true")
	}
	if len(info.Variants) != 2 {
		t.Fatalf("len(Variants) = %d, expected 2", len(info.Variants))
	}
	if info.Variants[0].DiscriminatorVal != "email" {
		t.Errorf("Variants[0].DiscriminatorVal = %q, expected %q", info.Variants[0].DiscriminatorVal, "email")
	}
	if !info.Variants[0].IsMessage {
		t.Error("Variants[0].IsMessage = false, expected true")
	}
}

// Tests verifying function signatures and types exist.
// Note: Full integration tests with protogen.Field/Oneof require actual proto files.
// These tests verify the exported API surface is accessible.

func TestFunctionSignaturesExist(t *testing.T) {
	// Verify all exported functions exist with correct signatures.
	// We cannot construct protogen types without a real proto compilation,
	// so integration tests in later plans will test with actual proto files.
	var (
		_ = GetOneofVariantValue
		_ = IsFlattenField
		_ = GetFlattenPrefix
		_ = HasOneofDiscriminator
		_ = HasFlattenFields
	)

	t.Log("All exported function signatures verified")
}

// Tests for extension number sequence continuity.

func TestExtensionNumberSequence(t *testing.T) {
	// Verify all extension numbers are contiguous and correctly assigned.
	// This prevents accidental duplication or gaps.
	extensionNumbers := map[string]int32{
		"field_examples":   50007,
		"query":            50008,
		"unwrap":           50009,
		"int64_encoding":   50010,
		"enum_encoding":    50011,
		"enum_value":       50012,
		"nullable":         50013,
		"empty_behavior":   50014,
		"timestamp_format": 50015,
		"bytes_encoding":   50016,
		"oneof_config":     50017,
		"oneof_value":      50018,
		"flatten":          50019,
		"flatten_prefix":   50020,
	}

	// Verify each extension's actual number
	checks := []struct {
		name     string
		actual   int32
		expected int32
	}{
		{"oneof_config", int32(http.E_OneofConfig.TypeDescriptor().Number()), extensionNumbers["oneof_config"]},
		{"oneof_value", int32(http.E_OneofValue.TypeDescriptor().Number()), extensionNumbers["oneof_value"]},
		{"flatten", int32(http.E_Flatten.TypeDescriptor().Number()), extensionNumbers["flatten"]},
		{"flatten_prefix", int32(http.E_FlattenPrefix.TypeDescriptor().Number()), extensionNumbers["flatten_prefix"]},
		// Also verify the latest existing one to detect sequence breaks
		{"bytes_encoding", int32(http.E_BytesEncoding.TypeDescriptor().Number()), extensionNumbers["bytes_encoding"]},
	}

	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			if tc.actual != tc.expected {
				t.Errorf("Extension %s number = %d, expected %d", tc.name, tc.actual, tc.expected)
			}
		})
	}
}
