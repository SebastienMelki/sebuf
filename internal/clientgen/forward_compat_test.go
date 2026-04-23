package clientgen

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// unmarshalResponse mirrors the generated unmarshalResponse method exactly.
// This lets us test the same logic the generated clients use.
func unmarshalResponse(body []byte, msg proto.Message, discardUnknown bool) error {
	if len(body) == 0 {
		return nil
	}
	if discardUnknown {
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		return opts.Unmarshal(body, msg)
	}
	return protojson.Unmarshal(body, msg)
}

// TestUnmarshalResponseBehavior tests the generated unmarshalResponse pattern
// end-to-end with an httptest.Server returning JSON with unknown fields.
func TestUnmarshalResponseBehavior(t *testing.T) {
	// Server returns a known field plus an unknown field
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"name":      "test.proto",
			"package":   "api.v1",
			"swap_rate": 0.015, // unknown field
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Fetch the response body (shared across subtests)
	fetchBody := func(t *testing.T) []byte {
		t.Helper()
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("failed to fetch: %v", err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		return body
	}

	t.Run("default strict mode rejects unknown fields", func(t *testing.T) {
		body := fetchBody(t)
		msg := &descriptorpb.FileDescriptorProto{}

		// discardUnknown=false → strict mode (default)
		err := unmarshalResponse(body, msg, false)
		if err == nil {
			t.Fatal("strict mode should reject unknown fields, but succeeded")
		}
		if !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("expected 'unknown field' error, got: %v", err)
		}
	})

	t.Run("client-level discard unknown fields", func(t *testing.T) {
		body := fetchBody(t)
		msg := &descriptorpb.FileDescriptorProto{}

		// discardUnknown=true → simulates WithXxxDiscardUnknownFields(true)
		err := unmarshalResponse(body, msg, true)
		if err != nil {
			t.Fatalf("discard mode should accept unknown fields, got: %v", err)
		}
		if msg.GetName() != "test.proto" {
			t.Errorf("expected name='test.proto', got %q", msg.GetName())
		}
		if msg.GetPackage() != "api.v1" {
			t.Errorf("expected package='api.v1', got %q", msg.GetPackage())
		}
	})

	t.Run("per-call override back to strict", func(t *testing.T) {
		body := fetchBody(t)
		msg := &descriptorpb.FileDescriptorProto{}

		// Simulate: client has discardUnknown=true, but per-call overrides to false
		// Mirrors the generated pattern:
		//   discardUnknown := c.discardUnknownFields        // true (client default)
		//   if callOpts.discardUnknownFields != nil {
		//       discardUnknown = *callOpts.discardUnknownFields  // false (per-call override)
		//   }
		clientDefault := true
		callOverride := false
		_ = clientDefault // client would set true, but per-call overrides

		err := unmarshalResponse(body, msg, callOverride)
		if err == nil {
			t.Fatal("per-call override to strict should reject unknown fields, but succeeded")
		}
	})
}

// TestForwardCompatibility verifies that protojson.UnmarshalOptions{DiscardUnknown: true}
// correctly handles unknown JSON fields of all types.
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

// TestSSEGoldenHasDiscardUnknownFields checks that the SSE golden file
// threads discardUnknownFields into EventStream.
func TestSSEGoldenHasDiscardUnknownFields(t *testing.T) {
	goldenFile := "testdata/golden/sse_client.pb.go"
	content, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read SSE golden file: %v", err)
	}

	src := string(content)

	checks := []struct {
		pattern string
		desc    string
	}{
		{"discardUnknownFields bool", "EventStream struct has discardUnknownFields field"},
		{"s.discardUnknownFields", "Next() checks discardUnknownFields"},
		{"discardUnknownFields: discardUnknown,", "stream creation passes resolved flag"},
	}

	for _, check := range checks {
		if !strings.Contains(src, check.pattern) {
			t.Errorf("SSE golden file missing expected pattern: %s (%s)", check.pattern, check.desc)
		}
	}
}
