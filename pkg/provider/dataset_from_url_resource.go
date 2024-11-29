package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
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
	provider *Provider
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

func (r *DatasetFromURLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatasetFromURLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateDatasetFromURL")
	createResp, err := r.provider.service.CreateDatasetFromURL(ctx, &client.CreateDatasetFromURLRequest{
		URL: data.URL.ValueString(),
	})
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
		err = r.provider.service.AddEntityToUseCase(ctx, useCaseID.ValueString(), "dataset", dataset.ID)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Dataset to Use Case %s", useCaseID), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasetFromURLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatasetFromURLResourceModel

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

func (r *DatasetFromURLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state DatasetFromURLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan DatasetFromURLResourceModel

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

func (r *DatasetFromURLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatasetFromURLResourceModel

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

func (r *DatasetFromURLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
