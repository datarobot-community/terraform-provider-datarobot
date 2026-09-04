package sync

// CLI source: cli/internal/workload/sync/engine_test.go +
// path_safety_test.go — the phase-5 local-half scenarios (downloads,
// conflict copies, remote deletes, restore-on-failure, traversal
// rejection) ported to this package's fakes.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedStamp is what conflictCopyStampFormat renders for fixedNow.
const fixedStamp = "20260901T120000Z"

func fixedNow() time.Time { return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC) }

// syncFixture is a planned Engine over a temp source.dir whose BASE,
// working tree and (drifted) REMOTE manifest are all seeded from
// path -> content maps. Remote content doubles as the bytes the fake
// Files API serves, so downloads verify against real hashes and sizes.
type syncFixture struct {
	dir    string
	files  *fakeFilesAPI
	engine *Engine
	plan   *SyncPlan
}

func newSyncFixture(t *testing.T, local, base, remote map[string]string) *syncFixture {
	t.Helper()

	dir := t.TempDir()
	writeProjectFiles(t, dir, local)

	require.NoError(t, wapi.Initialize(dir, wapi.InitOptions{
		ArtifactID:          "art-1",
		CatalogID:           "cat-1",
		LastSyncedVersionID: "ver-1",
	}))

	baseFiles := make(map[string]wapi.FileMeta, len(base))
	for path, body := range base {
		hash, size := hashContent(body)
		baseFiles[path] = wapi.FileMeta{Hash: hash, Size: size}
	}

	require.NoError(t, wapi.SaveManifest(dir, wapi.Manifest{
		Version: wapi.ManifestVersion,
		Files:   baseFiles,
	}))

	files := &fakeFilesAPI{
		allFiles: make(map[string]filesapi.FileMeta, len(remote)),
		blobs:    make(map[string]string, len(remote)),
	}

	for path, body := range remote {
		hash, size := hashContent(body)
		files.allFiles[path] = filesapi.FileMeta{Hash: hash, Size: size}
		files.blobs[path] = body
	}

	// The artifact reports ver-2 while .wapi/config.json still records
	// ver-1: drifted, so Plan builds REMOTE from AllFiles.
	e, err := New(dir, "art-1", files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })

	e.nowFn = fixedNow

	plan, err := e.Plan(context.Background())
	require.NoError(t, err)

	return &syncFixture{dir: dir, files: files, engine: e, plan: plan}
}

func readProjectFile(t *testing.T, dir, rel string) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	require.NoError(t, err)

	return string(body)
}

func requireAbsent(t *testing.T, dir, rel string) {
	t.Helper()

	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
	require.ErrorIs(t, err, os.ErrNotExist, "%s should not exist", rel)
}

func conflictCopiesOnDisk(t *testing.T, dir string) []string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, "*.LOCAL.*"))
	require.NoError(t, err)

	return matches
}

func TestEngine_ExecuteLocal_DownloadsRemoteAddedAndModifiedFiles(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "base body"},
		map[string]string{
			"agent.py":        "remote body",
			"utils/helper.py": "def help(): pass\n",
		},
	)

	require.Len(t, f.plan.Downloads, 2)

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	assert.Equal(t, "remote body", readProjectFile(t, f.dir, "agent.py"))
	assert.Equal(t, "def help(): pass\n", readProjectFile(t, f.dir, "utils/helper.py"),
		"a remote-added file in a new subdirectory must be created with its parents")
	assert.Equal(t, []string{"agent.py", "utils/helper.py"}, f.files.downloadedPaths())
	assert.Empty(t, f.engine.ConflictCopies())
}

func TestEngine_ExecuteLocal_ConflictKeepsLocalCopyAndTakesRemote(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote edit"},
	)

	require.Len(t, f.plan.Conflicts, 1)
	require.Equal(t, ClsConflict, f.plan.Conflicts[0].Classification)

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	// Remote wins (non-interactive `--yes` policy) and the local bytes
	// survive next to it under the timestamped copy.
	assert.Equal(t, "remote edit", readProjectFile(t, f.dir, "agent.py"))

	copyRel := "agent.py.LOCAL." + fixedStamp
	assert.Equal(t, "local edit", readProjectFile(t, f.dir, copyRel))
	assert.Equal(t, []string{copyRel}, f.engine.ConflictCopies())
}

func TestEngine_ExecuteLocal_DelEditConflictRenamesWithoutDownload(t *testing.T) {
	t.Parallel()

	// Local edited the file, the remote deleted it: keep the local bytes
	// as a conflict copy, and there is nothing to pull.
	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit"},
		map[string]string{"agent.py": "base body"},
		map[string]string{},
	)

	require.Len(t, f.plan.Conflicts, 1)
	require.Equal(t, ClsDelEditConflict, f.plan.Conflicts[0].Classification)

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	requireAbsent(t, f.dir, "agent.py")
	assert.Equal(t, "local edit", readProjectFile(t, f.dir, "agent.py.LOCAL."+fixedStamp))
	assert.Empty(t, f.files.downloadedPaths(), "a remote-deleted path has nothing to download")
}

