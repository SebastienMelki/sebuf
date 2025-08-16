package main

import (
	"flag"
	"github.com/SebastienMelki/sebuf/internal/httpgen"
	"google.golang.org/protobuf/compiler/protogen"
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