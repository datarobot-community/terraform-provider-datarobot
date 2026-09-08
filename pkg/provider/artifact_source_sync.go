package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	artifactsync "github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/sync"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func artifactSourceConfigured(data *ArtifactResourceModel) bool {
	return data.Source != nil && IsKnown(data.Source.Dir)
}

func artifactSourceNeedsUpload(plan, state *ArtifactResourceModel, priorArtifactID, newArtifactID string) bool {
	if !artifactSourceConfigured(plan) {
		return false
	}
	if priorArtifactID != newArtifactID {
		return true
	}
	if state == nil || state.Source == nil || !IsKnown(state.Source.DirHash) {
		return true
	}
	return !plan.Source.DirHash.Equal(state.Source.DirHash)
}

// artifactSourceRemoteDrifted reports whether the catalog version the
// artifact points at has moved past the one source.dir last synced. State
// carries the artifact's code_ref as Read refreshed it; the directory's
// last sync is in its state directory. They differ when the code_ref was
// re-pointed from somewhere else: the DataRobot CLI syncing this artifact
// from another checkout, or this resource last applied from one. Without
// this a plan for an unchanged tree shows nothing, and the catalog's
// changes never come down.
//
// A directory with no state yet has nothing to compare. One bound to a
// different catalog is left alone: that mismatch is refused, with the way
// out spelled out, once the tree changes and a sync actually runs.
func artifactSourceRemoteDrifted(plan, state *ArtifactResourceModel) bool {
	if state == nil || !artifactSourceConfigured(plan) {
		return false
	}

	liveVersion := catalogVersionIDFromModel(state)
	if liveVersion == "" {
		return false
	}

	absDir, err := artifactSourceAbsDir(plan)
	if err != nil {
		return false
	}

	cfg, err := wapi.LoadConfig(absDir)
	if err != nil {
		return false
	}

	if cfg.CatalogID != nil && *cfg.CatalogID != "" && *cfg.CatalogID != catalogIDFromModel(state) {
		return false
	}

	lastSynced := ""
	if cfg.LastSyncedVersionID != nil {
		lastSynced = *cfg.LastSyncedVersionID
	}

	return lastSynced != "" && lastSynced != liveVersion
}

// artifactSourceDriftWarning explains a plan whose dir_hash is known after
// apply while nothing under source.dir changed: the catalog moved.
func artifactSourceDriftWarning(diags *diag.Diagnostics, plan, state *ArtifactResourceModel) {
	if diags == nil {
		return
	}

	diags.AddAttributeWarning(
		path.Root("source").AtName("dir"),
		"Catalog changed since the last sync",
		fmt.Sprintf(
			"The artifact's code is at catalog version %s, which %s has not synced yet. "+
				"Apply brings the catalog's changes down into the directory (every file it writes or removes is reported) and, on a locked artifact, builds a new version that includes them. "+
				"A file changed on both sides fails the apply and is named.",
			catalogVersionIDFromModel(state), plan.Source.Dir.ValueString()),
	)
}

func catalogIDFromModel(data *ArtifactResourceModel) string {
	if data == nil || data.Spec == nil {
		return ""
	}
	for _, group := range data.Spec.ContainerGroups {
		for _, container := range group.Containers {
			ref := imageBuildConfigCodeRef(container.ImageBuildConfig)
			if ref == nil {
				continue
			}
			if IsKnown(ref.CatalogID) {
				return ref.CatalogID.ValueString()
			}
		}
	}
	return ""
}

func catalogVersionIDFromModel(data *ArtifactResourceModel) string {
	if data == nil || data.Spec == nil {
		return ""
	}
	for _, group := range data.Spec.ContainerGroups {
		for _, container := range group.Containers {
			ref := imageBuildConfigCodeRef(container.ImageBuildConfig)
			if ref == nil {
				continue
			}
			if IsKnown(ref.CatalogVersionID) {
				return ref.CatalogVersionID.ValueString()
			}
		}
	}
	return ""
}

