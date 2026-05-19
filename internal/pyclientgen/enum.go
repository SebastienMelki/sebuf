package pyclientgen

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/SebastienMelki/sebuf/internal/annotations"
)

// writeEnum emits a Python IntEnum for the given proto enum plus a sibling
// `_VALUES` dict mapping each variant to its custom JSON string when
// `(sebuf.http.enum_value)` is set. The dict is empty when no custom values
// are declared but is always emitted for shape stability.
func writeEnum(p printer, enum *protogen.Enum) {
	name := pythonEnumName(enum)

	p("class %s(IntEnum):", name)
	p(`    """Generated from proto enum %s."""`, enum.Desc.FullName())
	for _, value := range enum.Values {
		variantName := variantPythonName(enum, value)
		p("    %s = %d", variantName, value.Desc.Number())
	}
	p("")
	p("")

	if annotations.HasAnyEnumValueMapping(enum) {
		p("%s_JSON_VALUES: Mapping[%s, str] = {", name, name)
		for _, value := range enum.Values {
			override := annotations.GetEnumValueMapping(value)
			if override == "" {
				continue
			}
			variantName := variantPythonName(enum, value)
			p("    %s.%s: %q,", name, variantName, override)
		}
		p("}")
	} else {
		// Empty mapping kept for shape stability so the decoder can iterate
		// JSON_VALUES.items() without a special case.
		p("%s_JSON_VALUES: Mapping[%s, str] = {}", name, name)
	}
	p("")
	writeEnumDecoder(p, enum)
}

// writeEnumDecoder emits a `_decode_enum_<Name>` function that accepts either
// a JSON string (enum name or custom enum_value) or an int and returns the
// IntEnum member. Generated to_dict / from_dict uses this for STRING-encoded
// enums so unknown values raise instead of silently passing through.
func writeEnumDecoder(p printer, enum *protogen.Enum) {
	name := pythonEnumName(enum)
	p("def _decode_enum_%s(value: Any) -> %s:", name, name)
	p("    if isinstance(value, int):")
	p("        return %s(value)", name)
	p("    if isinstance(value, str):")
	p("        for member, json_value in %s_JSON_VALUES.items():", name)
	p("            if json_value == value:")
	p("                return member")
	p("        try:")
	p("            return %s[value]", name)
	p("        except KeyError:")
	p(`            raise ValueError(f"unknown %s value: {value!r}")`, name)
	p(`    raise TypeError(f"cannot decode %s from {type(value).__name__}")`, name)
	p("")
	p("")
}

// variantPythonName trims the redundant enum-name prefix from each variant.
// proto convention: enum Status { STATUS_ACTIVE = 1; } -> ACTIVE.
// When trimming would produce an invalid identifier (leading digit) we keep the original.
func variantPythonName(enum *protogen.Enum, value *protogen.EnumValue) string {
	prefix := strings.ToUpper(camelToSnake(string(enum.Desc.Name()))) + "_"
	name := string(value.Desc.Name())
	trimmed := strings.TrimPrefix(name, prefix)
	if trimmed == "" || isInvalidIdentifier(trimmed) {
		return escapePyKeyword(strings.ToLower(name))
	}
	return escapePyKeyword(strings.ToLower(trimmed))
}

func isInvalidIdentifier(s string) bool {
	if s == "" {
		return true
	}
	first := s[0]
	if first >= '0' && first <= '9' {
		return true
	}
	return false
}

// camelToSnake converts CamelCase to snake_case for enum-prefix trimming.
func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return b.String()
}
