package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SebastienMelki/sebuf/examples/multi-service-api/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/multi-service-api/api/proto/services/adminservice"
	"github.com/SebastienMelki/sebuf/examples/multi-service-api/api/proto/services/publicservice"
	"github.com/SebastienMelki/sebuf/examples/multi-service-api/api/proto/services/userservice"
)

// === Public Service Implementation ===

type PublicServiceImpl struct {
	startTime time.Time
}

func NewPublicServiceImpl() *PublicServiceImpl {
	return &PublicServiceImpl{startTime: time.Now()}
}

func (s *PublicServiceImpl) GetHealth(ctx context.Context, req *models.GetHealthRequest) (*models.GetHealthResponse, error) {
	return &models.GetHealthResponse{
		Status:        "healthy",
		Version:       "1.0.0",
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Components: map[string]string{
			"database": "healthy",
			"cache":    "healthy",
			"queue":    "healthy",
		},
	}, nil
}

func (s *PublicServiceImpl) GetAPIInfo(ctx context.Context, req *models.GetAPIInfoRequest) (*models.GetAPIInfoResponse, error) {
	return &models.GetAPIInfoResponse{
		Name:        "Multi-Service API",
		Version:     "1.0.0",
		Description: "Multi-tenant API demonstrating different authentication levels",
		AvailableServices: []string{
			"PublicService - No authentication required",
			"UserService - User authentication required",
			"AdminService - Admin authentication required",
		},
		DocumentationUrl: "https://docs.example.com",
	}, nil
}

// === User Service Implementation ===

type UserServiceImpl struct {
	users map[string]*models.User
}

func NewUserServiceImpl() *UserServiceImpl {
	svc := &UserServiceImpl{
		users: make(map[string]*models.User),
	}

	// Sample users
	now := time.Now().Unix()
	svc.users["user-xyz789"] = &models.User{
		Id:          "user-xyz789",
		TenantId:    "tenant-abc123",
		Email:       "john@example.com",
		Name:        "John Doe",
		AvatarUrl:   "https://example.com/avatar/john.png",
		Role:        models.Role_ROLE_USER,
		Active:      true,
		LastLoginAt: now,
		Audit: &models.AuditInfo{
			CreatedBy: "system",
			UpdatedBy: "john",
			Timestamps: &models.Timestamp{
				CreatedAt: now - 86400*30,
				UpdatedAt: now,
			},
		},
	}
	svc.users["user-admin456"] = &models.User{
		Id:       "user-admin456",
		TenantId: "tenant-abc123",
		Email:    "admin@example.com",
		Name:     "Admin User",
		Role:     models.Role_ROLE_ADMIN,
		Active:   true,
	}

	return svc
}

func (s *UserServiceImpl) GetCurrentUser(ctx context.Context, req *models.GetCurrentUserRequest) (*models.User, error) {
	// In production, extract user from JWT token in context
	return s.users["user-xyz789"], nil
}

func (s *UserServiceImpl) UpdateProfile(ctx context.Context, req *models.UpdateProfileRequest) (*models.User, error) {
	user := s.users["user-xyz789"]
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.AvatarUrl != nil {
		user.AvatarUrl = *req.AvatarUrl
	}
	return user, nil
}

func (s *UserServiceImpl) ListUsers(ctx context.Context, req *models.ListUsersRequest) (*models.ListUsersResponse, error) {
	var users []*models.User
	for _, u := range s.users {
		users = append(users, u)
	}
	return &models.ListUsersResponse{
		Users: users,
		Pagination: &models.Pagination{
			Page:       1,
			PageSize:   20,
			TotalCount: int32(len(users)),
			TotalPages: 1,
		},
	}, nil
}

// === Admin Service Implementation ===

type AdminServiceImpl struct {
	tenants map[string]*models.Tenant
	users   map[string]*models.User
}

func NewAdminServiceImpl() *AdminServiceImpl {
	svc := &AdminServiceImpl{
		tenants: make(map[string]*models.Tenant),
		users:   make(map[string]*models.User),
	}

	// Sample tenants
	now := time.Now().Unix()
	svc.tenants["tenant-abc123"] = &models.Tenant{
		Id:     "tenant-abc123",
		Name:   "Acme Corp",
		Domain: "acme.example.com",
		Active: true,
		Plan:   "enterprise",
		Settings: map[string]string{
			"theme":    "dark",
			"timezone": "America/Los_Angeles",
		},
		Audit: &models.AuditInfo{
			CreatedBy: "system",
			Timestamps: &models.Timestamp{
				CreatedAt: now - 86400*90,
				UpdatedAt: now,
			},
		},
	}
	svc.tenants["tenant-def456"] = &models.Tenant{
		Id:     "tenant-def456",
		Name:   "TechStart Inc",
		Domain: "techstart.example.com",
		Active: true,
		Plan:   "professional",
	}

	// Sample users
	svc.users["user-xyz789"] = &models.User{
		Id:       "user-xyz789",
		TenantId: "tenant-abc123",
		Email:    "john@acme.example.com",
		Name:     "John Doe",
		Role:     models.Role_ROLE_USER,
		Active:   true,
	}
	svc.users["user-admin456"] = &models.User{
		Id:       "user-admin456",
		TenantId: "tenant-abc123",
		Email:    "admin@acme.example.com",
		Name:     "Admin User",
		Role:     models.Role_ROLE_ADMIN,
		Active:   true,
	}

	return svc
}

