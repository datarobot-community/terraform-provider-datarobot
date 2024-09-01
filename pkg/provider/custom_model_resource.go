package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	defaultCustomModelType = "inference"

	defaultModerationTimeout       = 60
	defaultModerationTimeoutAction = "score"

	defaultMemoryMB      = 2048
	defaultReplicas      = 1
	defaultNetworkAccess = "PUBLIC"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithImportState = &CustomModelResource{}
var _ resource.ResourceWithModifyPlan = &CustomModelResource{}

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
			"runtime_parameter_values": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
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
			"positive_class_label": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The positive class label of the Custom Model.",
			},
			"negative_class_label": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The negative class label of the Custom Model.",
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
			"resource_settings": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The resource settings for the Custom Model.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					map[string]attr.Type{
						"memory_mb":      types.Int64Type,
						"replicas":       types.Int64Type,
						"network_access": types.StringType,
					},
					map[string]attr.Value{
						"memory_mb":      types.Int64Value(defaultMemoryMB),
						"replicas":       types.Int64Value(defaultReplicas),
						"network_access": types.StringValue(defaultNetworkAccess),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"memory_mb": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(defaultMemoryMB),
						MarkdownDescription: "The memory in MB for the Custom Model.",
					},
					"replicas": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(defaultReplicas),
						MarkdownDescription: "The replicas for the Custom Model.",
					},
					"network_access": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(defaultNetworkAccess),
						MarkdownDescription: "The network access for the Custom Model.",
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
	var plan CustomModelResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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

		traceAPICall("UpdateCustomModel")
		_, err = r.provider.service.UpdateCustomModel(ctx,
			createResp.CustomModelID,
			&client.CustomModelUpdate{
				Name:        name,
				Description: description,
			})
		if err != nil {
			resp.Diagnostics.AddError("Error updating Custom Model", err.Error())
			return
		}

		traceAPICall("WaitForCustomModelToBeReady")
		customModel, err := r.waitForCustomModelToBeReady(ctx, customModelID)
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
			return
		}
		baseEnvironmentID = customModel.LatestVersion.BaseEnvironmentID

		state.Name = types.StringValue(name)
		state.Description = types.StringValue(description)
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
			resp.Diagnostics.AddError("Error listing Execution Environments", err.Error())
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
			Name:               plan.Name.ValueString(),
			Description:        plan.Description.ValueString(),
			TargetType:         plan.TargetType.ValueString(),
			TargetName:         plan.Target.ValueString(),
			CustomModelType:    defaultCustomModelType,
			PositiveClassLabel: plan.PositiveClassLabel.ValueString(),
			NegativeClassLabel: plan.NegativeClassLabel.ValueString(),
			IsProxyModel:       plan.IsProxy.ValueBool(),
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
		if IsKnown(plan.PositiveClassLabel) {
			state.PositiveClassLabel = types.StringValue(plan.PositiveClassLabel.ValueString())
		}
		if IsKnown(plan.NegativeClassLabel) {
			state.NegativeClassLabel = types.StringValue(plan.NegativeClassLabel.ValueString())
		}
		if IsKnown(plan.IsProxy) {
			state.IsProxy = plan.IsProxy
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
		resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
		return
	}

	state.VersionID = types.StringValue(customModel.LatestVersion.ID)
	state.BaseEnvironmentID = types.StringValue(customModel.LatestVersion.BaseEnvironmentID)
	state.BaseEnvironmentVersionID = types.StringValue(customModel.LatestVersion.BaseEnvironmentVersionID)
	state.BaseEnvironmentName = plan.BaseEnvironmentName
	state.SourceRemoteRepositories = plan.SourceRemoteRepositories
	state.LocalFiles = plan.LocalFiles

	if IsKnown(plan.RuntimeParameterValues) {
		runtimeParameterValues := make([]RuntimeParameterValue, 0)
		if diags := plan.RuntimeParameterValues.ElementsAs(ctx, &runtimeParameterValues, false); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		params := make([]client.RuntimeParameterValueRequest, len(runtimeParameterValues))
		for i, param := range runtimeParameterValues {
			value := param.Value.ValueString()
			params[i] = client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     &value,
			}
		}
		jsonParams, err := json.Marshal(params)
		if err != nil {
			resp.Diagnostics.AddError("Error creating runtime parameter values", err.Error())
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
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}

		traceAPICall("WaitForCustomModelToBeReady")
		customModel, err = r.waitForCustomModelToBeReady(ctx, customModelID)
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
			return
		}

		state.VersionID = types.StringValue(customModel.LatestVersion.ID)
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
	}

	if plan.ResourceSettings != nil {
		resourceSettings := *plan.ResourceSettings
		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:       "false",
			BaseEnvironmentID:   customModel.LatestVersion.BaseEnvironmentID,
			Replicas:            resourceSettings.Replicas.ValueInt64(),
			NetworkEgressPolicy: resourceSettings.NetworkAccess.ValueString(),
			MaximumMemory:       resourceSettings.MemoryMB.ValueInt64() * 1024 * 1024, // convert MB to bytes
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}
		state.ResourceSettings = plan.ResourceSettings

		customModel, err = r.waitForCustomModelToBeReady(ctx, customModelID)
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
			return
		}
		state.VersionID = types.StringValue(customModel.LatestVersion.ID)
	}

	customModel, err = r.waitForCustomModelToBeReady(ctx, customModelID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
		return
	}

	state.RuntimeParameterValues, diags = formatRuntimeParameterValues(ctx, customModel.LatestVersion.RuntimeParameters)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *CustomModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomModelResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		return
	}

	id := data.ID.ValueString()

	traceAPICall("GetCustomModel")
	customModel, err := r.provider.service.GetCustomModel(ctx, id)
	if err != nil {
		if _, ok := err.(*client.NotFoundError); ok {
			resp.Diagnostics.AddWarning(
				"Custom Model not found",
				fmt.Sprintf("Custom Model with ID %s is not found. Removing from state.", id))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error getting Custom Model with ID %s", id),
				err.Error())
		}
		return
	}

	data.RuntimeParameterValues, diags = formatRuntimeParameterValues(ctx, customModel.LatestVersion.RuntimeParameters)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	loadCustomModelToTerraformState(
		ctx,
		id,
		customModel.Name,
		customModel.Description,
		data.SourceLLMBlueprintID.ValueString(),
		customModel.LatestVersion,
		&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomModelResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CustomModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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
				resp.Diagnostics.AddError("Error updating Custom Model", err.Error())
			}
			return
		}

		state.Name = types.StringValue(customModel.Name)
		state.Description = types.StringValue(customModel.Description)
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	traceAPICall("WaitForCustomModelToBeReady")
	customModel, err := r.waitForCustomModelToBeReady(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
		return
	}

	planRuntimeParametersValues, _ := types.ListValueFrom(
		ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":   types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			},
		}, []RuntimeParameterValue{})

	if IsKnown(plan.RuntimeParameterValues) {
		planRuntimeParametersValues = plan.RuntimeParameterValues
	}

	if !reflect.DeepEqual(planRuntimeParametersValues, state.RuntimeParameterValues) ||
		state.BaseEnvironmentName.ValueString() != plan.BaseEnvironmentName.ValueString() {

		traceAPICall("ListExecutionEnvironments")
		listResp, err := r.provider.service.ListExecutionEnvironments(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Error listing Execution Environments", err.Error())
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

		runtimeParameterValues := make([]RuntimeParameterValue, 0)
		if IsKnown(plan.RuntimeParameterValues) {
			if diags := plan.RuntimeParameterValues.ElementsAs(ctx, &runtimeParameterValues, false); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}

		params := make([]client.RuntimeParameterValueRequest, len(runtimeParameterValues))
		for i, param := range runtimeParameterValues {
			value := param.Value.ValueString()
			params[i] = client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     &value,
			}
		}

		// compute the runtime parameter values to reset
		runtimeParametersToReset := make([]RuntimeParameterValue, 0)
		if diags := state.RuntimeParameterValues.ElementsAs(ctx, &runtimeParametersToReset, false); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, param := range runtimeParametersToReset {
			found := false
			for _, newParam := range runtimeParameterValues {
				if param.Key.ValueString() == newParam.Key.ValueString() {
					found = true
					break
				}
			}
			if !found {
				params = append(params, client.RuntimeParameterValueRequest{
					FieldName: param.Key.ValueString(),
					Type:      param.Type.ValueString(),
					Value:     nil,
				})
			}
		}

		jsonParams, err := json.Marshal(params)
		if err != nil {
			resp.Diagnostics.AddError("Error creating runtime parameters", err.Error())
			return
		}

		traceAPICall("CreateCustomModelVersionCreateFromLatestRuntimeParams")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, id, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:            "false",
			BaseEnvironmentID:        baseEnvironmentID,
			BaseEnvironmentVersionID: baseEnvironmentVersionID,
			RuntimeParameterValues:   string(jsonParams),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}

		traceAPICall("WaitForCustomModelToBeReady")
		customModel, err = r.waitForCustomModelToBeReady(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
			return
		}
		latestVersion := customModel.LatestVersion.ID

		state.VersionID = types.StringValue(latestVersion)
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
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
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
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
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
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
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !reflect.DeepEqual(plan.ResourceSettings, state.ResourceSettings) {
		resourceSettings := *plan.ResourceSettings
		traceAPICall("CreateCustomModelVersionCreateFromLatestResources")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:       "false",
			BaseEnvironmentID:   customModel.LatestVersion.BaseEnvironmentID,
			Replicas:            resourceSettings.Replicas.ValueInt64(),
			NetworkEgressPolicy: resourceSettings.NetworkAccess.ValueString(),
			MaximumMemory:       resourceSettings.MemoryMB.ValueInt64() * 1024 * 1024, // convert MB to bytes
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}
		state.ResourceSettings = plan.ResourceSettings
	}

	traceAPICall("WaitForCustomModelToBeReady")
	customModel, err = r.waitForCustomModelToBeReady(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
		return
	}
	state.VersionID = types.StringValue(customModel.LatestVersion.ID)

	state.RuntimeParameterValues, diags = formatRuntimeParameterValues(ctx, customModel.LatestVersion.RuntimeParameters)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *CustomModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	traceAPICall("DeleteCustomModel")
	err := r.provider.service.DeleteCustomModel(ctx, data.ID.ValueString())
	if err != nil {
		if !errors.Is(err, &client.NotFoundError{}) {
			resp.Diagnostics.AddError("Error deleting Custom Model", err.Error())
			return
		}
	}
}

