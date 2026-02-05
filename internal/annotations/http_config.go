package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// HTTPConfig represents the HTTP configuration for a method.
type HTTPConfig struct {
	Path       string
	Method     string   // "GET", "POST", "PUT", "DELETE", "PATCH"
	PathParams []string // Path variable names extracted from path
}

// ServiceConfig represents the HTTP configuration for a service.
type ServiceConfig struct {
	BasePath string
}

// GetMethodHTTPConfig extracts HTTP configuration from method options.
// Returns nil if no HTTP config annotation is present.
func GetMethodHTTPConfig(method *protogen.Method) *HTTPConfig {
	options := method.Desc.Options()
	if options == nil {
		return nil
	}

	methodOptions, ok := options.(*descriptorpb.MethodOptions)
	if !ok {
		return nil
	}

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
		Method:     HTTPMethodToString(httpConfig.GetMethod()),
		PathParams: ExtractPathParams(path),
	}
}

// GetServiceBasePath extracts the base path from service options.
// Returns an empty string if no service config annotation is present.
func GetServiceBasePath(service *protogen.Service) string {
	options := service.Desc.Options()
	if options == nil {
		return ""
	}

	serviceOptions, ok := options.(*descriptorpb.ServiceOptions)
	if !ok {
		return ""
	}

	ext := proto.GetExtension(serviceOptions, http.E_ServiceConfig)
	if ext == nil {
		return ""
	}

	serviceConfig, ok := ext.(*http.ServiceConfig)
	if !ok || serviceConfig == nil {
		return ""
	}

	return serviceConfig.GetBasePath()
}
