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
var _ resource.Resource = &RegisteredModelFromLeaderboardResource{}
var _ resource.ResourceWithImportState = &RegisteredModelFromLeaderboardResource{}

func NewRegisteredModelFromLeaderboardResource() resource.Resource {
	return &RegisteredModelFromLeaderboardResource{}
}

// VectorDatabaseResource defines the resource implementation.
type RegisteredModelFromLeaderboardResource struct {
	provider *Provider
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

func (r *RegisteredModelFromLeaderboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateRegisteredModelFromLeaderboard")
	registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromLeaderboard(ctx, &client.CreateRegisteredModelFromLeaderboardRequest{
		ModelID:                       data.ModelID.ValueString(),
		RegisteredModelName:           StringValuePointerOptional(data.Name),
		PredictionThreshold:           Float64ValuePointerOptional(data.PredictionThreshold),
		ComputeAllTsIntervals:         BoolValuePointerOptional(data.ComputeAllTsIntervals),
		DistributionPredictionModelID: StringValuePointerOptional(data.DistributionPredictionModelID),
	})
	if err != nil {
		errMessage := checkNameAlreadyExists(err, data.Name.ValueString(), "Registered Model")
		resp.Diagnostics.AddError("Error creating Registered Model", errMessage)
		return
	}
	data.ID = types.StringValue(registeredModelVersion.RegisteredModelID)
	data.VersionID = types.StringValue(registeredModelVersion.ID)

	if IsKnown(data.Description) {
		traceAPICall("UpdateRegisteredModel")
		_, err := r.provider.service.UpdateRegisteredModel(ctx,
			registeredModelVersion.RegisteredModelID,
			&client.UpdateRegisteredModelRequest{
				Description: data.Description.ValueString(),
			})
		if err != nil {
			resp.Diagnostics.AddError("Error adding description to Registered Model", err.Error())
			return
		}
	}

	if IsKnown(data.VersionName) {
		traceAPICall("UpdateRegisteredModelVersion")
		registeredModelVersion, err = r.provider.service.UpdateRegisteredModelVersion(ctx, data.ID.ValueString(), data.VersionID.ValueString(), &client.UpdateRegisteredModelVersionRequest{
			Name: data.VersionName.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error adding name to Registered Model Version", err.Error())
			return
		}
	}
	data.VersionName = types.StringValue(registeredModelVersion.Name)

	err = waitForRegisteredModelVersionToBeReady(ctx, r.provider.service, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
	if err != nil {
		resp.Diagnostics.AddError("Registered model version is not ready", err.Error())
		return
	}

	for _, useCaseID := range data.UseCaseIDs {
		traceAPICall("AddRegisteredModelVersionToUseCase")
		if err = r.provider.service.AddEntityToUseCase(ctx, useCaseID.ValueString(), "registeredModelVersion", registeredModelVersion.ID); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Registered Model version to Use Case %s", useCaseID), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RegisteredModelFromLeaderboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetRegisteredModel")
	registeredModel, err := r.provider.service.GetRegisteredModel(ctx, data.ID.ValueString())
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

	traceAPICall("GetLatestRegisteredModelVersion")
	latestRegisteredModelVersion, err := r.provider.service.GetLatestRegisteredModelVersion(ctx, data.ID.ValueString())
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
	var plan RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.VersionID = state.VersionID

	traceAPICall("UpdateRegisteredModel")
	_, err := r.provider.service.UpdateRegisteredModel(ctx,
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
			errMessage := checkNameAlreadyExists(err, plan.Name.ValueString(), "Registered Model")
			resp.Diagnostics.AddError("Error updating Registered Model", errMessage)
		}
		return
	}

	if r.shouldCreateNewVersion(state, plan) {
		// create a new version of the same registered model
		traceAPICall("CreateRegisteredModelVersionFromLeaderboard")
		registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromLeaderboard(ctx, &client.CreateRegisteredModelFromLeaderboardRequest{
			ModelID:                       plan.ModelID.ValueString(),
			RegisteredModelID:             StringValuePointerOptional(plan.ID),
			PredictionThreshold:           Float64ValuePointerOptional(plan.PredictionThreshold),
			ComputeAllTsIntervals:         BoolValuePointerOptional(plan.ComputeAllTsIntervals),
			DistributionPredictionModelID: StringValuePointerOptional(plan.DistributionPredictionModelID),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Registered Model version", err.Error())
			return
		}
		plan.VersionID = types.StringValue(registeredModelVersion.ID)
	}

	if IsKnown(plan.VersionName) {
		traceAPICall("UpdateRegisteredModelVersion")
		if _, err = r.provider.service.UpdateRegisteredModelVersion(ctx, plan.ID.ValueString(), plan.VersionID.ValueString(), &client.UpdateRegisteredModelVersionRequest{
			Name: plan.VersionName.ValueString(),
		}); err != nil {
			resp.Diagnostics.AddError("Error updating Registered Model Version", err.Error())
			return
		}
	}

	err = waitForRegisteredModelVersionToBeReady(ctx, r.provider.service, plan.ID.ValueString(), plan.VersionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Registered model version not ready", err.Error())
		return
	}

	// check if we created a new version
	existingUseCaseIDs := state.UseCaseIDs
	if state.VersionID.ValueString() != plan.VersionID.ValueString() {
		existingUseCaseIDs = []types.String{}
	}

	if err = UpdateUseCasesForEntity(
		ctx,
		r.provider.service,
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
	var data RegisteredModelFromLeaderboardResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteRegisteredModel")
	err := r.provider.service.DeleteRegisteredModel(ctx, data.ID.ValueString())
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

func (r *RegisteredModelFromLeaderboardResource) shouldCreateNewVersion(state RegisteredModelFromLeaderboardResourceModel, plan RegisteredModelFromLeaderboardResourceModel) bool {
	return (IsKnown(plan.PredictionThreshold) && plan.PredictionThreshold.ValueFloat64() != state.PredictionThreshold.ValueFloat64()) ||
		(IsKnown(plan.ComputeAllTsIntervals) && plan.ComputeAllTsIntervals.ValueBool() != state.ComputeAllTsIntervals.ValueBool()) ||
		(IsKnown(plan.DistributionPredictionModelID) && plan.DistributionPredictionModelID.ValueString() != state.DistributionPredictionModelID.ValueString()) ||
		(plan.ModelID.ValueString() != state.ModelID.ValueString())
}