// artifactSourceStore adapts the provider's client service to
// sync.ArtifactStore. The artifact the resource already holds answers the
// engine's read (it was just created or patched, so a GET would only
// re-fetch what we have), and the artifact returned by the code_ref patch
// is kept so the caller can write the server's fresh view into state
// instead of the engine discarding it.
type artifactSourceStore struct {
	service client.Service
	current *client.Artifact
	patched *client.Artifact
}

func (s *artifactSourceStore) Get(ctx context.Context, artifactID string) (artifactsync.ArtifactInfo, error) {
	artifact := s.current
	if artifact == nil || artifact.ID != artifactID {
		traceAPICall("GetArtifact")
		fetched, err := s.service.GetArtifact(ctx, artifactID)
		if err != nil {
			return artifactsync.ArtifactInfo{}, err
		}
		artifact = fetched
	}

	info := artifactsync.ArtifactInfo{Locked: artifact.Status == client.ArtifactStatusLocked}
	if ref := client.ExtractCodeRef(artifact); ref != nil {
		info.CatalogID = ref.CatalogID
		info.CatalogVersionID = ref.CatalogVersionID
	}

	return info, nil
}

func (s *artifactSourceStore) PatchCodeRef(ctx context.Context, artifactID, catalogID, catalogVersionID string) error {
	traceAPICall("PatchArtifactCodeRef")
	artifact, err := s.service.PatchArtifactCodeRef(ctx, artifactID, catalogID, catalogVersionID)
	if err != nil {
		return err
	}
	s.patched = artifact

	return nil
}

// artifactSourceCatalogBinding is the catalog the directory already has
// code in, for a tree that has never been synced by the engine and so has
// no sync state directory yet. Terraform state is preferred over the
// artifact's live code_ref: a clone of a locked artifact is a fresh draft
// with no code_ref, and its new version still belongs in the catalog state
// points at.
func artifactSourceCatalogBinding(state *ArtifactResourceModel, artifact *client.Artifact) (catalogID, versionID string) {
	catalogID = catalogIDFromModel(state)
	versionID = catalogVersionIDFromModel(state)

	ref := client.ExtractCodeRef(artifact)
	if ref == nil {
		return catalogID, versionID
	}

	if catalogID == "" {
		catalogID = ref.CatalogID
	}
	if versionID == "" {
		versionID = ref.CatalogVersionID
	}

	return catalogID, versionID
}

// runArtifactSourceSync reconciles absDir with the catalog using the CLI
// three-way sync engine: BASE (the last synced manifest under
// absDir/.datarobot/workload/) against LOCAL (the directory, minus .drignore
// and system excludes) against REMOTE (the Files API catalog). It returns
// the sync result and, when the engine repointed the artifact, the patched
// artifact.
//
// Unlike the push-only uploader it replaces, this can also write to
// source.dir: remote-only files are downloaded and files the catalog
// dropped are removed, each announced through diags. A file edited on both
// sides since the last sync is refused before anything is written, the way
// the CLI's non-interactive mode refuses it: terraform apply has no TTY to
// ask on, and picking the catalog's version silently would replace an edit
// the user has not seen lose.
//
// seeded is the matcher to walk with when the starter ignore file could not
// be written, nil when the directory is the source of truth as usual.
func (r *ArtifactResource) runArtifactSourceSync(
	ctx context.Context,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	absDir string,
	seeded *ignore.Matcher,
	diags *diag.Diagnostics,
) (result *artifactsync.Result, patched *client.Artifact, err error) {
	store := &artifactSourceStore{service: r.provider.service, current: artifact}

	engine, err := artifactsync.New(absDir, artifact.ID, r.provider.service.FilesAPI(), store)
	if err != nil {
		return nil, nil, err
	}
	engine.BindCatalog(artifactSourceCatalogBinding(state, artifact))
	engine.UseIgnore(seeded)

	// Close releases the sync lock. A failure to release would make the
	// next apply fail on a lock nobody holds, so it is reported rather
	// than dropped, unless the sync itself already failed.
	defer func() {
		if closeErr := engine.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("release sync lock: %w", closeErr)
		}
	}()

	traceAPICall("SyncArtifactSource")
	plan, err := engine.Plan(ctx)
	if err != nil {
		return nil, nil, artifactSourceLockHint(err, absDir)
	}

	if engine.StaleRollbackRestored() {
		artifactSourceStaleRollbackWarning(diags, absDir)
	}

	if plan.HasConflicts() {
		return nil, nil, artifactSourceConflictError(absDir, plan.ConflictPaths())
	}

	// A locked artifact whose directory already matches it: nothing to
	// write, and the engine would (rightly) refuse to record a sync
	// against something immutable. The resource does not normally get
	// here, since it clones a locked artifact before syncing a changed
	// tree and keeps the version when nothing changed, but a caller that
	// does has nothing to do rather than an error to report.
	if engine.ArtifactLocked() && plan.IsEmpty() {
		return nil, nil, nil
	}

	if err := engine.ExecuteLocal(ctx); err != nil {
		return nil, nil, err
	}

	result, err = engine.ExecuteRemote(ctx)
	if err != nil {
		// The one failure that keeps the files ExecuteLocal wrote: the
		// catalog advanced, so the working tree was left matching it and
		// only the state directory is behind. Say what changed on disk
		// now, since this error is the last thing the apply prints.
		var persistErr *artifactsync.StatePersistError
		if errors.As(err, &persistErr) {
			artifactSourceMutationWarning(diags, plan)
		}

		return nil, nil, err
	}

	artifactSourceMutationWarning(diags, plan)

	return result, store.patched, nil
}

