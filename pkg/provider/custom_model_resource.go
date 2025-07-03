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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	defaultReplicas                = 1
	defaultNetworkAccess           = "PUBLIC"
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
		MarkdownDescription: "Custom Model",

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
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_llm_blueprint_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the source LLM Blueprint for the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"base_environment_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the base environment for the Custom Model.",
			},
			"base_environment_version_id": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "The ID of the base environment version for the Custom Model.",
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
							Validators:          RuntimeParameterTypeValidators(),
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
				Computed:            true,
				MarkdownDescription: "The target type of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: CustomModelTargetTypeValidators(),
			},
			"target_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The target name of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"positive_class_label": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The positive class label of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"negative_class_label": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The negative class label of the Custom Model.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			},
			"class_labels_file": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to file containing newline separated class labels for multiclass classification. Cannot be used with class_labels.",
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
			"folder_path_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of the folder path contents.",
			},
			"files": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: "The list of tuples, where values in each tuple are the local filesystem path and the path the file should be placed in the Custom Model. If list is of strings, then basenames will be used for tuples.",
			},
			"files_hashes": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "The hash of file contents for each file in files.",
				ElementType:         types.StringType,
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
							Validators:          GuardStagesValidators(),
						},
						"intervention": schema.SingleNestedAttribute{
							Required:            true,
							MarkdownDescription: "The intervention for the guard configuration.",
							Attributes: map[string]schema.Attribute{
								"action": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "The action of the guard intervention.",
									Validators:          GuardInterventionActionValidators(),
								},
								"message": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString("This message has triggered moderation criteria and therefore been blocked by the DataRobot moderation system."),
									MarkdownDescription: "The message of the guard intervention.",
								},
								"condition": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "The JSON-encoded condition of the guard intervention. e.g. `{\"comparand\": 0.5, \"comparator\": \"lessThan\"}`",
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
							Validators:          CustomModelLLMTypeValidators(),
						},
						"nemo_info": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Configuration info for NeMo guards.",
							Attributes: map[string]schema.Attribute{
								"actions": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "The actions for the NeMo information.",
								},
								"blocked_terms": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "NeMo guardrails blocked terms list.",
								},
								"llm_prompts": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "NeMo guardrails prompts.",
								},
								"main_config": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "Overall NeMo configuration YAML.",
								},
								"rails_config": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "NeMo guardrails configuration Colang.",
								},
							},
						},
						"additional_guard_config": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Additional guard configuration",
							Attributes: map[string]schema.Attribute{
								"cost": schema.SingleNestedAttribute{
									Optional:            true,
									MarkdownDescription: "Cost metric configuration",
									Attributes: map[string]schema.Attribute{
										"currency": schema.StringAttribute{
											Optional:            false,
											MarkdownDescription: "Currency for cost calculation (USD)",
										},
										"input_price": schema.Float64Attribute{
											Optional:            false,
											MarkdownDescription: "LLM Price for input_unit tokens",
										},
										"input_unit": schema.Int64Attribute{
											Optional:            false,
											MarkdownDescription: "No of input tokens for given price",
										},
										"output_price": schema.Float64Attribute{
											Optional:            false,
											MarkdownDescription: "LLM Price for output_unit tokens",
										},
										"output_unit": schema.Int64Attribute{
											Optional:            false,
											MarkdownDescription: "No of output tokens for given price",
										},
									},
								},
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
			"memory_mb": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The memory in MB for the Custom Model.",
				Default:             nil,
			},
			"replicas": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The replicas for the Custom Model.",
				Default:             int64default.StaticInt64(defaultReplicas),
			},
			"network_access": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The network access for the Custom Model.",
				Default:             stringdefault.StaticString(defaultNetworkAccess),
				Validators:          CustomModelNetworkEgressPolicyValidators(),
			},
			"resource_bundle_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A single identifier that represents a bundle of resources: Memory, CPU, GPU, etc.",
			},
			"use_case_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "The list of Use Case IDs to add the Custom Model version to.",
				ElementType:         types.StringType,
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
	var memoryMB int64

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CustomModelResourceModel
	var customModelID string
	var baseEnvironmentID string
	var baseEnvironmentVersionID string
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
		baseEnvironmentVersionID = customModel.LatestVersion.BaseEnvironmentVersionID

		state.Name = types.StringValue(name)
	} else {
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
		state.TargetType = types.StringValue(plan.TargetType.ValueString())
		state.ClassLabels = plan.ClassLabels
		state.ClassLabelsFile = plan.ClassLabelsFile
		state.Language = plan.Language
	}

	if IsKnown(plan.BaseEnvironmentID) || IsKnown(plan.BaseEnvironmentVersionID) {
		if IsKnown(plan.BaseEnvironmentID) {
			baseEnvironmentID = plan.BaseEnvironmentID.ValueString()
		}
		if IsKnown(plan.BaseEnvironmentVersionID) {
			baseEnvironmentVersionID = plan.BaseEnvironmentVersionID.ValueString()
		}

		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		customModelVersion, err := r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:            "false",
			BaseEnvironmentID:        baseEnvironmentID,
			BaseEnvironmentVersionID: baseEnvironmentVersionID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
			return
		}
		baseEnvironmentID = customModelVersion.BaseEnvironmentID
	}

	if plan.SourceRemoteRepositories != nil {
		for _, sourceRemoteRepository := range plan.SourceRemoteRepositories {
			if err := r.createCustomModelVersionFromRemoteRepository(
				ctx,
				sourceRemoteRepository,
				customModelID,
				baseEnvironmentID,
			); err != nil {
				resp.Diagnostics.AddError("Error creating Custom Model version from remote repository", err.Error())
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
	state.Description = types.StringValue(customModel.Description)
	state.SourceRemoteRepositories = plan.SourceRemoteRepositories
	state.FolderPath = plan.FolderPath
	state.FolderPathHash = plan.FolderPathHash
	state.Files = plan.Files
	state.FilesHashes = plan.FilesHashes
	state.TargetType = types.StringValue(customModel.TargetType)
	state.TargetName = types.StringValue(customModel.TargetName)
	state.PositiveClassLabel = types.StringValue(customModel.PositiveClassLabel)
	state.NegativeClassLabel = types.StringValue(customModel.NegativeClassLabel)
	state.PredictionThreshold = types.Float64Value(customModel.PredictionThreshold)
	state.IsProxy = types.BoolValue(customModel.IsProxyModel)
	state.DeploymentsCount = types.Int64Value(customModel.DeploymentsCount)

	if IsKnown(plan.RuntimeParameterValues) {
		runtimeParameterValues, err := convertRuntimeParameterValues(ctx, plan.RuntimeParameterValues)
		if err != nil {
			resp.Diagnostics.AddError("Error reading runtime parameter values", err.Error())
			return
		}

		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:          "false",
			BaseEnvironmentID:      baseEnvironmentID,
			RuntimeParameterValues: runtimeParameterValues,
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
		if err = r.createCustomModelVersionFromGuards(
			ctx,
			plan,
			customModelID,
			customModel.LatestVersion.ID,
			plan.GuardConfigurations,
			[]GuardConfiguration{},
		); err != nil {
			resp.Diagnostics.AddError("Error creating Custom Model version from Guards", err.Error())
			return
		}
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

	payload := &client.CreateCustomModelVersionFromLatestRequest{
		IsMajorUpdate:       "false",
		BaseEnvironmentID:   baseEnvironmentID,
		Replicas:            plan.Replicas.ValueInt64(),
		NetworkEgressPolicy: plan.NetworkAccess.ValueString(),
	}
	if IsKnown(plan.MemoryMB) {
		memoryMB = plan.MemoryMB.ValueInt64()
		state.MemoryMB = types.Int64Value(memoryMB)
		payload.MaximumMemory = memoryMB * 1024 * 1024
	}
	traceAPICall("CreateCustomModelVersionCreateFromLatest")
	if _, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModelID, payload); err != nil {
		resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
		return
	}
	state.Replicas = plan.Replicas
	state.NetworkAccess = plan.NetworkAccess

	if err = r.addResourceBundle(ctx, customModel, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error adding resource bundle", err.Error())
		return
	}

	customModel, err = r.waitForCustomModelToBeReady(ctx, customModelID)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for Custom Model to be ready", err.Error())
		return
	}
	state.VersionID = types.StringValue(customModel.LatestVersion.ID)

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

	for _, useCaseID := range plan.UseCaseIDs {
		traceAPICall("AddCustomModelVersionToUseCase")
		if err = addEntityToUseCase(
			ctx,
			r.provider.service,
			useCaseID.ValueString(),
			"customModelVersion",
			customModel.LatestVersion.ID,
		); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Error adding Custom Model version to Use Case %s", useCaseID), err.Error())
			return
		}
	}
	state.UseCaseIDs = plan.UseCaseIDs

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

	id := plan.ID.ValueString()

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

	if err := r.updateCustomModel(ctx, customModel, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error updating Custom Model", err.Error())
		return
	}

	if err = r.createNewCustomModelVersion(ctx, plan, customModel); err != nil {
		resp.Diagnostics.AddError("Error creating Custom Model version", err.Error())
		return
	}

	if customModel, err = r.provider.service.GetCustomModel(ctx, id); err != nil {
		resp.Diagnostics.AddError("Error getting Custom Model", err.Error())
		return
	}
	state.BaseEnvironmentID = types.StringValue(customModel.LatestVersion.BaseEnvironmentID)
	state.BaseEnvironmentVersionID = types.StringValue(customModel.LatestVersion.BaseEnvironmentVersionID)

	if err = r.updateRemoteRepositories(ctx, customModel, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error updating remote repositories", err.Error())
		return
	}

	if err = r.updateLocalFiles(ctx, customModel, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error updating Custom Model from files", err.Error())
		return
	}

	if err = r.updateGuardConfigurations(ctx, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error updating guard configurations", err.Error())
		return
	}

	if err = r.updateResourceSettings(ctx, customModel, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error updating resource settings", err.Error())
		return
	}

	if err = r.addResourceBundle(ctx, customModel, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error adding resource bundle", err.Error())
		return
	}

	if err = r.updateRuntimeParameterValues(ctx, customModel, plan); err != nil {
		resp.Diagnostics.AddError("Error updating runtime parameter values", err.Error())
		return
	}

	if err = r.updateTrainingDataset(ctx, customModel, &state, plan); err != nil {
		resp.Diagnostics.AddError("Error updating training dataset", err.Error())
		return
	}

	traceAPICall("GetCustomModel")
	if customModel, err = r.provider.service.GetCustomModel(ctx, id); err != nil {
		resp.Diagnostics.AddError("Error getting Custom Model", err.Error())
		return
	}
	state.VersionID = types.StringValue(customModel.LatestVersion.ID)

	if err = r.updateDependencyBuild(ctx, customModel); err != nil {
		resp.Diagnostics.AddError("Error updating Custom Model dependency build", err.Error())
		return
	}

	if err = updateUseCasesForEntity(
		ctx,
		r.provider.service,
		"customModelVersion",
		customModel.LatestVersion.ID,
		[]types.String{}, // there are no existing linked use cases because this is a new version
		plan.UseCaseIDs,
	); err != nil {
		resp.Diagnostics.AddError("Error updating Use Cases for Custom Model version", err.Error())
		return
	}
	state.UseCaseIDs = plan.UseCaseIDs

	if state.RuntimeParameterValues, diags = formatRuntimeParameterValues(
		ctx,
		customModel.LatestVersion.RuntimeParameters,
		plan.RuntimeParameterValues); diags.HasError() {
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
	if req.Plan.Raw.IsNull() {
		// Resource is being destroyed
		return
	}

	var plan CustomModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// compute file content hashes
	filesHashes, err := computeFilesHashes(ctx, plan.Files)
	if err != nil {
		resp.Diagnostics.AddError("Error calculating files hashes", err.Error())
		return
	}
	plan.FilesHashes = filesHashes

	folderPathHash, err := computeFolderHash(plan.FolderPath)
	if err != nil {
		resp.Diagnostics.AddError("Error calculating folder path hash", err.Error())
		return
	}
	plan.FolderPathHash = folderPathHash

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

	if req.State.Raw.IsNull() {
		// resource is being created
		return
	}

	var state CustomModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customModel, err := r.provider.service.GetCustomModel(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting custom model", err.Error())
		return
	}

	if customModel.DeploymentsCount > 0 {
		if state.TargetName != plan.TargetName {
			addCannotChangeAttributeError(resp, "target_name")
			return
		}
		if state.PositiveClassLabel != plan.PositiveClassLabel {
			addCannotChangeAttributeError(resp, "positive_class_label")
			return
		}
		if state.NegativeClassLabel != plan.NegativeClassLabel {
			addCannotChangeAttributeError(resp, "negative_class_label")
			return
		}
		if !reflect.DeepEqual(state.ClassLabels, plan.ClassLabels) {
			addCannotChangeAttributeError(resp, "class_labels")
			return
		}
		if state.ClassLabelsFile != plan.ClassLabelsFile {
			addCannotChangeAttributeError(resp, "class_labels_file")
			return
		}
	}

	// reset unknown version id if if hashess have been changed
	if !reflect.DeepEqual(plan.FilesHashes, state.FilesHashes) ||
		plan.FolderPathHash != state.FolderPathHash {
		plan.VersionID = types.StringUnknown()
	}

	if !IsKnown(plan.BaseEnvironmentID) {
		if plan.BaseEnvironmentVersionID == state.BaseEnvironmentVersionID {
			// use state base environment id if base environment version id is not changed
			plan.BaseEnvironmentID = state.BaseEnvironmentID
		}
	}

	if !IsKnown(plan.BaseEnvironmentVersionID) {
		if plan.BaseEnvironmentID == state.BaseEnvironmentID {
			// use state base environment version id if base environment id is not changed
			plan.BaseEnvironmentVersionID = state.BaseEnvironmentVersionID
		}
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

func addCannotChangeAttributeError(
	resp *resource.ModifyPlanResponse,
	attribute string,
) {
	resp.Diagnostics.AddError(
		"Custom Model Update Error",
		fmt.Sprintf("%s cannot be changed if the model was deployed.", attribute))
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
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("source_llm_blueprint_id"),
			path.MatchRoot("target_type"),
		),
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("source_llm_blueprint_id"),
			path.MatchRoot("base_environment_id"),
			path.MatchRoot("base_environment_version_id"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("resource_bundle_id"),
			path.MatchRoot("memory_mb"),
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
	state.TargetType = types.StringValue(customModel.TargetType)
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

	if customModel.LatestVersion.MaximumMemory != nil {
		state.MemoryMB = types.Int64Value(*customModel.LatestVersion.MaximumMemory / (1024 * 1024))
	}

	state.Replicas = types.Int64Value(defaultReplicas)
	if customModel.LatestVersion.Replicas != nil {
		state.Replicas = types.Int64Value(*customModel.LatestVersion.Replicas)
	}

	state.NetworkAccess = types.StringValue(defaultNetworkAccess)
	if customModel.LatestVersion.NetworkEgressPolicy != nil {
		state.NetworkAccess = types.StringValue(*customModel.LatestVersion.NetworkEgressPolicy)
	}
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
	err error,
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
		return
	}

	err = waitForTaskStatusToComplete(ctx, r.provider.service, statusID)
	if err != nil {
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
	err error,
) {
	getGuardConfigsResp, err := r.provider.service.GetGuardConfigurationsForCustomModelVersion(ctx, customModelVersion)
	if err != nil {
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
		return
	}

	for _, guardConfigToAdd := range guardConfigsToAdd {
		var guardTemplate *client.GuardTemplate
		for index := range guardTemplates {
			template := guardTemplates[index]
			if template.Name == guardConfigToAdd.TemplateName.ValueString() {
				guardTemplate = &template
				break
			}
		}

		if guardTemplate == nil {
			return
		}

		stages := make([]string, 0)
		for _, stage := range guardConfigToAdd.Stages {
			stages = append(stages, stage.ValueString())
		}

		var condition client.GuardCondition
		if err = json.Unmarshal([]byte(guardConfigToAdd.Intervention.Condition.ValueString()), &condition); err != nil {
			return
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
				Conditions:     []client.GuardCondition{condition},
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
			var deployment *client.Deployment
			deployment, err = r.provider.service.GetDeployment(ctx, guardConfigToAdd.DeploymentID.ValueString())
			if err != nil {
				return
			}

			newGuardConfig.ModelInfo = client.GuardModelInfo{
				InputColumnName:  guardConfigToAdd.InputColumnName.ValueString(),
				OutputColumnName: guardConfigToAdd.OutputColumnName.ValueString(),
				TargetType:       deployment.Model.TargetType,
			}
		}

		if guardConfigToAdd.NemoInfo != nil {
			setStringValueIfKnown(&newGuardConfig.NemoInfo.Actions, guardConfigToAdd.NemoInfo.Actions)
			setStringValueIfKnown(&newGuardConfig.NemoInfo.BlockedTerms, guardConfigToAdd.NemoInfo.BlockedTerms)
			setStringValueIfKnown(&newGuardConfig.NemoInfo.LlmPrompts, guardConfigToAdd.NemoInfo.LlmPrompts)
			setStringValueIfKnown(&newGuardConfig.NemoInfo.MainConfig, guardConfigToAdd.NemoInfo.MainConfig)
			setStringValueIfKnown(&newGuardConfig.NemoInfo.RailsConfig, guardConfigToAdd.NemoInfo.RailsConfig)
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
	if _, err = r.provider.service.CreateCustomModelVersionFromGuardConfigurations(ctx, customModelVersion, &client.CreateCustomModelVersionFromGuardsConfigurationRequest{
		CustomModelID: customModelID,
		Data:          newGuardConfigs,
		OverallConfig: overallModerationConfig,
	}); err != nil {
		return
	}

	return
}

func (r *CustomModelResource) createNewCustomModelVersion(
	ctx context.Context,
	plan CustomModelResourceModel,
	customModel *client.CustomModel,
) (
	err error,
) {
	// check for major version update
	if customModel.LatestVersion.IsFrozen {
		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		keepTrainingHoldoutData := true
		_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:           "true",
			BaseEnvironmentID:       customModel.LatestVersion.BaseEnvironmentID,
			KeepTrainingHoldoutData: &keepTrainingHoldoutData,
		})
		if err != nil {
			return
		}
	}

	updateRequest := &client.CreateCustomModelVersionFromLatestRequest{
		IsMajorUpdate: "false",
	}
	if IsKnown(plan.BaseEnvironmentID) {
		updateRequest.BaseEnvironmentID = plan.BaseEnvironmentID.ValueString()
	}
	if IsKnown(plan.BaseEnvironmentVersionID) {
		updateRequest.BaseEnvironmentVersionID = plan.BaseEnvironmentVersionID.ValueString()
	}

	traceAPICall("CreateCustomModelVersionCreateFromLatest")
	if _, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, updateRequest); err != nil {
		return
	}

	return
}

func (r *CustomModelResource) updateCustomModel(
	ctx context.Context,
	customModel *client.CustomModel,
	state *CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	var classLabels []string
	if classLabels, err = getClassLabels(plan); err != nil {
		return
	}

	updateRequest := &client.UpdateCustomModelRequest{
		Name:                plan.Name.ValueString(),
		Description:         plan.Description.ValueString(),
		PredictionThreshold: plan.PredictionThreshold.ValueFloat64(),
		Language:            plan.Language.ValueString(),
	}

	if customModel.DeploymentsCount < 1 {
		updateRequest.TargetName = plan.TargetName.ValueString()
		updateRequest.PositiveClassLabel = plan.PositiveClassLabel.ValueString()
		updateRequest.NegativeClassLabel = plan.NegativeClassLabel.ValueString()
		updateRequest.ClassLabels = classLabels
	}

	traceAPICall("UpdateCustomModel")
	if customModel, err = r.provider.service.UpdateCustomModel(ctx, customModel.ID, updateRequest); err != nil {
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

	return
}

func (r *CustomModelResource) updateRuntimeParameterValues(
	ctx context.Context,
	customModel *client.CustomModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	runtimeParameterValues := make([]RuntimeParameterValue, 0)
	if IsKnown(plan.RuntimeParameterValues) {
		if diags := plan.RuntimeParameterValues.ElementsAs(ctx, &runtimeParameterValues, false); diags.HasError() {
			err = fmt.Errorf("Error reading plan runtime parameter values: %s", diags.Errors()[0].Detail())
			return
		}
	}

	params := make([]client.RuntimeParameterValueRequest, 0)
	for _, param := range runtimeParameterValues {
		var value any
		if value, err = formatRuntimeParameterValue(param.Type.ValueString(), param.Value.ValueString()); err != nil {
			return
		}
		params = append(params, client.RuntimeParameterValueRequest{
			FieldName: param.Key.ValueString(),
			Type:      param.Type.ValueString(),
			Value:     &value,
		})
	}

	if len(params) > 0 {
		var jsonParams []byte
		if jsonParams, err = json.Marshal(params); err != nil {
			return
		}

		traceAPICall("CreateCustomModelVersionCreateFromLatestRuntimeParams")
		if _, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:          "false",
			BaseEnvironmentID:      customModel.LatestVersion.BaseEnvironmentID,
			RuntimeParameterValues: string(jsonParams),
		}); err != nil {
			return
		}
	}

	return
}

func (r *CustomModelResource) updateRemoteRepositories(
	ctx context.Context,
	customModel *client.CustomModel,
	state *CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	if !reflect.DeepEqual(plan.SourceRemoteRepositories, state.SourceRemoteRepositories) {
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
					var remoteRepository *client.RemoteRepositoryResponse
					remoteRepository, err = r.provider.service.GetRemoteRepository(ctx, oldSourceRemoteRepository.ID.ValueString())
					if err != nil {
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
			_, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
				IsMajorUpdate:     "false",
				BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
				FilesToDelete:     filesToDelete,
			})
			if err != nil {
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
				if err = r.createCustomModelVersionFromRemoteRepository(
					ctx,
					newSourceRemoteRepository,
					customModel.ID,
					customModel.LatestVersion.BaseEnvironmentID,
				); err != nil {
					return
				}
			}
		}
		state.SourceRemoteRepositories = plan.SourceRemoteRepositories
	}

	return
}

