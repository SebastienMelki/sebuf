package main

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/tsclientgen"
)

func main() {
	options := protogen.Options{}

	options.Run(func(plugin *protogen.Plugin) error {
		gen := tsclientgen.New(plugin)
		return gen.Generate()
	})
}
