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
			name:  "empty defaults to inline+flatten",
			param: "",
			want:  Options{ImportStyle: ImportStyleInline, OneofStyle: OneofStyleFlatten},
		},
		{
			name:  "import_style=modules",
			param: "import_style=modules",
			want:  Options{ImportStyle: ImportStyleModules, OneofStyle: OneofStyleFlatten},
		},
		{
			name:  "oneof_style=discriminated",
			param: "oneof_style=discriminated",
			want:  Options{ImportStyle: ImportStyleInline, OneofStyle: OneofStyleDiscriminated},
		},
		{
			name:  "unknown key ignored (paths=source_relative)",
			param: "paths=source_relative,import_style=modules",
			want:  Options{ImportStyle: ImportStyleModules, OneofStyle: OneofStyleFlatten},
		},
		{
			name:  "combined flags in any order",
			param: "oneof_style=discriminated,paths=source_relative,import_style=modules",
			want:  Options{ImportStyle: ImportStyleModules, OneofStyle: OneofStyleDiscriminated},
		},
		{
			name:  "explicit inline+flatten",
			param: "import_style=inline,oneof_style=flatten",
			want:  Options{ImportStyle: ImportStyleInline, OneofStyle: OneofStyleFlatten},
		},
		{
			name:    "invalid import_style value errors",
			param:   "import_style=bogus",
			wantErr: true,
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
