package data

import (
	"context"
	"errors"
	"fmt"

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
var _ resource.Resource = &DatasetFromURLResource{}
var _ resource.ResourceWithImportState = &DatasetFromURLResource{}

func NewDatasetFromURLResource() resource.Resource {
	return &DatasetFromURLResource{}
}

// DatasetFromURLResource defines the resource implementation.
type DatasetFromURLResource struct {
	service client.Service
}

func (r *DatasetFromURLResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_from_url"
}

func (r *DatasetFromURLResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to upload the Dataset from.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the Dataset.",
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Dataset to.",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *DatasetFromURLResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *DatasetFromURLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.DatasetFromURLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("CreateDatasetFromURL")
	createResp, err := r.service.CreateDatasetFromURL(ctx, &client.CreateDatasetFromURLRequest{
		URL: data.URL.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Dataset", err.Error())
		return
	}

	if common.IsKnown(data.Name) {
		common.TraceAPICall("UpdateDataset")
		_, err = r.service.UpdateDataset(ctx, createResp.ID, &client.UpdateDatasetRequest{
			Name: data.Name.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Dataset", err.Error())
			return
		}
	}

	dataset, err := common.WaitForDatasetToBeReady(ctx, r.service, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Dataset to be ready", err.Error())
		return
	}
	data.ID = types.StringValue(dataset.ID)

	for _, useCaseID := range data.UseCaseIDs {
		common.TraceAPICall("AddDatasetToUseCase")
		if err = common.AddEntityToUseCase(
			ctx,
			r.service,
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

func (r *DatasetFromURLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DatasetFromURLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetDataset")
	_, err := r.service.GetDataset(ctx, data.ID.ValueString())
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

func (r *DatasetFromURLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state models.DatasetFromURLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan models.DatasetFromURLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if common.IsKnown(plan.Name) {
		common.TraceAPICall("UpdateDataset")
		_, err := r.service.UpdateDataset(ctx, plan.ID.ValueString(), &client.UpdateDatasetRequest{
			Name: plan.Name.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Dataset", err.Error())
			return
		}
	}

	err := common.UpdateUseCasesForEntity(
		ctx,
		r.service,
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

func (r *DatasetFromURLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DatasetFromURLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteDataset")
	err := r.service.DeleteDataset(ctx, state.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Dataset info", err.Error())
		}
		return
	}
}

func (r *DatasetFromURLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
