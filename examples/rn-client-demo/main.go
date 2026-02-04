package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	pb "github.com/SebastienMelki/sebuf/examples/rn-client-demo/api/proto"
	"google.golang.org/protobuf/proto"
)

type taskService struct {
	mu     sync.RWMutex
	tasks  map[string]*pb.Task
	nextID int
}

func newTaskService() *taskService {
	svc := &taskService{
		tasks:  make(map[string]*pb.Task),
		nextID: 5,
	}
	svc.seedData()
	return svc
}

func strPtr(s string) *string { return &s }

func (s *taskService) seedData() {
	now := time.Now()
	s.tasks["task-1"] = &pb.Task{
		Id:          "task-1",
		Title:       "Set up project structure",
		Description: "Initialize repo, configure build tools, and set up CI",
		Priority:    pb.Priority_PRIORITY_HIGH,
		Status:      pb.TaskStatus_TASK_STATUS_DONE,
		Labels: []*pb.Label{
			{Name: "backend", Color: "#3b82f6"},
			{Name: "setup", Color: "#8b5cf6"},
		},
		Metadata:  map[string]string{"sprint": "1", "team": "platform"},
		CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339),
	}
	s.tasks["task-2"] = &pb.Task{
		Id:          "task-2",
		Title:       "Implement user authentication",
		Description: "Add JWT-based auth with refresh tokens",
		Priority:    pb.Priority_PRIORITY_HIGH,
		Status:      pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
		Labels: []*pb.Label{
			{Name: "backend", Color: "#3b82f6"},
			{Name: "security", Color: "#ef4444"},
		},
		Metadata:  map[string]string{"sprint": "2", "assignee": "alice"},
		DueDate:   strPtr("2025-12-31"),
		CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339),
	}
	s.tasks["task-3"] = &pb.Task{
		Id:          "task-3",
		Title:       "Design landing page",
		Description: "Create responsive landing page with feature highlights",
		Priority:    pb.Priority_PRIORITY_MEDIUM,
		Status:      pb.TaskStatus_TASK_STATUS_TODO,
		Labels: []*pb.Label{
			{Name: "frontend", Color: "#10b981"},
			{Name: "design", Color: "#f59e0b"},
		},
		CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
	}
	s.tasks["task-4"] = &pb.Task{
		Id:          "task-4",
		Title:       "Write API documentation",
		Description: "Document all endpoints with examples",
		Priority:    pb.Priority_PRIORITY_LOW,
		Status:      pb.TaskStatus_TASK_STATUS_TODO,
		Labels: []*pb.Label{
			{Name: "backend", Color: "#3b82f6"},
		},
		Metadata:  map[string]string{"sprint": "3"},
		DueDate:   strPtr("2026-01-15"),
		CreatedAt: now.Add(-12 * time.Hour).Format(time.RFC3339),
	}
}

func (s *taskService) ListTasks(_ context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*pb.Task
	for _, task := range s.tasks {
		if req.Status != "" && !matchStatus(task.Status, req.Status) {
			continue
		}
		tasks = append(tasks, task)
	}

	// Sort by created_at descending (newest first)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt > tasks[j].CreatedAt
	})

	total := int32(len(tasks))

	// Pagination
	if req.Offset > 0 && int(req.Offset) < len(tasks) {
		tasks = tasks[req.Offset:]
	} else if req.Offset > 0 {
		tasks = nil
	}
	if req.Limit > 0 && int(req.Limit) < len(tasks) {
		tasks = tasks[:req.Limit]
	}

	return &pb.ListTasksResponse{
		Tasks: tasks,
		Total: total,
	}, nil
}

func matchStatus(taskStatus pb.TaskStatus, filter string) bool {
	switch strings.ToLower(filter) {
	case "todo":
		return taskStatus == pb.TaskStatus_TASK_STATUS_TODO
	case "in_progress":
		return taskStatus == pb.TaskStatus_TASK_STATUS_IN_PROGRESS
	case "done":
		return taskStatus == pb.TaskStatus_TASK_STATUS_DONE
	default:
		return false
	}
}

func (s *taskService) GetTask(_ context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[req.Id]
	if !ok {
		return nil, &pb.TaskNotFoundError{
			TaskId:  req.Id,
			Message: fmt.Sprintf("task %q not found", req.Id),
		}
	}
	return task, nil
}

func (s *taskService) CreateTask(_ context.Context, req *pb.CreateTaskRequest) (*pb.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &pb.Task{
		Id:          fmt.Sprintf("task-%d", s.nextID),
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      pb.TaskStatus_TASK_STATUS_TODO,
		Labels:      req.Labels,
		Metadata:    req.Metadata,
		DueDate:     req.DueDate,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	s.nextID++
	s.tasks[task.Id] = task

	log.Printf("Created task: %s - %s (priority=%s)", task.Id, task.Title, task.Priority)
	return task, nil
}

func (s *taskService) UpdateTask(_ context.Context, req *pb.UpdateTaskRequest) (*pb.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[req.Id]
	if !ok {
		return nil, &pb.TaskNotFoundError{
			TaskId:  req.Id,
			Message: fmt.Sprintf("task %q not found", req.Id),
		}
	}

	task.Title = req.Title
	task.Description = req.Description
	task.Priority = req.Priority
	task.Status = req.Status
	task.Labels = req.Labels
	task.Metadata = req.Metadata
	task.DueDate = req.DueDate

	log.Printf("Updated task: %s - status=%s", task.Id, task.Status)
	return task, nil
}

func (s *taskService) DeleteTask(_ context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[req.Id]; !ok {
		return nil, &pb.TaskNotFoundError{
			TaskId:  req.Id,
			Message: fmt.Sprintf("task %q not found", req.Id),
		}
	}

	delete(s.tasks, req.Id)
	log.Printf("Deleted task: %s", req.Id)
	return &pb.DeleteTaskResponse{Success: true}, nil
}

func (s *taskService) GetTasksByLabel(_ context.Context, req *pb.GetTasksByLabelRequest) (*pb.TaskList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*pb.Task
	for _, task := range s.tasks {
		for _, label := range task.Labels {
			if strings.EqualFold(label.Name, req.Label) {
				tasks = append(tasks, task)
				break
			}
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt < tasks[j].CreatedAt
	})

	return &pb.TaskList{Tasks: tasks}, nil
}

func taskServiceErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	var notFound *pb.TaskNotFoundError
	if errors.As(err, &notFound) {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	return nil
}

// corsMiddleware adds CORS headers for Expo web/dev compatibility.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	service := newTaskService()
	mux := http.NewServeMux()

	if err := pb.RegisterTaskServiceServer(service,
		pb.WithMux(mux),
		pb.WithErrorHandler(taskServiceErrorHandler),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Task API server running on http://localhost:3000")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET    /api/v1/tasks              - List tasks (?status=todo&limit=10&offset=0)")
	fmt.Println("  GET    /api/v1/tasks/{id}          - Get task")
	fmt.Println("  POST   /api/v1/tasks               - Create task (requires X-Request-ID)")
	fmt.Println("  PUT    /api/v1/tasks/{id}           - Update task")
	fmt.Println("  DELETE /api/v1/tasks/{id}           - Delete task")
	fmt.Println("  GET    /api/v1/tasks/by-label       - Get tasks by label (?label=backend) -> Task[]")
	fmt.Println()
	fmt.Println("All endpoints require X-API-Key header")
	fmt.Println("4 seed tasks pre-loaded")
	fmt.Println("CORS enabled for all origins")

	log.Fatal(http.ListenAndServe(":3000", corsMiddleware(mux)))
}
