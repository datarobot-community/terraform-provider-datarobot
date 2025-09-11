package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithImportState = &DeploymentRetrainingPolicyResource{}

func NewDeploymentRetrainingPolicyResource() resource.Resource {
	return &DeploymentRetrainingPolicyResource{}
}

// VectorDatabaseResource defines the resource implementation.
type DeploymentRetrainingPolicyResource struct {
	service client.Service
}

func (r *DeploymentRetrainingPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_retraining_policy"
}

func (r *DeploymentRetrainingPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Deployment Retraining Policy",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Retraining Policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Deployment for the Retraining Policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Retraining Policy.",
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The description of the Retraining Policy.",
			},
			"action": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("create_challenger"),
				MarkdownDescription: "The the action to take on the resultant new model.",
				Validators:          common.RetrainingPolicyActionValidators(),
			},
			"feature_list_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("same_as_champion"),
				MarkdownDescription: "The feature list strategy used for modeling.",
				Validators:          common.RetrainingPolicyFeatureListStrategyValidators(),
			},
			"model_selection_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("autopilot_recommended"),
				MarkdownDescription: "Determines how the new model is selected when the retraining policy runs.",
				Validators:          common.RetrainingPolicyModelSelectionStrategyValidators(),
			},
			"autopilot_options": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Options for projects used to build new models.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"mode":                             types.StringType,
						"blend_best_models":                types.BoolType,
						"scoring_code_only":                types.BoolType,
						"run_leakage_removed_feature_list": types.BoolType,
						"shap_only_mode":                   types.BoolType,
					},
					map[string]attr.Value{
						"mode":                             types.StringValue("quick"),
						"blend_best_models":                types.BoolValue(false),
						"scoring_code_only":                types.BoolValue(false),
						"run_leakage_removed_feature_list": types.BoolValue(true),
						"shap_only_mode":                   types.BoolValue(false),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"blend_best_models": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Blend best models during Autopilot run. This option is not supported in SHAP-only mode.",
					},
					"mode": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("quick"),
						MarkdownDescription: "The autopilot mode.",
						Validators:          common.AutopilotModeValidators(),
					},
					"run_leakage_removed_feature_list": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						MarkdownDescription: "Run Autopilot on Leakage Removed feature list (if exists).",
					},
					"scoring_code_only": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Keep only models that can be converted to scorable java code during Autopilot run.",
					},
					"shap_only_mode": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Include only models with SHAP value support.",
					},
				},
			},
			"project_options": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Options for projects used to build new models.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"cv_method":       types.StringType,
						"holdout_pct":     types.Float64Type,
						"reps":            types.Float64Type,
						"metric":          types.StringType,
						"validation_pct":  types.Float64Type,
						"validation_type": types.StringType,
					},
					map[string]attr.Value{
						"cv_method":       types.StringValue("RandomCV"),
						"holdout_pct":     types.Float64Value(20.0),
						"reps":            types.Float64Value(5.0),
						"metric":          types.StringNull(),
						"validation_pct":  types.Float64Null(),
						"validation_type": types.StringValue("CV"),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"cv_method": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("RandomCV"),
						MarkdownDescription: "The partitioning method for projects used to build new models.",
						Validators:          common.CVMethodValidators(),
					},
					"holdout_pct": schema.Float64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             float64default.StaticFloat64(20.0),
						MarkdownDescription: "The percentage of dataset to assign to holdout set in projects used to build new models.",
						Validators: []validator.Float64{
							float64validator.Between(0.0, 98.0),
						},
					},
					"validation_pct": schema.Float64Attribute{
						Optional:            true,
						MarkdownDescription: "The percentage of dataset to assign to validation set in projects used to build new models.",
						Validators: []validator.Float64{
							float64validator.Between(1.0, 99.0),
						},
					},
					"metric": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The model selection metric in projects used to build new models.",
						Validators:          common.RetrainingPolicyModelSelectionMetricValidators(),
					},
					"reps": schema.Float64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             float64default.StaticFloat64(5.0),
						MarkdownDescription: "The number of cross validation folds to use for projects used to build new models.",
						Validators: []validator.Float64{
							float64validator.Between(2.0, 50.0),
						},
					},
					"validation_type": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("CV"),
						MarkdownDescription: "The validation type for projects used to build new models.",
						Validators:          common.ModelValidationTypeValidators(),
					},
				},
			},
			"project_options_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The project option strategy used for modeling.",
				Default:             stringdefault.StaticString("same_as_champion"),
				Validators:          common.ProjectOptionsStrategyValidators(),
			},
			"time_series_options": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Time Series project options used to build new models.",
				Attributes: map[string]schema.Attribute{
					"calendar_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The ID of the calendar to be used in this project.",
					},
					"differencing_method": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "For time series projects only. Used to specify which differencing method to apply if the data is stationary. For classification problems simple and seasonal are not allowed. Parameter periodicities must be specified if seasonal is chosen. Defaults to auto.",
					},
					"exponentially_weighted_moving_alpha": schema.Float64Attribute{
						Optional:            true,
						MarkdownDescription: "Discount factor (alpha) used for exponentially weighted moving features.",
						Validators: []validator.Float64{
							float64validator.Between(0.0, 1.0),
						},
					},
					"periodicities": schema.ListNestedAttribute{
						Optional:            true,
						MarkdownDescription: "A list of periodicities for time series projects only. For classification problems periodicities are not allowed. If this is provided, parameter 'differencing_method' will default to 'seasonal' if not provided or 'auto'.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"time_steps": schema.Int64Attribute{
									Required:            true,
									MarkdownDescription: "The number of time steps.",
									Validators: []validator.Int64{
										int64validator.AtLeast(0),
									},
								},
								"time_unit": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "The time unit or ROW if windowsBasisUnit is ROW",
									Validators:          common.TimeUnitValidators(),
								},
							},
						},
					},
					"treat_as_exponential": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "For time series projects only. Used to specify whether to treat data as exponential trend and apply transformations like log-transform. For classification problems always is not allowed. Defaults to auto.",
						Validators:          common.TreatAsExponentialValidators(),
					},
				},
			},
			"trigger": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Retraining policy trigger.",
				Attributes: map[string]schema.Attribute{
					"custom_job_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Custom job ID for the retraining policy.",
					},
					"min_interval_between_runs": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Minimal interval between policy runs in ISO 8601 duration string.",
					},
					"schedule": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Schedule for the retraining policy.",
						Attributes: map[string]schema.Attribute{
							"minute": schema.ListAttribute{
								Required:    true,
								Description: "Minutes of the day when the job will run.",
								ElementType: types.StringType,
							},
							"hour": schema.ListAttribute{
								Required:    true,
								Description: "Hours of the day when the job will run.",
								ElementType: types.StringType,
							},
							"month": schema.ListAttribute{
								Required:    true,
								Description: "Months of the year when the job will run.",
								ElementType: types.StringType,
							},
							"day_of_month": schema.ListAttribute{
								Required:    true,
								Description: "Days of the month when the job will run.",
								ElementType: types.StringType,
							},
							"day_of_week": schema.ListAttribute{
								Required:    true,
								Description: "Days of the week when the job will run.",
								ElementType: types.StringType,
							},
						},
					},
					"status_declines_to_failing": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Identifies when trigger type is based on deployment a health status, whether the policy will run when health status declines to failing.",
					},
					"status_declines_to_warning": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Identifies when trigger type is based on deployment a health status, whether the policy will run when health status declines to warning.",
					},
					"status_still_in_decline": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
						MarkdownDescription: "Identifies when trigger type is based on deployment a health status, whether the policy will run when health status still in decline.",
					},
					"type": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Type of retraining policy trigger.",
						Validators:          common.RetrainingPolicyTypeValidators(),
					},
				},
			},
			"use_case_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the use case to which the retraining policy belongs.",
			},
		},
	}
}

