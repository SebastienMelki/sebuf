package openapiv3

import (
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	yaml "go.yaml.in/yaml/v4"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/SebastienMelki/sebuf/http"
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
		schema.Type = []string{"integer"}
		schema.Format = "int32"

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		schema.Type = []string{"integer"}
		schema.Format = "int64"

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		schema.Type = []string{"integer"}
		schema.Format = "int32"
		zero := 0.0
		schema.Minimum = &zero

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		schema.Type = []string{"integer"}
		schema.Format = "int64"
		zero := 0.0
		schema.Minimum = &zero

	case protoreflect.FloatKind:
		schema.Type = []string{"number"}
		schema.Format = "float"

	case protoreflect.DoubleKind:
		schema.Type = []string{"number"}
		schema.Format = "double"

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
	if examples := getFieldExamples(field); len(examples) > 0 {
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

	// For maps, we need to determine the value type
	if field.Message != nil && len(field.Message.Fields) >= 2 {
		// Map entry messages have exactly 2 fields: key (field 1) and value (field 2)
		var valueField *protogen.Field
		const mapValueFieldNumber = 2 // value field is always number 2 in protobuf map entries
		for _, f := range field.Message.Fields {
			if f.Desc.Number() == mapValueFieldNumber {
				valueField = f
				break
			}
		}

		if valueField != nil {
			valueSchema := g.convertScalarField(valueField)
			schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
				A: valueSchema,
			}
		}
	}

	// If we couldn't determine the value type, allow any type
	if schema.AdditionalProperties == nil {
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
			B: true,
		}
	}

	// Add description from field comments
	if field.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(field.Comments.Leading))
	}

	// Apply validation constraints for the map itself
	extractValidationConstraints(field, schema)

	return base.CreateSchemaProxy(schema)
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
