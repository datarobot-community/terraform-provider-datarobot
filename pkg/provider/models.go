package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	DataRobotUserNameEnvVar     string = "DATAROBOT_USERNAME"
	DatarobotUserIDEnvVar       string = "DATAROBOT_USER_ID"
	DataRobotApiKeyEnvVar       string = "DATAROBOT_API_TOKEN"
	DataRobotEndpointEnvVar     string = "DATAROBOT_ENDPOINT"
	DataRobotTraceContextEnvVar string = "DATAROBOT_TRACE_CONTEXT"
	TimeoutMinutesEnvVar        string = "DATAROBOT_TIMEOUT_MINUTES"
	UserAgent                   string = "DataRobotTerraformClient"

	PromptRuntimeParameterName string = "PROMPT_COLUMN_NAME"
)

// NotebookResourceModel describes the notebook resource.
type NotebookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	FilePath  types.String `tfsdk:"file_path"`
	FileHash  types.String `tfsdk:"file_hash"`
	UseCaseID types.String `tfsdk:"use_case_id"`
	URL       types.String `tfsdk:"url"`
}

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

type DatasetFromDatasourceResourceModel struct {
	ID                        types.String   `tfsdk:"id"`
	DataSourceID              types.String   `tfsdk:"data_source_id"`
	CredentialID              types.String   `tfsdk:"credential_id"`
	DoSnapshot                types.Bool     `tfsdk:"do_snapshot"`
	PersistDataAfterIngestion types.Bool     `tfsdk:"persist_data_after_ingestion"`
	UseKerberos               types.Bool     `tfsdk:"use_kerberos"`
	SampleSizeRows            types.Int64    `tfsdk:"sample_size_rows"`
	Categories                []types.String `tfsdk:"categories"`
	UseCaseIDs                []types.String `tfsdk:"use_case_ids"`
}

// DatastoreResourceModel describes the datastore resource.
type DatastoreResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DataStoreType types.String `tfsdk:"data_store_type"`
	CanonicalName types.String `tfsdk:"canonical_name"`
	DriverID      types.String `tfsdk:"driver_id"`
	JDBCUrl       types.String `tfsdk:"jdbc_url"`
	Fields        []types.Map  `tfsdk:"fields"`
	ConnectorID   types.String `tfsdk:"connector_id"`
}

// DatasourceResourceModel describes the datasource resource.
type DatasourceResourceModel struct {
	ID             types.String          `tfsdk:"id"`
	DataSourceType types.String          `tfsdk:"data_source_type"`
	CanonicalName  types.String          `tfsdk:"canonical_name"`
	Params         DatasourceParamsModel `tfsdk:"params"`
}

type DatasourceParamsModel struct {
	DataStoreID     types.String `tfsdk:"data_store_id"`
	Table           types.String `tfsdk:"table"`
	Schema          types.String `tfsdk:"schema"`
	PartitionColumn types.String `tfsdk:"partition_column"`
	Query           types.String `tfsdk:"query"`
	FetchSize       types.Int64  `tfsdk:"fetch_size"`
	Path            types.String `tfsdk:"path"`
	Catalog         types.String `tfsdk:"catalog"`
}

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
	CustomModelLLMSettings *CustomModelLLMSettings `tfsdk:"custom_model_llm_settings"`
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
	NemoInfo           *NemoInfo         `tfsdk:"nemo_info"`
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
}

type CustomMetricJobResourceModel struct {
	ID                     types.String  `tfsdk:"id"`
	Name                   types.String  `tfsdk:"name"`
	Description            types.String  `tfsdk:"description"`
	EnvironmentID          types.String  `tfsdk:"environment_id"`
	EnvironmentVersionID   types.String  `tfsdk:"environment_version_id"`
	RuntimeParameterValues types.List    `tfsdk:"runtime_parameter_values"`
	FolderPath             types.String  `tfsdk:"folder_path"`
	FolderPathHash         types.String  `tfsdk:"folder_path_hash"`
	Files                  types.Dynamic `tfsdk:"files"`
	FilesHashes            types.List    `tfsdk:"files_hashes"`
	EgressNetworkPolicy    types.String  `tfsdk:"egress_network_policy"`
	ResourceBundleID       types.String  `tfsdk:"resource_bundle_id"`
	Directionality         types.String  `tfsdk:"directionality"`
	Units                  types.String  `tfsdk:"units"`
	Type                   types.String  `tfsdk:"type"`
	TimeStep               types.String  `tfsdk:"time_step"`
	IsModelSpecific        types.Bool    `tfsdk:"is_model_specific"`
}

