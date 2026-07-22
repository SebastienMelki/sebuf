package clientgen

import (
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// Generator handles HTTP client code generation for protobuf services.
type Generator struct {
	plugin       *protogen.Plugin
	fileNeedsSSE *bool // set per-file before writeImports
}

// New creates a new HTTP client generator.
func New(plugin *protogen.Plugin) *Generator {
	return &Generator{
		plugin: plugin,
	}
}

// Generate processes all files and generates HTTP clients.
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
	// Validate enum annotations first - fail fast if conflicting annotations exist
	if err := g.validateEnumAnnotationsInFile(file); err != nil {
		return fmt.Errorf("enum annotation validation failed: %w", err)
	}

	// Generate nullable encoding file if there are messages with nullable fields
	if err := g.generateNullableEncodingFile(file); err != nil {
		return err
	}

	// Generate empty_behavior encoding file if there are messages with empty_behavior fields
	if err := g.generateEmptyBehaviorEncodingFile(file); err != nil {
		return err
	}

	// Generate timestamp_format encoding file if there are messages with timestamp format annotations
	if err := g.generateTimestampFormatEncodingFile(file); err != nil {
		return err
	}

	// Generate bytes_encoding file if there are messages with non-default bytes encoding
	if err := g.generateBytesEncodingFile(file); err != nil {
		return err
	}

	// Generate flatten file if there are messages with flatten annotations
	if err := g.generateFlattenFile(file); err != nil {
		return err
	}

	// Generate oneof_discriminator file if there are messages with oneof_config annotations
	if err := g.generateOneofDiscriminatorFile(file); err != nil {
		return err
	}

	if len(file.Services) == 0 {
		return nil
	}

	// Generate client file
	if err := g.generateClientFile(file); err != nil {
		return err
	}

	// Generate encoding file if there are messages with int64_encoding=NUMBER annotations
	if err := g.generateInt64EncodingFile(file); err != nil {
		return err
	}

	// Generate enum encoding file if there are enums with custom enum_value annotations
	if err := g.generateEnumEncodingFile(file); err != nil {
		return err
	}

	// Generate enum-field encoding file so the client sends/receives custom enum_value strings
	// (protojson emits raw proto value names). Depends on the lookup maps emitted above.
	if err := g.generateEnumFieldEncodingFile(file); err != nil {
		return err
	}

	return nil
}

func (g *Generator) generateClientFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_client.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	// Check if any method needs a request body
	needsBytes := g.fileNeedsRequestBody(file)
	// Check if any method needs the net/url import (path params or query params)
	needsURL := g.fileNeedsURLImport(file)
	// Check if any method uses SSE streaming
	hasSSE := g.fileHasSSEMethods(file)
	g.fileNeedsSSE = &hasSSE

	g.writeHeader(gf, file)
	g.writeImports(gf, needsBytes, needsURL)

	// Generate content type constants once at file level
	g.generateContentTypeConstants(gf)

	// Generate sebufUnmarshaler interface once at file level
	g.generateSebufUnmarshalerInterface(gf)

	for _, service := range file.Services {
		if err := g.generateServiceClient(gf, file, service); err != nil {
			return err
		}
	}

	return nil
}

// fileNeedsURLImport checks if any method in the file needs the "net/url" import.
// This is true when path parameters (url.PathEscape) or query parameters (url.Values) are used.
func (g *Generator) fileNeedsURLImport(file *protogen.File) bool {
	for _, service := range file.Services {
		for _, method := range service.Methods {
			httpConfig := annotations.GetMethodHTTPConfig(method)
			// Path params use url.PathEscape
			if httpConfig != nil && len(httpConfig.PathParams) > 0 {
				return true
			}
			// Determine HTTP method
			httpMethod := http.MethodPost
			if httpConfig != nil && httpConfig.Method != "" {
				httpMethod = httpConfig.Method
			}
			// GET/DELETE with query params use url.Values
			if httpMethod == http.MethodGet || httpMethod == http.MethodDelete {
				queryParams := annotations.GetQueryParams(method.Input)
				if len(queryParams) > 0 {
					return true
				}
			}
		}
	}
	return false
}

// fileNeedsRequestBody checks if any method in the file needs a request body.
func (g *Generator) fileNeedsRequestBody(file *protogen.File) bool {
	for _, service := range file.Services {
		for _, method := range service.Methods {
			httpConfig := annotations.GetMethodHTTPConfig(method)
			httpMethod := http.MethodPost
			if httpConfig != nil && httpConfig.Method != "" {
				httpMethod = httpConfig.Method
			}
			if httpMethod == http.MethodPost || httpMethod == http.MethodPut || httpMethod == http.MethodPatch {
				return true
			}
		}
	}
	return false
}

