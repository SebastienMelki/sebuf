// cmd/protoc-gen-swift-http/main.go
package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/swifthttpgen"
)

func main() {
	var flags flag.FlagSet
	options := protogen.Options{
		ParamFunc: flags.Set,
	}

	options.Run(func(plugin *protogen.Plugin) error {
		gen := swifthttpgen.New(plugin)
		return gen.Generate()
	})

	// // Read request from stdin
	// input, err := io.ReadAll(os.Stdin)
	// if err != nil {
	// 	panic(err)
	// }

	// var req pluginpb.CodeGeneratorRequest
	// if unmarshalErr := proto.Unmarshal(input, &req); unmarshalErr != nil {
	// 	panic(unmarshalErr)
	// }

	// // Process with protogen helper
	// opts := protogen.Options{}

	// plugin, err := opts.New(&req)
	// if err != nil {
	// 	panic(err)
	// }

	// generator := swifthttpgen.New(plugin)
	// if err := generator.Generate(); err != nil {
	// 	panic(err)
	// }

	// // Write response to stdout
	// resp := plugin.Response()

	// output, err := proto.Marshal(resp)
	// if err != nil {
	// 	panic(err)
	// }

	// if _, writeErr := os.Stdout.Write(output); writeErr != nil {
	// 	panic(writeErr)
	// }
}
