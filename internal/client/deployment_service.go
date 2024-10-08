package client

type CreateDeploymentFromModelPackageRequest struct {
	ModelPackageID          string `json:"modelPackageId"`
	PredictionEnvironmentID string `json:"predictionEnvironmentId"`
	Label                   string `json:"label"`
	Importance              string `json:"importance"`
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
	FeatureDrift              *BasicSetting                      `json:"featureDrift,omitempty"`
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
	MinComputes int64 `json:"minComputes"`
	MaxComputes int64 `json:"maxComputes"`
	RealTime    bool  `json:"realTime"`
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

type UpdateDeploymentModelRequest struct {
	ModelPackageID string `json:"modelPackageId"`
	Reason         string `json:"reason"`
}

type TaskStatusResponse struct {
	StatusID    string `json:"statusId"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Code        int    `json:"code"`
	Description string `json:"description"`
	StatusType  string `json:"statusType"`
}
