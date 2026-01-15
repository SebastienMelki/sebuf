.PHONY: demo install generate run clean test

# Complete demo workflow - installs tools, generates code, runs server
demo:
	@rm -rf api docs
	@echo "Running RESTful CRUD demo..."
#	@$(MAKE) install
	@$(MAKE) generate
	@echo ""
	@echo "Demo ready! Starting server..."
	@echo ""
	@$(MAKE) run

# Install required tools
install:
	@echo "Installing sebuf plugins..."
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@GOPROXY=direct go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
	@GOPROXY=direct go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
	@echo "Tools installed"

# Generate code from proto files
generate:
	@echo "Fetching dependencies..."
	@buf dep update
	@echo "Generating code..."
	@buf generate
	@echo "Updating Go modules..."
	@go mod tidy
	@echo "Code generated and dependencies updated"

# Run the server
run: generate
	@go run main.go

# Clean generated files
clean:
	@rm -f *.pb.go
	@rm -f *_helpers.pb.go
	@rm -f *_http*.pb.go
	@rm -rf api/ docs/
	@echo "Cleaned generated files"

# Test the API endpoints (requires server running on :8080)
test:
	@echo "Testing RESTful CRUD API endpoints..."
	@echo ""
	@echo "=== 1. Create a product (POST) ==="
	@curl -s -X POST http://localhost:8080/api/v1/products \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{"name": "Wireless Headphones", "description": "Noise-canceling Bluetooth headphones", "price": 99.99, "stock_quantity": 100, "category_id": "cat-electronics", "tags": ["audio", "wireless"]}' \
		| python3 -m json.tool
	@echo ""
	@echo "=== 2. List products with pagination (GET) ==="
	@curl -s -X GET "http://localhost:8080/api/v1/products?page=1&limit=10&sort=price&desc=false" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		| python3 -m json.tool
	@echo ""
	@echo "=== 3. Get a specific product (GET with path param) ==="
	@curl -s -X GET "http://localhost:8080/api/v1/products/123e4567-e89b-12d3-a456-426614174001" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		| python3 -m json.tool
	@echo ""
	@echo "=== 4. Full update a product (PUT) ==="
	@curl -s -X PUT "http://localhost:8080/api/v1/products/123e4567-e89b-12d3-a456-426614174001" \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{"name": "Premium Wireless Headphones", "description": "Updated premium noise-canceling headphones", "price": 149.99, "stock_quantity": 50, "category_id": "cat-electronics", "tags": ["audio", "wireless", "premium"]}' \
		| python3 -m json.tool
	@echo ""
	@echo "=== 5. Partial update a product (PATCH) ==="
	@curl -s -X PATCH "http://localhost:8080/api/v1/products/123e4567-e89b-12d3-a456-426614174001" \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{"price": 129.99}' \
		| python3 -m json.tool
	@echo ""
	@echo "=== 6. Delete a product (DELETE with confirmation header) ==="
	@curl -s -X DELETE "http://localhost:8080/api/v1/products/123e4567-e89b-12d3-a456-426614174001" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-H "X-Confirm-Delete: true" \
		| python3 -m json.tool
	@echo ""
	@echo "=== All tests completed ==="
