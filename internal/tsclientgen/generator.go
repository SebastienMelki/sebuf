package tsclientgen

import (
	"fmt"
	"net/http"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

// Generator handles TypeScript HTTP client code generation for protobuf services.
type Generator struct {
	plugin *protogen.Plugin
	// ctx carries the emission state (self module + import tracker) for the
	// service file currently being written.
	ctx *tscommon.EmitContext
	// runtime selects the TypeScript message representation.
	runtime tscommon.MessageRuntime
}

// New creates a new TypeScript client generator.
func New(plugin *protogen.Plugin, runtime tscommon.MessageRuntime) *Generator {
	return &Generator{plugin: plugin, runtime: runtime}
}

// Generate emits one canonical type module per proto file, a shared errors
// module, and one slimmed client module per service file.
func (g *Generator) Generate() error {
	return g.generateModules()
}

func (g *Generator) generateServiceClient(p printer, service *protogen.Service) error {
	serviceName := service.GoName

	// Client options interface
	g.generateClientOptionsInterface(p, service)

	// Call options interface
	g.generateCallOptionsInterface(p, service)

	// Client class
	g.generateClientClass(p, service)

	_ = serviceName
	return nil
}

// generateClientOptionsInterface generates the {Service}ClientOptions interface.
func (g *Generator) generateClientOptionsInterface(p printer, service *protogen.Service) {
	serviceName := service.GoName

	p("export interface %sClientOptions {", serviceName)
	p("  fetch?: typeof fetch;")
	p("  defaultHeaders?: Record<string, string>;")

	// Add typed properties for service-level headers
	serviceHeaders := annotations.GetServiceHeaders(service)
	for _, header := range serviceHeaders {
		propName := headerNameToPropertyName(header.GetName())
		p("  %s?: string;", propName)
	}

	p("}")
	p("")
}

// generateCallOptionsInterface generates the {Service}CallOptions interface.
func (g *Generator) generateCallOptionsInterface(p printer, service *protogen.Service) {
	serviceName := service.GoName

	p("export interface %sCallOptions {", serviceName)
	p("  headers?: Record<string, string>;")
	p("  signal?: AbortSignal;")

	// Add typed properties for service-level headers (also available per-call)
	serviceHeaders := annotations.GetServiceHeaders(service)
	for _, header := range serviceHeaders {
		propName := headerNameToPropertyName(header.GetName())
		p("  %s?: string;", propName)
	}

	// Add typed properties for method-level headers
	seen := make(map[string]bool)
	for _, method := range service.Methods {
		methodHeaders := annotations.GetMethodHeaders(method)
		for _, header := range methodHeaders {
			propName := headerNameToPropertyName(header.GetName())
			if seen[propName] {
				continue
			}
			seen[propName] = true
			p("  %s?: string;", propName)
		}
	}

	p("}")
	p("")
}

// generateClientClass generates the client class with constructor and methods.
func (g *Generator) generateClientClass(p printer, service *protogen.Service) {
	serviceName := service.GoName

	p("export class %sClient {", serviceName)

	// Private fields
	p("  private baseURL: string;")
	p("  private fetchFn: typeof fetch;")
	p("  private defaultHeaders: Record<string, string>;")
	p("")

	// Constructor
	g.generateConstructor(p, service)

	// RPC methods
	for _, method := range service.Methods {
		g.generateRPCMethod(p, service, method)
	}

	// Error handler
	g.generateHandleError(p)

	p("}")
	p("")
}

// generateConstructor generates the client constructor.
func (g *Generator) generateConstructor(p printer, service *protogen.Service) {
	serviceName := service.GoName

	p("  constructor(baseURL: string, options?: %sClientOptions) {", serviceName)
	p(`    this.baseURL = baseURL.replace(/\/+$/, "");`)
	p("    this.fetchFn = options?.fetch ?? globalThis.fetch;")
	p("    this.defaultHeaders = { ...options?.defaultHeaders };")

	// Apply service-level headers from options
	serviceHeaders := annotations.GetServiceHeaders(service)
	for _, header := range serviceHeaders {
		propName := headerNameToPropertyName(header.GetName())
		headerName := header.GetName()
		p("    if (options?.%s) {", propName)
		p(`      this.defaultHeaders["%s"] = options.%s;`, headerName, propName)
		p("    }")
	}

	p("  }")
	p("")
}

// rpcMethodConfig holds the configuration for generating an RPC method.
type rpcMethodConfig struct {
	serviceName string
	methodName  string
	httpMethod  string
	fullPath    string
	pathParams  []string
	queryParams []annotations.QueryParam
	hasBody     bool
	isSSE       bool
}

// Empty protobuf messages can still be meaningful request values, such as
// JSON bodies for POST/PUT/PATCH, so don't infer usage from field count alone.
func (cfg *rpcMethodConfig) requestParamName() string {
	usesQueryParams := (cfg.httpMethod == http.MethodGet || cfg.httpMethod == http.MethodDelete) &&
		len(cfg.queryParams) > 0
	if cfg.hasBody || len(cfg.pathParams) > 0 || usesQueryParams {
		return "req"
	}
	return "_req"
}

func (g *Generator) buildRPCMethodConfig(service *protogen.Service, method *protogen.Method) *rpcMethodConfig {
	serviceName := service.GoName
	methodName := method.GoName

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
		methodName:  methodName,
		httpMethod:  httpMethod,
		fullPath:    fullPath,
		pathParams:  pathParams,
		queryParams: annotations.GetQueryParams(method.Input),
		hasBody:     httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH",
		isSSE:       isSSE,
	}
}

