package openapiv3

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	yaml "go.yaml.in/yaml/v4"
	"google.golang.org/protobuf/compiler/protogen"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// OutputFormat represents the output format for the OpenAPI document.
type OutputFormat string

const (
	FormatYAML OutputFormat = "yaml"
	FormatJSON OutputFormat = "json"
)

// HTTP method constants (lowercase for OpenAPI).
const (
	httpMethodGet    = "get"
	httpMethodPost   = "post"
	httpMethodPut    = "put"
	httpMethodDelete = "delete"
	httpMethodPatch  = "patch"
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

	// Add built-in error schemas (Error, ValidationError, FieldViolation)
	addBuiltinErrorSchemas(schemas)

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

// ProcessMessage processes a single message and adds it to the OpenAPI schemas.
// This is now exported to be called from main.go.
func (g *Generator) ProcessMessage(message *protogen.Message) {
	g.processMessage(message)
}

// Format returns the output format of the generator.
func (g *Generator) Format() OutputFormat {
	return g.format
}

// Doc returns the OpenAPI document.
func (g *Generator) Doc() *v3.Document {
	return g.doc
}

// Schemas returns the schemas map.
func (g *Generator) Schemas() *orderedmap.Map[string, *base.SchemaProxy] {
	return g.schemas
}

// ProcessService processes a single service and adds its paths to the OpenAPI document.
// This is now exported to be called from main.go.
func (g *Generator) ProcessService(service *protogen.Service) {
	// Update document info with service name
	g.doc.Info.Title = fmt.Sprintf("%s API", service.Desc.Name())

	// Process the service
	g.processService(service)
}

// CollectReferencedMessages recursively collects all messages referenced by a service.
// This includes input/output messages and all their nested field types.
func (g *Generator) CollectReferencedMessages(service *protogen.Service) {
	// Track processed messages to avoid infinite recursion
	processed := make(map[string]bool)

	// Collect messages from all methods
	for _, method := range service.Methods {
		g.collectMessageRecursive(method.Input, processed)
		g.collectMessageRecursive(method.Output, processed)
	}
}

// collectMessageRecursive recursively processes a message and all its dependencies.
func (g *Generator) collectMessageRecursive(message *protogen.Message, processed map[string]bool) {
	if message == nil {
		return
	}

	// Use the fully qualified name as the key to avoid duplicates
	key := string(message.Desc.FullName())
	if processed[key] {
		return
	}
	processed[key] = true

	// Process this message
	g.processMessage(message)

	// Process all field types
	for _, field := range message.Fields {
		if field.Message != nil {
			// Recursively process message fields
			g.collectMessageRecursive(field.Message, processed)
		}

		// For maps, the value type might be a message
		if field.Desc.IsMap() && field.Message != nil {
			// Map entry messages have a value field (field 2)
			for _, mapField := range field.Message.Fields {
				if mapField.Desc.Number() == 2 && mapField.Message != nil {
					g.collectMessageRecursive(mapField.Message, processed)
				}
			}
		}
	}

	// Process nested messages
	for _, nested := range message.Messages {
		g.collectMessageRecursive(nested, processed)
	}
}

// getSchemaName generates a schema name for a protobuf message.
// Since each service generates its own OpenAPI file, we can use simple message names
// without package prefixes to avoid collisions.
func (g *Generator) getSchemaName(message *protogen.Message) string {
	return string(message.Desc.Name())
}

// processMessage converts a protobuf message to an OpenAPI schema.
func (g *Generator) processMessage(message *protogen.Message) {
	schema := g.buildObjectSchema(message)
	schemaName := g.getSchemaName(message)
	g.schemas.Set(schemaName, schema)

	// Process nested messages recursively
	for _, nested := range message.Messages {
		g.processMessage(nested)
	}
}

// buildObjectSchema creates an OpenAPI object schema from a protobuf message.
//

func (g *Generator) buildObjectSchema(message *protogen.Message) *base.SchemaProxy {
	// Check for root-level unwrap
	if rootUnwrap := getRootUnwrapInfo(message); rootUnwrap != nil {
		return g.buildRootUnwrapSchema(message, rootUnwrap)
	}

	// Check if message has flatten fields -- use allOf for clean representation
	if annotations.HasFlattenFields(message) {
		return g.buildFlattenedObjectSchema(message)
	}

	// Check if message has discriminated oneofs
	if annotations.HasOneofDiscriminator(message) {
		return g.buildOneofDiscriminatorSchema(message)
	}

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

// buildOneofDiscriminatorSchema creates an OpenAPI schema for messages with discriminated oneofs.
// For flattened mode: generates per-variant schemas with oneOf + discriminator.
// For non-flattened mode: object schema with discriminator property and oneOf for variant routing.
//

func (g *Generator) buildOneofDiscriminatorSchema(message *protogen.Message) *base.SchemaProxy {
	// Collect discriminated oneofs
	var discriminatedOneofs []*annotations.OneofDiscriminatorInfo
	for _, oneof := range message.Oneofs {
		info := annotations.GetOneofDiscriminatorInfo(oneof)
		if info != nil {
			discriminatedOneofs = append(discriminatedOneofs, info)
		}
	}

	// Build set of fields that belong to discriminated oneofs
	oneofFields := make(map[string]bool)
	for _, info := range discriminatedOneofs {
		for _, variant := range info.Variants {
			oneofFields[string(variant.Field.Desc.Name())] = true
		}
	}

	// Check if any discriminated oneofs are flattened
	hasFlattenedOneof := false
	for _, info := range discriminatedOneofs {
		if info.Flatten {
			hasFlattenedOneof = true
			break
		}
	}

	if hasFlattenedOneof {
		return g.buildFlattenedOneofSchema(message, discriminatedOneofs, oneofFields)
	}
	return g.buildNestedOneofSchema(message, discriminatedOneofs, oneofFields)
}

// buildFlattenedOneofSchema generates per-variant schemas with oneOf + discriminator.
// Each variant schema includes common fields + discriminator field + variant's own fields.
func (g *Generator) buildFlattenedOneofSchema(
	message *protogen.Message,
	discriminatedOneofs []*annotations.OneofDiscriminatorInfo,
	oneofFields map[string]bool,
) *base.SchemaProxy {
	msgName := g.getSchemaName(message)

	// For each flattened oneof, generate per-variant schemas
	var allVariantRefs []*base.SchemaProxy
	var allMappings []*annotations.OneofDiscriminatorInfo

	for _, info := range discriminatedOneofs {
		if !info.Flatten {
			continue
		}

		for _, variant := range info.Variants {
			variantSchemaName := fmt.Sprintf("%s_%s", msgName, variant.DiscriminatorVal)

			// Build variant schema: common fields + discriminator + variant fields
			variantProps := orderedmap.New[string, *base.SchemaProxy]()
			var variantRequired []string

			// Add common (non-oneof) fields
			for _, field := range message.Fields {
				if oneofFields[string(field.Desc.Name())] {
					continue
				}
				fieldSchema := g.convertField(field)
				fieldName := field.Desc.JSONName()
				variantProps.Set(fieldName, fieldSchema)
			}

			// Add discriminator field
			discSchema := &base.Schema{
				Type: []string{"string"},
				Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Value: variant.DiscriminatorVal}},
			}
			variantProps.Set(info.Discriminator, base.CreateSchemaProxy(discSchema))
			variantRequired = append(variantRequired, info.Discriminator)

			// Add variant's message fields (flattened to parent level)
			if variant.IsMessage {
				for _, childField := range variant.Field.Message.Fields {
					childSchema := g.convertField(childField)
					variantProps.Set(childField.Desc.JSONName(), childSchema)
				}
			}

			variantSchema := &base.Schema{
				Type:       []string{"object"},
				Properties: variantProps,
			}
			if len(variantRequired) > 0 {
				variantSchema.Required = variantRequired
			}

			// Register the variant schema in components
			g.schemas.Set(variantSchemaName, base.CreateSchemaProxy(variantSchema))

			// Add ref to variant schema
			ref := fmt.Sprintf("#/components/schemas/%s", variantSchemaName)
			allVariantRefs = append(allVariantRefs, base.CreateSchemaProxyRef(ref))
		}

		allMappings = append(allMappings, info)
	}

	// Build main schema with oneOf + discriminator
	schema := &base.Schema{
		OneOf: allVariantRefs,
	}

	// Build discriminator with mapping (use first flattened oneof's discriminator)
	if len(allMappings) > 0 {
		info := allMappings[0]
		mapping := orderedmap.New[string, string]()
		for _, variant := range info.Variants {
			variantSchemaName := fmt.Sprintf("%s_%s", msgName, variant.DiscriminatorVal)
			mapping.Set(variant.DiscriminatorVal, fmt.Sprintf("#/components/schemas/%s", variantSchemaName))
		}
		schema.Discriminator = &base.Discriminator{
			PropertyName: info.Discriminator,
			Mapping:      mapping,
		}
	}

	// Add description from message comments
	if message.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(message.Comments.Leading))
	}

	return base.CreateSchemaProxy(schema)
}

