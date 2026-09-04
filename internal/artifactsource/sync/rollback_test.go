package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollback_BackupAndRestoreMutatedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	appPy := filepath.Join(dir, "app.py")
	require.NoError(t, os.WriteFile(appPy, []byte("version 1"), 0644))

	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())

	// Backup app.py
	require.NoError(t, rt.Backup("app.py"))
	// Backup a file that will be newly created
	require.NoError(t, rt.Backup("new_file.txt"))

	assert.True(t, HasRollback(dir))

	// Mutate app.py and create new_file.txt
	require.NoError(t, os.WriteFile(appPy, []byte("version 2 mutated"), 0644))
	newFilePath := filepath.Join(dir, "new_file.txt")
	require.NoError(t, os.WriteFile(newFilePath, []byte("new file content"), 0644))

	// Restore
	require.NoError(t, rt.Restore())

	// Verify app.py is restored to version 1 and new_file.txt was removed
	content, err := os.ReadFile(appPy)
	require.NoError(t, err)
	assert.Equal(t, "version 1", string(content))

	_, err = os.Stat(newFilePath)
	assert.True(t, os.IsNotExist(err))

	assert.False(t, HasRollback(dir))
}

func TestRollback_StaleRestore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	rollbackDir := filepath.Join(wapiDir, ".rollback")
	require.NoError(t, os.Mkdir(rollbackDir, 0755))

	// Manually create stale backed-up file in .rollback without a manifest
	staleFile := filepath.Join(rollbackDir, "config.py")
	require.NoError(t, os.WriteFile(staleFile, []byte("stale backup content"), 0644))

	// Modify file on working tree
	targetFile := filepath.Join(dir, "config.py")
	require.NoError(t, os.WriteFile(targetFile, []byte("bad current content"), 0644))

	// Calling RestoreRollback should recover the stale backup
	require.NoError(t, RestoreRollback(dir))

	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, "stale backup content", string(content))
	assert.False(t, HasRollback(dir))
}

func TestRollback_Init_NoWapiDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rt := NewRollbackTree(dir)
	err := rt.Init()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sync state directory")
}

func TestRollback_NestedDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	nestedFile := filepath.Join(dir, "src", "utils", "helper.py")
	require.NoError(t, os.MkdirAll(filepath.Dir(nestedFile), 0755))
	require.NoError(t, os.WriteFile(nestedFile, []byte("original nested content"), 0644))

	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())

	// Backup nested existing file and non-existing nested file
	require.NoError(t, rt.Backup("src/utils/helper.py"))
	require.NoError(t, rt.Backup("src/generated/model.py"))

	// Mutate existing nested file and create new nested file
	require.NoError(t, os.WriteFile(nestedFile, []byte("mutated nested content"), 0644))
	newNestedFile := filepath.Join(dir, "src", "generated", "model.py")
	require.NoError(t, os.MkdirAll(filepath.Dir(newNestedFile), 0755))
	require.NoError(t, os.WriteFile(newNestedFile, []byte("new nested model"), 0644))

	// Restore
	require.NoError(t, rt.Restore())

	// Verify original file restored and newly created nested file removed
	content, err := os.ReadFile(nestedFile)
	require.NoError(t, err)
	assert.Equal(t, "original nested content", string(content))

	_, err = os.Stat(newNestedFile)
	assert.True(t, os.IsNotExist(err))
	assert.False(t, HasRollback(dir))
}

func TestRollback_BackupDeduplication(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	appPy := filepath.Join(dir, "app.py")
	require.NoError(t, os.WriteFile(appPy, []byte("version 1"), 0644))

	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())

	// Call Backup multiple times for both existing and newly created files
	require.NoError(t, rt.Backup("app.py"))
	require.NoError(t, rt.Backup("app.py"))
	require.NoError(t, rt.Backup("new_file.txt"))
	require.NoError(t, rt.Backup("new_file.txt"))

	assert.Equal(t, []string{"app.py"}, rt.manifest.BackedUpFiles)
	assert.Equal(t, []string{"new_file.txt"}, rt.manifest.CreatedFiles)
}

func TestRollback_RestoreNoRollbackDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Calling RestoreRollback on a clean directory with no .rollback should safely return nil
	assert.NoError(t, RestoreRollback(dir))

	rt := NewRollbackTree(dir)
	assert.NoError(t, rt.Restore())
}

func TestRollback_CorruptManifestFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	rollbackDir := filepath.Join(wapiDir, ".rollback")
	require.NoError(t, os.Mkdir(rollbackDir, 0755))

	// Write corrupted manifest.json
	manifestPath := filepath.Join(rollbackDir, "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte("invalid-json{"), 0644))

	// Write backup file in .rollback
	backupFile := filepath.Join(rollbackDir, "script.py")
	require.NoError(t, os.WriteFile(backupFile, []byte("backup script version"), 0644))

	// Mutated target file in working tree
	targetFile := filepath.Join(dir, "script.py")
	require.NoError(t, os.WriteFile(targetFile, []byte("corrupted working copy"), 0644))

	// RestoreRollback should recover using the directory-walk fallback despite corrupted manifest
	require.NoError(t, RestoreRollback(dir))

	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, "backup script version", string(content))
	assert.False(t, HasRollback(dir))
}

func TestRollback_CreatedFileAlreadyDeleted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())

	// Track a new file that never actually got written, or was deleted before rollback
	require.NoError(t, rt.Backup("deleted_before_restore.txt"))

	// Restore should succeed without error
	require.NoError(t, rt.Restore())
	assert.False(t, HasRollback(dir))
}

func TestRollback_PathSafety(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())

	// Empty path
	assert.Error(t, rt.Backup(""))

	// Path escape with ../
	assert.Error(t, rt.Backup("../outside.txt"))
	assert.Error(t, rt.Backup("../../etc/passwd"))
	assert.Error(t, rt.Backup("foo/../../outside.txt"))

	// Absolute path
	absPath := filepath.Join(dir, "file.txt")
	assert.Error(t, rt.Backup(absPath))

	// Target inside .wapi/
	assert.Error(t, rt.Backup(".wapi/config.json"))
}

func TestRollback_BackupDirectoryError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	subDir := filepath.Join(dir, "somedir")
	require.NoError(t, os.Mkdir(subDir, 0755))

	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())

	err := rt.Backup("somedir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot backup directory somedir: only files are supported")
}

func TestRollback_RestoreManifestPathTraversal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	rollbackDir := filepath.Join(wapiDir, ".rollback")
	require.NoError(t, os.Mkdir(rollbackDir, 0755))

	// Malicious manifest targeting outside project
	manifestData := []byte(`{
		"backedUpFiles": ["../../evil.txt"],
		"createdFiles": ["../evil_created.txt"]
	}`)
	require.NoError(t, os.WriteFile(filepath.Join(rollbackDir, "manifest.json"), manifestData, 0644))

	err := RestoreRollback(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest")
}

func TestRollback_Discard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wapiDir := filepath.Join(dir, ".wapi")
	require.NoError(t, os.Mkdir(wapiDir, 0755))

	rt := NewRollbackTree(dir)
	// Discard before Init/Backup should succeed
	require.NoError(t, rt.Discard())

	require.NoError(t, rt.Init())
	require.NoError(t, rt.Backup("file.txt"))
	assert.True(t, HasRollback(dir))

	require.NoError(t, rt.Discard())
	assert.False(t, HasRollback(dir))

	// Consecutive discard call should be idempotent
	require.NoError(t, rt.Discard())
}
