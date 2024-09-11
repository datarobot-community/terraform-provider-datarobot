package client

type CreateDatasetRequest struct {
	DoSnapshot bool `json:"doSnapshot"`
}

type CreateDatasetResponse struct {
	ID string `json:"datasetId"`
}

type CreateDatasetVersionResponse struct {
	// The ID of the catalog entry.
	ID string `json:"catalogId"`
	// The ID of the latest version of the catalog entry.
	VersionID string `json:"catalogVersionId"`
	// ID that can be used with GET /api/v2/status/{statusId}/ to poll for the testing job's status.
	StatusID string `json:"statusId"`
}

type UpdateDatasetRequest struct {
	Name string `json:"name"`
}

type Dataset struct {
	ID                                    string  `json:"datasetId"`
	Name                                  string  `json:"name"`
	VersionID                             string  `json:"versionId"`
	DataSourceType                        string  `json:"dataSourceType"`
	ProcessingState                       string  `json:"processingState"`
	Error                                 *string `json:"error"`
	IsSnapshot                            bool    `json:"isSnapshot"`
	IsVectorDatabaseEligible              bool    `json:"isVectorDatabaseEligible"`
	IsDataEngineEligible                  bool    `json:"isDataEngineEligible"`
	IsWranglingEligible                   bool    `json:"isWranglingEligible"`
	IsPlaygroundEvaluationDatasetEligible bool    `json:"isPlaygroundEvaluationDatasetEligible"`
}
