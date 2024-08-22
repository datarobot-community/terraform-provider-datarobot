package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
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
	"github.com/omnistrate/terraform-provider-datarobot/internal/client"
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
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan VectorDatabaseResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var datasetID string
	if IsKnown(plan.DatasetID) {
		datasetID = plan.DatasetID.ValueString()
		_, err := r.provider.service.GetDataset(ctx, datasetID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting Dataset info",
				fmt.Sprintf("Unable to get Dataset, got error: %s", err),
			)
			return
		}
		err = r.waitForDatasetToBeReady(ctx, datasetID)
		if err != nil {
			resp.Diagnostics.AddError("Dataset not ready",
				"Dataset is not ready after 5 minutes or failed to check the status.")
			return
		}
	}

	var useCaseID string
	if IsKnown(plan.UseCaseID) {
		useCaseID = plan.UseCaseID.ValueString()
		_, err := r.provider.service.GetUseCase(ctx, useCaseID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting Use Case",
				fmt.Sprintf("Unable to get Use Case, got error: %s", err),
			)
			return
		}
	}

	chunkingParameters := plan.ChunkingParameters
	separators := make([]string, 0)
	for _, separator := range chunkingParameters.Separators.Elements() {
		separators = append(separators, separator.String())
	}

	traceAPICall("CreateVectorDatabase")
	createResp, err := r.provider.service.CreateVectorDatabase(ctx, &client.CreateVectorDatabaseRequest{
		DatasetID: datasetID,
		UseCaseID: useCaseID,
		Name:      plan.Name.ValueString(),
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
		resp.Diagnostics.AddError(
			"Error creating VectorDatabase",
			fmt.Sprintf("Unable to create VectorDatabase, got error: %s", err),
		)
		return
	}

	traceAPICall("GetVectorDatabase")
	getVectorDatabase, err := r.provider.service.GetVectorDatabase(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting VectorDatabase info",
			fmt.Sprintf("Unable to get VectorDatabase, got error: %s", err),
		)
		return
	}

	// Wait for the VectorDatabase to be ready
	err = r.waitForVectorDatabaseToBeReady(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Vector Database not ready",
			"Vector Database is not ready after 5 minutes or failed to check the status.")
		return
	}

	var state VectorDatabaseResourceModel

	loadVectorDatabaseFromFileToTerraformState(
		getVectorDatabase.ID,
		getVectorDatabase.Name,
		datasetID,
		useCaseID,
		chunkingParameters,
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VectorDatabaseResource) waitForVectorDatabaseToBeReady(ctx context.Context, vectorDatabaseId string) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 5 * time.Minute

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
	expBackoff.MaxElapsedTime = 5 * time.Minute

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

func (r *VectorDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state VectorDatabaseResourceModel
	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		return
	}

	id := state.ID.ValueString()
	useCaseId := state.UseCaseID.ValueString()
	datasetId := state.DatasetID.ValueString()
	chunkingParameters := state.ChunkingParameters

	traceAPICall("GetVectorDatabase")
	vectorDatabase, err := r.provider.service.GetVectorDatabase(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"VectorDatabase not found",
				fmt.Sprintf("VectorDatabase with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting VectorDatabase info",
				fmt.Sprintf("Unable to get VectorDatabase, got error: %s", err),
			)
		}
		return
	}

	loadVectorDatabaseFromFileToTerraformState(
		id,
		vectorDatabase.Name,
		datasetId,
		useCaseId,
		chunkingParameters,
		&state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VectorDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan VectorDatabaseResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state VectorDatabaseResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the use case id changes return and error
	// TODO : If the use case change we need to recreate the VectorDatabase
	newUseCaseID := plan.UseCaseID.ValueString()
	if state.UseCaseID.ValueString() != newUseCaseID {
		resp.Diagnostics.AddError(
			"Use Case ID change",
			"Changing the Use Case ID is not supported. Please create a new VectorDatabase.",
		)
		return
	}

	// If the data source id changes return an error
	// TODO : If the data source change we need to recreate the VectorDatabase
	newDatasetID := plan.DatasetID.ValueString()
	if state.DatasetID.ValueString() != newDatasetID {
		resp.Diagnostics.AddError(
			"Source file change",
			"Changing the source file is not supported. Please create a new VectorDatabase.",
		)
		return
	}

	// It he only fields that can be updated don't change, just return.
	newName := plan.Name.ValueString()
	if state.Name.ValueString() == newName {
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

	id := state.ID.ValueString()

	traceAPICall("UpdateVectorDatabase")
	useCase, err := r.provider.service.UpdateVectorDatabase(ctx,
		id,
		&client.UpdateVectorDatabaseRequest{
			Name: plan.Name.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"VectorDatabase not found",
				fmt.Sprintf("VectorDatabase with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating VectorDatabase",
				fmt.Sprintf("Unable to update VectorDatabase, got error: %s", err),
			)
		}
		return
	}

	loadVectorDatabaseFromFileToTerraformState(
		id,
		useCase.Name,
		newDatasetID,
		newUseCaseID,
		plan.ChunkingParameters,
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VectorDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state VectorDatabaseResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("DeleteVectorDatabase")
	err := r.provider.service.DeleteVectorDatabase(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// use case is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting VectorDatabase info",
				fmt.Sprintf("Unable to get  example, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *VectorDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadVectorDatabaseFromFileToTerraformState(
	id, name, dataSourceId, useCaseId string,
	chunkingParameters ChunkingParametersModel,
	state *VectorDatabaseResourceModel,
) {
	state.ID = types.StringValue(id)
	if name != "" {
		state.Name = types.StringValue(name)
	} else {
		state.Name = types.StringNull()
	}
	state.DatasetID = types.StringValue(dataSourceId)
	state.UseCaseID = types.StringValue(useCaseId)
	state.ChunkingParameters = chunkingParameters
}
