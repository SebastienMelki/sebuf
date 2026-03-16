package krakendgen

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Indent level constants for template output formatting.
const (
	indentEndpoint    = 1
	indentBackendList = 2
	indentBackendBody = 3
	indentExtraCfg    = 4
	indentExtraCfgVal = 5
	indentExtraInner  = 2
)

// GenerateTemplateFile converts a slice of Endpoint objects into a KrakenD
// Flexible Config template fragment (.tmpl). The output is a comma-separated
// list of endpoint objects (no wrapping array) designed to be included in a
// root krakend.tmpl via {{ template "service_endpoints.tmpl" . }}.
//
// Template output differs from JSON output:
//   - Hosts are replaced with {{ .vars.SERVICE_host }} variables
//   - JWT auth uses {{ template "jwt_auth_validator.tmpl" . }} directive
//   - Recaptcha uses {{ include "recpatcha_validator.tmpl" }} directive
//   - QoS configs (rate limit, circuit breaker, cache) are omitted
//   - Backends always include sd:static, disable_host_sanitize, return_error_code
//   - Timeout only appears when explicitly overridden at method level
//   - Headers can use {{ include "partial.tmpl" }} instead of inline arrays
func GenerateTemplateFile(endpoints []Endpoint) string {
	if len(endpoints) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, ep := range endpoints {
		if i > 0 {
			sb.WriteString(",\n")
		}
		writeEndpointTemplate(&sb, ep)
	}
	return sb.String()
}

// TemplateFileName returns the Flexible Config template file name for a
// service: e.g., "UserService" -> "user_service_endpoints.tmpl".
func TemplateFileName(serviceName string) string {
	return serviceNameToSnakeCase(serviceName) + "_endpoints.tmpl"
}

// hostVarName derives the Go template variable reference for a service's host.
// e.g., "UserService" -> "{{ .vars.user_service_host }}".
func hostVarName(serviceName string) string {
	return fmt.Sprintf("{{ .vars.%s_host }}", serviceNameToSnakeCase(serviceName))
}

// serviceNameToSnakeCase converts a PascalCase service name to snake_case,
// handling consecutive uppercase letters (acronyms) correctly.
// "UserService" -> "user_service", "JWTAuthService" -> "jwt_auth_service".
func serviceNameToSnakeCase(name string) string {
	runes := []rune(name)
	var sb strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prevLower := unicode.IsLower(runes[i-1])
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				// Insert underscore when transitioning from lower to upper,
				// or at the end of an acronym (upper followed by lower).
				if prevLower || (unicode.IsUpper(runes[i-1]) && nextLower) {
					sb.WriteByte('_')
				}
			}
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func writeEndpointTemplate(sb *strings.Builder, ep Endpoint) {
	sb.WriteString("{\n")
	writeTmplField(sb, indentEndpoint, "endpoint", ep.Endpoint)
	writeTmplField(sb, indentEndpoint, "method", ep.Method)
	writeTmplField(sb, indentEndpoint, "output_encoding", ep.OutputEncoding)

	if ep.IsMethodTimeout && ep.Timeout != "" {
		writeTmplField(sb, indentEndpoint, "timeout", ep.Timeout)
	}
	if ep.ConcurrentCalls > 0 {
		writeTmplRawField(sb, indentEndpoint, "concurrent_calls", strconv.Itoa(int(ep.ConcurrentCalls)))
	}

	writeHeadersOrPartial(sb, ep)
	writeQueryStrings(sb, ep)
	writeBackendTemplate(sb, ep)
	writeEndpointExtraConfigTemplate(sb, ep)

	sb.WriteString("}")
}

func writeHeadersOrPartial(sb *strings.Builder, ep Endpoint) {
	if ep.HeaderPartial != "" {
		indent(sb, indentEndpoint)
		fmt.Fprintf(sb, "{{ include \"%s\" }},\n", ep.HeaderPartial)
	} else if len(ep.InputHeaders) > 0 {
		writeTmplStringArray(sb, indentEndpoint, "input_headers", ep.InputHeaders)
	}
}

