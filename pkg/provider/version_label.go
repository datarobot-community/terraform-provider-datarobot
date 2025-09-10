package provider

import (
	"fmt"
	"regexp"
	"strconv"
)

// versionLabelPattern matches version labels like "v1", "v10", or "123".
var versionLabelPattern = regexp.MustCompile(`^v?(\d+)$`)

// nextLabelFromLatest computes the next version label from the latest.
// Output is always with 'v' prefix for consistency with past behavior.
// For valid inputs: increments the numeric part (e.g., "v1" -> "v2", "1" -> "v2").
// For invalid inputs: falls back to "v1".
//
// Examples:
//
//	"v10" -> "v11"
//	"1"   -> "v2"
//	"abc" -> "v1"
func nextLabelFromLatest(latest string) string {
	// Always return labels with 'v' prefix for consistency with past behavior.
	nextNum := 1
	if m := versionLabelPattern.FindStringSubmatch(latest); len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			nextNum = n + 1
		}
	}
	return fmt.Sprintf("v%d", nextNum)
}
