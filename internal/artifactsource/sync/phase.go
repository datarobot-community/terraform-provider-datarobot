package sync

// CLI source: cli/internal/workload/sync/phase0_preflight.go,
// phase1_gather.go, phase2_manifests.go.
//
// Provider differences from CLI: phase0 auto-initializes state instead
// of failing with "not linked" (see engine.go doc comment) and restores a
// stale rollback under the sync lock rather than ahead of it (see
// preflight); phase1/phase2 read from ArtifactInfo/filesapi.Client instead
// of workload.Artifact / cli/internal/drapi/filesapi.

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// preflight (phase 0) auto-initializes the sync state directory on the
// very first Plan call, acquires the exclusive sync lock, and then
// recovers a stale rollback tree left by a crashed prior sync.
//
// Ordering diverges from CLI phase0_preflight.go, which restores before
// locking: restoring rewrites the working tree, so it has to happen under
// the lock or it can undo the in-flight execute of a process that already
// holds it (which would then also fail with ErrLocked). Crash recovery is
// unaffected — the OS drops a dead holder's flock, so the next Plan to
// take the lock is the one that restores. Init has to stay ahead of the
// lock, because AcquireLock needs the state directory to exist.
func (e *Engine) preflight() error {
	if !wapi.Exists(e.projectDir) {
		// Initialize creates the state directory with a single atomic
		// os.Mkdir, so a concurrent first Plan that loses the race adopts
		// the winner's tree instead of failing on "already linked".
		err := wapi.Initialize(e.projectDir, wapi.InitOptions{ArtifactID: e.artifactID})
		if err != nil && !errors.Is(err, wapi.ErrAlreadyLinked) {
			return fmt.Errorf("auto-init .wapi/: %w", err)
		}
	}

	// Re-Plan on an Engine that still holds the lock reuses it: flock is
	// per open file description, so a second AcquireLock from this same
	// process would fail with ErrLocked against our own held lock.
	if e.lock == nil {
		lock, err := AcquireLock(e.projectDir)
		if err != nil {
			return err
		}
		e.lock = lock
	}

	restored := HasRollback(e.projectDir)
	if err := RestoreRollback(e.projectDir); err != nil {
		return fmt.Errorf("recover stale rollback: %w", err)
	}
	e.staleNote = restored

	return nil
}

// gather (phase 1) loads on-disk .wapi/ state, fetches the artifact, and
// computes the drift flag phase2 uses to decide whether to call AllFiles.
func (e *Engine) gather(ctx context.Context) error {
	cfg, err := wapi.LoadConfig(e.projectDir)
	if err != nil {
		return fmt.Errorf("read .wapi/config.json: %w", err)
	}
	e.config = cfg

	manifest, err := wapi.LoadManifest(e.projectDir)
	if err != nil {
		return fmt.Errorf("read .wapi/manifest.json: %w", err)
	}
	e.base = baseFromManifest(manifest)

	// The artifact this directory is bound to is the one we sync, and it
	// has to be the one the caller asked for. cfg.ArtifactID wins on
	// content (it survives across calls) but a divergence is a caller bug,
	// not something to paper over: syncing artifact B's files into
	// artifact A because .wapi/ still remembers A is worse than failing.
	// Auto-init writes e.artifactID, so this can only fire on state that
	// was already there.
	if cfg.ArtifactID != e.artifactID {
		return fmt.Errorf("%w: %s is bound to artifact %s, not %s",
			ErrArtifactMismatch, wapi.Dir(e.projectDir), cfg.ArtifactID, e.artifactID)
	}

	info, err := e.artifacts.Get(ctx, cfg.ArtifactID)
	if err != nil {
		return fmt.Errorf("fetch artifact %s: %w", cfg.ArtifactID, err)
	}

	// A locked artifact is immutable, so nothing can be pushed into it.
	// That refuses a sync, but it must not refuse a plan: the caller asks
	// for this diff before deciding what to do with the code, and on a
	// locked artifact what it decides is to mint a new version and roll
	// onto it. Refusing to count the files refuses the very deploy that
	// gets past the lock. CLI parity: phase1_gather.go gates the same
	// rejection on previewOnly, and phase 5 checks again, so the exemption
	// cannot let a write through. Plan is preview by definition here —
	// Execute is what returns ErrLockedArtifact once it lands.
	e.locked = info.Locked

	// Config's catalog ID is pinned for the artifact's draft lifetime and
	// wins over the artifact's live code_ref, which may have been bumped
	// by another writer.
	e.catalogID = info.CatalogID
	if cfg.CatalogID != nil && *cfg.CatalogID != "" {
		e.catalogID = *cfg.CatalogID
	}
	e.remoteVer = info.CatalogVersionID
	e.drifted = e.remoteVer != "" && e.remoteVer != ptrOrEmpty(cfg.LastSyncedVersionID)

	return nil
}

// buildManifests (phase 2) walks + hashes source.dir into LOCAL, then
// either fast-paths REMOTE from BASE (not drifted — the solo-developer
// path) or fetches it from the Files API (drifted).
func (e *Engine) buildManifests(ctx context.Context) error {
	matcher, err := ignore.New(e.projectDir)
	if err != nil {
		return fmt.Errorf("load ignore rules: %w", err)
	}

	files, err := artifactsource.CollectLocalFiles(e.projectDir, matcher.Match)
	if err != nil {
		return fmt.Errorf("walk project directory: %w", err)
	}

	local := make(LocalManifest, len(files))
	paths := make(map[string]struct{}, len(files))
	for _, f := range files {
		local[f.RelPath] = FileEntry{Hash: f.Hash, Size: f.Size}
		paths[f.RelPath] = struct{}{}
	}
	e.local = local

	if collisions := detectCaseCollisions(paths); len(collisions) > 0 {
		return errors.New(formatCaseCollisions(collisions))
	}

	if !e.drifted {
		e.remote = copyManifest(e.base)
		return nil
	}

	// CLI parity (phase2_manifests.go): with no catalog behind the
	// artifact there is no remote manifest to fetch, so REMOTE is empty
	// rather than an AllFiles call that can only fail. e.drifted already
	// implies a non-empty remoteVer, so the catalog ID is the open half.
	if e.catalogID == "" {
		e.remote = RemoteManifest{}
		return nil
	}

	remote, err := e.files.AllFiles(ctx, e.catalogID, e.remoteVer)
	if err != nil {
		return fmt.Errorf("fetch remote manifest: %w", err)
	}
	e.remote = fromFilesAPI(remote)

	return nil
}

func baseFromManifest(m wapi.Manifest) BaseManifest {
	out := make(BaseManifest, len(m.Files))
	for k, v := range m.Files {
		out[k] = FileEntry{Hash: v.Hash, Size: v.Size}
	}

	return out
}

func copyManifest(in BaseManifest) BaseManifest {
	out := make(BaseManifest, len(in))
	for k, v := range in {
		out[k] = v
	}

	return out
}

func fromFilesAPI(in map[string]filesapi.FileMeta) RemoteManifest {
	out := make(RemoteManifest, len(in))
	for k, v := range in {
		out[k] = FileEntry{Hash: v.Hash, Size: v.Size}
	}

	return out
}
