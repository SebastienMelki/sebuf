// Package main demonstrates the WithErrorHandler feature of sebuf.
//
// This example shows all the different ways you can use the ErrorHandler:
//  1. Logging errors without modifying the response
//  2. Adding custom headers to error responses
//  3. Setting custom HTTP status codes
//  4. Returning custom error response bodies
//  5. Writing directly to the response (full control)
//
// Run: go run main.go
// Then test with curl commands shown at startup.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/SebastienMelki/sebuf/examples/error-handler/api/proto/models"
	"github.com/SebastienMelki/sebuf/examples/error-handler/api/proto/services"
)

// ErrNotFound is a custom error for not found resources.
type ErrNotFound struct {
	ResourceType string
	ResourceID   string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.ResourceType, e.ResourceID)
}

// UserServiceImpl implements the UserService.
type UserServiceImpl struct {
	mu     sync.RWMutex
	users  map[string]*models.User
	nextID int
}

// NewUserServiceImpl creates a new UserServiceImpl.
func NewUserServiceImpl() *UserServiceImpl {
	return &UserServiceImpl{
		users:  make(map[string]*models.User),
		nextID: 1,
	}
}

// CreateUser creates a new user.
func (s *UserServiceImpl) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := &models.User{
		Id:        fmt.Sprintf("user-%d", s.nextID),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now().Unix(),
	}
	s.nextID++
	s.users[user.Id] = user
	return user, nil
}

// GetUser retrieves a user by ID.
func (s *UserServiceImpl) GetUser(ctx context.Context, req *models.GetUserRequest) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[req.Id]
	if !exists {
		return nil, &ErrNotFound{ResourceType: "user", ResourceID: req.Id}
	}
	return user, nil
}

// UpdateUser updates an existing user.
func (s *UserServiceImpl) UpdateUser(ctx context.Context, req *models.UpdateUserRequest) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[req.Id]
	if !exists {
		return nil, &ErrNotFound{ResourceType: "user", ResourceID: req.Id}
	}

	user.Name = req.Name
	user.Email = req.Email
	return user, nil
}

// DeleteUser deletes a user.
func (s *UserServiceImpl) DeleteUser(ctx context.Context, req *models.DeleteUserRequest) (*models.DeleteUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[req.Id]; !exists {
		return nil, &ErrNotFound{ResourceType: "user", ResourceID: req.Id}
	}

	delete(s.users, req.Id)
	return &models.DeleteUserResponse{Success: true}, nil
}

// Example 1: LoggingErrorHandler - Just logs errors without modifying the response.
// The framework handles the response normally.
func LoggingErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = "unknown"
	}
	log.Printf("[%s] %s %s - Error: %v", requestID, r.Method, r.URL.Path, err)
	return nil // Use default response
}

// Example 2: HeaderErrorHandler - Adds custom headers to error responses.
// Useful for adding correlation IDs, error codes, etc.
func HeaderErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	// Add custom headers to all error responses
	w.Header().Set("X-Error-ID", uuid.NewString())
	w.Header().Set("X-Error-Timestamp", time.Now().UTC().Format(time.RFC3339))

	// Check error type and add specific headers
	var notFound *ErrNotFound
	if errors.As(err, &notFound) {
		w.Header().Set("X-Error-Type", "not_found")
		w.Header().Set("X-Resource-Type", notFound.ResourceType)
	}

	return nil // Use default response with custom headers
}

// Example 3: StatusCodeErrorHandler - Sets custom HTTP status codes for specific errors.
// Useful when you want different status codes than the defaults.
func StatusCodeErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	// Check for not found errors - use 404
	var notFound *ErrNotFound
	if errors.As(err, &notFound) {
		w.WriteHeader(http.StatusNotFound)
		return nil // Framework will marshal default error with our status code
	}

	// Let framework handle other errors with default status codes
	return nil
}

// Example 4: CustomBodyErrorHandler - Returns custom error response bodies.
// Useful for standardized API error formats.
// Uses the proto messages defined in proto/models/errors.proto
func CustomBodyErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	// Check for not found errors - return NotFoundError proto message
	var notFound *ErrNotFound
	if errors.As(err, &notFound) {
		w.WriteHeader(http.StatusNotFound)
		return &models.NotFoundError{
			ResourceType: notFound.ResourceType,
			ResourceId:   notFound.ResourceID,
			Message:      err.Error(),
		}
	}

	// For other errors (including validation), let framework handle or return custom error
	return &models.CustomError{
		Code:      "INTERNAL_ERROR",
		Message:   err.Error(),
		RequestId: requestID,
		Timestamp: time.Now().Unix(),
	}
}

