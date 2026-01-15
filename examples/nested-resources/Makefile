.PHONY: demo install generate run clean test

# Complete demo workflow - installs tools, generates code, runs server
demo:
	@rm -rf api docs
	@echo "Running Nested Resources demo..."
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

# Test the API endpoints with nested resources
test:
	@echo "Testing Nested Resources API endpoints..."
	@echo ""
	@echo "=== Level 1: Organizations ==="
	@echo ""
	@echo "1. List organizations:"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs?page=1&limit=10" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "2. Get organization:"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs/org-abc123" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "=== Level 2: Teams (nested under org) ==="
	@echo ""
	@echo "3. List teams in organization:"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs/org-abc123/teams?page=1&limit=10" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "4. Get specific team:"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "=== Level 3: Members (nested under team) ==="
	@echo ""
	@echo "5. List team members (3-level nesting):"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/members?page=1&limit=10" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "6. Get specific member (deepest nesting - 3 path params):"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/members/member-456" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "=== Level 3: Projects (nested under team) ==="
	@echo ""
	@echo "7. List projects (3-level nesting with status filter):"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/projects?status=active" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "8. Get specific project (3 path params):"
	@curl -s -X GET "http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/projects/proj-def456" \
		-H "Authorization: Bearer test-token" \
		| python3 -m json.tool
	@echo ""
	@echo "=== Create Operations ==="
	@echo ""
	@echo "9. Create a new team (POST with 1 path param):"
	@curl -s -X POST "http://localhost:8080/api/v1/orgs/org-abc123/teams" \
		-H "Authorization: Bearer test-token" \
		-H "Content-Type: application/json" \
		-d '{"name": "New Team", "description": "A new team", "private": false}' \
		| python3 -m json.tool
	@echo ""
	@echo "10. Add member to team (POST with 2 path params):"
	@curl -s -X POST "http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/members" \
		-H "Authorization: Bearer test-token" \
		-H "Content-Type: application/json" \
		-d '{"user_id": "user-new123", "role": 1}' \
		| python3 -m json.tool
	@echo ""
	@echo "=== All nested resource tests completed ==="
