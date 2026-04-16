.PHONY: demo generate run clean test test-stream test-trades

SERVER_PID_FILE := /tmp/sse-demo-server.pid

# Complete demo: generate code, start server, run tests, stop server
demo:
	@rm -rf api
	@echo "=== SSE Streaming Demo ==="
	@echo ""
	@$(MAKE) generate
	@echo ""
	@echo "Starting server..."
	@$(MAKE) start-bg
	@echo ""
	@$(MAKE) test
	@echo ""
	@$(MAKE) test-stream
	@echo ""
	@$(MAKE) test-trades
	@echo ""
	@$(MAKE) stop
	@echo ""
	@echo "=== Demo complete ==="

# Generate code from proto files
generate:
	@echo "Fetching dependencies..."
	@buf dep update
	@echo "Generating code..."
	@buf generate
	@echo "Updating Go modules..."
	@go mod tidy
	@echo "Code generated"

# Run the server (foreground)
run: generate
	@go run main.go

# Start server in background
start-bg:
	@go run main.go > /tmp/sse-demo-server.log 2>&1 & echo $$! > $(SERVER_PID_FILE)
	@echo "Server started (PID $$(cat $(SERVER_PID_FILE)))"
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		curl -s -o /dev/null http://localhost:8080/api/v1/quotes/AAPL 2>/dev/null && break; \
		sleep 0.5; \
	done

# Stop background server
stop:
	@if [ -f $(SERVER_PID_FILE) ]; then \
		kill $$(cat $(SERVER_PID_FILE)) 2>/dev/null || true; \
		rm -f $(SERVER_PID_FILE); \
		echo "Server stopped"; \
	fi

# Test unary quote endpoint
test:
	@echo "--- GET /api/v1/quotes/AAPL (unary) ---"
	@curl -s http://localhost:8080/api/v1/quotes/AAPL | python3 -m json.tool
	@echo ""
	@echo "--- GET /api/v1/quotes/GOOG (unary) ---"
	@curl -s http://localhost:8080/api/v1/quotes/GOOG | python3 -m json.tool

# Test SSE quote stream (read 5 events then stop)
test-stream:
	@echo "--- GET /api/v1/quotes/TSLA/stream (SSE, 5 events) ---"
	@curl -s -N http://localhost:8080/api/v1/quotes/TSLA/stream 2>/dev/null | head -15

# Test SSE trade feed with query params
test-trades:
	@echo "--- GET /api/v1/trades/stream?symbol=AAPL&limit=5 (SSE trades) ---"
	@curl -s -N 'http://localhost:8080/api/v1/trades/stream?symbol=AAPL&limit=5'

# Clean generated files
clean:
	@rm -rf api
	@rm -f $(SERVER_PID_FILE)
	@echo "Cleaned"
