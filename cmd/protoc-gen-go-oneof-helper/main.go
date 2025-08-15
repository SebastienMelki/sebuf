// cmd/protoc-gen-go-oneof-helper/main.go
package main

import (
	"io"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/oneofhelper"
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

	// Process with protogen helper
	opts := protogen.Options{}

	plugin, err := opts.New(&req)
	if err != nil {
		panic(err)
	}

	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}

		oneofhelper.GenerateHelpers(plugin, file)
	}

	// Write response to stdout
	resp := plugin.Response()

	output, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}

	if _, writeErr := os.Stdout.Write(output); writeErr != nil {
		panic(writeErr)
	}
}
