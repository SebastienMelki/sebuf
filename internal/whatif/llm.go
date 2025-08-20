package whatif

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// LLMClient handles communication with OpenRouter for scenario generation.
type LLMClient struct {
	client *openai.Client
	model  string
	debug  bool
}

// NewLLMClient creates a new LLM client configured for OpenRouter.
func NewLLMClient(apiKey, model string) *LLMClient {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://openrouter.ai/api/v1"),
	)

	return &LLMClient{
		client: &client,
		model:  model,
		debug:  false,
	}
}

// SetDebug enables or disables debug output.
func (l *LLMClient) SetDebug(debug bool) {
	l.debug = debug
}


// ScenarioResponse is the structured response from the LLM.
type ScenarioResponse struct {
	Scenarios []ScenarioItem `json:"scenarios" jsonschema_description:"List of test scenarios"`
}

// ScenarioItem represents a single scenario in the response.
type ScenarioItem struct {
	Name         string `json:"name" jsonschema_description:"Snake case name like expired_token"`
	Description  string `json:"description" jsonschema_description:"Human readable description"`
	FunctionName string `json:"function_name" jsonschema_description:"Pascal case function name like ExpiredToken"`
	// Keep schema simple to avoid OpenAI validation issues
	// We'll parse proto field suggestions from the description
}

// ScenarioError represents an error scenario.
type ScenarioError struct {
	Code    int    `json:"code" jsonschema_description:"HTTP status code"`
	Message string `json:"message" jsonschema_description:"Error message"`
}

// FieldValueResponse is the response for the second LLM call to generate field values
type FieldValueResponse struct {
	FieldInstructions string `json:"field_instructions" jsonschema_description:"Field value instructions as formatted text"`
	IsError           bool   `json:"is_error" jsonschema_description:"Whether this is an error scenario"`
}

// GenerateSchema creates a JSON schema for the response type.
func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// Generate schema dynamically to ensure it's current
func getScenarioResponseSchema() interface{} {
	return GenerateSchema[ScenarioResponse]()
}

func getFieldValueResponseSchema() interface{} {
	return GenerateSchema[FieldValueResponse]()
}

// GenerateFieldValues is the second LLM call to generate specific proto field values
func (l *LLMClient) GenerateFieldValues(scenario Scenario, method *protogen.Method) (*FieldValueResponse, error) {
	prompt := l.buildFieldValuePrompt(scenario, method)
	
	if l.debug {
		fmt.Fprintf(os.Stderr, "Debug: Generating field values for scenario %s\n", scenario.Name)
	}

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "field_values",
		Description: openai.String("Proto field values for test scenario"),
		Schema:      getFieldValueResponseSchema(),
		Strict:      openai.Bool(true),
	}

	chat, err := l.client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are an expert at generating realistic test data for protobuf messages. Generate specific field values that match the scenario description."),
			openai.UserMessage(prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
		Model:       openai.ChatModel(l.model),
		Temperature: openai.Float(0.3), // Lower temperature for more consistent field values
		MaxTokens:   openai.Int(500),
	})

	if err != nil {
		if l.debug {
			fmt.Fprintf(os.Stderr, "Debug: Field value generation failed: %v\n", err)
		}
		return nil, fmt.Errorf("failed to generate field values: %w", err)
	}

	if l.debug && len(chat.Choices) > 0 {
		fmt.Fprintf(os.Stderr, "Debug: Field values response: %s\n", chat.Choices[0].Message.Content)
	}

	if len(chat.Choices) == 0 {
		return nil, fmt.Errorf("no field values response from LLM")
	}

	var response FieldValueResponse
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse field values response: %w", err)
	}

	return &response, nil
}

func (l *LLMClient) buildFieldValuePrompt(scenario Scenario, method *protogen.Method) string {
	responseStructure := l.describeMessageStructure(method.Output)
	
	return fmt.Sprintf(`Generate specific proto field values for this test scenario:

SCENARIO: %s
DESCRIPTION: %s

RESPONSE PROTO MESSAGE:
%s

Instructions:
1. If this is an error scenario, set is_error=true and provide field_instructions="ERROR: code message"
2. If this is a success scenario, set is_error=false and provide field_instructions with specific field values
3. Use format: "field1=value1, field2=value2, field3=value3"
4. Make values realistic and appropriate for the scenario
5. Use proper data types (strings in quotes, numbers as-is, booleans as true/false)

Examples:
- For "duplicate email" scenario: "ERROR: 409 Email already exists"  
- For "expired token" scenario: "accessToken=expired_abc123, expiresIn=-1, user=null"
- For "invalid email" scenario: "ERROR: 400 Invalid email format"
- For "successful login" scenario: "accessToken=valid_token_xyz, expiresIn=3600, user={id: user_123, name: John Doe}"

Focus on realistic test data that exercises the proto message properly.`,
		scenario.Name,
		scenario.Description,
		responseStructure)
}

