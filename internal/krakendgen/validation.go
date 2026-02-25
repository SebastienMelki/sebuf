package krakendgen

import (
	"fmt"
	"strings"
)

// ValidateRoutes checks the generated endpoints for route conflicts.
// It detects two classes of problems:
//   - Duplicate (path, method) tuples: two RPCs mapped to the exact same endpoint.
//   - Static vs parameterized segment conflicts: a literal segment and a path
//     parameter at the same trie depth for the same HTTP method.
//
// serviceName is included in error messages for developer context.
func ValidateRoutes(endpoints []Endpoint, serviceName string) error {
	if err := checkDuplicateRoutes(endpoints, serviceName); err != nil {
		return err
	}
	return checkSegmentConflicts(endpoints, serviceName)
}

// checkDuplicateRoutes returns an error if two endpoints share the same
// (path, method) tuple.
func checkDuplicateRoutes(endpoints []Endpoint, serviceName string) error {
	type routeKey struct {
		path   string
		method string
	}

	// We need RPC names for error messages but Endpoint doesn't carry them.
	// Use the backend URL pattern as a proxy -- each RPC produces a unique
	// endpoint path.  We track by index to get meaningful names.
	seen := make(map[routeKey]int) // key -> index of first occurrence
	for i, ep := range endpoints {
		key := routeKey{path: ep.Endpoint, method: ep.Method}
		if firstIdx, exists := seen[key]; exists {
			return fmt.Errorf(
				"service %s: duplicate route: %s %s (endpoints[%d] and endpoints[%d])",
				serviceName, ep.Method, ep.Endpoint, firstIdx, i,
			)
		}
		seen[key] = i
	}
	return nil
}

// routeNode is a trie node for path-segment conflict detection.
type routeNode struct {
	children   map[string]*routeNode
	paramChild *routeNode
	paramName  string
	// rpcIndex is set at leaf nodes for error reporting (-1 = not a leaf).
	rpcIndex int
}

func newRouteNode() *routeNode {
	return &routeNode{children: make(map[string]*routeNode), rpcIndex: -1}
}

// checkSegmentConflicts builds a trie per HTTP method and reports when a
// static segment and a parameterized segment coexist at the same depth.
func checkSegmentConflicts(endpoints []Endpoint, serviceName string) error {
	// One trie root per HTTP method.
	methodTries := make(map[string]*routeNode)

	for i, ep := range endpoints {
		root, ok := methodTries[ep.Method]
		if !ok {
			root = newRouteNode()
			methodTries[ep.Method] = root
		}

		segments := splitSegments(ep.Endpoint)
		node := root

		for _, seg := range segments {
			isParam := strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")

			if isParam {
				// Conflict: inserting a param when static children exist.
				if len(node.children) > 0 {
					// Find a static child name for the error message.
					for staticSeg, child := range node.children {
						return fmt.Errorf(
							"service %s: route conflict at /%s: parameter %q (endpoints[%d]) conflicts with static segment %q (endpoints[%d])",
							serviceName,
							buildPrefix(segments, seg),
							seg, i,
							staticSeg, child.rpcIndex,
						)
					}
				}

				if node.paramChild == nil {
					node.paramChild = newRouteNode()
					node.paramChild.paramName = seg
					node.paramChild.rpcIndex = i
				}
				node = node.paramChild
			} else {
				// Conflict: inserting a static segment when a param child exists.
				if node.paramChild != nil {
					return fmt.Errorf(
						"service %s: route conflict at /%s: static segment %q (endpoints[%d]) conflicts with parameter %q (endpoints[%d])",
						serviceName,
						buildPrefix(segments, seg),
						seg, i,
						node.paramChild.paramName, node.paramChild.rpcIndex,
					)
				}

				child, exists := node.children[seg]
				if !exists {
					child = newRouteNode()
					child.rpcIndex = i
					node.children[seg] = child
				}
				node = child
			}
		}
	}

	return nil
}

// splitSegments splits a path into non-empty segments.
// "/api/v1/users/{id}" -> ["api", "v1", "users", "{id}"]
func splitSegments(path string) []string {
	parts := strings.Split(path, "/")
	segments := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}
	return segments
}

// buildPrefix returns the path prefix up to (but not including) the given
// segment, for use in conflict error messages.
func buildPrefix(segments []string, current string) string {
	var prefix []string
	for _, s := range segments {
		if s == current {
			break
		}
		prefix = append(prefix, s)
	}
	if len(prefix) == 0 {
		return ""
	}
	return strings.Join(prefix, "/") + "/"
}
