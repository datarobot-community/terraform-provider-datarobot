package client

// MemorySpaceRequest represents a request to create or update a memory space.
type MemorySpaceRequest struct {
	Description        *string `json:"description,omitempty"`
	LLMModelName       *string `json:"llmModelName,omitempty"`
	LLMBaseURL         *string `json:"llmBaseUrl,omitempty"`
	CustomInstructions *string `json:"customInstructions,omitempty"`
}

// MemorySpaceResponse represents the API response for a memory space.
type MemorySpaceResponse struct {
	MemorySpaceID      string `json:"memorySpaceId"`
	Description        string `json:"description"`
	LLMModelName       string `json:"llmModelName"`
	LLMBaseURL         string `json:"llmBaseUrl"`
	CustomInstructions string `json:"customInstructions"`
}
