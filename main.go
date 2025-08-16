package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/SebastienMelki/sebuf/authv1"
	"net/http"
)

type SomeImplementation struct{}

func (s *SomeImplementation) Login(_ context.Context, _ *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	// Simulate a successful login
	return &authv1.LoginResponse{
		Token: "some-generated-token",
	}, nil
}

func main() {
	err := authv1.RegisterAuthServiceServer(&SomeImplementation{})
	if err != nil {
		fmt.Printf("failed to register auth service server: %v\n", err)
		return
	}

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", "8080"),
	}

	go func() {
		// Signal that we are ready to receive traffic
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("failed to start server: %v\n", err)
			return
		}
	}()

	select {}
}
