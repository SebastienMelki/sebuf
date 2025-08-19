package openapiv3

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gopkg.in/yaml.v3"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/SebastienMelki/sebuf/http"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name   string
		format OutputFormat
	}{
		{
			name:   "YAML format",
			format: FormatYAML,
		},
		{
			name:   "JSON format",
			format: FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator(tt.format)
			
			assert.NotNil(t, g)
			assert.Equal(t, tt.format, g.format)
			assert.NotNil(t, g.doc)
			assert.NotNil(t, g.schemas)
			
			// Check document structure
			assert.Equal(t, "3.1.0", g.doc.Version)
			assert.NotNil(t, g.doc.Info)
			assert.Equal(t, "Generated API", g.doc.Info.Title)
			assert.Equal(t, "1.0.0", g.doc.Info.Version)
			assert.NotNil(t, g.doc.Paths)
			assert.NotNil(t, g.doc.Components)
			assert.Equal(t, g.schemas, g.doc.Components.Schemas)
		})
	}
}

func TestProcessFile(t *testing.T) {
	g := NewGenerator(FormatYAML)
	
	// Create a mock file with messages and services
	file := &protogen.File{
		Desc: &mockFileDescriptor{
			packageName: "test.api",
		},
		Messages: []*protogen.Message{
			createMockMessage("TestMessage"),
		},
		Services: []*protogen.Service{
			createMockService("TestService"),
		},
	}
	
	g.ProcessFile(file)
	
	// Check that the title was updated
	assert.Equal(t, "test.api API", g.doc.Info.Title)
	
	// Check that schema was added
	_, exists := g.schemas.Get("TestMessage")
	assert.True(t, exists)
}

func TestProcessMessage(t *testing.T) {
	g := NewGenerator(FormatYAML)
	
	// Create a message with fields
	message := &protogen.Message{
		Desc: &mockMessageDescriptor{
			name: "UserMessage",
		},
		Fields: []*protogen.Field{
			{
				Desc: &mockFieldDescriptor{
					jsonName: "name",
					kind:     protoreflect.StringKind,
				},
			},
			{
				Desc: &mockFieldDescriptor{
					jsonName: "age",
					kind:     protoreflect.Int32Kind,
				},
			},
		},
		Messages: []*protogen.Message{}, // No nested messages
		Comments: protogen.CommentSet{
			Leading: "User message for testing",
		},
	}
	
	g.processMessage(message)
	
	// Check that schema was added
	_, exists := g.schemas.Get("UserMessage")
	assert.True(t, exists)
	
	// Get the schema and verify its structure
	schemaProxy, _ := g.schemas.Get("UserMessage")
	schema := extractSchemaFromProxy(t, schemaProxy)
	
	assert.Equal(t, []string{"object"}, schema.Type)
	assert.NotNil(t, schema.Properties)
	_, hasName := schema.Properties.Get("name")
	assert.True(t, hasName)
	_, hasAge := schema.Properties.Get("age")
	assert.True(t, hasAge)
	assert.Equal(t, "User message for testing", schema.Description)
}

func TestProcessService(t *testing.T) {
	g := NewGenerator(FormatYAML)
	
	// Create test messages first
	g.processMessage(&protogen.Message{
		Desc: &mockMessageDescriptor{name: "CreateRequest"},
		Fields: []*protogen.Field{
			{
				Desc: &mockFieldDescriptor{
					jsonName: "name",
					kind:     protoreflect.StringKind,
				},
			},
		},
	})
	
	g.processMessage(&protogen.Message{
		Desc: &mockMessageDescriptor{name: "CreateResponse"},
		Fields: []*protogen.Field{
			{
				Desc: &mockFieldDescriptor{
					jsonName: "id",
					kind:     protoreflect.StringKind,
				},
			},
		},
	})
	
	// Create service with method
	service := &protogen.Service{
		Desc: &mockServiceDescriptor{
			name: "UserService",
			options: func() *descriptorpb.ServiceOptions {
				opts := &descriptorpb.ServiceOptions{}
				basePath := "/api/v1"
				proto.SetExtension(opts, http.E_ServiceConfig, &http.ServiceConfig{
					BasePath: &basePath,
				})
				return opts
			}(),
		},
		Methods: []*protogen.Method{
			{
				Desc: &mockMethodDescriptor{
					name: "CreateUser",
					options: func() *descriptorpb.MethodOptions {
						opts := &descriptorpb.MethodOptions{}
						path := "/users"
						proto.SetExtension(opts, http.E_Config, &http.HttpConfig{
							Path: &path,
						})
						return opts
					}(),
				},
				Input: &protogen.Message{
					Desc: &mockMessageDescriptor{name: "CreateRequest"},
				},
				Output: &protogen.Message{
					Desc: &mockMessageDescriptor{name: "CreateResponse"},
				},
				Comments: protogen.CommentSet{
					Leading: "Creates a new user",
				},
			},
		},
	}
	
	g.processService(service)
	
	// Check that path was added
	_, exists := g.doc.Paths.PathItems.Get("/api/v1/users")
	assert.True(t, exists)
	
	// Get the path item and verify its structure
	pathItem, _ := g.doc.Paths.PathItems.Get("/api/v1/users")
	assert.NotNil(t, pathItem.Post)
	assert.Equal(t, "CreateUser", pathItem.Post.OperationId)
	assert.Equal(t, "CreateUser", pathItem.Post.Summary)
	assert.Equal(t, "Creates a new user", pathItem.Post.Description)
	assert.Contains(t, pathItem.Post.Tags, "UserService")
}

