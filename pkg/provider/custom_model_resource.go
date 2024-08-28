package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	defaultCustomModelType         = "inference"
	defaultModerationTimeout       = 60
	defaultModerationTimeoutAction = "score"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CustomModelResource{}
var _ resource.ResourceWithImportState = &CustomModelResource{}

func NewCustomModelResource() resource.Resource {
	return &CustomModelResource{}
}

// VectorDatabaseResource defines the resource implementation.
type CustomModelResource struct {
	provider *Provider
}

func (r *CustomModelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_model"
}

func (r *CustomModelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Data set from file",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the latest Custom Model version.",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Custom Model.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Custom Model.",
				Optional:            true,
			},
			"source_llm_blueprint_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the source LLM Blueprint for the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"base_environment_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the base environment for the Custom Model.",
			},
			"base_environment_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the base environment for the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"base_environment_version_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the base environment version for the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"runtime_parameters": schema.ListNestedAttribute{
				Optional: true,
				// Computed:            true,
				MarkdownDescription: "The runtime parameter values for the Custom Model.",
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
							MarkdownDescription: "The value of the runtime parameter.",
						},
					},
				},
			},
			"target_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The target type of the Custom Model.",
			},
			"target": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The target of the Custom Model.",
			},
			"is_proxy": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "The flag indicating if the Custom Model is a proxy model.",
			},
			"source_remote_repositories": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The source remote repositories for the Custom Model.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The ID of the source remote repository.",
						},
						"ref": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The reference of the source remote repository.",
						},
						"source_paths": schema.ListAttribute{
							Required:            true,
							MarkdownDescription: "The list of source paths in the source remote repository.",
							ElementType:         types.StringType,
						},
					},
				},
			},
			"local_files": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The list of local file paths used to build the Custom Model.",
			},
			"guard_configurations": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The guard configurations for the Custom Model.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"template_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The template name of the guard configuration.",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the guard configuration.",
						},
						"stages": schema.ListAttribute{
							Required:            true,
							MarkdownDescription: "The list of stages for the guard configuration.",
							ElementType:         types.StringType,
						},
						"intervention": schema.SingleNestedAttribute{
							Required:            true,
							MarkdownDescription: "The intervention for the guard configuration.",
							Attributes: map[string]schema.Attribute{
								"action": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "The action of the guard intervention.",
								},
								"message": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString("This message has triggered moderation criteria and therefore been blocked by the DataRobot moderation system."),
									MarkdownDescription: "The message of the guard intervention.",
								},
								"condition": schema.SingleNestedAttribute{
									Required:            true,
									MarkdownDescription: "The list of conditions for the guard intervention.",
									Attributes: map[string]schema.Attribute{
										"comparand": schema.Float64Attribute{
											Required:            true,
											MarkdownDescription: "The comparand of the guard condition.",
										},
										"comparator": schema.StringAttribute{
											Required:            true,
											MarkdownDescription: "The comparator of the guard condition.",
										},
									},
								},
							},
						},
						"deployment_id": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The deployment ID of this guard.",
						},
						"input_column_name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The input column name of this guard.",
						},
						"output_column_name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The output column name of this guard.",
						},
					},
				},
			},
			"overall_moderation_configuration": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The overall moderation configuration for the Custom Model.",
				Attributes: map[string]schema.Attribute{
					"timeout_sec": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(defaultModerationTimeout),
						MarkdownDescription: "The timeout in seconds of the overall moderation configuration.",
					},
					"timeout_action": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(defaultModerationTimeoutAction),
						MarkdownDescription: "The timeout action of the overall moderation configuration.",
					},
				},
			},
		},
	}
}

