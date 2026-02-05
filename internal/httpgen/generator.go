package httpgen

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// Generator handles HTTP code generation for protobuf services.
type Generator struct {
	plugin       *protogen.Plugin
	generateMock bool
	globalUnwrap *GlobalUnwrapInfo // Global unwrap info collected from all files
}

// Options configures the generator.
type Options struct {
	GenerateMock bool
}

// New creates a new HTTP generator.
func New(plugin *protogen.Plugin) *Generator {
	return &Generator{
		plugin: plugin,
	}
}

// NewWithOptions creates a new HTTP generator with options.
func NewWithOptions(plugin *protogen.Plugin, opts Options) *Generator {
	return &Generator{
		plugin:       plugin,
		generateMock: opts.GenerateMock,
	}
}

// Generate processes all files and generates HTTP handlers.
func (g *Generator) Generate() error {
	// Phase 1: Collect global unwrap information from ALL files first.
	// This enables cross-file unwrap resolution within the same package.
	var err error
	g.globalUnwrap, err = CollectGlobalUnwrapInfo(g.plugin.Files)
	if err != nil {
		return fmt.Errorf("collecting global unwrap info: %w", err)
	}

	// Phase 2: Generate code for each file
	for _, file := range g.plugin.Files {
		if !file.Generate {
			continue
		}
		if err = g.generateFile(file); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateFile(file *protogen.File) error {
	// Generate error implementation file if there are messages ending with Error
	if err := g.generateErrorImplFile(file); err != nil {
		return err
	}

	// Generate unwrap file if there are messages with unwrap annotations
	if err := g.generateUnwrapFile(file); err != nil {
		return err
	}

	if len(file.Services) == 0 {
		return nil
	}

	// Validate HTTP configurations - fail fast on any errors
	for _, service := range file.Services {
		if err := ValidateService(service); err != nil {
			return fmt.Errorf("validation error: %w", err)
		}
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

	// Generate mock file if requested
	if g.generateMock {
		if err := g.generateMockFile(file); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) generateHTTPFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_http.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)

	gf.P("import (")
	gf.P(`"context"`)
	gf.P()
	gf.P(`sebufhttp "github.com/SebastienMelki/sebuf/http"`)
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
	gf.P(
		"// Register",
		serviceName,
		"Server registers the HTTP handlers for service ",
		serviceName,
		" to the given mux.",
	)
	gf.P("func Register", serviceName, "Server(server ", serviceName, "Server, opts ...ServerOption) error {")
	gf.P("config := getConfiguration(opts...)")
	gf.P()

	// Get service-level base path if configured
	basePath := g.getServiceBasePath(service)

	// Get service-level headers
	gf.P("serviceHeaders := get", serviceName, "Headers()")
	gf.P()

	for i, method := range service.Methods {
		httpPath := g.getMethodPath(method, basePath, file.GoPackageName)
		httpMethod := g.getHTTPMethod(method)

		handlerName := fmt.Sprintf("%sHandler", annotations.LowerFirst(method.GoName))
		if i == 0 {
			gf.P("methodHeaders := get", method.GoName, "Headers()")
		} else {
			gf.P("methodHeaders = get", method.GoName, "Headers()")
		}
		gf.P(handlerName, " := BindingMiddleware[", method.Input.GoIdent, "](")
		gf.P("genericHandler(server.", method.GoName, ", config.errorHandler), serviceHeaders, methodHeaders,")
		gf.P(
			annotations.LowerFirst(method.GoName),
			"PathParams, ",
			annotations.LowerFirst(method.GoName),
			"QueryParams,",
		)
		gf.P(`"`, httpMethod, `", config.errorHandler,`)
		gf.P(")")
		gf.P()
		gf.P(`config.mux.Handle("`, httpMethod, ` `, httpPath, `", `, handlerName, `)`)
		gf.P()
	}

	gf.P("return nil")
	gf.P("}")
	gf.P()

	// Generate header getter functions
	if err := g.generateHeaderGetters(gf, service); err != nil {
		return err
	}

	// Generate path and query param configs
	if err := g.generateParamConfigs(gf, service); err != nil {
		return err
	}

	return nil
}

//nolint:funlen // This function generates a lot of boilerplate code
func (g *Generator) generateBindingFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_http_binding.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)

	gf.P("import (")
	gf.P(`"bytes"`)
	gf.P(`"context"`)
	gf.P(`"encoding/json"`)
	gf.P(`"errors"`)
	gf.P(`"fmt"`)
	gf.P(`"io"`)
	gf.P(`"net/http"`)
	gf.P(`"strconv"`)
	gf.P(`"strings"`)
	gf.P(`"sync"`)
	gf.P(`"time"`)
	gf.P(`"unicode/utf8"`)
	gf.P()
	gf.P(`protovalidate "buf.build/go/protovalidate"`)
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(`"google.golang.org/protobuf/proto"`)
	gf.P(`"google.golang.org/protobuf/reflect/protoreflect"`)
	gf.P()
	gf.P(`sebufhttp "github.com/SebastienMelki/sebuf/http"`)
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

	// PathParamConfig type
	gf.P("// PathParamConfig defines configuration for a path parameter.")
	gf.P("type PathParamConfig struct {")
	gf.P("URLParam  string // Parameter name in URL path")
	gf.P("FieldName string // Proto field name to bind to")
	gf.P("}")
	gf.P()

	// QueryParamConfig type
	gf.P("// QueryParamConfig defines configuration for a query parameter.")
	gf.P("type QueryParamConfig struct {")
	gf.P("QueryName string // Parameter name in query string")
	gf.P("FieldName string // Proto field name to bind to")
	gf.P("Required  bool   // Whether this parameter is required")
	gf.P("}")
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
	gf.P("// and validates them using protovalidate and header validation.")
	gf.P("// It supports path parameters, query parameters, and request body binding.")
	gf.P("func BindingMiddleware[Req any](next http.Handler, serviceHeaders, methodHeaders []*sebufhttp.Header,")
	gf.P(
		"pathParams []PathParamConfig, queryParams []QueryParamConfig, httpMethod string, errorHandler ErrorHandler) http.Handler {",
	)
	gf.P("return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {")
	gf.P("// Validate headers first")
	gf.P("if validationErr := validateHeaders(r, serviceHeaders, methodHeaders); validationErr != nil {")
	gf.P("writeErrorWithHandler(w, r, validationErr, errorHandler)")
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("toBind := new(Req)")
	gf.P()
	gf.P("// Bind path parameters")
	gf.P("if msg, ok := any(toBind).(proto.Message); ok {")
	gf.P("if err := bindPathParams(r, msg, pathParams); err != nil {")
	gf.P("writeErrorWithHandler(w, r, err, errorHandler)")
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("// Bind query parameters")
	gf.P("if err := bindQueryParams(r, msg, queryParams); err != nil {")
	gf.P("writeErrorWithHandler(w, r, err, errorHandler)")
	gf.P("return")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("// Bind body only for POST, PUT, PATCH methods")
	gf.P(`if httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH" {`)
	gf.P("err := bindDataBasedOnContentType(r, toBind)")
	gf.P("if err != nil {")
	gf.P("// For binding errors, return a simple validation error")
	gf.P("validationErr := &sebufhttp.ValidationError{")
	gf.P("Violations: []*sebufhttp.FieldViolation{")
	gf.P("{")
	gf.P(`Field: "body",`)
	gf.P(`Description: fmt.Sprintf("failed to parse request body: %v", err),`)
	gf.P("},")
	gf.P("},")
	gf.P("}")
	gf.P("writeErrorWithHandler(w, r, validationErr, errorHandler)")
	gf.P("return")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("// Validate the complete message")
	gf.P("if msg, ok := any(toBind).(proto.Message); ok {")
	gf.P("if err := ValidateMessage(msg); err != nil {")
	gf.P("writeErrorWithHandler(w, r, convertProtovalidateError(err), errorHandler)")
	gf.P("return")
	gf.P("}")
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
	gf.P("// Check for custom JSON unmarshaler (unwrap support)")
	gf.P("if unmarshaler, ok := any(toBind).(json.Unmarshaler); ok {")
	gf.P("return unmarshaler.UnmarshalJSON(bodyBytes)")
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

	// bindPathParams function - binds URL path parameters to proto message fields
	gf.P("// bindPathParams binds URL path parameters to proto message fields using Go 1.22+ PathValue.")
	gf.P(
		"func bindPathParams(r *http.Request, msg proto.Message, params []PathParamConfig) *sebufhttp.ValidationError {",
	)
	gf.P("if len(params) == 0 {")
	gf.P("return nil")
	gf.P("}")
	gf.P()
	gf.P("reflectMsg := msg.ProtoReflect()")
	gf.P("fields := reflectMsg.Descriptor().Fields()")
	gf.P()
	gf.P("for _, param := range params {")
	gf.P("value := r.PathValue(param.URLParam)")
	gf.P("if value == \"\" {")
	gf.P("return &sebufhttp.ValidationError{")
	gf.P("Violations: []*sebufhttp.FieldViolation{{")
	gf.P(`Field: param.FieldName,`)
	gf.P(`Description: fmt.Sprintf("missing required path parameter: %s", param.URLParam),`)
	gf.P("}},")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("field := fields.ByName(protoreflect.Name(param.FieldName))")
	gf.P("if field == nil {")
	gf.P("continue // Field not found, skip")
	gf.P("}")
	gf.P()
	gf.P("convertedValue, err := convertStringToFieldValue(value, field.Kind())")
	gf.P("if err != nil {")
	gf.P("return &sebufhttp.ValidationError{")
	gf.P("Violations: []*sebufhttp.FieldViolation{{")
	gf.P(`Field: param.FieldName,`)
	gf.P(`Description: fmt.Sprintf("invalid value for path parameter %s: %v", param.URLParam, err),`)
	gf.P("}},")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("reflectMsg.Set(field, convertedValue)")
	gf.P("}")
	gf.P()
	gf.P("return nil")
	gf.P("}")
	gf.P()

	// bindQueryParams function - binds URL query parameters to proto message fields
	gf.P("// bindQueryParams binds URL query parameters to proto message fields.")
	gf.P(
		"func bindQueryParams(r *http.Request, msg proto.Message, params []QueryParamConfig) *sebufhttp.ValidationError {",
	)
	gf.P("if len(params) == 0 {")
	gf.P("return nil")
	gf.P("}")
	gf.P()
	gf.P("query := r.URL.Query()")
	gf.P("reflectMsg := msg.ProtoReflect()")
	gf.P("fields := reflectMsg.Descriptor().Fields()")
	gf.P()
	gf.P("for _, param := range params {")
	gf.P("values := query[param.QueryName]")
	gf.P("if len(values) == 0 {")
	gf.P("if param.Required {")
	gf.P("return &sebufhttp.ValidationError{")
	gf.P("Violations: []*sebufhttp.FieldViolation{{")
	gf.P(`Field: param.FieldName,`)
	gf.P(`Description: fmt.Sprintf("missing required query parameter: %s", param.QueryName),`)
	gf.P("}},")
	gf.P("}")
	gf.P("}")
	gf.P("continue")
	gf.P("}")
	gf.P()
	gf.P("field := fields.ByName(protoreflect.Name(param.FieldName))")
	gf.P("if field == nil {")
	gf.P("continue // Field not found, skip")
	gf.P("}")
	gf.P()
	gf.P("// Handle repeated fields (arrays)")
	gf.P("if field.IsList() {")
	gf.P("list := reflectMsg.Mutable(field).List()")
	gf.P("for _, v := range values {")
	gf.P("converted, err := convertStringToFieldValue(v, field.Kind())")
	gf.P("if err != nil {")
	gf.P("return &sebufhttp.ValidationError{")
	gf.P("Violations: []*sebufhttp.FieldViolation{{")
	gf.P(`Field: param.FieldName,`)
	gf.P(`Description: fmt.Sprintf("invalid value for query parameter %s: %v", param.QueryName, err),`)
	gf.P("}},")
	gf.P("}")
	gf.P("}")
	gf.P("list.Append(converted)")
	gf.P("}")
	gf.P("} else {")
	gf.P("converted, err := convertStringToFieldValue(values[0], field.Kind())")
	gf.P("if err != nil {")
	gf.P("return &sebufhttp.ValidationError{")
	gf.P("Violations: []*sebufhttp.FieldViolation{{")
	gf.P(`Field: param.FieldName,`)
	gf.P(`Description: fmt.Sprintf("invalid value for query parameter %s: %v", param.QueryName, err),`)
	gf.P("}},")
	gf.P("}")
	gf.P("}")
	gf.P("reflectMsg.Set(field, converted)")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("return nil")
	gf.P("}")
	gf.P()

	// convertStringToFieldValue function - converts string values to protoreflect.Value
	gf.P("// convertStringToFieldValue converts a string value to the appropriate protoreflect.Value.")
	gf.P("func convertStringToFieldValue(value string, kind protoreflect.Kind) (protoreflect.Value, error) {")
	gf.P("switch kind {")
	gf.P("case protoreflect.StringKind:")
	gf.P("return protoreflect.ValueOfString(value), nil")
	gf.P("case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:")
	gf.P("v, err := strconv.ParseInt(value, 10, 32)")
	gf.P("if err != nil {")
	gf.P("return protoreflect.Value{}, err")
	gf.P("}")
	gf.P("return protoreflect.ValueOfInt32(int32(v)), nil")
	gf.P("case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:")
	gf.P("v, err := strconv.ParseInt(value, 10, 64)")
	gf.P("if err != nil {")
	gf.P("return protoreflect.Value{}, err")
	gf.P("}")
	gf.P("return protoreflect.ValueOfInt64(v), nil")
	gf.P("case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:")
	gf.P("v, err := strconv.ParseUint(value, 10, 32)")
	gf.P("if err != nil {")
	gf.P("return protoreflect.Value{}, err")
	gf.P("}")
	gf.P("return protoreflect.ValueOfUint32(uint32(v)), nil")
	gf.P("case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:")
	gf.P("v, err := strconv.ParseUint(value, 10, 64)")
	gf.P("if err != nil {")
	gf.P("return protoreflect.Value{}, err")
	gf.P("}")
	gf.P("return protoreflect.ValueOfUint64(v), nil")
	gf.P("case protoreflect.BoolKind:")
	gf.P("v, err := strconv.ParseBool(value)")
	gf.P("if err != nil {")
	gf.P("return protoreflect.Value{}, err")
	gf.P("}")
	gf.P("return protoreflect.ValueOfBool(v), nil")
	gf.P("case protoreflect.FloatKind:")
	gf.P("v, err := strconv.ParseFloat(value, 32)")
	gf.P("if err != nil {")
	gf.P("return protoreflect.Value{}, err")
	gf.P("}")
	gf.P("return protoreflect.ValueOfFloat32(float32(v)), nil")
	gf.P("case protoreflect.DoubleKind:")
	gf.P("v, err := strconv.ParseFloat(value, 64)")
	gf.P("if err != nil {")
	gf.P("return protoreflect.Value{}, err")
	gf.P("}")
	gf.P("return protoreflect.ValueOfFloat64(v), nil")
	gf.P("default:")
	gf.P(`return protoreflect.Value{}, fmt.Errorf("unsupported field type: %v", kind)`)
	gf.P("}")
	gf.P("}")
	gf.P()

	// genericHandler function
	gf.P(
		"func genericHandler[Req any, Res any](serve func(context.Context, Req) (Res, error), errorHandler ErrorHandler) http.HandlerFunc {",
	)
	gf.P("return func(w http.ResponseWriter, r *http.Request) {")
	gf.P("request := getRequest[Req](r.Context())")
	gf.P()
	gf.P("response, err := serve(r.Context(), request)")
	gf.P("if err != nil {")
	gf.P("// Check if error is already a proto.Message (e.g., custom proto error types)")
	gf.P("// If so, pass it directly - defaultErrorResponse will preserve its structure")
	gf.P("if _, ok := err.(proto.Message); ok {")
	gf.P("writeErrorWithHandler(w, r, err, errorHandler)")
	gf.P("return")
	gf.P("}")
	gf.P("errorMsg := &sebufhttp.Error{")
	gf.P("Message: err.Error(),")
	gf.P("}")
	gf.P("writeErrorWithHandler(w, r, errorMsg, errorHandler)")
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("responseBytes, err := marshalResponse(r, response)")
	gf.P("if err != nil {")
	gf.P("errorMsg := &sebufhttp.Error{")
	gf.P("Message: fmt.Sprintf(\"failed to marshal response: %v\", err),")
	gf.P("}")
	gf.P("writeErrorWithHandler(w, r, errorMsg, errorHandler)")
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("_, err = w.Write(responseBytes)")
	gf.P("if err != nil {")
	gf.P("errorMsg := &sebufhttp.Error{")
	gf.P("Message: fmt.Sprintf(\"failed to write response: %v\", err),")
	gf.P("}")
	gf.P("writeErrorWithHandler(w, r, errorMsg, errorHandler)")
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
	gf.P("// Check for custom JSON marshaler (unwrap support)")
	gf.P("if marshaler, ok := response.(json.Marshaler); ok {")
	gf.P("return marshaler.MarshalJSON()")
	gf.P("}")
	gf.P("return protojson.Marshal(msg)")
	gf.P("case BinaryContentType, ProtoContentType:")
	gf.P("return proto.Marshal(msg)")
	gf.P("default:")
	gf.P(`return nil, fmt.Errorf("unsupported content type: %s", contentType)`)
	gf.P("}")
	gf.P("}")
	gf.P()

	// Generate error response helpers
	g.generateErrorResponseFunctions(gf)

	// Generate validation support
	g.generateValidationFunctions(gf)

	// Generate header validation support
	g.generateHeaderValidationFunctions(gf)

	return nil
}

