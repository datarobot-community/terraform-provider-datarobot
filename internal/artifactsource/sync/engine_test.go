package sync

// CLI source: cli/internal/workload/sync/engine_test.go — scenarios ported
// (fast path / drift / first sync / locked artifact); fakes rewritten
// against this package's ArtifactStore/filesapi.Client instead of CLI's
// workload.Artifact / cli/internal/drapi/filesapi.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeArtifactStore is the in-memory ArtifactStore used by engine tests.
type fakeArtifactStore struct {
	GetFn func(ctx context.Context, artifactID string) (ArtifactInfo, error)
}

func (f *fakeArtifactStore) Get(ctx context.Context, artifactID string) (ArtifactInfo, error) {
	if f.GetFn == nil {
		return ArtifactInfo{}, errors.New("fakeArtifactStore.Get: no GetFn configured")
	}
	return f.GetFn(ctx, artifactID)
}

// fakeFilesAPI is the in-memory filesapi.Client fake used by engine
// tests. Unexpected methods error loudly so off-happy-path drift fails
// tests instead of silently returning zero values.
type fakeFilesAPI struct {
	allFiles       map[string]filesapi.FileMeta
	allFilesErr    error
	allFilesCalled bool
	lastCatalogID  string
	lastVersionID  string
}

func (f *fakeFilesAPI) CreateCatalog(context.Context) (*filesapi.CatalogResp, error) {
	return nil, errors.New("fakeFilesAPI: CreateCatalog not expected")
}

func (f *fakeFilesAPI) CreateStage(context.Context, string) (*filesapi.StageResp, error) {
	return nil, errors.New("fakeFilesAPI: CreateStage not expected")
}

func (f *fakeFilesAPI) UploadToStage(context.Context, string, string, string, int64, io.Reader) error {
	return errors.New("fakeFilesAPI: UploadToStage not expected")
}

func (f *fakeFilesAPI) ApplyStage(context.Context, string, string, string) (*filesapi.ApplyStageResp, error) {
	return nil, errors.New("fakeFilesAPI: ApplyStage not expected")
}

func (f *fakeFilesAPI) UploadFromZipNew(context.Context, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	return nil, errors.New("fakeFilesAPI: UploadFromZipNew not expected")
}

func (f *fakeFilesAPI) UploadFromZipExisting(context.Context, string, string, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	return nil, errors.New("fakeFilesAPI: UploadFromZipExisting not expected")
}

func (f *fakeFilesAPI) PollStatus(context.Context, string) (*filesapi.StatusResp, error) {
	return nil, errors.New("fakeFilesAPI: PollStatus not expected")
}

func (f *fakeFilesAPI) AllFiles(_ context.Context, catalogID, versionID string) (map[string]filesapi.FileMeta, error) {
	f.allFilesCalled = true
	f.lastCatalogID = catalogID
	f.lastVersionID = versionID
	if f.allFilesErr != nil {
		return nil, f.allFilesErr
	}
	return f.allFiles, nil
}

func (f *fakeFilesAPI) DownloadFile(context.Context, string, string, string, io.Writer) (string, int64, error) {
	return "", 0, errors.New("fakeFilesAPI: DownloadFile not expected")
}

func (f *fakeFilesAPI) DeleteFiles(context.Context, string, []string) (*filesapi.DeleteFilesResp, error) {
	return nil, errors.New("fakeFilesAPI: DeleteFiles not expected")
}

func (f *fakeFilesAPI) ListVersions(context.Context, string, int) ([]filesapi.CatalogVersion, error) {
	return nil, errors.New("fakeFilesAPI: ListVersions not expected")
}

func writeProjectFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for rel, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}
}

func hashContent(body string) (string, int64) {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:]), int64(len(body))
}

func draftInfo(catalogID, versionID string) ArtifactInfo {
	return ArtifactInfo{CatalogID: catalogID, CatalogVersionID: versionID}
}

func TestEngine_New_ValidatesArgs(t *testing.T) {
	t.Parallel()

	_, err := New("", "art-1", &fakeFilesAPI{}, &fakeArtifactStore{})
	assert.ErrorContains(t, err, "projectDir")

	_, err = New(t.TempDir(), "", &fakeFilesAPI{}, &fakeArtifactStore{})
	assert.ErrorContains(t, err, "artifactID")

	_, err = New(t.TempDir(), "art-1", nil, &fakeArtifactStore{})
	assert.ErrorContains(t, err, "files API")

	_, err = New(t.TempDir(), "art-1", &fakeFilesAPI{}, nil)
	assert.ErrorContains(t, err, "artifact store")
}

