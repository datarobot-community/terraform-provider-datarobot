package sync

// CLI source: cli/internal/workload/sync/phase5_execute.go (remote half:
// applyRemoteDeletesAndUploads / applyDeletes), upload_stage.go,
// upload_zip.go.
//
// Provider differences from CLI:
//   - The stage and zip uploaders are not re-ported. This file turns the
//     plan's Uploads rows into artifactsource.LocalFile values and calls
//     internal/artifactsource.UploadFiles, which already carries the
//     ported stage/zip split, the <=20 files / <=50 MB thresholds, the
//     REPLACE overwrite mode and the async zip status polling. CLI's
//     Uploader interface and ChooseUploader(plan) therefore have no
//     counterpart here.
//   - ExecuteRemote is a separate entry point from ExecuteLocal instead of
//     one phase5Execute: local filesystem mutation and remote catalog
//     mutation are reviewed and wired separately (see execute.go).
//   - ExecuteRemote refuses to run before ExecuteLocal, so remote-wins
//     conflict resolution always lands before local bytes are pushed.
//   - No Result.Duration / history entry: the CLI reports timings through
//     its own display package, which is not ported.

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// ErrLocalNotApplied is returned when ExecuteRemote runs before
// ExecuteLocal. The remote half pushes local bytes, so the remote-wins
// half has to have resolved conflicts and downloads first.
var ErrLocalNotApplied = errors.New("sync engine: ExecuteRemote called before ExecuteLocal")

// Result reports what a completed sync did. CatalogID and
// CatalogVersionID are the code_ref the artifact now points at, and are
// what the resource writes back into Terraform state; PreviousVersionID is
// the catalog version it pointed at when Plan ran, empty on a first sync.
type Result struct {
	CatalogID         string
	CatalogVersionID  string
	PreviousVersionID string
	Uploaded          int
	Downloaded        int
	Deleted           int
	Conflicts         int
	ConflictCopies    []string
}

// ExecuteRemote applies the remote half of the plan and then persists the
// new BASE: it deletes the paths the user removed locally, uploads added
// and modified files, patches the artifact's primary code_ref to the
// resulting catalog version, and writes .wapi/config.json plus
// .wapi/manifest.json.
//
// A failure in the remote half restores source.dir from the
// .wapi/.rollback/ tree ExecuteLocal left behind, so a failed apply does
// not leave downloaded or renamed files in the user's directory. A failure
// while persisting BASE deliberately does not: the catalog has already
// advanced, and undoing it is not possible, so the next Plan reconciles
// instead (the retained rollback tree makes that next run start from the
// pre-sync tree).
//
// The sync lock is not released here; Close does that.
func (e *Engine) ExecuteRemote(ctx context.Context) (*Result, error) {
	if e.plan == nil {
		return nil, ErrNoPlan
	}

	if e.lock == nil {
		return nil, ErrLockReleased
	}

	if !e.localApplied {
		return nil, ErrLocalNotApplied
	}

	if err := e.applyRemoteDeletesAndUploads(ctx); err != nil {
		return nil, e.restoreAfterRemoteFailure(err)
	}

	if err := e.persistState(); err != nil {
		return nil, err
	}

	return e.result(), nil
}

// restoreAfterRemoteFailure undoes the working-tree mutations ExecuteLocal
// made, joining any restore failure with the original error so a caller
// sees both. An empty plan leaves no rollback tree, so this is a no-op.
func (e *Engine) restoreAfterRemoteFailure(err error) error {
	if e.rollback == nil {
		return err
	}

	restoreErr := e.rollback.Restore()
	e.rollback = nil

	if restoreErr != nil {
		return errors.Join(err, fmt.Errorf("restore rollback: %w", restoreErr))
	}

	return err
}

// applyRemoteDeletesAndUploads sends LOCAL_DELETED paths to the Files API,
// uploads LOCAL_ADDED and LOCAL_MODIFIED files, and patches the artifact's
// code_ref when either produced a new catalog version.
func (e *Engine) applyRemoteDeletesAndUploads(ctx context.Context) error {
	newCatalogID := e.catalogID
	newVersionID := e.remoteVer

	deletedVer, err := e.applyRemoteDeletes(ctx)
	if err != nil {
		return err
	}

	if deletedVer != "" {
		newVersionID = deletedVer
	}

	if len(e.plan.Uploads) > 0 {
		catalogID, versionID, uploadErr := e.applyUploads(ctx)
		if uploadErr != nil {
			return uploadErr
		}

		newCatalogID = catalogID

		if versionID != "" {
			newVersionID = versionID
		}
	}

	if newVersionID != "" && newVersionID != e.remoteVer {
		if err := e.artifacts.PatchCodeRef(ctx, e.config.ArtifactID, newCatalogID, newVersionID); err != nil {
			return fmt.Errorf("update artifact code_ref: %w", err)
		}
	}

	e.newCatalogID = newCatalogID
	e.newVersionID = newVersionID

	return nil
}

// applyRemoteDeletes removes the catalog entries for files the user
// deleted locally, and returns the catalog version the delete produced.
// Nothing to delete, or no catalog yet (first sync), returns "".
func (e *Engine) applyRemoteDeletes(ctx context.Context) (string, error) {
	paths := uploadDeletedPaths(e.plan)
	if e.catalogID == "" || len(paths) == 0 {
		return "", nil
	}

	resp, err := e.files.DeleteFiles(ctx, e.catalogID, paths)
	if err != nil {
		return "", fmt.Errorf("delete remote files: %w", err)
	}

	if resp == nil {
		return "", nil
	}

	return resp.CatalogVersionID, nil
}

// applyUploads pushes the plan's Uploads rows through the existing
// stage/zip backend in internal/artifactsource. Upload paths come from the
// local walker, so they are already validated (see validateServerPaths).
func (e *Engine) applyUploads(ctx context.Context) (string, string, error) {
	files := make([]artifactsource.LocalFile, 0, len(e.plan.Uploads))

	for _, fa := range e.plan.Uploads {
		files = append(files, artifactsource.LocalFile{
			RelPath: fa.Path,
			AbsPath: filepath.Join(e.projectDir, filepath.FromSlash(fa.Path)),
			Size:    fa.LocalSize,
			Hash:    fa.LocalHash,
		})
	}

	catalogID, versionID, err := artifactsource.UploadFiles(ctx, e.files, e.catalogID, filesapi.OverwriteReplace, files)
	if err != nil {
		return "", "", fmt.Errorf("upload %d file(s): %w", len(files), err)
	}

	return catalogID, versionID, nil
}

// uploadDeletedPaths returns the paths the user deleted locally, which the
// executor removes from the catalog (LOCAL_DELETED / ActUploadDelete). The
// other half of plan.Deletes (ActDownloadDelete) is ExecuteLocal's job.
func uploadDeletedPaths(plan *SyncPlan) []string {
	out := make([]string, 0, len(plan.Deletes))

	for _, fa := range plan.Deletes {
		if fa.Action == ActUploadDelete {
			out = append(out, fa.Path)
		}
	}

	return out
}

// result summarizes the finished sync for the caller.
func (e *Engine) result() *Result {
	return &Result{
		CatalogID:         e.newCatalogID,
		CatalogVersionID:  e.syncedVersionID,
		PreviousVersionID: e.remoteVer,
		Uploaded:          len(e.plan.Uploads),
		Downloaded:        len(e.plan.Downloads),
		Deleted:           len(e.plan.Deletes),
		Conflicts:         len(e.plan.Conflicts),
		ConflictCopies:    e.conflictCopies,
	}
}
