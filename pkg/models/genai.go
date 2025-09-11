package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// VectorDatabaseResourceModel describes a vector database.
type VectorDatabaseResourceModel struct {
	ID                 types.String             `tfsdk:"id"`
	Version            types.Int64              `tfsdk:"version"`
	Name               types.String             `tfsdk:"name"`
	UseCaseID          types.String             `tfsdk:"use_case_id"`
	DatasetID          types.String             `tfsdk:"dataset_id"`
	ChunkingParameters *ChunkingParametersModel `tfsdk:"chunking_parameters"`
}

// ChunkingParametersModel represents the chunking parameters nested attribute.
type ChunkingParametersModel struct {
	EmbeddingModel         types.String   `tfsdk:"embedding_model"`
	ChunkOverlapPercentage types.Int64    `tfsdk:"chunk_overlap_percentage"`
	ChunkSize              types.Int64    `tfsdk:"chunk_size"`
	ChunkingMethod         types.String   `tfsdk:"chunking_method"`
	IsSeparatorRegex       types.Bool     `tfsdk:"is_separator_regex"`
	Separators             []types.String `tfsdk:"separators"`
}

// PlaygroundResourceModel describes the playground associated to a use case.
type PlaygroundResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	UseCaseID      types.String `tfsdk:"use_case_id"`
	PlaygroundType types.String `tfsdk:"playground_type"`
}

// LLMBlueprintResourceModel describes the LLM blueprint resource.
type LLMBlueprintResourceModel struct {
	ID                     types.String            `tfsdk:"id"`
	Name                   types.String            `tfsdk:"name"`
	Description            types.String            `tfsdk:"description"`
	PlaygroundID           types.String            `tfsdk:"playground_id"`
	VectorDatabaseID       types.String            `tfsdk:"vector_database_id"`
	VectorDatabaseSettings *VectorDatabaseSettings `tfsdk:"vector_database_settings"`
	LLMID                  types.String            `tfsdk:"llm_id"`
	LLMSettings            *LLMSettings            `tfsdk:"llm_settings"`
	PromptType             types.String            `tfsdk:"prompt_type"`
	CustomModelLLMSettings *CustomModelLLMSettings `tfsdk:"custom_model_llm_settings"`
}

// CustomModelLLMValidationResourceModel describes the custom model LLM validation resource.
type CustomModelLLMValidationResourceModel struct {
	ID                types.String `tfsdk:"id"`
	DeploymentID      types.String `tfsdk:"deployment_id"`
	ModelID           types.String `tfsdk:"model_id"`
	Name              types.String `tfsdk:"name"`
	PredictionTimeout types.Int64  `tfsdk:"prediction_timeout"`
	PromptColumnName  types.String `tfsdk:"prompt_column_name"`
	TargetColumnName  types.String `tfsdk:"target_column_name"`
	ChatModelID       types.String `tfsdk:"chat_model_id"`
	UseCaseID         types.String `tfsdk:"use_case_id"`
}

type VectorDatabaseSettings struct {
	MaxDocumentsRetrievedPerPrompt types.Int64 `tfsdk:"max_documents_retrieved_per_prompt"`
	MaxTokens                      types.Int64 `tfsdk:"max_tokens"`
}

type LLMSettings struct {
	MaxCompletionLength types.Int64   `tfsdk:"max_completion_length"`
	Temperature         types.Float64 `tfsdk:"temperature"`
	TopP                types.Float64 `tfsdk:"top_p"`
	SystemPrompt        types.String  `tfsdk:"system_prompt"`
	CustomModelID       types.String  `tfsdk:"custom_model_id"`
}

type CustomModelLLMSettings struct {
	ExternalLLMContextSize types.Int64  `tfsdk:"external_llm_context_size"`
	SystemPrompt           types.String `tfsdk:"system_prompt"`
	ValidationID           types.String `tfsdk:"validation_id"`
}


