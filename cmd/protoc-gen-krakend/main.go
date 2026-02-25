package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/krakendgen"
)

func main() {
	req := readRequest()
	plugin := createPlugin(req)
	generateFiles(plugin)
	writeResponse(plugin)
}

func readRequest() *pluginpb.CodeGeneratorRequest {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var req pluginpb.CodeGeneratorRequest
	if unmarshalErr := proto.Unmarshal(input, &req); unmarshalErr != nil {
		panic(unmarshalErr)
	}
	return &req
}

func createPlugin(req *pluginpb.CodeGeneratorRequest) *protogen.Plugin {
	opts := protogen.Options{}
	plugin, err := opts.New(req)
	if err != nil {
		panic(err)
	}
	return plugin
}

func generateFiles(plugin *protogen.Plugin) {
	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}
		processFileServices(plugin, file)
	}
}

func processFileServices(plugin *protogen.Plugin, file *protogen.File) {
	for _, service := range file.Services {
		writeServiceFile(plugin, service)
	}
}

func writeServiceFile(plugin *protogen.Plugin, service *protogen.Service) {
	endpoints, err := krakendgen.GenerateService(service)
	if err != nil {
		plugin.Error(err)
		return
	}

	// Ensure nil slice marshals as [] not null.
	if endpoints == nil {
		endpoints = []krakendgen.Endpoint{}
	}

	config := krakendgen.KrakenDConfig{
		Schema:    "https://www.krakend.io/schema/krakend.json",
		Version:   3,
		Endpoints: endpoints,
	}

	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		plugin.Error(fmt.Errorf("service %s: failed to marshal config: %w", service.Desc.Name(), err))
		return
	}

	filename := fmt.Sprintf("%s.krakend.json", service.Desc.Name())
	generatedFile := plugin.NewGeneratedFile(filename, "")
	if _, writeErr := generatedFile.Write(append(jsonBytes, '\n')); writeErr != nil {
		plugin.Error(fmt.Errorf("service %s: failed to write file: %w", service.Desc.Name(), writeErr))
	}
}

func writeResponse(plugin *protogen.Plugin) {
	resp := plugin.Response()
	resp.SupportedFeatures = proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL))

	respOutput, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}

	if _, writeErr := os.Stdout.Write(respOutput); writeErr != nil {
		panic(writeErr)
	}
}
