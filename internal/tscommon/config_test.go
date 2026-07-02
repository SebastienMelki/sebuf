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
			name:  "empty defaults to inline",
			param: "",
			want:  Options{ImportStyle: ImportStyleInline},
		},
		{
			name:  "import_style=modules",
			param: "import_style=modules",
			want:  Options{ImportStyle: ImportStyleModules},
		},
		{
			name:  "import_style=inline",
			param: "import_style=inline",
			want:  Options{ImportStyle: ImportStyleInline},
		},
		{
			name:  "unknown key ignored (paths=source_relative)",
			param: "paths=source_relative,import_style=modules",
			want:  Options{ImportStyle: ImportStyleModules},
		},
		{
			name:    "invalid import_style value errors",
			param:   "import_style=bogus",
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
