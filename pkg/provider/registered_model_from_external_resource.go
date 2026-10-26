package provider

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strconv"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RegisteredModelFromExternalResource{}
var _ resource.ResourceWithImportState = &RegisteredModelFromExternalResource{}
var _ resource.ResourceWithValidateConfig = &RegisteredModelFromExternalResource{}

var externalModelTargetTypes = []string{
	"Binary",
	"Regression",
	"Multiclass",
	"Multilabel",
	"TextGeneration",
	"GeoPoint",
	"AgenticWorkflow",
	"MCP",
}

// Target types for which the API does not require a target name.
var externalModelTargetTypesWithoutName = []string{"AgenticWorkflow", "MCP"}

func NewRegisteredModelFromExternalResource() resource.Resource {
	return &RegisteredModelFromExternalResource{}
}

// RegisteredModelFromExternalResource defines the resource implementation.
type RegisteredModelFromExternalResource struct {
	provider *Provider
}

func (r *RegisteredModelFromExternalResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registered_model_from_external"
}

func (r *RegisteredModelFromExternalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Registered Model for a model that runs outside of DataRobot (an external model), created from a JSON description of the model package.\n\n" +
			"Changing `name`, `description`, `version_name` or `model_description.description` updates the Registered Model or its latest version in place. " +
			"Changing `target.type`, `target.name`, `target.class_names`, adding or removing `timeseries`, or changing the time series fields that DataRobot requires to match across versions " +
			"replaces the Registered Model. Any other change creates a new version of the Registered Model.\n\n" +
			"On import, the model package fields are read from the latest version. A value DataRobot uses by default is imported as unset: " +
			"a `model_id` equal to the version ID, a `model_description.model_name` equal to the model ID, a `model_description.build_environment_type` of `Other`, " +
			"and `false` for `timeseries.is_new_series_support` and `timeseries.is_traditional_time_series`. `use_case_ids` is not imported.",

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
				MarkdownDescription: "The name of the Registered Model Version. If not set, a new version is named `<name> (v<version number>)`.",
			},
			"model_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the external model. Changing it creates a new version.",
			},
			"target": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The target information for the model package.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The target type of the model. Changing it replaces the Registered Model.",
						Validators: []validator.String{
							stringvalidator.OneOf(externalModelTargetTypes...),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"name": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The name of the target column. Required unless `type` is `AgenticWorkflow` or `MCP`. Changing it replaces the Registered Model.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"class_names": schema.ListAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "The class names for prediction results. For a `Binary` target, the first element is the minority (positive) class and the second is the majority (negative) class. Changing it replaces the Registered Model.",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
					},
					"prediction_threshold": schema.Float64Attribute{
						Optional:            true,
						MarkdownDescription: "The prediction threshold used for binary classification models, between 0 and 1.",
						Validators: []validator.Float64{
							float64validator.Between(0, 1),
						},
					},
					"prediction_probabilities_column": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The column containing prediction probabilities.",
					},
				},
			},
			"model_description": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The model description information for the model package.",
				Attributes: map[string]schema.Attribute{
					"model_name": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The model name.",
					},
					"description": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "A description of the model. Changing it updates the latest version in place; removing it creates a new version.",
					},
					"location": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The location of the model.",
					},
					"build_environment_type": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The build environment type of the model.",
						Validators: []validator.String{
							stringvalidator.OneOf("DataRobot", "Python", "R", "Java", "Julia", "Legacy", "Other"),
						},
					},
				},
			},
			"datasets": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The datasets for the model package. `target.name` is required when training or holdout data is set, whatever the target type. Not allowed when `target.type` is `AgenticWorkflow`.",
				Attributes: map[string]schema.Attribute{
					"training_data_catalog_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The ID of the training dataset.",
					},
					"training_data_catalog_version_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The version ID of the training dataset.",
					},
					"holdout_data_catalog_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The ID of the holdout dataset.",
					},
					"holdout_data_catalog_version_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The version ID of the holdout dataset.",
					},
				},
			},
			"timeseries": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The time series information for the model package. Adding or removing it replaces the Registered Model. Not allowed when `target.type` is `AgenticWorkflow`.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplaceIf(
						requiresReplaceIfObjectAddedOrRemoved,
						"Adding or removing timeseries replaces the Registered Model.",
						"Adding or removing `timeseries` replaces the Registered Model.",
					),
				},
				Attributes: map[string]schema.Attribute{
					"datetime_column_name": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The name of the forecast date column. Changing it replaces the Registered Model.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"datetime_column_format": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The date format of the forecast date and forecast point columns. Changing it replaces the Registered Model.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"forecast_point_column_name": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The name of the forecast point column.",
					},
					"forecast_distance_column_name": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The name of the forecast distance column.",
					},
					"forecast_distances_time_unit": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The time unit of the forecast distances. Changing it replaces the Registered Model.",
						Validators: []validator.String{
							stringvalidator.OneOf("MICROSECOND", "MILLISECOND", "SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "QUARTER", "YEAR"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"forecast_distances": schema.ListAttribute{
						Optional:            true,
						ElementType:         types.Int64Type,
						MarkdownDescription: "The list of forecast distances. Changing it replaces the Registered Model.",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
					},
					"series_column_name": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The name of the series column of a multiseries model. Setting or removing it replaces the Registered Model.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplaceIf(
								requiresReplaceIfStringAddedOrRemoved,
								"Setting or removing series_column_name replaces the Registered Model.",
								"Setting or removing `series_column_name` replaces the Registered Model.",
							),
						},
					},
					"feature_derivation_window_start": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "A negative number or zero: the start of the feature derivation window, relative to the forecast point, in forecast distance time units.",
						Validators: []validator.Int64{
							int64validator.AtMost(0),
						},
					},
					"feature_derivation_window_end": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "A negative number or zero: the end of the feature derivation window, relative to the forecast point, in forecast distance time units.",
						Validators: []validator.Int64{
							int64validator.AtMost(0),
						},
					},
					"effective_feature_derivation_window_start": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "A negative number or zero: the start of the effective feature derivation window.",
						Validators: []validator.Int64{
							int64validator.AtMost(0),
						},
					},
					"effective_feature_derivation_window_end": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "A negative number or zero: the end of the effective feature derivation window.",
						Validators: []validator.Int64{
							int64validator.AtMost(0),
						},
					},
					"is_cross_series": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether the model is cross-series.",
					},
					"is_new_series_support": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether the model is optimized to support new series.",
					},
					"is_traditional_time_series": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether the model is a traditional time series model.",
					},
				},
			},
			"text_generation": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The text generation information for the model package.",
				Attributes: map[string]schema.Attribute{
					"prompt": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The name of the prompt column.",
					},
				},
			},
			"geospatial_monitoring": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The geospatial monitoring information for the model package. Not allowed when `target.type` is `AgenticWorkflow`.",
				Attributes: map[string]schema.Attribute{
					"primary_location_column": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The name of the geospatial column to analyze.",
					},
				},
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Registered Model version to.",
				ElementType:         types.StringType,
			},
			"tags": schema.SetNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The list of tags to assign to the Registered Model version. Changing it creates a new version.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the tag.",
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The value of the tag.",
						},
					},
				},
			},
		},
	}
}

func (r *RegisteredModelFromExternalResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RegisteredModelFromExternalResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var targetType, targetName types.String
	var trainingDataID, trainingDataVersionID, holdoutDataID, holdoutDataVersionID types.String
	var timeseries, geospatialMonitoring types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("target").AtName("type"), &targetType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("target").AtName("name"), &targetName)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("datasets").AtName("training_data_catalog_id"), &trainingDataID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("datasets").AtName("training_data_catalog_version_id"), &trainingDataVersionID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("datasets").AtName("holdout_data_catalog_id"), &holdoutDataID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("datasets").AtName("holdout_data_catalog_version_id"), &holdoutDataVersionID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeseries"), &timeseries)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("geospatial_monitoring"), &geospatialMonitoring)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The checks below mirror the ones DataRobot makes on create, so that these
	// configurations fail at plan instead of at apply. Unknown values count as set,
	// since dataset IDs usually come from another resource.
	targetNamePath := path.Root("target").AtName("name")
	hasTrainingOrHoldoutData := !trainingDataID.IsNull() || !holdoutDataID.IsNull()
	switch {
	case !targetName.IsNull():
		// target.name is set, or will be.
	case IsKnown(targetType) && !contains(externalModelTargetTypesWithoutName, targetType.ValueString()):
		resp.Diagnostics.AddAttributeError(
			targetNamePath,
			"Missing target name",
			fmt.Sprintf("target.name is required when target.type is %q. Only AgenticWorkflow and MCP targets can omit it.", targetType.ValueString()),
		)
	case hasTrainingOrHoldoutData:
		resp.Diagnostics.AddAttributeError(
			targetNamePath,
			"Missing target name",
			"target.name is required when datasets has training or holdout data, whatever the target type.",
		)
	}

	if !IsKnown(targetType) || targetType.ValueString() != "AgenticWorkflow" {
		return
	}
	// datasets is checked by its IDs rather than as a block: a block with no IDs
	// is sent as an empty object, which DataRobot accepts.
	if hasTrainingOrHoldoutData || !trainingDataVersionID.IsNull() || !holdoutDataVersionID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("datasets"),
			"Datasets not allowed",
			"An AgenticWorkflow target is not compatible with training or holdout data.",
		)
	}
	if !timeseries.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("timeseries"),
			"Time series not allowed",
			"An AgenticWorkflow target is not compatible with timeseries.",
		)
	}
	if !geospatialMonitoring.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("geospatial_monitoring"),
			"Geospatial monitoring not allowed",
			"An AgenticWorkflow target is not compatible with geospatial_monitoring.",
		)
	}
}

