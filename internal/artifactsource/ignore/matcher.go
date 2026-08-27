package ignore

// CLI source: cli/internal/workload/ignore/matcher.go
//
// Provider differences from CLI:
//   - Canonical ignore file is .drignore; .wapiignore is a fallback.
//   - Extra system excludes: .datarobot.yaml, .terraform, terraform.tfstate*.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

const (
	// FileName is the canonical user-editable ignore file at the project root.
	FileName = ".drignore"
	// LegacyFileName is the CLI ignore file; loaded only when FileName is absent.
	LegacyFileName = ".wapiignore"
)

// systemExcludes are always-ignored paths, not overridable by the user file.
var systemExcludes = []string{
	".wapi",
	".git",
	".gitignore",
	".datarobot.yaml",
	".terraform",
	"terraform.tfstate",
}

// Matcher decides whether a path is excluded from sync. Match is safe for
// concurrent use after New or FromLines.
type Matcher struct {
	user *gitignore.GitIgnore // nil when the user has no ignore file
}

// New loads .drignore from projectDir if present, otherwise .wapiignore.
// A missing file is fine — only the hardcoded system excludes apply.
func New(projectDir string) (*Matcher, error) {
	for _, name := range []string{FileName, LegacyFileName} {
		path := filepath.Join(projectDir, name)
		gi, err := gitignore.CompileIgnoreFile(path)
		if err == nil {
			return &Matcher{user: gi}, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
	}

	return &Matcher{}, nil
}

// FromLines builds a Matcher from in-memory pattern lines. Empty/nil
// means "system excludes only".
func FromLines(lines []string) *Matcher {
	if len(lines) == 0 {
		return &Matcher{}
	}

	return &Matcher{user: gitignore.CompileIgnoreLines(lines...)}
}

// Match reports whether relPath should be excluded. isDir lets
// directory-only patterns ("build/") prune subtrees.
func (m *Matcher) Match(relPath string, isDir bool) bool {
	if relPath == "" {
		return false
	}

	if matchesSystemExclude(relPath) {
		return true
	}

	if m == nil || m.user == nil {
		return false
	}

	if m.user.MatchesPath(relPath) {
		return true
	}

	// Directory-only patterns need a trailing slash to match in go-gitignore.
	if isDir {
		return m.user.MatchesPath(relPath + "/")
	}

	return false
}

// matchesSystemExclude reports whether relPath is or lives inside a
// system-excluded path.
func matchesSystemExclude(relPath string) bool {
	for _, name := range systemExcludes {
		if relPath == name || strings.HasPrefix(relPath, name+"/") {
			return true
		}
	}

	// terraform.tfstate.* (backups, terraform.tfstate.d, …)
	return strings.HasPrefix(relPath, "terraform.tfstate.")
}
