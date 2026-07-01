package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
				MarkdownDescription: "ID of the Artifact version to deploy. When using `datarobot_artifact`, reference `datarobot_artifact.<name>.artifact_id` (not `.id`). Changing this value forces a new Workload to be created.",
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
					"container_groups": schema.ListNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Per-group runtime configuration.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Container group name (server-assigned, always `default`).",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"replica_count": schema.Int64Attribute{
									Optional:            true,
									Computed:            true,
									MarkdownDescription: "Number of replicas. Cannot be set alongside `autoscaling.enabled=true`. Set to `0` to explicitly clear it.",
									PlanModifiers: []planmodifier.Int64{
										int64planmodifier.UseStateForUnknown(),
									},
								},
								"autoscaling": schema.SingleNestedAttribute{
									Optional:            true,
									MarkdownDescription: "Autoscaling configuration. When set, takes precedence over `replica_count`.",
									Attributes: map[string]schema.Attribute{
										"enabled": schema.BoolAttribute{
											Optional:            true,
											Computed:            true,
											MarkdownDescription: "Whether autoscaling is enabled. Defaults to true.",
											PlanModifiers: []planmodifier.Bool{
												boolplanmodifier.UseStateForUnknown(),
											},
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
								"resource_bundles": schema.ListAttribute{
									Optional:            true,
									ElementType:         types.StringType,
									MarkdownDescription: "Ordered list of resource bundle IDs. One is selected at scheduling time.",
								},
								"bundle_selection_policy": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString("availability"),
									MarkdownDescription: "How to select among `resource_bundles`. Defaults to `availability`.",
								},
								"containers": schema.ListNestedAttribute{
									Optional:            true,
									MarkdownDescription: "Per-container resource allocation overrides.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Required:            true,
												MarkdownDescription: "Container name. Must match a container declared in the artifact group.",
											},
											"resource_allocation": schema.SingleNestedAttribute{
												Optional:            true,
												MarkdownDescription: "Resource allocation for this container.",
												Attributes: map[string]schema.Attribute{
													"cpu": schema.Float64Attribute{
														Optional:            true,
														MarkdownDescription: "CPU cores allocated to this container.",
													},
													"gpu": schema.Float64Attribute{
														Optional:            true,
														MarkdownDescription: "GPUs allocated to this container.",
													},
													"gpu_memory": schema.StringAttribute{
														Optional:            true,
														MarkdownDescription: "GPU VRAM allocated. Accepts human-readable strings (e.g. `\"15GB\"`, `\"512MB\"`, `\"4096Mi\"`) or raw byte integers. 1000-based suffixes: KB, MB, GB, TB. 1024-based suffixes: Ki/KiB, Mi/MiB, Gi/GiB.",
														Validators: []validator.String{
															memoryStringValidator{},
														},
													},
													"memory": schema.StringAttribute{
														Optional:            true,
														MarkdownDescription: "RAM allocated. Accepts human-readable strings (e.g. `\"8GB\"`, `\"512MB\"`, `\"4096Mi\"`) or raw byte integers. 1000-based suffixes: KB, MB, GB, TB. 1024-based suffixes: Ki/KiB, Mi/MiB, Gi/GiB.",
														Validators: []validator.String{
															memoryStringValidator{},
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

	workload, err = waitForWorkloadToBeRunning(ctx, r.provider.service, workload.ID, r.provider.service.BaseURL)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Workload to start", err.Error())
		return
	}

	planned := data
	loadWorkloadIntoModel(workload, &data)
	applySentinels(planned, &data)
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

	prior := data
	loadWorkloadIntoModel(workload, &data)
	applySentinels(prior, &data)
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
		planned := plan
		traceAPICall("UpdateWorkload")
		workload, err := r.provider.service.UpdateWorkload(ctx, id, workloadUpdateRequest(plan))
		if err != nil {
			resp.Diagnostics.AddError("Error updating Workload", err.Error())
			return
		}
		loadWorkloadIntoModel(workload, &plan)
		applySentinels(planned, &plan)
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
		return
	}

	if err := waitForWorkloadToBeDeleted(ctx, r.provider.service, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error waiting for Workload to be deleted", err.Error())
	}
}

func (r *WorkloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	traceAPICall("GetWorkload")
	workload, err := r.provider.service.GetWorkload(ctx, req.ID)
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddError("Workload not found",
				fmt.Sprintf("Workload with ID %s is not found.", req.ID))
		} else {
			resp.Diagnostics.AddError("Error importing Workload", err.Error())
		}
		return
	}

	var data WorkloadResourceModel
	data.ID = types.StringValue(req.ID)
	loadWorkloadIntoModel(workload, &data)
	if data.Description.ValueString() == "" {
		data.Description = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data WorkloadResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.Runtime.ContainerGroups) > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("runtime").AtName("container_groups"),
			"Too many container groups",
			"Currently, Workload API supports only 1 container group.",
		)
		return
	}

	for i, g := range data.Runtime.ContainerGroups {
		replicaCountSet := !g.ReplicaCount.IsNull() &&
			!g.ReplicaCount.IsUnknown() &&
			g.ReplicaCount.ValueInt64() != 0

		autoscalingSet := g.Autoscaling != nil &&
			!g.Autoscaling.Enabled.IsNull() &&
			!g.Autoscaling.Enabled.IsUnknown() &&
			g.Autoscaling.Enabled.ValueBool()

		if replicaCountSet && autoscalingSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("runtime").AtName("container_groups").AtListIndex(i),
				"Conflicting runtime configuration",
				"Cannot specify both replica_count and autoscaling. Set replica_count to 0 (or omit it) when using autoscaling or disable autoscaling.",
			)
		}

		if len(g.ResourceBundles) == 0 {
			if len(g.Containers) == 0 {
				resp.Diagnostics.AddAttributeError(
					path.Root("runtime").AtName("container_groups").AtListIndex(i).AtName("containers"),
					"Missing containers",
					"At least one container must be defined when resource_bundles is not set.",
				)
			}
			for j, c := range g.Containers {
				if c.ResourceAllocation == nil {
					resp.Diagnostics.AddAttributeError(
						path.Root("runtime").AtName("container_groups").AtListIndex(i).AtName("containers").AtListIndex(j).AtName("resource_allocation"),
						"Missing resource configuration",
						"resource_allocation is required for each container when resource_bundles is not set.",
					)
				}
			}
		}
	}
}

