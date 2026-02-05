package annotations

import (
	"regexp"
	"strings"
)

// pathParamRegex matches path variables like {user_id} or {id}.
var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// ExtractPathParams parses path variables from a path string.
// Example: "/users/{user_id}/posts/{post_id}" -> ["user_id", "post_id"].
func ExtractPathParams(path string) []string {
	matches := pathParamRegex.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}

	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

// BuildHTTPPath combines service base path with method path.
// Handles slash normalization between the two path segments.
func BuildHTTPPath(servicePath, methodPath string) string {
	// Handle empty paths
	if servicePath == "" && methodPath == "" {
		return "/"
	}
	if servicePath == "" {
		return EnsureLeadingSlash(methodPath)
	}
	if methodPath == "" {
		return EnsureLeadingSlash(servicePath)
	}

	// Clean and combine paths
	servicePath = strings.TrimSuffix(EnsureLeadingSlash(servicePath), "/")
	methodPath = strings.TrimPrefix(methodPath, "/")

	return servicePath + "/" + methodPath
}

// EnsureLeadingSlash ensures a path starts with "/".
func EnsureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