func (r *RegisteredModelFromExternalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RegisteredModelFromExternalResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize Tags to ensure it has the correct type
	var diags diag.Diagnostics
	data.Tags, diags = normalizeTagsSet(ctx, data.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := buildExternalModelPackageRequest(data)
	createRequest.Name = getExternalModelVersionName(data, 1)
	createRequest.RegisteredModelName = StringValuePointerOptional(data.Name)
	createRequest.RegisteredModelDescription = StringValuePointerOptional(data.Description)
	createRequest.Tags = convertSetTagsToClientTags(data.Tags)

	traceAPICall("CreateRegisteredModelFromExternal")
	registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromExternal(ctx, createRequest)
	if err != nil {
		errMessage := checkNameAlreadyExists(err, data.Name.ValueString(), "Registered Model")
		resp.Diagnostics.AddError("Error creating Registered Model", errMessage)
		return
	}
	data.ID = types.StringValue(registeredModelVersion.RegisteredModelID)
	data.VersionID = types.StringValue(registeredModelVersion.ID)
	data.VersionName = types.StringValue(registeredModelVersion.Name)

	// External model packages are not built, so unlike the other Registered Model
	// resources there is no need to wait for the version to be ready.

	for _, useCaseID := range data.UseCaseIDs {
		traceAPICall("AddRegisteredModelVersionToUseCase")
		if err = addEntityToUseCase(
			ctx,
			r.provider.service,
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

func (r *RegisteredModelFromExternalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RegisteredModelFromExternalResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize Tags to ensure it has the correct type
	var diags diag.Diagnostics
	data.Tags, diags = normalizeTagsSet(ctx, data.Tags)
	resp.Diagnostics.Append(diags...)
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

	// target is required, so it is only missing from state right after an import.
	// Otherwise the model package fields are kept as configured, like the other
	// Registered Model resources do.
	if data.Target == nil {
		traceAPICall("GetRegisteredModelVersion")
		registeredModelVersion, err := r.provider.service.GetRegisteredModelVersion(ctx, data.ID.ValueString(), latestRegisteredModelVersion.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error getting Registered Model Version", err.Error())
			return
		}
		setExternalModelPackageFromVersion(ctx, &data, registeredModelVersion, &resp.Diagnostics)
	}

	if len(latestRegisteredModelVersion.Tags) > 0 {
		data.Tags = initializeTagsFromModel(latestRegisteredModelVersion.Tags, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		data.Tags = types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegisteredModelFromExternalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RegisteredModelFromExternalResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RegisteredModelFromExternalResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize Tags from plan and state to ensure they have the correct type
	var diags diag.Diagnostics
	plan.Tags, diags = normalizeTagsSet(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	state.Tags, diags = normalizeTagsSet(ctx, state.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.VersionID = state.VersionID

	if plan.Name.ValueString() != state.Name.ValueString() ||
		plan.Description.ValueString() != state.Description.ValueString() {
		// The description is always sent, so that removing it from the configuration
		// clears it: the API accepts an empty description.
		description := plan.Description.ValueString()
		traceAPICall("UpdateRegisteredModel")
		_, err := r.provider.service.UpdateRegisteredModel(ctx,
			plan.ID.ValueString(),
			&client.UpdateRegisteredModelRequest{
				Name:        plan.Name.ValueString(),
				Description: &description,
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
	}

	if shouldCreateNewExternalModelVersion(state, plan) {
		versionName := plan.VersionName.ValueString()
		if !IsKnown(plan.VersionName) {
			traceAPICall("GetRegisteredModel")
			registeredModel, err := r.provider.service.GetRegisteredModel(ctx, plan.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error getting Registered Model", err.Error())
				return
			}
			versionName = getExternalModelVersionName(plan, registeredModel.LastVersionNum+1)
		}

		// create a new version of the same registered model
		newVersionRequest := buildExternalModelPackageRequest(plan)
		newVersionRequest.Name = versionName
		newVersionRequest.RegisteredModelID = StringValuePointerOptional(plan.ID)
		newVersionRequest.Tags = convertSetTagsToClientTags(plan.Tags)

		traceAPICall("CreateRegisteredModelVersionFromExternal")
		registeredModelVersion, err := r.provider.service.CreateRegisteredModelFromExternal(ctx, newVersionRequest)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Registered Model version", err.Error())
			return
		}
		plan.VersionID = types.StringValue(registeredModelVersion.ID)
		plan.VersionName = types.StringValue(registeredModelVersion.Name)
	} else {
		if !IsKnown(plan.VersionName) {
			plan.VersionName = state.VersionName
		}

		updateVersionRequest := &client.UpdateRegisteredModelVersionRequest{}
		if plan.VersionName.ValueString() != state.VersionName.ValueString() {
			updateVersionRequest.Name = plan.VersionName.ValueString()
		}
		if planDescription := getExternalModelVersionDescription(plan); planDescription != getExternalModelVersionDescription(state) {
			updateVersionRequest.Description = planDescription
		}

		if updateVersionRequest.Name != "" || updateVersionRequest.Description != "" {
			traceAPICall("UpdateRegisteredModelVersion")
			if _, err := r.provider.service.UpdateRegisteredModelVersion(ctx, plan.ID.ValueString(), plan.VersionID.ValueString(), updateVersionRequest); err != nil {
				resp.Diagnostics.AddError("Error updating Registered Model Version", err.Error())
				return
			}
		}
	}

	// check if we created a new version
	existingUseCaseIDs := state.UseCaseIDs
	if state.VersionID.ValueString() != plan.VersionID.ValueString() {
		existingUseCaseIDs = []types.String{}
	}

	if err := updateUseCasesForEntity(
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

func (r *RegisteredModelFromExternalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RegisteredModelFromExternalResourceModel

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

func (r *RegisteredModelFromExternalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func requiresReplaceIfObjectAddedOrRemoved(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.StateValue.IsNull() != req.PlanValue.IsNull()
}

func requiresReplaceIfStringAddedOrRemoved(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.StateValue.IsNull() != req.PlanValue.IsNull()
}

func getExternalModelVersionName(data RegisteredModelFromExternalResourceModel, versionNum int) string {
	if IsKnown(data.VersionName) {
		return data.VersionName.ValueString()
	}

	return fmt.Sprintf("%s (v%d)", data.Name.ValueString(), versionNum)
}

func getExternalModelVersionDescription(data RegisteredModelFromExternalResourceModel) string {
	if data.ModelDescription == nil {
		return ""
	}

	return data.ModelDescription.Description.ValueString()
}

// shouldCreateNewExternalModelVersion reports whether the planned changes can only be applied
// by adding a new version. Changes that would make the new version incompatible with the
// previous one replace the resource instead, through plan modifiers in the schema.
func shouldCreateNewExternalModelVersion(state RegisteredModelFromExternalResourceModel, plan RegisteredModelFromExternalResourceModel) bool {
	if !plan.Tags.Equal(state.Tags) {
		return true
	}

	planPackage := buildExternalModelPackageRequest(plan)
	statePackage := buildExternalModelPackageRequest(state)

	// The version description is updated in place, unless it is being removed:
	// the API cannot clear it, so removing it takes a new version.
	if getExternalModelVersionDescription(plan) != "" {
		clearExternalModelVersionDescription(planPackage)
		clearExternalModelVersionDescription(statePackage)
	}

	return !reflect.DeepEqual(planPackage, statePackage)
}

func clearExternalModelVersionDescription(request *client.CreateRegisteredModelFromExternalRequest) {
	if request.ModelDescription == nil {
		return
	}

	modelDescription := *request.ModelDescription
	modelDescription.Description = nil
	if modelDescription == (client.ExternalModelPackageDescription{}) {
		request.ModelDescription = nil
	} else {
		request.ModelDescription = &modelDescription
	}
}

// setExternalModelPackageFromVersion fills in the model package fields of an imported resource
// from its latest version. Where the API filled in a value because the request left it out,
// the field is left null, so that a configuration that also leaves it out shows no change.
func setExternalModelPackageFromVersion(ctx context.Context, data *RegisteredModelFromExternalResourceModel, version *client.RegisteredModelVersion, diags *diag.Diagnostics) {
	data.Target = &ExternalModelTarget{
		Type:                          types.StringValue(version.Target.Type),
		Name:                          stringValueOrNull(version.Target.Name),
		PredictionThreshold:           float64PointerValueAsConfigured(ctx, version.Target.PredictionThreshold, diags),
		PredictionProbabilitiesColumn: StringPointerValue(version.Target.PredictionProbabilitiesColumn),
	}
	if classNames := version.Target.ClassNames; len(classNames) > 0 {
		if version.Target.ClassCount != nil && *version.Target.ClassCount > len(classNames) {
			diags.AddWarning(
				"Target class names not imported",
				fmt.Sprintf("The API returns only the first %d of the %d class names of the Registered Model, so target.class_names is not imported.", len(classNames), *version.Target.ClassCount))
		} else {
			classNames = append([]string(nil), classNames...)
			// The API returns Binary class names in the order they were sent, and
			// Multiclass and Multilabel class names in reverse order.
			if version.Target.Type != "Binary" {
				slices.Reverse(classNames)
			}
			data.Target.ClassNames = convertToTfStringList(classNames)
		}
	}

	// Without a model ID, the API uses the ID of the model package, that is, of the version.
	data.ModelID = types.StringNull()
	if version.ModelID != version.ID {
		data.ModelID = types.StringValue(version.ModelID)
	}

	data.ModelDescription = nil
	if modelDescription := version.ModelDescription; modelDescription != nil {
		importedModelDescription := &ExternalModelDescription{
			ModelName:            types.StringNull(),
			Description:          StringPointerValue(modelDescription.Description),
			Location:             StringPointerValue(modelDescription.Location),
			BuildEnvironmentType: types.StringNull(),
		}
		// Without a model name, the API uses the model ID.
		if modelDescription.ModelName != version.ModelID {
			importedModelDescription.ModelName = stringValueOrNull(modelDescription.ModelName)
		}
		// Without a build environment type, the API uses "Other".
		if modelDescription.BuildEnvironmentType != "Other" {
			importedModelDescription.BuildEnvironmentType = stringValueOrNull(modelDescription.BuildEnvironmentType)
		}
		if *importedModelDescription != (ExternalModelDescription{
			ModelName:            types.StringNull(),
			Description:          types.StringNull(),
			Location:             types.StringNull(),
			BuildEnvironmentType: types.StringNull(),
		}) {
			data.ModelDescription = importedModelDescription
		}
	}

	data.Datasets = nil
	if datasets := version.Datasets; datasets != nil &&
		(datasets.TrainingDataCatalogID != nil || datasets.TrainingDataCatalogVersionID != nil ||
			datasets.HoldoutDataCatalogID != nil || datasets.HoldoutDataCatalogVersionID != nil) {
		data.Datasets = &ExternalModelDatasets{
			TrainingDataCatalogID:        StringPointerValue(datasets.TrainingDataCatalogID),
			TrainingDataCatalogVersionID: StringPointerValue(datasets.TrainingDataCatalogVersionID),
			HoldoutDataCatalogID:         StringPointerValue(datasets.HoldoutDataCatalogID),
			HoldoutDataCatalogVersionID:  StringPointerValue(datasets.HoldoutDataCatalogVersionID),
		}
	}

	// The API returns timeseries for every model package, with null fields when the model is
	// not a time series model.
	data.Timeseries = nil
	if timeseries := version.Timeseries; timeseries != nil && timeseries.DatetimeColumnName != nil {
		data.Timeseries = &ExternalModelTimeseries{
			DatetimeColumnName:                    StringPointerValue(timeseries.DatetimeColumnName),
			DatetimeColumnFormat:                  StringPointerValue(timeseries.DatetimeColumnFormat),
			ForecastPointColumnName:               StringPointerValue(timeseries.ForecastPointColumnName),
			ForecastDistanceColumnName:            StringPointerValue(timeseries.ForecastDistanceColumnName),
			ForecastDistancesTimeUnit:             StringPointerValue(timeseries.ForecastDistancesTimeUnit),
			SeriesColumnName:                      StringPointerValue(timeseries.SeriesColumnName),
			FeatureDerivationWindowStart:          Int64PointerValue(timeseries.FeatureDerivationWindowStart),
			FeatureDerivationWindowEnd:            Int64PointerValue(timeseries.FeatureDerivationWindowEnd),
			EffectiveFeatureDerivationWindowStart: Int64PointerValue(timeseries.EffectiveFeatureDerivationWindowStart),
			EffectiveFeatureDerivationWindowEnd:   Int64PointerValue(timeseries.EffectiveFeatureDerivationWindowEnd),
			IsCrossSeries:                         BoolPointerValue(timeseries.IsCrossSeries),
			// Without a value, the API uses false for these two.
			IsNewSeriesSupport:      boolValueOrNullIfFalse(timeseries.IsNewSeriesSupport),
			IsTraditionalTimeSeries: boolValueOrNullIfFalse(timeseries.IsTraditionalTimeSeries),
		}
		for _, forecastDistance := range timeseries.ForecastDistances {
			data.Timeseries.ForecastDistances = append(data.Timeseries.ForecastDistances, types.Int64Value(forecastDistance))
		}
	}

	data.TextGeneration = nil
	if prompt := version.TextGeneration.Prompt; prompt != nil && *prompt != "" {
		data.TextGeneration = &ExternalModelTextGeneration{
			Prompt: types.StringValue(*prompt),
		}
	}

	// The API stores geospatial monitoring for every model package, with a null column
	// when none was sent.
	data.GeospatialMonitoring = nil
	if geospatial := version.GeospatialMonitoring; geospatial != nil && geospatial.PrimaryLocationColumn != nil && *geospatial.PrimaryLocationColumn != "" {
		data.GeospatialMonitoring = &ExternalModelGeospatial{
			PrimaryLocationColumn: types.StringValue(*geospatial.PrimaryLocationColumn),
		}
	}
}

// float64PointerValueAsConfigured returns the value Terraform reads from a configuration that
// sets the number. Terraform parses numbers in configuration with 512 bits of precision, so a
// float64 such as 0.6 from the API would otherwise differ from a configured 0.6 and plan a change.
func float64PointerValueAsConfigured(ctx context.Context, value *float64, diags *diag.Diagnostics) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}

	configured, _, err := big.ParseFloat(strconv.FormatFloat(*value, 'g', -1, 64), 10, 512, big.ToNearestEven)
	if err != nil {
		diags.AddError("Error converting number", err.Error())
		return types.Float64Null()
	}

	attrValue, err := types.Float64Type.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.Number, configured))
	if err != nil {
		diags.AddError("Error converting number", err.Error())
		return types.Float64Null()
	}

	float64Value, ok := attrValue.(types.Float64)
	if !ok {
		diags.AddError("Error converting number", fmt.Sprintf("Expected a Float64 value, got %T.", attrValue))
		return types.Float64Null()
	}

	return float64Value
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func boolValueOrNullIfFalse(value *bool) types.Bool {
	if value == nil || !*value {
		return types.BoolNull()
	}
	return types.BoolValue(true)
}

// buildExternalModelPackageRequest converts the model package fields of the resource into a
// request. The version name and the Registered Model fields are set by the caller.
func buildExternalModelPackageRequest(data RegisteredModelFromExternalResourceModel) *client.CreateRegisteredModelFromExternalRequest {
	request := &client.CreateRegisteredModelFromExternalRequest{
		ModelID: StringValuePointerOptional(data.ModelID),
	}

	if data.Target != nil {
		request.Target = client.ExternalModelPackageTarget{
			Type:                          data.Target.Type.ValueString(),
			Name:                          StringValuePointerOptional(data.Target.Name),
			PredictionThreshold:           Float64ValuePointerOptional(data.Target.PredictionThreshold),
			PredictionProbabilitiesColumn: StringValuePointerOptional(data.Target.PredictionProbabilitiesColumn),
		}
		if len(data.Target.ClassNames) > 0 {
			request.Target.ClassNames = convertTfStringList(data.Target.ClassNames)
		}
	}

	if data.ModelDescription != nil {
		request.ModelDescription = &client.ExternalModelPackageDescription{
			ModelName:            StringValuePointerOptional(data.ModelDescription.ModelName),
			Description:          StringValuePointerOptional(data.ModelDescription.Description),
			Location:             StringValuePointerOptional(data.ModelDescription.Location),
			BuildEnvironmentType: StringValuePointerOptional(data.ModelDescription.BuildEnvironmentType),
		}
	}

	if data.Datasets != nil {
		request.Datasets = &client.ExternalModelPackageDatasets{
			TrainingDataCatalogID:        StringValuePointerOptional(data.Datasets.TrainingDataCatalogID),
			TrainingDataCatalogVersionID: StringValuePointerOptional(data.Datasets.TrainingDataCatalogVersionID),
			HoldoutDataCatalogID:         StringValuePointerOptional(data.Datasets.HoldoutDataCatalogID),
			HoldoutDataCatalogVersionID:  StringValuePointerOptional(data.Datasets.HoldoutDataCatalogVersionID),
		}
	}

	if data.Timeseries != nil {
		request.Timeseries = &client.ExternalModelPackageTimeseries{
			DatetimeColumnName:                    data.Timeseries.DatetimeColumnName.ValueString(),
			DatetimeColumnFormat:                  data.Timeseries.DatetimeColumnFormat.ValueString(),
			ForecastPointColumnName:               data.Timeseries.ForecastPointColumnName.ValueString(),
			ForecastDistanceColumnName:            data.Timeseries.ForecastDistanceColumnName.ValueString(),
			ForecastDistancesTimeUnit:             data.Timeseries.ForecastDistancesTimeUnit.ValueString(),
			SeriesColumnName:                      StringValuePointerOptional(data.Timeseries.SeriesColumnName),
			FeatureDerivationWindowStart:          Int64ValuePointerOptional(data.Timeseries.FeatureDerivationWindowStart),
			FeatureDerivationWindowEnd:            Int64ValuePointerOptional(data.Timeseries.FeatureDerivationWindowEnd),
			EffectiveFeatureDerivationWindowStart: Int64ValuePointerOptional(data.Timeseries.EffectiveFeatureDerivationWindowStart),
			EffectiveFeatureDerivationWindowEnd:   Int64ValuePointerOptional(data.Timeseries.EffectiveFeatureDerivationWindowEnd),
			IsCrossSeries:                         BoolValuePointerOptional(data.Timeseries.IsCrossSeries),
			IsNewSeriesSupport:                    BoolValuePointerOptional(data.Timeseries.IsNewSeriesSupport),
			IsTraditionalTimeSeries:               BoolValuePointerOptional(data.Timeseries.IsTraditionalTimeSeries),
		}
		for _, forecastDistance := range data.Timeseries.ForecastDistances {
			request.Timeseries.ForecastDistances = append(request.Timeseries.ForecastDistances, forecastDistance.ValueInt64())
		}
	}

	if data.TextGeneration != nil {
		request.TextGeneration = &client.ExternalModelPackageTextGeneration{
			Prompt: data.TextGeneration.Prompt.ValueString(),
		}
	}

	if data.GeospatialMonitoring != nil {
		request.GeospatialMonitoring = &client.ExternalModelPackageGeospatial{
			PrimaryLocationColumn: data.GeospatialMonitoring.PrimaryLocationColumn.ValueString(),
		}
	}

	return request
}
