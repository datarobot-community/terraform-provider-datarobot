package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	DataRobotApiKeyEnvVar   string = "DATAROBOT_API_TOKEN"
	DataRobotEndpointEnvVar string = "DATAROBOT_ENDPOINT"
	TimeoutMinutesEnvVar    string = "DATAROBOT_TIMEOUT_MINUTES"
	UserAgent               string = "DataRobotTerraformClient"
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

	// optional fields for S3 remote repositories
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"`
	AWSSessionToken    types.String `tfsdk:"aws_session_token"`
}

// DatasetFromFileResourceModel describes the datasource uploaded from a file.
type DatasetFromFileResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	FilePath   types.String   `tfsdk:"file_path"`
	FileHash   types.String   `tfsdk:"file_hash"`
	Name       types.String   `tfsdk:"name"`
	UseCaseIDs []types.String `tfsdk:"use_case_ids"`
}

type DatasetFromURLResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	URL        types.String   `tfsdk:"url"`
	Name       types.String   `tfsdk:"name"`
	UseCaseIDs []types.String `tfsdk:"use_case_ids"`
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
	ID                     types.String            `tfsdk:"id"`
	Name                   types.String            `tfsdk:"name"`
	Description            types.String            `tfsdk:"description"`
	PlaygroundID           types.String            `tfsdk:"playground_id"`
	VectorDatabaseID       types.String            `tfsdk:"vector_database_id"`
	VectorDatabaseSettings *VectorDatabaseSettings `tfsdk:"vector_database_settings"`
	LLMID                  types.String            `tfsdk:"llm_id"`
	LLMSettings            *LLMSettings            `tfsdk:"llm_settings"`
	PromptType             types.String            `tfsdk:"prompt_type"`
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

type FileTuple struct {
	LocalPath   string
	PathInModel string
}

type RuntimeParameterValue struct {
	Key   types.String `json:"key" tfsdk:"key"`
	Type  types.String `json:"type" tfsdk:"type"`
	Value types.String `json:"value" tfsdk:"value"`
}

type SourceRemoteRepository struct {
	ID          types.String   `tfsdk:"id"`
	Ref         types.String   `tfsdk:"ref"`
	SourcePaths []types.String `tfsdk:"source_paths"`
}

type GuardConfiguration struct {
	TemplateName       types.String      `tfsdk:"template_name"`
	Name               types.String      `tfsdk:"name"`
	Stages             []types.String    `tfsdk:"stages"`
	Intervention       GuardIntervention `tfsdk:"intervention"`
	DeploymentID       types.String      `tfsdk:"deployment_id"`
	InputColumnName    types.String      `tfsdk:"input_column_name"`
	OutputColumnName   types.String      `tfsdk:"output_column_name"`
	OpenAICredential   types.String      `tfsdk:"openai_credential"`
	OpenAIApiBase      types.String      `tfsdk:"openai_api_base"`
	OpenAIDeploymentID types.String      `tfsdk:"openai_deployment_id"`
	LlmType            types.String      `tfsdk:"llm_type"`
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

type OverallModerationConfiguration struct {
	TimeoutSec    types.Int64  `tfsdk:"timeout_sec"`
	TimeoutAction types.String `tfsdk:"timeout_action"`
}

// RegisteredModelResourceModel describes the registered model resource.
type RegisteredModelResourceModel struct {
	ID                   types.String   `tfsdk:"id"`
	VersionID            types.String   `tfsdk:"version_id"`
	VersionName          types.String   `tfsdk:"version_name"`
	Name                 types.String   `tfsdk:"name"`
	Description          types.String   `tfsdk:"description"`
	CustomModelVersionId types.String   `tfsdk:"custom_model_version_id"`
	UseCaseIDs           []types.String `tfsdk:"use_case_ids"`
}

type RegisteredModelFromLeaderboardResourceModel struct {
	ID                            types.String   `tfsdk:"id"`
	VersionID                     types.String   `tfsdk:"version_id"`
	VersionName                   types.String   `tfsdk:"version_name"`
	Name                          types.String   `tfsdk:"name"`
	Description                   types.String   `tfsdk:"description"`
	ModelID                       types.String   `tfsdk:"model_id"`
	PredictionThreshold           types.Float64  `tfsdk:"prediction_threshold"`
	ComputeAllTsIntervals         types.Bool     `tfsdk:"compute_all_ts_intervals"`
	DistributionPredictionModelID types.String   `tfsdk:"distribution_prediction_model_id"`
	UseCaseIDs                    []types.String `tfsdk:"use_case_ids"`
}

type Tag struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// GlobalModelDataSourceModel describes the global model data source resource.
type GlobalModelDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	ID        types.String `tfsdk:"id"`
	VersionID types.String `tfsdk:"version_id"`
}

