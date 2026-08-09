package testutil

import (
	"fmt"
	"strings"
)

// DiffStrings returns a bounded, line-oriented diff suitable for golden tests.
func DiffStrings(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	maxLines := max(len(expectedLines), len(actualLines))

	var diff strings.Builder
	diffCount := 0
	const maxDiffs = 20
	for i := 0; i < maxLines && diffCount < maxDiffs; i++ {
		var expectedLine, actualLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}
		if expectedLine == actualLine {
			continue
		}
		fmt.Fprintf(
			&diff,
			"Line %d:\n  expected: %s\n  actual:   %s\n",
			i+1,
			expectedLine,
			actualLine,
		)
		diffCount++
	}
	if diffCount == maxDiffs {
		diff.WriteString("... (more differences truncated)\n")
	}
	return diff.String()
}
