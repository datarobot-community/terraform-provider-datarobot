//go:build windows

package sync

import (
	"os"
	"syscall"
)

// CLI source: cli/internal/workload/sync/synclock.go

func lockFile(f *os.File) error {
	err := syscall.LockFile(syscall.Handle(f.Fd()), 0, 0, 1, 0)
	if err != nil {
		return ErrLocked
	}
	return nil
}

func unlockFile(f *os.File) error {
	return syscall.UnlockFile(syscall.Handle(f.Fd()), 0, 0, 1, 0)
}
