//go:build !windows

package sync

import (
	"fmt"
	"os"
	"syscall"
)

// CLI source: cli/internal/workload/sync/synclock.go

func lockFile(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
			return ErrLocked
		}
		return fmt.Errorf("acquire lock: %w", err)
	}
	return nil
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
