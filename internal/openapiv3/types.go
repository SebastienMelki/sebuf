package openapiv3

import (
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	yaml "go.yaml.in/yaml/v4"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/SebastienMelki/sebuf/http"
	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// Header type constants for OpenAPI type mapping.
const (
	headerTypeString  = "string"
	headerTypeInt32   = "int32"
	headerTypeInt64   = "int64"
	headerTypeUint64  = "uint64"
	headerTypeInteger = "integer"
	headerTypeNumber  = "number"
	headerTypeFloat   = "float"
	headerTypeDouble  = "double"
)

// convertField converts a protobuf field to an OpenAPI schema.
func (g *Generator) convertField(field *protogen.Field) *base.SchemaProxy {
	// Handle repeated fields (arrays)
	if field.Desc.IsList() {
		itemSchema := g.convertScalarField(field)
		arraySchema := &base.Schema{
			Type: []string{"array"},
			Items: &base.DynamicValue[*base.SchemaProxy, bool]{
				A: itemSchema,
			},
		}

		// Apply validation constraints for the array itself
		extractValidationConstraints(field, arraySchema)

		return base.CreateSchemaProxy(arraySchema)
	}

	// Handle map fields
	if field.Desc.IsMap() {
		return g.convertMapField(field)
	}

	// Handle optional fields (proto3 optional)
	schema := g.convertScalarField(field)
	if field.Desc.HasOptionalKeyword() {
		// For proto3 optional fields, we could add nullable: true
		// but OpenAPI 3.1 handles this differently than 3.0
		return schema
	}

	return schema
}

// convertScalarField handles scalar field types and message references.
func (g *Generator) convertScalarField(field *protogen.Field) *base.SchemaProxy {
	schema := &base.Schema{}

	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		schema.Type = []string{"boolean"}

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		schema.Type = []string{headerTypeInteger}
		schema.Format = headerTypeInt32

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		// Per proto3 JSON spec, int64 serializes as a string to avoid JavaScript precision loss
		schema.Type = []string{headerTypeString}
		schema.Format = headerTypeInt64

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		schema.Type = []string{headerTypeInteger}
		schema.Format = headerTypeInt32
		zero := 0.0
		schema.Minimum = &zero

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		// Per proto3 JSON spec, uint64 serializes as a string to avoid JavaScript precision loss
		schema.Type = []string{headerTypeString}
		schema.Format = headerTypeUint64
		// Note: Minimum constraint removed since type is now string

	case protoreflect.FloatKind:
		schema.Type = []string{headerTypeNumber}
		schema.Format = headerTypeFloat

	case protoreflect.DoubleKind:
		schema.Type = []string{headerTypeNumber}
		schema.Format = headerTypeDouble

	case protoreflect.StringKind:
		schema.Type = []string{"string"}

	case protoreflect.BytesKind:
		schema.Type = []string{"string"}
		schema.Format = "byte"

	case protoreflect.EnumKind:
		return g.convertEnumField(field)

	case protoreflect.MessageKind:
		// Reference to another message
		return base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", g.getSchemaName(field.Message)))

	case protoreflect.GroupKind:
		// Groups are deprecated but still supported
		if field.Message != nil {
			return base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", g.getSchemaName(field.Message)))
		}
		schema.Type = []string{"object"}

	default:
		// Fallback for unknown types
		schema.Type = []string{"string"}
	}

	// Add description from field comments
	if field.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(field.Comments.Leading))
	}

	// Apply buf.validate constraints
	extractValidationConstraints(field, schema)

	// Add field examples if available
	if examples := annotations.GetFieldExamples(field); len(examples) > 0 {
		// Set the first example as the default example
		schema.Example = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: examples[0],
		}

		// Add all examples using OpenAPI 3.1 examples array format
		schema.Examples = make([]*yaml.Node, len(examples))
		for i, example := range examples {
			schema.Examples[i] = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: example,
			}
		}
	}

	return base.CreateSchemaProxy(schema)
}

// convertEnumField converts a protobuf enum field to an OpenAPI schema.
func (g *Generator) convertEnumField(field *protogen.Field) *base.SchemaProxy {
	if field.Enum == nil {
		// Fallback if enum is not available
		return base.CreateSchemaProxy(&base.Schema{
			Type: []string{"string"},
		})
	}

	schema := &base.Schema{
		Type: []string{"string"},
		Enum: make([]*yaml.Node, 0, len(field.Enum.Values)),
	}

	// Add enum values
	for _, value := range field.Enum.Values {
		schema.Enum = append(schema.Enum, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: string(value.Desc.Name()),
		})
	}

	// Add description from enum comments
	if field.Enum.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(field.Enum.Comments.Leading))
	}

	return base.CreateSchemaProxy(schema)
}

