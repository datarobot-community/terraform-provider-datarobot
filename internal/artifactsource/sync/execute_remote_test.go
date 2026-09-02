package sync

// CLI source: cli/internal/workload/sync/engine_test.go — the phase-5
// remote-half and phase-6 scenarios (uploads, remote deletes, code_ref
// patch, new BASE manifest) ported to this package's fakes.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stageResponses opts the fake into the stage upload path.
func stageResponses(f *fakeFilesAPI, versionID string) {
	f.stageID = "stage-1"
	f.stageVersionID = versionID
}

// runSync drives the full pipeline the way the resource will: local half
// first, then remote half plus phase 6.
func runSync(t *testing.T, f *syncFixture) *Result {
	t.Helper()

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	result, err := f.engine.ExecuteRemote(context.Background())
	require.NoError(t, err)

	return result
}

func loadBase(t *testing.T, dir string) map[string]wapi.FileMeta {
	t.Helper()

	manifest, err := wapi.LoadManifest(dir)
	require.NoError(t, err)

	return manifest.Files
}

func TestEngine_ExecuteRemote_UploadsLocalChangesAndPatchesCodeRef(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit", "new.py": "brand new"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "base body"},
	)
	stageResponses(f.files, "ver-3")

	require.Len(t, f.plan.Uploads, 2)

	result := runSync(t, f)

	assert.Equal(t, []string{"agent.py", "new.py"}, f.files.stagedPaths())
	assert.Equal(t, "local edit", f.files.staged["agent.py"], "the stage upload must carry local bytes")
	assert.Equal(t, []string{"cat-1"}, f.files.stageCatalogIDs, "an existing catalog is reused")
	assert.Zero(t, f.files.createCatalogCalls)

	assert.Equal(t, []patchCall{{ArtifactID: "art-1", CatalogID: "cat-1", CatalogVersionID: "ver-3"}}, f.artifacts.patches)

	assert.Equal(t, "cat-1", result.CatalogID)
	assert.Equal(t, "ver-3", result.CatalogVersionID)
	assert.Equal(t, "ver-2", result.PreviousVersionID)
	assert.Equal(t, 2, result.Uploaded)

	// Phase 6: BASE now holds the uploaded local hashes, config points at
	// the new version, and the rollback tree is gone.
	localHash, localSize := hashContent("local edit")
	assert.Equal(t, wapi.FileMeta{Hash: localHash, Size: localSize}, loadBase(t, f.dir)["agent.py"])

	cfg, err := wapi.LoadConfig(f.dir)
	require.NoError(t, err)
	assert.Equal(t, "ver-3", ptrOrEmpty(cfg.LastSyncedVersionID))
	assert.Equal(t, "cat-1", ptrOrEmpty(cfg.CatalogID))
	assert.False(t, HasRollback(f.dir))
}

func TestEngine_ExecuteRemote_LocalDeleteRemovesCatalogFile(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same", "gone.py": "dropped"},
		map[string]string{"agent.py": "same", "gone.py": "dropped"},
	)
	f.files.deleteVersionID = "ver-3"

	require.Len(t, f.plan.Deletes, 1)
	require.Equal(t, ActUploadDelete, f.plan.Deletes[0].Action)

	result := runSync(t, f)

	assert.Equal(t, [][]string{{"gone.py"}}, f.files.deletedPaths)
	assert.Equal(t, "ver-3", result.CatalogVersionID)
	assert.Equal(t, []patchCall{{ArtifactID: "art-1", CatalogID: "cat-1", CatalogVersionID: "ver-3"}}, f.artifacts.patches)

	assert.NotContains(t, loadBase(t, f.dir), "gone.py", "a deleted path must leave the new BASE")
}

func TestEngine_ExecuteRemote_ConflictBaseTakesRemoteHash(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote edit"},
	)

	require.Len(t, f.plan.Conflicts, 1)

	result := runSync(t, f)

	// Remote won locally, so BASE records the remote hash and the sync
	// pushes nothing: the catalog version is unchanged and no patch runs.
	remoteHash, remoteSize := hashContent("remote edit")
	assert.Equal(t, wapi.FileMeta{Hash: remoteHash, Size: remoteSize}, loadBase(t, f.dir)["agent.py"])
	assert.Empty(t, f.artifacts.patches)
	assert.Equal(t, "ver-2", result.CatalogVersionID)
	assert.Equal(t, []string{"agent.py.LOCAL." + fixedStamp}, result.ConflictCopies)

	// The *.LOCAL.* copy is ignored, so it never enters BASE either.
	assert.NotContains(t, loadBase(t, f.dir), "agent.py.LOCAL."+fixedStamp)
}

func TestEngine_ExecuteRemote_EmptyPlanTouchesNoRemote(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
	)

	require.True(t, f.plan.IsEmpty())

	result := runSync(t, f)

	assert.Empty(t, f.files.stagedPaths())
	assert.Empty(t, f.files.deletedPaths)
	assert.Empty(t, f.artifacts.patches)

	// Nothing changed, but the observed remote version is recorded so the
	// next Plan is no longer treated as drifted.
	assert.Equal(t, "ver-2", result.CatalogVersionID)

	cfg, err := wapi.LoadConfig(f.dir)
	require.NoError(t, err)
	assert.Equal(t, "ver-2", ptrOrEmpty(cfg.LastSyncedVersionID))
}

