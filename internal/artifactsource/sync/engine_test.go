package sync

// CLI source: cli/internal/workload/sync/engine_test.go — scenarios ported
// (fast path / drift / first sync / locked artifact); fakes rewritten
// against this package's ArtifactStore/filesapi.Client instead of CLI's
// workload.Artifact / cli/internal/drapi/filesapi.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeArtifactStore is the in-memory ArtifactStore used by engine tests.
type fakeArtifactStore struct {
	GetFn   func(ctx context.Context, artifactID string) (ArtifactInfo, error)
	PatchFn func(ctx context.Context, artifactID, catalogID, catalogVersionID string) error

	// patches records every PatchCodeRef call, in order.
	patches []patchCall
}

// patchCall is one recorded PatchCodeRef call.
type patchCall struct {
	ArtifactID       string
	CatalogID        string
	CatalogVersionID string
}

func (f *fakeArtifactStore) PatchCodeRef(ctx context.Context, artifactID, catalogID, catalogVersionID string) error {
	f.patches = append(f.patches, patchCall{
		ArtifactID:       artifactID,
		CatalogID:        catalogID,
		CatalogVersionID: catalogVersionID,
	})

	if f.PatchFn == nil {
		return nil
	}

	return f.PatchFn(ctx, artifactID, catalogID, catalogVersionID)
}

func (f *fakeArtifactStore) Get(ctx context.Context, artifactID string) (ArtifactInfo, error) {
	if f.GetFn == nil {
		return ArtifactInfo{}, errors.New("fakeArtifactStore.Get: no GetFn configured")
	}
	return f.GetFn(ctx, artifactID)
}

// fakeFilesAPI is the in-memory filesapi.Client fake used by engine and
// execute tests. Unexpected methods error loudly so off-happy-path drift
// fails tests instead of silently returning zero values.
type fakeFilesAPI struct {
	allFiles       map[string]filesapi.FileMeta
	allFilesErr    error
	allFilesCalled bool
	lastCatalogID  string
	lastVersionID  string

	// allFilesByVersion serves a listing per catalog version, for tests
	// where BASE and REMOTE are read at different versions; a version with
	// no entry gets allFiles. allFilesErrByVersion fails one version's
	// listing only. allFilesVersions records every version listed, in call
	// order.
	allFilesByVersion    map[string]map[string]filesapi.FileMeta
	allFilesErrByVersion map[string]error
	allFilesVersions     []string

	// blobs is the remote file content DownloadFile serves, keyed by
	// path; a path with no entry (and no downloadErr) is a test bug and
	// fails the download. downloadErr forces a per-path failure.
	blobs       map[string]string
	downloadErr map[string]error

	// downloadHook, when set, runs at the start of every DownloadFile
	// call. Tests use it to order otherwise-concurrent downloads against
	// each other; it runs outside mu because it is allowed to block.
	downloadHook func(path string)

	// Upload and delete plumbing for the remote half. Each method stays
	// loud ("not expected") until the test opts it in by setting the
	// response it should return.
	newCatalogID    string // CreateCatalog result
	stageID         string // CreateStage result
	stageVersionID  string // ApplyStage result
	stageNoVersion  bool   // ApplyStage succeeds but reports no version
	zipVersionID    string // UploadFromZipExisting result
	deleteVersionID string // DeleteFiles result
	deleteNoVersion bool   // DeleteFiles succeeds but reports no version
	uploadErr       error  // fails every UploadToStage call

	// versions is the catalog history ListVersions serves, newest first.
	// Nil serves the ids the tests in this package use for drift, in the
	// order their names suggest, so a fixture that only needs "the live
	// version is newer than the last synced one" says nothing.
	versions []string

	mu                 sync.Mutex
	downloadPaths      []string
	staged             map[string]string // stage path -> uploaded bytes
	stageCatalogIDs    []string
	deletedPaths       [][]string
	createCatalogCalls int
	zipUploads         int
}

