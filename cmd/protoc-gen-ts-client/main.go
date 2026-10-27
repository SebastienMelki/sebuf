package main

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/SebastienMelki/sebuf/internal/tsclientgen"
	"github.com/SebastienMelki/sebuf/internal/tscommon"
)

func main() {
	var fieldNames string
	options := protogen.Options{}
	// protogen.Options.New errors on any parameter it doesn't recognize unless a
	// ParamFunc is registered. paths/module are handled internally and never
	// reach this func, so validate ts_runtime here and reject anything else —
	// a bad key or value must fail loudly rather than silently fall back.
	options.ParamFunc = func(name, value string) error {
		switch name {
		case "ts_runtime":
			if value != "protobuf-es" && value != "hand-rolled" {
				return fmt.Errorf("invalid ts_runtime %q (want protobuf-es or hand-rolled)", value)
			}
		case "ts_error_handling":
			if value != "throw" && value != "result" {
				return fmt.Errorf("invalid ts_error_handling %q (want throw or result)", value)
			}
		case "field_names":
			// Validated by the generator ("json" or "proto"), so the error message is
			// the same whether the generator is driven by the plugin or in-process.
			fieldNames = value
		default:
			return fmt.Errorf("unknown parameter %q", name)
		}
		return nil
	}

	options.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		param := plugin.Request.GetParameter()
		runtime := tscommon.ParseMessageRuntime(param)
		errorHandling := tscommon.ParseErrorHandling(param)
		if err := tscommon.ValidateRuntimeOptions(runtime, errorHandling); err != nil {
			return err
		}
		gen := tsclientgen.NewWithOptions(plugin, runtime, errorHandling, tsclientgen.Options{FieldNames: fieldNames})
		return gen.Generate()
	})
}
