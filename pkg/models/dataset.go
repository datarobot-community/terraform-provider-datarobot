package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// DatasetFromFileResourceModel describes the dataset uploaded from a local file.
type DatasetFromFileResourceModel struct {
    ID         types.String   `tfsdk:"id"`
    FilePath   types.String   `tfsdk:"file_path"`
    FileHash   types.String   `tfsdk:"file_hash"`
    Name       types.String   `tfsdk:"name"`
    UseCaseIDs []types.String `tfsdk:"use_case_ids"`
}
