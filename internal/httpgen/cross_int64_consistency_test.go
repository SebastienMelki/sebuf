package httpgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCrossFileInt64UnwrapUsesJsonMarshal asserts the generated unwrap code for the
// cross-file int64_encoding=NUMBER + unwrap scenario forwards opts to each item's
// MarshalJSONSebuf method (via an inline interface assertion) rather than calling
// protojson.Marshal(item) directly.
//
// Context: when the item type (e.g. Bar with int64_encoding=NUMBER) is defined in file A,
// and the response with the unwrap map is defined in file B that imports A, the unwrap
// generator must forward opts to Bar.MarshalJSONSebuf (from the encoding generator) so
// int64 fields come out as JSON numbers — not quoted strings.
//
// This test reads the golden file produced by TestHTTPGenGoldenFiles; run with
// UPDATE_GOLDEN=1 to (re)generate it after applying the fix.
func TestCrossFileInt64UnwrapUsesJsonMarshal(t *testing.T) {
	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	goldenFile := filepath.Join(baseDir, "testdata", "golden", "cross_int64_service_unwrap.pb.go")
	content, readErr := os.ReadFile(goldenFile)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			t.Skipf("Golden file not found: %s — run with UPDATE_GOLDEN=1 to generate it", goldenFile)
		}
		t.Fatalf("Failed to read golden file: %v", readErr)
	}

	code := string(content)

	// The unwrap generator must emit an inline interface assertion that forwards opts
	// to Bar.MarshalJSONSebuf. Direct protojson.Marshal(item) would bypass the custom
	// encoding and serialize int64 NUMBER fields as quoted strings.
	t.Run("no protojson.Marshal(item) for cross-file int64 NUMBER items", func(t *testing.T) {
		if strings.Contains(code, "protojson.Marshal(item)") {
			t.Error("cross_int64_service_unwrap.pb.go uses protojson.Marshal(item) directly: " +
				"Bar.MarshalJSONSebuf will be bypassed and int64 NUMBER fields will serialize as " +
				"quoted strings instead of numbers. The unwrap generator should emit an inline " +
				"MarshalJSONSebuf type assertion for every message-typed item.")
		}
	})

	t.Run("MarshalJSONSebuf forwarding present for cross-file int64 NUMBER items", func(t *testing.T) {
		if !strings.Contains(code, "m.MarshalJSONSebuf(opts)") {
			t.Error("cross_int64_service_unwrap.pb.go should emit an inline MarshalJSONSebuf forward " +
				"so Bar.MarshalJSONSebuf is invoked when marshaling Bar items in the unwrap loop")
		}
	})

	t.Run("json.Unmarshal used for cross-file int64 NUMBER item deserialization", func(t *testing.T) {
		// Unmarshal side: must use json.Unmarshal(itemRaw, item) not protojson.Unmarshal
		if strings.Contains(code, "protojson.Unmarshal(itemRaw, item)") {
			t.Error("cross_int64_service_unwrap.pb.go uses protojson.Unmarshal(itemRaw, item): " +
				"Bar.UnmarshalJSON will be bypassed and int64 NUMBER fields will fail to " +
				"parse from JSON numbers. Fix: hasEncodingMarshalJSON must check field annotations directly.")
		}
	})
}
