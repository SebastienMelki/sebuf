package main

import (
	"context"
	"fmt"
	"github.com/SebastienMelki/sebuf/test/gen"
	"log"
	"net/http"
)

type TestServer struct{}

func (s *TestServer) Echo(ctx context.Context, req *gen.EchoRequest) (*gen.EchoResponse, error) {
	return &gen.EchoResponse{
		Message: fmt.Sprintf("Echo: POOPOO %s", req.Message),
	}, nil
}

func main() {
	server := &TestServer{}

	err := gen.RegisterTestServiceServer(server)
	if err != nil {
		log.Fatalf("Failed to register server: %v", err)
	}

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
