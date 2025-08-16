# Getting Started with sebuf

> Complete tutorial: From protobuf definitions to production-ready HTTP APIs

This guide walks you through building a complete API using all three sebuf tools. You'll create a task management API with authentication, learn best practices, and deploy a working service.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Project Setup](#project-setup)
- [Defining the API](#defining-the-api)
- [Generating Code](#generating-code)
- [Implementing the Service](#implementing-the-service)
- [Testing the API](#testing-the-api)
- [Adding Authentication](#adding-authentication)
- [Documentation and Deployment](#documentation-and-deployment)
- [Next Steps](#next-steps)

## Prerequisites

### Required Tools

```bash
# Go (1.21 or later)
go version

# Protocol Buffers compiler
protoc --version

# Git (for version control)
git --version
```

### Installing sebuf

```bash
# Install all three sebuf tools
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest

# Verify installations
protoc-gen-go-oneof-helper --version
protoc-gen-go-http --version
protoc-gen-openapiv3 --version
```

### Install Dependencies

```bash
# Install standard protobuf tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Install sebuf annotations
go get github.com/SebastienMelki/sebuf/http
```

## Project Setup

### 1. Initialize Project

```bash
# Create project directory
mkdir taskapi
cd taskapi

# Initialize Go module
go mod init github.com/yourorg/taskapi

# Create directory structure
mkdir -p {api,cmd,internal,docs,scripts}
mkdir -p api/{tasks,auth}
mkdir -p internal/{server,auth,storage}
```

### 2. Project Structure

```
taskapi/
├── api/                    # Protobuf definitions
│   ├── tasks/
│   │   └── tasks.proto
│   └── auth/
│       └── auth.proto
├── cmd/                    # Main applications
│   └── server/
│       └── main.go
├── internal/               # Private application code
│   ├── server/
│   ├── auth/
│   └── storage/
├── docs/                   # Generated documentation
├── scripts/                # Build and deployment scripts
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 3. Create Makefile

```makefile
# Makefile
.PHONY: generate build test run clean docs

# Variables
MODULE_NAME := $(shell cat go.mod | head -1 | cut -d' ' -f2)
PROTO_FILES := $(shell find api -name "*.proto")

# Generate all code
generate:
	@echo "Generating code from protobuf definitions..."
	protoc --go_out=. --go_opt=module=$(MODULE_NAME) \
	       --go-oneof-helper_out=. \
	       --go-http_out=. \
	       --openapiv3_out=./docs \
	       --proto_path=. \
	       $(PROTO_FILES)

# Build server
build: generate
	@echo "Building server..."
	go build -o bin/taskapi ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run server locally
run: build
	@echo "Starting server on :8080..."
	./bin/taskapi

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	find . -name "*.pb.go" -delete
	find . -name "*_helpers.pb.go" -delete
	find . -name "*_http*.pb.go" -delete
	rm -rf bin/
	rm -f docs/*.yaml docs/*.json

# Generate documentation
docs: generate
	@echo "Documentation generated in docs/"
	@ls -la docs/

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run
```

## Defining the API

### 1. Authentication Service

Create `api/auth/auth.proto`:

```protobuf
syntax = "proto3";
package auth.v1;

import "sebuf/http/annotations.proto";

option go_package = "github.com/yourorg/taskapi/api/auth;auth";

// Authentication and authorization service
service AuthService {
  option (sebuf.http.service_config) = {
    base_path: "/api/v1/auth"
  };
  
  // Login with email and password
  rpc Login(LoginRequest) returns (LoginResponse) {
    option (sebuf.http.config) = {
      path: "/login"
    };
  };
  
  // Refresh an existing token
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse) {
    option (sebuf.http.config) = {
      path: "/refresh"
    };
  };
  
  // Logout and invalidate token
  rpc Logout(LogoutRequest) returns (LogoutResponse) {
    option (sebuf.http.config) = {
      path: "/logout"
    };
  };
}

// Login request with multiple authentication methods
message LoginRequest {
  oneof auth_method {
    EmailAuth email = 1;
    TokenAuth token = 2;
  }
}

// Email and password authentication
message EmailAuth {
  string email = 1;
  string password = 2;
}

// Token-based authentication (for refresh)
message TokenAuth {
  string token = 1;
}

// Successful login response
message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_in = 3;
  User user = 4;
}

// Token refresh request
message RefreshTokenRequest {
  string refresh_token = 1;
}

// Token refresh response
message RefreshTokenResponse {
  string access_token = 1;
  int64 expires_in = 2;
}

// Logout request
message LogoutRequest {
  string access_token = 1;
}

// Logout response
message LogoutResponse {
  bool success = 1;
}

// User information
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  int64 created_at = 4;
}
```

### 2. Task Management Service

Create `api/tasks/tasks.proto`:

```protobuf
syntax = "proto3";
package tasks.v1;

import "sebuf/http/annotations.proto";

option go_package = "github.com/yourorg/taskapi/api/tasks;tasks";

// Task management service
service TaskService {
  option (sebuf.http.service_config) = {
    base_path: "/api/v1/tasks"
  };
  
  // Create a new task
  rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse) {
    option (sebuf.http.config) = {
      path: ""
    };
  };
  
  // Get a task by ID
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse) {
    option (sebuf.http.config) = {
      path: "/get"
    };
  };
  
  // Update an existing task
  rpc UpdateTask(UpdateTaskRequest) returns (UpdateTaskResponse) {
    option (sebuf.http.config) = {
      path: "/update"
    };
  };
  
  // Delete a task
  rpc DeleteTask(DeleteTaskRequest) returns (DeleteTaskResponse) {
    option (sebuf.http.config) = {
      path: "/delete"
    };
  };
  
  // List tasks with filtering and pagination
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse) {
    option (sebuf.http.config) = {
      path: ""
    };
  };
}

// Task represents a single task
message Task {
  string id = 1;
  string title = 2;
  string description = 3;
  TaskStatus status = 4;
  TaskPriority priority = 5;
  string assignee_id = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
  optional int64 due_date = 9;
  repeated string tags = 10;
  map<string, string> metadata = 11;
}

// Task status enumeration
enum TaskStatus {
  TASK_STATUS_UNSPECIFIED = 0;
  TASK_STATUS_TODO = 1;
  TASK_STATUS_IN_PROGRESS = 2;
  TASK_STATUS_DONE = 3;
  TASK_STATUS_CANCELLED = 4;
}

// Task priority enumeration
enum TaskPriority {
  TASK_PRIORITY_UNSPECIFIED = 0;
  TASK_PRIORITY_LOW = 1;
  TASK_PRIORITY_MEDIUM = 2;
  TASK_PRIORITY_HIGH = 3;
  TASK_PRIORITY_URGENT = 4;
}

// Create task request
message CreateTaskRequest {
  string title = 1;
  string description = 2;
  TaskPriority priority = 3;
  optional string assignee_id = 4;
  optional int64 due_date = 5;
  repeated string tags = 6;
}

// Create task response
message CreateTaskResponse {
  Task task = 1;
}

// Get task request
message GetTaskRequest {
  string id = 1;
}

// Get task response
message GetTaskResponse {
  Task task = 1;
}

// Update task request
message UpdateTaskRequest {
  string id = 1;
  oneof update_field {
    string title = 2;
    string description = 3;
    TaskStatus status = 4;
    TaskPriority priority = 5;
    string assignee_id = 6;
    int64 due_date = 7;
  }
  repeated string tags = 8;
}

// Update task response  
message UpdateTaskResponse {
  Task task = 1;
}

// Delete task request
message DeleteTaskRequest {
  string id = 1;
}

// Delete task response
message DeleteTaskResponse {
  bool success = 1;
}

// List tasks request with filtering
message ListTasksRequest {
  int32 page_size = 1;
  string page_token = 2;
  optional TaskStatus status_filter = 3;
  optional TaskPriority priority_filter = 4;
  optional string assignee_filter = 5;
  repeated string tag_filters = 6;
}

// List tasks response
message ListTasksResponse {
  repeated Task tasks = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

## Generating Code

### 1. Generate All Code

```bash
# Generate protobuf code, HTTP handlers, oneof helpers, and OpenAPI spec
make generate
```

This creates:
- **Protobuf structs**: `api/auth/auth.pb.go`, `api/tasks/tasks.pb.go`
- **HTTP handlers**: `*_http.pb.go`, `*_http_binding.pb.go`, `*_http_config.pb.go`
- **Oneof helpers**: `*_helpers.pb.go`
- **OpenAPI specs**: `docs/auth.yaml`, `docs/tasks.yaml`

### 2. Verify Generated Code

```bash
# Check generated files
find . -name "*.pb.go" | head -10

# Check documentation
ls -la docs/
```

## Implementing the Service

### 1. Main Server

Create `cmd/server/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/yourorg/taskapi/api/auth"
    "github.com/yourorg/taskapi/api/tasks"
    "github.com/yourorg/taskapi/internal/server"
)

func main() {
    // Create HTTP mux
    mux := http.NewServeMux()
    
    // Initialize services
    authService := server.NewAuthService()
    taskService := server.NewTaskService()
    
    // Register HTTP handlers
    if err := auth.RegisterAuthServiceServer(authService, auth.WithMux(mux)); err != nil {
        log.Fatal("Failed to register auth service:", err)
    }
    
    if err := tasks.RegisterTaskServiceServer(taskService, tasks.WithMux(mux)); err != nil {
        log.Fatal("Failed to register task service:", err)
    }
    
    // Add CORS middleware
    handler := corsMiddleware(mux)
    
    // Create server
    srv := &http.Server{
        Addr:         ":8080",
        Handler:      handler,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
    }
    
    // Graceful shutdown
    go func() {
        fmt.Println("Server starting on :8080")
        fmt.Println("API Documentation: http://localhost:8080/docs")
        fmt.Println("Health Check: http://localhost:8080/health")
        
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server failed to start:", err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    fmt.Println("Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }
    
    fmt.Println("Server exited")
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### 2. Auth Service Implementation

Create `internal/server/auth_service.go`:

```go
package server

import (
    "context"
    "fmt"
    "time"

    "github.com/yourorg/taskapi/api/auth"
)

type AuthService struct {
    users map[string]*auth.User // In-memory store for demo
}

func NewAuthService() *AuthService {
    return &AuthService{
        users: map[string]*auth.User{
            "demo@example.com": {
                Id:        "user-001",
                Email:     "demo@example.com",
                Name:      "Demo User",
                CreatedAt: time.Now().Unix(),
            },
        },
    }
}

func (s *AuthService) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
    switch authMethod := req.AuthMethod.(type) {
    case *auth.LoginRequest_Email:
        return s.loginWithEmail(authMethod.Email)
    case *auth.LoginRequest_Token:
        return s.loginWithToken(authMethod.Token)
    default:
        return nil, fmt.Errorf("unknown authentication method")
    }
}

func (s *AuthService) loginWithEmail(emailAuth *auth.EmailAuth) (*auth.LoginResponse, error) {
    user, exists := s.users[emailAuth.Email]
    if !exists {
        return nil, fmt.Errorf("user not found")
    }
    
    // In production, verify password hash
    if emailAuth.Password != "demo123" {
        return nil, fmt.Errorf("invalid password")
    }
    
    return &auth.LoginResponse{
        AccessToken:  generateToken(user.Id),
        RefreshToken: generateRefreshToken(user.Id),
        ExpiresIn:    3600, // 1 hour
        User:         user,
    }, nil
}

func (s *AuthService) loginWithToken(tokenAuth *auth.TokenAuth) (*auth.LoginResponse, error) {
    // In production, validate and parse token
    userId := extractUserIdFromToken(tokenAuth.Token)
    if userId == "" {
        return nil, fmt.Errorf("invalid token")
    }
    
    for _, user := range s.users {
        if user.Id == userId {
            return &auth.LoginResponse{
                AccessToken:  generateToken(user.Id),
                RefreshToken: generateRefreshToken(user.Id),
                ExpiresIn:    3600,
                User:         user,
            }, nil
        }
    }
    
    return nil, fmt.Errorf("user not found")
}

func (s *AuthService) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest) (*auth.RefreshTokenResponse, error) {
    // In production, validate refresh token
    userId := extractUserIdFromRefreshToken(req.RefreshToken)
    if userId == "" {
        return nil, fmt.Errorf("invalid refresh token")
    }
    
    return &auth.RefreshTokenResponse{
        AccessToken: generateToken(userId),
        ExpiresIn:   3600,
    }, nil
}

func (s *AuthService) Logout(ctx context.Context, req *auth.LogoutRequest) (*auth.LogoutResponse, error) {
    // In production, invalidate token
    return &auth.LogoutResponse{
        Success: true,
    }, nil
}

// Helper functions (simplified for demo)
func generateToken(userId string) string {
    return fmt.Sprintf("access_token_%s_%d", userId, time.Now().Unix())
}

func generateRefreshToken(userId string) string {
    return fmt.Sprintf("refresh_token_%s_%d", userId, time.Now().Unix())
}

func extractUserIdFromToken(token string) string {
    // In production, parse JWT or validate token
    if token == "demo_token" {
        return "user-001"
    }
    return ""
}

func extractUserIdFromRefreshToken(token string) string {
    // In production, parse JWT or validate refresh token
    if token != "" {
        return "user-001"
    }
    return ""
}
```

### 3. Task Service Implementation

Create `internal/server/task_service.go`:

```go
package server

import (
    "context"
    "fmt"
    "strconv"
    "time"

    "github.com/yourorg/taskapi/api/tasks"
)

type TaskService struct {
    tasks   map[string]*tasks.Task
    nextId  int
}

func NewTaskService() *TaskService {
    return &TaskService{
        tasks:  make(map[string]*tasks.Task),
        nextId: 1,
    }
}

func (s *TaskService) CreateTask(ctx context.Context, req *tasks.CreateTaskRequest) (*tasks.CreateTaskResponse, error) {
    id := strconv.Itoa(s.nextId)
    s.nextId++
    
    task := &tasks.Task{
        Id:          id,
        Title:       req.Title,
        Description: req.Description,
        Status:      tasks.TaskStatus_TASK_STATUS_TODO,
        Priority:    req.Priority,
        CreatedAt:   time.Now().Unix(),
        UpdatedAt:   time.Now().Unix(),
        Tags:        req.Tags,
        Metadata:    make(map[string]string),
    }
    
    if req.AssigneeId != nil {
        task.AssigneeId = *req.AssigneeId
    }
    
    if req.DueDate != nil {
        task.DueDate = req.DueDate
    }
    
    s.tasks[id] = task
    
    return &tasks.CreateTaskResponse{
        Task: task,
    }, nil
}

func (s *TaskService) GetTask(ctx context.Context, req *tasks.GetTaskRequest) (*tasks.GetTaskResponse, error) {
    task, exists := s.tasks[req.Id]
    if !exists {
        return nil, fmt.Errorf("task not found: %s", req.Id)
    }
    
    return &tasks.GetTaskResponse{
        Task: task,
    }, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, req *tasks.UpdateTaskRequest) (*tasks.UpdateTaskResponse, error) {
    task, exists := s.tasks[req.Id]
    if !exists {
        return nil, fmt.Errorf("task not found: %s", req.Id)
    }
    
    // Update fields based on oneof field
    switch updateField := req.UpdateField.(type) {
    case *tasks.UpdateTaskRequest_Title:
        task.Title = updateField.Title
    case *tasks.UpdateTaskRequest_Description:
        task.Description = updateField.Description
    case *tasks.UpdateTaskRequest_Status:
        task.Status = updateField.Status
    case *tasks.UpdateTaskRequest_Priority:
        task.Priority = updateField.Priority
    case *tasks.UpdateTaskRequest_AssigneeId:
        task.AssigneeId = updateField.AssigneeId
    case *tasks.UpdateTaskRequest_DueDate:
        task.DueDate = &updateField.DueDate
    }
    
    if len(req.Tags) > 0 {
        task.Tags = req.Tags
    }
    
    task.UpdatedAt = time.Now().Unix()
    
    return &tasks.UpdateTaskResponse{
        Task: task,
    }, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, req *tasks.DeleteTaskRequest) (*tasks.DeleteTaskResponse, error) {
    _, exists := s.tasks[req.Id]
    if !exists {
        return nil, fmt.Errorf("task not found: %s", req.Id)
    }
    
    delete(s.tasks, req.Id)
    
    return &tasks.DeleteTaskResponse{
        Success: true,
    }, nil
}

func (s *TaskService) ListTasks(ctx context.Context, req *tasks.ListTasksRequest) (*tasks.ListTasksResponse, error) {
    var filteredTasks []*tasks.Task
    
    for _, task := range s.tasks {
        // Apply filters
        if req.StatusFilter != nil && task.Status != *req.StatusFilter {
            continue
        }
        
        if req.PriorityFilter != nil && task.Priority != *req.PriorityFilter {
            continue
        }
        
        if req.AssigneeFilter != nil && task.AssigneeId != *req.AssigneeFilter {
            continue
        }
        
        // Tag filter (task must have all specified tags)
        if len(req.TagFilters) > 0 {
            hasAllTags := true
            for _, filterTag := range req.TagFilters {
                found := false
                for _, taskTag := range task.Tags {
                    if taskTag == filterTag {
                        found = true
                        break
                    }
                }
                if !found {
                    hasAllTags = false
                    break
                }
            }
            if !hasAllTags {
                continue
            }
        }
        
        filteredTasks = append(filteredTasks, task)
    }
    
    return &tasks.ListTasksResponse{
        Tasks:      filteredTasks,
        TotalCount: int32(len(filteredTasks)),
    }, nil
}
```

## Testing the API

### 1. Build and Run

```bash
# Build the server
make build

# Run the server
make run
```

### 2. Test Authentication

```bash
# Test login with email (using generated oneof helper)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": {
      "email": "demo@example.com",
      "password": "demo123"
    }
  }'

# Expected response:
# {
#   "accessToken": "access_token_user-001_1234567890",
#   "refreshToken": "refresh_token_user-001_1234567890", 
#   "expiresIn": "3600",
#   "user": {
#     "id": "user-001",
#     "email": "demo@example.com",
#     "name": "Demo User",
#     "createdAt": "1234567890"
#   }
# }
```

### 3. Test Task Management

```bash
# Create a task
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Complete documentation",
    "description": "Write comprehensive API documentation",
    "priority": "TASK_PRIORITY_HIGH",
    "tags": ["documentation", "api"]
  }'

# Get the task (use ID from create response)
curl -X POST http://localhost:8080/api/v1/tasks/get \
  -H "Content-Type: application/json" \
  -d '{"id": "1"}'

# Update task status
curl -X POST http://localhost:8080/api/v1/tasks/update \
  -H "Content-Type: application/json" \
  -d '{
    "id": "1",
    "status": "TASK_STATUS_IN_PROGRESS"
  }'

# List all tasks
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "pageSize": 10
  }'
```

### 4. Test with Different Content Types

```bash
# Test with binary protobuf (you'll need protobuf tools)
# Create request.pb file with protobuf binary data
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/x-protobuf" \
  --data-binary @request.pb
```

## Adding Authentication

### 1. Create Middleware

Create `internal/server/middleware.go`:

```go
package server

import (
    "context"
    "net/http"
    "strings"
)

type contextKey string

const userContextKey contextKey = "user"

// AuthMiddleware validates authentication tokens
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for login endpoint
        if strings.HasSuffix(r.URL.Path, "/login") {
            next.ServeHTTP(w, r)
            return
        }
        
        // Extract token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Missing authorization header", http.StatusUnauthorized)
            return
        }
        
        // Validate Bearer token format
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
            return
        }
        
        token := parts[1]
        
        // Validate token (simplified for demo)
        userId := extractUserIdFromToken(token)
        if userId == "" {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        // Add user ID to context
        ctx := context.WithValue(r.Context(), userContextKey, userId)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// GetUserFromContext extracts user ID from request context
func GetUserFromContext(ctx context.Context) string {
    if userId, ok := ctx.Value(userContextKey).(string); ok {
        return userId
    }
    return ""
}
```

### 2. Apply Middleware

Update `cmd/server/main.go`:

```go
// Add authentication middleware
authHandler := server.AuthMiddleware(mux)
handler := corsMiddleware(authHandler)
```

### 3. Update Service Methods

Update task service to use authenticated user:

```go
func (s *TaskService) CreateTask(ctx context.Context, req *tasks.CreateTaskRequest) (*tasks.CreateTaskResponse, error) {
    // Get authenticated user
    userId := GetUserFromContext(ctx)
    if userId == "" {
        return nil, fmt.Errorf("unauthorized")
    }
    
    // Rest of the implementation...
}
```

## Documentation and Deployment

### 1. Serve OpenAPI Documentation

Add documentation endpoint to `cmd/server/main.go`:

```go
// Serve OpenAPI documentation
mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "docs/tasks.yaml")
})

// Health check endpoint
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
})
```

### 2. Create Docker Configuration

Create `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/bin/taskapi .
COPY --from=builder /app/docs ./docs

EXPOSE 8080
CMD ["./taskapi"]
```

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  taskapi:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
    restart: unless-stopped
    
  docs:
    image: swaggerapi/swagger-ui
    ports:
      - "8081:8080"
    environment:
      - SWAGGER_JSON=/app/tasks.yaml
    volumes:
      - ./docs:/app
    depends_on:
      - taskapi
```

### 3. Add CI/CD Pipeline

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
        
    - name: Install Protocol Buffers
      run: |
        sudo apt-get update
        sudo apt-get install -y protobuf-compiler
        
    - name: Install sebuf tools
      run: |
        go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
        go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-http@latest
        go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-openapiv3@latest
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        
    - name: Generate code
      run: make generate
      
    - name: Run tests
      run: make test
      
    - name: Build
      run: make build
      
    - name: Validate OpenAPI specs
      run: |
        npm install -g @apidevtools/swagger-cli
        swagger-cli validate docs/*.yaml
```

## Next Steps

### 1. Production Enhancements

**Database Integration:**
```go
// Replace in-memory storage with database
type TaskRepository interface {
    Create(ctx context.Context, task *tasks.Task) error
    Get(ctx context.Context, id string) (*tasks.Task, error)
    Update(ctx context.Context, task *tasks.Task) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filters *tasks.ListTasksRequest) ([]*tasks.Task, error)
}
```

**Proper Authentication:**
```bash
# Add JWT token support
go get github.com/golang-jwt/jwt/v5

# Add password hashing
go get golang.org/x/crypto/bcrypt
```

**Configuration Management:**
```bash
# Add configuration management
go get github.com/spf13/viper

# Add environment variable support
go get github.com/joho/godotenv
```

### 2. Advanced Features

**Rate Limiting:**
```go
// Add rate limiting middleware
go get golang.org/x/time/rate
```

**Metrics and Monitoring:**
```bash
# Add Prometheus metrics
go get github.com/prometheus/client_golang

# Add OpenTelemetry tracing
go get go.opentelemetry.io/otel
```

**API Versioning:**
```protobuf
// Create v2 API
package tasks.v2;

// Maintain backward compatibility
import "tasks/v1/tasks.proto";
```

### 3. Client Generation

**Generate TypeScript Client:**
```bash
# From OpenAPI spec
openapi-generator-cli generate \
  -i docs/tasks.yaml \
  -g typescript-fetch \
  -o clients/typescript

# From protobuf (alternative)
npm install -g grpc-web
protoc --js_out=import_style=commonjs:. \
       --grpc-web_out=import_style=typescript,mode=grpcwebtext:. \
       api/tasks/tasks.proto
```

**Generate Mobile Clients:**
```bash
# iOS Swift
openapi-generator-cli generate -i docs/tasks.yaml -g swift5 -o clients/ios

# Android Kotlin
openapi-generator-cli generate -i docs/tasks.yaml -g kotlin -o clients/android
```

### 4. Testing Strategy

**Integration Tests:**
```go
// Create integration test suite
func TestTaskAPIIntegration(t *testing.T) {
    // Start test server
    server := startTestServer()
    defer server.Close()
    
    // Test complete workflows
    testCreateAndRetrieveTask(t, server.URL)
    testAuthenticationFlow(t, server.URL)
    testTaskLifecycle(t, server.URL)
}
```

**Load Testing:**
```bash
# Install k6 for load testing
brew install k6

# Create load test script using generated OpenAPI spec
k6 run --vus 10 --duration 30s load_test.js
```

### 5. Deployment Options

**Kubernetes:**
```yaml
# Create Kubernetes manifests
apiVersion: apps/v1
kind: Deployment
metadata:
  name: taskapi
spec:
  replicas: 3
  selector:
    matchLabels:
      app: taskapi
  template:
    spec:
      containers:
      - name: taskapi
        image: yourorg/taskapi:latest
        ports:
        - containerPort: 8080
```

**Serverless:**
```bash
# Deploy to AWS Lambda
go get github.com/aws/aws-lambda-go/lambda

# Deploy to Google Cloud Functions
gcloud functions deploy taskapi --runtime go121 --trigger-http
```

### 6. Monitoring and Observability

**Structured Logging:**
```go
// Add structured logging
go get github.com/sirupsen/logrus

// Add request tracing
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        next.ServeHTTP(w, r)
        
        log.WithFields(logrus.Fields{
            "method":   r.Method,
            "path":     r.URL.Path,
            "duration": time.Since(start),
        }).Info("Request completed")
    })
}
```

**Health Checks:**
```go
// Enhanced health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
        "version":   version,
        "checks": map[string]string{
            "database": checkDatabase(),
            "cache":    checkCache(),
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}
```

---

**Congratulations!** You've built a complete HTTP API using sebuf. Your API now has:

- ✅ Type-safe protobuf definitions
- ✅ Generated HTTP handlers with JSON and binary support  
- ✅ Convenience constructors for complex types
- ✅ Comprehensive OpenAPI documentation
- ✅ Authentication and middleware support
- ✅ Production-ready structure

**Next Steps:**
- Explore [Framework Integration Examples](./examples/frameworks/)
- Learn [Advanced Patterns](./examples/patterns/)
- Check out [Deployment Guides](./examples/deployment/)
- Join the [Community Discussions](https://github.com/SebastienMelki/sebuf/discussions)