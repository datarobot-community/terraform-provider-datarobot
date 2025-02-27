package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DatasetFromFileResource{}
var _ resource.ResourceWithImportState = &DatasetFromFileResource{}
var _ resource.ResourceWithModifyPlan = &DatasetFromFileResource{}

func NewDatasetFromFileResource() resource.Resource {
	return &DatasetFromFileResource{}
}

// DatasetFromFileResource defines the resource implementation.
type DatasetFromFileResource struct {
	provider *Provider
}

func (r *DatasetFromFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_from_file"
}

func (r *DatasetFromFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Data set from file",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"file_path": schema.StringAttribute{
				MarkdownDescription: "The path to the file to upload.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of the file contents.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the Dataset. Defaults to the file name.",
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Dataset to.",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *DatasetFromFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*Provider); !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected  %T, got: %T. Please report this issue to the provider developers.", Provider{}, req.ProviderData),
		)
	}
}

func (r *DatasetFromFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatasetFromFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filePath := data.FilePath.ValueString()
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		resp.Diagnostics.AddError("Can't get info from file", err.Error())
		return
	}

	fileName := fileInfo.Name()
	fileReader, err := os.Open(filePath)
	if err != nil {
		resp.Diagnostics.AddError("Error opening file", err.Error())
		return
	}

	defer fileReader.Close()
	fileContent, err := io.ReadAll(fileReader)
	if err != nil {
		resp.Diagnostics.AddError("Error reading file", err.Error())
		return
	}

	traceAPICall("CreateDatasetFromFile")
	createResp, err := r.provider.service.CreateDatasetFromFile(ctx,
		fileName,
		fileContent,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Dataset", err.Error())
		return
	}

	if IsKnown(data.Name) {
		traceAPICall("UpdateDataset")
		_, err = r.provider.service.UpdateDataset(ctx, createResp.ID, &client.UpdateDatasetRequest{
			Name: data.Name.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Dataset", err.Error())
			return
		}
	}

	dataset, err := waitForDatasetToBeReady(ctx, r.provider.service, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Dataset to be ready", err.Error())
		return
	}
	data.ID = types.StringValue(dataset.ID)

	for _, useCaseID := range data.UseCaseIDs {
		traceAPICall("AddDatasetToUseCase")
		if err = addEntityToUseCase(
			ctx,
			r.provider.service,
			useCaseID.ValueString(),
			"dataset",
			dataset.ID,
		); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Dataset to Use Case %s", useCaseID), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasetFromFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatasetFromFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetDataset")
	_, err := r.provider.service.GetDataset(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Dataset not found",
				fmt.Sprintf("Dataset with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Dataset with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasetFromFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state DatasetFromFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan DatasetFromFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if IsKnown(plan.Name) {
		traceAPICall("UpdateDataset")
		_, err := r.provider.service.UpdateDataset(ctx, plan.ID.ValueString(), &client.UpdateDatasetRequest{
			Name: plan.Name.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Dataset", err.Error())
			return
		}
	}

	err := updateUseCasesForEntity(
		ctx,
		r.provider.service,
		"dataset",
		plan.ID.ValueString(),
		state.UseCaseIDs,
		plan.UseCaseIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Use Cases for Dataset", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *DatasetFromFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatasetFromFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteDataset")
	err := r.provider.service.DeleteDataset(ctx, state.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Dataset info", err.Error())
		}
		return
	}
}

func (r *DatasetFromFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r DatasetFromFileResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan DatasetFromFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// compute file content hash
	fileContentHash, err := computeFileHash(plan.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error calculating file hash", err.Error())
		return
	}
	plan.FileHash = types.StringValue(fileContentHash)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

	if req.State.Raw.IsNull() {
		// Resource is being created
		return
	}

	var state DatasetFromFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FileHash != state.FileHash {
		resp.RequiresReplace.Append(path.Root("file_hash"))
	}
}
