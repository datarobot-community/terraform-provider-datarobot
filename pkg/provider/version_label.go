package provider

import (
	"fmt"
	"regexp"
	"strconv"
)

// versionLabelPattern matches version labels with optional 'v' prefix followed by digits.
// Capture groups: [0]=full match, [1]=optional 'v' prefix, [2]=numeric part.
// Examples: "v1" -> ["v1", "v", "1"], "123" -> ["123", "", "123"].
var versionLabelPattern = regexp.MustCompile(`^(v)?(\d+)$`)

// nextLabelFromLatest computes the next version label from the current one.
// Always returns 'v'-prefixed labels for consistency with DataRobot's versioning scheme.
// For valid numeric inputs: increments the number and adds 'v' prefix if missing.
// For invalid inputs: returns "v1" as safe fallback.
//
// Behavior:
//   - "v10" -> "v11" (increment existing v-prefixed version)
//   - "5"   -> "v6"  (add v-prefix to bare number)
//   - "abc" -> "v1"  (fallback for invalid input)
//   - ""    -> "v1"  (fallback for empty input)
func nextLabelFromLatest(latest string) string {
	// Always return labels with 'v' prefix for consistency with past behavior.
	nextNum := 1
	if m := versionLabelPattern.FindStringSubmatch(latest); len(m) == 3 {
		// m[1] is the optional 'v' prefix, m[2] is the numeric part
		if n, err := strconv.Atoi(m[2]); err == nil {
			nextNum = n + 1
		}
	}
	return fmt.Sprintf("v%d", nextNum)
}
