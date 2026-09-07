package sync

// CLI source: cli/internal/workload/sync/phase6_state.go
//
// Provider differences from CLI:
//   - No history.log entry (AppendHistory is CLI UX and was not ported,
//     see the wapi package doc) and no Result.Duration.
//   - The rollback tree is dropped before the new state is written, not
//     after, and a failure to drop it is reported instead of ignored. See
//     persistState for why the order matters.
//   - manifest.json is written before config.json, so a crash between the
//     two leaves the directory looking drifted (config still names the
//     old version) and the next Plan re-fetches REMOTE, rather than
//     looking in sync with a BASE that is one sync stale.

import (
	"fmt"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
)

// persistState (phase 6) commits the sync: it drops the rollback tree that
// guarded ExecuteLocal's working-tree mutations, then records the new BASE
// manifest and the catalog pointers in the state directory.
//
// The rollback tree goes first. By now the catalog has advanced, so the
// working tree is right and the pre-sync copies in the rollback tree are
// wrong: a later Plan that restored them would revert downloads the
// catalog already agrees with, and against a freshly written BASE it would
// then read the missing files as local deletions to push. With the tree
// gone, a crash anywhere after this point leaves a working tree that
// matches the catalog and a BASE at most one sync behind, which the next
// Plan reconciles by re-fetching REMOTE. Nothing here rolls the catalog
// back; an error only means the state directory is behind.
func (e *Engine) persistState() error {
	if err := e.DiscardRollback(); err != nil {
		return fmt.Errorf("sync completed but the rollback tree could not be dropped: %w", err)
	}

	now := e.nowFn().UTC()

	cfg := e.config
	if e.newCatalogID != "" {
		catalogID := e.newCatalogID
		cfg.CatalogID = &catalogID
	}

	// A pull-only or no-op sync produces no new version; persist the
	// remote version observed during gather so the next Plan is not
	// treated as drifted.
	syncedVersionID := e.newVersionID
	if syncedVersionID == "" {
		syncedVersionID = e.remoteVer
	}

	if syncedVersionID != "" {
		cfg.LastSyncedVersionID = &syncedVersionID
	}

	if err := wapi.SaveManifest(e.projectDir, e.newBaseManifest(syncedVersionID, now)); err != nil {
		return fmt.Errorf("save sync state manifest.json: %w", err)
	}

	if err := wapi.SaveConfig(e.projectDir, cfg); err != nil {
		return fmt.Errorf("save sync state config.json: %w", err)
	}

	e.config = cfg
	e.syncedVersionID = syncedVersionID

	return nil
}

// newBaseManifest computes the BASE the next sync diffs against:
// REMOTE + uploads (local hashes) - deletes, with conflicts resolved to
// the remote hash because remote wins. Both halves of plan.Deletes drop
// out: an ActUploadDelete path was just removed from the catalog, and an
// ActDownloadDelete path was already absent from REMOTE.
func (e *Engine) newBaseManifest(syncedVersionID string, syncedAt time.Time) wapi.Manifest {
	files := make(map[string]wapi.FileMeta, len(e.remote))

	for path, entry := range e.remote {
		files[path] = wapi.FileMeta{Hash: entry.Hash, Size: entry.Size}
	}

	for _, fa := range e.plan.Uploads {
		files[fa.Path] = wapi.FileMeta{Hash: fa.LocalHash, Size: fa.LocalSize}
	}

	for _, fa := range e.plan.Deletes {
		delete(files, fa.Path)
	}

	for _, fa := range e.plan.Conflicts {
		// DEL_EDIT_CONFLICT has no remote side left: the local bytes now
		// live in the *.LOCAL.<ts> copy and the path stays absent.
		if fa.RemoteHash == "" {
			continue
		}

		files[fa.Path] = wapi.FileMeta{Hash: fa.RemoteHash, Size: fa.RemoteSize}
	}

	return wapi.Manifest{
		Version:         wapi.ManifestVersion,
		SyncedAt:        &syncedAt,
		SyncedVersionID: &syncedVersionID,
		Files:           files,
	}
}
