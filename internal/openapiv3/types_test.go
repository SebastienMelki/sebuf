package openapiv3

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"

	"github.com/SebastienMelki/sebuf/http"
)

// Test convertScalarField function with various protobuf field types
func TestConvertScalarField(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	tests := []struct {
		name           string
		kind           protoreflect.Kind
		expectedType   string
		expectedFormat string
		comment        string
		checkFn        func(*base.Schema) error
	}{
		{
			name:           "Bool field",
			kind:           protoreflect.BoolKind,
			expectedType:   "boolean",
			expectedFormat: "",
		},
		{
			name:           "Int32 field",
			kind:           protoreflect.Int32Kind,
			expectedType:   "integer",
			expectedFormat: "int32",
		},
		{
			name:           "Sint32 field",
			kind:           protoreflect.Sint32Kind,
			expectedType:   "integer",
			expectedFormat: "int32",
		},
		{
			name:           "Sfixed32 field",
			kind:           protoreflect.Sfixed32Kind,
			expectedType:   "integer",
			expectedFormat: "int32",
		},
		{
			name:           "Int64 field",
			kind:           protoreflect.Int64Kind,
			expectedType:   "integer",
			expectedFormat: "int64",
		},
		{
			name:           "Sint64 field",
			kind:           protoreflect.Sint64Kind,
			expectedType:   "integer",
			expectedFormat: "int64",
		},
		{
			name:           "Sfixed64 field",
			kind:           protoreflect.Sfixed64Kind,
			expectedType:   "integer",
			expectedFormat: "int64",
		},
		{
			name:           "Uint32 field",
			kind:           protoreflect.Uint32Kind,
			expectedType:   "integer",
			expectedFormat: "int32",
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != 0.0 {
					t.Error("Expected minimum 0 for uint32")
				}
				return nil
			},
		},
		{
			name:           "Fixed32 field",
			kind:           protoreflect.Fixed32Kind,
			expectedType:   "integer",
			expectedFormat: "int32",
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != 0.0 {
					t.Error("Expected minimum 0 for fixed32")
				}
				return nil
			},
		},
		{
			name:           "Uint64 field",
			kind:           protoreflect.Uint64Kind,
			expectedType:   "integer",
			expectedFormat: "int64",
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != 0.0 {
					t.Error("Expected minimum 0 for uint64")
				}
				return nil
			},
		},
		{
			name:           "Fixed64 field",
			kind:           protoreflect.Fixed64Kind,
			expectedType:   "integer",
			expectedFormat: "int64",
			checkFn: func(schema *base.Schema) error {
				if schema.Minimum == nil || *schema.Minimum != 0.0 {
					t.Error("Expected minimum 0 for fixed64")
				}
				return nil
			},
		},
		{
			name:           "Float field",
			kind:           protoreflect.FloatKind,
			expectedType:   "number",
			expectedFormat: "float",
		},
		{
			name:           "Double field",
			kind:           protoreflect.DoubleKind,
			expectedType:   "number",
			expectedFormat: "double",
		},
		{
			name:           "String field",
			kind:           protoreflect.StringKind,
			expectedType:   "string",
			expectedFormat: "",
		},
		{
			name:           "String field with comment",
			kind:           protoreflect.StringKind,
			expectedType:   "string",
			expectedFormat: "",
			comment:        "This is a test string field",
			checkFn: func(schema *base.Schema) error {
				if schema.Description != "This is a test string field" {
					t.Errorf("Expected description from comment, got: %s", schema.Description)
				}
				return nil
			},
		},
		{
			name:           "Bytes field",
			kind:           protoreflect.BytesKind,
			expectedType:   "string",
			expectedFormat: "byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock field
			field := &protogen.Field{
				Desc: &mockFieldDesc{
					name: "test_field",
					kind: tt.kind,
				},
				Comments: protogen.Comments(tt.comment),
			}

			schemaProxy := gen.convertScalarField(field)
			if schemaProxy == nil {
				t.Fatal("convertScalarField returned nil")
			}

			schema := schemaProxy.Schema()
			if schema == nil {
				t.Fatal("Schema proxy contains nil schema")
			}

			// Check type
			if len(schema.Type) != 1 || schema.Type[0] != tt.expectedType {
				t.Errorf("Expected type [%s], got %v", tt.expectedType, schema.Type)
			}

			// Check format
			if schema.Format != tt.expectedFormat {
				t.Errorf("Expected format %q, got %q", tt.expectedFormat, schema.Format)
			}

			// Run custom check function if provided
			if tt.checkFn != nil {
				if err := tt.checkFn(schema); err != nil {
					t.Errorf("Custom check failed: %v", err)
				}
			}
		})
	}
}

