package openapiv3

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// ValidateFiles rejects path and query parameters whose types cannot be represented
// in a URL, across every file the plugin was asked to generate.
//
// The OpenAPI generator has no other reason to fail, but it must reject the same
// protos as the five code generators: a message-kind query param used to be rendered
// as `type: string` here while every generator emitted something else, documenting a
// contract nothing implemented. See internal/annotations/url_params.go and issue #216.
func ValidateFiles(plugin *protogen.Plugin) error {
	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}
		if err := annotations.ValidateFileURLParams(file); err != nil {
			return err
		}
	}

	return nil
}
