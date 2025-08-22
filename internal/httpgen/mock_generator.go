package httpgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// generateMockFile generates a mock server implementation file.
func (g *Generator) generateMockFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_http_mock.pb.go"
	gf := g.plugin.NewGeneratedFile(filename, file.GoImportPath)

	g.writeHeader(gf, file)

	// Imports
	gf.P("import (")
	gf.P(`"context"`)
	gf.P(`cryptorand "crypto/rand"`)
	gf.P(`"fmt"`)
	gf.P(`"math/rand"`)
	gf.P(`"strconv"`)
	gf.P(`"time"`)
	gf.P()
	gf.P(`"google.golang.org/protobuf/proto"`)
	gf.P(")")
	gf.P()

	// Generate field examples storage
	if err := g.generateFieldExamplesStorage(gf, file); err != nil {
		return err
	}

	// Generate mock servers for each service
	for _, service := range file.Services {
		if err := g.generateMockService(gf, file, service); err != nil {
			return err
		}
	}

	// Generate helper functions
	g.generateMockHelpers(gf)

	return nil
}

// generateFieldExamplesStorage generates storage for field examples.
func (g *Generator) generateFieldExamplesStorage(gf *protogen.GeneratedFile, file *protogen.File) error {
	gf.P("// Field examples extracted from proto definitions")
	gf.P("var fieldExamples = map[string][]string{")

	// Collect all field examples from all messages
	for _, message := range file.Messages {
		g.collectMessageFieldExamples(gf, message, "")
	}

	gf.P("}")
	gf.P()

	return nil
}

// collectMessageFieldExamples recursively collects field examples.
func (g *Generator) collectMessageFieldExamples(gf *protogen.GeneratedFile, message *protogen.Message, prefix string) {
	messagePath := prefix + string(message.Desc.Name())

	for _, field := range message.Fields {
		examples := getFieldExamples(field)
		if len(examples) > 0 {
			fieldPath := messagePath + "." + string(field.Desc.Name())
			gf.P(`"`, fieldPath, `": {`)
			for _, example := range examples {
				gf.P(`"`, example, `",`)
			}
			gf.P("},")
		}
	}

	// Process nested messages
	for _, nested := range message.Messages {
		g.collectMessageFieldExamples(gf, nested, messagePath+".")
	}
}

// generateMockService generates a mock implementation for a service.
func (g *Generator) generateMockService(
	gf *protogen.GeneratedFile,
	_ *protogen.File,
	service *protogen.Service,
) error {
	serviceName := service.GoName

	// Mock server struct
	gf.P("// Mock", serviceName, "Server is a mock implementation of ", serviceName, "Server.")
	gf.P("type Mock", serviceName, "Server struct {")
	gf.P("// Add any mock-specific fields here")
	gf.P("}")
	gf.P()

	// Constructor
	gf.P("// NewMock", serviceName, "Server creates a new mock server for ", serviceName, ".")
	gf.P("func NewMock", serviceName, "Server() *Mock", serviceName, "Server {")
	gf.P("return &Mock", serviceName, "Server{}")
	gf.P("}")
	gf.P()

	// Generate mock methods
	for _, method := range service.Methods {
		if err := g.generateMockMethod(gf, service, method); err != nil {
			return err
		}
	}

	return nil
}

// generateMockMethod generates a mock implementation for an RPC method.
func (g *Generator) generateMockMethod(
	gf *protogen.GeneratedFile,
	service *protogen.Service,
	method *protogen.Method,
) error {
	methodName := method.GoName
	inputType := method.Input.GoIdent
	outputType := method.Output.GoIdent

	gf.P("// ", methodName, " is a mock implementation of ", service.GoName, "Server.", methodName, ".")
	gf.P(
		"func (m *Mock",
		service.GoName,
		"Server) ",
		methodName,
		"(ctx context.Context, req *",
		inputType,
		") (*",
		outputType,
		", error) {",
	)

	// Validate request
	gf.P("// Validate the request")
	gf.P("if msg, ok := any(req).(proto.Message); ok {")
	gf.P("if err := ValidateMessage(msg); err != nil {")
	gf.P("return nil, err")
	gf.P("}")
	gf.P("}")
	gf.P()

	// Generate response
	gf.P("// Generate mock response")
	gf.P("resp := &", outputType, "{}")
	gf.P()

	// Fill response fields
	g.generateMockFieldAssignments(gf, method.Output, "resp")

	gf.P("return resp, nil")
	gf.P("}")
	gf.P()

	return nil
}

