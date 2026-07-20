package main

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	services "github.com/SebastienMelki/sebuf/examples/market-data-unwrap/api/proto/services"
)

// An unwrap type must implement the stdlib JSON interfaces and both
// options-aware Sebuf methods.
var (
	_ json.Marshaler   = (*services.GetOptionBarsResponse)(nil)
	_ json.Unmarshaler = (*services.GetOptionBarsResponse)(nil)
	_ interface {
		MarshalJSONSebuf(protojson.MarshalOptions) ([]byte, error)
	} = (*services.GetOptionBarsResponse)(nil)
	_ interface {
		UnmarshalJSONSebuf([]byte, protojson.UnmarshalOptions) error
	} = (*services.GetOptionBarsResponse)(nil)
)

// TestUnwrapDirectDecode decodes one body with an unknown field through both
// paths on the generated type directly. The strict stdlib path must reject it
// and UnmarshalJSONSebuf with DiscardUnknown must accept it.
func TestUnwrapDirectDecode(t *testing.T) {
	body := []byte(`{"bars":{"AAPL240119C00190000":[{"c":191.2,"field_from_a_newer_server":"x"}]}}`)

	t.Run("stdlib json.Unmarshal stays strict on an unknown field", func(t *testing.T) {
		var resp services.GetOptionBarsResponse
		err := json.Unmarshal(body, &resp)
		if err == nil {
			t.Fatal("expected the strict default to reject the unknown field")
		}
		if !strings.Contains(err.Error(), "unknown field") {
			t.Errorf("error = %v, want it to name the unknown field", err)
		}
	})

	t.Run("UnmarshalJSONSebuf honors DiscardUnknown", func(t *testing.T) {
		var resp services.GetOptionBarsResponse
		if err := resp.UnmarshalJSONSebuf(body, protojson.UnmarshalOptions{DiscardUnknown: true}); err != nil {
			t.Fatalf("UnmarshalJSONSebuf with DiscardUnknown: %v", err)
		}
		bars := resp.GetBars()["AAPL240119C00190000"].GetBars()
		if len(bars) != 1 || bars[0].GetC() != 191.2 {
			t.Errorf("bars = %v, want one bar with close 191.2 surviving the unknown field", bars)
		}
	})
}