func (f *fakeFilesAPI) CreateCatalog(context.Context) (*filesapi.CatalogResp, error) {
	if f.newCatalogID == "" {
		return nil, errors.New("fakeFilesAPI: CreateCatalog not expected")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.createCatalogCalls++

	return &filesapi.CatalogResp{CatalogID: f.newCatalogID}, nil
}

func (f *fakeFilesAPI) CreateStage(_ context.Context, catalogID string) (*filesapi.StageResp, error) {
	if f.stageID == "" {
		return nil, errors.New("fakeFilesAPI: CreateStage not expected")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.stageCatalogIDs = append(f.stageCatalogIDs, catalogID)

	return &filesapi.StageResp{CatalogID: catalogID, StageID: f.stageID}, nil
}

// UploadToStage records the bytes it was handed so tests can assert that
// the upload set carries local content, not just local paths. Workers run
// in parallel, hence the mutex.
func (f *fakeFilesAPI) UploadToStage(_ context.Context, _, _, name string, _ int64, body io.Reader) error {
	if f.uploadErr != nil {
		return f.uploadErr
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.staged == nil {
		f.staged = make(map[string]string)
	}
	f.staged[name] = string(data)

	return nil
}

func (f *fakeFilesAPI) ApplyStage(context.Context, string, string, string) (*filesapi.ApplyStageResp, error) {
	if f.stageNoVersion {
		return &filesapi.ApplyStageResp{}, nil
	}
	if f.stageVersionID == "" {
		return nil, errors.New("fakeFilesAPI: ApplyStage not expected")
	}

	return &filesapi.ApplyStageResp{CatalogVersionID: f.stageVersionID}, nil
}

func (f *fakeFilesAPI) UploadFromZipNew(context.Context, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	return nil, errors.New("fakeFilesAPI: UploadFromZipNew not expected")
}

// UploadFromZipExisting returns no StatusID, so the upload completes
// inline and PollStatus stays unexpected.
func (f *fakeFilesAPI) UploadFromZipExisting(_ context.Context, catalogID, _, _ string, _ int64, _ io.Reader) (*filesapi.FromFileResp, error) {
	if f.zipVersionID == "" {
		return nil, errors.New("fakeFilesAPI: UploadFromZipExisting not expected")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.zipUploads++

	return &filesapi.FromFileResp{CatalogID: catalogID, CatalogVersionID: f.zipVersionID}, nil
}

func (f *fakeFilesAPI) PollStatus(context.Context, string) (*filesapi.StatusResp, error) {
	return nil, errors.New("fakeFilesAPI: PollStatus not expected")
}

func (f *fakeFilesAPI) AllFiles(_ context.Context, catalogID, versionID string) (map[string]filesapi.FileMeta, error) {
	f.allFilesCalled = true
	f.lastCatalogID = catalogID
	f.lastVersionID = versionID
	f.allFilesVersions = append(f.allFilesVersions, versionID)
	if f.allFilesErr != nil {
		return nil, f.allFilesErr
	}
	if err := f.allFilesErrByVersion[versionID]; err != nil {
		return nil, err
	}
	if listing, ok := f.allFilesByVersion[versionID]; ok {
		return listing, nil
	}
	return f.allFiles, nil
}

func (f *fakeFilesAPI) DownloadFile(_ context.Context, _, _, path string, w io.Writer) (string, int64, error) {
	f.mu.Lock()
	f.downloadPaths = append(f.downloadPaths, path)
	f.mu.Unlock()

	if f.downloadHook != nil {
		f.downloadHook(path)
	}

	if err := f.downloadErr[path]; err != nil {
		return "", 0, err
	}

	body, ok := f.blobs[path]
	if !ok {
		return "", 0, fmt.Errorf("fakeFilesAPI: no remote content configured for %q", path)
	}

	n, err := io.WriteString(w, body)

	return "", int64(n), err
}

// downloadedPaths returns the paths DownloadFile served, sorted so
// assertions do not depend on worker scheduling.
func (f *fakeFilesAPI) downloadedPaths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := append([]string(nil), f.downloadPaths...)
	sort.Strings(out)

	return out
}

func (f *fakeFilesAPI) DeleteFiles(_ context.Context, catalogID string, paths []string) (*filesapi.DeleteFilesResp, error) {
	if f.deleteVersionID == "" && !f.deleteNoVersion {
		return nil, errors.New("fakeFilesAPI: DeleteFiles not expected")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.deletedPaths = append(f.deletedPaths, append([]string(nil), paths...))

	if f.deleteNoVersion {
		return &filesapi.DeleteFilesResp{CatalogID: catalogID}, nil
	}

	return &filesapi.DeleteFilesResp{
		CatalogID:        catalogID,
		CatalogVersionID: f.deleteVersionID,
		NumFiles:         len(paths),
	}, nil
}

// stagedPaths returns the paths UploadToStage received, sorted so
// assertions do not depend on worker scheduling.
func (f *fakeFilesAPI) stagedPaths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0, len(f.staged))
	for path := range f.staged {
		out = append(out, path)
	}
	sort.Strings(out)

	return out
}

func (f *fakeFilesAPI) ListVersions(context.Context, string, int) ([]filesapi.CatalogVersion, error) {
	ids := f.versions
	if ids == nil {
		ids = []string{"ver-9", "ver-3", "ver-2", "ver-1"}
	}

	out := make([]filesapi.CatalogVersion, 0, len(ids))
	for _, id := range ids {
		out = append(out, filesapi.CatalogVersion{ID: id})
	}

	return out, nil
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

// listing is a catalog version's file list as AllFiles returns it, built
// from path -> content.
func listing(files map[string]string) map[string]filesapi.FileMeta {
	out := make(map[string]filesapi.FileMeta, len(files))
	for path, body := range files {
		hash, size := hashContent(body)
		out[path] = filesapi.FileMeta{Hash: hash, Size: size}
	}

	return out
}

func actionPaths(actions []FileAction) []string {
	out := make([]string, 0, len(actions))
	for _, fa := range actions {
		out = append(out, fa.Path)
	}

	return out
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

func TestEngine_Plan_LockedArtifactStillPlans(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) {
			return ArtifactInfo{Locked: true}, nil
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	// CLI parity (phase1_gather.go's previewOnly exemption): a locked
	// artifact must not refuse the plan, because the caller needs the diff
	// to decide whether a new version is required at all. The rejection
	// belongs to Execute.
	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.Len(t, plan.Uploads, 1, "the plan must still count the files a sync would move")
	assert.True(t, e.ArtifactLocked(), "Plan must report the lock instead of failing on it")
}

func TestEngine_Plan_LockedArtifactEmptyPlanNeedsNoClone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	agentHash, agentSize := hashContent("x")
	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: agentHash, Size: agentSize}},
	}))

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) {
			return ArtifactInfo{Locked: true, CatalogID: "cat-1", CatalogVersionID: "ver-1"}, nil
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	// The case the preview exemption exists for: in sync with a locked
	// artifact means no write is needed, so the caller can skip minting a
	// new version entirely. Rejecting at Plan would have hidden that.
	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, plan.IsEmpty())
	assert.True(t, e.ArtifactLocked())
}

