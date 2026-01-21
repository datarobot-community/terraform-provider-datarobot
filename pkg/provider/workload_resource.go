package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkloadResource{}
var _ resource.ResourceWithImportState = &WorkloadResource{}
var _ resource.ResourceWithConfigValidators = &WorkloadResource{}

func NewWorkloadResource() resource.Resource {
	return &WorkloadResource{}
}

type WorkloadResource struct {
	provider *Provider
}

func (r *WorkloadResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

func (r *WorkloadResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("artifact_id"),
			path.MatchRoot("artifact"),
		),
	}
}

func (r *WorkloadResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workload API Workload",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workload ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Workload name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"artifact_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Existing Artifact ID to run.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"artifact": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Inline artifact spec to create and run.",
				Attributes: map[string]schema.Attribute{
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
						MarkdownDescription: "Artifact status (draft/registered). Inline artifacts are typically draft.",
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
						MarkdownDescription: "Artifact collection ID (for versioning support).",
					},
					"spec": schema.SingleNestedAttribute{
						Required:            true,
						MarkdownDescription: "Artifact specification.",
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"runtime": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Runtime for the workload.",
				Attributes: map[string]schema.Attribute{
					"replica_count": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Number of replicas to spawn for the workload.",
					},
					"resources": schema.ListNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Resources for the workload (currently resource bundles).",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString("resource_bundle"),
									MarkdownDescription: "Resource type discriminator (currently only resource_bundle).",
								},
								"resource_bundle_id": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Bundle ID for the resources.",
								},
								"gpu_maker": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "GPU vendor/maker (populated by the system).",
								},
								"gpu_type_label": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "GPU type label for the bundle (populated by the system).",
								},
							},
						},
					},
					"autoscaling": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Autoscaling configuration for a workload.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:            true,
								Computed:            true,
								MarkdownDescription: "Whether autoscaling is enabled.",
							},
							"policies": schema.ListNestedAttribute{
								Required:            true,
								MarkdownDescription: "Autoscaling policies.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"scaling_metric": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "Metric used for scaling decisions.",
											Validators: []validator.String{
												stringvalidator.OneOf(
													string(client.ScalingMetricTypeCPUAverageUtilization),
													string(client.ScalingMetricTypeHTTPRequestsConcurrency),
												),
											},
										},
										"target": schema.Float64Attribute{
											Required:            true,
											MarkdownDescription: "Target value for the scaling metric.",
										},
										"min_count": schema.Int64Attribute{
											Required:            true,
											MarkdownDescription: "Minimum number of replicas.",
										},
										"max_count": schema.Int64Attribute{
											Required:            true,
											MarkdownDescription: "Maximum number of replicas.",
										},
										"priority": schema.Int64Attribute{
											Optional:            true,
											MarkdownDescription: "Policy priority when multiple policies are defined.",
										},
									},
								},
							},
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"running": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the workload should be started (calls /start or /stop).",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			// Computed fields
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workload status.",
			},
			"endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API endpoint to use to send service requests.",
			},
			"internal_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal URL for this workload.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp (RFC3339).",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last update timestamp (RFC3339).",
			},
			"running_since": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the workload entered RUNNING status.",
			},
			"creator": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Owner user details including ID, username and email.",
				Attributes: map[string]schema.Attribute{
					"id":       schema.StringAttribute{Computed: true},
					"username": schema.StringAttribute{Computed: true},
					"email":    schema.StringAttribute{Computed: true},
				},
			},
			"status_details": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Workload status details (conditions and log tail).",
				Attributes: map[string]schema.Attribute{
					"conditions": schema.ListNestedAttribute{
						Computed:            true,
						MarkdownDescription: "Parsed conditions from the latest workload status.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type":  schema.StringAttribute{Computed: true},
								"name":  schema.StringAttribute{Computed: true},
								"value": schema.StringAttribute{Computed: true},
							},
						},
					},
					"log_tail": schema.ListAttribute{
						Computed:            true,
						MarkdownDescription: "Tail of container logs from the latest workload status during startup.",
						ElementType:         types.StringType,
					},
				},
			},
		},
	}
}

