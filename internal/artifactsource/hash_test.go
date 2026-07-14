package artifactsource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashFile_KnownVector(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "hello.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

	hash, size, err := hashFile(path)
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)
	// SHA-256 of "hello"
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", hash)
}

func TestHashFile_TooLarge(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "big.bin")

	// Create a file just over the limit without allocating 5GiB in memory.
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Truncate(maxFileSizeBytes+1))
	require.NoError(t, f.Close())

	_, _, err = hashFile(path)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFileTooLarge)
}
