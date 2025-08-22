package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/openapiv3"
)

func main() {
	// Read request from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var req pluginpb.CodeGeneratorRequest
	if unmarshalErr := proto.Unmarshal(input, &req); unmarshalErr != nil {
		panic(unmarshalErr)
	}

	// Parse parameters for output format
	format := openapiv3.FormatYAML // default to YAML
	if req.Parameter != nil {
		params := parseParameters(req.GetParameter())
		if f, ok := params["format"]; ok {
			switch f {
			case "json":
				format = openapiv3.FormatJSON
			case "yaml", "yml":
				format = openapiv3.FormatYAML
			}
		}
	}

	// Process with protogen helper
	opts := protogen.Options{}
	plugin, err := opts.New(&req)
	if err != nil {
		panic(err)
	}

	// Generate OpenAPI document for each service in all proto files
	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}
		
		// Process each service separately
		for _, service := range file.Services {
			// Create a new generator for each service
			generator := openapiv3.NewGenerator(format)
			
			// Process all messages from the file (needed for schemas)
			for _, message := range file.Messages {
				generator.ProcessMessage(message)
			}
			
			// Process this specific service
			generator.ProcessService(service)
			
			// Render the OpenAPI document for this service
			output, err := generator.Render()
			if err != nil {
				panic(err)
			}
			
			// Determine output filename based on service name and format
			ext := "yaml"
			if format == openapiv3.FormatJSON {
				ext = "json"
			}
			filename := fmt.Sprintf("%s.openapi.%s", service.Desc.Name(), ext)
			
			// Write to generated file
			generatedFile := plugin.NewGeneratedFile(filename, "")
			if _, writeErr := generatedFile.Write(output); writeErr != nil {
				panic(writeErr)
			}
		}
	}

	// Write response to stdout
	resp := plugin.Response()
	respOutput, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}

	if _, writeErr := os.Stdout.Write(respOutput); writeErr != nil {
		panic(writeErr)
	}
}

// parseParameters parses protoc plugin parameters in the format "key=value,key2=value2".
func parseParameters(parameter string) map[string]string {
	params := make(map[string]string)
	if parameter == "" {
		return params
	}

	pairs := strings.Split(parameter, ",")
	for _, pair := range pairs {
		const splitLimit = 2
		if kv := strings.SplitN(pair, "=", splitLimit); len(kv) == splitLimit {
			params[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return params
}
