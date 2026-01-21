//go:build ignore

// This example demonstrates error handling patterns with the generated HTTP client.
// Run with: go run client_error_handling.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	sebufhttp "github.com/SebastienMelki/sebuf/http"

	"github.com/SebastienMelki/sebuf/examples/restful-crud/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/restful-crud/api/proto/services"
)

func main() {
	// Create client
	client := services.NewProductServiceClient(
		"http://localhost:8080",
		services.WithProductServiceAPIKey("123e4567-e89b-12d3-a456-426614174000"),
		services.WithProductServiceHTTPClient(&http.Client{
			Timeout: 10 * time.Second,
		}),
	)

	ctx := context.Background()

	fmt.Println("=== Error Handling Examples ===\n")

	// Example 1: Handling validation errors (HTTP 400)
	fmt.Println("1. Validation Error Handling")
	fmt.Println("   Sending invalid request (empty name, negative price)...")

	_, err := client.CreateProduct(ctx, &models.CreateProductRequest{
		Name:  "", // Invalid: name is required
		Price: -10, // Invalid: price must be positive
	})

	if err != nil {
		handleError(err)
	}

	// Example 2: Handling not found errors (HTTP 404)
	fmt.Println("\n2. Not Found Error Handling")
	fmt.Println("   Requesting non-existent product...")

	_, err = client.GetProduct(ctx, &models.GetProductRequest{
		ProductId: "non-existent-product-id",
	})

	if err != nil {
		handleError(err)
	}

	// Example 3: Handling missing required header
	fmt.Println("\n3. Missing Header Error Handling")
	fmt.Println("   Attempting delete without X-Confirm-Delete header...")

	// First create a product to delete
	product, err := client.CreateProduct(ctx, &models.CreateProductRequest{
		Name:       "Test Product",
		Price:      10.00,
		CategoryId: "test",
	})
	if err != nil {
		log.Printf("   Could not create test product: %v", err)
	} else {
		// Try to delete without confirmation header
		_, err = client.DeleteProduct(ctx, &models.DeleteProductRequest{
			ProductId: product.Id,
		})
		// Note: X-Confirm-Delete header is missing

		if err != nil {
			handleError(err)
		}
	}

	// Example 4: Handling network errors
	fmt.Println("\n4. Network Error Handling")
	fmt.Println("   Connecting to non-existent server...")

	badClient := services.NewProductServiceClient(
		"http://localhost:99999", // Invalid port
		services.WithProductServiceHTTPClient(&http.Client{
			Timeout: 2 * time.Second,
		}),
	)

	_, err = badClient.ListProducts(ctx, &models.ListProductsRequest{
		Page:  1,
		Limit: 10,
	})

	if err != nil {
		handleError(err)
	}

	// Example 5: Handling context cancellation
	fmt.Println("\n5. Context Cancellation Handling")
	fmt.Println("   Cancelling request mid-flight...")

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.ListProducts(cancelCtx, &models.ListProductsRequest{
		Page:  1,
		Limit: 10,
	})

	if err != nil {
		handleError(err)
	}

	fmt.Println("\n=== Error Handling Examples Complete ===")
}

// handleError demonstrates comprehensive error handling
func handleError(err error) {
	// Check for validation errors (HTTP 400)
	var validationErr *sebufhttp.ValidationError
	if errors.As(err, &validationErr) {
		fmt.Println("   [ValidationError] Request validation failed:")
		for _, violation := range validationErr.GetViolations() {
			fmt.Printf("      - Field '%s': %s\n", violation.GetField(), violation.GetDescription())
		}
		return
	}

	// Check for generic API errors
	var genericErr *sebufhttp.Error
	if errors.As(err, &genericErr) {
		fmt.Printf("   [APIError] %s\n", genericErr.GetMessage())
		return
	}

	// Check for context cancellation
	if errors.Is(err, context.Canceled) {
		fmt.Println("   [Canceled] Request was cancelled")
		return
	}

	// Check for context deadline exceeded
	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("   [Timeout] Request timed out")
		return
	}

	// Check for network errors
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		fmt.Printf("   [NetworkError] %s\n", netErr.Error())
		return
	}

	// Fallback for unknown errors
	fmt.Printf("   [UnknownError] %v\n", err)
}
