package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
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

	traceAPICall("PushDirectory")
	return artifactsource.PushDirectory(ctx, r.provider.service.FilesAPI(), opts)
}

func (r *ArtifactResource) syncArtifactSource(
	ctx context.Context,
	plan *ArtifactResourceModel,
	state *ArtifactResourceModel,
	artifact *client.Artifact,
	priorArtifactID string,
) (*client.Artifact, error) {
	if !artifactSourceNeedsUpload(plan, state, priorArtifactID, artifact.ID) {
		return artifact, nil
	}

	catalogID := catalogIDFromModel(state)
	if catalogID == "" {
		if ref := client.ExtractCodeRef(artifact); ref != nil {
			catalogID = ref.CatalogID
		}
	}

	pushResult, err := r.pushArtifactSource(ctx, plan, state, catalogID)
	if err != nil {
		return nil, fmt.Errorf("upload artifact source: %w", err)
	}

	traceAPICall("PatchArtifactCodeRef")
	artifact, err = r.provider.service.PatchArtifactCodeRef(
		ctx,
		artifact.ID,
		pushResult.CatalogID,
		pushResult.CatalogVersionID,
	)
	if err != nil {
		return nil, fmt.Errorf("patch artifact code reference: %w", err)
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
	dirHash, err := computeFolderHash(data.Source.Dir)
	if err == nil {
		data.Source.DirHash = dirHash
	}
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
		if plan.Status.ValueString() == string(client.ArtifactStatusDraft) || artifactNeedsNewVersion(*plan, *state) {
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
