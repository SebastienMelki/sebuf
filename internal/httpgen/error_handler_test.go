package httpgen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// generatedFiles holds the content of generated files for testing.
type generatedFiles struct {
	config  string
	binding string
	http    string
}

// generateTestFiles generates code using protoc and returns the file contents.
func generateTestFiles(t *testing.T, protoFile string) *generatedFiles {
	t.Helper()

	// Skip if protoc is not available
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not found, skipping error handler tests")
	}

	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(baseDir, "..", "..")
	protoDir := filepath.Join(baseDir, "testdata", "proto")
	tempDir := t.TempDir()

	// Generate code
	cmd := exec.Command("protoc",
		"--go_out="+tempDir,
		"--go_opt=paths=source_relative",
		"--go-http_out="+tempDir,
		"--go-http_opt=paths=source_relative",
		"--proto_path="+protoDir,
		"--proto_path="+filepath.Join(projectRoot, "proto"),
		protoFile,
	)
	cmd.Dir = protoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		t.Fatalf("protoc failed: %v\nstderr: %s", runErr, stderr.String())
	}

	baseName := strings.TrimSuffix(protoFile, ".proto")

	result := &generatedFiles{}

	// Read config file
	configPath := filepath.Join(tempDir, baseName+"_http_config.pb.go")
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read generated config file: %v", err)
	}
	result.config = string(configContent)

	// Read binding file
	bindingPath := filepath.Join(tempDir, baseName+"_http_binding.pb.go")
	bindingContent, err := os.ReadFile(bindingPath)
	if err != nil {
		t.Fatalf("Failed to read generated binding file: %v", err)
	}
	result.binding = string(bindingContent)

	// Read HTTP file
	httpPath := filepath.Join(tempDir, baseName+"_http.pb.go")
	httpContent, err := os.ReadFile(httpPath)
	if err != nil {
		t.Fatalf("Failed to read generated HTTP file: %v", err)
	}
	result.http = string(httpContent)

	return result
}

// TestErrorHandlerConfigGeneration tests that config file types are generated correctly.
func TestErrorHandlerConfigGeneration(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	t.Run("ErrorHandler type is generated", func(t *testing.T) {
		if !strings.Contains(
			files.config,
			"type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error) proto.Message",
		) {
			t.Error("ErrorHandler type definition not found")
		}
	})

	t.Run("serverConfiguration has errorHandler field", func(t *testing.T) {
		if !strings.Contains(files.config, "errorHandler ErrorHandler") {
			t.Error("errorHandler field not found in serverConfiguration")
		}
	})

	t.Run("WithErrorHandler function is generated", func(t *testing.T) {
		if !strings.Contains(files.config, "func WithErrorHandler(handler ErrorHandler) ServerOption") {
			t.Error("WithErrorHandler function not found")
		}
		if !strings.Contains(files.config, "c.errorHandler = handler") {
			t.Error("WithErrorHandler implementation not correct")
		}
	})

	t.Run("proto import is included", func(t *testing.T) {
		if !strings.Contains(files.config, `"google.golang.org/protobuf/proto"`) {
			t.Error("proto import not found")
		}
	})
}

// TestErrorHandlerDocumentation tests that documentation is generated correctly.
func TestErrorHandlerDocumentation(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	docStrings := []string{
		"ErrorHandler is called when an error occurs",
		"Set headers via w.Header().Set(...)",
		"Set status code via w.WriteHeader(...)",
		"Return a proto.Message to be marshaled",
		"Return nil to use the default error response",
		"If you write directly to w (via w.Write())",
		"errors.As() to inspect error types",
	}
	for _, doc := range docStrings {
		if !strings.Contains(files.config, doc) {
			t.Errorf("Documentation string not found: %s", doc)
		}
	}
}

