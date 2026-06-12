package provider

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PipelineEnvironmentResource{}
var _ resource.ResourceWithImportState = &PipelineEnvironmentResource{}
var _ resource.ResourceWithModifyPlan = &PipelineEnvironmentResource{}

func NewPipelineEnvironmentResource() resource.Resource {
	return &PipelineEnvironmentResource{}
}

type PipelineEnvironmentResource struct {
	provider *Provider
}

type PipelineEnvironmentResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Packages      types.List   `tfsdk:"packages"`
	LatestVersion types.Int64  `tfsdk:"latest_version"`
	LatestStatus  types.String `tfsdk:"latest_status"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *PipelineEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_environment"
}

func (r *PipelineEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A pipeline execution environment containing a versioned set of pip packages.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the pipeline environment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique name for the environment. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional description. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"packages": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of pip package specifiers (e.g. `numpy==1.26.0`). Packages can only be added; removing any forces a new resource.",
			},
			"latest_version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The latest version number of this environment.",
			},
			"latest_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Build status of the latest version: `CREATING`, `READY`, or `ERROR`.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp when the environment was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp when the environment was last updated.",
			},
		},
	}
}

func (r *PipelineEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PipelineEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PipelineEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	packages := make([]string, 0)
	resp.Diagnostics.Append(data.Packages.ElementsAs(ctx, &packages, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.PipelineEnvironmentCreateRequest{
		Name:     data.Name.ValueString(),
		Packages: packages,
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		createReq.Description = &desc
	}

	traceAPICall("CreatePipelineEnvironment")
	env, err := r.provider.service.CreatePipelineEnvironment(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Pipeline Environment", err.Error())
		return
	}

	loadPipelineEnvironmentIntoModel(env, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PipelineEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("GetPipelineEnvironment")
	env, err := r.provider.service.GetPipelineEnvironment(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Pipeline Environment", err.Error())
		return
	}

	loadPipelineEnvironmentIntoModel(env, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PipelineEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planPkgs := make([]string, 0)
	resp.Diagnostics.Append(plan.Packages.ElementsAs(ctx, &planPkgs, false)...)
	statePkgs := make([]string, 0)
	resp.Diagnostics.Append(state.Packages.ElementsAs(ctx, &statePkgs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existingSet := make(map[string]bool, len(statePkgs))
	for _, p := range statePkgs {
		existingSet[p] = true
	}
	var newPkgs []string
	for _, p := range planPkgs {
		if !existingSet[p] {
			newPkgs = append(newPkgs, p)
		}
	}

	if len(newPkgs) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	traceAPICall("UpdatePipelineEnvironment")
	env, err := r.provider.service.UpdatePipelineEnvironment(ctx, state.ID.ValueString(), &client.PipelineEnvironmentUpdateRequest{Packages: newPkgs})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Pipeline Environment", err.Error())
		return
	}

	loadPipelineEnvironmentIntoModel(env, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PipelineEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeletePipelineEnvironment")
	err := r.provider.service.DeletePipelineEnvironment(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); !ok {
			resp.Diagnostics.AddError("Error deleting Pipeline Environment", err.Error())
		}
	}
}

func (r *PipelineEnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state PipelineEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planPkgs := make([]string, 0)
	resp.Diagnostics.Append(plan.Packages.ElementsAs(ctx, &planPkgs, false)...)
	statePkgs := make([]string, 0)
	resp.Diagnostics.Append(state.Packages.ElementsAs(ctx, &statePkgs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planSet := make(map[string]bool, len(planPkgs))
	for _, p := range planPkgs {
		planSet[p] = true
	}
	for _, p := range statePkgs {
		if !planSet[p] {
			resp.RequiresReplace.Append(path.Root("packages"))
			return
		}
	}
}

func (r *PipelineEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadPipelineEnvironmentIntoModel(env *client.PipelineEnvironment, data *PipelineEnvironmentResourceModel) {
	data.ID = types.StringValue(env.EnvironmentID)
	data.Name = types.StringValue(env.Name)
	if env.Description != nil {
		data.Description = types.StringValue(*env.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.LatestVersion = types.Int64Value(int64(env.LatestVersion))
	data.CreatedAt = types.StringValue(env.CreatedAt)
	data.UpdatedAt = types.StringValue(env.UpdatedAt)

	if len(env.Versions) > 0 {
		latestVer := env.Versions[0]
		data.LatestStatus = types.StringValue(string(latestVer.Status))
		pkgVals := make([]attr.Value, len(latestVer.Packages))
		for i, p := range latestVer.Packages {
			pkgVals[i] = types.StringValue(p)
		}
		pkgList, _ := types.ListValue(types.StringType, pkgVals)
		data.Packages = pkgList
	} else {
		data.LatestStatus = types.StringValue("")
		data.Packages, _ = types.ListValue(types.StringType, []attr.Value{})
	}
}