type CustomMetricFromJobResourceModel struct {
	ID                 types.String             `tfsdk:"id"`
	DeploymentID       types.String             `tfsdk:"deployment_id"`
	CustomJobID        types.String             `tfsdk:"custom_job_id"`
	Name               types.String             `tfsdk:"name"`
	Description        types.String             `tfsdk:"description"`
	BaselineValue      types.Float64            `tfsdk:"baseline_value"`
	Timestamp          *MetricTimestampSpoofing `tfsdk:"timestamp"`
	Value              *ColumnNameValue         `tfsdk:"value"`
	Batch              *ColumnNameValue         `tfsdk:"batch"`
	SampleCount        *ColumnNameValue         `tfsdk:"sample_count"`
	Schedule           *Schedule                `tfsdk:"schedule"`
	ParameterOverrides types.List               `tfsdk:"parameter_overrides"`
}

type CustomMetricResourceModel struct {
	ID              types.String             `tfsdk:"id"`
	DeploymentID    types.String             `tfsdk:"deployment_id"`
	Name            types.String             `tfsdk:"name"`
	Description     types.String             `tfsdk:"description"`
	Directionality  types.String             `tfsdk:"directionality"`
	Units           types.String             `tfsdk:"units"`
	Type            types.String             `tfsdk:"type"`
	IsModelSpecific types.Bool               `tfsdk:"is_model_specific"`
	IsGeospatial    types.Bool               `tfsdk:"is_geospatial"`
	BaselineValue   types.Float64            `tfsdk:"baseline_value"`
	Timestamp       *MetricTimestampSpoofing `tfsdk:"timestamp"`
	Value           *ColumnNameValue         `tfsdk:"value"`
	SampleCount     *ColumnNameValue         `tfsdk:"sample_count"`
	Batch           *ColumnNameValue         `tfsdk:"batch"`
}

type MetricTimestampSpoofing struct {
	ColumnName types.String `tfsdk:"column_name"`
	TimeFormat types.String `tfsdk:"time_format"`
}

type ColumnNameValue struct {
	ColumnName types.String `tfsdk:"column_name"`
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
	RuntimeParameterValues   types.List     `tfsdk:"runtime_parameter_values"`
	UseCaseIDs               []types.String `tfsdk:"use_case_ids"`

	// settings
	PredictionsByForecastDateSettings *PredictionsByForecastDateSettings `tfsdk:"predictions_by_forecast_date_settings"`
	ChallengerModelsSettings          *BasicDeploymentSetting            `tfsdk:"challenger_models_settings"`
	SegmentAnalysisSettings           *SegmentAnalysisSettings           `tfsdk:"segment_analysis_settings"`
	BiasAndFairnessSettings           *BiasAndFairnessSettings           `tfsdk:"bias_and_fairness_settings"`
	ChallengerReplaySettings          *BasicDeploymentSetting            `tfsdk:"challenger_replay_settings"`
	BatchMonitoringSettings           *BasicDeploymentSetting            `tfsdk:"batch_monitoring_settings"`
	DriftTrackingSettings             *DriftTrackingSettings             `tfsdk:"drift_tracking_settings"`
	AssociationIDSettings             *AssociationIDSettings             `tfsdk:"association_id_settings"`
	PredictionsDataCollectionSettings *BasicDeploymentSetting            `tfsdk:"predictions_data_collection_settings"`
	PredictionWarningSettings         *PredictionWarningSettings         `tfsdk:"prediction_warning_settings"`
	PredictionIntervalsSettings       *PredictionIntervalsSettings       `tfsdk:"prediction_intervals_settings"`
	HealthSettings                    *HealthSettings                    `tfsdk:"health_settings"`
	PredictionsSettings               *PredictionsSettings               `tfsdk:"predictions_settings"`
	FeatureCacheSettings              *FeatureCacheSettings              `tfsdk:"feature_cache_settings"`
	RetrainingSettings                *RetrainingSettings                `tfsdk:"retraining_settings"`
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
	TargetDriftEnabled  types.Bool     `tfsdk:"target_drift_enabled"`
	FeatureDriftEnabled types.Bool     `tfsdk:"feature_drift_enabled"`
	FeatureSelection    types.String   `tfsdk:"feature_selection"`
	TrackedFeatures     []types.String `tfsdk:"tracked_features"`
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
	MinComputes      types.Int64  `tfsdk:"min_computes"`
	MaxComputes      types.Int64  `tfsdk:"max_computes"`
	ResourceBundleID types.String `tfsdk:"resource_bundle_id"`
}

