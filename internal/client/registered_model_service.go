package client

type CreateRegisteredModelFromCustomModelRequest struct {
	CustomModelVersionID string `json:"customModelVersionId"`
	Name                 string `json:"name"`
	RegisteredModelName  string `json:"registeredModelName,omitempty"`
	Prompt               string `json:"prompt,omitempty"`
	Tags                 []Tag  `json:"tags,omitempty"`

	// To create a new version of an existing registered model
	RegisteredModelID string `json:"registeredModelId,omitempty"`
}

type CreateRegisteredModelFromLeaderboardRequest struct {
	ModelID                       string   `json:"modelId"`
	RegisteredModelName           *string  `json:"registeredModelName,omitempty"`
	RegisteredModelID             *string  `json:"registeredModelId,omitempty"`
	PredictionThreshold           *float64 `json:"predictionThreshold,omitempty"`
	ComputeAllTsIntervals         *bool    `json:"computeAllTsIntervals,omitempty"`
	DistributionPredictionModelID *string  `json:"distributionPredictionModelId,omitempty"`
}

type CreateRegisteredModelFromExternalRequest struct {
	Name                       string                              `json:"name"`
	RegisteredModelName        *string                             `json:"registeredModelName,omitempty"`
	RegisteredModelDescription *string                             `json:"registeredModelDescription,omitempty"`
	ModelID                    *string                             `json:"modelId,omitempty"`
	Target                     ExternalModelPackageTarget          `json:"target"`
	ModelDescription           *ExternalModelPackageDescription    `json:"modelDescription,omitempty"`
	Datasets                   *ExternalModelPackageDatasets       `json:"datasets,omitempty"`
	Timeseries                 *ExternalModelPackageTimeseries     `json:"timeseries,omitempty"`
	TextGeneration             *ExternalModelPackageTextGeneration `json:"textGeneration,omitempty"`
	GeospatialMonitoring       *ExternalModelPackageGeospatial     `json:"geospatialMonitoring,omitempty"`
	Tags                       []Tag                               `json:"tags,omitempty"`

	// To create a new version of an existing registered model
	RegisteredModelID *string `json:"registeredModelId,omitempty"`
}

type ExternalModelPackageTarget struct {
	Type                          string   `json:"type"`
	Name                          *string  `json:"name,omitempty"`
	ClassNames                    []string `json:"classNames,omitempty"`
	PredictionThreshold           *float64 `json:"predictionThreshold,omitempty"`
	PredictionProbabilitiesColumn *string  `json:"predictionProbabilitiesColumn,omitempty"`
}

type ExternalModelPackageDescription struct {
	ModelName            *string `json:"modelName,omitempty"`
	Description          *string `json:"description,omitempty"`
	Location             *string `json:"location,omitempty"`
	BuildEnvironmentType *string `json:"buildEnvironmentType,omitempty"`
}

type ExternalModelPackageDatasets struct {
	TrainingDataCatalogID        *string `json:"trainingDataCatalogId,omitempty"`
	TrainingDataCatalogVersionID *string `json:"trainingDataCatalogVersionId,omitempty"`
	HoldoutDataCatalogID         *string `json:"holdoutDataCatalogId,omitempty"`
	HoldoutDataCatalogVersionID  *string `json:"holdoutDataCatalogVersionId,omitempty"`
}

type ExternalModelPackageTimeseries struct {
	DatetimeColumnName                    string  `json:"datetimeColumnName"`
	DatetimeColumnFormat                  string  `json:"datetimeColumnFormat"`
	ForecastPointColumnName               string  `json:"forecastPointColumnName"`
	ForecastDistanceColumnName            string  `json:"forecastDistanceColumnName"`
	ForecastDistancesTimeUnit             string  `json:"forecastDistancesTimeUnit"`
	ForecastDistances                     []int64 `json:"forecastDistances,omitempty"`
	SeriesColumnName                      *string `json:"seriesColumnName,omitempty"`
	FeatureDerivationWindowStart          *int64  `json:"featureDerivationWindowStart,omitempty"`
	FeatureDerivationWindowEnd            *int64  `json:"featureDerivationWindowEnd,omitempty"`
	EffectiveFeatureDerivationWindowStart *int64  `json:"effectiveFeatureDerivationWindowStart,omitempty"`
	EffectiveFeatureDerivationWindowEnd   *int64  `json:"effectiveFeatureDerivationWindowEnd,omitempty"`
	IsCrossSeries                         *bool   `json:"isCrossSeries,omitempty"`
	IsNewSeriesSupport                    *bool   `json:"isNewSeriesSupport,omitempty"`
	IsTraditionalTimeSeries               *bool   `json:"isTraditionalTimeSeries,omitempty"`
}

type ExternalModelPackageTextGeneration struct {
	Prompt string `json:"prompt"`
}

type ExternalModelPackageGeospatial struct {
	PrimaryLocationColumn string `json:"primaryLocationColumn"`
}

