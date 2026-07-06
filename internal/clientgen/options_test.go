package clientgen

import (
	"strings"
	"testing"
)

func TestOptions_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name:    "camel_case is valid",
			opts:    Options{JSONNaming: JSONNamingCamelCase},
			wantErr: false,
		},
		{
			name:    "snake_case is valid",
			opts:    Options{JSONNaming: JSONNamingSnakeCase},
			wantErr: false,
		},
		{
			// Empty is normalised to the default inside NewWithOptions,
			// so validate never sees it in practice. Direct callers still
			// get an error, which catches paths that bypass the constructor.
			name:    "empty string is rejected",
			opts:    Options{JSONNaming: ""},
			wantErr: true,
		},
		{
			name:    "unknown value is rejected",
			opts:    Options{JSONNaming: "kebab-case"},
			wantErr: true,
		},
		{
			name:    "case sensitivity is enforced",
			opts:    Options{JSONNaming: "Snake_Case"},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.opts.validate()
			if tc.wantErr && err == nil {
				t.Errorf("validate() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validate() = %v, want nil", err)
			}
		})
	}

	t.Run("error mentions allowed values", func(t *testing.T) {
		t.Parallel()
		err := Options{JSONNaming: "snakecase"}.validate()
		if err == nil {
			t.Fatal("expected error for invalid value")
		}
		msg := err.Error()
		for _, want := range []string{"snakecase", JSONNamingCamelCase, JSONNamingSnakeCase} {
			if !strings.Contains(msg, want) {
				t.Errorf("error message %q should mention %q", msg, want)
			}
		}
	})
}

func TestNewWithOptions_normalises_default_JSONNaming(t *testing.T) {
	t.Parallel()

	g := NewWithOptions(nil, Options{})
	if g.opts.JSONNaming != JSONNamingCamelCase {
		t.Errorf(
			"expected JSONNaming normalised to %q, got %q",
			JSONNamingCamelCase, g.opts.JSONNaming,
		)
	}
	if err := g.opts.validate(); err != nil {
		t.Errorf("normalised default should validate, got %v", err)
	}
}

func TestNewWithOptions_preserves_explicit_JSONNaming(t *testing.T) {
	t.Parallel()

	g := NewWithOptions(nil, Options{JSONNaming: JSONNamingSnakeCase})
	if g.opts.JSONNaming != JSONNamingSnakeCase {
		t.Errorf(
			"expected JSONNaming preserved as %q, got %q",
			JSONNamingSnakeCase, g.opts.JSONNaming,
		)
	}
}

func TestNew_uses_default_options(t *testing.T) {
	t.Parallel()

	g := New(nil)
	if g.opts.JSONNaming != JSONNamingCamelCase {
		t.Errorf(
			"New(plugin) should default JSONNaming to %q, got %q",
			JSONNamingCamelCase, g.opts.JSONNaming,
		)
	}
}
