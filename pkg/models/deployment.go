package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
	UseCaseID              types.String       `tfsdk:"use_case_id"`
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
