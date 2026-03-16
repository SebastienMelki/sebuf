package tscommon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestTSScalarType(t *testing.T) {
	tests := []struct {
		name string
		kind protoreflect.Kind
		want string
	}{
		{name: "StringKind", kind: protoreflect.StringKind, want: "string"},
		{name: "BoolKind", kind: protoreflect.BoolKind, want: "boolean"},
		{name: "Int32Kind", kind: protoreflect.Int32Kind, want: "number"},
		{name: "Sint32Kind", kind: protoreflect.Sint32Kind, want: "number"},
		{name: "Sfixed32Kind", kind: protoreflect.Sfixed32Kind, want: "number"},
		{name: "Uint32Kind", kind: protoreflect.Uint32Kind, want: "number"},
		{name: "Fixed32Kind", kind: protoreflect.Fixed32Kind, want: "number"},
		{name: "FloatKind", kind: protoreflect.FloatKind, want: "number"},
		{name: "DoubleKind", kind: protoreflect.DoubleKind, want: "number"},
		{name: "Int64Kind", kind: protoreflect.Int64Kind, want: "string"},
		{name: "Sint64Kind", kind: protoreflect.Sint64Kind, want: "string"},
		{name: "Sfixed64Kind", kind: protoreflect.Sfixed64Kind, want: "string"},
		{name: "Uint64Kind", kind: protoreflect.Uint64Kind, want: "string"},
		{name: "Fixed64Kind", kind: protoreflect.Fixed64Kind, want: "string"},
		{name: "BytesKind", kind: protoreflect.BytesKind, want: "string"},
		{name: "EnumKind", kind: protoreflect.EnumKind, want: "string"},
		{name: "MessageKind", kind: protoreflect.MessageKind, want: "unknown"},
		{name: "GroupKind", kind: protoreflect.GroupKind, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TSScalarType(tt.kind)
			if got != tt.want {
				t.Errorf("TSScalarType(%v) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestTSZeroCheck(t *testing.T) {
	tests := []struct {
		name      string
		fieldKind string
		want      string
	}{
		{name: "string", fieldKind: "string", want: ` !== ""`},
		{name: "bool", fieldKind: "bool", want: ""},
		{name: "int32", fieldKind: "int32", want: " !== 0"},
		{name: "sint32", fieldKind: "sint32", want: " !== 0"},
		{name: "sfixed32", fieldKind: "sfixed32", want: " !== 0"},
		{name: "uint32", fieldKind: "uint32", want: " !== 0"},
		{name: "fixed32", fieldKind: "fixed32", want: " !== 0"},
		{name: "float", fieldKind: "float", want: " !== 0"},
		{name: "double", fieldKind: "double", want: " !== 0"},
		{name: "int64", fieldKind: "int64", want: ` !== "0"`},
		{name: "sint64", fieldKind: "sint64", want: ` !== "0"`},
		{name: "sfixed64", fieldKind: "sfixed64", want: ` !== "0"`},
		{name: "uint64", fieldKind: "uint64", want: ` !== "0"`},
		{name: "fixed64", fieldKind: "fixed64", want: ` !== "0"`},
		{name: "enum", fieldKind: "enum", want: ""},
		{name: "unknown_kind", fieldKind: "unknown_kind", want: ` !== ""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TSZeroCheck(tt.fieldKind)
			if got != tt.want {
				t.Errorf("TSZeroCheck(%q) = %q, want %q", tt.fieldKind, got, tt.want)
			}
		})
	}
}

// readGoldenFile reads a golden file relative to the project root and returns its content.
func readGoldenFile(t *testing.T, projectRoot, relPath string) string {
	t.Helper()
	goldenPath := filepath.Join(projectRoot, relPath)
	content, readErr := os.ReadFile(goldenPath)
	if readErr != nil {
		t.Fatalf("Failed to read golden file %s: %v", relPath, readErr)
	}
	return string(content)
}

// TestTSEnumUnspecifiedValue_ViaGoldenOutput validates TSEnumUnspecifiedValue behavior
// through the golden file output rather than direct function calls.
// TSEnumUnspecifiedValue requires a real *protogen.Field with populated Enum and
// extension options, which cannot be easily mocked. Instead, we verify the generated
// output captures the correct behavior: custom enum_value annotations produce custom
// strings, while enums without annotations use the proto name.
func TestTSEnumUnspecifiedValue_ViaGoldenOutput(t *testing.T) {
	// Find project root from the test's working directory (internal/tscommon/)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..")

	t.Run("custom_enum_value_Status", func(t *testing.T) {
		s := readGoldenFile(
			t, projectRoot,
			"internal/tsclientgen/testdata/golden/enum_encoding_client.ts",
		)

		// Status enum has custom enum_value annotations: "unknown", "active", "inactive"
		// TSEnumUnspecifiedValue should return "unknown" (custom) for first value
		expected := `export type Status = "unknown" | "active" | "inactive";`
		if !strings.Contains(s, expected) {
			t.Errorf("Expected Status type with custom enum_value annotations:\n  %s", expected)
		}
	})

	t.Run("default_enum_value_Priority", func(t *testing.T) {
		s := readGoldenFile(
			t, projectRoot,
			"internal/tsclientgen/testdata/golden/enum_encoding_client.ts",
		)

		// Priority enum has NO custom enum_value annotations
		// TSEnumUnspecifiedValue should return "PRIORITY_LOW" (proto name) for first value
		expected := `export type Priority = "PRIORITY_LOW" | "PRIORITY_MEDIUM" | "PRIORITY_HIGH";`
		if !strings.Contains(s, expected) {
			t.Errorf("Expected Priority type with proto names:\n  %s", expected)
		}
	})

	t.Run("custom_enum_value_Region_query_params", func(t *testing.T) {
		s := readGoldenFile(
			t, projectRoot,
			"internal/tsclientgen/testdata/golden/query_params_client.ts",
		)

		// Region enum has custom enum_value annotations
		expected := `export type Region = "unspecified" | "americas" | "europe" | "asia";`
		if !strings.Contains(s, expected) {
			t.Errorf("Expected Region type with custom enum_value annotations:\n  %s", expected)
		}

		// TSZeroCheckForField for enum query params should use custom unspecified value
		if !strings.Contains(s, `req.region !== "unspecified"`) {
			t.Error(
				`Expected zero check to use custom enum_value "unspecified" for Region query param`,
			)
		}
	})
}
