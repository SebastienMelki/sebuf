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

	// Each enum field shape must be patched by json name.
	for _, field := range []string{
		`raw["status"]`,         // singular
		`raw["statusList"]`,     // repeated
		`raw["optionalStatus"]`, // proto3 optional
		`raw["statusMap"]`,      // map value
	} {
		if !strings.Contains(src, field) {
			t.Errorf("generated marshaler does not patch enum field %s", field)
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
