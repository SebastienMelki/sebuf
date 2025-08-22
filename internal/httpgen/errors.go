package httpgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// HTTPError is the interface that all generated errors must implement
type HTTPErrorInterface interface {
	error
	HTTPStatus() int
	ErrorCode() string
}

// ErrorConfig represents error configuration for a method
type MethodErrorConfig struct {
	CustomError      bool
	DefaultStatusCode int32
}


// getMethodErrorConfig extracts error configuration from method options
func getMethodErrorConfig(method *protogen.Method) *MethodErrorConfig {
	options := method.Desc.Options()
	if options == nil {
		return nil
	}

	methodOptions, ok := options.(*descriptorpb.MethodOptions)
	if !ok {
		return nil
	}

	ext := proto.GetExtension(methodOptions, http.E_ErrorConfig)
	if ext == nil {
		return nil
	}

	errorConfig, ok := ext.(*http.ErrorConfig)
	if !ok || errorConfig == nil {
		return nil
	}

	return &MethodErrorConfig{
		CustomError:       errorConfig.GetCustomError(),
		DefaultStatusCode: errorConfig.GetDefaultStatusCode(),
	}
}


// shouldUseCustomError determines if a method should use custom error types
func shouldUseCustomError(method *protogen.Method) bool {
	// Check method-level config
	methodConfig := getMethodErrorConfig(method)
	if methodConfig != nil {
		return methodConfig.CustomError
	}

	return false
}

// findCustomErrorMessage looks for a message named ${RpcName}Error
func findCustomErrorMessage(method *protogen.Method, file *protogen.File) *protogen.Message {
	errorMessageName := method.GoName + "Error"
	
	// Look in the same file
	for _, msg := range file.Messages {
		if msg.GoIdent.GoName == errorMessageName {
			return msg
		}
	}
	
	// TODO: Look in imported files if needed
	
	return nil
}

// generateErrorHandlingCode generates the error handling functions
func (g *Generator) generateErrorHandlingCode(gf *protogen.GeneratedFile) {
	// HTTPError interface
	gf.P("// HTTPError is the interface for errors with HTTP status codes")
	gf.P("type HTTPError interface {")
	gf.P("error")
	gf.P("HTTPStatus() int")
	gf.P("ErrorCode() string")
	gf.P("}")
	gf.P()

	// ValidationErrorWrapper
	gf.P("// ValidationErrorWrapper wraps a ValidationError and implements HTTPError")
	gf.P("type ValidationErrorWrapper struct {")
	gf.P("*sebufhttp.ValidationError")
	gf.P("}")
	gf.P()
	gf.P("func (e *ValidationErrorWrapper) Error() string {")
	gf.P("if e.ValidationError == nil {")
	gf.P(`return "validation error"`)
	gf.P("}")
	gf.P("return e.ValidationError.GetMessage()")
	gf.P("}")
	gf.P()
	gf.P("func (e *ValidationErrorWrapper) HTTPStatus() int {")
	gf.P("return 400")
	gf.P("}")
	gf.P()
	gf.P("func (e *ValidationErrorWrapper) ErrorCode() string {")
	gf.P(`return "VALIDATION_ERROR"`)
	gf.P("}")
	gf.P()

	// StandardErrorWrapper
	gf.P("// StandardErrorWrapper wraps a StandardError and implements HTTPError")
	gf.P("type StandardErrorWrapper struct {")
	gf.P("*sebufhttp.StandardError")
	gf.P("}")
	gf.P()
	gf.P("func (e *StandardErrorWrapper) Error() string {")
	gf.P("if e.StandardError == nil {")
	gf.P(`return "internal server error"`)
	gf.P("}")
	gf.P("return e.StandardError.GetMessage()")
	gf.P("}")
	gf.P()
	gf.P("func (e *StandardErrorWrapper) HTTPStatus() int {")
	gf.P("if e.StandardError == nil || e.StandardError.Code == sebufhttp.HTTPStatusCode_HTTP_STATUS_CODE_UNSPECIFIED {")
	gf.P("return 500")
	gf.P("}")
	gf.P("return int(e.StandardError.Code)")
	gf.P("}")
	gf.P()
	gf.P("func (e *StandardErrorWrapper) ErrorCode() string {")
	gf.P("if e.StandardError == nil {")
	gf.P(`return "INTERNAL_ERROR"`)
	gf.P("}")
	gf.P("return e.StandardError.Code.String()")
	gf.P("}")
	gf.P()

	// marshalError function
	gf.P("// marshalError marshals an error to bytes based on content type")
	gf.P("func marshalError(r *http.Request, err error) ([]byte, int) {")
	gf.P("var statusCode int")
	gf.P("var errorMessage proto.Message")
	gf.P()
	gf.P("// Check if it's an HTTPError")
	gf.P("if httpErr, ok := err.(HTTPError); ok {")
	gf.P("statusCode = httpErr.HTTPStatus()")
	gf.P()
	gf.P("// Check for specific error types")
	gf.P("switch e := err.(type) {")
	gf.P("case *ValidationErrorWrapper:")
	gf.P("errorMessage = e.ValidationError")
	gf.P("case *StandardErrorWrapper:")
	gf.P("errorMessage = e.StandardError")
	gf.P("default:")
	gf.P("// For custom errors that implement HTTPError")
	gf.P("// Try to extract the underlying proto message")
	gf.P("if msg, ok := err.(proto.Message); ok {")
	gf.P("errorMessage = msg")
	gf.P("} else {")
	gf.P("// Fallback to StandardError")
	gf.P("errorMessage = &sebufhttp.StandardError{")
	gf.P("Code: sebufhttp.HTTPStatusCode(statusCode),")
	gf.P("Message: err.Error(),")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P("} else {")
	gf.P("// Default to 500 Internal Server Error")
	gf.P("statusCode = 500")
	gf.P("errorMessage = &sebufhttp.StandardError{")
	gf.P("Code: sebufhttp.HTTPStatusCode_INTERNAL_SERVER_ERROR,")
	gf.P("Message: err.Error(),")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("// Marshal based on content type")
	gf.P(`contentType := r.Header.Get("Content-Type")`)
	gf.P(`if contentType == "" {`)
	gf.P(`contentType = r.Header.Get("Accept")`)
	gf.P(`}`)
	gf.P(`if contentType == "" {`)
	gf.P("contentType = JSONContentType")
	gf.P("}")
	gf.P()
	gf.P("var result []byte")
	gf.P("var marshalErr error")
	gf.P()
	gf.P("switch filterFlags(contentType) {")
	gf.P("case JSONContentType:")
	gf.P("result, marshalErr = protojson.Marshal(errorMessage)")
	gf.P("case BinaryContentType, ProtoContentType:")
	gf.P("result, marshalErr = proto.Marshal(errorMessage)")
	gf.P("default:")
	gf.P("// Default to JSON")
	gf.P("result, marshalErr = protojson.Marshal(errorMessage)")
	gf.P("}")
	gf.P()
	gf.P("if marshalErr != nil {")
	gf.P("// If we can't marshal the error, return a simple string")
	gf.P(`return []byte(fmt.Sprintf(`+"`"+`{"error":"%s"}`+"`"+`, err.Error())), statusCode`)
	gf.P("}")
	gf.P()
	gf.P("return result, statusCode")
	gf.P("}")
	gf.P()

	// convertValidationError function
	gf.P("// convertValidationError converts protovalidate errors to ValidationError")
	gf.P("func convertValidationError(err error) *ValidationErrorWrapper {")
	gf.P("// Extract violations from protovalidate error")
	gf.P("var violations []*sebufhttp.ValidationError_Violation")
	gf.P()
	gf.P("// Check if it's a protovalidate ValidationError")
	gf.P("var validationError *protovalidate.ValidationError")
	gf.P("if errors.As(err, &validationError) {")
	gf.P("for _, violation := range validationError.Violations {")
	gf.P("if violation.Proto != nil {")
	gf.P("fieldPath := \"\"")
	gf.P("if field := violation.Proto.GetField(); field != nil {")
	gf.P("// Extract field name from FieldPath")
	gf.P("if len(field.Elements) > 0 {")
	gf.P("fieldPath = field.Elements[len(field.Elements)-1].GetFieldName()")
	gf.P("}")
	gf.P("}")
	gf.P("violations = append(violations, &sebufhttp.ValidationError_Violation{")
	gf.P("ViolationType: &sebufhttp.ValidationError_Violation_Body{")
	gf.P("Body: &sebufhttp.ValidationError_BodyViolation{")
	gf.P("Field: fieldPath,")
	gf.P("Description: violation.Proto.GetMessage(),")
	gf.P("Constraint: violation.Proto.GetRuleId(),")
	gf.P("},")
	gf.P("},")
	gf.P("})")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("// Create a clean summary message")
	gf.P("message := \"Request validation failed\"")
	gf.P("if len(violations) == 1 {")
	gf.P("message = \"Validation failed for field: \" + violations[0].GetBody().GetField()")
	gf.P("} else if len(violations) > 1 {")
	gf.P("message = fmt.Sprintf(\"Validation failed for %d fields\", len(violations))")
	gf.P("}")
	gf.P()
	gf.P("validationErr := &sebufhttp.ValidationError{")
	gf.P("Message: message,")
	gf.P("Violations: violations,")
	gf.P("}")
	gf.P()
	gf.P("return &ValidationErrorWrapper{ValidationError: validationErr}")
	gf.P("}")
	gf.P()
}

