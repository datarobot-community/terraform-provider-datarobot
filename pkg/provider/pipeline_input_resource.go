package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PipelineInputResource{}
var _ resource.ResourceWithImportState = &PipelineInputResource{}
var _ resource.ResourceWithModifyPlan = &PipelineInputResource{}

func NewPipelineInputResource() resource.Resource {
	return &PipelineInputResource{}
}

type PipelineInputResource struct {
	provider *Provider
}

type PipelineInputResourceModel struct {
	ID         types.String `tfsdk:"id"`
	PipelineID types.String `tfsdk:"pipeline_id"`
	Version    types.Int64  `tfsdk:"version"`
	Payload    types.String `tfsdk:"payload"`
	State      types.String `tfsdk:"state"`
}

func (r *PipelineInputResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_input"
}

func (r *PipelineInputResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A JSON payload attached to a pipeline (draft or a specific locked version) used for runs and schedules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the pipeline input.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pipeline_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the parent pipeline.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Locked version number to attach this input to. Omit to attach to the pipeline draft. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"payload": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "JSON-encoded input payload. Use `jsonencode()` or `file()` to supply the value. Key order and whitespace are normalized.",
			},
			"state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Validation state of the input: `VALID` or `INVALID`.",
			},
		},
	}
}

func (r *PipelineInputResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PipelineInputResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PipelineInputResourceModel
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

	payload, err := unmarshalPayload(data.Payload.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("payload"), "Invalid JSON payload", err.Error())
		return
	}
	createReq := &client.PipelineInputCreateRequest{Payload: payload}

	var input *client.PipelineInput
	if data.Version.IsNull() || data.Version.IsUnknown() {
		traceAPICall("CreateDraftPipelineInput")
		input, err = r.provider.service.CreateDraftPipelineInput(ctx, data.PipelineID.ValueString(), createReq)
	} else {
		traceAPICall("CreateLockedPipelineInput")
		input, err = r.provider.service.CreateLockedPipelineInput(ctx, data.PipelineID.ValueString(), int(data.Version.ValueInt64()), createReq)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating Pipeline Input", err.Error())
		return
	}

	loadPipelineInputIntoModel(input, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineInputResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PipelineInputResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input *client.PipelineInput
	var err error

	if data.Version.IsNull() || data.Version.IsUnknown() {
		traceAPICall("GetDraftPipelineInput")
		input, err = r.provider.service.GetDraftPipelineInput(ctx, data.PipelineID.ValueString(), data.ID.ValueString())
	} else {
		traceAPICall("GetLockedPipelineInput")
		input, err = r.provider.service.GetLockedPipelineInput(ctx, data.PipelineID.ValueString(), int(data.Version.ValueInt64()), data.ID.ValueString())
	}
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Pipeline Input", err.Error())
		return
	}

	// Preserve the user-provided payload string (normalized) for consistent diffs.
	prevPayload := data.Payload
	loadPipelineInputIntoModel(input, &data)
	if normalizedPayloadsEqual(prevPayload.ValueString(), data.Payload.ValueString()) {
		data.Payload = prevPayload
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineInputResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PipelineInputResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := unmarshalPayload(plan.Payload.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("payload"), "Invalid JSON payload", err.Error())
		return
	}

	if state.Version.IsNull() || state.Version.IsUnknown() {
		// Draft input: patch in-place.
		traceAPICall("UpdateDraftPipelineInput")
		input, err := r.provider.service.UpdateDraftPipelineInput(ctx, state.PipelineID.ValueString(), state.ID.ValueString(), &client.PipelineInputUpdateRequest{Payload: payload})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Pipeline Input", err.Error())
			return
		}
		loadPipelineInputIntoModel(input, &plan)
	} else {
		// Locked input: delete and recreate.
		traceAPICall("DeleteLockedPipelineInput")
		err = r.provider.service.DeleteLockedPipelineInput(ctx, state.PipelineID.ValueString(), int(state.Version.ValueInt64()), state.ID.ValueString())
		if err != nil {
			if _, ok := err.(*client.NotFoundError); !ok {
				resp.Diagnostics.AddError("Error deleting locked Pipeline Input for recreation", err.Error())
				return
			}
		}

		traceAPICall("CreateLockedPipelineInput")
		input, createErr := r.provider.service.CreateLockedPipelineInput(ctx, plan.PipelineID.ValueString(), int(plan.Version.ValueInt64()), &client.PipelineInputCreateRequest{Payload: payload})
		if createErr != nil {
			resp.Diagnostics.AddError("Error recreating locked Pipeline Input", createErr.Error())
			return
		}
		loadPipelineInputIntoModel(input, &plan)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineInputResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PipelineInputResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	if data.Version.IsNull() || data.Version.IsUnknown() {
		traceAPICall("DeleteDraftPipelineInput")
		err = r.provider.service.DeleteDraftPipelineInput(ctx, data.PipelineID.ValueString(), data.ID.ValueString())
	} else {
		traceAPICall("DeleteLockedPipelineInput")
		err = r.provider.service.DeleteLockedPipelineInput(ctx, data.PipelineID.ValueString(), int(data.Version.ValueInt64()), data.ID.ValueString())
	}
	if err != nil {
		if _, ok := err.(*client.NotFoundError); !ok {
			resp.Diagnostics.AddError("Error deleting Pipeline Input", err.Error())
		}
	}
}

// ModifyPlan marks payload as requiring replacement for locked inputs.
// Locked inputs are delete-and-recreated (new ID), so any payload change is
// functionally a resource replacement — making it explicit prevents Terraform
// from raising an "inconsistent result" error after apply.
func (r *PipelineInputResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state PipelineInputResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.Version.IsNull() && !state.Version.IsUnknown() &&
		!normalizedPayloadsEqual(plan.Payload.ValueString(), state.Payload.ValueString()) {
		resp.RequiresReplace.Append(path.Root("payload"))
	}
}

// ImportState accepts "<pipeline_id>/<input_id>" for draft inputs or
// "<pipeline_id>/<version>/<input_id>" for locked inputs.
func (r *PipelineInputResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pipeline_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	case 3:
		version, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid import ID",
				fmt.Sprintf("Version must be an integer in <pipeline_id>/<version>/<input_id>, got: %s", req.ID))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pipeline_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), version)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected <pipeline_id>/<input_id> (draft) or <pipeline_id>/<version>/<input_id> (locked), got: %s", req.ID))
	}
}

func unmarshalPayload(raw string) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func normalizedPayloadsEqual(a, b string) bool {
	var pa, pb map[string]any
	if json.Unmarshal([]byte(a), &pa) != nil || json.Unmarshal([]byte(b), &pb) != nil {
		return a == b
	}
	na, _ := json.Marshal(pa)
	nb, _ := json.Marshal(pb)
	return string(na) == string(nb)
}

func loadPipelineInputIntoModel(input *client.PipelineInput, data *PipelineInputResourceModel) {
	data.ID = types.StringValue(input.InputID)
	data.PipelineID = types.StringValue(input.PipelineID)
	data.State = types.StringValue(string(input.State))
	// data.Version is intentionally NOT set from input.VersionID.
	// The API's version_id is an internal DB FK (auto-increment), not the
	// user-facing pipeline version number. Version is user-provided and must
	// be preserved from plan/state to avoid inconsistent result errors.

	normalized, err := json.Marshal(input.Payload)
	if err == nil {
		data.Payload = types.StringValue(string(normalized))
	}
}