func waitForWorkloadToBeRunning(ctx context.Context, s client.Service, id string, baseURL func() string) (*client.Workload, error) {
	expBackoff := getExponentialBackoff()

	var running *client.Workload
	operation := func() error {
		traceAPICall("GetWorkload")
		workload, err := s.GetWorkload(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}
		if workload.Status == client.ProtonStatusErrored {
			logsURL := baseURL() + "/console-nextgen/workloads/" + id + "/activity-log/otel-logs"
			return backoff.Permanent(fmt.Errorf("workload failed to start, review the workload logs for details: %s", logsURL))
		}
		if workload.Status != client.ProtonStatusRunning {
			return fmt.Errorf("workload is not running yet (status: %s)", workload.Status)
		}
		running = workload
		return nil
	}

	if err := backoff.Retry(operation, expBackoff); err != nil {
		return nil, err
	}

	return running, nil
}

func waitForWorkloadToBeDeleted(ctx context.Context, s client.Service, id string) error {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("GetWorkload")
		workload, err := s.GetWorkload(ctx, id)
		if err != nil {
			if _, ok := err.(*client.NotFoundError); ok {
				return nil
			}
			return backoff.Permanent(err)
		}
		return fmt.Errorf("workload is not deleted yet (status: %s)", workload.Status)
	}

	return backoff.Retry(operation, expBackoff)
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

	desc := data.Description.ValueString()
	req.Description = &desc

	if !data.Importance.IsNull() && !data.Importance.IsUnknown() {
		imp := client.WorkloadImportance(data.Importance.ValueString())
		req.Importance = &imp
	}

	return req
}

func workloadRuntimeToClient(runtime WorkloadRuntimeModel) client.WorkloadRuntime {
	r := client.WorkloadRuntime{}

	if len(runtime.ContainerGroups) > 0 {
		r.ContainerGroups = make([]client.GroupRuntime, len(runtime.ContainerGroups))
		for i, g := range runtime.ContainerGroups {
			r.ContainerGroups[i] = groupRuntimeToClient(g)
		}
	}

	return r
}

