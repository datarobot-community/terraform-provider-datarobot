package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	DataRobotApiKeyEnvVar   string = "DATAROBOT_API_KEY"
	DataRobotEndpointEnvVar string = "DATAROBOT_ENDPOINT"
	TimeoutMinutesEnvVar    string = "DATAROBOT_TIMEOUT_MINUTES"
	UserAgent               string = "terraform-provider-datarobot"
)

// UseCaseResourceModel describes the resource data model.
type UseCaseResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// RemoteRepositoryResourceModel describes the remote repository resource.
type RemoteRepositoryResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Location            types.String `tfsdk:"location"`
	SourceType          types.String `tfsdk:"source_type"`
	PersonalAccessToken types.String `tfsdk:"personal_access_token"`
}

// DatasetFromFileResourceModel describes the datasource uploaded from a file.
type DatasetFromFileResourceModel struct {
	ID         types.String `tfsdk:"id"`
	SourceFile types.String `tfsdk:"source_file"`
	UseCaseID  types.String `tfsdk:"use_case_id"`
}

// VectorDatabaseResourceModel describes a vector database.
type VectorDatabaseResourceModel struct {
	ID                 types.String             `tfsdk:"id"`
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
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	UseCaseID   types.String `tfsdk:"use_case_id"`
}

// LLMBlueprintResourceModel describes the LLM blueprint resource.
type LLMBlueprintResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	PlaygroundID     types.String `tfsdk:"playground_id"`
	VectorDatabaseID types.String `tfsdk:"vector_database_id"`
	LLMID            types.String `tfsdk:"llm_id"`
}

// ModelResourceModel describes the custom model resource.
type CustomModelResourceModel struct {
	ID                             types.String                    `tfsdk:"id"`
	VersionID                      types.String                    `tfsdk:"version_id"`
	Name                           types.String                    `tfsdk:"name"`
	Description                    types.String                    `tfsdk:"description"`
	SourceLLMBlueprintID           types.String                    `tfsdk:"source_llm_blueprint_id"`
	BaseEnvironmentName            types.String                    `tfsdk:"base_environment_name"`
	BaseEnvironmentID              types.String                    `tfsdk:"base_environment_id"`
	BaseEnvironmentVersionID       types.String                    `tfsdk:"base_environment_version_id"`
	RuntimeParameters              []RuntimeParameterValueModel    `tfsdk:"runtime_parameters"`
	SourceRemoteRepositories       []SourceRemoteRepository        `tfsdk:"source_remote_repositories"`
	LocalFiles                     []types.String                  `tfsdk:"local_files"`
	Target                         types.String                    `tfsdk:"target"`
	TargetType                     types.String                    `tfsdk:"target_type"`
	IsProxy                        types.Bool                      `tfsdk:"is_proxy"`
	GuardConfigurations            []GuardConfiguration            `tfsdk:"guard_configurations"`
	OverallModerationConfiguration *OverallModerationConfiguration `tfsdk:"overall_moderation_configuration"`
}

type RuntimeParameterValueModel struct {
	Key   types.String `tfsdk:"key"`
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

type SourceRemoteRepository struct {
	ID          types.String   `tfsdk:"id"`
	Ref         types.String   `tfsdk:"ref"`
	SourcePaths []types.String `tfsdk:"source_paths"`
}

type GuardConfiguration struct {
	TemplateName     types.String      `tfsdk:"template_name"`
	Name             types.String      `tfsdk:"name"`
	Stages           []types.String    `tfsdk:"stages"`
	Intervention     GuardIntervention `tfsdk:"intervention"`
	DeploymentID     types.String      `tfsdk:"deployment_id"`
	InputColumnName  types.String      `tfsdk:"input_column_name"`
	OutputColumnName types.String      `tfsdk:"output_column_name"`
}

type GuardIntervention struct {
	Action    types.String   `tfsdk:"action"`
	Condition GuardCondition `tfsdk:"condition"`
	Message   types.String   `tfsdk:"message"`
}

type GuardCondition struct {
	Comparand  types.Float64 `tfsdk:"comparand"`
	Comparator types.String  `tfsdk:"comparator"`
}

type OverallModerationConfiguration struct {
	TimeoutSec    types.Int64  `tfsdk:"timeout_sec"`
	TimeoutAction types.String `tfsdk:"timeout_action"`
}

// RegisteredModelResourceModel describes the registered model resource.
type RegisteredModelResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	VersionID            types.String `tfsdk:"version_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	CustomModelVersionId types.String `tfsdk:"custom_model_version_id"`
}

// GlobalModelDataSourceModel describes the global model data source resource.
type GlobalModelDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	ID        types.String `tfsdk:"id"`
	VersionID types.String `tfsdk:"version_id"`
}

// PredictionEnvironmentResourceModel describes the prediction environment resource.
type PredictionEnvironmentResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Platform    types.String `tfsdk:"platform"`
	Description types.String `tfsdk:"description"`
}

// DeploymentResourceModel describes the deployment resource.
type DeploymentResourceModel struct {
	ID                       types.String        `tfsdk:"id"`
	Label                    types.String        `tfsdk:"label"`
	RegisteredModelVersionID types.String        `tfsdk:"registered_model_version_id"`
	PredictionEnvironmentID  types.String        `tfsdk:"prediction_environment_id"`
	Settings                 *DeploymentSettings `tfsdk:"settings"`
}

type DeploymentSettings struct {
	AssociationID        *AssociationIDSetting `tfsdk:"association_id"`
	PredictionRowStorage types.Bool            `tfsdk:"prediction_row_storage"`
	ChallengerAnalysis   types.Bool            `tfsdk:"challenger_analysis"`
	PredictionsSettings  *PredictionsSetting   `tfsdk:"predictions_settings"`
}

type AssociationIDSetting struct {
	AutoGenerateID types.Bool   `tfsdk:"auto_generate_id"`
	FeatureName    types.String `tfsdk:"feature_name"`
}

type PredictionsSetting struct {
	MinComputes types.Int32 `tfsdk:"min_computes"`
	MaxComputes types.Int32 `tfsdk:"max_computes"`
	RealTime    types.Bool  `tfsdk:"real_time"`
}

// ApplicationResourceModel describes the chat application resource.

type ChatApplicationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	VersionID      types.String `tfsdk:"version_id"`
	Name           types.String `tfsdk:"name"`
	DeploymentID   types.String `tfsdk:"deployment_id"`
	ApplicationUrl types.String `tfsdk:"application_url"`
}

type ApplicationSourceResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	VersionID  types.String   `tfsdk:"version_id"`
	Name       types.String   `tfsdk:"name"`
	LocalFiles []types.String `tfsdk:"local_files"`
}

type CustomApplicationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	SourceVersionID types.String `tfsdk:"source_version_id"`
	Name            types.String `tfsdk:"name"`
	ApplicationUrl  types.String `tfsdk:"application_url"`
}

// CredentialResourceModel describes the credential resource.
type ApiTokenCredentialResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ApiToken    types.String `tfsdk:"api_token"`
}

type BasicCredentialResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	User        types.String `tfsdk:"user"`
	Password    types.String `tfsdk:"password"`
}

type GoogleCloudCredentialResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	SourceFile types.String `tfsdk:"source_file"`
}
