package sync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
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

// AcquireLock attempts to lock sync.lock inside projectDir's sync state
// directory. Returns ErrLocked if the lock is already held.
//
// The directory is resolved through wapi.Dir, not hardcoded: state lives
// at .datarobot/workload/ and only falls back to a legacy .wapi/ that the
// CLI has not migrated yet, so the lock has to follow it either way.
func AcquireLock(projectDir string) (*SyncLock, error) {
	stateDir := wapi.Dir(projectDir)
	info, err := os.Stat(stateDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("sync state directory %s does not exist", stateDir)
	}

	lockPath := filepath.Join(stateDir, lockFileName)
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
