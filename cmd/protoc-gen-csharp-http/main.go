package main

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/csharpgen"
)

func main() {
	options, cfg := csharpgen.NewOptions()
	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		gen := csharpgen.New(plugin, *cfg)
		return gen.Generate()
	})
}
