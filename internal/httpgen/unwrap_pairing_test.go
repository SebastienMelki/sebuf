package httpgen

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// minPairedTypes guards against the type discovery going blind. The regexes
// find 39 types today. The floor sits just below that, so a pattern that stops
// matching fails while removing a fixture or two does not.
const minPairedTypes = 30

// TestGoldenSebufMethodPairing asserts the pairing invariant over every
// checked-in golden file. Every type with a stdlib JSON method must also have
// its options-aware Sebuf twin, and every Sebuf method must have its stdlib
// wrapper. A type carrying only the stdlib half cannot receive protojson
// options. The test reads the goldens directly and does not run protoc.
func TestGoldenSebufMethodPairing(t *testing.T) {
	// pairingTODO lists golden files whose emitter still lacks the options-aware
	// twin, tracked in #235. Delete entries as the emitters are fixed; an empty
	// map is the goal state.
	pairingTODO := map[string]bool{
		"empty_behavior_empty_behavior.pb.go":           true,
		"flatten_flatten.pb.go":                         true,
		"nullable_nullable.pb.go":                       true,
		"oneof_discriminator_oneof_discriminator.pb.go": true,
		"timestamp_format_timestamp_format.pb.go":       true,
	}

	goldenDir := filepath.Join("testdata", "golden")
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("reading golden dir: %v", err)
	}

	marshalRe := regexp.MustCompile(`func \(\w+ \*(\w+)\) MarshalJSON\(`)
	marshalSebufRe := regexp.MustCompile(`func \(\w+ \*(\w+)\) MarshalJSONSebuf\(`)
	unmarshalRe := regexp.MustCompile(`func \(\w+ \*(\w+)\) UnmarshalJSON\(`)
	unmarshalSebufRe := regexp.MustCompile(`func \(\w+ \*(\w+)\) UnmarshalJSONSebuf\(`)

	// Enum types are exempt. Protojson options act on message fields and an
	// enum scalar has none, so enums carry only the plain pair. They are
	// recognizable by their value-receiver MarshalJSON.
	enumRe := regexp.MustCompile(`func \(\w+ (\w+)\) MarshalJSON\(`)

	pairedTypes := 0
	var skipped []string

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".pb.go") {
			continue
		}
		if pairingTODO[name] {
			skipped = append(skipped, name)
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

		seen := make(map[string]bool)
		for _, set := range []map[string]bool{marshal, marshalSebuf, unmarshal, unmarshalSebuf} {
			for typ := range set {
				seen[typ] = true
			}
		}
		pairedTypes += len(seen)

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
	}

	sort.Strings(skipped)
	for _, name := range skipped {
		t.Logf("skipped by pairingTODO (#235): %s", name)
	}

	if pairedTypes < minPairedTypes {
		t.Fatalf("checked %d types, want at least %d; the type patterns are matching nothing",
			pairedTypes, minPairedTypes)
	}
}
