package clientgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TestForwardCompatibility verifies that protojson.UnmarshalOptions{DiscardUnknown: true}
// correctly handles unknown JSON fields — the behavior generated clients use when
// WithXxxDiscardUnknownFields(true) is set.
func TestForwardCompatibility(t *testing.T) {
	tests := []struct {
		name string
		json string
		desc string
	}{
		{
			name: "single unknown field",
			json: `{"name": "user.proto", "package": "api.v1", "swap_rate": 0.015}`,
			desc: "API added a new numeric field",
		},
		{
			name: "multiple unknown fields",
			json: `{"name": "user.proto", "new_field": "value", "another_new": 42, "nested_new": {"key": "val"}}`,
			desc: "API added several new fields of different types",
		},
		{
			name: "unknown nested object",
			json: `{"name": "user.proto", "metadata": {"region": "us-east-1", "version": 3}}`,
			desc: "API added a new nested object field",
		},
		{
			name: "unknown array field",
			json: `{"name": "user.proto", "tags": ["alpha", "beta"]}`,
			desc: "API added a new array field",
		},
		{
			name: "unknown boolean field",
			json: `{"name": "user.proto", "deprecated": true}`,
			desc: "API added a new boolean field",
		},
		{
			name: "unknown null field",
			json: `{"name": "user.proto", "optional_field": null}`,
			desc: "API added a new nullable field set to null",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(tc.json)

			msg := &descriptorpb.FileDescriptorProto{}
			opts := protojson.UnmarshalOptions{DiscardUnknown: true}
			err := opts.Unmarshal(body, msg)
			if err != nil {
				t.Fatalf("DiscardUnknown should handle unknown fields gracefully (%s), got: %v", tc.desc, err)
			}

			if msg.GetName() != "user.proto" {
				t.Errorf("known field 'name' should be parsed correctly, got %q", msg.GetName())
			}
		})
	}
}

// TestStrictModeRejectsUnknownFields confirms that bare protojson.Unmarshal
// rejects unknown fields — the default behavior when DiscardUnknownFields is not set.
func TestStrictModeRejectsUnknownFields(t *testing.T) {
	body := []byte(`{"name": "user.proto", "swap_rate": 0.015}`)

	msg := &descriptorpb.FileDescriptorProto{}
	err := protojson.Unmarshal(body, msg)
	if err == nil {
		t.Fatal("bare protojson.Unmarshal should reject unknown fields, but it succeeded")
	}

	t.Logf("confirmed: strict mode rejects unknown fields: %v", err)
}

// TestGeneratedCodeHasDiscardUnknownFieldsOption checks that generated golden files
// contain the WithXxxDiscardUnknownFields client option and call option.
func TestGeneratedCodeHasDiscardUnknownFieldsOption(t *testing.T) {
	goldenDir := "testdata/golden"

	// Use backward_compat as a representative golden file
	goldenFile := filepath.Join(goldenDir, "backward_compat_client.pb.go")
	content, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	src := string(content)

	checks := []struct {
		pattern string
		desc    string
	}{
		{"discardUnknownFields bool", "client struct has discardUnknownFields field"},
		{"discardUnknownFields *bool", "call options struct has discardUnknownFields pointer field"},
		{"DiscardUnknownFields(discard bool)", "client option function exists"},
		{"CallDiscardUnknownFields(discard bool)", "call option function exists"},
		{"discardUnknown := c.discardUnknownFields", "runtime resolution reads client default"},
		{"callOpts.discardUnknownFields != nil", "runtime resolution checks per-call override"},
		{"discardUnknown = *callOpts.discardUnknownFields", "runtime resolution applies per-call override"},
		{"if discardUnknown {", "conditional DiscardUnknown in unmarshalResponse"},
	}

	for _, check := range checks {
		if !strings.Contains(src, check.pattern) {
			t.Errorf("golden file missing expected pattern: %s (%s)", check.pattern, check.desc)
		}
	}
}
