package httpgen

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

// For now, let's add a helper to parse paths from the existing authv1 proto format.
func parseExistingAnnotation(_ *protogen.Method) string {
	// This is a temporary parser for the existing sebuf.http.config format
	// that's used in authv1/service.proto

	// In the actual implementation, this would properly parse the extension
	// For now, we'll return empty and use default paths
	return ""
}

// getFieldExamples extracts example values from field options.
func getFieldExamples(field *protogen.Field) []string {
	options := field.Desc.Options()
	if options == nil {
		return nil
	}

	// Get the raw options
	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return nil
	}

	// Extract our custom extension using the generated code
	ext := proto.GetExtension(fieldOptions, http.E_FieldExamples)
	if ext == nil {
		return nil
	}

	fieldExamples, ok := ext.(*http.FieldExamples)
	if !ok || fieldExamples == nil {
		return nil
	}

	return fieldExamples.GetValues()
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
		})
	}

	return params
}

// UnwrapFieldInfo contains information about an unwrap field in a message.
type UnwrapFieldInfo struct {
	Field        *protogen.Field   // The field with unwrap=true
	ElementType  *protogen.Message // The element type of the repeated field (if message type)
	IsRootUnwrap bool              // True if this is a root-level unwrap (single field in message)
	IsMapField   bool              // True if the unwrap field is a map (only for root unwrap)
}

// hasUnwrapAnnotation checks if a field has the unwrap=true annotation.
func hasUnwrapAnnotation(field *protogen.Field) bool {
	options := field.Desc.Options()
	if options == nil {
		return false
	}

	fieldOptions, ok := options.(*descriptorpb.FieldOptions)
	if !ok {
		return false
	}

	ext := proto.GetExtension(fieldOptions, http.E_Unwrap)
	if ext == nil {
		return false
	}

	unwrap, ok := ext.(bool)
	return ok && unwrap
}

// getUnwrapField returns the unwrap field info for a message, or nil if none exists.
// Returns an error if the annotation is invalid (e.g., on non-repeated/non-map field, multiple unwrap fields).
//
// Root-level unwrap: When a message has exactly one field with unwrap=true on a map or repeated field,
// the entire message serializes to just that field's value (object for maps, array for repeated).
// This is detected by checking if the message has exactly one field total.
//
// Map-value unwrap (existing): When a repeated field has unwrap=true and the message is used as a map value,
// the wrapper is collapsed to just the array.
func getUnwrapField(message *protogen.Message) (*UnwrapFieldInfo, error) {
	var unwrapField *protogen.Field

	for _, field := range message.Fields {
		if !hasUnwrapAnnotation(field) {
			continue
		}

		// Validate: must be a repeated field or a map field
		isMap := field.Desc.IsMap()
		isList := field.Desc.IsList()
		if !isList && !isMap {
			return nil, &UnwrapValidationError{
				MessageName: string(message.Desc.Name()),
				FieldName:   string(field.Desc.Name()),
				Reason:      "unwrap annotation can only be used on repeated or map fields",
			}
		}

		// Validate: only one unwrap field per message
		if unwrapField != nil {
			return nil, &UnwrapValidationError{
				MessageName: string(message.Desc.Name()),
				FieldName:   string(field.Desc.Name()),
				Reason:      "only one field per message can have the unwrap annotation",
			}
		}

		unwrapField = field
	}

	if unwrapField == nil {
		return nil, nil //nolint:nilnil // nil,nil is intentional: no unwrap field exists, not an error
	}

	isMapField := unwrapField.Desc.IsMap()

	// Check for root-level unwrap: single field with unwrap annotation
	// Root unwrap is only valid when the message has exactly one field
	isRootUnwrap := len(message.Fields) == 1

	// For root unwrap on maps, validate that we're dealing with a map field
	// For non-root unwrap (map-value unwrap), the field must be a repeated field (not a map)
	if !isRootUnwrap && isMapField {
		return nil, &UnwrapValidationError{
			MessageName: string(message.Desc.Name()),
			FieldName:   string(unwrapField.Desc.Name()),
			Reason:      "map fields with unwrap annotation require the message to have exactly one field (root unwrap)",
		}
	}

	info := &UnwrapFieldInfo{
		Field:        unwrapField,
		IsRootUnwrap: isRootUnwrap,
		IsMapField:   isMapField,
	}

	// If the element type is a message, capture it
	// For repeated fields, this is the element message type
	// For map fields, we don't set ElementType here (handled separately in root unwrap logic)
	if unwrapField.Message != nil && !isMapField {
		info.ElementType = unwrapField.Message
	}

	return info, nil
}

// UnwrapValidationError represents an error in unwrap annotation validation.
type UnwrapValidationError struct {
	MessageName string
	FieldName   string
	Reason      string
}

func (e *UnwrapValidationError) Error() string {
	return "invalid unwrap annotation on " + e.MessageName + "." + e.FieldName + ": " + e.Reason
}
