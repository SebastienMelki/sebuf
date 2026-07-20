package main

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	models "github.com/SebastienMelki/sebuf/examples/market-data-unwrap/api/proto/models"
)

// Every root unwrap shape implements the stdlib interfaces and both
// options-aware methods.
var (
	_ json.Marshaler   = (*models.TradeList)(nil)
	_ json.Unmarshaler = (*models.TradeList)(nil)
	_ interface {
		MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error)
	} = (*models.TradeList)(nil)
	_ interface {
		UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error
	} = (*models.TradeList)(nil)
	_ interface {
		UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error
	} = (*models.QuotesBySymbol)(nil)
	_ interface {
		UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error
	} = (*models.SymbolNames)(nil)
	_ interface {
		UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error
	} = (*models.BarsBySymbol)(nil)
	_ interface {
		UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error
	} = (*models.BarListSequence)(nil)
)

func discard() protojson.UnmarshalOptions {
	return protojson.UnmarshalOptions{DiscardUnknown: true}
}

func wantUnknownFieldError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected the strict default to reject the unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("error = %v, want it to name the unknown field", err)
	}
}

// TestRootRepeatedUnwrapDecode covers the bare-array shape, the one from the
// original issue.
func TestRootRepeatedUnwrapDecode(t *testing.T) {
	body := []byte(`[{"i":"t1","p":1.5,"field_from_a_newer_server":"x"}]`)

	var strict models.TradeList
	wantUnknownFieldError(t, json.Unmarshal(body, &strict))

	var lenient models.TradeList
	if err := lenient.UnmarshalJSONSebuf(body, discard()); err != nil {
		t.Fatalf("UnmarshalJSONSebuf with DiscardUnknown: %v", err)
	}
	if trades := lenient.GetTrades(); len(trades) != 1 || trades[0].GetP() != 1.5 {
		t.Errorf("trades = %v, want one trade with price 1.5", trades)
	}
}

// TestRootMapMessageValuesDecode covers a root map with message values.
func TestRootMapMessageValuesDecode(t *testing.T) {
	body := []byte(`{"AAPL":{"bp":1.1,"ap":1.2,"field_from_a_newer_server":"x"}}`)

	var strict models.QuotesBySymbol
	wantUnknownFieldError(t, json.Unmarshal(body, &strict))

	var lenient models.QuotesBySymbol
	if err := lenient.UnmarshalJSONSebuf(body, discard()); err != nil {
		t.Fatalf("UnmarshalJSONSebuf with DiscardUnknown: %v", err)
	}
	if q := lenient.GetQuotes()["AAPL"]; q.GetAp() != 1.2 {
		t.Errorf("quote = %v, want ask 1.2", q)
	}
}

// TestRootMapScalarValuesDecode covers a root map with scalar values, where
// both paths must agree since there is no per-value message to carry unknown
// fields.
func TestRootMapScalarValuesDecode(t *testing.T) {
	body := []byte(`{"AAPL":"Apple Inc"}`)

	var viaWrapper models.SymbolNames
	if err := json.Unmarshal(body, &viaWrapper); err != nil {
		t.Fatalf("wrapper decode: %v", err)
	}
	var viaSebuf models.SymbolNames
	if err := viaSebuf.UnmarshalJSONSebuf(body, discard()); err != nil {
		t.Fatalf("Sebuf decode: %v", err)
	}
	if viaWrapper.GetNames()["AAPL"] != "Apple Inc" || viaSebuf.GetNames()["AAPL"] != "Apple Inc" {
		t.Errorf("names = %v / %v, want both to hold Apple Inc", viaWrapper.GetNames(), viaSebuf.GetNames())
	}
}

// TestRootMapWithValueUnwrapDecode covers the combined shape, so DiscardUnknown
// has to survive two unwrap hops before it reaches the bar.
func TestRootMapWithValueUnwrapDecode(t *testing.T) {
	body := []byte(`{"AAPL":[{"c":191.2,"field_from_a_newer_server":"x"}]}`)

	var strict models.BarsBySymbol
	wantUnknownFieldError(t, json.Unmarshal(body, &strict))

	var lenient models.BarsBySymbol
	if err := lenient.UnmarshalJSONSebuf(body, discard()); err != nil {
		t.Fatalf("UnmarshalJSONSebuf with DiscardUnknown: %v", err)
	}
	if bars := lenient.GetBars()["AAPL"].GetBars(); len(bars) != 1 || bars[0].GetC() != 191.2 {
		t.Errorf("bars = %v, want one bar with close 191.2", bars)
	}
}

// TestChildSebufDispatchRuns proves the inline child dispatch takes the Sebuf
// branch at runtime, not just in the generated text. Each element of
// BarListSequence is itself a root-unwrap message, so its JSON form is a bare
// array. A plain protojson call on the element would reject an array where it
// expects an object, which means these decodes can only succeed by routing
// through the element's own UnmarshalJSONSebuf.
func TestChildSebufDispatchRuns(t *testing.T) {
	t.Run("bare-array elements decode through the child's method", func(t *testing.T) {
		body := []byte(`[[{"c":1.5}],[{"c":2.5},{"c":3.5}]]`)
		var seq models.BarListSequence
		if err := seq.UnmarshalJSONSebuf(body, protojson.UnmarshalOptions{}); err != nil {
			t.Fatalf("UnmarshalJSONSebuf: %v", err)
		}
		lists := seq.GetLists()
		if len(lists) != 2 || len(lists[1].GetBars()) != 2 || lists[1].GetBars()[1].GetC() != 3.5 {
			t.Errorf("lists = %v, want two lists with the last bar closing at 3.5", lists)
		}
	})

	t.Run("DiscardUnknown reaches through both hops", func(t *testing.T) {
		body := []byte(`[[{"c":1.5,"field_from_a_newer_server":"x"}]]`)

		var strict models.BarListSequence
		wantUnknownFieldError(t, json.Unmarshal(body, &strict))

		var lenient models.BarListSequence
		if err := lenient.UnmarshalJSONSebuf(body, discard()); err != nil {
			t.Fatalf("UnmarshalJSONSebuf with DiscardUnknown: %v", err)
		}
		if bars := lenient.GetLists()[0].GetBars(); len(bars) != 1 || bars[0].GetC() != 1.5 {
			t.Errorf("bars = %v, want one bar with close 1.5", bars)
		}
	})
}
