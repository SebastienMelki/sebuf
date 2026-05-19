package pyclientgen

import "google.golang.org/protobuf/compiler/protogen"

// writeErrors emits the ApiError base class, ValidationError, FieldViolation,
// and one Exception subclass per proto message ending in "Error" referenced
// by the file's services.
func writeErrors(p printer, collected *collectedTypes) {
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

	for _, msg := range collected.OrderedErrors() {
		writeErrorClass(p, msg)
	}
}

// writeErrorClass emits a proto-message-ending-in-Error as both an ApiError
// subclass (catchable) and a dataclass-shaped container (inspectable). Fields
// are filled in by message.go; this function emits the class header and the
// __init__ that bridges the two interfaces.
func writeErrorClass(p printer, msg *protogen.Message) {
	className := pythonTypeName(msg)
	p("class %s(ApiError):", className)
	p(`    """Generated from proto message %s."""`, msg.Desc.FullName())
	// Placeholder body — replaced once message.go's field renderer is in place.
	p("    pass")
	p("")
	p("")
}