func (r *CustomModelResource) updateLocalFiles(
	ctx context.Context,
	customModel *client.CustomModel,
	state *CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	if !reflect.DeepEqual(plan.Files, state.Files) ||
		!reflect.DeepEqual(plan.FilesHashes, state.FilesHashes) ||
		plan.FolderPath != state.FolderPath ||
		plan.FolderPathHash != state.FolderPathHash {
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

		if err = r.createCustomModelVersionFromFiles(
			ctx,
			plan.FolderPath,
			plan.Files,
			customModel.ID,
			customModel.LatestVersion.BaseEnvironmentID,
		); err != nil {
			return
		}

		state.Files = plan.Files
		state.FolderPath = plan.FolderPath
		state.FolderPathHash = plan.FolderPathHash
		state.FilesHashes = plan.FilesHashes
	}

	return
}

func (r *CustomModelResource) updateGuardConfigurations(
	ctx context.Context,
	state *CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	var customModel *client.CustomModel
	customModel, err = r.provider.service.GetCustomModel(ctx, plan.ID.ValueString())
	if err != nil {
		return
	}

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

	if err = r.createCustomModelVersionFromGuards(
		ctx,
		plan,
		customModel.ID,
		customModel.LatestVersion.ID,
		guardsToAdd,
		guardsToRemove,
	); err != nil {
		return
	}
	state.GuardConfigurations = plan.GuardConfigurations
	state.OverallModerationConfiguration = plan.OverallModerationConfiguration

	return
}

