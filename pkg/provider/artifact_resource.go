package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ArtifactResource{}
var _ resource.ResourceWithImportState = &ArtifactResource{}
var _ resource.ResourceWithModifyPlan = &ArtifactResource{}
var _ resource.ResourceWithValidateConfig = &ArtifactResource{}

func NewArtifactResource() resource.Resource {
	return &ArtifactResource{}
}

type ArtifactResource struct {
	provider *Provider
}

func (r *ArtifactResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_artifact"
}

func (r *ArtifactResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	probeAttributes := artifactResourceProbeAttributes()

	dockerfileAttributes := map[string]schema.Attribute{
		"source": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("provided"),
			MarkdownDescription: "How the Dockerfile is obtained: `provided` (from source code) or `generated` (from an execution environment). Defaults to `provided`.",
			Validators:          ArtifactDockerfileSourceValidators(),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"path": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Relative path to the Dockerfile in the source code. Used when source is `provided`. Defaults to `./Dockerfile`. Null when source is `generated`.",
			PlanModifiers: []planmodifier.String{
				artifactDockerfilePathPlanModifier{},
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"execution_environment_id": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Execution environment ID for the base Docker image. Required when source is `generated`.",
		},
		"execution_environment_version_id": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Execution environment version ID that pins the base image. Required when source is `generated`.",
		},
		"entrypoint": schema.ListAttribute{
			Optional:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Entrypoint baked into the generated Dockerfile CMD. Required when source is `generated`.",
		},
	}

	codeRefAttrTypes := map[string]attr.Type{
		"catalog_id":         types.StringType,
		"catalog_version_id": types.StringType,
	}

	imageBuildConfigAttributes := map[string]schema.Attribute{
		"code_ref": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			Default:             objectdefault.StaticValue(types.ObjectNull(codeRefAttrTypes)),
			MarkdownDescription: "Reference to source code in the DataRobot catalog. Optional at create; required before image build or lock. When `source` is set, the provider uploads `source.dir` and populates this block.",
			Attributes: map[string]schema.Attribute{
				"catalog_id": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Files API catalog ID (24-character hex).",
				},
				"catalog_version_id": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Files API catalog version ID (24-character hex).",
				},
			},
		},
		"dockerfile": schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: "How the Dockerfile is obtained for the image build. Defaults to using `./Dockerfile` from the source code.",
			Attributes:          dockerfileAttributes,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Artifact definition for the Workload API. Artifacts define container images and runtime configuration for workloads.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable provider-generated identifier for this artifact resource. Does not change across artifact version updates.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"artifact_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The current artifact ID. Updated on every create or update that produces a new artifact version. Reference this field from dependent resources such as Workload.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Artifact.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the Artifact.",
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("service"),
				MarkdownDescription: "The artifact type: `service`, `nim`, or `agent`. Defaults to `service`.",
				Validators:          ArtifactTypeValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Artifact lifecycle status: `draft` (the current artifact version is mutable; " +
					"spec changes are applied in-place and `artifact_id` stays the same) or `locked` (artifact versions are immutable; " +
					"spec changes create a new version with a new `artifact_id` in the same `artifact_repository_id`). " +
					"Defaults to `locked`. Locking a draft artifact is one-way. Changing `status` from `locked` to `draft` " +
					"creates a new draft artifact (the Workload API cannot unlock in place).",
				Default:    stringdefault.StaticString(string(client.ArtifactStatusLocked)),
				Validators: ArtifactStatusValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"artifact_repository_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the artifact repository for versioning. Computed on first create if not provided; subsequent updates create new versions in the same repository.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"spec": artifactResourceSpecAttribute(probeAttributes, imageBuildConfigAttributes),
			"source": schema.SingleNestedAttribute{
				Optional: true,
				MarkdownDescription: "Local source directory to upload to the DataRobot catalog and attach to the primary container's `image_build_config.code_ref`. " +
					"When source content changes, the provider uploads, triggers an image build on the draft artifact, and (by default) waits for completion before proceeding. " +
					"On draft artifacts, uploads are applied in-place. On locked artifacts, source changes clone to a new draft version, upload, build, patch `code_ref`, and lock the new version.",
				Attributes: map[string]schema.Attribute{
					"dir": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Path to the local directory containing application source files to upload.",
					},
					"dir_hash": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "SHA-256 fingerprint of `dir` contents, used to detect changes and skip re-upload when unchanged.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"wait_for_build": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "When `true` (default), after a source upload the provider triggers an image build and polls until it completes before proceeding (for example, before locking). When `false`, the build is triggered but apply does not wait for `image_uri` to be populated.",
						Default:             booldefault.StaticBool(true),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *ArtifactResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please report this issue to the provider developers.", Provider{}, req.ProviderData),
		)
	}
}