type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type RegisteredModel struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	LastVersionNum int    `json:"lastVersionNum"`
	IsGlobal       bool   `json:"isGlobal"`
}

type UpdateRegisteredModelVersionRequest struct {
	Name                string   `json:"name,omitempty"`
	Description         string   `json:"description,omitempty"`
	PredictionThreshold *float64 `json:"predictionThreshold,omitempty"`
}

type ListRegisteredModelsRequest struct {
	IsGlobal bool   `url:"isGlobal,omitempty"`
	Search   string `url:"search,omitempty"`
}

type ListRegisteredModelsResponse struct {
	Data []RegisteredModel
	Next string
}

type UpdateRegisteredModelRequest struct {
	Name string `json:"name,omitempty"`
	// An empty string clears the description; nil leaves it unchanged.
	Description *string `json:"description,omitempty"`
}

type RegisteredModelVersion struct {
	ID                     string                        `json:"id"` // Registered model version ID
	Name                   string                        `json:"name"`
	BuildStatus            string                        `json:"buildStatus"`
	ModelID                string                        `json:"modelId"`
	RegisteredModelID      string                        `json:"registeredModelId"`
	RegisteredModelVersion int                           `json:"registeredModelVersion"`
	Stage                  string                        `json:"stage"`
	Target                 RegisteredModelVersionTarget  `json:"target"`
	Tags                   []Tag                         `json:"tags"`
	TextGeneration         RegisteredModelTextGeneration `json:"textGeneration"`

	ModelDescription     *RegisteredModelVersionModelDescription     `json:"modelDescription,omitempty"`
	Datasets             *RegisteredModelVersionDatasets             `json:"datasets,omitempty"`
	Timeseries           *RegisteredModelVersionTimeseries           `json:"timeseries,omitempty"`
	GeospatialMonitoring *RegisteredModelVersionGeospatialMonitoring `json:"geospatialMonitoring,omitempty"`
}

type RegisteredModelVersionModelDescription struct {
	ModelName            string  `json:"modelName"`
	Description          *string `json:"description,omitempty"`
	Location             *string `json:"location,omitempty"`
	BuildEnvironmentType string  `json:"buildEnvironmentType"`
}

type RegisteredModelVersionDatasets struct {
	TrainingDataCatalogID        *string `json:"trainingDataCatalogId,omitempty"`
	TrainingDataCatalogVersionID *string `json:"trainingDataCatalogVersionId,omitempty"`
	HoldoutDataCatalogID         *string `json:"holdoutDataCatalogId,omitempty"`
	HoldoutDataCatalogVersionID  *string `json:"holdoutDataCatalogVersionId,omitempty"`
}

type RegisteredModelVersionTimeseries struct {
	DatetimeColumnName                    *string `json:"datetimeColumnName,omitempty"`
	DatetimeColumnFormat                  *string `json:"datetimeColumnFormat,omitempty"`
	ForecastPointColumnName               *string `json:"forecastPointColumnName,omitempty"`
	ForecastDistanceColumnName            *string `json:"forecastDistanceColumnName,omitempty"`
	ForecastDistancesTimeUnit             *string `json:"forecastDistancesTimeUnit,omitempty"`
	ForecastDistances                     []int64 `json:"forecastDistances,omitempty"`
	SeriesColumnName                      *string `json:"seriesColumnName,omitempty"`
	FeatureDerivationWindowStart          *int64  `json:"featureDerivationWindowStart,omitempty"`
	FeatureDerivationWindowEnd            *int64  `json:"featureDerivationWindowEnd,omitempty"`
	EffectiveFeatureDerivationWindowStart *int64  `json:"effectiveFeatureDerivationWindowStart,omitempty"`
	EffectiveFeatureDerivationWindowEnd   *int64  `json:"effectiveFeatureDerivationWindowEnd,omitempty"`
	IsCrossSeries                         *bool   `json:"isCrossSeries,omitempty"`
	IsNewSeriesSupport                    *bool   `json:"isNewSeriesSupport,omitempty"`
	IsTraditionalTimeSeries               *bool   `json:"isTraditionalTimeSeries,omitempty"`
}

type RegisteredModelVersionGeospatialMonitoring struct {
	PrimaryLocationColumn *string `json:"primaryLocationColumn,omitempty"`
}

type RegisteredModelTextGeneration struct {
	Prompt *string `json:"prompt,omitempty"`
}

type RegisteredModelVersionTarget struct {
	Name                          string   `json:"name"`
	Type                          string   `json:"type"`
	PredictionThreshold           *float64 `json:"predictionThreshold,omitempty"`
	PredictionProbabilitiesColumn *string  `json:"predictionProbabilitiesColumn,omitempty"`
	ClassNames                    []string `json:"classNames,omitempty"`
	ClassCount                    *int     `json:"classCount,omitempty"`
}

type ListRegisteredModelVersionsResponse struct {
	Data []RegisteredModelVersion
	Next string
}
