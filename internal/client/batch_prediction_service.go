package client

type BatchPredictionJobDefinitionRequest struct {
	Enabled                      bool                `json:"enabled"`
	Name                         *string             `json:"name,omitempty"`
	Schedule                     *Schedule           `json:"schedule,omitempty"`
	DeploymentId                 string              `json:"deploymentId"`
	AbortOnError                 *bool               `json:"abortOnError,omitempty"`
	BatchJobType                 *string             `json:"batchJobType,omitempty"`
	ChunkSize                    any                 `json:"chunkSize,omitempty"`
	ColumnNamesRemapping         map[string]string   `json:"columnNamesRemapping,omitempty"`
	CSVSettings                  *CSVSettings        `json:"csvSettings,omitempty"`
	DisableRowLevelErrorHandling *bool               `json:"disableRowLevelErrorHandling,omitempty"`
	ExplanationAlgorithm         *string             `json:"explanationAlgorithm,omitempty"`
	ExplanationsMode             *string             `json:"explanationsMode,omitempty"`
	IncludePredictionStatus      *bool               `json:"includePredictionStatus,omitempty"`
	IncludeProbabilities         *bool               `json:"includeProbabilities,omitempty"`
	IncludeProbabilitiesClasses  []string            `json:"includeProbabilitiesClasses,omitempty"`
	IntakeSettings               *IntakeSettings     `json:"intakeSettings,omitempty"`
	MaxExplanations              *int64              `json:"maxExplanations,omitempty"`
	MaxNGramExplanations         any                 `json:"maxNGramExplanations,omitempty"`
	ModelId                      *string             `json:"modelId,omitempty"`
	ModelPackageId               *string             `json:"modelPackageId,omitempty"`
	MonitoringBatchPrefix        *string             `json:"monitoringBatchPrefix,omitempty"`
	NumConcurrent                *int64              `json:"numConcurrent,omitempty"`
	OutputSettings               *OutputSettings     `json:"outputSettings,omitempty"`
	PassthroughColumns           []string            `json:"passthroughColumns,omitempty"`
	PassthroughColumnsSet        *string             `json:"passthroughColumnsSet,omitempty"`
	PinnedModelId                *string             `json:"pinnedModelId,omitempty"`
	PredictionInstance           *PredictionInstance `json:"predictionInstance,omitempty"`
	PredictionThreshold          *float64            `json:"predictionThreshold,omitempty"`
	PredictionWarningEnabled     *bool               `json:"predictionWarningEnabled,omitempty"`
	SecondaryDatasetsConfigId    *string             `json:"secondaryDatasetsConfigId,omitempty"`
	SkipDriftTracking            *bool               `json:"skipDriftTracking,omitempty"`
	ThresholdHigh                *float64            `json:"thresholdHigh,omitempty"`
	ThresholdLow                 *float64            `json:"thresholdLow,omitempty"`
	TimeseriesSettings           *TimeseriesSettings `json:"timeseriesSettings,omitempty"`
}

type Schedule struct {
	Minute     any `json:"minute,omitempty"`
	Hour       any `json:"hour,omitempty"`
	Month      any `json:"month,omitempty"`
	DayOfMonth any `json:"dayOfMonth,omitempty"`
	DayOfWeek  any `json:"dayOfWeek,omitempty"`
}

type IntakeSettings struct {
	Type         string  `json:"type"`
	DatasetID    *string `json:"datasetId,omitempty"`
	File         *string `json:"file,omitempty"`
	URL          *string `json:"url,omitempty"`
	CredentialID *string `json:"credentialId,omitempty"`
	EndpointURL  *string `json:"endpointUrl,omitempty"`
	DataStoreID  *string `json:"dataStoreId,omitempty"`
	Query        *string `json:"query,omitempty"`
	Table        *string `json:"table,omitempty"`
	Schema       *string `json:"schema,omitempty"`
	Catalog      *string `json:"catalog,omitempty"`
	FetchSize    *int64  `json:"fetchSize,omitempty"`
}

type CSVSettings struct {
	Delimiter *string `json:"delimiter,omitempty"`
	Encoding  *string `json:"encoding,omitempty"`
	QuoteChar *string `json:"quotechar,omitempty"`
}

type OutputSettings struct {
	Type                   string   `json:"type"`
	CredentialID           *string  `json:"credentialId,omitempty"`
	URL                    *string  `json:"url,omitempty"`
	Path                   *string  `json:"path,omitempty"`
	EndpointURL            *string  `json:"endpointUrl,omitempty"`
	DataStoreID            *string  `json:"dataStoreId,omitempty"`
	Table                  *string  `json:"table,omitempty"`
	Schema                 *string  `json:"schema,omitempty"`
	Catalog                *string  `json:"catalog,omitempty"`
	StatementType          *string  `json:"statementType,omitempty"`
	UpdateColumns          []string `json:"updateColumns,omitempty"`
	WhereColumns           []string `json:"whereColumns,omitempty"`
	CreateTableIfNotExists *bool    `json:"createTableIfNotExists,omitempty"`
}

