package httpgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ValidationError represents a generation-time validation error.
type ValidationError struct {
	Service string
	Method  string
	Message string
}

// ValidateMethodConfig validates HTTP configuration for a method.
// Returns a list of validation errors found.
func ValidateMethodConfig(service *protogen.Service, method *protogen.Method) []ValidationError {
	var errors []ValidationError

	config := getMethodHTTPConfig(method)
	if config == nil {
		return nil
	}

	serviceName := string(service.Desc.Name())
	methodName := string(method.Desc.Name())
	inputMsgName := string(method.Input.Desc.Name())

	// 1. Validate path variables have corresponding fields
	for _, param := range config.PathParams {
		field := findFieldByProtoName(method.Input, param)
		if field == nil {
			errors = append(errors, ValidationError{
				Service: serviceName,
				Method:  methodName,
				Message: fmt.Sprintf(
					"path variable '{%s}' in path '%s' has no matching field in message '%s'. "+
						"Add a field named '%s' to the request message, or fix the path variable name.",
					param, config.Path, inputMsgName, param),
			})
			continue
		}

		// 2. Validate path variable field types (must be scalar)
		if !isPathParamCompatible(field) {
			errors = append(errors, ValidationError{
				Service: serviceName,
				Method:  methodName,
				Message: fmt.Sprintf(
					"path variable '{%s}' is bound to field '%s' of type '%s', but path parameters must be scalar types "+
						"(string, int32, int64, uint32, uint64, bool, float, double). "+
						"Change the field type or remove it from the path.",
					param,
					param,
					field.Desc.Kind(),
				),
			})
		}
	}

	// 3. Validate query parameter fields don't conflict with path params
	queryParams := getQueryParams(method.Input)
	for _, qp := range queryParams {
		for _, pathParam := range config.PathParams {
			if qp.FieldName == pathParam {
				errors = append(errors, ValidationError{
					Service: serviceName,
					Method:  methodName,
					Message: fmt.Sprintf(
						"field '%s' is used both as a path variable in '%s' and as a query parameter. "+
							"A field can only be bound to one parameter type. "+
							"Remove either the path variable or the query annotation.",
						qp.FieldName, config.Path),
				})
			}
		}
	}

	// 4. Error on GET/DELETE with unbound body fields
	httpMethod := config.Method
	if httpMethod == "" {
		httpMethod = "POST"
	}

	if httpMethod == "GET" || httpMethod == "DELETE" {
		bodyFields := getBodyFields(method.Input, config.PathParams, queryParams)
		if len(bodyFields) > 0 {
			fieldNames := make([]string, 0, len(bodyFields))
			for _, f := range bodyFields {
				fieldNames = append(fieldNames, string(f.Desc.Name()))
			}
			errors = append(errors, ValidationError{
				Service: serviceName,
				Method:  methodName,
				Message: fmt.Sprintf(
					"%s request has fields that are not bound to path or query parameters: %v. "+
						"%s requests cannot have a request body. "+
						"Either add [(sebuf.http.query)] annotations to these fields, "+
						"include them in the path as variables, or change the HTTP method to POST/PUT/PATCH.",
					httpMethod, fieldNames, httpMethod),
			})
		}
	}

	return errors
}

// findFieldByProtoName finds a field in a message by its proto name.
func findFieldByProtoName(message *protogen.Message, fieldName string) *protogen.Field {
	for _, field := range message.Fields {
		if string(field.Desc.Name()) == fieldName {
			return field
		}
	}
	return nil
}

// isPathParamCompatible checks if a field type can be used as a path parameter.
func isPathParamCompatible(field *protogen.Field) bool {
	switch field.Desc.Kind() {
	case protoreflect.StringKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind,
		protoreflect.BoolKind,
		protoreflect.FloatKind, protoreflect.DoubleKind:
		return true
	case protoreflect.EnumKind, protoreflect.BytesKind, protoreflect.MessageKind, protoreflect.GroupKind:
		return false
	}
	return false
}

// getBodyFields returns fields that are not bound to path or query parameters.
func getBodyFields(message *protogen.Message, pathParams []string, queryParams []QueryParam) []*protogen.Field {
	pathParamSet := make(map[string]bool)
	for _, p := range pathParams {
		pathParamSet[p] = true
	}

	queryParamSet := make(map[string]bool)
	for _, qp := range queryParams {
		queryParamSet[qp.FieldName] = true
	}

	var bodyFields []*protogen.Field
	for _, field := range message.Fields {
		fieldName := string(field.Desc.Name())
		if !pathParamSet[fieldName] && !queryParamSet[fieldName] {
			bodyFields = append(bodyFields, field)
		}
	}

	return bodyFields
}

// ValidateService validates all methods in a service.
// Returns an error if any validation issues are found, stopping code generation.
func ValidateService(service *protogen.Service) error {
	for _, method := range service.Methods {
		errors := ValidateMethodConfig(service, method)
		if len(errors) > 0 {
			// Return the first error to fail fast
			err := errors[0]
			return fmt.Errorf("%s.%s: %s", err.Service, err.Method, err.Message)
		}
	}
	return nil
}
