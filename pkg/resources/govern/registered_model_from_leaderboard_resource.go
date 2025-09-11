package govern

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
var _ resource.Resource = &RegisteredModelFromLeaderboardResource{}
var _ resource.ResourceWithImportState = &RegisteredModelFromLeaderboardResource{}

func NewRegisteredModelFromLeaderboardResource() resource.Resource {
	return &RegisteredModelFromLeaderboardResource{}
}

// VectorDatabaseResource defines the resource implementation.
type RegisteredModelFromLeaderboardResource struct {
	service client.Service
}

func (r *RegisteredModelFromLeaderboardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registered_model_from_leaderboard"
}

func (r *RegisteredModelFromLeaderboardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "registered model from leaderboard",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Registered Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Registered Model.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the Registered Model.",
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Registered Model Version.",
			},
			"version_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the Registered Model Version.",
			},
			"model_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the DataRobot model for this Registered Model.",
			},
			"prediction_threshold": schema.Float64Attribute{
				Optional:            true,
				MarkdownDescription: "The prediction threshold for the model.",
			},
			"compute_all_ts_intervals": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to compute all time series intervals (1-100 percentiles).",
			},
			"distribution_prediction_model_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the DataRobot distribution prediction model trained on predictions from the DataRobot model.",
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Registered Model version to.",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *RegisteredModelFromLeaderboardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *RegisteredModelFromLeaderboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("CreateRegisteredModelFromLeaderboard")
	registeredModelVersion, err := r.service.CreateRegisteredModelFromLeaderboard(ctx, &client.CreateRegisteredModelFromLeaderboardRequest{
		ModelID:                       data.ModelID.ValueString(),
		RegisteredModelName:           common.StringValuePointerOptional(data.Name),
		PredictionThreshold:           common.Float64ValuePointerOptional(data.PredictionThreshold),
		ComputeAllTsIntervals:         common.BoolValuePointerOptional(data.ComputeAllTsIntervals),
		DistributionPredictionModelID: common.StringValuePointerOptional(data.DistributionPredictionModelID),
	})
	if err != nil {
		errMessage := common.CheckRegisteredModelNameAlreadyExists(err, data.Name.ValueString())
		resp.Diagnostics.AddError("Error creating Registered Model", errMessage)
		return
	}
	data.ID = types.StringValue(registeredModelVersion.RegisteredModelID)
	data.VersionID = types.StringValue(registeredModelVersion.ID)

	if common.IsKnown(data.Description) {
		common.TraceAPICall("UpdateRegisteredModel")
		_, err := r.service.UpdateRegisteredModel(ctx,
			registeredModelVersion.RegisteredModelID,
			&client.UpdateRegisteredModelRequest{
				Description: data.Description.ValueString(),
			})
		if err != nil {
			resp.Diagnostics.AddError("Error adding description to Registered Model", err.Error())
			return
		}
	}

	if common.IsKnown(data.VersionName) {
		common.TraceAPICall("UpdateRegisteredModelVersion")
		registeredModelVersion, err = r.service.UpdateRegisteredModelVersion(ctx, data.ID.ValueString(), data.VersionID.ValueString(), &client.UpdateRegisteredModelVersionRequest{
			Name: data.VersionName.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error adding name to Registered Model Version", err.Error())
			return
		}
	}
	data.VersionName = types.StringValue(registeredModelVersion.Name)

	err = common.WaitForRegisteredModelVersionToBeReady(ctx, r.service, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
	if err != nil {
		resp.Diagnostics.AddError("Registered model version is not ready", err.Error())
		return
	}

	for _, useCaseID := range data.UseCaseIDs {
		common.TraceAPICall("AddRegisteredModelVersionToUseCase")
		if err = common.AddEntityToUseCase(
			ctx,
			r.service,
			useCaseID.ValueString(),
			"registeredModelVersion",
			registeredModelVersion.ID,
		); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Registered Model version to Use Case %s", useCaseID), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RegisteredModelFromLeaderboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	common.TraceAPICall("GetRegisteredModel")
	registeredModel, err := r.service.GetRegisteredModel(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Registered Model not found",
				fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Registered Model with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Name = types.StringValue(registeredModel.Name)
	if registeredModel.Description != "" {
		data.Description = types.StringValue(registeredModel.Description)
	}

	common.TraceAPICall("GetLatestRegisteredModelVersion")
	latestRegisteredModelVersion, err := r.service.GetLatestRegisteredModelVersion(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting Registered Model Version", err.Error())
		return
	}
	data.VersionID = types.StringValue(latestRegisteredModelVersion.ID)
	data.VersionName = types.StringValue(latestRegisteredModelVersion.Name)
	data.ModelID = types.StringValue(latestRegisteredModelVersion.ModelID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegisteredModelFromLeaderboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.VersionID = state.VersionID

	common.TraceAPICall("UpdateRegisteredModel")
	_, err := r.service.UpdateRegisteredModel(ctx,
		plan.ID.ValueString(),
		&client.UpdateRegisteredModelRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Registered Model not found",
				fmt.Sprintf("Registered Model with ID %s is not found. Removing from state.", plan.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			errMessage := common.CheckRegisteredModelNameAlreadyExists(err, plan.Name.ValueString())
			resp.Diagnostics.AddError("Error updating Registered Model", errMessage)
		}
		return
	}

	var registeredModelVersion *client.RegisteredModelVersion
	if r.shouldCreateNewVersion(state, plan) {
		// create a new version of the same registered model
		common.TraceAPICall("CreateRegisteredModelVersionFromLeaderboard")
		registeredModelVersion, err = r.service.CreateRegisteredModelFromLeaderboard(ctx, &client.CreateRegisteredModelFromLeaderboardRequest{
			ModelID:                       plan.ModelID.ValueString(),
			RegisteredModelID:             common.StringValuePointerOptional(plan.ID),
			PredictionThreshold:           common.Float64ValuePointerOptional(plan.PredictionThreshold),
			ComputeAllTsIntervals:         common.BoolValuePointerOptional(plan.ComputeAllTsIntervals),
			DistributionPredictionModelID: common.StringValuePointerOptional(plan.DistributionPredictionModelID),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Registered Model version", err.Error())
			return
		}
		plan.VersionID = types.StringValue(registeredModelVersion.ID)
	}

	if common.IsKnown(plan.VersionName) {
		common.TraceAPICall("UpdateRegisteredModelVersion")
		if registeredModelVersion, err = r.service.UpdateRegisteredModelVersion(ctx, plan.ID.ValueString(), plan.VersionID.ValueString(), &client.UpdateRegisteredModelVersionRequest{
			Name: plan.VersionName.ValueString(),
		}); err != nil {
			resp.Diagnostics.AddError("Error updating Registered Model Version", err.Error())
			return
		}
	}
	plan.VersionName = types.StringValue(registeredModelVersion.Name)

	err = common.WaitForRegisteredModelVersionToBeReady(ctx, r.service, plan.ID.ValueString(), plan.VersionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Registered model version not ready", err.Error())
		return
	}

	// check if we created a new version
	existingUseCaseIDs := state.UseCaseIDs
	if state.VersionID.ValueString() != plan.VersionID.ValueString() {
		existingUseCaseIDs = []types.String{}
	}

	if err = common.UpdateUseCasesForEntity(
		ctx,
		r.service,
		"registeredModelVersion",
		plan.VersionID.ValueString(),
		existingUseCaseIDs,
		plan.UseCaseIDs,
	); err != nil {
		resp.Diagnostics.AddError("Error updating Use Cases for Registered Model version", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *RegisteredModelFromLeaderboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteRegisteredModel")
	err := r.service.DeleteRegisteredModel(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Registered Model", err.Error())
			return
		}
	}
}

func (r *RegisteredModelFromLeaderboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RegisteredModelFromLeaderboardResource) shouldCreateNewVersion(state models.RegisteredModelFromLeaderboardResourceModel, plan models.RegisteredModelFromLeaderboardResourceModel) bool {
	return (common.IsKnown(plan.PredictionThreshold) && plan.PredictionThreshold.ValueFloat64() != state.PredictionThreshold.ValueFloat64()) ||
		(common.IsKnown(plan.ComputeAllTsIntervals) && plan.ComputeAllTsIntervals.ValueBool() != state.ComputeAllTsIntervals.ValueBool()) ||
		(common.IsKnown(plan.DistributionPredictionModelID) && plan.DistributionPredictionModelID.ValueString() != state.DistributionPredictionModelID.ValueString()) ||
		(plan.ModelID.ValueString() != state.ModelID.ValueString())
}
