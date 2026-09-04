package sync

// CLI source: cli/internal/workload/fileops/paths.go
// (DetectCaseCollisions, FormatCaseCollisions). Ported as unexported
// helpers since Engine.Plan is the only caller in this package.

import (
	"fmt"
	"sort"
	"strings"
)

// caseCollision groups paths that differ only in case — they would
// silently overwrite each other on a case-insensitive filesystem (macOS,
// Windows default).
type caseCollision struct {
	lowered string
	paths   []string
}

// detectCaseCollisions returns groups of paths that collide under
// case-insensitive comparison, sorted for deterministic error output.
func detectCaseCollisions(paths map[string]struct{}) []caseCollision {
	groups := make(map[string][]string, len(paths))
	for p := range paths {
		key := strings.ToLower(p)
		groups[key] = append(groups[key], p)
	}

	var collisions []caseCollision
	for key, ps := range groups {
		if len(ps) < 2 {
			continue
		}
		sort.Strings(ps)
		collisions = append(collisions, caseCollision{lowered: key, paths: ps})
	}

	sort.Slice(collisions, func(i, j int) bool { return collisions[i].lowered < collisions[j].lowered })

	return collisions
}

// formatCaseCollisions renders collisions as a multi-line error message.
func formatCaseCollisions(cs []caseCollision) string {
	var b strings.Builder

	b.WriteString("case-only path collisions detected (would silently overwrite on a case-insensitive filesystem):\n")
	for _, c := range cs {
		fmt.Fprintf(&b, "  - %s\n", strings.Join(c.paths, " vs "))
	}

	return strings.TrimRight(b.String(), "\n")
}
