package clientgen

import "fmt"

// JSON naming style constants for [Options.JSONNaming].
const (
	// JSONNamingCamelCase renders proto fields as lowerCamelCase in
	// JSON request bodies. This is the default.
	JSONNamingCamelCase = "camel_case"

	// JSONNamingSnakeCase preserves proto field names verbatim in
	// JSON request bodies via protojson's UseProtoNames option.
	JSONNamingSnakeCase = "snake_case"
)

// Options configures the generator.
type Options struct {
	// JSONNaming selects the field naming style for JSON request bodies.
	// See the JSONNaming* constants. Empty defaults to [JSONNamingCamelCase].
	JSONNaming string
}

// validate returns an error if any field is set to an unrecognised
// value. Assumes defaults have been filled in at construction time.
func (o Options) validate() error {
	switch o.JSONNaming {
	case JSONNamingCamelCase, JSONNamingSnakeCase:
		return nil
	default:
		return fmt.Errorf(
			"invalid json_naming %q; expected %q or %q",
			o.JSONNaming, JSONNamingCamelCase, JSONNamingSnakeCase,
		)
	}
}