func TestEngine_ExecuteLocal_EditDelConflictDownloadsOverDelete(t *testing.T) {
	t.Parallel()

	// Local deleted the file, the remote edited it: the remote wins and
	// the file comes back, with no conflict copy (there is no local edit
	// left to keep).
	f := newSyncFixture(t,
		map[string]string{"keep.py": "keep"},
		map[string]string{"keep.py": "keep", "agent.py": "base body"},
		map[string]string{"keep.py": "keep", "agent.py": "remote edit"},
	)

	require.Len(t, f.plan.Downloads, 1)
	require.Equal(t, ClsEditDelConflict, f.plan.Downloads[0].Classification)
	require.Empty(t, f.plan.Conflicts)

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	assert.Equal(t, "remote edit", readProjectFile(t, f.dir, "agent.py"))
	assert.Empty(t, conflictCopiesOnDisk(t, f.dir))
}

func TestEngine_ExecuteLocal_RemoteDeletedRemovesLocalFile(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "keep", "gone.py": "unchanged"},
		map[string]string{"agent.py": "keep", "gone.py": "unchanged"},
		map[string]string{"agent.py": "keep"},
	)

	require.Len(t, f.plan.Deletes, 1)
	require.Equal(t, ActDownloadDelete, f.plan.Deletes[0].Action)

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	requireAbsent(t, f.dir, "gone.py")
	assert.Equal(t, "keep", readProjectFile(t, f.dir, "agent.py"))
	assert.Empty(t, f.files.downloadedPaths())
}

func TestEngine_ExecuteLocal_RestoresTreeOnDownloadFailure(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{
			"agent.py": "local edit",
			"gone.py":  "unchanged",
			"calm.py":  "unchanged",
		},
		map[string]string{
			"agent.py": "base body",
			"gone.py":  "unchanged",
			"calm.py":  "unchanged",
		},
		map[string]string{
			"agent.py": "remote edit",
			"calm.py":  "unchanged",
			"added.py": "brand new",
		},
	)

	f.files.downloadErr = map[string]error{"added.py": errors.New("boom")}

	err := f.engine.ExecuteLocal(context.Background())
	require.ErrorContains(t, err, "boom")

	// Every local mutation is undone: the conflicted file keeps its local
	// bytes with no copy left behind, the remote-added file is gone, and
	// the remote-deleted file is back.
	assert.Equal(t, "local edit", readProjectFile(t, f.dir, "agent.py"))
	assert.Empty(t, conflictCopiesOnDisk(t, f.dir))
	requireAbsent(t, f.dir, "added.py")
	assert.Equal(t, "unchanged", readProjectFile(t, f.dir, "gone.py"))
	assert.False(t, HasRollback(f.dir), "restore must remove .wapi/.rollback/")
}

func TestEngine_ExecuteLocal_ChecksumMismatchRestoresTree(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote body"},
	)

	// The catalog advertised the hash of "remote body" but serves
	// different bytes of the same length, so the size check passes and
	// only the checksum catches it. Either way the swapped payload must
	// not be left over the user's file.
	f.files.blobs["agent.py"] = "REMOTE BODY"

	err := f.engine.ExecuteLocal(context.Background())
	require.ErrorContains(t, err, "checksum mismatch on agent.py")

	assert.Equal(t, "base body", readProjectFile(t, f.dir, "agent.py"))
	assert.False(t, HasRollback(f.dir))
}

func TestEngine_ExecuteLocal_ConflictCopyIgnoredByFollowUpPlan(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{
			"agent.py":  "local edit",
			".drignore": string(ignore.DefaultTemplate),
		},
		map[string]string{"agent.py": "base body", ".drignore": string(ignore.DefaultTemplate)},
		map[string]string{"agent.py": "remote edit", ".drignore": string(ignore.DefaultTemplate)},
	)

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	// Discard the rollback tree the way persisting BASE will. Without it
	// the follow-up Plan would treat the retained rollback as a crashed
	// sync and revert everything.
	require.NoError(t, f.engine.DiscardRollback())
	require.NoError(t, f.engine.Close())

	next, err := New(f.dir, "art-1", f.files, &fakeArtifactStore{
		GetFn: func(context.Context, string) (ArtifactInfo, error) { return draftInfo("cat-1", "ver-2"), nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = next.Close() })

	plan, err := next.Plan(context.Background())
	require.NoError(t, err)

	for _, fa := range plan.Uploads {
		assert.NotContains(t, fa.Path, ".LOCAL.", "the default .drignore excludes *.LOCAL.* from the upload set")
	}

	// Local and remote now agree on agent.py (CONVERGED) and the conflict
	// copy is ignored, so a second sync has nothing left to do.
	assert.True(t, plan.IsEmpty(), "follow-up plan should be empty, got %+v", plan)

	assert.Equal(t, "remote edit", readProjectFile(t, f.dir, "agent.py"),
		"the conflict resolution must survive the follow-up plan")
}