type PredictionInstance struct {
	HostName     string  `json:"hostName"`
	SSLEnabled   bool    `json:"sslEnabled"`
	ApiKey       *string `json:"apiKey,omitempty"`
	DatarobotKey *string `json:"datarobotKey,omitempty"`
}

type TimeseriesSettings struct {
	Type                             string  `json:"type"`
	ForecastPoint                    *string `json:"forecastPoint,omitempty"`
	RelaxKnownInAdvanceFeaturesCheck *bool   `json:"relaxKnownInAdvanceFeaturesCheck,omitempty"`
	PredictionsStartDate             *string `json:"predictionsStartDate,omitempty"`
	PredictionsEndDate               *string `json:"predictionsEndDate,omitempty"`
}

type BatchPredictionJob struct {
	CreatedBy                CreatedBy              `json:"createdBy"`
	ElapsedTimeSec           int64                  `json:"elapsedTimeSec"`
	FailedRows               int64                  `json:"failedRows"`
	ID                       string                 `json:"id"`
	IntakeDatasetDisplayName string                 `json:"intakeDatasetDisplayName"`
	JobIntakeSize            int64                  `json:"jobIntakeSize"`
	JobOutputSize            int64                  `json:"jobOutputSize"`
	JobSpec                  BatchPredictionJobSpec `json:"jobSpec"`
	Links                    Links                  `json:"links"`
	Logs                     []string               `json:"logs"`
	MonitoringBatchID        string                 `json:"monitoringBatchId"`
	PercentageCompleted      float64                `json:"percentageCompleted"`
	QueuePosition            int64                  `json:"queuePosition"`
	Queued                   bool                   `json:"queued"`
	ResultsDeleted           bool                   `json:"resultsDeleted"`
	ScoredRows               int64                  `json:"scoredRows"`
	SkippedRows              int64                  `json:"skippedRows"`
	Source                   string                 `json:"source"`
	Status                   string                 `json:"status"`
	StatusDetails            string                 `json:"statusDetails"`
}

type CreatedBy struct {
	FullName string `json:"fullName"`
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type BatchPredictionJobSpec struct {
	AbortOnError                 bool                `json:"abortOnError,omitempty"`
	BatchJobType                 string              `json:"batchJobType,omitempty"`
	ChunkSize                    any                 `json:"chunkSize,omitempty"`
	ColumnNamesRemapping         []map[string]string `json:"columnNamesRemapping,omitempty"`
	CSVSettings                  *CSVSettings        `json:"csvSettings,omitempty"`
	DeploymentID                 string              `json:"deploymentId,omitempty"`
	DisableRowLevelErrorHandling bool                `json:"disableRowLevelErrorHandling,omitempty"`
	ExplanationAlgorithm         string              `json:"explanationAlgorithm,omitempty"`
	IncludePredictionStatus      bool                `json:"includePredictionStatus,omitempty"`
	IncludeProbabilities         bool                `json:"includeProbabilities,omitempty"`
	IncludeProbabilitiesClasses  []string            `json:"includeProbabilitiesClasses,omitempty"`
	IntakeSettings               *IntakeSettings     `json:"intakeSettings,omitempty"`
	MaxExplanations              int64               `json:"maxExplanations,omitempty"`
	ModelID                      string              `json:"modelId,omitempty"`
	ModelPackageID               string              `json:"modelPackageId,omitempty"`
	MonitoringBatchPrefix        string              `json:"monitoringBatchPrefix,omitempty"`
	NumConcurrent                int64               `json:"numConcurrent,omitempty"`
	OutputSettings               *OutputSettings     `json:"outputSettings,omitempty"`
	PassthroughColumns           []string            `json:"passthroughColumns,omitempty"`
	PassthroughColumnsSet        *string             `json:"passthroughColumnsSet,omitempty"`
	PinnedModelID                string              `json:"pinnedModelId,omitempty"`
	PredictionInstance           *PredictionInstance `json:"predictionInstance,omitempty"`
	PredictionThreshold          *float64            `json:"predictionThreshold,omitempty"`
	PredictionWarningEnabled     bool                `json:"predictionWarningEnabled,omitempty"`
	RedactedFields               []string            `json:"redactedFields,omitempty"`
	SecondaryDatasetsConfigID    string              `json:"secondaryDatasetsConfigId,omitempty"`
	SkipDriftTracking            bool                `json:"skipDriftTracking,omitempty"`
	ThresholdHigh                *float64            `json:"thresholdHigh,omitempty"`
	ThresholdLow                 *float64            `json:"thresholdLow,omitempty"`
	TimeseriesSettings           *TimeseriesSettings `json:"timeseriesSettings,omitempty"`
}

type Links struct {
	CSVUpload string `json:"csvUpload"`
	Download  string `json:"download"`
	Self      string `json:"self"`
}

type BatchPredictionJobDefinition struct {
	ID                 string                 `json:"id"`
	Enabled            bool                   `json:"enabled"`
	Name               string                 `json:"name"`
	BatchPredictionJob BatchPredictionJobSpec `json:"batchPredictionJob"`
}
