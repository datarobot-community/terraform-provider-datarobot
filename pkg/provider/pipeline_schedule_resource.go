package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PipelineScheduleResource{}
var _ resource.ResourceWithImportState = &PipelineScheduleResource{}
var _ resource.ResourceWithValidateConfig = &PipelineScheduleResource{}

func NewPipelineScheduleResource() resource.Resource {
	return &PipelineScheduleResource{}
}

type PipelineScheduleResource struct {
	provider *Provider
}

type PipelineScheduleResourceModel struct {
	ID              types.String `tfsdk:"id"`
	PipelineID      types.String `tfsdk:"pipeline_id"`
	Version         types.Int64  `tfsdk:"version"`
	PipelineInputID types.String `tfsdk:"pipeline_input_id"`
	CronExpression  types.String `tfsdk:"cron_expression"`
	Timezone        types.String `tfsdk:"timezone"`
	Status          types.String `tfsdk:"status"`
}

func (r *PipelineScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_schedule"
}

func (r *PipelineScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A recurring schedule that fires a pipeline run against a locked pipeline version.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the pipeline schedule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pipeline_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the parent pipeline. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The locked pipeline version to schedule. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"pipeline_input_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the pipeline input to use for scheduled runs. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cron_expression": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Cron expression defining the schedule (e.g. `0 9 * * 1-5`).",
			},
			"timezone": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
				MarkdownDescription: "Timezone for the cron schedule. Defaults to `UTC`.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current status of the schedule: `ACTIVE`, `PAUSED`, or `DELETED`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PipelineScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PipelineScheduleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data PipelineScheduleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Version.IsNull() && !data.Version.IsUnknown() && data.Version.ValueInt64() < 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("version"),
			"Invalid version",
			"version must be >= 1. Schedules can only be attached to locked pipeline versions.",
		)
	}
}

func (r *PipelineScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PipelineScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled, err := r.provider.service.IsFeatureFlagEnabled(ctx, "PIPELINES_API_ENABLED")
	if err != nil {
		resp.Diagnostics.AddError("Error checking feature flag", err.Error())
		return
	}
	if !enabled {
		resp.Diagnostics.AddError(
			"Feature not enabled",
			"The PIPELINES_API_ENABLED feature flag is not enabled. Please contact DataRobot to enable Pipelines for your account.",
		)
		return
	}

	createReq := &client.PipelineScheduleCreateRequest{
		CronExpression:  data.CronExpression.ValueString(),
		PipelineInputID: data.PipelineInputID.ValueString(),
		Timezone:        data.Timezone.ValueString(),
	}

	traceAPICall("CreatePipelineSchedule")
	schedule, err := r.provider.service.CreatePipelineSchedule(ctx, data.PipelineID.ValueString(), int(data.Version.ValueInt64()), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Pipeline Schedule", err.Error())
		return
	}

	loadPipelineScheduleIntoModel(schedule, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PipelineScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("GetPipelineSchedule")
	schedule, err := r.provider.service.GetPipelineSchedule(ctx, data.PipelineID.ValueString(), int(data.Version.ValueInt64()), data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Pipeline Schedule", err.Error())
		return
	}

	// Preserve pipeline_input_id from state — the GET endpoint does not return it.
	inputID := data.PipelineInputID
	loadPipelineScheduleIntoModel(schedule, &data)
	data.PipelineInputID = inputID

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PipelineScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.PipelineScheduleUpdateRequest{}
	if !plan.CronExpression.Equal(state.CronExpression) {
		v := plan.CronExpression.ValueString()
		updateReq.CronExpression = &v
	}
	if !plan.Timezone.Equal(state.Timezone) {
		v := plan.Timezone.ValueString()
		updateReq.Timezone = &v
	}

	traceAPICall("UpdatePipelineSchedule")
	schedule, err := r.provider.service.UpdatePipelineSchedule(ctx, state.PipelineID.ValueString(), int(state.Version.ValueInt64()), state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Pipeline Schedule", err.Error())
		return
	}

	inputID := plan.PipelineInputID
	loadPipelineScheduleIntoModel(schedule, &plan)
	plan.PipelineInputID = inputID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PipelineScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeletePipelineSchedule")
	err := r.provider.service.DeletePipelineSchedule(ctx, data.PipelineID.ValueString(), int(data.Version.ValueInt64()), data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); !ok {
			resp.Diagnostics.AddError("Error deleting Pipeline Schedule", err.Error())
		}
	}
}

// ImportState accepts "<pipeline_id>/<version>/<schedule_id>".
func (r *PipelineScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected <pipeline_id>/<version>/<schedule_id>, got: %s", req.ID))
		return
	}
	version, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Version must be an integer in <pipeline_id>/<version>/<schedule_id>, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pipeline_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), version)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func loadPipelineScheduleIntoModel(s *client.PipelineSchedule, data *PipelineScheduleResourceModel) {
	data.ID = types.StringValue(s.ScheduleID)
	data.PipelineID = types.StringValue(s.PipelineID)
	data.Version = types.Int64Value(int64(s.Version))
	data.CronExpression = types.StringValue(s.CronExpression)
	data.Timezone = types.StringValue(s.Timezone)
	data.Status = types.StringValue(string(s.Status))
}
