# Simple CRUD API Example

A complete task management API demonstrating all sebuf tools working together.

## 🎯 What You'll Learn

- ✅ Defining protobuf services with HTTP annotations
- ✅ Using oneof fields for flexible request types
- ✅ Generating HTTP handlers automatically
- ✅ Creating comprehensive OpenAPI documentation
- ✅ Testing the complete API

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client        │───▶│  HTTP Handlers   │───▶│  Business Logic │
│  (JSON/Binary)  │    │  (Generated)     │    │  (Your Code)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │  OpenAPI Docs    │
                       │  (Generated)     │
                       └──────────────────┘
```

## 🚀 Quick Start

```bash
# Run the example
cd docs/examples/basic/crud-api
make run

# Test the API
make test

# View API docs
open http://localhost:8080/docs
```

## 📁 Project Structure

```
crud-api/
├── api/
│   └── tasks.proto          # Service definitions
├── cmd/
│   └── server/
│       └── main.go          # HTTP server
├── internal/
│   └── service/
│       └── tasks.go         # Business logic
├── tests/
│   └── api_test.go          # Integration tests
├── docs/                    # Generated OpenAPI specs
├── Makefile                 # Build automation
├── go.mod
└── README.md               # This file
```

## 📋 API Definition

### Service Definition (`api/tasks.proto`)

```protobuf
syntax = "proto3";
package tasks.v1;

import "sebuf/http/annotations.proto";

option go_package = "github.com/example/crud-api/tasks;tasks";

// Task management service with full CRUD operations
service TaskService {
  option (sebuf.http.service_config) = {
    base_path: "/api/v1/tasks"
  };
  
  // Create a new task
  rpc CreateTask(CreateTaskRequest) returns (Task) {
    option (sebuf.http.config) = { path: "" };
  };
  
  // Get a task by ID
  rpc GetTask(GetTaskRequest) returns (Task) {
    option (sebuf.http.config) = { path: "/get" };
  };
  
  // Update an existing task
  rpc UpdateTask(UpdateTaskRequest) returns (Task) {
    option (sebuf.http.config) = { path: "/update" };
  };
  
  // Delete a task
  rpc DeleteTask(DeleteTaskRequest) returns (DeleteTaskResponse) {
    option (sebuf.http.config) = { path: "/delete" };
  };
  
  // List tasks with filters
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse) {
    option (sebuf.http.config) = { path: "/list" };
  };
}

// Core task model
message Task {
  string id = 1;
  string title = 2;
  string description = 3;
  TaskStatus status = 4;
  TaskPriority priority = 5;
  int64 created_at = 6;
  int64 updated_at = 7;
  repeated string tags = 8;
}

// Task status enumeration
enum TaskStatus {
  TASK_STATUS_UNSPECIFIED = 0;
  TASK_STATUS_TODO = 1;
  TASK_STATUS_IN_PROGRESS = 2;
  TASK_STATUS_DONE = 3;
}

// Task priority enumeration
enum TaskPriority {
  TASK_PRIORITY_UNSPECIFIED = 0;
  TASK_PRIORITY_LOW = 1;
  TASK_PRIORITY_MEDIUM = 2;
  TASK_PRIORITY_HIGH = 3;
}

// Request messages
message CreateTaskRequest {
  string title = 1;
  string description = 2;
  TaskPriority priority = 3;
  repeated string tags = 4;
}

message GetTaskRequest {
  string id = 1;
}

message UpdateTaskRequest {
  string id = 1;
  // Use oneof for flexible updates
  oneof update_field {
    string title = 2;
    string description = 3;
    TaskStatus status = 4;
    TaskPriority priority = 5;
  }
  repeated string tags = 6;
}

message DeleteTaskRequest {
  string id = 1;
}

message ListTasksRequest {
  optional TaskStatus status_filter = 1;
  optional TaskPriority priority_filter = 2;
  int32 limit = 3;
}

// Response messages
message DeleteTaskResponse {
  bool success = 1;
}

message ListTasksResponse {
  repeated Task tasks = 1;
  int32 total_count = 2;
}
```

## 🔧 Implementation

### HTTP Server (`cmd/server/main.go`)

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    
    "github.com/example/crud-api/tasks"
    "github.com/example/crud-api/internal/service"
)

func main() {
    // Create service implementation
    taskService := service.NewTaskService()
    
    // Setup HTTP handlers
    mux := http.NewServeMux()
    
    // Register generated HTTP handlers
    err := tasks.RegisterTaskServiceServer(taskService, tasks.WithMux(mux))
    if err != nil {
        log.Fatal("Failed to register task service:", err)
    }
    
    // Add health check
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status": "healthy"}`))
    })
    
    // Serve OpenAPI documentation
    mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "docs/tasks.yaml")
    })
    
    // Start server
    fmt.Println("🚀 Task API Server starting on :8080")
    fmt.Println("📚 API Documentation: http://localhost:8080/docs")
    fmt.Println("❤️  Health Check: http://localhost:8080/health")
    
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### Business Logic (`internal/service/tasks.go`)

