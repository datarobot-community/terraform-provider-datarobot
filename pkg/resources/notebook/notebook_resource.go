package notebook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
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
	service client.Service
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
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *NotebookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.NotebookResourceModel

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

	common.TraceAPICall("ImportNotebookFromFile")
	importResp, err := r.service.ImportNotebookFromFile(ctx, fileName, content, useCaseID)
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
	var state models.NotebookResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("GetNotebook")
	notebook, err := r.service.GetNotebook(ctx, state.ID.ValueString())
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

	var plan, state models.NotebookResourceModel

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
	var plan, state models.NotebookResourceModel

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
		// Delete old Notebook and re-import it. If no use-case, it becomes a "classic" notebook.
		common.TraceAPICall("DeleteNoteook")
		err := r.service.DeleteNotebook(ctx, state.ID.ValueString())
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

		common.TraceAPICall("ImportNotebookFromFile")
		importResp, err := r.service.ImportNotebookFromFile(ctx, fileName, content, plan.UseCaseID.ValueString())
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
		common.TraceAPICall("UpdateNotebook")
		notebookResponse, err := r.service.UpdateNotebook(ctx, state.ID.ValueString(), plan.UseCaseID.ValueString())
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
	var state models.NotebookResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the notebook
	common.TraceAPICall("DeleteNotebook")
	err := r.service.DeleteNotebook(ctx, state.ID.ValueString())
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
