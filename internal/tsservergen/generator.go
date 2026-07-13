package tsservergen

import (
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	sebufhttp "github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

// Generator handles TypeScript server code generation for protobuf services.
type Generator struct {
	plugin *protogen.Plugin
	// ctx carries the emission state (self module + import tracker) for the
	// service file currently being written.
	ctx *tscommon.EmitContext
	// runtime selects the TypeScript message representation.
	runtime tscommon.MessageRuntime
}

// New creates a new TypeScript server generator.
func New(plugin *protogen.Plugin, runtime tscommon.MessageRuntime) *Generator {
	return &Generator{plugin: plugin, runtime: runtime}
}

// Generate emits one canonical type module per proto file, a shared errors
// module, and one slimmed server module per service file.
func (g *Generator) Generate() error {
	return g.generateModules()
}

func (g *Generator) writeServerTypes(p tscommon.Printer) {
	// ServerContext
	p("export interface ServerContext {")
	p("  request: Request;")
	p("  pathParams: Record<string, string>;")
	p("  headers: Record<string, string>;")
	p("}")
	p("")

	// ServerOptions
	p("export interface ServerOptions {")
	p("  onError?: (error: unknown, req: Request) => Response | Promise<Response>;")
	p("  validateRequest?: (methodName: string, body: unknown) => FieldViolation[] | undefined;")
	p("}")
	p("")

	// RouteDescriptor
	p("export interface RouteDescriptor {")
	p("  method: string;")
	p("  path: string;")
	p("  handler: (req: Request) => Promise<Response>;")
	p("}")
	p("")
}

// fileUsesHeaders returns true if any service in the file uses header annotations.
func (g *Generator) fileUsesHeaders(file *protogen.File) bool {
	for _, service := range file.Services {
		if len(annotations.GetServiceHeaders(service)) > 0 {
			return true
		}
		for _, method := range service.Methods {
			if len(annotations.GetMethodHeaders(method)) > 0 {
				return true
			}
		}
	}
	return false
}

// writeHeaderValidationHelpers writes format/type validation helper functions.
func (g *Generator) writeHeaderValidationHelpers(p tscommon.Printer) {
	g.writeHeaderRegexConstants(p)
	g.writeHeaderConfigType(p)
	g.writeValidateHeaderValueFn(p)
	g.writeValidateHeadersFn(p)
}

func (g *Generator) writeHeaderRegexConstants(p tscommon.Printer) {
	p("const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;")
	p("")
	p("const EMAIL_REGEX = /^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/;")
	p("")
	p("const DATETIME_REGEX = /^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(\\.[\\d]+)?(Z|[+-]\\d{2}:\\d{2})$/;")
	p("")
	p("const DATE_REGEX = /^\\d{4}-\\d{2}-\\d{2}$/;")
	p("")
	p("const TIME_REGEX = /^\\d{2}:\\d{2}:\\d{2}(\\.[\\d]+)?$/;")
	p("")
}

func (g *Generator) writeHeaderConfigType(p tscommon.Printer) {
	p("interface HeaderConfig {")
	p("  name: string;")
	p("  type: string;")
	p("  required: boolean;")
	p("  format?: string;")
	p("}")
	p("")
}

func (g *Generator) writeValidateHeaderValueFn(p tscommon.Printer) {
	p("function validateHeaderValue(value: string, config: HeaderConfig): string | undefined {")
	p("  switch (config.type) {")
	p(`    case "integer":`)
	p(`      if (!/^-?\d+$/.test(value)) return "must be an integer";`)
	p("      break;")
	p(`    case "number":`)
	p(`      if (isNaN(Number(value))) return "must be a number";`)
	p("      break;")
	p(`    case "boolean":`)
	p(`      if (value !== "true" && value !== "false" && value !== "1" && value !== "0")`)
	p(`        return "must be a boolean";`)
	p("      break;")
	p("  }")
	p("  if (config.format) {")
	p("    switch (config.format) {")
	p(`      case "uuid":`)
	p(`        if (!UUID_REGEX.test(value)) return "must be a valid UUID";`)
	p("        break;")
	p(`      case "email":`)
	p(`        if (!EMAIL_REGEX.test(value)) return "must be a valid email";`)
	p("        break;")
	p(`      case "date-time":`)
	p(`        if (!DATETIME_REGEX.test(value)) return "must be a valid date-time";`)
	p("        break;")
	p(`      case "date":`)
	p(`        if (!DATE_REGEX.test(value)) return "must be a valid date";`)
	p("        break;")
	p(`      case "time":`)
	p(`        if (!TIME_REGEX.test(value)) return "must be a valid time";`)
	p("        break;")
	p("    }")
	p("  }")
	p("  return undefined;")
	p("}")
	p("")
}

func (g *Generator) writeValidateHeadersFn(p tscommon.Printer) {
	p("function validateHeaders(")
	p("  req: Request,")
	p("  configs: HeaderConfig[],")
	p("): FieldViolation[] | undefined {")
	p("  const violations: FieldViolation[] = [];")
	p("  for (const config of configs) {")
	p("    const value = req.headers.get(config.name);")
	p("    if (value == null) {")
	p("      if (config.required) {")
	p("        violations.push({")
	p("          field: config.name,")
	p(`          description: "required header is missing",`)
	p("        });")
	p("      }")
	p("      continue;")
	p("    }")
	p("    const err = validateHeaderValue(value, config);")
	p("    if (err) {")
	p("      violations.push({")
	p("        field: config.name,")
	p("        description: `header ${config.name}: ${err}`,")
	p("      });")
	p("    }")
	p("  }")
	p("  return violations.length > 0 ? violations : undefined;")
	p("}")
	p("")
}

func (g *Generator) generateService(p tscommon.Printer, service *protogen.Service) error {
	// Handler interface
	g.generateHandlerInterface(p, service)

	// Route creation function
	return g.generateCreateRoutes(p, service)
}

// isSSEMethod checks if a method is annotated as SSE streaming.
func (g *Generator) isSSEMethod(method *protogen.Method) bool {
	config := annotations.GetMethodHTTPConfig(method)
	return config != nil && config.Stream
}

// generateHandlerInterface generates the XxxServiceHandler interface. In
// protobuf-es mode the handler receives the decoded request as its branded
// protoc-gen-es message type (imported from <proto>_pb.js) and returns a
// MessageInitShape of the response schema, so implementations may return either
// a plain init object or an already-branded message; the generated route wraps
// the return value in create(...) before encoding.
func (g *Generator) generateHandlerInterface(p tscommon.Printer, service *protogen.Service) {
	serviceName := service.GoName
	es := g.ctx.MessageRuntime == tscommon.MessageRuntimeES

	p("export interface %sHandler {", serviceName)
	for _, method := range service.Methods {
		methodName := annotations.LowerFirst(method.GoName)
		var inputType, outputType string
		if es {
			inputType = g.ctx.RefMessagePb(method.Input)
			outputType = "MessageInitShape<typeof " + g.ctx.RefMessageSchema(method.Output) + ">"
		} else {
			inputType = g.ctx.RefMessage(method.Input)
			outputType = g.resolveOutputType(method)
		}
		if g.isSSEMethod(method) {
			p("  %s(ctx: ServerContext, req: %s): ReadableStream<%s>;", methodName, inputType, outputType)
		} else {
			p("  %s(ctx: ServerContext, req: %s): Promise<%s>;", methodName, inputType, outputType)
		}
	}
	p("}")
	p("")
}

// resolveOutputType returns the TypeScript return type, handling root unwrap.
func (g *Generator) resolveOutputType(method *protogen.Method) string {
	msg := method.Output
	if annotations.IsRootUnwrap(msg) {
		return tscommon.RootUnwrapTSTypeCtx(g.ctx, msg)
	}
	return g.ctx.RefMessage(msg)
}

// pathParamField maps a URL path parameter to its corresponding request message field.
type pathParamField struct {
	protoName string // proto field name, e.g. "resource_id"
	jsonName  string // JSON/TS field name, e.g. "resourceId"
	field     *protogen.Field
}

// rpcRouteConfig holds config for generating a route handler.
type rpcRouteConfig struct {
	serviceName     string
	methodName      string
	httpMethod      string
	fullPath        string
	pathParams      []string
	pathParamFields []pathParamField
	queryParams     []annotations.QueryParam
	hasBody         bool
}

func (g *Generator) buildRPCRouteConfig(service *protogen.Service, method *protogen.Method) (*rpcRouteConfig, error) {
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

	basePath := annotations.GetServiceBasePath(service)
	fullPath := annotations.BuildHTTPPath(basePath, httpPath)

	// Validate and resolve path params against request message fields
	pathParamFields, err := resolvePathParamFields(pathParams, method)
	if err != nil {
		return nil, fmt.Errorf("service %s, method %s: %w", serviceName, methodName, err)
	}

	queryParams := annotations.GetQueryParams(method.Input)

	// Enum path/query parameters are not representable in protobuf-es mode
	// (numeric enums vs string URL params); fail loud rather than emit code that
	// fails downstream tsc.
	if g.ctx.MessageRuntime == tscommon.MessageRuntimeES {
		if enumErr := checkNoEnumParamsES(serviceName, methodName, pathParamFields, queryParams); enumErr != nil {
			return nil, enumErr
		}
	}

	return &rpcRouteConfig{
		serviceName:     serviceName,
		methodName:      methodName,
		httpMethod:      httpMethod,
		fullPath:        fullPath,
		pathParams:      pathParams,
		pathParamFields: pathParamFields,
		queryParams:     queryParams,
		hasBody:         httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH",
	}, nil
}

// checkNoEnumParamsES reports a generation-time error if any path or query
// parameter is enum-typed (including repeated enum query params), which
// protobuf-es mode cannot yet represent. See UnsupportedEnumParamError.
func checkNoEnumParamsES(
	service, method string,
	pathParamFields []pathParamField,
	queryParams []annotations.QueryParam,
) error {
	for _, ppf := range pathParamFields {
		if ppf.field != nil && ppf.field.Desc.Kind() == protoreflect.EnumKind {
			return tscommon.UnsupportedEnumParamError("path", ppf.protoName, service, method)
		}
	}
	for _, qp := range queryParams {
		if qp.Field != nil && qp.Field.Desc.Kind() == protoreflect.EnumKind {
			return tscommon.UnsupportedEnumParamError("query", qp.FieldName, service, method)
		}
	}
	return nil
}

// resolvePathParamFields validates that every path parameter has a matching field
// on the input message, and returns the proto→JSON name mapping.
func resolvePathParamFields(pathParams []string, method *protogen.Method) ([]pathParamField, error) {
	if len(pathParams) == 0 {
		return nil, nil
	}

	// Build lookup: proto field name → field
	fieldMap := make(map[string]*protogen.Field, len(method.Input.Fields))
	for _, f := range method.Input.Fields {
		fieldMap[string(f.Desc.Name())] = f
	}

	fields := make([]pathParamField, 0, len(pathParams))
	for _, param := range pathParams {
		f, ok := fieldMap[param]
		if !ok {
			return nil, fmt.Errorf(
				"path parameter {%s} has no matching field on request message %s — "+
					"add a field named '%s' to the message definition",
				param, method.Input.Desc.Name(), param,
			)
		}
		fields = append(fields, pathParamField{protoName: param, jsonName: f.Desc.JSONName(), field: f})
	}
	return fields, nil
}

// validateFieldCoverage verifies that for GET/DELETE methods, every field on the
// input message is accounted for — either as a path parameter or a query parameter.
// Fields that are neither would silently receive zero values, which is a proto definition error.
func validateFieldCoverage(cfg *rpcRouteConfig, method *protogen.Method) error {
	if cfg.hasBody {
		// POST/PUT/PATCH: non-path-param fields come from the JSON body — all fields are reachable
		return nil
	}

	// Build sets of covered field names (proto names)
	covered := make(map[string]bool, len(cfg.pathParams)+len(cfg.queryParams))
	for _, p := range cfg.pathParams {
		covered[p] = true
	}
	for _, q := range cfg.queryParams {
		covered[q.FieldName] = true
	}

	// Check every field on the input message
	var uncovered []string
	for _, f := range method.Input.Fields {
		name := string(f.Desc.Name())
		if !covered[name] {
			uncovered = append(uncovered, name)
		}
	}

	if len(uncovered) > 0 {
		return fmt.Errorf(
			"fields %v on request message %s are not reachable via path or query parameters for %s %s — "+
				"annotate them with (sebuf.http.query) or include them as path parameters",
			uncovered, method.Input.Desc.Name(), cfg.httpMethod, cfg.fullPath,
		)
	}
	return nil
}

// generateCreateRoutes generates the createXxxRoutes function.
func (g *Generator) generateCreateRoutes(p tscommon.Printer, service *protogen.Service) error {
	serviceName := service.GoName

	p("export function create%sRoutes(", serviceName)
	p("  handler: %sHandler,", serviceName)
	p("  options?: ServerOptions,")
	p("): RouteDescriptor[] {")
	p("  return [")
	for _, method := range service.Methods {
		if err := g.generateRouteEntry(p, service, method); err != nil {
			return err
		}
	}
	p("  ];")
	p("}")
	p("")
	return nil
}

// generateRouteEntry generates a single route descriptor entry.
//
//nolint:funlen // Route entry generation requires many sequential code generation blocks
func (g *Generator) generateRouteEntry(p tscommon.Printer, service *protogen.Service, method *protogen.Method) error {
	cfg, err := g.buildRPCRouteConfig(service, method)
	if err != nil {
		return err
	}
	if coverageErr := validateFieldCoverage(cfg, method); coverageErr != nil {
		return fmt.Errorf("service %s, method %s: %w", service.GoName, method.GoName, coverageErr)
	}

	if g.isSSEMethod(method) {
		return g.generateSSERouteEntry(p, service, method, cfg)
	}

	tsMethodName := annotations.LowerFirst(cfg.methodName)
	es := g.ctx.MessageRuntime == tscommon.MessageRuntimeES

	p("    {")
	p(`      method: "%s",`, cfg.httpMethod)
	p(`      path: "%s",`, cfg.fullPath)
	p("      handler: async (req: Request): Promise<Response> => {")

	// Try-catch wraps the handler body
	p("        try {")

	// Header validation (before body parsing)
	serviceHeaders := annotations.GetServiceHeaders(service)
	methodHeaders := annotations.GetMethodHeaders(method)
	g.generateHeaderValidation(p, serviceHeaders, methodHeaders)

	// Extract path params
	g.generatePathParamExtraction(p, cfg)

	// Parse request body or query params
	if cfg.hasBody {
		g.generateBodyParsing(p, method, tsMethodName)
		// For body methods, path params must be merged after JSON parse
		g.generatePathParamMerge(p, cfg, method)
	} else {
		// For non-body methods, path params are included in the body literal
		g.generateQueryParamParsing(p, cfg, method, tsMethodName)
	}

	// Build ServerContext
	p("          const ctx: ServerContext = {")
	p("            request: req,")
	p("            pathParams,")
	p("            headers: Object.fromEntries(req.headers.entries()),")
	p("          };")
	p("")

	// Call handler
	p("          const result = await handler.%s(ctx, body);", tsMethodName)

	// Return JSON response. In protobuf-es mode the result is normalized through
	// create(...) and serialized with toJson(...) so the wire shape matches the
	// canonical protojson encoding (Go-server parity); otherwise it is raw-cast.
	if es {
		resSchema := g.ctx.RefMessageSchema(method.Output)
		p("          return new Response(JSON.stringify(toJson(%s, create(%s, result))), {", resSchema, resSchema)
	} else {
		outputType := g.resolveOutputType(method)
		p("          return new Response(JSON.stringify(result as %s), {", outputType)
	}
	p("            status: 200,")
	p(`            headers: { "Content-Type": "application/json" },`)
	p("          });")

	// Catch block
	p("        } catch (err: unknown) {")
	p("          if (err instanceof ValidationError) {")
	p("            return new Response(JSON.stringify({ violations: err.violations }), {")
	p("              status: 400,")
	p(`              headers: { "Content-Type": "application/json" },`)
	p("            });")
	p("          }")
	p("          if (options?.onError) {")
	p("            return options.onError(err, req);")
	p("          }")
	p("          const message = err instanceof Error ? err.message : String(err);")
	p("          return new Response(JSON.stringify({ message }), {")
	p("            status: 500,")
	p(`            headers: { "Content-Type": "application/json" },`)
	p("          });")
	p("        }")

	p("      },")
	p("    },")
	return nil
}

// generateSSERouteEntry generates a route descriptor for an SSE streaming endpoint.
//
//nolint:funlen // SSE route entry generation requires many sequential code generation blocks
func (g *Generator) generateSSERouteEntry(
	p tscommon.Printer,
	service *protogen.Service,
	method *protogen.Method,
	cfg *rpcRouteConfig,
) error {
	tsMethodName := annotations.LowerFirst(cfg.methodName)
	es := g.ctx.MessageRuntime == tscommon.MessageRuntimeES

	p("    {")
	p(`      method: "%s",`, cfg.httpMethod)
	p(`      path: "%s",`, cfg.fullPath)
	p("      handler: async (req: Request): Promise<Response> => {")

	// Try-catch wraps the handler body
	p("        try {")

	// Header validation
	serviceHeaders := annotations.GetServiceHeaders(service)
	methodHeaders := annotations.GetMethodHeaders(method)
	g.generateHeaderValidation(p, serviceHeaders, methodHeaders)

	// Extract path params
	g.generatePathParamExtraction(p, cfg)

	// Parse request body or query params
	if cfg.hasBody {
		g.generateBodyParsing(p, method, tsMethodName)
		g.generatePathParamMerge(p, cfg, method)
	} else {
		g.generateQueryParamParsing(p, cfg, method, tsMethodName)
	}

	// Build ServerContext
	p("          const ctx: ServerContext = {")
	p("            request: req,")
	p("            pathParams,")
	p("            headers: Object.fromEntries(req.headers.entries()),")
	p("          };")
	p("")

	// Get the ReadableStream from handler
	p("          const stream = handler.%s(ctx, body);", tsMethodName)
	p("")

	// Convert ReadableStream<T> to SSE text stream
	p("          const sseStream = new ReadableStream({")
	p("            async start(controller) {")
	p("              const reader = stream.getReader();")
	p("              const encoder = new TextEncoder();")
	p("              try {")
	p("                while (true) {")
	p("                  const { done, value } = await reader.read();")
	p("                  if (done) break;")
	// In protobuf-es mode each yielded event is normalized through create(...)
	// and serialized with toJson(...) for canonical protojson (Go-server parity);
	// otherwise the value is stringified directly (no raw cast needed here).
	valueExpr := "value"
	if es {
		resSchema := g.ctx.RefMessageSchema(method.Output)
		valueExpr = fmt.Sprintf("toJson(%s, create(%s, value))", resSchema, resSchema)
	}
	p("                  controller.enqueue(encoder.encode(`data: ${JSON.stringify(%s)}\\n\\n`));", valueExpr)
	p("                }")
	p("                controller.close();")
	p("              } catch (err) {")
	p("                controller.enqueue(")
	p("                  encoder.encode(`event: error\\ndata: ${JSON.stringify({ message: String(err) })}\\n\\n`),")
	p("                );")
	p("                controller.close();")
	p("              }")
	p("            },")
	p("          });")
	p("")

	// Return SSE response
	p("          return new Response(sseStream, {")
	p("            headers: {")
	p(`              "Content-Type": "text/event-stream",`)
	p(`              "Cache-Control": "no-cache",`)
	p(`              "Connection": "keep-alive",`)
	p("            },")
	p("          });")

	// Catch block
	p("        } catch (err: unknown) {")
	p("          if (err instanceof ValidationError) {")
	p("            return new Response(JSON.stringify({ violations: err.violations }), {")
	p("              status: 400,")
	p(`              headers: { "Content-Type": "application/json" },`)
	p("            });")
	p("          }")
	p("          if (options?.onError) {")
	p("            return options.onError(err, req);")
	p("          }")
	p("          const message = err instanceof Error ? err.message : String(err);")
	p("          return new Response(JSON.stringify({ message }), {")
	p("            status: 500,")
	p(`            headers: { "Content-Type": "application/json" },`)
	p("          });")
	p("        }")

	p("      },")
	p("    },")
	return nil
}

// emitPathParamAssignment emits a single path param assignment with enum casting if needed.
func (g *Generator) emitPathParamAssignment(
	p tscommon.Printer,
	ppf pathParamField,
	prefix string,
	suffix string,
) {
	if ppf.field != nil && ppf.field.Desc.Kind() == protoreflect.EnumKind && ppf.field.Enum != nil {
		// Hand-rolled mode only: enums are string unions here, so the cast is
		// sound. protobuf-es mode rejects enum path params before emission (see
		// checkNoEnumParamsES).
		enumName := g.ctx.RefEnum(ppf.field.Enum)
		p(
			"%s%s: pathParams[\"%s\"] as %s%s",
			prefix, ppf.jsonName, ppf.protoName, enumName, suffix,
		)
	} else {
		p("%s%s: pathParams[\"%s\"]%s", prefix, ppf.jsonName, ppf.protoName, suffix)
	}
}

// generatePathParamMerge generates code to merge path params into the request body.
// This ensures the handler receives a fully populated request, matching Go generator behavior.
func (g *Generator) generatePathParamMerge(p tscommon.Printer, cfg *rpcRouteConfig, _ *protogen.Method) {
	if len(cfg.pathParamFields) == 0 {
		return
	}
	for _, ppf := range cfg.pathParamFields {
		if ppf.field != nil && ppf.field.Desc.Kind() == protoreflect.EnumKind && ppf.field.Enum != nil {
			// Hand-rolled mode only: protobuf-es mode rejects enum path params
			// before emission (see checkNoEnumParamsES).
			enumName := g.ctx.RefEnum(ppf.field.Enum)
			p(
				"          body.%s = pathParams[\"%s\"] as %s;",
				ppf.jsonName, ppf.protoName, enumName,
			)
		} else {
			p("          body.%s = pathParams[\"%s\"];", ppf.jsonName, ppf.protoName)
		}
	}
	p("")
}

// generateHeaderValidation generates header validation code.
func (g *Generator) generateHeaderValidation(
	p tscommon.Printer,
	serviceHeaders []*sebufhttp.Header,
	methodHeaders []*sebufhttp.Header,
) {
	// Combine all headers that need validation
	allHeaders := make([]*sebufhttp.Header, 0, len(serviceHeaders)+len(methodHeaders))
	allHeaders = append(allHeaders, serviceHeaders...)
	allHeaders = append(allHeaders, methodHeaders...)
	if len(allHeaders) == 0 {
		return
	}

	// Build inline header config for method-specific combination
	p("          const headerConfigs: HeaderConfig[] = [")
	for _, h := range allHeaders {
		formatStr := ""
		if h.GetFormat() != "" {
			formatStr = fmt.Sprintf(`, format: "%s"`, h.GetFormat())
		}
		p(`            { name: "%s", type: "%s", required: %t%s },`,
			h.GetName(), h.GetType(), h.GetRequired(), formatStr)
	}
	p("          ];")
	p("          const headerViolations = validateHeaders(req, headerConfigs);")
	p("          if (headerViolations) {")
	p("            throw new ValidationError(headerViolations);")
	p("          }")
	p("")
}

// generatePathParamExtraction generates code to extract path params from the URL.
func (g *Generator) generatePathParamExtraction(p tscommon.Printer, cfg *rpcRouteConfig) {
	if len(cfg.pathParams) == 0 {
		p("          const pathParams: Record<string, string> = {};")
		return
	}

	// Generate path matching to extract params
	// Build a regex from the path pattern
	p("          const pathParams: Record<string, string> = {};")
	p("          const url = new URL(req.url, \"http://localhost\");")
	p("          const pathSegments = url.pathname.split(\"/\");")

	// Calculate segment indices for each path param
	segments := strings.Split(cfg.fullPath, "/")
	for _, param := range cfg.pathParams {
		paramPlaceholder := "{" + param + "}"
		for i, seg := range segments {
			if seg == paramPlaceholder {
				p("          pathParams[\"%s\"] = decodeURIComponent(pathSegments[%d] ?? \"\");", param, i)
				break
			}
		}
	}
	p("")
}

// generateBodyParsing generates code to parse JSON request body. In protobuf-es
// mode the JSON body is decoded through fromJson with ignoreUnknownFields
// (forward-compat, mandatory), yielding a branded message; otherwise it is
// raw-cast to the request type.
func (g *Generator) generateBodyParsing(p tscommon.Printer, method *protogen.Method, tsMethodName string) {
	if g.ctx.MessageRuntime == tscommon.MessageRuntimeES {
		reqSchema := g.ctx.RefMessageSchema(method.Input)
		p("          const body = fromJson(%s, await req.json(), { ignoreUnknownFields: true });", reqSchema)
	} else {
		inputType := g.ctx.RefMessage(method.Input)
		p("          const body = await req.json() as %s;", inputType)
	}

	// Optional validation hook
	p("          if (options?.validateRequest) {")
	p("            const bodyViolations = options.validateRequest(\"%s\", body);", tsMethodName)
	p("            if (bodyViolations) {")
	p("              throw new ValidationError(bodyViolations);")
	p("            }")
	p("          }")
	p("")
}

// generateQueryParamParsing generates code to parse query parameters. In
// protobuf-es mode the request object built from path/query params is wrapped in
// create(<Req>Schema, {...}) so the handler receives a real branded message
// (never a raw cast); otherwise it is a type-annotated object literal.
func (g *Generator) generateQueryParamParsing(
	p tscommon.Printer,
	cfg *rpcRouteConfig,
	method *protogen.Method,
	tsMethodName string,
) {
	es := g.ctx.MessageRuntime == tscommon.MessageRuntimeES
	var inputType, reqSchema string
	if es {
		reqSchema = g.ctx.RefMessageSchema(method.Input)
	} else {
		inputType = g.ctx.RefMessage(method.Input)
	}

	// bodyOpen/bodyClose bracket the request object literal. In es mode the
	// literal is passed to create(...); otherwise it is a typed literal.
	bodyOpen := fmt.Sprintf("          const body: %s = {", inputType)
	bodyClose := "          };"
	if es {
		bodyOpen = fmt.Sprintf("          const body = create(%s, {", reqSchema)
		bodyClose = "          });"
	}

	if len(cfg.queryParams) == 0 {
		switch {
		case len(cfg.pathParamFields) > 0:
			p(bodyOpen)
			for _, ppf := range cfg.pathParamFields {
				g.emitPathParamAssignment(p, ppf, "            ", ",")
			}
			p(bodyClose)
		case es:
			p("          const body = create(%s, {});", reqSchema)
		default:
			p("          const body = {} as %s;", inputType)
		}
		p("")
		return
	}

	if len(cfg.pathParams) > 0 {
		// url already declared in path param extraction
		p("          const params = url.searchParams;")
	} else {
		p("          const url = new URL(req.url, \"http://localhost\");")
		p("          const params = url.searchParams;")
	}
	p(bodyOpen)
	// Include path param fields in the literal so TS sees all required properties
	for _, ppf := range cfg.pathParamFields {
		g.emitPathParamAssignment(p, ppf, "            ", ",")
	}
	for _, qp := range cfg.queryParams {
		g.generateQueryParamField(p, qp)
	}
	p(bodyClose)

	// Optional validation hook
	p("          if (options?.validateRequest) {")
	p("            const bodyViolations = options.validateRequest(\"%s\", body);", tsMethodName)
	p("            if (bodyViolations) {")
	p("              throw new ValidationError(bodyViolations);")
	p("            }")
	p("          }")
	p("")
}

// generateQueryParamField generates a single query parameter field extraction.
func (g *Generator) generateQueryParamField(p tscommon.Printer, qp annotations.QueryParam) {
	jsonName := qp.FieldJSONName
	paramName := qp.ParamName

	// Handle repeated fields: use getAll() for multi-value params
	if qp.Field != nil && qp.Field.Desc.IsList() {
		if qp.Field.Desc.Kind() == protoreflect.EnumKind && qp.Field.Enum != nil {
			p(`            %s: params.getAll("%s") as %s[],`, jsonName, paramName, g.ctx.RefEnum(qp.Field.Enum))
		} else {
			p(`            %s: params.getAll("%s"),`, jsonName, paramName)
		}
		return
	}

	if qp.Field != nil {
		// Check if it's an enum field — cast to enum type with UNSPECIFIED default
		if qp.Field.Desc.Kind() == protoreflect.EnumKind && qp.Field.Enum != nil {
			unspecified := tscommon.TSEnumUnspecifiedValue(qp.Field)
			p(
				`            %s: (params.get("%s") ?? %s) as %s,`,
				jsonName,
				paramName,
				unspecified,
				g.ctx.RefEnum(qp.Field.Enum),
			)
			return
		}

		tsType := tscommon.TSScalarTypeForField(qp.Field)
		switch tsType {
		case tscommon.TSNumber:
			p(`            %s: Number(params.get("%s") ?? "0"),`, jsonName, paramName)
		case tscommon.TSBoolean:
			p(`            %s: params.get("%s") === "true",`, jsonName, paramName)
		default:
			p(`            %s: params.get("%s") ?? "",`, jsonName, paramName)
		}
	} else {
		// Fallback based on field kind string
		switch qp.FieldKind {
		case "int32", "sint32", "sfixed32", "uint32", "fixed32", "float", "double":
			p(`            %s: Number(params.get("%s") ?? "0"),`, jsonName, paramName)
		case "int64", "sint64", "sfixed64", "uint64", "fixed64":
			p(`            %s: params.get("%s") ?? "0",`, jsonName, paramName)
		case "bool":
			p(`            %s: params.get("%s") === "true",`, jsonName, paramName)
		case "enum":
			p(`            %s: params.get("%s") ?? "",`, jsonName, paramName)
		default:
			p(`            %s: params.get("%s") ?? "",`, jsonName, paramName)
		}
	}
}