```go
package service

import (
    "context"
    "fmt"
    "strconv"
    "sync"
    "time"
    
    "github.com/example/crud-api/tasks"
)

// TaskService implements the TaskService interface
type TaskService struct {
    mu     sync.RWMutex
    tasks  map[string]*tasks.Task
    nextID int
}

// NewTaskService creates a new task service instance
func NewTaskService() *TaskService {
    return &TaskService{
        tasks:  make(map[string]*tasks.Task),
        nextID: 1,
    }
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(ctx context.Context, req *tasks.CreateTaskRequest) (*tasks.Task, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Generate unique ID
    id := strconv.Itoa(s.nextID)
    s.nextID++
    
    // Create task
    task := &tasks.Task{
        Id:          id,
        Title:       req.Title,
        Description: req.Description,
        Status:      tasks.TaskStatus_TASK_STATUS_TODO,
        Priority:    req.Priority,
        CreatedAt:   time.Now().Unix(),
        UpdatedAt:   time.Now().Unix(),
        Tags:        req.Tags,
    }
    
    // Store task
    s.tasks[id] = task
    
    return task, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(ctx context.Context, req *tasks.GetTaskRequest) (*tasks.Task, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    task, exists := s.tasks[req.Id]
    if !exists {
        return nil, fmt.Errorf("task not found: %s", req.Id)
    }
    
    return task, nil
}

// UpdateTask updates an existing task
func (s *TaskService) UpdateTask(ctx context.Context, req *tasks.UpdateTaskRequest) (*tasks.Task, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    task, exists := s.tasks[req.Id]
    if !exists {
        return nil, fmt.Errorf("task not found: %s", req.Id)
    }
    
    // Update based on oneof field (demonstrates oneof helper usage)
    switch field := req.UpdateField.(type) {
    case *tasks.UpdateTaskRequest_Title:
        task.Title = field.Title
    case *tasks.UpdateTaskRequest_Description:
        task.Description = field.Description
    case *tasks.UpdateTaskRequest_Status:
        task.Status = field.Status
    case *tasks.UpdateTaskRequest_Priority:
        task.Priority = field.Priority
    }
    
    // Update tags if provided
    if len(req.Tags) > 0 {
        task.Tags = req.Tags
    }
    
    task.UpdatedAt = time.Now().Unix()
    
    return task, nil
}

// DeleteTask deletes a task
func (s *TaskService) DeleteTask(ctx context.Context, req *tasks.DeleteTaskRequest) (*tasks.DeleteTaskResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    _, exists := s.tasks[req.Id]
    if !exists {
        return nil, fmt.Errorf("task not found: %s", req.Id)
    }
    
    delete(s.tasks, req.Id)
    
    return &tasks.DeleteTaskResponse{Success: true}, nil
}

// ListTasks lists tasks with optional filters
func (s *TaskService) ListTasks(ctx context.Context, req *tasks.ListTasksRequest) (*tasks.ListTasksResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    var filtered []*tasks.Task
    
    for _, task := range s.tasks {
        // Apply status filter
        if req.StatusFilter != nil && task.Status != *req.StatusFilter {
            continue
        }
        
        // Apply priority filter
        if req.PriorityFilter != nil && task.Priority != *req.PriorityFilter {
            continue
        }
        
        filtered = append(filtered, task)
    }
    
    // Apply limit
    if req.Limit > 0 && len(filtered) > int(req.Limit) {
        filtered = filtered[:req.Limit]
    }
    
    return &tasks.ListTasksResponse{
        Tasks:      filtered,
        TotalCount: int32(len(filtered)),
    }, nil
}
```

## 🧪 Testing

### Integration Tests (`tests/api_test.go`)

```go
package tests

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/example/crud-api/tasks"
    "github.com/example/crud-api/internal/service"
)

func setupTestServer() *httptest.Server {
    taskService := service.NewTaskService()
    mux := http.NewServeMux()
    
    tasks.RegisterTaskServiceServer(taskService, tasks.WithMux(mux))
    
    return httptest.NewServer(mux)
}

func TestTaskCRUD(t *testing.T) {
    server := setupTestServer()
    defer server.Close()
    
    // Test Create Task
    createReq := map[string]interface{}{
        "title":       "Test Task",
        "description": "A test task",
        "priority":    "TASK_PRIORITY_HIGH",
        "tags":        []string{"test", "api"},
    }
    
    resp := makeRequest(t, server.URL+"/api/v1/tasks", "POST", createReq)
    
    var task map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&task)
    
    taskID := task["id"].(string)
    
    if task["title"] != "Test Task" {
        t.Errorf("Expected title 'Test Task', got %v", task["title"])
    }
    
    // Test Get Task
    getReq := map[string]interface{}{"id": taskID}
    resp = makeRequest(t, server.URL+"/api/v1/tasks/get", "POST", getReq)
    
    json.NewDecoder(resp.Body).Decode(&task)
    
    if task["id"] != taskID {
        t.Errorf("Expected task ID %s, got %v", taskID, task["id"])
    }
    
    // Test Update Task (demonstrates oneof helper usage)
    updateReq := map[string]interface{}{
        "id":     taskID,
        "status": "TASK_STATUS_DONE",
    }
    
    resp = makeRequest(t, server.URL+"/api/v1/tasks/update", "POST", updateReq)
    
    json.NewDecoder(resp.Body).Decode(&task)
    
    if task["status"] != "TASK_STATUS_DONE" {
        t.Errorf("Expected status TASK_STATUS_DONE, got %v", task["status"])
    }
    
    // Test List Tasks
    listReq := map[string]interface{}{
        "statusFilter": "TASK_STATUS_DONE",
    }
    
    resp = makeRequest(t, server.URL+"/api/v1/tasks/list", "POST", listReq)
    
    var listResp map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&listResp)
    
    tasks := listResp["tasks"].([]interface{})
    if len(tasks) != 1 {
        t.Errorf("Expected 1 task, got %d", len(tasks))
    }
    
    // Test Delete Task
    deleteReq := map[string]interface{}{"id": taskID}
    resp = makeRequest(t, server.URL+"/api/v1/tasks/delete", "POST", deleteReq)
    
    var deleteResp map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&deleteResp)
    
    if deleteResp["success"] != true {
        t.Error("Expected successful deletion")
    }
}

func makeRequest(t *testing.T, url, method string, body interface{}) *http.Response {
    jsonBody, _ := json.Marshal(body)
    
    req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
    if err != nil {
        t.Fatal(err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        t.Fatal(err)
    }
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
    
    return resp
}
```

