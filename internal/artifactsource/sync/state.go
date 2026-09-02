package sync

// CLI source: cli/internal/workload/sync/phase6_state.go
//
// Provider differences from CLI:
//   - No .wapi/history.log entry (AppendHistory is CLI UX and was not
//     ported, see the wapi package doc) and no Result.Duration.
//   - A failure to drop the rollback tree is reported instead of ignored:
//     a retained tree makes the next Plan restore the pre-sync working
//     directory over content the catalog has already accepted, so it must
//     not pass silently.

import (
	"fmt"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
)

// persistState (phase 6) records the sync in .wapi/: the catalog pointers
// in config.json, the new BASE manifest, and then the rollback tree drop
// that commits ExecuteLocal's working-tree mutations.
//
// Order matters. BASE is written before the rollback tree is discarded, so
// a crash in between leaves the tree for the next Plan to recover. Nothing
// here rolls the catalog back: by this point the remote has advanced, and
// an error only means .wapi/ is behind, which the next apply reconciles.
func (e *Engine) persistState() error {
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

	if err := wapi.SaveConfig(e.projectDir, cfg); err != nil {
		return fmt.Errorf("save .wapi/config.json: %w", err)
	}

	if err := wapi.SaveManifest(e.projectDir, e.newBaseManifest(syncedVersionID, now)); err != nil {
		return fmt.Errorf("save .wapi/manifest.json: %w", err)
	}

	e.config = cfg
	e.syncedVersionID = syncedVersionID

	if err := e.DiscardRollback(); err != nil {
		return fmt.Errorf("sync completed but the rollback tree could not be dropped: %w", err)
	}

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
