package client

type CreateRegisteredModelFromCustomModelRequest struct {
	CustomModelVersionID string `json:"customModelVersionId"`
	Name                 string `json:"name"`
	RegisteredModelName  string `json:"registeredModelName,omitempty"`
	Prompt               string `json:"prompt,omitempty"`

	// To create a new version of an existing registered model
	RegisteredModelID string `json:"registeredModelId,omitempty"`
}

type RegisteredModelResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	LastVersionNum int    `json:"lastVersionNum"`
	IsGlobal       bool   `json:"isGlobal"`
}

type ListRegisteredModelsResponse struct {
	Data []RegisteredModelResponse
}

type RegisteredModelUpdate struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type RegisteredModelVersionResponse struct {
	ID                     string `json:"id"` // Registered model version ID
	Name                   string `json:"name"`
	BuildStatus            string `json:"buildStatus"`
	ModelID                string `json:"modelId"`
	RegisteredModelID      string `json:"registeredModelId"`
	RegisteredModelVersion int    `json:"registeredModelVersion"`
	Stage                  string `json:"stage"`
}

type ListRegisteredModelVersionsResponse struct {
	Data []RegisteredModelVersionResponse
}
