package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	services "github.com/SebastienMelki/sebuf/examples/market-data-unwrap/api/proto/services"
)

// TestUnwrapDiscardUnknownFields drives the generated client against a server
// whose response carries a field this client's proto does not know. With
// DiscardUnknownFields on, the decode must ignore it; the strict default must
// reject it.
func TestUnwrapDiscardUnknownFields(t *testing.T) {
	body := `{"bars":{"AAPL240119C00190000":[` +
		`{"t":"2025-12-15T15:05:00Z","o":190.5,"c":191.2,"v":12,` +
		`"field_from_a_newer_server":"ignore-me"}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	t.Run("unknown field is ignored when the option is on", func(t *testing.T) {
		client := services.NewMarketDataServiceClient(srv.URL,
			services.WithMarketDataServiceDiscardUnknownFields(true))

		resp, err := client.GetOptionBars(context.Background(), &services.GetOptionBarsRequest{})
		if err != nil {
			t.Fatalf("GetOptionBars with DiscardUnknownFields: %v", err)
		}

		bars := resp.GetBars()["AAPL240119C00190000"].GetBars()
		if len(bars) != 1 {
			t.Fatalf("got %d bars, want 1", len(bars))
		}
		if got := bars[0].GetC(); got != 191.2 {
			t.Errorf("close price = %v, want 191.2 (known fields must survive the unknown one)", got)
		}
	})

	t.Run("the same body fails the strict default", func(t *testing.T) {
		client := services.NewMarketDataServiceClient(srv.URL)

		_, err := client.GetOptionBars(context.Background(), &services.GetOptionBarsRequest{})
		if err == nil {
			t.Fatal("expected a strict-decoding error for the unknown field")
		}
		if !strings.Contains(err.Error(), "unknown field") {
			t.Errorf("error = %v, want it to name the unknown field", err)
		}
	})
}