// artifactSourceLockHint adds the likely cause to the engine's refusal to
// take the sync lock. The lock is per directory and per open file, so the
// other holder is usually not another process at all but a second
// datarobot_artifact resource in this same apply pointing at the same
// source.dir, which Terraform runs in parallel with this one.
func artifactSourceLockHint(err error, absDir string) error {
	if !errors.Is(err, artifactsync.ErrLocked) {
		return err
	}

	return fmt.Errorf("%w: %s is being synced by another sync, either a second datarobot_artifact resource with the same source.dir or a DataRobot CLI sync still running; "+
		"a directory can back only one resource, so give each its own", err, absDir)
}

// artifactSourceStaleRollbackWarning reports the one write into source.dir
// that happens before the plan is even built: a previous apply was
// interrupted between writing files and recording the sync, and preflight
// put the directory back the way that apply found it, from the copies it
// had kept.
func artifactSourceStaleRollbackWarning(diags *diag.Diagnostics, absDir string) {
	if diags == nil {
		return
	}

	diags.AddAttributeWarning(
		path.Root("source").AtName("dir"),
		"Interrupted sync rolled back",
		fmt.Sprintf(
			"A previous apply was interrupted while syncing %s, after it had started writing files there. "+
				"Those files were put back the way that apply found them before this sync planned, so the directory may differ from what you last saw in it. "+
				"Review it (a version control diff shows what moved) before committing.",
			absDir),
	)
}

// artifactSourceConflictError is the refusal for a plan in which a file
// changed both under source.dir and in the catalog since the last sync.
// The DataRobot CLI reads the same state directory, so it can resolve the
// conflict in place; the apply itself has touched nothing.
func artifactSourceConflictError(absDir string, conflicts []string) error {
	return fmt.Errorf(
		"%d file(s) changed both in %s and in the catalog since the last sync:\n%s\n\n"+
			"terraform apply cannot ask which side wins, so nothing was uploaded or written. "+
			"Resolve the conflict in that directory with the DataRobot CLI: `dr artifact code sync` shows both versions and asks per file, "+
			"and `dr artifact code sync --yes --accept-remote` takes the catalog version while keeping your copy as <path>.LOCAL.<timestamp>. "+
			"Then run terraform apply again",
		len(conflicts), absDir, artifactSourcePathList(conflicts))
}

