package oneofhelper

import (
	"testing"
)

func TestLowerFirstSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character uppercase",
			input:    "A",
			expected: "a",
		},
		{
			name:     "single character lowercase",
			input:    "a",
			expected: "a",
		},
		{
			name:     "pascal case",
			input:    "UserName",
			expected: "userName",
		},
		{
			name:     "all uppercase",
			input:    "API",
			expected: "aPI",
		},
		{
			name:     "already lowercase",
			input:    "userName",
			expected: "userName",
		},
		{
			name:     "single letter followed by numbers",
			input:    "Id123",
			expected: "id123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LowerFirst(tt.input)
			if result != tt.expected {
				t.Errorf("lowerFirst(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