func (r *DeploymentRetrainingPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil { return }
	accessor, ok := req.ProviderData.(common.ServiceAccessor)
	if !ok {
			resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
			return
	}
	r.service = accessor.GetService()
}

func (r *DeploymentRetrainingPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.DeploymentRetrainingPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.checkDeploymentRetrainingSettings(ctx, data); err != nil {
		resp.Diagnostics.AddError("Error checking deployment retraining settings", err.Error())
		return
	}

	request, err := buildRetrainingPolicyRequest(data)
	if err != nil {
		resp.Diagnostics.AddError("Error building Retraining Policy request", err.Error())
		return
	}

	common.TraceAPICall("CreateRetrainingPolicy")
	deploymentRetrainingPolicy, err := r.service.CreateRetrainingPolicy(ctx, data.DeploymentID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Retraining Policy", err.Error())
		return
	}
	data.ID = types.StringValue(deploymentRetrainingPolicy.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentRetrainingPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DeploymentRetrainingPolicyResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	id := data.ID.ValueString()

	common.TraceAPICall("GetRetrainingPolicy")
	deploymentRetrainingPolicy, err := r.service.GetRetrainingPolicy(ctx, data.DeploymentID.ValueString(), id)
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Retraining Policy not found",
				fmt.Sprintf("Retraining Policy with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Retraining Policy with ID %s", id),
				err.Error())
		}
		return
	}

	// Populate retraining policy data
	data.Name = types.StringValue(deploymentRetrainingPolicy.Name)
	data.Description = types.StringValue(deploymentRetrainingPolicy.Description)
	data.Action = types.StringValue(deploymentRetrainingPolicy.Action)
	data.ModelSelectionStrategy = types.StringValue(deploymentRetrainingPolicy.ModelSelectionStrategy)
	data.FeatureListStrategy = types.StringValue(deploymentRetrainingPolicy.FeatureListStrategy)
	data.ProjectOptionsStrategy = types.StringValue(deploymentRetrainingPolicy.ProjectOptionsStrategy)
	if deploymentRetrainingPolicy.UseCase != nil {
		data.UseCaseID = types.StringValue(deploymentRetrainingPolicy.UseCase.ID)
	}
	// Retrieve retraining settings
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving Retraining Settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentRetrainingPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.DeploymentRetrainingPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.checkDeploymentRetrainingSettings(ctx, data); err != nil {
		resp.Diagnostics.AddError("Error checking deployment retraining settings", err.Error())
		return
	}

	request, err := buildRetrainingPolicyRequest(data)
	if err != nil {
		resp.Diagnostics.AddError("Error building Retraining Policy request", err.Error())
		return
	}

	common.TraceAPICall("UpdateDeploymentRetrainingPolicy")
	_, err = r.service.UpdateRetrainingPolicy(ctx, data.DeploymentID.ValueString(), data.ID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Retraining Policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentRetrainingPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.DeploymentRetrainingPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.TraceAPICall("DeleteRetrainingPolicy")
	err := r.service.DeleteRetrainingPolicy(ctx, data.DeploymentID.ValueString(), data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Retraining Policy", err.Error())
			return
		}
	}
}

