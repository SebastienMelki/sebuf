// cmd/protoc-gen-go-oneof-helper/main.go
package main

import (
	"io"
	"os"

	"github.com/SebastienMelki/sebuf/internal/oneofhelper"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	// Read request from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var req pluginpb.CodeGeneratorRequest
	if err := proto.Unmarshal(input, &req); err != nil {
		panic(err)
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
	os.Stdout.Write(output)
}
