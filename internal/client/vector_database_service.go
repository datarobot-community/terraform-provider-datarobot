package client

type CreateVectorDatabaseRequest struct {
	DatasetID          string             `json:"datasetId"`
	Name               string             `json:"name"`
	UseCaseID          string             `json:"useCaseId"`
	ChunkingParameters ChunkingParameters `json:"chunkingParameters"`
}

type ChunkingParameters struct {
	ChunkOverlapPercentage int64    `json:"chunkOverlapPercentage"`
	ChunkSize              int64    `json:"chunkSize"`                // Value must be greater than or equal to 128
	ChunkingMethod         string   `json:"chunkingMethod"`           // [recursive, semantic]
	EmbeddingModel         string   `json:"embeddingModel,omitempty"` // [intfloat/e5-large-v2, intfloat/e5-base-v2, intfloat/multilingual-e5-base, sentence-transformers/all-MiniLM-L6-v2, jinaai/jina-embedding-t-en-v1, cl-nagoya/sup-simcse-ja-base]
	EmbeddingValidationId  string   `json:"embeddingValidationId,omitempty"`
	IsSeparatorRegex       bool     `json:"isSeparatorRegex,omitempty"`
	Separators             []string `json:"separators"`
}

type VectorDatabase struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	UseCaseID              string   `json:"useCaseId"`
	DatasetID              string   `json:"datasetId"`
	ExecutionStatus        string   `json:"executionStatus"`
	EmbeddingModel         string   `json:"embeddingModel"`
	ChunkSize              int64    `json:"chunkSize"`
	ChunkingMethod         string   `json:"chunkingMethod"`
	ChunkOverlapPercentage int64    `json:"chunkOverlapPercentage"`
	IsSeparatorRegex       bool     `json:"isSeparatorRegex"`
	Separators             []string `json:"separators"`
}

type UpdateVectorDatabaseRequest struct {
	Name string `json:"name"`
}
