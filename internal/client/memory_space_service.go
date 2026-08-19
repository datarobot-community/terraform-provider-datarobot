package client

// MemorySpaceRequest represents a request to create or update a memory space.
//
// The fields deliberately carry no omitempty. The update endpoint applies only the
// keys present in the body, so an omitted key leaves the stored value untouched and
// an explicit null is the only way to clear one. A nil pointer must therefore reach
// the wire as null rather than disappear.
type MemorySpaceRequest struct {
	Description        *string `json:"description"`
	LLMModelName       *string `json:"llmModelName"`
	LLMBaseURL         *string `json:"llmBaseUrl"`
	CustomInstructions *string `json:"customInstructions"`
}

// MemorySpaceResponse represents the API response for a memory space.
type MemorySpaceResponse struct {
	MemorySpaceID      string `json:"memorySpaceId"`
	Description        string `json:"description"`
	LLMModelName       string `json:"llmModelName"`
	LLMBaseURL         string `json:"llmBaseUrl"`
	CustomInstructions string `json:"customInstructions"`
}
