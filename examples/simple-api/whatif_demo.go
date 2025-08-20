package main

import (
	"context"
	"fmt"

	"github.com/SebastienMelki/sebuf/examples/simple-api/api"
)

// DemoWhatIfScenarios demonstrates the power of WhatIf scenarios for testing
func DemoWhatIfScenarios() {
	ctx := context.Background()

	fmt.Println("🧪 WhatIf Scenario Testing Demo")
	fmt.Println("================================")
	fmt.Println()

	// 1. Test database failure scenario
	fmt.Println("1️⃣ Testing: What if the database is down?")
	mockDB := api.NewWhatIfUserServiceServer(
		api.WhatIf.DatabaseDown(),
	)

	_, err := mockDB.CreateUser(ctx, &api.CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
	})
	if err != nil {
		fmt.Printf("   ✅ Expected error: %v\n", err)
	}
	fmt.Println()

	// 2. Test login-specific failure
	fmt.Println("2️⃣ Testing: What if login fails but other methods work?")
	mockLogin := api.NewWhatIfUserServiceServer(
		api.WhatIf.LoginError(),
	)

	// Login should fail
	_, err = mockLogin.Login(ctx, &api.LoginRequest{})
	if err != nil {
		fmt.Printf("   ✅ Login failed as expected: %v\n", err)
	}

	// But CreateUser should work
	user, err := mockLogin.CreateUser(ctx, &api.CreateUserRequest{
		Name:  "Bob",
		Email: "bob@example.com",
	})
	if err == nil && user != nil {
		fmt.Printf("   ✅ CreateUser worked as expected\n")
	}
	fmt.Println()

	// 3. Test combined scenarios
	fmt.Println("3️⃣ Testing: What if the service is slow AND GetUser fails?")
	mockCombined := api.NewWhatIfUserServiceServer(
		api.WhatIf.SlowResponse(),
		api.WhatIf.GetUserError(),
	)

	// GetUser should fail
	_, err = mockCombined.GetUser(ctx, &api.GetUserRequest{Id: "123"})
	if err != nil {
		fmt.Printf("   ✅ GetUser failed: %v\n", err)
	}

	// Login should work (but would be slow in real implementation)
	loginResp, err := mockCombined.Login(ctx, &api.LoginRequest{})
	if err == nil && loginResp != nil {
		fmt.Printf("   ✅ Login worked (with simulated slowness)\n")
	}
	fmt.Println()

	// 4. Default behavior
	fmt.Println("4️⃣ Testing: Default behavior without scenarios")
	mockDefault := api.NewWhatIfUserServiceServer()

	user, err = mockDefault.CreateUser(ctx, &api.CreateUserRequest{
		Name:  "Charlie",
		Email: "charlie@example.com",
	})
	if err == nil && user != nil {
		fmt.Printf("   ✅ All methods work by default\n")
	}
	fmt.Println()

	fmt.Println("🎯 Future with LLM Integration:")
	fmt.Println("================================")
	fmt.Println("When you generate with an OpenRouter API key:")
	fmt.Println()
	fmt.Println("  protoc-gen-go-whatif --openrouter_api_key=$KEY")
	fmt.Println()
	fmt.Println("The LLM will analyze your proto and generate scenarios like:")
	fmt.Println()
	fmt.Println("  📍 For Login method:")
	fmt.Println("     - api.WhatIf.Login.ExpiredToken()")
	fmt.Println("     - api.WhatIf.Login.InvalidCredentials()")
	fmt.Println("     - api.WhatIf.Login.AccountLocked()")
	fmt.Println("     - api.WhatIf.Login.Requires2FA()")
	fmt.Println("     - api.WhatIf.Login.PasswordExpired()")
	fmt.Println()
	fmt.Println("  📍 For GetUser method:")
	fmt.Println("     - api.WhatIf.GetUser.NotFound()")
	fmt.Println("     - api.WhatIf.GetUser.DeletedAccount()")
	fmt.Println("     - api.WhatIf.GetUser.NoPermission()")
	fmt.Println()
	fmt.Println("  📍 Service-wide scenarios:")
	fmt.Println("     - api.WhatIf.RateLimited()")
	fmt.Println("     - api.WhatIf.MaintenanceMode()")
	fmt.Println("     - api.WhatIf.HighLatency()")
	fmt.Println()
	fmt.Println("✨ Making comprehensive testing delightful!")
}

func main() {
	DemoWhatIfScenarios()
}
