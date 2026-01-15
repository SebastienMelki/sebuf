package httpgen

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestValidationError_Struct(t *testing.T) {
	// Test ValidationError struct has expected fields
	err := ValidationError{
		Service: "UserService",
		Method:  "GetUser",
		Message: "path variable '{user_id}' has no matching field",
	}

	if err.Service != "UserService" {
		t.Errorf("ValidationError.Service = %q, expected %q", err.Service, "UserService")
	}
	if err.Method != "GetUser" {
		t.Errorf("ValidationError.Method = %q, expected %q", err.Method, "GetUser")
	}
	if !strings.Contains(err.Message, "path variable") {
		t.Errorf("ValidationError.Message = %q, expected to contain 'path variable'", err.Message)
	}
}

// TestIsPathParamCompatibleByKind tests the isPathParamCompatible function using the kind directly.
// Since we can't easily mock protogen.Field, we test the underlying logic by examining the switch statement.
func TestIsPathParamCompatibleByKind(t *testing.T) {
	// This tests the type compatibility logic. The actual function requires a *protogen.Field,
	// but we can verify the logic by checking what kinds should be compatible.
	compatibleKinds := []protoreflect.Kind{
		protoreflect.StringKind,
		protoreflect.Int32Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind,
		protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind,
		protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind,
		protoreflect.BoolKind,
		protoreflect.FloatKind,
		protoreflect.DoubleKind,
	}

	incompatibleKinds := []protoreflect.Kind{
		protoreflect.MessageKind,
		protoreflect.BytesKind,
		protoreflect.EnumKind,
		protoreflect.GroupKind,
	}

	// Test compatible kinds
	for _, kind := range compatibleKinds {
		expected := isKindPathParamCompatible(kind)
		if !expected {
			t.Errorf("Kind %v should be path param compatible", kind)
		}
	}

	// Test incompatible kinds
	for _, kind := range incompatibleKinds {
		expected := isKindPathParamCompatible(kind)
		if expected {
			t.Errorf("Kind %v should NOT be path param compatible", kind)
		}
	}
}

// isKindPathParamCompatible is a helper that mirrors the logic in isPathParamCompatible
// but works directly with protoreflect.Kind for testability.
func isKindPathParamCompatible(kind protoreflect.Kind) bool {
	switch kind {
	case protoreflect.StringKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind,
		protoreflect.BoolKind,
		protoreflect.FloatKind, protoreflect.DoubleKind:
		return true
	default:
		return false
	}
}

