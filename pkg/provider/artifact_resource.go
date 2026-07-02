package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	probeAttributes := map[string]schema.Attribute{
		"path": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "URL path to query for health check.",
		},
		"port": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Port number to access on the container.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"scheme": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Scheme to use for connecting to the host (HTTP or HTTPS).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"host": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Host name to connect to, defaults to the pod IP.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"initial_delay_seconds": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Number of seconds to wait before the first probe is executed.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"period_seconds": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "How often (in seconds) to perform the probe.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"timeout_seconds": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Number of seconds after which the probe times out.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"failure_threshold": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Minimum consecutive failures for the probe to be considered failed.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
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
				MarkdownDescription: "The artifact type: `service` or `nim`. Defaults to `service`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Artifact lifecycle status: `draft` (mutable; supports in-place updates and image builds) or `locked` (immutable; spec changes create a new version). Defaults to `locked`. Locking a draft artifact is one-way.",
				Default:             stringdefault.StaticString(string(client.ArtifactStatusLocked)),
				Validators:          ArtifactStatusValidators(),
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
			"spec": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The artifact specification containing container group definitions.",
				Attributes: map[string]schema.Attribute{
					"container_groups": schema.ListNestedAttribute{
						Required:            true,
						MarkdownDescription: "List of container groups.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"containers": schema.ListNestedAttribute{
									Required:            true,
									MarkdownDescription: "List of containers in this group.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Optional:            true,
												Computed:            true,
												MarkdownDescription: "Name of the container.",
												PlanModifiers: []planmodifier.String{
													stringplanmodifier.UseStateForUnknown(),
												},
											},
											"image_uri": schema.StringAttribute{
												Required:            true,
												MarkdownDescription: "Docker image URI.",
											},
											"primary": schema.BoolAttribute{
												Optional:            true,
												Computed:            true,
												MarkdownDescription: "Whether this is the primary container.",
												PlanModifiers: []planmodifier.Bool{
													boolplanmodifier.UseStateForUnknown(),
												},
											},
											"description": schema.StringAttribute{
												Optional:            true,
												MarkdownDescription: "Description of the container.",
											},
											"port": schema.Int64Attribute{
												Optional:            true,
												Computed:            true,
												MarkdownDescription: "Container access port (1024-65535). Required for primary containers; omit for non-primary.",
												PlanModifiers: []planmodifier.Int64{
													int64planmodifier.UseStateForUnknown(),
												},
											},
											"entrypoint": schema.ListAttribute{
												Optional:            true,
												ElementType:         types.StringType,
												MarkdownDescription: "Container entrypoint.",
											},
											"environment_vars": schema.ListNestedAttribute{
												Optional:            true,
												MarkdownDescription: "Environment variables for the container.",
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"source": schema.StringAttribute{
															Optional:            true,
															Computed:            true,
															Default:             stringdefault.StaticString(client.EnvironmentVariableSourceString),
															MarkdownDescription: `Source type: "string" for plain text values, "dr-credential" for DataRobot credentials. Defaults to "string".`,
														},
														"name": schema.StringAttribute{
															Required:            true,
															MarkdownDescription: "Name of the environment variable.",
														},
														"value": schema.StringAttribute{
															Optional:            true,
															MarkdownDescription: `Value of the environment variable. Required when source is "string".`,
														},
														"dr_credential_id": schema.StringAttribute{
															Optional:            true,
															MarkdownDescription: `DataRobot credential ID. Required when source is "dr-credential".`,
														},
														"key": schema.StringAttribute{
															Optional:            true,
															MarkdownDescription: `Key within the credential. Required when source is "dr-credential".`,
														},
													},
												},
											},
											"startup_probe": schema.SingleNestedAttribute{
												Optional:            true,
												MarkdownDescription: "Container startup check configuration.",
												Attributes:          probeAttributes,
											},
											"readiness_probe": schema.SingleNestedAttribute{
												Optional:            true,
												MarkdownDescription: "Container readiness check configuration.",
												Attributes:          probeAttributes,
											},
											"liveness_probe": schema.SingleNestedAttribute{
												Optional:            true,
												MarkdownDescription: "Container liveness check configuration.",
												Attributes:          probeAttributes,
											},
										},
									},
								},
							},
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateArtifact")
	artifact, err := r.provider.service.CreateArtifact(ctx, artifactCreateRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Artifact", err.Error())
		return
	}

	data.ID = types.StringValue(uuid.NewString())
	loadArtifactIntoModel(artifact, &data)
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ArtifactResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// artifact_repository_id is Optional+Computed: when the user doesn't set it in config,
	// the plan value is null (UseStateForUnknown only applies to unknown, not null).
	// Preserve the computed value from state so subsequent versions are created in the same repo.
	if plan.ArtifactRepositoryID.IsNull() && !state.ArtifactRepositoryID.IsNull() {
		plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	}

	if state.Status.ValueString() == string(client.ArtifactStatusLocked) &&
		plan.Status.ValueString() == string(client.ArtifactStatusDraft) {
		resp.Diagnostics.AddError(
			"Invalid status change",
			"Cannot revert a locked artifact to draft.",
		)
		return
	}

	var artifact *client.Artifact
	var err error

	if state.Status.ValueString() == string(client.ArtifactStatusDraft) {
		traceAPICall("PatchArtifact")
		artifact, err = r.provider.service.PatchArtifact(
			ctx,
			state.ArtifactID.ValueString(),
			patchRequestFromPlan(plan, state),
		)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Artifact", err.Error())
			return
		}
	} else {
		traceAPICall("CreateUpdatedArtifact")
		artifact, err = r.provider.service.CreateArtifact(ctx, artifactCreateRequest(plan))
		if err != nil {
			resp.Diagnostics.AddError("Error creating new Artifact version", err.Error())
			return
		}
	}

	loadArtifactIntoModel(artifact, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state ArtifactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// artifact_repository_id is Optional+Computed. The Pulumi bridge passes null for unset
	// Optional+Computed fields (bypassing UseStateForUnknown), so we restore it from state
	// here to keep the plan accurate and avoid false positives in artifactNeedsNewVersion.
	if plan.ArtifactRepositoryID.IsNull() && !state.ArtifactRepositoryID.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("artifact_repository_id"), state.ArtifactRepositoryID)...)
		plan.ArtifactRepositoryID = state.ArtifactRepositoryID
	}

	if state.Status.ValueString() == string(client.ArtifactStatusLocked) &&
		plan.Status.ValueString() == string(client.ArtifactStatusDraft) {
		resp.Diagnostics.AddError(
			"Invalid status change",
			"Cannot revert a locked artifact to draft.",
		)
		return
	}

	if state.Status.ValueString() == string(client.ArtifactStatusLocked) && artifactNeedsNewVersion(plan, state) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("artifact_id"), types.StringUnknown())...)
	}
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
	for i := range plan.Spec.ContainerGroups {
		if !containerGroupsEqual(plan.Spec.ContainerGroups[i], state.Spec.ContainerGroups[i]) {
			return true
		}
	}
	return false
}

