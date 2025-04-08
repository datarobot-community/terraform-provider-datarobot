package client

type CreateDeploymentFromModelPackageRequest struct {
	ModelPackageID          string `json:"modelPackageId"`
	PredictionEnvironmentID string `json:"predictionEnvironmentId"`
	Label                   string `json:"label"`
	Importance              string `json:"importance"`
	RuntimeParameterValues  string `json:"runtimeParameterValues,omitempty"`
}

type DeploymentCreateResponse struct {
	ID string `json:"id"`
}

type Deployment struct {
	ID                    string                `json:"id"`
	Label                 string                `json:"label"`
	Status                string                `json:"status"`
	Model                 Model                 `json:"model"`
	ModelPackage          ModelPackage          `json:"modelPackage"`
	PredictionEnvironment PredictionEnvironment `json:"predictionEnvironment"`
	Importance            string                `json:"importance"`
}

type Model struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	TargetName string `json:"targetName"`
	TargetType string `json:"targetType"`
}

type ModelPackage struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	RegisteredModelID string `json:"registeredModelId"`
}

type UpdateDeploymentRequest struct {
	Label      string `json:"label"`
	Importance string `json:"importance"`
}

type DeploymentSettings struct {
	AssociationID             *AssociationIDSetting              `json:"associationId,omitempty"`
	BatchMonitoring           *BasicSetting                      `json:"batchMonitoring,omitempty"`
	BiasAndFairness           *BiasAndFairnessSetting            `json:"biasAndFairness,omitempty"`
	ChallengerModels          *BasicSetting                      `json:"challengerModels,omitempty"`
	FeatureDrift              *FeatureDriftSetting               `json:"featureDrift,omitempty"`
	Humility                  *BasicSetting                      `json:"humility,omitempty"`
	PredictionsSettings       *PredictionsSettings               `json:"predictionsSettings,omitempty"`
	PredictionsByForecastDate *PredictionsByForecastDateSettings `json:"predictionsByForecastDate,omitempty"`
	PredictionsDataCollection *BasicSetting                      `json:"predictionsDataCollection,omitempty"`
	PredictionIntervals       *PredictionIntervalsSetting        `json:"predictionIntervals,omitempty"`
	PredictionWarning         *PredictionWarningSetting          `json:"predictionWarning,omitempty"`
	SegmentAnalysis           *SegmentAnalysisSetting            `json:"segmentAnalysis,omitempty"`
	TargetDrift               *BasicSetting                      `json:"targetDrift,omitempty"`
}

type AssociationIDSetting struct {
	AutoGenerateID               bool     `json:"autoGenerateId"`
	RequiredInPredictionRequests bool     `json:"requiredInPredictionRequests"`
	ColumnNames                  []string `json:"columnNames,omitempty"`
}

type BasicSetting struct {
	Enabled bool `json:"enabled"`
}

type BiasAndFairnessSetting struct {
	ProtectedFeatures     []string `json:"protectedFeatures"`
	PreferableTargetValue bool     `json:"preferableTargetValue"`
	FairnessMetricsSet    string   `json:"fairnessMetricsSet"`
	FairnessThreshold     float64  `json:"fairnessThreshold"`
}

type PredictionsSettings struct {
	MinComputes      *int64  `json:"minComputes,omitempty"`
	MaxComputes      *int64  `json:"maxComputes,omitempty"`
	ResourceBundleID *string `json:"resourceBundleId,omitempty"`
}

type PredictionsByForecastDateSettings struct {
	Enabled        bool   `json:"enabled"`
	ColumnName     string `json:"columnName"`
	DatetimeFormat string `json:"datetimeFormat"`
}

type PredictionIntervalsSetting struct {
	Enabled     bool    `json:"enabled"`
	Percentiles []int64 `json:"percentiles"`
}

type PredictionWarningSetting struct {
	Enabled          bool              `json:"enabled"`
	CustomBoundaries *CustomBoundaries `json:"customBoundaries,omitempty"`
}

type CustomBoundaries struct {
	LowerBoundary float64 `json:"lowerBoundary"`
	UpperBoundary float64 `json:"upperBoundary"`
}

