package httpgen

import (
	"github.com/SebastienMelki/sebuf/http"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// HTTPConfig represents the HTTP configuration for a method
type HTTPConfig struct {
	Path string
}

// ServiceConfigImpl represents the HTTP configuration for a service
type ServiceConfigImpl struct {
	BasePath string
}

// These constants should match the extension numbers in annotations.proto
const (
	httpConfigExtension    = 50003
	serviceConfigExtension = 50004
)

// getMethodHTTPConfig extracts HTTP configuration from method options
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
		Path: httpConfig.Path,
	}
}

// getServiceHTTPConfig extracts HTTP configuration from service options
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
		BasePath: serviceConfig.BasePath,
	}
}

// For now, let's add a helper to parse paths from the existing authv1 proto format
func parseExistingAnnotation(method *protogen.Method) string {
	// This is a temporary parser for the existing sebuf.http.config format
	// that's used in authv1/service.proto

	// In the actual implementation, this would properly parse the extension
	// For now, we'll return empty and use default paths
	return ""
}