// PredictionEnvironmentResourceModel describes the prediction environment resource.
type PredictionEnvironmentResourceModel struct {
	ID                     types.String   `tfsdk:"id"`
	Name                   types.String   `tfsdk:"name"`
	Platform               types.String   `tfsdk:"platform"`
	Description            types.String   `tfsdk:"description"`
	SupportedModelFormats  []types.String `tfsdk:"supported_model_formats"`
	BatchJobsPriority      types.String   `tfsdk:"batch_jobs_priority"`
	BatchJobsMaxConcurrent types.Int64    `tfsdk:"batch_jobs_max_concurrent"`
	ManagedBy              types.String   `tfsdk:"managed_by"`
	CredentialID           types.String   `tfsdk:"credential_id"`
	DatastoreID            types.String   `tfsdk:"datastore_id"`
}

// DeploymentResourceModel describes the deployment resource.
type DeploymentResourceModel struct {
	ID                       types.String   `tfsdk:"id"`
	Label                    types.String   `tfsdk:"label"`
	RegisteredModelVersionID types.String   `tfsdk:"registered_model_version_id"`
	PredictionEnvironmentID  types.String   `tfsdk:"prediction_environment_id"`
	Importance               types.String   `tfsdk:"importance"`
	UseCaseIDs               []types.String `tfsdk:"use_case_ids"`

	// settings
	PredictionsByForecastDateSettings *PredictionsByForecastDateSettings `tfsdk:"predictions_by_forecast_date_settings"`
	ChallengerModelsSettings          *BasicDeploymentSetting            `tfsdk:"challenger_models_settings"`
	SegmentAnalysisSettings           *SegmentAnalysisSettings           `tfsdk:"segment_analysis_settings"`
	BiasAndFairnessSettings           *BiasAndFairnessSettings           `tfsdk:"bias_and_fairness_settings"`
	ChallengerReplaySettings          *BasicDeploymentSetting            `tfsdk:"challenger_replay_settings"`
	DriftTrackingSettings             *DriftTrackingSettings             `tfsdk:"drift_tracking_settings"`
	AssociationIDSettings             *AssociationIDSettings             `tfsdk:"association_id_settings"`
	PredictionsDataCollectionSettings *BasicDeploymentSetting            `tfsdk:"predictions_data_collection_settings"`
	PredictionWarningSettings         *PredictionWarningSettings         `tfsdk:"prediction_warning_settings"`
	PredictionIntervalsSettings       *PredictionIntervalsSettings       `tfsdk:"prediction_intervals_settings"`
	HealthSettings                    *HealthSettings                    `tfsdk:"health_settings"`
	PredictionsSettings               *PredictionsSettings               `tfsdk:"predictions_settings"`
}

type BasicDeploymentSetting struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type PredictionsByForecastDateSettings struct {
	Enabled        types.Bool   `tfsdk:"enabled"`
	ColumnName     types.String `tfsdk:"column_name"`
	DatetimeFormat types.String `tfsdk:"datetime_format"`
}

type SegmentAnalysisSettings struct {
	Enabled    types.Bool     `tfsdk:"enabled"`
	Attributes []types.String `tfsdk:"attributes"`
}

type BiasAndFairnessSettings struct {
	ProtectedFeatures     []types.String `tfsdk:"protected_features"`
	PreferableTargetValue types.Bool     `tfsdk:"preferable_target_value"`
	FairnessMetricSet     types.String   `tfsdk:"fairness_metric_set"`
	FairnessThreshold     types.Float64  `tfsdk:"fairness_threshold"`
}

type DriftTrackingSettings struct {
	TargetDriftEnabled  types.Bool `tfsdk:"target_drift_enabled"`
	FeatureDriftEnabled types.Bool `tfsdk:"feature_drift_enabled"`
}

type DeploymentSettings struct {
	AssociationID        *AssociationIDSettings `tfsdk:"association_id"`
	PredictionRowStorage types.Bool             `tfsdk:"prediction_row_storage"`
	ChallengerAnalysis   types.Bool             `tfsdk:"challenger_analysis"`
	PredictionsSettings  *PredictionsSettings   `tfsdk:"predictions_settings"`
}

type AssociationIDSettings struct {
	AutoGenerateID               types.Bool     `tfsdk:"auto_generate_id"`
	ColumnNames                  []types.String `tfsdk:"column_names"`
	RequiredInPredictionRequests types.Bool     `tfsdk:"required_in_prediction_requests"`
}

type PredictionWarningSettings struct {
	Enabled          types.Bool        `tfsdk:"enabled"`
	CustomBoundaries *CustomBoundaries `tfsdk:"custom_boundaries"`
}

type CustomBoundaries struct {
	UpperBoundary types.Float64 `tfsdk:"uppder_boundary"`
	LowerBoundary types.Float64 `tfsdk:"lower_boundary"`
}

type PredictionIntervalsSettings struct {
	Enabled     types.Bool    `tfsdk:"enabled"`
	Percentiles []types.Int64 `tfsdk:"percentiles"`
}

type HealthSettings struct {
	Service               *ServiceHealthSettings       `tfsdk:"service"`
	DataDrift             *DataDriftHealthSettings     `tfsdk:"data_drift"`
	Accuracy              *AccuracyHealthSettings      `tfsdk:"accuracy"`
	Fairness              *FairnessHealthSettings      `tfsdk:"fairness"`
	CustomMetrics         *CustomMetricsHealthSettings `tfsdk:"custom_metrics"`
	PredictionsTimeliness *TimelinessHealthSettings    `tfsdk:"predictions_timeliness"`
	ActualsTimeliness     *TimelinessHealthSettings    `tfsdk:"actuals_timeliness"`
}