func (s *AdminServiceImpl) ListTenants(ctx context.Context, req *models.ListTenantsRequest) (*models.ListTenantsResponse, error) {
	var tenants []*models.Tenant
	for _, t := range s.tenants {
		if req.IncludeInactive || t.Active {
			tenants = append(tenants, t)
		}
	}
	return &models.ListTenantsResponse{
		Tenants: tenants,
		Pagination: &models.Pagination{
			Page:       1,
			PageSize:   20,
			TotalCount: int32(len(tenants)),
			TotalPages: 1,
		},
	}, nil
}

func (s *AdminServiceImpl) CreateTenant(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
	tenant := &models.Tenant{
		Id:       fmt.Sprintf("tenant-%d", time.Now().UnixNano()),
		Name:     req.Name,
		Domain:   req.Domain,
		Active:   true,
		Plan:     req.Plan,
		Settings: req.Settings,
		Audit: &models.AuditInfo{
			CreatedBy: "admin",
			Timestamps: &models.Timestamp{
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			},
		},
	}
	s.tenants[tenant.Id] = tenant
	log.Printf("Created tenant: %s (%s)", tenant.Name, tenant.Id)
	return tenant, nil
}

func (s *AdminServiceImpl) DeleteTenant(ctx context.Context, req *models.DeleteTenantRequest) (*models.DeleteTenantResponse, error) {
	if _, exists := s.tenants[req.TenantId]; !exists {
		return nil, fmt.Errorf("tenant not found: %s", req.TenantId)
	}
	delete(s.tenants, req.TenantId)
	log.Printf("Deleted tenant: %s", req.TenantId)
	return &models.DeleteTenantResponse{
		Success: true,
		Message: fmt.Sprintf("Tenant %s deleted successfully", req.TenantId),
	}, nil
}

func (s *AdminServiceImpl) ListAllUsers(ctx context.Context, req *models.ListAllUsersRequest) (*models.ListAllUsersResponse, error) {
	var users []*models.User
	for _, u := range s.users {
		if req.TenantFilter == "" || u.TenantId == req.TenantFilter {
			users = append(users, u)
		}
	}
	return &models.ListAllUsersResponse{
		Users: users,
		Pagination: &models.Pagination{
			Page:       1,
			PageSize:   20,
			TotalCount: int32(len(users)),
			TotalPages: 1,
		},
	}, nil
}

func (s *AdminServiceImpl) ImpersonateUser(ctx context.Context, req *models.ImpersonateUserRequest) (*models.ImpersonateUserResponse, error) {
	user, exists := s.users[req.UserId]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", req.UserId)
	}
	log.Printf("Admin impersonating user: %s (%s)", user.Name, user.Id)
	return &models.ImpersonateUserResponse{
		ImpersonationToken: fmt.Sprintf("impersonate-%s-%d", user.Id, time.Now().Unix()),
		User:               user,
		ExpiresAt:          time.Now().Add(1 * time.Hour).Unix(),
	}, nil
}

func main() {
	mux := http.NewServeMux()

	// Register Public Service (no auth required)
	publicSvc := NewPublicServiceImpl()
	if err := publicservice.RegisterPublicServiceServer(publicSvc, publicservice.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	// Register User Service (user auth required)
	userSvc := NewUserServiceImpl()
	if err := userservice.RegisterUserServiceServer(userSvc, userservice.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	// Register Admin Service (admin auth required)
	adminSvc := NewAdminServiceImpl()
	if err := adminservice.RegisterAdminServiceServer(adminSvc, adminservice.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Multi-Service API Server starting on :8080")
	fmt.Println("")
	fmt.Println("This example demonstrates multiple services with different authentication levels.")
	fmt.Println("")
	fmt.Println("Services:")
	fmt.Println("")
	fmt.Println("  PUBLIC SERVICE (/api/v1/public)")
	fmt.Println("    No authentication required")
	fmt.Println("    - GET /api/v1/public/health    - Health check")
	fmt.Println("    - GET /api/v1/public/info      - API information")
	fmt.Println("")
	fmt.Println("  USER SERVICE (/api/v1/users)")
	fmt.Println("    Required headers: Authorization, X-Tenant-ID")
	fmt.Println("    - GET   /api/v1/users/me       - Get current user")
	fmt.Println("    - PATCH /api/v1/users/me       - Update profile")
	fmt.Println("    - GET   /api/v1/users          - List users in tenant")
	fmt.Println("")
	fmt.Println("  ADMIN SERVICE (/api/v1/admin)")
	fmt.Println("    Required headers: Authorization, X-Admin-Role")
	fmt.Println("    Method-specific headers for sensitive operations")
	fmt.Println("    - GET    /api/v1/admin/tenants                    - List all tenants")
	fmt.Println("    - POST   /api/v1/admin/tenants                    - Create tenant")
	fmt.Println("    - DELETE /api/v1/admin/tenants/{id}               - Delete tenant (+X-Confirm-Delete)")
	fmt.Println("    - GET    /api/v1/admin/users                      - List all users")
	fmt.Println("    - POST   /api/v1/admin/users/{id}/impersonate     - Impersonate user (+X-Audit-Reason)")
	fmt.Println("")
	fmt.Println("Test commands:")
	fmt.Println("  make test-public   - Test public endpoints")
	fmt.Println("  make test-user     - Test user-authenticated endpoints")
	fmt.Println("  make test-admin    - Test admin-authenticated endpoints")
	fmt.Println("")

	log.Fatal(http.ListenAndServe(":8080", mux))
}