// Example 5: FullControlErrorHandler - Writes directly to the response.
// When you call w.Write(), the framework does no further processing.
// Useful when you need complete control over the response format.
func FullControlErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)

	// Determine status code based on error type
	statusCode := http.StatusInternalServerError
	errorType := "internal_error"

	var notFound *ErrNotFound
	if errors.As(err, &notFound) {
		statusCode = http.StatusNotFound
		errorType = "not_found"
	}

	w.WriteHeader(statusCode)

	// Write custom JSON response directly
	response := fmt.Sprintf(`{"error":{"type":"%s","message":"%s","request_id":"%s","timestamp":%d}}`,
		errorType, err.Error(), requestID, time.Now().Unix())

	_, _ = w.Write([]byte(response))

	// Return nil - the framework won't write anything since we already called Write()
	return nil
}

// CombinedErrorHandler combines multiple behaviors: logging, custom headers,
// custom status codes, and custom response bodies.
// Uses proto messages from proto/models/errors.proto for custom responses.
func CombinedErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	// Generate or get request ID
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	// Always log the error
	log.Printf("[%s] %s %s - Error: %v", requestID, r.Method, r.URL.Path, err)

	// Always add error tracking headers
	w.Header().Set("X-Error-ID", uuid.NewString())
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("X-Error-Timestamp", time.Now().UTC().Format(time.RFC3339))

	// Handle not found errors - return NotFoundError proto message from errors.proto
	var notFound *ErrNotFound
	if errors.As(err, &notFound) {
		w.Header().Set("X-Error-Type", "not_found")
		w.WriteHeader(http.StatusNotFound)
		return &models.NotFoundError{
			ResourceType: notFound.ResourceType,
			ResourceId:   notFound.ResourceID,
			Message:      err.Error(),
		}
	}

	// For other service errors, return CustomError proto message from errors.proto
	return &models.CustomError{
		Code:      "INTERNAL_ERROR",
		Message:   err.Error(),
		RequestId: requestID,
		Timestamp: time.Now().Unix(),
	}
}

func main() {
	service := NewUserServiceImpl()
	mux := http.NewServeMux()

	// Choose which error handler to use by uncommenting one:

	// Option 1: Just logging (default response)
	// errorHandler := LoggingErrorHandler

	// Option 2: Add custom headers (default response with headers)
	// errorHandler := HeaderErrorHandler

	// Option 3: Custom status codes (default response with custom status)
	// errorHandler := StatusCodeErrorHandler

	// Option 4: Custom response bodies
	// errorHandler := CustomBodyErrorHandler

	// Option 5: Full control (write directly to response)
	// errorHandler := FullControlErrorHandler

	// Option 6: Combined (logging + headers + custom responses)
	errorHandler := CombinedErrorHandler

	// Register with custom error handler
	if err := services.RegisterUserServiceServer(service,
		services.WithMux(mux),
		services.WithErrorHandler(errorHandler),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Error Handler Example Server")
	fmt.Println("============================")
	fmt.Println()
	fmt.Println("Server starting on :8080")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/v1/users       - Create user")
	fmt.Println("  GET    /api/v1/users/{id}  - Get user")
	fmt.Println("  PUT    /api/v1/users/{id}  - Update user")
	fmt.Println("  DELETE /api/v1/users/{id}  - Delete user")
	fmt.Println()
	fmt.Println("Test Commands:")
	fmt.Println()
	fmt.Println("1. Create a user (success):")
	fmt.Println(`   curl -X POST http://localhost:8080/api/v1/users \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Println(`     -H "X-Request-ID: test-123" \`)
	fmt.Println(`     -d '{"name": "John Doe", "email": "john@example.com"}'`)
	fmt.Println()
	fmt.Println("2. Create user with validation error (short name):")
	fmt.Println(`   curl -X POST http://localhost:8080/api/v1/users \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Println(`     -d '{"name": "J", "email": "invalid-email"}'`)
	fmt.Println()
	fmt.Println("3. Get non-existent user (not found error):")
	fmt.Println(`   curl -v http://localhost:8080/api/v1/users/non-existent-id`)
	fmt.Println()
	fmt.Println("4. Get user with request ID header:")
	fmt.Println(`   curl -v http://localhost:8080/api/v1/users/non-existent-id \`)
	fmt.Println(`     -H "X-Request-ID: my-trace-id-456"`)
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8080", mux))
}
