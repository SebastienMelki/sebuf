package tscommon

import "testing"

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		want    Options
		wantErr bool
	}{
		{
			name:  "empty defaults to flatten",
			param: "",
			want:  Options{OneofStyle: OneofStyleFlatten},
		},
		{
			name:  "oneof_style=discriminated",
			param: "oneof_style=discriminated",
			want:  Options{OneofStyle: OneofStyleDiscriminated},
		},
		{
			name:  "oneof_style=flatten",
			param: "oneof_style=flatten",
			want:  Options{OneofStyle: OneofStyleFlatten},
		},
		{
			name:  "unknown key ignored (paths=source_relative)",
			param: "paths=source_relative,oneof_style=discriminated",
			want:  Options{OneofStyle: OneofStyleDiscriminated},
		},
		{
			name:    "invalid oneof_style value errors",
			param:   "oneof_style=bogus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOptions(tt.param)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseOptions(%q) = %+v, want error", tt.param, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseOptions(%q) unexpected error: %v", tt.param, err)
			}
			if got != tt.want {
				t.Errorf("ParseOptions(%q) = %+v, want %+v", tt.param, got, tt.want)
			}
		})
	}
}
