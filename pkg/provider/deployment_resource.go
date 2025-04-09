package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DeploymentResource{}
var _ resource.ResourceWithImportState = &DeploymentResource{}

func NewDeploymentResource() resource.Resource {
	return &DeploymentResource{}
}

// VectorDatabaseResource defines the resource implementation.
type DeploymentResource struct {
	provider *Provider
}

func (r *DeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *DeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Deployment",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Deployment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "The label of the Deployment.",
				Required:            true,
			},
			"registered_model_version_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the registered model version for this Deployment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"prediction_environment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the predication environment for this Deployment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Deployment to.",
				ElementType:         types.StringType,
			},
			"importance": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The importance of the Deployment.",
				Default:             stringdefault.StaticString("LOW"),
				Validators:          ImportanceValidators(),
			},
			"runtime_parameter_values": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The runtime parameter values for the Deployment.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the runtime parameter.",
						},
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The type of the runtime parameter.",
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The value of the runtime parameter (type conversion is handled internally).",
						},
					},
				},
			},
			"predictions_by_forecast_date_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The predictions by forecase date settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Is ’True’ if predictions by forecast date is enabled for this deployment.",
					},
					"column_name": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The column name in prediction datasets to be used as forecast date.",
					},
					"datetime_format": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The datetime format of the forecast date column in prediction datasets.",
					},
				},
			},
			"challenger_models_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The challenger models settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Is 'True' if challenger models is enabled for this deployment.",
					},
				},
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(path.MatchRoot("predictions_data_collection_settings")),
				},
			},
			"segment_analysis_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The segment analysis settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Set to 'True' if segment analysis is enabled for this deployment.",
					},
					"attributes": schema.ListAttribute{
						Optional:            true,
						MarkdownDescription: "A list of strings that gives the segment attributes selected for tracking.",
						ElementType:         types.StringType,
					},
				},
			},
			"bias_and_fairness_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Bias and fairness settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"protected_features": schema.ListAttribute{
						Required:            true,
						MarkdownDescription: "A list of features to mark as protected.",
						ElementType:         types.StringType,
					},
					"preferable_target_value": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "A target value that should be treated as a positive outcome for the prediction.",
					},
					"fairness_metric_set": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "A set of fairness metrics to use for calculating fairness.",
						Validators:          FairnessMetricSetValidators(),
					},
					"fairness_threshold": schema.Float64Attribute{
						Required:            true,
						MarkdownDescription: "Threshold value of the fairness metric. Cannot be less than 0 or greater than 1.",
						Validators:          Float64ZeroToOneValidators(),
					},
				},
			},
			"challenger_replay_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The challenger replay settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "If challenger replay is enabled.",
					},
				},
			},
			"batch_monitoring_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The batch monitoring settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "If batch monitoring is enabled.",
					},
				},
			},
			"drift_tracking_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The drift tracking settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"target_drift_enabled": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "If target drift tracking is to be turned on.",
					},
					"feature_drift_enabled": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "If feature drift tracking is to be turned on.",
					},
					"feature_selection": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The feature selection method to be used for drift tracking.",
						Validators:          FeatureSelectionValidators(),
					},
					"tracked_features": schema.ListAttribute{
						Optional:            true,
						MarkdownDescription: "List of features to be tracked for drift.",
						ElementType:         types.StringType,
					},
				},
			},
			"association_id_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Association ID settings for this Deployment.",
				Attributes: map[string]schema.Attribute{
					"auto_generate_id": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether to auto generate ID.",
					},
					"column_names": schema.ListAttribute{
						Optional:            true,
						MarkdownDescription: "Name of the columns to be used as association ID, currently only support a list of one string.",
						ElementType:         types.StringType,
					},
					"required_in_prediction_requests": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether the association ID column is required in prediction requests.",
					},
				},
			},
			"predictions_data_collection_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The predictions data collection settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "If predictions data collections is enabled for this Deployment.",
					},
				},
			},
			"prediction_warning_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The prediction warning settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "If target prediction warning is enabled for this Deployment.",
					},
					"custom_boundaries": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The custom boundaries for prediction warnings.",
						Attributes: map[string]schema.Attribute{
							"upper_boundary": schema.Float64Attribute{
								Optional:            true,
								MarkdownDescription: "All predictions greater than provided value will be considered anomalous.",
							},
							"lower_boundary": schema.Float64Attribute{
								Optional:            true,
								MarkdownDescription: "All predictions less than provided value will be considered anomalous.",
							},
						},
					},
				},
			},
			"prediction_intervals_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The prediction intervals settings for this Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Whether prediction intervals are enabled for this deployment.",
					},
					"percentiles": schema.ListAttribute{
						Optional:            true,
						MarkdownDescription: "List of enabled prediction intervals’ sizes for this deployment.",
						ElementType:         types.Int64Type,
					},
				},
			},
			"health_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The health settings for this Deployment.",
				Attributes: map[string]schema.Attribute{
					"service": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The service health settings for this Deployment.",
						Attributes: map[string]schema.Attribute{
							"batch_count": schema.Int64Attribute{
								Required:            true,
								MarkdownDescription: "The batch count for the service health settings.",
								Validators:          BatchCountValidators(),
							},
						},
					},
					"data_drift": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The data drift health settings for this Deployment.",
						Attributes: map[string]schema.Attribute{
							"batch_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The batch count for the data drift health settings.",
								Validators:          BatchCountValidators(),
							},
							"time_interval": schema.StringAttribute{
								Optional:            true,
								MarkdownDescription: "The time interval for the data drift health settings.",
							},
							"drift_threshold": schema.Float64Attribute{
								Optional:            true,
								MarkdownDescription: "The drift threshold for the data drift health settings.",
							},
							"importance_threshold": schema.Float64Attribute{
								Optional:            true,
								MarkdownDescription: "The importance threshold for the data drift health settings.",
							},
							"low_importance_warning_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The low importance warning count for the data drift health settings.",
							},
							"low_importance_failing_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The low importance failing count for the data drift health settings.",
							},
							"high_importance_warning_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The high importance warning count for the data drift health settings.",
							},
							"high_importance_failing_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The high importance failing count for the data drift health settings.",
							},
							"exclude_features": schema.ListAttribute{
								Optional:            true,
								MarkdownDescription: "The exclude features for the data drift health settings.",
								ElementType:         types.StringType,
							},
							"starred_features": schema.ListAttribute{
								Optional:            true,
								MarkdownDescription: "The starred features for the data drift health settings.",
								ElementType:         types.StringType,
							},
						},
					},
					"accuracy": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The accuracy health settings for this Deployment.",
						Attributes: map[string]schema.Attribute{
							"batch_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The batch count for the accuracy health settings.",
								Validators:          BatchCountValidators(),
							},
							"metric": schema.StringAttribute{
								Optional:            true,
								MarkdownDescription: "The metric for the accuracy health settings.",
							},
							"measurement": schema.StringAttribute{
								Optional:            true,
								MarkdownDescription: "The measurement for the accuracy health settings.",
							},
							"warning_threshold": schema.Float64Attribute{
								Optional:            true,
								MarkdownDescription: "The warning threshold for the accuracy health settings.",
							},
							"failing_threshold": schema.Float64Attribute{
								Optional:            true,
								MarkdownDescription: "The failing threshold for the accuracy health settings.",
							},
						},
					},
					"fairness": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The fairness health settings for this Deployment.",
						Attributes: map[string]schema.Attribute{
							"protected_class_warning_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The protected class warning count for the fairness health settings.",
							},
							"protected_class_failing_count": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "The protected class failing count for the fairness health settings.",
							},
						},
					},
					"custom_metrics": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The custom metrics health settings for this Deployment.",
						Attributes: map[string]schema.Attribute{
							"warning_conditions": schema.ListNestedAttribute{
								Optional:            true,
								MarkdownDescription: "The warning conditions for the custom metrics health settings.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"metric_id": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "The metric ID for the warning condition of the custom metrics health settings.",
										},
										"compare_operator": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "The compare operator for the warning condition of the custom metrics health settings.",
										},
										"threshold": schema.Float64Attribute{
											Required:            true,
											MarkdownDescription: "The threshold for the warning condition of the custom metrics health settings.",
										},
									},
								},
							},
							"failing_conditions": schema.ListNestedAttribute{
								Optional:            true,
								MarkdownDescription: "The failing conditions for the custom metrics health settings.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"metric_id": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "The metric ID for the failing condition of the custom metrics health settings.",
										},
										"compare_operator": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "The compare operator for the failing condition of the custom metrics health settings.",
										},
										"threshold": schema.Float64Attribute{
											Required:            true,
											MarkdownDescription: "The threshold for the failing condition of the custom metrics health settings.",
										},
									},
								},
							},
						},
					},
					"predictions_timeliness": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The predictions timeliness health settings for this Deployment.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Required:            true,
								MarkdownDescription: "If predictions timeliness is enabled for this Deployment.",
							},
							"expected_frequency": schema.StringAttribute{
								Optional:            true,
								MarkdownDescription: "The expected frequency for the predictions timeliness health settings.",
								Validators:          TimelinessFrequencyValidators(),
							},
						},
					},
					"actuals_timeliness": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "The actuals timeliness health settings for this Deployment.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Required:            true,
								MarkdownDescription: "If acutals timeliness is enabled for this Deployment.",
							},
							"expected_frequency": schema.StringAttribute{
								Optional:            true,
								MarkdownDescription: "The expected frequency for the actuals timeliness health settings.",
								Validators:          TimelinessFrequencyValidators(),
							},
						},
					},
				},
			},
			"predictions_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Settings for the predictions.",
				Attributes: map[string]schema.Attribute{
					"min_computes": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "The minimum number of computes to use for predictions.",
					},
					"max_computes": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "The maximum number of computes to use for predictions.",
					},
					"resource_bundle_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The resource bundle ID to use for predictions.",
					},
				},
			},
			"feature_cache_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The feature cache settings for this Deployment.",
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "If feature cache is enabled for this Deployment.",
					},
					"fetching": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "If feature cache fetching is enabled.",
					},
					"schedule": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "Defines the feature cache schedule.",
						Attributes: map[string]schema.Attribute{
							"minute": schema.ListAttribute{
								Required:    true,
								Description: "Minutes of the day.",
								ElementType: types.StringType,
							},
							"hour": schema.ListAttribute{
								Required:    true,
								Description: "Hours of the day.",
								ElementType: types.StringType,
							},
							"month": schema.ListAttribute{
								Required:    true,
								Description: "Months of the year.",
								ElementType: types.StringType,
							},
							"day_of_month": schema.ListAttribute{
								Required:    true,
								Description: "Days of the month.",
								ElementType: types.StringType,
							},
							"day_of_week": schema.ListAttribute{
								Required:    true,
								Description: "Days of the week.",
								ElementType: types.StringType,
							},
						},
					},
				},
			},
			"retraining_settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The retraining settings for this Deployment.",
				Attributes: map[string]schema.Attribute{
					"retraining_user_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "ID of the retraining user.",
					},
					"credential_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "ID of the credential used to refresh retraining dataset.",
					},
					"dataset_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "ID of the retraining dataset.",
					},
					"prediction_environment_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "ID of the prediction environment to associate with the challengers created by retraining policies.",
					},
				},
			},
		},
	}
}

