package pyclientgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// writeErrors emits the error hierarchy used by all generated clients:
//
//   - FieldViolation, ApiError, ValidationError (always present)
//   - One Exception subclass per proto message ending in "Error" (renders fields,
//     to_dict / from_dict, plus an ApiError __init__ bridge)
//   - _ERROR_CLASSES registry consumed by the client's _raise_for_status to map
//     non-2xx bodies to the most specific exception by required field shape.
func writeErrors(p printer, collected *collectedTypes) {
	writeErrorFoundation(p)
	for _, msg := range collected.OrderedErrors() {
		writeErrorClass(p, msg)
	}
	writeErrorRegistry(p, collected)
}

func writeErrorFoundation(p printer) {
	p("@dataclass")
	p("class FieldViolation:")
	p(`    """Single validation violation, matching sebuf.http.FieldViolation."""`)
	p("    field: str")
	p(`    description: str = ""`)
	p("")
	p("")
	p("class ApiError(Exception):")
	p(`    """Base exception for any non-2xx HTTP response."""`)
	p("    def __init__(")
	p("        self,")
	p("        status: int,")
	p("        body: bytes,")
	p("        headers: Optional[Mapping[str, str]] = None,")
	p("    ) -> None:")
	p("        self.status = status")
	p("        self.body = body")
	p("        self.headers = headers or {}")
	p(`        super().__init__(f"HTTP {status}")`)
	p("")
	p("")
	p("class ValidationError(ApiError):")
	p(`    """Raised on HTTP 400 when the server returns sebuf.http.ValidationError JSON."""`)
	p("    def __init__(")
	p("        self,")
	p("        status: int,")
	p("        body: bytes,")
	p("        headers: Optional[Mapping[str, str]] = None,")
	p("        violations: Optional[Sequence[FieldViolation]] = None,")
	p("    ) -> None:")
	p("        super().__init__(status, body, headers)")
	p("        self.violations: list[FieldViolation] = list(violations or [])")
	p("")
	p("")
}

// writeErrorClass emits a proto-message-ending-in-Error as an ApiError subclass
// with the same dataclass-like field surface as a regular message. It provides:
//
//   - per-field attributes with sensible defaults
//   - a populate(data) classmethod that constructs a partially-populated instance
//     from a parsed JSON dict
//   - a to_dict() returning the JSON wire form (for re-serialization)
func writeErrorClass(p printer, msg *protogen.Message) {
	className := pythonTypeName(msg)
	fields := visibleFields(msg)

	p("class %s(ApiError):", className)
	p(`    """Generated from proto message %s."""`, msg.Desc.FullName())
	p("    def __init__(")
	p("        self,")
	p("        status: int = 0,")
	p("        body: bytes = b\"\",")
	p("        headers: Optional[Mapping[str, str]] = None,")
	if len(fields) == 0 {
		p("    ) -> None:")
		p("        super().__init__(status, body, headers)")
		p("")
		p("    def to_dict(self) -> dict[str, Any]:")
		p("        return {}")
		p("")
		p("    @classmethod")
		p(
			`    def populate(cls, status: int, body: bytes, `+
				`headers: Optional[Mapping[str, str]], `+
				`data: Mapping[str, Any]) -> "%s":`,
			className,
		)
		p("        return cls(status=status, body=body, headers=headers)")
		p("")
		p("")
		return
	}

	for _, f := range fields {
		name := pythonFieldName(f)
		pyType := pythonFieldType(f)
		def := pythonFieldDefault(f)
		p("        %s: %s = %s,", name, pyType, def)
	}
	p("    ) -> None:")
	p("        super().__init__(status, body, headers)")
	for _, f := range fields {
		name := pythonFieldName(f)
		p("        self.%s = %s", name, name)
	}
	p("")

	writeErrorToDict(p, msg, fields)
	writeErrorPopulate(p, msg, className, fields)
	p("")
}

func writeErrorToDict(p printer, _ *protogen.Message, fields []*protogen.Field) {
	p("    def to_dict(self) -> dict[str, Any]:")
	p(`        """Serialize to the JSON wire form."""`)
	p("        d: dict[str, Any] = {}")
	for _, f := range fields {
		name := pythonFieldName(f)
		src := "self." + name
		writeFieldToDict(p, f) // reuses message.go logic; field names align ("self.<name>")
		_ = src
	}
	p("        return d")
	p("")
}

func writeErrorPopulate(p printer, _ *protogen.Message, className string, fields []*protogen.Field) {
	p("    @classmethod")
	p(
		`    def populate(cls, status: int, body: bytes, ` +
			`headers: Optional[Mapping[str, str]], ` +
			`data: Mapping[str, Any]) -> "` + className + `":`,
	)
	p(`        """Build an instance from a parsed JSON dict — only used by the client error path."""`)
	p("        kwargs: dict[str, Any] = {}")
	for _, f := range fields {
		writeFieldFromDict(p, f)
	}
	p("        return cls(status=status, body=body, headers=headers, **kwargs)")
	p("")
}

// writeErrorRegistry emits the lookup table consumed by the generated
// _raise_for_status. The registry stores each error class with the JSON field
// names that uniquely identify it. The client picks the first registry entry
// whose required keys are all present in the response body.
func writeErrorRegistry(p printer, collected *collectedTypes) {
	p("_ERROR_CLASSES: list[tuple[type[ApiError], set[str]]] = [")
	for _, msg := range collected.OrderedErrors() {
		className := pythonTypeName(msg)
		keys := errorMarkerKeys(msg)
		p("    (%s, %s),", className, formatPyStringSet(keys))
	}
	p("]")
	p("")
	p("")
}

// errorMarkerKeys returns the JSON field names the client should look for to
// match a response to this error class. We use every declared field as a key;
// a more sophisticated implementation could weight rarer fields higher.
func errorMarkerKeys(msg *protogen.Message) []string {
	out := make([]string, 0, len(msg.Fields))
	for _, f := range msg.Fields {
		out = append(out, jsonFieldName(f))
	}
	return out
}

// formatPyStringSet renders the registry key set as a valid Python literal.
// Empty sets must be written as `set()` because `{}` is an empty dict literal,
// which would violate the registry's `set[str]` type annotation.
func formatPyStringSet(keys []string) string {
	if len(keys) == 0 {
		return "set()"
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = `"` + k + `"`
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