type FeatureCacheSettings struct {
	Enabled  types.Bool `tfsdk:"enabled"`
	Fetching types.Bool `tfsdk:"fetching"`
	Schedule *Schedule  `tfsdk:"schedule"`
}

type RetrainingSettings struct {
	RetrainingUserID        types.String `tfsdk:"retraining_user_id"`
	DatasetID               types.String `tfsdk:"dataset_id"`
	CredentialID            types.String `tfsdk:"credential_id"`
	PredictionEnvironmentID types.String `tfsdk:"prediction_environment_id"`
}

// DeploymentRetrainingPolicyResourceModel describes the deployment retraining policy resource.
type DeploymentRetrainingPolicyResourceModel struct {
	ID                     types.String       `tfsdk:"id"`
	DeploymentID           types.String       `tfsdk:"deployment_id"`
	Name                   types.String       `tfsdk:"name"`
	Description            types.String       `tfsdk:"description"`
	Action                 types.String       `tfsdk:"action"`
	FeatureListStrategy    types.String       `tfsdk:"feature_list_strategy"`
	ModelSelectionStrategy types.String       `tfsdk:"model_selection_strategy"`
	AutopilotOptions       *AutopilotOptions  `tfsdk:"autopilot_options"`
	ProjectOptions         *ProjectOptions    `tfsdk:"project_options"`
	ProjectOptionsStrategy types.String       `tfsdk:"project_options_strategy"`
	TimeSeriesOptions      *TimeSeriesOptions `tfsdk:"time_series_options"`
	Trigger                *Trigger           `tfsdk:"trigger"`
}

type AutopilotOptions struct {
	BlendBestModels              types.Bool   `tfsdk:"blend_best_models"`
	Mode                         types.String `tfsdk:"mode"`
	RunLeakageRemovedFeatureList types.Bool   `tfsdk:"run_leakage_removed_feature_list"`
	ScoringCodeOnly              types.Bool   `tfsdk:"scoring_code_only"`
	ShapOnlyMode                 types.Bool   `tfsdk:"shap_only_mode"`
}

type ProjectOptions struct {
	CVMethod       types.String  `tfsdk:"cv_method"`
	HoldoutPct     types.Float64 `tfsdk:"holdout_pct"`
	ValidationPct  types.Float64 `tfsdk:"validation_pct"`
	Metric         types.String  `tfsdk:"metric"`
	Reps           types.Float64 `tfsdk:"reps"`
	ValidationType types.String  `tfsdk:"validation_type"`
}

type TimeSeriesOptions struct {
	CalendarID                       types.String  `tfsdk:"calendar_id"`
	DifferencingMethod               types.String  `tfsdk:"differencing_method"`
	ExponentiallyWeightedMovingAlpha types.Float64 `tfsdk:"exponentially_weighted_moving_alpha"`
	Periodicities                    []Periodicity `tfsdk:"periodicities"`
	TreatAsExponential               types.String  `tfsdk:"treat_as_exponential"`
}

