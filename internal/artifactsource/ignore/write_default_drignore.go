package ignore

import (
	"fmt"
	"os"
	"path/filepath"
)

// CLI source: cli/internal/workload/wapi/init.go (write-once .wapiignore drop)
//
// WriteDefaultDrignoreIfMissing writes DefaultTemplate to .drignore in
// projectDir when neither .drignore nor .wapiignore exists. It never
// overwrites an existing file. Returns true when a new file was written.
func WriteDefaultDrignoreIfMissing(projectDir string) (bool, error) {
	if UserIgnoreExists(projectDir) {
		return false, nil
	}

	path := filepath.Join(projectDir, FileName)
	if err := os.WriteFile(path, DefaultTemplate, 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}

	return true, nil
}

// UserIgnoreExists reports whether projectDir already has .drignore or .wapiignore.
func UserIgnoreExists(projectDir string) bool {
	for _, name := range []string{FileName, LegacyFileName} {
		if _, err := os.Stat(filepath.Join(projectDir, name)); err == nil {
			return true
		}
	}

	return false
}
