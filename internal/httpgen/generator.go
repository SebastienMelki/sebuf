package httpgen

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// Generator handles HTTP code generation for protobuf services
type Generator struct {
	plugin *protogen.Plugin
}

// New creates a new HTTP generator
func New(plugin *protogen.Plugin) *Generator {
	return &Generator{
		plugin: plugin,
	}
}

// Generate processes all files and generates HTTP handlers
func (g *Generator) Generate() error {
	for _, file := range g.plugin.Files {
		if !file.Generate {
			continue
		}
		if err := g.generateFile(file); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateFile(file *protogen.File) error {
	if len(file.Services) == 0 {
		return nil
	}

	// Generate main HTTP file
	if err := g.generateHTTPFile(file); err != nil {
		return err
	}

	// Generate binding file
	if err := g.generateBindingFile(file); err != nil {
		return err
	}

	// Generate config file
	if err := g.generateConfigFile(file); err != nil {
		return err
	}

	return nil
}

func (g *Generator) generateHTTPFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_http.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)

	gf.P("import (")
	gf.P(`"context"`)
	gf.P(")")
	gf.P()

	for _, service := range file.Services {
		if err := g.generateService(gf, file, service); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) generateService(gf *protogen.GeneratedFile, file *protogen.File, service *protogen.Service) error {
	serviceName := service.GoName

	// Generate service interface
	gf.P("// ", serviceName, "Server is the server API for ", serviceName, " service.")
	gf.P("type ", serviceName, "Server interface {")
	for _, method := range service.Methods {
		gf.P(method.GoName, "(context.Context, *", method.Input.GoIdent, ") (*", method.Output.GoIdent, ", error)")
	}
	gf.P("}")
	gf.P()

	// Generate registration function
	gf.P("// Register", serviceName, "Server registers the HTTP handlers for service ", serviceName, " to the given mux.")
	gf.P("func Register", serviceName, "Server(server ", serviceName, "Server, opts ...ServerOption) error {")
	gf.P("config := getConfiguration(opts...)")
	gf.P()

	// Get service-level base path if configured
	basePath := g.getServiceBasePath(service)

	for _, method := range service.Methods {
		httpPath := g.getMethodPath(method, basePath, file.GoPackageName)

		handlerName := fmt.Sprintf("%sHandler", lowerFirst(method.GoName))
		gf.P(handlerName, " := BindingMiddleware[", method.Input.GoIdent, "](")
		gf.P("genericHandler(server.", method.GoName, "),")
		gf.P(")")
		gf.P()
		gf.P(`config.mux.Handle("POST `, httpPath, `", `, handlerName, `)`)
		gf.P()
	}

	gf.P("return nil")
	gf.P("}")
	gf.P()

	return nil
}

func (g *Generator) generateBindingFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_http_binding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)

	gf.P("import (")
	gf.P(`"bytes"`)
	gf.P(`"context"`)
	gf.P(`"errors"`)
	gf.P(`"fmt"`)
	gf.P(`"io"`)
	gf.P(`"net/http"`)
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(`"google.golang.org/protobuf/proto"`)
	gf.P(")")
	gf.P()

	// Content type constants
	gf.P("const (")
	gf.P(`// JSONContentType is the content type for JSON`)
	gf.P(`JSONContentType = "application/json"`)
	gf.P(`// BinaryContentType is the content type for binary protobuf`)
	gf.P(`BinaryContentType = "application/octet-stream"`)
	gf.P(`// ProtoContentType is the content type for protobuf`)
	gf.P(`ProtoContentType = "application/x-protobuf"`)
	gf.P(")")
	gf.P()

	// Context key for request storage
	gf.P("type bodyCtxKey struct{}")
	gf.P()

	// getRequest function
	gf.P("func getRequest[Req any](ctx context.Context) Req {")
	gf.P("val := ctx.Value(bodyCtxKey{})")
	gf.P("request, ok := val.(Req)")
	gf.P("if ok {")
	gf.P("return request")
	gf.P("}")
	gf.P("return *new(Req)")
	gf.P("}")
	gf.P()

	// BindingMiddleware function
	gf.P("// BindingMiddleware creates a middleware that binds HTTP requests to protobuf messages")
	gf.P("func BindingMiddleware[Req any](next http.Handler) http.Handler {")
	gf.P("return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {")
	gf.P("toBind := new(Req)")
	gf.P()
	gf.P("err := bindDataBasedOnContentType(r, toBind)")
	gf.P("if err != nil {")
	gf.P(`http.Error(w, "bad request", http.StatusBadRequest)`)
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("ctx := context.WithValue(r.Context(), bodyCtxKey{}, toBind)")
	gf.P("next.ServeHTTP(w, r.WithContext(ctx))")
	gf.P("})")
	gf.P("}")
	gf.P()

	// filterFlags helper
	gf.P("func filterFlags(content string) string {")
	gf.P("for i, char := range content {")
	gf.P("if char == ' ' || char == ';' {")
	gf.P("return content[:i]")
	gf.P("}")
	gf.P("}")
	gf.P("return content")
	gf.P("}")
	gf.P()

	// bindDataBasedOnContentType function
	gf.P("func bindDataBasedOnContentType[Req any](r *http.Request, toBind *Req) error {")
	gf.P(`contentType := filterFlags(r.Header.Get("Content-Type"))`)
	gf.P("switch contentType {")
	gf.P("case JSONContentType:")
	gf.P("return bindDataFromJSONRequest(r, toBind)")
	gf.P("case BinaryContentType, ProtoContentType:")
	gf.P("return bindDataFromBinaryRequest(r, toBind)")
	gf.P("default:")
	gf.P("return bindDataFromBinaryRequest(r, toBind)")
	gf.P("}")
	gf.P("}")
	gf.P()

	// bindDataFromJSONRequest function
	gf.P("func bindDataFromJSONRequest[Req any](r *http.Request, toBind *Req) error {")
	gf.P("bodyBytes, err := io.ReadAll(r.Body)")
	gf.P("r.Body = io.NopCloser(bytes.NewReader(bodyBytes))")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("could not read request body: %w", err)`)
	gf.P("}")
	gf.P()
	gf.P("if len(bodyBytes) == 0 {")
	gf.P("return nil")
	gf.P("}")
	gf.P()
	gf.P("protoRequest, ok := any(toBind).(proto.Message)")
	gf.P("if !ok {")
	gf.P(`return errors.New("JSON request is not a protocol buffer message")`)
	gf.P("}")
	gf.P()
	gf.P("err = protojson.Unmarshal(bodyBytes, protoRequest)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("could not unmarshal request JSON: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()

	// bindDataFromBinaryRequest function
	gf.P("func bindDataFromBinaryRequest[Req any](r *http.Request, toBind *Req) error {")
	gf.P("bodyBytes, err := io.ReadAll(r.Body)")
	gf.P("r.Body = io.NopCloser(bytes.NewReader(bodyBytes))")
	gf.P()
	gf.P("if len(bodyBytes) == 0 {")
	gf.P("return nil")
	gf.P("}")
	gf.P()
	gf.P("if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {")
	gf.P(`return fmt.Errorf("could not read request body: %w", err)`)
	gf.P("}")
	gf.P()
	gf.P("protoRequest, ok := any(toBind).(proto.Message)")
	gf.P("if !ok {")
	gf.P(`return errors.New("binary request is not a protocol buffer message")`)
	gf.P("}")
	gf.P()
	gf.P("err = proto.Unmarshal(bodyBytes, protoRequest)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("could not unmarshal binary request: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()

	// genericHandler function
	gf.P("func genericHandler[Req any, Res any](serve func(context.Context, Req) (Res, error)) http.HandlerFunc {")
	gf.P("return func(w http.ResponseWriter, r *http.Request) {")
	gf.P("request := getRequest[Req](r.Context())")
	gf.P()
	gf.P("response, err := serve(r.Context(), request)")
	gf.P("if err != nil {")
	gf.P(`http.Error(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)`)
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("responseBytes, err := marshalResponse(r, response)")
	gf.P("if err != nil {")
	gf.P(`http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)`)
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("_, err = w.Write(responseBytes)")
	gf.P("if err != nil {")
	gf.P(`http.Error(w, fmt.Sprintf("failed to write response: %v", err), http.StatusInternalServerError)`)
	gf.P("return")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()

	// marshalResponse function
	gf.P("func marshalResponse(r *http.Request, response any) ([]byte, error) {")
	gf.P(`contentType := r.Header.Get("Content-Type")`)
	gf.P("if contentType == \"\" {")
	gf.P("contentType = JSONContentType")
	gf.P("}")
	gf.P()
	gf.P("msg, ok := response.(proto.Message)")
	gf.P("if !ok {")
	gf.P(`return nil, fmt.Errorf("response is not a protocol buffer message")`)
	gf.P("}")
	gf.P()
	gf.P("switch filterFlags(contentType) {")
	gf.P("case JSONContentType:")
	gf.P("return protojson.Marshal(msg)")
	gf.P("case BinaryContentType, ProtoContentType:")
	gf.P("return proto.Marshal(msg)")
	gf.P("default:")
	gf.P(`return nil, fmt.Errorf("unsupported content type: %s", contentType)`)
	gf.P("}")
	gf.P("}")
	gf.P()

	return nil
}

func (g *Generator) generateConfigFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_http_config.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)

	gf.P("import (")
	gf.P(`"net/http"`)
	gf.P(")")
	gf.P()

	// ServerOption type
	gf.P("// ServerOption configures a Server")
	gf.P("type ServerOption func(c *serverConfiguration)")
	gf.P()

	// serverConfiguration struct
	gf.P("type serverConfiguration struct {")
	gf.P("mux *http.ServeMux")
	gf.P("withMux bool")
	gf.P("}")
	gf.P()

	// getDefaultConfiguration function
	gf.P("func getDefaultConfiguration() *serverConfiguration {")
	gf.P("return &serverConfiguration{")
	gf.P("mux: http.DefaultServeMux,")
	gf.P("withMux: false,")
	gf.P("}")
	gf.P("}")
	gf.P()

	// getConfiguration function
	gf.P("func getConfiguration(options ...ServerOption) *serverConfiguration {")
	gf.P("configuration := getDefaultConfiguration()")
	gf.P("for _, option := range options {")
	gf.P("option(configuration)")
	gf.P("}")
	gf.P("return configuration")
	gf.P("}")
	gf.P()

	// WithMux option
	gf.P("// WithMux configures the Server to use the given ServeMux")
	gf.P("func WithMux(mux *http.ServeMux) ServerOption {")
	gf.P("return func(c *serverConfiguration) {")
	gf.P("c.mux = mux")
	gf.P("c.withMux = true")
	gf.P("}")
	gf.P("}")
	gf.P()

	return nil
}

func (g *Generator) writeHeader(gf *protogen.GeneratedFile, file *protogen.File) {
	gf.P("// Code generated by protoc-gen-go-http. DO NOT EDIT.")
	gf.P("// source: ", file.Desc.Path())
	gf.P()
	gf.P("package ", file.GoPackageName)
	gf.P()
}

// getMethodPath determines the HTTP path for a method
func (g *Generator) getMethodPath(method *protogen.Method, basePath string, packageName protogen.GoPackageName) string {
	// Try to get custom path from options
	customPath := g.getCustomPath(method)

	// If we have both base path and custom path, combine them
	if basePath != "" && customPath != "" {
		// Ensure proper path joining
		basePath = strings.TrimSuffix(basePath, "/")
		if !strings.HasPrefix(customPath, "/") {
			customPath = "/" + customPath
		}
		return basePath + customPath
	}

	// If only custom path, use it
	if customPath != "" {
		return customPath
	}

	// Generate default path
	if basePath != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(basePath, "/"), camelToSnake(method.GoName))
	}

	return fmt.Sprintf("/%s/%s", packageName, camelToSnake(method.GoName))
}

// getCustomPath extracts custom HTTP path from method options
func (g *Generator) getCustomPath(method *protogen.Method) string {
	config := getMethodHTTPConfig(method)
	if config != nil && config.Path != "" {
		return config.Path
	}

	// Try to parse existing annotation format (temporary)
	return parseExistingAnnotation(method)
}

// getServiceBasePath extracts base path from service options
func (g *Generator) getServiceBasePath(service *protogen.Service) string {
	config := getServiceHTTPConfig(service)
	if config != nil && config.BasePath != "" {
		return config.BasePath
	}
	return ""
}

// Helper functions
func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func camelToSnake(s string) string {
	var result []byte
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(r+'a'-'A'))
		} else {
			result = append(result, byte(r))
		}
	}
	return string(result)
}