type SegmentAnalysisSetting struct {
	Enabled    bool      `json:"enabled"`
	Attributes *[]string `json:"attributes,omitempty"`
}

type DeploymentChallengerReplaySettings struct {
	Enabled bool `json:"enabled"`
}

type FeatureDriftSetting struct {
	Enabled          bool      `json:"enabled"`
	FeatureSelection *string   `json:"featureSelection,omitempty"`
	TrackedFeatures  *[]string `json:"trackedFeatures,omitempty"`
}

type DeploymentHealthSettings struct {
	Service               *DeploymentServiceHealthSettings       `json:"service,omitempty"`
	DataDrift             *DeploymentDataDriftHealthSettings     `json:"dataDrift,omitempty"`
	Accuracy              *DeploymentAccuracyHealthSettings      `json:"accuracy,omitempty"`
	Fairness              *DeploymentFairnessHealthSettings      `json:"fairness,omitempty"`
	CustomMetrics         *DeploymentCustomMetricsHealthSettings `json:"customMetrics,omitempty"`
	PredictionsTimeliness *DeploymentTimelinessHealthSettings    `json:"predictionsTimeliness,omitempty"`
	ActualsTimeliness     *DeploymentTimelinessHealthSettings    `json:"actualsTimeliness,omitempty"`
}

type DeploymentServiceHealthSettings struct {
	BatchCount int64 `json:"batchCount,omitempty"`
}

type DeploymentDataDriftHealthSettings struct {
	TimeInterval               *string   `json:"timeInterval,omitempty"`
	BatchCount                 *int64    `json:"batchCount,omitempty"`
	DriftThreshold             *float64  `json:"driftThreshold,omitempty"`
	ImportanceThreshold        *float64  `json:"importanceThreshold,omitempty"`
	LowImportanceWarningCount  *int64    `json:"lowImportanceWarningCount,omitempty"`
	LowImportanceFailingCount  *int64    `json:"lowImportanceFailingCount,omitempty"`
	HighImportanceWarningCount *int64    `json:"highImportanceWarningCount,omitempty"`
	HighImportanceFailingCount *int64    `json:"highImportanceFailingCount,omitempty"`
	ExcludedFeatures           *[]string `json:"excludedFeatures,omitempty"`
	StarredFeatures            *[]string `json:"starredFeatures,omitempty"`
}

type DeploymentAccuracyHealthSettings struct {
	BatchCount       *int64   `json:"batchCount,omitempty"`
	Metric           *string  `json:"metric,omitempty"`
	Measurement      *string  `json:"measurement,omitempty"`
	WarningThreshold *float64 `json:"warningThreshold,omitempty"`
	FailingThreshold *float64 `json:"failingThreshold,omitempty"`
}

type DeploymentFairnessHealthSettings struct {
	ProtectedClassWarningCount *int64 `json:"protectedClassWarningCount,omitempty"`
	ProtectedClassFailingCount *int64 `json:"protectedClassFailingCount,omitempty"`
}

type DeploymentCustomMetricsHealthSettings struct {
	WarningConditions []CustomMetricCondition `json:"warningConditions"`
	FailingConditions []CustomMetricCondition `json:"failingConditions"`
}

type DeploymentTimelinessHealthSettings struct {
	Enabled           bool    `json:"enabled"`
	ExpectedFrequency *string `json:"expectedFrequency,omitempty"`
}

type DeploymentFeatureCacheSettings struct {
	Enabled  bool      `json:"enabled"`
	Fetching *bool     `json:"fetching,omitempty"`
	Schedule *Schedule `json:"schedule,omitempty"`
}

type CustomMetricCondition struct {
	MetricID        string  `json:"metricId"`
	CompareOperator string  `json:"compareOperator"`
	Threshold       float64 `json:"threshold"`
}

type ValidateDeployemntModelReplacementRequest struct {
	ModelPackageID string `json:"modelPackageId"`
}

type ValidateDeployemntModelReplacementResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type UpdateDeploymentRuntimeParametersRequest struct {
	RuntimeParameterValues string `json:"runtimeParameterValues"`
}

type UpdateDeploymentModelRequest struct {
	ModelPackageID string `json:"modelPackageId"`
	Reason         string `json:"reason"`
}

