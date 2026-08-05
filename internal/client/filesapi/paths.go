// CLI source: cli/internal/workload/fileops/paths.go (partial port)
//
// Provider differences from CLI:
//   - Only SafeRelPath and NormalizePath are copied; DetectCaseCollisions and
//     FormatCaseCollisions remain CLI-only (cli/internal/workload/fileops/paths.go).
//   - Functions live in package filesapi so allfiles.go can call them without importing fileops.
package filesapi

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// SafeRelPath rejects paths that aren't safe to join with a project root.
func SafeRelPath(p string) error {
	if p == "" {
		return errors.New("empty path")
	}

	if strings.ContainsRune(p, '\\') {
		return fmt.Errorf("backslash in path not allowed: %q", p)
	}

	if strings.HasPrefix(p, "/") || filepath.IsAbs(p) {
		return fmt.Errorf("absolute path not allowed: %q", p)
	}

	cleaned := path.Clean(p)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("path escapes project root: %q", p)
	}

	return nil
}

// NormalizePath returns a canonical manifest key: forward slashes, NFC
// Unicode, no leading "./", no trailing slash.
func NormalizePath(p string) string {
	p = filepath.ToSlash(p)
	p = norm.NFC.String(p)

	for strings.HasPrefix(p, "./") {
		p = p[2:]
	}

	return strings.TrimRight(p, "/")
}
