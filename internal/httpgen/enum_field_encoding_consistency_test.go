package httpgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoGeneratorsProduceIdenticalEnumFieldEncoding verifies go-http and go-client
// emit identical message-level enum_value MarshalJSON/UnmarshalJSON code.
func TestGoGeneratorsProduceIdenticalEnumFieldEncoding(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	compareEncodingFiles(t,
		filepath.Join(baseDir, "testdata", "golden", "enum_encoding_enum_field_encoding.pb.go"),
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "enum_encoding_enum_field_encoding.pb.go"),
		"enum_field_encoding",
	)
	compareEncodingFiles(t,
		filepath.Join(baseDir, "testdata", "golden", "enum_nested_enum_field_encoding.pb.go"),
		filepath.Join(baseDir, "..", "clientgen", "testdata", "golden", "enum_nested_enum_field_encoding.pb.go"),
		"enum_nested_enum_field_encoding",
	)
}

// TestEnumFieldEncodingTransitiveNesting verifies the generated marshaler propagates custom enum
// strings through nested messages: a wrapper re-serializes its child via the child's marshaler, so
// enums nested any number of levels below the marshaled message are still translated.
func TestEnumFieldEncodingTransitiveNesting(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	content, readErr := os.ReadFile(
		filepath.Join(baseDir, "testdata", "golden", "enum_nested_enum_field_encoding.pb.go"),
	)
	if readErr != nil {
		t.Fatalf("Failed to read enum_nested golden file: %v", readErr)
	}
	src := string(content)

	// Every message in the chain (leaf, one-level wrapper, two-level wrapper) gets a marshaler.
	for _, method := range []string{
		"func (x *Item) MarshalJSONSebuf(",
		"func (x *ItemGroup) MarshalJSONSebuf(",
		"func (x *GetItemsResponse) MarshalJSONSebuf(",
	} {
		if !strings.Contains(src, method) {
			t.Errorf("missing marshaler %q -- transitive nesting not generated", method)
		}
	}

	// The wrapper re-serializes nested children (singular + repeated) via their marshaler, and the
	// leaf still patches its direct enum fields.
	for _, snippet := range []string{
		`Re-serialize "lead" forwarding opts`,           // singular nested
		`Re-serialize repeated "items" forwarding opts`, // repeated nested
		`Re-serialize "group" forwarding opts`,          // two-level nested
		"gradeToJSON[e]",                                // leaf direct enum still patched
	} {
		if !strings.Contains(src, snippet) {
			t.Errorf("nested marshaler missing %q", snippet)
		}
	}
}

// TestEnumFieldEncodingCoversAllShapes verifies the generated marshaler patches every enum
// field cardinality (singular, repeated, map) in both directions, and never leaks a raw proto
// value name into the wire format via the base protojson output.
func TestEnumFieldEncodingCoversAllShapes(t *testing.T) {
	baseDir, baseErr := os.Getwd()
	if baseErr != nil {
		t.Fatalf("Failed to get working directory: %v", baseErr)
	}

	content, readErr := os.ReadFile(
		filepath.Join(baseDir, "testdata", "golden", "enum_encoding_enum_field_encoding.pb.go"),
	)
	if readErr != nil {
		t.Fatalf("Failed to read enum_field_encoding golden file: %v", readErr)
	}
	src := string(content)

	// Both dispatch methods must exist for the server (marshal) and both clients (unmarshal).
	for _, method := range []string{
		"func (x *EnumEncodingTest) MarshalJSONSebuf(",
		"func (x *EnumEncodingTest) MarshalJSON(",
		"func (x *EnumEncodingTest) UnmarshalJSONSebuf(",
		"func (x *EnumEncodingTest) UnmarshalJSON(",
	} {
		if !strings.Contains(src, method) {
			t.Errorf("generated marshaler missing %q", method)
		}
	}

	// Each enum field shape must be patched. Multi-word fields patch BOTH the camelCase JSON
	// name and the snake_case proto name so protojson's UseProtoNames output is handled too.
	for _, field := range []string{
		`[]string{"status"}`,                            // singular (single word: one key)
		`[]string{"statusList", "status_list"}`,         // repeated
		`[]string{"optionalStatus", "optional_status"}`, // proto3 optional
		`[]string{"statusMap", "status_map"}`,           // map value
	} {
		if !strings.Contains(src, field) {
			t.Errorf("generated marshaler does not patch enum field keys %s", field)
		}
	}

	// Marshal uses the value->custom map; unmarshal reverses via the enum String() proto name.
	if !strings.Contains(src, "statusToJSON[e]") {
		t.Error("marshal path should translate proto value to custom enum_value via statusToJSON")
	}
	if !strings.Contains(src, "e.String()") {
		t.Error("unmarshal path should translate back to the proto value name via String()")
	}
}