// Test convertField function for repeated (array) fields
func TestConvertFieldRepeated(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	// Create a repeated string field
	field := &protogen.Field{
		Desc: &mockFieldDesc{
			name: "tags",
			kind: protoreflect.StringKind,
			list: true,
		},
	}

	schemaProxy := gen.convertField(field)
	if schemaProxy == nil {
		t.Fatal("convertField returned nil")
	}

	schema := schemaProxy.Schema()
	if schema == nil {
		t.Fatal("Schema proxy contains nil schema")
	}

	// Check that it's an array type
	if len(schema.Type) != 1 || schema.Type[0] != "array" {
		t.Errorf("Expected type [array], got %v", schema.Type)
	}

	// Check that items is set
	if schema.Items == nil {
		t.Fatal("Array schema items is nil")
	}

	itemSchema := schema.Items.A
	if itemSchema == nil {
		t.Fatal("Array items schema is nil")
	}

	itemSchemaActual := itemSchema.Schema()
	if itemSchemaActual == nil {
		t.Fatal("Array items actual schema is nil")
	}

	// Check item type is string
	if len(itemSchemaActual.Type) != 1 || itemSchemaActual.Type[0] != "string" {
		t.Errorf("Expected array items type [string], got %v", itemSchemaActual.Type)
	}
}

// Test convertMapField function
func TestConvertMapField(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	// Create a mock map field (map<string, string>)
	// Map fields in protobuf are represented as messages with key and value fields
	valueField := &protogen.Field{
		Desc: &mockFieldDesc{
			name:   "value",
			kind:   protoreflect.StringKind,
			number: 2, // Value field is always number 2
		},
	}

	keyField := &protogen.Field{
		Desc: &mockFieldDesc{
			name:   "key", 
			kind:   protoreflect.StringKind,
			number: 1, // Key field is always number 1
		},
	}

	mapEntryMessage := &protogen.Message{
		Fields: []*protogen.Field{keyField, valueField},
	}

	field := &protogen.Field{
		Desc: &mockFieldDesc{
			name: "metadata",
			kind: protoreflect.MessageKind,
			isMap: true,
		},
		Message:  mapEntryMessage,
		Comments: protogen.Comments("Map of metadata"),
	}

	schemaProxy := gen.convertMapField(field)
	if schemaProxy == nil {
		t.Fatal("convertMapField returned nil")
	}

	schema := schemaProxy.Schema()
	if schema == nil {
		t.Fatal("Schema proxy contains nil schema")
	}

	// Check that it's an object type
	if len(schema.Type) != 1 || schema.Type[0] != "object" {
		t.Errorf("Expected type [object], got %v", schema.Type)
	}

	// Check that additionalProperties is set
	if schema.AdditionalProperties == nil {
		t.Fatal("Map schema additionalProperties is nil")
	}

	additionalPropsSchema := schema.AdditionalProperties.A
	if additionalPropsSchema == nil {
		t.Fatal("AdditionalProperties schema is nil")
	}

	additionalPropsActual := additionalPropsSchema.Schema()
	if additionalPropsActual == nil {
		t.Fatal("AdditionalProperties actual schema is nil")
	}

	// Check value type is string
	if len(additionalPropsActual.Type) != 1 || additionalPropsActual.Type[0] != "string" {
		t.Errorf("Expected map value type [string], got %v", additionalPropsActual.Type)
	}

	// Check description from comment
	if schema.Description != "Map of metadata" {
		t.Errorf("Expected description 'Map of metadata', got %q", schema.Description)
	}
}

// Test convertEnumField function
func TestConvertEnumField(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	// Create mock enum values
	enumValues := []*protogen.EnumValue{
		{
			Desc: &mockEnumValueDesc{
				name: "STATUS_UNSPECIFIED",
			},
		},
		{
			Desc: &mockEnumValueDesc{
				name: "STATUS_ACTIVE",
			},
		},
		{
			Desc: &mockEnumValueDesc{
				name: "STATUS_INACTIVE",
			},
		},
	}

	// Create mock enum
	enum := &protogen.Enum{
		Values:   enumValues,
		Comments: protogen.Comments("Status enumeration"),
	}

	// Create field with enum
	field := &protogen.Field{
		Desc: &mockFieldDesc{
			name: "status",
			kind: protoreflect.EnumKind,
		},
		Enum: enum,
	}

	schemaProxy := gen.convertEnumField(field)
	if schemaProxy == nil {
		t.Fatal("convertEnumField returned nil")
	}

	schema := schemaProxy.Schema()
	if schema == nil {
		t.Fatal("Schema proxy contains nil schema")
	}

	// Check that it's a string type
	if len(schema.Type) != 1 || schema.Type[0] != "string" {
		t.Errorf("Expected type [string], got %v", schema.Type)
	}

	// Check enum values
	if len(schema.Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(schema.Enum))
	}

	expectedValues := []string{"STATUS_UNSPECIFIED", "STATUS_ACTIVE", "STATUS_INACTIVE"}
	for i, expected := range expectedValues {
		if i >= len(schema.Enum) {
			t.Errorf("Missing enum value at index %d", i)
			continue
		}
		if schema.Enum[i].Value != expected {
			t.Errorf("Expected enum value %q at index %d, got %q", expected, i, schema.Enum[i].Value)
		}
	}

	// Check description from enum comment
	if schema.Description != "Status enumeration" {
		t.Errorf("Expected description 'Status enumeration', got %q", schema.Description)
	}
}