func (r *DeploymentRetrainingPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DeploymentRetrainingPolicyResource) checkDeploymentRetrainingSettings(ctx context.Context, data models.DeploymentRetrainingPolicyResourceModel) (err error) {
	if data.Trigger != nil && data.Trigger.Schedule != nil {
		var userInfo *client.UserInfo
		userInfo, err = r.service.GetUserInfo(ctx)
		if err != nil {
			return
		}

		if _, err = r.service.UpdateDeploymentRetrainingSettings(ctx, data.DeploymentID.ValueString(), &client.DeploymentRetrainingSettings{
			RetrainingUserID: &userInfo.UID,
		}); err != nil {
			return
		}
	}

	return
}

func buildRetrainingPolicyRequest(data models.DeploymentRetrainingPolicyResourceModel) (request *client.RetrainingPolicyRequest, err error) {
	request = &client.RetrainingPolicyRequest{
		Name:                   common.StringValuePointerOptional(data.Name),
		Description:            common.StringValuePointerOptional(data.Description),
		Action:                 common.StringValuePointerOptional(data.Action),
		FeatureListStrategy:    common.StringValuePointerOptional(data.FeatureListStrategy),
		ModelSelectionStrategy: common.StringValuePointerOptional(data.ModelSelectionStrategy),
		ProjectOptionsStrategy: common.StringValuePointerOptional(data.ProjectOptionsStrategy),
	}

	if data.AutopilotOptions != nil {
		request.AutopilotOptions = &client.AutopilotOptions{
			BlendBestModels:              common.BoolValuePointerOptional(data.AutopilotOptions.BlendBestModels),
			Mode:                         common.StringValuePointerOptional(data.AutopilotOptions.Mode),
			RunLeakageRemovedFeatureList: common.BoolValuePointerOptional(data.AutopilotOptions.RunLeakageRemovedFeatureList),
			ScoringCodeOnly:              common.BoolValuePointerOptional(data.AutopilotOptions.ScoringCodeOnly),
			ShapOnlyMode:                 common.BoolValuePointerOptional(data.AutopilotOptions.ShapOnlyMode),
		}
	}

	if data.ProjectOptions != nil {
		request.ProjectOptions = &client.ProjectOptions{
			CvMethod:       common.StringValuePointerOptional(data.ProjectOptions.CVMethod),
			HoldoutPct:     common.Float64ValuePointerOptional(data.ProjectOptions.HoldoutPct),
			ValidationPct:  common.Float64ValuePointerOptional(data.ProjectOptions.ValidationPct),
			Metric:         common.StringValuePointerOptional(data.ProjectOptions.Metric),
			Reps:           common.Float64ValuePointerOptional(data.ProjectOptions.Reps),
			ValidationType: common.StringValuePointerOptional(data.ProjectOptions.ValidationType),
		}
	}

	if data.TimeSeriesOptions != nil {
		request.TimeSeriesOptions = &client.TimeSeriesOptions{
			CalendarID:                       common.StringValuePointerOptional(data.TimeSeriesOptions.CalendarID),
			DifferencingMethod:               common.StringValuePointerOptional(data.TimeSeriesOptions.DifferencingMethod),
			ExponentiallyWeightedMovingAlpha: common.Float64ValuePointerOptional(data.TimeSeriesOptions.ExponentiallyWeightedMovingAlpha),
			TreatAsExponential:               common.StringValuePointerOptional(data.TimeSeriesOptions.TreatAsExponential),
		}
	}

	if data.Trigger != nil {
		request.Trigger = &client.Trigger{
			CustomJobID:             common.StringValuePointerOptional(data.Trigger.CustomJobID),
			MinIntervalBetweenRuns:  common.StringValuePointerOptional(data.Trigger.MinIntervalBetweenRuns),
			StatusDeclinesToFailing: common.BoolValuePointerOptional(data.Trigger.StatusDeclinesToFailing),
			StatusDeclinesToWarning: common.BoolValuePointerOptional(data.Trigger.StatusDeclinesToWarning),
			StatusStillInDecline:    common.BoolValuePointerOptional(data.Trigger.StatusStillInDecline),
			Type:                    common.StringValuePointerOptional(data.Trigger.Type),
		}
		if data.Trigger.Schedule != nil {
			var schedule client.Schedule
			if schedule, err = common.ConvertSchedule(*data.Trigger.Schedule); err != nil {
				return
			}
			request.Trigger.Schedule = &schedule
		}
	}

	return
}
