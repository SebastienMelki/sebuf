package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/tsclientgen"
)

func main() {
	var flags flag.FlagSet
	var fieldNames string
	flags.StringVar(&fieldNames, "field_names", "json", "TypeScript field naming: json or proto")

	options := protogen.Options{ParamFunc: flags.Set}

	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		gen := tsclientgen.New(plugin, tsclientgen.Options{FieldNames: fieldNames})
		return gen.Generate()
	})
}
