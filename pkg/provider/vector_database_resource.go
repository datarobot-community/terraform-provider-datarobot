package provider

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultEmbeddingModel         = "jinaai/jina-embedding-t-en-v1"
	defaultChunkOverlapPercentage = 0
	defaultChunkSize              = 256
	defaultChunkingMethod         = "recursive"
	defaultIsSeparatorRegex       = false
)

var defaultSeparators = []string{"\n\n", "\n", " ", ""}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VectorDatabaseResource{}
var _ resource.ResourceWithImportState = &VectorDatabaseResource{}

func NewVectorDatabaseResource() resource.Resource {
	return &VectorDatabaseResource{}
}

// VectorDatabaseResource defines the resource implementation.
type VectorDatabaseResource struct {
	provider *Provider
}

func (r *VectorDatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vector_database"
}

func (r *VectorDatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Vector database",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the VectorDatabase.",
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The version of the VectorDatabase.",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VectorDatabase.",
				Required:            true,
			},
			"dataset_id": schema.StringAttribute{
				MarkdownDescription: "The id of the Vector Database.",
				Required:            true,
			},
			"use_case_id": schema.StringAttribute{
				MarkdownDescription: "The id of the Use Case.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"chunking_parameters": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The chunking parameters for the Model.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"embedding_model":          types.StringType,
						"chunk_overlap_percentage": types.Int64Type,
						"chunk_size":               types.Int64Type,
						"chunking_method":          types.StringType,
						"is_separator_regex":       types.BoolType,
						"separators":               types.ListType{ElemType: types.StringType},
					},
					map[string]attr.Value{
						"embedding_model":          types.StringValue(defaultEmbeddingModel),
						"chunk_overlap_percentage": types.Int64Value(defaultChunkOverlapPercentage),
						"chunk_size":               types.Int64Value(defaultChunkSize),
						"chunking_method":          types.StringValue(defaultChunkingMethod),
						"is_separator_regex":       types.BoolValue(defaultIsSeparatorRegex),
						"separators": types.ListValueMust(
							types.StringType,
							[]attr.Value{
								types.StringValue(defaultSeparators[0]),
								types.StringValue(defaultSeparators[1]),
								types.StringValue(defaultSeparators[2]),
								types.StringValue(defaultSeparators[3]),
							},
						),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"embedding_model": schema.StringAttribute{
						MarkdownDescription: "The id of the Embedding Model.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(defaultEmbeddingModel),
						Validators:          EmbeddingModelValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"chunk_overlap_percentage": schema.Int64Attribute{
						MarkdownDescription: "The percentage of overlap between chunks.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(defaultChunkOverlapPercentage),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"chunk_size": schema.Int64Attribute{
						MarkdownDescription: "The size of the chunks.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(defaultChunkSize),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"chunking_method": schema.StringAttribute{
						MarkdownDescription: "The method used to chunk the data.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(defaultChunkingMethod),
						Validators:          ChunkingMethodValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"is_separator_regex": schema.BoolAttribute{
						MarkdownDescription: "Whether the separator is a regex.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(defaultIsSeparatorRegex),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"separators": schema.ListAttribute{
						MarkdownDescription: "The separators used to split the data.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Default: listdefault.StaticValue(types.ListValueMust(
							types.StringType,
							[]attr.Value{
								types.StringValue(defaultSeparators[0]),
								types.StringValue(defaultSeparators[1]),
								types.StringValue(defaultSeparators[2]),
								types.StringValue(defaultSeparators[3]),
							},
						)),
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *VectorDatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VectorDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VectorDatabaseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var datasetID string
	if IsKnown(data.DatasetID) {
		datasetID = data.DatasetID.ValueString()
		_, err := r.provider.service.GetDataset(ctx, datasetID)
		if err != nil {
			resp.Diagnostics.AddError("Error getting Dataset info", err.Error())
			return
		}
		dataset, err := waitForDatasetToBeReady(ctx, r.provider.service, datasetID)
		if err != nil {
			resp.Diagnostics.AddError("Dataset not ready", err.Error())
			return
		}
		if !dataset.IsVectorDatabaseEligible {
			resp.Diagnostics.AddError("Dataset not eligible for VectorDatabase", "Dataset is not eligible for VectorDatabase")
			return
		}
	}

	var useCaseID string
	if IsKnown(data.UseCaseID) {
		useCaseID = data.UseCaseID.ValueString()
		_, err := r.provider.service.GetUseCase(ctx, useCaseID)
		if err != nil {
			resp.Diagnostics.AddError("Error getting Use Case", err.Error())
			return
		}
	}

	separators := make([]string, 0)
	for _, separator := range data.ChunkingParameters.Separators {
		separators = append(separators, separator.ValueString())
	}

	traceAPICall("CreateVectorDatabase")
	vectorDatabase, err := r.provider.service.CreateVectorDatabase(ctx, &client.CreateVectorDatabaseRequest{
		DatasetID: datasetID,
		UseCaseID: useCaseID,
		Name:      data.Name.ValueString(),
		ChunkingParameters: client.ChunkingParameters{
			EmbeddingModel:         data.ChunkingParameters.EmbeddingModel.ValueString(),
			ChunkOverlapPercentage: data.ChunkingParameters.ChunkOverlapPercentage.ValueInt64(),
			ChunkSize:              data.ChunkingParameters.ChunkSize.ValueInt64(),
			ChunkingMethod:         data.ChunkingParameters.ChunkingMethod.ValueString(),
			IsSeparatorRegex:       data.ChunkingParameters.IsSeparatorRegex.ValueBool(),
			Separators:             separators,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating VectorDatabase", err.Error())
		return
	}
	loadVectorDatabaseToTerraformState(vectorDatabase, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)

	err = r.waitForVectorDatabaseToBeReady(ctx, vectorDatabase.ID)
	if err != nil {
		resp.Diagnostics.AddError("Vector Database not ready", err.Error())
		return
	}
}

func (r *VectorDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VectorDatabaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetVectorDatabase")
	vectorDatabase, err := r.provider.service.GetVectorDatabase(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"VectorDatabase not found",
				fmt.Sprintf("VectorDatabase with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting VectorDatabase with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	loadVectorDatabaseToTerraformState(vectorDatabase, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VectorDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VectorDatabaseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state VectorDatabaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vectorDatabase *client.VectorDatabase
	var err error
	if !reflect.DeepEqual(state.ChunkingParameters, plan.ChunkingParameters) ||
		!reflect.DeepEqual(state.DatasetID, plan.DatasetID) ||
		!reflect.DeepEqual(state.UseCaseID, plan.UseCaseID) {
		// create new vector database version
		separators := make([]string, 0)
		for _, separator := range plan.ChunkingParameters.Separators {
			separators = append(separators, separator.ValueString())
		}

		traceAPICall("CreateVectorDatabase")
		vectorDatabase, err = r.provider.service.CreateVectorDatabase(ctx, &client.CreateVectorDatabaseRequest{
			ParentVectorDatabaseID: state.ID.ValueStringPointer(),
			DatasetID:              plan.DatasetID.ValueString(),
			UseCaseID:              plan.UseCaseID.ValueString(),
			Name:                   plan.Name.ValueString(),
			ChunkingParameters: client.ChunkingParameters{
				EmbeddingModel:         plan.ChunkingParameters.EmbeddingModel.ValueString(),
				ChunkOverlapPercentage: plan.ChunkingParameters.ChunkOverlapPercentage.ValueInt64(),
				ChunkSize:              plan.ChunkingParameters.ChunkSize.ValueInt64(),
				ChunkingMethod:         plan.ChunkingParameters.ChunkingMethod.ValueString(),
				IsSeparatorRegex:       plan.ChunkingParameters.IsSeparatorRegex.ValueBool(),
				Separators:             separators,
			},
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating VectorDatabase version", err.Error())
			return
		}

		err = r.waitForVectorDatabaseToBeReady(ctx, vectorDatabase.ID)
		if err != nil {
			resp.Diagnostics.AddError("Vector Database not ready", err.Error())
			return
		}
	} else {
		traceAPICall("UpdateVectorDatabase")
		vectorDatabase, err = r.provider.service.UpdateVectorDatabase(ctx,
			state.ID.ValueString(),
			&client.UpdateVectorDatabaseRequest{
				Name: plan.Name.ValueString(),
			})
		if err != nil {
			if errors.Is(err, &client.NotFoundError{}) {
				resp.Diagnostics.AddWarning(
					"VectorDatabase not found",
					fmt.Sprintf("VectorDatabase with ID %s is not found. Removing from state.", state.ID.ValueString()))
				resp.State.RemoveResource(ctx)
			} else {
				resp.Diagnostics.AddError("Error updating VectorDatabase", err.Error())
			}
			return
		}
	}

	loadVectorDatabaseToTerraformState(vectorDatabase, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *VectorDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VectorDatabaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteVectorDatabase")
	err := r.provider.service.DeleteVectorDatabase(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting VectorDatabase", err.Error())
			return
		}
	}
}

func (r *VectorDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *VectorDatabaseResource) waitForVectorDatabaseToBeReady(ctx context.Context, vectorDatabaseId string) error {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("IsVectorDatabaseReady")
		ready, err := r.provider.service.IsVectorDatabaseReady(ctx, vectorDatabaseId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("dataset is not ready is not ready for a vector database")
		}
		return nil
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return err
	}
	return nil
}

func loadVectorDatabaseToTerraformState(vectorDatabase *client.VectorDatabase, data *VectorDatabaseResourceModel) {
	data.ID = types.StringValue(vectorDatabase.ID)
	data.Version = types.Int64Value(vectorDatabase.Version)
	data.Name = types.StringValue(vectorDatabase.Name)
	data.DatasetID = types.StringValue(vectorDatabase.DatasetID)
	data.UseCaseID = types.StringValue(vectorDatabase.UseCaseID)
	data.ChunkingParameters = &ChunkingParametersModel{
		EmbeddingModel:         types.StringValue(vectorDatabase.EmbeddingModel),
		ChunkOverlapPercentage: types.Int64Value(vectorDatabase.ChunkOverlapPercentage),
		ChunkSize:              types.Int64Value(vectorDatabase.ChunkSize),
		ChunkingMethod:         types.StringValue(vectorDatabase.ChunkingMethod),
		IsSeparatorRegex:       types.BoolValue(vectorDatabase.IsSeparatorRegex),
	}
	separatorList := make([]types.String, 0)
	for _, separator := range vectorDatabase.Separators {
		separatorList = append(separatorList, types.StringValue(separator))
	}
	data.ChunkingParameters.Separators = separatorList
}
