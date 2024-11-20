package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DatasourceResource{}
var _ resource.ResourceWithImportState = &DatasourceResource{}

func NewDatasourceResource() resource.Resource {
	return &DatasourceResource{}
}

// DatasourceResource defines the resource implementation.
type DatasourceResource struct {
	provider *Provider
}

func (r *DatasourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datasource"
}

func (r *DatasourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the data source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"data_source_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of data source.",
				Validators:          DataStoreTypeValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"canonical_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user-friendly name of the data source.",
			},
			"params": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The data source parameters.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"data_store_id": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The id of the DataStore.",
					},
					"table": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The name of specified database table.",
					},
					"schema": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The name of the schema associated with the table.",
					},
					"partition_column": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The name of the partition column.",
					},
					"query": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The user specified SQL query.",
					},
					"fetch_size": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "A user specified fetch size in the range [1, 20000]. By default a fetchSize will be assigned to balance throughput and memory usage.",
					},
					"path": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The user-specified path for BLOB storage.",
					},
					"catalog": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The Catalog name in the database if supported.",
					},
				},
			},
		},
	}
}

func (r *DatasourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatasourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatasourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createDatasourceRequest := &client.CreateDatasourceRequest{
		CanonicalName: data.CanonicalName.ValueString(),
		Type:          data.DataSourceType.ValueString(),
		Params: client.DatasourceParams{
			DataStoreID:     data.Params.DataStoreID.ValueString(),
			Table:           StringValuePointerOptional(data.Params.Table),
			Schema:          StringValuePointerOptional(data.Params.Schema),
			PartitionColumn: StringValuePointerOptional(data.Params.PartitionColumn),
			FetchSize:       Int64ValuePointerOptional(data.Params.FetchSize),
			Query:           StringValuePointerOptional(data.Params.Query),
			Path:            StringValuePointerOptional(data.Params.Path),
			Catalog:         StringValuePointerOptional(data.Params.Catalog),
		},
	}

	traceAPICall("CreateDatasource")
	dataSource, err := r.provider.service.CreateDatasource(ctx, createDatasourceRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Datasource", err.Error())
		return
	}
	data.ID = types.StringValue(dataSource.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatasourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetDatasource")
	dataSource, err := r.provider.service.GetDatasource(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Datasource not found",
				fmt.Sprintf("Datasource with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Datasource with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.CanonicalName = types.StringValue(dataSource.CanonicalName)
	data.DataSourceType = types.StringValue(dataSource.Type)
	data.Params.DataStoreID = types.StringValue(dataSource.Params.DataStoreID)
	data.Params.Catalog = types.StringPointerValue(dataSource.Params.Catalog)
	data.Params.Schema = types.StringPointerValue(dataSource.Params.Schema)
	data.Params.Path = types.StringPointerValue(dataSource.Params.Path)
	data.Params.Table = types.StringPointerValue(dataSource.Params.Table)
	data.Params.PartitionColumn = types.StringPointerValue(dataSource.Params.PartitionColumn)
	data.Params.Query = types.StringPointerValue(dataSource.Params.Query)
	data.Params.FetchSize = types.Int64PointerValue(dataSource.Params.FetchSize)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DatasourceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("UpdateDatasource")
	_, err := r.provider.service.UpdateDatasource(ctx, data.ID.ValueString(), &client.UpdateDatasourceRequest{
		CanonicalName: data.CanonicalName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Datasource", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatasourceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteDatasource")
	err := r.provider.service.DeleteDatasource(ctx, state.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Datasource", err.Error())
		}
		return
	}
}

func (r *DatasourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