type Periodicity struct {
	TimeSteps types.Int64  `tfsdk:"time_steps"`
	TimeUnit  types.String `tfsdk:"time_unit"`
}

type Trigger struct {
	CustomJobID             types.String `tfsdk:"custom_job_id"`
	MinIntervalBetweenRuns  types.String `tfsdk:"min_interval_between_runs"`
	Schedule                *Schedule    `tfsdk:"schedule"`
	StatusDeclinesToFailing types.Bool   `tfsdk:"status_declines_to_failing"`
	StatusDeclinesToWarning types.Bool   `tfsdk:"status_declines_to_warning"`
	StatusStillInDecline    types.Bool   `tfsdk:"status_still_in_decline"`
	Type                    types.String `tfsdk:"type"`
}

type NotificationPolicyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ChannelID         types.String `tfsdk:"channel_id"`
	ChannelScope      types.String `tfsdk:"channel_scope"`
	RelatedEntityID   types.String `tfsdk:"related_entity_id"`
	RelatedEntityType types.String `tfsdk:"related_entity_type"`
	Active            types.Bool   `tfsdk:"active"`
	EventGroup        types.String `tfsdk:"event_group"`
	EventType         types.String `tfsdk:"event_type"`
	MaximalFrequency  types.String `tfsdk:"maximal_frequency"`
}

type NotificationChannelResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	ChannelType       types.String   `tfsdk:"channel_type"`
	ContentType       types.String   `tfsdk:"content_type"`
	CustomHeaders     []CustomHeader `tfsdk:"custom_headers"`
	DREntities        []DREntity     `tfsdk:"dr_entities"`
	EmailAddress      types.String   `tfsdk:"email_address"`
	LanguageCode      types.String   `tfsdk:"language_code"`
	PayloadUrl        types.String   `tfsdk:"payload_url"`
	RelatedEntityID   types.String   `tfsdk:"related_entity_id"`
	RelatedEntityType types.String   `tfsdk:"related_entity_type"`
	SecretToken       types.String   `tfsdk:"secret_token"`
	ValidateSsl       types.Bool     `tfsdk:"validate_ssl"`
	VerificationCode  types.String   `tfsdk:"verification_code"`
}

type CustomHeader struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type DREntity struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
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
	AllowAutoStopping        types.Bool     `tfsdk:"allow_auto_stopping"`
}

type ApplicationSourceResourceModel struct {
	ID                       types.String                `tfsdk:"id"`
	VersionID                types.String                `tfsdk:"version_id"`
	Name                     types.String                `tfsdk:"name"`
	BaseEnvironmentID        types.String                `tfsdk:"base_environment_id"`
	BaseEnvironmentVersionID types.String                `tfsdk:"base_environment_version_id"`
	FolderPath               types.String                `tfsdk:"folder_path"`
	FolderPathHash           types.String                `tfsdk:"folder_path_hash"`
	Files                    types.Dynamic               `tfsdk:"files"`
	FilesHashes              types.List                  `tfsdk:"files_hashes"`
	Resources                *ApplicationSourceResources `tfsdk:"resources"`
	RuntimeParameterValues   types.List                  `tfsdk:"runtime_parameter_values"`
}

type ApplicationSourceFromTemplateResourceModel struct {
	ID                       types.String                `tfsdk:"id"`
	VersionID                types.String                `tfsdk:"version_id"`
	TemplateID               types.String                `tfsdk:"template_id"`
	Name                     types.String                `tfsdk:"name"`
	BaseEnvironmentID        types.String                `tfsdk:"base_environment_id"`
	BaseEnvironmentVersionID types.String                `tfsdk:"base_environment_version_id"`
	FolderPath               types.String                `tfsdk:"folder_path"`
	FolderPathHash           types.String                `tfsdk:"folder_path_hash"`
	Files                    types.Dynamic               `tfsdk:"files"`
	FilesHashes              types.List                  `tfsdk:"files_hashes"`
	Resources                *ApplicationSourceResources `tfsdk:"resources"`
	RuntimeParameterValues   types.List                  `tfsdk:"runtime_parameter_values"`
}

