package wapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize_CreatesLayout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := Initialize(dir, InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	})
	require.NoError(t, err)

	entries, err := os.ReadDir(filepath.Join(dir, DirName))
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.ElementsMatch(t, []string{gitignoreFile, configFile, manifestFile}, names)

	gi, err := os.ReadFile(gitignorePath(dir))
	require.NoError(t, err)
	assert.Equal(t, "*\n", string(gi))

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "art-1", cfg.ArtifactID)
	require.NotNil(t, cfg.CatalogID)
	assert.Equal(t, "cat-1", *cfg.CatalogID)
	require.NotNil(t, cfg.LastSyncedVersionID)
	assert.Equal(t, "ver-1", *cfg.LastSyncedVersionID)
	assert.Equal(t, ProviderWriter, cfg.CLIVersion)

	raw, err := os.ReadFile(manifestPath(dir))
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	assert.EqualValues(t, 1, parsed["version"])
	assert.Nil(t, parsed["syncedAt"])
	assert.Nil(t, parsed["syncedVersionId"])
	files, ok := parsed["files"].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, files)
}

func TestInitialize_AlreadyLinked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, Initialize(dir, InitOptions{ArtifactID: "art-1"}))

	err := Initialize(dir, InitOptions{ArtifactID: "art-2"})
	assert.ErrorIs(t, err, ErrAlreadyLinked)

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "art-1", cfg.ArtifactID)
}

func TestConfigManifest_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, Initialize(dir, InitOptions{ArtifactID: "art-1"}))

	catalog := "cat-9"
	version := "ver-9"
	created := time.Date(2026, 4, 10, 9, 15, 0, 0, time.UTC)
	require.NoError(t, SaveConfig(dir, Config{
		ArtifactID:          "art-1",
		CatalogID:           &catalog,
		LastSyncedVersionID: &version,
		CreatedAt:           created,
		CLIVersion:          ProviderWriter,
	}))

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "art-1", cfg.ArtifactID)
	require.NotNil(t, cfg.CatalogID)
	assert.Equal(t, catalog, *cfg.CatalogID)
	require.NotNil(t, cfg.LastSyncedVersionID)
	assert.Equal(t, version, *cfg.LastSyncedVersionID)
	assert.True(t, created.Equal(cfg.CreatedAt))

	synced := time.Date(2026, 4, 10, 9, 30, 0, 0, time.UTC)
	require.NoError(t, SaveManifest(dir, Manifest{
		Version:         ManifestVersion,
		SyncedAt:        &synced,
		SyncedVersionID: &version,
		Files: map[string]FileMeta{
			"app.py": {Hash: "aaa", Size: 12},
		},
	}))

	m, err := LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, ManifestVersion, m.Version)
	require.NotNil(t, m.SyncedAt)
	assert.True(t, synced.Equal(*m.SyncedAt))
	require.NotNil(t, m.SyncedVersionID)
	assert.Equal(t, version, *m.SyncedVersionID)
	assert.Equal(t, FileMeta{Hash: "aaa", Size: 12}, m.Files["app.py"])
}

func TestLoad_NotInitialized(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := LoadConfig(dir)
	assert.ErrorIs(t, err, ErrNotInitialized)
	_, err = LoadManifest(dir)
	assert.ErrorIs(t, err, ErrNotInitialized)
}

func TestInitialize_NullsForEmptyOptionals(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, Initialize(dir, InitOptions{ArtifactID: "art-1"}))

	raw, err := os.ReadFile(configPath(dir))
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	assert.Nil(t, parsed["catalogId"])
	assert.Nil(t, parsed["lastSyncedVersionId"])

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Nil(t, cfg.CatalogID)
	assert.Nil(t, cfg.LastSyncedVersionID)
}

func TestLoad_CorruptedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, Initialize(dir, InitOptions{ArtifactID: "art-1"}))
	require.NoError(t, os.WriteFile(configPath(dir), []byte("not json"), 0o600))
	require.NoError(t, os.WriteFile(manifestPath(dir), []byte("not json"), 0o600))

	_, err := LoadConfig(dir)
	var cfgErr *CorruptedError
	require.ErrorAs(t, err, &cfgErr)
	assert.Equal(t, configPath(dir), cfgErr.Path)

	_, err = LoadManifest(dir)
	var manErr *CorruptedError
	require.ErrorAs(t, err, &manErr)
	assert.Equal(t, manifestPath(dir), manErr.Path)
}

func TestSave_NotInitialized(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := SaveConfig(dir, Config{ArtifactID: "art-1"})
	assert.ErrorIs(t, err, ErrNotInitialized)
	err = SaveManifest(dir, Manifest{Version: ManifestVersion})
	assert.ErrorIs(t, err, ErrNotInitialized)
}

func TestInitialize_EmptyArtifactID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := Initialize(dir, InitOptions{})
	require.Error(t, err)
	assert.False(t, Exists(dir))
}