func (r *CustomModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r CustomModelResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		// Resource is being created or destroyed
		return
	}

	var plan CustomModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.RuntimeParameterValues) {
		// use empty list if runtime parameter values are unknown
		plan.RuntimeParameterValues, _ = types.ListValueFrom(
			ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"type":  types.StringType,
					"value": types.StringType,
				},
			}, []RuntimeParameterValue{})
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func loadCustomModelToTerraformState(
	ctx context.Context,
	id,
	name,
	description,
	sourceBlueprintId string,
	latestVersion client.CustomModelVersionResponse,
	state *CustomModelResourceModel,
) {
	state.ID = types.StringValue(id)
	state.VersionID = types.StringValue(latestVersion.ID)
	state.Name = types.StringValue(name)
	if description != "" {
		state.Description = types.StringValue(description)
	}
	if sourceBlueprintId != "" {
		state.SourceLLMBlueprintID = types.StringValue(sourceBlueprintId)
	}
	state.BaseEnvironmentID = types.StringValue(latestVersion.BaseEnvironmentID)
	state.BaseEnvironmentVersionID = types.StringValue(latestVersion.BaseEnvironmentVersionID)
	loadResourceSettingsToTerraformState(latestVersion, state)
}

func loadResourceSettingsToTerraformState(
	customModelVersion client.CustomModelVersionResponse,
	state *CustomModelResourceModel,
) {
	resourceSettings := &CustomModelResourceSettings{
		MemoryMB:      types.Int64Value(defaultMemoryMB),
		Replicas:      types.Int64Value(defaultReplicas),
		NetworkAccess: types.StringValue(defaultNetworkAccess),
	}

	if customModelVersion.MaximumMemory != nil {
		resourceSettings.MemoryMB = types.Int64Value(*customModelVersion.MaximumMemory / 1024 / 1024) // convert bytes to MB
	}
	if customModelVersion.Replicas != nil {
		resourceSettings.Replicas = types.Int64Value(*customModelVersion.Replicas)
	}
	if customModelVersion.NetworkEgressPolicy != nil {
		resourceSettings.NetworkAccess = types.StringValue(*customModelVersion.NetworkEgressPolicy)
	}

	state.ResourceSettings = resourceSettings
}