// artifactSourceMutationWarning names the files the sync wrote or removed
// under source.dir. Terraform only plans the resource's attributes, so a
// download or a local removal would otherwise first show up in a later
// `git status`.
func artifactSourceMutationWarning(diags *diag.Diagnostics, plan *artifactsync.SyncPlan) {
	if diags == nil || plan == nil {
		return
	}

	written := make([]string, 0, len(plan.Downloads))
	for _, fa := range plan.Downloads {
		written = append(written, fa.Path)
	}

	var removed []string
	for _, fa := range plan.Deletes {
		if fa.Action == artifactsync.ActDownloadDelete {
			removed = append(removed, fa.Path)
		}
	}

	if len(written)+len(removed) == 0 {
		return
	}

	var b strings.Builder
	b.WriteString("The catalog had changes this directory did not, so the sync brought them down.")
	if len(written) > 0 {
		fmt.Fprintf(&b, "\n\nWritten (%d):\n%s", len(written), artifactSourcePathList(written))
	}
	if len(removed) > 0 {
		fmt.Fprintf(&b, "\n\nRemoved (%d):\n%s", len(removed), artifactSourcePathList(removed))
	}
	b.WriteString("\n\nThese files now match the catalog. Review the change (a version control diff shows exactly what moved) before committing it.")

	diags.AddAttributeWarning(path.Root("source").AtName("dir"), "Source directory updated from the catalog", b.String())
}

// artifactSourcePathListMax bounds the paths a diagnostic spells out; the
// count in the same message says how many there really were.
const artifactSourcePathListMax = 20

func artifactSourcePathList(paths []string) string {
	shown := paths
	if len(shown) > artifactSourcePathListMax {
		shown = shown[:artifactSourcePathListMax]
	}

	lines := make([]string, 0, len(shown)+1)
	for _, p := range shown {
		lines = append(lines, "  "+p)
	}
	if rest := len(paths) - len(shown); rest > 0 {
		lines = append(lines, fmt.Sprintf("  ... and %d more", rest))
	}

	return strings.Join(lines, "\n")
}

// plannedArtifactSourceDirHash is the dir_hash a plan commits to. A tree
// that matches state keeps the known value, so the plan shows no diff. A
// tree that differs plans as unknown rather than as the digest just
// computed: apply reconciles the directory with the catalog and can add or
// remove files while doing so, and the value saved afterwards has to be the
// digest of what is on disk then. A known value here would fail that apply
// with "inconsistent result" whenever the sync pulled anything.
func plannedArtifactSourceDirHash(current types.String, state *ArtifactResourceModel) types.String {
	if state != nil && state.Source != nil && IsKnown(state.Source.DirHash) && current.Equal(state.Source.DirHash) {
		return current
	}

	return types.StringUnknown()
}

// artifactSourceAbsDir resolves source.dir. The ignore file is looked up at that
// path and the fingerprint walks from it, so everything that touches the source
// tree resolves it the same way.
func artifactSourceAbsDir(data *ArtifactResourceModel) (string, error) {
	dir := data.Source.Dir.ValueString()
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve source directory %q: %w", dir, err)
	}

	return absDir, nil
}

// seedArtifactSourceIgnoreFile writes the starter .drignore at absDir when
// generate_ignore is on and the project has neither ignore file. It returns the
// matcher the upload must use in place of reading the directory, nil when the
// directory can be read as usual.
//
// It runs ahead of artifactSourceNeedsUpload rather than inside the upload,
// because plan has already accounted for the file: with generate_ignore on and
// no ignore file present, computeArtifactSourceDirHash folds a synthetic
// .drignore into dir_hash. A tree whose only pending change is that file
// therefore plans as unchanged, the upload is skipped, and the file plan
// promised is never written -- so a config that flips generate_ignore to true,
// or state carried across a provider upgrade, could record generate_ignore =
// true with nothing on disk. Seeding before the gate makes the promise hold
// whether or not anything was uploaded.
//
// A failed write is a warning rather than an error. generate_ignore defaults to
// true, so failing here would break applies that worked before the attribute
// existed: a source.dir mounted read-only in CI, a 0555 tree, or a directory
// sitting at the .drignore name. The upload falls back to the template's own
// patterns, which is the set plan hashed, so the failure costs the user the
// file on disk and nothing else. In particular it does not widen the upload.
func seedArtifactSourceIgnoreFile(absDir string, generateIgnore bool, diags *diag.Diagnostics) *ignore.Matcher {
	if !generateIgnore {
		return nil
	}

	_, err := ignore.WriteDefaultDrignoreIfMissing(absDir)
	if err == nil {
		return nil
	}

	// WriteDefaultDrignoreIfMissing already names the path it failed to write,
	// so this adds the consequence and the ways out, not the path again.
	diags.AddAttributeWarning(
		path.Root("source").AtName("dir"),
		fmt.Sprintf("Could not write %s", ignore.FileName),
		fmt.Sprintf(
			"%s\n\n"+
				"The upload continues with the default patterns, the same set this plan "+
				"hashed, so nothing extra is uploaded. What is missing is the file on disk, "+
				"which means this repeats on every apply. Add a %s to the directory yourself, "+
				"make the directory writable, or set generate_ignore = false.",
			err, ignore.FileName),
	)

	return ignore.FromDefaultTemplate()
}