// TestResponseCaptureGeneration tests that responseCapture type is generated correctly.
func TestResponseCaptureGeneration(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	t.Run("type is generated", func(t *testing.T) {
		if !strings.Contains(files.binding, "type responseCapture struct") {
			t.Error("responseCapture type not found")
		}
		if !strings.Contains(files.binding, "http.ResponseWriter") {
			t.Error("responseCapture should embed http.ResponseWriter")
		}
		if !strings.Contains(files.binding, "wroteHeader bool") {
			t.Error("responseCapture should have wroteHeader field")
		}
		if !strings.Contains(files.binding, "written") || !strings.Contains(files.binding, "bool") {
			t.Error("responseCapture should have written field")
		}
	})

	t.Run("WriteHeader method is generated", func(t *testing.T) {
		if !strings.Contains(files.binding, "func (rc *responseCapture) WriteHeader(code int)") {
			t.Error("responseCapture WriteHeader method not found")
		}
		if !strings.Contains(files.binding, "rc.wroteHeader = true") {
			t.Error("WriteHeader should set wroteHeader to true")
		}
	})

	t.Run("Write method is generated", func(t *testing.T) {
		if !strings.Contains(files.binding, "func (rc *responseCapture) Write(b []byte) (int, error)") {
			t.Error("responseCapture Write method not found")
		}
		if !strings.Contains(files.binding, "rc.written = true") {
			t.Error("Write should set written to true")
		}
	})
}

// TestWriteErrorWithHandlerGeneration tests that writeErrorWithHandler is generated correctly.
func TestWriteErrorWithHandlerGeneration(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	t.Run("function is generated", func(t *testing.T) {
		if !strings.Contains(
			files.binding,
			"func writeErrorWithHandler(w http.ResponseWriter, r *http.Request, err error, handler ErrorHandler)",
		) {
			t.Error("writeErrorWithHandler function not found")
		}
	})

	t.Run("handles nil handler", func(t *testing.T) {
		if !strings.Contains(files.binding, "if handler != nil") {
			t.Error("writeErrorWithHandler should check for nil handler")
		}
	})

	t.Run("uses responseCapture", func(t *testing.T) {
		if !strings.Contains(files.binding, "capture = &responseCapture{ResponseWriter: w}") {
			t.Error("writeErrorWithHandler should create responseCapture")
		}
	})

	t.Run("checks for direct write", func(t *testing.T) {
		if !strings.Contains(files.binding, "if capture.written") {
			t.Error("writeErrorWithHandler should check capture.written")
		}
	})

	t.Run("checks for custom status", func(t *testing.T) {
		if !strings.Contains(files.binding, "capture.wroteHeader") {
			t.Error("writeErrorWithHandler should check capture.wroteHeader")
		}
	})
}

// TestDefaultErrorFunctionsGeneration tests that default error functions are generated.
func TestDefaultErrorFunctionsGeneration(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	t.Run("defaultErrorResponse is generated", func(t *testing.T) {
		if !strings.Contains(files.binding, "func defaultErrorResponse(err error) proto.Message") {
			t.Error("defaultErrorResponse function not found")
		}
	})

	t.Run("defaultErrorResponse handles ValidationError", func(t *testing.T) {
		if !strings.Contains(files.binding, "var valErr *sebufhttp.ValidationError") {
			t.Error("defaultErrorResponse should check for ValidationError")
		}
	})

	t.Run("defaultErrorResponse handles Error", func(t *testing.T) {
		if !strings.Contains(files.binding, "var handlerErr *sebufhttp.Error") {
			t.Error("defaultErrorResponse should check for Error")
		}
	})

	t.Run("defaultErrorStatusCode is generated", func(t *testing.T) {
		if !strings.Contains(files.binding, "func defaultErrorStatusCode(err error) int") {
			t.Error("defaultErrorStatusCode function not found")
		}
	})

	t.Run("defaultErrorStatusCode returns BadRequest", func(t *testing.T) {
		if !strings.Contains(files.binding, "return http.StatusBadRequest") {
			t.Error("defaultErrorStatusCode should return BadRequest for validation errors")
		}
	})

	t.Run("defaultErrorStatusCode returns InternalServerError", func(t *testing.T) {
		if !strings.Contains(files.binding, "return http.StatusInternalServerError") {
			t.Error("defaultErrorStatusCode should return InternalServerError as default")
		}
	})

	t.Run("convertProtovalidateError is generated", func(t *testing.T) {
		if !strings.Contains(files.binding, "func convertProtovalidateError(err error) *sebufhttp.ValidationError") {
			t.Error("convertProtovalidateError function not found")
		}
	})

	t.Run("writeResponseBody is generated", func(t *testing.T) {
		if !strings.Contains(
			files.binding,
			"func writeResponseBody(w http.ResponseWriter, r *http.Request, msg proto.Message)",
		) {
			t.Error("writeResponseBody function not found")
		}
	})
}