func (g *Generator) writeHeader(gf *protogen.GeneratedFile, file *protogen.File) {
	gf.P("// Code generated by protoc-gen-go-client. DO NOT EDIT.")
	gf.P("// source: ", file.Desc.Path())
	gf.P()
	gf.P("package ", file.GoPackageName)
	gf.P()
}

func (g *Generator) writeImports(gf *protogen.GeneratedFile, needsBytes, needsURL bool) {
	needsSSE := g.fileNeedsSSE != nil && *g.fileNeedsSSE
	gf.P("import (")
	if needsSSE {
		gf.P(`"bufio"`)
	}
	if needsBytes {
		gf.P(`"bytes"`)
	}
	gf.P(`"context"`)
	gf.P(`"encoding/json"`)
	gf.P(`"fmt"`)
	gf.P(`"io"`)
	gf.P(`"net/http"`)
	if needsURL {
		gf.P(`"net/url"`)
	}
	gf.P(`"strings"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/encoding/protojson"`)
	gf.P(`"google.golang.org/protobuf/proto"`)
	gf.P()
	gf.P(`sebufhttp "github.com/SebastienMelki/sebuf/http"`)
	gf.P(")")
	gf.P()
}

func (g *Generator) generateServiceClient(
	gf *protogen.GeneratedFile,
	file *protogen.File,
	service *protogen.Service,
) error {
	serviceName := service.GoName

	// Generate client interface
	g.generateClientInterface(gf, service)

	// Generate client struct
	g.generateClientStruct(gf, serviceName)

	// Generate ClientOption type and options
	g.generateClientOptions(gf, serviceName)

	// Generate CallOption type and options
	g.generateCallOptions(gf, serviceName)

	// Generate header helper options from annotations
	g.generateHeaderHelperOptions(gf, service)

	// Generate constructor
	g.generateConstructor(gf, serviceName)

	// Generate EventStream type if any SSE methods
	if g.serviceHasSSEMethods(service) {
		g.generateEventStreamType(gf, serviceName)
	}

	// Generate RPC methods
	for _, method := range service.Methods {
		if err := g.generateRPCMethod(gf, file, service, method); err != nil {
			return err
		}
	}

	// Generate helper methods
	g.generateHelperMethods(gf, serviceName)

	return nil
}

func (g *Generator) generateClientInterface(gf *protogen.GeneratedFile, service *protogen.Service) {
	serviceName := service.GoName

	gf.P("// ", serviceName, "Client is the client API for ", serviceName, " service.")
	gf.P("type ", serviceName, "Client interface {")
	for _, method := range service.Methods {
		httpConfig := annotations.GetMethodHTTPConfig(method)
		isSSE := httpConfig != nil && httpConfig.Stream
		if isSSE {
			gf.P(
				method.GoName,
				"(ctx context.Context, req *",
				method.Input.GoIdent,
				", opts ...",
				serviceName,
				"CallOption) (*",
				serviceName,
				"EventStream[*",
				method.Output.GoIdent,
				"], error)",
			)
		} else {
			gf.P(
				method.GoName,
				"(ctx context.Context, req *",
				method.Input.GoIdent,
				", opts ...",
				serviceName,
				"CallOption) (*",
				method.Output.GoIdent,
				", error)",
			)
		}
	}
	gf.P("}")
	gf.P()
}

func (g *Generator) generateClientStruct(gf *protogen.GeneratedFile, serviceName string) {
	lowerName := annotations.LowerFirst(serviceName)

	gf.P("// ", lowerName, "Client is the implementation of ", serviceName, "Client.")
	gf.P("type ", lowerName, "Client struct {")
	gf.P("baseURL string")
	gf.P("httpClient *http.Client")
	gf.P("contentType string")
	gf.P("defaultHeaders map[string]string")
	gf.P("discardUnknownFields bool")
	gf.P("}")
	gf.P()

	// Ensure struct implements interface
	gf.P("var _ ", serviceName, "Client = (*", lowerName, "Client)(nil)")
	gf.P()
}

func (g *Generator) generateContentTypeConstants(gf *protogen.GeneratedFile) {
	gf.P("const (")
	gf.P("// ContentTypeJSON is the content type for JSON requests/responses.")
	gf.P(`ContentTypeJSON = "application/json"`)
	gf.P("// ContentTypeProto is the content type for binary protobuf requests/responses.")
	gf.P(`ContentTypeProto = "application/x-protobuf"`)
	gf.P(")")
	gf.P()
}

