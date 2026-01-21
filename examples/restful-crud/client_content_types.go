//go:build ignore

// This example demonstrates content type switching between JSON and binary protobuf.
// Run with: go run client_content_types.go
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
	ctx := context.Background()

	fmt.Println("=== Content Type Examples ===\n")

	// Example 1: Default JSON client
	fmt.Println("1. JSON Client (Default)")
	jsonClient := services.NewProductServiceClient(
		"http://localhost:8080",
		services.WithProductServiceAPIKey("123e4567-e89b-12d3-a456-426614174000"),
		// JSON is the default, no need to specify
	)

	product, err := createTestProduct(ctx, jsonClient, "JSON Product")
	if err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Created product via JSON: %s (ID: %s)\n", product.Name, product.Id)
	}

	// Example 2: Binary protobuf client (better performance for large payloads)
	fmt.Println("\n2. Binary Protobuf Client (Better Performance)")
	protoClient := services.NewProductServiceClient(
		"http://localhost:8080",
		services.WithProductServiceAPIKey("123e4567-e89b-12d3-a456-426614174000"),
		services.WithProductServiceContentType(services.ContentTypeProto),
	)

	product, err = createTestProduct(ctx, protoClient, "Protobuf Product")
	if err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Created product via Protobuf: %s (ID: %s)\n", product.Name, product.Id)
	}

	// Example 3: Per-request content type override
	fmt.Println("\n3. Per-Request Content Type Override")
	fmt.Println("   Using JSON client but overriding to Protobuf for one request...")

	// Client defaults to JSON
	mixedClient := services.NewProductServiceClient(
		"http://localhost:8080",
		services.WithProductServiceAPIKey("123e4567-e89b-12d3-a456-426614174000"),
		// Default: JSON
	)

	// But this specific request uses Protobuf
	product, err = mixedClient.CreateProduct(ctx, &models.CreateProductRequest{
		Name:        "Mixed Content Product",
		Description: "Created with Protobuf override",
		Price:       39.99,
		CategoryId:  "mixed",
	},
		services.WithProductServiceCallContentType(services.ContentTypeProto),
	)

	if err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Created product with Protobuf override: %s\n", product.Name)
	}

	// Example 4: Performance comparison
	fmt.Println("\n4. Performance Comparison (JSON vs Protobuf)")
	runPerformanceComparison(ctx, jsonClient, protoClient)

	// Example 5: When to use each content type
	fmt.Println("\n5. Content Type Guidelines")
	fmt.Println("   JSON (application/json):")
	fmt.Println("      - Human-readable, easy to debug")
	fmt.Println("      - Better for browser/JavaScript clients")
	fmt.Println("      - Larger payload size")
	fmt.Println("      - Default choice for most APIs")
	fmt.Println("")
	fmt.Println("   Protobuf (application/x-protobuf):")
	fmt.Println("      - Smaller payload size (30-50% smaller)")
	fmt.Println("      - Faster serialization/deserialization")
	fmt.Println("      - Better for high-throughput services")
	fmt.Println("      - Better for large payloads or batch operations")
	fmt.Println("      - Requires protobuf-aware clients")

	fmt.Println("\n=== Content Type Examples Complete ===")
}

func createTestProduct(ctx context.Context, client services.ProductServiceClient, name string) (*models.Product, error) {
	return client.CreateProduct(ctx, &models.CreateProductRequest{
		Name:        name,
		Description: "Test product for content type demo",
		Price:       29.99,
		CategoryId:  "test",
		Tags:        []string{"demo", "test"},
	})
}

func runPerformanceComparison(ctx context.Context, jsonClient, protoClient services.ProductServiceClient) {
	iterations := 10

	// Measure JSON performance
	jsonStart := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := jsonClient.ListProducts(ctx, &models.ListProductsRequest{
			Page:  1,
			Limit: 50,
		})
		if err != nil {
			log.Printf("   JSON request %d failed: %v", i, err)
			return
		}
	}
	jsonDuration := time.Since(jsonStart)

	// Measure Protobuf performance
	protoStart := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := protoClient.ListProducts(ctx, &models.ListProductsRequest{
			Page:  1,
			Limit: 50,
		})
		if err != nil {
			log.Printf("   Protobuf request %d failed: %v", i, err)
			return
		}
	}
	protoDuration := time.Since(protoStart)

	fmt.Printf("   JSON:     %d requests in %v (avg: %v/req)\n",
		iterations, jsonDuration, jsonDuration/time.Duration(iterations))
	fmt.Printf("   Protobuf: %d requests in %v (avg: %v/req)\n",
		iterations, protoDuration, protoDuration/time.Duration(iterations))

	if protoDuration < jsonDuration {
		improvement := float64(jsonDuration-protoDuration) / float64(jsonDuration) * 100
		fmt.Printf("   Protobuf is %.1f%% faster\n", improvement)
	} else {
		fmt.Println("   (Results may vary based on payload size and network conditions)")
	}
}
