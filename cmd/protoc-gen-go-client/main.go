package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/clientgen"
)

func main() {
	var flags flag.FlagSet
	var jsonNaming string
	flags.StringVar(
		&jsonNaming,
		"json_naming",
		"",
		"JSON naming for request bodies: camel_case (default) or snake_case",
	)

	options := protogen.Options{
		ParamFunc: flags.Set,
	}

	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		opts := clientgen.Options{
			JSONNaming: jsonNaming,
		}
		gen := clientgen.NewWithOptions(plugin, opts)
		return gen.Generate()
	})
}