func (r *DeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentResourceModel

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := &client.CreateDeploymentFromModelPackageRequest{
		ModelPackageID:          data.RegisteredModelVersionID.ValueString(),
		PredictionEnvironmentID: data.PredictionEnvironmentID.ValueString(),
		Label:                   data.Label.ValueString(),
		Importance:              data.Importance.ValueString(),
	}

	if IsKnown(data.RuntimeParameterValues) {
		runtimeParameterValues, err := convertRuntimeParameterValues(ctx, data.RuntimeParameterValues)
		if err != nil {
			resp.Diagnostics.AddError("Error reading runtime parameter values", err.Error())
			return
		}
		request.RuntimeParameterValues = runtimeParameterValues
	}

	traceAPICall("CreateDeployment")
	createResp, statusID, err := r.provider.service.CreateDeploymentFromModelPackage(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Deployment", err.Error())
		return
	}
	if statusID == "" {
		resp.Diagnostics.AddError("Unable to find Deployment creation task", "Status ID is empty")
	}

	err = waitForTaskStatusToComplete(ctx, r.provider.service, statusID)
	if err != nil {
		traceAPICall("DeleteDeployment")
		_ = r.provider.service.DeleteDeployment(ctx, createResp.ID)
		resp.Diagnostics.AddError("Deployment creation failed", err.Error())
		return
	}

	deployment, err := r.waitForDeploymentToBeReady(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Deployment is not ready", err.Error())
		return
	}
	data.ID = types.StringValue(deployment.ID)

	// Deployment must be inactive in order to update Resource Bundle
	deactivatedDeployment := false
	if data.PredictionsSettings != nil && IsKnown(data.PredictionsSettings.ResourceBundleID) {
		if err = r.deactivateDeployment(ctx, createResp.ID); err != nil {
			return
		}
		deactivatedDeployment = true
	}

	err = r.updateDeploymentSettings(ctx, createResp.ID, data)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Deployment settings", err.Error())
		return
	}

	if deactivatedDeployment {
		if err = r.activateDeployment(ctx, createResp.ID); err != nil {
			return
		}
	}

	for _, useCaseID := range data.UseCaseIDs {
		traceAPICall("AddDeploymentToUseCase")
		if err = addEntityToUseCase(
			ctx,
			r.provider.service,
			useCaseID.ValueString(),
			"deployment",
			deployment.ID,
		); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Deployment to Use Case %s", useCaseID), err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	traceAPICall("GetDeployment")
	deployment, err := r.provider.service.GetDeployment(ctx, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Deployment not found",
				fmt.Sprintf("Deployment with ID %s is not found. Removing from state.", data.ID.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Deployment with ID %s", data.ID.ValueString()),
				err.Error())
		}
		return
	}
	data.Label = types.StringValue(deployment.Label)
	data.RegisteredModelVersionID = types.StringValue(deployment.ModelPackage.ID)
	data.PredictionEnvironmentID = types.StringValue(deployment.PredictionEnvironment.ID)
	data.Importance = types.StringValue(deployment.Importance)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	traceAPICall("UpdateDeployment")
	_, err := r.provider.service.UpdateDeployment(ctx,
		id,
		&client.UpdateDeploymentRequest{
			Label:      plan.Label.ValueString(),
			Importance: plan.Importance.ValueString(),
		})
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Deployment not found",
				fmt.Sprintf("Deployment with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error updating Deployment", err.Error())
		}
		return
	}

	_, err = r.waitForDeploymentToBeReady(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Deployment not ready", err.Error())
		return
	}

	if plan.RegisteredModelVersionID != state.RegisteredModelVersionID {
		traceAPICall("ValidateDeploymentModelReplacement")
		validateModelReplacementResp, err := r.provider.service.ValidateDeploymentModelReplacement(ctx, id, &client.ValidateDeployemntModelReplacementRequest{
			ModelPackageID: plan.RegisteredModelVersionID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error validating Deployment model replacement", err.Error())
			return
		}

		if validateModelReplacementResp.Status != "passing" {
			resp.Diagnostics.AddError("Invalid Deployment model replacement", validateModelReplacementResp.Message)
			return
		}

		traceAPICall("UpdateDeploymentModel")
		_, statusId, err := r.provider.service.UpdateDeploymentModel(ctx, id, &client.UpdateDeploymentModelRequest{
			ModelPackageID: plan.RegisteredModelVersionID.ValueString(),
			Reason:         "OTHER",
		})
		if err != nil {
			resp.Diagnostics.AddError("Error replacing Deployment model", err.Error())
			return
		}
		if statusId == "" {
			resp.Diagnostics.AddError("Unable to find Deployment model replacement task", "Status ID is empty")
		}

		// model replacement is an async operation, separate from waiting for the deployment to be ready
		err = waitForTaskStatusToComplete(ctx, r.provider.service, statusId)
		if err != nil {
			resp.Diagnostics.AddError("Deployment model replacement task not completed", err.Error())
			return
		}

		_, err = r.waitForDeploymentToBeReady(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Deployment not ready after model replacement", err.Error())
			return
		}
	}
	// check if updating retgraining settings
	if plan.RetrainingSettings != nil && !reflect.DeepEqual(plan.RetrainingSettings, state.RetrainingSettings) {
		err = r.updateRetrainingSettings(ctx, id, plan.RetrainingSettings)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Deployment retraining settings", err.Error())
			return
		}
	}

	// Deployment must be inactive in order to update Resource Bundle
	deactivatedDeployment := false
	if plan.PredictionsSettings != nil && IsKnown(plan.PredictionsSettings.ResourceBundleID) {
		// check that Resource Bundle has been modified
		if state.PredictionsSettings == nil || plan.PredictionsSettings.ResourceBundleID != state.PredictionsSettings.ResourceBundleID {
			if err = r.deactivateDeployment(ctx, id); err != nil {
				return
			}
			deactivatedDeployment = true
		}
	}

	err = r.updateDeploymentSettings(ctx, id, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Deployment settings", err.Error())
		return
	}

	if deactivatedDeployment {
		if err = r.activateDeployment(ctx, id); err != nil {
			return
		}
	}

	err = r.updateDeploymentRuntimeParameters(ctx, id, plan, state)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Deployment runtime parameters", err.Error())
		return
	}

	if err = updateUseCasesForEntity(
		ctx,
		r.provider.service,
		"deployment",
		plan.ID.ValueString(),
		state.UseCaseIDs,
		plan.UseCaseIDs,
	); err != nil {
		resp.Diagnostics.AddError("Error updating Use Cases for Deployment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *DeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteDeployment")
	err := r.provider.service.DeleteDeployment(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Deployment", err.Error())
			return
		}
	}
}

