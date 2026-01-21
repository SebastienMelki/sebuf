package main

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/clientgen"
)

func main() {
	options := protogen.Options{}

	options.Run(func(plugin *protogen.Plugin) error {
		gen := clientgen.New(plugin)
		return gen.Generate()
	})
}