func (r *ArtifactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ArtifactResourceModel

	resp.Diagnostics.Append(decodePlanArtifactModel(ctx, req.Plan, nil, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := artifactCreateRequest(data)
	targetLocked := createReq.Status == client.ArtifactStatusLocked
	if artifactSourceConfigured(&data) && targetLocked {
		createReq.Status = client.ArtifactStatusDraft
	}

	traceAPICall("CreateArtifact")
	artifact, err := r.provider.service.CreateArtifact(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Artifact", err.Error())
		return
	}

	userSuppliedRepository := IsKnown(data.ArtifactRepositoryID)
	createdArtifact := artifact
	if artifactSourceConfigured(&data) {
		syncedArtifact, syncErr := r.syncArtifactSourceAndBuild(ctx, &data, nil, createdArtifact, "")
		if syncErr != nil {
			var timeoutErr *client.ArtifactBuildTimeoutError
			isTimeout := errors.As(syncErr, &timeoutErr)
			if !isTimeout {
				r.rollbackArtifactCreate(ctx, createdArtifact, !userSuppliedRepository)
			}
			summary := "Error uploading artifact source"
			var buildErr *artifactBuildSyncError
			if errors.As(syncErr, &buildErr) {
				if isTimeout {
					summary = "Timeout waiting for artifact image build"
				} else {
					summary = "Error building artifact image"
				}
			}
			resp.Diagnostics.AddError(summary, syncErr.Error())
			return
		}
		artifact = syncedArtifact
	}

	if targetLocked && artifactSourceConfigured(&data) {
		preLockArtifact := artifact
		// The lock response can omit or lag container.build, which would discard the
		// terminal build metadata syncArtifactBuild just pinned from WaitForArtifactBuild.
		pinnedBuild := primaryContainerBuildInfo(preLockArtifact)
		lockedArtifact, lockErr := r.lockArtifact(ctx, preLockArtifact.ID)
		if lockErr != nil {
			r.rollbackArtifactCreate(ctx, preLockArtifact, !userSuppliedRepository)
			resp.Diagnostics.AddError("Error locking Artifact after source upload", lockErr.Error())
			return
		}
		artifact = lockedArtifact
		applyBuildInfoToPrimaryContainer(artifact, pinnedBuild)
	}

	data.ID = types.StringValue(uuid.NewString())
	loadArtifactIntoModel(artifact, &data)
	refreshArtifactSourceDirHash(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ArtifactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ArtifactResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ArtifactID.IsNull() || data.ArtifactID.IsUnknown() {
		return
	}

	traceAPICall("GetArtifact")
	artifact, err := r.provider.service.GetArtifact(ctx, data.ArtifactID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Artifact not found",
				fmt.Sprintf("Artifact with ID %s is not found. Removing from state.", data.ArtifactID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Artifact with ID %s", data.ArtifactID.ValueString()),
				err.Error())
		}
		return
	}

	loadArtifactIntoModel(artifact, &data)
	refreshArtifactSourceDirHash(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ArtifactResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(decodePlanArtifactModel(ctx, req.Plan, &state, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// artifact_repository_id is Optional+Computed: when the user doesn't set it in config,
	// the plan value is null (UseStateForUnknown only applies to unknown, not null).
	// Preserve the computed value from state so subsequent versions are created in the same repo.
	if plan.ArtifactRepositoryID.IsNull() && !state.ArtifactRepositoryID.IsNull() {
		plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	}

	priorArtifactID := state.ArtifactID.ValueString()
	lockedSourceCloneNeeded := artifactLockedSourceCloneNeeded(plan, state)
	deferLock := artifactSourceDeferLock(plan, state)

	var artifact *client.Artifact
	var err error

	switch {
	case state.Status.ValueString() == string(client.ArtifactStatusDraft):
		traceAPICall("PatchArtifact")
		artifact, err = r.provider.service.PatchArtifact(
			ctx,
			priorArtifactID,
			patchRequestFromPlan(plan, state, deferLock),
		)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Artifact", err.Error())
			return
		}
	case lockedSourceCloneNeeded:
		createReq := artifactCreateRequest(plan)
		createReq.Status = client.ArtifactStatusDraft
		traceAPICall("CreateUpdatedArtifact")
		artifact, err = r.provider.service.CreateArtifact(ctx, createReq)
		if err != nil {
			resp.Diagnostics.AddError("Error creating draft Artifact for source update", err.Error())
			return
		}
	default:
		traceAPICall("CreateUpdatedArtifact")
		artifact, err = r.provider.service.CreateArtifact(ctx, artifactCreateRequest(plan))
		if err != nil {
			resp.Diagnostics.AddError("Error creating new Artifact version", err.Error())
			return
		}
	}

	createdNewVersion := state.Status.ValueString() != string(client.ArtifactStatusDraft)
	if artifactSourceConfigured(&plan) {
		syncedArtifact, syncErr := r.syncArtifactSourceAndBuild(ctx, &plan, &state, artifact, priorArtifactID)
		if syncErr != nil {
			if createdNewVersion {
				persistPartialArtifactUpdate(ctx, resp, artifact, &plan, &state)
			}
			summary := "Error uploading artifact source"
			var buildErr *artifactBuildSyncError
			if errors.As(syncErr, &buildErr) {
				var timeoutErr *client.ArtifactBuildTimeoutError
				if errors.As(syncErr, &timeoutErr) {
					summary = "Timeout waiting for artifact image build"
				} else {
					summary = "Error building artifact image"
				}
			}
			resp.Diagnostics.AddError(summary, syncErr.Error())
			return
		}
		artifact = syncedArtifact
	}

	if plan.Status.ValueString() == string(client.ArtifactStatusLocked) &&
		artifact.Status != client.ArtifactStatusLocked &&
		(deferLock || lockedSourceCloneNeeded) {
		preLockArtifact := artifact
		// See the Create path: locking must not drop the pinned build metadata.
		pinnedBuild := primaryContainerBuildInfo(preLockArtifact)
		lockedArtifact, lockErr := r.lockArtifact(ctx, preLockArtifact.ID)
		if lockErr != nil {
			if createdNewVersion || lockedSourceCloneNeeded {
				persistPartialArtifactUpdate(ctx, resp, preLockArtifact, &plan, &state)
			}
			resp.Diagnostics.AddError("Error locking Artifact after source upload", lockErr.Error())
			return
		}
		artifact = lockedArtifact
		applyBuildInfoToPrimaryContainer(artifact, pinnedBuild)
	}

	loadArtifactIntoModel(artifact, &plan)
	refreshArtifactSourceDirHash(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// persistPartialArtifactUpdate records a newly created draft version in state when a later
// step (source upload or lock) fails. The prior source.dir_hash is kept so a retry still
// sees a pending upload instead of treating the new tree as already synced.
func persistPartialArtifactUpdate(
	ctx context.Context,
	resp *resource.UpdateResponse,
	artifact *client.Artifact,
	plan, state *ArtifactResourceModel,
) {
	loadArtifactIntoModel(artifact, plan)
	if plan.Source != nil && state.Source != nil {
		plan.Source.DirHash = state.Source.DirHash
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ArtifactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ArtifactResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ArtifactRepositoryID.IsNull() || data.ArtifactRepositoryID.IsUnknown() {
		return
	}

	traceAPICall("DeleteArtifactRepository")
	if err := r.provider.service.DeleteArtifactRepository(ctx, data.ArtifactRepositoryID.ValueString()); err != nil {
		if _, ok := err.(*client.NotFoundError); !ok {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error deleting Artifact Repository with ID %s", data.ArtifactRepositoryID.ValueString()),
				err.Error(),
			)
		}
	}
}

func (r *ArtifactResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ArtifactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config ArtifactResourceModel
	var configPtr *ArtifactResourceModel
	if !req.Config.Raw.IsNull() {
		resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
		if resp.Diagnostics.HasError() {
			return
		}
		configPtr = &config
	}

	if plan.Source != nil && IsKnown(plan.Source.Dir) {
		dirHash, err := computeFolderHash(plan.Source.Dir)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("source").AtName("dir"),
				"Error calculating source directory hash",
				err.Error(),
			)
			return
		}
		plan.Source.DirHash = dirHash
	}

	var statePtr *ArtifactResourceModel
	var state ArtifactResourceModel
	isCreate := req.State.Raw.IsNull()
	if !isCreate {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		statePtr = &state

		if plan.ArtifactRepositoryID.IsNull() && !state.ArtifactRepositoryID.IsNull() {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("artifact_repository_id"), state.ArtifactRepositoryID)...)
			plan.ArtifactRepositoryID = state.ArtifactRepositoryID
		}

		if artifactModifyPlanNeedsUnknownArtifactID(plan, state) {
			plan.ArtifactID = types.StringUnknown()
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("artifact_id"), types.StringUnknown())...)
		}
	}

	applySourceManagedCodeRefsToPlan(&plan, statePtr, isCreate)
	applySourceManagedImageURIToPlan(configPtr, &plan, statePtr, isCreate)
	applySourceManagedBuildToPlan(&plan, statePtr, isCreate)

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func decodePlanArtifactModel(ctx context.Context, plan tfsdk.Plan, state *ArtifactResourceModel, data *ArtifactResourceModel) diag.Diagnostics {
	var modifyResp resource.ModifyPlanResponse
	modifyResp.Plan = plan

	var diags diag.Diagnostics
	diags.Append(nullUnknownCodeRefsForDecode(ctx, modifyResp.Plan, &modifyResp, state)...)
	if diags.HasError() {
		return diags
	}
	diags.Append(modifyResp.Plan.Get(ctx, data)...)
	return diags
}

func nullUnknownCodeRefsForDecode(ctx context.Context, plan tfsdk.Plan, resp *resource.ModifyPlanResponse, state *ArtifactResourceModel) diag.Diagnostics {
	return walkPlanCodeRefPaths(ctx, plan, func(codeRefPath path.Path, gi, ci int, isPrimary bool) diag.Diagnostics {
		var codeRef types.Object
		var diags diag.Diagnostics
		diags.Append(plan.GetAttribute(ctx, codeRefPath, &codeRef)...)
		if diags.HasError() || !codeRef.IsUnknown() {
			return diags
		}

		if isPrimary {
			if ref := primaryCodeRefFromState(state); ref != nil {
				codeRefValue, valueDiags := types.ObjectValue(artifactCodeRefObjectType.AttrTypes, map[string]attr.Value{
					"catalog_id":         ref.CatalogID,
					"catalog_version_id": ref.CatalogVersionID,
				})
				diags.Append(valueDiags...)
				if diags.HasError() {
					return diags
				}
				diags.Append(resp.Plan.SetAttribute(ctx, codeRefPath, codeRefValue)...)
				return diags
			}
		}

		diags.Append(resp.Plan.SetAttribute(ctx, codeRefPath, types.ObjectNull(artifactCodeRefObjectType.AttrTypes))...)
		return diags
	})
}

func planObjectContainerIsPrimary(container types.Object, containerCount int) bool {
	if container.IsNull() || container.IsUnknown() {
		return false
	}
	attrs := container.Attributes()
	primaryAttr, ok := attrs["primary"]
	if !ok {
		return containerCount == 1
	}
	primary, ok := primaryAttr.(types.Bool)
	if !ok || primary.IsNull() || primary.IsUnknown() {
		return containerCount == 1
	}
	return primary.ValueBool()
}

func walkPlanCodeRefPaths(ctx context.Context, plan tfsdk.Plan, fn func(path.Path, int, int, bool) diag.Diagnostics) diag.Diagnostics {
	var spec types.Object
	diags := plan.GetAttribute(ctx, path.Root("spec"), &spec)
	if diags.HasError() || spec.IsNull() || spec.IsUnknown() {
		return diags
	}

	specAttrs := spec.Attributes()
	groupsAttr, ok := specAttrs["container_groups"]
	if !ok {
		return diags
	}
	groups, ok := groupsAttr.(types.List)
	if !ok || groups.IsNull() || groups.IsUnknown() {
		return diags
	}

	for gi, groupElem := range groups.Elements() {
		group, ok := groupElem.(types.Object)
		if !ok || group.IsNull() || group.IsUnknown() {
			continue
		}
		groupAttrs := group.Attributes()
		containersAttr, ok := groupAttrs["containers"]
		if !ok {
			continue
		}
		containers, ok := containersAttr.(types.List)
		if !ok || containers.IsNull() || containers.IsUnknown() {
			continue
		}

		for ci, containerElem := range containers.Elements() {
			container, ok := containerElem.(types.Object)
			if !ok || container.IsNull() || container.IsUnknown() {
				continue
			}
			containerAttrs := container.Attributes()
			buildConfigAttr, ok := containerAttrs["image_build_config"]
			if !ok {
				continue
			}
			buildConfig, ok := buildConfigAttr.(types.Object)
			if !ok || buildConfig.IsNull() || buildConfig.IsUnknown() {
				continue
			}

			codeRefPath := path.Root("spec").
				AtName("container_groups").AtListIndex(gi).
				AtName("containers").AtListIndex(ci).
				AtName("image_build_config").AtName("code_ref")
			diags.Append(fn(codeRefPath, gi, ci, planObjectContainerIsPrimary(container, len(containers.Elements())))...)
			if diags.HasError() {
				return diags
			}
		}
	}

	return diags
}

var artifactCodeRefObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"catalog_id":         types.StringType,
		"catalog_version_id": types.StringType,
	},
}