func writeQueryStrings(sb *strings.Builder, ep Endpoint) {
	if len(ep.InputQueryStrings) > 0 {
		writeTmplStringArray(sb, indentEndpoint, "input_query_strings", ep.InputQueryStrings)
	}
}

func writeBackendTemplate(sb *strings.Builder, ep Endpoint) {
	if len(ep.Backend) == 0 {
		return
	}
	b := ep.Backend[0]
	host := hostVarName(ep.ServiceName)

	indent(sb, indentEndpoint)
	sb.WriteString("\"backend\": [\n")
	indent(sb, indentBackendList)
	sb.WriteString("{\n")
	writeTmplField(sb, indentBackendBody, "url_pattern", b.URLPattern)
	writeTmplField(sb, indentBackendBody, "encoding", b.Encoding)
	writeTmplField(sb, indentBackendBody, "sd", "static")
	writeTmplField(sb, indentBackendBody, "method", b.Method)

	// Host uses Go template variable — written as raw value (not JSON-escaped).
	indent(sb, indentBackendBody)
	fmt.Fprintf(sb, "\"host\": [\"%s\"],\n", host)

	writeTmplRawField(sb, indentBackendBody, "disable_host_sanitize", "false")

	// Backend extra_config always includes backend/http return_error_code.
	indent(sb, indentBackendBody)
	sb.WriteString("\"extra_config\": {\n")
	indent(sb, indentExtraCfg)
	sb.WriteString("\"backend/http\": {\n")
	indent(sb, indentExtraCfgVal)
	sb.WriteString("\"return_error_code\": true\n")
	indent(sb, indentExtraCfg)
	sb.WriteString("}\n")
	indent(sb, indentBackendBody)
	sb.WriteString("}\n")

	indent(sb, indentBackendList)
	sb.WriteString("}\n")
	indent(sb, indentEndpoint)
	if ep.HasJWT || ep.HasRecaptcha {
		sb.WriteString("],\n")
	} else {
		sb.WriteString("]\n")
	}
}

func writeEndpointExtraConfigTemplate(sb *strings.Builder, ep Endpoint) {
	if !ep.HasJWT && !ep.HasRecaptcha {
		return
	}

	indent(sb, indentEndpoint)
	sb.WriteString("\"extra_config\": {\n")

	switch {
	case ep.HasRecaptcha && ep.HasJWT:
		indent(sb, indentExtraInner)
		sb.WriteString("{{ include \"recpatcha_validator.tmpl\" }},\n")
		indent(sb, indentExtraInner)
		sb.WriteString("{{ template \"jwt_auth_validator.tmpl\" . }}\n")
	case ep.HasRecaptcha:
		indent(sb, indentExtraInner)
		sb.WriteString("{{ include \"recpatcha_validator.tmpl\" }}\n")
	case ep.HasJWT:
		indent(sb, indentExtraInner)
		sb.WriteString("{{ template \"jwt_auth_validator.tmpl\" . }}\n")
	}

	indent(sb, indentEndpoint)
	sb.WriteString("}\n")
}

// --- Formatting helpers ---

func indent(sb *strings.Builder, level int) {
	for range level {
		sb.WriteString("    ")
	}
}

func writeTmplField(sb *strings.Builder, level int, key, value string) {
	indent(sb, level)
	encoded, _ := json.Marshal(value)
	fmt.Fprintf(sb, "%q: %s,\n", key, encoded)
}

func writeTmplRawField(sb *strings.Builder, level int, key, value string) {
	indent(sb, level)
	fmt.Fprintf(sb, "%q: %s,\n", key, value)
}

func writeTmplStringArray(sb *strings.Builder, level int, key string, values []string) {
	indent(sb, level)
	encoded, _ := json.Marshal(values)
	fmt.Fprintf(sb, "%q: %s,\n", key, encoded)
}
