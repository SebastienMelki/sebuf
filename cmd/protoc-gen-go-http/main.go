package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/httpgen"
)

func main() {
	var flags flag.FlagSet
	options := protogen.Options{
		ParamFunc: flags.Set,
	}

	options.Run(func(plugin *protogen.Plugin) error {
		gen := httpgen.New(plugin)
		return gen.Generate()
	})
}
