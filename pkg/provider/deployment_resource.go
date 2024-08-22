package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/omnistrate/terraform-provider-datarobot/internal/client"
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
							"min_computes": schema.NumberAttribute{
								Required:            true,
								MarkdownDescription: "The minimum number of computes to use for predictions.",
							},
							"max_computes": schema.NumberAttribute{
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
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan DeploymentResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.RegisteredModelVersionID) {
		resp.Diagnostics.AddError(
			"Invalid registered model version ID",
			"Registered Model Version ID is required to create a Deployment.",
		)
		return
	}
	registeredModelVersionID := plan.RegisteredModelVersionID.ValueString()

	if !IsKnown(plan.PredictionEnvironmentID) {
		resp.Diagnostics.AddError(
			"Invalid prediction environment ID",
			"Prediction Environment ID is required to create a Deployment.",
		)
		return
	}
	predictionEnvironmentID := plan.PredictionEnvironmentID.ValueString()

	if !IsKnown(plan.Label) {
		resp.Diagnostics.AddError(
			"Invalid label",
			"Label is required to create a Deployment.",
		)
		return
	}
	label := plan.Label.ValueString()

	traceAPICall("CreateDeployment")
	createResp, err := r.provider.service.CreateDeploymentFromModelPackage(ctx, &client.CreateDeploymentFromModelPackageRequest{
		ModelPackageID:          registeredModelVersionID,
		PredictionEnvironmentID: predictionEnvironmentID,
		Label:                   label,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Deployment",
			fmt.Sprintf("Unable to create Deployment, got error: %s", err),
		)
		return
	}

	deployment, err := r.waitForDeploymentToBeReady(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Deployment not ready",
			"Deployment is not ready after 5 minutes or failed to check the status.")
		return
	}

	if plan.Settings != nil {
		deployment, err = r.updateDeploymentSettings(ctx, createResp.ID, plan.Settings)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Deployment settings",
				fmt.Sprintf("Unable to update Deployment settings, got error: %s", err),
			)
			return
		}
	}

	var state DeploymentResourceModel
	loadDeploymentToTerraformState(
		deployment.ID,
		deployment.Label,
		registeredModelVersionID,
		predictionEnvironmentID,
		plan.Settings,
		&state,
	)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state DeploymentResourceModel
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
	registeredModelVersionID := state.RegisteredModelVersionID.ValueString()
	predictionEnvironmentID := state.PredictionEnvironmentID.ValueString()

	traceAPICall("GetDeployment")
	deployment, err := r.provider.service.GetDeployment(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Deployment not found",
				fmt.Sprintf("Deployment with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Deployment info",
				fmt.Sprintf("Unable to get Deployment, got error: %s", err),
			)
		}
		return
	}

	loadDeploymentToTerraformState(
		id,
		deployment.Label,
		registeredModelVersionID,
		predictionEnvironmentID,
		state.Settings,
		&state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan DeploymentResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DeploymentResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
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
				resp.Diagnostics.AddError(
					"Error updating Deployment",
					fmt.Sprintf("Unable to update Deployment, got error: %s", err),
				)
			}
			return
		}

		_, err = r.waitForDeploymentToBeReady(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Deployment not ready",
				"Deployment is not ready after 5 minutes or failed to check the status.")
			return
		}
	}

	if plan.RegisteredModelVersionID != state.RegisteredModelVersionID {
		traceAPICall("ValidateDeploymentModelReplacement")
		validateModelReplacementResp, err := r.provider.service.ValidateDeploymentModelReplacement(ctx, id, &client.ValidateDeployemntModelReplacementRequest{
			ModelPackageID: plan.RegisteredModelVersionID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error validating model replacement",
				fmt.Sprintf("Unable to validate model replacement, got error: %s", err),
			)
			return
		}

		if validateModelReplacementResp.Status != "passing" {
			// TODO: should we try to replace the entire Deployment here?
			resp.Diagnostics.AddError(
				"Invalid model replacement",
				fmt.Sprintf("Model Replacement is invalid for the Deployment, got error: %s", err),
			)
			return
		}

		traceAPICall("UpdateDeploymentModel")
		_, err = r.provider.service.UpdateDeploymentModel(ctx, id, &client.UpdateDeploymentModelRequest{
			ModelPackageID: plan.RegisteredModelVersionID.ValueString(),
			Reason:         "OTHER",
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Deployment model",
				fmt.Sprintf("Unable to update Deployment model, got error: %s", err),
			)
			return
		}

		// TODO: where does this status ID come from?
		// model replacement is an async operation, separate from waiting for the deployment to be ready
		// err = r.provider.service.WaitForTaskStatus(ctx, id)
		// if err != nil {
		// 	resp.Diagnostics.AddError(
		// 		"Update Task not completed",
		// 		fmt.Sprintf("Error replacing model for deployemnt: %s", err),
		// 	)
		// 	return
		// }

		_, err = r.waitForDeploymentToBeReady(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Deployment not ready",
				"Deployment is not ready after 5 minutes or failed to check the status.")
			return
		}
	}

	if plan.Settings != state.Settings {
		_, err := r.updateDeploymentSettings(ctx, id, plan.Settings)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Deployment settings",
				fmt.Sprintf("Unable to update Deployment settings, got error: %s", err),
			)
			return
		}
		state.Settings = plan.Settings
	}

	loadDeploymentToTerraformState(
		id,
		plan.Label.ValueString(),
		plan.RegisteredModelVersionID.ValueString(),
		plan.PredictionEnvironmentID.ValueString(),
		plan.Settings,
		&state,
	)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state DeploymentResourceModel

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

	traceAPICall("DeleteDeployment")
	err := r.provider.service.DeleteDeployment(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// Deployment is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error deleting Deployment",
				fmt.Sprintf("Unable to delete Deployment, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *DeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadDeploymentToTerraformState(
	id string,
	label string,
	registeredModelVersionId string,
	predictionEnvironmentId string,
	settings *DeploymentSettings,
	state *DeploymentResourceModel,
) {
	state.ID = types.StringValue(id)
	state.Label = types.StringValue(label)
	state.RegisteredModelVersionID = types.StringValue(registeredModelVersionId)
	state.PredictionEnvironmentID = types.StringValue(predictionEnvironmentId)
	state.Settings = settings
}

func (r *DeploymentResource) waitForDeploymentToBeReady(ctx context.Context, id string) (*client.DeploymentRetrieveResponse, error) {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 5 * time.Minute

	operation := func() error {
		ready, err := r.provider.service.IsDeploymentReady(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("deployment is not ready")
		}
		return nil
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
) (*client.DeploymentRetrieveResponse, error) {
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
			req.PredictionsSettings = &client.PredictionsSetting{
				MinComputes: int(settings.PredictionsSettings.MinComputes.ValueInt32()),
				MaxComputes: int(settings.PredictionsSettings.MaxComputes.ValueInt32()),
				RealTime:    settings.PredictionsSettings.RealTime.ValueBool(),
			}
		}
	}

	traceAPICall("UpdateDeploymentSettings")
	_, err := r.provider.service.UpdateDeploymentSettings(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("unable to update Deployment settings, got error: %s", err)
	}

	deployment, err := r.waitForDeploymentToBeReady(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("deployment is not ready after 5 minutes or failed to check the status")
	}

	return deployment, nil
}
