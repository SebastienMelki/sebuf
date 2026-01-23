//go:build ignore

// This file demonstrates how to use the generated HTTP client.
// Run with: go run client_example.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SebastienMelki/sebuf/examples/market-data-unwrap/api/proto/services"
)

func main() {
	// Create a client with authentication headers
	client := services.NewMarketDataServiceClient(
		"http://localhost:8080",
		// Set API credentials for all requests
		services.WithMarketDataServiceAPCAAPIKEYID("test-api-key"),
		services.WithMarketDataServiceAPCAAPISECRETKEY("test-secret-key"),
		// Use a custom HTTP client with timeout
		services.WithMarketDataServiceHTTPClient(&http.Client{
			Timeout: 30 * time.Second,
		}),
	)

	ctx := context.Background()

	// Example 1: Get historical option bars
	fmt.Println("Fetching historical option bars...")
	fmt.Println("================================")
	barsResp, err := client.GetOptionBars(ctx, &services.GetOptionBarsRequest{
		Symbols:   "TSLA260123C00335000",
		Timeframe: "1Day",
		Limit:     5,
		Sort:      "desc",
	})
	if err != nil {
		log.Fatalf("Failed to get option bars: %v", err)
	}

	// Pretty print the response to show unwrap in action
	fmt.Println("\nResponse (with unwrap - bars are arrays, not wrapped objects):")
	prettyPrint(barsResp)

	// Show the structure of the bars map
	fmt.Printf("\nNumber of symbols in response: %d\n", len(barsResp.Bars))
	for symbol, barList := range barsResp.Bars {
		fmt.Printf("  %s: %d bars\n", symbol, len(barList.Bars))
		if len(barList.Bars) > 0 {
			bar := barList.Bars[0]
			fmt.Printf("    First bar: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f, Volume=%d\n",
				bar.O, bar.H, bar.L, bar.C, bar.V)
		}
	}

	if barsResp.NextPageToken != "" {
		fmt.Printf("\nNext page token: %s\n", barsResp.NextPageToken)
	}

	// Example 2: Get latest option bars
	fmt.Println("\n\nFetching latest option bars...")
	fmt.Println("==============================")
	latestResp, err := client.GetLatestOptionBars(ctx, &services.GetLatestOptionBarsRequest{
		Symbols: "TSLA260123C00335000,AAPL250117C00200000",
		Feed:    "opra",
	})
	if err != nil {
		log.Fatalf("Failed to get latest option bars: %v", err)
	}

	fmt.Println("\nResponse:")
	prettyPrint(latestResp)

	fmt.Printf("\nLatest bars for %d symbols\n", len(latestResp.Bars))
	for symbol, bar := range latestResp.Bars {
		fmt.Printf("  %s: Close=%.2f, Volume=%d, Timestamp=%s\n",
			symbol, bar.C, bar.V, bar.T)
	}

	// Example 3: Using per-call options to override headers
	fmt.Println("\n\nDemonstrating per-call options...")
	fmt.Println("=================================")
	_, err = client.GetLatestOptionBars(ctx, &services.GetLatestOptionBarsRequest{
		Symbols: "SPY250321C00500000",
		Feed:    "opra",
	},
		// Override API key for this specific request
		services.WithMarketDataServiceCallAPCAAPIKEYID("different-api-key"),
		// Add custom header
		services.WithMarketDataServiceHeader("X-Request-ID", "test-request-123"),
	)
	if err != nil {
		log.Printf("Request failed: %v", err)
	} else {
		fmt.Println("Request with per-call options succeeded")
	}

	fmt.Println("\nClient example completed!")
}

func prettyPrint(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
