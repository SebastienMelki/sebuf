package openapiv3

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"google.golang.org/protobuf/compiler/protogen"
	"gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

// OutputFormat represents the output format for the OpenAPI document.
type OutputFormat string

const (
	FormatYAML OutputFormat = "yaml"
	FormatJSON OutputFormat = "json"
)

// Generator generates OpenAPI v3.1 documents from Protocol Buffer definitions.
type Generator struct {
	doc     *v3.Document
	schemas *orderedmap.Map[string, *base.SchemaProxy]
	format  OutputFormat
}

// NewGenerator creates a new OpenAPI generator with the specified output format.
func NewGenerator(format OutputFormat) *Generator {
	schemas := orderedmap.New[string, *base.SchemaProxy]()

	return &Generator{
		format:  format,
		schemas: schemas,
		doc: &v3.Document{
			Version: "3.1.0",
			Info: &base.Info{
				Title:   "Generated API",
				Version: "1.0.0",
			},
			Paths: &v3.Paths{
				PathItems: orderedmap.New[string, *v3.PathItem](),
			},
			Components: &v3.Components{
				Schemas: schemas,
			},
		},
	}
}

// ProcessFile processes a single proto file and adds its definitions to the OpenAPI document.
func (g *Generator) ProcessFile(file *protogen.File) {
	// Update document info from the first file processed
	if g.doc.Info.Title == "Generated API" && file.Desc.Package() != "" {
		g.doc.Info.Title = fmt.Sprintf("%s API", file.Desc.Package())
	}

	// Process all messages to create schemas
	for _, message := range file.Messages {
		g.processMessage(message)
	}

	// Process all services to create paths
	for _, service := range file.Services {
		g.processService(service)
	}
}

// processMessage converts a protobuf message to an OpenAPI schema.
func (g *Generator) processMessage(message *protogen.Message) {
	schema := g.buildObjectSchema(message)
	schemaName := string(message.Desc.Name())
	g.schemas.Set(schemaName, schema)

	// Process nested messages recursively
	for _, nested := range message.Messages {
		g.processMessage(nested)
	}
}

// buildObjectSchema creates an OpenAPI object schema from a protobuf message.
func (g *Generator) buildObjectSchema(message *protogen.Message) *base.SchemaProxy {
	properties := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	for _, field := range message.Fields {
		fieldSchema := g.convertField(field)
		fieldName := field.Desc.JSONName()
		properties.Set(fieldName, fieldSchema)

		// Check if field has the required constraint from buf.validate
		if checkIfFieldRequired(field) {
			required = append(required, fieldName)
		}
	}

	schema := &base.Schema{
		Type:       []string{"object"},
		Properties: properties,
	}

	if len(required) > 0 {
		schema.Required = required
	}

	// Add description from comments
	if message.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(message.Comments.Leading))
	}

	return base.CreateSchemaProxy(schema)
}

// processService converts a protobuf service to OpenAPI paths.
func (g *Generator) processService(service *protogen.Service) {
	for _, method := range service.Methods {
		g.processMethod(service, method)
	}
}

// processMethod converts a protobuf RPC method to an OpenAPI operation.
func (g *Generator) processMethod(service *protogen.Service, method *protogen.Method) {
	// Extract HTTP configuration from annotations
	var path string
	serviceConfig := getServiceHTTPConfig(service)
	methodConfig := getMethodHTTPConfig(method)

	if serviceConfig != nil || methodConfig != nil {
		// Use sebuf.http annotations
		servicePath := ""
		methodPath := ""

		if serviceConfig != nil {
			servicePath = serviceConfig.BasePath
		}
		if methodConfig != nil {
			methodPath = methodConfig.Path
		}

		path = buildHTTPPath(servicePath, methodPath)
	} else {
		// Fallback to gRPC-style path
		path = fmt.Sprintf("/%s/%s", service.Desc.Name(), method.Desc.Name())
	}

	// Create operation
	operation := &v3.Operation{
		OperationId: string(method.Desc.Name()),
		Summary:     string(method.Desc.Name()),
		Tags:        []string{string(service.Desc.Name())},
	}

	// Add description from comments
	if method.Comments.Leading != "" {
		operation.Description = strings.TrimSpace(string(method.Comments.Leading))
	}

	// Extract and add header parameters
	serviceHeaders := getServiceHeaders(service)
	methodHeaders := getMethodHeaders(method)
	allHeaders := combineHeaders(serviceHeaders, methodHeaders)

	if len(allHeaders) > 0 {
		headerParameters := convertHeadersToParameters(allHeaders)
		operation.Parameters = headerParameters
	}

	// Add request body for the input message
	inputSchemaRef := fmt.Sprintf("#/components/schemas/%s", method.Input.Desc.Name())
	operation.RequestBody = &v3.RequestBody{
		Required: proto.Bool(true), // Convert bool to *bool
		Content:  orderedmap.New[string, *v3.MediaType](),
	}
	operation.RequestBody.Content.Set("application/json", &v3.MediaType{
		Schema: base.CreateSchemaProxyRef(inputSchemaRef),
	})

	// Add response for the output message
	outputSchemaRef := fmt.Sprintf("#/components/schemas/%s", method.Output.Desc.Name())
	responses := orderedmap.New[string, *v3.Response]()

	successResponse := &v3.Response{
		Description: "Successful response",
		Content:     orderedmap.New[string, *v3.MediaType](),
	}
	successResponse.Content.Set("application/json", &v3.MediaType{
		Schema: base.CreateSchemaProxyRef(outputSchemaRef),
	})
	responses.Set("200", successResponse)

	// Add default error response
	errorSchema := base.CreateSchemaProxy(&base.Schema{
		Type: []string{"object"},
		Properties: func() *orderedmap.Map[string, *base.SchemaProxy] {
			props := orderedmap.New[string, *base.SchemaProxy]()
			props.Set("error", base.CreateSchemaProxy(&base.Schema{
				Type: []string{"string"},
			}))
			props.Set("code", base.CreateSchemaProxy(&base.Schema{
				Type: []string{"integer"},
			}))
			return props
		}(),
	})

	errorResponse := &v3.Response{
		Description: "Error response",
		Content:     orderedmap.New[string, *v3.MediaType](),
	}
	errorResponse.Content.Set("application/json", &v3.MediaType{
		Schema: errorSchema,
	})
	responses.Set("default", errorResponse)

	operation.Responses = &v3.Responses{
		Codes: responses,
	}

	// Create path item and add to document
	pathItem := &v3.PathItem{
		Post: operation, // Default to POST for gRPC-style operations
	}

	g.doc.Paths.PathItems.Set(path, pathItem)
}

// Render outputs the OpenAPI document in the specified format.
func (g *Generator) Render() ([]byte, error) {
	switch g.format {
	case FormatJSON:
		// First marshal to YAML (which works correctly with libopenapi)
		yamlData, err := yaml.Marshal(g.doc)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
		}
		// Then convert YAML to JSON
		jsonData, err := k8syaml.YAMLToJSON(yamlData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
		}
		return jsonData, nil
	case FormatYAML:
		return yaml.Marshal(g.doc)
	default:
		return yaml.Marshal(g.doc)
	}
}
