package main

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/tscommon"
	"github.com/SebastienMelki/sebuf/internal/tsservergen"
)

func main() {
	options := protogen.Options{}
	// protogen.Options.New errors on any parameter it doesn't recognize unless a
	// ParamFunc is registered. paths/module are handled internally and never
	// reach this func, so validate ts_runtime here and reject anything else —
	// a bad key or value must fail loudly rather than silently fall back.
	options.ParamFunc = func(name, value string) error {
		if name != "ts_runtime" {
			return fmt.Errorf("unknown parameter %q", name)
		}
		if value != "protobuf-es" && value != "hand-rolled" {
			return fmt.Errorf("invalid ts_runtime %q (want protobuf-es or hand-rolled)", value)
		}
		return nil
	}

	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		runtime := tscommon.ParseMessageRuntime(plugin.Request.GetParameter())
		gen := tsservergen.New(plugin, runtime)
		return gen.Generate()
	})
}
