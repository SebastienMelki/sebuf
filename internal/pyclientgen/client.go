package pyclientgen

import (
	"net/http"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// writeServiceClient emits a single client class for a proto service, including
// the typed *ClientOptions and *CallOptions dataclasses, header surface, and
// per-RPC methods. SSE methods (stream=true) raise NotImplementedError.
func writeServiceClient(p printer, service *protogen.Service) {
	serviceName := string(service.Desc.Name())

	writeClientOptionsClass(p, service, serviceName)
	writeCallOptionsClass(p, service, serviceName)
	writeClientClass(p, service, serviceName)
}

func writeClientOptionsClass(p printer, service *protogen.Service, serviceName string) {
	p("@dataclass")
	p("class %sClientOptions:", serviceName)
	p(`    """Construct-time options for %sClient."""`, serviceName)
	p("    transport: Optional[HttpTransport] = None")
	p("    default_headers: Optional[Mapping[str, str]] = None")
	p("    timeout: Optional[float] = None")
	p(`    content_type: str = "application/json"`)

	// Typed kwargs for every service-level header annotation.
	serviceHeaders := annotations.GetServiceHeaders(service)
	for _, header := range serviceHeaders {
		p("    %s: Optional[str] = None", headerOptionName(header.GetName()))
	}
	p("")
	p("")
}

func writeCallOptionsClass(p printer, service *protogen.Service, serviceName string) {
	p("@dataclass")
	p("class %sCallOptions:", serviceName)
	p(`    """Per-call options for %sClient methods."""`, serviceName)
	p("    headers: Optional[Mapping[str, str]] = None")
	p("    timeout: Optional[float] = None")
	p("    content_type: Optional[str] = None")

	// Method-level headers and service-level headers are both available per-call;
	// service headers are also on the client options. Dedup against service set.
	seen := make(map[string]bool)
	for _, header := range annotations.GetServiceHeaders(service) {
		seen[header.GetName()] = true
		p("    %s: Optional[str] = None", headerOptionName(header.GetName()))
	}
	for _, method := range service.Methods {
		for _, header := range annotations.GetMethodHeaders(method) {
			if seen[header.GetName()] {
				continue
			}
			seen[header.GetName()] = true
			p("    %s: Optional[str] = None", headerOptionName(header.GetName()))
		}
	}
	p("")
	p("")
}

func writeClientClass(p printer, service *protogen.Service, serviceName string) {
	p("class %sClient:", serviceName)
	p(`    """Generated client for %s."""`, service.Desc.FullName())

	writeClientConstructor(p, service, serviceName)

	for _, method := range service.Methods {
		writeRPCMethod(p, service, method, serviceName)
	}

	writeErrorHandler(p)
	p("")
}

func writeClientConstructor(p printer, service *protogen.Service, serviceName string) {
	p("    def __init__(")
	p("        self,")
	p("        base_url: str,")
	p("        options: Optional[%sClientOptions] = None,", serviceName)
	p("    ) -> None:")
	p(`        self._base_url = base_url.rstrip("/")`)
	p("        opts = options or %sClientOptions()", serviceName)
	p("        self._transport: HttpTransport = opts.transport or UrllibTransport()")
	p("        self._default_headers: dict[str, str] = dict(opts.default_headers or {})")
	p("        self._timeout = opts.timeout")
	p("        self._content_type = opts.content_type")

	// Apply typed service-header options onto default headers.
	for _, header := range annotations.GetServiceHeaders(service) {
		propName := headerOptionName(header.GetName())
		p("        if opts.%s is not None:", propName)
		p(`            self._default_headers["%s"] = opts.%s`, header.GetName(), propName)
	}

	p("")
}

func writeRPCMethod(p printer, service *protogen.Service, method *protogen.Method, serviceName string) {
	cfg := buildMethodConfig(service, method)

	if cfg.isSSE {
		writeSSEMethodStub(p, method, serviceName, cfg)
		return
	}

	pyMethodName := snakeCase(string(method.Desc.Name()))
	inputType := pythonTypeName(method.Input)
	outputType := resolveOutputType(method)

	p("    def %s(", pyMethodName)
	p("        self,")
	p("        req: %s,", inputType)
	p("        options: Optional[%sCallOptions] = None,", serviceName)
	p("    ) -> %s:", outputType)
	p(`        """Calls %s."""`, method.Desc.FullName())
	p("        opts = options or %sCallOptions()", serviceName)
	p(`        content_type = opts.content_type or self._content_type`)
	p(`        if content_type != "application/json":`)
	p(`            raise NotImplementedError("only application/json is implemented; see docs/python-generation.md")`)

	writePathBuilding(p, cfg)
	writeQueryBuilding(p, cfg)
	writeHeaderBuilding(p, service, method, cfg)
	writeBodyBuilding(p, cfg)
	writeTransportCall(p, cfg)
	writeResponseParsing(p, method)
	p("")
}

func writeSSEMethodStub(p printer, method *protogen.Method, serviceName string, cfg *methodConfig) {
	pyMethodName := snakeCase(string(method.Desc.Name()))
	inputType := pythonTypeName(method.Input)
	outputType := resolveOutputType(method)

	p("    def %s(", pyMethodName)
	p("        self,")
	p("        req: %s,", inputType)
	p("        options: Optional[%sCallOptions] = None,", serviceName)
	p("    ) -> Iterator[%s]:", outputType)
	p(`        """SSE streaming is not yet supported by protoc-gen-py-client."""`)
	p(`        raise NotImplementedError(`)
	p(`            "SSE streaming is not yet supported in py-client. "`)
	p(`            "Track support at https://github.com/SebastienMelki/sebuf/issues (label: py-client)."`)
	p(`        )`)
	_ = cfg
}

func writePathBuilding(p printer, cfg *methodConfig) {
	p(`        path = "%s"`, cfg.fullPath)
	for _, param := range cfg.pathParams {
		pyName := snakeCase(param)
		p(`        path = path.replace("{%s}", urllib.parse.quote(str(req.%s), safe=""))`, param, pyName)
	}
}

func writeQueryBuilding(p printer, cfg *methodConfig) {
	if cfg.httpMethod != http.MethodGet && cfg.httpMethod != http.MethodDelete {
		return
	}
	if len(cfg.queryParams) == 0 {
		return
	}
	p("        query_pairs: list[tuple[str, str]] = []")
	for _, qp := range cfg.queryParams {
		writeQueryParamAppend(p, qp)
	}
	p("        if query_pairs:")
	p(`            path = path + "?" + urllib.parse.urlencode(query_pairs, doseq=True)`)
}

func writeQueryParamAppend(p printer, qp annotations.QueryParam) {
	pyField := escapePyKeyword(string(qp.Field.Desc.Name()))
	src := "req." + pyField
	if qp.Field.Desc.IsList() {
		p("        if %s:", src)
		p(`            for _v in %s:`, src)
		p(`                query_pairs.append(("%s", str(_v)))`, qp.ParamName)
		return
	}
	//nolint:exhaustive // default covers all numeric/message kinds with a single expression
	switch qp.Field.Desc.Kind() {
	case protoreflect.StringKind:
		p("        if %s:", src)
		p(`            query_pairs.append(("%s", str(%s)))`, qp.ParamName, src)
	case protoreflect.BoolKind:
		p("        if %s:", src)
		p(`            query_pairs.append(("%s", "true" if %s else "false"))`, qp.ParamName, src)
	default:
		p("        if %s is not None and %s != 0:", src, src)
		p(`            query_pairs.append(("%s", str(%s)))`, qp.ParamName, src)
	}
}

func writeHeaderBuilding(p printer, service *protogen.Service, method *protogen.Method, cfg *methodConfig) {
	p("        headers: dict[str, str] = dict(self._default_headers)")
	p(`        headers["Content-Type"] = content_type`)
	p(`        headers["Accept"] = "application/json"`)
	p("        if opts.headers:")
	p("            headers.update(opts.headers)")

	for _, header := range annotations.GetServiceHeaders(service) {
		propName := headerOptionName(header.GetName())
		p("        if opts.%s is not None:", propName)
		p(`            headers["%s"] = opts.%s`, header.GetName(), propName)
	}
	for _, header := range annotations.GetMethodHeaders(method) {
		propName := headerOptionName(header.GetName())
		p("        if opts.%s is not None:", propName)
		p(`            headers["%s"] = opts.%s`, header.GetName(), propName)
	}
	_ = cfg
}

func writeBodyBuilding(p printer, cfg *methodConfig) {
	if !cfg.hasBody {
		p("        body: Optional[bytes] = None")
		return
	}
	p(`        body = json.dumps(req.to_dict()).encode("utf-8")`)
}

func writeTransportCall(p printer, cfg *methodConfig) {
	p("        resp = self._transport.request(")
	p(`            method="%s",`, cfg.httpMethod)
	p(`            url=self._base_url + path,`)
	p("            headers=headers,")
	p("            body=body,")
	p(`            timeout=opts.timeout if opts.timeout is not None else self._timeout,`)
	p("        )")
}

func writeResponseParsing(p printer, method *protogen.Method) {
	outputType := resolveOutputType(method)
	p("        if resp.status >= 400:")
	p("            self._raise_for_status(resp)")
	p("        if not resp.body:")
	if outputType == pyNone {
		p("            return None")
	} else {
		p("            return %s()", outputType)
	}
	p(`        return %s.from_dict(json.loads(resp.body))`, outputType)
}

func writeErrorHandler(p printer) {
	p("    def _raise_for_status(self, resp: HttpResponse) -> None:")
	p(`        """Map a non-2xx response to the most specific exception available."""`)
	p(`        body = resp.body or b""`)
	p("        parsed: Any = None")
	p(`        ctype = (resp.headers or {}).get("Content-Type", "")`)
	p(`        looks_jsonish = "json" in ctype.lower() or body[:1] in (b"{", b"[")`)
	p("        if looks_jsonish:")
	p("            try:")
	p(`                parsed = json.loads(body.decode("utf-8"))`)
	p("            except (ValueError, UnicodeDecodeError):")
	p("                parsed = None")
	p(`        if resp.status == 400 and isinstance(parsed, dict) and "violations" in parsed:`)
	p("            violations = [")
	p(`                FieldViolation(field=v.get("field", ""), description=v.get("description", ""))`)
	p(`                for v in parsed.get("violations", [])`)
	p("            ]")
	p("            raise ValidationError(resp.status, body, resp.headers, violations)")
	p("        if isinstance(parsed, dict):")
	p("            for err_cls, required_keys in _ERROR_CLASSES:")
	p("                if required_keys and required_keys.issubset(parsed.keys()):")
	p("                    raise err_cls.populate(resp.status, body, resp.headers, parsed)")
	p("        raise ApiError(resp.status, body, resp.headers)")
}

// resolveOutputType returns the Python class name for a method's response type.
// Root-unwrapped messages keep the wrapper class name (the wrapper still has
// to_dict / from_dict that return the unwrapped shape).
func resolveOutputType(method *protogen.Method) string {
	return pythonTypeName(method.Output)
}

// methodConfig captures every detail of an RPC method needed for generation.
type methodConfig struct {
	methodName  string
	httpMethod  string
	fullPath    string
	pathParams  []string
	queryParams []annotations.QueryParam
	hasBody     bool
	isSSE       bool
}

func buildMethodConfig(service *protogen.Service, method *protogen.Method) *methodConfig {
	methodName := string(method.Desc.Name())
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

	isSSE := httpConfig != nil && httpConfig.Stream
	hasBody := httpMethod == http.MethodPost || httpMethod == http.MethodPut || httpMethod == http.MethodPatch

	return &methodConfig{
		methodName:  methodName,
		httpMethod:  httpMethod,
		fullPath:    fullPath,
		pathParams:  pathParams,
		queryParams: annotations.GetQueryParams(method.Input),
		hasBody:     hasBody,
		isSSE:       isSSE,
	}
}

// snakeCaseExtraCapacity is the expected number of underscores inserted when
// converting CamelCase to snake_case. Pre-sizing the output avoids realloc on
// most identifiers.
const snakeCaseExtraCapacity = 4

// snakeCase converts CamelCase to snake_case. Adapted from PR #132 (@elzalem).
// protogen returns CamelCase by default and Python methods are conventionally
// snake_case.
func snakeCase(s string) string {
	out := make([]byte, 0, len(s)+snakeCaseExtraCapacity)
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 && (s[i-1] < 'A' || s[i-1] > 'Z') {
				out = append(out, '_')
			}
			out = append(out, c+('a'-'A'))
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}

// headerOptionName converts an HTTP header name to a Python keyword argument.
// "X-API-Key" -> "api_key", "X-Request-ID" -> "request_id". The original header
// name is always preserved when writing the request.
func headerOptionName(headerName string) string {
	name := strings.TrimPrefix(headerName, "X-")
	name = strings.TrimPrefix(name, "x-")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ToLower(name)
	return escapePyKeyword(name)
}
