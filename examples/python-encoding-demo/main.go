// Encoding-demo Go server. Returns hard-coded fixture data on every endpoint
// so the Python client can round-trip the wire form and assert that the
// generator's encoders/decoders match the Go HTTP plugin's output byte for
// byte. Each RPC exists to exercise exactly one JSON-mapping annotation.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	pb "github.com/SebastienMelki/sebuf/examples/python-encoding-demo/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type encodingService struct{}

// Fixed reference instant the demo uses so the wire form is deterministic.
// 2026-05-19T12:34:56.789Z = 1779537296 seconds, 1779537296789 millis.
var fixedTime = time.Date(2026, 5, 19, 12, 34, 56, 789_000_000, time.UTC)

// ---------------------------------------------------------------------------
// Enum value override
// ---------------------------------------------------------------------------

func (encodingService) GetEnumExample(_ context.Context, _ *pb.GetEnumExampleRequest) (*pb.EnumExample, error) {
	return &pb.EnumExample{
		Visibility: pb.Visibility_VISIBILITY_TEAM,
		Label:      "team-doc",
	}, nil
}

// ---------------------------------------------------------------------------
// Timestamps (every timestamp_format variant)
// ---------------------------------------------------------------------------

func (encodingService) GetTimestampsExample(_ context.Context, _ *pb.GetTimestampsExampleRequest) (*pb.TimestampsExample, error) {
	ts := timestamppb.New(fixedTime)
	return &pb.TimestampsExample{
		Rfc3339:     ts,
		UnixSeconds: ts,
		UnixMillis:  ts,
		DateOnly:    ts,
	}, nil
}

// ---------------------------------------------------------------------------
// int64 encoding
// ---------------------------------------------------------------------------

func (encodingService) GetInt64StringExample(_ context.Context, _ *pb.GetInt64StringExampleRequest) (*pb.Int64StringExample, error) {
	// 9_007_199_254_740_993 is 2^53 + 1 — fits in int64 but would silently
	// lose precision in a JS number. STRING wire form preserves it.
	return &pb.Int64StringExample{Value: 9_007_199_254_740_993}, nil
}

func (encodingService) GetInt64NumberExample(_ context.Context, _ *pb.GetInt64NumberExampleRequest) (*pb.Int64NumberExample, error) {
	return &pb.Int64NumberExample{Value: 12345}, nil
}

// ---------------------------------------------------------------------------
// bytes encoding
// ---------------------------------------------------------------------------

func (encodingService) GetBytesBase64Example(_ context.Context, _ *pb.GetBytesBase64ExampleRequest) (*pb.BytesBase64Example, error) {
	return &pb.BytesBase64Example{Data: []byte("Hello, sebuf!")}, nil
}

func (encodingService) GetBytesHexExample(_ context.Context, _ *pb.GetBytesHexExampleRequest) (*pb.BytesHexExample, error) {
	return &pb.BytesHexExample{Data: []byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe}}, nil
}

// ---------------------------------------------------------------------------
// flatten + flatten_prefix
// ---------------------------------------------------------------------------