func (g *Generator) generateConfigFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_http_config.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)
	g.generateConfigImports(gf)
	g.generateErrorHandlerType(gf)
	g.generateServerOptionType(gf)
	g.generateServerConfigurationStruct(gf)
	g.generateConfigFunctions(gf)
	g.generateServerOptions(gf)

	return nil
}

func (g *Generator) generateConfigImports(gf *protogen.GeneratedFile) {
	gf.P("import (")
	gf.P(`"net/http"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/proto"`)
	gf.P(")")
	gf.P()
}

func (g *Generator) generateErrorHandlerType(gf *protogen.GeneratedFile) {
	gf.P("// ErrorHandler is called when an error occurs.")
	gf.P("//")
	gf.P("// You can:")
	gf.P("//   - Set headers via w.Header().Set(...)")
	gf.P("//   - Set status code via w.WriteHeader(...)")
	gf.P("//   - Return a proto.Message to be marshaled as the response body")
	gf.P("//   - Return nil to use the default error response (ValidationError or Error)")
	gf.P("//")
	gf.P("// If you write directly to w (via w.Write()), the response is considered")
	gf.P("// complete and no further writing occurs.")
	gf.P("//")
	gf.P("// Use errors.As() to inspect error types: *sebufhttp.ValidationError or *sebufhttp.Error")
	gf.P("type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error) proto.Message")
	gf.P()
}

