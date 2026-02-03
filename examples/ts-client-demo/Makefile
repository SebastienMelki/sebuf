.PHONY: demo generate server client clean install-client build-plugins

# Full demo: generate code, start server, run TS client, stop server
demo: generate install-client
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
	echo "Running TypeScript client..."; \
	echo ""; \
	cd client && npx tsx main.ts; \
	EXIT_CODE=$$?; \
	echo ""; \
	echo "Stopping server (PID $$SERVER_PID)..."; \
	kill $$SERVER_PID 2>/dev/null; \
	exit $$EXIT_CODE

# Build the protoc plugin binaries
build-plugins:
	@echo "Building protoc plugins..."
	@cd ../.. && go build -o bin/protoc-gen-go-http ./cmd/protoc-gen-go-http
	@cd ../.. && go build -o bin/protoc-gen-ts-client ./cmd/protoc-gen-ts-client

# Generate Go + TypeScript code from proto
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

# Install TypeScript dependencies
install-client:
	@echo "Installing TypeScript dependencies..."
	@cd client && npm install --silent

# Run just the TypeScript client (assumes server is running)
client: install-client
	@cd client && npx tsx main.ts

# Clean generated files
clean:
	@rm -rf api/ client/generated/ client/node_modules/ client/dist/
	@echo "Cleaned generated files"
