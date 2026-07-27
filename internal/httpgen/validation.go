package httpgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
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

	config := annotations.GetMethodHTTPConfig(method)
	if config == nil {
		return nil
	}

	serviceName := string(service.Desc.Name())
	methodName := string(method.Desc.Name())
	inputMsgName := string(method.Input.Desc.Name())

	// 1 & 2. Validate path variables resolve to a field of a URL-representable type
	errors = append(errors, validatePathVariables(method, config, serviceName, methodName)...)

	// 3. Validate query parameter types, and that they don't collide with path params
	queryParams := annotations.GetQueryParams(method.Input)
	errors = append(errors, validateQueryParamFields(queryParams, config, serviceName, methodName, inputMsgName)...)

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

// getBodyFields returns fields that are not bound to path or query parameters.
func getBodyFields(
	message *protogen.Message,
	pathParams []string,
	queryParams []annotations.QueryParam,
) []*protogen.Field {
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

// validatePathVariables checks that every path variable resolves to a field whose type
// can be represented in a URL path segment.
func validatePathVariables(
	method *protogen.Method,
	config *annotations.HTTPConfig,
	serviceName, methodName string,
) []ValidationError {
	var errors []ValidationError

	inputMsgName := string(method.Input.Desc.Name())

	for _, param := range config.PathParams {
		field := annotations.FindFieldByProtoName(method.Input, param)
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

		urlErr := annotations.ValidatePathParamField(field, param, inputMsgName)
		if urlErr == nil {
			continue
		}

		errors = append(errors, ValidationError{
			Service: serviceName,
			Method:  methodName,
			Message: urlErr.Error(),
		})
	}

	return errors
}

// validateQueryParamFields checks that every query parameter has a URL-representable
// type and is not also bound as a path variable.
func validateQueryParamFields(
	queryParams []annotations.QueryParam,
	config *annotations.HTTPConfig,
	serviceName, methodName, inputMsgName string,
) []ValidationError {
	var errors []ValidationError

	for _, qp := range queryParams {
		if !annotations.IsURLParamCompatible(qp.Field) {
			urlErr := &annotations.URLParamValidationError{
				MessageName: inputMsgName,
				FieldName:   qp.FieldName,
				ParamName:   qp.ParamName,
				Location:    annotations.URLParamLocationQuery,
				TypeName:    annotations.URLParamTypeName(qp.Field),
			}
			errors = append(errors, ValidationError{
				Service: serviceName,
				Method:  methodName,
				Message: urlErr.Error(),
			})
		}

		for _, pathParam := range config.PathParams {
			if qp.FieldName != pathParam {
				continue
			}
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

	return errors
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
