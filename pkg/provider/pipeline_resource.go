package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PipelineResource{}
var _ resource.ResourceWithImportState = &PipelineResource{}
var _ resource.ResourceWithModifyPlan = &PipelineResource{}

func NewPipelineResource() resource.Resource {
	return &PipelineResource{}
}

type PipelineResource struct {
	provider *Provider
}

type PipelineResourceModel struct {
	ID             types.String `tfsdk:"id"`
	SourceFile     types.String `tfsdk:"source_file"`
	SourceFileHash types.String `tfsdk:"source_file_hash"`
	Description    types.String `tfsdk:"description"`
	Mode           types.String `tfsdk:"mode"`
	CurrentVersion types.Int64  `tfsdk:"current_version"`
	TaskNames      types.List   `tfsdk:"task_names"`
}

func (r *PipelineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (r *PipelineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A DataRobot pipeline defined by a Python source file using `@pipeline` and `@task` decorators.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the pipeline.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_file": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to the Python source file defining the pipeline. Content changes are tracked via `source_file_hash`.",
			},
			"source_file_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SHA-256 hash of the source file contents. Changes when the file is edited, triggering an update.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional description of the pipeline.",
			},
			"mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(client.PipelineModeDraft)),
				MarkdownDescription: "Pipeline lifecycle mode: `draft` (mutable) or `locked` (immutable). Once locked, any change forces a new resource.",
			},
			"current_version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The latest version number. Null while the pipeline is in draft mode.",
			},
			"task_names": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Names of the `@task`-decorated functions extracted from the source file.",
			},
		},
	}
}

func (r *PipelineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, fileName, err := readSourceFile(data.SourceFile.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("source_file"), "Cannot read source file", err.Error())
		return
	}
	if hash, hashErr := computeFileHash(data.SourceFile.ValueString()); hashErr == nil {
		data.SourceFileHash = types.StringValue(hash)
	}

	var desc *string
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		d := data.Description.ValueString()
		desc = &d
	}

	traceAPICall("CreatePipeline")
	pipeline, err := r.provider.service.CreatePipeline(ctx, fileName, content, desc)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Pipeline", err.Error())
		return
	}

	if data.Mode.ValueString() == string(client.PipelineModeLocked) {
		traceAPICall("LockPipeline")
		pipeline, err = r.provider.service.LockPipeline(ctx, pipeline.PipelineID)
		if err != nil {
			resp.Diagnostics.AddError("Error locking Pipeline", err.Error())
			return
		}
	}

	loadPipelineIntoModel(pipeline, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("GetPipeline")
	pipeline, err := r.provider.service.GetPipeline(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Pipeline", err.Error())
		return
	}

	// Preserve source_file and source_file_hash from prior state — the API does not return the file path.
	sourceFile := data.SourceFile
	sourceFileHash := data.SourceFileHash
	loadPipelineIntoModel(pipeline, &data)
	data.SourceFile = sourceFile
	data.SourceFileHash = sourceFileHash

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle draft→locked transition.
	if state.Mode.ValueString() == string(client.PipelineModeDraft) && plan.Mode.ValueString() == string(client.PipelineModeLocked) {
		// Re-upload if the file also changed before locking.
		if !plan.SourceFileHash.Equal(state.SourceFileHash) {
			content, fileName, err := readSourceFile(plan.SourceFile.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("source_file"), "Cannot read source file", err.Error())
				return
			}
			if hash, hashErr := computeFileHash(plan.SourceFile.ValueString()); hashErr == nil {
				plan.SourceFileHash = types.StringValue(hash)
			}
			traceAPICall("UpdatePipelineDraft")
			_, err = r.provider.service.UpdatePipelineDraft(ctx, state.ID.ValueString(), fileName, content)
			if err != nil {
				resp.Diagnostics.AddError("Error updating Pipeline draft", err.Error())
				return
			}
		}

		traceAPICall("LockPipeline")
		pipeline, err := r.provider.service.LockPipeline(ctx, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error locking Pipeline", err.Error())
			return
		}
		loadPipelineIntoModel(pipeline, &plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// Draft update: re-upload the source file.
	content, fileName, err := readSourceFile(plan.SourceFile.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("source_file"), "Cannot read source file", err.Error())
		return
	}
	if hash, hashErr := computeFileHash(plan.SourceFile.ValueString()); hashErr == nil {
		plan.SourceFileHash = types.StringValue(hash)
	}

	traceAPICall("UpdatePipelineDraft")
	pipeline, err := r.provider.service.UpdatePipelineDraft(ctx, state.ID.ValueString(), fileName, content)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Pipeline draft", err.Error())
		return
	}

	loadPipelineIntoModel(pipeline, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeletePipeline")
	err := r.provider.service.DeletePipeline(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); !ok {
			resp.Diagnostics.AddError("Error deleting Pipeline", err.Error())
		}
	}
}

func (r *PipelineResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan PipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compute and propagate the hash so Terraform can detect file-content drift.
	if !plan.SourceFile.IsNull() && !plan.SourceFile.IsUnknown() {
		if hash, err := computeFileHash(plan.SourceFile.ValueString()); err == nil {
			plan.SourceFileHash = types.StringValue(hash)
		}
	}

	// If the pipeline is already locked, changes to source_file, description, or mode require replacement.
	if !req.State.Raw.IsNull() {
		var state PipelineResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if state.Mode.ValueString() == string(client.PipelineModeLocked) {
			if !plan.SourceFileHash.Equal(state.SourceFileHash) {
				resp.RequiresReplace.Append(path.Root("source_file"))
			}
			if !plan.Description.Equal(state.Description) {
				resp.RequiresReplace.Append(path.Root("description"))
			}
			if !plan.Mode.Equal(state.Mode) {
				resp.RequiresReplace.Append(path.Root("mode"))
			}
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *PipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func readSourceFile(filePath string) (content []byte, fileName string, err error) {
	content, err = os.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}
	fileName = filepath.Base(filePath)
	return content, fileName, nil
}

func loadPipelineIntoModel(p *client.Pipeline, data *PipelineResourceModel) {
	data.ID = types.StringValue(p.PipelineID)
	data.Mode = types.StringValue(string(p.Mode))

	if len(p.Versions) > 0 {
		data.CurrentVersion = types.Int64Value(int64(p.Versions[0].Version))
	} else {
		data.CurrentVersion = types.Int64Null()
	}

	electronNames := p.ElectronNames
	if len(p.Versions) > 0 && len(p.Versions[0].ElectronNames) > 0 {
		electronNames = p.Versions[0].ElectronNames
	}
	if len(electronNames) > 0 {
		vals := make([]attr.Value, len(electronNames))
		for i, name := range electronNames {
			vals[i] = types.StringValue(name)
		}
		data.TaskNames, _ = types.ListValue(types.StringType, vals)
	} else {
		data.TaskNames, _ = types.ListValue(types.StringType, []attr.Value{})
	}
}