func (encodingService) GetFlattenExample(_ context.Context, _ *pb.GetFlattenExampleRequest) (*pb.FlattenExample, error) {
	return &pb.FlattenExample{
		Name: "Alice",
		AuthorAddress: &pb.Address{
			Street:  "1 Sebuf Way",
			City:    "Casablanca",
			ZipCode: "20000",
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Oneof — nested + flattened variants
// ---------------------------------------------------------------------------

func (encodingService) GetOneofNested(_ context.Context, req *pb.GetOneofNestedRequest) (*pb.OneofNestedExample, error) {
	if req.Variant == "link" {
		return &pb.OneofNestedExample{
			Id: "n1",
			Content: &pb.OneofNestedExample_Link{
				Link: &pb.LinkAttachment{Url: "https://example.com", Title: "Example"},
			},
		}, nil
	}
	return &pb.OneofNestedExample{
		Id: "n1",
		Content: &pb.OneofNestedExample_Image{
			Image: &pb.ImageAttachment{Url: "https://example.com/img.png", Width: 800, Height: 600},
		},
	}, nil
}

func (encodingService) GetOneofFlattened(_ context.Context, req *pb.GetOneofFlattenedRequest) (*pb.OneofFlattenedExample, error) {
	if req.Variant == "image" {
		return &pb.OneofFlattenedExample{
			Id:      "f1",
			Payload: &pb.OneofFlattenedExample_Image{Image: &pb.ImagePayload{Url: "https://example.com/i.png"}},
		}, nil
	}
	return &pb.OneofFlattenedExample{
		Id:      "f1",
		Payload: &pb.OneofFlattenedExample_Text{Text: &pb.TextPayload{Body: "hello world"}},
	}, nil
}

// ---------------------------------------------------------------------------
// Unwrap — root repeated, root map, map-value
// ---------------------------------------------------------------------------

func sampleItems() []*pb.Item {
	return []*pb.Item{
		{Id: "i1", Name: "Apple"},
		{Id: "i2", Name: "Banana"},
		{Id: "i3", Name: "Cherry"},
	}
}

func (encodingService) GetItemList(_ context.Context, _ *pb.GetItemListRequest) (*pb.ItemList, error) {
	return &pb.ItemList{Items: sampleItems()}, nil
}

func (encodingService) GetItemMap(_ context.Context, _ *pb.GetItemMapRequest) (*pb.ItemMap, error) {
	items := sampleItems()
	m := make(map[string]*pb.Item, len(items))
	for _, it := range items {
		m[it.Id] = it
	}
	return &pb.ItemMap{Items: m}, nil
}

func (encodingService) GetItemsByCategory(_ context.Context, _ *pb.GetItemsByCategoryRequest) (*pb.ItemsByCategory, error) {
	return &pb.ItemsByCategory{
		Buckets: map[string]*pb.ItemBucket{
			"fruits":  {Items: []*pb.Item{{Id: "i1", Name: "Apple"}, {Id: "i2", Name: "Banana"}}},
			"berries": {Items: []*pb.Item{{Id: "i3", Name: "Cherry"}}},
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Python keyword field + repeated query params
// ---------------------------------------------------------------------------

func (encodingService) GetPyKeyword(_ context.Context, _ *pb.GetPyKeywordRequest) (*pb.PyKeywordExample, error) {
	return &pb.PyKeywordExample{
		From:   "sender@example.com",
		Class:  "first-class",
		Return: "200 OK",
		Normal: "no-keyword-here",
	}, nil
}

func (encodingService) ListItems(_ context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	all := sampleItems()
	var filtered []*pb.Item
	if len(req.Tag) == 0 {
		filtered = all
	} else {
		set := map[string]bool{}
		for _, t := range req.Tag {
			set[t] = true
		}
		// Pretend Apple is tagged "fruit"+"red", Banana is "fruit"+"yellow",
		// Cherry is "fruit"+"red"+"berry". Filter by intersection.
		tags := map[string][]string{
			"i1": {"fruit", "red"},
			"i2": {"fruit", "yellow"},
			"i3": {"fruit", "red", "berry"},
		}
		for _, it := range all {
			for _, t := range tags[it.Id] {
				if set[t] {
					filtered = append(filtered, it)
					break
				}
			}
		}
	}
	if req.Limit > 0 && int(req.Limit) < len(filtered) {
		filtered = filtered[:req.Limit]
	}
	return &pb.ListItemsResponse{Items: filtered, Total: int32(len(filtered))}, nil
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func main() {
	mux := http.NewServeMux()
	if err := pb.RegisterEncodingDemoServiceServer(encodingService{}, pb.WithMux(mux)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Encoding demo server running on http://localhost:3001")
	fmt.Println("Endpoints exercise every JSON-mapping annotation supported by protoc-gen-py-client.")
	log.Fatal(http.ListenAndServe(":3001", mux))
}