func (g *Generator) generateServerOptionType(gf *protogen.GeneratedFile) {
	gf.P("// ServerOption configures a Server")
	gf.P("type ServerOption func(c *serverConfiguration)")
	gf.P()
}

func (g *Generator) generateServerConfigurationStruct(gf *protogen.GeneratedFile) {
	gf.P("type serverConfiguration struct {")
	gf.P("mux *http.ServeMux")
	gf.P("withMux bool")
	gf.P("errorHandler ErrorHandler")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateConfigFunctions(gf *protogen.GeneratedFile) {
	gf.P("func getDefaultConfiguration() *serverConfiguration {")
	gf.P("return &serverConfiguration{")
	gf.P("mux: http.DefaultServeMux,")
	gf.P("withMux: false,")
	gf.P("}")
	gf.P("}")
	gf.P()

	gf.P("func getConfiguration(options ...ServerOption) *serverConfiguration {")
	gf.P("configuration := getDefaultConfiguration()")
	gf.P("for _, option := range options {")
	gf.P("option(configuration)")
	gf.P("}")
	gf.P("return configuration")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateServerOptions(gf *protogen.GeneratedFile) {
	gf.P("// WithMux configures the Server to use the given ServeMux")
	gf.P("func WithMux(mux *http.ServeMux) ServerOption {")
	gf.P("return func(c *serverConfiguration) {")
	gf.P("c.mux = mux")
	gf.P("c.withMux = true")
	gf.P("}")
	gf.P("}")
	gf.P()

	gf.P("// WithErrorHandler configures a custom error handler for the server.")
	gf.P("func WithErrorHandler(handler ErrorHandler) ServerOption {")
	gf.P("return func(c *serverConfiguration) {")
	gf.P("c.errorHandler = handler")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) writeHeader(gf *protogen.GeneratedFile, file *protogen.File) {
	gf.P("// Code generated by protoc-gen-go-http. DO NOT EDIT.")
	gf.P("// source: ", file.Desc.Path())
	gf.P()
	gf.P("package ", file.GoPackageName)
	gf.P()
}

// getMethodPath determines the HTTP path for a method.
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

// getCustomPath extracts custom HTTP path from method options.
func (g *Generator) getCustomPath(method *protogen.Method) string {
	config := annotations.GetMethodHTTPConfig(method)
	if config != nil && config.Path != "" {
		return config.Path
	}

	return ""
}

// getServiceBasePath extracts base path from service options.
func (g *Generator) getServiceBasePath(service *protogen.Service) string {
	return annotations.GetServiceBasePath(service)
}

// getHTTPMethod returns the HTTP method for a method. Defaults to POST for backward compatibility.
func (g *Generator) getHTTPMethod(method *protogen.Method) string {
	config := annotations.GetMethodHTTPConfig(method)
	if config != nil && config.Method != "" {
		return config.Method
	}
	return "POST"
}

// getPathParams extracts path parameter names from method configuration.
func (g *Generator) getPathParams(method *protogen.Method) []string {
	config := annotations.GetMethodHTTPConfig(method)
	if config != nil {
		return config.PathParams
	}
	return nil
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

// generateErrorResponseFunctions generates error response helper functions.
func (g *Generator) generateErrorResponseFunctions(gf *protogen.GeneratedFile) {
	g.generateResponseCaptureType(gf)
	g.generateWriteProtoMessageResponseFunc(gf)
	g.generateWriteValidationErrorResponseFunc(gf)
	g.generateWriteValidationErrorFunc(gf)
	g.generateWriteErrorResponseFunc(gf)
	g.generateConvertProtovalidateErrorFunc(gf)
	g.generateDefaultErrorResponseFunc(gf)
	g.generateDefaultErrorStatusCodeFunc(gf)
	g.generateWriteErrorWithHandlerFunc(gf)
	g.generateWriteResponseBodyFunc(gf)
}

// generateWriteProtoMessageResponseFunc generates a helper function for writing protobuf messages as responses.
func (g *Generator) generateWriteProtoMessageResponseFunc(gf *protogen.GeneratedFile) {
	gf.P("// writeProtoMessageResponse writes a protobuf message as an HTTP response")
	gf.P(
		"func writeProtoMessageResponse(w http.ResponseWriter, r *http.Request, msg proto.Message, statusCode int, fallbackMsg string) {",
	)
	gf.P(`contentType := r.Header.Get("Content-Type")`)
	gf.P("if contentType == \"\" {")
	gf.P("contentType = JSONContentType")
	gf.P("}")
	gf.P()
	gf.P("var responseBytes []byte")
	gf.P("var err error")
	gf.P()
	gf.P("switch filterFlags(contentType) {")
	gf.P("case JSONContentType:")
	gf.P("responseBytes, err = protojson.Marshal(msg)")
	gf.P("case BinaryContentType, ProtoContentType:")
	gf.P("responseBytes, err = proto.Marshal(msg)")
	gf.P("default:")
	gf.P("// Default to JSON for error responses")
	gf.P("responseBytes, err = protojson.Marshal(msg)")
	gf.P("}")
	gf.P()
	gf.P("if err != nil {")
	gf.P("// Fallback to plain text error if marshaling fails")
	gf.P("http.Error(w, fallbackMsg, statusCode)")
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("w.WriteHeader(statusCode)")
	gf.P("_, _ = w.Write(responseBytes)")
	gf.P("}")
	gf.P()
}

// generateWriteValidationErrorResponseFunc generates the writeValidationErrorResponse function.
func (g *Generator) generateWriteValidationErrorResponseFunc(gf *protogen.GeneratedFile) {
	gf.P("// writeValidationErrorResponse writes a ValidationError as a response")
	gf.P(
		"func writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErr *sebufhttp.ValidationError) {",
	)
	gf.P(`writeProtoMessageResponse(w, r, validationErr, http.StatusBadRequest, "validation failed")`)
	gf.P("}")
	gf.P()
}

// generateWriteValidationErrorFunc generates the writeValidationError function for protovalidate errors.
func (g *Generator) generateWriteValidationErrorFunc(gf *protogen.GeneratedFile) {
	gf.P("// writeValidationError converts a protovalidate error to ValidationError and writes it as response")
	gf.P("func writeValidationError(w http.ResponseWriter, r *http.Request, err error) {")
	gf.P("validationErr := convertProtovalidateError(err)")
	gf.P("writeValidationErrorResponse(w, r, validationErr)")
	gf.P("}")
	gf.P()
}

// generateWriteErrorResponseFunc generates the writeErrorResponse function.
func (g *Generator) generateWriteErrorResponseFunc(gf *protogen.GeneratedFile) {
	gf.P("// writeErrorResponse writes an Error as a response")
	gf.P("func writeErrorResponse(w http.ResponseWriter, r *http.Request, errorMsg *sebufhttp.Error) {")
	gf.P(`writeProtoMessageResponse(w, r, errorMsg, http.StatusInternalServerError, "internal server error")`)
	gf.P("}")
	gf.P()
}

// generateResponseCaptureType generates the responseCapture type for tracking handler writes.
func (g *Generator) generateResponseCaptureType(gf *protogen.GeneratedFile) {
	gf.P("// responseCapture wraps ResponseWriter to track if Write or WriteHeader was called")
	gf.P("type responseCapture struct {")
	gf.P("http.ResponseWriter")
	gf.P("wroteHeader bool")
	gf.P("written bool")
	gf.P("}")
	gf.P()
	gf.P("func (rc *responseCapture) WriteHeader(code int) {")
	gf.P("rc.wroteHeader = true")
	gf.P("rc.ResponseWriter.WriteHeader(code)")
	gf.P("}")
	gf.P()
	gf.P("func (rc *responseCapture) Write(b []byte) (int, error) {")
	gf.P("rc.written = true")
	gf.P("return rc.ResponseWriter.Write(b)")
	gf.P("}")
	gf.P()
}

// generateConvertProtovalidateErrorFunc generates a function to convert protovalidate errors.
func (g *Generator) generateConvertProtovalidateErrorFunc(gf *protogen.GeneratedFile) {
	gf.P("// convertProtovalidateError converts a protovalidate error to ValidationError")
	gf.P("func convertProtovalidateError(err error) *sebufhttp.ValidationError {")
	gf.P("validationErr := &sebufhttp.ValidationError{}")
	gf.P()
	gf.P("// Handle protovalidate.ValidationError")
	gf.P("var valErr *protovalidate.ValidationError")
	gf.P("if errors.As(err, &valErr) {")
	gf.P("for _, violation := range valErr.Violations {")
	g.generateFieldPathExtraction(gf)
	gf.P("validationErr.Violations = append(validationErr.Violations, &sebufhttp.FieldViolation{")
	gf.P("Field: fieldPath,")
	gf.P("Description: violation.Proto.GetMessage(),")
	gf.P("})")
	gf.P("}")
	gf.P("} else {")
	gf.P("// Shouldn't happen, but handle as generic error")
	gf.P("validationErr.Violations = append(validationErr.Violations, &sebufhttp.FieldViolation{")
	gf.P(`Field: "unknown",`)
	gf.P("Description: err.Error(),")
	gf.P("})")
	gf.P("}")
	gf.P()
	gf.P("return validationErr")
	gf.P("}")
	gf.P()
}

// generateDefaultErrorResponseFunc generates the defaultErrorResponse helper function.
func (g *Generator) generateDefaultErrorResponseFunc(gf *protogen.GeneratedFile) {
	gf.P("// defaultErrorResponse returns the appropriate error response message based on error type")
	gf.P("func defaultErrorResponse(err error) proto.Message {")
	gf.P("var valErr *sebufhttp.ValidationError")
	gf.P("if errors.As(err, &valErr) {")
	gf.P("return valErr")
	gf.P("}")
	gf.P("var handlerErr *sebufhttp.Error")
	gf.P("if errors.As(err, &handlerErr) {")
	gf.P("return handlerErr")
	gf.P("}")
	gf.P("// Check if error is already a proto.Message (e.g., custom proto error types)")
	gf.P("if protoErr, ok := err.(proto.Message); ok {")
	gf.P("return protoErr")
	gf.P("}")
	gf.P("return &sebufhttp.Error{Message: err.Error()}")
	gf.P("}")
	gf.P()
}

// generateDefaultErrorStatusCodeFunc generates the defaultErrorStatusCode helper function.
func (g *Generator) generateDefaultErrorStatusCodeFunc(gf *protogen.GeneratedFile) {
	gf.P("// defaultErrorStatusCode returns the appropriate HTTP status code based on error type")
	gf.P("func defaultErrorStatusCode(err error) int {")
	gf.P("var valErr *sebufhttp.ValidationError")
	gf.P("if errors.As(err, &valErr) {")
	gf.P("return http.StatusBadRequest")
	gf.P("}")
	gf.P("return http.StatusInternalServerError")
	gf.P("}")
	gf.P()
}

// generateWriteErrorWithHandlerFunc generates the writeErrorWithHandler function.
func (g *Generator) generateWriteErrorWithHandlerFunc(gf *protogen.GeneratedFile) {
	gf.P("// writeErrorWithHandler calls custom handler if set, then marshals response")
	gf.P("func writeErrorWithHandler(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler) {")
	gf.P("var response proto.Message")
	gf.P("var capture *responseCapture")
	gf.P()
	gf.P("if handler != nil {")
	gf.P("capture = &responseCapture{ResponseWriter: w}")
	gf.P("response = handler(capture, r, err)")
	gf.P("if capture.written {")
	gf.P("return // Handler wrote directly, done")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("// Determine response if handler didn't provide one")
	gf.P("if response == nil {")
	gf.P("response = defaultErrorResponse(err)")
	gf.P("}")
	gf.P()
	gf.P("// Determine status code")
	gf.P("statusCode := defaultErrorStatusCode(err)")
	gf.P()
	gf.P("// If handler already set status, don't set it again")
	gf.P("if capture != nil && capture.wroteHeader {")
	gf.P("// Handler set status, just write the body")
	gf.P("writeResponseBody(w, r, response)")
	gf.P("return")
	gf.P("}")
	gf.P()
	gf.P("// Write full response with status code")
	gf.P(`writeProtoMessageResponse(w, r, response, statusCode, "error processing request")`)
	gf.P("}")
	gf.P()
}

func (g *Generator) generateWriteResponseBodyFunc(gf *protogen.GeneratedFile) {
	gf.P("// writeResponseBody writes the response body without setting status code")
	gf.P("func writeResponseBody(w http.ResponseWriter, r *http.Request, msg proto.Message) {")
	gf.P(`contentType := r.Header.Get("Content-Type")`)
	gf.P("if contentType == \"\" {")
	gf.P("contentType = JSONContentType")
	gf.P("}")
	gf.P()
	gf.P("var responseBytes []byte")
	gf.P("var err error")
	gf.P()
	gf.P("switch filterFlags(contentType) {")
	gf.P("case JSONContentType:")
	gf.P("responseBytes, err = protojson.Marshal(msg)")
	gf.P("case BinaryContentType, ProtoContentType:")
	gf.P("responseBytes, err = proto.Marshal(msg)")
	gf.P("default:")
	gf.P("responseBytes, err = protojson.Marshal(msg)")
	gf.P("}")
	gf.P()
	gf.P("if err != nil {")
	gf.P("return // Can't write anything meaningful")
	gf.P("}")
	gf.P()
	gf.P("_, _ = w.Write(responseBytes)")
	gf.P("}")
	gf.P()
}

// generateFieldPathExtraction generates the field path extraction logic.
func (g *Generator) generateFieldPathExtraction(gf *protogen.GeneratedFile) {
	gf.P("// Extract field path from violation")
	gf.P("fieldPath := \"\"")
	gf.P("if violation.Proto != nil && violation.Proto.GetField() != nil {")
	gf.P("elements := violation.Proto.GetField().GetElements()")
	gf.P("if len(elements) > 0 {")
	gf.P("fieldPath = elements[0].GetFieldName()")
	gf.P("for i := 1; i < len(elements); i++ {")
	gf.P("fieldPath += \".\" + elements[i].GetFieldName()")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P("if fieldPath == \"\" {")
	gf.P("fieldPath = \"unknown\"")
	gf.P("}")
	gf.P()
}

// generateValidationFunctions generates the validation support code.
func (g *Generator) generateValidationFunctions(gf *protogen.GeneratedFile) {
	// Global validator instance
	gf.P("var (")
	gf.P("// Global validator instance - created once and reused")
	gf.P("validatorOnce sync.Once")
	gf.P("validator protovalidate.Validator")
	gf.P("validatorErr error")
	gf.P(")")
	gf.P()

	// getValidator function
	gf.P("// getValidator returns a cached validator instance")
	gf.P("func getValidator() (protovalidate.Validator, error) {")
	gf.P("validatorOnce.Do(func() {")
	gf.P("validator, validatorErr = protovalidate.New()")
	gf.P("})")
	gf.P("return validator, validatorErr")
	gf.P("}")
	gf.P()

	// ValidateMessage function
	gf.P("// ValidateMessage validates a protobuf message using protovalidate")
	gf.P("func ValidateMessage(msg proto.Message) error {")
	gf.P("// Get cached validator")
	gf.P("v, err := getValidator()")
	gf.P("if err != nil {")
	gf.P("// If we can't create a validator, log and continue")
	gf.P("// This allows the service to run even if validation setup fails")
	gf.P("return nil")
	gf.P("}")
	gf.P()
	gf.P("// Validate the message and return any error")
	gf.P("return v.Validate(msg)")
	gf.P("}")
	gf.P()
}

// generateHeaderValidationFunctions generates header validation support code.
func (g *Generator) generateHeaderValidationFunctions(gf *protogen.GeneratedFile) {
	g.generateValidateHeadersFunction(gf)
	g.generateValidateHeaderValueFunction(gf)
	g.generateTypeValidators(gf)
	g.generateFormatValidators(gf)
}

// generateValidateHeadersFunction generates the main header validation function.
func (g *Generator) generateValidateHeadersFunction(gf *protogen.GeneratedFile) {
	gf.P("// validateHeaders validates required headers for a service and method")
	gf.P("// Returns a ValidationError if any required headers are missing or invalid")
	gf.P(
		"func validateHeaders(r *http.Request, serviceHeaders, methodHeaders []*sebufhttp.Header) *sebufhttp.ValidationError {",
	)
	g.generateHeaderMergeLogic(gf)
	g.generateHeaderValidationLoop(gf)
	g.generateValidationErrorReturn(gf)
	gf.P("}")
	gf.P()
}

// generateHeaderMergeLogic generates the logic to merge service and method headers.
func (g *Generator) generateHeaderMergeLogic(gf *protogen.GeneratedFile) {
	gf.P("// Merge service and method headers, with method headers taking precedence")
	gf.P("allHeaders := make(map[string]*sebufhttp.Header)")
	gf.P()
	gf.P("// Add service headers first")
	gf.P("for _, header := range serviceHeaders {")
	gf.P("if header.GetRequired() {")
	gf.P("allHeaders[strings.ToLower(header.GetName())] = header")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("// Add method headers (override service headers if same name)")
	gf.P("for _, header := range methodHeaders {")
	gf.P("if header.GetRequired() {")
	gf.P("allHeaders[strings.ToLower(header.GetName())] = header")
	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateHeaderValidationLoop generates the main header validation loop.
func (g *Generator) generateHeaderValidationLoop(gf *protogen.GeneratedFile) {
	gf.P("// Collect all validation violations")
	gf.P("var violations []*sebufhttp.FieldViolation")
	gf.P()
	gf.P("// Validate each required header")
	gf.P("for _, headerSpec := range allHeaders {")
	gf.P("value := r.Header.Get(headerSpec.GetName())")
	gf.P("if value == \"\" {")
	gf.P("violations = append(violations, &sebufhttp.FieldViolation{")
	gf.P("Field: headerSpec.GetName(),")
	gf.P(`Description: fmt.Sprintf("required header '%s' is missing", headerSpec.GetName()),`)
	gf.P("})")
	gf.P("continue")
	gf.P("}")
	gf.P()
	gf.P("if err := validateHeaderValue(headerSpec, value); err != nil {")
	gf.P("violations = append(violations, &sebufhttp.FieldViolation{")
	gf.P("Field: headerSpec.GetName(),")
	gf.P(`Description: fmt.Sprintf("header '%s' validation failed: %v", headerSpec.GetName(), err),`)
	gf.P("})")
	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateValidationErrorReturn generates the validation error return logic.
func (g *Generator) generateValidationErrorReturn(gf *protogen.GeneratedFile) {
	gf.P("// Return ValidationError if there are violations")
	gf.P("if len(violations) > 0 {")
	gf.P("return &sebufhttp.ValidationError{")
	gf.P("Violations: violations,")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("return nil")
}

// generateValidateHeaderValueFunction generates the header value validation function.
func (g *Generator) generateValidateHeaderValueFunction(gf *protogen.GeneratedFile) {
	gf.P("// validateHeaderValue validates a single header value against its specification")
	gf.P("func validateHeaderValue(headerSpec *sebufhttp.Header, value string) error {")
	gf.P("headerType := headerSpec.GetType()")
	gf.P("format := headerSpec.GetFormat()")
	gf.P()
	gf.P("// Validate based on type")
	gf.P("switch headerType {")
	gf.P("case \"string\":")
	gf.P("return validateStringHeader(value, format)")
	gf.P("case \"integer\":")
	gf.P("return validateIntegerHeader(value)")
	gf.P("case \"number\":")
	gf.P("return validateNumberHeader(value)")
	gf.P("case \"boolean\":")
	gf.P("return validateBooleanHeader(value)")
	gf.P("case \"array\":")
	gf.P("return validateArrayHeader(value)")
	gf.P("default:")
	gf.P("// Default to string validation if type is not specified")
	gf.P("return validateStringHeader(value, format)")
	gf.P("}")
	gf.P("}")
	gf.P()
}

// generateTypeValidators generates type-specific validation functions.
func (g *Generator) generateTypeValidators(gf *protogen.GeneratedFile) {
	g.generateStringValidator(gf)
	g.generateNumericValidators(gf)
	g.generateArrayValidator(gf)
}

// generateStringValidator generates string header validation function.
func (g *Generator) generateStringValidator(gf *protogen.GeneratedFile) {
	gf.P("// validateStringHeader validates string headers with optional format validation")
	gf.P("func validateStringHeader(value, format string) error {")
	gf.P("if !utf8.ValidString(value) {")
	gf.P(`return fmt.Errorf("value is not valid UTF-8")`)
	gf.P("}")
	gf.P()
	gf.P("// Apply format-specific validation")
	gf.P("switch format {")
	gf.P("case \"uuid\":")
	gf.P("return validateUUIDFormat(value)")
	gf.P("case \"email\":")
	gf.P("return validateEmailFormat(value)")
	gf.P("case \"date-time\":")
	gf.P("return validateDateTimeFormat(value)")
	gf.P("case \"date\":")
	gf.P("return validateDateFormat(value)")
	gf.P("case \"time\":")
	gf.P("return validateTimeFormat(value)")
	gf.P("}")
	gf.P()
	gf.P("return nil")
	gf.P("}")
	gf.P()
}

// generateNumericValidators generates numeric header validation functions.
//
//nolint:dupl // Code generation patterns naturally have similar structure
func (g *Generator) generateNumericValidators(gf *protogen.GeneratedFile) {
	// Integer header validation
	gf.P("// validateIntegerHeader validates integer headers")
	gf.P("func validateIntegerHeader(value string) error {")
	gf.P("_, err := strconv.ParseInt(value, 10, 64)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("value is not a valid integer: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()

	// Number header validation
	gf.P("// validateNumberHeader validates numeric headers (float)")
	gf.P("func validateNumberHeader(value string) error {")
	gf.P("_, err := strconv.ParseFloat(value, 64)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("value is not a valid number: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()

	// Boolean header validation
	gf.P("// validateBooleanHeader validates boolean headers")
	gf.P("func validateBooleanHeader(value string) error {")
	gf.P("_, err := strconv.ParseBool(value)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("value is not a valid boolean: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()
}

// generateArrayValidator generates array header validation function.
func (g *Generator) generateArrayValidator(gf *protogen.GeneratedFile) {
	gf.P("// validateArrayHeader validates array headers (comma-separated values)")
	gf.P("func validateArrayHeader(value string) error {")
	gf.P("// Arrays are typically comma-separated values")
	gf.P("// Basic validation: ensure it's not empty")
	gf.P("if strings.TrimSpace(value) == \"\" {")
	gf.P(`return fmt.Errorf("array value cannot be empty")`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()
}

// generateFormatValidators generates format-specific validation functions.
func (g *Generator) generateFormatValidators(gf *protogen.GeneratedFile) {
	g.generateUUIDValidator(gf)
	g.generateEmailValidator(gf)
	g.generateDateTimeValidators(gf)
}

// generateUUIDValidator generates UUID format validation function.
func (g *Generator) generateUUIDValidator(gf *protogen.GeneratedFile) {
	gf.P("// validateUUIDFormat validates UUID format (basic check)")
	gf.P("func validateUUIDFormat(value string) error {")
	gf.P("// Basic UUID format check: 8-4-4-4-12 hex digits")
	gf.P("if len(value) != 36 {")
	gf.P(`return fmt.Errorf("UUID must be 36 characters long")`)
	gf.P("}")
	gf.P()
	gf.P("// Check for correct dash positions")
	gf.P("if value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {")
	gf.P(`return fmt.Errorf("invalid UUID format")`)
	gf.P("}")
	gf.P()
	gf.P("return nil")
	gf.P("}")
	gf.P()
}

// generateEmailValidator generates email format validation function.
func (g *Generator) generateEmailValidator(gf *protogen.GeneratedFile) {
	gf.P("// validateEmailFormat validates email format (basic check)")
	gf.P("func validateEmailFormat(value string) error {")
	gf.P("// Basic email format check")
	gf.P("if !strings.Contains(value, \"@\") {")
	gf.P(`return fmt.Errorf("invalid email format: missing @")`)
	gf.P("}")
	gf.P()
	gf.P("parts := strings.Split(value, \"@\")")
	gf.P("if len(parts) != 2 || parts[0] == \"\" || parts[1] == \"\" {")
	gf.P(`return fmt.Errorf("invalid email format")`)
	gf.P("}")
	gf.P()
	gf.P("return nil")
	gf.P("}")
	gf.P()
}

// generateDateTimeValidators generates date/time format validation functions.
//
//nolint:dupl // Code generation patterns naturally have similar structure
func (g *Generator) generateDateTimeValidators(gf *protogen.GeneratedFile) {
	gf.P("// validateDateTimeFormat validates RFC3339 date-time format")
	gf.P("func validateDateTimeFormat(value string) error {")
	gf.P("_, err := time.Parse(time.RFC3339, value)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("invalid date-time format, expected RFC3339: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()

	gf.P("// validateDateFormat validates date format (YYYY-MM-DD)")
	gf.P("func validateDateFormat(value string) error {")
	gf.P("_, err := time.Parse(\"2006-01-02\", value)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()

	gf.P("// validateTimeFormat validates time format (HH:MM:SS)")
	gf.P("func validateTimeFormat(value string) error {")
	gf.P("_, err := time.Parse(\"15:04:05\", value)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Errorf("invalid time format, expected HH:MM:SS: %w", err)`)
	gf.P("}")
	gf.P("return nil")
	gf.P("}")
	gf.P()
}

// generateHeaderGetters generates functions to get headers for service and methods.
func (g *Generator) generateHeaderGetters(gf *protogen.GeneratedFile, service *protogen.Service) error {
	// Generate service headers getter function
	serviceName := service.GoName
	gf.P("// get", serviceName, "Headers returns the service-level required headers for ", serviceName)
	gf.P("func get", serviceName, "Headers() []*sebufhttp.Header {")

	// Get actual service headers if they exist
	serviceHeaders := annotations.GetServiceHeaders(service)
	if len(serviceHeaders) > 0 {
		gf.P("return []*sebufhttp.Header{")
		for _, header := range serviceHeaders {
			g.generateHeaderLiteral(gf, header)
		}
		gf.P("}")
	} else {
		gf.P("return nil")
	}
	gf.P("}")
	gf.P()

	// Generate method headers getter functions
	for _, method := range service.Methods {
		gf.P("// get", method.GoName, "Headers returns the method-level required headers for ", method.GoName)
		gf.P("func get", method.GoName, "Headers() []*sebufhttp.Header {")

		// Get actual method headers if they exist
		methodHeaders := annotations.GetMethodHeaders(method)
		if len(methodHeaders) > 0 {
			gf.P("return []*sebufhttp.Header{")
			for _, header := range methodHeaders {
				g.generateHeaderLiteral(gf, header)
			}
			gf.P("}")
		} else {
			gf.P("return nil")
		}
		gf.P("}")
		gf.P()
	}

	return nil
}

// generateHeaderLiteral generates a header literal in Go code.
func (g *Generator) generateHeaderLiteral(gf *protogen.GeneratedFile, header *http.Header) {
	gf.P("{")
	gf.P(`Name: "`, header.GetName(), `",`)
	gf.P(`Description: "`, header.GetDescription(), `",`)
	gf.P(`Type: "`, header.GetType(), `",`)
	gf.P(`Required: `, strconv.FormatBool(header.GetRequired()), `,`)
	gf.P(`Format: "`, header.GetFormat(), `",`)
	gf.P(`Example: "`, header.GetExample(), `",`)
	gf.P(`Deprecated: `, strconv.FormatBool(header.GetDeprecated()), `,`)
	gf.P("},")
}

// generateParamConfigs generates path and query parameter configurations for each method.
func (g *Generator) generateParamConfigs(gf *protogen.GeneratedFile, service *protogen.Service) error {
	for _, method := range service.Methods {
		methodName := annotations.LowerFirst(method.GoName)

		// Generate path params config
		pathParams := g.getPathParams(method)
		gf.P("// ", methodName, "PathParams contains path parameter configuration for ", method.GoName)
		gf.P("var ", methodName, "PathParams = []PathParamConfig{")
		for _, param := range pathParams {
			gf.P("{URLParam: \"", param, "\", FieldName: \"", param, "\"},")
		}
		gf.P("}")
		gf.P()

		// Generate query params config
		queryParams := annotations.GetQueryParams(method.Input)
		gf.P("// ", methodName, "QueryParams contains query parameter configuration for ", method.GoName)
		gf.P("var ", methodName, "QueryParams = []QueryParamConfig{")
		for _, qp := range queryParams {
			gf.P(
				"{QueryName: \"",
				qp.ParamName,
				"\", FieldName: \"",
				qp.FieldName,
				"\", Required: ",
				strconv.FormatBool(qp.Required),
				"},",
			)
		}
		gf.P("}")
		gf.P()
	}

	return nil
}

func (g *Generator) generateErrorImplFile(file *protogen.File) error {
	// Find messages ending with "Error"
	var errorMessages []*protogen.Message
	for _, message := range file.Messages {
		if strings.HasSuffix(message.GoIdent.GoName, "Error") {
			errorMessages = append(errorMessages, message)
		}
	}

	// If no error messages found, skip generation
	if len(errorMessages) == 0 {
		return nil
	}

	// Generate error implementation file
	filename := file.GeneratedFilenamePrefix + "_error_impl.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)

	gf.P("import (")
	gf.P(`"fmt"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(")")
	gf.P()

	// Generate Error() methods for each error message
	for _, message := range errorMessages {
		g.generateErrorMethod(gf, message)
		gf.P()
	}

	return nil
}

func (g *Generator) generateErrorMethod(gf *protogen.GeneratedFile, message *protogen.Message) {
	typeName := message.GoIdent.GoName

	gf.P("// Error implements the error interface for ", typeName, ".")
	gf.P("// This allows ", typeName, " to be used with errors.As() and errors.Is().")
	gf.P("func (e *", typeName, ") Error() string {")
	gf.P("if e == nil {")
	gf.P(`return "`, strings.ToLower(typeName), `: <nil>"`)
	gf.P("}")
	gf.P()
	gf.P("jsonBytes, err := protojson.Marshal(e)")
	gf.P("if err != nil {")
	gf.P(`return fmt.Sprintf("`, strings.ToLower(typeName), `: failed to serialize (%v)", err)`)
	gf.P("}")
	gf.P()
	gf.P("return string(jsonBytes)")
	gf.P("}")
}
