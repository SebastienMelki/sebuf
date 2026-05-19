// Errors-demo Go server. Each endpoint deterministically raises a specific
// typed *Error so the Python client can verify that the registry-based
// disambiguation in _raise_for_status picks the right subclass. The batch
// endpoint exercises the *Error.from_dict path: errors are embedded as a
// field on a regular response message (not raised as a status error).
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	pb "github.com/SebastienMelki/sebuf/examples/python-errors-demo/api/proto"
	"google.golang.org/protobuf/proto"
)

type errorsService struct{}

func (errorsService) GetItem(_ context.Context, req *pb.GetItemRequest) (*pb.Item, error) {
	if req.Id == "missing" {
		return nil, &pb.NotFoundError{ResourceType: "item", ResourceId: req.Id}
	}
	return &pb.Item{Id: req.Id, Title: "Found-" + req.Id}, nil
}

func (errorsService) CreateItem(_ context.Context, req *pb.CreateItemRequest) (*pb.Item, error) {
	if req.Title == "duplicate" {
		return nil, &pb.ConflictError{
			ResourceType: "item",
			Title:        req.Title,
			ExistingId:   "existing-1",
		}
	}
	return &pb.Item{Id: "new-1", Title: req.Title}, nil
}

func (errorsService) DeleteItem(_ context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	if req.Id == "missing" {
		return nil, &pb.NotFoundError{ResourceType: "item", ResourceId: req.Id}
	}
	return &pb.DeleteItemResponse{Deleted: true}, nil
}

func (errorsService) TriggerRateLimit(_ context.Context, _ *pb.TriggerRateLimitRequest) (*pb.Item, error) {
	return nil, &pb.RateLimitError{
		RetryAfterSeconds: 30,
		Detail:            "demo rate limit",
	}
}

// BatchCreateItems returns a 200 with per-row results. Empty titles or the
// literal "duplicate" produce embedded FieldValidationError values; other
// titles produce embedded Item values. The request type is BatchItemInput
// (not CreateItemRequest) precisely so buf.validate doesn't reject invalid
// rows at the gateway — we want them to reach this handler.
func (errorsService) BatchCreateItems(_ context.Context, req *pb.BatchCreateItemsRequest) (*pb.BatchCreateItemsResponse, error) {
	results := make([]*pb.BatchCreateItemResult, 0, len(req.Items))
	for i, item := range req.Items {
		title := item.Title
		switch {
		case strings.TrimSpace(title) == "":
			results = append(results, &pb.BatchCreateItemResult{
				Title: title,
				Error: &pb.FieldValidationError{
					Field:       "title",
					Description: "title is required",
				},
			})
		case title == "duplicate":
			results = append(results, &pb.BatchCreateItemResult{
				Title: title,
				Error: &pb.FieldValidationError{
					Field:       "title",
					Description: "title already exists",
				},
			})
		default:
			results = append(results, &pb.BatchCreateItemResult{
				Title: title,
				Item:  &pb.Item{Id: fmt.Sprintf("batch-%d", i), Title: title},
			})
		}
	}
	return &pb.BatchCreateItemsResponse{Results: results}, nil
}

// errorMapper translates typed proto errors into appropriate HTTP statuses so
// the Python client receives a non-2xx response (which is what triggers the
// _raise_for_status registry lookup).
func errorMapper(w http.ResponseWriter, _ *http.Request, err error) proto.Message {
	var nf *pb.NotFoundError
	if errors.As(err, &nf) {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}
	var cf *pb.ConflictError
	if errors.As(err, &cf) {
		w.WriteHeader(http.StatusConflict)
		return nil
	}
	var rl *pb.RateLimitError
	if errors.As(err, &rl) {
		w.WriteHeader(http.StatusTooManyRequests)
		return nil
	}
	return nil
}

func main() {
	mux := http.NewServeMux()
	if err := pb.RegisterErrorsDemoServiceServer(
		errorsService{},
		pb.WithMux(mux),
		pb.WithErrorHandler(errorMapper),
	); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Errors demo server running on http://localhost:3002")
	log.Fatal(http.ListenAndServe(":3002", mux))
}