func containerGroupsEqual(a, b ArtifactContainerGroupModel) bool {
	if len(a.Containers) != len(b.Containers) {
		return false
	}
	for i := range a.Containers {
		if !containersEqual(a.Containers[i], b.Containers[i]) {
			return false
		}
	}
	return true
}

func containersEqual(a, b ArtifactContainerModel) bool {
	if !a.Name.Equal(b.Name) ||
		!a.ImageURI.Equal(b.ImageURI) ||
		!a.Primary.Equal(b.Primary) ||
		!a.Description.Equal(b.Description) ||
		!a.Port.Equal(b.Port) ||
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
		a.FailureThreshold.Equal(b.FailureThreshold)
}

func (r *ArtifactResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ArtifactResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Spec == nil {
		return
	}
	if len(data.Spec.ContainerGroups) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("spec").AtName("container_groups"),
			"Missing container groups",
			"At least one container group must be defined in the artifact spec.",
		)
	} else if len(data.Spec.ContainerGroups) > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("spec").AtName("container_groups"),
			"Too many container groups",
			"Currently, Workload API supports only 1 container group.",
		)
	}

	for gi, group := range data.Spec.ContainerGroups {
		for ci, container := range group.Containers {
			for ei, ev := range container.EnvironmentVars {
				if ev.Source.IsUnknown() {
					continue
				}
				evPath := path.Root("spec").
					AtName("container_groups").AtListIndex(gi).
					AtName("containers").AtListIndex(ci).
					AtName("environment_vars").AtListIndex(ei)

				source := ev.Source.ValueString()
				if ev.Source.IsNull() {
					source = client.EnvironmentVariableSourceString
				}

				switch source {
				case client.EnvironmentVariableSourceString:
					if ev.Value.IsNull() || ev.Value.IsUnknown() {
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
					if ev.DrCredentialID.IsNull() || ev.DrCredentialID.IsUnknown() {
						resp.Diagnostics.AddAttributeError(evPath.AtName("dr_credential_id"),
							"Missing dr_credential_id",
							`"dr_credential_id" is required when source is "dr-credential".`)
					}
					if ev.Key.IsNull() || ev.Key.IsUnknown() {
						resp.Diagnostics.AddAttributeError(evPath.AtName("key"),
							"Missing key",
							`"key" is required when source is "dr-credential".`)
					}
					if !ev.Value.IsNull() && !ev.Value.IsUnknown() {
						resp.Diagnostics.AddAttributeError(evPath.AtName("value"),
							"Unexpected field",
							`"value" must not be set when source is "dr-credential".`)
					}
				default:
					resp.Diagnostics.AddAttributeError(evPath.AtName("source"),
						"Invalid source",
						fmt.Sprintf(`Invalid source %q. Allowed values: "string", "dr-credential".`, source))
				}
			}
		}
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
		Spec:        artifactSpecToClient(*data.Spec),
	}
	if !data.ArtifactRepositoryID.IsNull() && !data.ArtifactRepositoryID.IsUnknown() {
		repoID := data.ArtifactRepositoryID.ValueString()
		req.ArtifactRepositoryID = &repoID
	}
	return req
}