func (r *CustomModelResource) updateResourceSettings(
	ctx context.Context,
	customModel *client.CustomModel,
	state *CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	payload := &client.CreateCustomModelVersionFromLatestRequest{
		IsMajorUpdate:       "false",
		BaseEnvironmentID:   customModel.LatestVersion.BaseEnvironmentID,
		Replicas:            plan.Replicas.ValueInt64(),
		NetworkEgressPolicy: plan.NetworkAccess.ValueString(),
	}
	if IsKnown(plan.MemoryMB) {
		maxMemory := plan.MemoryMB.ValueInt64() * 1024 * 1024
		payload.MaximumMemory = maxMemory
		state.MemoryMB = types.Int64Value(plan.MemoryMB.ValueInt64())
	}

	traceAPICall("CreateCustomModelVersionCreateFromLatestResources")
	if _, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, payload); err != nil {
		return
	}
	state.Replicas = plan.Replicas
	state.NetworkAccess = plan.NetworkAccess

	return
}

func (r *CustomModelResource) updateTrainingDataset(
	ctx context.Context,
	customModel *client.CustomModel,
	state *CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
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
			return
		}

		err = r.assignTrainingDataset(
			ctx,
			customModel.ID,
			customModel.LatestVersion.BaseEnvironmentID,
			plan.TrainingDatasetID,
			plan.TrainingDataPartitionColumn,
			state)
		if err != nil {
			return
		}
	}

	return
}