func TestEngine_Plan_AutoInitsWapiWhenMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "print('hi')\n"})

	assert.False(t, wapi.Exists(dir))

	e, err := New(dir, "art-abc-123", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.NoError(t, err)

	assert.True(t, wapi.Exists(dir))

	cfg, err := wapi.LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "art-abc-123", cfg.ArtifactID)
}

func TestEngine_Plan_FirstSyncEmptyRemote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{
		"agent.py":        "print('hi')\n",
		"utils/helper.py": "def help(): pass\n",
	})

	files := &fakeFilesAPI{}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	uploadPaths := make([]string, 0, len(plan.Uploads))
	for _, fa := range plan.Uploads {
		uploadPaths = append(uploadPaths, fa.Path)
	}
	assert.ElementsMatch(t, []string{"agent.py", "utils/helper.py"}, uploadPaths)
	assert.Empty(t, plan.Downloads)
	assert.Empty(t, plan.Deletes)
	assert.Empty(t, plan.Conflicts)
	assert.False(t, files.allFilesCalled, "AllFiles must not be called when remote catalog version is empty")
}

func TestEngine_Plan_EmptyWhenNotDrifted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	hash, size := hashContent("x")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
	}))

	files := &fakeFilesAPI{}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-1"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, plan.IsEmpty(), "plan should be empty when local == base == remote: %+v", plan)
	assert.False(t, files.allFilesCalled, "AllFiles must not be called when not drifted")
	assert.Equal(t, "ver-1", plan.OldVersionShort)
}

func TestEngine_Plan_DriftedFetchesAllFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "local content"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	baseHash, baseSize := hashContent("base content")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: baseHash, Size: baseSize}},
	}))

	// Remote moved on to ver-2 without us: drifted, so Plan must call
	// AllFiles(ver-2) instead of reusing BASE.
	remoteHash, remoteSize := hashContent("remote content")
	files := &fakeFilesAPI{allFiles: map[string]filesapi.FileMeta{
		"agent.py":     {Hash: remoteHash, Size: remoteSize},
		"new-file.txt": {Hash: "aaa", Size: 3},
	}}

	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, files.allFilesCalled, "AllFiles must be called when drifted")

	require.True(t, plan.HasConflicts())
	assert.Equal(t, "agent.py", plan.Conflicts[0].Path, "local and remote both changed agent.py from BASE")

	require.Len(t, plan.Downloads, 1)
	assert.Equal(t, "new-file.txt", plan.Downloads[0].Path)
}

func TestEngine_Plan_IgnoredFilesAbsentFromLocal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{
		"agent.py":        "print('hi')\n",
		".venv/site.py":   "should be ignored\n",
		".datarobot.yaml": "spec: {}\n",
	})
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".drignore"), []byte(".venv\n"), 0o644))

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	uploadPaths := make([]string, 0, len(plan.Uploads))
	for _, fa := range plan.Uploads {
		uploadPaths = append(uploadPaths, fa.Path)
	}
	// .venv is excluded by the user's .drignore; .datarobot.yaml is a
	// hardcoded system exclude regardless of .drignore contents.
	assert.ElementsMatch(t, []string{"agent.py", ".drignore"}, uploadPaths)
}

func TestEngine_Plan_LockedArtifactRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) {
			return ArtifactInfo{Locked: true}, nil
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLockedArtifact)
	assert.Contains(t, err.Error(), "locked")

	// The lock must be released so a later Plan (after the resource
	// clones to a draft) is not blocked by this failed attempt.
	assert.NoError(t, e.Close())
}

func TestEngine_Plan_AllFilesErrorReleasesLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	files := &fakeFilesAPI{allFilesErr: errors.New("boom")}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)

	_, err = e.Plan(context.Background())
	require.ErrorContains(t, err, "boom")

	// A second Engine must be able to acquire the lock immediately: the
	// failed Plan above must have released it.
	e2, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e2.Close() })

	lock, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NoError(t, lock.Unlock())
}

