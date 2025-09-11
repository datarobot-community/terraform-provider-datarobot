package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// VectorDatabaseResourceModel describes a vector database.
type VectorDatabaseResourceModel struct {
	ID                 types.String             `tfsdk:"id"`
	Version            types.Int64              `tfsdk:"version"`
	Name               types.String             `tfsdk:"name"`
	UseCaseID          types.String             `tfsdk:"use_case_id"`
	DatasetID          types.String             `tfsdk:"dataset_id"`
	ChunkingParameters *ChunkingParametersModel `tfsdk:"chunking_parameters"`
}

// ChunkingParametersModel represents the chunking parameters nested attribute.
type ChunkingParametersModel struct {
	EmbeddingModel         types.String   `tfsdk:"embedding_model"`
	ChunkOverlapPercentage types.Int64    `tfsdk:"chunk_overlap_percentage"`
	ChunkSize              types.Int64    `tfsdk:"chunk_size"`
	ChunkingMethod         types.String   `tfsdk:"chunking_method"`
	IsSeparatorRegex       types.Bool     `tfsdk:"is_separator_regex"`
	Separators             []types.String `tfsdk:"separators"`
}