func groupRuntimeToClient(g WorkloadGroupRuntimeModel) client.GroupRuntime {
	gr := client.GroupRuntime{}

	if !g.Name.IsNull() && !g.Name.IsUnknown() {
		gr.Name = g.Name.ValueString()
	}

	if !g.ReplicaCount.IsNull() && !g.ReplicaCount.IsUnknown() && g.ReplicaCount.ValueInt64() != 0 {
		v := g.ReplicaCount.ValueInt64()
		gr.ReplicaCount = &v
	}

	if !g.BundleSelectionPolicy.IsNull() && !g.BundleSelectionPolicy.IsUnknown() {
		v := g.BundleSelectionPolicy.ValueString()
		gr.BundleSelectionPolicy = &v
	}

	if g.Autoscaling != nil {
		gr.Autoscaling = &client.AutoscalingProperties{}
		if !g.Autoscaling.Enabled.IsNull() && !g.Autoscaling.Enabled.IsUnknown() {
			enabled := g.Autoscaling.Enabled.ValueBool()
			gr.Autoscaling.Enabled = &enabled
		}
		gr.Autoscaling.Policies = make([]client.AutoscalingPolicy, len(g.Autoscaling.Policies))
		for i, p := range g.Autoscaling.Policies {
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
			gr.Autoscaling.Policies[i] = policy
		}
	}

	if len(g.ResourceBundles) > 0 {
		gr.ResourceBundles = make([]string, len(g.ResourceBundles))
		for i, rb := range g.ResourceBundles {
			gr.ResourceBundles[i] = rb.ValueString()
		}
	}

	if len(g.Containers) > 0 {
		gr.Containers = make([]client.ContainerOverride, len(g.Containers))
		for i, c := range g.Containers {
			gr.Containers[i] = containerOverrideToClient(c)
		}
	}

	return gr
}

func containerOverrideToClient(c WorkloadContainerOverrideModel) client.ContainerOverride {
	co := client.ContainerOverride{Name: c.Name.ValueString()}
	if c.ResourceAllocation != nil {
		ra := &client.ResourceAllocation{}
		if !c.ResourceAllocation.CPU.IsNull() && !c.ResourceAllocation.CPU.IsUnknown() {
			v := c.ResourceAllocation.CPU.ValueFloat64()
			ra.CPU = &v
		}
		if !c.ResourceAllocation.GPU.IsNull() && !c.ResourceAllocation.GPU.IsUnknown() {
			v := c.ResourceAllocation.GPU.ValueFloat64()
			ra.GPU = &v
		}
		if !c.ResourceAllocation.GPUMemory.IsNull() && !c.ResourceAllocation.GPUMemory.IsUnknown() {
			v, _ := parseMemoryBytes(c.ResourceAllocation.GPUMemory.ValueString())
			ra.GPUMemory = &v
		}
		if !c.ResourceAllocation.Memory.IsNull() && !c.ResourceAllocation.Memory.IsUnknown() {
			v, _ := parseMemoryBytes(c.ResourceAllocation.Memory.ValueString())
			ra.Memory = &v
		}
		co.ResourceAllocation = ra
	}
	return co
}

// applySentinels reconciles state values that the API would otherwise misrepresent:
//   - replica_count=0 per group signals "explicitly cleared"; restore it to prevent a perpetual diff.
//   - description="" is indistinguishable from "not set"; restore null when user omitted it.
//   - container_groups=nil (user omitted the block) must stay nil even if the API returns groups.
//   - resource_bundles=nil (user omitted the field) must stay nil even if the API injects defaults.
//   - bundle_selection_policy=null (user omitted the field) must stay null even if the API returns a default.
//   - memory/gpu_memory: API normalizes to bytes; restore the user's original string format if values are semantically equal.
func applySentinels(desired WorkloadResourceModel, data *WorkloadResourceModel) {
	if desired.Description.IsNull() && data.Description.ValueString() == "" {
		data.Description = types.StringNull()
	}
	if desired.Runtime.ContainerGroups == nil {
		data.Runtime.ContainerGroups = nil
		return
	}
	for i := range desired.Runtime.ContainerGroups {
		if i >= len(data.Runtime.ContainerGroups) {
			break
		}
		dg := desired.Runtime.ContainerGroups[i]
		if !dg.ReplicaCount.IsNull() && !dg.ReplicaCount.IsUnknown() && dg.ReplicaCount.ValueInt64() == 0 {
			data.Runtime.ContainerGroups[i].ReplicaCount = dg.ReplicaCount
		}
		if dg.ResourceBundles == nil {
			data.Runtime.ContainerGroups[i].ResourceBundles = nil
		}
		data.Runtime.ContainerGroups[i].BundleSelectionPolicy = dg.BundleSelectionPolicy
		for j := range dg.Containers {
			if j >= len(data.Runtime.ContainerGroups[i].Containers) {
				break
			}
			restoreMemoryFormat(dg.Containers[j], &data.Runtime.ContainerGroups[i].Containers[j])
		}
	}
}