type UpdateDeploymentStatusRequest struct {
	Status string `json:"status"`
}

type TaskStatusResponse struct {
	StatusID    string `json:"statusId"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Code        int    `json:"code"`
	Description string `json:"description"`
	StatusType  string `json:"statusType"`
}

type CreateCustomMetricFromJobRequest struct {
	CustomJobID        string                         `json:"customJobId"`
	Name               string                         `json:"name"`
	Description        *string                        `json:"description,omitempty"`
	BaselineValues     []MetricBaselineValue          `json:"baselineValues,omitempty"`
	Timestamp          *MetricTimestampSpoofing       `json:"timestamp,omitempty"`
	Value              *ColumnNameValue               `json:"value,omitempty"`
	SampleCount        *ColumnNameValue               `json:"sampleCount,omitempty"`
	Batch              *ColumnNameValue               `json:"batch,omitempty"`
	Schedule           *Schedule                      `json:"schedule,omitempty"`
	ParameterOverrides []RuntimeParameterValueRequest `json:"parameterOverrides,omitempty"`
}

type MetricTimestampSpoofing struct {
	ColumnName *string `json:"columnName,omitempty"`
	TimeFormat *string `json:"timeFormat,omitempty"`
}

type MetricBaselineValue struct {
	Value float64 `json:"value"`
}

type ColumnNameValue struct {
	ColumnName string `json:"columnName"`
}

type CreateCustomMetricRequest struct {
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	Directionality  string                   `json:"directionality"`
	Units           string                   `json:"units"`
	Type            string                   `json:"type"`
	IsModelSpecific bool                     `json:"isModelSpecific"`
	IsGeospatial    bool                     `json:"isGeospatial"`
	BaselineValues  []MetricBaselineValue    `json:"baselineValues,omitempty"`
	Timestamp       *MetricTimestampSpoofing `json:"timestamp,omitempty"`
	Value           *ColumnNameValue         `json:"value,omitempty"`
	SampleCount     *ColumnNameValue         `json:"sampleCount,omitempty"`
	Batch           *ColumnNameValue         `json:"batch,omitempty"`
}

type CustomMetric struct {
	ID              string                   `json:"id"`
	CustomJobID     string                   `json:"customJobId"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description,omitempty"`
	Units           string                   `json:"units"`
	Directionality  string                   `json:"directionality"`
	Type            string                   `json:"type"`
	IsModelSpecific bool                     `json:"isModelSpecific"`
	IsGeospatial    bool                     `json:"isGeospatial"`
	TimeStep        string                   `json:"timeStep,omitempty"`
	BaselineValues  *[]MetricBaselineValue   `json:"baselineValues,omitempty"`
	Timestamp       *MetricTimestampSpoofing `json:"timestamp,omitempty"`
	Value           *ColumnNameValue         `json:"value,omitempty"`
	SampleCount     *ColumnNameValue         `json:"sampleCount,omitempty"`
	Batch           *ColumnNameValue         `json:"batch,omitempty"`
}

type UpdateCustomMetricRequest struct {
	Name           *string                  `json:"name,omitempty"`
	Units          *string                  `json:"units,omitempty"`
	Directionality *string                  `json:"directionality,omitempty"`
	Type           *string                  `json:"type,omitempty"`
	Description    *string                  `json:"description,omitempty"`
	BaselineValues *[]MetricBaselineValue   `json:"baselineValues,omitempty"`
	Timestamp      *MetricTimestampSpoofing `json:"timestamp,omitempty"`
	Value          *ColumnNameValue         `json:"value,omitempty"`
	SampleCount    *ColumnNameValue         `json:"sampleCount,omitempty"`
	Batch          *ColumnNameValue         `json:"batch,omitempty"`
}