// buildNestedOneofSchema generates an object schema with discriminator property
// and oneOf for non-flattened variant routing.
func (g *Generator) buildNestedOneofSchema(
	message *protogen.Message,
	discriminatedOneofs []*annotations.OneofDiscriminatorInfo,
	oneofFields map[string]bool,
) *base.SchemaProxy {
	properties := orderedmap.New[string, *base.SchemaProxy]()
	var required []string

	// Add non-oneof fields
	for _, field := range message.Fields {
		if oneofFields[string(field.Desc.Name())] {
			continue
		}
		fieldSchema := g.convertField(field)
		fieldName := field.Desc.JSONName()
		properties.Set(fieldName, fieldSchema)

		if checkIfFieldRequired(field) {
			required = append(required, fieldName)
		}
	}

	// For each non-flattened discriminated oneof, add discriminator property and oneOf
	var oneOfSchemas []*base.SchemaProxy
	var discInfo *annotations.OneofDiscriminatorInfo

	for _, info := range discriminatedOneofs {
		discInfo = info

		// Add discriminator property with enum of all possible values
		var enumValues []*yaml.Node
		for _, variant := range info.Variants {
			enumValues = append(enumValues, &yaml.Node{Kind: yaml.ScalarNode, Value: variant.DiscriminatorVal})
		}
		discSchema := &base.Schema{
			Type: []string{"string"},
			Enum: enumValues,
		}
		properties.Set(info.Discriminator, base.CreateSchemaProxy(discSchema))

		// Build oneOf with per-variant schemas (each wraps variant under its field name)
		for _, variant := range info.Variants {
			variantProps := orderedmap.New[string, *base.SchemaProxy]()

			if variant.IsMessage {
				fieldJSONName := variant.Field.Desc.JSONName()
				ref := fmt.Sprintf("#/components/schemas/%s", string(variant.Field.Message.Desc.Name()))
				variantProps.Set(fieldJSONName, base.CreateSchemaProxyRef(ref))
			} else {
				fieldJSONName := variant.Field.Desc.JSONName()
				scalarSchema := g.convertScalarField(variant.Field)
				variantProps.Set(fieldJSONName, scalarSchema)
			}

			variantSchema := &base.Schema{
				Type:       []string{"object"},
				Properties: variantProps,
			}
			oneOfSchemas = append(oneOfSchemas, base.CreateSchemaProxy(variantSchema))
		}
	}

	schema := &base.Schema{
		Type:       []string{"object"},
		Properties: properties,
	}

	if len(required) > 0 {
		schema.Required = required
	}

	if len(oneOfSchemas) > 0 {
		schema.OneOf = oneOfSchemas
	}

	// Add discriminator mapping
	if discInfo != nil {
		mapping := orderedmap.New[string, string]()
		for _, variant := range discInfo.Variants {
			if variant.IsMessage {
				mapping.Set(
					variant.DiscriminatorVal,
					fmt.Sprintf("#/components/schemas/%s", string(variant.Field.Message.Desc.Name())),
				)
			}
		}
		schema.Discriminator = &base.Discriminator{
			PropertyName: discInfo.Discriminator,
			Mapping:      mapping,
		}
	}

	// Add description from comments
	if message.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(message.Comments.Leading))
	}

	return base.CreateSchemaProxy(schema)
}

