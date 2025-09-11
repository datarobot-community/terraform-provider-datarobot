package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NotebookResourceModel describes the notebook resource.
type NotebookResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	FilePath  types.String `tfsdk:"file_path"`
	FileHash  types.String `tfsdk:"file_hash"`
	UseCaseID types.String `tfsdk:"use_case_id"`
	URL       types.String `tfsdk:"url"`
}
