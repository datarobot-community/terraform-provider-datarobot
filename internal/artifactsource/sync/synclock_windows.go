//go:build windows

package sync

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// CLI source: cli/internal/workload/sync/synclock.go

// lockRegionLength is the number of bytes locked in the lock file. Locking a
// single byte at offset 0 is enough to make the lock mutually exclusive.
const lockRegionLength = 1

func lockFile(f *os.File) error {
	// LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY mirrors the Unix LOCK_EX|LOCK_NB semantics.
	err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockRegionLength,
		0,
		&windows.Overlapped{},
	)
	if err != nil {
		// Only contention maps to ErrLocked; bad handles, I/O failures and
		// unsupported filesystems must surface as themselves.
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
			return ErrLocked
		}
		return fmt.Errorf("acquire lock: %w", err)
	}
	return nil
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockRegionLength, 0, &windows.Overlapped{})
}