func (r *ArtifactResource) syncArtifactSource(
	ctx context.Context,
	plan *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	priorArtifactID string,
	diags *diag.Diagnostics,
) (*client.Artifact, bool, error) {
	// Seeding reads source.dir before artifactSourceNeedsUpload gets to answer
	// for an unconfigured source, so the check it used to rely on happens here.
	if !artifactSourceConfigured(plan) {
		return artifact, false, nil
	}

	absDir, err := artifactSourceAbsDir(plan)
	if err != nil {
		return nil, false, err
	}

	// Deliberately ahead of the sync gate below: see seedArtifactSourceIgnoreFile.
	seeded := seedArtifactSourceIgnoreFile(absDir, artifactSourceGenerateIgnore(plan), diags)

	if !artifactSourceNeedsUpload(plan, state, priorArtifactID, artifact.ID) {
		return artifact, false, nil
	}

	artifactApplyProgressUploading(artifact.ID)

	result, patched, err := r.runArtifactSourceSync(ctx, state, artifact, absDir, seeded, diags)
	if err != nil {
		return nil, false, fmt.Errorf("sync artifact source: %w", err)
	}

	if result == nil {
		return artifact, false, nil
	}

	artifactApplyProgressSourceSynced(result)

	if patched != nil {
		artifact = patched
	}

	return artifact, true, nil
}

// artifactApplyProgressSourceSynced reports the sync's counts on the apply
// progress stream. Files written or removed under source.dir are also
// raised as a warning diagnostic (artifactSourceMutationWarning), since the
// progress stream is only visible with TF_LOG set.
func artifactApplyProgressSourceSynced(result *artifactsync.Result) {
	if result == nil {
		return
	}

	if result.Uploaded+result.Downloaded+result.DeletedRemote+result.DeletedLocal == 0 {
		emitArtifactApplyProgress("Source already matches the catalog; nothing to sync.")
		return
	}

	emitArtifactApplyProgress(fmt.Sprintf(
		"Synced source: %d uploaded, %d downloaded, %d removed from the catalog, %d removed locally.",
		result.Uploaded, result.Downloaded, result.DeletedRemote, result.DeletedLocal,
	))
}

// syncArtifactSourceAndBuild syncs source when needed, then triggers an image build
// when the sync ran against a draft artifact with image_build_config.
func (r *ArtifactResource) syncArtifactSourceAndBuild(
	ctx context.Context,
	plan *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	priorArtifactID string,
	diags *diag.Diagnostics,
) (*client.Artifact, error) {
	artifact, uploaded, err := r.syncArtifactSource(ctx, plan, state, artifact, priorArtifactID, diags)
	if err != nil {
		return nil, err
	}

	if !artifactBuildNeededAfterUpload(plan, artifact, uploaded) {
		return artifact, nil
	}

	waitForBuild := artifactSourceWaitForBuild(plan)
	repoID := ""
	if artifact.ArtifactRepositoryID != nil {
		repoID = *artifact.ArtifactRepositoryID
	}
	artifact, _, err = r.syncArtifactBuild(ctx, artifact.ID, repoID, waitForBuild, nil)
	if err != nil {
		return artifact, &artifactBuildSyncError{cause: err}
	}

	return artifact, nil
}

