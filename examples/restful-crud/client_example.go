//go:build ignore

// This file demonstrates how to use the generated HTTP client.
// Run with: go run client_example.go
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
	// Create a client with custom options
	client := services.NewProductServiceClient(
		"http://localhost:8080",
		// Set API key for all requests
		services.WithProductServiceAPIKey("123e4567-e89b-12d3-a456-426614174000"),
		// Use a custom HTTP client with timeout
		services.WithProductServiceHTTPClient(&http.Client{
			Timeout: 30 * time.Second,
		}),
	)

	ctx := context.Background()

	// Example 1: Create a product
	fmt.Println("Creating a product...")
	product, err := client.CreateProduct(ctx, &models.CreateProductRequest{
		Name:        "Example Widget",
		Description: "A high-quality widget",
		Price:       19.99,
		Category:    "widgets",
		Tags:        []string{"new", "featured"},
	})
	if err != nil {
		log.Fatalf("Failed to create product: %v", err)
	}
	fmt.Printf("Created product: %s (ID: %s)\n", product.Name, product.Id)

	// Example 2: Get a product
	fmt.Println("\nGetting the product...")
	retrieved, err := client.GetProduct(ctx, &models.GetProductRequest{
		ProductId: product.Id,
	})
	if err != nil {
		log.Fatalf("Failed to get product: %v", err)
	}
	fmt.Printf("Retrieved product: %s - $%.2f\n", retrieved.Name, retrieved.Price)

	// Example 3: List products with pagination and filtering
	fmt.Println("\nListing products...")
	list, err := client.ListProducts(ctx, &models.ListProductsRequest{
		Page:     1,
		Limit:    10,
		Category: "widgets",
		SortBy:   "created_at",
	})
	if err != nil {
		log.Fatalf("Failed to list products: %v", err)
	}
	fmt.Printf("Found %d products (page %d of %d)\n",
		len(list.Products), list.Page, list.TotalPages)

	// Example 4: Update a product (full update)
	fmt.Println("\nUpdating product...")
	updated, err := client.UpdateProduct(ctx, &models.UpdateProductRequest{
		ProductId:   product.Id,
		Name:        "Updated Widget",
		Description: "An even better widget",
		Price:       24.99,
		Category:    "widgets",
		Tags:        []string{"updated", "premium"},
	})
	if err != nil {
		log.Fatalf("Failed to update product: %v", err)
	}
	fmt.Printf("Updated product: %s - $%.2f\n", updated.Name, updated.Price)

	// Example 5: Patch a product (partial update)
	fmt.Println("\nPatching product price...")
	patched, err := client.PatchProduct(ctx, &models.PatchProductRequest{
		ProductId: product.Id,
		Price:     &models.PatchProductRequest_PriceValue{PriceValue: 29.99},
	})
	if err != nil {
		log.Fatalf("Failed to patch product: %v", err)
	}
	fmt.Printf("Patched product price: $%.2f\n", patched.Price)

	// Example 6: Delete a product (with confirmation header)
	fmt.Println("\nDeleting product...")
	_, err = client.DeleteProduct(ctx, &models.DeleteProductRequest{
		ProductId: product.Id,
	},
		// Add required confirmation header for this specific call
		services.WithProductServiceCallConfirmDelete("true"),
	)
	if err != nil {
		log.Fatalf("Failed to delete product: %v", err)
	}
	fmt.Println("Product deleted successfully")

	// Example 7: Using per-request options
	fmt.Println("\nDemonstrating per-request options...")
	_, err = client.ListProducts(ctx, &models.ListProductsRequest{
		Page:  1,
		Limit: 5,
	},
		// Override content type for this request only
		services.WithProductServiceCallContentType(services.ContentTypeProto),
		// Add a custom header for this request only
		services.WithProductServiceHeader("X-Custom-Header", "custom-value"),
	)
	if err != nil {
		log.Printf("Note: Request may fail if server isn't running: %v", err)
	}

	fmt.Println("\nClient example completed!")
}