// TestErrorHandlerIntegration tests that errorHandler is passed through the generated code.
func TestErrorHandlerIntegration(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	t.Run("genericHandler receives errorHandler", func(t *testing.T) {
		if !strings.Contains(files.http, "config.errorHandler), serviceHeaders, methodHeaders") {
			t.Error("genericHandler should receive config.errorHandler")
		}
	})

	t.Run("BindingMiddleware receives errorHandler", func(t *testing.T) {
		if !strings.Contains(files.http, ", config.errorHandler,") {
			t.Error("BindingMiddleware should receive config.errorHandler")
		}
	})
}

// TestErrorHandlerEdgeCases tests edge cases in the generated error handler code.
func TestErrorHandlerEdgeCases(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	t.Run("BindingMiddleware signature includes errorHandler", func(t *testing.T) {
		if !strings.Contains(files.binding, "httpMethod string, errorHandler ErrorHandler) http.Handler") {
			t.Error("BindingMiddleware should have errorHandler as last parameter")
		}
	})

	t.Run("genericHandler signature includes errorHandler", func(t *testing.T) {
		if !strings.Contains(
			files.binding,
			"func genericHandler[Req any, Res any](serve func(context.Context, Req) (Res, error), errorHandler ErrorHandler) http.HandlerFunc",
		) {
			t.Error("genericHandler should have errorHandler parameter")
		}
	})

	t.Run("header validation uses writeErrorWithHandler", func(t *testing.T) {
		if !strings.Contains(files.binding, "writeErrorWithHandler(w, r, validationErr, errorHandler)") {
			t.Error("Header validation should use writeErrorWithHandler")
		}
	})

	t.Run("path param binding uses writeErrorWithHandler", func(t *testing.T) {
		if !strings.Contains(files.binding, "writeErrorWithHandler(w, r, err, errorHandler)") {
			t.Error("Path param binding should use writeErrorWithHandler")
		}
	})

	t.Run("body validation uses writeErrorWithHandler", func(t *testing.T) {
		if !strings.Contains(
			files.binding,
			"writeErrorWithHandler(w, r, convertProtovalidateError(err), errorHandler)",
		) {
			t.Error("Body validation should use writeErrorWithHandler with convertProtovalidateError")
		}
	})

	t.Run("handler errors use writeErrorWithHandler", func(t *testing.T) {
		if !strings.Contains(files.binding, "writeErrorWithHandler(w, r, errorMsg, errorHandler)") {
			t.Error("Handler errors should use writeErrorWithHandler")
		}
	})

	t.Run("writeErrorWithHandler calls defaultErrorResponse when response is nil", func(t *testing.T) {
		if !strings.Contains(files.binding, "response = defaultErrorResponse(err)") {
			t.Error("writeErrorWithHandler should call defaultErrorResponse when response is nil")
		}
	})

	t.Run("writeErrorWithHandler calls defaultErrorStatusCode", func(t *testing.T) {
		if !strings.Contains(files.binding, "statusCode := defaultErrorStatusCode(err)") {
			t.Error("writeErrorWithHandler should call defaultErrorStatusCode")
		}
	})

	t.Run("writeErrorWithHandler uses writeResponseBody for custom status", func(t *testing.T) {
		if !strings.Contains(files.binding, "writeResponseBody(w, r, response)") {
			t.Error("writeErrorWithHandler should use writeResponseBody when handler set status")
		}
	})
}

// TestErrorHandlerBackwardCompatibility ensures backward compatibility.
func TestErrorHandlerBackwardCompatibility(t *testing.T) {
	files := generateTestFiles(t, "backward_compat.proto")

	t.Run("WithMux still works", func(t *testing.T) {
		if !strings.Contains(files.config, "func WithMux(mux *http.ServeMux) ServerOption") {
			t.Error("WithMux function should still be available")
		}
	})

	t.Run("getDefaultConfiguration does not set errorHandler", func(t *testing.T) {
		if strings.Contains(files.config, "errorHandler:") {
			t.Error("getDefaultConfiguration should not initialize errorHandler")
		}
	})

	t.Run("original error functions still exist", func(t *testing.T) {
		if !strings.Contains(files.binding, "func writeProtoMessageResponse") {
			t.Error("writeProtoMessageResponse should still exist")
		}
		if !strings.Contains(files.binding, "func writeValidationErrorResponse") {
			t.Error("writeValidationErrorResponse should still exist")
		}
		if !strings.Contains(files.binding, "func writeErrorResponse") {
			t.Error("writeErrorResponse should still exist")
		}
	})
}

