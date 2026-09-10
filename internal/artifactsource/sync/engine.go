package sync

// CLI source: cli/internal/workload/sync/engine.go
//
// Provider differences from CLI:
//   - Phase 5 is split in two entry points instead of one phase5Execute:
//     ExecuteLocal (execute.go) mutates source.dir, ExecuteRemote
//     (execute_remote.go) mutates the catalog and then runs phase 6
//     (state.go). The two are reviewed and wired separately because only
//     the first changes files the user owns.
//   - No Options{DryRun, ShowDiffs, Yes}: terraform apply has no TTY and is
//     always non-interactive, remote-wins-on-conflict (see the plan's
//     "Non-interactive policy" section) — there is nothing to gate on.
//   - ArtifactStore is a narrow interface over the caller's own artifact
//     client (Get + PatchCodeRef) instead of the CLI's workload.Artifact,
//     so this package stays independent of internal/client.
//   - A missing state directory (.datarobot/workload/, see the wapi
//     package) auto-initializes instead of erroring "not linked": there is
//     no `dr artifact code init` step in a Terraform-managed tree, and
//     BindCatalog lets the resource seed the catalog pointers the CLI would
//     have taken from `init` flags.
//   - The artifact ID is re-bound on every Plan instead of being fixed at
//     init: Terraform, not the state directory, owns artifact identity, and
//     a source change on a locked artifact clones to a new artifact ID
//     against the same directory (see gather in phase.go). What the
//     directory stays bound to is the catalog; ErrCatalogMismatch guards
//     that.
//   - UseIgnore lets the resource hand over the ignore matcher it already
//     resolved, so the walk applies the same patterns plan hashed with even
//     when the directory could not be given its starter .drignore.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// ErrLockedArtifact reports that the bound artifact is locked and so
// cannot be written to. Plan does not return it — planning a locked
// artifact is allowed and only reports the fact via ArtifactLocked (see
// gather) — Execute does, once it lands (mirrors CLI phase1_gather.go,
// where the check is gated on previewOnly and re-run in phase 5).
var ErrLockedArtifact = errors.New("artifact is locked (immutable); cannot sync in place, clone to a draft first")

// ErrCatalogMismatch is returned by Plan when the state directory pins one
// catalog while the artifact the Engine was constructed for keeps its code
// in another. Following the artifact would not be a config edit: BASE
// describes the pinned catalog, so reusing it as the common ancestor of a
// different one would produce a meaningless diff. Whoever wants to retarget
// a source.dir has to reset its state directory with it.
var ErrCatalogMismatch = errors.New("state directory is bound to a different catalog")

// ErrCatalogRolledBack is returned by Plan when the artifact's code_ref
// points at a catalog version older than the one the directory last
// synced. The three-way merge takes BASE for a common ancestor of both
// sides; a BASE that is a descendant of REMOTE would read every file added
// since as a remote deletion and unlink it locally. Someone re-pointed the
// artifact backwards on purpose (a rollback), and a merge cannot resolve
// that: the caller has to say which side wins.
var ErrCatalogRolledBack = errors.New("catalog rolled back behind the directory's last sync")

// ErrDirectoryOwned is returned by Plan when the state directory is bound
// to an artifact that still exists and belongs to another lineage than the
// one being synced: neither the same artifact, nor the one the caller says
// it supersedes, nor one from the same artifact repository. That is a
// second resource over one directory. Two syncs over one directory would
// each treat their own artifact's version as REMOTE and pull the other's
// upload back out of the working tree.
var ErrDirectoryOwned = errors.New("directory already backs another artifact repository")

// ErrArtifactNotFound is what ArtifactStore.Get wraps when the artifact no
// longer exists. gather treats a state directory bound to such an artifact
// as free: a destroyed-and-recreated resource leaves exactly that behind.
var ErrArtifactNotFound = errors.New("artifact not found")