func (r *CustomModelResource) addResourceBundle(
	ctx context.Context,
	customModel *client.CustomModel,
	state *CustomModelResourceModel,
	plan CustomModelResourceModel,
) (
	err error,
) {
	if IsKnown(plan.ResourceBundleID) {
		traceAPICall("CreateCustomModelVersionCreateFromLatest")
		if _, err = r.provider.service.CreateCustomModelVersionCreateFromLatest(ctx, customModel.ID, &client.CreateCustomModelVersionFromLatestRequest{
			IsMajorUpdate:     "false",
			BaseEnvironmentID: customModel.LatestVersion.BaseEnvironmentID,
			ResourceBundleID:  plan.ResourceBundleID.ValueStringPointer(),
		}); err != nil {
			return
		}
		state.MemoryMB = types.Int64Null() // reset memory if resource bundle is set

	}

	state.ResourceBundleID = plan.ResourceBundleID

	return
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

func (r *CustomModelResource) updateDependencyBuild(
	ctx context.Context,
	customModel *client.CustomModel,
) (
	err error,
) {
	if len(customModel.LatestVersion.Dependencies) > 0 {
		traceAPICall("GetDependencyBuild")
		_, err = r.provider.service.GetDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID)
		if err != nil { // if not found, must create a new one
			traceAPICall("CreateDependencyBuild")
			if _, err = r.provider.service.CreateDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID); err != nil {
				return
			}

			if err = r.waitForDependencyBuild(ctx, customModel.ID, customModel.LatestVersion.ID); err != nil {
				return
			}
		}
	}

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
