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

	pb "github.com/SebastienMelki/sebuf/examples/ts-client-demo/api/proto"
	"google.golang.org/protobuf/proto"
)

type noteService struct {
	mu     sync.RWMutex
	notes  map[string]*pb.Note
	nextID int
}

func newNoteService() *noteService {
	svc := &noteService{
		notes:  make(map[string]*pb.Note),
		nextID: 5,
	}
	svc.seedData()
	return svc
}

func strPtr(s string) *string { return &s }

func (s *noteService) seedData() {
	now := time.Now()
	s.notes["note-1"] = &pb.Note{
		Id:       "note-1",
		Title:    "Design API schema",
		Content:  "Define proto messages and service RPCs",
		Priority: pb.Priority_PRIORITY_HIGH,
		Status:   pb.Status_STATUS_DONE,
		Tags: []*pb.Tag{
			{Name: "backend", Color: "#3b82f6"},
			{Name: "design", Color: "#8b5cf6"},
		},
		Metadata:  map[string]string{"sprint": "12", "team": "platform"},
		CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339),
	}
	s.notes["note-2"] = &pb.Note{
		Id:       "note-2",
		Title:    "Write unit tests",
		Content:  "Cover edge cases for validation and error handling",
		Priority: pb.Priority_PRIORITY_MEDIUM,
		Status:   pb.Status_STATUS_IN_PROGRESS,
		Tags: []*pb.Tag{
			{Name: "backend", Color: "#3b82f6"},
			{Name: "testing", Color: "#10b981"},
		},
		Metadata:  map[string]string{"sprint": "12"},
		DueDate:   strPtr("2025-12-31"),
		CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339),
	}
	s.notes["note-3"] = &pb.Note{
		Id:       "note-3",
		Title:    "Update documentation",
		Content:  "Add examples for all new features",
		Priority: pb.Priority_PRIORITY_LOW,
		Status:   pb.Status_STATUS_PENDING,
		Tags: []*pb.Tag{
			{Name: "docs", Color: "#f59e0b"},
		},
		CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
	}
	s.notes["note-4"] = &pb.Note{
		Id:       "note-4",
		Title:    "Fix login bug",
		Content:  "Session expires too early on mobile",
		Priority: pb.Priority_PRIORITY_URGENT,
		Status:   pb.Status_STATUS_PENDING,
		Tags: []*pb.Tag{
			{Name: "backend", Color: "#3b82f6"},
			{Name: "bug", Color: "#ef4444"},
		},
		Metadata:  map[string]string{"reporter": "alice", "severity": "critical"},
		DueDate:   strPtr("2025-06-15"),
		CreatedAt: now.Add(-12 * time.Hour).Format(time.RFC3339),
	}
}

func (s *noteService) ListNotes(_ context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var notes []*pb.Note
	for _, note := range s.notes {
		if req.Status != "" && !matchStatus(note.Status, req.Status) {
			continue
		}
		if req.Priority != "" && !matchPriority(note.Priority, req.Priority) {
			continue
		}
		notes = append(notes, note)
	}

	// Sort
	sortNotes(notes, req.Sort)

	total := int32(len(notes))

	// Pagination
	if req.Offset > 0 && int(req.Offset) < len(notes) {
		notes = notes[req.Offset:]
	} else if req.Offset > 0 {
		notes = nil
	}
	if req.Limit > 0 && int(req.Limit) < len(notes) {
		notes = notes[:req.Limit]
	}

	return &pb.ListNotesResponse{
		Notes: notes,
		Total: total,
	}, nil
}

func matchStatus(noteStatus pb.Status, filter string) bool {
	switch strings.ToLower(filter) {
	case "pending":
		return noteStatus == pb.Status_STATUS_PENDING
	case "in_progress":
		return noteStatus == pb.Status_STATUS_IN_PROGRESS
	case "done":
		return noteStatus == pb.Status_STATUS_DONE
	case "archived":
		return noteStatus == pb.Status_STATUS_ARCHIVED
	default:
		return false
	}
}

func matchPriority(notePriority pb.Priority, filter string) bool {
	switch strings.ToLower(filter) {
	case "low":
		return notePriority == pb.Priority_PRIORITY_LOW
	case "medium":
		return notePriority == pb.Priority_PRIORITY_MEDIUM
	case "high":
		return notePriority == pb.Priority_PRIORITY_HIGH
	case "urgent":
		return notePriority == pb.Priority_PRIORITY_URGENT
	default:
		return false
	}
}

