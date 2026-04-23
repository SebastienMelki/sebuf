// This example demonstrates sebuf's forward compatibility feature.
//
// A mock API server returns JSON with extra fields not in the proto schema.
// The generated QuoteServiceClient is tested in three modes:
//  1. Default (strict) — unknown fields cause an error
//  2. WithQuoteServiceDiscardUnknownFields(true) — client-level, all RPCs tolerate unknown fields
//  3. WithQuoteServiceCallDiscardUnknownFields(false) — per-call override back to strict
//
// Run from the example directory:
//
//	cd examples/forward-compatibility && go run .
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/SebastienMelki/sebuf/examples/forward-compatibility/api/proto/services"
)

func main() {
	log.SetFlags(0)

	// Mock API server that returns known fields plus unknown fields.
	// This simulates an API that evolved and added "swap_rate" and "region"
	// which are not in our QuoteResponse proto definition.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"symbol":    "AAPL",
			"price":     185.50,
			"currency":  "USD",
			"swap_rate": 0.015,     // not in proto
			"region":    "us-east", // not in proto
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	req := &services.GetQuoteRequest{Symbol: "AAPL"}
	failed := false

	// --- 1. Default (strict mode) ---
	log.Println("=== 1. Default client (strict mode) ===")
	strictClient := services.NewQuoteServiceClient(server.URL)
	_, err := strictClient.GetQuote(ctx, req)
	if err != nil {
		log.Println("  Error:", shortenErr(err))
		log.Println("  Expected: strict mode rejects unknown fields like 'swap_rate'")
	} else {
		log.Println("  UNEXPECTED: should have failed on unknown fields")
		failed = true
	}

	// --- 2. Client-level: discard unknown fields ---
	log.Println()
	log.Println("=== 2. WithQuoteServiceDiscardUnknownFields(true) ===")
	discardClient := services.NewQuoteServiceClient(server.URL,
		services.WithQuoteServiceDiscardUnknownFields(true),
	)
	quote, err := discardClient.GetQuote(ctx, req)
	if err != nil {
		log.Println("  UNEXPECTED error:", err)
		failed = true
	} else {
		log.Printf("  OK: symbol=%s price=%.2f currency=%s", quote.Symbol, quote.Price, quote.Currency)
		log.Println("  Unknown fields 'swap_rate' and 'region' silently discarded")
	}

	// --- 3. Per-call override back to strict ---
	log.Println()
	log.Println("=== 3. Per-call override: WithQuoteServiceCallDiscardUnknownFields(false) ===")
	// Client has discard=true, but this specific call overrides to strict
	_, err = discardClient.GetQuote(ctx, req,
		services.WithQuoteServiceCallDiscardUnknownFields(false),
	)
	if err != nil {
		log.Println("  Error:", shortenErr(err))
		log.Println("  Expected: per-call override to strict rejects unknown fields")
	} else {
		log.Println("  UNEXPECTED: should have failed with per-call strict override")
		failed = true
	}

	if failed {
		os.Exit(1)
	}
}

// shortenErr trims the verbose protojson error for cleaner output.
func shortenErr(err error) string {
	s := err.Error()
	if idx := strings.Index(s, "unknown field"); idx >= 0 {
		return s[idx:]
	}
	return s
}
