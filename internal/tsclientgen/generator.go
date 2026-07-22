package tsclientgen

import (
	"fmt"
	"net/http"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

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
	// errorHandling selects how client methods surface failures (throw vs Result).
	errorHandling tscommon.ErrorHandling
}

// New creates a new TypeScript client generator.
func New(
	plugin *protogen.Plugin,
	runtime tscommon.MessageRuntime,
	errorHandling tscommon.ErrorHandling,
) *Generator {
	return &Generator{plugin: plugin, runtime: runtime, errorHandling: errorHandling}
}

// recv is the member-access prefix for the client's shared config in generated
// method bodies. The hand-rolled client is a class, so config lives on the
// instance ("this."); the protobuf-es standalone functions read config from
// per-call locals (baseURL/fetchFn) and a top-level handleError, so no prefix.
func (g *Generator) recv() string {
	if g.ctx.MessageRuntime == tscommon.MessageRuntimeES {
		return ""
	}
	return "this."
}

// Generate emits one canonical type module per proto file, a shared errors
// module, and one slimmed client module per service file.
func (g *Generator) Generate() error {
	return g.generateModules()
}

func (g *Generator) generateServiceClient(p printer, service *protogen.Service) error {
	serviceName := service.GoName

	// Enum path/query parameters are not representable in protobuf-es mode
	// (numeric enums vs string URL params); fail loud rather than emit code that
	// fails downstream tsc. Body/response messages carrying JSON-mapping
	// annotations es-mode cannot honor are rejected the same way.
	if g.ctx.MessageRuntime == tscommon.MessageRuntimeES {
		for _, method := range service.Methods {
			if err := checkNoEnumParamsES(service, method); err != nil {
				return err
			}
			role := "response"
			if cfg := annotations.GetMethodHTTPConfig(method); cfg != nil && cfg.Stream {
				role = "SSE event"
			}
			if err := tscommon.CheckESMessageAnnotations(
				service.GoName, method.GoName, "request", method.Input,
			); err != nil {
				return err
			}
			if err := tscommon.CheckESMessageAnnotations(
				service.GoName, method.GoName, role, method.Output,
			); err != nil {
				return err
			}
		}
	}

	// protobuf-es mode emits standalone, tree-shakeable per-RPC functions that
	// take their config (baseURL/fetch/headers) per call; the hand-rolled runtime
	// emits a configure-once client class.
	if g.ctx.MessageRuntime == tscommon.MessageRuntimeES {
		// Only services that declare typed headers need their own options type
		// (extending the shared RequestOptions); the rest use the shared type.
		if serviceHasHeaderProps(service) {
			g.generateRequestOptionsInterface(p, service)
		}
		g.generateStandaloneFunctions(p, service)
	} else {
		g.generateClientOptionsInterface(p, service)
		g.generateCallOptionsInterface(p, service)
		g.generateClientClass(p, service)
	}

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

// handleErrorNeeded reports whether the private handleError helper is referenced
// by any emitted method. In throw mode every method (unary + SSE) calls it; in
// Result mode only SSE methods do (unary methods return via decodeError).
func (g *Generator) handleErrorNeeded(service *protogen.Service) bool {
	if g.ctx.ErrorHandling != tscommon.ErrorHandlingResult {
		return len(service.Methods) > 0
	}
	for _, m := range service.Methods {
		if cfg := annotations.GetMethodHTTPConfig(m); cfg != nil && cfg.Stream {
			return true
		}
	}
	return false
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

	// Error handler. In Result mode unary methods return errors via decodeError
	// and never call handleError; only SSE methods still throw through it, so
	// omit it when there is no SSE method (else it trips noUnusedLocals).
	if g.handleErrorNeeded(service) {
		g.generateHandleError(p)
	}

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

// serviceHasHeaderProps reports whether the service declares any typed header
// (service- or method-level), i.e. whether its RequestOptions needs extra typed
// properties beyond the shared base.
func serviceHasHeaderProps(service *protogen.Service) bool {
	if len(annotations.GetServiceHeaders(service)) > 0 {
		return true
	}
	for _, method := range service.Methods {
		if len(annotations.GetMethodHeaders(method)) > 0 {
			return true
		}
	}
	return false
}

// esRequestOptionsType returns the options type name used in a standalone RPC
// function's signature: the shared RequestOptions (imported from client.ts) for
// header-less services, or the service's own {Service}RequestOptions (which
// extends the shared base) when it declares typed headers.
func (g *Generator) esRequestOptionsType(service *protogen.Service) string {
	if serviceHasHeaderProps(service) {
		return service.GoName + "RequestOptions"
	}
	return g.ctx.RefClient("RequestOptions", true)
}

// generateRequestOptionsInterface generates the {Service}RequestOptions interface
// for a service that declares typed headers. It extends the shared RequestOptions
// base (baseURL/fetch/headers/signal) and adds only the typed service-/method-
// level header properties, so the common fields are not duplicated per service.
func (g *Generator) generateRequestOptionsInterface(p printer, service *protogen.Service) {
	serviceName := service.GoName
	base := g.ctx.RefClient("RequestOptions", true)

	p("export interface %sRequestOptions extends %s {", serviceName, base)

	// Typed properties for service- and method-level headers (deduped).
	seen := make(map[string]bool)
	for _, header := range annotations.GetServiceHeaders(service) {
		propName := headerNameToPropertyName(header.GetName())
		if seen[propName] {
			continue
		}
		seen[propName] = true
		p("  %s?: string;", propName)
	}
	for _, method := range service.Methods {
		for _, header := range annotations.GetMethodHeaders(method) {
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

// generateStandaloneFunctions emits the protobuf-es client as one top-level
// exported function per RPC (plus a shared, unexported handleError when needed).
// Standalone functions let bundlers tree-shake unused RPCs — the whole service
// is not pulled in just because one method is imported.
func (g *Generator) generateStandaloneFunctions(p printer, service *protogen.Service) {
	for _, method := range service.Methods {
		g.generateRPCMethod(p, service, method)
	}

	// Shared error handler. Omitted when unreferenced (Result mode with no SSE
	// method) so it does not trip noUnusedLocals.
	if g.handleErrorNeeded(service) {
		g.generateHandleError(p)
		p("")
	}
}

// generateESConfigPrologue emits the per-call config setup at the top of a
// standalone protobuf-es function body: baseURL normalization and fetch
// resolution from the required options argument.
func (g *Generator) generateESConfigPrologue(p printer) {
	p(`    const baseURL = options.baseURL.replace(/\/+$/, "");`)
	p("    const fetchFn = options.fetch ?? globalThis.fetch;")
}

// rpcMethodConfig holds the configuration for generating an RPC method.

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

// checkNoEnumParamsES reports a generation-time error if the method has an
// enum-typed path or query parameter (including repeated enum query params),
// which protobuf-es mode cannot yet represent. See UnsupportedEnumParamError.
func checkNoEnumParamsES(service *protogen.Service, method *protogen.Method) error {
	httpConfig := annotations.GetMethodHTTPConfig(method)
	var pathParams []string
	if httpConfig != nil {
		pathParams = httpConfig.PathParams
	}
	if len(pathParams) > 0 {
		fieldMap := make(map[string]*protogen.Field, len(method.Input.Fields))
		for _, f := range method.Input.Fields {
			fieldMap[string(f.Desc.Name())] = f
		}
		for _, param := range pathParams {
			if f, ok := fieldMap[param]; ok && f.Desc.Kind() == protoreflect.EnumKind {
				return tscommon.UnsupportedEnumParamError("path", param, service.GoName, method.GoName)
			}
		}
	}
	for _, qp := range annotations.GetQueryParams(method.Input) {
		if qp.Field != nil && qp.Field.Desc.Kind() == protoreflect.EnumKind {
			return tscommon.UnsupportedEnumParamError("query", qp.FieldName, service.GoName, method.GoName)
		}
	}
	return nil
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
		g.ctx.NeedProtobufES("MessageInitShape", "fromJson")
		if cfg.hasBody {
			g.ctx.NeedProtobufES("create", "toJson")
		}
	} else {
		inputType = g.ctx.RefMessage(method.Input)
		outputType = g.resolveOutputType(method)
	}

	tsMethodName := annotations.LowerFirst(cfg.methodName)

	// In es + Result mode the method returns a discriminated Result union
	// (never throws); otherwise it returns the decoded message directly.
	returnType := outputType
	if es && g.ctx.ErrorHandling == tscommon.ErrorHandlingResult {
		resultT := g.ctx.RefResult("Result", true)
		clientErr := g.ctx.RefResult("ClientError", true)
		returnType = fmt.Sprintf("%s<%s, %s>", resultT, outputType, clientErr)
	}

	reqParam := cfg.requestParamName()
	if es {
		// Standalone, tree-shakeable function; config passed per call.
		p("export async function %s(%s: %s, options: %s): Promise<%s> {",
			tsMethodName, reqParam, inputType, g.esRequestOptionsType(service), returnType)
		g.generateESConfigPrologue(p)
	} else {
		p("  async %s(%s: %s, options?: %sCallOptions): Promise<%s> {",
			tsMethodName, reqParam, inputType, cfg.serviceName, returnType)
	}

	// Build URL with path params
	g.generateURLBuilding(p, cfg)

	// Build headers
	g.generateHeaderMerging(p, service, method)

	// Build fetch options
	g.generateFetchCall(p, cfg, reqSchema)

	// Handle response
	g.generateResponseHandling(p, outputType, resSchema)

	if es {
		p("}")
	} else {
		p("  }")
	}
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
		g.ctx.NeedProtobufES("MessageInitShape", "fromJson")
		if cfg.hasBody {
			g.ctx.NeedProtobufES("create", "toJson")
		}
	} else {
		inputType = g.ctx.RefMessage(method.Input)
		outputType = g.resolveOutputType(method)
	}
	tsMethodName := annotations.LowerFirst(cfg.methodName)

	reqParam := cfg.requestParamName()
	if es {
		p("export async function* %s(%s: %s, options: %s): AsyncGenerator<%s> {",
			tsMethodName, reqParam, inputType, g.esRequestOptionsType(service), outputType)
		g.generateESConfigPrologue(p)
	} else {
		p("  async *%s(%s: %s, options?: %sCallOptions): AsyncGenerator<%s> {",
			tsMethodName, reqParam, inputType, cfg.serviceName, outputType)
	}

	// Build URL with path params
	g.generateURLBuilding(p, cfg)

	// Build headers (use Accept instead of Content-Type for SSE)
	g.generateSSEHeaderMerging(p, service, method)

	// Fetch call
	g.generateSSEFetchCall(p, cfg, reqSchema)

	// SSE stream parsing
	g.generateSSEStreamParsing(p, outputType, resSchema)

	if es {
		p("}")
	} else {
		p("  }")
	}
	p("")
}

// generateSSEHeaderMerging generates header construction for SSE requests.
func (g *Generator) generateSSEHeaderMerging(p printer, service *protogen.Service, method *protogen.Method) {
	p("    const headers: Record<string, string> = {")
	p(`      "Accept": "text/event-stream",`)
	if g.ctx.MessageRuntime != tscommon.MessageRuntimeES {
		p("      ...%sdefaultHeaders,", g.recv())
	}
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
		p("    const resp = await %sfetchFn(url, {", g.recv())
		p(`      method: "%s",`, cfg.httpMethod)
		p("      headers,")
		p("      body: %s,", body)
		p("      signal: options?.signal,")
		p("    });")
	} else {
		p("    const resp = await %sfetchFn(url, {", g.recv())
		p(`      method: "%s",`, cfg.httpMethod)
		p("      headers,")
		p("      signal: options?.signal,")
		p("    });")
	}
	p("")

	p("    if (!resp.ok) {")
	p("      return %shandleError(resp);", g.recv())
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
	// `path` is only reassigned when there are path params to substitute; declare
	// it const otherwise so the generated code reflects that it never changes.
	pathDecl := "const"
	if len(cfg.pathParams) > 0 {
		pathDecl = "let"
	}
	p(`    %s path = "%s";`, pathDecl, cfg.fullPath)

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
				// String(v) coerces non-string element types (int32/bool/…) so the
				// URLSearchParams.append call typechecks; it is a no-op for strings.
				p("    if (req.%s && req.%s.length > 0) req.%s.forEach(v => params.append(\"%s\", String(v)));",
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
		p(`    const url = %sbaseURL + path + (params.toString() ? "?" + params.toString() : "");`, g.recv())
	} else {
		p("    const url = %sbaseURL + path;", g.recv())
	}

	p("")
}

// generateHeaderMerging generates header construction from defaults + options.
func (g *Generator) generateHeaderMerging(p printer, service *protogen.Service, method *protogen.Method) {
	p("    const headers: Record<string, string> = {")
	p(`      "Content-Type": "application/json",`)
	if g.ctx.MessageRuntime != tscommon.MessageRuntimeES {
		p("      ...%sdefaultHeaders,", g.recv())
	}
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
		p("    const resp = await %sfetchFn(url, {", g.recv())
		p(`      method: "%s",`, cfg.httpMethod)
		p("      headers,")
		p("      body: %s,", body)
		p("      signal: options?.signal,")
		p("    });")
	} else {
		p("    const resp = await %sfetchFn(url, {", g.recv())
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
	// es + Result mode: return the discriminated union instead of throwing.
	// resSchema is always non-empty here (Result requires es).
	if resSchema != "" && g.ctx.ErrorHandling == tscommon.ErrorHandlingResult {
		decodeErr := g.ctx.RefResult("decodeError", false)
		p("    if (!resp.ok) {")
		p("      return { ok: false, error: await %s(resp) };", decodeErr)
		p("    }")
		p("")
		p("    return { ok: true, data: fromJson(%s, await resp.json(), { ignoreUnknownFields: true }) };", resSchema)
		return
	}

	p("    if (!resp.ok) {")
	p("      return %shandleError(resp);", g.recv())
	p("    }")
	p("")
	if resSchema != "" {
		p("    return fromJson(%s, await resp.json(), { ignoreUnknownFields: true });", resSchema)
		return
	}
	p("    return await resp.json() as %s;", outputType)
}

// generateHandleError generates the error handler. In the hand-rolled class it is
// a private method; in protobuf-es mode it is a top-level (unexported) function
// shared by the standalone RPC functions in the file.
func (g *Generator) generateHandleError(p printer) {
	es := g.ctx.MessageRuntime == tscommon.MessageRuntimeES
	if es {
		p("async function handleError(resp: Response): Promise<never> {")
	} else {
		p("  private async handleError(resp: Response): Promise<never> {")
	}
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
	if es {
		p("}")
	} else {
		p("  }")
	}
}