func TestEngine_ExecuteLocal_EmptyPlanTouchesNothing(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
	)

	require.True(t, f.plan.IsEmpty())
	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	assert.Empty(t, f.files.downloadedPaths())
	assert.False(t, HasRollback(f.dir), "an empty plan must not create a rollback tree")
}

func TestEngine_ExecuteLocal_WithoutPlan(t *testing.T) {
	t.Parallel()

	e, err := New(t.TempDir(), "art-1", &fakeFilesAPI{}, &fakeArtifactStore{})
	require.NoError(t, err)

	assert.ErrorIs(t, e.ExecuteLocal(context.Background()), ErrNoPlan)
}

func TestEngine_ExecuteLocal_AfterCloseIsRefused(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote edit"},
	)

	require.NoError(t, f.engine.Close())

	// source.dir must never be mutated without .wapi/sync.lock held.
	assert.ErrorIs(t, f.engine.ExecuteLocal(context.Background()), ErrLockReleased)
	assert.Equal(t, "local edit", readProjectFile(t, f.dir, "agent.py"))
}

func TestEngine_ExecuteLocal_RejectsUnsafeServerPaths(t *testing.T) {
	t.Parallel()

	for name, plan := range map[string]*SyncPlan{
		"download": {Downloads: []FileAction{{Path: "../escape.txt", Action: ActDownloadAdd}}},
		"conflict": {Conflicts: []FileAction{{Path: "../escape.txt", Action: ActConflictCopy}}},
		"delete":   {Deletes: []FileAction{{Path: "../escape.txt", Action: ActDownloadDelete}}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newSyncFixture(t,
				map[string]string{"agent.py": "same"},
				map[string]string{"agent.py": "same"},
				map[string]string{"agent.py": "same"},
			)
			f.engine.plan = plan

			err := f.engine.ExecuteLocal(context.Background())
			require.ErrorContains(t, err, "escapes project root")

			assert.False(t, HasRollback(f.dir), "unsafe paths must be rejected before any filesystem work")
			assert.Empty(t, f.files.downloadedPaths())
		})
	}
}

func TestEngine_ExecuteLocal_RefusesPlanAboveRollbackCap(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
		map[string]string{"agent.py": "same"},
	)

	oversized := &SyncPlan{}
	for i := 0; i <= RollbackMaxFiles; i++ {
		oversized.Uploads = append(oversized.Uploads, FileAction{
			Path:   fmt.Sprintf("gen/f%d.py", i),
			Action: ActUploadAdd,
		})
	}
	f.engine.plan = oversized

	err := f.engine.ExecuteLocal(context.Background())
	require.ErrorContains(t, err, "RollbackMaxFiles=1000")

	assert.False(t, HasRollback(f.dir))
}

func TestEngine_ExecuteLocal_RolledBackRunReportsNoConflictCopies(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t,
		map[string]string{"agent.py": "local edit"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote edit", "added.py": "brand new"},
	)

	f.files.downloadErr = map[string]error{"added.py": errors.New("boom")}

	require.ErrorContains(t, f.engine.ExecuteLocal(context.Background()), "boom")

	// Restore deleted the *.LOCAL.<ts> copy applyConflictCopies had just
	// made, so reporting it in a diagnostic would point the user at a
	// path that is not on disk.
	require.Empty(t, conflictCopiesOnDisk(t, f.dir))
	assert.Empty(t, f.engine.ConflictCopies())

	// A retry re-derives the copies from the same plan; it must not stack
	// a second entry on top of the rolled-back attempt's.
	f.files.downloadErr = nil
	require.NoError(t, f.engine.ExecuteLocal(context.Background()))

	assert.Equal(t, []string{"agent.py.LOCAL." + fixedStamp}, f.engine.ConflictCopies())
}

func TestEngine_DiscardRollback_KeepsTreeWhenDiscardFails(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("skipping read-only directory test when running as root")
	}

	f := newSyncFixture(t,
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "base body"},
		map[string]string{"agent.py": "remote body"},
	)

	require.NoError(t, f.engine.ExecuteLocal(context.Background()))
	require.True(t, HasRollback(f.dir))

	// A read-only state directory makes RemoveAll(.rollback) fail.
	stateDir := wapi.Dir(f.dir)
	require.NoError(t, os.Chmod(stateDir, 0o500))

	t.Cleanup(func() { _ = os.Chmod(stateDir, 0o755) })

	if err := f.engine.DiscardRollback(); err == nil {
		t.Skip("skipping: filesystem did not enforce directory read-only permissions")
	}

	require.NoError(t, os.Chmod(stateDir, 0o755))
	require.True(t, HasRollback(f.dir), "the failed discard left the tree on disk")

	// The engine must still hold the tree so the retry actually removes
	// it; returning nil here would leave .rollback/ for the next Plan to
	// mistake for a crashed sync and revert this run's mutations.
	require.NoError(t, f.engine.DiscardRollback())
	assert.False(t, HasRollback(f.dir))
}
