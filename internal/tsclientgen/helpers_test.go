package tsclientgen

import "testing"

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"FooBar", "fooBar"},
		{"A", "a"},
		{"", ""},
		{"aBC", "aBC"},
		{"ABC", "aBC"},
		{"getUser", "getUser"},
		{"GetUser", "getUser"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := lowerFirst(tt.input)
			if got != tt.want {
				t.Errorf("lowerFirst(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSnakeToLowerCamel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user_id", "userId"},
		{"a_b_c", "aBC"},
		{"single", "single"},
		{"page_size", "pageSize"},
		{"note_id", "noteId"},
		{"created_at", "createdAt"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := snakeToLowerCamel(tt.input)
			if got != tt.want {
				t.Errorf("snakeToLowerCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHeaderNameToPropertyName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"X-API-Key", "apiKey"},
		{"X-Request-ID", "requestId"},
		{"X-Tenant-ID", "tenantId"},
		{"Authorization", "authorization"},
		{"X-Idempotency-Key", "idempotencyKey"},
		{"X-Correlation-ID", "correlationId"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := headerNameToPropertyName(tt.input)
			if got != tt.want {
				t.Errorf("headerNameToPropertyName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
