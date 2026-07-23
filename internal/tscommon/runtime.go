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

// ErrorHandling selects how generated client methods surface RPC failures.
type ErrorHandling int

const (
	// ErrorHandlingThrow throws (ValidationError/ApiError) on failure and returns
	// the decoded message on success. It is the zero value and historical default.
	ErrorHandlingThrow ErrorHandling = iota
	// ErrorHandlingResult returns a discriminated Result union
	// ({ ok, data } | { ok, error }) instead of throwing, with a typed error side.
	ErrorHandlingResult
)

// ParseErrorHandling scans a comma-separated protoc plugin parameter string for
// a ts_error_handling option. ts_error_handling=result selects the typed Result
// return; an explicit throw, an absent option, or any unrecognized value resolve
// to ErrorHandlingThrow.
func ParseErrorHandling(param string) ErrorHandling {
	for _, part := range strings.Split(param, ",") {
		part = strings.TrimSpace(part)
		name, value, found := strings.Cut(part, "=")
		if !found || name != "ts_error_handling" {
			continue
		}
		if value == "result" {
			return ErrorHandlingResult
		}
	}
	return ErrorHandlingThrow
}

// ValidateRuntimeOptions rejects unsupported runtime/error-handling combinations.
// ts_error_handling=result requires ts_runtime=protobuf-es: the typed-error decode
// relies on protobuf-es message schemas (fromJson + $typeName) that only exist in
// es-mode. Fail loud rather than emit code referencing symbols that aren't there.
func ValidateRuntimeOptions(runtime MessageRuntime, errorHandling ErrorHandling) error {
	if errorHandling == ErrorHandlingResult && runtime != MessageRuntimeES {
		return fmt.Errorf(
			"ts_error_handling=result requires ts_runtime=protobuf-es " +
				"(the typed Result error side is decoded via protobuf-es schemas)",
		)
	}
	return nil
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