func artifactNeedsNewVersion(plan, state ArtifactResourceModel) bool {
	if !plan.Name.Equal(state.Name) {
		return true
	}
	if !plan.Description.Equal(state.Description) {
		return true
	}
	if !plan.ArtifactRepositoryID.Equal(state.ArtifactRepositoryID) {
		return true
	}
	if plan.Spec == nil || state.Spec == nil {
		return plan.Spec != state.Spec
	}
	if len(plan.Spec.ContainerGroups) != len(state.Spec.ContainerGroups) {
		return true
	}
	// When source manages code_ref / image_uri, plan values are null or pre-build while
	// state holds the last upload/build results — ignore those managed diffs on the
	// provider-managed primary container only; sidecar changes always count.
	ignoreManagedCodeRef := artifactSourceConfigured(&plan) && !artifactHasManualCodeRef(plan.Spec)
	ignoreManagedImageURI := artifactSourceConfigured(&plan)
	for i := range plan.Spec.ContainerGroups {
		if !containerGroupsEqual(plan.Spec.ContainerGroups[i], state.Spec.ContainerGroups[i], ignoreManagedCodeRef, ignoreManagedImageURI) {
			return true
		}
	}
	return a2aEnabledNeedsNewVersion(plan.Spec.A2AEnabled, state.Spec.A2AEnabled)
}

// a2aEnabledNeedsNewVersion reports a real on/off change. Config null and false
// are the same wire value, so treating them as different would mint a new
// locked-artifact version (and roll the workload) for no API change.
func a2aEnabledNeedsNewVersion(plan, state types.Bool) bool {
	return a2aEnabledOn(plan) != a2aEnabledOn(state)
}

func a2aEnabledOn(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool()
}

func containerGroupsEqual(a, b ArtifactContainerGroupModel, ignoreManagedCodeRef, ignoreManagedImageURI bool) bool {
	if len(a.Containers) != len(b.Containers) {
		return false
	}
	for i := range a.Containers {
		skipImageURI := ignoreManagedImageURI && artifactContainerIsPrimary(a.Containers[i], a)
		if !containersEqual(a.Containers[i], b.Containers[i], ignoreManagedCodeRef, skipImageURI) {
			return false
		}
	}
	return true
}

func containersEqual(a, b ArtifactContainerModel, ignoreManagedCodeRef, ignoreImageURI bool) bool {
	if !a.Name.Equal(b.Name) ||
		(!ignoreImageURI && !a.ImageURI.Equal(b.ImageURI)) ||
		!a.Primary.Equal(b.Primary) ||
		!a.Description.Equal(b.Description) ||
		!a.Port.Equal(b.Port) ||
		!imageBuildConfigEqual(a.ImageBuildConfig, b.ImageBuildConfig, ignoreManagedCodeRef) ||
		!probesEqual(a.StartupProbe, b.StartupProbe) ||
		!probesEqual(a.ReadinessProbe, b.ReadinessProbe) ||
		!probesEqual(a.LivenessProbe, b.LivenessProbe) {
		return false
	}
	if len(a.Entrypoint) != len(b.Entrypoint) {
		return false
	}
	for i := range a.Entrypoint {
		if !a.Entrypoint[i].Equal(b.Entrypoint[i]) {
			return false
		}
	}
	if len(a.EnvironmentVars) != len(b.EnvironmentVars) {
		return false
	}
	for i := range a.EnvironmentVars {
		if !a.EnvironmentVars[i].Source.Equal(b.EnvironmentVars[i].Source) ||
			!a.EnvironmentVars[i].Name.Equal(b.EnvironmentVars[i].Name) ||
			!a.EnvironmentVars[i].Value.Equal(b.EnvironmentVars[i].Value) ||
			!a.EnvironmentVars[i].DrCredentialID.Equal(b.EnvironmentVars[i].DrCredentialID) ||
			!a.EnvironmentVars[i].Key.Equal(b.EnvironmentVars[i].Key) {
			return false
		}
	}
	return true
}

