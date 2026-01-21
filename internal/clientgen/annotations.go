package clientgen

import (
	"regexp"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// HTTP method constants.
const (
	httpMethodGET    = "GET"
	httpMethodPOST   = "POST"
	httpMethodPUT    = "PUT"
	httpMethodDELETE = "DELETE"
	httpMethodPATCH  = "PATCH"
)

// pathParamRegex matches path variables like {user_id} or {id}.
var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// HTTPConfig represents the HTTP configuration for a method.
type HTTPConfig struct {
	Path       string
	Method     string   // "GET", "POST", "PUT", "DELETE", "PATCH"
	PathParams []string // Path variable names extracted from path
}

// QueryParam represents a query parameter configuration extracted from a field.
type QueryParam struct {
	FieldName   string // Proto field name
	FieldGoName string // Go field name
	ParamName   string // Query parameter name
	Required    bool
	FieldKind   string // Proto field kind (string, int32, bool, etc.)
}

// ServiceConfigImpl represents the HTTP configuration for a service.
type ServiceConfigImpl struct {
	BasePath string
}

// getMethodHTTPConfig extracts HTTP configuration from method options.
func getMethodHTTPConfig(method *protogen.Method) *HTTPConfig {
	options := method.Desc.Options()
	if options == nil {
		return nil
	}

	// Get the raw options
	methodOptions, ok := options.(*descriptorpb.MethodOptions)
	if !ok {
		return nil
	}

	// Extract our custom extension using the generated code
	ext := proto.GetExtension(methodOptions, http.E_Config)
	if ext == nil {
		return nil
	}

	httpConfig, ok := ext.(*http.HttpConfig)
	if !ok || httpConfig == nil {
		return nil
	}

	path := httpConfig.GetPath()

	return &HTTPConfig{
		Path:       path,
		Method:     httpMethodToString(httpConfig.GetMethod()),
		PathParams: extractPathParams(path),
	}
}

// httpMethodToString converts HttpMethod enum to string. Returns "POST" for unspecified (backward compatibility).
func httpMethodToString(m http.HttpMethod) string {
	switch m {
	case http.HttpMethod_HTTP_METHOD_GET:
		return httpMethodGET
	case http.HttpMethod_HTTP_METHOD_POST:
		return httpMethodPOST
	case http.HttpMethod_HTTP_METHOD_PUT:
		return httpMethodPUT
	case http.HttpMethod_HTTP_METHOD_DELETE:
		return httpMethodDELETE
	case http.HttpMethod_HTTP_METHOD_PATCH:
		return httpMethodPATCH
	case http.HttpMethod_HTTP_METHOD_UNSPECIFIED:
		// HTTP_METHOD_UNSPECIFIED defaults to POST for backward compatibility
		return httpMethodPOST
	}
	// Any unknown value defaults to POST for backward compatibility
	return httpMethodPOST
}

// extractPathParams parses path variables from a path string.
// Example: "/users/{user_id}/posts/{post_id}" -> ["user_id", "post_id"].
func extractPathParams(path string) []string {
	matches := pathParamRegex.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}

	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

// getServiceHTTPConfig extracts HTTP configuration from service options.
func getServiceHTTPConfig(service *protogen.Service) *ServiceConfigImpl {
	options := service.Desc.Options()
	if options == nil {
		return nil
	}

	// Get the raw options
	serviceOptions, ok := options.(*descriptorpb.ServiceOptions)
	if !ok {
		return nil
	}

	// Extract our custom extension using the generated code
	ext := proto.GetExtension(serviceOptions, http.E_ServiceConfig)
	if ext == nil {
		return nil
	}

	serviceConfig, ok := ext.(*http.ServiceConfig)
	if !ok || serviceConfig == nil {
		return nil
	}

	return &ServiceConfigImpl{
		BasePath: serviceConfig.GetBasePath(),
	}
}

// getServiceHeaders extracts header configuration from service options.
func getServiceHeaders(service *protogen.Service) []*http.Header {
	options := service.Desc.Options()
	if options == nil {
		return nil
	}

	// Get the raw options
	serviceOptions, ok := options.(*descriptorpb.ServiceOptions)
	if !ok {
		return nil
	}

	// Extract our custom extension using the generated code
	ext := proto.GetExtension(serviceOptions, http.E_ServiceHeaders)
	if ext == nil {
		return nil
	}

	serviceHeaders, ok := ext.(*http.ServiceHeaders)
	if !ok || serviceHeaders == nil {
		return nil
	}

	return serviceHeaders.GetRequiredHeaders()
}

// getMethodHeaders extracts header configuration from method options.
func getMethodHeaders(method *protogen.Method) []*http.Header {
	options := method.Desc.Options()
	if options == nil {
		return nil
	}

	// Get the raw options
	methodOptions, ok := options.(*descriptorpb.MethodOptions)
	if !ok {
		return nil
	}

	// Extract our custom extension using the generated code
	ext := proto.GetExtension(methodOptions, http.E_MethodHeaders)
	if ext == nil {
		return nil
	}

	methodHeaders, ok := ext.(*http.MethodHeaders)
	if !ok || methodHeaders == nil {
		return nil
	}

	return methodHeaders.GetRequiredHeaders()
}

// getQueryParams extracts query parameter configurations from message fields.
func getQueryParams(message *protogen.Message) []QueryParam {
	var params []QueryParam

	for _, field := range message.Fields {
		options := field.Desc.Options()
		if options == nil {
			continue
		}

		fieldOptions, ok := options.(*descriptorpb.FieldOptions)
		if !ok {
			continue
		}

		ext := proto.GetExtension(fieldOptions, http.E_Query)
		if ext == nil {
			continue
		}

		queryConfig, ok := ext.(*http.QueryConfig)
		if !ok || queryConfig == nil {
			continue
		}

		// Use the configured name, or default to the proto field name
		paramName := queryConfig.GetName()
		if paramName == "" {
			paramName = string(field.Desc.Name())
		}

		params = append(params, QueryParam{
			FieldName:   string(field.Desc.Name()),
			FieldGoName: field.GoName,
			ParamName:   paramName,
			Required:    queryConfig.GetRequired(),
			FieldKind:   field.Desc.Kind().String(),
		})
	}

	return params
}
