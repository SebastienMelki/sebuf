.PHONY: demo install generate run clean test test-validation

# Complete demo workflow - installs tools, generates code, runs server
demo:
	@rm -rf api docs
	@echo "Running Validation Showcase demo..."
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

# Test valid requests
test:
	@echo "Testing Validation Showcase API endpoints..."
	@echo ""
	@echo "=== 1. Create a valid order ==="
	@curl -s -X POST http://localhost:8080/api/v1/orders \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"contact": { \
				"email": "john.doe@example.com", \
				"phone": "+14155551234" \
			}, \
			"shipping_address": { \
				"street": "123 Main Street, Apt 4B", \
				"city": "San Francisco", \
				"state": "CA", \
				"postal_code": "94102", \
				"country": "US" \
			}, \
			"items": [{ \
				"product_id": "123e4567-e89b-12d3-a456-426614174000", \
				"product_name": "Wireless Headphones", \
				"sku": "SKU-ABC123DEF", \
				"quantity": 2, \
				"unit_price": 99.99, \
				"discount_percent": 10.0, \
				"options": {"color": "blue", "size": "large"} \
			}], \
			"coupon_codes": ["SAVE20"], \
			"payment_method": 1 \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== 2. Validate an address ==="
	@curl -s -X POST http://localhost:8080/api/v1/validate/address \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"address": { \
				"street": "456 Oak Avenue", \
				"city": "Los Angeles", \
				"state": "CA", \
				"postal_code": "90001", \
				"country": "US" \
			} \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== 3. Validate a coupon code ==="
	@curl -s -X POST http://localhost:8080/api/v1/validate/coupon \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{"code": "SAVE20", "subtotal": 199.99}' \
		| python3 -m json.tool
	@echo ""
	@echo "=== All valid requests completed ==="

# Test validation errors (expect HTTP 400 responses)
test-validation:
	@echo "Testing validation error responses..."
	@echo ""
	@echo "=== 1. Invalid email format ==="
	@curl -s -X POST http://localhost:8080/api/v1/orders \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"contact": { "email": "not-an-email" }, \
			"shipping_address": { "street": "123 Main St", "city": "SF", "state": "CA", "postal_code": "94102", "country": "US" }, \
			"items": [{"product_id": "123e4567-e89b-12d3-a456-426614174000", "product_name": "Test", "sku": "SKU-TEST123", "quantity": 1, "unit_price": 10.0}], \
			"payment_method": 1 \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== 2. Invalid state code (must be 2 uppercase letters) ==="
	@curl -s -X POST http://localhost:8080/api/v1/validate/address \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"address": { "street": "123 Main St", "city": "SF", "state": "California", "postal_code": "94102", "country": "US" } \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== 3. Invalid postal code format ==="
	@curl -s -X POST http://localhost:8080/api/v1/validate/address \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"address": { "street": "123 Main St", "city": "SF", "state": "CA", "postal_code": "ABCDE", "country": "US" } \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== 4. Quantity out of range (must be 1-100) ==="
	@curl -s -X POST http://localhost:8080/api/v1/orders \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"contact": { "email": "test@example.com" }, \
			"shipping_address": { "street": "123 Main St", "city": "SF", "state": "CA", "postal_code": "94102", "country": "US" }, \
			"items": [{"product_id": "123e4567-e89b-12d3-a456-426614174000", "product_name": "Test", "sku": "SKU-TEST123", "quantity": 500, "unit_price": 10.0}], \
			"payment_method": 1 \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== 5. Empty items array (must have at least 1 item) ==="
	@curl -s -X POST http://localhost:8080/api/v1/orders \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"contact": { "email": "test@example.com" }, \
			"shipping_address": { "street": "123 Main St", "city": "SF", "state": "CA", "postal_code": "94102", "country": "US" }, \
			"items": [], \
			"payment_method": 1 \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== 6. Invalid coupon code format ==="
	@curl -s -X POST http://localhost:8080/api/v1/validate/coupon \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{"code": "invalid-coupon!", "subtotal": 99.99}' \
		| python3 -m json.tool
	@echo ""
	@echo "=== 7. Unsupported country ==="
	@curl -s -X POST http://localhost:8080/api/v1/validate/address \
		-H "Content-Type: application/json" \
		-H "X-API-Key: 123e4567-e89b-12d3-a456-426614174000" \
		-d '{ \
			"address": { "street": "123 Main St", "city": "Tokyo", "state": "TK", "postal_code": "12345", "country": "JP" } \
		}' | python3 -m json.tool
	@echo ""
	@echo "=== All validation tests completed ==="