func (r *DeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DeploymentResource) waitForDeploymentToBeReady(ctx context.Context, id string) (*client.Deployment, error) {
	return r.waitForDeploymentStatus(ctx, id, "active")
}

func (r *DeploymentResource) waitForDeploymentStatus(ctx context.Context, id string, status string) (*client.Deployment, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		deployment, err := r.provider.service.GetDeployment(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}

		if deployment.Status == status {
			return nil
		} else if strings.Contains(deployment.Status, "error") {
			return backoff.Permanent(errors.New("deployment has errored"))
		}

		return errors.New("deployment is not ready")
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return nil, err
	}

	traceAPICall("GetDeployment")
	return r.provider.service.GetDeployment(ctx, id)
}

func (r *DeploymentResource) updateDeploymentSettings(
	ctx context.Context,
	id string,
	data DeploymentResourceModel,
) (err error) {
	req := &client.DeploymentSettings{}
	if data.PredictionsByForecastDateSettings != nil {
		req.PredictionsByForecastDate = &client.PredictionsByForecastDateSettings{
			Enabled:        data.PredictionsByForecastDateSettings.Enabled.ValueBool(),
			ColumnName:     data.PredictionsByForecastDateSettings.ColumnName.ValueString(),
			DatetimeFormat: data.PredictionsByForecastDateSettings.DatetimeFormat.ValueString(),
		}
	}

	if data.ChallengerModelsSettings != nil {
		req.ChallengerModels = &client.BasicSetting{
			Enabled: data.ChallengerModelsSettings.Enabled.ValueBool(),
		}
	}

	if data.BatchMonitoringSettings != nil {
		req.BatchMonitoring = &client.BasicSetting{
			Enabled: data.BatchMonitoringSettings.Enabled.ValueBool(),
		}
	}

	if data.SegmentAnalysisSettings != nil {
		req.SegmentAnalysis = &client.SegmentAnalysisSetting{
			Enabled:    data.SegmentAnalysisSettings.Enabled.ValueBool(),
			Attributes: convertTfStringListToPtr(data.SegmentAnalysisSettings.Attributes),
		}
	}

	if data.BiasAndFairnessSettings != nil {
		protectedFeatures := make([]string, len(data.BiasAndFairnessSettings.ProtectedFeatures))
		for i, protectedFeature := range data.BiasAndFairnessSettings.ProtectedFeatures {
			protectedFeatures[i] = protectedFeature.ValueString()
		}
		req.BiasAndFairness = &client.BiasAndFairnessSetting{
			ProtectedFeatures:     protectedFeatures,
			PreferableTargetValue: data.BiasAndFairnessSettings.PreferableTargetValue.ValueBool(),
			FairnessMetricsSet:    data.BiasAndFairnessSettings.FairnessMetricSet.ValueString(),
			FairnessThreshold:     data.BiasAndFairnessSettings.FairnessThreshold.ValueFloat64(),
		}
	}

	if data.DriftTrackingSettings != nil {
		req.FeatureDrift = &client.FeatureDriftSetting{
			Enabled:          data.DriftTrackingSettings.FeatureDriftEnabled.ValueBool(),
			FeatureSelection: StringValuePointerOptional(data.DriftTrackingSettings.FeatureSelection),
			TrackedFeatures:  convertTfStringListToPtr(data.DriftTrackingSettings.TrackedFeatures),
		}
		req.TargetDrift = &client.BasicSetting{
			Enabled: data.DriftTrackingSettings.TargetDriftEnabled.ValueBool(),
		}
	}

	if data.AssociationIDSettings != nil {
		columnNames := make([]string, len(data.AssociationIDSettings.ColumnNames))
		for i, columnName := range data.AssociationIDSettings.ColumnNames {
			columnNames[i] = columnName.ValueString()
		}
		req.AssociationID = &client.AssociationIDSetting{
			AutoGenerateID:               data.AssociationIDSettings.AutoGenerateID.ValueBool(),
			ColumnNames:                  columnNames,
			RequiredInPredictionRequests: data.AssociationIDSettings.RequiredInPredictionRequests.ValueBool(),
		}
	}

	if data.PredictionsDataCollectionSettings != nil {
		req.PredictionsDataCollection = &client.BasicSetting{
			Enabled: data.PredictionsDataCollectionSettings.Enabled.ValueBool(),
		}
	}

	if data.PredictionWarningSettings != nil {
		req.PredictionWarning = &client.PredictionWarningSetting{
			Enabled: data.PredictionWarningSettings.Enabled.ValueBool(),
		}
		if data.PredictionWarningSettings.CustomBoundaries != nil {
			req.PredictionWarning.CustomBoundaries = &client.CustomBoundaries{
				LowerBoundary: data.PredictionWarningSettings.CustomBoundaries.LowerBoundary.ValueFloat64(),
				UpperBoundary: data.PredictionWarningSettings.CustomBoundaries.UpperBoundary.ValueFloat64(),
			}
		}
	}

	if data.PredictionIntervalsSettings != nil {
		percentils := make([]int64, len(data.PredictionIntervalsSettings.Percentiles))
		for i, percentile := range data.PredictionIntervalsSettings.Percentiles {
			percentils[i] = percentile.ValueInt64()
		}
		req.PredictionIntervals = &client.PredictionIntervalsSetting{
			Enabled:     data.PredictionIntervalsSettings.Enabled.ValueBool(),
			Percentiles: percentils,
		}
	}

	if data.PredictionsSettings != nil {
		req.PredictionsSettings = &client.PredictionsSettings{
			MinComputes:      Int64ValuePointerOptional(data.PredictionsSettings.MinComputes),
			MaxComputes:      Int64ValuePointerOptional(data.PredictionsSettings.MaxComputes),
			ResourceBundleID: StringValuePointerOptional(data.PredictionsSettings.ResourceBundleID),
		}
	}

	traceAPICall("UpdateDeploymentSettings")
	_, err = r.provider.service.UpdateDeploymentSettings(ctx, id, req)
	if err != nil {
		return
	}

	if data.ChallengerReplaySettings != nil {
		req := &client.DeploymentChallengerReplaySettings{
			Enabled: data.ChallengerReplaySettings.Enabled.ValueBool(),
		}

		traceAPICall("UpdateDeploymentChallengerReplaySettings")
		_, err = r.provider.service.UpdateDeploymentChallengerReplaySettings(ctx, id, req)
		if err != nil {
			return
		}
	}

	if data.HealthSettings != nil {
		req := &client.DeploymentHealthSettings{}

		if data.HealthSettings.Service != nil {
			req.Service = &client.DeploymentServiceHealthSettings{
				BatchCount: data.HealthSettings.Service.BatchCount.ValueInt64(),
			}
		}

		if data.HealthSettings.DataDrift != nil {
			req.DataDrift = &client.DeploymentDataDriftHealthSettings{
				BatchCount:                 Int64ValuePointerOptional(data.HealthSettings.DataDrift.BatchCount),
				TimeInterval:               StringValuePointerOptional(data.HealthSettings.DataDrift.TimeInterval),
				DriftThreshold:             Float64ValuePointerOptional(data.HealthSettings.DataDrift.DriftThreshold),
				ImportanceThreshold:        Float64ValuePointerOptional(data.HealthSettings.DataDrift.ImportanceThreshold),
				LowImportanceWarningCount:  Int64ValuePointerOptional(data.HealthSettings.DataDrift.LowImportanceWarningCount),
				LowImportanceFailingCount:  Int64ValuePointerOptional(data.HealthSettings.DataDrift.LowImportanceFailingCount),
				HighImportanceWarningCount: Int64ValuePointerOptional(data.HealthSettings.DataDrift.HighImportanceWarningCount),
				HighImportanceFailingCount: Int64ValuePointerOptional(data.HealthSettings.DataDrift.HighImportanceFailingCount),
				ExcludedFeatures:           convertTfStringListToPtr(data.HealthSettings.DataDrift.ExcludeFeatures),
				StarredFeatures:            convertTfStringListToPtr(data.HealthSettings.DataDrift.StarredFeatures),
			}
		}

		if data.HealthSettings.Accuracy != nil {
			req.Accuracy = &client.DeploymentAccuracyHealthSettings{
				BatchCount:       Int64ValuePointerOptional(data.HealthSettings.Accuracy.BatchCount),
				Metric:           StringValuePointerOptional(data.HealthSettings.Accuracy.Metric),
				Measurement:      StringValuePointerOptional(data.HealthSettings.Accuracy.Measurement),
				WarningThreshold: Float64ValuePointerOptional(data.HealthSettings.Accuracy.WarningThreshold),
				FailingThreshold: Float64ValuePointerOptional(data.HealthSettings.Accuracy.FailingThreshold),
			}
		}

		if data.HealthSettings.Fairness != nil {
			req.Fairness = &client.DeploymentFairnessHealthSettings{
				ProtectedClassWarningCount: Int64ValuePointerOptional(data.HealthSettings.Fairness.ProtectedClassWarningCount),
				ProtectedClassFailingCount: Int64ValuePointerOptional(data.HealthSettings.Fairness.ProtectedClassFailingCount),
			}
		}

		if data.HealthSettings.CustomMetrics != nil {
			req.CustomMetrics = &client.DeploymentCustomMetricsHealthSettings{
				WarningConditions: convertCustomMetricConditions(data.HealthSettings.CustomMetrics.WarningConditions),
				FailingConditions: convertCustomMetricConditions(data.HealthSettings.CustomMetrics.FailingConditions),
			}
		}

		if data.HealthSettings.PredictionsTimeliness != nil {
			req.PredictionsTimeliness = &client.DeploymentTimelinessHealthSettings{
				Enabled:           data.HealthSettings.PredictionsTimeliness.Enabled.ValueBool(),
				ExpectedFrequency: StringValuePointerOptional(data.HealthSettings.PredictionsTimeliness.ExpectedFrequency),
			}
		}

		if data.HealthSettings.ActualsTimeliness != nil {
			req.ActualsTimeliness = &client.DeploymentTimelinessHealthSettings{
				Enabled:           data.HealthSettings.ActualsTimeliness.Enabled.ValueBool(),
				ExpectedFrequency: StringValuePointerOptional(data.HealthSettings.ActualsTimeliness.ExpectedFrequency),
			}
		}

		traceAPICall("UpdateDeploymentHealthSettings")
		_, err = r.provider.service.UpdateDeploymentHealthSettings(ctx, id, req)
		if err != nil {
			return
		}
	}

	if data.FeatureCacheSettings != nil {
		req := &client.DeploymentFeatureCacheSettings{
			Enabled:  data.FeatureCacheSettings.Enabled.ValueBool(),
			Fetching: BoolValuePointerOptional(data.FeatureCacheSettings.Fetching),
		}

		if data.FeatureCacheSettings.Schedule != nil {
			var schedule client.Schedule
			if schedule, err = convertSchedule(*data.FeatureCacheSettings.Schedule); err != nil {
				return
			}
			req.Schedule = &schedule
		}

		traceAPICall("UpdateDeploymentFeatureCacheSettings")
		_, err = r.provider.service.UpdateDeploymentFeatureCacheSettings(ctx, id, req)
		if err != nil {
			return
		}
	}
	return
}