type RetrainingPolicyRequest struct {
	Name                   *string            `json:"name,omitempty"`
	Description            *string            `json:"description,omitempty"`
	Action                 *string            `json:"action,omitempty"`
	AutopilotOptions       *AutopilotOptions  `json:"autopilotOptions,omitempty"`
	FeatureListStrategy    *string            `json:"featureListStrategy,omitempty"`
	ModelSelectionStrategy *string            `json:"modelSelectionStrategy,omitempty"`
	ProjectOptions         *ProjectOptions    `json:"projectOptions,omitempty"`
	ProjectOptionsStrategy *string            `json:"projectOptionsStrategy,omitempty"`
	TimeSeriesOptions      *TimeSeriesOptions `json:"timeSeriesOptions,omitempty"`
	Trigger                *Trigger           `json:"trigger,omitempty"`
}

type AutopilotOptions struct {
	BlendBestModels              *bool   `json:"blendBestModels,omitempty"`
	Mode                         *string `json:"mode,omitempty"`
	RunLeakageRemovedFeatureList *bool   `json:"runLeakageRemovedFeatureList,omitempty"`
	ScoringCodeOnly              *bool   `json:"scoringCodeOnly,omitempty"`
	ShapOnlyMode                 *bool   `json:"shapOnlyMode,omitempty"`
}

type ProjectOptions struct {
	CvMethod       *string  `json:"cvMethod,omitempty"`
	HoldoutPct     *float64 `json:"holdoutPct,omitempty"`
	Metric         *string  `json:"metric,omitempty"`
	Reps           *float64 `json:"reps,omitempty"`
	ValidationPct  *float64 `json:"validationPct,omitempty"`
	ValidationType *string  `json:"validationType,omitempty"`
}

type TimeSeriesOptions struct {
	CalendarID                       *string       `json:"calendarId,omitempty"`
	DifferencingMethod               *string       `json:"differencingMethod,omitempty"`
	ExponentiallyWeightedMovingAlpha *float64      `json:"exponentiallyWeightedMovingAlpha,omitempty"`
	Periodicities                    []Periodicity `json:"periodicities,omitempty"`
	TreatAsExponential               *string       `json:"treatAsExponential,omitempty"`
}

type Periodicity struct {
	TimeSteps *int    `json:"timeSteps,omitempty"`
	TimeUnit  *string `json:"timeUnit,omitempty"`
}

type Trigger struct {
	CustomJobID             *string   `json:"customJobId,omitempty"`
	MinIntervalBetweenRuns  *string   `json:"minIntervalBetweenRuns,omitempty"`
	Schedule                *Schedule `json:"schedule,omitempty"`
	StatusDeclinesToFailing *bool     `json:"statusDeclinesToFailing,omitempty"`
	StatusDeclinesToWarning *bool     `json:"statusDeclinesToWarning,omitempty"`
	StatusStillInDecline    *bool     `json:"statusStillInDecline,omitempty"`
	Type                    *string   `json:"type"`
}

type RetrainingPolicy struct {
	ID                     string             `json:"id"`
	Name                   string             `json:"name"`
	Description            string             `json:"description"`
	Action                 string             `json:"action,omitempty"`
	AutopilotOptions       *AutopilotOptions  `json:"autopilotOptions,omitempty"`
	FeatureListStrategy    string             `json:"featureListStrategy,omitempty"`
	ModelSelectionStrategy string             `json:"modelSelectionStrategy,omitempty"`
	ProjectOptions         *ProjectOptions    `json:"projectOptions,omitempty"`
	ProjectOptionsStrategy string             `json:"projectOptionsStrategy,omitempty"`
	TimeSeriesOptions      *TimeSeriesOptions `json:"timeSeriesOptions,omitempty"`
	Trigger                *Trigger           `json:"trigger,omitempty"`
}

type RetrainingSettings struct{}

type UpdateRetrainingSettingsRequest struct {
	CredentialID            *string `json:"credentialId,omitempty"`
	DatasetID               *string `json:"datasetId,omitempty"`
	PredictionEnvironmentID *string `json:"predictionEnvironmentId,omitempty"`
	RetrainingUserID        *string `json:"retrainingUserId,omitempty"`
}

type RetrainingSettingsItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RetrainingUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type RetrainingSettingsRetrieve struct {
	Dataset               RetrainingSettingsItem `json:"dataset"`
	PredictionEnvironment RetrainingSettingsItem `json:"predictionEnvironment"`
	RetrainingUser        RetrainingUser         `json:"retrainingUser"`
}
