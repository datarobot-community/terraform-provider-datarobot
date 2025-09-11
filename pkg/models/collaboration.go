package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// UseCaseResourceModel describes the resource data model.
type UseCaseResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