func TestEngine_Plan_RejectsCatalogMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	// State pinned to cat-1 by an earlier sync, then an Engine for an
	// artifact whose code lives somewhere else: an import of an unrelated
	// artifact into a resource that reuses this directory, for instance.
	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	files := &fakeFilesAPI{}
	e, err := New(dir, "art-2", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-2", "ver-9"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	// The directory is bound to this resource's previous artifact: a clone, not a second resource.
	e.PreviousArtifact("art-1")

	_, err = e.Plan(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCatalogMismatch)
	assert.Contains(t, err.Error(), "cat-1")
	assert.Contains(t, err.Error(), "cat-2")
	assert.Contains(t, err.Error(), "art-2")
	assert.False(t, files.allFilesCalled, "must fail before diffing against the wrong catalog")

	// The lock must be released so the caller can retry after re-linking.
	lock, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NoError(t, lock.Unlock())
}

func TestEngine_Plan_RebindsArtifactWithoutCodeRefToPinnedCatalog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	// A resource destroyed and re-created over a synced directory: the new
	// artifact has no code yet, and the directory's catalog is still the
	// right place for it. The rebind is allowed and BASE is kept.
	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-old",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))
	hash, size := hashContent("x")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
	}))

	e, err := New(dir, "art-new", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	// The directory is bound to this resource's previous artifact: a clone, not a second resource.
	e.PreviousArtifact("art-old")

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, plan.IsEmpty(), "BASE still describes this directory: %+v", plan)
	assert.Equal(t, "cat-1", e.catalogID)
	assert.Equal(t, "art-new", e.config.ArtifactID)
}

func TestEngine_Plan_UsesInjectedIgnoreMatcher(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{
		"agent.py":       "x",
		".venv/lib/a.py": "vendored",
	})

	store := &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	}

	// No ignore file on disk: the directory alone would upload .venv.
	bare, err := New(dir, "art-1", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	barePlan, err := bare.Plan(context.Background())
	require.NoError(t, err)
	require.NoError(t, bare.Close())
	require.Len(t, barePlan.Uploads, 2, "guard the precondition this test stands on")

	// The resource hashed the tree with the template's patterns and could
	// not write them to disk; the sync has to walk with the same set.
	seeded, err := New(dir, "art-1", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	t.Cleanup(func() { _ = seeded.Close() })
	seeded.UseIgnore(ignore.FromDefaultTemplate())

	plan, err := seeded.Plan(context.Background())
	require.NoError(t, err)
	require.Len(t, plan.Uploads, 1)
	assert.Equal(t, "agent.py", plan.Uploads[0].Path)
}

func TestEngine_Plan_DriftedWithoutCatalogSkipsAllFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		LastSyncedVersionID: "ver-1",
	}))

	files := &fakeFilesAPI{}

	// Drifted (live version != last synced) but no catalog ID resolves
	// from either config or the artifact. CLI parity: REMOTE is empty
	// rather than an AllFiles("", ver) call that can only fail.
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.False(t, files.allFilesCalled, "must not call AllFiles with an empty catalog ID")
	assert.Len(t, plan.Uploads, 1)
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
	// A recorded BASE, so the failing listing is REMOTE's, not BASE's.
	synced := "ver-1"
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{Version: wapi.ManifestVersion, SyncedVersionID: &synced}))

	files := &fakeFilesAPI{allFilesErr: errors.New("boom")}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)

	_, err = e.Plan(context.Background())
	require.ErrorContains(t, err, "fetch remote manifest")
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

