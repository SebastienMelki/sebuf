.PHONY: demo install generate run clean test

# Complete demo workflow - generates code, runs the side-by-side servers
demo:
	@rm -rf api
	@echo "Running marshal-options demo..."
	@$(MAKE) generate
	@echo ""
	@echo "Demo ready! Starting servers on :8080 (default) and :8081 (EmitUnpopulated)..."
	@echo ""
	@$(MAKE) run

# Install required tools
install:
	@echo "Installing sebuf plugins..."
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@GOPROXY=direct go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
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

# Run the server pair
run: generate
	@go run main.go

# Clean generated files
clean:
	@rm -rf api
	@echo "Cleaned generated files"

# Hit both servers and show the difference
test:
	@echo "=== Default opts (port 8080) — zero values omitted ==="
	@curl -s http://localhost:8080/api/v1/status
	@echo ""
	@echo ""
	@echo "=== EmitUnpopulated: true (port 8081) — zero values surfaced ==="
	@curl -s http://localhost:8081/api/v1/status
	@echo ""