func (r *CustomModelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan CustomModelResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.Name) {
		resp.Diagnostics.AddError(
			"Invalid name",
			"Name is required to create a Custom Model.",
		)
		return
	}

	var state CustomModelResourceModel
	var customModelID string
	var baseEnvironmentID string
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()

	if IsKnown(plan.SourceLLMBlueprintID) {
		sourceBlueprintID := plan.SourceLLMBlueprintID.ValueString()

		traceAPICall("CreateCustomModelFromLLMBlueprint")
		createResp, err := r.provider.service.CreateCustomModelFromLLMBlueprint(ctx, &client.CreateCustomModelFromLLMBlueprintRequest{
			LLMBlueprintID: sourceBlueprintID,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating Custom Model",
				fmt.Sprintf("Unable to create Custom Model, got error: %s", err),
			)
			return
		}
		customModelID = createResp.CustomModelID

		state.ID = types.StringValue(createResp.CustomModelID)
		state.SourceLLMBlueprintID = types.StringValue(sourceBlueprintID)
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		traceAPICall("UpdateCustomModel")
		_, err = r.provider.service.UpdateCustomModel(ctx,
			createResp.CustomModelID,
			&client.CustomModelUpdate{
				Name:        name,
				Description: description,
			})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating Custom Model",
				fmt.Sprintf("Unable to update Custom Model, got error: %s", err),
			)
			return
		}

		traceAPICall("WaitForCustomModelToBeReady")
		customModel, err := r.waitForCustomModelToBeReady(ctx, customModelID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for Custom Model to be ready",
				fmt.Sprintf("Unable to wait for Custom Model to be ready, got error: %s", err),
			)
			return
		}
		baseEnvironmentID = customModel.LatestVersion.BaseEnvironmentID

		state.Name = types.StringValue(name)
		state.Description = types.StringValue(description)
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		if !IsKnown(plan.Target) || !IsKnown(plan.TargetType) || !IsKnown(plan.BaseEnvironmentName) {
			resp.Diagnostics.AddError(
				"Invalid Custom Model configuration",
				"Target, Target Type, and Base Environment Name are required to create a Custom Model without a Source LLM Blueprint.",
			)
			return
		}

		// verify the base environment exists
		traceAPICall("ListExecutionEnvironments")
		listResp, err := r.provider.service.ListExecutionEnvironments(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing Execution Environments",
				fmt.Sprintf("Unable to list Execution Environments, got error: %s", err),
			)
			return
		}

		for _, executionEnvironment := range listResp.Data {
			if executionEnvironment.Name == plan.BaseEnvironmentName.ValueString() {
				baseEnvironmentID = executionEnvironment.ID
				break
			}
		}

		if baseEnvironmentID == "" {
			resp.Diagnostics.AddError(
				"Base Environment not found",
				fmt.Sprintf("Base Environment with name %s is not found.", plan.BaseEnvironmentName.ValueString()),
			)
			return
		}

		traceAPICall("CreateCustomModel")
		createResp, err := r.provider.service.CreateCustomModel(ctx, &client.CreateCustomModelRequest{
			Name:            plan.Name.ValueString(),
			Description:     plan.Description.ValueString(),
			TargetType:      plan.TargetType.ValueString(),
			TargetName:      plan.Target.ValueString(),
			CustomModelType: defaultCustomModelType,
			IsProxyModel:    plan.IsProxy.ValueBool(),
			IsTrainingDataForVersionsPermanentlyEnabled: true,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating Custom Model",
				fmt.Sprintf("Unable to create Custom Model, got error: %s", err),
			)
			return
		}
		customModelID = createResp.ID

		state.ID = types.StringValue(customModelID)
		state.Name = types.StringValue(name)
		state.Description = types.StringValue(description)
		state.Target = types.StringValue(plan.Target.ValueString())
		state.TargetType = types.StringValue(plan.TargetType.ValueString())
		if IsKnown(plan.IsProxy) {
			state.IsProxy = plan.IsProxy
		}
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if plan.SourceRemoteRepositories == nil && plan.LocalFiles == nil {
			resp.Diagnostics.AddError(
				"Invalid Custom Model configuration",
				"Source Remote Repository or Local Files are required to create a Custom Model without a Source LLM Blueprint.",
			)
			return
		}
	}

	if plan.SourceRemoteRepositories != nil {
		for _, sourceRemoteRepository := range plan.SourceRemoteRepositories {
			errSummary, errDetail := r.createCustomModelVersionFromRemoteRepository(
				ctx,
				sourceRemoteRepository,
				customModelID,
				baseEnvironmentID,
			)
			if errSummary != "" {
				resp.Diagnostics.AddError(
					errSummary,
					errDetail,
				)
				return
			}
		}
	}

	errSummary, errDetail := r.createCustomModelVersionFromFiles(
		ctx,
		plan.LocalFiles,
		customModelID,
		baseEnvironmentID)
	if errSummary != "" {
		resp.Diagnostics.AddError(
			errSummary,
			errDetail,
		)
		return
	}

	traceAPICall("WaitForCustomModelToBeReady")
	customModel, err := r.waitForCustomModelToBeReady(ctx, customModelID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for Custom Model to be ready",
			fmt.Sprintf("Unable to wait for Custom Model to be ready, got error: %s", err),
		)
		return
	}

	state.VersionID = types.StringValue(customModel.LatestVersion.ID)
	state.BaseEnvironmentID = types.StringValue(customModel.LatestVersion.BaseEnvironmentID)
	state.BaseEnvironmentVersionID = types.StringValue(customModel.LatestVersion.BaseEnvironmentVersionID)
	state.BaseEnvironmentName = plan.BaseEnvironmentName
	state.SourceRemoteRepositories = plan.SourceRemoteRepositories
	state.LocalFiles = plan.LocalFiles
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(plan.RuntimeParameters) > 0 {
		params := make([]client.RuntimeParameterValueRequest, len(plan.RuntimeParameters))
		for i, param := range plan.RuntimeParameters {
			params[i] = client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     param.Value.ValueString(),
			}
		}
		jsonParams, err := json.Marshal(params)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating runtime parameters",
				fmt.Sprintf("Unable to create runtime parameters, got error: %s", err),
			)
			return
		}

		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:            "false",
			BaseEnvironmentID:        customModel.LatestVersion.BaseEnvironmentID,
			BaseEnvironmentVersionID: customModel.LatestVersion.BaseEnvironmentVersionID,
			RuntimeParameterValues:   string(jsonParams),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating Custom Model version",
				fmt.Sprintf("Unable to create Custom Model version, got error: %s", err),
			)
			return
		}

		traceAPICall("WaitForCustomModelToBeReady")
		customModel, err = r.waitForCustomModelToBeReady(ctx, customModelID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for Custom Model to be ready",
				fmt.Sprintf("Unable to wait for Custom Model to be ready, got error: %s", err),
			)
			return
		}

		paramKeys := make([]string, 0)
		for _, param := range plan.RuntimeParameters {
			paramKeys = append(paramKeys, param.Key.ValueString())
		}
		loadRuntimeParametersToTerraformState(paramKeys, customModel.LatestVersion.RuntimeParameters, &state)
		state.VersionID = types.StringValue(customModel.LatestVersion.ID)
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if len(plan.GuardConfigurations) > 0 {
		newVersion, errSummary, errDetail := r.createCustomModelVersionFromGuards(
			ctx,
			plan,
			customModelID,
			customModel.LatestVersion.ID,
			plan.GuardConfigurations,
			[]GuardConfiguration{},
		)
		if errSummary != "" {
			resp.Diagnostics.AddError(
				errSummary,
				errDetail,
			)
			return
		}
		state.VersionID = types.StringValue(newVersion)
		state.GuardConfigurations = plan.GuardConfigurations
		state.OverallModerationConfiguration = plan.OverallModerationConfiguration
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		traceAPICall("WaitForCustomModelToBeReady")
		_, err = r.waitForCustomModelToBeReady(ctx, customModelID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for Custom Model to be ready",
				fmt.Sprintf("Unable to wait for Custom Model to be ready, got error: %s", err),
			)
			return
		}
	}
}

