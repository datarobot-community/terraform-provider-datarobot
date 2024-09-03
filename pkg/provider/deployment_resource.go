package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
			"settings": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The settings for the Deployment.",
				Attributes: map[string]schema.Attribute{
					"association_id": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Used to associate predictions back to your actual data.",
						Attributes: map[string]schema.Attribute{
							"auto_generate_id": schema.BoolAttribute{
								Required:            true,
								MarkdownDescription: "Whether to automatically generate an association ID.",
							},
							"feature_name": schema.StringAttribute{
								Required:            true,
								MarkdownDescription: "The name of the feature to use as the association ID.",
							},
						},
					},
					"prediction_row_storage": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Used to score predictions made by the challenger models and compare performance with the deployed model.",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"challenger_analysis": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Used to compare the performance of the deployed model with the challenger models.",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"predictions_settings": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Settings for the predictions.",
						Attributes: map[string]schema.Attribute{
							"min_computes": schema.Int64Attribute{
								Required:            true,
								MarkdownDescription: "The minimum number of computes to use for predictions.",
							},
							"max_computes": schema.Int64Attribute{
								Required:            true,
								MarkdownDescription: "The maximum number of computes to use for predictions.",
							},
							"real_time": schema.BoolAttribute{
								Required:            true,
								MarkdownDescription: "Whether to use real-time predictions.",
							},
						},
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("CreateDeployment")
	createResp, err := r.provider.service.CreateDeploymentFromModelPackage(ctx, &client.CreateDeploymentFromModelPackageRequest{
		ModelPackageID:          data.RegisteredModelVersionID.ValueString(),
		PredictionEnvironmentID: data.PredictionEnvironmentID.ValueString(),
		Label:                   data.Label.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Deployment", err.Error())
		return
	}

	deployment, err := r.waitForDeploymentToBeReady(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Deployment is not ready", err.Error())
		return
	}
	data.ID = types.StringValue(deployment.ID)

	if data.Settings != nil {
		err = r.updateDeploymentSettings(ctx, createResp.ID, data.Settings)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Deployment settings", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
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

	// TODO: read deployment settings from various sources

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	if plan.Label.ValueString() != state.Label.ValueString() {
		traceAPICall("UpdateDeployment")
		_, err := r.provider.service.UpdateDeployment(ctx,
			id,
			&client.UpdateDeploymentRequest{
				Label: plan.Label.ValueString(),
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

	if plan.Settings != state.Settings {
		err := r.updateDeploymentSettings(ctx, id, plan.Settings)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Deployment settings", err.Error())
			return
		}
		state.Settings = plan.Settings
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

func (r *DeploymentResource) waitForDeploymentToBeReady(ctx context.Context, id string) (*client.DeploymentRetrieveResponse, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		deployment, err := r.provider.service.GetDeployment(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}

		if deployment.Status == "active" {
			return nil
		} else if deployment.Status == "errored" {
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
	settings *DeploymentSettings,
) (err error) {
	req := &client.DeploymentSettings{}
	if settings != nil {
		// Association ID
		req.AssociationID = &client.AssociationIDSetting{
			AutoGenerateID: true,
			ColumnNames:    []string{"id"},
		}
		if settings.AssociationID != nil {
			req.AssociationID = &client.AssociationIDSetting{
				AutoGenerateID: settings.AssociationID.AutoGenerateID.ValueBool(),
				ColumnNames:    []string{settings.AssociationID.FeatureName.ValueString()},
			}
		}

		// Prediction Row Storage
		req.PredictionsDataCollection = &client.BasicSetting{
			Enabled: true,
		}
		if IsKnown(settings.PredictionRowStorage) {
			req.PredictionsDataCollection = &client.BasicSetting{
				Enabled: settings.PredictionRowStorage.ValueBool(),
			}
		}

		// Challenger Analysis
		req.ChallengerModels = &client.BasicSetting{
			Enabled: false,
		}
		if IsKnown(settings.ChallengerAnalysis) {
			req.ChallengerModels = &client.BasicSetting{
				Enabled: settings.ChallengerAnalysis.ValueBool(),
			}
		}

		// Predictions Settings
		if settings.PredictionsSettings != nil {
			req.PredictionsSettings = &client.PredictionsSettings{
				MinComputes: int(settings.PredictionsSettings.MinComputes.ValueInt64()),
				MaxComputes: int(settings.PredictionsSettings.MaxComputes.ValueInt64()),
				RealTime:    settings.PredictionsSettings.RealTime.ValueBool(),
			}
		}
	}

	traceAPICall("UpdateDeploymentSettings")
	_, err = r.provider.service.UpdateDeploymentSettings(ctx, id, req)
	if err != nil {
		return
	}

	_, err = r.waitForDeploymentToBeReady(ctx, id)
	if err != nil {
		return
	}

	return
}