func convertCustomMetricConditions(conditions []CustomMetricCondition) []client.CustomMetricCondition {
	customMetricConditions := make([]client.CustomMetricCondition, 0)
	for _, condition := range conditions {
		customMetricConditions = append(customMetricConditions, client.CustomMetricCondition{
			MetricID:        condition.MetricID.ValueString(),
			CompareOperator: condition.CompareOperator.ValueString(),
			Threshold:       condition.Threshold.ValueFloat64(),
		})
	}
	return customMetricConditions
}

func (r *DeploymentResource) updateRetrainingSettings(
	ctx context.Context,
	id string,
	data *RetrainingSettings,
) (err error) {
	if data == nil {
		return
	}
	// get retraining settings
	retrainingSettings, err := r.provider.service.GetDeploymentRetrainingSettings(ctx, id)
	if err != nil {
		return
	}
	req := &client.DeploymentRetrainingSettings{
		RetrainingUserID:        StringValuePointerOptional(data.RetrainingUserID),
		CredentialID:            StringValuePointerOptional(data.CredentialID),
		DatasetID:               StringValuePointerOptional(data.DatasetID),
		PredictionEnvironmentID: StringValuePointerOptional(data.PredictionEnvironmentID),
	}

	// Compare with existing retraining settings and update only if there are changes
	if !reflect.DeepEqual(req, retrainingSettings) {
		traceAPICall("UpdateDeploymentRetrainingSettings")
		_, err = r.provider.service.UpdateDeploymentRetrainingSettings(ctx, id, req)
		if err != nil {
			return
		}
	}
	return
}

