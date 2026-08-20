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