func TestEngine_Plan_FetchArtifactErrorReleasesLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) {
			return ArtifactInfo{}, errors.New("network down")
		},
	})
	require.NoError(t, err)

	_, err = e.Plan(context.Background())
	require.ErrorContains(t, err, "network down")

	// A fetch failure in gather (phase 1) must still release the lock
	// acquired during preflight (phase 0), same as an AllFiles failure in
	// manifests (phase 2, covered above).
	lock, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NoError(t, lock.Unlock())
}

func TestEngine_Plan_LockContentionAcrossEngines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	artifacts := &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	}

	e1, err := New(dir, "art-1", &fakeFilesAPI{}, artifacts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e1.Close() })

	_, err = e1.Plan(context.Background())
	require.NoError(t, err) // e1 now holds .wapi/sync.lock and never closes it

	e2, err := New(dir, "art-1", &fakeFilesAPI{}, artifacts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e2.Close() })

	// A second Engine (e.g. a concurrent apply, or a retry that
	// constructed a fresh Engine after a prior one leaked its lock) must
	// not be able to plan while e1 still holds the lock.
	_, err = e2.Plan(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLocked)

	require.NoError(t, e1.Close())

	// Once e1 releases, e2 (or a third Engine) can proceed.
	_, err = e2.Plan(context.Background())
	require.NoError(t, err)
}

func TestEngine_Plan_RecoversStaleRollback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{ArtifactID: "art-1"}))
	writeProjectFiles(t, dir, map[string]string{"agent.py": "original content\n"})

	// Simulate a sync that crashed mid-execute: it backed up agent.py
	// before mutating it, but never reached Restore/Discard.
	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())
	require.NoError(t, rt.Backup("agent.py"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.py"), []byte("mid-sync-mutation\n"), 0o644))
	require.True(t, HasRollback(dir))

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.NoError(t, err)

	assert.True(t, e.StaleRollbackRestored())
	assert.False(t, HasRollback(dir), "preflight must remove .wapi/.rollback/ after restoring it")

	restored, err := os.ReadFile(filepath.Join(dir, "agent.py"))
	require.NoError(t, err)
	assert.Equal(t, "original content\n", string(restored), "phase0 must restore the pre-crash content")
}

func TestEngine_Plan_NoStaleRollbackToRestore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.NoError(t, err)
	assert.False(t, e.StaleRollbackRestored())
}

func TestEngine_Plan_ConfigCatalogIDWinsOverArtifactCatalogID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-config",
		LastSyncedVersionID: "ver-1",
	}))

	hash, size := hashContent("x")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
	}))

	files := &fakeFilesAPI{allFiles: map[string]filesapi.FileMeta{"agent.py": {Hash: hash, Size: size}}}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) {
			// The artifact's live code_ref reports a different catalog
			// than the one pinned in .wapi/config.json; config must win
			// (it stays pinned for the artifact's draft lifetime).
			return draftInfo("cat-other-live", "ver-2"), nil
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.NoError(t, err)

	require.True(t, files.allFilesCalled)
	assert.Equal(t, "cat-config", files.lastCatalogID)
	assert.Equal(t, "ver-2", files.lastVersionID)
}

func TestEngine_Plan_LocalAndRemoteDeletesDetected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{
		"keep.py":           "keep",
		"remote-deleted.py": "still-here-locally",
	})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	keepHash, keepSize := hashContent("keep")
	remoteDeletedHash, remoteDeletedSize := hashContent("still-here-locally")
	localDeletedHash, localDeletedSize := hashContent("was-here-in-base")

	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files: map[string]wapi.FileMeta{
			"keep.py":           {Hash: keepHash, Size: keepSize},
			"remote-deleted.py": {Hash: remoteDeletedHash, Size: remoteDeletedSize},
			"local-deleted.py":  {Hash: localDeletedHash, Size: localDeletedSize},
		},
	}))

	// Drifted remote manifest: local-deleted.py is still there remotely
	// (matches BASE, so it downloads-over-delete... no: local removed it
	// and remote didn't touch it -> LOCAL_DELETED); remote-deleted.py is
	// gone remotely while local left it untouched -> REMOTE_DELETED.
	files := &fakeFilesAPI{allFiles: map[string]filesapi.FileMeta{
		"keep.py":          {Hash: keepHash, Size: keepSize},
		"local-deleted.py": {Hash: localDeletedHash, Size: localDeletedSize},
	}}

	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	deletePaths := make([]string, 0, len(plan.Deletes))
	for _, fa := range plan.Deletes {
		deletePaths = append(deletePaths, fa.Path)
	}
	assert.ElementsMatch(t, []string{"local-deleted.py", "remote-deleted.py"}, deletePaths)
	assert.Empty(t, plan.Uploads)
	assert.Empty(t, plan.Conflicts)
	assert.Empty(t, plan.Downloads)
}

