package pyclientgen

// pyKeywords lists every Python 3.10 reserved keyword. Identifiers colliding with
// any entry are escaped by appending a single trailing underscore (PEP 8 §Naming).
//
// To regenerate when targeting a newer Python version:
//
//	python3 -c "import keyword; print(sorted(keyword.kwlist + keyword.softkwlist))"
//
// Source: https://docs.python.org/3.10/reference/lexical_analysis.html#keywords
//
//nolint:gochecknoglobals // intentional constant lookup table
var pyKeywords = map[string]bool{
	pyFalse:    true,
	pyNone:     true,
	"True":     true,
	"and":      true,
	"as":       true,
	"assert":   true,
	"async":    true,
	"await":    true,
	"break":    true,
	"class":    true,
	"continue": true,
	"def":      true,
	"del":      true,
	"elif":     true,
	"else":     true,
	"except":   true,
	"finally":  true,
	"for":      true,
	"from":     true,
	"global":   true,
	"if":       true,
	"import":   true,
	"in":       true,
	"is":       true,
	"lambda":   true,
	"nonlocal": true,
	"not":      true,
	"or":       true,
	"pass":     true,
	"raise":    true,
	"return":   true,
	"try":      true,
	"while":    true,
	"with":     true,
	"yield":    true,
	// Soft keywords — these are only reserved in specific contexts (match/case)
	// but escaping them avoids surprises in field names.
	"match": true,
	"case":  true,
}

// escapePyKeyword returns the identifier, or the identifier suffixed with `_`
// when it collides with a Python keyword. The escape is reversible by the
// generated to_dict / from_dict serialization which preserves the original
// proto field name as the JSON key.
func escapePyKeyword(name string) string {
	if pyKeywords[name] {
		return name + "_"
	}
	return name
}