// ArtifactInfo is the minimal artifact view Plan needs: whether the
// artifact is locked, and its current code_ref (empty CatalogID /
// CatalogVersionID before any code has ever been uploaded).
type ArtifactInfo struct {
	Locked           bool
	CatalogID        string
	CatalogVersionID string
	// RepositoryID is the artifact repository the artifact belongs to,
	// which every version of one resource shares; empty when unknown.
	RepositoryID string
}

// ArtifactStore reads and updates the artifact backing this sync. The
// resource adapts its own client service to this interface; PatchCodeRef
// wraps PatchArtifactCodeRef and discards the returned artifact. Get
// returns an error wrapping ErrArtifactNotFound for an artifact that no
// longer exists, which gather relies on to tell a re-created resource from
// a second one.
type ArtifactStore interface {
	Get(ctx context.Context, artifactID string) (ArtifactInfo, error)
	PatchCodeRef(ctx context.Context, artifactID, catalogID, catalogVersionID string) error
}

// Engine runs the CLI three-way sync pipeline (BASE / LOCAL / REMOTE)
// against a single source.dir. Construct with New, call Plan, then
// ExecuteLocal, then ExecuteRemote, then Close to release the sync lock
// acquired during preflight.
type Engine struct {
	projectDir string
	artifactID string
	files      filesapi.Client
	artifacts  ArtifactStore
	nowFn      func() time.Time
	ignore     *ignore.Matcher // nil: read the directory's own ignore file

	seedCatalogID string
	seedVersionID string

	// previousArtifactID is the artifact the caller managed before
	// artifactID; a state directory still bound to it is the caller's own.
	previousArtifactID string

	config wapi.Config
	base   BaseManifest
	// baseExtra is what the loaded manifest.json held beyond the fields
	// this build knows (a newer CLI's keys); phase 6 writes it back.
	baseExtra map[string]json.RawMessage
	local     LocalManifest
	remote    RemoteManifest
	catalogID string
	remoteVer string
	// artifactVer is the version the artifact's own code_ref pointed at
	// during gather, which is not always remoteVer: a draft cloned from a
	// locked artifact starts with no code_ref at all, while the state
	// directory still describes the version this directory last pushed.
	artifactVer string
	drifted     bool

	plan           *SyncPlan
	lock           *SyncLock
	staleNote      bool
	locked         bool
	rollback       *RollbackTree
	conflictCopies []string
	localApplied   bool

	// Set by ExecuteRemote and phase 6: the catalog the uploads landed
	// in, the version they produced, and the version persisted to
	// config.json (which falls back to the observed remote version on a
	// pull-only sync).
	newCatalogID    string
	newVersionID    string
	syncedVersionID string
}

// New constructs an Engine bound to projectDir. artifactID is the artifact
// this sync targets: it seeds config.json when the directory has no state
// yet, and replaces the recorded ID on every later Plan, so a directory
// follows its resource across artifact versions.
func New(projectDir, artifactID string, files filesapi.Client, artifacts ArtifactStore) (*Engine, error) {
	if projectDir == "" {
		return nil, errors.New("sync.New: projectDir is required")
	}
	if artifactID == "" {
		return nil, errors.New("sync.New: artifactID is required")
	}
	if files == nil {
		return nil, errors.New("sync.New: files API client is required")
	}
	if artifacts == nil {
		return nil, errors.New("sync.New: artifact store is required")
	}

	return &Engine{
		projectDir: projectDir,
		artifactID: artifactID,
		files:      files,
		artifacts:  artifacts,
		// Seam for tests that assert on the *.LOCAL.<ts> conflict-copy
		// suffix; production always uses the wall clock.
		nowFn: time.Now,
	}, nil
}

// BindCatalog tells the engine where the caller knows the directory's code
// to be: the catalog and the version its record of the artifact points at.
//
// For a directory that has code in the catalog but no state yet (the shape
// a tree is in when it was last uploaded by the push-only uploader this
// engine replaces) both seed the state directory, so that first Plan is a
// plain push (BASE empty, REMOTE not drifted) instead of creating a second
// catalog beside the one Terraform state already points at. Once
// config.json exists the file's own catalog wins. The version has one more
// use after that: an artifact with no code_ref of its own (a draft just
// cloned from a locked one) diffs against it rather than against the
// version the directory last synced, which can be older (see gather).
//
// Must be called before Plan.
//
// No CLI counterpart: `dr artifact code init` takes the same values from
// its own flags.
func (e *Engine) BindCatalog(catalogID, catalogVersionID string) {
	e.seedCatalogID = catalogID
	e.seedVersionID = catalogVersionID
}

