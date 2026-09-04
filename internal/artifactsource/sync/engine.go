package sync

// CLI source: cli/internal/workload/sync/engine.go
//
// Provider differences from CLI:
//   - Plan (phases 0-4, see phase.go) plus the local half of phase 5
//     (ExecuteLocal, see execute.go). The remote half (uploads, remote
//     deletes, code_ref patch) and phase 6 (persisting BASE) are not
//     implemented here.
//   - No Options{DryRun, ShowDiffs, Yes}: terraform apply has no TTY and is
//     always non-interactive, remote-wins-on-conflict (see the plan's
//     "Non-interactive policy" section) — there is nothing to gate on.
//   - ArtifactStore is a narrow interface over the caller's own artifact
//     client (Locked/CatalogID/CatalogVersionID) instead of the CLI's
//     workload.Artifact, so this package stays independent of
//     internal/client.
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

// ErrLockedArtifact reports that the bound artifact is locked and so
// cannot be written to. Plan does not return it — planning a locked
// artifact is allowed and only reports the fact via ArtifactLocked (see
// gather) — Execute does, once it lands (mirrors CLI phase1_gather.go,
// where the check is gated on previewOnly and re-run in phase 5).
var ErrLockedArtifact = errors.New("artifact is locked (immutable); cannot sync in place, clone to a draft first")

// ErrArtifactMismatch is returned by Plan when the state directory is
// already bound to a different artifact than the Engine was constructed
// for. Rebinding is not a config edit: BASE describes the *old*
// artifact's catalog, so reusing it as the common ancestor of a different
// artifact would produce a meaningless diff. Whoever wants to retarget a
// source.dir has to reset BASE with it, which is the resource's call.
var ErrArtifactMismatch = errors.New("state directory is bound to a different artifact")

// ArtifactInfo is the minimal artifact view Plan needs: whether the
// artifact is locked, and its current code_ref (empty CatalogID /
// CatalogVersionID before any code has ever been uploaded).
type ArtifactInfo struct {
	Locked           bool
	CatalogID        string
	CatalogVersionID string
}

// ArtifactStore fetches ArtifactInfo for the artifact backing this sync.
// The resource adapts its own client.Artifact to this interface.
type ArtifactStore interface {
	Get(ctx context.Context, artifactID string) (ArtifactInfo, error)
}

// Engine runs the CLI three-way sync pipeline (BASE / LOCAL / REMOTE)
// against a single source.dir. Construct with New, call Plan, then
// ExecuteLocal, then Close to release the .wapi/sync.lock acquired during
// preflight.
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
	locked         bool
	rollback       *RollbackTree
	conflictCopies []string
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
// .wapi/.rollback/ tree left by a previously interrupted sync.
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
