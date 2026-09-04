//go:build windows

package sync

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLockFile_NonContentionErrorIsNotErrLocked verifies that failures unrelated to
// contention (here: an already-closed handle) surface as themselves instead of
// masquerading as "sync lock is held by another process".
func TestLockFile_NonContentionErrorIsNotErrLocked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	f, err := os.OpenFile(filepath.Join(dir, lockFileName), os.O_CREATE|os.O_RDWR, 0600)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = lockFile(f)
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrLocked), "invalid handle must not be reported as ErrLocked, got: %v", err)
	assert.Contains(t, err.Error(), "acquire lock:")
}