func (r *WorkloadResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkloadResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateWorkload")
	created, err := r.provider.service.CreateWorkload(ctx, workloadToClientCreateRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Workload", err.Error())
		return
	}

	// Start if requested. Workload is created as SUBMITTED.
	if IsKnown(data.Running) && data.Running.ValueBool() {
		traceAPICall("StartWorkload")
		if err := r.provider.service.StartWorkload(ctx, created.ID); err != nil {
			resp.Diagnostics.AddError("Error starting Workload", err.Error())
			return
		}
	}

	traceAPICall("GetWorkload")
	latest, err := r.provider.service.GetWorkload(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Workload after create", err.Error())
		return
	}

	state, diags := workloadFromClient(ctx, latest)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Preserve desired running flag in state
	state.Running = data.Running
	// Keep config fields from plan (API may not return them reliably).
	// This also prevents "inconsistent result after apply" when the API
	// injects default runtime resources not specified in the config.
	state.Artifact = data.Artifact
	state.ArtifactID = data.ArtifactID
	state.Runtime = data.Runtime
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *WorkloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkloadResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetWorkload")
	w, err := r.provider.service.GetWorkload(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Workload not found",
				fmt.Sprintf("Workload with ID %s is not found. Removing from state.", data.ID.ValueString()),
			)
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Workload with ID %s", data.ID.ValueString()),
				err.Error(),
			)
		}
		return
	}

	state, diags := workloadFromClient(ctx, w)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keep the user's desired running flag (do not override with remote state).
	state.Running = data.Running
	// Keep config fields from prior state (API does not return them).
	state.Artifact = data.Artifact
	state.ArtifactID = data.ArtifactID
	state.Runtime = data.Runtime
	state.Name = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state WorkloadResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only supported in-place update is start/stop via running flag.
	if IsKnown(plan.Running) && IsKnown(state.Running) && plan.Running.ValueBool() != state.Running.ValueBool() {
		if plan.Running.ValueBool() {
			traceAPICall("StartWorkload")
			if err := r.provider.service.StartWorkload(ctx, state.ID.ValueString()); err != nil {
				resp.Diagnostics.AddError("Error starting Workload", err.Error())
				return
			}
		} else {
			traceAPICall("StopWorkload")
			if err := r.provider.service.StopWorkload(ctx, state.ID.ValueString()); err != nil {
				resp.Diagnostics.AddError("Error stopping Workload", err.Error())
				return
			}
		}
	}

	// Refresh computed fields
	traceAPICall("GetWorkload")
	w, err := r.provider.service.GetWorkload(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Workload not found",
				fmt.Sprintf("Workload with ID %s is not found. Removing from state.", state.ID.ValueString()),
			)
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error reading Workload after update", err.Error())
		}
		return
	}

	newState, diags := workloadFromClient(ctx, w)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve desired running flag and immutable config
	newState.Running = plan.Running
	newState.Artifact = plan.Artifact
	newState.ArtifactID = plan.ArtifactID
	newState.Runtime = plan.Runtime
	newState.Name = plan.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *WorkloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkloadResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteWorkload")
	err := r.provider.service.DeleteWorkload(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Workload", err.Error())
			return
		}
	}
}

