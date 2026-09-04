package sync

import (
	"errors"
	"os"
	"path/filepath"
	stdsync "sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireLock_NotInitialized(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := AcquireLock(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sync state directory")
}

func TestAcquireLock_WapiIsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiPath := filepath.Join(dir, ".wapi")
	require.NoError(t, os.WriteFile(wapiPath, []byte("not a directory"), 0644))

	_, err := AcquireLock(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sync state directory")
}

func TestAcquireLock_Exclusivity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".wapi"), 0755))

	lock1, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, lock1)

	// Second acquire on the same projectDir must fail with ErrLocked
	_, err = AcquireLock(dir)
	assert.ErrorIs(t, err, ErrLocked)

	// Unlock first lock
	require.NoError(t, lock1.Unlock())

	// Third acquire should now succeed
	lock2, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, lock2)
	require.NoError(t, lock2.Unlock())
}

func TestAcquireLock_UnlockRetainsLockFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".wapi"), 0755))

	lock, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, lock)

	lockFilePath := filepath.Join(dir, ".wapi", lockFileName)
	assert.FileExists(t, lockFilePath)

	require.NoError(t, lock.Unlock())
	// Lock file remains on disk to prevent flock-unlink race conditions across processes
	assert.FileExists(t, lockFilePath)
}

func TestAcquireLock_ReadOnlyDirectory(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("skipping read-only directory test when running as root")
	}

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0555))
	t.Cleanup(func() {
		_ = os.Chmod(wapiDir, 0755)
	})

	_, err := AcquireLock(dir)
	if err == nil {
		t.Skip("skipping: filesystem did not enforce directory read-only permissions")
	}
	assert.Contains(t, err.Error(), "open lock file:")
}

func TestAcquireLock_Concurrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".wapi"), 0755))

	const numGoroutines = 10
	var wg stdsync.WaitGroup
	var mu stdsync.Mutex
	var acquiredLocks []*SyncLock
	var lockedErrors int

	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			lock, err := AcquireLock(dir)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				acquiredLocks = append(acquiredLocks, lock)
			} else if errors.Is(err, ErrLocked) {
				lockedErrors++
			}
		}()
	}

	close(start)
	wg.Wait()

	assert.Len(t, acquiredLocks, 1)
	assert.Equal(t, numGoroutines-1, lockedErrors)

	for _, l := range acquiredLocks {
		require.NoError(t, l.Unlock())
	}
}

func TestSyncLock_NilUnlock(t *testing.T) {
	t.Parallel()

	var l *SyncLock
	assert.NoError(t, l.Unlock())
}

func TestSyncLock_UnlockIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".wapi"), 0755))

	lock, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, lock)

	// First unlock
	assert.NoError(t, lock.Unlock())

	// Second unlock on the same instance should safely return nil
	assert.NoError(t, lock.Unlock())
}
