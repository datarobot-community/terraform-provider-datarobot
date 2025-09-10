package models

import "github.com/hashicorp/terraform-plugin-framework/types"


type RuntimeParameterValue struct {
	Key   types.String `json:"key" tfsdk:"key"`
	Type  types.String `json:"type" tfsdk:"type"`
	Value types.String `json:"value" tfsdk:"value"`
}


type FileTuple struct {
	LocalPath   string
	PathInModel string
}