// buildFlattenedObjectSchema creates an OpenAPI schema using allOf for messages with flatten fields.
// Non-flattened fields go into one object schema, each flattened field's children go into
// separate object schemas with prefixed property names.
func (g *Generator) buildFlattenedObjectSchema(message *protogen.Message) *base.SchemaProxy {
	var allOfSchemas []*base.SchemaProxy

	// First: collect non-flattened fields into a base object schema
	baseProps := orderedmap.New[string, *base.SchemaProxy]()
	var baseRequired []string

	for _, field := range message.Fields {
		if annotations.IsFlattenField(field) {
			continue
		}

		fieldSchema := g.convertField(field)
		fieldName := field.Desc.JSONName()
		baseProps.Set(fieldName, fieldSchema)

		if checkIfFieldRequired(field) {
			baseRequired = append(baseRequired, fieldName)
		}
	}

	if baseProps.Len() > 0 {
		baseSchema := &base.Schema{
			Type:       []string{"object"},
			Properties: baseProps,
		}
		if len(baseRequired) > 0 {
			baseSchema.Required = baseRequired
		}
		allOfSchemas = append(allOfSchemas, base.CreateSchemaProxy(baseSchema))
	}

	// Second: for each flattened field, create an object schema with prefixed properties
	for _, field := range message.Fields {
		if !annotations.IsFlattenField(field) || field.Message == nil {
			continue
		}

		prefix := annotations.GetFlattenPrefix(field)
		flatProps := orderedmap.New[string, *base.SchemaProxy]()

		for _, childField := range field.Message.Fields {
			childSchema := g.convertField(childField)
			flattenedName := prefix + childField.Desc.JSONName()
			flatProps.Set(flattenedName, childSchema)
		}

		flatSchema := &base.Schema{
			Type:       []string{"object"},
			Properties: flatProps,
		}
		if prefix != "" {
			flatSchema.Description = fmt.Sprintf("Flattened from %s with prefix %q", field.Desc.Name(), prefix)
		} else {
			flatSchema.Description = fmt.Sprintf("Flattened from %s", field.Desc.Name())
		}
		allOfSchemas = append(allOfSchemas, base.CreateSchemaProxy(flatSchema))
	}

	schema := &base.Schema{
		AllOf: allOfSchemas,
	}

	// Add description from message comments
	if message.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(message.Comments.Leading))
	}

	return base.CreateSchemaProxy(schema)
}

