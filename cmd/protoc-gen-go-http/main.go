package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/httpgen"
)

func main() {
	var flags flag.FlagSet
	var generateMock bool
	flags.BoolVar(&generateMock, "generate_mock", false, "generate mock server implementation")

	options := protogen.Options{
		ParamFunc: flags.Set,
	}

	options.Run(func(plugin *protogen.Plugin) error {
		opts := httpgen.Options{
			GenerateMock: generateMock,
		}
		gen := httpgen.NewWithOptions(plugin, opts)
		return gen.Generate()
	})
}
