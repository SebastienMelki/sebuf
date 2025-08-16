package authv1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
)

const (
	JSONContentType   = "application/json"
	BinaryContentType = "application/octet-stream"
	ProtoContentType  = "application/x-protobuf"
)

type bodyCtxKey struct{}

func getRequest[Req any](ctx context.Context) Req {
	val := ctx.Value(bodyCtxKey{})
	request, ok := val.(Req)
	if ok {
		return request
	}
	return *new(Req)
}

func BindingMiddleware[Req any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		toBind := new(Req)

		err := bindDataBasedOnContentType(r, toBind)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), bodyCtxKey{}, toBind)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}

func bindDataBasedOnContentType[Req any](r *http.Request, toBind *Req) error {
	contentType := filterFlags(r.Header.Get("Content-Type"))
	switch contentType {
	case JSONContentType:
		return bindDataFromJSONRequest(r, toBind)
	case BinaryContentType, ProtoContentType:
		return bindDataFromBinaryRequest(r, toBind)
	default:
		return bindDataFromBinaryRequest(r, toBind)
	}
}

func bindDataFromJSONRequest[Req any](r *http.Request, toBind *Req) error {
	bodyBytes, err := io.ReadAll(r.Body)

	// Refill it in case someone needs down the chain
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}

	if len(bodyBytes) == 0 {
		return nil
	}

	protoRequest, ok := any(toBind).(proto.Message)

	if !ok {
		return errors.New("JSON request is not a protocol buffer message")
	}

	err = protojson.Unmarshal(bodyBytes, protoRequest)

	if err != nil {
		return fmt.Errorf("could not unmarshal request JSON: %w", err)
	}
	return nil
}

func bindDataFromBinaryRequest[Req any](r *http.Request, toBind *Req) error {
	bodyBytes, err := io.ReadAll(r.Body)
	// Refill it in case someone needs down the chain
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// We are not using content-type correctly fundamentally, we should use "accept" for cases likes /search/trending.
	// To avoid crashing we will stop binary binding when the body is empty
	if len(bodyBytes) == 0 {
		return nil
	}

	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("could not read request body: %w", err)
	}

	protoRequest, ok := any(toBind).(proto.Message)

	if !ok {
		return errors.New("binary request is not a protocol buffer message")
	}

	err = proto.Unmarshal(bodyBytes, protoRequest)

	if err != nil {
		return fmt.Errorf("could not unmarshal binary request: %w", err)
	}

	return nil
}

func genericHandler[Req any, Res any](serve func(context.Context, Req) (Res, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := getRequest[Req](r.Context())

		response, err := serve(r.Context(), request)
		if err != nil {

			return
		}

		responseBytes, err := marshalResponse(r, response)

		if err != nil {
			http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(responseBytes)

		if err != nil {
			http.Error(w, fmt.Sprintf("failed to write response: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func marshalResponse(r *http.Request, response any) ([]byte, error) {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = JSONContentType
	}

	msg, ok := response.(proto.Message)

	if !ok {
		return nil, fmt.Errorf("response is not a protocol buffer message")
	}
	switch contentType {
	case JSONContentType:
		return protojson.Marshal(msg)
	case BinaryContentType, ProtoContentType:
		return proto.Marshal(msg)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

}