func (r *ArtifactResource) rollbackArtifactCreate(ctx context.Context, artifact *client.Artifact, deleteRepository bool) {
	if artifact == nil || !deleteRepository || artifact.ArtifactRepositoryID == nil {
		return
	}
	traceAPICall("DeleteArtifactRepository")
	_ = r.provider.service.DeleteArtifactRepository(ctx, *artifact.ArtifactRepositoryID)
}

// artifactSourcePendingUpload reports whether the planned source tree differs from state.
func artifactSourcePendingUpload(plan, state *ArtifactResourceModel, priorArtifactID string) bool {
	return artifactSourceConfigured(plan) &&
		artifactSourceNeedsUpload(plan, state, priorArtifactID, priorArtifactID)
}

// artifactLockedSourceCloneNeeded is true when a locked artifact needs a draft clone
// before upload (source dir change or spec change that creates a new version).
// Locked artifacts are immutable; the provider clones to draft, uploads, patches code_ref,
// then locks the new version (mirrors CLI guidance in cli/internal/workload/sync/phase1_gather.go).
func artifactLockedSourceCloneNeeded(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	if !artifactSourceConfigured(&plan) {
		return false
	}
	priorArtifactID := state.ArtifactID.ValueString()
	if artifactSourceNeedsUpload(&plan, &state, priorArtifactID, priorArtifactID) {
		return true
	}
	return artifactNeedsNewVersion(plan, state)
}

// artifactUpdateKeepsLockedVersion reports whether an update of a locked
// artifact changes nothing on the server. A locked version is immutable,
// so an update either mints a new version or leaves the artifact alone. It
// leaves it alone when the artifact stays locked, the spec is unchanged
// and the source tree has nothing to sync, which is what is left once the
// clone case is ruled out: source.wait_for_build, source.generate_ignore,
// or a source.dir path whose content matches state. Minting a version for
// those would deploy a copy of the same code for nothing, and then fail
// the apply on top: the plan promised artifact_id would not change, and
// the sync would refuse to record itself against a locked artifact.
func artifactUpdateKeepsLockedVersion(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	if plan.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	if artifactNeedsNewVersion(plan, state) {
		return false
	}
	return !artifactSourcePendingUpload(&plan, &state, state.ArtifactID.ValueString())
}

// artifactSourceDeferLock is true when a draft→locked transition must wait until after source upload.
func artifactSourceDeferLock(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusDraft) {
		return false
	}
	if plan.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	return artifactSourcePendingUpload(&plan, &state, state.ArtifactID.ValueString())
}

// artifactModifyPlanNeedsUnknownArtifactID is true when apply will produce a new artifact version.
func artifactModifyPlanNeedsUnknownArtifactID(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	if plan.Status.ValueString() == string(client.ArtifactStatusDraft) {
		return true
	}
	if artifactNeedsNewVersion(plan, state) {
		return true
	}
	return artifactLockedSourceCloneNeeded(plan, state)
}

// lockArtifact promotes a draft artifact to locked via PATCH {"status": "locked"}.
// Ported from CLI LockArtifact in cli/internal/workload/artifact.go.
func (r *ArtifactResource) lockArtifact(ctx context.Context, artifactID string) (*client.Artifact, error) {
	locked := client.ArtifactStatusLocked
	traceAPICall("PatchArtifact")
	return r.provider.service.PatchArtifact(ctx, artifactID, &client.PatchArtifactRequest{
		Status: &locked,
	})
}