// rootUnwrapInfo holds information about a root unwrap field.
type rootUnwrapInfo struct {
	field        *protogen.Field
	isMap        bool
	valueMessage *protogen.Message // For maps: the value message type
	valueUnwrap  *protogen.Field   // For maps: if value has unwrap field
}

// getRootUnwrapInfo checks if a message has root-level unwrap and returns info about it.
// Root unwrap requires exactly one field with unwrap=true on a map or repeated field.
func getRootUnwrapInfo(message *protogen.Message) *rootUnwrapInfo {
	// Root unwrap requires exactly one field
	if len(message.Fields) != 1 {
		return nil
	}

	field := message.Fields[0]
	if !annotations.HasUnwrapAnnotation(field) {
		return nil
	}

	// Must be a map or repeated field
	isMap := field.Desc.IsMap()
	isList := field.Desc.IsList()
	if !isMap && !isList {
		return nil
	}

	info := &rootUnwrapInfo{
		field: field,
		isMap: isMap,
	}

	// For maps, get value message and check for nested unwrap
	if isMap {
		valueField := getMapValueField(field)
		if valueField != nil && valueField.Message != nil {
			info.valueMessage = valueField.Message
			// Check if value has unwrap
			if unwrapField := annotations.FindUnwrapField(valueField.Message); unwrapField != nil {
				info.valueUnwrap = unwrapField
			}
		}
	}

	return info
}

// buildRootUnwrapSchema creates a schema for a root-level unwrap message.
func (g *Generator) buildRootUnwrapSchema(message *protogen.Message, rootUnwrap *rootUnwrapInfo) *base.SchemaProxy {
	var schema *base.Schema

	if rootUnwrap.isMap {
		// Root map unwrap: type=object with additionalProperties
		schema = g.buildRootMapUnwrapSchema(rootUnwrap)
	} else {
		// Root repeated unwrap: type=array
		itemSchema := g.convertScalarField(rootUnwrap.field)
		schema = &base.Schema{
			Type: []string{"array"},
			Items: &base.DynamicValue[*base.SchemaProxy, bool]{
				A: itemSchema,
			},
		}
	}

	// Add description from message comments
	if message.Comments.Leading != "" {
		schema.Description = strings.TrimSpace(string(message.Comments.Leading))
	}

	return base.CreateSchemaProxy(schema)
}

