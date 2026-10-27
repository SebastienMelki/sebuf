package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/pyclientgen"
)

func main() {
	var flags flag.FlagSet
	var fieldNames string
	flags.StringVar(&fieldNames, "field_names", "json",
		"JSON key naming in to_dict/from_dict: json (lowerCamelCase json_name) or proto (the proto field name)")
	options := protogen.Options{ParamFunc: flags.Set}

	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		gen := pyclientgen.NewWithOptions(plugin, pyclientgen.Options{FieldNames: fieldNames})
		return gen.Generate()
	})
}
