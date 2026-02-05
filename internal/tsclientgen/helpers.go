package tsclientgen

import (
	"strings"
)

// snakeToLowerCamel converts "user_id" to "userId".
func snakeToLowerCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 && i > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// headerNameToPropertyName converts "X-API-Key" to "apiKey".
func headerNameToPropertyName(headerName string) string {
	name := strings.TrimPrefix(headerName, "X-")
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		if i == 0 {
			parts[i] = strings.ToLower(part)
		} else {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}