func TestEngine_Plan_DriftUsesPinnedCatalogAndLiveVersion(t *testing.T) {
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
			// Another writer moved the artifact's code_ref to a newer
			// version of the pinned catalog. The version is what marks
			// the remote as drifted; the catalog stays the pinned one
			// (a code_ref in a different catalog is ErrCatalogMismatch).
			return draftInfo("cat-config", "ver-2"), nil
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

// The tests below cover the Terraform-specific bindings the CLI has no
// equivalent for: the artifact ID follows the resource across artifact
// versions, and the catalog pointers can be seeded from Terraform state
// for a tree that predates .wapi/.

func TestEngine_Plan_RebindsArtifactIDFromCaller(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-old",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	hash, size := hashContent("x")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
	}))

	var gotID string
	e, err := New(dir, "art-new", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(_ context.Context, artifactID string) (ArtifactInfo, error) {
			gotID = artifactID
			return draftInfo("cat-1", "ver-1"), nil
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	// The directory is bound to this resource's previous artifact: a clone, not a second resource.
	e.PreviousArtifact("art-old")

	_, err = e.Plan(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "art-new", gotID, "the caller's artifact ID must win over the one recorded in .wapi/")
	assert.Equal(t, "art-new", e.config.ArtifactID)
}

func TestEngine_Plan_ClonedArtifactDiffsAgainstLastSyncedVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-old",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	hash, size := hashContent("x")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
	}))

	files := &fakeFilesAPI{}
	// A draft just cloned from a locked artifact: no code_ref of its own.
	e, err := New(dir, "art-clone", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	// The directory is bound to this resource's previous artifact: a clone, not a second resource.
	e.PreviousArtifact("art-old")

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	assert.True(t, plan.IsEmpty(), "an unchanged tree must not re-upload just because the artifact is new: %+v", plan)
	assert.False(t, files.allFilesCalled, "the clone is not drifted, so REMOTE comes from BASE")
	assert.Equal(t, "ver-1", e.remoteVer, "REMOTE is the version this directory last pushed")
	assert.Empty(t, e.artifactVer, "the clone's own code_ref is still empty")
}

func TestEngine_ExecuteRemote_PointsClonedArtifactAtLastSyncedVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-old",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	hash, size := hashContent("x")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
	}))

	files := &fakeFilesAPI{}
	artifacts := &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	}

	e, err := New(dir, "art-clone", files, artifacts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	// The directory is bound to this resource's previous artifact: a clone, not a second resource.
	e.PreviousArtifact("art-old")

	result, err := e.Run(context.Background())
	require.NoError(t, err)

	// Nothing changed on either side, but the clone still has to be
	// pointed at the code its directory matches.
	assert.Equal(t, []patchCall{{ArtifactID: "art-clone", CatalogID: "cat-1", CatalogVersionID: "ver-1"}}, artifacts.patches)
	assert.Zero(t, result.Uploaded)
	assert.Equal(t, "cat-1", result.CatalogID)
	assert.Equal(t, "ver-1", result.CatalogVersionID)

	cfg, err := wapi.LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "art-clone", cfg.ArtifactID, "phase 6 persists the re-bound artifact ID")
}

func TestEngine_BindCatalog_SeedsWapiForATreeWithoutIt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	files := &fakeFilesAPI{allFiles: listing(map[string]string{"agent.py": "x"})}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-state", "ver-state"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	// Terraform state says the tree already has code in cat-state; the
	// first engine run must sync with that catalog, not create a new one.
	e.BindCatalog("cat-state", "ver-state")

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "cat-state", e.catalogID)
	assert.Equal(t, []string{"ver-state"}, files.allFilesVersions,
		"BASE is the seeded version's file list; the catalog is not drifted, so that one listing is REMOTE too")
	assert.Equal(t, "cat-state", files.lastCatalogID)
	assert.True(t, plan.IsEmpty(), "the tree matches the version it was seeded from: %+v", plan)

	cfg, err := wapi.LoadConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg.CatalogID)
	assert.Equal(t, "cat-state", *cfg.CatalogID)
	require.NotNil(t, cfg.LastSyncedVersionID)
	assert.Equal(t, "ver-state", *cfg.LastSyncedVersionID)
}

