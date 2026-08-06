package artifactsource

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildZip_EntryNamesAndDeflate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.py"), []byte("print(1)"), 0o644))
	sub := filepath.Join(root, "lib")
	require.NoError(t, os.Mkdir(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "util.py"), []byte("pass"), 0o644))

	files := []LocalFile{
		{RelPath: "main.py", AbsPath: filepath.Join(root, "main.py")},
		{RelPath: "lib/util.py", AbsPath: filepath.Join(root, "lib", "util.py")},
	}

	zipPath, err := buildZip(files)
	require.NoError(t, err)
	defer func() { _ = os.Remove(zipPath) }()

	r, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	require.Len(t, r.File, 2)
	assert.Equal(t, "main.py", r.File[0].Name)
	assert.Equal(t, zip.Deflate, r.File[0].Method)
	assert.Equal(t, "lib/util.py", r.File[1].Name)
	assert.Equal(t, zip.Deflate, r.File[1].Method)

	for _, name := range []string{r.File[0].Name, r.File[1].Name} {
		assert.NotContains(t, name, root)
		assert.False(t, filepath.IsAbs(name))
	}
}

func TestBuildZip_NoExtraRootDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "only.txt"), []byte("x"), 0o644))

	files := []LocalFile{{RelPath: "only.txt", AbsPath: filepath.Join(root, "only.txt")}}
	zipPath, err := buildZip(files)
	require.NoError(t, err)
	defer func() { _ = os.Remove(zipPath) }()

	r, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	require.Len(t, r.File, 1)
	assert.Equal(t, "only.txt", r.File[0].Name)
}
