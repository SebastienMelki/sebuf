package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGoldenSebufMethodPairing asserts the pairing invariant over the checked-in
// *_unwrap.pb.go and *_encoding.pb.go golden files. Every type there with a
// stdlib JSON method must also have its options-aware Sebuf twin, and every
// Sebuf method must have its stdlib wrapper. A type carrying only the stdlib
// half cannot receive protojson options. The test reads the goldens directly
// and does not run protoc.
func TestGoldenSebufMethodPairing(t *testing.T) {
	goldenDir := filepath.Join("testdata", "golden")
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("reading golden dir: %v", err)
	}

	marshalRe := regexp.MustCompile(`func \(x \*(\w+)\) MarshalJSON\(`)
	marshalSebufRe := regexp.MustCompile(`func \(x \*(\w+)\) MarshalJSONSebuf\(`)
	unmarshalRe := regexp.MustCompile(`func \(x \*(\w+)\) UnmarshalJSON\(`)
	unmarshalSebufRe := regexp.MustCompile(`func \(x \*(\w+)\) UnmarshalJSONSebuf\(`)

	checked := 0
	// Enum types are exempt. Protojson options act on message fields and an
	// enum scalar has none, so enums carry only the plain pair. They are
	// recognizable by their value-receiver MarshalJSON.
	enumRe := regexp.MustCompile(`func \(x (\w+)\) MarshalJSON\(`)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, "_unwrap.pb.go") && !strings.HasSuffix(name, "_encoding.pb.go") {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(goldenDir, name))
		if readErr != nil {
			t.Fatalf("reading %s: %v", name, readErr)
		}
		code := string(content)

		types := func(re *regexp.Regexp) map[string]bool {
			out := make(map[string]bool)
			for _, m := range re.FindAllStringSubmatch(code, -1) {
				out[m[1]] = true
			}
			return out
		}

		marshal, marshalSebuf := types(marshalRe), types(marshalSebufRe)
		unmarshal, unmarshalSebuf := types(unmarshalRe), types(unmarshalSebufRe)
		for _, m := range enumRe.FindAllStringSubmatch(code, -1) {
			delete(marshal, m[1])
			delete(unmarshal, m[1])
		}

		for typ := range marshal {
			if !marshalSebuf[typ] {
				t.Errorf("%s: %s has MarshalJSON but no MarshalJSONSebuf; opts cannot reach it", name, typ)
			}
		}
		for typ := range unmarshal {
			if !unmarshalSebuf[typ] {
				t.Errorf("%s: %s has UnmarshalJSON but no UnmarshalJSONSebuf; opts cannot reach it", name, typ)
			}
		}
		for typ := range marshalSebuf {
			if !marshal[typ] {
				t.Errorf("%s: %s has MarshalJSONSebuf but no MarshalJSON wrapper for stdlib callers", name, typ)
			}
		}
		for typ := range unmarshalSebuf {
			if !unmarshal[typ] {
				t.Errorf("%s: %s has UnmarshalJSONSebuf but no UnmarshalJSON wrapper for stdlib callers", name, typ)
			}
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("no _unwrap.pb.go or _encoding.pb.go golden files found; the invariant checked nothing")
	}
}
