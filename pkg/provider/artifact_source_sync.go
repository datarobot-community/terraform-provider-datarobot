package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
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
			if container.ImageBuildConfig == nil || container.ImageBuildConfig.CodeRef == nil {
				continue
			}
			if IsKnown(container.ImageBuildConfig.CodeRef.CatalogID) {
				return container.ImageBuildConfig.CodeRef.CatalogID.ValueString()
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
			if container.ImageBuildConfig == nil || container.ImageBuildConfig.CodeRef == nil {
				continue
			}
			if IsKnown(container.ImageBuildConfig.CodeRef.CatalogVersionID) {
				return container.ImageBuildConfig.CodeRef.CatalogVersionID.ValueString()
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

func (r *ArtifactResource) rollbackArtifactCreate(ctx context.Context, artifact *client.Artifact) {
	if artifact == nil || artifact.ArtifactRepositoryID == nil {
		return
	}
	traceAPICall("DeleteArtifactRepository")
	_ = r.provider.service.DeleteArtifactRepository(ctx, *artifact.ArtifactRepositoryID)
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
