package ignore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteDefaultDrignoreIfMissing_WritesOnce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wrote, err := WriteDefaultDrignoreIfMissing(dir)
	require.NoError(t, err)
	assert.True(t, wrote)

	got, err := os.ReadFile(filepath.Join(dir, FileName))
	require.NoError(t, err)
	assert.Equal(t, DefaultTemplate, got)

	wrote, err = WriteDefaultDrignoreIfMissing(dir)
	require.NoError(t, err)
	assert.False(t, wrote)
}

func TestWriteDefaultDrignoreIfMissing_DoesNotOverwriteCustom(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	custom := []byte("*.tmp\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), custom, 0o644))

	wrote, err := WriteDefaultDrignoreIfMissing(dir)
	require.NoError(t, err)
	assert.False(t, wrote)

	got, err := os.ReadFile(filepath.Join(dir, FileName))
	require.NoError(t, err)
	assert.Equal(t, custom, got)
}

func TestWriteDefaultDrignoreIfMissing_SkipsWhenWapiignoreExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte("*.log\n"), 0o644))

	wrote, err := WriteDefaultDrignoreIfMissing(dir)
	require.NoError(t, err)
	assert.False(t, wrote)
	assert.NoFileExists(t, filepath.Join(dir, FileName))
}

// A directory at the ignore file's name is not an ignore file, so the writer
// treats it as absent and the attempted write reports what is actually wrong.
// The alternative, skipping quietly, moves the failure to New, where the
// message is about reading a file the user never asked to be read.
func TestWriteDefaultDrignoreIfMissing_DirectoryAtTheName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, FileName), 0o755))

	wrote, err := WriteDefaultDrignoreIfMissing(dir)
	assert.False(t, wrote)
	require.Error(t, err)
	assert.Contains(t, err.Error(), FileName)
}

// The writer asks Locate rather than stat'ing the names itself, so a project
// that has only the legacy file is left alone.
func TestWriteDefaultDrignoreIfMissing_AgreesWithLocate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte("*.log\n"), 0o644))
	require.NotEmpty(t, Locate(dir))

	wrote, err := WriteDefaultDrignoreIfMissing(dir)
	require.NoError(t, err)
	assert.False(t, wrote)
	assert.NoFileExists(t, filepath.Join(dir, FileName))
}
