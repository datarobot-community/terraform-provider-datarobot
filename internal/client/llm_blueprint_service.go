package client

type CreateLLMBlueprintRequest struct {
	Name                   string                  `json:"name"`
	PlaygroundID           string                  `json:"playgroundId"`
	Description            string                  `json:"description,omitempty"`
	VectorDatabaseID       string                  `json:"vectorDatabaseId,omitempty"`
	VectorDatabaseSettings *VectorDatabaseSettings `json:"vectorDatabaseSettings,omitempty"`
	LLMID                  *string                 `json:"llmId,omitempty"`
	LLMSettings            interface{}             `json:"llmSettings,omitempty"`
	PromptType             string                  `json:"promptType,omitempty"`
}

type VectorDatabaseSettings struct {
	MaxDocumentsRetrievedPerPrompt int64 `json:"maxDocumentsRetrievedPerPrompt"`
	MaxTokens                      int64 `json:"maxTokens"`
}

type LLMSettings struct {
	MaxCompletionLength *int64   `json:"maxCompletionLength,omitempty"`
	Temperature         *float64 `json:"temperature,omitempty"`
	TopP                *float64 `json:"topP,omitempty"`
	SystemPrompt        *string  `json:"systemPrompt,omitempty"`
}

type CustomModelLLMSettings struct {
	ExternalLLMContextSize *int64  `json:"externalLlmContextSize"`
	SystemPrompt           *string `json:"systemPrompt,omitempty"`
	ValidationID           *string `json:"validationId,omitempty"`
}

type UpdateLLMBlueprintRequest struct {
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	VectorDatabaseID string `json:"vectorDatabaseId,omitempty"`
	LLMID            string `json:"llmId,omitempty"`
}

type LLMBlueprint struct {
	ID                     string                  `json:"id"`
	Name                   string                  `json:"name"`
	Description            string                  `json:"description"`
	PlaygroundID           string                  `json:"playgroundId"`
	VectorDatabaseID       string                  `json:"vectorDatabaseId"`
	LLMID                  *string                 `json:"llmId,omitempty"`
	LLMSettings            *LLMSettings            `json:"llmSettings,omitempty"`
	CustomModelLLMSettings *CustomModelLLMSettings `json:"customModelLlmSettings,omitempty"`
	PromptType             string                  `json:"promptType"`
}

type LanguageModelDefinitionAPIFormatted struct {
	ID string `json:"id"`
}