// generateCustomErrorImpl generates the error interface implementation for custom error types
func (g *Generator) generateCustomErrorImpl(gf *protogen.GeneratedFile, message *protogen.Message, method *protogen.Method) {
	errorType := message.GoIdent.GoName
	
	// Implement error interface
	gf.P("// Error implements the error interface for ", errorType)
	gf.P("func (e *", errorType, ") Error() string {")
	gf.P(`return fmt.Sprintf("`, strings.ToLower(method.GoName), ` error: %v", e)`)
	gf.P("}")
	gf.P()
	
	// Implement HTTPError interface
	gf.P("// HTTPStatus returns the HTTP status code for ", errorType)
	gf.P("func (e *", errorType, ") HTTPStatus() int {")
	
	// Check if the error has a code field
	hasCodeField := false
	for _, field := range message.Fields {
		if field.GoName == "Code" {
			hasCodeField = true
			break
		}
	}
	
	if hasCodeField {
		gf.P("if e.Code != sebufhttp.HTTPStatusCode_HTTP_STATUS_CODE_UNSPECIFIED {")
		gf.P("return int(e.Code)")
		gf.P("}")
	}
	
	// Default status code
	gf.P("return 500 // Default to Internal Server Error")
	gf.P("}")
	gf.P()
	
	gf.P("// ErrorCode returns the error code for ", errorType)
	gf.P("func (e *", errorType, ") ErrorCode() string {")
	gf.P(`return "`, strings.ToUpper(camelToSnake(method.GoName)), `_ERROR"`)
	gf.P("}")
	gf.P()
}

// getErrorReturnType determines what error type a method should return
func getErrorReturnType(method *protogen.Method, file *protogen.File) string {
	if shouldUseCustomError(method) {
		if customError := findCustomErrorMessage(method, file); customError != nil {
			return "*" + customError.GoIdent.GoName
		}
	}
	return "error"
}