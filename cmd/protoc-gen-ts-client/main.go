package main

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/tsclientgen"
	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

func main() {
	options := protogen.Options{}
	// protogen.Options.New errors on any parameter it doesn't recognize unless a
	// ParamFunc is registered. Accept ts_runtime here (it's parsed from the raw
	// parameter string below) so it isn't rejected as an unknown parameter.
	options.ParamFunc = func(_, _ string) error { return nil }

	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		runtime := tscommon.ParseMessageRuntime(plugin.Request.GetParameter())
		gen := tsclientgen.New(plugin, runtime)
		return gen.Generate()
	})
}
