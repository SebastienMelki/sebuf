package openapiv3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"
)

func TestConvertScalarField(t *testing.T) {
	tests := []struct {
		name         string
		fieldKind    protoreflect.Kind
		expectedType []string
		expectedFormat string
		expectedMin  *float64
	}{
		{
			name:         "boolean field",
			fieldKind:    protoreflect.BoolKind,
			expectedType: []string{"boolean"},
		},
		{
			name:         "int32 field",
			fieldKind:    protoreflect.Int32Kind,
			expectedType: []string{"integer"},
			expectedFormat: "int32",
		},
		{
			name:         "int64 field",
			fieldKind:    protoreflect.Int64Kind,
			expectedType: []string{"integer"},
			expectedFormat: "int64",
		},
		{
			name:         "uint32 field",
			fieldKind:    protoreflect.Uint32Kind,
			expectedType: []string{"integer"},
			expectedFormat: "int32",
			expectedMin:  func() *float64 { v := 0.0; return &v }(),
		},
		{
			name:         "uint64 field",
			fieldKind:    protoreflect.Uint64Kind,
			expectedType: []string{"integer"},
			expectedFormat: "int64",
			expectedMin:  func() *float64 { v := 0.0; return &v }(),
		},
		{
			name:         "float field",
			fieldKind:    protoreflect.FloatKind,
			expectedType: []string{"number"},
			expectedFormat: "float",
		},
		{
			name:         "double field",
			fieldKind:    protoreflect.DoubleKind,
			expectedType: []string{"number"},
			expectedFormat: "double",
		},
		{
			name:         "string field",
			fieldKind:    protoreflect.StringKind,
			expectedType: []string{"string"},
		},
		{
			name:         "bytes field",
			fieldKind:    protoreflect.BytesKind,
			expectedType: []string{"string"},
			expectedFormat: "byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock field with the specified kind
			field := createMockField(tt.fieldKind, false, false)
			g := NewGenerator(FormatYAML)
			
			schemaProxy := g.convertScalarField(field)
			require.NotNil(t, schemaProxy)
			
			// Extract the actual schema from the proxy
			schema := extractSchemaFromProxy(t, schemaProxy)
			
			assert.Equal(t, tt.expectedType, schema.Type)
			if tt.expectedFormat != "" {
				assert.Equal(t, tt.expectedFormat, schema.Format)
			}
			if tt.expectedMin != nil {
				assert.Equal(t, *tt.expectedMin, *schema.Minimum)
			}
		})
	}
}

func TestConvertField_RepeatedField(t *testing.T) {
	// Create a mock repeated string field
	field := createMockField(protoreflect.StringKind, true, false)
	g := NewGenerator(FormatYAML)
	
	schemaProxy := g.convertField(field)
	require.NotNil(t, schemaProxy)
	
	// Extract the actual schema from the proxy
	schema := extractSchemaFromProxy(t, schemaProxy)
	
	assert.Equal(t, []string{"array"}, schema.Type)
	assert.NotNil(t, schema.Items)
	assert.NotNil(t, schema.Items.A)
	
	// Check the items schema
	itemSchema := extractSchemaFromProxy(t, schema.Items.A)
	assert.Equal(t, []string{"string"}, itemSchema.Type)
}

func TestConvertField_MapField(t *testing.T) {
	// Create a mock map field
	field := createMockField(protoreflect.MessageKind, false, true)
	g := NewGenerator(FormatYAML)
	
	schemaProxy := g.convertMapField(field)
	require.NotNil(t, schemaProxy)
	
	// Extract the actual schema from the proxy
	schema := extractSchemaFromProxy(t, schemaProxy)
	
	assert.Equal(t, []string{"object"}, schema.Type)
	assert.NotNil(t, schema.AdditionalProperties)
}