func (r *DeploymentResource) updateDeploymentRuntimeParameters(
	ctx context.Context,
	id string,
	plan DeploymentResourceModel,
	state DeploymentResourceModel,
) (err error) {
	if !IsKnown(plan.RuntimeParameterValues) && !IsKnown(state.RuntimeParameterValues) {
		return
	}

	if !reflect.DeepEqual(plan.RuntimeParameterValues, state.RuntimeParameterValues) {
		var stateRuntimeParameterValues []RuntimeParameterValue
		if diags := state.RuntimeParameterValues.ElementsAs(ctx, &stateRuntimeParameterValues, false); diags.HasError() {
			err = errors.New("Error converting runtime parameter values")
			return
		}
		var planRuntimeParameterValues []RuntimeParameterValue
		if diags := plan.RuntimeParameterValues.ElementsAs(ctx, &planRuntimeParameterValues, false); diags.HasError() {
			err = errors.New("Error converting runtime parameter values")
			return
		}

		// reset runtime parameters that are not in the plan
		newRuntimeParameterValues := make([]client.RuntimeParameterValueRequest, 0)
		for _, stateRuntimeParameterValue := range stateRuntimeParameterValues {
			found := false
			for _, planRuntimeParameterValue := range planRuntimeParameterValues {
				if stateRuntimeParameterValue.Key == planRuntimeParameterValue.Key {
					found = true
					break
				}
			}
			if !found {
				newRuntimeParameterValues = append(newRuntimeParameterValues, client.RuntimeParameterValueRequest{
					FieldName: stateRuntimeParameterValue.Key.ValueString(),
					Type:      stateRuntimeParameterValue.Type.ValueString(),
					Value:     nil,
				})
			}
		}

		// add runtime parameters that are in the plan
		for _, param := range planRuntimeParameterValues {
			var value any
			value, err = formatRuntimeParameterValue(param.Type.ValueString(), param.Value.ValueString())
			if err != nil {
				return
			}
			newRuntimeParameterValues = append(newRuntimeParameterValues, client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     &value,
			})
		}

		if len(newRuntimeParameterValues) == 0 {
			return
		}

		// the Deployment must be inactive in order to update runtime parameters
		if err = r.deactivateDeployment(ctx, id); err != nil {
			return
		}

		var newRuntimeParameterValuesBytes []byte
		newRuntimeParameterValuesBytes, err = json.Marshal(newRuntimeParameterValues)
		if err != nil {
			return
		}

		traceAPICall("UpdateDeploymentRuntimeParameters")
		if _, err = r.provider.service.UpdateDeploymentRuntimeParameters(ctx, id, &client.UpdateDeploymentRuntimeParametersRequest{
			RuntimeParameterValues: string(newRuntimeParameterValuesBytes),
		}); err != nil {
			return
		}

		if err = r.activateDeployment(ctx, id); err != nil {
			return
		}
	}

	return
}

func (r *DeploymentResource) deactivateDeployment(ctx context.Context, id string) (err error) {
	traceAPICall("DeactivateDeployment")
	if _, err = r.provider.service.DeactivateDeployment(ctx, id); err != nil {
		err = fmt.Errorf("Error deactivating deployment: %w", err)
		return
	}
	if _, err = r.waitForDeploymentStatus(ctx, id, "inactive"); err != nil {
		err = fmt.Errorf("Error waiting for deployment to be inactive: %w", err)
		return
	}

	return
}

func (r *DeploymentResource) activateDeployment(ctx context.Context, id string) (err error) {
	traceAPICall("ActivateDeployment")
	if _, err = r.provider.service.ActivateDeployment(ctx, id); err != nil {
		err = fmt.Errorf("Error activating deployment: %w", err)
		return
	}
	if _, err = r.waitForDeploymentToBeReady(ctx, id); err != nil {
		err = fmt.Errorf("Error waiting for deployment to be ready: %w", err)
		return
	}

	return
}
