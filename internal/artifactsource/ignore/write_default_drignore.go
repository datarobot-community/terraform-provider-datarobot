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
//
// Presence is Locate's answer rather than a second stat here, so the writer and
// the reader cannot disagree about what counts as an ignore file. Where they
// would differ is a directory sitting at one of these names: Locate calls that
// absent, so the write is attempted and fails naming the real problem, instead
// of being skipped and surfacing later as a read error from New.
func WriteDefaultDrignoreIfMissing(projectDir string) (bool, error) {
	if Locate(projectDir) != "" {
		return false, nil
	}

	path := filepath.Join(projectDir, FileName)
	// 0644 rather than gosec's preferred 0600: this file is part of the user's
	// source tree, gets committed, and is read by teammates and CI. Owner-only
	// permissions on a file we created for them would be a surprise to debug.
	//nolint:gosec // G306: a committed, team-shared ignore file is world-readable by design.
	if err := os.WriteFile(path, DefaultTemplate, 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}

	return true, nil
}
