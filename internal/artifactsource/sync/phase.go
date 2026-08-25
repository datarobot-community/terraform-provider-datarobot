package sync

// CLI source: cli/internal/workload/sync/phase0_preflight.go,
// phase1_gather.go, phase2_manifests.go.
//
// Provider differences from CLI: phase0 auto-initializes .wapi/ instead
// of failing with "not linked" (see engine.go doc comment); phase1/phase2
// read from ArtifactInfo/filesapi.Client instead of workload.Artifact /
// cli/internal/drapi/filesapi.

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// preflight (phase 0) recovers a stale rollback tree left by a crashed
// prior sync, auto-initializes .wapi/ on the very first Plan call, and
// acquires the exclusive sync lock. Recovery runs before the lock so a
// crashed-mid-sync process gets cleaned up by whoever runs next.
func (e *Engine) preflight() error {
	restored := HasRollback(e.projectDir)
	if err := RestoreRollback(e.projectDir); err != nil {
		return fmt.Errorf("recover stale rollback: %w", err)
	}
	e.staleNote = restored

	if !wapi.Exists(e.projectDir) {
		if err := wapi.Initialize(e.projectDir, wapi.InitOptions{ArtifactID: e.artifactID}); err != nil {
			return fmt.Errorf("auto-init .wapi/: %w", err)
		}
	}

	lock, err := AcquireLock(e.projectDir)
	if err != nil {
		return err
	}
	e.lock = lock

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

	info, err := e.artifacts.Get(ctx, cfg.ArtifactID)
	if err != nil {
		return fmt.Errorf("fetch artifact %s: %w", cfg.ArtifactID, err)
	}
	if info.Locked {
		return ErrLockedArtifact
	}

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
