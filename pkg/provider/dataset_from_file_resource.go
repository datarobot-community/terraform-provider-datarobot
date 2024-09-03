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
			"source_file": schema.StringAttribute{
				MarkdownDescription: "The path to the file to upload.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"use_case_id": schema.StringAttribute{
				MarkdownDescription: "The id of the Use Case.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

	filePath := data.SourceFile.ValueString()
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		resp.Diagnostics.AddError("Can't get info from source file", err.Error())
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

	dataset, err := waitForDatasetToBeReady(ctx, r.provider.service, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Dataset info", err.Error())
		return
	}
	data.ID = types.StringValue(dataset.ID)

	traceAPICall("LinkDatasetToUseCase")
	err = r.provider.service.LinkDatasetToUseCase(ctx, data.UseCaseID.ValueString(), dataset.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error linking Dataset to Use Case", err.Error())
		return
	}

	_, err = waitForDatasetToBeReady(ctx, r.provider.service, dataset.ID)
	if err != nil {
		resp.Diagnostics.AddError("Dataset not ready", err.Error())
		return
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
	// This resource does not support updates.
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