func (r *CustomModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state CustomModelResourceModel
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

	traceAPICall("GetCustomModel")
	customModel, err := r.provider.service.GetCustomModel(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddWarning(
				"Custom Model not found",
				fmt.Sprintf("Custom Model with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Custom Model info",
				fmt.Sprintf("Unable to get Custom Model, got error: %s", err),
			)
		}
		return
	}

	paramKeys := make([]string, 0)
	for _, param := range state.RuntimeParameters {
		paramKeys = append(paramKeys, param.Key.ValueString())
	}
	loadCustomModelToTerraformState(
		id,
		customModel.LatestVersion.ID,
		customModel.Name,
		customModel.Description,
		state.SourceLLMBlueprintID.ValueString(),
		customModel.LatestVersion.BaseEnvironmentID,
		customModel.LatestVersion.BaseEnvironmentVersionID,
		paramKeys,
		customModel.LatestVersion.RuntimeParameters,
		&state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *CustomModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var plan CustomModelResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CustomModelResourceModel

	// Read Terraform state data into the model
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	newName := plan.Name.ValueString()
	newDescription := plan.Description.ValueString()

	if state.Name.ValueString() != newName ||
		state.Description.ValueString() != newDescription {
		traceAPICall("UpdateCustomModel")
		customModel, err := r.provider.service.UpdateCustomModel(ctx,
			id,
			&client.CustomModelUpdate{
				Name:        plan.Name.ValueString(),
				Description: plan.Description.ValueString(),
			})
		if err != nil {
			if errors.Is(err, &client.NotFoundError{}) {
				resp.Diagnostics.AddWarning(
					"Custom Model not found",
					fmt.Sprintf("Custom Model with ID %s is not found. Removing from state.", id))
				resp.State.RemoveResource(ctx)
			} else {
				resp.Diagnostics.AddError(
					"Error updating Custom Model",
					fmt.Sprintf("Unable to update Custom Model, got error: %s", err),
				)
			}
			return
		}

		state.Name = types.StringValue(customModel.Name)
		state.Description = types.StringValue(customModel.Description)
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	traceAPICall("WaitForCustomModelToBeReady")
	customModel, err := r.waitForCustomModelToBeReady(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for Custom Model to be ready",
			fmt.Sprintf("Unable to wait for Custom Model to be ready, got error: %s", err),
		)
		return
	}

	if len(plan.RuntimeParameters) > 0 ||
		state.BaseEnvironmentName.ValueString() != plan.BaseEnvironmentName.ValueString() {
		listResp, err := r.provider.service.ListExecutionEnvironments(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing Execution Environments",
				fmt.Sprintf("Unable to list Execution Environments, got error: %s", err),
			)
			return
		}

		baseEnvironmentID := customModel.LatestVersion.BaseEnvironmentID
		baseEnvironmentVersionID := customModel.LatestVersion.BaseEnvironmentVersionID

		if state.BaseEnvironmentName.ValueString() != plan.BaseEnvironmentName.ValueString() {
			var newBaseEnvironmentID string
			var newBaseEnvironmentVersionID string
			for _, executionEnvironment := range listResp.Data {
				if executionEnvironment.Name == plan.BaseEnvironmentName.ValueString() {
					newBaseEnvironmentID = executionEnvironment.ID
					newBaseEnvironmentVersionID = executionEnvironment.LatestVersion.ID
					break
				}
			}

			if newBaseEnvironmentID == "" {
				resp.Diagnostics.AddError(
					"Base Environment not found",
					fmt.Sprintf("Base Environment with name %s is not found.", plan.BaseEnvironmentName.ValueString()),
				)
				return
			}

			baseEnvironmentID = newBaseEnvironmentID
			baseEnvironmentVersionID = newBaseEnvironmentVersionID
		}

		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		params := make([]client.RuntimeParameterValueRequest, len(plan.RuntimeParameters))
		for i, param := range plan.RuntimeParameters {
			params[i] = client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     param.Value.ValueString(),
			}
		}
		jsonParams, err := json.Marshal(params)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating runtime parameters",
				fmt.Sprintf("Unable to create runtime parameters, got error: %s", err),
			)
			return
		}
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, id, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:            "false",
			BaseEnvironmentID:        baseEnvironmentID,
			BaseEnvironmentVersionID: baseEnvironmentVersionID,
			RuntimeParameterValues:   string(jsonParams),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating Custom Model version",
				fmt.Sprintf("Unable to create Custom Model version, got error: %s", err),
			)
			return
		}

		traceAPICall("WaitForCustomModelToBeReady")
		customModel, err = r.waitForCustomModelToBeReady(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for Custom Model to be ready",
				fmt.Sprintf("Unable to wait for Custom Model to be ready, got error: %s", err),
			)
			return
		}
		latestVersion := customModel.LatestVersion.ID

		paramKeys := make([]string, 0)
		for _, param := range plan.RuntimeParameters {
			paramKeys = append(paramKeys, param.Key.ValueString())
		}
		loadRuntimeParametersToTerraformState(paramKeys, customModel.LatestVersion.RuntimeParameters, &state)
		state.VersionID = types.StringValue(latestVersion)
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !reflect.DeepEqual(plan.SourceRemoteRepositories, state.SourceRemoteRepositories) {
		errSummary, errDetail := r.updateRemoteRepositories(ctx, customModel, state, plan)
		if errSummary != "" {
			resp.Diagnostics.AddError(
				errSummary,
				errDetail,
			)
			return
		}

		state.SourceRemoteRepositories = plan.SourceRemoteRepositories
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !reflect.DeepEqual(plan.LocalFiles, state.LocalFiles) {
		errSummary, errDetail := r.updateLocalFiles(ctx, customModel, state, plan)
		if errSummary != "" {
			resp.Diagnostics.AddError(
				errSummary,
				errDetail,
			)
			return
		}

		state.LocalFiles = plan.LocalFiles
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !reflect.DeepEqual(plan.GuardConfigurations, state.GuardConfigurations) {
		guardsToAdd := make([]GuardConfiguration, 0)
		for _, guard := range plan.GuardConfigurations {
			found := false
			for _, stateGuard := range state.GuardConfigurations {
				if reflect.DeepEqual(guard, stateGuard) {
					found = true
					break
				}
			}
			if !found {
				guardsToAdd = append(guardsToAdd, guard)
			}
		}

		guardsToRemove := make([]GuardConfiguration, 0)
		for _, stateGuard := range state.GuardConfigurations {
			found := false
			for _, guard := range plan.GuardConfigurations {
				if reflect.DeepEqual(guard, stateGuard) {
					found = true
					break
				}
			}
			if !found {
				guardsToRemove = append(guardsToRemove, stateGuard)
			}
		}

		newVersion, errSummary, errDetail := r.createCustomModelVersionFromGuards(
			ctx,
			plan,
			customModel.ID,
			customModel.LatestVersion.ID,
			guardsToAdd,
			guardsToRemove,
		)
		if errSummary != "" {
			resp.Diagnostics.AddError(
				errSummary,
				errDetail,
			)
			return
		}
		state.VersionID = types.StringValue(newVersion)
		state.GuardConfigurations = plan.GuardConfigurations
		state.OverallModerationConfiguration = plan.OverallModerationConfiguration
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	traceAPICall("WaitForCustomModelToBeReady")
	customModel, err = r.waitForCustomModelToBeReady(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for Custom Model to be ready",
			fmt.Sprintf("Unable to wait for Custom Model to be ready, got error: %s", err),
		)
		return
	}
	state.VersionID = types.StringValue(customModel.LatestVersion.ID)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *CustomModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.provider == nil || !r.provider.configured {
		addConfigureProviderErr(&resp.Diagnostics)
		return
	}

	var state CustomModelResourceModel

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

	traceAPICall("DeleteCustomModel")
	err := r.provider.service.DeleteCustomModel(ctx, id)
	if err != nil {
		if errors.Is(err, &client.NotFoundError{}) {
			// custom model is already gone, ignore the error and remove from state
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error getting Custom Model info",
				fmt.Sprintf("Unable to get  example, got error: %s", err),
			)
		}
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r *CustomModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func loadCustomModelToTerraformState(
	id,
	versionID,
	name,
	description string,
	sourceBlueprintId string,
	baseEnvironmentId string,
	baseEnvironmentVersionId string,
	paramKeys []string,
	runtimeParameterValues []client.RuntimeParameterResponse,
	state *CustomModelResourceModel,
) {
	state.ID = types.StringValue(id)
	state.VersionID = types.StringValue(versionID)
	state.Name = types.StringValue(name)
	state.Description = types.StringValue(description)
	if sourceBlueprintId != "" {
		state.SourceLLMBlueprintID = types.StringValue(sourceBlueprintId)
	}
	state.BaseEnvironmentID = types.StringValue(baseEnvironmentId)
	state.BaseEnvironmentVersionID = types.StringValue(baseEnvironmentVersionId)
	loadRuntimeParametersToTerraformState(paramKeys, runtimeParameterValues, state)
}

func loadRuntimeParametersToTerraformState(
	paramKeys []string,
	runtimeParameterValues []client.RuntimeParameterResponse,
	state *CustomModelResourceModel,
) {
	if len(runtimeParameterValues) == 0 {
		return
	}

	// copy parameter in stable order
	parameters := make([]RuntimeParameterValueModel, 0)
	sort.SliceStable(runtimeParameterValues, func(i, j int) bool {
		return runtimeParameterValues[i].FieldName < runtimeParameterValues[j].FieldName
	})
	for _, param := range runtimeParameterValues {
		for _, key := range paramKeys {
			if strings.EqualFold(key, param.FieldName) {
				parameters = append(parameters, RuntimeParameterValueModel{
					Key:   types.StringValue(param.FieldName),
					Type:  types.StringValue(param.Type),
					Value: types.StringValue(fmt.Sprintf("%v", param.CurrentValue)),
				})
			}
		}
	}
	state.RuntimeParameters = parameters
}

func (r *CustomModelResource) waitForCustomModelToBeReady(ctx context.Context, customModelId string) (*client.CustomModelResponse, error) {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 60 * time.Minute

	operation := func() error {
		ready, err := r.provider.service.IsCustomModelReady(ctx, customModelId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("custom model is not ready")
		}
		return nil
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return nil, err
	}

	traceAPICall("GetCustomModel")
	return r.provider.service.GetCustomModel(ctx, customModelId)
}

func (r *CustomModelResource) createCustomModelVersionFromRemoteRepository(
	ctx context.Context,
	sourceRemoteRepository SourceRemoteRepository,
	customModelID string,
	baseEnvironmentID string,
) (
	errSummary string,
	errDetail string,
) {
	sourcePaths := make([]string, 0)
	for _, path := range sourceRemoteRepository.SourcePaths {
		sourcePaths = append(sourcePaths, path.ValueString())
	}

	traceAPICall("CreateCustomModelVersionFromRemoteRepository")
	_, err := r.provider.service.CreateCustomModelVersionFromRemoteRepository(ctx, customModelID, &client.CreateCustomModelVersionFromRemoteRepositoryRequest{
		IsMajorUpdate:     false,
		BaseEnvironmentID: baseEnvironmentID,
		RepositoryID:      sourceRemoteRepository.ID.ValueString(),
		Ref:               sourceRemoteRepository.Ref.ValueString(),
		SourcePath:        sourcePaths,
	})
	if err != nil {
		errSummary = "Error creating Custom Model version from remote repository"
		errDetail = fmt.Sprintf("Unable to create Custom Model version from remote repository, got error: %s", err)
		return
	}

	// TODO: this is an async task, need to figure out how the status ID is generated
	time.Sleep(20 * time.Second)
	// err = r.provider.service.WaitForTaskStatus(ctx, id)
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Update Task not completed",
	// 		fmt.Sprintf("Error replacing model for deployemnt: %s", err),
	// 	)
	// 	return
	// }

	return
}

func (r *CustomModelResource) createCustomModelVersionFromFiles(
	ctx context.Context,
	files []basetypes.StringValue,
	customModelID string,
	baseEnvironmentID string,
) (
	errSummary string,
	errDetail string,
) {
	if files == nil {
		return
	}

	localFiles := make([]client.FileInfo, 0)
	for _, file := range files {
		filePath := file.ValueString()
		fileReader, err := os.Open(filePath)
		if err != nil {
			errSummary = "Error opening local file"
			errDetail = fmt.Sprintf("Error opening local file: %v", err)
			return
		}
		defer fileReader.Close()
		fileContent, err := io.ReadAll(fileReader)
		if err != nil {
			errSummary = "Error reading local file"
			errDetail = fmt.Sprintf("Error reading local file: %v", err)
			return
		}

		localFiles = append(localFiles, client.FileInfo{
			Name:    filePath,
			Path:    filePath,
			Content: fileContent,
		})
	}

	traceAPICall("CreateCustomModelVersionFromLocalFiles")
	_, err := r.provider.service.CreateCustomModelVersionFromFiles(ctx, customModelID, &client.CreateCustomModelVersionFromFilesRequest{
		BaseEnvironmentID: baseEnvironmentID,
		Files:             localFiles,
	})
	if err != nil {
		errSummary = "Error creating Custom Model version from local files"
		errDetail = fmt.Sprintf("Unable to create Custom Model version from local files, got error: %s", err)
		return
	}

	return
}

func (r *CustomModelResource) createCustomModelVersionFromGuards(
	ctx context.Context,
	plan CustomModelResourceModel,
	customModelID string,
	customModelVersion string,
	guardConfigsToAdd []GuardConfiguration,
	guardConfigsToRemove []GuardConfiguration,
) (
	latestVersion string,
	errSummary string,
	errDetail string,
) {
	getGuardConfigsResp, err := r.provider.service.GetGuardConfigurationsForCustomModelVersion(ctx, customModelVersion)
	if err != nil {
		errSummary = "Error getting guard configurations"
		errDetail = fmt.Sprintf("Unable to get guard configurations, got error: %s", err)
		return
	}

	newGuardConfigs := make([]client.GuardConfiguration, 0)
	for _, existingGuardConfig := range getGuardConfigsResp.Data {
		// check if the existing guard config is in the list of guards to remove
		found := false
		for _, guardConfigToRemove := range guardConfigsToRemove {
			if existingGuardConfig.Name == guardConfigToRemove.Name.ValueString() {
				found = true
				break
			}
		}

		if found {
			continue
		}

		intervention := existingGuardConfig.Intervention
		if intervention.Message == "" {
			intervention.Message = "This message has triggered moderation criteria and therefore been blocked by the DataRobot moderation system."
		}

		newGuardConfig := client.GuardConfiguration{
			Name:         existingGuardConfig.Name,
			Description:  existingGuardConfig.Description,
			Type:         existingGuardConfig.Type,
			Stages:       existingGuardConfig.Stages,
			Intervention: intervention,
			DeploymentID: existingGuardConfig.DeploymentID,
			NemoInfo:     existingGuardConfig.NemoInfo,
			ModelInfo:    existingGuardConfig.ModelInfo,
		}

		if existingGuardConfig.OOTBType != "" {
			newGuardConfig.OOTBType = existingGuardConfig.OOTBType
		}

		newGuardConfigs = append(newGuardConfigs, newGuardConfig)
	}

	guardTemplates, err := r.provider.service.ListGuardTemplates(ctx)
	if err != nil {
		errSummary = "Error listing guard templates"
		errDetail = fmt.Sprintf("Unable to list guard templates, got error: %s", err)
		return
	}

	for _, guardConfigToAdd := range guardConfigsToAdd {
		var guardTemplate *client.GuardTemplate
		for index := range guardTemplates.Data {
			template := guardTemplates.Data[index]
			if template.Name == guardConfigToAdd.TemplateName.ValueString() {
				guardTemplate = &template
				break
			}
		}

		if guardTemplate == nil {
			errSummary = "Guard template not found"
			errDetail = fmt.Sprintf("Guard template with name %s is not found.", guardConfigToAdd.TemplateName.ValueString())
			return
		}

		stages := make([]string, 0)
		for _, stage := range guardConfigToAdd.Stages {
			stages = append(stages, stage.ValueString())
		}

		newGuardConfig := client.GuardConfiguration{
			Name:        guardConfigToAdd.Name.ValueString(),
			Description: guardTemplate.Description,
			Type:        guardTemplate.Type,
			Stages:      stages,
			Intervention: client.GuardIntervention{
				Action:         guardConfigToAdd.Intervention.Action.ValueString(),
				AllowedActions: guardTemplate.Intervention.AllowedActions,
				Message:        guardConfigToAdd.Intervention.Message.ValueString(),
				Conditions: []client.GuardCondition{
					{
						Comparand:  guardConfigToAdd.Intervention.Condition.Comparand.ValueFloat64(),
						Comparator: guardConfigToAdd.Intervention.Condition.Comparator.ValueString(),
					},
				},
			},
			ModelInfo: guardTemplate.ModelInfo,
			// TODO: allow user to input Nemo Info
			NemoInfo:     guardTemplate.NemoInfo,
			DeploymentID: guardConfigToAdd.DeploymentID.ValueString(),
		}

		if guardTemplate.OOTBType != "" {
			newGuardConfig.OOTBType = guardTemplate.OOTBType
		}

		if IsKnown(guardConfigToAdd.InputColumnName) && IsKnown(guardConfigToAdd.OutputColumnName) {
			traceAPICall("GetDeployment")
			deployment, err := r.provider.service.GetDeployment(ctx, guardConfigToAdd.DeploymentID.ValueString())
			if err != nil {
				errSummary = "Error getting deployment"
				errDetail = fmt.Sprintf("Unable to get deployment info for guard, got error: %s", err)
				return
			}

			newGuardConfig.ModelInfo = client.GuardModelInfo{
				InputColumnName:  guardConfigToAdd.InputColumnName.ValueString(),
				OutputColumnName: guardConfigToAdd.OutputColumnName.ValueString(),
				TargetType:       deployment.Model.TargetType,
			}
		}

		newGuardConfigs = append(newGuardConfigs, newGuardConfig)
	}

	overallModerationConfig := client.OverallModerationConfiguration{
		TimeoutSec:    defaultModerationTimeout,
		TimeoutAction: defaultModerationTimeoutAction,
	}

	if plan.OverallModerationConfiguration != nil {
		overallModerationConfig.TimeoutSec = int(plan.OverallModerationConfiguration.TimeoutSec.ValueInt64())
		overallModerationConfig.TimeoutAction = plan.OverallModerationConfiguration.TimeoutAction.ValueString()
	}

	traceAPICall("CreateCustomModelVersionFromGuardConfigurations")
	createFromGuardsResp, err := r.provider.service.CreateCustomModelVersionFromGuardConfigurations(ctx, customModelVersion, &client.CreateCustomModelVersionFromGuardsConfigurationRequest{
		CustomModelID: customModelID,
		Data:          newGuardConfigs,
		OverallConfig: overallModerationConfig,
	})
	if err != nil {
		errSummary = "Error creating Custom Model version from guard configurations"
		errDetail = fmt.Sprintf("Unable to create Custom Model version from guard configurations, got error: %s", err)
		return
	}

	latestVersion = createFromGuardsResp.CustomModelVersionID
	return
}

func (r *CustomModelResource) updateRemoteRepositories(
	ctx context.Context,
	customModel *client.CustomModelResponse,
	state CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	errSummary string,
	errDetail string,
) {
	filesToDelete := make([]string, 0)
	for _, oldSourceRemoteRepository := range state.SourceRemoteRepositories {
		for _, filePath := range oldSourceRemoteRepository.SourcePaths {
			found := false

			for _, newSourceRemoteRepository := range plan.SourceRemoteRepositories {
				if oldSourceRemoteRepository.ID == newSourceRemoteRepository.ID &&
					oldSourceRemoteRepository.Ref == newSourceRemoteRepository.Ref &&
					contains(newSourceRemoteRepository.SourcePaths, filePath) {
					found = true
					break
				}
			}

			if !found {
				remoteRepository, err := r.provider.service.GetRemoteRepository(ctx, oldSourceRemoteRepository.ID.ValueString())
				if err != nil {
					errSummary = "Error getting remote repository"
					errDetail = fmt.Sprintf("Unable to get remote repository, got error: %s", err)
					return
				}

				for _, item := range customModel.LatestVersion.Items {
					if item.RepositoryFilePath == filePath.ValueString() &&
						item.Ref == oldSourceRemoteRepository.Ref.ValueString() &&
						item.FileSource == remoteRepository.SourceType &&
						item.RepositoryLocation == remoteRepository.Location {
						filesToDelete = append(filesToDelete, item.ID)
					}
				}
			}
		}
	}

	if len(filesToDelete) > 0 {
		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		_, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:     "false",
			BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
			FilesToDelete:     filesToDelete,
		})
		if err != nil {
			errSummary = "Error creating Custom Model version"
			errDetail = fmt.Sprintf("Unable to create Custom Model version, got error: %s", err)
			return
		}
	}

	for _, newSourceRemoteRepository := range plan.SourceRemoteRepositories {
		filesToAdd := make([]string, 0)
		for _, filePath := range newSourceRemoteRepository.SourcePaths {
			found := false

			for _, oldSourceRemoteRepository := range state.SourceRemoteRepositories {
				if oldSourceRemoteRepository.ID == newSourceRemoteRepository.ID &&
					oldSourceRemoteRepository.Ref == newSourceRemoteRepository.Ref &&
					contains(oldSourceRemoteRepository.SourcePaths, filePath) {
					found = true
					break
				}
			}

			if !found {
				filesToAdd = append(filesToAdd, filePath.ValueString())
			}
		}

		if len(filesToAdd) > 0 {
			errSummary, errDetail = r.createCustomModelVersionFromRemoteRepository(
				ctx,
				newSourceRemoteRepository,
				customModel.ID,
				customModel.LatestVersion.BaseEnvironmentID,
			)
			if errSummary != "" {
				return
			}
		}
	}

	return
}

