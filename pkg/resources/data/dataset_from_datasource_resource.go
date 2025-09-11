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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DatasetFromDatasourceResource{}
var _ resource.ResourceWithImportState = &DatasetFromDatasourceResource{}

func NewDatasetFromDatasourceResource() resource.Resource {
	return &DatasetFromDatasourceResource{}
}

// DatasetFromDatasourceResource defines the resource implementation.
type DatasetFromDatasourceResource struct {
	service client.Service
}

func (r *DatasetFromDatasourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_from_datasource"
}

func (r *DatasetFromDatasourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Data Set from Data Source.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"data_source_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID for the DataSource to use as the source of data.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"credential_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the set of credentials to use.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"do_snapshot": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "If unset, uses the server default: True. If true, creates a snapshot dataset; if false, creates a remote dataset.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"persist_data_after_ingestion": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "If unset, uses the server default: True. If true, will enforce saving all data (for download and sampling) and will allow a user to view extended data profile (which includes data statistics like min/max/median/mean, histogram, etc.). If false, will not enforce saving data. The data schema (feature names and types) still will be available.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"use_kerberos": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If unset, uses the server default: False. If true, use kerberos authentication for database authentication.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"sample_size_rows": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The number of rows fetched during dataset registration.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"categories": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "An array of strings describing the intended use of the dataset.",
				ElementType:         types.StringType,
				Validators:          common.DatasetCategoryValidators(),
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Dataset to.",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *DatasetFromDatasourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *DatasetFromDatasourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.DatasetFromDatasourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	categories := make([]string, len(data.Categories))
	for i, category := range data.Categories {
		categories[i] = category.ValueString()
	}

	createRequest := &client.CreateDatasetFromDatasourceRequest{
		DataSourceID:              data.DataSourceID.ValueString(),
		CredentialID:              data.CredentialID.ValueString(),
		UseKerberos:               common.BoolValuePointerOptional(data.UseKerberos),
		PersistDataAfterIngestion: common.BoolValuePointerOptional(data.PersistDataAfterIngestion),
		DoSnapshot:                common.BoolValuePointerOptional(data.DoSnapshot),
		Categories:                categories,
	}

	if common.IsKnown(data.SampleSizeRows) {
		createRequest.SampleSize = &client.DatasetSampleSize{
			Type:  "rows",
			Value: data.SampleSizeRows.ValueInt64(),
		}
	}

	common.TraceAPICall("CreateDatasetFromDatasource")
	createResp, err := r.service.CreateDatasetFromDataSource(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Dataset", err.Error())
		return
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

func (r *DatasetFromDatasourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DatasetFromDatasourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetDataset")
	dataset, err := r.service.GetDataset(ctx, data.ID.ValueString())
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
	data.DataSourceID = types.StringValue(dataset.DataSourceID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasetFromDatasourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state models.DatasetFromDatasourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan models.DatasetFromDatasourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	categories := make([]string, len(plan.Categories))
	for i, category := range plan.Categories {
		categories[i] = category.ValueString()
	}

	common.TraceAPICall("UpdateDataset")
	_, err := r.service.UpdateDataset(ctx, plan.ID.ValueString(), &client.UpdateDatasetRequest{
		Categories: &categories,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Dataset", err.Error())
		return
	}

	err = common.UpdateUseCasesForEntity(
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

func (r *DatasetFromDatasourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DatasetFromDatasourceResourceModel

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

func (r *DatasetFromDatasourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
