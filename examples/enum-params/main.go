package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/SebastienMelki/sebuf/examples/enum-params/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/enum-params/api/proto/services"
)

// sampleHoldings is a simple in-memory portfolio.
var sampleHoldings = []*models.Holding{
	{Symbol: "AAPL", Name: "Apple Inc.", AssetClass: models.AssetClass_ASSET_CLASS_EQUITY, Quantity: 50, CurrentPrice: 195.0, TotalValue: 9750.0},
	{Symbol: "GOOGL", Name: "Alphabet Inc.", AssetClass: models.AssetClass_ASSET_CLASS_EQUITY, Quantity: 20, CurrentPrice: 140.0, TotalValue: 2800.0},
	{Symbol: "BTC", Name: "Bitcoin", AssetClass: models.AssetClass_ASSET_CLASS_CRYPTO, Quantity: 0.5, CurrentPrice: 65000.0, TotalValue: 32500.0},
	{Symbol: "GLD", Name: "Gold ETF", AssetClass: models.AssetClass_ASSET_CLASS_COMMODITY, Quantity: 100, CurrentPrice: 220.0, TotalValue: 22000.0},
	{Symbol: "TLT", Name: "20+ Year Treasury Bond ETF", AssetClass: models.AssetClass_ASSET_CLASS_FIXED_INCOME, Quantity: 30, CurrentPrice: 92.0, TotalValue: 2760.0},
}

type server struct{}

func (s *server) GetPortfolio(_ context.Context, req *services.GetPortfolioRequest) (*models.PortfolioSummary, error) {
	var total float64
	for _, h := range sampleHoldings {
		total += h.TotalValue
	}
	return &models.PortfolioSummary{
		Holdings:   sampleHoldings,
		TotalValue: total,
		Timeframe:  req.Timeframe,
		Count:      int32(len(sampleHoldings)),
	}, nil
}

func (s *server) GetByAssetClass(_ context.Context, req *services.GetByAssetClassRequest) (*models.PortfolioSummary, error) {
	var filtered []*models.Holding
	var total float64
	for _, h := range sampleHoldings {
		if h.AssetClass == req.AssetClass {
			filtered = append(filtered, h)
			total += h.TotalValue
		}
	}
	return &models.PortfolioSummary{
		Holdings:   filtered,
		TotalValue: total,
		Timeframe:  req.Timeframe,
		Count:      int32(len(filtered)),
	}, nil
}

func main() {
	mux := http.NewServeMux()

	if err := services.RegisterPortfolioServiceServer(&server{}, services.WithMux(mux)); err != nil {
		log.Fatalf("Failed to register: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Enum Params Example Server")
	fmt.Println("==========================")
	fmt.Printf("Listening on :%s\n\n", port)
	fmt.Println("Try these URLs:")
	fmt.Println("  Enum query param (by name):")
	fmt.Printf("    curl http://localhost:%s/api/v1/portfolio?timeframe=TIMEFRAME_1D\n", port)
	fmt.Println("  Enum query param (by number):")
	fmt.Printf("    curl http://localhost:%s/api/v1/portfolio?timeframe=3\n", port)
	fmt.Println("  Enum path param (by name):")
	fmt.Printf("    curl http://localhost:%s/api/v1/portfolio/asset-class/ASSET_CLASS_EQUITY\n", port)
	fmt.Println("  Enum path + query param:")
	fmt.Printf("    curl http://localhost:%s/api/v1/portfolio/asset-class/ASSET_CLASS_CRYPTO?timeframe=TIMEFRAME_ALL\n", port)
	fmt.Println("")

	log.Fatal(http.ListenAndServe(":"+port, mux))
}
