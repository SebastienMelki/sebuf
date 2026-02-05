package annotations

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
)

// QueryParam represents a query parameter configuration extracted from a field.
// This is the unified struct containing all fields needed by all 4 generators.
type QueryParam struct {
	FieldName     string          // Proto field name (e.g., "page_number")
	FieldGoName   string          // Go field name (e.g., "PageNumber")
	FieldJSONName string          // JSON field name / camelCase (e.g., "pageNumber")
	ParamName     string          // Query parameter name (e.g., "page")
	Required      bool            // Whether the parameter is required
	FieldKind     string          // Proto field kind (e.g., "string", "int32", "bool")
	Field         *protogen.Field // Raw protogen field reference
}

// GetQueryParams extracts query parameter configurations from message fields.
// Returns all fields that have the sebuf.http.query annotation.
func GetQueryParams(message *protogen.Message) []QueryParam {
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
			FieldName:     string(field.Desc.Name()),
			FieldGoName:   field.GoName,
			FieldJSONName: field.Desc.JSONName(),
			ParamName:     paramName,
			Required:      queryConfig.GetRequired(),
			FieldKind:     field.Desc.Kind().String(),
			Field:         field,
		})
	}

	return params
}
