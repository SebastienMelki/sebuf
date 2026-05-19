package pyclientgen

import (
	"google.golang.org/protobuf/compiler/protogen"
)

// writeServiceClient emits a single client class for a proto service, including
// the typed *ClientOptions and *CallOptions dataclasses.
//
// This scaffold emits just the class headers; method bodies, header handling,
// path/query encoding, response parsing, and SSE detection land in the next commit.
func writeServiceClient(p printer, service *protogen.Service) {
	serviceName := string(service.Desc.Name())

	p("@dataclass")
	p("class %sClientOptions:", serviceName)
	p("    transport: Optional[HttpTransport] = None")
	p("    default_headers: Optional[Mapping[str, str]] = None")
	p("    timeout: Optional[float] = None")
	p(`    content_type: str = "application/json"`)
	p("")
	p("")
	p("@dataclass")
	p("class %sCallOptions:", serviceName)
	p("    headers: Optional[Mapping[str, str]] = None")
	p("    timeout: Optional[float] = None")
	p("    content_type: Optional[str] = None")
	p("")
	p("")
	p("class %sClient:", serviceName)
	p(`    """Generated client for %s."""`, service.Desc.FullName())
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
	p("")
	for _, method := range service.Methods {
		p("    def %s(self, req: Any, options: Optional[%sCallOptions] = None) -> Any:",
			snakeCase(string(method.Desc.Name())), serviceName)
		p("        raise NotImplementedError  # filled in by next commit")
		p("")
	}
	p("")
}

// snakeCase converts CamelCase to snake_case. Lifted from PR #132 (@elzalem)
// because the proto generator package uses CamelCase by default and Python
// methods are conventionally snake_case.
func snakeCase(s string) string {
	out := make([]byte, 0, len(s)+4)
	for i := 0; i < len(s); i++ {
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
