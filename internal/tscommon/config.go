package tscommon

import (
	"fmt"
	"strings"
)

// ImportStyle controls how generated TypeScript references types that are
// defined in other proto files.
type ImportStyle int

const (
	// ImportStyleInline (default) inlines every referenced message/enum into the
	// generated file, with no cross-file imports. This is the historical
	// behavior and the zero value.
	ImportStyleInline ImportStyle = iota
	// ImportStyleModules emits one canonical type module per proto file and
	// imports referenced types across files.
	ImportStyleModules
)

// OneofStyle controls how oneof fields WITHOUT an explicit sebuf.http
// oneof_config annotation are rendered. Annotated oneofs always follow their
// annotation regardless of this option.
type OneofStyle int

const (
	// OneofStyleFlatten (default) renders each oneof variant as an independent
	// optional field. This is the historical behavior and the zero value.
	OneofStyleFlatten OneofStyle = iota
	// OneofStyleDiscriminated renders un-annotated oneofs as discriminated
	// unions keyed by a synthesized "$case" discriminator.
	OneofStyleDiscriminated
)

// Options holds the parsed plugin options shared by the TypeScript generators
// (protoc-gen-ts-client and protoc-gen-ts-server).
type Options struct {
	ImportStyle ImportStyle
	OneofStyle  OneofStyle
}

// ParseOptions parses a protoc plugin parameter string of the form
// "key=value,key2=value2" into Options.
//
// Unknown keys are ignored on purpose: protoc concatenates every "--<plugin>_opt"
// flag into a single comma-separated parameter, so framework keys such as
// "paths=source_relative" reach us here too. A KNOWN key with an unrecognized
// value is a hard error — a silent fallback to the default would mask typos.
func ParseOptions(parameter string) (Options, error) {
	opts := Options{}
	if parameter == "" {
		return opts, nil
	}

	const kvParts = 2
	for _, pair := range strings.Split(parameter, ",") {
		kv := strings.SplitN(pair, "=", kvParts)
		if len(kv) != kvParts {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "import_style":
			switch value {
			case "modules":
				opts.ImportStyle = ImportStyleModules
			case "inline":
				opts.ImportStyle = ImportStyleInline
			default:
				return opts, fmt.Errorf("invalid import_style %q (want \"modules\" or \"inline\")", value)
			}
		case "oneof_style":
			switch value {
			case "discriminated":
				opts.OneofStyle = OneofStyleDiscriminated
			case "flatten":
				opts.OneofStyle = OneofStyleFlatten
			default:
				return opts, fmt.Errorf("invalid oneof_style %q (want \"discriminated\" or \"flatten\")", value)
			}
		default:
			// Ignore unknown keys (e.g. paths=source_relative).
		}
	}
	return opts, nil
}