// refreshArtifactSourceDirHash sets source.dir_hash from the current local tree.
// Call only after a successful Create/Update so state records the hash that was
// just uploaded. Do not call from Read: terraform refresh runs before plan, and
// replacing the last-applied hash with the current disk hash hides local edits.
//
// A tree that cannot be hashed leaves dir_hash null rather than whatever the
// plan held. The plan value is unknown whenever the tree changed, and an
// unknown in the state an apply returns is rejected by Terraform, which then
// discards the whole result and with it the artifact just created. Null costs
// one more sync: the next plan sees no recorded hash and syncs the directory
// again, which uploads nothing if the tree did not change.
func refreshArtifactSourceDirHash(data *ArtifactResourceModel) error {
	if !artifactSourceConfigured(data) {
		return nil
	}
	dirHash, err := computeArtifactSourceDirHash(data)
	if err != nil {
		data.Source.DirHash = types.StringNull()
		return err
	}
	data.Source.DirHash = dirHash
	return nil
}

// artifactSourceDirHashWarning reports a refreshArtifactSourceDirHash
// failure. The sync itself is done, so this is a warning: what it costs is
// the next plan re-syncing a directory that may not have changed.
func artifactSourceDirHashWarning(diags *diag.Diagnostics, err error) {
	if diags == nil {
		return
	}

	diags.AddAttributeWarning(
		path.Root("source").AtName("dir_hash"),
		"Could not fingerprint the source directory after apply",
		fmt.Sprintf("%s\n\nThe sync itself completed. dir_hash is left unset, so the next plan shows the directory as changed and apply syncs it again, which uploads nothing if the tree did not change.", err),
	)
}

func artifactSourceGenerateIgnore(data *ArtifactResourceModel) bool {
	if data == nil || data.Source == nil {
		return true
	}
	if data.Source.GenerateIgnore.IsNull() || data.Source.GenerateIgnore.IsUnknown() {
		return true
	}
	return data.Source.GenerateIgnore.ValueBool()
}

// artifactSourceIgnoreDiagnostics reports the two ignore-file conditions the
// matcher can detect but nothing else in an apply would mention: the deprecated
// .wapiignore name still being in effect, and a second ignore file whose
// patterns are silently inert. The first sign of the latter otherwise is a
// .venv on the remote.
//
// These are raised during planning so the user reads them before anything is
// uploaded, rather than after the catalog already has the file.
//
// A matcher that fails to load is not reported here, and which side covers that
// depends on the branch computeArtifactSourceDirHash takes. Everywhere but the
// default path -- generate_ignore on with no ignore file present -- compute
// calls New on this same directory and turns the failure into an attribute
// error, so repeating it would print the problem twice. On the default path
// compute hashes from the template and opens nothing, so plan stays silent by
// design. A directory sitting at the .drignore name, which Locate calls absent,
// lands there; apply names it in the warning seedArtifactSourceIgnoreFile
// raises when the write fails.
func artifactSourceIgnoreDiagnostics(data *ArtifactResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data == nil || data.Source == nil || !IsKnown(data.Source.Dir) {
		return diags
	}

	absDir, err := artifactSourceAbsDir(data)
	if err != nil {
		return diags
	}

	matcher, err := ignore.New(absDir)
	if err != nil {
		return diags
	}

	dirPath := path.Root("source").AtName("dir")
	if notice := matcher.Notice(); notice != "" {
		diags.AddAttributeWarning(dirPath, "Deprecated ignore file name", notice)
	}
	if shadow := matcher.ShadowWarning(); shadow != "" {
		diags.AddAttributeWarning(dirPath, "Ignore file present but not applied", shadow)
	}

	return diags
}

func computeArtifactSourceDirHash(data *ArtifactResourceModel) (types.String, error) {
	hash := types.StringNull()
	if data == nil || data.Source == nil || !IsKnown(data.Source.Dir) {
		return hash, nil
	}

	absDir, err := artifactSourceAbsDir(data)
	if err != nil {
		return hash, err
	}

	generateIgnore := artifactSourceGenerateIgnore(data)
	var matcher *ignore.Matcher
	var extra []artifactsource.LocalFile

	if generateIgnore && ignore.Locate(absDir) == "" {
		matcher = ignore.FromDefaultTemplate()
		sum := sha256.Sum256(ignore.DefaultTemplate)
		extra = []artifactsource.LocalFile{{
			RelPath: ignore.FileName,
			Hash:    hex.EncodeToString(sum[:]),
			Size:    int64(len(ignore.DefaultTemplate)),
		}}
	} else {
		matcher, err = ignore.New(absDir)
		if err != nil {
			return hash, err
		}
	}

	digest, err := artifactsource.FingerprintDirectory(absDir, matcher.Match, extra)
	if err != nil {
		return hash, err
	}

	return types.StringValue(digest), nil
}