// generateMockFieldAssignments generates field assignments for a message.
func (g *Generator) generateMockFieldAssignments(
	gf *protogen.GeneratedFile,
	message *protogen.Message,
	varName string,
) {
	messageName := string(message.Desc.Name())

	for _, field := range message.Fields {
		fieldName := field.GoName
		fieldPath := messageName + "." + string(field.Desc.Name())

		// Generate assignment based on field type
		switch field.Desc.Kind() {
		case protoreflect.StringKind:
			gf.P(
				varName,
				".",
				fieldName,
				" = selectStringExample(\"",
				fieldPath,
				"\", ",
				g.getDefaultGenerator(field),
				")",
			)
		case protoreflect.Int32Kind, protoreflect.Int64Kind:
			gf.P(varName, ".", fieldName, " = selectIntExample(\"", fieldPath, "\", ", g.getDefaultValue(field), ")")
		case protoreflect.BoolKind:
			gf.P(varName, ".", fieldName, " = selectBoolExample(\"", fieldPath, "\", ", g.getDefaultValue(field), ")")
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			gf.P(varName, ".", fieldName, " = selectFloatExample(\"", fieldPath, "\", ", g.getDefaultValue(field), ")")
		case protoreflect.MessageKind:
			if field.Desc.IsList() {
				gf.P("// TODO: Handle repeated message field ", fieldName)
			} else {
				gf.P(varName, ".", fieldName, " = &", field.Message.GoIdent, "{}")
				g.generateMockFieldAssignments(gf, field.Message, varName+"."+fieldName)
			}
		case protoreflect.EnumKind,
			protoreflect.Sint32Kind,
			protoreflect.Uint32Kind,
			protoreflect.Sint64Kind,
			protoreflect.Uint64Kind,
			protoreflect.Sfixed32Kind,
			protoreflect.Fixed32Kind,
			protoreflect.Sfixed64Kind,
			protoreflect.Fixed64Kind,
			protoreflect.BytesKind,
			protoreflect.GroupKind:
			gf.P("// TODO: Handle field ", fieldName, " of type ", field.Desc.Kind())
		default:
			gf.P("// TODO: Handle field ", fieldName, " of type ", field.Desc.Kind())
		}
	}
}

// getDefaultGenerator returns a function name for generating default values.
func (g *Generator) getDefaultGenerator(field *protogen.Field) string {
	fieldName := strings.ToLower(string(field.Desc.Name()))

	// Check for common field names
	switch {
	case strings.Contains(fieldName, "id"):
		return "generateUUID"
	case strings.Contains(fieldName, "email"):
		return "generateEmail"
	case strings.Contains(fieldName, "name"):
		return "generateName"
	case strings.Contains(fieldName, "phone"):
		return "generatePhone"
	case strings.Contains(fieldName, "address"):
		return "generateAddress"
	case strings.Contains(fieldName, "url"):
		return "generateURL"
	default:
		return "generateString"
	}
}

// getDefaultValue returns a default value for a field.
func (g *Generator) getDefaultValue(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.Int32Kind, protoreflect.Int64Kind:
		return "42"
	case protoreflect.BoolKind:
		return "true"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "3.14"
	case protoreflect.EnumKind,
		protoreflect.Sint32Kind,
		protoreflect.Uint32Kind,
		protoreflect.Sint64Kind,
		protoreflect.Uint64Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Fixed32Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Fixed64Kind,
		protoreflect.StringKind,
		protoreflect.BytesKind,
		protoreflect.MessageKind,
		protoreflect.GroupKind:
		return `""`
	default:
		return `""`
	}
}

// generateMockHelpers generates helper functions for mock data generation.
func (g *Generator) generateMockHelpers(gf *protogen.GeneratedFile) {
	g.generateExampleSelectors(gf)
	g.generateDefaultGenerators(gf)
	g.generateInitFunction(gf)
}

