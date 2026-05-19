.PHONY: demo generate server client clean build-plugins

# Full demo: generate code, start server, run Python client, stop server
demo: generate
	@echo ""
	@echo "Starting Go server in background..."
	@go run main.go & SERVER_PID=$$!; \
	echo "Waiting for server to be ready..."; \
	for i in $$(seq 1 30); do \
		if curl -s -o /dev/null http://localhost:3000/api/v1/notes 2>/dev/null; then \
			break; \
		fi; \
		sleep 1; \
	done; \
	echo ""; \
	echo "Running Python client..."; \
	echo ""; \
	cd client && python3 main.py; \
	EXIT_CODE=$$?; \
	echo ""; \
	echo "Stopping server (PID $$SERVER_PID)..."; \
	kill $$SERVER_PID 2>/dev/null; \
	exit $$EXIT_CODE

# Build the protoc plugin binaries
build-plugins:
	@echo "Building protoc plugins..."
	@cd ../.. && go build -o bin/protoc-gen-go-http ./cmd/protoc-gen-go-http
	@cd ../.. && go build -o bin/protoc-gen-py-client ./cmd/protoc-gen-py-client

# Generate Go + Python code from proto
generate: build-plugins
	@echo "Fetching buf dependencies..."
	@buf dep update
	@echo "Generating code..."
	@buf generate
	@echo "Updating Go modules..."
	@go mod tidy
	@echo "Code generated successfully"

# Run the Go server (blocks)
server: generate
	@go run main.go

# Run just the Python client (assumes server is running)
client:
	@cd client && python3 main.py

# Clean generated files
clean:
	@rm -rf api/ client/generated/ client/__pycache__/
	@echo "Cleaned generated files"
