.PHONY: demo install generate run clean test test-public test-user test-admin

# Complete demo workflow - installs tools, generates code, runs server
demo:
	@rm -rf api docs
	@echo "Running Multi-Service API demo..."
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

# Test all endpoints
test: test-public test-user test-admin

# Test public endpoints (no auth required)
test-public:
	@echo "=== Testing PUBLIC Service (no authentication) ==="
	@echo ""
	@echo "1. Health check (no auth):"
	@curl -s -X GET "http://localhost:8080/api/v1/public/health" \
		| python3 -m json.tool
	@echo ""
	@echo "2. API info (no auth):"
	@curl -s -X GET "http://localhost:8080/api/v1/public/info" \
		| python3 -m json.tool
	@echo ""

# Test user-authenticated endpoints
test-user:
	@echo "=== Testing USER Service (user authentication required) ==="
	@echo ""
	@echo "3. Get current user (requires Authorization + X-Tenant-ID):"
	@curl -s -X GET "http://localhost:8080/api/v1/users/me" \
		-H "Authorization: Bearer user-token-xyz" \
		-H "X-Tenant-ID: tenant-abc123" \
		| python3 -m json.tool
	@echo ""
	@echo "4. Update profile (PATCH):"
	@curl -s -X PATCH "http://localhost:8080/api/v1/users/me" \
		-H "Authorization: Bearer user-token-xyz" \
		-H "X-Tenant-ID: tenant-abc123" \
		-H "Content-Type: application/json" \
		-d '{"name": "Updated Name"}' \
		| python3 -m json.tool
	@echo ""
	@echo "5. List users in tenant:"
	@curl -s -X GET "http://localhost:8080/api/v1/users?page=1&limit=10" \
		-H "Authorization: Bearer user-token-xyz" \
		-H "X-Tenant-ID: tenant-abc123" \
		| python3 -m json.tool
	@echo ""
	@echo "6. FAIL: Missing X-Tenant-ID header:"
	@curl -s -X GET "http://localhost:8080/api/v1/users/me" \
		-H "Authorization: Bearer user-token-xyz" \
		| python3 -m json.tool
	@echo ""

# Test admin-authenticated endpoints
test-admin:
	@echo "=== Testing ADMIN Service (admin authentication + role required) ==="
	@echo ""
	@echo "7. List all tenants (admin only):"
	@curl -s -X GET "http://localhost:8080/api/v1/admin/tenants?page=1&limit=10" \
		-H "Authorization: Bearer admin-token-xyz" \
		-H "X-Admin-Role: admin" \
		| python3 -m json.tool
	@echo ""
	@echo "8. Create a new tenant:"
	@curl -s -X POST "http://localhost:8080/api/v1/admin/tenants" \
		-H "Authorization: Bearer admin-token-xyz" \
		-H "X-Admin-Role: admin" \
		-H "Content-Type: application/json" \
		-d '{"name": "New Tenant", "domain": "newtenant.example.com", "plan": "professional"}' \
		| python3 -m json.tool
	@echo ""
	@echo "9. List all users across tenants:"
	@curl -s -X GET "http://localhost:8080/api/v1/admin/users?page=1&limit=10" \
		-H "Authorization: Bearer admin-token-xyz" \
		-H "X-Admin-Role: admin" \
		| python3 -m json.tool
	@echo ""
	@echo "10. Delete tenant (requires X-Confirm-Delete header):"
	@curl -s -X DELETE "http://localhost:8080/api/v1/admin/tenants/tenant-to-delete" \
		-H "Authorization: Bearer admin-token-xyz" \
		-H "X-Admin-Role: super_admin" \
		-H "X-Confirm-Delete: true" \
		| python3 -m json.tool
	@echo ""
	@echo "11. Impersonate user (requires X-Audit-Reason header):"
	@curl -s -X POST "http://localhost:8080/api/v1/admin/users/user-xyz789/impersonate" \
		-H "Authorization: Bearer admin-token-xyz" \
		-H "X-Admin-Role: super_admin" \
		-H "X-Audit-Reason: Customer support ticket #12345" \
		| python3 -m json.tool
	@echo ""
	@echo "12. FAIL: Delete without X-Confirm-Delete:"
	@curl -s -X DELETE "http://localhost:8080/api/v1/admin/tenants/tenant-abc123" \
		-H "Authorization: Bearer admin-token-xyz" \
		-H "X-Admin-Role: admin" \
		| python3 -m json.tool
	@echo ""
	@echo "=== All tests completed ==="
