package authv1

import (
	"context"
	"fmt"
	"net/http"
)

func NotImplementedHandler(path string) http.Handler {
	return http.HandlerFunc(NotImplemented(path))
}

func NotImplemented(path string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, fmt.Sprintf("%s is not implemented", path), http.StatusNotImplemented)
	}
}

func init() {
	path := "/anghami/auth/v1/"
	http.DefaultServeMux.Handle(fmt.Sprintf("POST %s", path), NotImplementedHandler(path))
}

type AuthServiceServer interface {
	Login(context.Context, *LoginRequest) (*LoginResponse, error)
}

func RegisterAuthServiceServer(server AuthServiceServer, opts ...ServerOption) error {
	config := getConfiguration(opts...)
	loginHandler := BindingMiddleware[LoginRequest](
		genericHandler[*LoginRequest, *LoginResponse](server.Login),
	)

	config.mux.Handle("POST /anghami/auth/v1/login", loginHandler)

	return nil
}
