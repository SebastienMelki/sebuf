package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/SebastienMelki/sebuf/examples/market-data-unwrap/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/market-data-unwrap/api/proto/services"
)

// MarketDataServer implements the MarketDataServiceServer interface with sample data.
type MarketDataServer struct{}

// GetOptionBars returns historical option bars for the requested symbols.
func (s *MarketDataServer) GetOptionBars(ctx context.Context, req *services.GetOptionBarsRequest) (*services.GetOptionBarsResponse, error) {
	// Parse symbols from comma-separated string
	symbols := strings.Split(req.Symbols, ",")

	// Build response with sample data for each symbol
	resp := &services.GetOptionBarsResponse{
		Bars:          make(map[string]*models.OptionBarsList),
		NextPageToken: "eyJwYWdlIjogMn0=",
	}

	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		if symbol == "" {
			continue
		}

		// Create sample bars for this symbol
		resp.Bars[symbol] = &models.OptionBarsList{
			Bars: []*models.OptionBar{
				{
					T:  "2025-12-15T15:05:00Z",
					O:  143.08,
					H:  143.50,
					L:  142.90,
					C:  143.08,
					V:  1500,
					N:  45,
					Vw: 143.15,
				},
				{
					T:  "2025-12-15T16:05:00Z",
					O:  143.10,
					H:  145.80,
					L:  143.00,
					C:  145.34,
					V:  3200,
					N:  89,
					Vw: 144.50,
				},
			},
		}
	}

	return resp, nil
}

// GetLatestOptionBars returns the latest option bar for each requested symbol.
func (s *MarketDataServer) GetLatestOptionBars(ctx context.Context, req *services.GetLatestOptionBarsRequest) (*services.GetLatestOptionBarsResponse, error) {
	// Parse symbols from comma-separated string
	symbols := strings.Split(req.Symbols, ",")

	// Build response with sample data for each symbol
	resp := &services.GetLatestOptionBarsResponse{
		Bars: make(map[string]*models.OptionBar),
	}

	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		if symbol == "" {
			continue
		}

		// Create sample latest bar for this symbol
		resp.Bars[symbol] = &models.OptionBar{
			T:  "2025-12-15T16:30:00Z",
			O:  145.00,
			H:  146.20,
			L:  144.80,
			C:  145.75,
			V:  850,
			N:  28,
			Vw: 145.50,
		}
	}

	return resp, nil
}

func main() {
	// Create server with sample data
	server := &MarketDataServer{}

	// Create HTTP mux and register handlers
	mux := http.NewServeMux()
	err := services.RegisterMarketDataServiceServer(server, services.WithMux(mux))
	if err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}

	// Print server info
	fmt.Println("Market Data API Server (Unwrap Demo)")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("This demo shows the 'unwrap' annotation for map values.")
	fmt.Println("The response JSON has map values as arrays, not wrapped objects.")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET /v2/options/bars         - Get historical option bars")
	fmt.Println("  GET /v2/options/bars/latest  - Get latest option bars")
	fmt.Println()
	fmt.Println("Required headers:")
	fmt.Println("  APCA-API-KEY-ID: <your-api-key>")
	fmt.Println("  APCA-API-SECRET-KEY: <your-secret>")
	fmt.Println()
	fmt.Println("Example request:")
	fmt.Println("  curl -X GET 'http://localhost:8080/v2/options/bars?symbols=TSLA260123C00335000&timeframe=1Day&sort=desc' \\")
	fmt.Println("       -H 'APCA-API-KEY-ID: test-key' \\")
	fmt.Println("       -H 'APCA-API-SECRET-KEY: test-secret'")
	fmt.Println()
	fmt.Println("Expected response format (with unwrap):")
	fmt.Println(`  {"bars": {"TSLA260123C00335000": [{"c": 143.08, ...}]}, "nextPageToken": "..."}`)
	fmt.Println()
	fmt.Println("Without unwrap, it would be:")
	fmt.Println(`  {"bars": {"TSLA260123C00335000": {"bars": [{"c": 143.08, ...}]}}, "nextPageToken": "..."}`)
	fmt.Println()
	fmt.Println("Server starting on :8080...")
	fmt.Println()

	// Start server
	log.Fatal(http.ListenAndServe(":8080", mux))
}