func sortNotes(notes []*pb.Note, sortField string) {
	switch strings.ToLower(sortField) {
	case "title":
		sort.Slice(notes, func(i, j int) bool {
			return notes[i].Title < notes[j].Title
		})
	case "priority":
		sort.Slice(notes, func(i, j int) bool {
			return notes[i].Priority > notes[j].Priority // higher priority first
		})
	case "created_at":
		sort.Slice(notes, func(i, j int) bool {
			return notes[i].CreatedAt < notes[j].CreatedAt
		})
	default:
		// Default: sort by created_at descending (newest first)
		sort.Slice(notes, func(i, j int) bool {
			return notes[i].CreatedAt > notes[j].CreatedAt
		})
	}
}

func (s *noteService) GetNote(_ context.Context, req *pb.GetNoteRequest) (*pb.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	note, ok := s.notes[req.Id]
	if !ok {
		return nil, &pb.NotFoundError{
			ResourceType: "note",
			ResourceId:   req.Id,
		}
	}
	return note, nil
}

func (s *noteService) CreateNote(_ context.Context, req *pb.CreateNoteRequest) (*pb.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note := &pb.Note{
		Id:        fmt.Sprintf("note-%d", s.nextID),
		Title:     req.Title,
		Content:   req.Content,
		Priority:  req.Priority,
		Status:    pb.Status_STATUS_PENDING,
		Tags:      req.Tags,
		Metadata:  req.Metadata,
		DueDate:   req.DueDate,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	s.nextID++
	s.notes[note.Id] = note

	log.Printf("Created note: %s - %s (priority=%s)", note.Id, note.Title, note.Priority)
	return note, nil
}

func (s *noteService) UpdateNote(_ context.Context, req *pb.UpdateNoteRequest) (*pb.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[req.Id]
	if !ok {
		return nil, &pb.NotFoundError{
			ResourceType: "note",
			ResourceId:   req.Id,
		}
	}

	note.Title = req.Title
	note.Content = req.Content
	note.Priority = req.Priority
	note.Status = req.Status
	note.Tags = req.Tags
	note.Metadata = req.Metadata
	note.DueDate = req.DueDate

	log.Printf("Updated note: %s - status=%s", note.Id, note.Status)
	return note, nil
}

func (s *noteService) ArchiveNote(_ context.Context, req *pb.ArchiveNoteRequest) (*pb.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[req.Id]
	if !ok {
		return nil, &pb.NotFoundError{
			ResourceType: "note",
			ResourceId:   req.Id,
		}
	}

	note.Status = pb.Status_STATUS_ARCHIVED
	log.Printf("Archived note: %s", note.Id)
	return note, nil
}

func (s *noteService) DeleteNote(_ context.Context, req *pb.DeleteNoteRequest) (*pb.DeleteNoteResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notes[req.Id]; !ok {
		return nil, &pb.NotFoundError{
			ResourceType: "note",
			ResourceId:   req.Id,
		}
	}

	delete(s.notes, req.Id)
	log.Printf("Deleted note: %s", req.Id)
	return &pb.DeleteNoteResponse{Success: true}, nil
}

func (s *noteService) GetNotesByTag(_ context.Context, req *pb.GetNotesByTagRequest) (*pb.NoteList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var notes []*pb.Note
	for _, note := range s.notes {
		for _, tag := range note.Tags {
			if strings.EqualFold(tag.Name, req.Tag) {
				notes = append(notes, note)
				break
			}
		}
	}

	sortNotes(notes, "created_at")

	return &pb.NoteList{Notes: notes}, nil
}

func noteServiceErrorHandler(w http.ResponseWriter, r *http.Request, err error) proto.Message {
	var notFound *pb.NotFoundError
	if errors.As(err, &notFound) {
		w.WriteHeader(http.StatusNotFound)
		return nil // use default serialization of the NotFoundError proto
	}
	return nil // fall through to default handling
}

func main() {
	service := newNoteService()
	mux := http.NewServeMux()

	if err := pb.RegisterNoteServiceServer(service,
		pb.WithMux(mux),
		pb.WithErrorHandler(noteServiceErrorHandler),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Note API server running on http://localhost:3000")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET    /api/v1/notes              - List notes (?status=pending&priority=high&sort=title&limit=10&offset=0)")
	fmt.Println("  GET    /api/v1/notes/{id}          - Get note")
	fmt.Println("  POST   /api/v1/notes               - Create note (requires X-Request-ID)")
	fmt.Println("  PUT    /api/v1/notes/{id}           - Update note (requires X-Idempotency-Key)")
	fmt.Println("  PATCH  /api/v1/notes/{id}/archive   - Archive note")
	fmt.Println("  DELETE /api/v1/notes/{id}           - Delete note")
	fmt.Println("  GET    /api/v1/notes/by-tag         - Get notes by tag (?tag=backend) -> Note[]")
	fmt.Println()
	fmt.Println("All endpoints require X-API-Key and X-Tenant-ID headers")
	fmt.Println("4 seed notes pre-loaded")

	log.Fatal(http.ListenAndServe(":3000", mux))
}
