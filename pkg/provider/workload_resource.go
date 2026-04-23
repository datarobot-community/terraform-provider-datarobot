package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &WorkloadResource{}
var _ resource.ResourceWithImportState = &WorkloadResource{}
var _ resource.ResourceWithValidateConfig = &WorkloadResource{}

func NewWorkloadResource() resource.Resource {
	return &WorkloadResource{}
}

type WorkloadResource struct {
	provider *Provider
}

func (r *WorkloadResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

func (r *WorkloadResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Workload runs a containerized artifact in the cluster and exposes an inference endpoint.\n\n" +
			"Several attributes (including `runtime` and `artifact_id`) trigger replacement when changed. " +
			"To avoid downtime during replacements, it is recommended to set `create_before_destroy` in the resource lifecycle:\n\n" +
			"```hcl\n" +
			"lifecycle {\n" +
			"  create_before_destroy = true\n" +
			"}\n" +
			"```",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Workload.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Workload.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A human-readable description of the Workload.",
			},
			"importance": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("low"),
				MarkdownDescription: "Priority level for the Workload: `critical`, `high`, `moderate`, or `low`. Defaults to `low`.",
			},
			"artifact_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the Artifact to deploy. Changing this value forces a new Workload to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The inference endpoint URL for the Workload.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current status of the Workload: `unknown`, `submitted`, `initializing`, `running`, `stopping`, `stopped`, or `errored`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"runtime": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Runtime configuration for the Workload.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"replica_count": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Number of replicas to run. Cannot be used together with `autoscaling`. Omitting this field retains the current value. Set to `0` to explicitly clear it (e.g. when switching to autoscaling).",
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"autoscaling": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Autoscaling configuration. When set, takes precedence over replica_count.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:            true,
								Computed:            true,
								MarkdownDescription: "Whether autoscaling is enabled. Defaults to true.",
							},
							"policies": schema.ListNestedAttribute{
								Required:            true,
								MarkdownDescription: "Scaling policies that define when and how to scale.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"scaling_metric": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "Metric used for scaling decisions: `cpuAverageUtilization`, `httpRequestsConcurrency`, `gpuCacheUtilization`, or `gpuRequestQueueDepth`.",
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
											Computed:            true,
											MarkdownDescription: "Policy priority when multiple policies are defined.",
											PlanModifiers: []planmodifier.Int64{
												int64planmodifier.UseStateForUnknown(),
											},
										},
									},
								},
							},
						},
					},
					"resources": schema.ListNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Resource bundles assigned to the Workload. When empty the server infers an appropriate bundle.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"resource_bundle_id": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "ID of the resource bundle (e.g. `cpu.nano`).",
								},
							},
						},
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
	workload, err := r.provider.service.CreateWorkload(ctx, workloadCreateRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Workload", err.Error())
		return
	}

	workload, err = waitForWorkloadToBeRunning(ctx, r.provider.service, workload.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Workload to start", err.Error())
		return
	}

	plannedRuntime := data.Runtime
	loadWorkloadIntoModel(workload, &data)
	applyRuntimeSentinels(plannedRuntime, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
	workload, err := r.provider.service.GetWorkload(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Workload not found",
				fmt.Sprintf("Workload with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Workload with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	priorRuntime := data.Runtime
	loadWorkloadIntoModel(workload, &data)
	applyRuntimeSentinels(priorRuntime, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state WorkloadResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	if workloadMetadataChanged(plan, state) {
		traceAPICall("UpdateWorkload")
		workload, err := r.provider.service.UpdateWorkload(ctx, id, workloadUpdateRequest(plan))
		if err != nil {
			resp.Diagnostics.AddError("Error updating Workload", err.Error())
			return
		}
		loadWorkloadIntoModel(workload, &plan)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
		if _, ok := err.(*client.NotFoundError); !ok {
			resp.Diagnostics.AddError("Error deleting Workload", err.Error())
		}
	}
}

func (r *WorkloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *WorkloadResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data WorkloadResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	replicaCountSet := !data.Runtime.ReplicaCount.IsNull() &&
		!data.Runtime.ReplicaCount.IsUnknown() &&
		data.Runtime.ReplicaCount.ValueInt64() != 0

	autoscalingSet := data.Runtime.Autoscaling != nil &&
		!data.Runtime.Autoscaling.Enabled.IsNull() &&
		!data.Runtime.Autoscaling.Enabled.IsUnknown() &&
		data.Runtime.Autoscaling.Enabled.ValueBool()

	if replicaCountSet && autoscalingSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("runtime"),
			"Conflicting runtime configuration",
			"Cannot specify both replica_count and autoscaling. Set replica_count to 0 (or omit it) when using autoscaling or disable autoscaling.",
		)
	}
}

func waitForWorkloadToBeRunning(ctx context.Context, s client.Service, id string) (*client.Workload, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("GetWorkload")
		workload, err := s.GetWorkload(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}
		if workload.Status == client.ProtonStatusErrored {
			return backoff.Permanent(errors.New("workload failed to start, review the workload events for details"))
		}
		if workload.Status != client.ProtonStatusRunning {
			return fmt.Errorf("workload is not running yet (status: %s)", workload.Status)
		}
		return nil
	}

	if err := backoff.Retry(operation, expBackoff); err != nil {
		return nil, err
	}

	traceAPICall("GetWorkload")
	return s.GetWorkload(ctx, id)
}

func workloadCreateRequest(data WorkloadResourceModel) *client.CreateWorkloadRequest {
	req := &client.CreateWorkloadRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Importance:  client.WorkloadImportance(data.Importance.ValueString()),
		Runtime:     workloadRuntimeToClient(data.Runtime),
	}

	artifactID := data.ArtifactID.ValueString()
	req.ArtifactID = &artifactID

	return req
}

