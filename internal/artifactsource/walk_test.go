package artifactsource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkDirectory_SkipsSymlinks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "real.txt"), []byte("ok"), 0o644))

	require.NoError(t, os.Symlink("real.txt", filepath.Join(root, "link.txt")))

	entries, err := walkDirectory(root, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "real.txt", entries[0].RelPath)
}

func TestWalkDirectory_IgnorePrunesDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "skip"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skip", "hidden.txt"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "keep.txt"), []byte("y"), 0o644))

	ignore := func(rel string, isDir bool) bool {
		return rel == "skip" && isDir
	}

	entries, err := walkDirectory(root, ignore)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "keep.txt", entries[0].RelPath)
}

func TestWalkDirectory_NormalizesPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "file.txt"), []byte("z"), 0o644))

	entries, err := walkDirectory(root, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "a/b/file.txt", entries[0].RelPath)
}

func TestWalkDirectory_EmptyDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	entries, err := walkDirectory(root, nil)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestWalkDirectory_NotADirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))

	_, err := walkDirectory(file, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}
