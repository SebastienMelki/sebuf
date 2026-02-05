package annotations

import (
	"strings"

	"github.com/SebastienMelki/sebuf/http"
)

// HTTP method string constants (uppercase).
const (
	methodGET    = "GET"
	methodPOST   = "POST"
	methodPUT    = "PUT"
	methodDELETE = "DELETE"
	methodPATCH  = "PATCH"
)

// HTTPMethodToString converts HttpMethod enum to an uppercase string.
// Returns "POST" for unspecified or unknown values (backward compatibility).
func HTTPMethodToString(m http.HttpMethod) string {
	switch m {
	case http.HttpMethod_HTTP_METHOD_GET:
		return methodGET
	case http.HttpMethod_HTTP_METHOD_POST:
		return methodPOST
	case http.HttpMethod_HTTP_METHOD_PUT:
		return methodPUT
	case http.HttpMethod_HTTP_METHOD_DELETE:
		return methodDELETE
	case http.HttpMethod_HTTP_METHOD_PATCH:
		return methodPATCH
	case http.HttpMethod_HTTP_METHOD_UNSPECIFIED:
		// HTTP_METHOD_UNSPECIFIED defaults to POST for backward compatibility
		return methodPOST
	}
	// Any unknown value defaults to POST for backward compatibility
	return methodPOST
}

// HTTPMethodToLower converts HttpMethod enum to a lowercase string.
// Returns "post" for unspecified or unknown values (backward compatibility).
// Used by OpenAPI generator which requires lowercase method names.
func HTTPMethodToLower(m http.HttpMethod) string {
	return strings.ToLower(HTTPMethodToString(m))
}
