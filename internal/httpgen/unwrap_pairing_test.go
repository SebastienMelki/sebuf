package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGoldenSebufMethodPairing pins the structural invariant behind issue #204:
// every generated type with a stdlib JSON method must also have its options-aware
// Sebuf twin, in both directions. The original bug was exactly a broken pairing
// (unwrap types had UnmarshalJSON but no UnmarshalJSONSebuf, so the client
// dispatcher could never pass DiscardUnknown through). Scanning the checked-in
// golden files needs no protoc, so this runs everywhere and catches the whole
// class if any emitter regresses.
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
	for _, entry := range entries {
		name := entry.Name()
		// Scoped to the unwrap emitters' output. The encoding emitters have
		// their own pairing status (int64 pairs both methods; bytes and enum
		// currently do not) and are outside this invariant.
		if !strings.HasSuffix(name, "_unwrap.pb.go") {
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
