package httpgen

import (
	"testing"
)

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Normal cases
		{"PascalCase to camelCase", "CreateUser", "createUser"},
		{"single word", "User", "user"},
		{"already lowercase", "create", "create"},
		{"camelCase unchanged after first", "getUser", "getUser"},

		// Edge cases
		{"empty string", "", ""},
		{"single uppercase char", "A", "a"},
		{"single lowercase char", "a", "a"},
		{"all uppercase", "ABC", "aBC"},
		{"starts with number", "123User", "123User"},

		// Method name patterns
		{"GetUser", "GetUser", "getUser"},
		{"ListUsers", "ListUsers", "listUsers"},
		{"CreateUserProfile", "CreateUserProfile", "createUserProfile"},
		{"DeleteByID", "DeleteByID", "deleteByID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lowerFirst(tt.input)
			if result != tt.expected {
				t.Errorf("lowerFirst(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Normal cases
		{"simple PascalCase", "CreateUser", "create_user"},
		{"camelCase", "createUser", "create_user"},
		{"multiple words", "GetUserById", "get_user_by_id"},
		{"three words", "ListAllUsers", "list_all_users"},

		// Edge cases
		{"empty string", "", ""},
		{"single lowercase word", "user", "user"},
		{"single uppercase word", "User", "user"},
		{"already snake_case", "create_user", "create_user"},
		{"consecutive uppercase", "HTTPMethod", "h_t_t_p_method"},
		{"acronym at end", "getUserID", "get_user_i_d"},

		// Numbers
		{"with numbers", "GetV2Api", "get_v2_api"},
		{"numbers at end", "User123", "user123"},

		// Method name patterns commonly seen in proto
		{"GetUser", "GetUser", "get_user"},
		{"ListUsers", "ListUsers", "list_users"},
		{"CreateUserProfile", "CreateUserProfile", "create_user_profile"},
		{"DeleteUser", "DeleteUser", "delete_user"},
		{"UpdateUserStatus", "UpdateUserStatus", "update_user_status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := camelToSnake(tt.input)
			if result != tt.expected {
				t.Errorf("camelToSnake(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerator_New(t *testing.T) {
	// Test creating a new generator without plugin (nil is acceptable for unit test)
	gen := New(nil)
	if gen == nil {
		t.Fatal("New(nil) returned nil, expected non-nil Generator")
	}
	if gen.generateMock {
		t.Error("New() generator should have generateMock = false by default")
	}
}

func TestGenerator_NewWithOptions(t *testing.T) {
	tests := []struct {
		name       string
		opts       Options
		expectMock bool
	}{
		{"default options", Options{}, false},
		{"with mock enabled", Options{GenerateMock: true}, true},
		{"with mock disabled", Options{GenerateMock: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewWithOptions(nil, tt.opts)
			if gen == nil {
				t.Error("NewWithOptions returned nil")
				return
			}
			if gen.generateMock != tt.expectMock {
				t.Errorf("generator.generateMock = %v, expected %v", gen.generateMock, tt.expectMock)
			}
		})
	}
}

func TestOptions_Struct(t *testing.T) {
	// Test Options struct can be properly initialized
	opts := Options{
		GenerateMock: true,
	}
	if !opts.GenerateMock {
		t.Error("Options.GenerateMock should be true")
	}
}

// Benchmark tests.
func BenchmarkLowerFirst(b *testing.B) {
	inputs := []string{"CreateUser", "GetUser", "ListUsers", "DeleteUser", "UpdateUserProfile"}
	for i := range b.N {
		lowerFirst(inputs[i%len(inputs)])
	}
}

func BenchmarkCamelToSnake(b *testing.B) {
	inputs := []string{"CreateUser", "GetUserById", "ListAllUsers", "DeleteUserProfile", "UpdateUserStatus"}
	for i := range b.N {
		camelToSnake(inputs[i%len(inputs)])
	}
}
