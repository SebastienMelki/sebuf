//go:build ignore

// This example demonstrates per-call options for customizing individual requests.
// Run with: go run client_per_call_options.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SebastienMelki/sebuf/examples/restful-crud/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/restful-crud/api/proto/services"
)

func main() {
	// Create base client with default configuration
	client := services.NewProductServiceClient(
		"http://localhost:8080",
		services.WithProductServiceAPIKey("123e4567-e89b-12d3-a456-426614174000"),
		services.WithProductServiceHTTPClient(&http.Client{
			Timeout: 30 * time.Second,
		}),
	)

	ctx := context.Background()

	fmt.Println("=== Per-Call Options Examples ===\n")

	// Example 1: Adding custom headers per request
	fmt.Println("1. Custom Headers Per Request")
	fmt.Println("   Adding X-Request-ID and X-Correlation-ID for tracing...")

	requestID := "req-12345-abcde"
	correlationID := "corr-67890-fghij"

	product, err := client.CreateProduct(ctx, &models.CreateProductRequest{
		Name:        "Traced Product",
		Description: "Product with tracing headers",
		Price:       49.99,
		CategoryId:  "traced",
	},
		services.WithProductServiceHeader("X-Request-ID", requestID),
		services.WithProductServiceHeader("X-Correlation-ID", correlationID),
	)

	if err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Created product with tracing: %s\n", product.Name)
		fmt.Printf("   Request-ID: %s, Correlation-ID: %s\n", requestID, correlationID)
	}

	// Example 2: Using generated header helpers
	fmt.Println("\n2. Generated Header Helpers")
	fmt.Println("   Using WithProductServiceCallConfirmDelete for delete operation...")

	if product != nil {
		_, err = client.DeleteProduct(ctx, &models.DeleteProductRequest{
			ProductId: product.Id,
		},
			// Generated helper for X-Confirm-Delete header
			services.WithProductServiceCallConfirmDelete("true"),
		)

		if err != nil {
			log.Printf("   Error: %v\n", err)
		} else {
			fmt.Printf("   Successfully deleted product: %s\n", product.Id)
		}
	}

	// Example 3: Overriding content type per request
	fmt.Println("\n3. Content Type Override Per Request")
	fmt.Println("   Switching from JSON to Protobuf for bulk operation...")

	// List with Protobuf for better performance on large responses
	list, err := client.ListProducts(ctx, &models.ListProductsRequest{
		Page:  1,
		Limit: 100,
	},
		services.WithProductServiceCallContentType(services.ContentTypeProto),
	)

	if err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Retrieved %d products using Protobuf\n", len(list.Products))
	}

	// Example 4: Combining multiple per-call options
	fmt.Println("\n4. Combining Multiple Options")
	fmt.Println("   Adding tracing headers + content type override...")

	product, err = client.CreateProduct(ctx, &models.CreateProductRequest{
		Name:        "Multi-Option Product",
		Description: "Created with multiple call options",
		Price:       79.99,
		CategoryId:  "multi",
		Tags:        []string{"traced", "protobuf"},
	},
		// Tracing headers
		services.WithProductServiceHeader("X-Request-ID", "multi-req-001"),
		services.WithProductServiceHeader("X-Trace-ID", "trace-abc-123"),
		// Content type override
		services.WithProductServiceCallContentType(services.ContentTypeProto),
		// Custom business header
		services.WithProductServiceHeader("X-Priority", "high"),
	)

	if err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Created product with multiple options: %s\n", product.Name)
	}

	// Example 5: Dynamic headers based on context
	fmt.Println("\n5. Dynamic Headers Based on Context")
	demonstrateDynamicHeaders(ctx, client)

	// Example 6: A/B testing with headers
	fmt.Println("\n6. A/B Testing with Custom Headers")
	demonstrateABTesting(ctx, client)

	fmt.Println("\n=== Per-Call Options Examples Complete ===")
}

// demonstrateDynamicHeaders shows how to add headers based on runtime conditions
func demonstrateDynamicHeaders(ctx context.Context, client services.ProductServiceClient) {
	// Simulating different user contexts
	users := []struct {
		userID   string
		tenantID string
		role     string
	}{
		{"user-001", "tenant-a", "admin"},
		{"user-002", "tenant-b", "viewer"},
		{"user-003", "tenant-a", "editor"},
	}

	for _, user := range users {
		fmt.Printf("   Fetching products for user %s (tenant: %s, role: %s)...\n",
			user.userID, user.tenantID, user.role)

		_, err := client.ListProducts(ctx, &models.ListProductsRequest{
			Page:  1,
			Limit: 10,
		},
			// Dynamic headers based on user context
			services.WithProductServiceHeader("X-User-ID", user.userID),
			services.WithProductServiceHeader("X-Tenant-ID", user.tenantID),
			services.WithProductServiceHeader("X-User-Role", user.role),
		)

		if err != nil {
			log.Printf("      Error: %v\n", err)
		} else {
			fmt.Println("      Success!")
		}
	}
}

// demonstrateABTesting shows how to use headers for A/B testing
func demonstrateABTesting(ctx context.Context, client services.ProductServiceClient) {
	variants := []string{"control", "variant-a", "variant-b"}

	for _, variant := range variants {
		fmt.Printf("   Testing variant: %s\n", variant)

		_, err := client.ListProducts(ctx, &models.ListProductsRequest{
			Page:  1,
			Limit: 5,
		},
			services.WithProductServiceHeader("X-Experiment-Variant", variant),
			services.WithProductServiceHeader("X-Experiment-Name", "product-list-redesign"),
		)

		if err != nil {
			log.Printf("      Error: %v\n", err)
		} else {
			fmt.Printf("      Variant %s: request completed\n", variant)
		}
	}
}
