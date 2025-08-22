package openapiv3

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"
)

// Test NewGenerator constructor
func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name     string
		format   OutputFormat
		expected OutputFormat
	}{
		{
			name:     "YAML format",
			format:   FormatYAML,
			expected: FormatYAML,
		},
		{
			name:     "JSON format",
			format:   FormatJSON,
			expected: FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.format)

			// Check generator is not nil
			if gen == nil {
				t.Fatal("NewGenerator returned nil")
			}

			// Check format is set correctly
			if gen.format != tt.expected {
				t.Errorf("Expected format %v, got %v", tt.expected, gen.format)
			}

			// Check document is initialized
			if gen.doc == nil {
				t.Error("Document is nil")
			}

			// Check schemas map is initialized
			if gen.schemas == nil {
				t.Error("Schemas map is nil")
			}

			// Check document structure
			if gen.doc.Version != "3.1.0" {
				t.Errorf("Expected OpenAPI version 3.1.0, got %s", gen.doc.Version)
			}

			if gen.doc.Info == nil {
				t.Error("Info is nil")
			}

			if gen.doc.Info.Title != "Generated API" {
				t.Errorf("Expected default title 'Generated API', got %s", gen.doc.Info.Title)
			}

			if gen.doc.Info.Version != "1.0.0" {
				t.Errorf("Expected default version '1.0.0', got %s", gen.doc.Info.Version)
			}

			// Check paths are initialized
			if gen.doc.Paths == nil {
				t.Error("Paths is nil")
			}

			if gen.doc.Paths.PathItems == nil {
				t.Error("PathItems is nil")
			}

			// Check components are initialized
			if gen.doc.Components == nil {
				t.Error("Components is nil")
			}

			if gen.doc.Components.Schemas == nil {
				t.Error("Components.Schemas is nil")
			}
		})
	}
}

// Test ProcessService method
func TestProcessService(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	// Create a mock service
	service := createMockService("TestService", []mockMethod{
		{
			name:   "TestMethod",
			input:  createMockMessage("TestRequest", nil),
			output: createMockMessage("TestResponse", nil),
		},
	})

	// Process the service
	gen.ProcessService(service)

	// Check that the document title was updated
	expectedTitle := "TestService API"
	if gen.doc.Info.Title != expectedTitle {
		t.Errorf("Expected title %q, got %q", expectedTitle, gen.doc.Info.Title)
	}

	// Check that paths were added
	if gen.doc.Paths.PathItems.Len() == 0 {
		t.Error("No paths were added to the document")
	}

	// Check that the correct path was added
	expectedPath := "/TestService/TestMethod"
	pathItem := gen.doc.Paths.PathItems.GetOrZero(expectedPath)
	if pathItem == nil {
		t.Errorf("Expected path %q not found", expectedPath)
		// Debug: list all paths
		for pair := range gen.doc.Paths.PathItems.Iterate() {
			t.Logf("Found path: %s", pair.Key())
		}
	}
}

// Test ProcessMessage method
func TestProcessMessage(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	// Create a mock message
	message := createMockMessage("TestMessage", []mockField{
		{
			name:     "test_field",
			jsonName: "testField",
			kind:     protoreflect.StringKind,
		},
	})

	// Process the message
	gen.ProcessMessage(message)

	// Check that schema was added
	if gen.schemas.Len() == 0 {
		t.Error("No schemas were added")
	}

	// Check that the correct schema was added
	schemaProxy := gen.schemas.GetOrZero("TestMessage")
	if schemaProxy == nil {
		t.Error("Expected schema 'TestMessage' not found")
	}
}

