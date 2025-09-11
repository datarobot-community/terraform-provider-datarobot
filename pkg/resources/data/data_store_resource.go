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
var _ resource.Resource = &DatastoreResource{}
var _ resource.ResourceWithImportState = &DatastoreResource{}

func NewDatastoreResource() resource.Resource {
	return &DatastoreResource{}
}

// DatastoreResource defines the resource implementation.
type DatastoreResource struct {
	service client.Service
}

func (r *DatastoreResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datastore"
}

func (r *DatastoreResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Data store",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the data store.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"data_store_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of data store.",
				Validators:          common.DataStoreTypeValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"canonical_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user-friendly name of the data store.",
			},
			"driver_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The identifier of the DataDriver if data_store_type is JDBC or DR_DATABASE_V1",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"jdbc_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The full JDBC URL (for example: jdbc:postgresql://my.dbaddress.org:5432/my_db).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connector_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The identifier of the Connector if data_store_type is DR_CONNECTOR_V1",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fields": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "If the type is dr-database-v1, then the fields specify the configuration.",
				ElementType:         types.MapType{ElemType: types.StringType},
			},
		},
	}
}

func (r *DatastoreResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *DatastoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.DatastoreResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createDatastoreRequest := &client.CreateDatastoreRequest{
		CanonicalName: data.CanonicalName.ValueString(),
		Type:          data.DataStoreType.ValueString(),
		Params: client.DatastoreParams{
			DriverID:    common.StringValuePointerOptional(data.DriverID),
			ConnectorID: common.StringValuePointerOptional(data.ConnectorID),
			JDBCUrl:     common.StringValuePointerOptional(data.JDBCUrl),
		},
	}

	fields := convertFields(data)
	if data.DataStoreType.ValueString() == "jdbc" {
		createDatastoreRequest.Params.JDBCFields = fields
	} else {
		createDatastoreRequest.Params.Fields = fields
	}

	common.TraceAPICall("CreateDatastore")
	dataStore, err := r.service.CreateDatastore(ctx, createDatastoreRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Datastore", err.Error())
		return
	}
	data.ID = types.StringValue(dataStore.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatastoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DatastoreResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetDatastore")
	dataStore, err := r.service.GetDatastore(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Datastore not found",
				fmt.Sprintf("Datastore with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Datastore with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.CanonicalName = types.StringValue(dataStore.CanonicalName)
	data.DataStoreType = types.StringValue(dataStore.Type)
	data.DriverID = types.StringPointerValue(dataStore.Params.DriverID)
	data.ConnectorID = types.StringPointerValue(dataStore.Params.ConnectorID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatastoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.DatastoreResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateDatastoreRequest := &client.UpdateDatastoreRequest{
		CanonicalName: data.CanonicalName.ValueString(),
		Params:        client.DatastoreParams{},
	}

	fields := convertFields(data)
	if data.DataStoreType.ValueString() == "jdbc" {
		updateDatastoreRequest.Params.JDBCFields = fields
	} else {
		updateDatastoreRequest.Params.Fields = fields
	}

	common.TraceAPICall("UpdateDatastore")
	_, err := r.service.UpdateDatastore(ctx, data.ID.ValueString(), updateDatastoreRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Datastore", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatastoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DatastoreResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteDatastore")
	err := r.service.DeleteDatastore(ctx, state.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error getting Datastore", err.Error())
		}
		return
	}
}

func (r *DatastoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func convertFields(data models.DatastoreResourceModel) []map[string]string {
	fieldsList := make([]map[string]string, 0)
	for _, field := range data.Fields {
		fieldsList = append(fieldsList, common.ConvertTfStringMap(field))
	}
	return fieldsList
}
