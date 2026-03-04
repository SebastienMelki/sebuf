package httpgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCrossFileInt64UnwrapUsesJsonMarshal asserts the generated unwrap code for the
// cross-file int64_encoding=NUMBER + unwrap scenario uses json.Marshal(item) rather than
// protojson.Marshal(item).
//
// Context: when the item type (e.g. Bar with int64_encoding=NUMBER) is defined in file A,
// and the response with the unwrap map is defined in file B that imports A, the unwrap
// generator must call json.Marshal(item) so that Bar.MarshalJSON (from the encoding
// generator) is invoked and int64 fields come out as JSON numbers — not quoted strings.
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

	// The unwrap generator must use json.Marshal(item) so that Bar.MarshalJSON is called.
	// If it uses protojson.Marshal(item) instead, the custom encoding is bypassed and
	// int64 fields with NUMBER encoding will be serialized as quoted strings at runtime.
	t.Run("no protojson.Marshal(item) for cross-file int64 NUMBER items", func(t *testing.T) {
		if strings.Contains(code, "protojson.Marshal(item)") {
			t.Error("cross_int64_service_unwrap.pb.go uses protojson.Marshal(item): " +
				"Bar.MarshalJSON will be bypassed and int64 NUMBER fields will serialize as " +
				"quoted strings instead of numbers. Fix: hasEncodingMarshalJSON must call " +
				"hasInt64NumberFields(msg) directly rather than looking up g.directEncodingMsgNames.")
		}
	})

	t.Run("json.Marshal(item) present for cross-file int64 NUMBER items", func(t *testing.T) {
		if !strings.Contains(code, "json.Marshal(item)") {
			t.Error("cross_int64_service_unwrap.pb.go should contain json.Marshal(item) " +
				"so that Bar.MarshalJSON is called when marshaling Bar items in the unwrap loop")
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
