// Package main demonstrates enum_value JSON encoding on the generated Go HTTP
// server. The proto RiskLevel enum carries (sebuf.http.enum_value) annotations
// ("low"/"medium"/"high"). The server both emits those custom strings in
// responses and accepts them in requests — matching the OpenAPI docs and the
// TypeScript/Python clients, instead of the raw proto names (RISK_LEVEL_LOW).
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	api "github.com/SebastienMelki/sebuf/examples/enum-encoding/api/proto/services"
)

// suggestionHandler returns a curated suggestion. It echoes the request's
// requested_risk into the response, which proves the server parsed the custom
// enum string ("low") on the way in as well as emitting it on the way out.
type suggestionHandler struct{}

func (suggestionHandler) GetSuggestion(
	_ context.Context,
	req *api.GetSuggestionRequest,
) (*api.EasyOptionSuggestion, error) {
	risk := req.GetRequestedRisk()
	if risk == api.RiskLevel_RISK_LEVEL_UNSPECIFIED {
		risk = api.RiskLevel_RISK_LEVEL_LOW
	}
	return &api.EasyOptionSuggestion{
		Symbol:              req.GetSymbol(),
		ProbabilityOfProfit: 0.62,
		RiskLevel:           risk,
		AlternateRiskLevels: []api.RiskLevel{
			api.RiskLevel_RISK_LEVEL_MEDIUM,
			api.RiskLevel_RISK_LEVEL_HIGH,
		},
		RiskBySymbol: map[string]api.RiskLevel{
			"AAPL": api.RiskLevel_RISK_LEVEL_LOW,
			"TSLA": api.RiskLevel_RISK_LEVEL_HIGH,
		},
	}, nil
}

func main() {
	mux := http.NewServeMux()
	if err := api.RegisterSuggestionServiceServer(
		suggestionHandler{},
		api.WithMux(mux),
	); err != nil {
		log.Fatalf("register server: %v", err)
	}

	fmt.Println("listening on :8080")
	fmt.Println(`  curl -s localhost:8080/api/v1/suggestion -d '{"symbol":"AAPL","requestedRisk":"low"}'`)
	fmt.Println("  -> risk_level is \"low\" (not RISK_LEVEL_LOW)")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
