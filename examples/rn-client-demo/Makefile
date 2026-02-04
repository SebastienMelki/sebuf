.PHONY: demo generate server install-app start-app clean build-plugins

# Full demo: generate code, start server, install app deps
demo: generate install-app
	@echo ""
	@echo "Starting Go server in background..."
	@go run main.go & SERVER_PID=$$!; \
	echo "Waiting for server to be ready..."; \
	for i in $$(seq 1 30); do \
		if curl -s -o /dev/null http://localhost:3000/api/v1/tasks 2>/dev/null; then \
			break; \
		fi; \
		sleep 1; \
	done; \
	echo ""; \
	echo "Server is ready! Start the Expo app with:"; \
	echo "  cd app && npx expo start"; \
	echo ""; \
	echo "Press Ctrl+C to stop the server."; \
	wait $$SERVER_PID

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

# Install Expo app dependencies
install-app:
	@echo "Installing Expo app dependencies..."
	@cd app && npm install --silent
	@cd app && npx expo install --fix

# Start Expo dev server (assumes npm install done)
start-app:
	@cd app && npx expo start

# Clean generated files
clean:
	@rm -rf api/ app/generated/ app/node_modules/ app/.expo/
	@echo "Cleaned generated files"
