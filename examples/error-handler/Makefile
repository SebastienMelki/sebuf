.PHONY: demo install generate run clean test

# Complete demo workflow - installs tools, generates code, runs server
demo:
	@rm -rf api docs
	@echo "Running error-handler demo..."
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
	@rm -rf api docs
	@echo "Cleaned generated files"

# Test the API endpoints
test:
	@echo "Testing API endpoints..."
	@echo "1. Create user (success):"
	@curl -X POST http://localhost:8080/api/v1/users \
		-H "Content-Type: application/json" \
		-H "X-Request-ID: test-123" \
		-d '{"name": "John Doe", "email": "john@example.com"}' \
		2>/dev/null | python3 -m json.tool || true
	@echo ""
	@echo "2. Validation error (short name):"
	@curl -X POST http://localhost:8080/api/v1/users \
		-H "Content-Type: application/json" \
		-d '{"name": "J", "email": "invalid"}' \
		2>/dev/null | python3 -m json.tool || true
	@echo ""
	@echo "3. Not found error:"
	@curl -v http://localhost:8080/api/v1/users/non-existent 2>&1 | grep -E "^(<|>|HTTP|{)"
