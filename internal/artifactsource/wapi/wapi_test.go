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

	assert.Equal(t, filepath.Join(dir, RootDirName, StateDirName), Dir(dir),
		"state belongs under the directory the CLI resolves, not the legacy root entry")
	assert.NoDirExists(t, filepath.Join(dir, LegacyDirName))

	entries, err := os.ReadDir(Dir(dir))
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
	assert.Contains(t, err.Error(), "artifactId")

	// Initialize mkdirs the .datarobot parent on its way to the state dir, so
	// this is what pins the check ahead of every filesystem change: a rejected
	// call leaves the project exactly as it found it, with no stray parent.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.False(t, Exists(dir))
}

// --- state directory location -------------------------------------------------

func TestDir_FallsBackToLegacy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	legacy := filepath.Join(dir, LegacyDirName)
	require.NoError(t, os.Mkdir(legacy, 0o755))

	assert.Equal(t, legacy, Dir(dir), "an un-migrated project is read where its state still stands")
	assert.True(t, Exists(dir))
}

func TestDir_PrefersCurrentOverLegacy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	current := filepath.Join(dir, RootDirName, StateDirName)
	require.NoError(t, os.MkdirAll(current, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, LegacyDirName), 0o755))

	// Matches the CLI's EnsureMigrated: with both present the current location
	// wins and the two trees are never merged.
	assert.Equal(t, current, Dir(dir))
}

func TestInitialize_AlreadyLinkedAtLegacy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	legacy := filepath.Join(dir, LegacyDirName)
	require.NoError(t, os.Mkdir(legacy, 0o755))

	err := Initialize(dir, InitOptions{ArtifactID: "art-1"})
	assert.ErrorIs(t, err, ErrAlreadyLinked)
	assert.NoDirExists(t, filepath.Join(dir, RootDirName, StateDirName),
		"a legacy-linked project must not acquire a second state directory")
}

func TestRoundTrip_InLegacyDir(t *testing.T) {
	t.Parallel()

	// A tree the CLI linked before the move, which nothing has migrated yet:
	// the provider has to read and write that same config.json.
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, LegacyDirName), 0o755))

	require.NoError(t, SaveConfig(dir, Config{ArtifactID: "art-1", CLIVersion: ProviderWriter}))
	require.NoError(t, SaveManifest(dir, Manifest{Version: ManifestVersion}))

	assert.FileExists(t, filepath.Join(dir, LegacyDirName, configFile))
	assert.NoDirExists(t, filepath.Join(dir, RootDirName))

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "art-1", cfg.ArtifactID)

	m, err := LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, ManifestVersion, m.Version)
}

// --- CLI-written keys the provider does not own -------------------------------

func TestSaveConfig_PreservesLastBuiltVersionID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, Initialize(dir, InitOptions{ArtifactID: "art-1"}))

	// Stand in for a CLI write: the provider never sets this key, but it must
	// survive a load/save round trip because the CLI reads it to decide whether
	// code moved since the last image build.
	cliWritten := `{
  "artifactId": "art-1",
  "catalogId": "cat-1",
  "lastSyncedVersionId": "ver-1",
  "lastBuiltVersionId": "built-42",
  "createdAt": "2026-04-10T09:15:00Z",
  "cliVersion": "1.2.3"
}`
	require.NoError(t, os.WriteFile(configPath(dir), []byte(cliWritten), 0o600))

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg.LastBuiltVersionID)
	assert.Equal(t, "built-42", *cfg.LastBuiltVersionID)

	require.NoError(t, SaveConfig(dir, cfg))

	raw, err := os.ReadFile(configPath(dir))
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	assert.Equal(t, "built-42", parsed["lastBuiltVersionId"])
}

func TestInitialize_WritesNullLastBuiltVersionID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, Initialize(dir, InitOptions{ArtifactID: "art-1"}))

	raw, err := os.ReadFile(configPath(dir))
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))

	// The CLI validates this key with `omitempty`, so an explicit null is valid
	// for a project the provider linked and nothing has built yet.
	require.Contains(t, parsed, "lastBuiltVersionId")
	assert.Nil(t, parsed["lastBuiltVersionId"])
}