// generateRPCMethod generates a single async RPC method.
func (g *Generator) generateRPCMethod(p printer, service *protogen.Service, method *protogen.Method) {
	cfg := g.buildRPCMethodConfig(service, method)

	if cfg.isSSE {
		g.generateSSERPCMethod(p, service, method, cfg)
		return
	}

	es := g.ctx.MessageRuntime == tscommon.MessageRuntimeES

	var inputType, outputType, reqSchema, resSchema string
	if es {
		// protobuf-es mode: consumers pass a plain init shape; the client routes
		// the request through create/toJson and decodes the response with fromJson.
		reqSchema = g.ctx.RefMessageSchema(method.Input)
		resSchema = g.ctx.RefMessageSchema(method.Output)
		inputType = "MessageInitShape<typeof " + reqSchema + ">"
		outputType = g.ctx.RefMessagePb(method.Output)
	} else {
		inputType = g.ctx.RefMessage(method.Input)
		outputType = g.resolveOutputType(method)
	}

	tsMethodName := annotations.LowerFirst(cfg.methodName)

	reqParam := cfg.requestParamName()
	p("  async %s(%s: %s, options?: %sCallOptions): Promise<%s> {",
		tsMethodName, reqParam, inputType, cfg.serviceName, outputType)

	// Build URL with path params
	g.generateURLBuilding(p, cfg)

	// Build headers
	g.generateHeaderMerging(p, service, method)

	// Build fetch options
	g.generateFetchCall(p, cfg, reqSchema)

	// Handle response
	g.generateResponseHandling(p, outputType, resSchema)

	p("  }")
	p("")
}

// generateSSERPCMethod generates an async generator method for SSE streaming.
func (g *Generator) generateSSERPCMethod(
	p printer,
	service *protogen.Service,
	method *protogen.Method,
	cfg *rpcMethodConfig,
) {
	es := g.ctx.MessageRuntime == tscommon.MessageRuntimeES

	var inputType, outputType, reqSchema, resSchema string
	if es {
		// protobuf-es mode mirrors the unary path: the request is an init shape
		// encoded through create/toJson (when a body is sent), and each streamed
		// event is decoded with fromJson. The generator yields the decoded message.
		reqSchema = g.ctx.RefMessageSchema(method.Input)
		resSchema = g.ctx.RefMessageSchema(method.Output)
		inputType = "MessageInitShape<typeof " + reqSchema + ">"
		outputType = g.ctx.RefMessagePb(method.Output)
	} else {
		inputType = g.ctx.RefMessage(method.Input)
		outputType = g.resolveOutputType(method)
	}
	tsMethodName := annotations.LowerFirst(cfg.methodName)

	reqParam := cfg.requestParamName()
	p("  async *%s(%s: %s, options?: %sCallOptions): AsyncGenerator<%s> {",
		tsMethodName, reqParam, inputType, cfg.serviceName, outputType)

	// Build URL with path params
	g.generateURLBuilding(p, cfg)

	// Build headers (use Accept instead of Content-Type for SSE)
	g.generateSSEHeaderMerging(p, service, method)

	// Fetch call
	g.generateSSEFetchCall(p, cfg, reqSchema)

	// SSE stream parsing
	g.generateSSEStreamParsing(p, outputType, resSchema)

	p("  }")
	p("")
}