type ApplicationSourceResources struct {
	Replicas        types.Int64  `tfsdk:"replicas"`
	SessionAffinity types.Bool   `tfsdk:"session_affinity"`
	ResourceLabel   types.String `tfsdk:"resource_label"`
}

type CustomApplicationResourceModel struct {
	ID                       types.String   `tfsdk:"id"`
	SourceID                 types.String   `tfsdk:"source_id"`
	SourceVersionID          types.String   `tfsdk:"source_version_id"`
	Name                     types.String   `tfsdk:"name"`
	ApplicationUrl           types.String   `tfsdk:"application_url"`
	ExternalAccessEnabled    types.Bool     `tfsdk:"external_access_enabled"`
	ExternalAccessRecipients []types.String `tfsdk:"external_access_recipients"`
	AllowAutoStopping        types.Bool     `tfsdk:"allow_auto_stopping"`
	UseCaseIDs               []types.String `tfsdk:"use_case_ids"`
}

type CustomApplicationFromEnvironmentResourceModel struct {
	ID                       types.String   `tfsdk:"id"`
	EnvironmentID            types.String   `tfsdk:"environment_id"`
	EnvironmentVersionID     types.String   `tfsdk:"environment_version_id"`
	Name                     types.String   `tfsdk:"name"`
	ApplicationUrl           types.String   `tfsdk:"application_url"`
	ExternalAccessEnabled    types.Bool     `tfsdk:"external_access_enabled"`
	ExternalAccessRecipients []types.String `tfsdk:"external_access_recipients"`
	AllowAutoStopping        types.Bool     `tfsdk:"allow_auto_stopping"`
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

type AwsCredentialResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"`
	AWSSessionToken    types.String `tfsdk:"aws_session_token"`
	ConfigID           types.String `tfsdk:"config_id"`
}

type AzureCredentialResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	AzureConnectionString types.String `tfsdk:"azure_connection_string"`
}

// ExecutionEnvironmentDataSourceModel describes the execution environment data source resource.
type ExecutionEnvironmentDataSourceModel struct {
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	Description         types.String `tfsdk:"description"`
	ProgrammingLanguage types.String `tfsdk:"programming_language"`
	VersionID           types.String `tfsdk:"version_id"`
}

// ExecutionEnvironmentResourceModel describes the execution environment resource.
type ExecutionEnvironmentResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	ProgrammingLanguage types.String   `tfsdk:"programming_language"`
	Description         types.String   `tfsdk:"description"`
	UseCases            []types.String `tfsdk:"use_cases"`
	VersionID           types.String   `tfsdk:"version_id"`
	VersionDescription  types.String   `tfsdk:"version_description"`
	DockerContextPath   types.String   `tfsdk:"docker_context_path"`
	DockerContextHash   types.String   `tfsdk:"docker_context_hash"`
	DockerImage         types.String   `tfsdk:"docker_image"`
	BuildStatus         types.String   `tfsdk:"build_status"`
}

