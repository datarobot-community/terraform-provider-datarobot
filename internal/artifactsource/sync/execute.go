package sync

// CLI source: cli/internal/workload/sync/phase5_execute.go (local half).
//
// Provider differences from CLI:
//   - Local half only. ExecuteLocal keeps the rollback tree and the
//     sync lock instead of finalizing them; the remote half (deletes,
//     uploads, PatchArtifactCodeRef) and the BASE rewrite are not run here.
//   - Always the CLI's `--yes` path: terraform apply has no TTY, so
//     conflicts resolve remote-wins with the local bytes kept as
//     *.LOCAL.<ts>. There is no prompt and no display package.
//   - RollbackTree.Backup records an absent path as a created-file
//     itself, so the CLI's separate rb.TrackCreated calls collapse into
//     Backup here.
//   - conflictCopies holds paths relative to source.dir (the CLI stores
//     absolute paths) so they can be reported in Terraform diagnostics.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// conflictCopyStampFormat is the UTC suffix appended to a conflicted
// local file: agent.py -> agent.py.LOCAL.20260901T120000Z.
const conflictCopyStampFormat = "20060102T150405Z"

// ErrNoPlan is returned when ExecuteLocal runs before Plan.
var ErrNoPlan = errors.New("sync engine: Execute called before Plan")

// ErrLockReleased is returned when ExecuteLocal runs after Close, i.e.
// without the .wapi/sync.lock that guards mutations of source.dir.
var ErrLockReleased = errors.New("sync engine: Execute called after the sync lock was released")

// ExecuteLocal applies the local half of the plan: conflict copies, then
// remote-to-local downloads and the local removals the remote asked for.
// It mutates source.dir. Every path it is about to overwrite or unlink is
// copied into .wapi/.rollback/ first, and any failure restores the tree to
// its pre-execute state.
//
// The remote half (deletes, uploads, code_ref patch) and the new BASE
// manifest are not applied here, so ExecuteLocal leaves .wapi/.rollback/
// in place: a later phase still has to run and can still need to undo
// these mutations. A caller that stops here must call DiscardRollback,
// otherwise the next Plan's stale-rollback recovery treats this run as a
// crash and reverts it.
//
// The sync lock stays held so the remote half runs in the same session;
// Close releases it. Calling ExecuteLocal after Close returns
// ErrLockReleased rather than mutating source.dir unguarded.
func (e *Engine) ExecuteLocal(ctx context.Context) error {
	if e.plan == nil {
		return ErrNoPlan
	}

	if e.plan.IsEmpty() {
		return nil
	}

	if e.lock == nil {
		return ErrLockReleased
	}

	// Reject server-controlled traversal paths before any filesystem op
	// runs, including RollbackTree.Backup, which stats and copies the
	// source path and would otherwise reach outside projectDir.
	if err := validateServerPaths(e.plan); err != nil {
		return err
	}

	if n := planFileCount(e.plan); n > RollbackMaxFiles {
		return fmt.Errorf("sync plan touches %d files, above RollbackMaxFiles=%d; refusing to run", n, RollbackMaxFiles)
	}

	rb := NewRollbackTree(e.projectDir)
	if err := rb.Init(); err != nil {
		return err
	}

	// ConflictCopies reports what is on disk, so an attempt starts from an
	// empty list rather than carrying the entries of one already rolled
	// back: a retry on the same Engine re-derives every copy from the plan.
	e.conflictCopies = nil

	if err := e.executeLocalPlan(ctx, rb); err != nil {
		if restoreErr := rb.Restore(); restoreErr != nil {
			// Restore stopped partway, so some *.LOCAL.<ts> copies may
			// still be there. Keep reporting the ones this attempt made
			// rather than hiding files the user has to deal with.
			return errors.Join(err, fmt.Errorf("restore rollback: %w", restoreErr))
		}

		// A completed restore removed every copy this attempt created.
		e.conflictCopies = nil

		return err
	}

	e.rollback = rb

	return nil
}

// DiscardRollback drops the retained .wapi/.rollback/ tree, committing the
// mutations ExecuteLocal made. Phase 6 calls this once the new BASE
// manifest is on disk; it is idempotent.
func (e *Engine) DiscardRollback() error {
	if e.rollback == nil {
		return nil
	}

	err := e.rollback.Discard()
	e.rollback = nil

	return err
}

// ConflictCopies returns the *.LOCAL.<ts> paths ExecuteLocal created,
// relative to source.dir.
func (e *Engine) ConflictCopies() []string { return e.conflictCopies }

func (e *Engine) executeLocalPlan(ctx context.Context, rb *RollbackTree) error {
	if err := e.applyConflictCopies(rb); err != nil {
		return fmt.Errorf("conflict copies: %w", err)
	}

	if err := e.applyDownloads(ctx, rb); err != nil {
		return fmt.Errorf("downloads: %w", err)
	}

	return nil
}

