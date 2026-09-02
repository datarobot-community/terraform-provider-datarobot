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
//   - Missing .wapi/ auto-initializes instead of erroring "not linked":
//     there is no `dr workload code init` step in a Terraform-managed tree.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// ErrLockedArtifact is returned by Plan when the bound artifact is locked.
// Locked artifacts are immutable; cloning to a new draft artifact remains
// the resource's job before Plan runs again (mirrors CLI phase1_gather.go).
var ErrLockedArtifact = errors.New("artifact is locked (immutable); cannot sync in place, clone to a draft first")

// ArtifactInfo is the minimal artifact view Plan needs: whether the
// artifact is locked, and its current code_ref (empty CatalogID /
// CatalogVersionID before any code has ever been uploaded).
type ArtifactInfo struct {
	Locked           bool
	CatalogID        string
	CatalogVersionID string
}

// ArtifactStore reads and updates the artifact backing this sync. The
// resource adapts its own client service to this interface; PatchCodeRef
// wraps PatchArtifactCodeRef and discards the returned artifact.
type ArtifactStore interface {
	Get(ctx context.Context, artifactID string) (ArtifactInfo, error)
	PatchCodeRef(ctx context.Context, artifactID, catalogID, catalogVersionID string) error
}

// Engine runs the CLI three-way sync pipeline (BASE / LOCAL / REMOTE)
// against a single source.dir. Construct with New, call Plan, then
// ExecuteLocal, then ExecuteRemote, then Close to release the
// .wapi/sync.lock acquired during preflight.
type Engine struct {
	projectDir string
	artifactID string
	files      filesapi.Client
	artifacts  ArtifactStore
	nowFn      func() time.Time

	config         wapi.Config
	base           BaseManifest
	local          LocalManifest
	remote         RemoteManifest
	catalogID      string
	remoteVer      string
	drifted        bool
	plan           *SyncPlan
	lock           *SyncLock
	staleNote      bool
	rollback       *RollbackTree
	conflictCopies []string
	localApplied   bool

	// Set by ExecuteRemote and phase 6: the catalog the uploads landed
	// in, the version they produced, and the version persisted to
	// .wapi/config.json (which falls back to the observed remote version
	// on a pull-only sync).
	newCatalogID    string
	newVersionID    string
	syncedVersionID string
}

// New constructs an Engine bound to projectDir. artifactID seeds .wapi/ on
// the very first Plan call, when no .wapi/config.json exists yet; once
// initialized, later Plan calls trust config.json's own ArtifactID.
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

// Plan runs phases 0-4 (preflight, gather, manifests, diff, sort) and
// returns the resulting SyncPlan without mutating the remote catalog or
// the local working tree (beyond acquiring the lock / recovering a stale
// rollback). The lock is held until Close releases it.
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
// .wapi/.rollback/ tree left by a previously interrupted sync.
func (e *Engine) StaleRollbackRestored() bool { return e.staleNote }

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
// so a caller sees both instead of silently losing the lock error.
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
