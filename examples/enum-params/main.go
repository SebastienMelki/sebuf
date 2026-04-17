package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/SebastienMelki/sebuf/examples/enum-params/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/enum-params/api/proto/services"
)

// PortfolioService demonstrates enum query and path parameters.
type PortfolioService struct {
	holdings []*models.Holding
}

func NewPortfolioService() *PortfolioService {
	return &PortfolioService{
		holdings: []*models.Holding{
			{Symbol: "AAPL", Name: "Apple Inc.", Quantity: 10, CurrentPrice: 175.50, AssetClass: models.AssetClass_ASSET_CLASS_EQUITY},
			{Symbol: "BTC", Name: "Bitcoin", Quantity: 0.5, CurrentPrice: 43000.00, AssetClass: models.AssetClass_ASSET_CLASS_CRYPTO},
			{Symbol: "GLD", Name: "Gold ETF", Quantity: 20, CurrentPrice: 185.00, AssetClass: models.AssetClass_ASSET_CLASS_COMMODITY},
		},
	}
}

func (s *PortfolioService) GetPortfolio(_ context.Context, req *models.GetPortfolioRequest) (*models.GetPortfolioResponse, error) {
	var filtered []*models.Holding
	var total float64

	for _, h := range s.holdings {
		if req.AssetClass != models.AssetClass_ASSET_CLASS_UNSPECIFIED && h.AssetClass != req.AssetClass {
			continue
		}
		filtered = append(filtered, h)
		total += h.Quantity * h.CurrentPrice
	}

	return &models.GetPortfolioResponse{
		Holdings:   filtered,
		TotalValue: total,
	}, nil
}

func (s *PortfolioService) GetAssetClass(_ context.Context, req *models.GetAssetClassRequest) (*models.GetAssetClassResponse, error) {
	var filtered []*models.Holding
	var total float64

	for _, h := range s.holdings {
		if h.AssetClass == req.AssetClass {
			filtered = append(filtered, h)
			total += h.Quantity * h.CurrentPrice
		}
	}

	return &models.GetAssetClassResponse{
		AssetClass: req.AssetClass,
		Holdings:   filtered,
		TotalValue: total,
	}, nil
}

func main() {
	service := NewPortfolioService()
	mux := http.NewServeMux()

	if err := services.RegisterPortfolioServiceServer(service, services.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("")
	fmt.Println("Enum query parameter examples:")
	fmt.Println("  curl http://localhost:8080/api/v1/portfolio?timeframe=TIMEFRAME_MONTH")
	fmt.Println("  curl http://localhost:8080/api/v1/portfolio?timeframe=3")
	fmt.Println("  curl 'http://localhost:8080/api/v1/portfolio?asset_class=ASSET_CLASS_EQUITY'")
	fmt.Println("")
	fmt.Println("Enum path parameter examples:")
	fmt.Println("  curl http://localhost:8080/api/v1/asset-classes/ASSET_CLASS_EQUITY")
	fmt.Println("  curl http://localhost:8080/api/v1/asset-classes/1")
	fmt.Println("")
	fmt.Println("Invalid enum (returns 400):")
	fmt.Println("  curl http://localhost:8080/api/v1/asset-classes/INVALID")

	log.Fatal(http.ListenAndServe(":8080", mux))
}
