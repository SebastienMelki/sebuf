package openapiv3

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

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

// buildHTTPPath combines service base path with method path
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

// ensureLeadingSlash ensures a path starts with "/"
func ensureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}