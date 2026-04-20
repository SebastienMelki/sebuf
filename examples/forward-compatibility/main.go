// This program demonstrates sebuf's forward compatibility options.
//
// sebuf-generated clients provide WithXxxDiscardUnknownFields(true) as a
// client option and WithXxxCallDiscardUnknownFields(true) as a per-call
// option to silently discard unknown fields in JSON responses.
//
// Run: go run ./examples/forward-compatibility/
//
// Usage in generated clients:
//
//	// Service-level: all RPCs discard unknown fields
//	client := services.NewMarketDataServiceClient(baseURL,
//	    services.WithMarketDataServiceDiscardUnknownFields(true),
//	)
//
//	// Per-call: override for a single RPC
//	resp, err := client.GetQuote(ctx, req,
//	    services.WithMarketDataServiceCallDiscardUnknownFields(true),
//	)
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	// Start a mock API server that returns extra fields not in our proto schema.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"name":      "user.proto",
			"package":   "api.v1",
			"swap_rate": 0.015,                       // new field — not in proto
			"region":    "us-east-1",                  // new field — not in proto
			"metadata":  map[string]any{"version": 3}, // new field — not in proto
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	body := fetchBody(server.URL)

	fmt.Println("=== sebuf Forward Compatibility Example ===")
	fmt.Println()
	fmt.Printf("API response (contains unknown fields 'swap_rate', 'region', 'metadata'):\n  %s\n", body)

	// --- Default (strict) ---
	fmt.Println("1) Default (strict) — no DiscardUnknownFields option:")
	msg1 := &descriptorpb.FileDescriptorProto{}
	err := unmarshal(body, msg1, false)
	if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
		fmt.Println("   This is correct — strict mode rejects unknown fields.")
	} else {
		fmt.Printf("   OK: name=%q\n", msg1.GetName())
	}
	fmt.Println()

	// --- With DiscardUnknownFields(true) ---
	fmt.Println("2) WithXxxDiscardUnknownFields(true) — forward compatible:")
	msg2 := &descriptorpb.FileDescriptorProto{}
	err = unmarshal(body, msg2, true)
	if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
	} else {
		fmt.Printf("   OK: name=%q package=%q\n", msg2.GetName(), msg2.GetPackage())
		fmt.Println("   Unknown fields silently discarded — client keeps working.")
	}
	fmt.Println()

	fmt.Println("=== Usage ===")
	fmt.Println()
	fmt.Println("  // Service-level default (all RPCs)")
	fmt.Println("  client := services.NewMarketDataServiceClient(baseURL,")
	fmt.Println("      services.WithMarketDataServiceDiscardUnknownFields(true),")
	fmt.Println("  )")
	fmt.Println()
	fmt.Println("  // Per-call override (single RPC)")
	fmt.Println("  resp, err := client.GetQuote(ctx, req,")
	fmt.Println("      services.WithMarketDataServiceCallDiscardUnknownFields(true),")
	fmt.Println("  )")
}

// unmarshal mirrors the generated unmarshalResponse method.
// discardUnknown is resolved from client default + per-call override.
func unmarshal(body []byte, msg proto.Message, discardUnknown bool) error {
	if discardUnknown {
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		return opts.Unmarshal(body, msg)
	}
	return protojson.Unmarshal(body, msg)
}

func fetchBody(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return body
}