const systemPrompt = `You are an expert at generating test scenarios for gRPC/HTTP APIs.
Given a service method definition, generate realistic "what if" scenarios that test edge cases, error conditions, and unusual behaviors.

Focus on:
1. Authentication/authorization failures
2. Validation errors
3. Resource states (not found, deleted, locked)
4. Rate limiting and quotas
5. Temporal issues (expired, not yet valid)
6. Partial failures
7. Data inconsistencies
8. Network issues and timeouts

Generate concise, focused scenarios. Each scenario should test one specific condition.
Use snake_case for names, PascalCase for function names.`

// GetCompletion uses the official OpenAI Go SDK with structured outputs
func (l *LLMClient) GetCompletion(ctx context.Context, userPrompt string) (*ScenarioResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if l.debug {
		fmt.Fprintf(os.Stderr, "Debug: Calling LLM with model %s\n", l.model)
		// Debug the schema
		schemaBytes, _ := json.MarshalIndent(getScenarioResponseSchema(), "", "  ")
		fmt.Fprintf(os.Stderr, "Debug: Schema: %s\n", string(schemaBytes))
	}

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "scenarios",
		Description: openai.String("Test scenarios for API methods"),
		Schema:      getScenarioResponseSchema(),
		Strict:      openai.Bool(true),
	}

	chat, err := l.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
		Model:       openai.ChatModel(l.model),
		Temperature: openai.Float(0.7),
		MaxTokens:   openai.Int(1000),
	})

	if err != nil {
		if l.debug {
			fmt.Fprintf(os.Stderr, "Debug: LLM call failed: %v\n", err)
		}
		return nil, fmt.Errorf("failed to generate scenarios: %w", err)
	}

	if l.debug {
		fmt.Fprintf(os.Stderr, "Debug: LLM call successful\n")
		if len(chat.Choices) > 0 {
			fmt.Fprintf(os.Stderr, "Debug: Response: %s\n", chat.Choices[0].Message.Content)
		}
	}

	if len(chat.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	var response ScenarioResponse
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GenerateServiceScenarios generates scenarios that apply to all methods in a service.
func (l *LLMClient) GenerateServiceScenarios(service *protogen.Service) ([]Scenario, error) {
	prompt := l.buildServicePrompt(service)
	
	if l.debug {
		fmt.Fprintf(os.Stderr, "Debug: Generating service scenarios for %s\n", service.GoName)
	}

	response, err := l.GetCompletion(context.Background(), prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service scenarios: %w", err)
	}

	scenarios := l.convertToScenarios(response, ServiceLevel, "")
	return scenarios, nil
}

// GenerateMethodScenarios generates scenarios for a specific method using two-step LLM approach.
func (l *LLMClient) GenerateMethodScenarios(service *protogen.Service, method *protogen.Method) ([]Scenario, error) {
	// Step 1: Generate abstract scenarios
	prompt := l.buildMethodPrompt(service, method)
	
	if l.debug {
		fmt.Fprintf(os.Stderr, "Debug: Generating method scenarios for %s.%s (Step 1: Abstract scenarios)\n", service.GoName, method.GoName)
	}

	response, err := l.GetCompletion(context.Background(), prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate method scenarios: %w", err)
	}

	scenarios := l.convertToScenarios(response, MethodLevel, method.GoName)
	
	// Step 2: Generate field values for each scenario
	for i := range scenarios {
		if l.debug {
			fmt.Fprintf(os.Stderr, "Debug: Generating field values for scenario %s (Step 2)\n", scenarios[i].Name)
		}
		
		fieldResponse, err := l.GenerateFieldValues(scenarios[i], method)
		if err != nil {
			if l.debug {
				fmt.Fprintf(os.Stderr, "Warning: Failed to generate field values for %s: %v\n", scenarios[i].Name, err)
			}
			// Continue with basic scenario if field generation fails
			continue
		}
		
		// Parse field instructions and update scenario
		l.applyFieldInstructions(&scenarios[i], fieldResponse)
	}
	
	return scenarios, nil
}

func (l *LLMClient) applyFieldInstructions(scenario *Scenario, fieldResponse *FieldValueResponse) {
	if fieldResponse.IsError {
		// Parse error instructions
		if errorMatch := regexp.MustCompile(`ERROR:\s*(\d+)\s+(.*)`).FindStringSubmatch(fieldResponse.FieldInstructions); len(errorMatch) > 2 {
			code, _ := strconv.Atoi(errorMatch[1])
			scenario.Error = &ErrorScenario{
				Code:    code,
				Message: strings.TrimSpace(errorMatch[2]),
			}
		}
	} else {
		// Parse field value instructions
		scenario.FieldValues = l.parseFieldValues(fieldResponse.FieldInstructions)
	}
}

func (l *LLMClient) convertToScenarios(response *ScenarioResponse, level ScenarioLevel, method string) []Scenario {
	var scenarios []Scenario
	
	for _, item := range response.Scenarios {
		scenario := Scenario{
			Name:         item.Name,
			Description:  item.Description,
			FunctionName: item.FunctionName,
			Level:        level,
			Method:       method,
			FieldValues:  nil, // Will be populated by second LLM call
			Error:        nil, // Will be populated by second LLM call
		}
		
		scenarios = append(scenarios, scenario)
	}
	
	return scenarios
}

func (l *LLMClient) parseFieldValues(fieldStr string) map[string]interface{} {
	fieldValues := make(map[string]interface{})
	
	// Split by comma and parse key=value pairs
	pairs := strings.Split(fieldStr, ",")
	for _, pair := range pairs {
		if kv := strings.SplitN(strings.TrimSpace(pair), "=", 2); len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			
			// Handle different value types
			switch {
			case value == "":
				fieldValues[key] = ""
			case value == "true" || value == "false":
				fieldValues[key] = value == "true"
			case regexp.MustCompile(`^-?\d+$`).MatchString(value):
				if intVal, err := strconv.Atoi(value); err == nil {
					fieldValues[key] = intVal
				}
			default:
				fieldValues[key] = value
			}
		}
	}
	
	return fieldValues
}

