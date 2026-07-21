package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/SebastienMelki/sebuf/examples/enum-encoding/api/proto/services"
)

func post(t *testing.T, opts ...api.ServerOption) string {
	t.Helper()
	mux := http.NewServeMux()
	opts = append([]api.ServerOption{api.WithMux(mux)}, opts...)
	if err := api.RegisterSuggestionServiceServer(suggestionHandler{}, opts...); err != nil {
		t.Fatalf("register server: %v", err)
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(
		srv.URL+"/api/v1/suggestions", "application/json",
		strings.NewReader(`{"underlyingSymbol":"AAPL","requestedRisk":"low"}`),
	)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (request with \"low\" should parse)", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(raw)
}

// TestEnumValueFullMatrix is the end-to-end proof across the full matrix: annotated vs
// unannotated enums, each appearing directly on the marshaled message and nested below it.
// Annotated enums serialize as their custom strings at every depth; unannotated enums keep
// their proto names. The request's "low" must also be accepted (request parsing).
func TestEnumValueFullMatrix(t *testing.T) {
	body := post(t)

	// Annotated enums -> custom strings, at every depth.
	for _, w := range []string{
		`"overallRisk":"low"`, // annotated, DIRECT on the marshaled message
		`"riskLevel":"low"`,   // annotated, nested one level (echoed from request)
		`"riskLevel":"high"`,  // annotated, nested (second element)
		`"type":"call"`,       // annotated, nested two levels
		`"type":"put"`,        // annotated, nested two levels (second element)
	} {
		if !strings.Contains(body, w) {
			t.Errorf("response missing annotated custom value %q\nbody: %s", w, body)
		}
	}

	// No annotated enum should leak its proto name.
	if strings.Contains(body, "RISK_LEVEL_") || strings.Contains(body, "OPTION_TYPE_") {
		t.Errorf("response leaked raw proto enum names for annotated enums:\n%s", body)
	}

	// Unannotated enum (Sentiment) keeps its proto name, both direct and nested.
	for _, w := range []string{
		`"marketSentiment":"SENTIMENT_BULLISH"`, // unannotated, DIRECT
		`"sentiment":"SENTIMENT_BULLISH"`,       // unannotated, nested
	} {
		if !strings.Contains(body, w) {
			t.Errorf("response missing unannotated proto name %q\nbody: %s", w, body)
		}
	}
}

// TestEnumValueWithUseProtoNames verifies the fix holds under protojson UseProtoNames, which emits
// snake_case field keys, at every nesting depth. The nested container fields are multi-word
// (option_suggestions, options_contract) so the marshaler must patch the snake_case key protojson
// emitted — not a camelCase key, which would both leak the raw enum and add a duplicate field.
func TestEnumValueWithUseProtoNames(t *testing.T) {
	body := post(t, api.WithMarshalOptions(protojson.MarshalOptions{UseProtoNames: true}))

	if strings.Contains(body, "RISK_LEVEL_") || strings.Contains(body, "OPTION_TYPE_") {
		t.Errorf("UseProtoNames response leaked raw proto enum names:\n%s", body)
	}
	// Custom strings present under snake_case, at every depth.
	for _, w := range []string{`"overall_risk":"low"`, `"risk_level":"low"`, `"type":"call"`} {
		if !strings.Contains(body, w) {
			t.Errorf("UseProtoNames response missing %q\nbody: %s", w, body)
		}
	}
	// No duplicate camelCase keys for the multi-word nested containers.
	for _, dup := range []string{`"optionSuggestions"`, `"optionsContract"`} {
		if strings.Contains(body, dup) {
			t.Errorf("UseProtoNames response has a duplicate camelCase key %s\nbody: %s", dup, body)
		}
	}
}
