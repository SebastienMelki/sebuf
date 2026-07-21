.PHONY: test demo install generate run clean curl

# Default: generate code and run the non-blocking end-to-end proof.
test: generate
	@go test ./...

# Demo workflow - generates code, then starts the blocking server (Ctrl+C to stop).
demo: generate run

# Install required tools
install:
	@echo "Installing sebuf plugins..."
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@GOPROXY=direct go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
	@echo "Tools installed"

# Generate code from proto files (requires ../../bin plugins: run `make build` at repo root)
generate:
	@echo "Generating code..."
	@buf generate
	@go mod tidy
	@echo "Code generated"

# Run the server (blocks until Ctrl+C; hit it from another terminal with `make curl`)
run:
	@echo "Starting server on :8080 (Ctrl+C to stop). From another terminal: make curl"
	@go run main.go

# Hit the running server (start it with `make run` first)
curl:
	@curl -s localhost:8080/api/v1/suggestion -d '{"symbol":"AAPL","requestedRisk":"low"}'
	@echo ""

# Clean generated files
clean:
	@rm -rf api
	@echo "Cleaned generated files"