func TestEngine_ExecuteRemote_FirstSyncCreatesCatalog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProjectFiles(t, dir, map[string]string{"agent.py": "print('hi')\n"})

	files := &fakeFilesAPI{newCatalogID: "cat-new"}
	stageResponses(files, "ver-1")

	artifacts := &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("", ""), nil },
	}

	e, err := New(dir, "art-1", files, artifacts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	_, err = e.Plan(context.Background())
	require.NoError(t, err)
	require.NoError(t, e.ExecuteLocal(context.Background()))

	result, err := e.ExecuteRemote(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, files.createCatalogCalls, "an artifact with no code_ref needs a new catalog")
	assert.Equal(t, "cat-new", result.CatalogID)
	assert.Equal(t, []patchCall{{ArtifactID: "art-1", CatalogID: "cat-new", CatalogVersionID: "ver-1"}}, artifacts.patches)

	cfg, err := wapi.LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "cat-new", ptrOrEmpty(cfg.CatalogID))
}

func TestEngine_ExecuteRemote_LargeChangeSetUsesZip(t *testing.T) {
	t.Parallel()

	local := make(map[string]string, artifactsource.StageVsZipFileThreshold+1)
	for i := 0; i <= artifactsource.StageVsZipFileThreshold; i++ {
		local[fmt.Sprintf("gen/f%d.py", i)] = fmt.Sprintf("body %d", i)
	}

	f := newSyncFixture(t, local, map[string]string{}, map[string]string{})
	f.files.zipVersionID = "ver-3"

	require.Len(t, f.plan.Uploads, artifactsource.StageVsZipFileThreshold+1)

	result := runSync(t, f)

	// Above StageVsZipFileThreshold files the shared uploader switches to
	// zip, so the stage endpoints are never touched.
	assert.Equal(t, 1, f.files.zipUploads)
	assert.Empty(t, f.files.stagedPaths())
	assert.Equal(t, "ver-3", result.CatalogVersionID)
}

func TestEngine_ExecuteRemote_RestoresLocalTreeOnUploadFailure(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit", "new.py": "brand new"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote edit", "added.py": "pulled"},
	)
	stageResponses(f.files, "ver-3")
	f.files.uploadErr = errors.New("boom")

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))
	require.Equal(t, "remote edit", readProjectFile(t, f.dir, "agent.py"))

	_, err := f.engine.ExecuteRemote(context.Background())
	require.ErrorContains(t, err, "boom")

	// A failed upload undoes the local half too, so source.dir is back to
	// what the user had before the apply.
	assert.Equal(t, "local edit", readProjectFile(t, f.dir, "agent.py"))
	assert.Empty(t, conflictCopiesOnDisk(t, f.dir))
	requireAbsent(t, f.dir, "added.py")
	assert.False(t, HasRollback(f.dir))
	assert.Empty(t, f.artifacts.patches, "code_ref must not move when the upload failed")

	// BASE is untouched, so the next apply replans from the same state.
	baseHash, _ := hashContent("base body")
	assert.Equal(t, baseHash, loadBase(t, f.dir)["agent.py"].Hash)
}

func TestEngine_ExecuteRemote_StateSaveFailureKeepsRemoteVersion(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "base body"},
	)
	stageResponses(f.files, "ver-3")

	// Make .wapi/manifest.json un-writable by turning it into a
	// directory: the atomic rename over it fails.
	manifestPath := filepath.Join(f.dir, wapi.DirName, "manifest.json")
	require.NoError(t, os.Remove(manifestPath))
	require.NoError(t, os.Mkdir(manifestPath, 0o755))

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	_, err := f.engine.ExecuteRemote(context.Background())
	require.ErrorContains(t, err, "save .wapi/manifest.json")

	// The catalog has already advanced, so phase 6 must not roll it back;
	// the next apply reconciles from the retained rollback tree.
	assert.Equal(t, []patchCall{{ArtifactID: "art-1", CatalogID: "cat-1", CatalogVersionID: "ver-3"}}, f.artifacts.patches)
	assert.Equal(t, "local edit", readProjectFile(t, f.dir, "agent.py"))
}

func TestEngine_ExecuteRemote_RefusedBeforeExecuteLocal(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote edit"},
	)

	// Pushing local bytes before the remote-wins half ran would upload
	// content the user is about to have overwritten locally.
	_, err := f.engine.ExecuteRemote(context.Background())
	require.ErrorIs(t, err, ErrLocalNotApplied)

	assert.Empty(t, f.files.stagedPaths())
	assert.Empty(t, f.artifacts.patches)
}

func TestEngine_ExecuteRemote_WithoutPlanOrLock(t *testing.T) {
	t.Parallel()

	e, err := New(t.TempDir(), "art-1", &fakeFilesAPI{}, &fakeArtifactStore{})
	require.NoError(t, err)

	_, err = e.ExecuteRemote(context.Background())
	assert.ErrorIs(t, err, ErrNoPlan)

	f := newSyncFixture(t,
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
	)
	require.NoError(t, f.engine.Close())

	_, err = f.engine.ExecuteRemote(context.Background())
	assert.ErrorIs(t, err, ErrLockReleased)
}
