package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VectorDatabase.",
				Required:            true,
			},
			"dataset_id": schema.StringAttribute{
				MarkdownDescription: "The id of the Vector Database.",
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
			"chunking_parameters": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The chunking parameters for the Model.",
				Attributes: map[string]schema.Attribute{
					"embedding_model": schema.StringAttribute{
						MarkdownDescription: "The id of the Embedding Model.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("jinaai/jina-embedding-t-en-v1"),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"chunk_overlap_percentage": schema.Int32Attribute{
						MarkdownDescription: "The percentage of overlap between chunks.",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(0),
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier.UseStateForUnknown(),
						},
					},
					"chunk_size": schema.Int32Attribute{
						MarkdownDescription: "The size of the chunks.",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(256),
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier.UseStateForUnknown(),
						},
					},
					"chunking_method": schema.StringAttribute{
						MarkdownDescription: "The method used to chunk the data.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("recursive"),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"is_separator_regex": schema.BoolAttribute{
						MarkdownDescription: "Whether the separator is a regex.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
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
								types.StringValue("↵↵"),
								types.StringValue("↵"),
								types.StringValue(" "),
								types.StringValue(""),
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
		err = r.waitForDatasetToBeReady(ctx, datasetID)
		if err != nil {
			resp.Diagnostics.AddError("Dataset not ready", err.Error())
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

	chunkingParameters := data.ChunkingParameters
	separators := make([]string, 0)
	for _, separator := range chunkingParameters.Separators.Elements() {
		separators = append(separators, separator.String())
	}

	traceAPICall("CreateVectorDatabase")
	createResp, err := r.provider.service.CreateVectorDatabase(ctx, &client.CreateVectorDatabaseRequest{
		DatasetID: datasetID,
		UseCaseID: useCaseID,
		Name:      data.Name.ValueString(),
		ChunkingParameters: client.ChunkingParameters{
			EmbeddingModel:         chunkingParameters.EmbeddingModel.ValueString(),
			ChunkOverlapPercentage: chunkingParameters.ChunkOverlapPercentage.ValueInt32(),
			ChunkSize:              chunkingParameters.ChunkSize.ValueInt32(),
			ChunkingMethod:         chunkingParameters.ChunkingMethod.ValueString(),
			IsSeparatorRegex:       chunkingParameters.IsSeparatorRegex.ValueBool(),
			Separators:             separators,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating VectorDatabase", err.Error())
		return
	}
	data.ID = types.StringValue(createResp.ID)

	err = r.waitForVectorDatabaseToBeReady(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Vector Database not ready", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
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
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"VectorDatabase not found",
				fmt.Sprintf("VectorDatabase with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error getting VectorDatabase", err.Error())
		}
		return
	}
	data.Name = types.StringValue(vectorDatabase.Name)
	data.DatasetID = types.StringValue(vectorDatabase.DatasetID)
	data.UseCaseID = types.StringValue(vectorDatabase.UseCaseID)

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

	// If the chunking parameters change return an error
	// TODO : Support what can be changed or handle more gracefully
	newChunkingParameters := plan.ChunkingParameters
	if state.ChunkingParameters.EmbeddingModel.ValueString() != newChunkingParameters.EmbeddingModel.ValueString() ||
		state.ChunkingParameters.ChunkOverlapPercentage.ValueInt32() != newChunkingParameters.ChunkOverlapPercentage.ValueInt32() ||
		state.ChunkingParameters.ChunkSize.ValueInt32() != newChunkingParameters.ChunkSize.ValueInt32() ||
		state.ChunkingParameters.ChunkingMethod.ValueString() != newChunkingParameters.ChunkingMethod.ValueString() ||
		state.ChunkingParameters.IsSeparatorRegex.ValueBool() != newChunkingParameters.IsSeparatorRegex.ValueBool() ||
		!state.ChunkingParameters.Separators.Equal(newChunkingParameters.Separators) {
		resp.Diagnostics.AddError(
			"Chunking parameters change",
			"Changing the chunking parameters is not supported. Please create a new VectorDatabase.",
		)
		return
	}

	traceAPICall("UpdateVectorDatabase")
	_, err := r.provider.service.UpdateVectorDatabase(ctx,
		plan.ID.ValueString(),
		&client.UpdateVectorDatabaseRequest{
			Name: plan.Name.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"VectorDatabase not found",
				fmt.Sprintf("VectorDatabase with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating VectorDatabase", err.Error())
		}
		return
	}

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
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 30 * time.Minute

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

func (r *VectorDatabaseResource) waitForDatasetToBeReady(ctx context.Context, datasetId string) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 30 * time.Minute

	operation := func() error {
		ready, err := r.provider.service.IsDatasetReadyForVectorDatabase(ctx, datasetId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("dataset is not ready")
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
