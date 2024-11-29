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
var _ resource.Resource = &DatastoreResource{}
var _ resource.ResourceWithImportState = &DatastoreResource{}

func NewDatastoreResource() resource.Resource {
	return &DatastoreResource{}
}

// DatastoreResource defines the resource implementation.
type DatastoreResource struct {
	provider *Provider
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
				Validators:          DataStoreTypeValidators(),
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

func (r *DatastoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatastoreResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createDatastoreRequest := &client.CreateDatastoreRequest{
		CanonicalName: data.CanonicalName.ValueString(),
		Type:          data.DataStoreType.ValueString(),
		Params: client.DatastoreParams{
			DriverID:    StringValuePointerOptional(data.DriverID),
			ConnectorID: StringValuePointerOptional(data.ConnectorID),
			JDBCUrl:     StringValuePointerOptional(data.JDBCUrl),
		},
	}

	fields := convertFields(data)
	if data.DataStoreType.ValueString() == "jdbc" {
		createDatastoreRequest.Params.JDBCFields = fields
	} else {
		createDatastoreRequest.Params.Fields = fields
	}

	traceAPICall("CreateDatastore")
	dataStore, err := r.provider.service.CreateDatastore(ctx, createDatastoreRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Datastore", err.Error())
		return
	}
	data.ID = types.StringValue(dataStore.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatastoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatastoreResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetDatastore")
	dataStore, err := r.provider.service.GetDatastore(ctx, data.ID.ValueString())
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
	var data DatastoreResourceModel

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

	traceAPICall("UpdateDatastore")
	_, err := r.provider.service.UpdateDatastore(ctx, data.ID.ValueString(), updateDatastoreRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Datastore", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatastoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatastoreResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteDatastore")
	err := r.provider.service.DeleteDatastore(ctx, state.ID.ValueString())
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

func convertFields(data DatastoreResourceModel) []map[string]string {
	fieldsList := make([]map[string]string, 0)
	for _, field := range data.Fields {
		fieldsList = append(fieldsList, convertTfStringMap(field))
	}
	return fieldsList
}