func (r *WorkloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func workloadToClientCreateRequest(tf WorkloadResourceModel) *client.CreateWorkloadRequest {
	req := &client.CreateWorkloadRequest{
		Runtime: workloadRuntimeToClient(tf.Runtime),
	}

	if IsKnown(tf.Name) {
		name := tf.Name.ValueString()
		req.Name = &name
	}

	if IsKnown(tf.ArtifactID) {
		id := tf.ArtifactID.ValueString()
		req.ArtifactID = &id
	}

	if tf.Artifact != nil {
		req.Artifact = workloadInlineArtifactToClient(*tf.Artifact)
	}

	return req
}

func workloadInlineArtifactToClient(tf WorkloadInlineArtifactModel) *client.InputArtifact {
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

func workloadRuntimeToClient(tf WorkloadRuntimeModel) client.WorkloadRuntime {
	out := client.WorkloadRuntime{}
	out.ReplicaCount = Int64ValuePointerOptional(tf.ReplicaCount)

	// resources
	if len(tf.Resources) > 0 {
		out.Resources = make([]client.ResourceBundleResources, 0, len(tf.Resources))
		for _, r := range tf.Resources {
			if !IsKnown(r.ResourceBundleID) {
				continue
			}
			out.Resources = append(out.Resources, client.ResourceBundleResources{
				ResourceBundleID: r.ResourceBundleID.ValueString(),
				Type:             "resource_bundle",
			})
		}
	}

	// autoscaling
	if tf.Autoscaling != nil {
		a := tf.Autoscaling
		policies := make([]client.AutoscalingPolicy, 0, len(a.Policies))
		for _, p := range a.Policies {
			policy := client.AutoscalingPolicy{
				ScalingMetric: client.ScalingMetricType(p.ScalingMetric.ValueString()),
				Target:        p.Target.ValueFloat64(),
				MinCount:      p.MinCount.ValueInt64(),
				MaxCount:      p.MaxCount.ValueInt64(),
				Priority:      Int64ValuePointerOptional(p.Priority),
			}
			policies = append(policies, policy)
		}
		out.Autoscaling = &client.AutoscalingPropertiesInput{
			Enabled:  (!a.Enabled.IsNull() && !a.Enabled.IsUnknown() && a.Enabled.ValueBool()) || a.Enabled.IsNull() || a.Enabled.IsUnknown(),
			Policies: policies,
		}
		if IsKnown(a.Enabled) {
			out.Autoscaling.Enabled = a.Enabled.ValueBool()
		}
	}

	return out
}

func workloadFromClient(ctx context.Context, w *client.WorkloadFormatted) (WorkloadResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	out := WorkloadResourceModel{
		ID:           types.StringValue(w.ID),
		Name:         types.StringValue(w.Name),
		Status:       types.StringValue(string(w.Status)),
		InternalURL:  types.StringValue(w.InternalURL),
		CreatedAt:    types.StringValue(w.CreatedAt),
		UpdatedAt:    types.StringValue(w.UpdatedAt),
		ArtifactID:   types.StringValue(w.ArtifactID),
		Endpoint:     StringPointerValue(w.Endpoint),
		RunningSince: StringPointerValue(w.RunningSince),
	}

	creatorObj, creatorDiags := creatorObjectValueFromClient(ctx, w.Creator)
	diags.Append(creatorDiags...)
	out.Creator = creatorObj

	// runtime (computed portion)
	out.Runtime = workloadRuntimeFromClient(ctx, w.Runtime, &diags)

	// status details
	sdObj, sdDiags := workloadStatusDetailsObjectValueFromClient(ctx, w.StatusDetails)
	diags.Append(sdDiags...)
	out.StatusDetails = sdObj

	return out, diags
}

func workloadRuntimeFromClient(ctx context.Context, rt client.WorkloadRuntimeFormatted, diags *diag.Diagnostics) WorkloadRuntimeModel {
	out := WorkloadRuntimeModel{
		ReplicaCount: types.Int64Null(),
		Resources:    []WorkloadResourceBundleModel{},
	}
	if rt.ReplicaCount != nil {
		out.ReplicaCount = types.Int64Value(*rt.ReplicaCount)
	}

	if len(rt.Resources) > 0 {
		out.Resources = make([]WorkloadResourceBundleModel, 0, len(rt.Resources))
		for _, r := range rt.Resources {
			out.Resources = append(out.Resources, WorkloadResourceBundleModel{
				ResourceBundleID: types.StringValue(r.ResourceBundleID),
				Type:             types.StringValue(r.Type),
				GPUMaker:         StringPointerValue(r.GPUMaker),
				GPUTypeLabel:     StringPointerValue(r.GPUTypeLabel),
			})
		}
	}

	if rt.Autoscaling != nil {
		tfA := &WorkloadAutoscalingModel{
			Enabled:  types.BoolValue(rt.Autoscaling.Enabled),
			Policies: []WorkloadAutoscalingPolicyModel{},
		}
		if len(rt.Autoscaling.Policies) > 0 {
			tfA.Policies = make([]WorkloadAutoscalingPolicyModel, 0, len(rt.Autoscaling.Policies))
			for _, p := range rt.Autoscaling.Policies {
				tfA.Policies = append(tfA.Policies, WorkloadAutoscalingPolicyModel{
					ScalingMetric: types.StringValue(string(p.ScalingMetric)),
					Target:        types.Float64Value(p.Target),
					MinCount:      types.Int64Value(p.MinCount),
					MaxCount:      types.Int64Value(p.MaxCount),
					Priority:      Int64PointerValue(p.Priority),
				})
			}
		}
		out.Autoscaling = tfA
	}

	_ = ctx
	_ = diags
	return out
}

// status_details mapping is handled by workloadStatusDetailsObjectValueFromClient in utils.go