func TestEngine_BindCatalog_IgnoredOnceWapiExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-existing",
		LastSyncedVersionID: "ver-1",
	}))

	e, err := New(dir, "art-1", &fakeFilesAPI{}, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-existing", "ver-1"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	e.BindCatalog("cat-seed", "ver-seed")

	_, err = e.Plan(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "cat-existing", e.catalogID, ".wapi/config.json wins once it exists")
}

func TestEngine_BindCatalog_VersionIsTheBaselineForAnArtifactWithoutCodeRef(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})

	// The directory last synced ver-1 ...
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

	// ... but the resource's record of the locked artifact being cloned
	// says ver-2, which also holds helper.py. The clone itself has no
	// code_ref yet.
	pulledHash, pulledSize := hashContent("pulled")
	files := &fakeFilesAPI{
		allFiles: map[string]filesapi.FileMeta{
			"agent.py":  {Hash: hash, Size: size},
			"helper.py": {Hash: pulledHash, Size: pulledSize},
		},
		blobs: map[string]string{"helper.py": "pulled"},
	}
	e, err := New(dir, "art-clone", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	// The directory is bound to this resource's previous artifact: a clone, not a second resource.
	e.PreviousArtifact("art-1")

	e.BindCatalog("cat-1", "ver-2")

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	assert.True(t, files.allFilesCalled, "the directory is behind the resource, so REMOTE is fetched")
	assert.Equal(t, "ver-2", files.lastVersionID)
	assert.Empty(t, plan.Uploads)
	require.Len(t, plan.Downloads, 1)
	assert.Equal(t, "helper.py", plan.Downloads[0].Path)
}

// unsyncedPlan plans local against a directory that has never synced, the
// shape of a fresh clone or a CI checkout: no state directory, and
// Terraform state pinning cat-1 at ver-1. The artifact reports remoteVer.
func unsyncedPlan(t *testing.T, local map[string]string, remoteVer string, files *fakeFilesAPI) (*Engine, *SyncPlan, error) {
	t.Helper()

	dir := t.TempDir()
	writeProjectFiles(t, dir, local)

	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", remoteVer), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	e.BindCatalog("cat-1", "ver-1")

	plan, err := e.Plan(context.Background())

	return e, plan, err
}

func TestEngine_Plan_UnsyncedDirectoryReadsBaseFromPinnedVersion(t *testing.T) {
	t.Parallel()

	files := &fakeFilesAPI{allFiles: listing(map[string]string{
		"agent.py":  "same",
		"edited.py": "old",
		"gone.py":   "deleted locally",
	})}

	e, plan, err := unsyncedPlan(t, map[string]string{"agent.py": "same", "edited.py": "new"}, "ver-1", files)
	require.NoError(t, err)

	assert.Equal(t, []string{"ver-1"}, files.allFilesVersions, "one listing serves BASE and the undrifted REMOTE")
	require.Len(t, plan.Deletes, 1)
	assert.Equal(t, "gone.py", plan.Deletes[0].Path)
	assert.Equal(t, ClsLocalDeleted, plan.Deletes[0].Classification)
	assert.Equal(t, ActUploadDelete, plan.Deletes[0].Action)
	assert.Equal(t, []string{"edited.py"}, actionPaths(plan.Uploads), "an unchanged file is not uploaded again")
	assert.Empty(t, plan.Downloads)
	assert.Empty(t, plan.Conflicts)
	assert.Equal(t, "ver-1", e.AdoptedBaseVersion())

	// The adopted BASE is recorded against the pinned version, so the CLI
	// and a later Plan read it from disk instead of an empty manifest.
	manifest, err := wapi.LoadManifest(e.projectDir)
	require.NoError(t, err)
	require.NotNil(t, manifest.SyncedVersionID)
	assert.Equal(t, "ver-1", *manifest.SyncedVersionID)
	assert.NotNil(t, manifest.SyncedAt)
	assert.ElementsMatch(t, []string{"agent.py", "edited.py", "gone.py"}, mapKeys(manifest.Files))

	require.NoError(t, e.Close())
	again, err := New(e.projectDir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-1"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = again.Close() })
	replan, err := again.Plan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"ver-1"}, files.allFilesVersions, "a recorded BASE is not listed again")
	assert.Equal(t, "ver-1", again.AdoptedBaseVersion(), "a retry before any sync still reports the adopted BASE")
	assert.Equal(t, actionPaths(plan.Deletes), actionPaths(replan.Deletes))
}