func imageBuildConfigEqual(a, b *ArtifactImageBuildConfigModel, ignoreManagedCodeRef bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !ignoreManagedCodeRef && !codeRefEqual(imageBuildConfigCodeRef(a), imageBuildConfigCodeRef(b)) {
		return false
	}
	return dockerfileEqual(a.Dockerfile, b.Dockerfile)
}

func codeRefEqual(a, b *ArtifactCodeRefModel) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.CatalogID.Equal(b.CatalogID) && a.CatalogVersionID.Equal(b.CatalogVersionID)
}

func dockerfileEqual(a, b *ArtifactDockerfileModel) bool {
	aNorm := normalizeDockerfileForEqual(a)
	bNorm := normalizeDockerfileForEqual(b)

	if !aNorm.Source.Equal(bNorm.Source) ||
		!aNorm.Path.Equal(bNorm.Path) ||
		!aNorm.ExecutionEnvironmentID.Equal(bNorm.ExecutionEnvironmentID) ||
		!aNorm.ExecutionEnvironmentVersionID.Equal(bNorm.ExecutionEnvironmentVersionID) {
		return false
	}
	if len(aNorm.Entrypoint) != len(bNorm.Entrypoint) {
		return false
	}
	for i := range aNorm.Entrypoint {
		if !aNorm.Entrypoint[i].Equal(bNorm.Entrypoint[i]) {
			return false
		}
	}
	return true
}

// normalizeDockerfileForEqual applies API/schema defaults so omitted provided-dockerfile
// blocks and null path in state compare equal to explicit {source: provided, path: ./Dockerfile}.
func normalizeDockerfileForEqual(df *ArtifactDockerfileModel) ArtifactDockerfileModel {
	if df == nil {
		return defaultProvidedDockerfileModel()
	}

	source := "provided"
	if !df.Source.IsNull() && !df.Source.IsUnknown() {
		source = df.Source.ValueString()
	}

	if source == "generated" {
		return ArtifactDockerfileModel{
			Source:                        types.StringValue("generated"),
			Path:                          types.StringNull(),
			ExecutionEnvironmentID:        df.ExecutionEnvironmentID,
			ExecutionEnvironmentVersionID: df.ExecutionEnvironmentVersionID,
			Entrypoint:                    df.Entrypoint,
		}
	}

	path := "./Dockerfile"
	if !df.Path.IsNull() && !df.Path.IsUnknown() && df.Path.ValueString() != "" {
		path = df.Path.ValueString()
	}

	return ArtifactDockerfileModel{
		Source:                        types.StringValue("provided"),
		Path:                          types.StringValue(path),
		ExecutionEnvironmentID:        types.StringNull(),
		ExecutionEnvironmentVersionID: types.StringNull(),
	}
}

func defaultProvidedDockerfileModel() ArtifactDockerfileModel {
	return ArtifactDockerfileModel{
		Source:                        types.StringValue("provided"),
		Path:                          types.StringValue("./Dockerfile"),
		ExecutionEnvironmentID:        types.StringNull(),
		ExecutionEnvironmentVersionID: types.StringNull(),
	}
}

func probesEqual(a, b *ArtifactProbeConfigModel) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Path.Equal(b.Path) &&
		a.Port.Equal(b.Port) &&
		a.Scheme.Equal(b.Scheme) &&
		a.Host.Equal(b.Host) &&
		a.InitialDelaySeconds.Equal(b.InitialDelaySeconds) &&
		a.PeriodSeconds.Equal(b.PeriodSeconds) &&
		a.TimeoutSeconds.Equal(b.TimeoutSeconds) &&
		a.FailureThreshold.Equal(b.FailureThreshold) &&
		a.SuccessThreshold.Equal(b.SuccessThreshold)
}

func validateArtifactContainerGroupsCount(resp *resource.ValidateConfigResponse, groups []ArtifactContainerGroupModel) {
	if len(groups) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("spec").AtName("container_groups"),
			"Missing container groups",
			"At least one container group must be defined in the artifact spec.",
		)
	} else if len(groups) > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("spec").AtName("container_groups"),
			"Too many container groups",
			"Currently, Workload API supports only 1 container group.",
		)
	}
}

func validateArtifactEnvironmentVar(resp *resource.ValidateConfigResponse, evPath path.Path, ev ArtifactEnvironmentVariableModel) {
	if ev.Source.IsUnknown() {
		return
	}

	source := ev.Source.ValueString()
	if ev.Source.IsNull() {
		source = client.EnvironmentVariableSourceString
	}

	switch source {
	case client.EnvironmentVariableSourceString:
		if ev.Name.IsNull() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("name"),
				"Missing name",
				`"name" is required when source is "string".`)
		}
		if !ev.Value.IsUnknown() && ev.Value.IsNull() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("value"),
				"Missing value",
				`"value" is required when source is "string".`)
		}
		if !ev.DrCredentialID.IsNull() && !ev.DrCredentialID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("dr_credential_id"),
				"Unexpected field",
				`"dr_credential_id" must not be set when source is "string".`)
		}
		if !ev.Key.IsNull() && !ev.Key.IsUnknown() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("key"),
				"Unexpected field",
				`"key" must not be set when source is "string".`)
		}
	case client.EnvironmentVariableSourceCredential:
		if ev.Name.IsNull() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("name"),
				"Missing name",
				`"name" is required when source is "dr-credential".`)
		}
		if !ev.DrCredentialID.IsUnknown() && ev.DrCredentialID.IsNull() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("dr_credential_id"),
				"Missing dr_credential_id",
				`"dr_credential_id" is required when source is "dr-credential".`)
		}
		if !ev.Key.IsUnknown() && ev.Key.IsNull() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("key"),
				"Missing key",
				`"key" is required when source is "dr-credential".`)
		}
		if !ev.Value.IsNull() && !ev.Value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("value"),
				"Unexpected field",
				`"value" must not be set when source is "dr-credential".`)
		}
	case client.EnvironmentVariableSourceAPIKey:
		if !ev.Value.IsNull() && !ev.Value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("value"),
				"Unexpected field",
				`"value" must not be set when source is "api-key"; the platform resolves the token value.`)
		}
		if !ev.DrCredentialID.IsNull() && !ev.DrCredentialID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("dr_credential_id"),
				"Unexpected field",
				`"dr_credential_id" must not be set when source is "api-key".`)
		}
		if !ev.Key.IsNull() && !ev.Key.IsUnknown() {
			resp.Diagnostics.AddAttributeError(evPath.AtName("key"),
				"Unexpected field",
				`"key" must not be set when source is "api-key".`)
		}
	default:
		resp.Diagnostics.AddAttributeError(evPath.AtName("source"),
			"Invalid source",
			fmt.Sprintf(`Invalid source %q. Allowed values: "string", "dr-credential", "api-key".`, source))
	}
}

