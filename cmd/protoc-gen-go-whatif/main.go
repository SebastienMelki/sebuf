package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/whatif"
)

func main() {
	var flags flag.FlagSet
	var apiKey string
	var model string
	var debug bool

	flags.StringVar(&apiKey, "openrouter_api_key", "", "OpenRouter API key for LLM scenario generation")
	flags.StringVar(&model, "model", "anthropic/claude-3-haiku", "Model to use for scenario generation (default: anthropic/claude-3-haiku)")
	flags.BoolVar(&debug, "debug", false, "Enable debug output")

	options := protogen.Options{
		ParamFunc: flags.Set,
	}

	options.Run(func(plugin *protogen.Plugin) error {
		gen := whatif.New(plugin, whatif.Options{
			OpenRouterAPIKey: apiKey,
			Model:            model,
			Debug:            debug,
		})
		return gen.Generate()
	})
}