func TestEngine_Plan_AdoptedBaseMarkerNeedsMatchingVersion(t *testing.T) {
	t.Parallel()

	// The CLI drops keys it does not know when it saves, but a manifest
	// that names another version than the marker was rewritten by a sync
	// either way, so its BASE was not adopted.
	dir := linkedDir(t, "art-1")
	manifest, err := wapi.LoadManifest(dir)
	require.NoError(t, err)
	manifest.Extra = withAdoptedBaseMarker(map[string]json.RawMessage{"cliKey": json.RawMessage(`true`)}, "ver-0")
	require.NoError(t, wapi.SaveManifest(dir, manifest))

	files := &fakeFilesAPI{allFiles: listing(map[string]string{"agent.py": "x"})}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-1"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.NoError(t, err)
	assert.Empty(t, e.AdoptedBaseVersion())
	assert.Equal(t, map[string]json.RawMessage{"cliKey": json.RawMessage(`true`)}, e.baseExtra, "the marker is dropped, other keys are kept")
}

func mapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

func TestEngine_Plan_UnsyncedDirectoryDriftedKeepsRemoteAddition(t *testing.T) {
	t.Parallel()

	// State pins ver-1; since then someone added colleague.py, and the
	// artifact now points at ver-3.
	pinned := map[string]string{"agent.py": "same", "gone.py": "deleted locally"}
	live := map[string]string{"agent.py": "same", "gone.py": "deleted locally", "colleague.py": "remote add"}
	files := &fakeFilesAPI{allFilesByVersion: map[string]map[string]filesapi.FileMeta{
		"ver-1": listing(pinned),
		"ver-3": listing(live),
	}}

	_, plan, err := unsyncedPlan(t, map[string]string{"agent.py": "same"}, "ver-3", files)
	require.NoError(t, err)

	assert.Equal(t, []string{"ver-1", "ver-3"}, files.allFilesVersions, "BASE at the pinned version, REMOTE at the artifact's")
	require.Len(t, plan.Deletes, 1)
	assert.Equal(t, "gone.py", plan.Deletes[0].Path)
	assert.Equal(t, ActUploadDelete, plan.Deletes[0].Action, "a local deletion goes to the catalog instead of being downloaded back")
	require.Len(t, plan.Downloads, 1)
	assert.Equal(t, "colleague.py", plan.Downloads[0].Path)
	assert.Equal(t, ClsRemoteAdded, plan.Downloads[0].Classification)
	assert.Empty(t, plan.Uploads)
}

func TestEngine_Plan_UnsyncedDirectoryDeletesIgnoredCatalogPath(t *testing.T) {
	t.Parallel()

	// local.tfvars is a system exclude: it is on disk but never walked. A
	// recorded BASE holding it would delete it from the catalog, and an
	// adopted one does the same.
	files := &fakeFilesAPI{allFiles: listing(map[string]string{"agent.py": "same", "local.tfvars": "secret = 1"})}

	_, plan, err := unsyncedPlan(t, map[string]string{"agent.py": "same", "local.tfvars": "secret = 1"}, "ver-1", files)
	require.NoError(t, err)

	assert.Equal(t, []string{"local.tfvars"}, actionPaths(plan.Deletes))
	assert.Empty(t, plan.Uploads)
}

func TestEngine_Plan_RecordedEmptyBaseIsKept(t *testing.T) {
	t.Parallel()

	// A sync that removed the last file records an empty BASE; it is not
	// the seeded placeholder and must not be replaced.
	dir := linkedDir(t, "art-1")
	synced := "ver-1"
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{Version: wapi.ManifestVersion, SyncedVersionID: &synced}))

	files := &fakeFilesAPI{allFiles: listing(map[string]string{"agent.py": "x"})}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-1"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	assert.Empty(t, files.allFilesVersions)
	assert.Equal(t, []string{"agent.py"}, actionPaths(plan.Uploads))
}

func TestEngine_Plan_CatalogPinnedWithoutVersionSkipsBaseRead(t *testing.T) {
	t.Parallel()

	// What `dr artifact code init` writes for an artifact that has code:
	// the catalog, but no version. The merge against an empty BASE stays
	// the CLI's. (A new artifact, with nothing pinned at all, is
	// TestEngine_Plan_FirstSyncEmptyRemote.)
	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "local"})
	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{ArtifactID: "art-1", CatalogID: "cat-1"}))

	files := &fakeFilesAPI{allFiles: listing(map[string]string{"agent.py": "remote"})}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-3"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"ver-3"}, files.allFilesVersions, "only REMOTE is listed")
	require.Len(t, plan.Conflicts, 1)
	assert.Equal(t, ClsAddConflict, plan.Conflicts[0].Classification)
}