// Test Render method
func TestRender(t *testing.T) {
	tests := []struct {
		name       string
		format     OutputFormat
		setupFunc  func(*Generator)
		wantErr    bool
		checkFunc  func([]byte) error
	}{
		{
			name:   "YAML format",
			format: FormatYAML,
			setupFunc: func(g *Generator) {
				// Add a simple path to make output non-empty
				pathItem := &v3.PathItem{
					Post: &v3.Operation{
						OperationId: "test",
						Summary:     "Test operation",
					},
				}
				g.doc.Paths.PathItems.Set("/test", pathItem)
			},
			wantErr: false,
			checkFunc: func(data []byte) error {
				// Check that it's valid YAML by unmarshaling
				var result interface{}
				if err := yaml.Unmarshal(data, &result); err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:   "JSON format",
			format: FormatJSON,
			setupFunc: func(g *Generator) {
				// Add a simple path to make output non-empty
				pathItem := &v3.PathItem{
					Post: &v3.Operation{
						OperationId: "test",
						Summary:     "Test operation",
					},
				}
				g.doc.Paths.PathItems.Set("/test", pathItem)
			},
			wantErr: false,
			checkFunc: func(data []byte) error {
				// Check that it looks like JSON (starts with '{' and ends with '}')
				str := strings.TrimSpace(string(data))
				if !strings.HasPrefix(str, "{") || !strings.HasSuffix(str, "}") {
					t.Errorf("Output doesn't look like JSON: %s", str[:min(100, len(str))])
				}
				return nil
			},
		},
		{
			name:   "Default format (falls back to YAML)",
			format: OutputFormat("invalid"),
			setupFunc: func(g *Generator) {
				pathItem := &v3.PathItem{
					Post: &v3.Operation{
						OperationId: "test",
						Summary:     "Test operation",
					},
				}
				g.doc.Paths.PathItems.Set("/test", pathItem)
			},
			wantErr: false,
			checkFunc: func(data []byte) error {
				var result interface{}
				if err := yaml.Unmarshal(data, &result); err != nil {
					return err
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.format)
			if tt.setupFunc != nil {
				tt.setupFunc(gen)
			}

			data, err := gen.Render()

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(data) == 0 {
				t.Error("Render() returned empty data")
			}

			if tt.checkFunc != nil {
				if err := tt.checkFunc(data); err != nil {
					t.Errorf("Output validation failed: %v", err)
				}
			}
		})
	}
}

// Test buildObjectSchema method
func TestBuildObjectSchema(t *testing.T) {
	gen := NewGenerator(FormatYAML)

	// Create a mock message with various field types
	message := createMockMessage("TestMessage", []mockField{
		{
			name:     "string_field",
			jsonName: "stringField",
			kind:     protoreflect.StringKind,
			comment:  "A string field",
		},
		{
			name:     "int_field", 
			jsonName: "intField",
			kind:     protoreflect.Int32Kind,
		},
		{
			name:     "bool_field",
			jsonName: "boolField", 
			kind:     protoreflect.BoolKind,
		},
	})

	// Add leading comment to message
	message.Comments.Leading = protogen.Comments("Test message for schema building")

	schema := gen.buildObjectSchema(message)

	if schema == nil {
		t.Fatal("buildObjectSchema returned nil")
	}

	// Get the actual schema
	actualSchema := schema.Schema()
	if actualSchema == nil {
		t.Fatal("Schema proxy contains nil schema")
	}

	// Check schema type
	if len(actualSchema.Type) != 1 || actualSchema.Type[0] != "object" {
		t.Errorf("Expected schema type [object], got %v", actualSchema.Type)
	}

	// Check properties are set
	if actualSchema.Properties == nil {
		t.Error("Schema properties is nil")
	}

	expectedProperties := []string{"stringField", "intField", "boolField"}
	if actualSchema.Properties.Len() != len(expectedProperties) {
		t.Errorf("Expected %d properties, got %d", len(expectedProperties), actualSchema.Properties.Len())
	}

	// Check each property exists
	for _, propName := range expectedProperties {
		if prop := actualSchema.Properties.GetOrZero(propName); prop == nil {
			t.Errorf("Expected property %s not found", propName)
		}
	}

	// Check description was set from comments
	if actualSchema.Description != "Test message for schema building" {
		t.Errorf("Expected description from comments, got %q", actualSchema.Description)
	}
}

// === Mock helpers ===

type mockField struct {
	name     string
	jsonName string
	kind     protoreflect.Kind
	comment  string
	list     bool
	optional bool
}

type mockMethod struct {
	name   string
	input  *protogen.Message
	output *protogen.Message
}

func createMockMessage(name string, fields []mockField) *protogen.Message {
	// Create mock descriptor
	msgDesc := &descriptorpb.DescriptorProto{
		Name: proto.String(name),
	}

	// Create mock message
	message := &protogen.Message{
		Desc: &mockMessageDesc{
			name:   name,
			fields: fields,
		},
		Fields: make([]*protogen.Field, len(fields)),
	}

	// Create mock fields
	for i, f := range fields {
		message.Fields[i] = &protogen.Field{
			Desc: &mockFieldDesc{
				name:     f.name,
				jsonName: f.jsonName,
				kind:     f.kind,
				list:     f.list,
				optional: f.optional,
			},
			Comments: protogen.Comments(f.comment),
		}
	}

	return message
}

func createMockService(name string, methods []mockMethod) *protogen.Service {
	service := &protogen.Service{
		Desc: &mockServiceDesc{
			name: name,
		},
		Methods: make([]*protogen.Method, len(methods)),
	}

	for i, m := range methods {
		service.Methods[i] = &protogen.Method{
			Desc: &mockMethodDesc{
				name: m.name,
			},
			Input:  m.input,
			Output: m.output,
		}
	}

	return service
}

// === Mock descriptor implementations ===

type mockMessageDesc struct {
	name   string
	fields []mockField
}

func (m *mockMessageDesc) Name() protoreflect.Name         { return protoreflect.Name(m.name) }
func (m *mockMessageDesc) FullName() protoreflect.FullName { return protoreflect.FullName(m.name) }
func (m *mockMessageDesc) IsPlaceholder() bool            { return false }
func (m *mockMessageDesc) Options() protoreflect.ProtoMessage {
	return &descriptorpb.MessageOptions{}
}
func (m *mockMessageDesc) Index() int                                { return 0 }
func (m *mockMessageDesc) Syntax() protoreflect.Syntax              { return protoreflect.Proto3 }
func (m *mockMessageDesc) IsMapEntry() bool                         { return false }
func (m *mockMessageDesc) Fields() protoreflect.FieldDescriptors    { return nil }
func (m *mockMessageDesc) Oneofs() protoreflect.OneofDescriptors    { return nil }
func (m *mockMessageDesc) ReservedNames() protoreflect.Names        { return nil }
func (m *mockMessageDesc) ReservedRanges() protoreflect.FieldRanges { return nil }
func (m *mockMessageDesc) RequiredNumbers() protoreflect.FieldNumbers {
	return nil
}
func (m *mockMessageDesc) ExtensionRanges() protoreflect.FieldRanges { return nil }
func (m *mockMessageDesc) ExtensionRangeOptions(int) protoreflect.ProtoMessage {
	return nil
}
func (m *mockMessageDesc) Messages() protoreflect.MessageDescriptors { return nil }
func (m *mockMessageDesc) Enums() protoreflect.EnumDescriptors       { return nil }
func (m *mockMessageDesc) Extensions() protoreflect.ExtensionDescriptors {
	return nil
}
func (m *mockMessageDesc) Parent() protoreflect.Descriptor   { return nil }
func (m *mockMessageDesc) ParentFile() protoreflect.FileDescriptor { return nil }

type mockFieldDesc struct {
	name     string
	jsonName string
	kind     protoreflect.Kind
	list     bool
	optional bool
	isMap    bool
	number   protoreflect.FieldNumber
}

func (f *mockFieldDesc) Name() protoreflect.Name         { return protoreflect.Name(f.name) }
func (f *mockFieldDesc) FullName() protoreflect.FullName { return protoreflect.FullName(f.name) }
func (f *mockFieldDesc) IsPlaceholder() bool            { return false }
func (f *mockFieldDesc) Options() protoreflect.ProtoMessage {
	return &descriptorpb.FieldOptions{}
}
func (f *mockFieldDesc) Index() int                            { return 0 }
func (f *mockFieldDesc) Syntax() protoreflect.Syntax          { return protoreflect.Proto3 }
func (f *mockFieldDesc) Number() protoreflect.FieldNumber {
	if f.number != 0 {
		return f.number
	}
	return 1
}
func (f *mockFieldDesc) Cardinality() protoreflect.Cardinality { return protoreflect.Optional }
func (f *mockFieldDesc) Kind() protoreflect.Kind              { return f.kind }
func (f *mockFieldDesc) HasJSONName() bool                    { return f.jsonName != "" }
func (f *mockFieldDesc) JSONName() string {
	if f.jsonName != "" {
		return f.jsonName
	}
	return f.name
}
func (f *mockFieldDesc) TextName() string                     { return f.name }
func (f *mockFieldDesc) HasPresence() bool                    { return f.optional }
func (f *mockFieldDesc) IsExtension() bool                    { return false }
func (f *mockFieldDesc) IsWeak() bool                         { return false }
func (f *mockFieldDesc) IsPacked() bool                       { return false }
func (f *mockFieldDesc) IsList() bool                         { return f.list }
func (f *mockFieldDesc) IsMap() bool                          { return f.isMap }
func (f *mockFieldDesc) MapKey() protoreflect.FieldDescriptor { return nil }
func (f *mockFieldDesc) MapValue() protoreflect.FieldDescriptor { return nil }
func (f *mockFieldDesc) HasDefault() bool                      { return false }
func (f *mockFieldDesc) Default() protoreflect.Value           { return protoreflect.Value{} }
func (f *mockFieldDesc) DefaultEnumValue() protoreflect.EnumValueDescriptor { return nil }
func (f *mockFieldDesc) ContainingOneof() protoreflect.OneofDescriptor { return nil }
func (f *mockFieldDesc) ContainingMessage() protoreflect.MessageDescriptor { return nil }
func (f *mockFieldDesc) Enum() protoreflect.EnumDescriptor      { return nil }
func (f *mockFieldDesc) Message() protoreflect.MessageDescriptor { return nil }
func (f *mockFieldDesc) Parent() protoreflect.Descriptor       { return nil }
func (f *mockFieldDesc) ParentFile() protoreflect.FileDescriptor { return nil }
func (f *mockFieldDesc) HasOptionalKeyword() bool              { return f.optional }

type mockServiceDesc struct {
	name string
}

func (s *mockServiceDesc) Name() protoreflect.Name         { return protoreflect.Name(s.name) }
func (s *mockServiceDesc) FullName() protoreflect.FullName { return protoreflect.FullName(s.name) }
func (s *mockServiceDesc) IsPlaceholder() bool            { return false }
func (s *mockServiceDesc) Options() protoreflect.ProtoMessage {
	return &descriptorpb.ServiceOptions{}
}
func (s *mockServiceDesc) Index() int                     { return 0 }
func (s *mockServiceDesc) Syntax() protoreflect.Syntax   { return protoreflect.Proto3 }
func (s *mockServiceDesc) Methods() protoreflect.MethodDescriptors { return nil }
func (s *mockServiceDesc) Parent() protoreflect.Descriptor         { return nil }
func (s *mockServiceDesc) ParentFile() protoreflect.FileDescriptor { return nil }

type mockMethodDesc struct {
	name string
}

func (m *mockMethodDesc) Name() protoreflect.Name         { return protoreflect.Name(m.name) }
func (m *mockMethodDesc) FullName() protoreflect.FullName { return protoreflect.FullName(m.name) }
func (m *mockMethodDesc) IsPlaceholder() bool            { return false }
func (m *mockMethodDesc) Options() protoreflect.ProtoMessage {
	return &descriptorpb.MethodOptions{}
}
func (m *mockMethodDesc) Index() int                          { return 0 }
func (m *mockMethodDesc) Syntax() protoreflect.Syntax        { return protoreflect.Proto3 }
func (m *mockMethodDesc) IsStreamingClient() bool            { return false }
func (m *mockMethodDesc) IsStreamingServer() bool            { return false }
func (m *mockMethodDesc) Input() protoreflect.MessageDescriptor  { return nil }
func (m *mockMethodDesc) Output() protoreflect.MessageDescriptor { return nil }
func (m *mockMethodDesc) Parent() protoreflect.Descriptor        { return nil }
func (m *mockMethodDesc) ParentFile() protoreflect.FileDescriptor { return nil }

// Helper function for min (not available in older Go versions)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}