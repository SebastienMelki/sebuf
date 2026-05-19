// Package pyclientgen generates Python HTTP clients from protobuf service definitions.
//
// The generated code depends only on the Python standard library. Users may inject
// any duck-typed HttpTransport (e.g. httpx, requests, aiohttp) via client options;
// the default is a UrllibTransport built on urllib.request.
//
// The generator emits one Python file per .proto source containing:
//   - Dataclasses for each message
//   - IntEnums for each enum
//   - A transport Protocol and stdlib default
//   - An error hierarchy (ApiError, ValidationError, per-*Error-message classes)
//   - One client class per service with typed options and per-call options
//
// Server-Sent Events methods are detected and emit stubs that raise NotImplementedError.
package pyclientgen

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

// Generator produces Python HTTP client code for protobuf services.
type Generator struct {
	plugin *protogen.Plugin
}

// New creates a Python client generator.
func New(plugin *protogen.Plugin) *Generator {
	return &Generator{plugin: plugin}
}

// Generate iterates all input files and emits a {file}_client.py per source.
func (g *Generator) Generate() error {
	for _, file := range g.plugin.Files {
		if !file.Generate {
			continue
		}
		if err := g.generateFile(file); err != nil {
			return fmt.Errorf("pyclientgen: %s: %w", file.Desc.Path(), err)
		}
	}
	return nil
}

func (g *Generator) generateFile(file *protogen.File) error {
	if len(file.Services) == 0 && !hasGeneratableTypes(file) {
		return nil
	}
	return g.generateClientFile(file)
}

// hasGeneratableTypes returns true if the file has messages or enums worth emitting
// even when no service is declared. The generator emits message dataclasses and
// IntEnum classes alongside the client.
func hasGeneratableTypes(file *protogen.File) bool {
	return len(file.Messages) > 0 || len(file.Enums) > 0
}

func (g *Generator) generateClientFile(file *protogen.File) error {
	filename := file.GeneratedFilenamePrefix + "_client.py"
	gf := g.plugin.NewGeneratedFile(filename, "")

	p := newPrinter(gf)

	collected := collectFileTypes(file)

	writeHeader(p, file)
	writeImports(p, collected)
	writeTransport(p)

	// Enums must precede the error classes — an *Error message with an enum
	// field carries a default value expression like `code: Reason = Reason.X`
	// which is evaluated at def-time, so a forward reference to a
	// not-yet-declared enum would raise NameError on import. (Reported by
	// @yashagarwal-sarwa on #172.) The same ordering keeps message-typed
	// fields safe because message defaults are always None.
	for _, enum := range collected.OrderedEnums() {
		writeEnum(p, enum)
	}
	writeErrors(p, collected)
	for _, msg := range collected.OrderedMessages() {
		writeMessage(p, msg)
	}

	for _, service := range file.Services {
		writeServiceClient(p, service)
	}

	return nil
}