// PreviousArtifact names the artifact the caller managed before
// artifactID, so a state directory still bound to it is recognized as the
// caller's own rather than another resource's: a locked artifact clones to
// a new draft over the same directory. Must be called before Plan.
func (e *Engine) PreviousArtifact(artifactID string) {
	e.previousArtifactID = artifactID
}

// UseIgnore makes Plan walk source.dir with m instead of loading the
// directory's own ignore file. The resource calls it when it could not
// write the starter .drignore: plan hashed the tree with the template's
// patterns, and the sync has to upload that same set rather than the wider
// one a directory with no ignore file would produce. A nil m restores the
// default. Must be called before Plan.
func (e *Engine) UseIgnore(m *ignore.Matcher) {
	e.ignore = m
}

// Plan runs phases 0-4 (preflight, gather, manifests, diff, sort) and
// returns the resulting SyncPlan without mutating the remote catalog or
// the local working tree (beyond acquiring the lock / recovering a stale
// rollback). The lock is held until Close releases it, or until a failing
// Plan releases it on the way out.
//
// Plan may be called again on the same Engine to refresh the plan (the
// resource re-plans between terraform plan and apply); the repeat call
// reuses the lock this Engine already holds instead of deadlocking
// against itself. An Engine is not safe for concurrent use.
func (e *Engine) Plan(ctx context.Context) (*SyncPlan, error) {
	if err := e.preflight(); err != nil {
		return nil, e.joinReleaseErr(err)
	}
	if err := e.gather(ctx); err != nil {
		return nil, e.joinReleaseErr(err)
	}
	if err := e.buildManifests(ctx); err != nil {
		return nil, e.joinReleaseErr(err)
	}

	plan := Diff(e.base, e.local, e.remote)
	plan.OldVersionShort = ShortVer(ptrOrEmpty(e.config.LastSyncedVersionID))
	plan.Sort()
	e.plan = plan

	return plan, nil
}

// StaleRollbackRestored reports whether preflight restored a stale
// rollback tree left by a previously interrupted sync.
func (e *Engine) StaleRollbackRestored() bool { return e.staleNote }

// ArtifactLocked reports whether the artifact this plan was built against
// is locked. The plan is still accurate — it describes what a sync would
// move — but nothing can be written into a locked artifact, so a caller
// with a non-empty plan has to mint a new version (or clone to a draft)
// before executing it. An empty plan against a locked artifact means
// there is nothing to write and so nothing to clone for.
func (e *Engine) ArtifactLocked() bool { return e.locked }

// Close releases the sync lock. Idempotent; safe to call even if Plan
// never ran or returned an error (which already releases the lock).
func (e *Engine) Close() error {
	return e.releaseLock()
}

func (e *Engine) releaseLock() error {
	if e.lock == nil {
		return nil
	}

	err := e.lock.Unlock()
	e.lock = nil

	return err
}

// joinReleaseErr releases the lock and joins any release failure with err
// so a caller sees both instead of silently losing the lock error. Only
// Plan's own failure paths call it, so the lock it drops is always one
// this Engine holds and no longer needs.
func (e *Engine) joinReleaseErr(err error) error {
	if relErr := e.releaseLock(); relErr != nil {
		return errors.Join(err, fmt.Errorf("release lock: %w", relErr))
	}

	return err
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// ShortVer truncates a hex version ID to 8 chars for display.
// CLI source: cli/internal/workload/sync/phase3_diff.go.
func ShortVer(s string) string {
	const shortLen = 8

	if len(s) > shortLen {
		return s[:shortLen]
	}

	return s
}
