# KrakenD Gateway Example Makefile
#
# Workflow:
#   make generate    — Generate KrakenD config + Go HTTP handlers from proto
#   make run         — Run the backend server locally (no gateway)
#   make demo        — Generate all code, start gateway + backend via Docker
#   make test        — Curl the gateway to verify endpoints work
#   make clean       — Remove generated files and stop containers

.PHONY: generate generate-krakend generate-go run demo test docker-up docker-down clean

# Plugin binaries (built from repo root with `make build`).
PLUGIN_KRAKEND  := ../../bin/protoc-gen-krakend
PLUGIN_GO_HTTP  := ../../bin/protoc-gen-go-http

# Proto paths.
PROTO_PATH_LOCAL := proto
PROTO_PATH_SEBUF := ../../proto

# Proto source files.
PROTOS := proto/services/user_service.proto proto/services/product_service.proto

# Module path for go_opt stripping.
MODULE := github.com/SebastienMelki/sebuf/examples/krakend-gateway

# --------------------------------------------------------------------------
# Generate all code from proto definitions.
# --------------------------------------------------------------------------

generate: generate-krakend generate-go
	@go mod tidy
	@echo "Done. All code generated."

# Generate a single krakend.json from all service protos.
generate-krakend:
	@echo "Generating KrakenD config..."
	@mkdir -p gateway
	@protoc \
		--plugin=protoc-gen-krakend=$(PLUGIN_KRAKEND) \
		--krakend_out=./gateway \
		--proto_path=$(PROTO_PATH_LOCAL) \
		--proto_path=$(PROTO_PATH_SEBUF) \
		$(PROTOS)
	@echo "Generated: gateway/krakend.json"

# Generate Go protobuf types + HTTP handlers.
generate-go:
	@echo "Generating Go code..."
	@rm -rf api
	@mkdir -p api
	@protoc \
		--go_out=./api \
		--go_opt=module=$(MODULE)/api \
		--plugin=protoc-gen-go-http=$(PLUGIN_GO_HTTP) \
		--go-http_out=./api \
		--go-http_opt=module=$(MODULE)/api \
		--proto_path=$(PROTO_PATH_LOCAL) \
		--proto_path=$(PROTO_PATH_SEBUF) \
		$(PROTOS) proto/models/common.proto
	@echo "Generated Go code in api/"

# --------------------------------------------------------------------------
# Run / Docker targets
# --------------------------------------------------------------------------

run:
	@go run main.go

docker-up:
	@docker compose up -d --build
	@echo ""
	@echo "Gateway running at http://localhost:8080"
	@echo "Backend running at http://localhost:8081 (direct)"
	@echo ""

docker-down:
	@docker compose down

demo: generate docker-up
	@echo "Try: make test"

# --------------------------------------------------------------------------
# Test the running gateway with curl
# --------------------------------------------------------------------------

test:
	@echo "=== ProductService (no JWT required) ==="
	@echo ""
	@echo "--- ListProducts ---"
	@curl -s http://localhost:8080/api/v1/products \
		-H "X-API-Key: test-key" | python3 -m json.tool || true
	@echo ""
	@echo "--- GetProduct ---"
	@curl -s http://localhost:8080/api/v1/products/prod-1 \
		-H "X-API-Key: test-key" | python3 -m json.tool || true
	@echo ""
	@echo "--- CreateProduct ---"
	@curl -s -X POST http://localhost:8080/api/v1/products \
		-H "Content-Type: application/json" \
		-H "X-API-Key: test-key" \
		-d '{"name":"Widget","description":"A fine widget","priceCents":999,"category":"tools"}' \
		| python3 -m json.tool || true
	@echo ""
	@echo "--- DeleteProduct ---"
	@curl -s -X DELETE http://localhost:8080/api/v1/products/prod-1 \
		-H "X-API-Key: test-key" | python3 -m json.tool || true
	@echo ""
	@echo "=== UserService (JWT required — expect 401) ==="
	@echo ""
	@echo "--- GetUser (no token) ---"
	@curl -s -w "\nHTTP Status: %{http_code}\n" http://localhost:8080/api/v1/users/user-1 \
		-H "X-API-Key: test-key" \
		-H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000" || true
	@echo ""

# --------------------------------------------------------------------------
# Clean up
# --------------------------------------------------------------------------

clean:
	@rm -rf api gateway/krakend.json
	@docker compose down 2>/dev/null || true
	@echo "Cleaned."
