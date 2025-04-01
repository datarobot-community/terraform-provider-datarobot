package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NotebookResource{}
var _ resource.ResourceWithImportState = &NotebookResource{}
var _ resource.ResourceWithModifyPlan = &NotebookResource{}

func NewNotebookResource() resource.Resource {
	return &NotebookResource{}
}

// NotebookResource defines the resource implementation.
type NotebookResource struct {
	provider *Provider
}

func (r *NotebookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notebook"
}

func (r *NotebookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: `
Notebook resource for importing and managing Jupyter notebooks in DataRobot.

**NOTE**

The synchronization of the file hash is one way. The provider will track changes of the Notebook file on disk
and update _only_ when that changes. If the remote Notebook changes, the provider will not update the local file.
`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Notebook.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the Notebook.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"file_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The path to the .ipynb file to import.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The SHA-256 hash of the file contents.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL to the Notebook.",
			},
			"use_case_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The Use Case ID to add the Notebook to.",
			},
		},
	}
}

func (r *NotebookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Provider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}
}

func (r *NotebookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NotebookResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get use case ID if specified
	var useCaseID string
	if !plan.UseCaseID.IsNull() {
		useCaseID = plan.UseCaseID.ValueString()
	}
	content, fileName, hashStr, err := GetNotebookFileInfo(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading file", err.Error())
		return
	}

	traceAPICall("ImportNotebookFromFile")
	importResp, err := r.provider.service.ImportNotebookFromFile(ctx, fileName, content, useCaseID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing notebook", err.Error())
		return
	}

	// Update model
	plan.ID = types.StringValue(importResp.ID)
	plan.Name = types.StringValue(importResp.Name)
	plan.FileHash = types.StringValue(hashStr)
	plan.URL = types.StringValue(importResp.URL)

	// Save model into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotebookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NotebookResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("GetNotebook")
	notebook, err := r.provider.service.GetNotebook(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading notebook", err.Error())
		return
	}

	// Update resource state with the current state
	state.ID = types.StringValue(notebook.ID)
	state.Name = types.StringValue(notebook.Name)
	state.URL = types.StringValue(notebook.URL)

	// Store the UseCaseID if available.
	if notebook.UseCaseID != "" {
		state.UseCaseID = types.StringValue(notebook.UseCaseID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotebookResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If we're planning a delete, don't do anything
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state NotebookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If we don't have state yet (creating) or plan is destroying, no need to check for changes
	if req.State.Raw.IsNull() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Calculate current hash from file
	_, _, currentHashStr, err := GetNotebookFileInfo(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading file", err.Error())
		return
	}

	// If the file has changed, update the planned value for file_hash
	if currentHashStr != state.FileHash.ValueString() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hash"), types.StringValue(currentHashStr))...)

		// Mark ID and URL as unknown since they will change with reimport
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("url"), types.StringUnknown())...)
	}
}

func (r *NotebookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NotebookResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle updates that need to replace the file on the server:
	// 1. If the file contents change, we need to re-import the notebook.
	// 2. If we no longer have a use case, we need to re-import the notebook.
	if (!state.UseCaseID.IsNull() && plan.UseCaseID.IsNull()) || !state.FileHash.Equal(plan.FileHash) {
		// Delete old Notebook and re-import it.
		traceAPICall("DeleteNoteook")
		err := r.provider.service.DeleteNotebook(ctx, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error deleting notebook", err.Error())
			return
		}
		// Re-import the notebook.
		content, fileName, hashStr, err := GetNotebookFileInfo(plan.FilePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading file", err.Error())
			return
		}

		traceAPICall("ImportNotebookFromFile")
		importResp, err := r.provider.service.ImportNotebookFromFile(ctx, fileName, content, plan.UseCaseID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error importing notebook", err.Error())
			return
		}
		state.FileHash = types.StringValue(hashStr)
		state.ID = types.StringValue(importResp.ID)
		state.Name = types.StringValue(importResp.Name)
		state.UseCaseID = plan.UseCaseID
		state.URL = types.StringValue(importResp.URL)

	} else if !plan.UseCaseID.Equal(state.UseCaseID) && !plan.UseCaseID.IsNull() {
		// Update the use case with the new UseCaseID
		traceAPICall("UpdateNotebook")
		notebookResponse, err := r.provider.service.UpdateNotebook(ctx, state.ID.ValueString(), plan.UseCaseID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating notebook", err.Error())
			return
		}
		// Update the state with the plan
		state.UseCaseID = plan.UseCaseID
		state.URL = types.StringValue(notebookResponse.URL)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotebookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotebookResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the notebook
	traceAPICall("DeleteNotebook")
	err := r.provider.service.DeleteNotebook(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting notebook", err.Error())
		return
	}

}

func (r *NotebookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func GetNotebookFileInfo(filePath string) ([]byte, string, string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("unable to read file at %s: %s", filePath, err)
	}

	// Calculate file hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	fileName := filepath.Base(filePath)

	return content, fileName, hashStr, nil
}
