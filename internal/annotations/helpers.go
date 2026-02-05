package annotations

import "strings"

// LowerFirst converts "FooBar" to "fooBar".
func LowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}
