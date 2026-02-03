package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	pb "github.com/SebastienMelki/sebuf/examples/ts-client-demo/api/proto"
)

type noteService struct {
	mu     sync.RWMutex
	notes  map[string]*pb.Note
	nextID int
}

func newNoteService() *noteService {
	return &noteService{
		notes:  make(map[string]*pb.Note),
		nextID: 1,
	}
}

func (s *noteService) ListNotes(_ context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var notes []*pb.Note
	for _, note := range s.notes {
		// Filter by status if provided
		if req.Status != "" {
			if req.Status == "done" && !note.Done {
				continue
			}
			if req.Status == "pending" && note.Done {
				continue
			}
		}
		notes = append(notes, note)
	}

	// Apply limit
	if req.Limit > 0 && int(req.Limit) < len(notes) {
		notes = notes[:req.Limit]
	}

	return &pb.ListNotesResponse{
		Notes: notes,
		Total: int32(len(notes)),
	}, nil
}

func (s *noteService) GetNote(_ context.Context, req *pb.GetNoteRequest) (*pb.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	note, ok := s.notes[req.Id]
	if !ok {
		return nil, fmt.Errorf("note not found: %s", req.Id)
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
		Done:      false,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	s.nextID++
	s.notes[note.Id] = note

	log.Printf("Created note: %s - %s", note.Id, note.Title)
	return note, nil
}

func (s *noteService) UpdateNote(_ context.Context, req *pb.UpdateNoteRequest) (*pb.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[req.Id]
	if !ok {
		return nil, fmt.Errorf("note not found: %s", req.Id)
	}

	note.Title = req.Title
	note.Content = req.Content
	note.Done = req.Done

	log.Printf("Updated note: %s - done=%v", note.Id, note.Done)
	return note, nil
}

func (s *noteService) DeleteNote(_ context.Context, req *pb.DeleteNoteRequest) (*pb.DeleteNoteResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notes[req.Id]; !ok {
		return nil, fmt.Errorf("note not found: %s", req.Id)
	}

	delete(s.notes, req.Id)
	log.Printf("Deleted note: %s", req.Id)

	return &pb.DeleteNoteResponse{Success: true}, nil
}

func main() {
	service := newNoteService()
	mux := http.NewServeMux()

	if err := pb.RegisterNoteServiceServer(service, pb.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Note API server running on http://localhost:3000")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET    /api/v1/notes          - List notes (?status=done&limit=10)")
	fmt.Println("  GET    /api/v1/notes/{id}      - Get note")
	fmt.Println("  POST   /api/v1/notes           - Create note (requires X-Request-ID)")
	fmt.Println("  PUT    /api/v1/notes/{id}       - Update note")
	fmt.Println("  DELETE /api/v1/notes/{id}       - Delete note")
	fmt.Println()
	fmt.Println("All endpoints require X-API-Key header")

	log.Fatal(http.ListenAndServe(":3000", mux))
}
