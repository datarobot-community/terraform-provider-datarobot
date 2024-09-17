package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
var _ resource.ResourceWithConfigValidators = &CustomModelResource{}

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
							MarkdownDescription: "The value of the runtime parameter (type conversion is handled internally).",
						},
					},
				},
			},
			"target_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The target type of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The target name of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringRequiresReplaceIfDeployed(),
				},
			},
			"positive_class_label": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The positive class label of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringRequiresReplaceIfDeployed(),
				},
			},
			"negative_class_label": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The negative class label of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringRequiresReplaceIfDeployed(),
				},
			},
			"prediction_threshold": schema.Float64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The prediction threshold of the Custom Model.",
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
			"language": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The language used to build the Custom Model.",
			},
			"is_proxy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Flag indicating if the Custom Model is a proxy model.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"class_labels": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Class labels for multiclass classification. Cannot be used with class_labels_file.",
				PlanModifiers: []planmodifier.List{
					listRequiresReplaceIfDeployed(),
				},
			},
			"class_labels_file": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to file containing newline separated class labels for multiclass classification. Cannot be used with class_labels.",
				PlanModifiers: []planmodifier.String{
					stringRequiresReplaceIfDeployed(),
				},
			},
			"deployments_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The number of deployments for the Custom Model.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
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
			"folder_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The path to a folder containing files to build the Custom Model. Each file in the folder is uploaded under path relative to a folder path.",
			},
			"files": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: "The list of tuples, where values in each tuple are the local filesystem path and the path the file should be placed in the Custom Model. If list is of strings, then basenames will be used for tuples.",
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
						"openai_credential": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The ID of an OpenAI credential for this guard.",
							Validators: []validator.String{
								stringvalidator.AlsoRequires(
									path.MatchRelative().AtParent().AtName("llm_type"),
								),
							},
						},
						"openai_deployment_id": schema.StringAttribute{
							Validators: []validator.String{
								stringvalidator.AlsoRequires(
									path.MatchRelative().AtParent().AtName("openai_credential"),
									path.MatchRelative().AtParent().AtName("openai_api_base"),
								),
							},
							Optional:            true,
							MarkdownDescription: "The ID of an OpenAI deployment for this guard.",
						},
						"openai_api_base": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The OpenAI API base URL for this guard.",
							Validators: []validator.String{
								stringvalidator.AlsoRequires(
									path.MatchRelative().AtParent().AtName("openai_credential"),
									path.MatchRelative().AtParent().AtName("openai_deployment_id"),
								),
							},
						},
						"llm_type": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The LLM type for this guard.",
							Validators: []validator.String{
								stringvalidator.OneOf("openAi", "azureOpenAi"),
								stringvalidator.AlsoRequires(
									path.MatchRelative().AtParent().AtName("openai_credential"),
								),
							},
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
			"training_dataset_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the training dataset assigned to the Custom Model.",
			},
			"training_dataset_version_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The version ID of the training dataset assigned to the Custom Model.",
			},
			"training_dataset_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the training dataset assigned to the Custom Model.",
			},
			"training_data_partition_column": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The name of the partition column in the training dataset assigned to the Custom Model.",
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
			resp.Diagnostics.AddError("Error creating Custom Model", err.Error())
			return
		}
		customModelID = createResp.CustomModelID

		state.ID = types.StringValue(createResp.CustomModelID)
		state.SourceLLMBlueprintID = types.StringValue(sourceBlueprintID)

		traceAPICall("UpdateCustomModel")
		_, err = r.provider.service.UpdateCustomModel(ctx,
			createResp.CustomModelID,
			&client.UpdateCustomModelRequest{
				Name:        name,
				Description: description,
				TargetName:  plan.TargetName.ValueString(),
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
		state.Description = plan.Description
	} else {
		if !IsKnown(plan.TargetType) || !IsKnown(plan.BaseEnvironmentName) {
			resp.Diagnostics.AddError(
				"Invalid Custom Model configuration",
				"Target Type, and Base Environment Name are required to create a Custom Model without a Source LLM Blueprint.",
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

		classLabels, err := getClassLabels(plan)
		if err != nil {
			resp.Diagnostics.AddError("Error getting class labels from file", err.Error())
		}

		traceAPICall("CreateCustomModel")
		createResp, err := r.provider.service.CreateCustomModel(ctx, &client.CreateCustomModelRequest{
			Name:                plan.Name.ValueString(),
			Description:         plan.Description.ValueString(),
			TargetType:          plan.TargetType.ValueString(),
			TargetName:          plan.TargetName.ValueString(),
			CustomModelType:     defaultCustomModelType,
			PositiveClassLabel:  plan.PositiveClassLabel.ValueString(),
			NegativeClassLabel:  plan.NegativeClassLabel.ValueString(),
			PredictionThreshold: plan.PredictionThreshold.ValueFloat64(),
			Language:            plan.Language.ValueString(),
			IsProxyModel:        plan.IsProxy.ValueBool(),
			ClassLabels:         classLabels,
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
		state.Description = plan.Description
		state.TargetType = types.StringValue(plan.TargetType.ValueString())
		state.ClassLabels = plan.ClassLabels
		state.ClassLabelsFile = plan.ClassLabelsFile
		state.Language = plan.Language

		if plan.SourceRemoteRepositories == nil && !IsKnown(plan.FolderPath) && !IsKnown(plan.Files) {
			resp.Diagnostics.AddError(
				"Invalid Custom Model configuration",
				"Source Remote Repository, Folder Path, or Files are required to create a Custom Model without a Source LLM Blueprint.",
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

	err := r.createCustomModelVersionFromFiles(
		ctx,
		plan.FolderPath,
		plan.Files,
		customModelID,
		baseEnvironmentID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Custom Model version from files", err.Error())
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
	state.FolderPath = plan.FolderPath
	state.Files = plan.Files
	state.TargetName = types.StringValue(customModel.TargetName)
	state.PositiveClassLabel = types.StringValue(customModel.PositiveClassLabel)
	state.NegativeClassLabel = types.StringValue(customModel.NegativeClassLabel)
	state.PredictionThreshold = types.Float64Value(customModel.PredictionThreshold)
	state.IsProxy = types.BoolValue(customModel.IsProxyModel)
	state.DeploymentsCount = types.Int64Value(customModel.DeploymentsCount)

	if IsKnown(plan.RuntimeParameterValues) {
		runtimeParameterValues := make([]RuntimeParameterValue, 0)
		if diags := plan.RuntimeParameterValues.ElementsAs(ctx, &runtimeParameterValues, false); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		params := make([]client.RuntimeParameterValueRequest, len(runtimeParameterValues))
		for i, param := range runtimeParameterValues {
			value, err := formatRuntimeParameterValue(param.Type.ValueString(), param.Value.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error formatting runtime parameter value", err.Error())
				return
			}
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
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:          "false",
			BaseEnvironmentID:      baseEnvironmentID,
			RuntimeParameterValues: string(jsonParams),
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

	err = r.assignTrainingDataset(
		ctx,
		customModelID,
		baseEnvironmentID,
		plan.TrainingDatasetID,
		plan.TrainingDataPartitionColumn,
		&state)
	if err != nil {
		resp.Diagnostics.AddError("Error assigning training dataset to Custom Model", err.Error())
		return
	}

	if plan.ResourceSettings != nil {
		resourceSettings := *plan.ResourceSettings
		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:       "false",
			BaseEnvironmentID:   baseEnvironmentID,
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

	if len(customModel.LatestVersion.Dependencies) > 0 {
		traceAPICall("CreateDependencyBuild")
		_, err := r.provider.service.CreateDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model dependency build", err.Error())
			return
		}

		err = r.waitForDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for Custom Model dependency build", err.Error())
			return
		}

	}

	state.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		customModel.LatestVersion.RuntimeParameters,
		plan.RuntimeParameterValues)
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

	data.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		customModel.LatestVersion.RuntimeParameters,
		data.RuntimeParameterValues)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	loadCustomModelToTerraformState(
		*customModel,
		data.SourceLLMBlueprintID.ValueString(),
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

	classLabels, err := getClassLabels(plan)
	if err != nil {
		resp.Diagnostics.AddError("Error getting class labels from file", err.Error())
	}

	updateRequest := &client.UpdateCustomModelRequest{
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		PredictionThreshold: plan.PredictionThreshold.ValueFloat64(),
		Language:            plan.Language.ValueString(),
	}

	if state.DeploymentsCount.ValueInt64() < 1 {
		updateRequest.TargetName = plan.TargetName.ValueString()
		updateRequest.PositiveClassLabel = plan.PositiveClassLabel.ValueString()
		updateRequest.NegativeClassLabel = plan.NegativeClassLabel.ValueString()
		updateRequest.ClassLabels = classLabels
	}

	traceAPICall("UpdateCustomModel")
	customModel, err := r.provider.service.UpdateCustomModel(ctx, id, updateRequest)
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
	if customModel.Description != "" {
		state.Description = types.StringValue(customModel.Description)
	}
	state.TargetName = types.StringValue(customModel.TargetName)
	state.PositiveClassLabel = types.StringValue(customModel.PositiveClassLabel)
	state.NegativeClassLabel = types.StringValue(customModel.NegativeClassLabel)
	state.PredictionThreshold = types.Float64Value(customModel.PredictionThreshold)
	if customModel.Language != "" {
		state.Language = types.StringValue(customModel.Language)
	}
	state.ClassLabels = plan.ClassLabels
	state.ClassLabelsFile = plan.ClassLabelsFile
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if customModel.LatestVersion.IsFrozen {
		_, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, id, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:     "true",
			BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}
		customModel, err = r.provider.service.GetCustomModel(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error getting Custom Model", err.Error())
			return
		}
	}

	planRuntimeParametersValues, _ := listValueFromRuntimParameters(ctx, []RuntimeParameterValue{})

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

		params := make([]client.RuntimeParameterValueRequest, 0)
		for _, param := range runtimeParameterValues {
			value, err := formatRuntimeParameterValue(param.Type.ValueString(), param.Value.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error formatting runtime parameter value", err.Error())
				return
			}
			params = append(params, client.RuntimeParameterValueRequest{
				FieldName: param.Key.ValueString(),
				Type:      param.Type.ValueString(),
				Value:     &value,
			})
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
		createVersionFromLatestResp, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, id, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:            "false",
			BaseEnvironmentID:        baseEnvironmentID,
			BaseEnvironmentVersionID: baseEnvironmentVersionID,
			RuntimeParameterValues:   string(jsonParams),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}
		latestVersion := createVersionFromLatestResp.ID

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

	if !reflect.DeepEqual(plan.Files, state.Files) || plan.FolderPath != state.FolderPath {
		err = r.updateLocalFiles(ctx, customModel, plan)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Custom Model from files", err.Error())
			return
		}

		state.Files = plan.Files
		state.FolderPath = plan.FolderPath
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
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
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

	if plan.TrainingDatasetID != state.TrainingDatasetID ||
		plan.TrainingDataPartitionColumn != state.TrainingDataPartitionColumn {
		keepTrainingHoldoutData := false
		traceAPICall("CreateCustomModelVersionFromLatest")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:           "true",
			BaseEnvironmentID:       customModel.LatestVersion.BaseEnvironmentID,
			KeepTrainingHoldoutData: &keepTrainingHoldoutData,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}

		err = r.assignTrainingDataset(
			ctx,
			customModel.ID,
			customModel.LatestVersion.BaseEnvironmentID,
			plan.TrainingDatasetID,
			plan.TrainingDataPartitionColumn,
			&state)
		if err != nil {
			resp.Diagnostics.AddError("Error assigning training dataset to Custom Model", err.Error())
			return
		}
	}

	traceAPICall("WaitForCustomModelToBeReady")
	customModel, err = r.waitForCustomModelToBeReady(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
		return
	}
	state.VersionID = types.StringValue(customModel.LatestVersion.ID)

	if len(customModel.LatestVersion.Dependencies) > 0 {
		traceAPICall("GetDependencyBuild")
		_, err := r.provider.service.GetDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID)
		if err != nil { // if not found, must create a new one
			traceAPICall("CreateDependencyBuild")
			_, err := r.provider.service.CreateDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID)
			if err != nil {
				resp.Diagnostics.AddError("Error creating Custom Model dependency build", err.Error())
				return
			}

			err = r.waitForDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID)
			if err != nil {
				resp.Diagnostics.AddError("Error waiting for Custom Model dependency build", err.Error())
				return
			}
		}
	}

	state.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		customModel.LatestVersion.RuntimeParameters,
		plan.RuntimeParameterValues)
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

	var state CustomModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !IsKnown(plan.RuntimeParameterValues) {
		// use empty list if runtime parameter values are unknown
		plan.RuntimeParameterValues, _ = listValueFromRuntimParameters(ctx, []RuntimeParameterValue{})
	}

	if plan.TrainingDatasetID == state.TrainingDatasetID &&
		plan.TrainingDataPartitionColumn == state.TrainingDataPartitionColumn {
		// traning dataset is not changed
		plan.TrainingDatasetVersionID = state.TrainingDatasetVersionID
		plan.TrainingDatasetName = state.TrainingDatasetName
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r CustomModelResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("class_labels"),
			path.MatchRoot("class_labels_file"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("positive_class_label"),
			path.MatchRoot("negative_class_label"),
		),
	}
}

func loadCustomModelToTerraformState(
	customModel client.CustomModel,
	sourceBlueprintId string,
	state *CustomModelResourceModel,
) {
	state.ID = types.StringValue(customModel.ID)
	state.Name = types.StringValue(customModel.Name)
	if customModel.Description != "" {
		state.Description = types.StringValue(customModel.Description)
	}
	if sourceBlueprintId != "" {
		state.SourceLLMBlueprintID = types.StringValue(sourceBlueprintId)
	}
	state.TargetName = types.StringValue(customModel.TargetName)
	state.PositiveClassLabel = types.StringValue(customModel.PositiveClassLabel)
	state.NegativeClassLabel = types.StringValue(customModel.NegativeClassLabel)
	state.PredictionThreshold = types.Float64Value(customModel.PredictionThreshold)
	if customModel.Language != "" {
		state.Language = types.StringValue(customModel.Language)
	}
	state.IsProxy = types.BoolValue(customModel.IsProxyModel)
	state.DeploymentsCount = types.Int64Value(customModel.DeploymentsCount)
	state.VersionID = types.StringValue(customModel.LatestVersion.ID)
	state.BaseEnvironmentID = types.StringValue(customModel.LatestVersion.BaseEnvironmentID)
	state.BaseEnvironmentVersionID = types.StringValue(customModel.LatestVersion.BaseEnvironmentVersionID)

	if len(customModel.ClassLabels) > 0 {
		classLabels := make([]types.String, len(customModel.ClassLabels))
		for i, classLabel := range customModel.ClassLabels {
			classLabels[i] = types.StringValue(classLabel)
		}
		state.ClassLabels = classLabels
	}

	if customModel.LatestVersion.TrainingData != nil {
		state.TrainingDatasetID = types.StringValue(customModel.LatestVersion.TrainingData.DatasetID)
		state.TrainingDatasetVersionID = types.StringValue(customModel.LatestVersion.TrainingData.DatasetVersionID)
		state.TrainingDatasetName = types.StringValue(customModel.LatestVersion.TrainingData.DatasetName)
	}

	if customModel.LatestVersion.HoldoutData != nil {
		state.TrainingDataPartitionColumn = types.StringPointerValue(customModel.LatestVersion.HoldoutData.PartitionColumn)
	}

	loadResourceSettingsToTerraformState(customModel.LatestVersion, state)
}

func loadResourceSettingsToTerraformState(
	customModelVersion client.CustomModelVersion,
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

func (r *CustomModelResource) waitForCustomModelToBeReady(ctx context.Context, customModelId string) (*client.CustomModel, error) {
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

func (r *CustomModelResource) waitForTrainingDataToBeAssigned(ctx context.Context, customModelId string) error {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("GetCustomModel")
		customModel, err := r.provider.service.GetCustomModel(ctx, customModelId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if customModel.LatestVersion.TrainingData.AssignmentError != nil {
			return backoff.Permanent(errors.New(customModel.LatestVersion.TrainingData.AssignmentError.Message))
		}
		if customModel.LatestVersion.TrainingData.AssignmentInProgress {
			return errors.New("assignment in progress")
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

func (r *CustomModelResource) waitForDependencyBuild(ctx context.Context, id, versionID string) error {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("GetDependencyBuild")
		dependencyBuild, err := r.provider.service.GetDependencyBuild(ctx, id, versionID)
		if err != nil {
			return backoff.Permanent(err)
		}
		if dependencyBuild.BuildStatus == "failed" {
			return backoff.Permanent(errors.New("dependency build failed"))
		}
		if dependencyBuild.BuildStatus != "success" {
			return errors.New("dependency build in progress")
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
	folderPath types.String,
	files types.Dynamic,
	customModelID string,
	baseEnvironmentID string,
) (
	err error,
) {
	localFiles, err := prepareLocalFiles(folderPath, files)
	if err != nil {
		return
	}

	traceAPICall("CreateCustomModelVersionFromLocalFiles")
	_, err = r.provider.service.CreateCustomModelVersionFromFiles(ctx, customModelID, &client.CreateCustomModelVersionFromFilesRequest{
		BaseEnvironmentID: baseEnvironmentID,
		Files:             localFiles,
	})
	if err != nil {
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
			Name:               existingGuardConfig.Name,
			Description:        existingGuardConfig.Description,
			Type:               existingGuardConfig.Type,
			Stages:             existingGuardConfig.Stages,
			Intervention:       intervention,
			DeploymentID:       existingGuardConfig.DeploymentID,
			NemoInfo:           existingGuardConfig.NemoInfo,
			ModelInfo:          existingGuardConfig.ModelInfo,
			OpenAICredential:   existingGuardConfig.OpenAICredential,
			OpenAIApiBase:      existingGuardConfig.OpenAIApiBase,
			OpenAIDeploymentID: existingGuardConfig.OpenAIDeploymentID,
			LlmType:            existingGuardConfig.LlmType,
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
			ModelInfo:    guardTemplate.ModelInfo,
			DeploymentID: guardConfigToAdd.DeploymentID.ValueString(),

			// TODO: allow user to input Nemo Info
			NemoInfo: guardTemplate.NemoInfo,

			// Faithfulness Guard specific fields
			OpenAICredential:   guardConfigToAdd.OpenAICredential.ValueString(),
			OpenAIApiBase:      guardConfigToAdd.OpenAIApiBase.ValueString(),
			OpenAIDeploymentID: guardConfigToAdd.OpenAIDeploymentID.ValueString(),
			LlmType:            guardConfigToAdd.LlmType.ValueString(),
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
	customModel *client.CustomModel,
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
		_, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
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
	customModel *client.CustomModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	filesToDelete := make([]string, 0)
	for _, item := range customModel.LatestVersion.Items {
		if item.FileSource == "local" {
			filesToDelete = append(filesToDelete, item.ID)
		}
	}

	if len(filesToDelete) > 0 {
		traceAPICall("CreateCustomModelVersionCreateFromLatestDeleteFiles")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:     "false",
			BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
			FilesToDelete:     filesToDelete,
		})
		if err != nil {
			return
		}
	}

	return r.createCustomModelVersionFromFiles(
		ctx,
		plan.FolderPath,
		plan.Files,
		customModel.ID,
		customModel.LatestVersion.BaseEnvironmentID,
	)
}

func (r *CustomModelResource) assignTrainingDataset(
	ctx context.Context,
	customModelID string,
	baseEnvironmentID string,
	trainingDatasetID types.String,
	trainingDataPartitionColumn types.String,
	state *CustomModelResourceModel,
) (
	err error,
) {
	if IsKnown(trainingDatasetID) {
		trainingData := client.CustomModelTrainingData{
			DatasetID: trainingDatasetID.ValueString(),
		}

		var jsonTrainingData []byte
		jsonTrainingData, err = json.Marshal(trainingData)
		if err != nil {
			return
		}

		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		keepTrainingHoldoutData := false
		createVersionFromLatestRequest := &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:           "false",
			BaseEnvironmentID:       baseEnvironmentID,
			KeepTrainingHoldoutData: &keepTrainingHoldoutData,
			TrainingData:            string(jsonTrainingData),
		}

		if IsKnown(trainingDataPartitionColumn) {
			partitionColumn := trainingDataPartitionColumn.ValueString()

			var jsonHoldoutData []byte
			jsonHoldoutData, err = json.Marshal(client.CustomModelHoldoutData{
				PartitionColumn: &partitionColumn,
			})
			if err != nil {
				return
			}
			createVersionFromLatestRequest.HoldoutData = string(jsonHoldoutData)
		}

		var customModelVersion *client.CustomModelVersion
		customModelVersion, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, createVersionFromLatestRequest)
		if err != nil {
			return
		}
		state.TrainingDatasetVersionID = types.StringValue(customModelVersion.TrainingData.DatasetVersionID)
		state.TrainingDatasetName = types.StringValue(customModelVersion.TrainingData.DatasetName)

		err = r.waitForTrainingDataToBeAssigned(ctx, customModelID)
		if err != nil {
			return
		}
	} else {
		state.TrainingDatasetVersionID = types.StringNull()
		state.TrainingDatasetName = types.StringNull()
	}

	state.TrainingDatasetID = trainingDatasetID
	state.TrainingDataPartitionColumn = trainingDataPartitionColumn

	return
}

func getClassLabels(plan CustomModelResourceModel) (classLabels []string, err error) {
	if IsKnown(plan.ClassLabelsFile) {
		var classLabelFile *os.File
		classLabelFile, err = os.Open(plan.ClassLabelsFile.ValueString())
		if err != nil {
			return
		}
		defer classLabelFile.Close()

		scanner := bufio.NewScanner(classLabelFile)
		for scanner.Scan() {
			classLabels = append(classLabels, scanner.Text())
		}

		if err = scanner.Err(); err != nil {
			return
		}
	} else {
		for _, classLabel := range plan.ClassLabels {
			classLabels = append(classLabels, classLabel.ValueString())
		}
	}

	return
}

func stringRequiresReplaceIfDeployed() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			if req.PlanValue.IsUnknown() {
				resp.RequiresReplace = false
				return
			}

			var state CustomModelResourceModel

			diags := req.State.Get(ctx, &state)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if state.DeploymentsCount.ValueInt64() > 0 {
				resp.RequiresReplace = true
				return
			}
		},
		"Requires replace if the model was deployed.",
		"Requires replace if the model was deployed.",
	)
}

func listRequiresReplaceIfDeployed() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
			if req.PlanValue.IsUnknown() {
				resp.RequiresReplace = false
				return
			}

			var state CustomModelResourceModel

			diags := req.State.Get(ctx, &state)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if state.DeploymentsCount.ValueInt64() > 0 {
				resp.RequiresReplace = true
				return
			}
		},
		"Requires replace if the model was deployed.",
		"Requires replace if the model was deployed.",
	)
}
