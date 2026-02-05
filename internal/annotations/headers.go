package annotations

import (
	"sort"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// GetServiceHeaders extracts header configuration from service options.
// Returns nil if no service headers annotation is present.
func GetServiceHeaders(service *protogen.Service) []*http.Header {
	options := service.Desc.Options()
	if options == nil {
		return nil
	}

	serviceOptions, ok := options.(*descriptorpb.ServiceOptions)
	if !ok {
		return nil
	}

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

// GetMethodHeaders extracts header configuration from method options.
// Returns nil if no method headers annotation is present.
func GetMethodHeaders(method *protogen.Method) []*http.Header {
	options := method.Desc.Options()
	if options == nil {
		return nil
	}

	methodOptions, ok := options.(*descriptorpb.MethodOptions)
	if !ok {
		return nil
	}

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

// CombineHeaders merges service headers with method headers, with method headers
// taking precedence. The result is sorted by header name for deterministic output.
// Headers with empty names are skipped.
func CombineHeaders(serviceHeaders, methodHeaders []*http.Header) []*http.Header {
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

	// Get sorted header names for deterministic output
	headerNames := make([]string, 0, len(headerMap))
	for name := range headerMap {
		headerNames = append(headerNames, name)
	}
	sort.Strings(headerNames)

	// Build result in sorted order
	result := make([]*http.Header, 0, len(headerMap))
	for _, name := range headerNames {
		result = append(result, headerMap[name])
	}

	return result
}