// generateExampleSelectors generates functions to select examples from predefined values.
func (g *Generator) generateExampleSelectors(gf *protogen.GeneratedFile) {
	// String example selector
	gf.P("// selectStringExample selects a random example or generates a default value.")
	gf.P("func selectStringExample(fieldPath string, defaultGenerator func() string) string {")
	gf.P("if examples, ok := fieldExamples[fieldPath]; ok && len(examples) > 0 {")
	gf.P("return examples[rand.Intn(len(examples))]")
	gf.P("}")
	gf.P("return defaultGenerator()")
	gf.P("}")
	gf.P()

	// Int example selector
	gf.P("// selectIntExample selects a random example or returns a default value.")
	gf.P("func selectIntExample(fieldPath string, defaultValue int64) int64 {")
	gf.P("if examples, ok := fieldExamples[fieldPath]; ok && len(examples) > 0 {")
	gf.P("example := examples[rand.Intn(len(examples))]")
	gf.P("if v, err := strconv.ParseInt(example, 10, 64); err == nil {")
	gf.P("return v")
	gf.P("}")
	gf.P("}")
	gf.P("return defaultValue")
	gf.P("}")
	gf.P()

	// Bool example selector
	gf.P("// selectBoolExample selects a random example or returns a default value.")
	gf.P("func selectBoolExample(fieldPath string, defaultValue bool) bool {")
	gf.P("if examples, ok := fieldExamples[fieldPath]; ok && len(examples) > 0 {")
	gf.P("example := examples[rand.Intn(len(examples))]")
	gf.P("if v, err := strconv.ParseBool(example); err == nil {")
	gf.P("return v")
	gf.P("}")
	gf.P("}")
	gf.P("return defaultValue")
	gf.P("}")
	gf.P()

	// Float example selector
	gf.P("// selectFloatExample selects a random example or returns a default value.")
	gf.P("func selectFloatExample(fieldPath string, defaultValue float64) float64 {")
	gf.P("if examples, ok := fieldExamples[fieldPath]; ok && len(examples) > 0 {")
	gf.P("example := examples[rand.Intn(len(examples))]")
	gf.P("if v, err := strconv.ParseFloat(example, 64); err == nil {")
	gf.P("return v")
	gf.P("}")
	gf.P("}")
	gf.P("return defaultValue")
	gf.P("}")
	gf.P()
}

// generateDefaultGenerators generates default value generator functions.
func (g *Generator) generateDefaultGenerators(gf *protogen.GeneratedFile) {
	gf.P("// Default value generators")
	gf.P("func generateUUID() string {")
	gf.P("var b [16]byte")
	gf.P("_, err := cryptorand.Read(b[:])")
	gf.P("if err != nil {")
	gf.P(`return "550e8400-e29b-41d4-a716-446655440000" // fallback`)
	gf.P("}")
	gf.P("b[6] = (b[6] & 0x0f) | 0x40 // Version 4")
	gf.P("b[8] = (b[8] & 0x3f) | 0x80 // Variant bits")
	gf.P("return fmt.Sprintf(\"%x-%x-%x-%x-%x\", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])")
	gf.P("}")
	gf.P()

	gf.P("func generateEmail() string {")
	gf.P(`return "user@example.com"`)
	gf.P("}")
	gf.P()

	gf.P("func generateName() string {")
	gf.P(`names := []string{"Alice Johnson", "Bob Smith", "Charlie Davis", "Diana Wilson"}`)
	gf.P("return names[rand.Intn(len(names))]")
	gf.P("}")
	gf.P()

	gf.P("func generatePhone() string {")
	gf.P(`return "+1-555-0123"`)
	gf.P("}")
	gf.P()

	gf.P("func generateAddress() string {")
	gf.P(`return "123 Main Street, Anytown, USA"`)
	gf.P("}")
	gf.P()

	gf.P("func generateURL() string {")
	gf.P(`return "https://example.com"`)
	gf.P("}")
	gf.P()

	gf.P("func generateString() string {")
	gf.P(`return "example string"`)
	gf.P("}")
	gf.P()
}

// generateInitFunction generates the init function for seeding random number generator.
func (g *Generator) generateInitFunction(gf *protogen.GeneratedFile) {
	gf.P("func init() {")
	gf.P("rand.Seed(time.Now().UnixNano())")
	gf.P("}")
}
