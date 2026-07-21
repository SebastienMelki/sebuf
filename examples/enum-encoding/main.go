// Package main demonstrates enum_value JSON encoding on the generated Go HTTP
// server, including transitive nesting: the RPC returns a wrapper message whose
// custom enums (RiskLevel, and OptionType two levels down) are nested below the
// message the server marshals. The server emits the custom strings
// ("low"/"call") at every depth and accepts them in requests, matching the
// OpenAPI docs and the TypeScript/Python clients.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	api "github.com/SebastienMelki/sebuf/examples/enum-encoding/api/proto/services"
)

// suggestionHandler returns curated suggestions. It echoes the request's
// requested_risk into the response, proving the server parsed the custom enum
// string ("low") on the way in as well as emitting it on the way out.
type suggestionHandler struct{}

func (suggestionHandler) GetEasyOptions(
	_ context.Context,
	req *api.GetEasyOptionsRequest,
) (*api.GetEasyOptionsResponse, error) {
	risk := req.GetRequestedRisk()
	if risk == api.RiskLevel_RISK_LEVEL_UNSPECIFIED {
		risk = api.RiskLevel_RISK_LEVEL_LOW
	}
	return &api.GetEasyOptionsResponse{
		// Direct (unnested) enums on the marshaled message.
		OverallRisk:     risk,                            // annotated  -> "low"
		MarketSentiment: api.Sentiment_SENTIMENT_BULLISH, // unannotated -> "SENTIMENT_BULLISH"
		Data: []*api.EasyOptionSuggestion{
			{
				Contract: &api.OptionsContract{
					Symbol:      req.GetUnderlyingSymbol(),
					Type:        api.OptionType_OPTION_TYPE_CALL, // annotated, nested   -> "call"
					Sentiment:   api.Sentiment_SENTIMENT_BULLISH, // unannotated, nested -> "SENTIMENT_BULLISH"
					StrikePrice: 330,
				},
				ProbabilityOfProfit: 0.62,
				RiskLevel:           risk, // annotated, nested -> "low"
			},
			{
				Contract: &api.OptionsContract{
					Symbol:      req.GetUnderlyingSymbol(),
					Type:        api.OptionType_OPTION_TYPE_PUT,
					Sentiment:   api.Sentiment_SENTIMENT_BEARISH,
					StrikePrice: 300,
				},
				ProbabilityOfProfit: 0.41,
				RiskLevel:           api.RiskLevel_RISK_LEVEL_HIGH,
			},
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
	fmt.Println(`  curl -s localhost:8080/api/v1/suggestions -d '{"underlyingSymbol":"AAPL","requestedRisk":"low"}'`)
	fmt.Println(`  -> data[].risk_level is "low"/"high" and data[].contract.type is "call"/"put"`)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