// Test getFieldExamples function
func TestGetFieldExamples(t *testing.T) {
	tests := []struct {
		name              string
		hasFieldExamples  bool
		exampleValues     []string
		expectedCount     int
	}{
		{
			name:              "Field with examples",
			hasFieldExamples:  true,
			exampleValues:     []string{"example1", "example2", "example3"},
			expectedCount:     3,
		},
		{
			name:              "Field without examples", 
			hasFieldExamples:  false,
			exampleValues:     nil,
			expectedCount:     0,
		},
		{
			name:              "Field with single example",
			hasFieldExamples:  true,
			exampleValues:     []string{"single_example"},
			expectedCount:     1,
		},
		{
			name:              "Field with empty examples",
			hasFieldExamples:  true,
			exampleValues:     []string{},
			expectedCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &protogen.Field{
				Desc: &mockFieldDescWithExamples{
					name:             "test_field",
					hasFieldExamples: tt.hasFieldExamples,
					exampleValues:    tt.exampleValues,
				},
			}

			examples := getFieldExamples(field)

			if len(examples) != tt.expectedCount {
				t.Errorf("Expected %d examples, got %d", tt.expectedCount, len(examples))
			}

			for i, expected := range tt.exampleValues {
				if i >= len(examples) {
					t.Errorf("Missing example at index %d", i)
					continue
				}
				if examples[i] != expected {
					t.Errorf("Expected example %q at index %d, got %q", expected, i, examples[i])
				}
			}
		})
	}
}

// Test field with examples in schema
func TestConvertScalarFieldWithExamples(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	field := &protogen.Field{
		Desc: &mockFieldDescWithExamples{
			name:             "email",
			kind:             protoreflect.StringKind,
			hasFieldExamples: true,
			exampleValues:    []string{"user@example.com", "admin@example.com", "test@example.com"},
		},
		Comments: protogen.Comments("User email address"),
	}

	schemaProxy := gen.convertScalarField(field)
	if schemaProxy == nil {
		t.Fatal("convertScalarField returned nil")
	}

	schema := schemaProxy.Schema()
	if schema == nil {
		t.Fatal("Schema proxy contains nil schema")
	}

	// Check that examples were set
	if len(schema.Examples) != 3 {
		t.Errorf("Expected 3 examples, got %d", len(schema.Examples))
	}

	// Check that the first example is set as the default example
	if schema.Example == nil {
		t.Error("Expected default example to be set")
	} else if schema.Example.Value != "user@example.com" {
		t.Errorf("Expected default example 'user@example.com', got %q", schema.Example.Value)
	}

	// Check all examples
	expectedExamples := []string{"user@example.com", "admin@example.com", "test@example.com"}
	for i, expected := range expectedExamples {
		if i >= len(schema.Examples) {
			t.Errorf("Missing example at index %d", i)
			continue
		}
		if schema.Examples[i].Value != expected {
			t.Errorf("Expected example %q at index %d, got %q", expected, i, schema.Examples[i].Value)
		}
	}
}

// Test optional field handling  
func TestConvertFieldOptional(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	// Create an optional field
	field := &protogen.Field{
		Desc: &mockFieldDesc{
			name:     "optional_field",
			kind:     protoreflect.StringKind,
			optional: true,
		},
	}

	schemaProxy := gen.convertField(field)
	if schemaProxy == nil {
		t.Fatal("convertField returned nil")
	}

	schema := schemaProxy.Schema()
	if schema == nil {
		t.Fatal("Schema proxy contains nil schema")
	}

	// Check that it's still a string type (optional is handled at message level)
	if len(schema.Type) != 1 || schema.Type[0] != "string" {
		t.Errorf("Expected type [string], got %v", schema.Type)
	}
}

// === Additional mock implementations ===

type mockEnumValueDesc struct {
	name string
}

