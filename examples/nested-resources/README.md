# Nested Resources Example

This example demonstrates complex nested resource hierarchies with multiple path parameters, following a GitHub-like pattern for organizations, teams, members, and projects.

## Features Demonstrated

| Feature | Description |
|---------|-------------|
| Single path parameter | `/orgs/{org_id}` |
| Two path parameters | `/orgs/{org_id}/teams/{team_id}` |
| Three path parameters | `/orgs/{org_id}/teams/{team_id}/members/{member_id}` |
| Path + query params | `/orgs/{org_id}/teams?include_private=true&page=1` |
| Path + request body | `POST /orgs/{org_id}/teams` with JSON body |

## Resource Hierarchy

```
Organization (Level 1)
    |
    +-- Team (Level 2)
          |
          +-- Member (Level 3)
          |
          +-- Project (Level 3)
```

## Quick Start

```bash
# Generate code and run the server
make demo

# Test nested resource endpoints
make test
```

## API Endpoints

### Organizations (1 path parameter)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs` | List organizations |
| GET | `/api/v1/orgs/{org_id}` | Get organization |
| POST | `/api/v1/orgs` | Create organization |

### Teams (2 path parameters)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{org_id}/teams` | List teams |
| GET | `/api/v1/orgs/{org_id}/teams/{team_id}` | Get team |
| POST | `/api/v1/orgs/{org_id}/teams` | Create team |

### Members (3 path parameters)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{org_id}/teams/{team_id}/members` | List members |
| GET | `/api/v1/orgs/{org_id}/teams/{team_id}/members/{member_id}` | Get member |
| POST | `/api/v1/orgs/{org_id}/teams/{team_id}/members` | Add member |
| DELETE | `/api/v1/orgs/{org_id}/teams/{team_id}/members/{member_id}` | Remove member |

### Projects (3 path parameters)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/orgs/{org_id}/teams/{team_id}/projects` | List projects |
| GET | `/api/v1/orgs/{org_id}/teams/{team_id}/projects/{project_id}` | Get project |
| POST | `/api/v1/orgs/{org_id}/teams/{team_id}/projects` | Create project |

## Example Requests

### Get Organization (1 path parameter)

```bash
curl -X GET http://localhost:8080/api/v1/orgs/org-abc123 \
  -H "Authorization: Bearer test-token"
```

### Get Team (2 path parameters)

```bash
curl -X GET http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789 \
  -H "Authorization: Bearer test-token"
```

### Get Member (3 path parameters - deepest nesting)

```bash
curl -X GET http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/members/member-456 \
  -H "Authorization: Bearer test-token"
```

### List Teams with Query Parameters

```bash
curl -X GET "http://localhost:8080/api/v1/orgs/org-abc123/teams?page=1&limit=10&include_private=true" \
  -H "Authorization: Bearer test-token"
```

### Create Team (path parameter + body)

```bash
curl -X POST http://localhost:8080/api/v1/orgs/org-abc123/teams \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Team",
    "description": "A new team",
    "private": false
  }'
```

### Add Member (2 path parameters + body)

```bash
curl -X POST http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/members \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-new123",
    "role": 1
  }'
```

### Remove Member (3 path parameters)

```bash
curl -X DELETE http://localhost:8080/api/v1/orgs/org-abc123/teams/team-xyz789/members/member-456 \
  -H "Authorization: Bearer test-token"
```

## Proto Definition Pattern

Path parameters are automatically extracted from the URL and bound to request fields with matching names:

```protobuf
// Request with 3 path parameters
message GetTeamMemberRequest {
  string org_id = 1;     // Bound from {org_id} in path
  string team_id = 2;    // Bound from {team_id} in path
  string member_id = 3;  // Bound from {member_id} in path
}

// Service definition
service OrganizationService {
  rpc GetTeamMember(GetTeamMemberRequest) returns (Member) {
    option (sebuf.http.config) = {
      path: "/orgs/{org_id}/teams/{team_id}/members/{member_id}"
      method: HTTP_METHOD_GET
    };
  }
}
```

## Combining Path and Query Parameters

```protobuf
message ListTeamMembersRequest {
  // From path
  string org_id = 1;
  string team_id = 2;

  // From query string
  int32 page = 3 [(sebuf.http.query) = { name: "page" }];
  int32 limit = 4 [(sebuf.http.query) = { name: "limit" }];
  string role_filter = 5 [(sebuf.http.query) = { name: "role" }];
}
```

## Generated Files

After running `make generate`:

```
api/
  proto/
    models/
      organization.pb.go           # All message types
    services/
      organization_service.pb.go         # Service interface
      organization_service_http.pb.go    # HTTP handlers (12 endpoints!)
      organization_service_http_binding.pb.go  # Request binding
      organization_service_http_config.pb.go   # Server options
      organization_service_http_mock.pb.go     # Mock implementation
docs/
  OrganizationService.openapi.yaml  # OpenAPI with all nested paths
  OrganizationService.openapi.json
```

## Key Concepts

### Path Parameter Binding

Path parameters like `{org_id}` are automatically extracted from the URL and bound to the corresponding request message field. The field name must match the path variable name.

### Hierarchical Resource Ownership

The nested URL structure enforces resource ownership:
- A team belongs to an organization
- A member/project belongs to a team within an organization

This is validated by including parent IDs in each request, allowing your service to verify ownership.

### REST Best Practices

This example follows REST conventions:
- Collection endpoints use plural nouns (`/orgs`, `/teams`, `/members`)
- Single resource endpoints include the ID (`/orgs/{org_id}`)
- Nested resources show parent-child relationships in the URL
- HTTP verbs match the operation (GET=read, POST=create, DELETE=remove)
