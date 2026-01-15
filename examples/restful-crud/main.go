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

// ProductService implements the ProductServiceServer interface.
type ProductService struct {
	products map[string]*models.Product
	nextID   int
}

// NewProductService creates a new ProductService with sample data.
func NewProductService() *ProductService {
	svc := &ProductService{
		products: make(map[string]*models.Product),
		nextID:   1,
	}

	// Add some sample products
	sampleProducts := []*models.Product{
		{
			Id:            "123e4567-e89b-12d3-a456-426614174001",
			Name:          "Wireless Bluetooth Headphones",
			Description:   "High-quality wireless headphones with active noise cancellation",
			Price:         99.99,
			StockQuantity: 150,
			CategoryId:    "cat-electronics",
			Tags:          []string{"audio", "wireless", "bluetooth"},
			CreatedAt:     time.Now().Unix(),
			UpdatedAt:     time.Now().Unix(),
		},
		{
			Id:            "123e4567-e89b-12d3-a456-426614174002",
			Name:          "USB-C Charging Cable",
			Description:   "Fast charging USB-C cable, 2 meters",
			Price:         19.99,
			StockQuantity: 500,
			CategoryId:    "cat-accessories",
			Tags:          []string{"cable", "usb-c", "charging"},
			CreatedAt:     time.Now().Unix(),
			UpdatedAt:     time.Now().Unix(),
		},
		{
			Id:            "123e4567-e89b-12d3-a456-426614174003",
			Name:          "Mechanical Keyboard",
			Description:   "RGB mechanical keyboard with Cherry MX switches",
			Price:         149.99,
			StockQuantity: 75,
			CategoryId:    "cat-electronics",
			Tags:          []string{"keyboard", "mechanical", "gaming"},
			CreatedAt:     time.Now().Unix(),
			UpdatedAt:     time.Now().Unix(),
		},
	}

	for _, p := range sampleProducts {
		svc.products[p.Id] = p
	}
	svc.nextID = 4

	return svc
}

// ListProducts returns a paginated list of products with optional filtering.
func (s *ProductService) ListProducts(ctx context.Context, req *models.ListProductsRequest) (*models.ListProductsResponse, error) {
	// Apply filters and collect matching products
	var filtered []*models.Product
	for _, p := range s.products {
		// Category filter
		if req.Category != "" && p.CategoryId != req.Category {
			continue
		}
		// Price filters
		if req.MinPrice > 0 && p.Price < req.MinPrice {
			continue
		}
		if req.MaxPrice > 0 && p.Price > req.MaxPrice {
			continue
		}
		// Search filter (simple substring match)
		if req.Search != "" {
			// In production, use proper full-text search
			found := false
			if contains(p.Name, req.Search) || contains(p.Description, req.Search) {
				found = true
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, p)
	}

	// Calculate pagination
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	limit := int(req.Limit)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	totalCount := len(filtered)
	totalPages := (totalCount + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit
	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	return &models.ListProductsResponse{
		Products:   filtered[start:end],
		TotalCount: int32(totalCount),
		Page:       int32(page),
		TotalPages: int32(totalPages),
	}, nil
}

// GetProduct retrieves a single product by ID.
func (s *ProductService) GetProduct(ctx context.Context, req *models.GetProductRequest) (*models.Product, error) {
	product, exists := s.products[req.ProductId]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", req.ProductId)
	}
	return product, nil
}

// CreateProduct creates a new product.
func (s *ProductService) CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.Product, error) {
	product := &models.Product{
		Id:            fmt.Sprintf("123e4567-e89b-12d3-a456-42661417400%d", s.nextID),
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
		CategoryId:    req.CategoryId,
		Tags:          req.Tags,
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}
	s.nextID++
	s.products[product.Id] = product
	log.Printf("Created product: %s", product.Id)
	return product, nil
}

// UpdateProduct performs a full update of an existing product.
func (s *ProductService) UpdateProduct(ctx context.Context, req *models.UpdateProductRequest) (*models.Product, error) {
	product, exists := s.products[req.ProductId]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", req.ProductId)
	}

	// Full replacement - all fields are updated
	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.StockQuantity = req.StockQuantity
	product.CategoryId = req.CategoryId
	product.Tags = req.Tags
	product.UpdatedAt = time.Now().Unix()

	log.Printf("Updated product (PUT): %s", product.Id)
	return product, nil
}

