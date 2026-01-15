# Multi-Service API Example

This example demonstrates how to organize multiple services with different authentication requirements and header configurations in a single API.

## Features Demonstrated

| Feature | Description |
|---------|-------------|
| Multiple services | 3 services with different base paths and auth levels |
| No authentication | Public service with no headers required |
| User authentication | User service requiring Authorization + X-Tenant-ID |
| Admin authentication | Admin service requiring Authorization + X-Admin-Role |
| Method-level headers | Extra headers for sensitive operations (X-Confirm-Delete, X-Audit-Reason) |
| Shared models | Common types used across services |
| Multi-tenancy | Tenant-scoped user access vs. cross-tenant admin access |

## Service Architecture

```
Multi-Service API
    |
    +-- PublicService (/api/v1/public)
    |       No authentication required
    |       - Health check, API info
    |
    +-- UserService (/api/v1/users)
    |       User authentication required
    |       Headers: Authorization, X-Tenant-ID
    |       - Get/update profile, list tenant users
    |
    +-- AdminService (/api/v1/admin)
            Admin authentication required
            Headers: Authorization, X-Admin-Role
            Method-specific: X-Confirm-Delete, X-Audit-Reason
            - Tenant management, cross-tenant user access
```

## Quick Start

```bash
# Generate code and run the server
make demo

# Test different authentication levels
make test-public   # No auth required
make test-user     # User auth required
make test-admin    # Admin auth required
```

## API Endpoints

### PublicService (No Authentication)

| Method | Endpoint | Headers Required | Description |
|--------|----------|------------------|-------------|
| GET | `/api/v1/public/health` | None | Health check |
| GET | `/api/v1/public/info` | None | API information |

### UserService (User Authentication)

| Method | Endpoint | Headers Required | Description |
|--------|----------|------------------|-------------|
| GET | `/api/v1/users/me` | Authorization, X-Tenant-ID | Get current user |
| PATCH | `/api/v1/users/me` | Authorization, X-Tenant-ID | Update profile |
| GET | `/api/v1/users` | Authorization, X-Tenant-ID | List users in tenant |

### AdminService (Admin Authentication)

| Method | Endpoint | Headers Required | Description |
|--------|----------|------------------|-------------|
| GET | `/api/v1/admin/tenants` | Authorization, X-Admin-Role | List all tenants |
| POST | `/api/v1/admin/tenants` | Authorization, X-Admin-Role | Create tenant |
| DELETE | `/api/v1/admin/tenants/{id}` | Authorization, X-Admin-Role, **X-Confirm-Delete** | Delete tenant |
| GET | `/api/v1/admin/users` | Authorization, X-Admin-Role | List all users |
| POST | `/api/v1/admin/users/{id}/impersonate` | Authorization, X-Admin-Role, **X-Audit-Reason** | Impersonate user |

## Example Requests

### Public Endpoints (No Auth)

```bash
# Health check - accessible to everyone
curl -X GET http://localhost:8080/api/v1/public/health

# API info - accessible to everyone
curl -X GET http://localhost:8080/api/v1/public/info
```

### User Endpoints (User Auth)

```bash
# Get current user
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer user-token-xyz" \
  -H "X-Tenant-ID: tenant-abc123"

# Update profile
curl -X PATCH http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer user-token-xyz" \
  -H "X-Tenant-ID: tenant-abc123" \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name"}'

# Missing X-Tenant-ID returns HTTP 400
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer user-token-xyz"
# => {"violations":[{"field":"X-Tenant-ID","description":"required header missing"}]}
```

### Admin Endpoints (Admin Auth)

```bash
# List all tenants
curl -X GET http://localhost:8080/api/v1/admin/tenants \
  -H "Authorization: Bearer admin-token-xyz" \
  -H "X-Admin-Role: admin"

# Create tenant
curl -X POST http://localhost:8080/api/v1/admin/tenants \
  -H "Authorization: Bearer admin-token-xyz" \
  -H "X-Admin-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"name": "New Tenant", "domain": "new.example.com", "plan": "professional"}'

# Delete tenant (requires confirmation header)
curl -X DELETE http://localhost:8080/api/v1/admin/tenants/tenant-xyz \
  -H "Authorization: Bearer admin-token-xyz" \
  -H "X-Admin-Role: super_admin" \
  -H "X-Confirm-Delete: true"

# Impersonate user (requires audit reason)
curl -X POST http://localhost:8080/api/v1/admin/users/user-xyz/impersonate \
  -H "Authorization: Bearer admin-token-xyz" \
  -H "X-Admin-Role: super_admin" \
  -H "X-Audit-Reason: Customer support ticket #12345"
```

## Header Configuration Patterns

### Service-Level Headers

Applied to ALL methods in a service:

```protobuf
service UserService {
  option (sebuf.http.service_headers) = {
    required_headers: [
      {
        name: "Authorization"
        type: "string"
        required: true
      },
      {
        name: "X-Tenant-ID"
        type: "string"
        required: true
        format: "uuid"
      }
    ]
  };
}
```

### Method-Level Headers

Applied to specific methods only (in addition to service headers):

```protobuf
rpc DeleteTenant(DeleteTenantRequest) returns (DeleteTenantResponse) {
  option (sebuf.http.method_headers) = {
    required_headers: [
      {
        name: "X-Confirm-Delete"
        type: "string"
        required: true
        description: "Must be 'true' to confirm deletion"
      }
    ]
  };
}
```

### No Headers (Public)

Simply omit `service_headers` option:

```protobuf
service PublicService {
  option (sebuf.http.service_config) = { base_path: "/api/v1/public" };
  // No service_headers = public endpoints
}
```

## Generated Files

After running `make generate`:

```
api/
  proto/
    models/
      shared.pb.go              # Shared types (Role, Pagination, etc.)
      tenant.pb.go              # Tenant model
      user.pb.go                # User model
      health.pb.go              # Health/Info models
    services/
      public_service.pb.go      # Public service interface
      public_service_http.pb.go # Public handlers (no header validation)
      user_service.pb.go        # User service interface
      user_service_http.pb.go   # User handlers (user header validation)
      admin_service.pb.go       # Admin service interface
      admin_service_http.pb.go  # Admin handlers (admin + method headers)
docs/
  PublicService.openapi.yaml    # No security schemes
  UserService.openapi.yaml      # User authentication security
  AdminService.openapi.yaml     # Admin authentication + method-specific
```

## Key Concepts

### Authentication Levels

1. **Public**: No headers required, accessible to everyone
2. **User**: Requires user authentication, scoped to tenant
3. **Admin**: Requires admin authentication, cross-tenant access

### Method-Level Header Overrides

Method-level headers are **added** to service-level headers, not replaced. This allows:
- Common headers for all methods (auth)
- Extra headers for sensitive operations (confirmation, audit)

### Multi-Tenancy Pattern

- User endpoints include `X-Tenant-ID` for tenant isolation
- Admin endpoints can access across tenants without tenant header
- Useful for SaaS applications with tenant-scoped data