func TestConvertEnumField(t *testing.T) {
	// Create a mock enum field
	enumValues := []string{"UNSPECIFIED", "ACTIVE", "INACTIVE"}
	field := createMockEnumField(enumValues)
	g := NewGenerator(FormatYAML)
	
	schemaProxy := g.convertEnumField(field)
	require.NotNil(t, schemaProxy)
	
	// Extract the actual schema from the proxy
	schema := extractSchemaFromProxy(t, schemaProxy)
	
	assert.Equal(t, []string{"string"}, schema.Type)
	assert.Equal(t, len(enumValues), len(schema.Enum))
	
	// Check enum values
	for i, enumNode := range schema.Enum {
		assert.Equal(t, enumValues[i], enumNode.Value)
	}
}

func TestConvertEnumField_NilEnum(t *testing.T) {
	// Create a field with nil enum
	field := &protogen.Field{
		Desc: &mockFieldDescriptor{
			kind: protoreflect.EnumKind,
		},
		Enum: nil,
	}
	
	g := NewGenerator(FormatYAML)
	schemaProxy := g.convertEnumField(field)
	require.NotNil(t, schemaProxy)
	
	// Extract the actual schema from the proxy
	schema := extractSchemaFromProxy(t, schemaProxy)
	
	// Should fallback to string type
	assert.Equal(t, []string{"string"}, schema.Type)
	assert.Nil(t, schema.Enum)
}

// Helper functions for testing

func createMockField(kind protoreflect.Kind, isList, isMap bool) *protogen.Field {
	return &protogen.Field{
		Desc: &mockFieldDescriptor{
			kind:   kind,
			isList: isList,
			isMap:  isMap,
		},
		Comments: protogen.CommentSet{
			Leading: "Test field comment",
		},
	}
}

func createMockEnumField(values []string) *protogen.Field {
	enumValues := make([]*protogen.EnumValue, len(values))
	for i, v := range values {
		enumValues[i] = &protogen.EnumValue{
			Desc: &mockEnumValueDescriptor{
				name: protoreflect.Name(v),
			},
		}
	}
	
	return &protogen.Field{
		Desc: &mockFieldDescriptor{
			kind: protoreflect.EnumKind,
		},
		Enum: &protogen.Enum{
			Values: enumValues,
			Comments: protogen.CommentSet{
				Leading: "Test enum comment",
			},
		},
	}
}

func extractSchemaFromProxy(t *testing.T, proxy *base.SchemaProxy) *base.Schema {
	t.Helper()
	
	// Build the schema to get the actual Schema object
	built, err := proxy.BuildSchema()
	require.NoError(t, err)
	require.NotNil(t, built)
	
	return built.Schema
}

// Mock implementations for testing

type mockFieldDescriptor struct {
	protoreflect.FieldDescriptor
	kind       protoreflect.Kind
	isList     bool
	isMap      bool
	hasOptional bool
	jsonName   string
	options    *descriptorpb.FieldOptions
}

func (m *mockFieldDescriptor) Kind() protoreflect.Kind { return m.kind }
func (m *mockFieldDescriptor) IsList() bool { return m.isList }
func (m *mockFieldDescriptor) IsMap() bool { return m.isMap }
func (m *mockFieldDescriptor) HasOptionalKeyword() bool { return m.hasOptional }
func (m *mockFieldDescriptor) JSONName() string {
	if m.jsonName != "" {
		return m.jsonName
	}
	return "test_field"
}
func (m *mockFieldDescriptor) Options() protoreflect.ProtoMessage {
	if m.options != nil {
		return m.options
	}
	return &descriptorpb.FieldOptions{}
}
func (m *mockFieldDescriptor) Number() protoreflect.FieldNumber { return 1 }

type mockEnumValueDescriptor struct {
	protoreflect.EnumValueDescriptor
	name protoreflect.Name
}

func (m *mockEnumValueDescriptor) Name() protoreflect.Name { return m.name }

type mockMessageDescriptor struct {
	protoreflect.MessageDescriptor
	name protoreflect.Name
}

func (m *mockMessageDescriptor) Name() protoreflect.Name { return m.name }