// buildRootMapUnwrapSchema builds the schema for a root map unwrap.
func (g *Generator) buildRootMapUnwrapSchema(rootUnwrap *rootUnwrapInfo) *base.Schema {
	schema := &base.Schema{
		Type: []string{"object"},
	}

	// Determine the additionalProperties schema
	switch {
	case rootUnwrap.valueUnwrap != nil:
		// Combined unwrap: map values are unwrapped arrays
		schema.AdditionalProperties = g.createUnwrapArraySchema(rootUnwrap.valueUnwrap)
	case rootUnwrap.valueMessage != nil:
		// Map with message values
		schemaRef := fmt.Sprintf("#/components/schemas/%s", g.getSchemaName(rootUnwrap.valueMessage))
		schema.AdditionalProperties = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxyRef(schemaRef),
		}
	default:
		// Map with scalar values
		schema.AdditionalProperties = g.buildScalarAdditionalProperties(rootUnwrap)
	}

	return schema
}

// buildScalarAdditionalProperties builds additionalProperties for scalar map values.
func (g *Generator) buildScalarAdditionalProperties(
	rootUnwrap *rootUnwrapInfo,
) *base.DynamicValue[*base.SchemaProxy, bool] {
	valueField := getMapValueField(rootUnwrap.field)
	if valueField != nil {
		return &base.DynamicValue[*base.SchemaProxy, bool]{
			A: g.convertScalarField(valueField),
		}
	}
	return &base.DynamicValue[*base.SchemaProxy, bool]{B: true}
}

// processService converts a protobuf service to OpenAPI paths.
func (g *Generator) processService(service *protogen.Service) {
	for _, method := range service.Methods {
		g.processMethod(service, method)
	}
}

// methodHTTPInfo holds extracted HTTP configuration for a method.
type methodHTTPInfo struct {
	path       string
	httpMethod string
	pathParams []string
}

// extractMethodHTTPInfo extracts HTTP configuration from service and method annotations.
func extractMethodHTTPInfo(service *protogen.Service, method *protogen.Method) methodHTTPInfo {
	servicePath := annotations.GetServiceBasePath(service)
	methodConfig := annotations.GetMethodHTTPConfig(method)

	var path, httpMethod string
	var pathParams []string

	if servicePath != "" || methodConfig != nil {
		methodPath := ""

		if methodConfig != nil {
			methodPath = methodConfig.Path
			// Shared annotations return UPPERCASE methods; OpenAPI requires lowercase
			httpMethod = strings.ToLower(methodConfig.Method)
			pathParams = methodConfig.PathParams
		}

		path = annotations.BuildHTTPPath(servicePath, methodPath)
	} else {
		path = fmt.Sprintf("/%s/%s", service.Desc.Name(), method.Desc.Name())
	}

	if httpMethod == "" {
		httpMethod = httpMethodPost
	}

	return methodHTTPInfo{path: path, httpMethod: httpMethod, pathParams: pathParams}
}

// buildPathParameters creates OpenAPI path parameters from path variable names.
func (g *Generator) buildPathParameters(method *protogen.Method, pathParams []string) []*v3.Parameter {
	var parameters []*v3.Parameter
	for _, paramName := range pathParams {
		field := findFieldByName(method.Input, paramName)
		pathParam := &v3.Parameter{
			Name:     paramName,
			In:       "path",
			Required: proto.Bool(true),
		}
		if field != nil {
			pathParam.Schema = g.createFieldSchema(field)
			pathParam.Description = strings.TrimSpace(string(field.Comments.Leading))
		} else {
			pathParam.Schema = base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}})
		}
		parameters = append(parameters, pathParam)
	}
	return parameters
}

// buildQueryParameters creates OpenAPI query parameters from method input.
func (g *Generator) buildQueryParameters(method *protogen.Method) []*v3.Parameter {
	var parameters []*v3.Parameter
	queryParams := annotations.GetQueryParams(method.Input)
	for _, qp := range queryParams {
		queryParam := &v3.Parameter{
			Name:     qp.ParamName,
			In:       "query",
			Required: &qp.Required,
		}
		if qp.Field != nil {
			queryParam.Schema = g.createFieldSchema(qp.Field)
			queryParam.Description = strings.TrimSpace(string(qp.Field.Comments.Leading))
		} else {
			queryParam.Schema = base.CreateSchemaProxy(&base.Schema{Type: []string{"string"}})
		}
		parameters = append(parameters, queryParam)
	}
	return parameters
}