// applyConflictCopies renames every conflicted local file to
// <path>.LOCAL.<ts> so the remote bytes can land at the original path.
// EDIT_DEL_CONFLICT (local deleted, remote edited) never reaches this
// list: ActionFor maps it to ActDownloadOverDel, so it is a plain
// download and there is no local edit left to preserve.
func (e *Engine) applyConflictCopies(rb *RollbackTree) error {
	stamp := e.nowFn().UTC().Format(conflictCopyStampFormat)

	for _, fa := range e.plan.Conflicts {
		copyRel := fa.Path + ".LOCAL." + stamp

		// Backing up the original records it for restore; backing up the
		// (still absent) copy target records it for removal on restore.
		if err := rb.Backup(fa.Path); err != nil {
			return err
		}

		if err := rb.Backup(copyRel); err != nil {
			return err
		}

		src := filepath.Join(e.projectDir, filepath.FromSlash(fa.Path))
		if err := os.Rename(src, filepath.Join(e.projectDir, filepath.FromSlash(copyRel))); err != nil {
			return fmt.Errorf("rename %s -> %s: %w", fa.Path, copyRel, err)
		}

		e.conflictCopies = append(e.conflictCopies, copyRel)
	}

	return nil
}

// applyDownloads backs up everything the remote is about to overwrite or
// unlink, removes the files the remote deleted, then pulls remote bytes
// for the download rows plus the conflict paths just renamed away.
func (e *Engine) applyDownloads(ctx context.Context, rb *RollbackTree) error {
	pulls := pullList(e.plan)
	removals := remoteDeletedPaths(e.plan)

	if len(pulls) == 0 && len(removals) == 0 {
		return nil
	}

	if e.catalogID == "" || e.remoteVer == "" {
		// Unreachable for a plan built by Diff: remote-side rows require
		// a REMOTE manifest, which requires a code_ref. Fail loudly
		// rather than silently skipping remote-wins work, since conflict
		// copies have already renamed the local originals away.
		return fmt.Errorf("plan pulls %d file(s) and removes %d locally, but the artifact has no code_ref", len(pulls), len(removals))
	}

	if err := e.backupDownloadTargets(rb, removals); err != nil {
		return err
	}

	if err := e.removeRemoteDeletedFiles(removals); err != nil {
		return err
	}

	return e.downloadFiles(ctx, e.catalogID, e.remoteVer, pulls)
}

func (e *Engine) backupDownloadTargets(rb *RollbackTree, removals []string) error {
	for _, fa := range e.plan.Downloads {
		if err := rb.Backup(fa.Path); err != nil {
			return err
		}
	}

	// ActDownloadDelete rows live in plan.Deletes, so back them up here,
	// before removeRemoteDeletedFiles unlinks them. Conflict paths were
	// already backed up by applyConflictCopies.
	for _, path := range removals {
		if err := rb.Backup(path); err != nil {
			return err
		}
	}

	return nil
}

// removeRemoteDeletedFiles deletes the local copies of files the remote
// dropped. A missing file is tolerated so a retried apply stays idempotent.
func (e *Engine) removeRemoteDeletedFiles(paths []string) error {
	for _, path := range paths {
		// ExecuteLocal already rejected unsafe server paths up front via
		// validateServerPaths; re-check here so the per-call-site
		// invariant survives future refactors that bypass that entry point.
		if err := filesapi.SafeRelPath(path); err != nil {
			return fmt.Errorf("server returned unsafe delete path %q: %w", path, err)
		}

		abs := filepath.Join(e.projectDir, filepath.FromSlash(path))
		if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}

	return nil
}

// pullList collects the rows that need remote bytes on disk: every
// download, plus the conflicts whose remote side still exists.
// DEL_EDIT_CONFLICT (local edited, remote deleted) is skipped — there is
// nothing to pull, only the rename to *.LOCAL.<ts>.
func pullList(plan *SyncPlan) []FileAction {
	out := make([]FileAction, 0, len(plan.Downloads)+len(plan.Conflicts))
	out = append(out, plan.Downloads...)

	for _, fa := range plan.Conflicts {
		if fa.Classification == ClsDelEditConflict {
			continue
		}

		out = append(out, fa)
	}

	return out
}

// remoteDeletedPaths returns the paths the remote deleted, which the
// executor removes locally (REMOTE_DELETED / ActDownloadDelete). The
// other half of plan.Deletes (ActUploadDelete) is the remote half's job.
func remoteDeletedPaths(plan *SyncPlan) []string {
	out := make([]string, 0, len(plan.Deletes))

	for _, fa := range plan.Deletes {
		if fa.Action == ActDownloadDelete {
			out = append(out, fa.Path)
		}
	}

	return out
}

// validateServerPaths rejects plan rows whose path came from the Files
// API and is not safe to join with projectDir. Uploads and ActUploadDelete
// rows are skipped: those paths come from the local walker, which already
// produced them by walking projectDir.
func validateServerPaths(plan *SyncPlan) error {
	for _, fa := range plan.Downloads {
		if err := filesapi.SafeRelPath(fa.Path); err != nil {
			return fmt.Errorf("server returned unsafe download path %q: %w", fa.Path, err)
		}
	}

	for _, fa := range plan.Conflicts {
		if err := filesapi.SafeRelPath(fa.Path); err != nil {
			return fmt.Errorf("server returned unsafe conflict path %q: %w", fa.Path, err)
		}
	}

	for _, path := range remoteDeletedPaths(plan) {
		if err := filesapi.SafeRelPath(path); err != nil {
			return fmt.Errorf("server returned unsafe delete path %q: %w", path, err)
		}
	}

	return nil
}

func planFileCount(plan *SyncPlan) int {
	return len(plan.Uploads) + len(plan.Downloads) + len(plan.Deletes) + len(plan.Conflicts)
}