// generateSSEHeaderMerging generates header construction for SSE requests.
func (g *Generator) generateSSEHeaderMerging(p printer, service *protogen.Service, method *protogen.Method) {
	p("    const headers: Record<string, string> = {")
	p(`      "Accept": "text/event-stream",`)
	p("      ...this.defaultHeaders,")
	p("      ...options?.headers,")
	p("    };")

	// Apply service-level headers from call options
	serviceHeaders := annotations.GetServiceHeaders(service)
	for _, header := range serviceHeaders {
		propName := headerNameToPropertyName(header.GetName())
		headerName := header.GetName()
		p("    if (options?.%s) headers[\"%s\"] = options.%s;", propName, headerName, propName)
	}

	// Apply method-level headers from call options
	methodHeaders := annotations.GetMethodHeaders(method)
	for _, header := range methodHeaders {
		propName := headerNameToPropertyName(header.GetName())
		headerName := header.GetName()
		p("    if (options?.%s) headers[\"%s\"] = options.%s;", propName, headerName, propName)
	}

	p("")
}

// generateSSEFetchCall generates the fetch invocation for SSE. In protobuf-es
// mode (reqSchema non-empty) a request body is encoded through create/toJson,
// matching the unary path; otherwise the request object is serialized directly.
func (g *Generator) generateSSEFetchCall(p printer, cfg *rpcMethodConfig, reqSchema string) {
	if cfg.hasBody {
		body := "JSON.stringify(req)"
		if reqSchema != "" {
			body = fmt.Sprintf("JSON.stringify(toJson(%s, create(%s, req)))", reqSchema, reqSchema)
		}
		p("    const resp = await this.fetchFn(url, {")
		p(`      method: "%s",`, cfg.httpMethod)
		p("      headers,")
		p("      body: %s,", body)
		p("      signal: options?.signal,")
		p("    });")
	} else {
		p("    const resp = await this.fetchFn(url, {")
		p(`      method: "%s",`, cfg.httpMethod)
		p("      headers,")
		p("      signal: options?.signal,")
		p("    });")
	}
	p("")

	p("    if (!resp.ok) {")
	p("      return this.handleError(resp);")
	p("    }")
	p("")
}

// generateSSEStreamParsing generates the ReadableStream SSE parsing logic. In
// protobuf-es mode (resSchema non-empty) each streamed event is decoded through
// fromJson with ignoreUnknownFields (forward-compat, mandatory); otherwise the
// parsed JSON is raw-cast to the output type.
func (g *Generator) generateSSEStreamParsing(p printer, outputType, resSchema string) {
	p("    const reader = resp.body!.getReader();")
	p("    const decoder = new TextDecoder();")
	p(`    let buffer = "";`)
	p("")
	p("    try {")
	p("      while (true) {")
	p("        const { done, value } = await reader.read();")
	p("        if (done) break;")
	p("        buffer += decoder.decode(value, { stream: true });")
	p(`        const lines = buffer.split("\n");`)
	p(`        buffer = lines.pop() || "";`)
	p("        for (const line of lines) {")
	p(`          if (line.startsWith("data: ")) {`)
	p("            const data = line.slice(6);")
	if resSchema != "" {
		p("            yield fromJson(%s, JSON.parse(data), { ignoreUnknownFields: true });", resSchema)
	} else {
		p("            yield JSON.parse(data) as %s;", outputType)
	}
	p("          }")
	p("        }")
	p("      }")
	p("    } finally {")
	p("      reader.releaseLock();")
	p("    }")
}

// resolveOutputType returns the TypeScript return type, handling root unwrap.
func (g *Generator) resolveOutputType(method *protogen.Method) string {
	msg := method.Output
	if annotations.IsRootUnwrap(msg) {
		return tscommon.RootUnwrapTSTypeCtx(g.ctx, msg)
	}
	return g.ctx.RefMessage(msg)
}