func TestEngine_Plan_PinnedVersionGoneFallsBackToEmptyBase(t *testing.T) {
	t.Parallel()

	files := &fakeFilesAPI{allFilesErr: fmt.Errorf("GET request ... : response 404 Not Found: %w", filesapi.ErrNotFound)}

	e, plan, err := unsyncedPlan(t, map[string]string{"agent.py": "x", "util.py": "y"}, "ver-1", files)
	require.NoError(t, err)

	assert.Equal(t, "ver-1", e.UnreadableBaseVersion())
	assert.Empty(t, e.AdoptedBaseVersion())
	assert.ElementsMatch(t, []string{"agent.py", "util.py"}, actionPaths(plan.Uploads), "an empty BASE pushes the tree")
	assert.Empty(t, plan.Deletes)

	manifest, err := wapi.LoadManifest(e.projectDir)
	require.NoError(t, err)
	assert.Nil(t, manifest.SyncedVersionID, "nothing was read, so nothing is recorded")
}

func TestEngine_Plan_PinnedVersionGoneDriftedDownloadsDeletedFile(t *testing.T) {
	t.Parallel()

	// With no BASE to diff against, a file deleted locally that the moved
	// catalog still holds reads as a remote addition, as before the fix.
	files := &fakeFilesAPI{
		allFilesErrByVersion: map[string]error{"ver-1": fmt.Errorf("response 410 Gone: %w", filesapi.ErrNotFound)},
		allFilesByVersion: map[string]map[string]filesapi.FileMeta{
			"ver-3": listing(map[string]string{"agent.py": "same", "gone.py": "deleted locally"}),
		},
	}

	e, plan, err := unsyncedPlan(t, map[string]string{"agent.py": "same"}, "ver-3", files)
	require.NoError(t, err)

	assert.Equal(t, "ver-1", e.UnreadableBaseVersion())
	assert.Equal(t, []string{"gone.py"}, actionPaths(plan.Downloads))
	assert.Empty(t, plan.Deletes)
}

func TestEngine_Plan_BaseReadErrorFailsAndReleasesLock(t *testing.T) {
	t.Parallel()

	files := &fakeFilesAPI{allFilesErr: errors.New("boom")}

	e, _, err := unsyncedPlan(t, map[string]string{"agent.py": "x"}, "ver-1", files)
	require.ErrorContains(t, err, "fetch base manifest for catalog version ver-1")
	require.ErrorContains(t, err, "boom")
	assert.Empty(t, e.UnreadableBaseVersion())

	lock, err := AcquireLock(e.projectDir)
	require.NoError(t, err, "a failed Plan must release the sync lock")
	require.NoError(t, lock.Unlock())
}

// linkedDir is a project directory with one file, whose state directory
// last synced ver-1 of cat-1 with that file in BASE.
func linkedDir(t *testing.T, artifactID string) string {
	t.Helper()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "x"})
	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          artifactID,
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))
	hash, size := hashContent("x")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
	}))

	return dir
}

func TestEngine_Plan_RefusesRolledBackCatalog(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-1")
	// BASE holds agent.py and extra.py from ver-2; the artifact was
	// re-pointed at ver-1, which never had extra.py.
	writeProjectFiles(t, dir, map[string]string{"extra.py": "only in ver-2"})
	hash, size := hashContent("x")
	extraHash, extraSize := hashContent("only in ver-2")
	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   map[string]wapi.FileMeta{"agent.py": {Hash: hash, Size: size}, "extra.py": {Hash: extraHash, Size: extraSize}},
	}))
	cfg, err := wapi.LoadConfig(dir)
	require.NoError(t, err)
	two := "ver-2"
	cfg.LastSyncedVersionID = &two
	require.NoError(t, wapi.SaveConfig(dir, cfg))

	files := &fakeFilesAPI{
		allFiles: map[string]filesapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
		versions: []string{"ver-2", "ver-1"},
	}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-1"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.ErrorIs(t, err, ErrCatalogRolledBack)
	assert.ErrorContains(t, err, "older than the version this directory last synced")

	// Refused before anything was read or written: extra.py is still
	// there, and the state directory still says ver-2.
	assert.Equal(t, "only in ver-2", readProjectFile(t, dir, "extra.py"))
	after, err := wapi.LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "ver-2", *after.LastSyncedVersionID)
	assert.False(t, files.allFilesCalled, "no REMOTE is fetched for a merge that is refused")

	// A failed Plan releases the lock.
	lock, err := AcquireLock(dir)
	require.NoError(t, err)
	require.NoError(t, lock.Unlock())
}

func TestEngine_Plan_RefusesUnorderableDrift(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-1")
	files := &fakeFilesAPI{
		allFiles: map[string]filesapi.FileMeta{},
		versions: []string{"ver-7"}, // neither ver-2 (live) nor ver-1 (last synced) is in the history served
	}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.ErrorContains(t, err, "could not place versions")
	assert.False(t, files.allFilesCalled)
}