func TestEngine_Plan_RepeatPlanKeepsHeldLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	artifacts := &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	}

	e, err := New(dir, "art-1", &fakeFilesAPI{}, artifacts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	first, err := e.Plan(context.Background())
	require.NoError(t, err)
	require.Len(t, first.Uploads, 1)

	// Re-planning on an Engine that already holds the lock must reuse it
	// rather than flock against itself and then release on the way out.
	writeProjectFiles(t, dir, map[string]string{"helper.py": "y"})
	second, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.Len(t, second.Uploads, 2, "re-Plan must pick up files added since the first Plan")

	// The lock survived the second Plan: a separate Engine still can't take it.
	other, err := New(dir, "art-1", &fakeFilesAPI{}, artifacts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = other.Close() })

	_, err = other.Plan(context.Background())
	assert.ErrorIs(t, err, ErrLocked, "re-Plan must not have dropped the lock held since the first Plan")

	require.NoError(t, e.Close())

	_, err = other.Plan(context.Background())
	assert.NoError(t, err)
}

func TestEngine_Plan_RestoresRollbackOnlyUnderLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{ArtifactID: "art-1"}))
	writeProjectFiles(t, dir, map[string]string{"agent.py": "original content\n"})

	// Stand in for another process mid-execute: it holds .wapi/sync.lock
	// and has an active rollback tree protecting its in-flight write.
	holder, err := AcquireLock(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = holder.Unlock() })

	rt := NewRollbackTree(dir)
	require.NoError(t, rt.Init())
	require.NoError(t, rt.Backup("agent.py"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.py"), []byte("in-flight write\n"), 0o644))

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	assert.ErrorIs(t, err, ErrLocked)

	// Preflight bailed at the lock, so the holder's rollback tree and its
	// in-flight write are both untouched.
	assert.True(t, HasRollback(dir), "must not restore a rollback tree owned by the lock holder")
	assert.False(t, e.StaleRollbackRestored())

	body, err := os.ReadFile(filepath.Join(dir, "agent.py"))
	require.NoError(t, err)
	assert.Equal(t, "in-flight write\n", string(body), "must not roll back the lock holder's write")
}

func TestEngine_Plan_AutoInitToleratesAlreadyLinked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	// Drive the lost-auto-init-race branch deterministically: with the
	// state path occupied by a regular file, wapi.Exists reports false
	// (not a directory) while Initialize's os.Mkdir fails with ErrExist
	// and so returns ErrAlreadyLinked — exactly what a concurrent first
	// Plan sees when another one created .wapi/ between those two calls.
	// The real race can't be scheduled from a test.
	stateDir := filepath.Join(dir, wapi.RootDirName, wapi.StateDirName)
	require.NoError(t, os.MkdirAll(filepath.Dir(stateDir), 0o755))
	require.NoError(t, os.WriteFile(stateDir, []byte("occupied"), 0o644))

	require.False(t, wapi.Exists(dir), "guard the precondition this test stands on")
	require.ErrorIs(t,
		wapi.Initialize(dir, wapi.InitOptions{ArtifactID: "art-1"}),
		wapi.ErrAlreadyLinked,
		"guard the precondition this test stands on")

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	// preflight must walk past ErrAlreadyLinked to the lock step and fail
	// there (this contrived tree has no usable state directory) instead of
	// bailing out at auto-init.
	err = e.preflight()
	require.Error(t, err)
	assert.NotErrorIs(t, err, wapi.ErrAlreadyLinked)
	assert.NotContains(t, err.Error(), "auto-init",
		"losing the auto-init race must not fail preflight")
	assert.Contains(t, err.Error(), "sync state directory")
}