// convertMapField converts a protobuf map field to an OpenAPI schema.
func (g *Generator) convertMapField(field *protogen.Field) *base.SchemaProxy {
	schema := &base.Schema{
		Type: []string{"object"},
	}

	// Set additional properties based on map value type
	schema.AdditionalProperties = g.getMapValueSchema(field)

	// Add description from field comments
	if field.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(field.Comments.Leading))
	}

	// Apply validation constraints for the map itself
	extractValidationConstraints(field, schema)

	return base.CreateSchemaProxy(schema)
}

// getMapValueSchema returns the schema for the map's value type.
func (g *Generator) getMapValueSchema(field *protogen.Field) *base.DynamicValue[*base.SchemaProxy, bool] {
	valueField := getMapValueField(field)
	if valueField == nil {
		// Couldn't determine value type, allow any type
		return &base.DynamicValue[*base.SchemaProxy, bool]{B: true}
	}

	// Check if value is a message with an unwrap field
	if valueField.Message != nil {
		if unwrapField := annotations.FindUnwrapField(valueField.Message); unwrapField != nil {
			return g.createUnwrapArraySchema(unwrapField)
		}
	}

	// Normal scalar or message type
	valueSchema := g.convertScalarField(valueField)
	return &base.DynamicValue[*base.SchemaProxy, bool]{A: valueSchema}
}

// getMapValueField extracts the value field from a map field's entry message.
func getMapValueField(field *protogen.Field) *protogen.Field {
	if field.Message == nil || len(field.Message.Fields) < 2 {
		return nil
	}

	// Map entry messages have exactly 2 fields: key (field 1) and value (field 2)
	const mapValueFieldNumber = 2
	for _, f := range field.Message.Fields {
		if f.Desc.Number() == mapValueFieldNumber {
			return f
		}
	}
	return nil
}

// createUnwrapArraySchema creates an array schema for an unwrap field's element type.
func (g *Generator) createUnwrapArraySchema(unwrapField *protogen.Field) *base.DynamicValue[*base.SchemaProxy, bool] {
	itemSchema := g.convertScalarField(unwrapField)
	arraySchema := &base.Schema{
		Type: []string{"array"},
		Items: &base.DynamicValue[*base.SchemaProxy, bool]{
			A: itemSchema,
		},
	}
	return &base.DynamicValue[*base.SchemaProxy, bool]{
		A: base.CreateSchemaProxy(arraySchema),
	}
}

// convertHeadersToParameters converts proto headers to OpenAPI parameters.
func convertHeadersToParameters(headers []*http.Header) []*v3.Parameter {
	if len(headers) == 0 {
		return nil
	}

	parameters := make([]*v3.Parameter, 0, len(headers))

	for _, header := range headers {
		if header.GetName() == "" {
			continue // Skip headers without names
		}

		// Create the schema for the header
		schema := &base.Schema{
			Type: []string{mapHeaderTypeToOpenAPI(header.GetType())},
		}

		// Add format if specified
		if header.GetFormat() != "" {
			schema.Format = header.GetFormat()
		}

		// Add example if specified
		if header.GetExample() != "" {
			schema.Example = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: header.GetExample(),
			}
		}

		// Create the parameter
		parameter := &v3.Parameter{
			Name:        header.GetName(),
			In:          "header",
			Required:    &header.Required,
			Schema:      base.CreateSchemaProxy(schema),
			Description: header.GetDescription(),
		}

		// Set deprecated if specified
		if header.GetDeprecated() {
			parameter.Deprecated = true
		}

		parameters = append(parameters, parameter)
	}

	return parameters
}

// mapHeaderTypeToOpenAPI maps proto header types to OpenAPI schema types.
func mapHeaderTypeToOpenAPI(headerType string) string {
	switch strings.ToLower(headerType) {
	case headerTypeString, "":
		return headerTypeString
	case headerTypeInteger, "int", headerTypeInt32, headerTypeInt64:
		return headerTypeInteger
	case headerTypeNumber, headerTypeFloat, headerTypeDouble:
		return headerTypeNumber
	case "boolean", "bool":
		return "boolean"
	case "array":
		return "array"
	default:
		// Default to string for unknown types
		return headerTypeString
	}
}