func (r *CustomModelResource) waitForCustomModelToBeReady(ctx context.Context, customModelId string) (*client.CustomModelResponse, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("IsCustomModelReady")
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
	_, statusID, err := r.provider.service.CreateCustomModelVersionFromRemoteRepository(ctx, customModelID, &client.CreateCustomModelVersionFromRemoteRepositoryRequest{
		IsMajorUpdate:     false,
		BaseEnvironmentID: baseEnvironmentID,
		RepositoryID:      sourceRemoteRepository.ID.ValueString(),
		Ref:               sourceRemoteRepository.Ref.ValueString(),
		SourcePath:        sourcePaths,
	})
	if err != nil {
		errSummary = "Error creating Custom Model version from remote repository"
		errDetail = err.Error()
		return
	}

	err = waitForTaskStatusToComplete(ctx, r.provider.service, statusID)
	if err != nil {
		errSummary = "Error waiting for Custom Model version to be created from remote repository"
		errDetail = err.Error()
		return
	}

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
			errDetail = err.Error()
			return
		}
		defer fileReader.Close()
		fileContent, err := io.ReadAll(fileReader)
		if err != nil {
			errSummary = "Error reading local file"
			errDetail = err.Error()
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
		errDetail = err.Error()
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
		errDetail = err.Error()
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
		errDetail = err.Error()
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
				errSummary = fmt.Sprintf("Error getting deployment with ID %s", guardConfigToAdd.DeploymentID.ValueString())
				errDetail = err.Error()
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
		errDetail = err.Error()
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
					errDetail = err.Error()
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
		traceAPICall("CreateCustomModelVersionCreateFromLatestDeleteFiles")
		_, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:     "false",
			BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
			FilesToDelete:     filesToDelete,
		})
		if err != nil {
			errSummary = "Error creating Custom Model version"
			errDetail = err.Error()
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
		traceAPICall("CreateCustomModelVersionCreateFromLatestDeleteFiles")
		_, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionCreateFromLatestRequest{
			IsMajorUpdate:     "false",
			BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
			FilesToDelete:     filesToDelete,
		})
		if err != nil {
			errSummary = "Error creating Custom Model version"
			errDetail = err.Error()
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