func cloneCodeRefModel(ref *ArtifactCodeRefModel) *ArtifactCodeRefModel {
	if ref == nil {
		return nil
	}
	return &ArtifactCodeRefModel{
		CatalogID:        ref.CatalogID,
		CatalogVersionID: ref.CatalogVersionID,
	}
}

func primaryCodeRefFromState(state *ArtifactResourceModel) *ArtifactCodeRefModel {
	if state == nil || state.Spec == nil {
		return nil
	}
	for _, group := range state.Spec.ContainerGroups {
		for _, container := range group.Containers {
			if !artifactContainerIsPrimary(container, group) {
				continue
			}
			ref := imageBuildConfigCodeRef(container.ImageBuildConfig)
			if ref == nil {
				return nil
			}
			if IsKnown(ref.CatalogID) {
				return cloneCodeRefModel(ref)
			}
			return nil
		}
	}
	return nil
}

func applySourceManagedCodeRefsToPlan(plan, state *ArtifactResourceModel, isCreate bool) {
	if !artifactSourceConfigured(plan) || plan.Spec == nil || artifactHasManualCodeRef(plan.Spec) {
		return
	}

	needsUnknown := sourceManagedCodeRefNeedsUnknown(plan, state, isCreate)
	stateCodeRef := primaryCodeRefFromState(state)

	for gi := range plan.Spec.ContainerGroups {
		group := plan.Spec.ContainerGroups[gi]
		for ci := range group.Containers {
			container := &group.Containers[ci]
			if container.ImageBuildConfig == nil {
				continue
			}
			if !artifactContainerIsPrimary(*container, group) {
				continue
			}
			if codeRefManuallySet(imageBuildConfigCodeRef(container.ImageBuildConfig)) {
				continue
			}

			if needsUnknown {
				container.ImageBuildConfig.CodeRef = types.ObjectUnknown(artifactCodeRefAttrTypes())
				continue
			}

			if stateCodeRef != nil {
				_ = setImageBuildConfigCodeRef(container.ImageBuildConfig, cloneCodeRefModel(stateCodeRef))
			}
		}
	}
}

func sourceManagedCodeRefNeedsUnknown(plan, state *ArtifactResourceModel, isCreate bool) bool {
	if isCreate || state == nil {
		return true
	}

	priorArtifactID := state.ArtifactID.ValueString()
	newArtifactID := priorArtifactID
	if !plan.ArtifactID.IsNull() && !plan.ArtifactID.IsUnknown() {
		newArtifactID = plan.ArtifactID.ValueString()
	} else if plan.ArtifactID.IsUnknown() {
		return true
	}

	if artifactSourceNeedsUpload(plan, state, priorArtifactID, newArtifactID) {
		return true
	}

	if state.Status.ValueString() == string(client.ArtifactStatusLocked) {
		// locked→draft or a spec change always yields a new version, so code_ref is unknown.
		// Source-dir changes already returned above via artifactSourceNeedsUpload.
		return plan.Status.ValueString() == string(client.ArtifactStatusDraft) || artifactNeedsNewVersion(*plan, *state)
	}

	return false
}

func codeRefManuallySet(ref *ArtifactCodeRefModel) bool {
	if ref == nil {
		return false
	}
	return IsKnown(ref.CatalogID) || IsKnown(ref.CatalogVersionID)
}

func artifactContainerIsPrimary(container ArtifactContainerModel, group ArtifactContainerGroupModel) bool {
	isPrimary := !container.Primary.IsNull() && !container.Primary.IsUnknown() && container.Primary.ValueBool()
	if !isPrimary && len(group.Containers) == 1 &&
		(container.Primary.IsNull() || container.Primary.IsUnknown()) {
		isPrimary = true
	}
	return isPrimary
}
