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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		err := wapi.Initialize(e.projectDir, wapi.InitOptions{
			ArtifactID:          e.artifactID,
			CatalogID:           e.seedCatalogID,
			LastSyncedVersionID: e.seedVersionID,
		})
		if err != nil && !errors.Is(err, wapi.ErrAlreadyLinked) {
			return fmt.Errorf("create sync state directory (source.dir must be writable, the sync keeps its last-synced manifest there): %w", err)
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

// gather (phase 1) loads the on-disk sync state, fetches the artifact, and
// computes the drift flag phase2 uses to decide whether to call AllFiles.
func (e *Engine) gather(ctx context.Context) error {
	cfg, err := wapi.LoadConfig(e.projectDir)
	if errors.Is(err, wapi.ErrNotInitialized) {
		// The directory exists (preflight saw it) but config.json is not
		// there yet: another process is between creating the directory
		// and writing its files. Same answer as a held lock.
		return fmt.Errorf("%w: another sync is initializing %s", ErrLocked, wapi.Dir(e.projectDir))
	}
	if err != nil {
		return fmt.Errorf("read sync state config.json: %w", err)
	}

	info, err := e.artifacts.Get(ctx, e.artifactID)
	if err != nil {
		return fmt.Errorf("fetch artifact %s: %w", e.artifactID, err)
	}

	if err := e.refuseOwnedDirectory(ctx, cfg.ArtifactID, info); err != nil {
		return err
	}

	// The caller's artifact ID wins over the one the state directory was
	// initialized with. A source change on a locked artifact clones to a
	// new draft artifact against the same directory and the same catalog,
	// and a destroyed-and-recreated resource starts a new artifact over a
	// directory that was synced before, so config.json has to follow the
	// resource instead of pinning an artifact that has been superseded.
	// What the directory is really bound to is the catalog, and gather
	// checks that below once the artifact is known. No CLI counterpart:
	// `dr artifact code init` binds one artifact ID for the life of the
	// directory.
	cfg.ArtifactID = e.artifactID

	manifest, err := wapi.LoadManifest(e.projectDir)
	if err != nil {
		return fmt.Errorf("read sync state manifest.json: %w", err)
	}
	e.base = baseFromManifest(manifest)
	e.baseExtra, e.adoptedBase = splitAdoptedBaseMarker(manifest)
	e.baseRecorded = manifest.SyncedVersionID != nil || len(manifest.Files) > 0
	e.unreadableBase = ""

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

	// A directory can pin a catalog that was deleted since (destroying the
	// resource keeps its catalog, but a cleanup script, the UI or another
	// client can delete it), and every sync against that pin fails. So a pin
	// the artifact's code_ref does not agree with is checked, and a deleted
	// one is dropped with its BASE: the directory starts over the way
	// preflight starts one with no state, from what BindCatalog says unless
	// that is the deleted catalog too, in which case the sync creates a new
	// one. Only a 410 counts, since the Files API also answers 404 for a live
	// catalog the caller has no role on; any other answer, an error
	// included, keeps the pin as before, and an error is logged at debug
	// level. No CLI counterpart: the CLI never brings a new artifact to a
	// directory that is already linked.
	seedVersionID := e.seedVersionID
	if pinned := ptrOrEmpty(cfg.CatalogID); pinned != "" && info.CatalogID != pinned {
		_, err := e.files.ListVersions(ctx, pinned, 1)
		switch {
		case errors.Is(err, filesapi.ErrGone):
			catalogID, versionID := e.seedCatalogID, e.seedVersionID
			if catalogID == "" || catalogID == pinned {
				catalogID, versionID = "", ""
			}
			cfg.CatalogID = ptrOrNil(catalogID)
			cfg.LastSyncedVersionID = ptrOrNil(versionID)
			seedVersionID = versionID
			e.base = BaseManifest{}
			e.baseRecorded = false
			e.adoptedBase = ""
		case err != nil:
			tflog.Debug(ctx, "Keeping the catalog the source directory is bound to: the Files API did not report it deleted", map[string]interface{}{
				"catalog_id": pinned,
				"state_dir":  wapi.Dir(e.projectDir),
				"error":      err.Error(),
			})
		}
	}
	e.config = cfg

	// BASE describes one catalog, the one config.json pins. An artifact
	// with no code_ref yet is fine (a draft just cloned from a locked
	// artifact, or a resource re-created over a synced directory), and so
	// is one whose code_ref points at the pinned catalog. One whose code
	// lives in a different catalog is not: diffing against a BASE that
	// never described that catalog would upload and delete the wrong
	// files, so refuse and let the user re-link the directory.
	if pinned := ptrOrEmpty(cfg.CatalogID); pinned != "" && info.CatalogID != "" && info.CatalogID != pinned {
		return fmt.Errorf("%w: %s records catalog %s, but artifact %s has its code in catalog %s; remove that directory to re-link this source tree to the artifact's catalog",
			ErrCatalogMismatch, wapi.Dir(e.projectDir), pinned, e.artifactID, info.CatalogID)
	}

	// Config's catalog ID is pinned for the artifact's draft lifetime and
	// wins over the artifact's live code_ref, which may have been bumped
	// by another writer.
	e.catalogID = info.CatalogID
	if cfg.CatalogID != nil && *cfg.CatalogID != "" {
		e.catalogID = *cfg.CatalogID
	}
	e.artifactVer = info.CatalogVersionID
	e.remoteVer = info.CatalogVersionID

	// A freshly cloned draft carries no code_ref yet, so diff against a
	// version the code is known to be at rather than against nothing:
	// otherwise every clone re-uploads the whole tree, and the clone of an
	// unchanged tree would leave the new artifact with no code at all. The
	// caller's version (BindCatalog: what its record of the previous
	// artifact points at) wins over the one this directory last synced. A
	// locked artifact's code_ref cannot move, so the two only differ when
	// the directory is behind the resource, and then the catalog counts
	// as drifted and REMOTE is fetched rather than copied from a BASE that
	// predates it.
	if e.remoteVer == "" {
		e.remoteVer = seedVersionID
	}
	if e.remoteVer == "" {
		e.remoteVer = ptrOrEmpty(cfg.LastSyncedVersionID)
	}

	e.drifted = e.remoteVer != "" && e.remoteVer != ptrOrEmpty(cfg.LastSyncedVersionID)

	if e.drifted {
		if err := e.refuseRolledBackCatalog(ctx, ptrOrEmpty(cfg.LastSyncedVersionID)); err != nil {
			return err
		}
	}

	return nil
}

// refuseOwnedDirectory (phase 1, under the sync lock so a parallel apply
// cannot slip past it) rejects a state directory bound to an artifact that
// still exists and belongs to another lineage than current. The shapes a
// re-bound directory legitimately takes are what the rebind in gather
// exists for, and all pass: a clone of a locked artifact (PreviousArtifact,
// or any older version from the same artifact repository, which is where a
// checkout that last applied a few versions ago sits) and a
// destroyed-and-recreated resource, whose old artifact is gone, or, for a
// locked version destroy had to leave in a repository the resource did not
// create, was released from the directory by that destroy. The
// lineage is the artifact repository, because that is what every version
// of one resource shares; two resources deliberately pointed at one
// repository and one directory are indistinguishable from one resource's
// history and are not caught here.
func (e *Engine) refuseOwnedDirectory(ctx context.Context, owner string, current ArtifactInfo) error {
	if owner == "" || owner == e.artifactID || owner == e.previousArtifactID {
		return nil
	}

	other, err := e.artifacts.Get(ctx, owner)
	if errors.Is(err, ErrArtifactNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check which artifact %s is bound to: %w", wapi.Dir(e.projectDir), err)
	}

	if other.RepositoryID != "" && other.RepositoryID == current.RepositoryID {
		return nil
	}

	return fmt.Errorf("%w: %s already backs artifact %s (repository %s), which still exists: a directory can back only one artifact repository, the versions of one datarobot_artifact resource, because the sync state under it describes one catalog. "+
		"Give this resource its own directory, or remove %s if that artifact is no longer managed from here",
		ErrDirectoryOwned, e.projectDir, owner, other.RepositoryID, wapi.Dir(e.projectDir))
}

// versionOrderLimit bounds the catalog history read to order two versions.
// A catalog gains one version per sync, so hundreds is generous; a version
// further back than that is refused as unorderable rather than guessed.
const versionOrderLimit = 500

// refuseRolledBackCatalog (phase 1) checks that a drifted catalog moved
// forward: the version the artifact points at has to be newer than the one
// the directory last synced. Older means the artifact was re-pointed
// backwards, a rollback, and BASE is then a descendant of REMOTE rather
// than an ancestor of both sides. Diffing against it would classify every
// file added since as REMOTE_DELETED and unlink it locally, so the sync
// refuses and leaves the choice to the caller. No CLI counterpart: the CLI
// merges against whatever code_ref it finds.
func (e *Engine) refuseRolledBackCatalog(ctx context.Context, lastSynced string) error {
	if lastSynced == "" || e.catalogID == "" {
		return nil
	}

	versions, err := e.files.ListVersions(ctx, e.catalogID, versionOrderLimit)
	if err != nil {
		return fmt.Errorf("order catalog versions %s and %s: %w", ShortVer(lastSynced), ShortVer(e.remoteVer), err)
	}

	// Newest first, as the Files API lists them.
	liveIdx, lastIdx := -1, -1
	for i, v := range versions {
		if v.ID == e.remoteVer {
			liveIdx = i
		}
		if v.ID == lastSynced {
			lastIdx = i
		}
	}

	stateDir := wapi.Dir(e.projectDir)
	if liveIdx < 0 || lastIdx < 0 {
		return fmt.Errorf("could not place versions %s (artifact) and %s (last synced) in the history of catalog %s, so the sync cannot tell whether the catalog moved forward; "+
			"resolve in the directory with the DataRobot CLI (`dr artifact code sync`), or remove %s to make the directory the source of truth again",
			ShortVer(e.remoteVer), ShortVer(lastSynced), e.catalogID, stateDir)
	}
	if liveIdx > lastIdx {
		return fmt.Errorf("%w: artifact %s points at catalog version %s, older than the version this directory last synced (%s). A sync against it would read every file added since as deleted and remove it locally. "+
			"Re-point the artifact forward, resolve in the directory with the DataRobot CLI (`dr artifact code sync`), or remove %s to make the directory the source of truth again",
			ErrCatalogRolledBack, e.artifactID, ShortVer(e.remoteVer), ShortVer(lastSynced), stateDir)
	}

	return nil
}

// buildManifests (phase 2) walks + hashes source.dir into LOCAL, then
// either fast-paths REMOTE from BASE (not drifted — the solo-developer
// path) or fetches it from the Files API (drifted).
func (e *Engine) buildManifests(ctx context.Context) error {
	matcher := e.ignore
	if matcher == nil {
		var err error
		matcher, err = ignore.New(e.projectDir)
		if err != nil {
			return fmt.Errorf("load ignore rules: %w", err)
		}
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

	if err := e.adoptPinnedBase(ctx); err != nil {
		return err
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

// adoptPinnedBase (phase 2) gives a directory that has never recorded a
// sync the file list of the catalog version it is pinned at as BASE. Such a
// directory is the state preflight seeds from BindCatalog: a fresh clone or a
// CI checkout, because the state directory ignores itself. Its manifest is
// empty, and an empty BASE cannot express a deletion: a file removed from
// the tree is then in none of BASE, LOCAL and REMOTE, so it stays in the
// catalog, or reads as a remote addition and is downloaded back when the
// catalog moved on.
//
// The listing is recorded in manifest.json against the pinned version right
// away, so the DataRobot CLI, which the conflict refusal sends the user to,
// merges this directory against the same BASE instead of the empty one.
// The manifest also carries adoptedBaseKey until the next sync rewrites it,
// so a Plan retried after a failed apply still knows its BASE was adopted.
//
// Only a version pinned in config.json counts, never the artifact's own
// code_ref: `dr artifact code init` pins the catalog but no version, and
// such a directory keeps the CLI's merge against an empty BASE. A pinned
// version the Files API cannot find keeps the empty BASE, which only
// uploads, and is reported through UnreadableBaseVersion; failing instead
// would fail every apply, since preflight seeds the same pin again
// whenever the directory is removed.
func (e *Engine) adoptPinnedBase(ctx context.Context) error {
	catalogID := ptrOrEmpty(e.config.CatalogID)
	versionID := ptrOrEmpty(e.config.LastSyncedVersionID)
	if e.baseRecorded || catalogID == "" || versionID == "" {
		return nil
	}

	files, err := e.files.AllFiles(ctx, catalogID, versionID)
	if errors.Is(err, filesapi.ErrNotFound) {
		e.unreadableBase = versionID
		return nil
	}
	if err != nil {
		return fmt.Errorf("fetch base manifest for catalog version %s: %w", ShortVer(versionID), err)
	}

	e.base = fromFilesAPI(files)

	recorded := make(map[string]wapi.FileMeta, len(e.base))
	for path, entry := range e.base {
		recorded[path] = wapi.FileMeta{Hash: entry.Hash, Size: entry.Size}
	}
	now := e.nowFn().UTC()
	if err := wapi.SaveManifest(e.projectDir, wapi.Manifest{
		Version:         wapi.ManifestVersion,
		SyncedAt:        &now,
		SyncedVersionID: &versionID,
		Files:           recorded,
		Extra:           withAdoptedBaseMarker(e.baseExtra, versionID),
	}); err != nil {
		return fmt.Errorf("record base manifest for catalog version %s: %w", ShortVer(versionID), err)
	}
	e.baseRecorded = true
	e.adoptedBase = versionID

	return nil
}

// adoptedBaseKey is the manifest.json key adoptPinnedBase records the
// pinned version under. It is the provider's alone: the CLI ignores keys it
// does not know and drops them when it saves, and phase 6 writes baseExtra,
// from which gather has removed it, so any completed sync clears it.
const adoptedBaseKey = "providerAdoptedBaseVersionId"

// splitAdoptedBaseMarker returns m's unknown keys without adoptedBaseKey,
// and the version that key names when it still describes m: a manifest a
// later sync rewrote names another version, and does not count as adopted.
func splitAdoptedBaseMarker(m wapi.Manifest) (map[string]json.RawMessage, string) {
	raw, ok := m.Extra[adoptedBaseKey]
	if !ok {
		return m.Extra, ""
	}

	extra := make(map[string]json.RawMessage, len(m.Extra)-1)
	for k, v := range m.Extra {
		if k != adoptedBaseKey {
			extra[k] = v
		}
	}
	if len(extra) == 0 {
		extra = nil
	}

	var versionID string
	if json.Unmarshal(raw, &versionID) != nil || m.SyncedVersionID == nil || *m.SyncedVersionID != versionID {
		return extra, ""
	}

	return extra, versionID
}

// withAdoptedBaseMarker returns extra plus adoptedBaseKey naming versionID,
// leaving extra itself untouched.
func withAdoptedBaseMarker(extra map[string]json.RawMessage, versionID string) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(extra)+1)
	for k, v := range extra {
		out[k] = v
	}
	raw, _ := json.Marshal(versionID)
	out[adoptedBaseKey] = raw

	return out
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