func patchRequestFromPlan(plan, state ArtifactResourceModel) *client.PatchArtifactRequest {
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	spec := artifactSpecToClient(*plan.Spec)

	req := &client.PatchArtifactRequest{
		Name:        &name,
		Description: &description,
		Spec:        &spec,
	}

	if plan.Status.ValueString() == string(client.ArtifactStatusLocked) &&
		state.Status.ValueString() == string(client.ArtifactStatusDraft) {
		locked := client.ArtifactStatusLocked
		req.Status = &locked
	}

	return req
}

func artifactSpecToClient(spec ArtifactSpecModel) client.ArtifactSpec {
	groups := make([]client.ArtifactContainerGroup, len(spec.ContainerGroups))
	for i, g := range spec.ContainerGroups {
		groups[i] = artifactContainerGroupToClient(g)
	}
	return client.ArtifactSpec{
		ContainerGroups: groups,
	}
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
		ImageURI:    c.ImageURI.ValueString(),
		Description: c.Description.ValueString(),
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
			if ev.Source.ValueString() == client.EnvironmentVariableSourceCredential {
				envVar.DrCredentialID = ev.DrCredentialID.ValueString()
				envVar.Key = ev.Key.ValueString()
			} else {
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
	return ArtifactSpecModel{ContainerGroups: groups}
}

func loadContainerFromAPI(c client.ArtifactContainer, prior *ArtifactContainerModel) ArtifactContainerModel {
	model := ArtifactContainerModel{
		ImageURI: types.StringValue(c.ImageURI),
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
			m := ArtifactEnvironmentVariableModel{
				Source:         types.StringValue(ev.Source),
				Name:           types.StringValue(ev.Name),
				Value:          types.StringNull(),
				DrCredentialID: types.StringNull(),
				Key:            types.StringNull(),
			}
			if ev.Source == client.EnvironmentVariableSourceCredential {
				m.DrCredentialID = types.StringValue(ev.DrCredentialID)
				m.Key = types.StringValue(ev.Key)
			} else {
				m.Value = types.StringValue(ev.Value)
			}
			model.EnvironmentVars[i] = m
		}
	} else if prior != nil && prior.EnvironmentVars != nil {
		model.EnvironmentVars = []ArtifactEnvironmentVariableModel{}
	}

	model.StartupProbe = loadProbeFromAPI(c.StartupProbe)
	model.ReadinessProbe = loadProbeFromAPI(c.ReadinessProbe)
	model.LivenessProbe = loadProbeFromAPI(c.LivenessProbe)

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
	return m
}