func (g *Generator) generateSebufUnmarshalerInterface(gf *protogen.GeneratedFile) {
	gf.P("// sebufUnmarshaler is implemented by generated messages with custom JSON unmarshaling.")
	gf.P("// It allows passing protojson.UnmarshalOptions (e.g. DiscardUnknown) through custom unmarshalers.")
	gf.P("type sebufUnmarshaler interface {")
	gf.P("UnmarshalJSONSebuf(data []byte, opts protojson.UnmarshalOptions) error")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateClientOptions(gf *protogen.GeneratedFile, serviceName string) {
	lowerName := annotations.LowerFirst(serviceName)

	gf.P("// ", serviceName, "ClientOption configures a ", serviceName, " client.")
	gf.P("type ", serviceName, "ClientOption func(*", lowerName, "Client)")
	gf.P()

	// With{Service}HTTPClient
	gf.P("// With", serviceName, "HTTPClient sets the HTTP client to use for requests.")
	gf.P("func With", serviceName, "HTTPClient(client *http.Client) ", serviceName, "ClientOption {")
	gf.P("return func(c *", lowerName, "Client) {")
	gf.P("c.httpClient = client")
	gf.P("}")
	gf.P("}")
	gf.P()

	// With{Service}ContentType
	gf.P("// With", serviceName, "ContentType sets the default content type for requests.")
	gf.P("// Use ContentTypeJSON or ContentTypeProto.")
	gf.P("func With", serviceName, "ContentType(contentType string) ", serviceName, "ClientOption {")
	gf.P("return func(c *", lowerName, "Client) {")
	gf.P("c.contentType = contentType")
	gf.P("}")
	gf.P("}")
	gf.P()

	// With{Service}DefaultHeader
	gf.P("// With", serviceName, "DefaultHeader sets a default header to include in all requests.")
	gf.P("func With", serviceName, "DefaultHeader(key, value string) ", serviceName, "ClientOption {")
	gf.P("return func(c *", lowerName, "Client) {")
	gf.P("if c.defaultHeaders == nil {")
	gf.P("c.defaultHeaders = make(map[string]string)")
	gf.P("}")
	gf.P("c.defaultHeaders[key] = value")
	gf.P("}")
	gf.P("}")
	gf.P()

	// With{Service}DiscardUnknownFields
	gf.P("// With", serviceName, "DiscardUnknownFields sets whether to discard unknown fields in JSON responses.")
	gf.P("// When true, unknown fields are silently ignored instead of causing unmarshal errors.")
	gf.P("func With", serviceName, "DiscardUnknownFields(discard bool) ", serviceName, "ClientOption {")
	gf.P("return func(c *", lowerName, "Client) {")
	gf.P("c.discardUnknownFields = discard")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateCallOptions(gf *protogen.GeneratedFile, serviceName string) {
	lowerName := annotations.LowerFirst(serviceName)

	// CallOption type
	gf.P("// ", serviceName, "CallOption configures a single RPC call.")
	gf.P("type ", serviceName, "CallOption func(*", lowerName, "CallOptions)")
	gf.P()

	// CallOptions struct
	gf.P("// ", lowerName, "CallOptions holds options for a single RPC call.")
	gf.P("type ", lowerName, "CallOptions struct {")
	gf.P("headers map[string]string")
	gf.P("contentType string")
	gf.P("discardUnknownFields *bool")
	gf.P("}")
	gf.P()

	// With{Service}Header
	gf.P("// With", serviceName, "Header adds a header to a single request.")
	gf.P("func With", serviceName, "Header(key, value string) ", serviceName, "CallOption {")
	gf.P("return func(o *", lowerName, "CallOptions) {")
	gf.P("if o.headers == nil {")
	gf.P("o.headers = make(map[string]string)")
	gf.P("}")
	gf.P("o.headers[key] = value")
	gf.P("}")
	gf.P("}")
	gf.P()

	// With{Service}CallContentType
	gf.P("// With", serviceName, "CallContentType sets the content type for a single request.")
	gf.P("func With", serviceName, "CallContentType(contentType string) ", serviceName, "CallOption {")
	gf.P("return func(o *", lowerName, "CallOptions) {")
	gf.P("o.contentType = contentType")
	gf.P("}")
	gf.P("}")
	gf.P()

	// With{Service}CallDiscardUnknownFields
	gf.P(
		"// With", serviceName,
		"CallDiscardUnknownFields sets whether to discard unknown fields for a single request.",
	)
	gf.P("// Overrides the client-level setting from With", serviceName, "DiscardUnknownFields.")
	gf.P("func With", serviceName, "CallDiscardUnknownFields(discard bool) ", serviceName, "CallOption {")
	gf.P("return func(o *", lowerName, "CallOptions) {")
	gf.P("o.discardUnknownFields = &discard")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateHeaderHelperOptions(gf *protogen.GeneratedFile, service *protogen.Service) {
	serviceName := service.GoName

	// Generate helper options from service headers
	serviceHeaders := annotations.GetServiceHeaders(service)
	for _, header := range serviceHeaders {
		g.generateHeaderOption(gf, serviceName, header, true)
	}

	// Generate helper options from method headers
	for _, method := range service.Methods {
		methodHeaders := annotations.GetMethodHeaders(method)
		for _, header := range methodHeaders {
			g.generateHeaderOption(gf, serviceName, header, false)
		}
	}
}

func (g *Generator) generateHeaderOption(
	gf *protogen.GeneratedFile,
	serviceName string,
	header *sebufhttp.Header,
	isClientOption bool,
) {
	headerName := header.GetName()
	funcName := headerNameToFuncName(headerName)
	description := header.GetDescription()
	if description == "" {
		description = fmt.Sprintf("sets the %s header", headerName)
	}

	if isClientOption {
		// Generate ClientOption for service-level headers
		gf.P("// With", serviceName, funcName, " ", description)
		gf.P("func With", serviceName, funcName, "(value string) ", serviceName, "ClientOption {")
		gf.P("return With", serviceName, "DefaultHeader(\"", headerName, "\", value)")
		gf.P("}")
		gf.P()
	}

	// Generate CallOption for both service and method headers
	gf.P("// With", serviceName, "Call", funcName, " ", description, " for a single request.")
	gf.P("func With", serviceName, "Call", funcName, "(value string) ", serviceName, "CallOption {")
	gf.P("return With", serviceName, "Header(\"", headerName, "\", value)")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateConstructor(gf *protogen.GeneratedFile, serviceName string) {
	lowerName := annotations.LowerFirst(serviceName)

	gf.P("// New", serviceName, "Client creates a new ", serviceName, " client.")
	gf.P(
		"func New",
		serviceName,
		"Client(baseURL string, opts ...",
		serviceName,
		"ClientOption) ",
		serviceName,
		"Client {",
	)
	gf.P("c := &", lowerName, "Client{")
	gf.P("baseURL: strings.TrimSuffix(baseURL, \"/\"),")
	gf.P("httpClient: http.DefaultClient,")
	gf.P("contentType: ContentTypeJSON,")
	gf.P("defaultHeaders: make(map[string]string),")
	gf.P("}")
	gf.P()
	gf.P("for _, opt := range opts {")
	gf.P("opt(c)")
	gf.P("}")
	gf.P()
	gf.P("return c")
	gf.P("}")
	gf.P()
}

// rpcMethodConfig holds the configuration for generating an RPC method.
type rpcMethodConfig struct {
	serviceName string
	lowerName   string
	methodName  string
	httpMethod  string
	fullPath    string
	pathParams  []string
	queryParams []annotations.QueryParam
	hasBody     bool
	isSSE       bool
}

func (g *Generator) buildRPCMethodConfig(service *protogen.Service, method *protogen.Method) *rpcMethodConfig {
	serviceName := service.GoName
	methodName := method.GoName

	// Get HTTP config
	httpConfig := annotations.GetMethodHTTPConfig(method)
	httpMethod := http.MethodPost
	httpPath := "/" + annotations.LowerFirst(methodName)
	var pathParams []string

	if httpConfig != nil {
		if httpConfig.Method != "" {
			httpMethod = httpConfig.Method
		}
		if httpConfig.Path != "" {
			httpPath = httpConfig.Path
		}
		pathParams = httpConfig.PathParams
	}

	// Get base path from service config
	basePath := annotations.GetServiceBasePath(service)

	// Combine base path and method path
	fullPath := annotations.BuildHTTPPath(basePath, httpPath)

	isSSE := httpConfig != nil && httpConfig.Stream

	return &rpcMethodConfig{
		serviceName: serviceName,
		lowerName:   annotations.LowerFirst(serviceName),
		methodName:  methodName,
		httpMethod:  httpMethod,
		fullPath:    fullPath,
		pathParams:  pathParams,
		queryParams: annotations.GetQueryParams(method.Input),
		hasBody:     httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH",
		isSSE:       isSSE,
	}
}

func (g *Generator) generateRPCMethod(
	gf *protogen.GeneratedFile,
	_ *protogen.File,
	service *protogen.Service,
	method *protogen.Method,
) error {
	cfg := g.buildRPCMethodConfig(service, method)

	if cfg.isSSE {
		return g.generateSSERPCMethod(gf, cfg, method)
	}

	g.generateRPCMethodSignature(gf, cfg, method)
	g.generateRPCMethodCallOptions(gf, cfg)
	g.generateRPCMethodURLBuilding(gf, cfg)
	g.generateRPCMethodRequest(gf, cfg)
	g.generateRPCMethodHeaders(gf, cfg)
	g.generateRPCMethodExecution(gf)
	g.generateRPCMethodResponse(gf, method)

	return nil
}

// generateSSERPCMethod generates a client method for SSE streaming endpoints.
//
//nolint:funlen // SSE method generation requires many sequential code blocks
func (g *Generator) generateSSERPCMethod(
	gf *protogen.GeneratedFile,
	cfg *rpcMethodConfig,
	method *protogen.Method,
) error {
	// Method signature
	gf.P("// ", cfg.methodName, " calls the ", cfg.methodName, " SSE streaming RPC.")
	gf.P(
		"func (c *", cfg.lowerName, "Client) ", cfg.methodName,
		"(ctx context.Context, req *", method.Input.GoIdent,
		", opts ...", cfg.serviceName, "CallOption) (*",
		cfg.serviceName, "EventStream[*", method.Output.GoIdent, "], error) {",
	)

	// Call options
	g.generateRPCMethodCallOptions(gf, cfg)

	// URL building
	g.generateRPCMethodURLBuilding(gf, cfg)

	// Content type
	gf.P()
	gf.P("contentType := c.contentType")
	gf.P("if callOpts.contentType != \"\" {")
	gf.P("contentType = callOpts.contentType")
	gf.P("}")
	gf.P()

	// Create request
	if cfg.hasBody {
		gf.P("// Marshal request body")
		gf.P("body, err := c.marshalRequest(req, contentType)")
		gf.P("if err != nil {")
		gf.P("return nil, fmt.Errorf(\"failed to marshal request: %w\", err)")
		gf.P("}")
		gf.P()
		gf.P("// Create HTTP request")
		gf.P("httpReq, err := http.NewRequestWithContext(ctx, \"", cfg.httpMethod, "\", reqURL, bytes.NewReader(body))")
	} else {
		gf.P("// Create HTTP request")
		gf.P("httpReq, err := http.NewRequestWithContext(ctx, \"", cfg.httpMethod, "\", reqURL, nil)")
	}
	gf.P("if err != nil {")
	gf.P("return nil, fmt.Errorf(\"failed to create request: %w\", err)")
	gf.P("}")

	// Headers
	gf.P()
	gf.P("// Set headers")
	gf.P("httpReq.Header.Set(\"Content-Type\", contentType)")
	gf.P("httpReq.Header.Set(\"Accept\", \"text/event-stream\")")
	gf.P("for k, v := range c.defaultHeaders {")
	gf.P("httpReq.Header.Set(k, v)")
	gf.P("}")
	gf.P("for k, v := range callOpts.headers {")
	gf.P("httpReq.Header.Set(k, v)")
	gf.P("}")

	// Execute - do NOT defer resp.Body.Close() since caller owns the stream
	gf.P()
	gf.P("// Execute request")
	gf.P("resp, err := c.httpClient.Do(httpReq)")
	gf.P("if err != nil {")
	gf.P("return nil, fmt.Errorf(\"failed to execute request: %w\", err)")
	gf.P("}")
	gf.P()

	// Error handling (close body on error)
	gf.P("// Check for error status codes")
	gf.P("if resp.StatusCode >= 400 {")
	gf.P("defer resp.Body.Close()")
	gf.P("respBody, readErr := io.ReadAll(resp.Body)")
	gf.P("if readErr != nil {")
	gf.P("return nil, fmt.Errorf(\"failed to read error response: %w\", readErr)")
	gf.P("}")
	gf.P("return nil, c.handleErrorResponse(resp.StatusCode, respBody, contentType)")
	gf.P("}")
	gf.P()

	// Resolve discardUnknownFields: per-call option overrides client default
	gf.P("discardUnknown := c.discardUnknownFields")
	gf.P("if callOpts.discardUnknownFields != nil {")
	gf.P("discardUnknown = *callOpts.discardUnknownFields")
	gf.P("}")
	gf.P()

	// Return EventStream
	gf.P("return &", cfg.serviceName, "EventStream[*", method.Output.GoIdent, "]{")
	gf.P("resp:                 resp,")
	gf.P("reader:               bufio.NewReader(resp.Body),")
	gf.P("discardUnknownFields: discardUnknown,")
	gf.P("}, nil")
	gf.P("}")
	gf.P()

	return nil
}

func (g *Generator) generateRPCMethodSignature(
	gf *protogen.GeneratedFile,
	cfg *rpcMethodConfig,
	method *protogen.Method,
) {
	gf.P("// ", cfg.methodName, " calls the ", cfg.methodName, " RPC.")
	gf.P(
		"func (c *", cfg.lowerName, "Client) ", cfg.methodName,
		"(ctx context.Context, req *", method.Input.GoIdent,
		", opts ...", cfg.serviceName, "CallOption) (*", method.Output.GoIdent, ", error) {",
	)
}

func (g *Generator) generateRPCMethodCallOptions(gf *protogen.GeneratedFile, cfg *rpcMethodConfig) {
	gf.P("callOpts := &", cfg.lowerName, "CallOptions{}")
	gf.P("for _, opt := range opts {")
	gf.P("opt(callOpts)")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateRPCMethodURLBuilding(gf *protogen.GeneratedFile, cfg *rpcMethodConfig) {
	gf.P("// Build URL")
	g.generateURLBuilding(gf, cfg.fullPath, cfg.pathParams, cfg.queryParams, cfg.httpMethod)
}

func (g *Generator) generateRPCMethodRequest(gf *protogen.GeneratedFile, cfg *rpcMethodConfig) {
	gf.P()
	gf.P("contentType := c.contentType")
	gf.P("if callOpts.contentType != \"\" {")
	gf.P("contentType = callOpts.contentType")
	gf.P("}")
	gf.P()

	if cfg.hasBody {
		gf.P("// Marshal request body")
		gf.P("body, err := c.marshalRequest(req, contentType)")
		gf.P("if err != nil {")
		gf.P("return nil, fmt.Errorf(\"failed to marshal request: %w\", err)")
		gf.P("}")
		gf.P()
		gf.P("// Create HTTP request")
		gf.P("httpReq, err := http.NewRequestWithContext(ctx, \"", cfg.httpMethod, "\", reqURL, bytes.NewReader(body))")
	} else {
		gf.P("// Create HTTP request")
		gf.P("httpReq, err := http.NewRequestWithContext(ctx, \"", cfg.httpMethod, "\", reqURL, nil)")
	}

	gf.P("if err != nil {")
	gf.P("return nil, fmt.Errorf(\"failed to create request: %w\", err)")
	gf.P("}")
}

func (g *Generator) generateRPCMethodHeaders(gf *protogen.GeneratedFile, _ *rpcMethodConfig) {
	gf.P()
	gf.P("// Set headers")
	gf.P("httpReq.Header.Set(\"Content-Type\", contentType)")
	gf.P("for k, v := range c.defaultHeaders {")
	gf.P("httpReq.Header.Set(k, v)")
	gf.P("}")
	gf.P("for k, v := range callOpts.headers {")
	gf.P("httpReq.Header.Set(k, v)")
	gf.P("}")
}

func (g *Generator) generateRPCMethodExecution(gf *protogen.GeneratedFile) {
	gf.P()
	gf.P("// Execute request")
	gf.P("resp, err := c.httpClient.Do(httpReq)")
	gf.P("if err != nil {")
	gf.P("return nil, fmt.Errorf(\"failed to execute request: %w\", err)")
	gf.P("}")
	gf.P("defer resp.Body.Close()")
}

func (g *Generator) generateRPCMethodResponse(gf *protogen.GeneratedFile, method *protogen.Method) {
	gf.P()
	gf.P("// Read response body")
	gf.P("respBody, err := io.ReadAll(resp.Body)")
	gf.P("if err != nil {")
	gf.P("return nil, fmt.Errorf(\"failed to read response body: %w\", err)")
	gf.P("}")
	gf.P()
	gf.P("// Check for error status codes")
	gf.P("if resp.StatusCode >= 400 {")
	gf.P("return nil, c.handleErrorResponse(resp.StatusCode, respBody, contentType)")
	gf.P("}")
	gf.P()
	gf.P("// Resolve discardUnknownFields: per-call option overrides client default")
	gf.P("discardUnknown := c.discardUnknownFields")
	gf.P("if callOpts.discardUnknownFields != nil {")
	gf.P("discardUnknown = *callOpts.discardUnknownFields")
	gf.P("}")
	gf.P()
	gf.P("// Unmarshal response")
	gf.P("result := &", method.Output.GoIdent, "{}")
	gf.P("if err := c.unmarshalResponse(respBody, result, contentType, discardUnknown); err != nil {")
	gf.P("return nil, fmt.Errorf(\"failed to unmarshal response: %w\", err)")
	gf.P("}")
	gf.P()
	gf.P("return result, nil")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateURLBuilding(
	gf *protogen.GeneratedFile,
	fullPath string,
	pathParams []string,
	queryParams []annotations.QueryParam,
	httpMethod string,
) {
	// Start with base path
	gf.P("path := \"", fullPath, "\"")

	// Replace path parameters
	if len(pathParams) > 0 {
		for _, param := range pathParams {
			goFieldName := snakeToUpperCamel(param)
			gf.P(
				"path = strings.Replace(path, \"{",
				param,
				"}\", url.PathEscape(fmt.Sprint(req.",
				goFieldName,
				")), 1)",
			)
		}
	}

	gf.P("reqURL := c.baseURL + path")

	// Add query parameters for GET/DELETE
	if (httpMethod == "GET" || httpMethod == "DELETE") && len(queryParams) > 0 {
		gf.P()
		gf.P("// Add query parameters")
		gf.P("queryParams := url.Values{}")
		for _, qp := range queryParams {
			g.generateQueryParamEncoding(gf, qp)
		}
		gf.P("if len(queryParams) > 0 {")
		gf.P("reqURL += \"?\" + queryParams.Encode()")
		gf.P("}")
	}
}

func (g *Generator) generateQueryParamEncoding(gf *protogen.GeneratedFile, qp annotations.QueryParam) {
	fieldGoName := qp.FieldGoName
	paramName := qp.ParamName

	// Handle repeated fields: iterate and Add() each value individually
	if qp.Field != nil && qp.Field.Desc.IsList() {
		gf.P("for _, v := range req.", fieldGoName, " {")
		gf.P("queryParams.Add(\"", paramName, "\", fmt.Sprint(v))")
		gf.P("}")
		return
	}

	// Scalar fields: zero-value check + Set()
	gf.P("if req.", fieldGoName, " != ", getZeroValue(qp), " {")
	gf.P("queryParams.Set(\"", paramName, "\", fmt.Sprint(req.", fieldGoName, "))")
	gf.P("}")
}

func (g *Generator) generateHelperMethods(gf *protogen.GeneratedFile, serviceName string) {
	lowerName := annotations.LowerFirst(serviceName)
	g.generateMarshalRequestMethod(gf, lowerName)
	g.generateHandleErrorResponseMethod(gf, lowerName)
	g.generateUnmarshalResponseMethod(gf, lowerName)
}

func (g *Generator) generateMarshalRequestMethod(gf *protogen.GeneratedFile, lowerName string) {
	gf.P("func (c *", lowerName, "Client) marshalRequest(req proto.Message, contentType string) ([]byte, error) {")
	gf.P("switch contentType {")
	gf.P("case ContentTypeJSON:")
	gf.P("// Check for custom JSON marshaler (unwrap support)")
	gf.P("if marshaler, ok := req.(json.Marshaler); ok {")
	gf.P("return marshaler.MarshalJSON()")
	gf.P("}")
	gf.P("return protojson.Marshal(req)")
	gf.P("case ContentTypeProto:")
	gf.P("return proto.Marshal(req)")
	gf.P("default:")
	gf.P("return protojson.Marshal(req)")
	gf.P("}")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateHandleErrorResponseMethod(gf *protogen.GeneratedFile, lowerName string) {
	gf.P("func (c *", lowerName, "Client) handleErrorResponse(statusCode int, body []byte, contentType string) error {")
	gf.P("// Try to parse as ValidationError first (for 400 errors)")
	gf.P("// Always use strict mode (false) for error parsing to avoid loose JSON")
	gf.P("// falsely matching ValidationError or Error types.")
	gf.P("if statusCode == http.StatusBadRequest {")
	gf.P("validationErr := &sebufhttp.ValidationError{}")
	gf.P("if unmarshalErr := c.unmarshalResponse(body, validationErr, contentType, false); unmarshalErr == nil {")
	gf.P("return validationErr")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("// Try to parse as generic Error")
	gf.P("genericErr := &sebufhttp.Error{}")
	gf.P("if unmarshalErr := c.unmarshalResponse(body, genericErr, contentType, false); unmarshalErr == nil {")
	gf.P("return genericErr")
	gf.P("}")
	gf.P()
	gf.P("// Fallback to raw error message")
	gf.P("return fmt.Errorf(\"request failed with status %d: %s\", statusCode, string(body))")
	gf.P("}")
	gf.P()
}

func (g *Generator) generateUnmarshalResponseMethod(gf *protogen.GeneratedFile, lowerName string) {
	gf.P(
		"func (c *",
		lowerName,
		"Client) unmarshalResponse(body []byte, msg proto.Message, contentType string, discardUnknown bool) error {",
	)
	gf.P("if len(body) == 0 {")
	gf.P("return nil")
	gf.P("}")
	gf.P()
	gf.P("opts := protojson.UnmarshalOptions{DiscardUnknown: discardUnknown}")
	gf.P()
	gf.P("switch contentType {")
	gf.P("case ContentTypeJSON:")
	gf.P("// Check for sebuf-generated custom unmarshaler (passes options through)")
	gf.P("if u, ok := msg.(sebufUnmarshaler); ok {")
	gf.P("return u.UnmarshalJSONSebuf(body, opts)")
	gf.P("}")
	gf.P("// Check for third-party json.Unmarshaler (best effort, cannot pass options)")
	gf.P("if u, ok := msg.(json.Unmarshaler); ok {")
	gf.P("return u.UnmarshalJSON(body)")
	gf.P("}")
	gf.P("return opts.Unmarshal(body, msg)")
	gf.P("case ContentTypeProto:")
	gf.P("return proto.Unmarshal(body, msg)")
	gf.P("default:")
	gf.P("return opts.Unmarshal(body, msg)")
	gf.P("}")
	gf.P("}")
	gf.P()
}

// Helper functions

func snakeToUpperCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func headerNameToFuncName(headerName string) string {
	// Convert X-API-Key to APIKey, X-Request-ID to RequestID
	name := strings.TrimPrefix(headerName, "X-")
	name = strings.ReplaceAll(name, "-", "")
	return name
}

// fileHasSSEMethods checks if any method in the file uses SSE streaming.
func (g *Generator) fileHasSSEMethods(file *protogen.File) bool {
	for _, service := range file.Services {
		for _, method := range service.Methods {
			httpConfig := annotations.GetMethodHTTPConfig(method)
			if httpConfig != nil && httpConfig.Stream {
				return true
			}
		}
	}
	return false
}

// serviceHasSSEMethods checks if any method in the service uses SSE streaming.
func (g *Generator) serviceHasSSEMethods(service *protogen.Service) bool {
	for _, method := range service.Methods {
		httpConfig := annotations.GetMethodHTTPConfig(method)
		if httpConfig != nil && httpConfig.Stream {
			return true
		}
	}
	return false
}

// generateEventStreamType generates the EventStream generic type for SSE streaming.
//
//nolint:funlen // SSE event stream generation requires many sequential code blocks
func (g *Generator) generateEventStreamType(gf *protogen.GeneratedFile, serviceName string) {
	gf.P("// ", serviceName, "EventStream reads Server-Sent Events from a streaming endpoint.")
	gf.P("type ", serviceName, "EventStream[T proto.Message] struct {")
	gf.P("resp                 *http.Response")
	gf.P("reader               *bufio.Reader")
	gf.P("err                  error")
	gf.P("discardUnknownFields bool")
	gf.P("}")
	gf.P()

	gf.P("// Next reads the next event from the stream.")
	gf.P("// Returns false when the stream ends or an error occurs.")
	gf.P("func (s *", serviceName, "EventStream[T]) Next(event T) bool {")
	gf.P("for {")
	gf.P("line, err := s.reader.ReadString('\\n')")
	gf.P("if err != nil {")
	gf.P("if err != io.EOF {")
	gf.P("s.err = err")
	gf.P("}")
	gf.P("return false")
	gf.P("}")
	gf.P("line = strings.TrimRight(line, \"\\r\\n\")")
	gf.P(`if !strings.HasPrefix(line, "data: ") {`)
	gf.P("continue")
	gf.P("}")
	gf.P(`data := strings.TrimPrefix(line, "data: ")`)
	gf.P("opts := protojson.UnmarshalOptions{DiscardUnknown: s.discardUnknownFields}")
	gf.P("var unmarshalErr error")
	gf.P("if u, ok := any(event).(sebufUnmarshaler); ok {")
	gf.P("unmarshalErr = u.UnmarshalJSONSebuf([]byte(data), opts)")
	gf.P("} else if u, ok := any(event).(json.Unmarshaler); ok {")
	gf.P("unmarshalErr = u.UnmarshalJSON([]byte(data))")
	gf.P("} else {")
	gf.P("unmarshalErr = opts.Unmarshal([]byte(data), event)")
	gf.P("}")
	gf.P("if unmarshalErr != nil {")
	gf.P(`s.err = fmt.Errorf("failed to unmarshal SSE event: %w", unmarshalErr)`)
	gf.P("return false")
	gf.P("}")
	gf.P("return true")
	gf.P("}")
	gf.P("}")
	gf.P()

	gf.P("// Err returns any error encountered during streaming.")
	gf.P("func (s *", serviceName, "EventStream[T]) Err() error {")
	gf.P("return s.err")
	gf.P("}")
	gf.P()

	gf.P("// Close closes the underlying HTTP response body.")
	gf.P("func (s *", serviceName, "EventStream[T]) Close() error {")
	gf.P("return s.resp.Body.Close()")
	gf.P("}")
	gf.P()
}

func getZeroValue(qp annotations.QueryParam) string {
	// Return the appropriate zero value based on field kind
	switch qp.FieldKind {
	case "string":
		return `""`
	case "bool":
		return "false"
	case "int32", "sint32", "sfixed32", "int64", "sint64", "sfixed64",
		"uint32", "fixed32", "uint64", "fixed64":
		return "0"
	case "float", "double":
		return "0"
	case "enum":
		return "0"
	default:
		return `""`
	}
}
