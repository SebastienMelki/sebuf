package annotations

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// URLParamScalarKinds is the human-readable list of proto field kinds that can be
// bound from a URL. It is shared by every generator's error message so the wording
// cannot drift between plugins.
const URLParamScalarKinds = "string, int32, int64, uint32, uint64, bool, float, double, enum"

// URL parameter locations, used by URLParamValidationError.
const (
	URLParamLocationPath  = "path"
	URLParamLocationQuery = "query"
)

// URLParamValidationError describes a field bound to a URL path or query parameter
// whose type cannot be represented in a URL.
type URLParamValidationError struct {
	MessageName string // Request message name (e.g., "ListClientRestrictionsRequest")
	FieldName   string // Proto field name (e.g., "client_id")
	ParamName   string // Path variable or query parameter name (e.g., "clientId")
	Location    string // URLParamLocationPath or URLParamLocationQuery
	TypeName    string // Descriptive field type (e.g., "message (core.v1.UserClientID)")
}

func (e *URLParamValidationError) Error() string {
	if e.Location == URLParamLocationPath {
		return fmt.Sprintf(
			"path variable '{%s}' is bound to field '%s' on message '%s' of type '%s', "+
				"but path parameters must be scalar types (%s). "+
				"Change the field type or remove it from the path.",
			e.ParamName, e.FieldName, e.MessageName, e.TypeName, URLParamScalarKinds)
	}

	return fmt.Sprintf(
		"field '%s' on message '%s' is annotated with (sebuf.http.query) as parameter '%s', "+
			"but has unsupported type '%s'. Query parameters must be scalar types (%s) "+
			"or repeated scalars. Replace it with a scalar field, or move it into the "+
			"request body by using POST/PUT/PATCH.",
		e.FieldName, e.MessageName, e.ParamName, e.TypeName, URLParamScalarKinds)
}

// IsURLParamKindCompatible reports whether a proto field kind can be represented in a
// URL path segment or query value.
//
// Message, group, and bytes kinds cannot: there is no canonical URL encoding for them,
// and every generator previously fell through to a string-shaped default that produced
// non-compiling Go, corrupted Python/TypeScript values, or a runtime 400. See issue #216.
func IsURLParamKindCompatible(kind protoreflect.Kind) bool {
	switch kind {
	case protoreflect.StringKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind,
		protoreflect.BoolKind,
		protoreflect.FloatKind, protoreflect.DoubleKind,
		protoreflect.EnumKind:
		return true
	case protoreflect.BytesKind, protoreflect.MessageKind, protoreflect.GroupKind:
		return false
	}
	return false
}

// IsURLParamCompatible reports whether a field can be bound from a URL path segment or
// query value. Map fields report MessageKind and are therefore rejected as well.
func IsURLParamCompatible(field *protogen.Field) bool {
	return IsURLParamKindCompatible(field.Desc.Kind())
}

// URLParamTypeName returns a descriptive type name for a field, used in validation
// errors. Message-kind fields are qualified with their full proto type name so the
// error names the offending type rather than just "message".
func URLParamTypeName(field *protogen.Field) string {
	if field.Desc.IsMap() {
		return "map"
	}

	kind := field.Desc.Kind().String()
	if field.Desc.Kind() == protoreflect.MessageKind && field.Message != nil {
		return fmt.Sprintf("%s (%s)", kind, field.Message.Desc.FullName())
	}

	return kind
}

// ValidateQueryParams checks that every (sebuf.http.query)-annotated field on message
// has a type that can be represented in a query string. Returns the first error found.
func ValidateQueryParams(message *protogen.Message) error {
	for _, qp := range GetQueryParams(message) {
		if IsURLParamCompatible(qp.Field) {
			continue
		}

		return &URLParamValidationError{
			MessageName: string(message.Desc.Name()),
			FieldName:   qp.FieldName,
			ParamName:   qp.ParamName,
			Location:    URLParamLocationQuery,
			TypeName:    URLParamTypeName(qp.Field),
		}
	}

	return nil
}

// ValidatePathParams checks that every path variable on the method's HTTP config is
// bound to a field whose type can be represented in a URL path segment.
//
// Path variables with no matching field are skipped: that is a separate diagnostic,
// reported by httpgen so it is not duplicated across every plugin.
func ValidatePathParams(method *protogen.Method) error {
	config := GetMethodHTTPConfig(method)
	if config == nil {
		return nil
	}

	for _, param := range config.PathParams {
		field := FindFieldByProtoName(method.Input, param)
		if field == nil || IsURLParamCompatible(field) {
			continue
		}

		return &URLParamValidationError{
			MessageName: string(method.Input.Desc.Name()),
			FieldName:   string(field.Desc.Name()),
			ParamName:   param,
			Location:    URLParamLocationPath,
			TypeName:    URLParamTypeName(field),
		}
	}

	return nil
}

// ValidateMethodURLParams validates that every path- and query-bound field of the
// method's request message has a URL-representable type.
//
// Every generator calls this so a proto that cannot be generated correctly fails at
// generation time with one clear message, rather than producing broken output that
// differs per plugin.
func ValidateMethodURLParams(method *protogen.Method) error {
	if err := ValidatePathParams(method); err != nil {
		return err
	}

	return ValidateQueryParams(method.Input)
}

// ValidateFileURLParams validates the URL parameters of every method of every service
// in the file. This is the single call site each generator wires in, so all six plugins
// reject the same protos with the same message.
//
// The "Service.Method: " prefix matches httpgen's ValidateService formatting.
func ValidateFileURLParams(file *protogen.File) error {
	for _, service := range file.Services {
		for _, method := range service.Methods {
			if err := ValidateMethodURLParams(method); err != nil {
				return fmt.Errorf("%s.%s: %w", service.Desc.Name(), method.Desc.Name(), err)
			}
		}
	}

	return nil
}

// FindFieldByProtoName returns the field with the given proto name, or nil.
func FindFieldByProtoName(message *protogen.Message, fieldName string) *protogen.Field {
	for _, field := range message.Fields {
		if string(field.Desc.Name()) == fieldName {
			return field
		}
	}

	return nil
}