// restoreMemoryFormat keeps the user's original memory string in state when its byte value matches
// what the API returned, preventing perpetual diffs caused by the API normalizing to raw bytes.
func restoreMemoryFormat(desired WorkloadContainerOverrideModel, data *WorkloadContainerOverrideModel) {
	if desired.ResourceAllocation == nil || data.ResourceAllocation == nil {
		return
	}
	ra := desired.ResourceAllocation
	if !ra.Memory.IsNull() && !data.ResourceAllocation.Memory.IsNull() {
		if desiredBytes, err := parseMemoryBytes(ra.Memory.ValueString()); err == nil {
			if dataBytes, err := parseMemoryBytes(data.ResourceAllocation.Memory.ValueString()); err == nil && desiredBytes == dataBytes {
				data.ResourceAllocation.Memory = ra.Memory
			}
		}
	}
	if !ra.GPUMemory.IsNull() && !data.ResourceAllocation.GPUMemory.IsNull() {
		if desiredBytes, err := parseMemoryBytes(ra.GPUMemory.ValueString()); err == nil {
			if dataBytes, err := parseMemoryBytes(data.ResourceAllocation.GPUMemory.ValueString()); err == nil && desiredBytes == dataBytes {
				data.ResourceAllocation.GPUMemory = ra.GPUMemory
			}
		}
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

	if workload.Description != nil {
		data.Description = types.StringValue(*workload.Description)
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

func loadWorkloadRuntimeFromAPI(runtime client.WorkloadRuntime) WorkloadRuntimeModel {
	model := WorkloadRuntimeModel{}

	if len(runtime.ContainerGroups) > 0 {
		model.ContainerGroups = make([]WorkloadGroupRuntimeModel, len(runtime.ContainerGroups))
		for i, g := range runtime.ContainerGroups {
			model.ContainerGroups[i] = loadGroupRuntimeFromAPI(g)
		}
	}

	return model
}

func loadGroupRuntimeFromAPI(g client.GroupRuntime) WorkloadGroupRuntimeModel {
	name := g.Name
	if name == "" {
		name = "default"
	}
	m := WorkloadGroupRuntimeModel{
		Name: types.StringValue(name),
	}

	if g.ReplicaCount != nil {
		m.ReplicaCount = types.Int64Value(*g.ReplicaCount)
	} else {
		m.ReplicaCount = types.Int64Null()
	}

	bundleSelectionPolicy := "availability"
	if g.BundleSelectionPolicy != nil {
		bundleSelectionPolicy = *g.BundleSelectionPolicy
	}
	m.BundleSelectionPolicy = types.StringValue(bundleSelectionPolicy)

	if g.Autoscaling != nil {
		autoscaling := &WorkloadAutoscalingModel{}
		if g.Autoscaling.Enabled != nil {
			autoscaling.Enabled = types.BoolValue(*g.Autoscaling.Enabled)
		} else {
			autoscaling.Enabled = types.BoolNull()
		}
		autoscaling.Policies = make([]WorkloadAutoscalingPolicyModel, len(g.Autoscaling.Policies))
		for i, p := range g.Autoscaling.Policies {
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
		m.Autoscaling = autoscaling
	}

	if len(g.ResourceBundles) > 0 {
		m.ResourceBundles = make([]types.String, len(g.ResourceBundles))
		for i, rb := range g.ResourceBundles {
			m.ResourceBundles[i] = types.StringValue(rb)
		}
	}

	if len(g.Containers) > 0 {
		m.Containers = make([]WorkloadContainerOverrideModel, len(g.Containers))
		for i, c := range g.Containers {
			m.Containers[i] = loadContainerOverrideFromAPI(c)
		}
	}

	return m
}

func loadContainerOverrideFromAPI(c client.ContainerOverride) WorkloadContainerOverrideModel {
	m := WorkloadContainerOverrideModel{
		Name: types.StringValue(c.Name),
	}
	if c.ResourceAllocation != nil {
		ra := &WorkloadResourceAllocationModel{}
		if c.ResourceAllocation.CPU != nil {
			ra.CPU = types.Float64Value(*c.ResourceAllocation.CPU)
		} else {
			ra.CPU = types.Float64Null()
		}
		if c.ResourceAllocation.GPU != nil {
			ra.GPU = types.Float64Value(*c.ResourceAllocation.GPU)
		} else {
			ra.GPU = types.Float64Null()
		}
		if c.ResourceAllocation.GPUMemory != nil {
			ra.GPUMemory = types.StringValue(strconv.FormatInt(*c.ResourceAllocation.GPUMemory, 10))
		} else {
			ra.GPUMemory = types.StringNull()
		}
		if c.ResourceAllocation.Memory != nil {
			ra.Memory = types.StringValue(strconv.FormatInt(*c.ResourceAllocation.Memory, 10))
		} else {
			ra.Memory = types.StringNull()
		}
		m.ResourceAllocation = ra
	}
	return m
}
