package client

type PredictionEnvironmentRequest struct {
	Name                             string   `json:"name"`
	Description                      string   `json:"description"`
	Platform                         string   `json:"platform"`
	Priority                         *int     `json:"priority,omitempty"`
	MaxConcurrentBatchPredictionsJob *int64   `json:"maxConcurrentBatchPredictionsJob,omitempty"`
	ManagedBy                        string   `json:"managedBy"`
	IsManagedByManagementAgent       bool     `json:"isManagedByManagementAgent"`
	SupportedModelFormats            []string `json:"supportedModelFormats,omitempty"`
	CredentialID                     *string  `json:"credentialId,omitempty"`
	DatastoreID                      *string  `json:"datastoreId,omitempty"`
}

type PredictionEnvironment struct {
	ID                         string `json:"id"`
	Name                       string `json:"name"`
	Description                string `json:"description,omitempty"`
	Platform                   string `json:"platform"`
	IsManagedByManagementAgent bool   `json:"isManagedByManagementAgent"`
	ManagedBy                  string `json:"managedBy"`
}
