// Package main demonstrates WithMarshalOptions by running two HTTP servers
// side by side: one with default protojson behavior, one with
// EmitUnpopulated: true. The same handler returns a Status with proto3
// zero values; only the second server surfaces them in the response.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"

	api "github.com/SebastienMelki/sebuf/examples/marshal-options/api/proto/services"
)

// statusHandler returns the same proto3-zero-value Status to every caller.
// Whether those zero values appear in the JSON response is controlled
// entirely by the server's MarshalOptions.
type statusHandler struct{}

func (statusHandler) GetStatus(
	_ context.Context,
	_ *api.GetStatusRequest,
) (*api.Status, error) {
	return &api.Status{
		AcceptingOrders: false, // proto3 zero — semantically meaningful
		SubscriberCount: 0,     // proto3 zero — real "0 subscribers" state
		Note:            "",    // proto3 zero — operator left it blank
	}, nil
}

func main() {
	handler := statusHandler{}

	defaultMux := http.NewServeMux()
	if err := api.RegisterOfferingServiceServer(
		handler,
		api.WithMux(defaultMux),
	); err != nil {
		log.Fatalf("register default server: %v", err)
	}

	emitMux := http.NewServeMux()
	if err := api.RegisterOfferingServiceServer(
		handler,
		api.WithMux(emitMux),
		api.WithMarshalOptions(protojson.MarshalOptions{
			EmitUnpopulated: true,
		}),
	); err != nil {
		log.Fatalf("register emit-unpopulated server: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		fmt.Println("[default opts]          listening on :8080")
		fmt.Println("  curl http://localhost:8080/api/v1/status")
		if err := http.ListenAndServe(":8080", defaultMux); err != nil {
			log.Fatalf("default server: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		fmt.Println("[EmitUnpopulated: true] listening on :8081")
		fmt.Println("  curl http://localhost:8081/api/v1/status")
		if err := http.ListenAndServe(":8081", emitMux); err != nil {
			log.Fatalf("emit-unpopulated server: %v", err)
		}
	}()

	wg.Wait()
}