// buildResponses creates the standard response map for an operation.
func (g *Generator) buildResponses(method *protogen.Method) *orderedmap.Map[string, *v3.Response] {
	responses := orderedmap.New[string, *v3.Response]()

	// Success response
	outputSchemaRef := fmt.Sprintf("#/components/schemas/%s", g.getSchemaName(method.Output))
	successResponse := &v3.Response{
		Description: "Successful response",
		Content:     orderedmap.New[string, *v3.MediaType](),
	}
	successResponse.Content.Set("application/json", &v3.MediaType{
		Schema: base.CreateSchemaProxyRef(outputSchemaRef),
	})
	responses.Set("200", successResponse)

	// Validation error response
	validationErrorResponse := &v3.Response{
		Description: "Validation error",
		Content:     orderedmap.New[string, *v3.MediaType](),
	}
	validationErrorResponse.Content.Set("application/json", &v3.MediaType{
		Schema: base.CreateSchemaProxyRef("#/components/schemas/ValidationError"),
	})
	responses.Set("400", validationErrorResponse)

	// Default error response - references the Error component schema
	// which matches the sebuf.http.Error proto message (single "message" field)
	errorResponse := &v3.Response{
		Description: "Error response",
		Content:     orderedmap.New[string, *v3.MediaType](),
	}
	errorResponse.Content.Set("application/json", &v3.MediaType{
		Schema: base.CreateSchemaProxyRef("#/components/schemas/Error"),
	})
	responses.Set("default", errorResponse)

	return responses
}

// assignOperationToPathItem assigns an operation to the correct HTTP method on a path item.
func assignOperationToPathItem(pathItem *v3.PathItem, httpMethod string, operation *v3.Operation) {
	switch httpMethod {
	case httpMethodGet:
		pathItem.Get = operation
	case httpMethodPost:
		pathItem.Post = operation
	case httpMethodPut:
		pathItem.Put = operation
	case httpMethodDelete:
		pathItem.Delete = operation
	case httpMethodPatch:
		pathItem.Patch = operation
	default:
		pathItem.Post = operation
	}
}

// processMethod converts a protobuf RPC method to an OpenAPI operation.
func (g *Generator) processMethod(service *protogen.Service, method *protogen.Method) {
	info := extractMethodHTTPInfo(service, method)

	operation := &v3.Operation{
		OperationId: string(method.Desc.Name()),
		Summary:     string(method.Desc.Name()),
		Tags:        []string{string(service.Desc.Name())},
	}

	if method.Comments.Leading != "" {
		operation.Description = strings.TrimSpace(string(method.Comments.Leading))
	}

	// Build parameters
	var parameters []*v3.Parameter
	allHeaders := annotations.CombineHeaders(
		annotations.GetServiceHeaders(service),
		annotations.GetMethodHeaders(method),
	)
	if len(allHeaders) > 0 {
		parameters = convertHeadersToParameters(allHeaders)
	}
	parameters = append(parameters, g.buildPathParameters(method, info.pathParams)...)
	parameters = append(parameters, g.buildQueryParameters(method)...)

	if len(parameters) > 0 {
		operation.Parameters = parameters
	}

	// Add request body for POST, PUT, PATCH
	if info.httpMethod == httpMethodPost || info.httpMethod == httpMethodPut || info.httpMethod == httpMethodPatch {
		inputSchemaRef := fmt.Sprintf("#/components/schemas/%s", g.getSchemaName(method.Input))
		operation.RequestBody = &v3.RequestBody{
			Required: proto.Bool(true),
			Content:  orderedmap.New[string, *v3.MediaType](),
		}
		operation.RequestBody.Content.Set("application/json", &v3.MediaType{
			Schema: base.CreateSchemaProxyRef(inputSchemaRef),
		})
	}

	operation.Responses = &v3.Responses{Codes: g.buildResponses(method)}

	// Add to path items
	existingPathItem, exists := g.doc.Paths.PathItems.Get(info.path)
	if !exists {
		existingPathItem = &v3.PathItem{}
	}
	assignOperationToPathItem(existingPathItem, info.httpMethod, operation)
	g.doc.Paths.PathItems.Set(info.path, existingPathItem)
}

