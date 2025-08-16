package httpgen

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
)

// HTTPConfig represents the HTTP configuration for a method
type HTTPConfig struct {
	Path             string
	ResponseHeaders  map[string]string
	TimeoutSeconds   int32
}

// ServiceConfig represents the HTTP configuration for a service
type ServiceConfig struct {
	BasePath       string
	CommonHeaders  map[string]string
}

// These constants should match the extension numbers in annotations.proto
const (
	httpConfigExtension    = 72295728
	serviceConfigExtension = 72295729
)

// getMethodHTTPConfig extracts HTTP configuration from method options
func getMethodHTTPConfig(method *protogen.Method) *HTTPConfig {
	options := method.Desc.Options()
	if options == nil {
		return nil
	}

	// Get the raw options
	_, ok := options.(*descriptorpb.MethodOptions)
	if !ok {
		return nil
	}

	// Try to extract our custom extension
	// Note: In a real implementation, you'd import the generated Go code
	// from annotations.proto and use it directly. For now, we'll parse manually.
	
	// For simplicity, we'll check if the method has a comment with the path
	// This is a temporary solution until we have the annotations proto compiled
	if len(method.Comments.Leading.String()) > 0 {
		// Parse comments for @http-path directive (temporary)
		// In production, this would use the actual protobuf extensions
	}

	return nil
}

// getServiceHTTPConfig extracts HTTP configuration from service options
func getServiceHTTPConfig(service *protogen.Service) *ServiceConfig {
	options := service.Desc.Options()
	if options == nil {
		return nil
	}

	// Similar to method config, in production this would use the generated extension code
	return nil
}

// For now, let's add a helper to parse paths from the existing authv1 proto format
func parseExistingAnnotation(method *protogen.Method) string {
	// This is a temporary parser for the existing sebuf.http.config format
	// that's used in authv1/service.proto
	
	// In the actual implementation, this would properly parse the extension
	// For now, we'll return empty and use default paths
	return ""
}