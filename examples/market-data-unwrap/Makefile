.PHONY: demo install generate run client clean test test-unwrap

# Complete demo workflow - installs tools, generates code, runs server
demo:
	@rm -rf api docs
	@echo "🚀 Running market data unwrap demo..."
	@$(MAKE) generate
	@echo ""
	@echo "✅ Demo ready! Starting server..."
	@echo ""
	@$(MAKE) run

# Install required tools
install:
	@echo "Installing sebuf plugins..."
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@GOPROXY=direct go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
	@GOPROXY=direct go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-client@latest
	@GOPROXY=direct go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
	@echo "✅ Tools installed"

# Generate code from proto files
generate:
	@echo "Fetching dependencies..."
	@buf dep update
	@echo "Generating code..."
	@buf generate
	@echo "Updating Go modules..."
	@go mod tidy
	@echo "✅ Code generated and dependencies updated"

# Run the server
run: generate
	@go run main.go

# Run the client example (requires server running)
client:
	@go run client_example.go

# Clean generated files
clean:
	@rm -rf api docs
	@echo "✅ Cleaned generated files"

# Test the API endpoints with valid unwrapped response
test:
	@echo "Testing API endpoints..."
	@echo ""
	@echo "1. Get option bars (demonstrates unwrap serialization):"
	@curl -s -X GET "http://localhost:8080/v2/options/bars?symbols=TSLA260123C00335000&timeframe=1Day" \
		-H "APCA-API-KEY-ID: test-key" \
		-H "APCA-API-SECRET-KEY: test-secret" \
		| python3 -m json.tool
	@echo ""
	@echo "2. Get latest option bars:"
	@curl -s -X GET "http://localhost:8080/v2/options/bars/latest?symbols=TSLA260123C00335000" \
		-H "APCA-API-KEY-ID: test-key" \
		-H "APCA-API-SECRET-KEY: test-secret" \
		| python3 -m json.tool

# Test to show the unwrap behavior - compare JSON structure
test-unwrap:
	@echo "Demonstrating unwrap behavior..."
	@echo ""
	@echo "With unwrap annotation, the response looks like:"
	@echo '{"bars": {"TSLA260123C00335000": [{"c": 143.08, ...}, {"c": 145.34, ...}]}}'
	@echo ""
	@echo "Without unwrap, it would look like:"
	@echo '{"bars": {"TSLA260123C00335000": {"bars": [{"c": 143.08, ...}, {"c": 145.34, ...}]}}}'
	@echo ""
	@echo "Making actual request to see the unwrapped response:"
	@curl -s -X GET "http://localhost:8080/v2/options/bars?symbols=TSLA260123C00335000&timeframe=1Day" \
		-H "APCA-API-KEY-ID: test-key" \
		-H "APCA-API-SECRET-KEY: test-secret" \
		| python3 -m json.tool

# Test validation errors
test-validation:
	@echo "Testing validation errors..."
	@echo ""
	@echo "1. Missing required symbol parameter (should return 400):"
	@curl -s -X GET "http://localhost:8080/v2/options/bars?timeframe=1Day" \
		-H "APCA-API-KEY-ID: test-key" \
		-H "APCA-API-SECRET-KEY: test-secret" \
		| python3 -m json.tool
	@echo ""
	@echo "2. Invalid sort value (should return 400):"
	@curl -s -X GET "http://localhost:8080/v2/options/bars?symbols=TSLA&timeframe=1Day&sort=invalid" \
		-H "APCA-API-KEY-ID: test-key" \
		-H "APCA-API-SECRET-KEY: test-secret" \
		| python3 -m json.tool
	@echo ""
	@echo "3. Missing required header (should return 400):"
	@curl -s -X GET "http://localhost:8080/v2/options/bars?symbols=TSLA&timeframe=1Day" \
		| python3 -m json.tool