func validateArtifactContainer(
	resp *resource.ValidateConfigResponse,
	containerPath path.Path,
	container ArtifactContainerModel,
	status, artifactType string,
	containerCount int,
	sourceConfigured bool,
) {
	hasImageURI := !container.ImageURI.IsNull() &&
		!container.ImageURI.IsUnknown() &&
		container.ImageURI.ValueString() != ""
	hasBuildConfig := container.ImageBuildConfig != nil

	if !hasImageURI && !hasBuildConfig {
		resp.Diagnostics.AddAttributeError(
			containerPath,
			"Missing image source",
			"Each container must set `image_uri`, `image_build_config`, or both.",
		)
	}

	if hasBuildConfig {
		validateImageBuildConfigPrimary(resp, containerPath, container, containerCount)
		validateImageBuildConfig(resp, containerPath, container.ImageBuildConfig, artifactType)
		if status == string(client.ArtifactStatusLocked) && !hasImageURI && !sourceConfigured {
			resp.Diagnostics.AddAttributeError(
				containerPath.AtName("image_build_config"),
				"Incomplete build configuration for locked artifact",
				"Locked artifacts with `image_build_config` require `image_uri` (complete the image build before locking). Use `status = \"draft\"` until the image build finishes.",
			)
		}
	}

	for ei, ev := range container.EnvironmentVars {
		validateArtifactEnvironmentVar(resp, containerPath.AtName("environment_vars").AtListIndex(ei), ev)
	}
}

func (r *ArtifactResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ArtifactResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateArtifactSource(resp, data)
	validateArtifactA2AEnabled(resp, data)

	if data.Spec == nil {
		return
	}
	validateArtifactContainerGroupsCount(resp, data.Spec.ContainerGroups)

	status := string(client.ArtifactStatusLocked)
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		status = data.Status.ValueString()
	}
	artifactType := "service"
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		artifactType = data.Type.ValueString()
	}
	sourceConfigured := artifactSourceConfigured(&data)

	for gi, group := range data.Spec.ContainerGroups {
		for ci, container := range group.Containers {
			containerPath := path.Root("spec").
				AtName("container_groups").AtListIndex(gi).
				AtName("containers").AtListIndex(ci)
			validateArtifactContainer(resp, containerPath, container, status, artifactType, len(group.Containers), sourceConfigured)
		}
	}
}

func validateArtifactA2AEnabled(resp *resource.ValidateConfigResponse, data ArtifactResourceModel) {
	if data.Spec == nil {
		return
	}
	if data.Spec.A2AEnabled.IsNull() || data.Spec.A2AEnabled.IsUnknown() {
		return
	}
	// Root variables are unknown during Terraform's validate walk. Do not treat
	// type = var.artifact_type as service or plan fails even when the value is agent.
	if data.Type.IsUnknown() {
		return
	}

	artifactType := string(client.ArtifactTypeService)
	if !data.Type.IsNull() {
		artifactType = data.Type.ValueString()
	}
	if artifactType == string(client.ArtifactTypeAgent) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("spec").AtName("a2a_enabled"),
		"Unsupported a2a_enabled",
		"`a2a_enabled` is only valid on agent artifacts.",
	)
}

func validateArtifactSource(resp *resource.ValidateConfigResponse, data ArtifactResourceModel) {
	if data.Source == nil {
		return
	}

	sourcePath := path.Root("source")

	if !IsKnown(data.Source.Dir) {
		resp.Diagnostics.AddAttributeError(
			sourcePath.AtName("dir"),
			"Missing source directory",
			"`source.dir` is required when the `source` block is set.",
		)
		return
	}

	dir := data.Source.Dir.ValueString()
	absDir, err := filepath.Abs(dir)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			sourcePath.AtName("dir"),
			"Invalid source directory path",
			fmt.Sprintf("Could not resolve %q to an absolute path: %s", dir, err),
		)
		return
	}

	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			resp.Diagnostics.AddAttributeError(
				sourcePath.AtName("dir"),
				"Source directory not found",
				fmt.Sprintf("Directory %q does not exist.", absDir),
			)
		} else {
			resp.Diagnostics.AddAttributeError(
				sourcePath.AtName("dir"),
				"Source directory not accessible",
				fmt.Sprintf("Could not access %q: %s", absDir, err),
			)
		}
		return
	}
	if !info.IsDir() {
		resp.Diagnostics.AddAttributeError(
			sourcePath.AtName("dir"),
			"Invalid source directory",
			fmt.Sprintf("%q is not a directory.", absDir),
		)
		return
	}

	artifactType := string(client.ArtifactTypeService)
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		artifactType = data.Type.ValueString()
	}
	if artifactType == string(client.ArtifactTypeNim) {
		resp.Diagnostics.AddAttributeError(
			sourcePath,
			"Unsupported source on NIM artifacts",
			"NIM artifacts cannot use `source`; upload code through the NIM workflow instead.",
		)
	}

	if data.Spec == nil {
		resp.Diagnostics.AddAttributeError(
			sourcePath,
			"Missing image build target",
			"`source` requires a primary container with `image_build_config` in `spec`.",
		)
		return
	}

	if !artifactHasPrimaryImageBuildConfig(data.Spec) {
		resp.Diagnostics.AddAttributeError(
			sourcePath,
			"Missing image build target",
			"`source` requires a primary container with `image_build_config` in `spec`.",
		)
	}

	status := string(client.ArtifactStatusLocked)
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		status = data.Status.ValueString()
	}
	if status == string(client.ArtifactStatusLocked) &&
		!data.Source.WaitForBuild.IsNull() &&
		!data.Source.WaitForBuild.IsUnknown() &&
		!data.Source.WaitForBuild.ValueBool() &&
		!artifactHasPrimaryImageURI(data.Spec) {
		resp.Diagnostics.AddAttributeError(
			sourcePath.AtName("wait_for_build"),
			"Invalid wait_for_build on locked artifact",
			"Locked artifacts require a completed image build before locking. Setting `wait_for_build = false` is only supported when `status = \"draft\"` or when `image_uri` is explicitly specified.",
		)
	}

	if artifactHasManualCodeRef(data.Spec) {
		resp.Diagnostics.AddAttributeError(
			sourcePath,
			"Conflicting code_ref",
			"Do not set `image_build_config.code_ref` when `source` is set; the provider manages `code_ref` from `source.dir`.",
		)
	}
}

func artifactHasPrimaryImageURI(spec *ArtifactSpecModel) bool {
	if spec == nil {
		return false
	}
	for _, group := range spec.ContainerGroups {
		for _, container := range group.Containers {
			isPrimary := !container.Primary.IsNull() && !container.Primary.IsUnknown() && container.Primary.ValueBool()
			if !isPrimary && len(group.Containers) == 1 &&
				(container.Primary.IsNull() || container.Primary.IsUnknown()) {
				isPrimary = true
			}
			if isPrimary && !container.ImageURI.IsNull() && !container.ImageURI.IsUnknown() && container.ImageURI.ValueString() != "" {
				return true
			}
		}
	}
	return false
}

func artifactHasPrimaryImageBuildConfig(spec *ArtifactSpecModel) bool {
	for _, group := range spec.ContainerGroups {
		for _, container := range group.Containers {
			isPrimary := !container.Primary.IsNull() && !container.Primary.IsUnknown() && container.Primary.ValueBool()
			if !isPrimary && len(group.Containers) == 1 &&
				(container.Primary.IsNull() || container.Primary.IsUnknown()) {
				isPrimary = true
			}
			if isPrimary && container.ImageBuildConfig != nil {
				return true
			}
		}
	}
	return false
}

