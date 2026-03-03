package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ArtifactResource{}
var _ resource.ResourceWithImportState = &ArtifactResource{}

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
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workload API Artifact",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Artifact ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Artifact name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Artifact description.",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(client.ArtifactStatusDraft)),
				MarkdownDescription: "Artifact status. Draft artifacts are mutable; registered artifacts are immutable.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(client.ArtifactStatusDraft),
						string(client.ArtifactStatusRegistered),
					),
				},
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(client.ArtifactTypeGeneric)),
				MarkdownDescription: "Artifact type.",
			},
			"artifact_collection_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Artifact collection ID this artifact belongs to (for versioning support).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Artifact version number (assigned when registered).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp (RFC3339).",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last update timestamp (RFC3339).",
			},
			"creator": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Owner user details including ID, username and email.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Creator user ID.",
					},
					"username": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Creator username.",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Creator email address.",
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Artifact specification (containers, probes, and runtime metadata).",
				Attributes: map[string]schema.Attribute{
					"container_groups": schema.ListNestedAttribute{
						Optional:            true,
						MarkdownDescription: "List of container groups.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"containers": schema.ListNestedAttribute{
									Optional:            true,
									MarkdownDescription: "List of containers in this group.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"image_uri": schema.StringAttribute{
												Required:            true,
												MarkdownDescription: "Docker image URI.",
											},
											"resource_request": schema.SingleNestedAttribute{
												Required:            true,
												MarkdownDescription: "Resources required by this container.",
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
											"name": schema.StringAttribute{
												Optional:            true,
												MarkdownDescription: "Container name.",
											},
											"description": schema.StringAttribute{
												Optional:            true,
												MarkdownDescription: "Container description.",
											},
											"entrypoint": schema.ListAttribute{
												Optional:            true,
												MarkdownDescription: "Container entrypoint.",
												ElementType:         types.StringType,
											},
											"environment_vars": schema.ListNestedAttribute{
												Optional:            true,
												MarkdownDescription: "Environment variables.",
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"name": schema.StringAttribute{
															Required:            true,
															MarkdownDescription: "Environment variable name.",
														},
														"value": schema.StringAttribute{
															Required:            true,
															MarkdownDescription: "Environment variable value.",
														},
													},
												},
											},
											"port": schema.Int64Attribute{
												Optional:            true,
												MarkdownDescription: "Container access port (>= 1024).",
											},
											"primary": schema.BoolAttribute{
												Optional:            true,
												MarkdownDescription: "Whether this is the primary container.",
											},
											"liveness_probe":  artifactProbeSchema(),
											"readiness_probe": artifactProbeSchema(),
											"startup_probe":   artifactProbeSchema(),
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

func artifactProbeSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "HTTP probe configuration.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "URL path to query for health check.",
			},
			"scheme": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("HTTP"),
				MarkdownDescription: "Scheme to use for connecting (HTTP/HTTPS).",
			},
			"port": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(8080),
				MarkdownDescription: "Port to access on the container.",
			},
			"host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Host name to connect to (defaults to the pod IP).",
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
				MarkdownDescription: "Probe timeout in seconds.",
			},
			"period_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
				MarkdownDescription: "Probe period in seconds.",
			},
			"initial_delay_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
				MarkdownDescription: "Initial delay in seconds.",
			},
			"failure_threshold": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
				MarkdownDescription: "Minimum consecutive failures for the probe to be considered failed.",
			},
			"http_headers": schema.MapAttribute{
				Optional:            true,
				MarkdownDescription: "HTTP headers for the probe.",
				ElementType:         types.StringType,
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
	created, err := r.provider.service.CreateArtifact(ctx, artifactToClientInput(data))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Artifact", err.Error())
		return
	}

	state, diags := artifactFromClient(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
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
				fmt.Sprintf("Artifact with ID %s is not found. Removing from state.", data.ID.ValueString()),
			)
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Artifact with ID %s", data.ID.ValueString()),
				err.Error(),
			)
		}
		return
	}

	state, diags := artifactFromClient(ctx, artifact)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ArtifactResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateArtifact")
	updated, err := r.provider.service.UpdateArtifact(ctx, plan.ID.ValueString(), artifactToClientInput(plan))
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Artifact not found",
				fmt.Sprintf("Artifact with ID %s is not found. Removing from state.", plan.ID.ValueString()),
			)
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Artifact", err.Error())
		}
		return
	}

	state, diags := artifactFromClient(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ArtifactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ArtifactResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteArtifact")
	err := r.provider.service.DeleteArtifact(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Artifact", err.Error())
			return
		}
	}
}