func workloadUpdateRequest(data WorkloadResourceModel) *client.UpdateWorkloadRequest {
	req := &client.UpdateWorkloadRequest{}

	name := data.Name.ValueString()
	req.Name = &name

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		req.Description = &desc
	}

	if !data.Importance.IsNull() && !data.Importance.IsUnknown() {
		imp := client.WorkloadImportance(data.Importance.ValueString())
		req.Importance = &imp
	}

	return req
}

func workloadRuntimeToClient(runtime WorkloadRuntimeModel) client.ProtonRuntime {
	r := client.ProtonRuntime{}

	if !runtime.ReplicaCount.IsNull() && !runtime.ReplicaCount.IsUnknown() && runtime.ReplicaCount.ValueInt64() != 0 {
		v := runtime.ReplicaCount.ValueInt64()
		r.ReplicaCount = &v
	}

	if runtime.Autoscaling != nil {
		r.Autoscaling = &client.AutoscalingProperties{}

		if !runtime.Autoscaling.Enabled.IsNull() && !runtime.Autoscaling.Enabled.IsUnknown() {
			enabled := runtime.Autoscaling.Enabled.ValueBool()
			r.Autoscaling.Enabled = &enabled
		}

		r.Autoscaling.Policies = make([]client.AutoscalingPolicy, len(runtime.Autoscaling.Policies))
		for i, p := range runtime.Autoscaling.Policies {
			policy := client.AutoscalingPolicy{
				ScalingMetric: p.ScalingMetric.ValueString(),
				Target:        p.Target.ValueFloat64(),
				MinCount:      p.MinCount.ValueInt64(),
				MaxCount:      p.MaxCount.ValueInt64(),
			}
			if !p.Priority.IsNull() && !p.Priority.IsUnknown() {
				v := p.Priority.ValueInt64()
				policy.Priority = &v
			}
			r.Autoscaling.Policies[i] = policy
		}
	}

	if len(runtime.Resources) > 0 {
		r.Resources = make([]client.ResourceBundleResources, len(runtime.Resources))
		for i, rb := range runtime.Resources {
			r.Resources[i] = client.ResourceBundleResources{
				Type:             "resource_bundle",
				ResourceBundleID: rb.ResourceBundleID.ValueString(),
			}
		}
	}

	return r
}

// applyRuntimeSentinels restores sentinel values from the desired runtime into the
// model after it has been overwritten by API response data. Currently, replica_count=0
// is the sentinel meaning "explicitly cleared" — the API omits the field in that case
// but we must keep 0 in state so subsequent plans show no diff.
func applyRuntimeSentinels(desired WorkloadRuntimeModel, data *WorkloadResourceModel) {
	if !desired.ReplicaCount.IsNull() && !desired.ReplicaCount.IsUnknown() && desired.ReplicaCount.ValueInt64() == 0 {
		data.Runtime.ReplicaCount = desired.ReplicaCount
	}
}

func workloadMetadataChanged(plan, state WorkloadResourceModel) bool {
	return !plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!plan.Importance.Equal(state.Importance)
}


func loadWorkloadIntoModel(workload *client.Workload, data *WorkloadResourceModel) {
	data.ID = types.StringValue(workload.ID)
	data.Name = types.StringValue(workload.Name)
	data.Status = types.StringValue(string(workload.Status))
	data.Importance = types.StringValue(string(workload.Importance))

	if workload.Description != "" {
		data.Description = types.StringValue(workload.Description)
	} else {
		data.Description = types.StringNull()
	}

	if workload.ArtifactID != nil {
		data.ArtifactID = types.StringValue(*workload.ArtifactID)
	}

	if workload.Endpoint != nil {
		data.Endpoint = types.StringValue(*workload.Endpoint)
	} else {
		data.Endpoint = types.StringNull()
	}

	data.Runtime = loadWorkloadRuntimeFromAPI(workload.Runtime)
}

func loadWorkloadRuntimeFromAPI(runtime client.ProtonRuntimeFormatted) WorkloadRuntimeModel {
	model := WorkloadRuntimeModel{}

	if runtime.ReplicaCount != nil {
		model.ReplicaCount = types.Int64Value(*runtime.ReplicaCount)
	} else {
		model.ReplicaCount = types.Int64Null()
	}

	if runtime.Autoscaling != nil {
		autoscaling := &WorkloadAutoscalingModel{}

		if runtime.Autoscaling.Enabled != nil {
			autoscaling.Enabled = types.BoolValue(*runtime.Autoscaling.Enabled)
		} else {
			autoscaling.Enabled = types.BoolNull()
		}

		autoscaling.Policies = make([]WorkloadAutoscalingPolicyModel, len(runtime.Autoscaling.Policies))
		for i, p := range runtime.Autoscaling.Policies {
			policy := WorkloadAutoscalingPolicyModel{
				ScalingMetric: types.StringValue(p.ScalingMetric),
				Target:        types.Float64Value(p.Target),
				MinCount:      types.Int64Value(p.MinCount),
				MaxCount:      types.Int64Value(p.MaxCount),
			}
			if p.Priority != nil {
				policy.Priority = types.Int64Value(*p.Priority)
			} else {
				policy.Priority = types.Int64Null()
			}
			autoscaling.Policies[i] = policy
		}

		model.Autoscaling = autoscaling
	}

	if len(runtime.Resources) > 0 {
		model.Resources = make([]WorkloadResourceBundleModel, len(runtime.Resources))
		for i, rb := range runtime.Resources {
			model.Resources[i] = WorkloadResourceBundleModel{
				ResourceBundleID: types.StringValue(rb.ResourceBundleID),
			}
		}
	}

	return model
}