func (e *mockEnumValueDesc) Name() protoreflect.Name         { return protoreflect.Name(e.name) }
func (e *mockEnumValueDesc) FullName() protoreflect.FullName { return protoreflect.FullName(e.name) }
func (e *mockEnumValueDesc) IsPlaceholder() bool            { return false }
func (e *mockEnumValueDesc) Options() protoreflect.ProtoMessage {
	return &descriptorpb.EnumValueOptions{}
}
func (e *mockEnumValueDesc) Index() int                     { return 0 }
func (e *mockEnumValueDesc) Syntax() protoreflect.Syntax   { return protoreflect.Proto3 }
func (e *mockEnumValueDesc) Number() protoreflect.EnumNumber { return 0 }
func (e *mockEnumValueDesc) Parent() protoreflect.Descriptor { return nil }
func (e *mockEnumValueDesc) ParentFile() protoreflect.FileDescriptor { return nil }

type mockFieldDescWithExamples struct {
	name             string
	jsonName         string
	kind             protoreflect.Kind
	hasFieldExamples bool
	exampleValues    []string
	number           protoreflect.FieldNumber
}

func (f *mockFieldDescWithExamples) Name() protoreflect.Name { return protoreflect.Name(f.name) }
func (f *mockFieldDescWithExamples) FullName() protoreflect.FullName {
	return protoreflect.FullName(f.name)
}
func (f *mockFieldDescWithExamples) IsPlaceholder() bool { return false }
func (f *mockFieldDescWithExamples) Options() protoreflect.ProtoMessage {
	options := &descriptorpb.FieldOptions{}
	if f.hasFieldExamples {
		// In a real implementation, this would set the sebuf.http.field_examples extension
		fieldExamples := &http.FieldExamples{
			Values: f.exampleValues,
		}
		proto.SetExtension(options, http.E_FieldExamples, fieldExamples)
	}
	return options
}
func (f *mockFieldDescWithExamples) Index() int { return 0 }
func (f *mockFieldDescWithExamples) Syntax() protoreflect.Syntax { return protoreflect.Proto3 }
func (f *mockFieldDescWithExamples) Number() protoreflect.FieldNumber {
	if f.number != 0 {
		return f.number
	}
	return 1
}
func (f *mockFieldDescWithExamples) Cardinality() protoreflect.Cardinality {
	return protoreflect.Optional
}
func (f *mockFieldDescWithExamples) Kind() protoreflect.Kind { return f.kind }
func (f *mockFieldDescWithExamples) HasJSONName() bool       { return f.jsonName != "" }
func (f *mockFieldDescWithExamples) JSONName() string {
	if f.jsonName != "" {
		return f.jsonName
	}
	return f.name
}
func (f *mockFieldDescWithExamples) TextName() string                           { return f.name }
func (f *mockFieldDescWithExamples) HasPresence() bool                          { return false }
func (f *mockFieldDescWithExamples) IsExtension() bool                          { return false }
func (f *mockFieldDescWithExamples) IsWeak() bool                               { return false }
func (f *mockFieldDescWithExamples) IsPacked() bool                             { return false }
func (f *mockFieldDescWithExamples) IsList() bool                               { return false }
func (f *mockFieldDescWithExamples) IsMap() bool                                { return false }
func (f *mockFieldDescWithExamples) MapKey() protoreflect.FieldDescriptor       { return nil }
func (f *mockFieldDescWithExamples) MapValue() protoreflect.FieldDescriptor     { return nil }
func (f *mockFieldDescWithExamples) HasDefault() bool                           { return false }
func (f *mockFieldDescWithExamples) Default() protoreflect.Value                { return protoreflect.Value{} }
func (f *mockFieldDescWithExamples) DefaultEnumValue() protoreflect.EnumValueDescriptor { return nil }
func (f *mockFieldDescWithExamples) ContainingOneof() protoreflect.OneofDescriptor { return nil }
func (f *mockFieldDescWithExamples) ContainingMessage() protoreflect.MessageDescriptor { return nil }
func (f *mockFieldDescWithExamples) Enum() protoreflect.EnumDescriptor { return nil }
func (f *mockFieldDescWithExamples) Message() protoreflect.MessageDescriptor { return nil }
func (f *mockFieldDescWithExamples) Parent() protoreflect.Descriptor { return nil }
func (f *mockFieldDescWithExamples) ParentFile() protoreflect.FileDescriptor { return nil }
func (f *mockFieldDescWithExamples) HasOptionalKeyword() bool { return false }

// Update mockFieldDesc to support map and number fields
func (f *mockFieldDesc) Number() protoreflect.FieldNumber {
	if f.number != 0 {
		return f.number
	}
	return 1
}
func (f *mockFieldDesc) IsMap() bool { return f.isMap }