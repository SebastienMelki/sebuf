package main

import (
	"context"
	"testing"

	"github.com/SebastienMelki/sebuf/examples/simple-api/api"
)

func TestWhatIfScenarios(t *testing.T) {
	ctx := context.Background()

	t.Run("database down scenario", func(t *testing.T) {
		// Create mock with database down scenario
		mock := api.NewWhatIfUserServiceServer(
			api.WhatIf.DatabaseDown(),
		)

		// All methods should fail
		_, err := mock.CreateUser(ctx, &api.CreateUserRequest{
			Name:  "Alice",
			Email: "alice@example.com",
		})
		if err == nil {
			t.Fatal("expected error for database down scenario")
		}
		if err.Error() != "Service temporarily unavailable" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("login error scenario", func(t *testing.T) {
		// Create mock with login-specific error
		mock := api.NewWhatIfUserServiceServer(
			api.WhatIf.LoginError(),
		)

		// Login should fail
		_, err := mock.Login(ctx, &api.LoginRequest{})
		if err == nil {
			t.Fatal("expected error for login error scenario")
		}

		// But other methods should work
		user, err := mock.GetUser(ctx, &api.GetUserRequest{Id: "123"})
		if err != nil {
			t.Fatalf("GetUser should work: %v", err)
		}
		if user == nil {
			t.Fatal("expected user response")
		}
	})

	t.Run("multiple scenarios", func(t *testing.T) {
		// Combine multiple scenarios
		mock := api.NewWhatIfUserServiceServer(
			api.WhatIf.SlowResponse(),
			api.WhatIf.GetUserError(),
		)

		// GetUser should fail due to specific scenario
		_, err := mock.GetUser(ctx, &api.GetUserRequest{Id: "123"})
		if err == nil {
			t.Fatal("expected error for GetUser")
		}

		// Login should work but be slow (in real implementation)
		resp, err := mock.Login(ctx, &api.LoginRequest{})
		if err != nil {
			t.Fatalf("Login should work: %v", err)
		}
		if resp == nil {
			t.Fatal("expected login response")
		}
	})

	t.Run("default behavior without scenarios", func(t *testing.T) {
		// No scenarios - everything should work with default responses
		mock := api.NewWhatIfUserServiceServer()

		user, err := mock.CreateUser(ctx, &api.CreateUserRequest{
			Name:  "Bob",
			Email: "bob@example.com",
		})
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		if user == nil {
			t.Fatal("expected user response")
		}
	})
}