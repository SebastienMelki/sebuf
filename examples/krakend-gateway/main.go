package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/SebastienMelki/sebuf/examples/krakend-gateway/api/productservice"
	"github.com/SebastienMelki/sebuf/examples/krakend-gateway/api/userservice"
)

// ---------------------------------------------------------------------------
// UserService
// ---------------------------------------------------------------------------

type userServer struct {
	users  map[string]*userservice.User
	nextID int
}

func newUserServer() *userServer {
	s := &userServer{
		users:  make(map[string]*userservice.User),
		nextID: 4,
	}
	for _, u := range []*userservice.User{
		{Id: "user-1", Name: "Alice", Email: "alice@example.com", Status: "active"},
		{Id: "user-2", Name: "Bob", Email: "bob@example.com", Status: "active"},
		{Id: "user-3", Name: "Charlie", Email: "charlie@example.com", Status: "inactive"},
	} {
		s.users[u.Id] = u
	}
	return s
}

func (s *userServer) CreateUser(_ context.Context, req *userservice.CreateUserRequest) (*userservice.User, error) {
	u := &userservice.User{
		Id:     fmt.Sprintf("user-%d", s.nextID),
		Name:   req.Name,
		Email:  req.Email,
		Status: "active",
	}
	s.nextID++
	s.users[u.Id] = u
	log.Printf("Created user: %s", u.Id)
	return u, nil
}

func (s *userServer) GetUser(_ context.Context, req *userservice.GetUserRequest) (*userservice.User, error) {
	u, ok := s.users[req.Id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", req.Id)
	}
	return u, nil
}

func (s *userServer) ListUsers(_ context.Context, req *userservice.ListUsersRequest) (*userservice.ListUsersResponse, error) {
	var filtered []*userservice.User
	for _, u := range s.users {
		if req.Status != "" && u.Status != req.Status {
			continue
		}
		filtered = append(filtered, u)
	}
	return &userservice.ListUsersResponse{Users: filtered}, nil
}

func (s *userServer) UpdateUser(_ context.Context, req *userservice.UpdateUserRequest) (*userservice.User, error) {
	u, ok := s.users[req.Id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", req.Id)
	}
	if req.Name != "" {
		u.Name = req.Name
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	log.Printf("Updated user: %s", u.Id)
	return u, nil
}

// ---------------------------------------------------------------------------
// ProductService
// ---------------------------------------------------------------------------

type productServer struct {
	products map[string]*productservice.Product
	nextID   int
}

func newProductServer() *productServer {
	s := &productServer{
		products: make(map[string]*productservice.Product),
		nextID:   4,
	}
	for _, p := range []*productservice.Product{
		{Id: "prod-1", Name: "Laptop", Description: "A fast laptop", PriceCents: 99900, Category: "electronics"},
		{Id: "prod-2", Name: "Keyboard", Description: "Mechanical keyboard", PriceCents: 14900, Category: "electronics"},
		{Id: "prod-3", Name: "Desk Lamp", Description: "LED desk lamp", PriceCents: 3900, Category: "home"},
	} {
		s.products[p.Id] = p
	}
	return s
}

func (s *productServer) ListProducts(_ context.Context, _ *productservice.ListProductsRequest) (*productservice.ProductList, error) {
	var list []*productservice.Product
	for _, p := range s.products {
		list = append(list, p)
	}
	return &productservice.ProductList{Products: list}, nil
}

func (s *productServer) GetProduct(_ context.Context, req *productservice.GetProductRequest) (*productservice.Product, error) {
	p, ok := s.products[req.Id]
	if !ok {
		return nil, fmt.Errorf("product not found: %s", req.Id)
	}
	return p, nil
}

func (s *productServer) CreateProduct(_ context.Context, req *productservice.CreateProductRequest) (*productservice.Product, error) {
	p := &productservice.Product{
		Id:          fmt.Sprintf("prod-%d", s.nextID),
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Category:    req.Category,
	}
	s.nextID++
	s.products[p.Id] = p
	log.Printf("Created product: %s", p.Id)
	return p, nil
}

func (s *productServer) DeleteProduct(_ context.Context, req *productservice.DeleteProductRequest) (*productservice.Empty, error) {
	if _, ok := s.products[req.Id]; !ok {
		return nil, fmt.Errorf("product not found: %s", req.Id)
	}
	delete(s.products, req.Id)
	log.Printf("Deleted product: %s", req.Id)
	return &productservice.Empty{}, nil
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	mux := http.NewServeMux()

	if err := userservice.RegisterUserServiceServer(newUserServer(), userservice.WithMux(mux)); err != nil {
		log.Fatal(err)
	}
	if err := productservice.RegisterProductServiceServer(newProductServer(), productservice.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	log.Println("Backend server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
