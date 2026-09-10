package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func pathSet(paths ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		out[p] = struct{}{}
	}
	return out
}

func TestDetectCaseCollisions_NoCollisions(t *testing.T) {
	t.Parallel()

	got := detectCaseCollisions(pathSet("agent.py", "utils/helper.py"))
	assert.Empty(t, got)
}

func TestDetectCaseCollisions_SingleCollision(t *testing.T) {
	t.Parallel()

	got := detectCaseCollisions(pathSet("Config.yaml", "config.yaml"))
	assert.Equal(t, []caseCollision{{lowered: "config.yaml", paths: []string{"Config.yaml", "config.yaml"}}}, got)
}

func TestDetectCaseCollisions_MultiCollision(t *testing.T) {
	t.Parallel()

	got := detectCaseCollisions(pathSet("a.py", "A.py", "b.py", "B.py"))
	assert.Equal(t, []caseCollision{
		{lowered: "a.py", paths: []string{"A.py", "a.py"}},
		{lowered: "b.py", paths: []string{"B.py", "b.py"}},
	}, got)
}

func TestFormatCaseCollisions_ListsEachGroup(t *testing.T) {
	t.Parallel()

	msg := formatCaseCollisions([]caseCollision{{lowered: "config.yaml", paths: []string{"Config.yaml", "config.yaml"}}})
	assert.Contains(t, msg, "case-only path collisions")
	assert.Contains(t, msg, "Config.yaml vs config.yaml")
}
