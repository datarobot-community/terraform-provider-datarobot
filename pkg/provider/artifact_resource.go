package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
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
				MarkdownDescription: "The ID of the Artifact.",
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
														"name": schema.StringAttribute{
															Required:            true,
															MarkdownDescription: "Name of the environment variable.",
														},
														"value": schema.StringAttribute{
															Required:            true,
															MarkdownDescription: "Value of the environment variable.",
														},
													},
												},
											},
											"resource_request": schema.SingleNestedAttribute{
												Required:            true,
												MarkdownDescription: "Resource requirements for the container.",
												Attributes: map[string]schema.Attribute{
													"cpu": schema.Float64Attribute{
														Required:            true,
														MarkdownDescription: "Number of CPU cores required.",
													},
													"memory": schema.Int64Attribute{
														Required:            true,
														MarkdownDescription: "Memory required in bytes.",
													},
													"gpu": schema.Int64Attribute{
														Optional:            true,
														MarkdownDescription: "Number of GPUs required.",
													},
													"gpu_type": schema.StringAttribute{
														Optional:            true,
														MarkdownDescription: "GPU type required (e.g., NVIDIA-A100).",
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

	loadArtifactIntoModel(artifact, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ArtifactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ArtifactResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetArtifact")
	artifact, err := r.provider.service.GetArtifact(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Artifact not found",
				fmt.Sprintf("Artifact with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Artifact with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	loadArtifactIntoModel(artifact, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ArtifactResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateUpdatedArtifact")
	artifact, err := r.provider.service.CreateArtifact(ctx, artifactCreateRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating new Artifact version", err.Error())
		return
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

	resp.Diagnostics.AddWarning(
		"Locked artifacts can't be deleted from the API",
		fmt.Sprintf("Artifact %s was not removed from %s artifact repository.", data.ID.ValueString(), data.ArtifactRepositoryID.ValueString()),
	)
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

	if artifactNeedsNewVersion(plan, state) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
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
		!resourceRequestsEqual(a.ResourceRequest, b.ResourceRequest) ||
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
		if !a.EnvironmentVars[i].Name.Equal(b.EnvironmentVars[i].Name) ||
			!a.EnvironmentVars[i].Value.Equal(b.EnvironmentVars[i].Value) {
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

func resourceRequestsEqual(a, b ArtifactResourceRequestModel) bool {
	return a.CPU.Equal(b.CPU) &&
		a.Memory.Equal(b.Memory) &&
		a.GPU.Equal(b.GPU) &&
		a.GPUType.Equal(b.GPUType)
}

func (r *ArtifactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func artifactCreateRequest(data ArtifactResourceModel) *client.CreateArtifactRequest {
	req := &client.CreateArtifactRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Type:        client.ArtifactType(data.Type.ValueString()),
		Status:      client.ArtifactStatusLocked,
		Spec:        artifactSpecToClient(*data.Spec),
	}
	if !data.ArtifactRepositoryID.IsNull() && !data.ArtifactRepositoryID.IsUnknown() {
		repoID := data.ArtifactRepositoryID.ValueString()
		req.ArtifactRepositoryID = &repoID
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
		ImageURI:        c.ImageURI.ValueString(),
		Description:     c.Description.ValueString(),
		ResourceRequest: artifactResourceRequestToClient(c.ResourceRequest),
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
			container.EnvironmentVars[i] = client.ArtifactEnvironmentVariable{
				Name:  ev.Name.ValueString(),
				Value: ev.Value.ValueString(),
			}
		}
	}

	container.StartupProbe = artifactProbeToClient(c.StartupProbe)
	container.ReadinessProbe = artifactProbeToClient(c.ReadinessProbe)
	container.LivenessProbe = artifactProbeToClient(c.LivenessProbe)

	return container
}

func artifactResourceRequestToClient(rr ArtifactResourceRequestModel) client.ArtifactResourceRequest {
	req := client.ArtifactResourceRequest{
		CPU:    rr.CPU.ValueFloat64(),
		Memory: rr.Memory.ValueInt64(),
	}
	if !rr.GPU.IsNull() && !rr.GPU.IsUnknown() {
		gpu := rr.GPU.ValueInt64()
		req.GPU = &gpu
	}
	if !rr.GPUType.IsNull() && !rr.GPUType.IsUnknown() {
		gpuType := rr.GPUType.ValueString()
		req.GPUType = &gpuType
	}
	return req
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
	data.ID = types.StringValue(artifact.ID)
	data.Name = types.StringValue(artifact.Name)
	if artifact.Description != "" {
		data.Description = types.StringValue(artifact.Description)
	} else if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}
	data.Type = types.StringValue(string(artifact.Type))

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
			var priorDescription types.String
			if prior != nil && i < len(prior.ContainerGroups) && j < len(prior.ContainerGroups[i].Containers) {
				priorDescription = prior.ContainerGroups[i].Containers[j].Description
			}
			containers[j] = loadContainerFromAPI(c, priorDescription)
		}
		groups[i] = ArtifactContainerGroupModel{Containers: containers}
	}
	return ArtifactSpecModel{ContainerGroups: groups}
}

func loadContainerFromAPI(c client.ArtifactContainer, priorDescription types.String) ArtifactContainerModel {
	model := ArtifactContainerModel{
		ImageURI: types.StringValue(c.ImageURI),
		ResourceRequest: ArtifactResourceRequestModel{
			CPU:    types.Float64Value(c.ResourceRequest.CPU),
			Memory: types.Int64Value(c.ResourceRequest.Memory),
		},
	}

	if c.Name != nil {
		model.Name = types.StringValue(*c.Name)
	} else {
		model.Name = types.StringNull()
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

	if c.ResourceRequest.GPU != nil && *c.ResourceRequest.GPU != 0 {
		model.ResourceRequest.GPU = types.Int64Value(*c.ResourceRequest.GPU)
	} else {
		model.ResourceRequest.GPU = types.Int64Null()
	}

	if c.ResourceRequest.GPUType != nil {
		model.ResourceRequest.GPUType = types.StringValue(*c.ResourceRequest.GPUType)
	} else {
		model.ResourceRequest.GPUType = types.StringNull()
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
			model.EnvironmentVars[i] = ArtifactEnvironmentVariableModel{
				Name:  types.StringValue(ev.Name),
				Value: types.StringValue(ev.Value),
			}
		}
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
