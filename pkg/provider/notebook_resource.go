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
		MarkdownDescription: "Notebook resource for importing and managing Jupyter notebooks in DataRobot",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Notebook.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the Notebook.",
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
					stringplanmodifier.UseStateForUnknown(),
				},
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

	// Read file
	filePath := plan.FilePath.ValueString()
	content, err := os.ReadFile(filePath)
	if err != nil {
		resp.Diagnostics.AddError("Error reading file", fmt.Sprintf("Unable to read file at %s: %s", filePath, err))
		return
	}

	// Calculate file hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Get use case ID if specified
	var useCaseID string
	if !plan.UseCaseID.IsNull() {
		useCaseID = plan.UseCaseID.ValueString()
	}

	fileName := filepath.Base(filePath)

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

	// Store the UseCaseID if available.
	if notebook.UseCaseID != "" {
		state.UseCaseID = types.StringValue(notebook.UseCaseID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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

	// Check if use case ID has changed
	if !plan.UseCaseID.Equal(state.UseCaseID) {
		// Remove notebook from old use case if it exists
		if !state.UseCaseID.IsNull() {
			oldUseCaseID := state.UseCaseID.ValueString()
			traceAPICall("RemoveEntityFromUseCase")
			err := r.provider.service.RemoveEntityFromUseCase(ctx, oldUseCaseID, "notebook", state.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error removing notebook from use case",
					fmt.Sprintf("Error removing notebook from use case %s: %s", oldUseCaseID, err.Error()),
				)
			}
		}

		// Add notebook to new use case if specified
		if !plan.UseCaseID.IsNull() {
			newUseCaseID := plan.UseCaseID.ValueString()
			traceAPICall("AddEntityToUseCase")
			err := r.provider.service.AddEntityToUseCase(ctx, newUseCaseID, "notebook", state.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Error adding notebook to use case",
					fmt.Sprintf("Error adding notebook to use case %s: %s", newUseCaseID, err.Error()),
				)
			}
		}
	}

	// Update the state with the plan
	state.UseCaseID = plan.UseCaseID

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotebookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotebookResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove notebook from all use cases first
	useCaseID := state.UseCaseID.ValueString()
	traceAPICall("RemoveEntityFromUseCase")
	err := r.provider.service.RemoveEntityFromUseCase(ctx, useCaseID, "notebook", state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Error removing notebook from use case",
			fmt.Sprintf("Error removing notebook from use case %s: %s", useCaseID, err.Error()),
		)
	}

	// Delete the notebook
	traceAPICall("DeleteNotebook")
	err = r.provider.service.DeleteNotebook(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting notebook", err.Error())
		return
	}
}

func (r *NotebookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