// PatchProduct performs a partial update of an existing product.
func (s *ProductService) PatchProduct(ctx context.Context, req *models.PatchProductRequest) (*models.Product, error) {
	product, exists := s.products[req.ProductId]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", req.ProductId)
	}

	// Partial update - only update provided fields
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.StockQuantity != nil {
		product.StockQuantity = *req.StockQuantity
	}
	if req.CategoryId != nil {
		product.CategoryId = *req.CategoryId
	}
	product.UpdatedAt = time.Now().Unix()

	log.Printf("Updated product (PATCH): %s", product.Id)
	return product, nil
}

// DeleteProduct removes a product from the catalog.
func (s *ProductService) DeleteProduct(ctx context.Context, req *models.DeleteProductRequest) (*models.DeleteProductResponse, error) {
	_, exists := s.products[req.ProductId]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", req.ProductId)
	}

	delete(s.products, req.ProductId)
	log.Printf("Deleted product: %s", req.ProductId)

	return &models.DeleteProductResponse{
		Success: true,
		Message: fmt.Sprintf("Product %s deleted successfully", req.ProductId),
	}, nil
}

// contains performs a case-insensitive substring search.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsLower(s, substr)))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldSubstr(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFoldSubstr(s1, s2 string) bool {
	for i := 0; i < len(s1); i++ {
		c1, c2 := s1[i], s2[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

func main() {
	// Use our custom service implementation
	service := NewProductService()
	mux := http.NewServeMux()

	// Register the HTTP handlers (generated by protoc-gen-go-http)
	if err := services.RegisterProductServiceServer(service, services.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("RESTful CRUD API Server starting on :8080")
	fmt.Println("")
	fmt.Println("This example demonstrates all HTTP verbs with path and query parameters.")
	fmt.Println("")
	fmt.Println("Endpoints:")
	fmt.Println("  GET    /api/v1/products              - List products (with pagination/filtering)")
	fmt.Println("  GET    /api/v1/products/{product_id} - Get a single product")
	fmt.Println("  POST   /api/v1/products              - Create a new product")
	fmt.Println("  PUT    /api/v1/products/{product_id} - Full update (replace entire resource)")
	fmt.Println("  PATCH  /api/v1/products/{product_id} - Partial update (only provided fields)")
	fmt.Println("  DELETE /api/v1/products/{product_id} - Delete a product")
	fmt.Println("")
	fmt.Println("Required Headers:")
	fmt.Println("  X-API-Key: <uuid>                    - Required for all operations")
	fmt.Println("  X-Confirm-Delete: true               - Required for DELETE operations")
	fmt.Println("")
	fmt.Println("Query Parameters (for GET /products):")
	fmt.Println("  page=1         - Page number")
	fmt.Println("  limit=20       - Items per page (max 100)")
	fmt.Println("  category=<id>  - Filter by category")
	fmt.Println("  min_price=0    - Minimum price filter")
	fmt.Println("  max_price=999  - Maximum price filter")
	fmt.Println("  sort=price     - Sort field (name, price, created_at)")
	fmt.Println("  desc=false     - Sort descending")
	fmt.Println("  q=<search>     - Search in name/description")
	fmt.Println("")
	fmt.Println("Example requests:")
	fmt.Println("")
	fmt.Println("List products:")
	fmt.Println("  curl -X GET 'http://localhost:8080/api/v1/products?page=1&limit=10' \\")
	fmt.Println("    -H 'X-API-Key: 123e4567-e89b-12d3-a456-426614174000'")
	fmt.Println("")
	fmt.Println("Create product:")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/products \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -H 'X-API-Key: 123e4567-e89b-12d3-a456-426614174000' \\")
	fmt.Println("    -d '{\"name\": \"New Product\", \"price\": 29.99, \"stock_quantity\": 50}'")
	fmt.Println("")

	log.Fatal(http.ListenAndServe(":8080", mux))
}