func (l *LLMClient) buildServicePrompt(service *protogen.Service) string {
	var methods []string
	for _, method := range service.Methods {
		methods = append(methods, method.GoName)
	}

	return fmt.Sprintf(`Generate service-level test scenarios for:

Service: %s
Methods: %s

Generate 3-4 scenarios that would affect ALL methods in this service.
Think about infrastructure issues, service-wide failures, and cross-cutting concerns.

MUST include these specific scenarios:
- slow_response → SlowResponse: All responses are slow due to system performance issues

Also generate 2-3 additional scenarios like:
- database down, maintenance mode, authentication service unavailable`,
		service.GoName,
		strings.Join(methods, ", "))
}

func (l *LLMClient) buildMethodPrompt(service *protogen.Service, method *protogen.Method) string {
	requestFields := l.describeMessage(method.Input)
	responseFields := l.describeMessage(method.Output)
	semantics := l.inferMethodSemantics(method.GoName)

	return fmt.Sprintf(`Generate method-specific test scenarios for:

Service: %s
Method: %s
Request: %s { %s }
Response: %s { %s }

Method semantics: %s

Generate 5-7 scenarios specific to this method's functionality:

MUST include these specific scenarios:
- %s_error → %sError: Generic error case for %s method
- %s_success → %sSuccess: Successful case with realistic response data

Also generate 4-5 additional realistic scenarios:
- SUCCESS scenarios (2): Happy path variations with different realistic data
- ERROR scenarios (3): Validation errors, authentication issues, business logic failures
- Examples: rate limiting, resource not found, permission denied, invalid formats

Balance between success scenarios (2-3) and error scenarios (3-4) for comprehensive testing.`,
		service.GoName,
		method.GoName,
		method.Input.GoIdent.GoName,
		requestFields,
		method.Output.GoIdent.GoName,
		responseFields,
		semantics,
		strings.ToLower(method.GoName),
		method.GoName,
		method.GoName,
		strings.ToLower(method.GoName),
		method.GoName)
}