// generateURLBuilding generates URL construction with path and query params.
func (g *Generator) generateURLBuilding(p printer, cfg *rpcMethodConfig) {
	p(`    let path = "%s";`, cfg.fullPath)

	// Path parameter substitution
	for _, param := range cfg.pathParams {
		jsonName := snakeToLowerCamel(param)
		p(`    path = path.replace("{%s}", encodeURIComponent(String(req.%s)));`, param, jsonName)
	}

	// Query parameters
	//nolint:nestif // Query param generation requires multiple nested conditions
	if (cfg.httpMethod == "GET" || cfg.httpMethod == "DELETE") && len(cfg.queryParams) > 0 {
		p("    const params = new URLSearchParams();")
		for _, qp := range cfg.queryParams {
			// Handle repeated fields: use forEach + append for multi-value params
			if qp.Field != nil && qp.Field.Desc.IsList() {
				p("    if (req.%s && req.%s.length > 0) req.%s.forEach(v => params.append(\"%s\", v));",
					qp.FieldJSONName, qp.FieldJSONName, qp.FieldJSONName, qp.ParamName)
				continue
			}

			// Use field-aware zero check when field reference is available
			var check string
			if qp.Field != nil {
				check = tsZeroCheckForField(qp.Field)
			} else {
				check = tsZeroCheck(qp.FieldKind)
			}
			if check == "" {
				// bool: only add if true (undefined is already falsy)
				p("    if (req.%s) params.set(\"%s\", String(req.%s));",
					qp.FieldJSONName, qp.ParamName, qp.FieldJSONName)
			} else {
				// Guard against undefined/null before zero-value check
				p("    if (req.%s != null && req.%s%s) params.set(\"%s\", String(req.%s));",
					qp.FieldJSONName, qp.FieldJSONName, check, qp.ParamName, qp.FieldJSONName)
			}
		}
		p(`    const url = this.baseURL + path + (params.toString() ? "?" + params.toString() : "");`)
	} else {
		p("    const url = this.baseURL + path;")
	}

	p("")
}

// generateHeaderMerging generates header construction from defaults + options.
func (g *Generator) generateHeaderMerging(p printer, service *protogen.Service, method *protogen.Method) {
	p("    const headers: Record<string, string> = {")
	p(`      "Content-Type": "application/json",`)
	p("      ...this.defaultHeaders,")
	p("      ...options?.headers,")
	p("    };")

	// Apply service-level headers from call options
	serviceHeaders := annotations.GetServiceHeaders(service)
	for _, header := range serviceHeaders {
		propName := headerNameToPropertyName(header.GetName())
		headerName := header.GetName()
		p("    if (options?.%s) headers[\"%s\"] = options.%s;", propName, headerName, propName)
	}

	// Apply method-level headers from call options
	methodHeaders := annotations.GetMethodHeaders(method)
	for _, header := range methodHeaders {
		propName := headerNameToPropertyName(header.GetName())
		headerName := header.GetName()
		p("    if (options?.%s) headers[\"%s\"] = options.%s;", propName, headerName, propName)
	}

	p("")
}

// generateFetchCall generates the fetch invocation. In protobuf-es mode
// (reqSchema non-empty) the request body is encoded through create/toJson;
// otherwise the request object is serialized directly.
func (g *Generator) generateFetchCall(p printer, cfg *rpcMethodConfig, reqSchema string) {
	if cfg.hasBody {
		body := "JSON.stringify(req)"
		if reqSchema != "" {
			body = fmt.Sprintf("JSON.stringify(toJson(%s, create(%s, req)))", reqSchema, reqSchema)
		}
		p("    const resp = await this.fetchFn(url, {")
		p(`      method: "%s",`, cfg.httpMethod)
		p("      headers,")
		p("      body: %s,", body)
		p("      signal: options?.signal,")
		p("    });")
	} else {
		p("    const resp = await this.fetchFn(url, {")
		p(`      method: "%s",`, cfg.httpMethod)
		p("      headers,")
		p("      signal: options?.signal,")
		p("    });")
	}
	p("")
}

// generateResponseHandling generates response parsing and error handling. In
// protobuf-es mode (resSchema non-empty) the response is decoded through fromJson
// with ignoreUnknownFields (forward-compat); otherwise it is raw-cast.
func (g *Generator) generateResponseHandling(p printer, outputType, resSchema string) {
	p("    if (!resp.ok) {")
	p("      return this.handleError(resp);")
	p("    }")
	p("")
	if resSchema != "" {
		p("    return fromJson(%s, await resp.json(), { ignoreUnknownFields: true });", resSchema)
		return
	}
	p("    return await resp.json() as %s;", outputType)
}

// generateHandleError generates the private error handler method.
func (g *Generator) generateHandleError(p printer) {
	p("  private async handleError(resp: Response): Promise<never> {")
	p("    const body = await resp.text();")
	p("    if (resp.status === 400) {")
	p("      try {")
	p("        const parsed = JSON.parse(body);")
	p("        if (parsed.violations) {")
	p("          throw new ValidationError(parsed.violations);")
	p("        }")
	p("      } catch (e) {")
	p("        if (e instanceof ValidationError) throw e;")
	p("      }")
	p("    }")
	p("    throw new ApiError(resp.status, `Request failed with status ${resp.status}`, body);")
	p("  }")
}