func artifactHasManualCodeRef(spec *ArtifactSpecModel) bool {
	for _, group := range spec.ContainerGroups {
		for _, container := range group.Containers {
			ref := imageBuildConfigCodeRef(container.ImageBuildConfig)
			if ref == nil {
				continue
			}
			if IsKnown(ref.CatalogID) && IsKnown(ref.CatalogVersionID) {
				return true
			}
		}
	}
	return false
}

func validateImageBuildConfigPrimary(
	resp *resource.ValidateConfigResponse,
	containerPath path.Path,
	container ArtifactContainerModel,
	containerCount int,
) {
	if container.ImageBuildConfig == nil {
		return
	}

	if !container.Primary.IsNull() && !container.Primary.IsUnknown() && container.Primary.ValueBool() {
		return
	}
	if containerCount == 1 && (container.Primary.IsNull() || container.Primary.IsUnknown()) {
		// Workload API auto-marks the sole container as primary when primary is omitted.
		return
	}

	resp.Diagnostics.AddAttributeError(
		containerPath.AtName("image_build_config"),
		"Unsupported on non-primary container",
		"`image_build_config` is only permitted on the primary container.",
	)
}

func validateImageBuildConfig(resp *resource.ValidateConfigResponse, containerPath path.Path, cfg *ArtifactImageBuildConfigModel, artifactType string) {
	if cfg == nil {
		return
	}

	if ref := imageBuildConfigCodeRef(cfg); ref != nil &&
		!ref.CatalogID.IsNull() &&
		!ref.CatalogVersionID.IsNull() &&
		artifactType == string(client.ArtifactTypeNim) {
		resp.Diagnostics.AddAttributeError(
			containerPath.AtName("image_build_config").AtName("code_ref"),
			"Unsupported code reference",
			"NIM artifacts cannot include `code_ref` in `image_build_config`.",
		)
	}

	source := "provided"
	if cfg.Dockerfile != nil && !cfg.Dockerfile.Source.IsNull() && !cfg.Dockerfile.Source.IsUnknown() {
		source = cfg.Dockerfile.Source.ValueString()
	}

	dockerfilePath := containerPath.AtName("image_build_config").AtName("dockerfile")
	switch source {
	case "generated":
		if cfg.Dockerfile == nil {
			resp.Diagnostics.AddAttributeError(
				dockerfilePath,
				"Incomplete generated dockerfile",
				"`execution_environment_id`, `execution_environment_version_id`, and `entrypoint` are required when source is `generated`.",
			)
			return
		}
		if cfg.Dockerfile.ExecutionEnvironmentID.IsNull() || cfg.Dockerfile.ExecutionEnvironmentID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				dockerfilePath.AtName("execution_environment_id"),
				"Missing execution environment ID",
				"`execution_environment_id` is required when dockerfile source is `generated`.",
			)
		}
		if cfg.Dockerfile.ExecutionEnvironmentVersionID.IsNull() || cfg.Dockerfile.ExecutionEnvironmentVersionID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				dockerfilePath.AtName("execution_environment_version_id"),
				"Missing execution environment version ID",
				"`execution_environment_version_id` is required when dockerfile source is `generated`.",
			)
		}
		if len(cfg.Dockerfile.Entrypoint) == 0 {
			resp.Diagnostics.AddAttributeError(
				dockerfilePath.AtName("entrypoint"),
				"Missing entrypoint",
				"`entrypoint` is required when dockerfile source is `generated`.",
			)
		}
		if IsKnown(cfg.Dockerfile.Path) {
			resp.Diagnostics.AddAttributeError(
				dockerfilePath.AtName("path"),
				"Conflicting dockerfile path",
				"`path` is not used when dockerfile source is `generated` and would be silently discarded; remove it or set `source = \"provided\"`.",
			)
		}
	case "provided":
		// path defaults to ./Dockerfile at marshal time
	default:
		resp.Diagnostics.AddAttributeError(
			dockerfilePath.AtName("source"),
			"Invalid dockerfile source",
			fmt.Sprintf("Invalid dockerfile source %q. Allowed values: \"provided\", \"generated\".", source),
		)
	}
}

func (r *ArtifactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(uuid.NewString()))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("artifact_id"), types.StringValue(req.ID))...)
}

func artifactCreateRequest(data ArtifactResourceModel) *client.CreateArtifactRequest {
	status := client.ArtifactStatusLocked
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		status = client.ArtifactStatus(data.Status.ValueString())
	}

	req := &client.CreateArtifactRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Type:        client.ArtifactType(data.Type.ValueString()),
		Status:      status,
		Spec:        artifactSpecToClient(*data.Spec, client.ArtifactType(data.Type.ValueString())),
	}
	if !data.ArtifactRepositoryID.IsNull() && !data.ArtifactRepositoryID.IsUnknown() {
		repoID := data.ArtifactRepositoryID.ValueString()
		req.ArtifactRepositoryID = &repoID
	}
	return req
}

func patchRequestFromPlan(plan, state ArtifactResourceModel, deferLock bool) *client.PatchArtifactRequest {
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	spec := artifactSpecToClient(*plan.Spec, client.ArtifactType(plan.Type.ValueString()))

	req := &client.PatchArtifactRequest{
		Name:        &name,
		Description: &description,
		Spec:        &spec,
	}

	if !deferLock &&
		plan.Status.ValueString() == string(client.ArtifactStatusLocked) &&
		state.Status.ValueString() == string(client.ArtifactStatusDraft) {
		locked := client.ArtifactStatusLocked
		req.Status = &locked
	}

	return req
}

func artifactSpecToClient(spec ArtifactSpecModel, artifactType client.ArtifactType) client.ArtifactSpec {
	groups := make([]client.ArtifactContainerGroup, len(spec.ContainerGroups))
	for i, g := range spec.ContainerGroups {
		groups[i] = artifactContainerGroupToClient(g)
	}
	out := client.ArtifactSpec{
		ContainerGroups: groups,
	}
	// Agents always send a2aEnabled so a draft PATCH can turn it off. omitempty
	// would drop a nil pointer and leave the stored API value unchanged.
	if artifactType == client.ArtifactTypeAgent && !spec.A2AEnabled.IsUnknown() {
		enabled := !spec.A2AEnabled.IsNull() && spec.A2AEnabled.ValueBool()
		out.A2AEnabled = &enabled
	}
	return out
}

func artifactContainerGroupToClient(g ArtifactContainerGroupModel) client.ArtifactContainerGroup {
	containers := make([]client.ArtifactContainer, len(g.Containers))
	for i, c := range g.Containers {
		containers[i] = artifactContainerToClient(c)
	}
	return client.ArtifactContainerGroup{Containers: containers}
}