func (r *CustomModelResource) updateLocalFiles(
	ctx context.Context,
	customModel *client.CustomModelResponse,
	state CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	errSummary string,
	errDetail string,
) {
	filesToDelete := make([]string, 0)
	for _, file := range state.LocalFiles {
		if !contains(plan.LocalFiles, file) {
			for _, item := range customModel.LatestVersion.Items {
				if item.FilePath == file.ValueString() && item.FileSource == "local" {
					filesToDelete = append(filesToDelete, item.ID)
				}
			}
		}
	}

	if len(filesToDelete) > 0 {
		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		_, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:     "false",
			BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
			FilesToDelete:     filesToDelete,
		})
		if err != nil {
			errSummary = "Error creating Custom Model version"
			errDetail = fmt.Sprintf("Unable to create Custom Model version, got error: %s", err)
			return
		}
	}

	filesToAdd := make([]string, 0)
	for _, file := range plan.LocalFiles {
		if !contains(state.LocalFiles, file) {
			filesToAdd = append(filesToAdd, file.ValueString())
		}
	}

	if len(filesToAdd) > 0 {
		return r.createCustomModelVersionFromFiles(
			ctx,
			plan.LocalFiles,
			customModel.ID,
			customModel.LatestVersion.BaseEnvironmentID,
		)
	}

	return
}