// TestProtoMessageErrorPreservation tests that proto.Message errors are preserved without wrapping.
func TestProtoMessageErrorPreservation(t *testing.T) {
	files := generateTestFiles(t, "http_verbs_comprehensive.proto")

	t.Run("genericHandler checks for proto.Message errors", func(t *testing.T) {
		// The genericHandler should check if the error is already a proto.Message
		if !strings.Contains(files.binding, "if protoErr, ok := err.(proto.Message); ok {") {
			t.Error("genericHandler should check if error is a proto.Message")
		}
	})

	t.Run("genericHandler passes proto.Message errors directly to writeErrorWithHandler", func(t *testing.T) {
		// When error is a proto.Message, it should be passed directly without wrapping
		if !strings.Contains(files.binding, "writeErrorWithHandler(w, r, protoErr, errorHandler)") {
			t.Error("genericHandler should pass proto.Message errors directly to writeErrorWithHandler")
		}
	})

	t.Run("genericHandler has comment explaining proto.Message check", func(t *testing.T) {
		if !strings.Contains(files.binding, "Check if error is already a proto.Message") {
			t.Error("genericHandler should have comment explaining proto.Message check")
		}
	})

	t.Run("defaultErrorResponse checks for proto.Message errors", func(t *testing.T) {
		// defaultErrorResponse should also check for proto.Message errors as fallback
		if !strings.Contains(files.binding, "if protoErr, ok := err.(proto.Message); ok {") {
			t.Error("defaultErrorResponse should check if error is a proto.Message")
		}
	})

	t.Run("defaultErrorResponse returns proto.Message errors directly", func(t *testing.T) {
		// The function should return the proto.Message directly without wrapping
		if !strings.Contains(files.binding, "return protoErr") {
			t.Error("defaultErrorResponse should return proto.Message errors directly")
		}
	})

	t.Run("proto.Message check comes before sebufhttp.Error wrapping", func(t *testing.T) {
		// In genericHandler, the proto.Message check should come before the sebufhttp.Error wrapping
		binding := files.binding
		protoMsgCheckPos := strings.Index(binding, "if protoErr, ok := err.(proto.Message)")
		sebufErrorPos := strings.Index(binding, "errorMsg := &sebufhttp.Error{")

		if protoMsgCheckPos == -1 || sebufErrorPos == -1 {
			t.Error("Both proto.Message check and sebufhttp.Error creation should exist")
			return
		}

		if protoMsgCheckPos > sebufErrorPos {
			t.Error("proto.Message check should come before sebufhttp.Error wrapping in genericHandler")
		}
	})

	t.Run("defaultErrorResponse proto.Message check comes after known error types", func(t *testing.T) {
		// In defaultErrorResponse, proto.Message check should be after ValidationError and Error checks
		binding := files.binding

		// Find the defaultErrorResponse function
		funcStart := strings.Index(binding, "func defaultErrorResponse(err error) proto.Message {")
		if funcStart == -1 {
			t.Error("defaultErrorResponse function not found")
			return
		}

		funcBody := binding[funcStart:]

		// Check order: ValidationError -> Error -> proto.Message -> fallback
		valErrPos := strings.Index(funcBody, "var valErr *sebufhttp.ValidationError")
		handlerErrPos := strings.Index(funcBody, "var handlerErr *sebufhttp.Error")
		protoMsgPos := strings.Index(funcBody, "if protoErr, ok := err.(proto.Message)")
		fallbackPos := strings.Index(funcBody, "return &sebufhttp.Error{Message: err.Error()}")

		if valErrPos == -1 || handlerErrPos == -1 || protoMsgPos == -1 || fallbackPos == -1 {
			t.Error("defaultErrorResponse should have all four error handling cases")
			return
		}

		if !(valErrPos < handlerErrPos && handlerErrPos < protoMsgPos && protoMsgPos < fallbackPos) {
			t.Error("defaultErrorResponse should check errors in order: ValidationError, Error, proto.Message, fallback")
		}
	})
}
