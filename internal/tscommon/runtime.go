package tscommon

import (
	"fmt"
	"strings"
)

// MessageRuntime selects how generated TypeScript represents protobuf messages.
type MessageRuntime int

const (
	// MessageRuntimeHandRolled emits hand-rolled TypeScript interfaces and plain
	// JSON (de)serialization. It is the zero value and the historical default.
	MessageRuntimeHandRolled MessageRuntime = iota
	// MessageRuntimeES emits code that targets the protobuf-es runtime.
	MessageRuntimeES
)

// ParseMessageRuntime scans a comma-separated protoc plugin parameter string
// (as passed via CodeGeneratorRequest.parameter) for a ts_runtime option and
// returns the selected runtime. An explicit ts_runtime=hand-rolled, an absent
// option, or any unrecognized value all resolve to MessageRuntimeHandRolled.
func ParseMessageRuntime(param string) MessageRuntime {
	for _, part := range strings.Split(param, ",") {
		part = strings.TrimSpace(part)
		name, value, found := strings.Cut(part, "=")
		if !found || name != "ts_runtime" {
			continue
		}
		if value == "protobuf-es" {
			return MessageRuntimeES
		}
	}
	return MessageRuntimeHandRolled
}

// UnsupportedEnumParamError builds the generation-time error returned when an
// enum-typed path or query parameter is encountered in protobuf-es runtime
// mode. protobuf-es represents enums as numeric values, but path/query
// parameters arrive as strings on the wire; safely bridging the two requires a
// name<->number conversion that is not yet implemented, so the generator fails
// loud here instead of emitting code that fails downstream `tsc`. paramKind is
// "path" or "query"; fieldName is the proto field name.
func UnsupportedEnumParamError(paramKind, fieldName, service, method string) error {
	return fmt.Errorf(
		"ts_runtime=protobuf-es: enum %s parameter %q on %s.%s is not yet supported "+
			"(protobuf-es enums are numeric; string<->enum conversion is not implemented)",
		paramKind, fieldName, service, method,
	)
}