func artifactContainerToClient(c ArtifactContainerModel) client.ArtifactContainer {
	container := client.ArtifactContainer{
		Description: c.Description.ValueString(),
	}

	if !c.ImageURI.IsNull() && !c.ImageURI.IsUnknown() && c.ImageURI.ValueString() != "" {
		container.ImageURI = c.ImageURI.ValueString()
	}

	if c.ImageBuildConfig != nil {
		container.ImageBuildConfig = artifactImageBuildConfigToClient(c.ImageBuildConfig)
	}

	if !c.Name.IsNull() && !c.Name.IsUnknown() {
		name := c.Name.ValueString()
		container.Name = &name
	}

	if !c.Primary.IsNull() && !c.Primary.IsUnknown() {
		primary := c.Primary.ValueBool()
		container.Primary = &primary
	}

	if !c.Port.IsNull() && !c.Port.IsUnknown() {
		port := c.Port.ValueInt64()
		container.Port = &port
	}

	if len(c.Entrypoint) > 0 {
		container.Entrypoint = make([]string, len(c.Entrypoint))
		for i, e := range c.Entrypoint {
			container.Entrypoint[i] = e.ValueString()
		}
	}

	if len(c.EnvironmentVars) > 0 {
		container.EnvironmentVars = make([]client.ArtifactEnvironmentVariable, len(c.EnvironmentVars))
		for i, ev := range c.EnvironmentVars {
			envVar := client.ArtifactEnvironmentVariable{
				Source: ev.Source.ValueString(),
				Name:   ev.Name.ValueString(),
			}
			switch ev.Source.ValueString() {
			case client.EnvironmentVariableSourceCredential:
				envVar.DrCredentialID = ev.DrCredentialID.ValueString()
				envVar.Key = ev.Key.ValueString()
			case client.EnvironmentVariableSourceAPIKey:
				// Only source (and name, when set) are sent; the platform
				// resolves the token value.
			default:
				envVar.Value = ev.Value.ValueString()
			}
			container.EnvironmentVars[i] = envVar
		}
	}

	container.StartupProbe = artifactProbeToClient(c.StartupProbe)
	container.ReadinessProbe = artifactProbeToClient(c.ReadinessProbe)
	container.LivenessProbe = artifactProbeToClient(c.LivenessProbe)

	return container
}

func artifactImageBuildConfigToClient(cfg *ArtifactImageBuildConfigModel) *client.ArtifactImageBuildConfig {
	if cfg == nil {
		return nil
	}

	result := &client.ArtifactImageBuildConfig{
		Dockerfile: artifactDockerfileToClient(cfg.Dockerfile),
	}

	if ref := imageBuildConfigCodeRef(cfg); ref != nil &&
		!ref.CatalogID.IsNull() && !ref.CatalogID.IsUnknown() &&
		!ref.CatalogVersionID.IsNull() && !ref.CatalogVersionID.IsUnknown() {
		result.CodeRef = &client.ArtifactCodeRef{
			Type:     "datarobot",
			Provider: "datarobot",
			DataRobot: client.ArtifactDataRobotCodeRef{
				CatalogID:        ref.CatalogID.ValueString(),
				CatalogVersionID: ref.CatalogVersionID.ValueString(),
			},
		}
	}

	return result
}

func artifactDockerfileToClient(df *ArtifactDockerfileModel) *client.ArtifactDockerfileConfig {
	source := "provided"
	if df != nil && !df.Source.IsNull() && !df.Source.IsUnknown() {
		source = df.Source.ValueString()
	}

	result := &client.ArtifactDockerfileConfig{Source: source}
	if source == "generated" {
		if df == nil {
			return result
		}
		result.ExecutionEnvironmentID = df.ExecutionEnvironmentID.ValueString()
		result.ExecutionEnvironmentVersionID = df.ExecutionEnvironmentVersionID.ValueString()
		if len(df.Entrypoint) > 0 {
			result.Entrypoint = make([]string, len(df.Entrypoint))
			for i, e := range df.Entrypoint {
				result.Entrypoint[i] = e.ValueString()
			}
		}
		return result
	}

	path := "./Dockerfile"
	if df != nil && !df.Path.IsNull() && !df.Path.IsUnknown() && df.Path.ValueString() != "" {
		path = df.Path.ValueString()
	}
	result.Path = path
	return result
}

func artifactProbeToClient(probe *ArtifactProbeConfigModel) *client.ArtifactProbeConfig {
	if probe == nil {
		return nil
	}
	p := &client.ArtifactProbeConfig{
		Path: probe.Path.ValueString(),
	}
	if !probe.Port.IsNull() && !probe.Port.IsUnknown() {
		port := probe.Port.ValueInt64()
		p.Port = &port
	}
	if !probe.Scheme.IsNull() && !probe.Scheme.IsUnknown() {
		scheme := probe.Scheme.ValueString()
		p.Scheme = &scheme
	}
	if !probe.Host.IsNull() && !probe.Host.IsUnknown() {
		host := probe.Host.ValueString()
		p.Host = &host
	}
	if !probe.InitialDelaySeconds.IsNull() && !probe.InitialDelaySeconds.IsUnknown() {
		v := probe.InitialDelaySeconds.ValueInt64()
		p.InitialDelaySeconds = &v
	}
	if !probe.PeriodSeconds.IsNull() && !probe.PeriodSeconds.IsUnknown() {
		v := probe.PeriodSeconds.ValueInt64()
		p.PeriodSeconds = &v
	}
	if !probe.TimeoutSeconds.IsNull() && !probe.TimeoutSeconds.IsUnknown() {
		v := probe.TimeoutSeconds.ValueInt64()
		p.TimeoutSeconds = &v
	}
	if !probe.FailureThreshold.IsNull() && !probe.FailureThreshold.IsUnknown() {
		v := probe.FailureThreshold.ValueInt64()
		p.FailureThreshold = &v
	}
	if !probe.SuccessThreshold.IsNull() && !probe.SuccessThreshold.IsUnknown() {
		v := probe.SuccessThreshold.ValueInt64()
		p.SuccessThreshold = &v
	}
	return p
}

func loadArtifactIntoModel(artifact *client.Artifact, data *ArtifactResourceModel) {
	data.ArtifactID = types.StringValue(artifact.ID)
	data.Name = types.StringValue(artifact.Name)
	if artifact.Description != "" {
		data.Description = types.StringValue(artifact.Description)
	} else if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}
	data.Type = types.StringValue(string(artifact.Type))
	data.Status = types.StringValue(string(artifact.Status))

	if artifact.ArtifactRepositoryID != nil {
		data.ArtifactRepositoryID = types.StringValue(*artifact.ArtifactRepositoryID)
	} else {
		data.ArtifactRepositoryID = types.StringNull()
	}

	spec := loadArtifactSpecFromAPI(artifact.Spec, data.Spec)
	data.Spec = &spec
}

func loadArtifactSpecFromAPI(spec client.ArtifactSpec, prior *ArtifactSpecModel) ArtifactSpecModel {
	groups := make([]ArtifactContainerGroupModel, len(spec.ContainerGroups))
	for i, g := range spec.ContainerGroups {
		containers := make([]ArtifactContainerModel, len(g.Containers))
		for j, c := range g.Containers {
			var priorContainer *ArtifactContainerModel
			if prior != nil && i < len(prior.ContainerGroups) && j < len(prior.ContainerGroups[i].Containers) {
				priorContainer = &prior.ContainerGroups[i].Containers[j]
			}
			containers[j] = loadContainerFromAPI(c, priorContainer)
		}
		groups[i] = ArtifactContainerGroupModel{Containers: containers}
	}
	return ArtifactSpecModel{
		ContainerGroups: groups,
		A2AEnabled:      loadA2AEnabledFromAPI(spec.A2AEnabled, prior),
	}
}

