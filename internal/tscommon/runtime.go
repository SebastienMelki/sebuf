package tscommon

import "strings"

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
