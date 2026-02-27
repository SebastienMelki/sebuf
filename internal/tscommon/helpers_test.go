package tscommon

import "testing"

func TestSnakeToLowerCamel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "two_words", input: "user_id", want: "userId"},
		{name: "two_words_longer", input: "page_number", want: "pageNumber"},
		{name: "no_underscores", input: "simple", want: "simple"},
		{name: "three_single_chars", input: "a_b_c", want: "aBC"},
		{name: "empty_string", input: "", want: ""},
		{name: "mixed_case", input: "already_camelCase", want: "alreadyCamelCase"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SnakeToLowerCamel(tt.input)
			if got != tt.want {
				t.Errorf("SnakeToLowerCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSnakeToUpperCamel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "two_words", input: "user_id", want: "UserId"},
		{name: "two_words_longer", input: "page_number", want: "PageNumber"},
		{name: "no_underscores", input: "simple", want: "Simple"},
		{name: "empty_string", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SnakeToUpperCamel(tt.input)
			if got != tt.want {
				t.Errorf("SnakeToUpperCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHeaderNameToPropertyName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "x_prefix_api_key", input: "X-API-Key", want: "apiKey"},
		{name: "x_prefix_request_id", input: "X-Request-ID", want: "requestId"},
		{name: "no_x_prefix", input: "Content-Type", want: "contentType"},
		{name: "x_prefix_tenant_id", input: "X-Tenant-ID", want: "tenantId"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HeaderNameToPropertyName(tt.input)
			if got != tt.want {
				t.Errorf("HeaderNameToPropertyName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
