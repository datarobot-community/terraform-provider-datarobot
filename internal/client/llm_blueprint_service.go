package client

type CreateLLMBlueprintRequest struct {
	Name             string `json:"name"`
	PlaygroundID     string `json:"playgroundId"`
	Description      string `json:"description,omitempty"`
	VectorDatabaseID string `json:"vectorDatabaseId,omitempty"`
	LLMID            string `json:"llmId,omitempty"`
}

type CreateLLMBlueprintResponse struct {
	ID string `json:"id"`
}

type UpdateLLMBlueprintRequest struct {
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	VectorDatabaseID string `json:"vectorDatabaseId,omitempty"`
	LLMID            string `json:"llmId,omitempty"`
}

type LLMBlueprintResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	PlaygroundID     string `json:"playgroundId"`
	VectorDatabaseID string `json:"vectorDatabaseId"`
	LLMID            string `json:"llmId"`
}

type ListLLMsResponse struct {
	Data []LanguageModelDefinitionAPIFormatted `json:"data"`
}

type LanguageModelDefinitionAPIFormatted struct {
	ID string `json:"id"`
}