// BatchPredictionJobModel describes the batch prediction job resource.
type BatchPredictionJobDefinitionResourceModel struct {
	ID                          types.String        `tfsdk:"id"`
	DeploymentID                types.String        `tfsdk:"deployment_id"`
	Name                        types.String        `tfsdk:"name"`
	Enabled                     types.Bool          `tfsdk:"enabled"`
	Schedule                    *Schedule           `tfsdk:"schedule"`
	AbortOnError                types.Bool          `tfsdk:"abort_on_error"`
	ChunkSize                   types.Dynamic       `tfsdk:"chunk_size"`
	ColumnNamesRemapping        types.Map           `tfsdk:"column_names_remapping"`
	CSVSettings                 *CSVSettings        `tfsdk:"csv_settings"`
	ExplanationAlgorithm        types.String        `tfsdk:"explanation_algorithm"`
	IncludePredictionStatus     types.Bool          `tfsdk:"include_prediction_status"`
	IncludeProbabilities        types.Bool          `tfsdk:"include_probabilities"`
	IncludeProbabilitiesClasses []types.String      `tfsdk:"include_probabilities_classes"`
	IntakeSettings              IntakeSettings      `tfsdk:"intake_settings"`
	MaxExplanations             types.Int64         `tfsdk:"max_explanations"`
	NumConcurrent               types.Int64         `tfsdk:"num_concurrent"`
	OutputSettings              *OutputSettings     `tfsdk:"output_settings"`
	PassthroughColumns          []types.String      `tfsdk:"passthrough_columns"`
	PassthroughColumnsSet       types.String        `tfsdk:"passthrough_columns_set"`
	PredictionInstance          *PredictionInstance `tfsdk:"prediction_instance"`
	PredictionThreshold         types.Float64       `tfsdk:"prediction_threshold"`
	PredictionWarningEnabled    types.Bool          `tfsdk:"prediction_warning_enabled"`
	SkipDriftTracking           types.Bool          `tfsdk:"skip_drift_tracking"`
	ThresholdHigh               types.Float64       `tfsdk:"threshold_high"`
	ThresholdLow                types.Float64       `tfsdk:"threshold_low"`
	TimeseriesSettings          *TimeseriesSettings `tfsdk:"timeseries_settings"`
}

type Schedule struct {
	Minute     []types.String `tfsdk:"minute"`
	Hour       []types.String `tfsdk:"hour"`
	Month      []types.String `tfsdk:"month"`
	DayOfMonth []types.String `tfsdk:"day_of_month"`
	DayOfWeek  []types.String `tfsdk:"day_of_week"`
}

type IntakeSettings struct {
	Type         types.String `tfsdk:"type"`
	DatasetID    types.String `tfsdk:"dataset_id"`
	File         types.String `tfsdk:"file"`
	URL          types.String `tfsdk:"url"`
	CredentialID types.String `tfsdk:"credential_id"`
	EndpointURL  types.String `tfsdk:"endpoint_url"`
	DataStoreID  types.String `tfsdk:"data_store_id"`
	Query        types.String `tfsdk:"query"`
	Table        types.String `tfsdk:"table"`
	Schema       types.String `tfsdk:"schema"`
	Catalog      types.String `tfsdk:"catalog"`
	FetchSize    types.Int64  `tfsdk:"fetch_size"`
}

type CSVSettings struct {
	Delimiter types.String `tfsdk:"delimiter"`
	Encoding  types.String `tfsdk:"encoding"`
	QuoteChar types.String `tfsdk:"quotechar"`
}

type OutputSettings struct {
	CredentialID           types.String   `tfsdk:"credential_id"`
	Type                   types.String   `tfsdk:"type"`
	URL                    types.String   `tfsdk:"url"`
	Path                   types.String   `tfsdk:"path"`
	EndpointURL            types.String   `tfsdk:"endpoint_url"`
	DataStoreID            types.String   `tfsdk:"data_store_id"`
	Table                  types.String   `tfsdk:"table"`
	Schema                 types.String   `tfsdk:"schema"`
	Catalog                types.String   `tfsdk:"catalog"`
	StatementType          types.String   `tfsdk:"statement_type"`
	UpdateColumns          []types.String `tfsdk:"update_columns"`
	WhereColumns           []types.String `tfsdk:"where_columns"`
	CreateTableIfNotExists types.Bool     `tfsdk:"create_table_if_not_exists"`
}

type PredictionInstance struct {
	ApiKey       types.String `tfsdk:"api_key"`
	DatarobotKey types.String `tfsdk:"datarobot_key"`
	HostName     types.String `tfsdk:"host_name"`
	SSLEnabled   types.Bool   `tfsdk:"ssl_enabled"`
}

type TimeseriesSettings struct {
	ForecastPoint                    types.String `tfsdk:"forecast_point"`
	RelaxKnownInAdvanceFeaturesCheck types.Bool   `tfsdk:"relax_known_in_advance_features_check"`
	Type                             types.String `tfsdk:"type"`
	PredictionsStartDate             types.String `tfsdk:"predictions_start_date"`
	PredictionsEndDate               types.String `tfsdk:"predictions_end_date"`
}