## 🔨 Build Configuration

### Makefile

```makefile
MODULE_NAME := github.com/example/crud-api

.PHONY: generate build run test clean

# Generate all code from protobuf
generate:
	@echo "🔄 Generating code..."
	protoc --go_out=. --go_opt=module=$(MODULE_NAME) \
	       --go-oneof-helper_out=. \
	       --go-http_out=. \
	       --openapiv3_out=./docs \
	       --proto_path=. \
	       api/tasks.proto

# Build the server
build: generate
	@echo "🔨 Building server..."
	go build -o bin/server ./cmd/server

# Run the server
run: build
	@echo "🚀 Starting server..."
	./bin/server

# Run tests
test: generate
	@echo "🧪 Running tests..."
	go test ./tests/...

# Clean generated files
clean:
	@echo "🧹 Cleaning..."
	rm -f api/*.pb.go
	rm -f api/*_helpers.pb.go
	rm -f api/*_http*.pb.go
	rm -f docs/*.yaml
	rm -rf bin/

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod tidy
```

### Go Module (`go.mod`)

```go
module github.com/example/crud-api

go 1.21

require (
    github.com/SebastienMelki/sebuf v0.1.0
    google.golang.org/protobuf v1.36.7
)
```

## 🚀 Usage Examples

### 1. Start the Server

```bash
make run
```

### 2. Create a Task

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Learn sebuf",
    "description": "Complete the sebuf tutorial",
    "priority": "TASK_PRIORITY_HIGH",
    "tags": ["learning", "protobuf"]
  }'
```

### 3. Get a Task

```bash
curl -X POST http://localhost:8080/api/v1/tasks/get \
  -H "Content-Type: application/json" \
  -d '{"id": "1"}'
```

### 4. Update Task Status (using oneof helper)

```bash
curl -X POST http://localhost:8080/api/v1/tasks/update \
  -H "Content-Type: application/json" \
  -d '{
    "id": "1",
    "status": "TASK_STATUS_DONE"
  }'
```

### 5. List Tasks with Filter

```bash
curl -X POST http://localhost:8080/api/v1/tasks/list \
  -H "Content-Type: application/json" \
  -d '{
    "statusFilter": "TASK_STATUS_DONE",
    "limit": 10
  }'
```

### 6. Delete a Task

```bash
curl -X POST http://localhost:8080/api/v1/tasks/delete \
  -H "Content-Type: application/json" \
  -d '{"id": "1"}'
```

## 📚 Generated Documentation

After running `make generate`, you can view the auto-generated OpenAPI documentation:

```bash
# View the raw OpenAPI spec
cat docs/tasks.yaml

# Serve with Swagger UI
docker run -p 8081:8080 -v $(pwd)/docs:/app swaggerapi/swagger-ui

# Open in browser
open http://localhost:8081
```

## 🎓 Learning Points

This example demonstrates:

1. **Service Definition**: Clean protobuf service with HTTP annotations
2. **Oneof Usage**: Flexible update operations using oneof fields
3. **Generated Code**: Automatic HTTP handlers with JSON support
4. **Type Safety**: Full protobuf type checking throughout
5. **Documentation**: Auto-generated OpenAPI specifications
6. **Testing**: Integration testing of the complete API

## 🚀 Next Steps

- **Add authentication**: Integrate with the [auth service example](../auth-service/)
- **Add persistence**: Replace in-memory storage with database
- **Add middleware**: Logging, rate limiting, etc.
- **Deploy**: Try the [Docker deployment example](../../deployment/docker/)

## 🤝 Contributing

Found an issue or have an improvement? We'd love your contribution!

1. Fork the repository
2. Make your changes
3. Add tests
4. Submit a pull request

See the [Contributing Guide](../../../CONTRIBUTING.md) for more details.