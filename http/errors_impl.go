package http

import (
	"fmt"
	"strings"
)

// Error implements the error interface for ValidationError.
// This allows ValidationError to be used with errors.As() and errors.Is().
func (e *ValidationError) Error() string {
	if e == nil {
		return "validation error: <nil>"
	}
	
	if len(e.Violations) == 0 {
		return "validation error: no violations"
	}
	
	if len(e.Violations) == 1 {
		v := e.Violations[0]
		return fmt.Sprintf("validation error: %s: %s", v.Field, v.Description)
	}
	
	// Multiple violations
	var violations []string
	for _, v := range e.Violations {
		violations = append(violations, fmt.Sprintf("%s: %s", v.Field, v.Description))
	}
	
	return fmt.Sprintf("validation error: [%s]", strings.Join(violations, ", "))
}

// Error implements the error interface for Error.
// This allows Error to be used with errors.As() and errors.Is().
func (e *Error) Error() string {
	if e == nil {
		return "error: <nil>"
	}
	
	if e.Message == "" {
		return "error: empty message"
	}
	
	return e.Message
}