type ServiceHealthSettings struct {
	BatchCount types.Int64 `tfsdk:"batch_count"`
}

type DataDriftHealthSettings struct {
	BatchCount                 types.Int64    `tfsdk:"batch_count"`
	TimeInterval               types.String   `tfsdk:"time_interval"`
	DriftThreshold             types.Float64  `tfsdk:"drift_threshold"`
	ImportanceThreshold        types.Float64  `tfsdk:"importance_threshold"`
	LowImportanceWarningCount  types.Int64    `tfsdk:"low_importance_warning_count"`
	LowImportanceFailingCount  types.Int64    `tfsdk:"low_importance_failing_count"`
	HighImportanceWarningCount types.Int64    `tfsdk:"high_importance_warning_count"`
	HighImportanceFailingCount types.Int64    `tfsdk:"high_importance_failing_count"`
	ExcludeFeatures            []types.String `tfsdk:"exclude_features"`
	StarredFeatures            []types.String `tfsdk:"starred_features"`
}

type AccuracyHealthSettings struct {
	BatchCount       types.Int64   `tfsdk:"batch_count"`
	Metric           types.String  `tfsdk:"metric"`
	Measurement      types.String  `tfsdk:"measurement"`
	WarningThreshold types.Float64 `tfsdk:"warning_threshold"`
	FailingThreshold types.Float64 `tfsdk:"failing_threshold"`
}

type FairnessHealthSettings struct {
	ProtectedClassWarningCount types.Int64 `tfsdk:"protected_class_warning_count"`
	ProtectedClassFailingCount types.Int64 `tfsdk:"protected_class_failing_count"`
}

type CustomMetricsHealthSettings struct {
	WarningConditions []CustomMetricCondition `tfsdk:"warning_conditions"`
	FailingConditions []CustomMetricCondition `tfsdk:"failing_conditions"`
}

type CustomMetricCondition struct {
	MetricID        types.String  `tfsdk:"metric_id"`
	CompareOperator types.String  `tfsdk:"compare_operator"`
	Threshold       types.Float64 `tfsdk:"threshold"`
}

type TimelinessHealthSettings struct {
	Enabled           types.Bool   `tfsdk:"enabled"`
	ExpectedFrequency types.String `tfsdk:"expected_frequency"`
}

type PredictionsSettings struct {
	MinComputes types.Int64 `tfsdk:"min_computes"`
	MaxComputes types.Int64 `tfsdk:"max_computes"`
}

// QAApplicationResourceModel describes the Q&A application resource.

type QAApplicationResourceModel struct {
	ID                       types.String   `tfsdk:"id"`
	SourceID                 types.String   `tfsdk:"source_id"`
	SourceVersionID          types.String   `tfsdk:"source_version_id"`
	Name                     types.String   `tfsdk:"name"`
	DeploymentID             types.String   `tfsdk:"deployment_id"`
	ApplicationUrl           types.String   `tfsdk:"application_url"`
	ExternalAccessEnabled    types.Bool     `tfsdk:"external_access_enabled"`
	ExternalAccessRecipients []types.String `tfsdk:"external_access_recipients"`
}

type ApplicationSourceResourceModel struct {
	ID                       types.String  `tfsdk:"id"`
	VersionID                types.String  `tfsdk:"version_id"`
	Name                     types.String  `tfsdk:"name"`
	BaseEnvironmentID        types.String  `tfsdk:"base_environment_id"`
	BaseEnvironmentVersionID types.String  `tfsdk:"base_environment_version_id"`
	FolderPath               types.String  `tfsdk:"folder_path"`
	FolderPathHash           types.String  `tfsdk:"folder_path_hash"`
	Files                    types.Dynamic `tfsdk:"files"`
	FilesHashes              types.List    `tfsdk:"files_hashes"`
	Replicas                 types.Int64   `tfsdk:"replicas"`
	RuntimeParameterValues   types.List    `tfsdk:"runtime_parameter_values"`
}

type CustomApplicationResourceModel struct {
	ID                       types.String   `tfsdk:"id"`
	SourceID                 types.String   `tfsdk:"source_id"`
	SourceVersionID          types.String   `tfsdk:"source_version_id"`
	Name                     types.String   `tfsdk:"name"`
	ApplicationUrl           types.String   `tfsdk:"application_url"`
	ExternalAccessEnabled    types.Bool     `tfsdk:"external_access_enabled"`
	ExternalAccessRecipients []types.String `tfsdk:"external_access_recipients"`
	UseCaseIDs               []types.String `tfsdk:"use_case_ids"`
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
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	GCPKey         types.String `tfsdk:"gcp_key"`
	GCPKeyFile     types.String `tfsdk:"gcp_key_file"`
	GCPKeyFileHash types.String `tfsdk:"gcp_key_file_hash"`
}
