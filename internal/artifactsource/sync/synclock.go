package sync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// CLI source: cli/internal/workload/sync/synclock.go

const lockFileName = "sync.lock"

// ErrLocked is returned when the sync lock is already held by another process.
var ErrLocked = errors.New("sync lock is held by another process")

// SyncLock represents an active exclusive lock on .wapi/sync.lock.
type SyncLock struct {
	lockPath string
	file     *os.File
}

// AcquireLock attempts to lock .wapi/sync.lock in projectDir.
// Returns ErrLocked if the lock is already held.
func AcquireLock(projectDir string) (*SyncLock, error) {
	wapiDir := filepath.Join(projectDir, ".wapi")
	info, err := os.Stat(wapiDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf(".wapi directory does not exist in %s", projectDir)
	}

	lockPath := filepath.Join(wapiDir, lockFileName)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := lockFile(f); err != nil {
		_ = f.Close()
		return nil, err
	}

	return &SyncLock{
		lockPath: lockPath,
		file:     f,
	}, nil
}

// Unlock releases the lock and closes the lock file handle.
// The lock file is not removed on unlock to avoid flock-unlink race conditions across processes.
func (l *SyncLock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = unlockFile(l.file)
	_ = l.file.Close()
	l.file = nil
	return nil
}
