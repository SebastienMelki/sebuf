package main

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/tsclientgen"
	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

func main() {
	options := protogen.Options{}

	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		opts, err := tscommon.ParseOptions(plugin.Request.GetParameter())
		if err != nil {
			return err
		}
		gen := tsclientgen.New(plugin, opts)
		return gen.Generate()
	})
}