func TestProcessMethod_WithHeaders(t *testing.T) {
	g := NewGenerator(FormatYAML)
	
	// Create test messages
	g.processMessage(&protogen.Message{
		Desc: &mockMessageDescriptor{name: "Request"},
	})
	g.processMessage(&protogen.Message{
		Desc: &mockMessageDescriptor{name: "Response"},
	})
	
	// Create service with headers
	service := &protogen.Service{
		Desc: &mockServiceDescriptor{
			name: "HeaderService",
			options: func() *descriptorpb.ServiceOptions {
				opts := &descriptorpb.ServiceOptions{}
				proto.SetExtension(opts, http.E_ServiceHeaders, &http.ServiceHeaders{
					RequiredHeaders: []*http.Header{
						{
							Name:        proto.String("X-API-Key"),
							Description: proto.String("API key"),
							Type:        proto.String("string"),
							Required:    true,
						},
					},
				})
				return opts
			}(),
		},
	}
	
	// Create method with additional headers
	method := &protogen.Method{
		Desc: &mockMethodDescriptor{
			name: "TestMethod",
			options: func() *descriptorpb.MethodOptions {
				opts := &descriptorpb.MethodOptions{}
				proto.SetExtension(opts, http.E_MethodHeaders, &http.MethodHeaders{
					RequiredHeaders: []*http.Header{
						{
							Name:        proto.String("X-Request-ID"),
							Description: proto.String("Request ID"),
							Type:        proto.String("string"),
							Format:      proto.String("uuid"),
							Required:    true,
						},
					},
				})
				return opts
			}(),
		},
		Input: &protogen.Message{
			Desc: &mockMessageDescriptor{name: "Request"},
		},
		Output: &protogen.Message{
			Desc: &mockMessageDescriptor{name: "Response"},
		},
	}
	
	g.processMethod(service, method)
	
	// Get the operation from the generated path
	pathItem, exists := g.doc.Paths.PathItems.Get("/HeaderService/TestMethod")
	require.True(t, exists)
	require.NotNil(t, pathItem.Post)
	
	// Check parameters (headers)
	assert.NotNil(t, pathItem.Post.Parameters)
	assert.Len(t, pathItem.Post.Parameters, 2) // Service header + method header
	
	// Verify header parameters
	paramNames := make(map[string]bool)
	for _, param := range pathItem.Post.Parameters {
		paramNames[param.Name] = true
		assert.Equal(t, "header", param.In)
	}
	assert.True(t, paramNames["X-API-Key"])
	assert.True(t, paramNames["X-Request-ID"])
}

func TestRender_YAML(t *testing.T) {
	g := NewGenerator(FormatYAML)
	
	// Add some content
	g.doc.Info.Title = "Test API"
	
	output, err := g.Render()
	require.NoError(t, err)
	require.NotNil(t, output)
	
	// Verify YAML structure
	var doc map[string]interface{}
	err = yaml.Unmarshal(output, &doc)
	require.NoError(t, err)
	
	assert.Equal(t, "3.1.0", doc["openapi"])
	info := doc["info"].(map[string]interface{})
	assert.Equal(t, "Test API", info["title"])
}

func TestRender_JSON(t *testing.T) {
	g := NewGenerator(FormatJSON)
	
	// Add some content
	g.doc.Info.Title = "Test API"
	
	output, err := g.Render()
	require.NoError(t, err)
	require.NotNil(t, output)
	
	// Verify JSON structure
	var doc map[string]interface{}
	err = json.Unmarshal(output, &doc)
	require.NoError(t, err)
	
	assert.Equal(t, "3.1.0", doc["openapi"])
	info := doc["info"].(map[string]interface{})
	assert.Equal(t, "Test API", info["title"])
}

func TestBuildObjectSchema_WithRequiredFields(t *testing.T) {
	g := NewGenerator(FormatYAML)
	
	// Create message with required field
	message := &protogen.Message{
		Desc: &mockMessageDescriptor{
			name: "RequiredFieldMessage",
		},
		Fields: []*protogen.Field{
			{
				Desc: &mockFieldDescriptor{
					jsonName: "required_field",
					kind:     protoreflect.StringKind,
					options: func() *descriptorpb.FieldOptions {
						opts := &descriptorpb.FieldOptions{}
						proto.SetExtension(opts, validate.E_Field, &validate.FieldRules{
							Required: proto.Bool(true),
						})
						return opts
					}(),
				},
			},
			{
				Desc: &mockFieldDescriptor{
					jsonName: "optional_field",
					kind:     protoreflect.StringKind,
				},
			},
		},
	}
	
	schemaProxy := g.buildObjectSchema(message)
	schema := extractSchemaFromProxy(t, schemaProxy)
	
	assert.Equal(t, []string{"object"}, schema.Type)
	assert.NotNil(t, schema.Properties)
	_, hasRequired := schema.Properties.Get("required_field")
	assert.True(t, hasRequired)
	_, hasOptional := schema.Properties.Get("optional_field")
	assert.True(t, hasOptional)
	
	// Check required fields
	assert.Contains(t, schema.Required, "required_field")
	assert.NotContains(t, schema.Required, "optional_field")
}

// Helper functions for creating mock objects

func createMockMessage(name string) *protogen.Message {
	return &protogen.Message{
		Desc: &mockMessageDescriptor{
			name: protoreflect.Name(name),
		},
		Fields:   []*protogen.Field{},
		Messages: []*protogen.Message{},
	}
}

func createMockService(name string) *protogen.Service {
	return &protogen.Service{
		Desc: &mockServiceDescriptor{
			name: protoreflect.Name(name),
		},
		Methods: []*protogen.Method{},
	}
}

// Mock file descriptor
type mockFileDescriptor struct {
	protoreflect.FileDescriptor
	packageName string
}

func (m *mockFileDescriptor) Package() protoreflect.FullName {
	return protoreflect.FullName(m.packageName)
}


