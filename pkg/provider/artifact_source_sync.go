package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	artifactsync "github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/sync"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
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
// no .wapi/ yet. Terraform state is preferred over the artifact's live
// code_ref: a clone of a locked artifact is a fresh draft with no code_ref,
// and its new version still belongs in the catalog state points at.
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

// runArtifactSourceSync reconciles source.dir with the catalog using the
// CLI three-way sync engine: BASE (source.dir/.wapi/manifest.json) against
// LOCAL (the directory, minus .drignore and system excludes) against
// REMOTE (the Files API catalog). It returns the sync result and, when the
// engine repointed the artifact, the patched artifact.
//
// Unlike the push-only uploader it replaces, this can also write to
// source.dir — remote-only files are downloaded and a file edited on both
// sides is kept as <path>.LOCAL.<timestamp> while the remote version wins
// (terraform apply has no TTY to prompt on, so it always runs the CLI's
// --yes policy).
func (r *ArtifactResource) runArtifactSourceSync(
	ctx context.Context,
	data *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
) (result *artifactsync.Result, patched *client.Artifact, err error) {
	absDir, err := artifactSourceAbsDir(data)
	if err != nil {
		return nil, nil, err
	}

	if artifactSourceGenerateIgnore(data) {
		if _, err := ignore.WriteDefaultDrignoreIfMissing(absDir); err != nil {
			return nil, nil, err
		}
	}

	store := &artifactSourceStore{service: r.provider.service, current: artifact}

	engine, err := artifactsync.New(absDir, artifact.ID, r.provider.service.FilesAPI(), store)
	if err != nil {
		return nil, nil, err
	}
	engine.BindCatalog(artifactSourceCatalogBinding(state, artifact))

	// Close releases .wapi/sync.lock. A failure to release would make the
	// next apply fail on a lock nobody holds, so it is reported rather
	// than dropped, unless the sync itself already failed.
	defer func() {
		if closeErr := engine.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("release sync lock: %w", closeErr)
		}
	}()

	traceAPICall("SyncArtifactSource")
	result, err = engine.Run(ctx)
	if err != nil {
		return nil, nil, err
	}

	return result, store.patched, nil
}

// syncArtifactSource reconciles source.dir when the planned tree differs
// from state, and reports whether the sync ran (which is what gates the
// follow-up image build).
func (r *ArtifactResource) syncArtifactSource(
	ctx context.Context,
	plan *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	priorArtifactID string,
) (*client.Artifact, bool, error) {
	if !artifactSourceNeedsUpload(plan, state, priorArtifactID, artifact.ID) {
		return artifact, false, nil
	}

	artifactApplyProgressUploading(artifact.ID)

	result, patched, err := r.runArtifactSourceSync(ctx, plan, state, artifact)
	if err != nil {
		return nil, false, fmt.Errorf("sync artifact source: %w", err)
	}

	artifactApplyProgressSourceSynced(result)

	if patched != nil {
		artifact = patched
	}

	return artifact, true, nil
}

// artifactApplyProgressSourceSynced reports what the sync did to
// source.dir. Downloads, local deletions and *.LOCAL.* copies all change
// files the user owns, so they are announced during apply instead of left
// for the user to find in a later `git status`.
func artifactApplyProgressSourceSynced(result *artifactsync.Result) {
	if result == nil {
		return
	}

	if result.Uploaded+result.Downloaded+result.Deleted+result.Conflicts == 0 {
		emitArtifactApplyProgress("Source already matches the catalog; nothing to sync.")
		return
	}

	emitArtifactApplyProgress(fmt.Sprintf(
		"Synced source: %d uploaded, %d downloaded, %d deleted, %d conflicted.",
		result.Uploaded, result.Downloaded, result.Deleted, result.Conflicts,
	))

	for _, path := range result.ConflictCopies {
		emitArtifactApplyProgress(fmt.Sprintf(
			"Conflict on both sides: the catalog version won, your local file was kept as %s", path))
	}
}

// syncArtifactSourceAndBuild syncs source when needed, then triggers an image build
// when the sync ran against a draft artifact with image_build_config.
func (r *ArtifactResource) syncArtifactSourceAndBuild(
	ctx context.Context,
	plan *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	priorArtifactID string,
) (*client.Artifact, error) {
	artifact, uploaded, err := r.syncArtifactSource(ctx, plan, state, artifact, priorArtifactID)
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

func (r *ArtifactResource) rollbackArtifactCreate(ctx context.Context, artifact *client.Artifact) {
	if artifact == nil || artifact.ArtifactRepositoryID == nil {
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

// artifactLockedSourceCloneNeeded is true when a locked artifact needs a new code upload.
// Locked artifacts are immutable; the provider clones to draft, uploads, patches code_ref,
// then locks the new version (mirrors CLI guidance in cli/internal/workload/sync/phase1_gather.go).
func artifactLockedSourceCloneNeeded(plan, state ArtifactResourceModel) bool {
	if state.Status.ValueString() != string(client.ArtifactStatusLocked) {
		return false
	}
	return artifactSourcePendingUpload(&plan, &state, state.ArtifactID.ValueString())
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

func refreshArtifactSourceDirHash(data *ArtifactResourceModel) {
	if !artifactSourceConfigured(data) {
		return
	}
	dirHash, err := computeArtifactSourceDirHash(data)
	if err == nil {
		data.Source.DirHash = dirHash
	}
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

// artifactSourceAbsDir resolves source.dir against the process working
// directory, which is the module directory during apply.
func artifactSourceAbsDir(data *ArtifactResourceModel) (string, error) {
	dir := data.Source.Dir.ValueString()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve source directory %q: %w", dir, err)
	}

	return absDir, nil
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

	if generateIgnore && !ignore.UserIgnoreExists(absDir) {
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

func applySourceManagedCodeRefsToPlan(plan, state *ArtifactResourceModel, isCreate bool) {
	if !artifactSourceConfigured(plan) || plan.Spec == nil || artifactHasManualCodeRef(plan.Spec) {
		return
	}

	needsUnknown := sourceManagedCodeRefNeedsUnknown(plan, state, isCreate)

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

			// If the state has a known catalog ID and doesn't needUnknown, copy it into new plan
			if state != nil &&
				gi < len(state.Spec.ContainerGroups) &&
				ci < len(state.Spec.ContainerGroups[gi].Containers) {
				stateContainer := state.Spec.ContainerGroups[gi].Containers[ci]
				if stateContainer.ImageBuildConfig != nil &&
					!stateContainer.ImageBuildConfig.CodeRef.IsNull() &&
					!stateContainer.ImageBuildConfig.CodeRef.IsUnknown() {
					container.ImageBuildConfig.CodeRef = stateContainer.ImageBuildConfig.CodeRef
				}
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
		if plan.Status.ValueString() == string(client.ArtifactStatusDraft) {
			return true
		}
		if artifactSourceConfigured(plan) {
			return artifactLockedSourceCloneNeeded(*plan, *state)
		}
		if artifactNeedsNewVersion(*plan, *state) {
			return true
		}
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
