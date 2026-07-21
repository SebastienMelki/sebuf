package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/SebastienMelki/sebuf/examples/enum-encoding/api/proto/services"
)

// TestEnumValueServerEncoding is the end-to-end proof for the enum_value fix: the
// generated Go server emits the custom enum strings ("low"/"medium"/"high") in
// JSON responses and accepts them in JSON requests, rather than the raw proto
// value names (RISK_LEVEL_LOW).
func TestEnumValueServerEncoding(t *testing.T) {
	mux := http.NewServeMux()
	if err := api.RegisterSuggestionServiceServer(suggestionHandler{}, api.WithMux(mux)); err != nil {
		t.Fatalf("register server: %v", err)
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Send the custom enum string "low" in the request body — proves request parsing.
	reqBody := `{"symbol":"AAPL","requestedRisk":"low"}`
	resp, err := http.Post(srv.URL+"/api/v1/suggestion", "application/json", strings.NewReader(reqBody))
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
	body := string(raw)

	// The raw proto names must NOT appear anywhere in the response.
	if strings.Contains(body, "RISK_LEVEL_") {
		t.Errorf("response leaked raw proto enum names:\n%s", body)
	}

	// Singular, echoed-from-request, repeated, and map enum values must all use custom strings.
	wants := []string{
		`"riskLevel":"low"`, // singular (echoed request value -> proves both directions)
		`"medium"`,          // repeated element
		`"high"`,            // repeated element + map value
	}
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("response missing %q\nbody: %s", w, body)
		}
	}
}
