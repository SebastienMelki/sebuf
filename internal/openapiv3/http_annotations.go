package openapiv3

import (
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	yaml "go.yaml.in/yaml/v4"

	"github.com/SebastienMelki/sebuf/http"
)

// HTTPConfig represents the HTTP configuration for a method.
type HTTPConfig struct {
	Path string
}

// ServiceHTTPConfig represents the HTTP configuration for a service.
type ServiceHTTPConfig struct {
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

	return &HTTPConfig{
		Path: httpConfig.GetPath(),
	}
}

// getServiceHTTPConfig extracts HTTP configuration from service options.
func getServiceHTTPConfig(service *protogen.Service) *ServiceHTTPConfig {
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

	return &ServiceHTTPConfig{
		BasePath: serviceConfig.GetBasePath(),
	}
}

// buildHTTPPath combines service base path with method path.
func buildHTTPPath(servicePath, methodPath string) string {
	// Handle empty paths
	if servicePath == "" && methodPath == "" {
		return "/"
	}
	if servicePath == "" {
		return ensureLeadingSlash(methodPath)
	}
	if methodPath == "" {
		return ensureLeadingSlash(servicePath)
	}

	// Clean and combine paths
	servicePath = strings.TrimSuffix(ensureLeadingSlash(servicePath), "/")
	methodPath = strings.TrimPrefix(methodPath, "/")

	return servicePath + "/" + methodPath
}

// ensureLeadingSlash ensures a path starts with "/".
func ensureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
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

// combineHeaders merges service headers with method headers, with method headers taking precedence.
func combineHeaders(serviceHeaders, methodHeaders []*http.Header) []*http.Header {
	if len(serviceHeaders) == 0 {
		return methodHeaders
	}
	if len(methodHeaders) == 0 {
		return serviceHeaders
	}

	// Create a map to track headers by name for deduplication
	headerMap := make(map[string]*http.Header)

	// Add service headers first
	for _, header := range serviceHeaders {
		if header.GetName() != "" {
			headerMap[header.GetName()] = header
		}
	}

	// Add method headers, overriding service headers with same name
	for _, header := range methodHeaders {
		if header.GetName() != "" {
			headerMap[header.GetName()] = header
		}
	}

	// Convert back to slice, sorted by header name for deterministic output
	result := make([]*http.Header, 0, len(headerMap))

	// Get sorted header names
	headerNames := make([]string, 0, len(headerMap))
	for name := range headerMap {
		headerNames = append(headerNames, name)
	}

	// Sort header names to ensure deterministic order
	for i := 0; i < len(headerNames); i++ {
		for j := i + 1; j < len(headerNames); j++ {
			if headerNames[i] > headerNames[j] {
				headerNames[i], headerNames[j] = headerNames[j], headerNames[i]
			}
		}
	}

	// Add headers in sorted order
	for _, name := range headerNames {
		result = append(result, headerMap[name])
	}

	return result
}

// convertHeadersToParameters converts proto headers to OpenAPI parameters.
func convertHeadersToParameters(headers []*http.Header) []*v3.Parameter {
	if len(headers) == 0 {
		return nil
	}

	parameters := make([]*v3.Parameter, 0, len(headers))

	for _, header := range headers {
		if header.GetName() == "" {
			continue // Skip headers without names
		}

		// Create the schema for the header
		schema := &base.Schema{
			Type: []string{mapHeaderTypeToOpenAPI(header.GetType())},
		}

		// Add format if specified
		if header.GetFormat() != "" {
			schema.Format = header.GetFormat()
		}

		// Add example if specified
		if header.GetExample() != "" {
			schema.Example = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: header.GetExample(),
			}
		}

		// Create the parameter
		parameter := &v3.Parameter{
			Name:        header.GetName(),
			In:          "header",
			Required:    &header.Required,
			Schema:      base.CreateSchemaProxy(schema),
			Description: header.GetDescription(),
		}

		// Set deprecated if specified
		if header.GetDeprecated() {
			parameter.Deprecated = true
		}

		parameters = append(parameters, parameter)
	}

	return parameters
}

const (
	headerTypeString  = "string"
	headerTypeInt32   = "int32"
	headerTypeInt64   = "int64"
	headerTypeInteger = "integer"
)

// mapHeaderTypeToOpenAPI maps proto header types to OpenAPI schema types.
func mapHeaderTypeToOpenAPI(headerType string) string {
	switch strings.ToLower(headerType) {
	case headerTypeString, "":
		return headerTypeString
	case headerTypeInteger, "int", headerTypeInt32, headerTypeInt64:
		return headerTypeInteger
	case "number", "float", "double":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "array":
		return "array"
	default:
		// Default to string for unknown types
		return headerTypeString
	}
}
