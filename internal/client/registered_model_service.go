package client

type CreateRegisteredModelFromCustomModelRequest struct {
	CustomModelVersionID string `json:"customModelVersionId"`
	Name                 string `json:"name"`
	RegisteredModelName  string `json:"registeredModelName,omitempty"`
	Prompt               string `json:"prompt,omitempty"`

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
	PredictionThreshold *float64 `json:"predictionThreshold,omitempty"`
}

type ListRegisteredModelsResponse struct {
	Data []RegisteredModel
}

type UpdateRegisteredModelRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type RegisteredModelVersion struct {
	ID                     string                       `json:"id"` // Registered model version ID
	Name                   string                       `json:"name"`
	BuildStatus            string                       `json:"buildStatus"`
	ModelID                string                       `json:"modelId"`
	RegisteredModelID      string                       `json:"registeredModelId"`
	RegisteredModelVersion int                          `json:"registeredModelVersion"`
	Stage                  string                       `json:"stage"`
	Target                 RegisteredModelVersionTarget `json:"target"`
	Tags                   []Tag                        `json:"tags"`
}

type RegisteredModelVersionTarget struct {
	Name                string   `json:"name"`
	Type                string   `json:"type"`
	PredictionThreshold *float64 `json:"predictionThreshold,omitempty"`
}

type ListRegisteredModelVersionsResponse struct {
	Data []RegisteredModelVersion
}
