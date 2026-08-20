package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
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

func (r *ArtifactResource) pushArtifactSource(
	ctx context.Context,
	data *ArtifactResourceModel,
	prior *ArtifactResourceModel,
	existingCatalogID string,
) (*artifactsource.Result, error) {
	dir := data.Source.Dir.ValueString()
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve source directory %q: %w", dir, err)
	}

	opts := artifactsource.Options{
		Dir:       absDir,
		CatalogID: existingCatalogID,
	}
	if prior != nil {
		opts.CatalogVersionID = catalogVersionIDFromModel(prior)
	}

	if artifactSourceGenerateIgnore(data) {
		if _, err := ignore.WriteDefaultDrignoreIfMissing(absDir); err != nil {
			return nil, fmt.Errorf("write %s: %w", ignore.FileName, err)
		}
	}

	matcher, err := ignore.New(absDir)
	if err != nil {
		return nil, fmt.Errorf("load ignore rules: %w", err)
	}
	opts.Ignore = matcher.Match

	traceAPICall("PushDirectory")
	return artifactsource.PushDirectory(ctx, r.provider.service.FilesAPI(), opts)
}

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

	catalogID := catalogIDFromModel(state)
	if catalogID == "" {
		if ref := client.ExtractCodeRef(artifact); ref != nil {
			catalogID = ref.CatalogID
		}
	}

	pushResult, err := r.pushArtifactSource(ctx, plan, state, catalogID)
	if err != nil {
		return nil, false, fmt.Errorf("upload artifact source: %w", err)
	}

	traceAPICall("PatchArtifactCodeRef")
	artifact, err = r.provider.service.PatchArtifactCodeRef(
		ctx,
		artifact.ID,
		pushResult.CatalogID,
		pushResult.CatalogVersionID,
	)
	if err != nil {
		return nil, true, fmt.Errorf("patch artifact code reference: %w", err)
	}

	return artifact, true, nil
}

// syncArtifactSourceAndBuild uploads source when needed, then triggers an image build
// when upload produced new code on a draft artifact with image_build_config.
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

func computeArtifactSourceDirHash(data *ArtifactResourceModel) (types.String, error) {
	hash := types.StringNull()
	if data == nil || data.Source == nil || !IsKnown(data.Source.Dir) {
		return hash, nil
	}

	absDir, err := filepath.Abs(data.Source.Dir.ValueString())
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
