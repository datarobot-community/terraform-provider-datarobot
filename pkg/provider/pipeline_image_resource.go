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

var _ resource.Resource = &PipelineImageResource{}
var _ resource.ResourceWithImportState = &PipelineImageResource{}
var _ resource.ResourceWithModifyPlan = &PipelineImageResource{}

func NewPipelineImageResource() resource.Resource {
	return &PipelineImageResource{}
}

type PipelineImageResource struct {
	provider *Provider
}

type PipelineImageResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Packages      types.List   `tfsdk:"packages"`
	LatestVersion types.Int64  `tfsdk:"latest_version"`
	LatestStatus  types.String `tfsdk:"latest_status"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *PipelineImageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_image"
}

func (r *PipelineImageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A pipeline execution image containing a versioned set of pip packages.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the pipeline image.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique name for the image. Changing this forces a new resource.",
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
				MarkdownDescription: "The latest version number of this image.",
			},
			"latest_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Build status of the latest version: `CREATING`, `READY`, or `ERROR`.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp when the image was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp when the image was last updated.",
			},
		},
	}
}

func (r *PipelineImageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PipelineImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PipelineImageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	packages := make([]string, 0)
	resp.Diagnostics.Append(data.Packages.ElementsAs(ctx, &packages, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.PipelineImageCreateRequest{
		Name:     data.Name.ValueString(),
		Packages: packages,
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		createReq.Description = &desc
	}

	traceAPICall("CreatePipelineImage")
	image, err := r.provider.service.CreatePipelineImage(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Pipeline Image", err.Error())
		return
	}

	loadPipelineImageIntoModel(image, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PipelineImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("GetPipelineImage")
	image, err := r.provider.service.GetPipelineImage(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Pipeline Image", err.Error())
		return
	}

	loadPipelineImageIntoModel(image, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PipelineImageResourceModel
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

	traceAPICall("UpdatePipelineImage")
	image, err := r.provider.service.UpdatePipelineImage(ctx, state.ID.ValueString(), &client.PipelineImageUpdateRequest{Packages: newPkgs})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Pipeline Image", err.Error())
		return
	}

	loadPipelineImageIntoModel(image, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PipelineImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeletePipelineImage")
	err := r.provider.service.DeletePipelineImage(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); !ok {
			resp.Diagnostics.AddError("Error deleting Pipeline Image", err.Error())
		}
	}
}

func (r *PipelineImageResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state PipelineImageResourceModel
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

func (r *PipelineImageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadPipelineImageIntoModel(image *client.PipelineImage, data *PipelineImageResourceModel) {
	data.ID = types.StringValue(image.ImageID)
	data.Name = types.StringValue(image.Name)
	if image.Description != nil {
		data.Description = types.StringValue(*image.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.LatestVersion = types.Int64Value(int64(image.LatestVersion))
	data.CreatedAt = types.StringValue(image.CreatedAt)
	data.UpdatedAt = types.StringValue(image.UpdatedAt)

	if len(image.Versions) > 0 {
		latestVer := image.Versions[0]
		for _, v := range image.Versions {
			if v.Version == image.LatestVersion {
				latestVer = v
				break
			}
		}
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
