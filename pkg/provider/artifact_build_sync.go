package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
)

// artifactBuildNeededAfterUpload reports whether a source upload should be followed
// by an image build trigger. Builds may only run on draft artifacts (WAPI constraint).
func artifactBuildNeededAfterUpload(plan *ArtifactResourceModel, artifact *client.Artifact, uploaded bool) bool {
	if !uploaded || plan == nil || artifact == nil {
		return false
	}
	if artifact.Status != client.ArtifactStatusDraft {
		return false
	}
	if !artifactHasPrimaryImageBuildConfig(plan.Spec) {
		return false
	}
	return artifactSourceConfigured(plan)
}

// artifactSourceWaitForBuild returns whether the provider should poll until the build
// reaches a terminal success status. Defaults to true when unset in config.
func artifactSourceWaitForBuild(plan *ArtifactResourceModel) bool {
	if plan == nil || plan.Source == nil {
		return true
	}
	if plan.Source.WaitForBuild.IsNull() || plan.Source.WaitForBuild.IsUnknown() {
		return true
	}
	return plan.Source.WaitForBuild.ValueBool()
}

// syncArtifactBuild triggers an image build for the artifact. When waitForBuild is true,
// polls until completion and refreshes the artifact so image_uri is populated on the
// primary container. Ported from cli/internal/workload/build.go (TriggerArtifactBuild,
// WaitForBuild, BuildSummaryFor image URI fetch).
func (r *ArtifactResource) syncArtifactBuild(
	ctx context.Context,
	artifactID string,
	waitForBuild bool,
	opts *client.WaitForArtifactBuildOptions,
) (*client.Artifact, string, error) {
	traceAPICall("TriggerArtifactBuild")
	trigger, err := r.provider.service.TriggerArtifactBuild(ctx, artifactID)
	if err != nil {
		return nil, "", fmt.Errorf("trigger artifact build: %w", err)
	}
	if trigger == nil || len(trigger.BuildIDs) == 0 {
		return nil, "", fmt.Errorf("trigger artifact build: empty buildIds")
	}

	buildID := trigger.BuildIDs[0]

	if waitForBuild {
		traceAPICall("WaitForArtifactBuild")
		if _, err := r.provider.service.WaitForArtifactBuild(ctx, artifactID, buildID, opts); err != nil {
			return nil, buildID, fmt.Errorf("wait for artifact build: %w", err)
		}
	}

	traceAPICall("GetArtifact")
	artifact, err := r.provider.service.GetArtifact(ctx, artifactID)
	if err != nil {
		msg := "refresh artifact after build"
		if !waitForBuild {
			msg = "refresh artifact after build trigger"
		}
		return nil, buildID, fmt.Errorf("%s: %w", msg, err)
	}
	if waitForBuild && artifactPrimaryContainerImageURI(artifact) == "" {
		return artifact, buildID, fmt.Errorf("artifact build completed but primary container image_uri is empty")
	}
	return artifact, buildID, nil
}

func artifactPrimaryContainerImageURI(artifact *client.Artifact) string {
	if artifact == nil {
		return ""
	}

	for _, group := range artifact.Spec.ContainerGroups {
		for _, container := range group.Containers {
			if container.Primary != nil && *container.Primary {
				return container.ImageURI
			}
		}
	}

	if len(artifact.Spec.ContainerGroups) == 0 {
		return ""
	}
	if len(artifact.Spec.ContainerGroups[0].Containers) == 0 {
		return ""
	}

	return artifact.Spec.ContainerGroups[0].Containers[0].ImageURI
}