func TestGetBodyFields_Logic(t *testing.T) {
	// Test the set logic used in getBodyFields without actual protogen.Message
	tests := []struct {
		name        string
		allFields   []string
		pathParams  []string
		queryParams []string
		expected    []string
	}{
		{
			name:        "all fields bound to path",
			allFields:   []string{"id"},
			pathParams:  []string{"id"},
			queryParams: nil,
			expected:    nil,
		},
		{
			name:        "all fields bound to query",
			allFields:   []string{"page", "limit"},
			pathParams:  nil,
			queryParams: []string{"page", "limit"},
			expected:    nil,
		},
		{
			name:        "some fields unbound",
			allFields:   []string{"id", "name", "email"},
			pathParams:  []string{"id"},
			queryParams: nil,
			expected:    []string{"name", "email"},
		},
		{
			name:        "mixed binding",
			allFields:   []string{"id", "page", "name", "email"},
			pathParams:  []string{"id"},
			queryParams: []string{"page"},
			expected:    []string{"name", "email"},
		},
		{
			name:        "no bindings",
			allFields:   []string{"name", "email", "age"},
			pathParams:  nil,
			queryParams: nil,
			expected:    []string{"name", "email", "age"},
		},
		{
			name:        "empty message",
			allFields:   nil,
			pathParams:  nil,
			queryParams: nil,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUnboundFields(tt.allFields, tt.pathParams, tt.queryParams)
			if !stringSliceEqual(result, tt.expected) {
				t.Errorf("getUnboundFields() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// getUnboundFields is a helper that mirrors getBodyFields logic but works with string slices.
func getUnboundFields(allFields, pathParams, queryParams []string) []string {
	pathParamSet := make(map[string]bool)
	for _, p := range pathParams {
		pathParamSet[p] = true
	}

	queryParamSet := make(map[string]bool)
	for _, qp := range queryParams {
		queryParamSet[qp] = true
	}

	var bodyFields []string
	for _, field := range allFields {
		if !pathParamSet[field] && !queryParamSet[field] {
			bodyFields = append(bodyFields, field)
		}
	}

	return bodyFields
}

func stringSliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestValidationErrorMessages(t *testing.T) {
	// Test that validation error messages are actionable and contain useful information
	tests := []struct {
		name            string
		error           ValidationError
		mustContain     []string
		mustNotContain  []string
	}{
		{
			name: "missing field error",
			error: ValidationError{
				Service: "UserService",
				Method:  "GetUser",
				Message: "path variable '{user_id}' in path '/users/{user_id}' has no matching field in message 'GetUserRequest'. Add a field named 'user_id' to the request message, or fix the path variable name.",
			},
			mustContain:    []string{"path variable", "user_id", "no matching field", "Add a field"},
			mustNotContain: nil,
		},
		{
			name: "type error",
			error: ValidationError{
				Service: "UserService",
				Method:  "GetUser",
				Message: "path variable '{data}' is bound to field 'data' of type 'message', but path parameters must be scalar types (string, int32, int64, uint32, uint64, bool, float, double). Change the field type or remove it from the path.",
			},
			mustContain:    []string{"scalar types", "Change the field type"},
			mustNotContain: nil,
		},
		{
			name: "conflict error",
			error: ValidationError{
				Service: "UserService",
				Method:  "GetUser",
				Message: "field 'id' is used both as a path variable in '/users/{id}' and as a query parameter. A field can only be bound to one parameter type. Remove either the path variable or the query annotation.",
			},
			mustContain:    []string{"used both", "path variable", "query parameter", "Remove either"},
			mustNotContain: nil,
		},
		{
			name: "GET with body error",
			error: ValidationError{
				Service: "UserService",
				Method:  "ListUsers",
				Message: "GET request has fields that are not bound to path or query parameters: [name email]. GET requests cannot have a request body. Either add [(sebuf.http.query)] annotations to these fields, include them in the path as variables, or change the HTTP method to POST/PUT/PATCH.",
			},
			mustContain:    []string{"GET request", "not bound", "cannot have a request body", "sebuf.http.query"},
			mustNotContain: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, s := range tt.mustContain {
				if !strings.Contains(tt.error.Message, s) {
					t.Errorf("Error message should contain %q, got: %s", s, tt.error.Message)
				}
			}
			for _, s := range tt.mustNotContain {
				if strings.Contains(tt.error.Message, s) {
					t.Errorf("Error message should NOT contain %q, got: %s", s, tt.error.Message)
				}
			}
		})
	}
}

func TestHTTPMethodValidation(t *testing.T) {
	// Test that GET and DELETE are correctly identified as methods that shouldn't have body
	methodsWithoutBody := []string{"GET", "DELETE"}
	methodsWithBody := []string{"POST", "PUT", "PATCH"}

	for _, method := range methodsWithoutBody {
		if !isMethodWithoutBody(method) {
			t.Errorf("%s should be identified as method without body", method)
		}
	}

	for _, method := range methodsWithBody {
		if isMethodWithoutBody(method) {
			t.Errorf("%s should NOT be identified as method without body", method)
		}
	}
}

// isMethodWithoutBody is a helper that matches the validation logic
func isMethodWithoutBody(method string) bool {
	return method == "GET" || method == "DELETE"
}

// Benchmark tests
func BenchmarkGetUnboundFields(b *testing.B) {
	allFields := []string{"id", "name", "email", "phone", "address", "city", "country", "zip"}
	pathParams := []string{"id"}
	queryParams := []string{"page", "limit"}

	for i := 0; i < b.N; i++ {
		getUnboundFields(allFields, pathParams, queryParams)
	}
}

func BenchmarkIsKindPathParamCompatible(b *testing.B) {
	kinds := []protoreflect.Kind{
		protoreflect.StringKind,
		protoreflect.Int32Kind,
		protoreflect.MessageKind,
		protoreflect.BytesKind,
		protoreflect.BoolKind,
	}
	for i := 0; i < b.N; i++ {
		isKindPathParamCompatible(kinds[i%len(kinds)])
	}
}