func (r *ArtifactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func artifactToClientInput(tf ArtifactResourceModel) *client.InputArtifact {
	req := &client.InputArtifact{
		Name: tf.Name.ValueString(),
		Spec: artifactSpecToClient(tf.Spec),
	}

	if IsKnown(tf.Description) {
		req.Description = tf.Description.ValueString()
	}
	if IsKnown(tf.Status) {
		req.Status = client.ArtifactStatus(tf.Status.ValueString())
	}
	if IsKnown(tf.Type) {
		req.Type = client.ArtifactType(tf.Type.ValueString())
	}
	req.ArtifactCollectionID = StringValuePointerOptional(tf.ArtifactCollectionID)

	return req
}

func artifactSpecToClient(spec ArtifactSpecModel) client.ArtifactSpecInput {
	out := client.ArtifactSpecInput{}
	if len(spec.ContainerGroups) == 0 {
		return out
	}

	out.ContainerGroups = make([]client.ContainerGroupInput, 0, len(spec.ContainerGroups))
	for _, group := range spec.ContainerGroups {
		cg := client.ContainerGroupInput{}
		if len(group.Containers) > 0 {
			cg.Containers = make([]client.Container, 0, len(group.Containers))
			for _, c := range group.Containers {
				cc := client.Container{
					ImageURI:        c.ImageURI.ValueString(),
					ResourceRequest: resourceRequestToClient(c.ResourceRequest),
					Description:     "",
				}
				if IsKnown(c.Description) {
					cc.Description = c.Description.ValueString()
				}
				cc.Name = StringValuePointerOptional(c.Name)
				cc.Port = Int64ValuePointerOptional(c.Port)
				cc.Primary = BoolValuePointerOptional(c.Primary)

				if len(c.Entrypoint) > 0 {
					entrypoint := make([]string, 0, len(c.Entrypoint))
					for _, e := range c.Entrypoint {
						if IsKnown(e) {
							entrypoint = append(entrypoint, e.ValueString())
						}
					}
					cc.Entrypoint = entrypoint
				}

				if len(c.EnvironmentVars) > 0 {
					env := make([]client.EnvironmentVariable, 0, len(c.EnvironmentVars))
					for _, v := range c.EnvironmentVars {
						if !IsKnown(v.Name) || !IsKnown(v.Value) {
							continue
						}
						env = append(env, client.EnvironmentVariable{
							Name:  v.Name.ValueString(),
							Value: v.Value.ValueString(),
						})
					}
					cc.EnvironmentVars = env
				}

				cc.LivenessProbe = probeToClient(c.LivenessProbe)
				cc.ReadinessProbe = probeToClient(c.ReadinessProbe)
				cc.StartupProbe = probeToClient(c.StartupProbe)

				cg.Containers = append(cg.Containers, cc)
			}
		}
		out.ContainerGroups = append(out.ContainerGroups, cg)
	}

	return out
}

func resourceRequestToClient(tf ArtifactResourceRequestModel) client.ResourceRequest {
	out := client.ResourceRequest{
		CPU:    tf.CPU.ValueFloat64(),
		Memory: tf.Memory.ValueInt64(),
	}
	if IsKnown(tf.GPU) {
		gpu := tf.GPU.ValueInt64()
		out.GPU = &gpu
	}
	out.GPUType = StringValuePointerOptional(tf.GPUType)
	return out
}

func probeToClient(tf *ArtifactProbeConfigModel) *client.ProbeConfig {
	if tf == nil {
		return nil
	}

	out := &client.ProbeConfig{
		Path: tf.Path.ValueString(),
	}
	if IsKnown(tf.Scheme) {
		out.Scheme = tf.Scheme.ValueString()
	}
	if IsKnown(tf.Port) {
		port := tf.Port.ValueInt64()
		out.Port = &port
	}
	out.Host = StringValuePointerOptional(tf.Host)
	if IsKnown(tf.TimeoutSeconds) {
		timeout := tf.TimeoutSeconds.ValueInt64()
		out.TimeoutSeconds = &timeout
	}
	if IsKnown(tf.PeriodSeconds) {
		period := tf.PeriodSeconds.ValueInt64()
		out.PeriodSeconds = &period
	}
	if IsKnown(tf.InitialDelaySeconds) {
		delay := tf.InitialDelaySeconds.ValueInt64()
		out.InitialDelaySeconds = &delay
	}
	if IsKnown(tf.FailureThreshold) {
		failure := tf.FailureThreshold.ValueInt64()
		out.FailureThreshold = &failure
	}
	if IsKnown(tf.HTTPHeaders) {
		out.HTTPHeaders = convertTfStringMap(tf.HTTPHeaders)
	}
	return out
}

func artifactFromClient(ctx context.Context, a *client.ArtifactFormatted) (ArtifactResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	out := ArtifactResourceModel{
		ID:                   types.StringValue(a.ID),
		Name:                 types.StringValue(a.Name),
		Status:               types.StringValue(string(a.Status)),
		Type:                 types.StringValue(string(a.Type)),
		ArtifactCollectionID: StringPointerValue(a.ArtifactCollectionID),
		Version:              types.Int64Value(a.Version),
		CreatedAt:            types.StringValue(a.CreatedAt),
		UpdatedAt:            types.StringValue(a.UpdatedAt),
	}

	if a.Description != "" {
		out.Description = types.StringValue(a.Description)
	} else {
		out.Description = types.StringNull()
	}

	creatorObj, creatorDiags := creatorObjectValueFromClient(ctx, a.Creator)
	diags.Append(creatorDiags...)
	out.Creator = creatorObj

	out.Spec = artifactSpecFromClient(ctx, a.Spec, &diags)
	return out, diags
}

func artifactSpecFromClient(ctx context.Context, spec client.ArtifactSpecOutput, diags *diag.Diagnostics) ArtifactSpecModel {
	out := ArtifactSpecModel{
		ContainerGroups: make([]ArtifactContainerGroupModel, 0),
	}
	if len(spec.ContainerGroups) == 0 {
		return out
	}

	out.ContainerGroups = make([]ArtifactContainerGroupModel, 0, len(spec.ContainerGroups))
	for _, cg := range spec.ContainerGroups {
		group := ArtifactContainerGroupModel{
			Containers: make([]ArtifactContainerModel, 0),
		}
		if len(cg.Containers) > 0 {
			group.Containers = make([]ArtifactContainerModel, 0, len(cg.Containers))
			for _, c := range cg.Containers {
				container := ArtifactContainerModel{
					ImageURI: types.StringValue(c.ImageURI),
					ResourceRequest: ArtifactResourceRequestModel{
						CPU:     types.Float64Value(c.ResourceRequest.CPU),
						Memory:  types.Int64Value(c.ResourceRequest.Memory),
						GPU:     types.Int64Null(),
						GPUType: StringPointerValue(c.ResourceRequest.GPUType),
					},
					Name:        StringPointerValue(c.Name),
					Description: types.StringNull(),
					Port:        Int64PointerValue(c.Port),
					Primary:     BoolPointerValue(c.Primary),
				}

				if c.Description != "" {
					container.Description = types.StringValue(c.Description)
				}
				if c.ResourceRequest.GPU != nil {
					container.ResourceRequest.GPU = types.Int64Value(*c.ResourceRequest.GPU)
				}

				if len(c.Entrypoint) > 0 {
					container.Entrypoint = convertToTfStringList(c.Entrypoint)
				} else {
					container.Entrypoint = []types.String{}
				}

				if len(c.EnvironmentVars) > 0 {
					env := make([]ArtifactEnvironmentVarModel, 0, len(c.EnvironmentVars))
					for _, v := range c.EnvironmentVars {
						env = append(env, ArtifactEnvironmentVarModel{
							Name:  types.StringValue(v.Name),
							Value: types.StringValue(v.Value),
						})
					}
					container.EnvironmentVars = env
				} else {
					container.EnvironmentVars = []ArtifactEnvironmentVarModel{}
				}

				container.LivenessProbe = probeFromClient(ctx, c.LivenessProbe, diags)
				container.ReadinessProbe = probeFromClient(ctx, c.ReadinessProbe, diags)
				container.StartupProbe = probeFromClient(ctx, c.StartupProbe, diags)

				group.Containers = append(group.Containers, container)
			}
		}
		out.ContainerGroups = append(out.ContainerGroups, group)
	}

	return out
}

func probeFromClient(ctx context.Context, p *client.ProbeConfig, diags *diag.Diagnostics) *ArtifactProbeConfigModel {
	_ = ctx
	if p == nil {
		return nil
	}

	tf := &ArtifactProbeConfigModel{
		Path:                types.StringValue(p.Path),
		Scheme:              types.StringNull(),
		Port:                types.Int64Null(),
		Host:                StringPointerValue(p.Host),
		TimeoutSeconds:      types.Int64Null(),
		PeriodSeconds:       types.Int64Null(),
		InitialDelaySeconds: types.Int64Null(),
		FailureThreshold:    types.Int64Null(),
		HTTPHeaders:         types.MapNull(types.StringType),
	}

	if p.Scheme != "" {
		tf.Scheme = types.StringValue(p.Scheme)
	}
	if p.Port != nil {
		tf.Port = types.Int64Value(*p.Port)
	}
	if p.TimeoutSeconds != nil {
		tf.TimeoutSeconds = types.Int64Value(*p.TimeoutSeconds)
	}
	if p.PeriodSeconds != nil {
		tf.PeriodSeconds = types.Int64Value(*p.PeriodSeconds)
	}
	if p.InitialDelaySeconds != nil {
		tf.InitialDelaySeconds = types.Int64Value(*p.InitialDelaySeconds)
	}
	if p.FailureThreshold != nil {
		tf.FailureThreshold = types.Int64Value(*p.FailureThreshold)
	}

	if p.HTTPHeaders != nil {
		m, d := types.MapValueFrom(ctx, types.StringType, p.HTTPHeaders)
		diags.Append(d...)
		tf.HTTPHeaders = m
	}

	return tf
}