func loadA2AEnabledFromAPI(apiValue *bool, prior *ArtifactSpecModel) types.Bool {
	if prior != nil && !prior.A2AEnabled.IsNull() && !prior.A2AEnabled.IsUnknown() {
		if apiValue != nil {
			return types.BoolValue(*apiValue)
		}
		return prior.A2AEnabled
	}
	// Config omitted the flag. Keep null when the API is off so apply can
	// round-trip; surface true so a UI-enabled A2A value shows as drift.
	if apiValue != nil && *apiValue {
		return types.BoolValue(true)
	}
	return types.BoolNull()
}

func loadContainerFromAPI(c client.ArtifactContainer, prior *ArtifactContainerModel) ArtifactContainerModel {
	model := ArtifactContainerModel{}

	if c.ImageURI != "" {
		model.ImageURI = types.StringValue(c.ImageURI)
	} else {
		model.ImageURI = types.StringNull()
	}

	if c.ImageBuildConfig != nil {
		var priorBuild *ArtifactImageBuildConfigModel
		if prior != nil {
			priorBuild = prior.ImageBuildConfig
		}
		model.ImageBuildConfig = loadImageBuildConfigFromAPI(c.ImageBuildConfig, priorBuild)
	}

	if c.Name != nil {
		model.Name = types.StringValue(*c.Name)
	} else {
		model.Name = types.StringNull()
	}

	var priorDescription types.String
	if prior != nil {
		priorDescription = prior.Description
	}
	if c.Description != "" {
		model.Description = types.StringValue(c.Description)
	} else if priorDescription.IsUnknown() {
		model.Description = types.StringNull()
	} else {
		model.Description = priorDescription
	}

	if c.Primary != nil {
		model.Primary = types.BoolValue(*c.Primary)
	} else {
		model.Primary = types.BoolNull()
	}

	if c.Port != nil {
		model.Port = types.Int64Value(*c.Port)
	} else {
		model.Port = types.Int64Null()
	}

	if len(c.Entrypoint) > 0 {
		model.Entrypoint = make([]types.String, len(c.Entrypoint))
		for i, e := range c.Entrypoint {
			model.Entrypoint[i] = types.StringValue(e)
		}
	}

	if len(c.EnvironmentVars) > 0 {
		model.EnvironmentVars = make([]ArtifactEnvironmentVariableModel, len(c.EnvironmentVars))
		for i, ev := range c.EnvironmentVars {
			model.EnvironmentVars[i] = environmentVarModelFromAPI(ev)
		}
	} else if prior != nil && prior.EnvironmentVars != nil {
		model.EnvironmentVars = []ArtifactEnvironmentVariableModel{}
	}

	model.StartupProbe = loadProbeFromAPI(c.StartupProbe)
	model.ReadinessProbe = loadProbeFromAPI(c.ReadinessProbe)
	model.LivenessProbe = loadProbeFromAPI(c.LivenessProbe)
	model.Build = loadContainerBuildObjectFromAPI(c.Build)

	return model
}

func loadImageBuildConfigFromAPI(cfg *client.ArtifactImageBuildConfig, prior *ArtifactImageBuildConfigModel) *ArtifactImageBuildConfigModel {
	if cfg == nil {
		return nil
	}

	model := &ArtifactImageBuildConfigModel{
		CodeRef: types.ObjectNull(artifactCodeRefAttrTypes()),
	}
	if cfg.CodeRef != nil && (cfg.CodeRef.DataRobot.CatalogID != "" || cfg.CodeRef.DataRobot.CatalogVersionID != "") {
		_ = setImageBuildConfigCodeRef(model, &ArtifactCodeRefModel{
			CatalogID:        types.StringValue(cfg.CodeRef.DataRobot.CatalogID),
			CatalogVersionID: types.StringValue(cfg.CodeRef.DataRobot.CatalogVersionID),
		})
	} else if prior != nil && !prior.CodeRef.IsNull() && !prior.CodeRef.IsUnknown() {
		model.CodeRef = prior.CodeRef
	}

	if cfg.Dockerfile != nil {
		model.Dockerfile = loadDockerfileFromAPI(cfg.Dockerfile)
	} else if prior != nil && prior.Dockerfile != nil {
		model.Dockerfile = prior.Dockerfile
	}

	return model
}

func loadDockerfileFromAPI(df *client.ArtifactDockerfileConfig) *ArtifactDockerfileModel {
	if df == nil {
		return nil
	}

	model := &ArtifactDockerfileModel{
		Source: types.StringValue(df.Source),
	}

	if df.Source == "generated" {
		if df.ExecutionEnvironmentID != "" {
			model.ExecutionEnvironmentID = types.StringValue(df.ExecutionEnvironmentID)
		} else {
			model.ExecutionEnvironmentID = types.StringNull()
		}
		if df.ExecutionEnvironmentVersionID != "" {
			model.ExecutionEnvironmentVersionID = types.StringValue(df.ExecutionEnvironmentVersionID)
		} else {
			model.ExecutionEnvironmentVersionID = types.StringNull()
		}
		if len(df.Entrypoint) > 0 {
			model.Entrypoint = make([]types.String, len(df.Entrypoint))
			for i, e := range df.Entrypoint {
				model.Entrypoint[i] = types.StringValue(e)
			}
		}
		model.Path = types.StringNull()
		return model
	}

	if df.Path != "" {
		model.Path = types.StringValue(df.Path)
	} else {
		model.Path = types.StringValue("./Dockerfile")
	}
	return model
}

func loadProbeFromAPI(probe *client.ArtifactProbeConfig) *ArtifactProbeConfigModel {
	if probe == nil {
		return nil
	}
	m := &ArtifactProbeConfigModel{
		Path: types.StringValue(probe.Path),
	}
	if probe.Port != nil {
		m.Port = types.Int64Value(*probe.Port)
	} else {
		m.Port = types.Int64Null()
	}
	if probe.Scheme != nil {
		m.Scheme = types.StringValue(*probe.Scheme)
	} else {
		m.Scheme = types.StringNull()
	}
	if probe.Host != nil {
		m.Host = types.StringValue(*probe.Host)
	} else {
		m.Host = types.StringNull()
	}
	if probe.InitialDelaySeconds != nil {
		m.InitialDelaySeconds = types.Int64Value(*probe.InitialDelaySeconds)
	} else {
		m.InitialDelaySeconds = types.Int64Null()
	}
	if probe.PeriodSeconds != nil {
		m.PeriodSeconds = types.Int64Value(*probe.PeriodSeconds)
	} else {
		m.PeriodSeconds = types.Int64Null()
	}
	if probe.TimeoutSeconds != nil {
		m.TimeoutSeconds = types.Int64Value(*probe.TimeoutSeconds)
	} else {
		m.TimeoutSeconds = types.Int64Null()
	}
	if probe.FailureThreshold != nil {
		m.FailureThreshold = types.Int64Value(*probe.FailureThreshold)
	} else {
		m.FailureThreshold = types.Int64Null()
	}
	if probe.SuccessThreshold != nil {
		m.SuccessThreshold = types.Int64Value(*probe.SuccessThreshold)
	} else {
		m.SuccessThreshold = types.Int64Null()
	}
	return m
}