func TestEngine_Plan_ForwardDriftPassesTheOrderCheck(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-1")
	hash, size := hashContent("x")
	files := &fakeFilesAPI{
		allFiles: map[string]filesapi.FileMeta{"agent.py": {Hash: hash, Size: size}},
		versions: []string{"ver-2", "ver-1"},
	}
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, files.allFilesCalled)
	assert.True(t, plan.IsEmpty())
}

func TestEngine_Plan_RefusesDirectoryBoundToAnotherLiveArtifact(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-1")
	var asked []string
	store := &fakeArtifactStore{
		GetFn: func(_ context.Context, id string) (ArtifactInfo, error) {
			asked = append(asked, id)
			return draftInfo("cat-1", "ver-1"), nil // art-1 is alive
		},
	}
	e, err := New(dir, "art-2", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.ErrorIs(t, err, ErrDirectoryOwned)
	assert.ErrorContains(t, err, "already backs artifact art-1")
	assert.Equal(t, []string{"art-2", "art-1"}, asked, "the artifact being synced, then the owner")

	cfg, err := wapi.LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "art-1", cfg.ArtifactID, "the refusal leaves the binding alone")
}

func TestEngine_Plan_AllowsDirectoryBoundToAnOlderVersionOfTheSameRepository(t *testing.T) {
	t.Parallel()

	// A checkout that last applied two versions ago: its state names
	// art-1, the resource is now on art-3, and both live in repo-a.
	dir := linkedDir(t, "art-1")
	store := &fakeArtifactStore{
		GetFn: func(_ context.Context, id string) (ArtifactInfo, error) {
			switch id {
			case "art-1":
				return ArtifactInfo{CatalogID: "cat-1", CatalogVersionID: "ver-1", RepositoryID: "repo-a"}, nil
			case "art-3":
				return ArtifactInfo{RepositoryID: "repo-a"}, nil
			}
			return ArtifactInfo{}, errors.New("unexpected artifact " + id)
		},
	}
	e, err := New(dir, "art-3", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	e.PreviousArtifact("art-2")

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, plan.IsEmpty())
	assert.Equal(t, "art-3", e.config.ArtifactID)
}

func TestEngine_Plan_RefusesDirectoryBoundToAnotherRepository(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-1")
	store := &fakeArtifactStore{
		GetFn: func(_ context.Context, id string) (ArtifactInfo, error) {
			if id == "art-1" {
				return ArtifactInfo{CatalogID: "cat-1", CatalogVersionID: "ver-1", RepositoryID: "repo-a"}, nil
			}
			return ArtifactInfo{RepositoryID: "repo-b"}, nil
		},
	}
	e, err := New(dir, "art-2", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.ErrorIs(t, err, ErrDirectoryOwned)
	assert.ErrorContains(t, err, "repository repo-a")
}

func TestEngine_Plan_AllowsDirectoryBoundToThePreviousArtifact(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-1")
	store := &fakeArtifactStore{
		GetFn: func(_ context.Context, id string) (ArtifactInfo, error) {
			if id != "art-clone" {
				return ArtifactInfo{}, errors.New("only the artifact being synced should be read")
			}
			return draftInfo("", ""), nil
		},
	}
	e, err := New(dir, "art-clone", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	e.PreviousArtifact("art-1")

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, plan.IsEmpty())
}

func TestEngine_Plan_AllowsDirectoryBoundToAGoneArtifact(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-old")
	store := &fakeArtifactStore{
		GetFn: func(_ context.Context, id string) (ArtifactInfo, error) {
			if id == "art-old" {
				return ArtifactInfo{}, fmt.Errorf("%w: 404", ErrArtifactNotFound)
			}
			return draftInfo("", ""), nil
		},
	}
	e, err := New(dir, "art-new", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)
	assert.True(t, plan.IsEmpty())
	assert.Equal(t, "art-new", e.config.ArtifactID, "the directory follows the re-created resource")
}

func TestEngine_Plan_OwnerLookupFailureIsReported(t *testing.T) {
	t.Parallel()

	dir := linkedDir(t, "art-1")
	store := &fakeArtifactStore{
		GetFn: func(_ context.Context, id string) (ArtifactInfo, error) {
			if id == "art-1" {
				return ArtifactInfo{}, errors.New("503 from the API")
			}
			return draftInfo("", ""), nil
		},
	}
	e, err := New(dir, "art-2", &fakeFilesAPI{}, store)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.ErrorContains(t, err, "check which artifact")
	require.ErrorContains(t, err, "503 from the API")
}