// findFieldByName finds a field in a message by its proto name.
func findFieldByName(message *protogen.Message, fieldName string) *protogen.Field {
	for _, field := range message.Fields {
		if string(field.Desc.Name()) == fieldName {
			return field
		}
	}
	return nil
}

// createFieldSchema creates an OpenAPI schema for a protobuf field.
func (g *Generator) createFieldSchema(field *protogen.Field) *base.SchemaProxy {
	schema := &base.Schema{}

	switch field.Desc.Kind().String() {
	case headerTypeString:
		schema.Type = []string{headerTypeString}
	case headerTypeInt32, "sint32", "sfixed32":
		schema.Type = []string{headerTypeInteger}
		schema.Format = headerTypeInt32
	case headerTypeInt64, "sint64", "sfixed64":
		// Per proto3 JSON spec, int64 serializes as a string
		schema.Type = []string{headerTypeString}
		schema.Format = headerTypeInt64
	case "uint32", "fixed32":
		schema.Type = []string{headerTypeInteger}
		schema.Format = headerTypeInt32
	case "uint64", "fixed64":
		// Per proto3 JSON spec, uint64 serializes as a string
		schema.Type = []string{headerTypeString}
		schema.Format = headerTypeUint64
	case "bool":
		schema.Type = []string{"boolean"}
	case headerTypeFloat:
		schema.Type = []string{headerTypeNumber}
		schema.Format = headerTypeFloat
	case headerTypeDouble:
		schema.Type = []string{headerTypeNumber}
		schema.Format = headerTypeDouble
	default:
		schema.Type = []string{headerTypeString}
	}

	return base.CreateSchemaProxy(schema)
}

// addBuiltinErrorSchemas adds the Error, ValidationError, and FieldViolation schemas to the components.
// These schemas match the proto definitions in proto/sebuf/http/errors.proto.
func addBuiltinErrorSchemas(schemas *orderedmap.Map[string, *base.SchemaProxy]) {
	// Add Error schema - matches sebuf.http.Error proto message
	// Error has a single "message" field (string)
	errorProps := orderedmap.New[string, *base.SchemaProxy]()
	errorProps.Set("message", base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"string"},
		Description: "Error message (e.g., 'user not found', 'database connection failed')",
	}))

	errorSchema := base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"object"},
		Description: "Error is returned when a handler encounters an error. It contains a simple error message that the developer can customize.",
		Properties:  errorProps,
	})
	schemas.Set("Error", errorSchema)

	// Add FieldViolation schema - matches sebuf.http.FieldViolation proto message
	fieldViolationProps := orderedmap.New[string, *base.SchemaProxy]()
	fieldViolationProps.Set("field", base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"string"},
		Description: "The field path that failed validation (e.g., 'user.email' for nested fields). For header validation, this will be the header name (e.g., 'X-API-Key')",
	}))
	fieldViolationProps.Set("description", base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"string"},
		Description: "Human-readable description of the validation violation (e.g., 'must be a valid email address', 'required field missing')",
	}))

	fieldViolationSchema := base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"object"},
		Description: "FieldViolation describes a single validation error for a specific field.",
		Properties:  fieldViolationProps,
		Required:    []string{"field", "description"},
	})
	schemas.Set("FieldViolation", fieldViolationSchema)

	// Add ValidationError schema - matches sebuf.http.ValidationError proto message
	validationErrorProps := orderedmap.New[string, *base.SchemaProxy]()
	validationErrorProps.Set("violations", base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"array"},
		Description: "List of validation violations",
		Items: &base.DynamicValue[*base.SchemaProxy, bool]{
			A: base.CreateSchemaProxyRef("#/components/schemas/FieldViolation"),
		},
	}))

	validationErrorSchema := base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"object"},
		Description: "ValidationError is returned when request validation fails. It contains a list of field violations describing what went wrong.",
		Properties:  validationErrorProps,
		Required:    []string{"violations"},
	})
	schemas.Set("ValidationError", validationErrorSchema)
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
