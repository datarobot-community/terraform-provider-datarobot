package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type artifactBuildSyncError struct {
	cause error
}

func (e *artifactBuildSyncError) Error() string {
	return e.cause.Error()
}

func (e *artifactBuildSyncError) Unwrap() error {
	return e.cause
}

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
	artifactRepositoryID string,
	waitForBuild bool,
	opts *client.WaitForArtifactBuildOptions,
) (*client.Artifact, string, error) {
	traceAPICall("TriggerArtifactBuild")
	trigger, err := r.provider.service.TriggerArtifactBuild(ctx, artifactID)
	if err != nil {
		return nil, "", r.enrichArtifactBuildError(ctx, artifactID, artifactRepositoryID, "", fmt.Errorf("trigger artifact build: %w", err))
	}
	if trigger == nil || len(trigger.BuildIDs) == 0 {
		return nil, "", r.enrichArtifactBuildError(
			ctx,
			artifactID,
			artifactRepositoryID,
			"",
			fmt.Errorf("trigger artifact build: empty buildIds"),
		)
	}

	buildID := trigger.BuildIDs[0]

	if waitForBuild {
		traceAPICall("WaitForArtifactBuild")
		waitOpts := artifactBuildWaitOptions(opts)
		if _, err := r.provider.service.WaitForArtifactBuild(ctx, artifactID, buildID, waitOpts); err != nil {
			return nil, buildID, r.enrichArtifactBuildError(
				ctx,
				artifactID,
				artifactRepositoryID,
				buildID,
				fmt.Errorf("wait for artifact build: %w", err),
			)
		}
	}

	traceAPICall("GetArtifact")
	artifact, err := r.provider.service.GetArtifact(ctx, artifactID)
	if err != nil {
		msg := "refresh artifact after build"
		if !waitForBuild {
			msg = "refresh artifact after build trigger"
		}
		return nil, buildID, r.enrichArtifactBuildError(
			ctx,
			artifactID,
			artifactRepositoryID,
			buildID,
			fmt.Errorf("%s: %w", msg, err),
		)
	}
	if waitForBuild && artifactPrimaryContainerImageURI(artifact) == "" {
		return artifact, buildID, r.enrichArtifactBuildError(
			ctx,
			artifactID,
			artifactRepositoryID,
			buildID,
			fmt.Errorf("artifact build completed but primary container image_uri is empty"),
		)
	}
	return artifact, buildID, nil
}

func artifactBuildWaitOptions(opts *client.WaitForArtifactBuildOptions) *client.WaitForArtifactBuildOptions {
	merged := &client.WaitForArtifactBuildOptions{}
	if opts != nil {
		*merged = *opts
	}
	if merged.OnOtelLogLine == nil {
		merged.OnOtelLogLine = func(entry client.OtelLogEntry) {
			line := client.FormatOtelLogEntry(entry)
			emitArtifactBuildLogLine(line)
		}
	}
	return merged
}

// artifactModifyPlanNeedsUnknownImageURI is true when apply will upload source and trigger
// a build that populates the primary container image_uri.
func artifactModifyPlanNeedsUnknownImageURI(plan *ArtifactResourceModel, state *ArtifactResourceModel, isCreate bool) bool {
	if !artifactSourceConfigured(plan) || !artifactHasPrimaryImageBuildConfig(plan.Spec) {
		return false
	}

	priorArtifactID := ""
	if state != nil && IsKnown(state.ArtifactID) {
		priorArtifactID = state.ArtifactID.ValueString()
	}

	newArtifactID := priorArtifactID
	if state != nil && artifactModifyPlanNeedsUnknownArtifactID(*plan, *state) {
		// Assign "new-version" to signal that a new artifact version will be created,
		// so that artifactSourceNeedsUpload logic treats this as a new ID not matching any prior version.
		newArtifactID = "new-version"
	}

	if !artifactSourceNeedsUpload(plan, state, priorArtifactID, newArtifactID) {
		return false
	}

	if isCreate {
		return true
	}

	if state == nil {
		return false
	}

	if state.Status.ValueString() == string(client.ArtifactStatusLocked) {
		if plan.Status.ValueString() == string(client.ArtifactStatusDraft) {
			return true
		}
		return artifactLockedSourceCloneNeeded(*plan, *state)
	}

	if plan.Status.ValueString() == string(client.ArtifactStatusLocked) {
		return artifactSourceDeferLock(*plan, *state)
	}

	return state.Status.ValueString() == string(client.ArtifactStatusDraft)
}

func applySourceManagedImageURIToPlan(config, plan, state *ArtifactResourceModel, isCreate bool) {
	if !artifactModifyPlanNeedsUnknownImageURI(plan, state, isCreate) || plan.Spec == nil {
		return
	}
	if config != nil && artifactHasManualImageURI(config.Spec) {
		return
	}

	for gi := range plan.Spec.ContainerGroups {
		group := plan.Spec.ContainerGroups[gi]
		for ci := range group.Containers {
			container := &group.Containers[ci]
			if !artifactContainerIsPrimary(*container, group) {
				continue
			}
			if config != nil && containerImageURIManuallySet(config, gi, ci) {
				continue
			}
			container.ImageURI = types.StringUnknown()
		}
	}
}

func artifactHasManualImageURI(spec *ArtifactSpecModel) bool {
	if spec == nil {
		return false
	}
	for _, group := range spec.ContainerGroups {
		for _, container := range group.Containers {
			if !container.ImageURI.IsNull() && !container.ImageURI.IsUnknown() && container.ImageURI.ValueString() != "" {
				return true
			}
		}
	}
	return false
}

func containerImageURIManuallySet(model *ArtifactResourceModel, gi, ci int) bool {
	if model == nil || model.Spec == nil {
		return false
	}
	if gi >= len(model.Spec.ContainerGroups) {
		return false
	}
	group := model.Spec.ContainerGroups[gi]
	if ci >= len(group.Containers) {
		return false
	}
	uri := group.Containers[ci].ImageURI
	return !uri.IsNull() && !uri.IsUnknown() && uri.ValueString() != ""
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