// ModelResourceModel describes the custom model resource.
type CustomModelResourceModel struct {
	ID                             types.String                    `tfsdk:"id"`
	VersionID                      types.String                    `tfsdk:"version_id"`
	Name                           types.String                    `tfsdk:"name"`
	Description                    types.String                    `tfsdk:"description"`
	SourceLLMBlueprintID           types.String                    `tfsdk:"source_llm_blueprint_id"`
	BaseEnvironmentID              types.String                    `tfsdk:"base_environment_id"`
	BaseEnvironmentVersionID       types.String                    `tfsdk:"base_environment_version_id"`
	RuntimeParameterValues         types.List                      `tfsdk:"runtime_parameter_values"`
	SourceRemoteRepositories       []SourceRemoteRepository        `tfsdk:"source_remote_repositories"`
	FolderPath                     types.String                    `tfsdk:"folder_path"`
	FolderPathHash                 types.String                    `tfsdk:"folder_path_hash"`
	Files                          types.Dynamic                   `tfsdk:"files"`
	FilesHashes                    types.List                      `tfsdk:"files_hashes"`
	TargetName                     types.String                    `tfsdk:"target_name"`
	TargetType                     types.String                    `tfsdk:"target_type"`
	PositiveClassLabel             types.String                    `tfsdk:"positive_class_label"`
	NegativeClassLabel             types.String                    `tfsdk:"negative_class_label"`
	PredictionThreshold            types.Float64                   `tfsdk:"prediction_threshold"`
	Language                       types.String                    `tfsdk:"language"`
	IsProxy                        types.Bool                      `tfsdk:"is_proxy"`
	ClassLabels                    []types.String                  `tfsdk:"class_labels"`
	ClassLabelsFile                types.String                    `tfsdk:"class_labels_file"`
	DeploymentsCount               types.Int64                     `tfsdk:"deployments_count"`
	GuardConfigurations            []GuardConfiguration            `tfsdk:"guard_configurations"`
	OverallModerationConfiguration *OverallModerationConfiguration `tfsdk:"overall_moderation_configuration"`
	TrainingDatasetID              types.String                    `tfsdk:"training_dataset_id"`
	TrainingDatasetVersionID       types.String                    `tfsdk:"training_dataset_version_id"`
	TrainingDatasetName            types.String                    `tfsdk:"training_dataset_name"`
	TrainingDataPartitionColumn    types.String                    `tfsdk:"training_data_partition_column"`
	MemoryMB                       types.Int64                     `tfsdk:"memory_mb"`
	Replicas                       types.Int64                     `tfsdk:"replicas"`
	NetworkAccess                  types.String                    `tfsdk:"network_access"`
	ResourceBundleID               types.String                    `tfsdk:"resource_bundle_id"`
	UseCaseIDs                     []types.String                  `tfsdk:"use_case_ids"`
}

type SourceRemoteRepository struct {
	ID          types.String   `tfsdk:"id"`
	Ref         types.String   `tfsdk:"ref"`
	SourcePaths []types.String `tfsdk:"source_paths"`
}


type GuardConfiguration struct {
	TemplateName          types.String           `tfsdk:"template_name"`
	Name                  types.String           `tfsdk:"name"`
	Stages                []types.String         `tfsdk:"stages"`
	Intervention          GuardIntervention      `tfsdk:"intervention"`
	DeploymentID          types.String           `tfsdk:"deployment_id"`
	InputColumnName       types.String           `tfsdk:"input_column_name"`
	OutputColumnName      types.String           `tfsdk:"output_column_name"`
	OpenAICredential      types.String           `tfsdk:"openai_credential"`
	OpenAIApiBase         types.String           `tfsdk:"openai_api_base"`
	OpenAIDeploymentID    types.String           `tfsdk:"openai_deployment_id"`
	LlmType               types.String           `tfsdk:"llm_type"`
	NemoInfo              *NemoInfo              `tfsdk:"nemo_info"`
	AdditionalGuardConfig *AdditionalGuardConfig `tfsdk:"additional_guard_config"`
}


type GuardIntervention struct {
	Action    types.String `tfsdk:"action"`
	Condition types.String `tfsdk:"condition"`
	Message   types.String `tfsdk:"message"`
}

type GuardCondition struct {
	Comparand  types.Float64 `tfsdk:"comparand"`
	Comparator types.String  `tfsdk:"comparator"`
}

type NemoInfo struct {
	Actions      types.String `tfsdk:"actions"`
	BlockedTerms types.String `tfsdk:"blocked_terms"`
	LlmPrompts   types.String `tfsdk:"llm_prompts"`
	MainConfig   types.String `tfsdk:"main_config"`
	RailsConfig  types.String `tfsdk:"rails_config"`
}

type OverallModerationConfiguration struct {
	TimeoutSec    types.Int64  `tfsdk:"timeout_sec"`
	TimeoutAction types.String `tfsdk:"timeout_action"`
}

type AdditionalGuardConfig struct {
	Cost GuardCostInfo `tfsdk:"cost"`
}

type GuardCostInfo struct {
	Currency    types.String  `tfsdk:"currency"`
	InputPrice  types.Float64 `tfsdk:"input_price"`
	InputUnit   types.Int64   `tfsdk:"input_unit"`
	OutputPrice types.Float64 `tfsdk:"output_price"`
	OutputUnit  types.Int64   `tfsdk:"output_unit"`
}


// CustomJobResourceModel describes the custom job resource.
type CustomJobResourceModel struct {
	ID                     types.String  `tfsdk:"id"`
	Name                   types.String  `tfsdk:"name"`
	Description            types.String  `tfsdk:"description"`
	JobType                types.String  `tfsdk:"job_type"`
	EnvironmentID          types.String  `tfsdk:"environment_id"`
	EnvironmentVersionID   types.String  `tfsdk:"environment_version_id"`
	RuntimeParameterValues types.List    `tfsdk:"runtime_parameter_values"`
	FolderPath             types.String  `tfsdk:"folder_path"`
	FolderPathHash         types.String  `tfsdk:"folder_path_hash"`
	Files                  types.Dynamic `tfsdk:"files"`
	FilesHashes            types.List    `tfsdk:"files_hashes"`
	EgressNetworkPolicy    types.String  `tfsdk:"egress_network_policy"`
	ResourceBundleID       types.String  `tfsdk:"resource_bundle_id"`
	Schedule               *Schedule     `tfsdk:"schedule"`
	ScheduleID             types.String  `tfsdk:"schedule_id"`
}
