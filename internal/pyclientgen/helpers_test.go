package pyclientgen

import "testing"

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CreateUser", "create_user"},
		{"GetNote", "get_note"},
		{"ListNotes", "list_notes"},
		{"StreamEvents", "stream_events"},
		{"GetNoteByID", "get_note_by_id"},
		{"single", "single"},
		{"A", "a"},
		{"", ""},
		// Already-lowercase-but-camelish should round-trip.
		{"userId", "user_id"},
		// All caps should collapse into one lowercase run; we do not insert
		// underscores between consecutive uppercase letters because protogen
		// returns CamelCase, not ALL_CAPS.
		{"HTTPMethod", "httpmethod"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := snakeCase(tt.input)
			if got != tt.want {
				t.Errorf("snakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHeaderOptionName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"X-API-Key", "api_key"},
		{"X-Request-ID", "request_id"},
		{"X-Tenant-ID", "tenant_id"},
		{"Authorization", "authorization"},
		{"X-Idempotency-Key", "idempotency_key"},
		{"X-Correlation-ID", "correlation_id"},
		// Lowercase x- prefix should also be stripped.
		{"x-api-key", "api_key"},
		// A header name that snake-cases into a Python keyword must escape.
		{"X-Class", "class_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := headerOptionName(tt.input)
			if got != tt.want {
				t.Errorf("headerOptionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapePyKeyword(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Hard keywords get a trailing underscore.
		{"class", "class_"},
		{"def", "def_"},
		{"from", "from_"},
		{"return", "return_"},
		{"None", "None_"},
		{"True", "True_"},
		{"False", "False_"},
		// Soft keywords are escaped too — collisions in match/case contexts
		// would otherwise crash at runtime.
		{"match", "match_"},
		{"case", "case_"},
		// Non-keyword identifiers pass through.
		{"name", "name"},
		{"resource_id", "resource_id"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapePyKeyword(tt.input)
			if got != tt.want {
				t.Errorf("escapePyKeyword(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatPyStringSet(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		want string
	}{
		{
			name: "empty set must use set() not braces",
			keys: nil,
			want: "set()",
		},
		{
			name: "empty slice also uses set()",
			keys: []string{},
			want: "set()",
		},
		{
			name: "single key",
			keys: []string{"resourceId"},
			want: `{"resourceId"}`,
		},
		{
			name: "multiple keys preserved in order",
			keys: []string{"resourceType", "resourceId"},
			want: `{"resourceType", "resourceId"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPyStringSet(tt.keys)
			if got != tt.want {
				t.Errorf("formatPyStringSet(%v) = %q, want %q", tt.keys, got, tt.want)
			}
		})
	}
}

func TestStripOptional(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Optional[str]", "str"},
		{"Optional[list[int]]", "list[int]"},
		{"Optional[dict[str, Any]]", "dict[str, Any]"},
		{"str", "str"},
		{"list[str]", "list[str]"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripOptional(tt.input)
			if got != tt.want {
				t.Errorf("stripOptional(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