func (l *LLMClient) describeMessageStructure(msg *protogen.Message) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("message %s {", msg.GoIdent.GoName))
	
	for _, field := range msg.Fields {
		fieldDesc := l.describeField(field)
		lines = append(lines, fmt.Sprintf("  %s", fieldDesc))
	}
	
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func (l *LLMClient) describeField(field *protogen.Field) string {
	var parts []string
	
	// Field type and name
	fieldType := l.getFieldType(field)
	if field.Desc.IsList() {
		fieldType = "repeated " + fieldType
	}
	if field.Desc.HasPresence() {
		fieldType = "optional " + fieldType
	}
	
	parts = append(parts, fmt.Sprintf("%s %s = %d", fieldType, field.Desc.Name(), field.Desc.Number()))
	
	// Add validation info if available
	if validation := l.extractValidation(field); validation != "" {
		parts = append(parts, fmt.Sprintf("// %s", validation))
	}
	
	return strings.Join(parts, " ")
}

func (l *LLMClient) extractValidation(field *protogen.Field) string {
	// Extract buf.validate rules from field options
	// This is simplified - in a real implementation you'd parse the actual validation options
	switch field.Desc.Kind() {
	case protoreflect.StringKind:
		if strings.Contains(string(field.Desc.Name()), "email") {
			return "must be valid email"
		}
		if strings.Contains(string(field.Desc.Name()), "id") {
			return "must be valid UUID"
		}
		if strings.Contains(string(field.Desc.Name()), "name") {
			return "min_len=2, max_len=100"
		}
		if strings.Contains(string(field.Desc.Name()), "password") {
			return "min_len=8"
		}
		if strings.Contains(string(field.Desc.Name()), "token") {
			return "min_len=10"
		}
	case protoreflect.Int32Kind, protoreflect.Int64Kind:
		if strings.Contains(string(field.Desc.Name()), "age") {
			return "gte=18, lte=120"
		}
		if strings.Contains(string(field.Desc.Name()), "expires") {
			return "timestamp in seconds"
		}
	}
	return ""
}

func (l *LLMClient) describeMessage(msg *protogen.Message) string {
	var fields []string
	for _, field := range msg.Fields {
		fieldType := l.getFieldType(field)
		fieldDesc := fmt.Sprintf("%s %s", field.GoName, fieldType)
		
		if field.Desc.HasPresence() {
			fieldDesc += " (optional)"
		}
		if field.Desc.IsList() {
			fieldDesc += " (repeated)"
		}
		
		fields = append(fields, fieldDesc)
	}
	return strings.Join(fields, ", ")
}

func (l *LLMClient) getFieldType(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.StringKind:
		return "string"
	case protoreflect.Int32Kind, protoreflect.Int64Kind:
		return "int"
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "float"
	case protoreflect.MessageKind:
		return field.Message.GoIdent.GoName
	case protoreflect.EnumKind:
		return field.Enum.GoIdent.GoName
	default:
		return "unknown"
	}
}

func (l *LLMClient) inferMethodSemantics(methodName string) string {
	lower := strings.ToLower(methodName)
	
	switch {
	case strings.Contains(lower, "login") || strings.Contains(lower, "auth"):
		return "Authentication method - likely returns tokens, session info, or authentication errors"
	case strings.Contains(lower, "create") || strings.Contains(lower, "add"):
		return "Creation method - creates new resources, may fail on duplicates or validation"
	case strings.Contains(lower, "get") || strings.Contains(lower, "fetch") || strings.Contains(lower, "read"):
		return "Retrieval method - fetches resources, may return not found errors"
	case strings.Contains(lower, "update") || strings.Contains(lower, "edit"):
		return "Update method - modifies existing resources, may fail on not found or validation"
	case strings.Contains(lower, "delete") || strings.Contains(lower, "remove"):
		return "Deletion method - removes resources, may fail on not found or dependencies"
	case strings.Contains(lower, "list") || strings.Contains(lower, "search"):
		return "Listing method - returns collections, may have pagination or filtering"
	case strings.Contains(lower, "verify") || strings.Contains(lower, "validate"):
		return "Validation method - checks validity, returns success/failure status"
	default:
		return "General method - standard operation with typical success/failure modes"
	}
}